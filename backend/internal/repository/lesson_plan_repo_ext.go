package repository

// lesson_plan_repo_ext.go — 教案扩展数据访问层
//
// 职责：
//   - 提示词模板CRUD（创建/查询/列表/更新/继承链解析）
//   - 组件萃取CRUD（创建/列表/确认/查询）
//   - 对话记录CRUD（追加/查询/清空/按阶段截断）
//
// v77新增：TruncateConversationFromStage — 按阶段分隔符截断对话记录
// v84新增：GetCurrentStageMessages — 提取当前阶段的对话消息（分层记忆Working层）
//          BuildEpisodicSummaryFromOutputs — 从阶段产出物构建历史摘要（Episodic层）

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// TruncateConversationFromStage 按阶段分隔符截断对话记录（v77新增）
//
// 逻辑：
//   1. 读取完整对话记录
//   2. 从头扫描，找到内容以 "__STAGE_SEP__" 开头且包含目标阶段名的 system 消息
//   3. 保留该分隔符之前的所有消息，截掉分隔符及之后的消息
//   4. 将截断后的消息写回数据库
//
// 参数：
//   - planID: 教案ID
//   - targetStageCode: 目标阶段代码（如 "write"）
//   - stageCodeToNameMap: 阶段代码→阶段名映射，用于匹配分隔符内容
//
// 如果找不到目标阶段的分隔符，则清空全部对话记录（兜底行为）
func TruncateConversationFromStage(ctx context.Context, planID string, targetStageCode string, stageCodeToNameMap map[string]string) error {
	// 读取完整对话记录
	msgs, err := GetConversationLog(ctx, planID)
	if err != nil {
		return fmt.Errorf("截断对话-读取失败: %w", err)
	}

	// 如果对话为空，无需截断
	if len(msgs) == 0 {
		return nil
	}

	// 查找目标阶段的分隔符位置
	// 分隔符格式: __STAGE_SEP__阶段名__AI角色
	stageSepPrefix := "__STAGE_SEP__"
	targetStageName := ""
	if name, ok := stageCodeToNameMap[targetStageCode]; ok {
		targetStageName = name
	}

	truncateIdx := -1
	for i, msg := range msgs {
		if msg.Role == "system" && strings.HasPrefix(msg.Content, stageSepPrefix) {
			// 提取分隔符中的阶段名
			rest := msg.Content[len(stageSepPrefix):]
			parts := strings.SplitN(rest, "__", 2)
			sepStageName := ""
			if len(parts) > 0 {
				sepStageName = parts[0]
			}

			// 匹配方式：优先按阶段名匹配，也按阶段代码匹配（兜底）
			if sepStageName == targetStageName || sepStageName == targetStageCode {
				truncateIdx = i
				break
			}
		}
	}

	if truncateIdx == -1 {
		// 找不到分隔符，清空全部对话（兜底）
		return ClearConversationLog(ctx, planID)
	}

	// 保留分隔符之前的消息
	truncated := msgs[:truncateIdx]

	// 序列化并写回
	truncatedJSON, err := json.Marshal(truncated)
	if err != nil {
		return fmt.Errorf("截断对话-序列化失败: %w", err)
	}

	now := time.Now()
	_, err = database.DB.Exec(ctx,
		`UPDATE lesson_plans SET conversation_log = $1::jsonb, updated_at = $2 WHERE id = $3`,
		string(truncatedJSON), now, planID,
	)
	if err != nil {
		return fmt.Errorf("截断对话-写回失败: %w", err)
	}

	return nil
}

// ==================== v84新增：分层记忆 Working 层 ====================

// GetCurrentStageMessages 从conversation_log中提取当前阶段的对话消息
//
// v84新增：分层记忆架构的Working Memory层核心函数
//
// 逻辑：
//   1. 读取完整conversation_log
//   2. 从后向前扫描，找到最后一个 __STAGE_SEP__ 分隔符
//   3. 返回该分隔符之后的所有非system消息（即当前阶段的用户+AI对话）
//   4. 如果找不到分隔符（第一个阶段或旧教案），返回全部非system消息
//
// 返回值：
//   - currentMessages: 当前阶段的对话消息列表（不含分隔符system消息）
//   - error: 错误信息
func GetCurrentStageMessages(ctx context.Context, planID string) ([]*models.ConversationMessage, error) {
	// 读取完整对话记录
	allMsgs, err := GetConversationLog(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("获取当前阶段消息失败: %w", err)
	}

	if len(allMsgs) == 0 {
		return []*models.ConversationMessage{}, nil
	}

	// 从后向前扫描，找到最后一个阶段分隔符
	stageSepPrefix := "__STAGE_SEP__"
	lastSepIdx := -1
	for i := len(allMsgs) - 1; i >= 0; i-- {
		if allMsgs[i].Role == "system" && strings.HasPrefix(allMsgs[i].Content, stageSepPrefix) {
			lastSepIdx = i
			break
		}
	}

	// 确定当前阶段消息的起始位置
	startIdx := 0
	if lastSepIdx >= 0 {
		// 找到分隔符，从分隔符之后开始
		startIdx = lastSepIdx + 1
	}
	// 如果没找到分隔符（第一个阶段或旧教案），从头开始

	// 提取当前阶段的非system消息（过滤掉分隔符等system消息）
	var currentMsgs []*models.ConversationMessage
	for i := startIdx; i < len(allMsgs); i++ {
		msg := allMsgs[i]
		// 跳过阶段分隔符类型的system消息
		if msg.Role == "system" && strings.HasPrefix(msg.Content, stageSepPrefix) {
			continue
		}
		currentMsgs = append(currentMsgs, msg)
	}

	if currentMsgs == nil {
		currentMsgs = []*models.ConversationMessage{}
	}

	return currentMsgs, nil
}

