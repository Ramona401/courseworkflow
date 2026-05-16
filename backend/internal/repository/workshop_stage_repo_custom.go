package repository

// workshop_stage_repo_custom.go — 配方自定义阶段数据访问
//
// 从 workshop_stage_repo.go 拆分,负责:
//   - GetRecipeStageByCode  按配方ID+阶段代码查询
//   - CreateRecipeStage     创建配方自定义阶段
//   - UpdateRecipeStage     更新配方自定义阶段
//   - DeleteRecipeStage     删除配方自定义阶段(物理删除)
//   - CountRecipeStages     统计配方自定义阶段数量

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// GetRecipeStageByCode 按配方ID+阶段代码查询单个自定义阶段
func GetRecipeStageByCode(ctx context.Context, recipeID string, stageCode string) (*models.WorkshopStage, error) {
	s := &models.WorkshopStage{}
	query := `SELECT ` + stageSelectColumns + `
		FROM workshop_stages
		WHERE source = 'recipe' AND recipe_id = $1 AND stage_code = $2`
	err := database.DB.QueryRow(ctx, query, recipeID, stageCode).Scan(
		&s.ID, &s.StageCode, &s.StageName, &s.StageOrder, &s.Source, &s.RecipeID,
		&s.AIRole, &s.SystemPrompt,
		&s.OutputFormat,
		&s.ComponentTypes,
		&s.GateMode, &s.Skippable, &s.Status,
		&s.PromptVariants,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("查询配方自定义阶段失败: %w", err)
	}
	return s, nil
}

// CreateRecipeStage 创建配方自定义阶段
func CreateRecipeStage(ctx context.Context, recipeID string, req *models.CreateCustomStageRequest) (*models.WorkshopStage, error) {
	// 检查同一配方内stage_code是否重复
	_, err := GetRecipeStageByCode(ctx, recipeID, req.StageCode)
	if err == nil {
		return nil, ErrStageCodeConflict
	}

	// 同时检查是否与系统阶段代码冲突
	_, err = GetStageByCode(ctx, models.StageSourceSystem, req.StageCode)
	if err == nil {
		return nil, fmt.Errorf("阶段代码 %s 与系统阶段冲突", req.StageCode)
	}

	// 计算排序序号
	var maxOrder int
	err = database.DB.QueryRow(ctx,
		`SELECT COALESCE(MAX(stage_order), 0) FROM workshop_stages WHERE source = 'recipe' AND recipe_id = $1`,
		recipeID,
	).Scan(&maxOrder)
	if err != nil {
		return nil, fmt.Errorf("查询最大排序序号失败: %w", err)
	}

	gateMode := req.GateMode
	if gateMode == "" {
		gateMode = models.StageGateSuggest
	}
	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "{}"
	}
	componentTypes := req.ComponentTypes
	if componentTypes == "" {
		componentTypes = "[]"
	}
	promptVariants := req.PromptVariants
	if promptVariants == "" {
		promptVariants = "{}"
	}

	s := &models.WorkshopStage{}
	query := `
		INSERT INTO workshop_stages (
			stage_code, stage_name, stage_order, source, recipe_id,
			ai_role, system_prompt, prompt_variants,
			output_format, component_types,
			gate_mode, skippable, status
		) VALUES ($1, $2, $3, 'recipe', $4, $5, $6, $7, $8, $9, $10, $11, 'active')
		RETURNING id, stage_code, stage_name, stage_order, source, recipe_id,
			ai_role, system_prompt,
			COALESCE(output_format::text, '{}'),
			COALESCE(component_types::text, '[]'),
			gate_mode, skippable, status,
			COALESCE(prompt_variants::text, '{}'),
			created_at, updated_at
	`
	err = database.DB.QueryRow(ctx, query,
		req.StageCode, req.StageName, maxOrder+100, recipeID,
		req.AIRole, req.SystemPrompt, promptVariants,
		outputFormat, componentTypes,
		gateMode, req.Skippable,
	).Scan(
		&s.ID, &s.StageCode, &s.StageName, &s.StageOrder, &s.Source, &s.RecipeID,
		&s.AIRole, &s.SystemPrompt,
		&s.OutputFormat,
		&s.ComponentTypes,
		&s.GateMode, &s.Skippable, &s.Status,
		&s.PromptVariants,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建配方自定义阶段失败: %w", err)
	}
	return s, nil
}

// UpdateRecipeStage 更新配方自定义阶段
func UpdateRecipeStage(ctx context.Context, recipeID string, stageCode string, req *models.UpdateCustomStageRequest) error {
	gateMode := req.GateMode
	if gateMode == "" {
		gateMode = models.StageGateSuggest
	}
	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "{}"
	}
	componentTypes := req.ComponentTypes
	if componentTypes == "" {
		componentTypes = "[]"
	}
	promptVariants := req.PromptVariants
	if promptVariants == "" {
		promptVariants = "{}"
	}

	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stages
		SET stage_name = $1, ai_role = $2, system_prompt = $3,
		    prompt_variants = $4, output_format = $5, component_types = $6,
		    gate_mode = $7, skippable = $8, updated_at = $9
		WHERE source = 'recipe' AND recipe_id = $10 AND stage_code = $11
	`,
		req.StageName, req.AIRole, req.SystemPrompt,
		promptVariants, outputFormat, componentTypes,
		gateMode, req.Skippable, now,
		recipeID, stageCode,
	)
	if err != nil {
		return fmt.Errorf("更新配方自定义阶段失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

// DeleteRecipeStage 删除配方自定义阶段(物理删除)
func DeleteRecipeStage(ctx context.Context, recipeID string, stageCode string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM workshop_stages WHERE source = 'recipe' AND recipe_id = $1 AND stage_code = $2`,
		recipeID, stageCode,
	)
	if err != nil {
		return fmt.Errorf("删除配方自定义阶段失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

// CountRecipeStages 统计配方自定义阶段数量
func CountRecipeStages(ctx context.Context, recipeID string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM workshop_stages WHERE source = 'recipe' AND recipe_id = $1 AND status = 'active'`,
		recipeID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计配方自定义阶段数量失败: %w", err)
	}
	return count, nil
}
