package repository

// workshop_stage_repo.go — 阶段化备课工坊数据访问层
//
// Phase 7B 新增：阶段定义查询+产出物CRUD+教案阶段状态更新
// 阶段管理新增：GetAllSystemStages+UpdateSystemStage
// 迭代1 修改：
//   - 所有SELECT增加prompt_variants列（COALESCE兜底'{}'）
//   - UpdateSystemStage增加prompt_variants参数
//   - scanStageRowsFromRows增加PromptVariants字段扫描
// 迭代5 修改：
//   - 新增 CreateRecipeStage 创建配方自定义阶段
//   - 新增 UpdateRecipeStage 更新配方自定义阶段
//   - 新增 DeleteRecipeStage 删除配方自定义阶段
//   - 新增 GetRecipeStageByCode 按配方ID+阶段代码查询单个自定义阶段
//   - 新增 CountRecipeStages 统计配方自定义阶段数量
// BugFix：
//   - UpdateStageOutputContent 存入前验证JSON有效性，无效时存{}并保留原始内容到narrative

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrStageNotFound       = errors.New("阶段不存在")
	ErrStageOutputNotFound = errors.New("阶段产出物不存在")
	ErrStageCodeConflict   = errors.New("阶段代码已存在") // 迭代5新增
)

var wsRepoLog = logger.WithModule("workshop_stage_repo")

// ==================== 阶段SELECT列常量（迭代1统一维护）====================

// stageSelectColumns 阶段定义表的标准SELECT列（16列）
const stageSelectColumns = `
	id, stage_code, stage_name, stage_order, source, recipe_id,
	ai_role, system_prompt,
	COALESCE(output_format::text, '{}'),
	COALESCE(component_types::text, '[]'),
	gate_mode, skippable, status,
	COALESCE(prompt_variants::text, '{}'),
	created_at, updated_at
`

// ==================== 阶段定义查询 ====================

// GetSystemDefaultStages 获取系统预设的默认阶段列表（仅active，按stage_order排序）
func GetSystemDefaultStages(ctx context.Context) ([]*models.WorkshopStage, error) {
	query := `SELECT ` + stageSelectColumns + `
		FROM workshop_stages
		WHERE source = 'system' AND status = 'active'
		ORDER BY stage_order ASC`
	return scanStageRows(ctx, query)
}

// GetAllSystemStages 获取全部系统阶段（含disabled，供管理页面使用）
func GetAllSystemStages(ctx context.Context) ([]*models.WorkshopStage, error) {
	query := `SELECT ` + stageSelectColumns + `
		FROM workshop_stages
		WHERE source = 'system'
		ORDER BY stage_order ASC`
	return scanStageRows(ctx, query)
}

// GetRecipeStages 获取指定配方的自定义阶段列表
func GetRecipeStages(ctx context.Context, recipeID string) ([]*models.WorkshopStage, error) {
	query := `SELECT ` + stageSelectColumns + `
		FROM workshop_stages
		WHERE source = 'recipe' AND recipe_id = $1 AND status = 'active'
		ORDER BY stage_order ASC`
	return scanStageRowsWithArgs(ctx, query, recipeID)
}

// GetStageByCode 根据source和stage_code查询单个阶段定义
func GetStageByCode(ctx context.Context, source string, stageCode string) (*models.WorkshopStage, error) {
	s := &models.WorkshopStage{}
	query := `SELECT ` + stageSelectColumns + `
		FROM workshop_stages
		WHERE source = $1 AND stage_code = $2 AND status = 'active'`
	err := database.DB.QueryRow(ctx, query, source, stageCode).Scan(
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
		return nil, fmt.Errorf("查询阶段定义失败: %w", err)
	}
	return s, nil
}

// ==================== 迭代5新增：配方自定义阶段 CRUD ====================

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

// DeleteRecipeStage 删除配方自定义阶段（物理删除）
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

// ==================== 系统阶段更新（Admin管理页面）====================

