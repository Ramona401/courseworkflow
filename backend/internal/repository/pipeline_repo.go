package repository

// pipeline_repo.go — Pipeline主记录数据访问层（主文件）
//
// 职责：
//   - Dashboard统计
//   - Pipeline主记录CRUD（创建/查询/列表/状态更新/删除）
//   - 2审辅助方法（审核轮次/退回原因/步骤重置）
//   - Pipeline分配（by LessonPlanID查询）
//   - 工具函数（buildPlaceholders）
//
// Step/EvalRound CRUD → pipeline_repo_steps.go
// GeneratedPages CRUD + 分配 → pipeline_repo_pages.go

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrPipelineNotFound = fmt.Errorf("Pipeline不存在")
	ErrStepNotFound     = fmt.Errorf("Pipeline步骤不存在")
)

// ==================== Dashboard统计 ====================

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalCourses     int   `json:"total_courses"`
	CoursesWithIndex int   `json:"courses_with_index"`
	TotalPipelines   int   `json:"total_pipelines"`
	RunningPipelines int   `json:"running_pipelines"`
	ReviewQueue      int   `json:"review_queue"`
	Finalized        int   `json:"finalized"`
	Failed           int   `json:"failed"`
	// 修复E-05：改用LEFT JOIN一次聚合，消除N+1子查询
	PassedCount     int   `json:"passed_count"`
	TotalTokensUsed int64 `json:"total_tokens_used"`
}

// GetDashboardStats 获取仪表盘统计数据（一次查询获取所有统计）
func GetDashboardStats() (*DashboardStats, error) {
	ctx := context.Background()
	stats := &DashboardStats{}

	// 1. 课程统计
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM course_indexes ci WHERE ci.course_id = c.id))
		 FROM courses c`).Scan(&stats.TotalCourses, &stats.CoursesWithIndex)
	if err != nil {
		return nil, fmt.Errorf("查询课程统计失败: %w", err)
	}

	// 2. Pipeline各状态统计 + 达标数
	// 修复E-05：PassedCount使用LEFT JOIN+CTE一次聚合，不再对每行执行子查询
	err = database.DB.QueryRow(ctx,
		`WITH eval_passed AS (
		        SELECT DISTINCT ps.pipeline_id
		        FROM pipeline_steps ps
		        WHERE ps.step_name = 'evaluator'
		          AND ps.status = 'done'
		          AND ps.step_data IS NOT NULL
		          AND (ps.step_data->>'avg_total')::numeric >= 9.0
		)
		SELECT
		        COUNT(*),
		        COUNT(*) FILTER (WHERE p.status = 'running'),
		        COUNT(*) FILTER (WHERE p.status = 'review_queue'),
		        COUNT(*) FILTER (WHERE p.status = 'finalized'),
		        COUNT(*) FILTER (WHERE p.status = 'failed'),
		        COUNT(ep.pipeline_id)
		FROM pipelines p
		LEFT JOIN eval_passed ep ON ep.pipeline_id = p.id`).Scan(
		&stats.TotalPipelines,
		&stats.RunningPipelines,
		&stats.ReviewQueue,
		&stats.Finalized,
		&stats.Failed,
		&stats.PassedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("查询Pipeline统计失败: %w", err)
	}

	// 3. AI token消耗统计（非关键，失败不影响其他）
	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(tokens_used), 0) FROM pipeline_steps WHERE tokens_used > 0`,
	).Scan(&stats.TotalTokensUsed)

	return stats, nil
}

// ==================== Pipeline主记录CRUD ====================

