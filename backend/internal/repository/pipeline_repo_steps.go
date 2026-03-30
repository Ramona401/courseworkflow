package repository

// pipeline_repo_steps.go — Pipeline步骤 + 评估轮次数据访问层
//
// 职责：
//   - PipelineStep CRUD（查询/启动/完成/失败）
//   - EvalRound CRUD（创建/更新/完成/失败/查询/删除）

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

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

// ==================== EvalRound CRUD ====================

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