// UpdateSystemStage 更新系统阶段的可编辑字段
func UpdateSystemStage(ctx context.Context, stageCode string, req *models.UpdateStageRequest) error {
	promptVariants := req.PromptVariants
	if promptVariants == "" {
		promptVariants = "{}"
	}

	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stages
		SET stage_name = $1, ai_role = $2, system_prompt = $3,
		    output_format = $4, component_types = $5,
		    gate_mode = $6, skippable = $7, status = $8,
		    prompt_variants = $9,
		    updated_at = $10
		WHERE source = 'system' AND stage_code = $11
	`,
		req.StageName, req.AIRole, req.SystemPrompt,
		req.OutputFormat, req.ComponentTypes,
		req.GateMode, req.Skippable, req.Status,
		promptVariants,
		now, stageCode,
	)
	if err != nil {
		return fmt.Errorf("更新系统阶段失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

// ==================== 阶段产出物 CRUD ====================

// CreateStageOutput 创建阶段产出记录（教案进入某阶段时调用）
func CreateStageOutput(ctx context.Context, out *models.WorkshopStageOutput) error {
	query := `
		INSERT INTO workshop_stage_outputs (
			lesson_plan_id, stage_code, stage_order,
			structured_output, narrative_output, conversation_snapshot,
			model_used, tokens_used, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := database.DB.QueryRow(ctx, query,
		out.LessonPlanID, out.StageCode, out.StageOrder,
		out.StructuredOutput, out.NarrativeOutput, out.ConversationSnapshot,
		out.ModelUsed, out.TokensUsed, out.Status,
	).Scan(&out.ID, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建阶段产出失败: %w", err)
	}
	return nil
}

// GetStageOutput 查询指定教案指定阶段的产出物
func GetStageOutput(ctx context.Context, lessonPlanID string, stageCode string) (*models.WorkshopStageOutput, error) {
	out := &models.WorkshopStageOutput{}
	query := `
		SELECT id, lesson_plan_id, stage_code, stage_order,
		       COALESCE(structured_output::text, '{}'),
		       COALESCE(narrative_output, ''),
		       COALESCE(conversation_snapshot::text, '[]'),
		       COALESCE(model_used, ''), COALESCE(tokens_used, 0),
		       status, completed_at, created_at, updated_at
		FROM workshop_stage_outputs
		WHERE lesson_plan_id = $1 AND stage_code = $2
	`
	err := database.DB.QueryRow(ctx, query, lessonPlanID, stageCode).Scan(
		&out.ID, &out.LessonPlanID, &out.StageCode, &out.StageOrder,
		&out.StructuredOutput,
		&out.NarrativeOutput,
		&out.ConversationSnapshot,
		&out.ModelUsed, &out.TokensUsed,
		&out.Status, &out.CompletedAt, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageOutputNotFound
		}
		return nil, fmt.Errorf("查询阶段产出失败: %w", err)
	}
	return out, nil
}

