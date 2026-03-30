package repository

// lesson_plan_repo_ext.go — 教案扩展数据访问层
//
// 职责：
//   - 提示词模板CRUD（创建/查询/列表/更新/继承链解析）
//   - 组件萃取CRUD（创建/列表/确认/查询）
//   - 对话记录CRUD（追加/查询/清空）

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 提示词模板CRUD ====================

// CreatePromptTemplate 创建提示词模板
func CreatePromptTemplate(ctx context.Context, pt *models.PromptTemplate) error {
	query := `
		INSERT INTO prompt_templates (
			name, description, level, owner_id, parent_template_id,
			system_prompt, context_rules, generation_rules, review_rules,
			output_format, custom_instructions,
			subject, grade_range, is_default, version, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17
		)
		RETURNING id, created_at, updated_at
	`
	var contextRules, genRules, reviewRules, outputFmt *string
	if pt.ContextRules != nil {
		v := *pt.ContextRules
		if v == "" {
			v = "{}"
		}
		contextRules = &v
	}
	if pt.GenerationRules != nil {
		v := *pt.GenerationRules
		if v == "" {
			v = "{}"
		}
		genRules = &v
	}
	if pt.ReviewRules != nil {
		v := *pt.ReviewRules
		if v == "" {
			v = "{}"
		}
		reviewRules = &v
	}
	if pt.OutputFormat != nil {
		v := *pt.OutputFormat
		if v == "" {
			v = "{}"
		}
		outputFmt = &v
	}
	version := pt.Version
	if version <= 0 {
		version = 1
	}
	status := pt.Status
	if status == "" {
		status = "active"
	}

	err := database.DB.QueryRow(ctx, query,
		pt.Name, pt.Description, pt.Level, pt.OwnerID, pt.ParentTemplateID,
		pt.SystemPrompt, contextRules, genRules, reviewRules,
		outputFmt, pt.CustomInstructions,
		pt.Subject, pt.GradeRange, pt.IsDefault, version, status, pt.CreatedBy,
	).Scan(&pt.ID, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建提示词模板失败: %w", err)
	}
	return nil
}

