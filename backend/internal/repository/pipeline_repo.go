package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== Pipeline 错误常量 ====================

var (
	ErrPipelineNotFound = errors.New("Pipeline不存在")
	ErrStepNotFound     = errors.New("Pipeline步骤不存在")
)

// ==================== Dashboard 统计（P4.5-D新增）====================

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	// 课程统计
	TotalCourses     int `json:"total_courses"`
	CoursesWithIndex int `json:"courses_with_index"`
	// Pipeline统计
	TotalPipelines   int `json:"total_pipelines"`
	RunningPipelines int `json:"running_pipelines"`
	ReviewQueue      int `json:"review_queue"`
	Finalized        int `json:"finalized"`
	Failed           int `json:"failed"`
	// 达标统计（evaluator均分≥9.0）
	PassedCount int `json:"passed_count"`
	// AI消耗统计
	TotalTokensUsed int64 `json:"total_tokens_used"`
}

// GetDashboardStats 获取仪表盘统计数据
// P4.5-D新增：一次查询获取所有统计数据，避免多次查询
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
	err = database.DB.QueryRow(ctx,
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status = 'review_queue'),
			COUNT(*) FILTER (WHERE status = 'finalized'),
			COUNT(*) FILTER (WHERE status = 'failed'),
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM pipeline_steps ps
				WHERE ps.pipeline_id = p.id AND ps.step_name = 'evaluator'
					AND ps.status = 'done' AND ps.step_data IS NOT NULL
					AND (ps.step_data->>'avg_total')::numeric >= 9.0
			))
		 FROM pipelines p`).Scan(
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

	// 3. AI token消耗统计
	err = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(tokens_used), 0) FROM pipeline_steps WHERE tokens_used > 0`).Scan(&stats.TotalTokensUsed)
	if err != nil {
		// token统计非关键，失败不影响其他
		stats.TotalTokensUsed = 0
	}

	return stats, nil
}

// ==================== Pipeline CRUD ====================