// ListStageOutputs 查询指定教案的所有阶段产出物（按stage_order排序）
func ListStageOutputs(ctx context.Context, lessonPlanID string) ([]*models.WorkshopStageOutput, error) {
	query := `
		SELECT id, lesson_plan_id, stage_code, stage_order,
		       COALESCE(structured_output::text, '{}'),
		       COALESCE(narrative_output, ''),
		       COALESCE(conversation_snapshot::text, '[]'),
		       COALESCE(model_used, ''), COALESCE(tokens_used, 0),
		       status, completed_at, created_at, updated_at
		FROM workshop_stage_outputs
		WHERE lesson_plan_id = $1
		ORDER BY stage_order ASC
	`
	rows, err := database.DB.Query(ctx, query, lessonPlanID)
	if err != nil {
		return nil, fmt.Errorf("查询教案阶段产出列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.WorkshopStageOutput
	for rows.Next() {
		out := &models.WorkshopStageOutput{}
		if err := rows.Scan(
			&out.ID, &out.LessonPlanID, &out.StageCode, &out.StageOrder,
			&out.StructuredOutput,
			&out.NarrativeOutput,
			&out.ConversationSnapshot,
			&out.ModelUsed, &out.TokensUsed,
			&out.Status, &out.CompletedAt, &out.CreatedAt, &out.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描阶段产出行失败: %w", err)
		}
		items = append(items, out)
	}
	if items == nil {
		items = []*models.WorkshopStageOutput{}
	}
	return items, nil
}

// UpdateStageOutputContent 更新阶段产出物内容（AI生成产出物时调用）
//
// BugFix：structured_output 是 JSONB 字段，存入前必须验证 JSON 有效性。
// 当 AI 输出的 JSON 包含未转义的中文引号等特殊字符时，json.Unmarshal 会失败。
// 此时将 structured_output 降级为 {}，将原始内容附加到 narrative_output 保留。
func UpdateStageOutputContent(ctx context.Context, lessonPlanID string, stageCode string, structuredOutput string, narrativeOutput string, modelUsed string, tokensUsed int) error {
	safeStructured := structuredOutput
	safeNarrative := narrativeOutput

	// 验证 structuredOutput 是否为有效 JSON
	if structuredOutput != "" && structuredOutput != "{}" {
		var testParse interface{}
		if err := json.Unmarshal([]byte(structuredOutput), &testParse); err != nil {
			// JSON 无效：降级存储，保留原始内容到 narrative
			wsRepoLog.Warn("structured_output JSON无效，降级存储",
				"stage_code", stageCode,
				"plan_id", lessonPlanID,
				"error", err.Error(),
				"raw_len", len(structuredOutput),
			)
			safeStructured = "{}"
			// 将原始JSON内容追加到narrative，供后续容错提取使用
			if safeNarrative == "" {
				safeNarrative = structuredOutput
			} else {
				safeNarrative = safeNarrative + "\n\n[原始产出物备份]\n" + structuredOutput
			}
		}
	}

	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stage_outputs
		SET structured_output = $1, narrative_output = $2,
		    model_used = $3, tokens_used = $4, updated_at = $5
		WHERE lesson_plan_id = $6 AND stage_code = $7
	`, safeStructured, safeNarrative, modelUsed, tokensUsed, now, lessonPlanID, stageCode)
	if err != nil {
		return fmt.Errorf("更新阶段产出内容失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageOutputNotFound
	}
	return nil
}

// CompleteStageOutput 标记阶段产出为已完成（附带对话快照）
func CompleteStageOutput(ctx context.Context, lessonPlanID string, stageCode string, conversationSnapshot string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stage_outputs
		SET status = $1, completed_at = $2, conversation_snapshot = $3, updated_at = $2
		WHERE lesson_plan_id = $4 AND stage_code = $5
	`, models.StageOutputCompleted, now, conversationSnapshot, lessonPlanID, stageCode)
	if err != nil {
		return fmt.Errorf("完成阶段产出失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageOutputNotFound
	}
	return nil
}

// SkipStageOutput 标记阶段产出为已跳过
func SkipStageOutput(ctx context.Context, lessonPlanID string, stageCode string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stage_outputs
		SET status = $1, completed_at = $2, updated_at = $2
		WHERE lesson_plan_id = $3 AND stage_code = $4
	`, models.StageOutputSkipped, now, lessonPlanID, stageCode)
	if err != nil {
		return fmt.Errorf("跳过阶段产出失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStageOutputNotFound
	}
	return nil
}

// ResetStageOutput 重置单个阶段的产出物为初始状态
// 迭代12新增：重启阶段时调用，清空产出内容，状态设回 in_progress
func ResetStageOutput(ctx context.Context, lessonPlanID string, stageCode string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE workshop_stage_outputs
		SET structured_output = '{}', narrative_output = '', conversation_snapshot = '[]',
		    model_used = '', tokens_used = 0, status = $1, completed_at = NULL, updated_at = $2
		WHERE lesson_plan_id = $3 AND stage_code = $4
	`, models.StageOutputInProgress, now, lessonPlanID, stageCode)
	if err != nil {
		return fmt.Errorf("重置阶段产出失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		// 不存在也不算错误，可能该阶段还没创建产出记录
		return nil
	}
	return nil
}

// DeleteStageOutputsAfter 删除指定阶段之后的所有产出物
// 迭代12新增：重启某阶段时，该阶段之后的阶段产出物需要删除
func DeleteStageOutputsAfter(ctx context.Context, lessonPlanID string, stageOrder int) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM workshop_stage_outputs WHERE lesson_plan_id = $1 AND stage_order > $2`,
		lessonPlanID, stageOrder,
	)
	if err != nil {
		return fmt.Errorf("删除后续阶段产出失败: %w", err)
	}
	return nil
}

// ==================== 教案阶段状态更新 ====================

// UpdateLessonPlanCurrentStage 更新教案的当前阶段
func UpdateLessonPlanCurrentStage(ctx context.Context, lessonPlanID string, stageCode string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET current_stage = $1, updated_at = $2 WHERE id = $3`,
		stageCode, now, lessonPlanID,
	)
	if err != nil {
		return fmt.Errorf("更新教案当前阶段失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanStageConfig 写入教案的阶段配置快照
func UpdateLessonPlanStageConfig(ctx context.Context, lessonPlanID string, stageConfigJSON string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET stage_config = $1, updated_at = $2 WHERE id = $3`,
		stageConfigJSON, now, lessonPlanID,
	)
	if err != nil {
		return fmt.Errorf("写入教案阶段配置失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ==================== 内部扫描辅助函数 ====================

// scanStageRows 扫描阶段定义多行结果（无参数查询）
func scanStageRows(ctx context.Context, query string) ([]*models.WorkshopStage, error) {
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询阶段定义失败: %w", err)
	}
	defer rows.Close()
	return scanStageRowsFromRows(rows)
}

// scanStageRowsWithArgs 扫描阶段定义多行结果（带参数查询）
func scanStageRowsWithArgs(ctx context.Context, query string, args ...interface{}) ([]*models.WorkshopStage, error) {
	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询阶段定义失败: %w", err)
	}
	defer rows.Close()
	return scanStageRowsFromRows(rows)
}

// scanStageRowsFromRows 从pgx.Rows中逐行扫描阶段定义
func scanStageRowsFromRows(rows pgx.Rows) ([]*models.WorkshopStage, error) {
	var items []*models.WorkshopStage
	for rows.Next() {
		s := &models.WorkshopStage{}
		if err := rows.Scan(
			&s.ID, &s.StageCode, &s.StageName, &s.StageOrder, &s.Source, &s.RecipeID,
			&s.AIRole, &s.SystemPrompt,
			&s.OutputFormat,
			&s.ComponentTypes,
			&s.GateMode, &s.Skippable, &s.Status,
			&s.PromptVariants,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描阶段定义行失败: %w", err)
		}
		items = append(items, s)
	}
	if items == nil {
		items = []*models.WorkshopStage{}
	}
	return items, nil
}
