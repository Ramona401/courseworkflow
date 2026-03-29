package repository

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

// ==================== 错误常量 ====================

var (
	ErrLessonPlanNotFound = errors.New("教案不存在")
	ErrTemplateNotFound   = errors.New("提示词模板不存在")
	ErrExtractionNotFound = errors.New("萃取记录不存在")
)

// ==================== 教案 CRUD ====================

// CreateLessonPlan 创建教案
func CreateLessonPlan(ctx context.Context, lp *models.LessonPlan) error {
	query := `
		INSERT INTO lesson_plans (
			title, subject, grade, topic, duration_minutes,
			content_markdown, content_structured, generation_config,
			matched_components, conversation_log,
			status, visibility, author_id, group_id, school_id, template_id
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`
	// 设置默认值
	dur := lp.DurationMinutes
	if dur <= 0 {
		dur = 45
	}
	contentMd := lp.ContentMarkdown
	contentStruct := lp.ContentStructured
	if contentStruct == "" {
		contentStruct = "{}"
	}
	genConfig := lp.GenerationConfig
	if genConfig == "" {
		genConfig = "{}"
	}
	matchedComp := lp.MatchedComponents
	if matchedComp == "" {
		matchedComp = "[]"
	}
	convLog := lp.ConversationLog
	if convLog == "" {
		convLog = "[]"
	}
	status := lp.Status
	if status == "" {
		status = "draft"
	}
	visibility := lp.Visibility
	if visibility == "" {
		visibility = "personal"
	}

	err := database.DB.QueryRow(ctx, query,
		lp.Title, lp.Subject, lp.Grade, lp.Topic, dur,
		contentMd, contentStruct, genConfig, matchedComp, convLog,
		status, visibility, lp.AuthorID, lp.GroupID, lp.SchoolID, lp.TemplateID,
	).Scan(&lp.ID, &lp.CreatedAt, &lp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建教案失败: %w", err)
	}
	return nil
}