// GetPromptTemplateByID 根据ID查询提示词模板
func GetPromptTemplateByID(ctx context.Context, id string) (*models.PromptTemplate, error) {
	pt := &models.PromptTemplate{}
	query := `
		SELECT id, name, description, level, owner_id, parent_template_id,
		       system_prompt, context_rules, generation_rules, review_rules,
		       output_format, custom_instructions,
		       subject, grade_range, is_default, version, status, created_by,
		       created_at, updated_at
		FROM prompt_templates WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&pt.ID, &pt.Name, &pt.Description, &pt.Level, &pt.OwnerID, &pt.ParentTemplateID,
		&pt.SystemPrompt, &pt.ContextRules, &pt.GenerationRules, &pt.ReviewRules,
		&pt.OutputFormat, &pt.CustomInstructions,
		&pt.Subject, &pt.GradeRange, &pt.IsDefault, &pt.Version, &pt.Status, &pt.CreatedBy,
		&pt.CreatedAt, &pt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("查询提示词模板失败: %w", err)
	}
	return pt, nil
}

// ListPromptTemplates 获取提示词模板列表（支持level/ownerID筛选）
func ListPromptTemplates(ctx context.Context, level string, ownerID string) ([]*models.PromptTemplateListItem, error) {
	where := " WHERE pt.status = 'active'"
	args := []interface{}{}
	argIdx := 1

	if level != "" {
		where += fmt.Sprintf(" AND pt.level = $%d", argIdx)
		args = append(args, level)
		argIdx++
	}
	if ownerID != "" {
		where += fmt.Sprintf(" AND pt.owner_id = $%d", argIdx)
		args = append(args, ownerID)
		argIdx++
	}

	query := `
		SELECT pt.id, pt.name, pt.level, pt.owner_id, pt.parent_template_id,
		       pt.subject, pt.grade_range, pt.is_default, pt.version, pt.status, pt.created_at
		FROM prompt_templates pt
	` + where + " ORDER BY pt.level ASC, pt.name ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PromptTemplateListItem
	for rows.Next() {
		item := &models.PromptTemplateListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.Level, &item.OwnerID, &item.ParentTemplateID,
			&item.Subject, &item.GradeRange, &item.IsDefault, &item.Version,
			&item.Status, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描模板行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdatePromptTemplate 更新提示词模板（版本号+1）
func UpdatePromptTemplate(ctx context.Context, id string, req *models.UpdatePromptTemplateRequest) error {
	now := time.Now()
	var contextRules, genRules, reviewRules, outputFmt *string
	if req.ContextRules != "" {
		v := req.ContextRules
		contextRules = &v
	}
	if req.GenerationRules != "" {
		v := req.GenerationRules
		genRules = &v
	}
	if req.ReviewRules != "" {
		v := req.ReviewRules
		reviewRules = &v
	}
	if req.OutputFormat != "" {
		v := req.OutputFormat
		outputFmt = &v
	}
	status := req.Status
	if status == "" {
		status = "active"
	}

	result, err := database.DB.Exec(ctx, `
		UPDATE prompt_templates
		SET name = $1, description = $2, system_prompt = $3,
		    context_rules = $4, generation_rules = $5, review_rules = $6,
		    output_format = $7, custom_instructions = $8,
		    subject = $9, grade_range = $10, is_default = $11,
		    version = version + 1, status = $12, updated_at = $13
		WHERE id = $14
	`,
		req.Name, req.Description, req.SystemPrompt,
		contextRules, genRules, reviewRules,
		outputFmt, req.CustomInstructions,
		req.Subject, req.GradeRange, req.IsDefault,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// ResolvePromptTemplateChain 解析提示词模板继承链（从叶到根向上遍历，子级覆盖父级）
// 最多遍历10层防止循环继承
func ResolvePromptTemplateChain(ctx context.Context, templateID string) (*models.ResolvedPromptTemplate, error) {
	resolved := &models.ResolvedPromptTemplate{}
	var chain []string
	currentID := templateID

	for i := 0; i < 10 && currentID != ""; i++ {
		pt, err := GetPromptTemplateByID(ctx, currentID)
		if err != nil {
			break
		}
		chain = append([]string{pt.ID}, chain...) // 头插，根在前

		// 子级非NULL非空时覆盖父级
		if pt.SystemPrompt != nil && *pt.SystemPrompt != "" && resolved.SystemPrompt == "" {
			resolved.SystemPrompt = *pt.SystemPrompt
		}
		if pt.ContextRules != nil && *pt.ContextRules != "" && *pt.ContextRules != "{}" && resolved.ContextRules == "" {
			resolved.ContextRules = *pt.ContextRules
		}
		if pt.GenerationRules != nil && *pt.GenerationRules != "" && *pt.GenerationRules != "{}" && resolved.GenerationRules == "" {
			resolved.GenerationRules = *pt.GenerationRules
		}
		if pt.ReviewRules != nil && *pt.ReviewRules != "" && *pt.ReviewRules != "{}" && resolved.ReviewRules == "" {
			resolved.ReviewRules = *pt.ReviewRules
		}
		if pt.OutputFormat != nil && *pt.OutputFormat != "" && *pt.OutputFormat != "{}" && resolved.OutputFormat == "" {
			resolved.OutputFormat = *pt.OutputFormat
		}
		if pt.CustomInstructions != nil && *pt.CustomInstructions != "" && resolved.CustomInstructions == "" {
			resolved.CustomInstructions = *pt.CustomInstructions
		}

		if pt.ParentTemplateID != nil {
			currentID = *pt.ParentTemplateID
		} else {
			currentID = ""
		}
	}

	resolved.InheritanceChain = chain
	return resolved, nil
}

// ==================== 组件萃取CRUD ====================

// CreateComponentExtraction 创建组件萃取记录
func CreateComponentExtraction(ctx context.Context, ce *models.ComponentExtraction) error {
	status := ce.Status
	if status == "" {
		status = "pending"
	}
	err := database.DB.QueryRow(ctx, `
		INSERT INTO component_extractions (
			source_type, source_lesson_plan_id, source_content,
			extracted_component_id, extraction_type, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`,
		ce.SourceType, ce.SourceLessonPlanID, ce.SourceContent,
		ce.ExtractedComponentID, ce.ExtractionType, status, ce.CreatedBy,
	).Scan(&ce.ID, &ce.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建萃取记录失败: %w", err)
	}
	return nil
}

// ListPendingExtractions 获取待确认的萃取记录列表
func ListPendingExtractions(ctx context.Context, limit int) ([]*models.ComponentExtraction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := database.DB.Query(ctx, `
		SELECT id, source_type, source_lesson_plan_id, source_content,
		       extracted_component_id, extraction_type, status,
		       confirmed_by, confirmed_at, created_by, created_at
		FROM component_extractions
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待确认萃取失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ComponentExtraction
	for rows.Next() {
		ce := &models.ComponentExtraction{}
		err := rows.Scan(
			&ce.ID, &ce.SourceType, &ce.SourceLessonPlanID, &ce.SourceContent,
			&ce.ExtractedComponentID, &ce.ExtractionType, &ce.Status,
			&ce.ConfirmedBy, &ce.ConfirmedAt, &ce.CreatedBy, &ce.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描萃取行失败: %w", err)
		}
		items = append(items, ce)
	}
	return items, nil
}

// ConfirmExtraction 确认/拒绝萃取记录
func ConfirmExtraction(ctx context.Context, id string, confirmedBy string, decision string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE component_extractions SET status = $1, confirmed_by = $2, confirmed_at = $3 WHERE id = $4`,
		decision, confirmedBy, now, id,
	)
	if err != nil {
		return fmt.Errorf("确认萃取记录失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrExtractionNotFound
	}
	return nil
}

// GetExtractionByID 根据ID查询萃取记录
func GetExtractionByID(ctx context.Context, id string) (*models.ComponentExtraction, error) {
	ce := &models.ComponentExtraction{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, source_type, source_lesson_plan_id, source_content,
		       extracted_component_id, extraction_type, status,
		       confirmed_by, confirmed_at, created_by, created_at
		FROM component_extractions WHERE id = $1
	`, id).Scan(
		&ce.ID, &ce.SourceType, &ce.SourceLessonPlanID, &ce.SourceContent,
		&ce.ExtractedComponentID, &ce.ExtractionType, &ce.Status,
		&ce.ConfirmedBy, &ce.ConfirmedAt, &ce.CreatedBy, &ce.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExtractionNotFound
		}
		return nil, fmt.Errorf("查询萃取记录失败: %w", err)
	}
	return ce, nil
}

