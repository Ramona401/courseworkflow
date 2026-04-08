package repository

// lesson_plan_repo.go — 教案数据访问层（主文件）
//
// 职责：
//   - 错误常量
//   - 教案CRUD（创建/查询/列表/更新内容/更新状态/更新可见范围/更新评审/Fork/删除）
//   - 教案评审CRUD（创建/列表）
//
// v56修改：CreateLessonPlan/GetLessonPlanByID 增加 recipe_id 字段
// v58修改：GetLessonPlanByID 增加 current_stage + stage_config 字段
// 迭代7B修改：CreateLessonPlan 增加 textbook_page_ids 字段
//             GetLessonPlanByID 增加 textbook_page_ids 字段
// 提示词模板+组件萃取+对话记录 → lesson_plan_repo_ext.go

import (
	"context"
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

// ==================== 教案CRUD ====================

// CreateLessonPlan 创建教案
// v56：新增recipe_id字段
// 迭代7B：新增textbook_page_ids字段
func CreateLessonPlan(ctx context.Context, lp *models.LessonPlan) error {
	query := `
		INSERT INTO lesson_plans (
			title, subject, grade, topic, duration_minutes,
			content_markdown, content_structured, generation_config,
			matched_components, conversation_log,
			status, visibility, author_id, group_id, school_id, template_id, recipe_id,
			textbook_page_ids
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17,
			$18
		)
		RETURNING id, created_at, updated_at
	`
	dur := lp.DurationMinutes
	if dur <= 0 {
		dur = 45
	}
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
	// 迭代7B：课本图片ID列表默认空数组
	textbookIDs := lp.TextbookPageIDs
	if textbookIDs == "" {
		textbookIDs = "[]"
	}

	err := database.DB.QueryRow(ctx, query,
		lp.Title, lp.Subject, lp.Grade, lp.Topic, dur,
		lp.ContentMarkdown, contentStruct, genConfig, matchedComp, convLog,
		status, visibility, lp.AuthorID, lp.GroupID, lp.SchoolID, lp.TemplateID, lp.RecipeID,
		textbookIDs,
	).Scan(&lp.ID, &lp.CreatedAt, &lp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建教案失败: %w", err)
	}
	return nil
}

// GetLessonPlanByID 根据ID查询教案
// v56：新增recipe_id字段
// v58：新增current_stage + stage_config字段
// 迭代7B：新增textbook_page_ids字段
func GetLessonPlanByID(ctx context.Context, id string) (*models.LessonPlan, error) {
	lp := &models.LessonPlan{}
	query := `
		SELECT id, title, subject, grade, topic, duration_minutes,
		       content_markdown, content_structured, generation_config,
		       matched_components, conversation_log,
		       ai_review_score, ai_review_result, ai_review_history,
		       status, visibility, author_id, group_id, school_id,
		       forked_from, fork_count, template_id, recipe_id,
		       view_count, use_count, version,
		       current_stage, COALESCE(stage_config::text, '[]'),
		       COALESCE(textbook_page_ids::text, '[]'),
                       COALESCE(lesson_index, ''), idx_cognitive_level, idx_pedagogy_intensity,
                       idx_structure_type, idx_quality_level,
		       created_at, updated_at
		FROM lesson_plans WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&lp.ID, &lp.Title, &lp.Subject, &lp.Grade, &lp.Topic, &lp.DurationMinutes,
		&lp.ContentMarkdown, &lp.ContentStructured, &lp.GenerationConfig,
		&lp.MatchedComponents, &lp.ConversationLog,
		&lp.AIReviewScore, &lp.AIReviewResult, &lp.AIReviewHistory,
		&lp.Status, &lp.Visibility, &lp.AuthorID, &lp.GroupID, &lp.SchoolID,
		&lp.ForkedFrom, &lp.ForkCount, &lp.TemplateID, &lp.RecipeID,
		&lp.ViewCount, &lp.UseCount, &lp.Version,
		&lp.CurrentStage, &lp.StageConfig,
		&lp.TextbookPageIDs,
                &lp.LessonIndex, &lp.IdxCognitiveLevel, &lp.IdxPedagogyIntensity,
                &lp.IdxStructureType, &lp.IdxQualityLevel,
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

// ListLessonPlans 获取教案列表（支持多条件筛选+分页）
func ListLessonPlans(ctx context.Context, authorID string, groupID string, status string, subject string, grade string, limit int, offset int, qualityLevel int, structureType int, cognitiveLevel int, pedagogyIntensity int) ([]*models.LessonPlanListItem, int, error) {
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

	// v86新增：AOCI索引维度筛选
	if qualityLevel > 0 {
		where += fmt.Sprintf(" AND lp.idx_quality_level >= $%d", argIdx)
		args = append(args, qualityLevel)
		argIdx++
	}
	if structureType > 0 {
		where += fmt.Sprintf(" AND lp.idx_structure_type = $%d", argIdx)
		args = append(args, structureType)
		argIdx++
	}
	if cognitiveLevel > 0 {
		where += fmt.Sprintf(" AND lp.idx_cognitive_level >= $%d", argIdx)
		args = append(args, cognitiveLevel)
		argIdx++
	}
	if pedagogyIntensity > 0 {
		where += fmt.Sprintf(" AND lp.idx_pedagogy_intensity = $%d", argIdx)
		args = append(args, pedagogyIntensity)
		argIdx++
	}

	var total int
	if err := database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM lesson_plans lp"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询教案总数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	listQuery := fmt.Sprintf(`
		SELECT lp.id, lp.title, lp.subject, lp.grade, lp.topic, lp.duration_minutes,
		       lp.status, lp.visibility, lp.author_id,
		       COALESCE(u.display_name, '') AS author_name,
		       lp.ai_review_score, lp.fork_count, lp.view_count,
		       lp.forked_from, lp.recipe_id,
		       COALESCE(tr.name, '') AS recipe_name,
                       COALESCE(lp.lesson_index, '') AS lesson_index,
                       lp.idx_quality_level,
		       lp.created_at, lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u ON u.id = lp.author_id
		LEFT JOIN teaching_recipes tr ON tr.id = lp.recipe_id
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
			&item.ForkedFrom, &item.RecipeID, &item.RecipeName, &item.LessonIndex, &item.IdxQualityLevel,
			&item.CreatedAt, &item.UpdatedAt,
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
	source, err := GetLessonPlanByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
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
		RecipeID:          source.RecipeID,
		TextbookPageIDs:   source.TextbookPageIDs, // 迭代7B：fork时保留课本关联
	}
	if err := CreateLessonPlan(ctx, newLP); err != nil {
		return nil, err
	}
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

// ==================== 教案评审CRUD ====================

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


// ==================== v86新增：教案索引写入 ====================

// UpdateLessonPlanIndex 更新教案的AOCI索引（索引文本+冗余列）
func UpdateLessonPlanIndex(ctx context.Context, planID string, indexText string, cogLevel int, pedIntensity int, structType int, qualLevel int) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET lesson_index = $1,
		    idx_cognitive_level = $2,
		    idx_pedagogy_intensity = $3,
		    idx_structure_type = $4,
		    idx_quality_level = $5,
		    updated_at = $6
		WHERE id = $7
	`, indexText, cogLevel, pedIntensity, structType, qualLevel, now, planID)
	if err != nil {
		return fmt.Errorf("更新教案索引失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}