// CreatePipeline 创建Pipeline主记录
// 同时批量创建8个步骤记录（在事务中执行）
// P4.6更新：从7步扩展为8步（自动跟随StepDefinitions）
func CreatePipeline(p *models.Pipeline) error {
	ctx := context.Background()

	// 开启事务：Pipeline和步骤必须原子创建
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. 插入Pipeline主记录
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

	// 2. 批量插入步骤记录（全部初始状态为pending，数量由StepDefinitions决定）
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

	// 3. 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetPipelineByID 根据ID获取Pipeline主记录
// P4.6更新：读取review_round字段
func GetPipelineByID(id string) (*models.Pipeline, error) {
	ctx := context.Background()
	p := &models.Pipeline{}

	// config字段是JSONB，需要用*string接收可能的NULL值
	var configStr *string
	var errorMsg *string
	var courseName *string

	err := database.DB.QueryRow(ctx,
		`SELECT id, course_code, course_name, external_module_id, started_by,
			started_at, completed_at, current_step, status, auto_mode,
			error_message, config::text, review_round, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id).Scan(
		&p.ID, &p.CourseCode, &courseName, &p.ExternalModuleID, &p.StartedBy,
		&p.StartedAt, &p.CompletedAt, &p.CurrentStep, &p.Status, &p.AutoMode,
		&errorMsg, &configStr, &p.ReviewRound, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// 处理可能为NULL的字段
	if courseName != nil {
		p.CourseName = *courseName
	}
	if errorMsg != nil {
		p.ErrorMessage = *errorMsg
	}
	if configStr != nil {
		p.Config = *configStr
	}

	return p, nil
}

// pipelineListSelectSQL P4.5-A: Pipeline列表查询的SELECT子句（含3个分数子查询）
// 从pipeline_steps.step_data JSONB字段提取evaluator均分/meta仲裁分/translator最终分
// P4.6更新：新增review_round字段
// 注意：JSONB提取用->>返回text，需要::numeric转换为数值类型
const pipelineListSelectSQL = `SELECT p.id, p.course_code, p.course_name, p.external_module_id,
	p.current_step, p.status, p.auto_mode, p.error_message,
	p.started_by, p.started_at, p.completed_at, p.created_at,
	p.review_round,
	COALESCE((SELECT COUNT(*) FROM pipeline_steps ps
		WHERE ps.pipeline_id = p.id AND ps.status = 'done'), 0) AS steps_completed,
	(SELECT (ps_eval.step_data->>'avg_total')::numeric
	 FROM pipeline_steps ps_eval
	 WHERE ps_eval.pipeline_id = p.id AND ps_eval.step_name = 'evaluator'
		AND ps_eval.status = 'done' AND ps_eval.step_data IS NOT NULL
	 LIMIT 1) AS eval_avg_score,
	(SELECT (ps_meta.step_data->>'total_final')::numeric
	 FROM pipeline_steps ps_meta
	 WHERE ps_meta.pipeline_id = p.id AND ps_meta.step_name = 'meta'
		AND ps_meta.status = 'done' AND ps_meta.step_data IS NOT NULL
	 LIMIT 1) AS meta_score,
	(SELECT (ps_trans.step_data->>'final_score')::numeric
	 FROM pipeline_steps ps_trans
	 WHERE ps_trans.pipeline_id = p.id AND ps_trans.step_name = 'translator'
		AND ps_trans.status = 'done' AND ps_trans.step_data IS NOT NULL
	 LIMIT 1) AS translator_score
FROM pipelines p`

// scanPipelineListRow 扫描Pipeline列表行（含3个分数字段+review_round）
// P4.5-A: 统一扫描逻辑，ListPipelines和ListPipelinesForUser共用
// P4.6更新：新增review_round字段扫描
func scanPipelineListRow(rows interface{ Scan(dest ...interface{}) error }) (*models.PipelineListItem, error) {
	item := &models.PipelineListItem{}
	var courseName, errorMsg *string

	err := rows.Scan(
		&item.ID, &item.CourseCode, &courseName, &item.ExternalModuleID,
		&item.CurrentStep, &item.Status, &item.AutoMode, &errorMsg,
		&item.StartedBy, &item.StartedAt, &item.CompletedAt, &item.CreatedAt,
		&item.ReviewRound,
		&item.StepsCompleted,
		&item.EvalAvgScore,    // P4.5-A: evaluator均分（*float64，NULL自动映射为nil）
		&item.MetaScore,       // P4.5-A: meta仲裁分
		&item.TranslatorScore, // P4.5-A: translator最终分
	)
	if err != nil {
		return nil, fmt.Errorf("扫描Pipeline行失败: %w", err)
	}

	// 处理NULL字段
	if courseName != nil {
		item.CourseName = *courseName
	}
	if errorMsg != nil {
		item.ErrorMessage = *errorMsg
	}

	// 附加中文名
	item.CurrentStepName = models.StepNameMap[item.CurrentStep]
	item.StatusName = models.PipelineStatusNameMap[item.Status]
	item.StepsTotal = models.TotalSteps

	return item, nil
}

// ListPipelines 获取Pipeline列表（按创建时间倒序）
// P4.5-A增强：含每个Pipeline的evaluator均分、meta仲裁分、translator最终分
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
// admin看所有，非admin只看自己发起的或分配课程的Pipeline
// P4.5-A增强：含3个分数字段
func ListPipelinesForUser(userID string, role string) ([]*models.PipelineListItem, error) {
	ctx := context.Background()
	var query string
	var args []interface{}

	if role == "admin" {
		// admin看所有Pipeline
		query = pipelineListSelectSQL + " ORDER BY p.created_at DESC"
	} else {
		// 非admin：看自己发起的 + 分配给自己课程的Pipeline
		query = pipelineListSelectSQL + `
		 WHERE p.started_by = $1
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
// 用于防止同一课程重复创建Pipeline
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
		// 没找到活跃Pipeline → 可以创建
		return false, "", nil
	}
	return true, id, nil
}

// ==================== P4.6-3 新增：2审辅助方法 ====================

// UpdatePipelineReviewRound 更新Pipeline的审核轮次（review_round字段）
// P4.6-3新增：验收失败后将review_round从1设为2，标记进入2审流程
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

// ResetStepsForRetrial 批量重置指定步骤为pending状态（清除旧数据）
// P4.6-3新增：2审流程需要重置evaluator/meta/translator/generator/review步骤
// 在事务中执行：重置步骤状态+清空输出数据+重置时间和错误信息
func ResetStepsForRetrial(pipelineID string, stepNames []string) error {
	ctx := context.Background()
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, stepName := range stepNames {
		_, err = tx.Exec(ctx,
			`UPDATE pipeline_steps
			 SET status = $3, started_at = NULL, completed_at = NULL,
				 duration_ms = 0, attempts = 0, step_data = NULL,
				 error_message = NULL, model_used = NULL, tokens_used = 0,
				 updated_at = NOW()
			 WHERE pipeline_id = $1 AND step_name = $2`,
			pipelineID, stepName, models.StepStatusPending,
		)
		if err != nil {
			return fmt.Errorf("重置步骤 %s 失败: %w", stepName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交重置事务失败: %w", err)
	}
	return nil
}

// ==================== Pipeline Step CRUD ====================

// GetStepsByPipelineID 获取指定Pipeline的所有步骤（按步骤序号排序）
func GetStepsByPipelineID(pipelineID string) ([]*models.PipelineStep, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, step_name, step_order, status,
			started_at, completed_at, duration_ms, attempts,
			step_data::text, error_message, model_used, tokens_used,
			created_at, updated_at
		 FROM pipeline_steps
		 WHERE pipeline_id = $1
		 ORDER BY step_order ASC`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("查询Pipeline步骤失败: %w", err)
	}
	defer rows.Close()

	var steps []*models.PipelineStep
	for rows.Next() {
		s := &models.PipelineStep{}
		var stepData, errorMsg, modelUsed *string

		err := rows.Scan(
			&s.ID, &s.PipelineID, &s.StepName, &s.StepOrder, &s.Status,
			&s.StartedAt, &s.CompletedAt, &s.DurationMs, &s.Attempts,
			&stepData, &errorMsg, &modelUsed, &s.TokensUsed,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描步骤行失败: %w", err)
		}

		if stepData != nil {
			s.StepData = *stepData
		}
		if errorMsg != nil {
			s.ErrorMessage = *errorMsg
		}
		if modelUsed != nil {
			s.ModelUsed = *modelUsed
		}

		steps = append(steps, s)
	}
	return steps, nil
}

// GetStepByName 获取指定Pipeline的指定步骤
func GetStepByName(pipelineID string, stepName string) (*models.PipelineStep, error) {
	ctx := context.Background()
	s := &models.PipelineStep{}
	var stepData, errorMsg, modelUsed *string

	err := database.DB.QueryRow(ctx,
		`SELECT id, pipeline_id, step_name, step_order, status,
			started_at, completed_at, duration_ms, attempts,
			step_data::text, error_message, model_used, tokens_used,
			created_at, updated_at
		 FROM pipeline_steps
		 WHERE pipeline_id = $1 AND step_name = $2`, pipelineID, stepName).Scan(
		&s.ID, &s.PipelineID, &s.StepName, &s.StepOrder, &s.Status,
		&s.StartedAt, &s.CompletedAt, &s.DurationMs, &s.Attempts,
		&stepData, &errorMsg, &modelUsed, &s.TokensUsed,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, ErrStepNotFound
	}

	if stepData != nil {
		s.StepData = *stepData
	}
	if errorMsg != nil {
		s.ErrorMessage = *errorMsg
	}
	if modelUsed != nil {
		s.ModelUsed = *modelUsed
	}

	return s, nil
}

// StartStep 标记步骤开始执行（更新状态+开始时间+尝试次数）
func StartStep(pipelineID string, stepName string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
		 SET status = $3, started_at = NOW(), attempts = attempts + 1, updated_at = NOW()
		 WHERE pipeline_id = $1 AND step_name = $2`,
		pipelineID, stepName, models.StepStatusRunning)
	if err != nil {
		return fmt.Errorf("启动步骤 %s 失败: %w", stepName, err)
	}
	return nil
}

// CompleteStep 标记步骤完成（更新状态+完成时间+耗时+输出数据）
func CompleteStep(pipelineID string, stepName string, durationMs int64, stepData string, modelUsed string, tokensUsed int) error {
	ctx := context.Background()

	// step_data可能为空JSON或有效JSON，都用$4::jsonb存储
	var stepDataParam interface{}
	if stepData != "" && stepData != "{}" {
		stepDataParam = stepData
	}

	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
		 SET status = $3, completed_at = NOW(), duration_ms = $4,
			 step_data = $5::jsonb, model_used = $6, tokens_used = $7, updated_at = NOW()
		 WHERE pipeline_id = $1 AND step_name = $2`,
		pipelineID, stepName, models.StepStatusDone, durationMs,
		stepDataParam, modelUsed, tokensUsed)
	if err != nil {
		return fmt.Errorf("完成步骤 %s 失败: %w", stepName, err)
	}
	return nil
}

// FailStep 标记步骤失败（更新状态+完成时间+耗时+错误信息）
func FailStep(pipelineID string, stepName string, durationMs int64, errMsg string) error {
	ctx := context.Background()
	now := time.Now()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
		 SET status = $3, completed_at = $4, duration_ms = $5,
			 error_message = $6, updated_at = $4
		 WHERE pipeline_id = $1 AND step_name = $2`,
		pipelineID, stepName, models.StepStatusFailed, now, durationMs, errMsg)
	if err != nil {
		return fmt.Errorf("标记步骤 %s 失败: %w", stepName, err)
	}
	return nil
}

// ==================== Eval Rounds CRUD（P4-3新增）====================

// CreateEvalRound 创建评估轮次记录
func CreateEvalRound(pipelineID string, roundNumber int) (*models.EvalRoundRecord, error) {
	ctx := context.Background()
	r := &models.EvalRoundRecord{}
	err := database.DB.QueryRow(ctx,
		`INSERT INTO eval_rounds (pipeline_id, round_number, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, pipeline_id, round_number, status, created_at`,
		pipelineID, roundNumber, models.StepStatusPending,
	).Scan(&r.ID, &r.PipelineID, &r.RoundNumber, &r.Status, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建评估轮次R%d失败: %w", roundNumber, err)
	}
	return r, nil
}

// UpdateEvalRoundRunning 标记评估轮次为运行中
func UpdateEvalRoundRunning(roundID string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE eval_rounds SET status = $2 WHERE id = $1`,
		roundID, models.StepStatusRunning)
	if err != nil {
		return fmt.Errorf("更新评估轮次状态失败: %w", err)
	}
	return nil
}

// CompleteEvalRound 完成评估轮次（写入评分和输出）
func CompleteEvalRound(roundID string, output string, scoreTotal float64,
	scoreE1 float64, scoreE2 float64, scoreE3 float64, scoreE4 float64,
	dimensions string, modelUsed string, tokensUsed int) error {
	ctx := context.Background()

	// dimensions是JSON字符串，用$8::jsonb存入JSONB字段
	var dimParam interface{}
	if dimensions != "" && dimensions != "{}" {
		dimParam = dimensions
	}

	_, err := database.DB.Exec(ctx,
		`UPDATE eval_rounds
		 SET status = $2, output = $3, score_total = $4,
			 score_e1 = $5, score_e2 = $6, score_e3 = $7, score_e4 = $8,
			 dimensions = $9::jsonb, model_used = $10, tokens_used = $11
		 WHERE id = $1`,
		roundID, models.StepStatusDone, output, scoreTotal,
		scoreE1, scoreE2, scoreE3, scoreE4,
		dimParam, modelUsed, tokensUsed)
	if err != nil {
		return fmt.Errorf("完成评估轮次失败: %w", err)
	}
	return nil
}

// FailEvalRound 标记评估轮次失败
func FailEvalRound(roundID string, output string, errMsg string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE eval_rounds SET status = $2, output = $3, dimensions = $4::jsonb WHERE id = $1`,
		roundID, models.StepStatusFailed, output,
		fmt.Sprintf(`{"error":"%s"}`, errMsg))
	if err != nil {
		return fmt.Errorf("标记评估轮次失败: %w", err)
	}
	return nil
}