// ==================== 对话记录CRUD ====================

// AppendConversationMessage 追加一条消息到教案对话记录（原子操作）
func AppendConversationMessage(ctx context.Context, planID string, msg *models.ConversationMessage) error {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET conversation_log = conversation_log || $1::jsonb,
		    updated_at = $2
		WHERE id = $3
	`, string(msgJSON), now, planID)
	if err != nil {
		return fmt.Errorf("追加对话消息失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// GetConversationLog 获取教案的完整对话记录
func GetConversationLog(ctx context.Context, planID string) ([]*models.ConversationMessage, error) {
	var logJSON string
	err := database.DB.QueryRow(ctx,
		`SELECT conversation_log FROM lesson_plans WHERE id = $1`, planID,
	).Scan(&logJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf("获取对话记录失败: %w", err)
	}
	if logJSON == "" || logJSON == "[]" || logJSON == "null" {
		return []*models.ConversationMessage{}, nil
	}
	var msgs []*models.ConversationMessage
	if err := json.Unmarshal([]byte(logJSON), &msgs); err != nil {
		return nil, fmt.Errorf("解析对话记录失败: %w", err)
	}
	return msgs, nil
}

// ClearConversationLog 清空对话记录（重新开始备课时使用）
func ClearConversationLog(ctx context.Context, planID string) error {
	now := time.Now()
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET conversation_log = '[]', updated_at = $1 WHERE id = $2`,
		now, planID,
	)
	return err
}
