package repository

// pipeline_repo_steps.go — Pipeline步骤 + 评估轮次数据访问层
//
// 职责：
//   - PipelineStep CRUD（查询/启动/完成/失败）
//   - EvalRound CRUD（创建/更新/完成/失败/查询/删除）
//   - 历史轮次查询（回溯用）
//   - 跨轮次步骤数据查询（二审注入一审上下文用）
//
// v68变更：新增 GetStepByNameAndRound 方法，支持按指定 review_round 查询单个步骤数据，
//          用于二审时读取一审的 meta/translator/reviewer 步骤 step_data 作为上下文注入

import (
	"context"
	"encoding/json"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== Pipeline Step CRUD ====================

// GetStepsByPipelineID 获取指定Pipeline的所有步骤（按步骤序号排序）
// 只返回当前review_round的步骤数据
func GetStepsByPipelineID(pipelineID string) ([]*models.PipelineStep, error) {
	ctx := context.Background()
	// 读取当前最新review_round
	var curRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&curRound)
	if curRound == 0 {
		curRound = 1
	}

	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, step_name, step_order, status,
                        started_at, completed_at, duration_ms, attempts,
                        step_data::text, error_message, model_used, tokens_used,
                        created_at, updated_at
                 FROM pipeline_steps
                 WHERE pipeline_id = $1 AND review_round = $2
                 ORDER BY step_order ASC`, pipelineID, curRound)
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
// 按review_round降序取最新的一条
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
                 WHERE pipeline_id = $1 AND step_name = $2
                 ORDER BY review_round DESC LIMIT 1`, pipelineID, stepName).Scan(
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

// GetStepByNameAndRound 获取指定Pipeline、指定步骤名、指定review_round的步骤数据
// v68新增：用于二审时精确读取一审（review_round=1）的特定步骤step_data
// 典型场景：二审evaluator/meta需要注入一审meta的修改方案和translator的翻译结果作为上下文
// 参数：
//   - pipelineID: Pipeline唯一标识
//   - stepName: 步骤名称（如 "meta"、"translator"）
//   - round: 目标review_round（如 1 表示一审）
//
// 返回：
//   - *models.PipelineStep: 步骤数据（含step_data）
//   - error: 未找到时返回 ErrStepNotFound
func GetStepByNameAndRound(pipelineID string, stepName string, round int) (*models.PipelineStep, error) {
	ctx := context.Background()
	s := &models.PipelineStep{}
	var stepData, errorMsg, modelUsed *string

	err := database.DB.QueryRow(ctx,
		`SELECT id, pipeline_id, step_name, step_order, status,
                        started_at, completed_at, duration_ms, attempts,
                        step_data::text, error_message, model_used, tokens_used,
                        created_at, updated_at
                 FROM pipeline_steps
                 WHERE pipeline_id = $1 AND step_name = $2 AND review_round = $3
                 LIMIT 1`, pipelineID, stepName, round).Scan(
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
// 委托给StartStepForRound，round=0自动读取当前review_round
func StartStep(pipelineID string, stepName string) error {
	return StartStepForRound(pipelineID, stepName, 0)
}

// StartStepForRound 启动指定轮次的步骤，round=0时自动从pipelines表读取当前review_round
func StartStepForRound(pipelineID string, stepName string, round int) error {
	ctx := context.Background()

	if round == 0 {
		var reviewRound int
		_ = database.DB.QueryRow(ctx,
			`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
		).Scan(&reviewRound)
		if reviewRound == 0 {
			reviewRound = 1
		}
		round = reviewRound
	}

	// 确保该轮次步骤记录存在（如果不存在则从已有记录复制step_order）
	_, _ = database.DB.Exec(ctx,
		`INSERT INTO pipeline_steps
                    (pipeline_id, step_name, step_order, status, review_round)
                 SELECT $1, $2, step_order, 'pending', $3
                 FROM pipeline_steps
                 WHERE pipeline_id = $1 AND step_name = $2
                 ORDER BY review_round ASC LIMIT 1
                 ON CONFLICT (pipeline_id, step_name, review_round) DO NOTHING`,
		pipelineID, stepName, round)

	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
                 SET status = $3, started_at = NOW(),
                     attempts = attempts + 1, updated_at = NOW()
                 WHERE pipeline_id = $1 AND step_name = $2 AND review_round = $4`,
		pipelineID, stepName, models.StepStatusRunning, round)
	if err != nil {
		return fmt.Errorf("启动步骤%s失败: %w", stepName, err)
	}
	return nil
}

// CompleteStep 标记步骤完成（更新状态+完成时间+耗时+输出数据）
func CompleteStep(pipelineID string, stepName string, durationMs int64, stepData string, modelUsed string, tokensUsed int) error {
	ctx := context.Background()

	// 读取当前review_round
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}

	var stepDataParam interface{}
	if stepData != "" && stepData != "{}" {
		stepDataParam = stepData
	}

	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
                 SET status = $3, completed_at = NOW(), duration_ms = $4,
                     step_data = $5::jsonb, model_used = $6, tokens_used = $7, updated_at = NOW()
                 WHERE pipeline_id = $1 AND step_name = $2 AND review_round = $8`,
		pipelineID, stepName, models.StepStatusDone, durationMs,
		stepDataParam, modelUsed, tokensUsed, reviewRound)
	if err != nil {
		return fmt.Errorf("完成步骤%s失败: %w", stepName, err)
	}
	return nil
}

// FailStep 标记步骤失败（更新状态+完成时间+耗时+错误信息）
func FailStep(pipelineID string, stepName string, durationMs int64, errMsg string) error {
	ctx := context.Background()

	// 读取当前review_round
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}

	_, err := database.DB.Exec(ctx,
		`UPDATE pipeline_steps
                 SET status = 'failed', completed_at = NOW(),
                     duration_ms = $3, error_message = $4, updated_at = NOW()
                 WHERE pipeline_id = $1 AND step_name = $2 AND review_round = $5`,
		pipelineID, stepName, durationMs, errMsg, reviewRound)
	if err != nil {
		return fmt.Errorf("标记步骤%s失败: %w", stepName, err)
	}
	return nil
}

// ==================== EvalRound CRUD ====================

// CreateEvalRound 创建评估轮次记录
// 【v54修复】VALUES占位符从3个改为4个，补上review_round参数
func CreateEvalRound(pipelineID string, roundNumber int) (*models.EvalRoundRecord, error) {
	ctx := context.Background()

	// 读取当前review_round
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}

	r := &models.EvalRoundRecord{}
	err := database.DB.QueryRow(ctx,
		`INSERT INTO eval_rounds (pipeline_id, round_number, status, review_round)
                 VALUES ($1, $2, $3, $4)
                 RETURNING id, pipeline_id, round_number, status, created_at`,
		pipelineID, roundNumber, models.StepStatusPending, reviewRound,
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

// evalErrDimensions 用于 FailEvalRound 的JSON序列化结构体
// 修复E-06：改用json.Marshal替代字符串拼接，防止errMsg含引号时破坏JSON结构
type evalErrDimensions struct {
	Error string `json:"error"`
}

// FailEvalRound 标记评估轮次失败
// 修复E-06：使用json.Marshal序列化，自动转义所有特殊字符
func FailEvalRound(roundID string, output string, errMsg string) error {
	ctx := context.Background()

	dimJSON, marshalErr := json.Marshal(evalErrDimensions{Error: errMsg})
	if marshalErr != nil {
		dimJSON = []byte(`{"error":"serialization_failed"}`)
	}

	_, err := database.DB.Exec(ctx,
		`UPDATE eval_rounds SET status = $2, output = $3, dimensions = $4::jsonb WHERE id = $1`,
		roundID, models.StepStatusFailed, output, string(dimJSON))
	if err != nil {
		return fmt.Errorf("标记评估轮次失败: %w", err)
	}
	return nil
}

// GetEvalRoundsByPipelineID 获取指定Pipeline的所有评估轮次（按轮次序号排序）
// 只返回当前review_round的eval记录
func GetEvalRoundsByPipelineID(pipelineID string) ([]*models.EvalRoundRecord, error) {
	ctx := context.Background()
	// 读取当前review_round，只返回本轮的eval记录
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}

	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, round_number, status, output,
                        score_total, score_e1, score_e2, score_e3, score_e4,
                        dimensions::text, model_used, tokens_used, created_at
                 FROM eval_rounds
                 WHERE pipeline_id = $1 AND review_round = $2
                 ORDER BY round_number ASC`, pipelineID, reviewRound)
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

// DeleteEvalRoundsByPipelineID 删除指定Pipeline当前review_round的评估轮次（重跑时清理旧数据）
// 只删当前review_round的eval_rounds，保留历史轮次数据
func DeleteEvalRoundsByPipelineID(pipelineID string) error {
	ctx := context.Background()
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}
	_, err := database.DB.Exec(ctx,
		`DELETE FROM eval_rounds WHERE pipeline_id = $1 AND review_round = $2`,
		pipelineID, reviewRound)
	if err != nil {
		return fmt.Errorf("删除评估轮次失败: %w", err)
	}
	return nil
}

// ==================== 历史轮次查询 ====================

// GetStepsByRound 获取指定review_round的所有步骤数据（用于历史轮次回溯）
func GetStepsByRound(pipelineID string, round int) ([]*models.PipelineStep, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, step_name, step_order, status,
                        started_at, completed_at, duration_ms, attempts,
                        step_data::text, error_message, model_used, tokens_used,
                        created_at, updated_at
                 FROM pipeline_steps
                 WHERE pipeline_id = $1 AND review_round = $2
                 ORDER BY step_order ASC`, pipelineID, round)
	if err != nil {
		return nil, fmt.Errorf("查询历史步骤失败: %w", err)
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
			return nil, fmt.Errorf("扫描历史步骤行失败: %w", err)
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

// GetEvalRoundsByRound 获取指定review_round的所有评估轮次数据（用于历史轮次回溯）
func GetEvalRoundsByRound(pipelineID string, round int) ([]*models.EvalRoundRecord, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, round_number, status, output,
                        score_total, score_e1, score_e2, score_e3, score_e4,
                        dimensions::text, model_used, tokens_used, created_at
                 FROM eval_rounds
                 WHERE pipeline_id = $1 AND review_round = $2
                 ORDER BY round_number ASC`, pipelineID, round)
	if err != nil {
		return nil, fmt.Errorf("查询历史评估轮次失败: %w", err)
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
			return nil, fmt.Errorf("扫描历史评估行失败: %w", err)
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

// GetAvailableRounds 获取该Pipeline有哪些历史轮次
func GetAvailableRounds(pipelineID string) ([]int, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT DISTINCT review_round FROM pipeline_steps
                 WHERE pipeline_id = $1
                 ORDER BY review_round ASC`, pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rounds []int
	for rows.Next() {
		var r int
		if err := rows.Scan(&r); err == nil {
			rounds = append(rounds, r)
		}
	}
	return rounds, nil
}