// BuildEpisodicSummaryFromOutputs 从阶段产出物构建Episodic Memory摘要文本
//
// v84新增：分层记忆架构的Episodic Memory层核心函数
//
// 与 BuildPriorOutputsContext（系统提示词中的前序产出注入）不同：
//   - BuildPriorOutputsContext 注入到 systemPrompt，包含 structured_output 原文
//   - BuildEpisodicSummaryFromOutputs 注入到 userPrompt，只包含 narrative_output 摘要
//   - 两者互补：systemPrompt 提供结构化数据，userPrompt 提供对话式摘要
//
// 参数：
//   - outputs: 当前阶段之前所有阶段的产出物列表
//
// 返回值：
//   - 格式化的摘要文本，用于注入到 BuildStageChatPromptV2 的 userPrompt 中
//   - 如果没有已完成的历史阶段，返回空字符串
func BuildEpisodicSummaryFromOutputs(outputs []*models.WorkshopStageOutput) string {
	if len(outputs) == 0 {
		return ""
	}

	var sb strings.Builder
	hasContent := false

	for _, out := range outputs {
		// 跳过未完成的阶段（只有completed和skipped的有意义）
		if out.Status != models.StageOutputCompleted && out.Status != models.StageOutputSkipped {
			continue
		}

		stageName := stageCodeToNameForRepo(out.StageCode)

		if out.Status == models.StageOutputSkipped {
			if !hasContent {
				sb.WriteString("【历史阶段摘要】\n")
				hasContent = true
			}
			sb.WriteString(fmt.Sprintf("· %s：已跳过\n", stageName))
			continue
		}

		// 优先使用 narrative_output 作为摘要
		narrative := strings.TrimSpace(out.NarrativeOutput)
		if narrative == "" {
			// 如果 narrative 为空，尝试从 structured_output 提取简要信息
			narrative = extractBriefFromStructured(out.StructuredOutput)
		}
		if narrative == "" {
			continue
		}

		if !hasContent {
			sb.WriteString("【历史阶段摘要】\n")
			hasContent = true
		}

		// 每个阶段摘要限制在300字以内
		if len([]rune(narrative)) > 300 {
			narrative = string([]rune(narrative)[:300]) + "..."
		}
		sb.WriteString(fmt.Sprintf("· %s：%s\n", stageName, narrative))
	}

	return sb.String()
}

// stageCodeToNameForRepo 阶段代码转中文名（repo层内部使用，避免跨包依赖）
func stageCodeToNameForRepo(code string) string {
	nameMap := map[string]string{
		"analyze": "教学分析",
		"design":  "教学设计",
		"write":   "教案撰写",
		"review":  "AI评审",
		"revise":  "修订定稿",
	}
	if name, ok := nameMap[code]; ok {
		return name
	}
	return code
}

// extractBriefFromStructured 从structured_output JSON中提取简要摘要
//
// 当 narrative_output 为空时，尝试从结构化数据中提取关键信息作为兜底摘要
// 支持的字段（按优先级）：summary > stage > content_markdown(长度信息)
func extractBriefFromStructured(structuredJSON string) string {
	if structuredJSON == "" || structuredJSON == "{}" {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(structuredJSON), &data); err != nil {
		return ""
	}

	// 优先取 summary 字段
	if summary, ok := data["summary"]; ok {
		if s, ok := summary.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}

	// 次优先：如果有 content_markdown，返回长度信息
	if content, ok := data["content_markdown"]; ok {
		if c, ok := content.(string); ok && len(c) > 0 {
			return fmt.Sprintf("已生成教案正文（%d字符）", len(c))
		}
	}

	// 兜底：如果有 total_score（review阶段）
	if score, ok := data["total_score"]; ok {
		return fmt.Sprintf("评审总分：%.1f", score)
	}

	return ""
}