// CreatePipeline 创建Pipeline主记录+8个步骤记录（原子事务）
func CreatePipeline(p *models.Pipeline) error {
	ctx := context.Background()

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 插入Pipeline主记录
	err = tx.QueryRow(ctx,
		`INSERT INTO pipelines (course_code, course_name, external_module_id, started_by,
		        current_step, status, auto_mode, config, review_round)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		 RETURNING id, started_at, created_at, updated_at`,
		p.CourseCode, p.CourseName, p.ExternalModuleID, p.StartedBy,
		p.CurrentStep, p.Status, p.AutoMode, p.Config, p.ReviewRound,
	).Scan(&p.ID, &p.StartedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("插入Pipeline失败: %w", err)
	}

	// 批量插入步骤记录（全部初始pending）
	for _, sd := range models.StepDefinitions {
		_, err = tx.Exec(ctx,
			`INSERT INTO pipeline_steps (pipeline_id, step_name, step_order, status)
			 VALUES ($1, $2, $3, $4)`,
			p.ID, sd.Name, sd.Order, models.StepStatusPending,
		)
		if err != nil {
			return fmt.Errorf("插入步骤 %s 失败: %w", sd.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// GetPipelineByID 根据ID获取Pipeline主记录
func GetPipelineByID(id string) (*models.Pipeline, error) {
	ctx := context.Background()
	p := &models.Pipeline{}

	var configStr, errorMsg, courseName, rejectReason *string

	err := database.DB.QueryRow(ctx,
		`SELECT id, course_code, course_name, external_module_id, started_by,
		        started_at, completed_at, current_step, status, auto_mode,
		        error_message, config::text, review_round, assigned_to,
		        reject_reason, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id).Scan(
		&p.ID, &p.CourseCode, &courseName, &p.ExternalModuleID, &p.StartedBy,
		&p.StartedAt, &p.CompletedAt, &p.CurrentStep, &p.Status, &p.AutoMode,
		&errorMsg, &configStr, &p.ReviewRound, &p.AssignedTo,
		&rejectReason, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	if courseName != nil {
		p.CourseName = *courseName
	}
	if errorMsg != nil {
		p.ErrorMessage = *errorMsg
	}
	if configStr != nil {
		p.Config = *configStr
	}
	if rejectReason != nil {
		p.RejectReason = *rejectReason
	}
	return p, nil
}

// GetAssignedUserName 根据用户ID获取display_name（单条查询，O(1)）
// 修复v33 P-03：替代原来每次GetPipelineDetail都查全量用户表的N+1问题
func GetAssignedUserName(userID string) string {
	if userID == "" {
		return ""
	}
	ctx := context.Background()
	var displayName string
	if err := database.DB.QueryRow(ctx,
		`SELECT display_name FROM users WHERE id = $1`, userID).Scan(&displayName); err != nil {
		return ""
	}
	return displayName
}

// pipelineListSelectSQL Pipeline列表查询SELECT子句
// v69优化（编号8）：从5个关联子查询改为LEFT JOIN预聚合，消除N+1查询
// 原方案：每行5个子查询（assigned_name/steps_completed/eval/meta/translator分数）
// 新方案：预聚合CTE计算steps_completed，3个分数用LEFT JOIN pipeline_steps按step_name精确匹配
const pipelineListSelectSQL = `SELECT p.id, p.course_code, p.course_name, p.external_module_id,
        p.current_step, p.status, p.auto_mode, p.error_message,
        p.started_by, p.started_at, p.completed_at, p.created_at,
        p.review_round, p.assigned_to,
        u_assign.display_name AS assigned_name,
        COALESCE(sc.done_count, 0) AS steps_completed,
        (ps_eval.step_data->>'avg_total')::numeric AS eval_avg_score,
        (ps_meta.step_data->>'total_final')::numeric AS meta_score,
        (ps_trans.step_data->>'final_score')::numeric AS translator_score
FROM pipelines p
LEFT JOIN users u_assign ON u_assign.id = p.assigned_to
LEFT JOIN (
        SELECT pipeline_id, COUNT(*) AS done_count
        FROM pipeline_steps WHERE status = 'done'
        GROUP BY pipeline_id
) sc ON sc.pipeline_id = p.id
LEFT JOIN LATERAL (
        SELECT step_data FROM pipeline_steps
        WHERE pipeline_id = p.id AND step_name = 'evaluator' AND status = 'done' AND step_data IS NOT NULL
        ORDER BY review_round DESC LIMIT 1
) ps_eval ON true
LEFT JOIN LATERAL (
        SELECT step_data FROM pipeline_steps
        WHERE pipeline_id = p.id AND step_name = 'meta' AND status = 'done' AND step_data IS NOT NULL
        ORDER BY review_round DESC LIMIT 1
) ps_meta ON true
LEFT JOIN LATERAL (
        SELECT step_data FROM pipeline_steps
        WHERE pipeline_id = p.id AND step_name = 'translator' AND status = 'done' AND step_data IS NOT NULL
        ORDER BY review_round DESC LIMIT 1
) ps_trans ON true`

// scanPipelineListRow 扫描Pipeline列表行（含3个分数字段）
func scanPipelineListRow(rows interface{ Scan(dest ...interface{}) error }) (*models.PipelineListItem, error) {
	item := &models.PipelineListItem{}
	var courseName, errorMsg, assignedName *string

	err := rows.Scan(
		&item.ID, &item.CourseCode, &courseName, &item.ExternalModuleID,
		&item.CurrentStep, &item.Status, &item.AutoMode, &errorMsg,
		&item.StartedBy, &item.StartedAt, &item.CompletedAt, &item.CreatedAt,
		&item.ReviewRound, &item.AssignedTo, &assignedName,
		&item.StepsCompleted,
		&item.EvalAvgScore,
		&item.MetaScore,
		&item.TranslatorScore,
	)
	if err != nil {
		return nil, fmt.Errorf("扫描Pipeline行失败: %w", err)
	}

	if courseName != nil {
		item.CourseName = *courseName
	}
	if errorMsg != nil {
		item.ErrorMessage = *errorMsg
	}
	if assignedName != nil {
		item.AssignedName = *assignedName
	}

	item.CurrentStepName = models.StepNameMap[item.CurrentStep]
	item.StatusName      = models.PipelineStatusNameMap[item.Status]
	item.StepsTotal      = models.TotalSteps
	return item, nil
}

// ListPipelines 获取Pipeline列表（按创建时间倒序）
func ListPipelines() ([]*models.PipelineListItem, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx, pipelineListSelectSQL+" ORDER BY p.created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("查询Pipeline列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PipelineListItem
	for rows.Next() {
		item, err := scanPipelineListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ListPipelinesForUser 获取指定用户可见的Pipeline列表
// admin/senior_operator看所有，其他角色只看自己发起或分配的
func ListPipelinesForUser(userID string, role string) ([]*models.PipelineListItem, error) {
	ctx := context.Background()
	var query string
	var args []interface{}

	if role == "admin" || role == "senior_operator" {
		query = pipelineListSelectSQL + " ORDER BY p.created_at DESC"
	} else {
		query = pipelineListSelectSQL + `
		 WHERE p.started_by = $1
		        OR p.assigned_to = $1
		        OR p.course_code IN (SELECT uca.course_code FROM user_course_assignments uca WHERE uca.user_id = $1)
		 ORDER BY p.created_at DESC`
		args = append(args, userID)
	}

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询用户Pipeline列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PipelineListItem
	for rows.Next() {
		item, err := scanPipelineListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdatePipelineStatus 更新Pipeline状态和当前步骤
func UpdatePipelineStatus(id string, currentStep string, status string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines SET current_step = $2, status = $3, updated_at = NOW()
		 WHERE id = $1`, id, currentStep, status)
	if err != nil {
		return fmt.Errorf("更新Pipeline状态失败: %w", err)
	}
	return nil
}

// UpdatePipelineError 更新Pipeline错误信息并标记为失败
func UpdatePipelineError(id string, currentStep string, errMsg string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines SET current_step = $2, status = $3, error_message = $4, updated_at = NOW()
		 WHERE id = $1`, id, currentStep, models.PipelineStatusFailed, errMsg)
	if err != nil {
		return fmt.Errorf("更新Pipeline错误失败: %w", err)
	}
	return nil
}

// CompletePipeline 标记Pipeline完成（进入审核队列或定稿）
func CompletePipeline(id string, status string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines SET status = $2, completed_at = NOW(), updated_at = NOW()
		 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("完成Pipeline失败: %w", err)
	}
	return nil
}

// DeletePipeline 删除Pipeline（级联删除步骤）
func DeletePipeline(id string) error {
	ctx := context.Background()
	result, err := database.DB.Exec(ctx,
		`DELETE FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除Pipeline失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrPipelineNotFound
	}
	return nil
}

// CheckActivePipelineExists 检查某课程是否已有运行中的Pipeline
func CheckActivePipelineExists(courseCode string) (bool, string, error) {
	ctx := context.Background()
	var id string
	err := database.DB.QueryRow(ctx,
		`SELECT id FROM pipelines
		 WHERE course_code = $1 AND status IN ($2, $3)
		 LIMIT 1`,
		courseCode, models.PipelineStatusPending, models.PipelineStatusRunning,
	).Scan(&id)
	if err != nil {
		return false, "", nil
	}
	return true, id, nil
}

// ==================== 2审辅助方法 ====================

// UpdatePipelineReviewRound 更新Pipeline审核轮次
// 验收失败后将review_round从1设为2，标记进入2审流程
func UpdatePipelineReviewRound(id string, reviewRound int) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines SET review_round = $2, updated_at = NOW()
		 WHERE id = $1`, id, reviewRound)
	if err != nil {
		return fmt.Errorf("更新Pipeline审核轮次失败: %w", err)
	}
	return nil
}

// ResetStepsForRetrial 批量重置指定步骤为pending（清除旧数据，事务执行）
// 2审流程需要重置evaluator/meta/translator/generator/review步骤
func ResetStepsForRetrial(pipelineID string, stepNames []string) error {
	// v54重构：不再重置旧步骤，而是为新轮次插入全新的步骤记录
	// 历史数据（旧review_round的步骤）保留，支持完整回溯
	ctx := context.Background()

	// 读取新的review_round（已被UpdatePipelineReviewRound更新）
	var reviewRound int
	if err := database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound); err != nil {
		return fmt.Errorf("读取review_round失败: %w", err)
	}

	// 读取步骤的 step_order 映射（从旧记录中取）
	orderRows, err := database.DB.Query(ctx,
		`SELECT DISTINCT ON (step_name) step_name, step_order
		 FROM pipeline_steps
		 WHERE pipeline_id = $1
		 ORDER BY step_name, review_round ASC`, pipelineID)
	if err != nil {
		return fmt.Errorf("查询步骤order失败: %w", err)
	}
	defer orderRows.Close()
	orderMap := make(map[string]int)
	for orderRows.Next() {
		var name string
		var order int
		if err := orderRows.Scan(&name, &order); err == nil {
			orderMap[name] = order
		}
	}

	// 为每个需要重置的步骤插入新轮次记录
	for _, stepName := range stepNames {
		order := orderMap[stepName]
		_, err := database.DB.Exec(ctx,
			`INSERT INTO pipeline_steps
			    (pipeline_id, step_name, step_order, status, review_round)
			 VALUES ($1, $2, $3, 'pending', $4)
			 ON CONFLICT (pipeline_id, step_name, review_round) DO UPDATE
			    SET status = 'pending', started_at = NULL, completed_at = NULL,
			        duration_ms = 0, attempts = 0, step_data = NULL,
			        error_message = NULL, tokens_used = 0, updated_at = NOW()`,
			pipelineID, stepName, order, reviewRound)
		if err != nil {
			return fmt.Errorf("插入新轮次步骤%s失败: %w", stepName, err)
		}
	}
	return nil
}


func UpdatePipelineRejectReason(id string, rejectReason string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines
		 SET current_step = 'review',
		     status = $2,
		     reject_reason = $3,
		     updated_at = NOW()
		 WHERE id = $1`,
		id, models.PipelineStatusReviewQueue, rejectReason)
	if err != nil {
		return fmt.Errorf("更新退回原因失败: %w", err)
	}
	return nil
}

// ==================== 教案关联查询 ====================

// GetPipelineByLessonPlanID 根据教案ID查询关联的Pipeline（Phase6新增）
func GetPipelineByLessonPlanID(lessonPlanID string) (*models.Pipeline, error) {
	ctx := context.Background()
	var id string
	err := database.DB.QueryRow(ctx,
		`SELECT id FROM pipelines WHERE lesson_plan_id = $1 ORDER BY created_at DESC LIMIT 1`,
		lessonPlanID,
	).Scan(&id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	return GetPipelineByID(id)
}

// ==================== 工具函数 ====================

// buildPlaceholders 构建SQL占位符字符串，如"$1,$2,$3"
// 备用函数，当前BatchAssignPipelines已改用ANY
func buildPlaceholders(count int, startIdx int) string {
	if count == 0 {
		return ""
	}
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = fmt.Sprintf("$%d", startIdx+i)
	}
	return strings.Join(parts, ",")
}

// 抑制buildPlaceholders未使用警告
var _ = buildPlaceholders