// GetEvalRoundsByPipelineID 获取指定Pipeline的所有评估轮次（按轮次序号排序）
func GetEvalRoundsByPipelineID(pipelineID string) ([]*models.EvalRoundRecord, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, round_number, status, output,
			score_total, score_e1, score_e2, score_e3, score_e4,
			dimensions::text, model_used, tokens_used, created_at
		 FROM eval_rounds
		 WHERE pipeline_id = $1
		 ORDER BY round_number ASC`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("查询评估轮次失败: %w", err)
	}
	defer rows.Close()

	var rounds []*models.EvalRoundRecord
	for rows.Next() {
		r := &models.EvalRoundRecord{}
		var output, dimensions, modelUsed *string
		err := rows.Scan(
			&r.ID, &r.PipelineID, &r.RoundNumber, &r.Status, &output,
			&r.ScoreTotal, &r.ScoreE1, &r.ScoreE2, &r.ScoreE3, &r.ScoreE4,
			&dimensions, &modelUsed, &r.TokensUsed, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描评估轮次行失败: %w", err)
		}
		if output != nil {
			r.Output = *output
		}
		if dimensions != nil {
			r.Dimensions = *dimensions
		}
		if modelUsed != nil {
			r.ModelUsed = *modelUsed
		}
		rounds = append(rounds, r)
	}
	return rounds, nil
}

// DeleteEvalRoundsByPipelineID 删除指定Pipeline的所有评估轮次（重跑时清理旧数据）
func DeleteEvalRoundsByPipelineID(pipelineID string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`DELETE FROM eval_rounds WHERE pipeline_id = $1`, pipelineID)
	if err != nil {
		return fmt.Errorf("删除评估轮次失败: %w", err)
	}
	return nil
}

// ==================== Generated Pages CRUD（P4-6新增）====================

// CreateGeneratedPage 创建生成页面记录
// P4.5-E更新：新增changeReason参数，存储Translator给出的修改理由和指令
func CreateGeneratedPage(pipelineID string, pageNumber int, pageTitle string,
	operation string, originalHTML string, generatedHTML string, finalHTML string,
	lessonID *int, mergeSources string, changeReason string) error {
	ctx := context.Background()

	var mergeParam interface{}
	if mergeSources != "" && mergeSources != "null" {
		mergeParam = mergeSources
	}

	_, err := database.DB.Exec(ctx,
		`INSERT INTO generated_pages (pipeline_id, page_number, page_title,
			operation, original_html, generated_html, final_html,
			decision, lesson_id, merge_sources, change_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9::jsonb, $10)`,
		pipelineID, pageNumber, pageTitle,
		operation, originalHTML, generatedHTML, finalHTML,
		lessonID, mergeParam, changeReason)
	if err != nil {
		return fmt.Errorf("创建生成页面P%d失败: %w", pageNumber, err)
	}
	return nil
}

// GetGeneratedPagesByPipelineID 获取指定Pipeline的所有生成页面（按页码排序，不含完整HTML，只含长度）
func GetGeneratedPagesByPipelineID(pipelineID string) ([]*GeneratedPageRow, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			LENGTH(COALESCE(original_html,'')) as orig_len,
			LENGTH(COALESCE(generated_html,'')) as gen_len,
			LENGTH(COALESCE(final_html,'')) as final_len,
			decision, lesson_id, merge_sources::text,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1
		 ORDER BY page_number ASC`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("查询生成页面失败: %w", err)
	}
	defer rows.Close()

	var pages []*GeneratedPageRow
	for rows.Next() {
		p := &GeneratedPageRow{}
		var pageTitle, decision, mergeSources *string
		var lessonID *int
		err := rows.Scan(
			&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
			&p.OrigLen, &p.GenLen, &p.FinalLen,
			&decision, &lessonID, &mergeSources,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描生成页面行失败: %w", err)
		}
		if pageTitle != nil {
			p.PageTitle = *pageTitle
		}
		if decision != nil {
			p.Decision = *decision
		}
		if lessonID != nil {
			p.LessonID = lessonID
		}
		if mergeSources != nil {
			p.MergeSources = *mergeSources
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// GetGeneratedPagesWithHTML 获取指定Pipeline的所有生成页面（含完整HTML内容+修改理由）
// P4.5-C新增：审核页面需要展示完整HTML预览和对比
// P4.5-E更新：返回change_reason字段供审核页面展示修改意图
func GetGeneratedPagesWithHTML(pipelineID string) ([]*GeneratedPageFullRow, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			COALESCE(original_html, '') as original_html,
			COALESCE(generated_html, '') as generated_html,
			COALESCE(final_html, '') as final_html,
			decision, lesson_id, merge_sources::text,
			COALESCE(change_reason, '') as change_reason,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1
		 ORDER BY page_number ASC`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("查询生成页面（含HTML）失败: %w", err)
	}
	defer rows.Close()

	var pages []*GeneratedPageFullRow
	for rows.Next() {
		p := &GeneratedPageFullRow{}
		var pageTitle, decision, mergeSources *string
		var lessonID *int
		err := rows.Scan(
			&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
			&p.OriginalHTML, &p.GeneratedHTML, &p.FinalHTML,
			&decision, &lessonID, &mergeSources,
			&p.ChangeReason,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描生成页面行（含HTML）失败: %w", err)
		}
		if pageTitle != nil {
			p.PageTitle = *pageTitle
		}
		if decision != nil {
			p.Decision = *decision
		}
		if lessonID != nil {
			p.LessonID = lessonID
		}
		if mergeSources != nil {
			p.MergeSources = *mergeSources
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// UpdatePageDecision 更新单页审核决策
// P4.5-C新增：支持approve/reject/edit决策，edit时同时更新final_html
func UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	ctx := context.Background()

	if finalHTML != nil {
		// 带HTML更新（edit模式：审核员手动修改了HTML内容）
		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, final_html = $4, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2`,
			pipelineID, pageNumber, decision, *finalHTML)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策（含HTML）失败: %w", pageNumber, err)
		}
	} else {
		// 不带HTML更新（approve/reject模式）
		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2`,
			pipelineID, pageNumber, decision)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策失败: %w", pageNumber, err)
		}
	}
	return nil
}

// GetPageDecisionStats 获取Pipeline页面审核决策统计
// P4.5-C新增：用于判断是否所有页面都已决策（支持定稿检查）
func GetPageDecisionStats(pipelineID string) (total int, decided int, err error) {
	ctx := context.Background()
	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*),
			COUNT(*) FILTER (WHERE decision IN ('approve', 'reject', 'edit'))
		 FROM generated_pages WHERE pipeline_id = $1`, pipelineID).Scan(&total, &decided)
	if err != nil {
		return 0, 0, fmt.Errorf("查询页面决策统计失败: %w", err)
	}
	return total, decided, nil
}

