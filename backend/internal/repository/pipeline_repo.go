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

// ==================== Pipeline CRUD ====================

// CreatePipeline 创建Pipeline主记录
// 同时批量创建7个步骤记录（在事务中执行）
func CreatePipeline(p *models.Pipeline) error {
	ctx := context.Background()

	// 开启事务：Pipeline和7个步骤必须原子创建
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. 插入Pipeline主记录
	err = tx.QueryRow(ctx,
		`INSERT INTO pipelines (course_code, course_name, external_module_id, started_by,
			current_step, status, auto_mode, config)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		 RETURNING id, started_at, created_at, updated_at`,
		p.CourseCode, p.CourseName, p.ExternalModuleID, p.StartedBy,
		p.CurrentStep, p.Status, p.AutoMode, p.Config,
	).Scan(&p.ID, &p.StartedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("插入Pipeline失败: %w", err)
	}

	// 2. 批量插入7个步骤记录（全部初始状态为pending）
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
			error_message, config::text, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id).Scan(
		&p.ID, &p.CourseCode, &courseName, &p.ExternalModuleID, &p.StartedBy,
		&p.StartedAt, &p.CompletedAt, &p.CurrentStep, &p.Status, &p.AutoMode,
		&errorMsg, &configStr, &p.CreatedAt, &p.UpdatedAt,
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

// ListPipelines 获取Pipeline列表（按创建时间倒序）
// 含每个Pipeline已完成步骤数的统计
func ListPipelines() ([]*models.PipelineListItem, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT p.id, p.course_code, p.course_name, p.external_module_id,
			p.current_step, p.status, p.auto_mode, p.error_message,
			p.started_by, p.started_at, p.completed_at, p.created_at,
			COALESCE((SELECT COUNT(*) FROM pipeline_steps ps
				WHERE ps.pipeline_id = p.id AND ps.status = 'done'), 0) AS steps_completed
		 FROM pipelines p
		 ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询Pipeline列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PipelineListItem
	for rows.Next() {
		item := &models.PipelineListItem{}
		var courseName, errorMsg *string

		err := rows.Scan(
			&item.ID, &item.CourseCode, &courseName, &item.ExternalModuleID,
			&item.CurrentStep, &item.Status, &item.AutoMode, &errorMsg,
			&item.StartedBy, &item.StartedAt, &item.CompletedAt, &item.CreatedAt,
			&item.StepsCompleted,
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

		items = append(items, item)
	}
	return items, nil
}

// ListPipelinesForUser 获取指定用户可见的Pipeline列表
// admin看所有，非admin只看自己发起的或分配课程的Pipeline
func ListPipelinesForUser(userID string, role string) ([]*models.PipelineListItem, error) {
	ctx := context.Background()
	var query string
	var args []interface{}

	if role == "admin" {
		// admin看所有Pipeline
		query = `SELECT p.id, p.course_code, p.course_name, p.external_module_id,
				p.current_step, p.status, p.auto_mode, p.error_message,
				p.started_by, p.started_at, p.completed_at, p.created_at,
				COALESCE((SELECT COUNT(*) FROM pipeline_steps ps
					WHERE ps.pipeline_id = p.id AND ps.status = 'done'), 0) AS steps_completed
			 FROM pipelines p
			 ORDER BY p.created_at DESC`
	} else {
		// 非admin：看自己发起的 + 分配给自己课程的Pipeline
		query = `SELECT p.id, p.course_code, p.course_name, p.external_module_id,
				p.current_step, p.status, p.auto_mode, p.error_message,
				p.started_by, p.started_at, p.completed_at, p.created_at,
				COALESCE((SELECT COUNT(*) FROM pipeline_steps ps
					WHERE ps.pipeline_id = p.id AND ps.status = 'done'), 0) AS steps_completed
			 FROM pipelines p
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
		item := &models.PipelineListItem{}
		var courseName, errorMsg *string

		err := rows.Scan(
			&item.ID, &item.CourseCode, &courseName, &item.ExternalModuleID,
			&item.CurrentStep, &item.Status, &item.AutoMode, &errorMsg,
			&item.StartedBy, &item.StartedAt, &item.CompletedAt, &item.CreatedAt,
			&item.StepsCompleted,
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
		item.CurrentStepName = models.StepNameMap[item.CurrentStep]
		item.StatusName = models.PipelineStatusNameMap[item.Status]
		item.StepsTotal = models.TotalSteps

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
func CreateGeneratedPage(pipelineID string, pageNumber int, pageTitle string,
	operation string, originalHTML string, generatedHTML string, finalHTML string,
	lessonID *int, mergeSources string) error {
	ctx := context.Background()

	var mergeParam interface{}
	if mergeSources != "" && mergeSources != "null" {
		mergeParam = mergeSources
	}

	_, err := database.DB.Exec(ctx,
		`INSERT INTO generated_pages (pipeline_id, page_number, page_title,
			operation, original_html, generated_html, final_html,
			decision, lesson_id, merge_sources)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9::jsonb)`,
		pipelineID, pageNumber, pageTitle,
		operation, originalHTML, generatedHTML, finalHTML,
		lessonID, mergeParam)
	if err != nil {
		return fmt.Errorf("创建生成页面P%d失败: %w", pageNumber, err)
	}
	return nil
}

// GetGeneratedPagesByPipelineID 获取指定Pipeline的所有生成页面（按页码排序）
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