// GetLessonPlanByID 根据ID查询教案
func GetLessonPlanByID(ctx context.Context, id string) (*models.LessonPlan, error) {
	lp := &models.LessonPlan{}
	query := `
		SELECT id, title, subject, grade, topic, duration_minutes,
		       content_markdown, content_structured, generation_config,
		       matched_components, conversation_log,
		       ai_review_score, ai_review_result, ai_review_history,
		       status, visibility, author_id, group_id, school_id,
		       forked_from, fork_count, template_id,
		       view_count, use_count, version,
		       created_at, updated_at
		FROM lesson_plans WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&lp.ID, &lp.Title, &lp.Subject, &lp.Grade, &lp.Topic, &lp.DurationMinutes,
		&lp.ContentMarkdown, &lp.ContentStructured, &lp.GenerationConfig,
		&lp.MatchedComponents, &lp.ConversationLog,
		&lp.AIReviewScore, &lp.AIReviewResult, &lp.AIReviewHistory,
		&lp.Status, &lp.Visibility, &lp.AuthorID, &lp.GroupID, &lp.SchoolID,
		&lp.ForkedFrom, &lp.ForkCount, &lp.TemplateID,
		&lp.ViewCount, &lp.UseCount, &lp.Version,
		&lp.CreatedAt, &lp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf("查询教案失败: %w", err)
	}
	return lp, nil
}

// ListLessonPlans 获取教案列表（支持多条件筛选）
func ListLessonPlans(ctx context.Context, authorID string, groupID string, status string, subject string, grade string, limit int, offset int) ([]*models.LessonPlanListItem, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if authorID != "" {
		where += fmt.Sprintf(" AND lp.author_id = $%d", argIdx)
		args = append(args, authorID)
		argIdx++
	}
	if groupID != "" {
		where += fmt.Sprintf(" AND lp.group_id = $%d", argIdx)
		args = append(args, groupID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND lp.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if subject != "" {
		where += fmt.Sprintf(" AND lp.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if grade != "" {
		where += fmt.Sprintf(" AND lp.grade = $%d", argIdx)
		args = append(args, grade)
		argIdx++
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM lesson_plans lp" + where
	var total int
	err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询教案总数失败: %w", err)
	}

	// 查询列表
	if limit <= 0 {
		limit = 20
	}
	listQuery := fmt.Sprintf(`
		SELECT lp.id, lp.title, lp.subject, lp.grade, lp.topic, lp.duration_minutes,
		       lp.status, lp.visibility, lp.author_id,
		       COALESCE(u.display_name, '') AS author_name,
		       lp.ai_review_score, lp.fork_count, lp.view_count,
		       lp.forked_from, lp.created_at, lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u ON u.id = lp.author_id
		%s
		ORDER BY lp.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询教案列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.LessonPlanListItem
	for rows.Next() {
		item := &models.LessonPlanListItem{}
		err := rows.Scan(
			&item.ID, &item.Title, &item.Subject, &item.Grade, &item.Topic, &item.DurationMinutes,
			&item.Status, &item.Visibility, &item.AuthorID, &item.AuthorName,
			&item.AIReviewScore, &item.ForkCount, &item.ViewCount,
			&item.ForkedFrom, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描教案行失败: %w", err)
		}
		item.StatusName = models.LPStatusNameMap[item.Status]
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateLessonPlanContent 更新教案内容
func UpdateLessonPlanContent(ctx context.Context, id string, title string, contentMd string, contentStruct string, durMinutes int) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET title = $1, content_markdown = $2, content_structured = $3,
		    duration_minutes = $4, version = version + 1, updated_at = $5
		WHERE id = $6
	`, title, contentMd, contentStruct, durMinutes, now, id)
	if err != nil {
		return fmt.Errorf("更新教案内容失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanStatus 更新教案状态
func UpdateLessonPlanStatus(ctx context.Context, id string, status string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET status = $1, updated_at = $2 WHERE id = $3`,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案状态失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanVisibility 更新教案可见范围
func UpdateLessonPlanVisibility(ctx context.Context, id string, visibility string, groupID *string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET visibility = $1, group_id = $2, updated_at = $3 WHERE id = $4`,
		visibility, groupID, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案可见范围失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanAIReview 更新教案AI评审结果
func UpdateLessonPlanAIReview(ctx context.Context, id string, score float64, result string, history string) error {
	now := time.Now()
	res, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET ai_review_score = $1, ai_review_result = $2, ai_review_history = $3, updated_at = $4
		WHERE id = $5
	`, score, result, history, now, id)
	if err != nil {
		return fmt.Errorf("更新AI评审结果失败: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ForkLessonPlan 复制教案（fork）
func ForkLessonPlan(ctx context.Context, sourceID string, newAuthorID string) (*models.LessonPlan, error) {
	// 先获取源教案
	source, err := GetLessonPlanByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	// 创建新教案
	newLP := &models.LessonPlan{
		Title:             source.Title + "（副本）",
		Subject:           source.Subject,
		Grade:             source.Grade,
		Topic:             source.Topic,
		DurationMinutes:   source.DurationMinutes,
		ContentMarkdown:   source.ContentMarkdown,
		ContentStructured: source.ContentStructured,
		GenerationConfig:  source.GenerationConfig,
		MatchedComponents: source.MatchedComponents,
		Status:            "draft",
		Visibility:        "personal",
		AuthorID:          newAuthorID,
		ForkedFrom:        &sourceID,
		TemplateID:        source.TemplateID,
	}
	err = CreateLessonPlan(ctx, newLP)
	if err != nil {
		return nil, err
	}

	// 增加源教案的fork计数
	_, err = database.DB.Exec(ctx,
		`UPDATE lesson_plans SET fork_count = fork_count + 1 WHERE id = $1`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("更新fork计数失败: %w", err)
	}

	return newLP, nil
}

// IncrementLessonPlanView 增加教案浏览次数
func IncrementLessonPlanView(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

// DeleteLessonPlan 删除教案（物理删除，级联删除评审记录）
func DeleteLessonPlan(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM lesson_plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除教案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ==================== 教案评审 CRUD ====================

// CreateLessonPlanReview 创建教案评审记录
func CreateLessonPlanReview(ctx context.Context, review *models.LessonPlanReview) error {
	query := `
		INSERT INTO lesson_plan_reviews (lesson_plan_id, reviewer_id, decision, score, dimensions, comments, suggestions, round)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	dimensions := review.Dimensions
	if dimensions == "" {
		dimensions = "{}"
	}
	suggestions := review.Suggestions
	if suggestions == "" {
		suggestions = "[]"
	}
	err := database.DB.QueryRow(ctx, query,
		review.LessonPlanID, review.ReviewerID, review.Decision,
		review.Score, dimensions, review.Comments, suggestions, review.Round,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建评审记录失败: %w", err)
	}
	return nil
}

// ListLessonPlanReviews 获取教案的评审记录列表
func ListLessonPlanReviews(ctx context.Context, lessonPlanID string) ([]*models.LessonPlanReviewItem, error) {
	query := `
		SELECT r.id, r.reviewer_id, COALESCE(u.display_name, '') AS reviewer_name,
		       r.decision, r.score, r.comments, r.round, r.created_at
		FROM lesson_plan_reviews r
		LEFT JOIN users u ON u.id = r.reviewer_id
		WHERE r.lesson_plan_id = $1
		ORDER BY r.round ASC, r.created_at ASC
	`
	rows, err := database.DB.Query(ctx, query, lessonPlanID)
	if err != nil {
		return nil, fmt.Errorf("查询评审记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.LessonPlanReviewItem
	for rows.Next() {
		item := &models.LessonPlanReviewItem{}
		err := rows.Scan(
			&item.ID, &item.ReviewerID, &item.ReviewerName,
			&item.Decision, &item.Score, &item.Comments, &item.Round, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描评审记录行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// ==================== 提示词模板 CRUD ====================

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
	// 解引用*string字段，NULL保持为NULL传入数据库
	var contextRules, genRules, reviewRules, outputFmt *string
	if pt.ContextRules != nil {
		v := *pt.ContextRules; if v == "" { v = "{}" }; contextRules = &v
	}
	if pt.GenerationRules != nil {
		v := *pt.GenerationRules; if v == "" { v = "{}" }; genRules = &v
	}
	if pt.ReviewRules != nil {
		v := *pt.ReviewRules; if v == "" { v = "{}" }; reviewRules = &v
	}
	if pt.OutputFormat != nil {
		v := *pt.OutputFormat; if v == "" { v = "{}" }; outputFmt = &v
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

// ListPromptTemplates 获取提示词模板列表
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

// UpdatePromptTemplate 更新提示词模板
func UpdatePromptTemplate(ctx context.Context, id string, req *models.UpdatePromptTemplateRequest) error {
	now := time.Now()
	// UpdatePromptTemplateRequest中字段仍是string，空字符串→NULL
	var contextRules, genRules, reviewRules, outputFmt *string
	if req.ContextRules != "" {
		v := req.ContextRules; contextRules = &v
	}
	if req.GenerationRules != "" {
		v := req.GenerationRules; genRules = &v
	}
	if req.ReviewRules != "" {
		v := req.ReviewRules; reviewRules = &v
	}
	if req.OutputFormat != "" {
		v := req.OutputFormat; outputFmt = &v
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

// ResolvePromptTemplateChain 解析提示词模板继承链（从叶到根向上遍历）
// 子级字段非空时覆盖父级，最终返回合并后的完整模板
func ResolvePromptTemplateChain(ctx context.Context, templateID string) (*models.ResolvedPromptTemplate, error) {
	resolved := &models.ResolvedPromptTemplate{}
	var chain []string
	currentID := templateID

	// 向上遍历继承链（最多10层防死循环）
	for i := 0; i < 10 && currentID != ""; i++ {
		pt, err := GetPromptTemplateByID(ctx, currentID)
		if err != nil {
			break
		}
		chain = append([]string{pt.ID}, chain...) // 头插，根在前

		// 子级覆盖父级（非NULL且非空才覆盖）
		// 字段类型为*string，NULL时为nil
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

		// 继续向上
		if pt.ParentTemplateID != nil {
			currentID = *pt.ParentTemplateID
		} else {
			currentID = ""
		}
	}

	resolved.InheritanceChain = chain
	return resolved, nil
}

// ==================== 组件萃取 CRUD ====================

// CreateComponentExtraction 创建组件萃取记录
func CreateComponentExtraction(ctx context.Context, ce *models.ComponentExtraction) error {
	query := `
		INSERT INTO component_extractions (
			source_type, source_lesson_plan_id, source_content,
			extracted_component_id, extraction_type, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	status := ce.Status
	if status == "" {
		status = "pending"
	}
	err := database.DB.QueryRow(ctx, query,
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
	query := `
		SELECT id, source_type, source_lesson_plan_id, source_content,
		       extracted_component_id, extraction_type, status,
		       confirmed_by, confirmed_at, created_by, created_at
		FROM component_extractions
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := database.DB.Query(ctx, query, limit)
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

// ==================== 对话记录 CRUD（Phase3新增） ====================

// AppendConversationMessage 追加一条消息到教案对话记录
// 使用PostgreSQL jsonb_insert保证原子性
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

// GetExtractionByID 根据ID查询萃取记录（Phase5新增）
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