// DeleteGeneratedPagesByPipelineID 删除指定Pipeline的所有生成页面（重跑时清理）
func DeleteGeneratedPagesByPipelineID(pipelineID string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`DELETE FROM generated_pages WHERE pipeline_id = $1`, pipelineID)
	if err != nil {
		return fmt.Errorf("删除生成页面失败: %w", err)
	}
	return nil
}

// GeneratedPageRow 生成页面查询行（不含完整HTML，只含长度）
type GeneratedPageRow struct {
	ID           string     `json:"id"`
	PipelineID   string     `json:"pipeline_id"`
	PageNumber   int        `json:"page_number"`
	PageTitle    string     `json:"page_title"`
	Operation    string     `json:"operation"`
	OrigLen      int        `json:"orig_len"`
	GenLen       int        `json:"gen_len"`
	FinalLen     int        `json:"final_len"`
	Decision     string     `json:"decision"`
	LessonID     *int       `json:"lesson_id"`
	MergeSources string     `json:"merge_sources"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// GeneratedPageFullRow 生成页面查询行（含完整HTML内容+修改理由）
// P4.5-C新增：审核页面需要完整HTML用于预览和对比
// P4.5-E更新：新增ChangeReason字段，存储Translator给出的修改理由
type GeneratedPageFullRow struct {
	ID            string     `json:"id"`
	PipelineID    string     `json:"pipeline_id"`
	PageNumber    int        `json:"page_number"`
	PageTitle     string     `json:"page_title"`
	Operation     string     `json:"operation"`
	OriginalHTML  string     `json:"original_html"`
	GeneratedHTML string     `json:"generated_html"`
	FinalHTML     string     `json:"final_html"`
	Decision      string     `json:"decision"`
	LessonID      *int       `json:"lesson_id"`
	MergeSources  string     `json:"merge_sources"`
	ChangeReason  string     `json:"change_reason"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// UpdateGeneratedPageHTML 更新指定页面的generated_html和final_html
// P4.5-E-2新增：AI快修功能，审核员输入修改指令后AI重新生成HTML，需要更新数据库
func UpdateGeneratedPageHTML(pipelineID string, pageNumber int, generatedHTML string, finalHTML string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE generated_pages
		 SET generated_html = $3, final_html = $4, updated_at = NOW()
		 WHERE pipeline_id = $1 AND page_number = $2`,
		pipelineID, pageNumber, generatedHTML, finalHTML)
	if err != nil {
		return fmt.Errorf("更新页面P%d的HTML失败: %w", pageNumber, err)
	}
	return nil
}

// ==================== P4.6-4 夜间批量验收辅助方法 ====================

// ListFinalizedPipelineIDs 获取所有finalized状态的Pipeline ID列表
// P4.6-4新增：夜间批量验收和手动批量验收使用
// 返回按创建时间正序排列的ID列表（先创建的先验收）
func ListFinalizedPipelineIDs() ([]string, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id FROM pipelines WHERE status = $1 ORDER BY created_at ASC`,
		models.PipelineStatusFinalized)
	if err != nil {
		return nil, fmt.Errorf("查询finalized Pipeline列表失败: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描Pipeline ID失败: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
