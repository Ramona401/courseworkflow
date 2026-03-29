package repository

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
	ErrComponentNotFound = errors.New("组件不存在")
)

// ==================== 组件 CRUD ====================

// CreateComponent 创建组件
func CreateComponent(ctx context.Context, c *models.LessonPlanComponent) error {
	query := `
		INSERT INTO lesson_plan_components (
			library_type, subject, grade_range, tags, injection_mode,
			display_label, COALESCE(design_logic, ''), COALESCE(example_snippet, ''), COALESCE(full_guide, ''), content,
			source, source_ref, scope, scope_ref_id, created_by, review_status, status
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id, created_at, updated_at
	`
	// 设置默认值
	subject := c.Subject
	if subject == "" {
		subject = "general"
	}
	tags := c.Tags
	if tags == "" {
		tags = "[]"
	}
	injectionMode := c.InjectionMode
	if injectionMode == "" {
		injectionMode = "on_demand"
	}
	content := c.Content
	if content == "" {
		content = "{}"
	}
	source := c.Source
	if source == "" {
		source = "manual"
	}
	scope := c.Scope
	if scope == "" {
		scope = "global"
	}
	reviewStatus := c.ReviewStatus
	if reviewStatus == "" {
		reviewStatus = "approved"
	}

	err := database.DB.QueryRow(ctx, query,
		c.LibraryType, subject, c.GradeRange, tags, injectionMode,
		c.DisplayLabel, c.DesignLogic, c.ExampleSnippet, c.FullGuide, content,
		source, c.SourceRef, scope, c.ScopeRefID, c.CreatedBy, reviewStatus, "active",
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建组件失败: %w", err)
	}
	return nil
}

// GetComponentByID 根据ID查询组件
func GetComponentByID(ctx context.Context, id string) (*models.LessonPlanComponent, error) {
	c := &models.LessonPlanComponent{}
	query := `
		SELECT id, library_type, subject, COALESCE(grade_range, ''), tags, injection_mode,
		       display_label, COALESCE(design_logic, ''), COALESCE(example_snippet, ''), COALESCE(full_guide, ''), content,
		       source, COALESCE(source_ref, ''), quality_score, usage_count, select_count,
		       like_count, dislike_count, scope, scope_ref_id,
		       created_by, review_status, reviewed_by, reviewed_at,
		       status, created_at, updated_at
		FROM lesson_plan_components WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.LibraryType, &c.Subject, &c.GradeRange, &c.Tags, &c.InjectionMode,
		&c.DisplayLabel, &c.DesignLogic, &c.ExampleSnippet, &c.FullGuide, &c.Content,
		&c.Source, &c.SourceRef, &c.QualityScore, &c.UsageCount, &c.SelectCount,
		&c.LikeCount, &c.DislikeCount, &c.Scope, &c.ScopeRefID,
		&c.CreatedBy, &c.ReviewStatus, &c.ReviewedBy, &c.ReviewedAt,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrComponentNotFound
		}
		return nil, fmt.Errorf("查询组件失败: %w", err)
	}
	return c, nil
}

// ListComponents 获取组件列表（支持多条件筛选）
func ListComponents(ctx context.Context, libraryType string, subject string, reviewStatus string, scope string, limit int, offset int) ([]*models.ComponentListItem, int, error) {
	// 构建WHERE条件
	where := " WHERE c.status = 'active'"
	args := []interface{}{}
	argIdx := 1

	if libraryType != "" {
		where += fmt.Sprintf(" AND c.library_type = $%d", argIdx)
		args = append(args, libraryType)
		argIdx++
	}
	if subject != "" {
		where += fmt.Sprintf(" AND (c.subject = $%d OR c.subject = 'general')", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if reviewStatus != "" {
		where += fmt.Sprintf(" AND c.review_status = $%d", argIdx)
		args = append(args, reviewStatus)
		argIdx++
	}
	if scope != "" {
		where += fmt.Sprintf(" AND c.scope = $%d", argIdx)
		args = append(args, scope)
		argIdx++
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM lesson_plan_components c" + where
	var total int
	err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询组件总数失败: %w", err)
	}

	// 查询列表
	if limit <= 0 {
		limit = 50
	}
	listQuery := `
		SELECT c.id, c.library_type, c.subject, COALESCE(c.grade_range, ''), c.injection_mode,
		       c.display_label, c.quality_score, c.usage_count, c.select_count,
		       c.source, c.review_status, c.scope, c.status, c.created_at
		FROM lesson_plan_components c
	` + where + fmt.Sprintf(" ORDER BY c.quality_score DESC, c.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询组件列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ComponentListItem
	for rows.Next() {
		item := &models.ComponentListItem{}
		err := rows.Scan(
			&item.ID, &item.LibraryType, &item.Subject, &item.GradeRange, &item.InjectionMode,
			&item.DisplayLabel, &item.QualityScore, &item.UsageCount, &item.SelectCount,
			&item.Source, &item.ReviewStatus, &item.Scope, &item.Status, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描组件行失败: %w", err)
		}
		// 填充中文名
		item.LibraryName = models.LibraryTypeNameMap[item.LibraryType]
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateComponent 更新组件
func UpdateComponent(ctx context.Context, id string, req *models.UpdateComponentRequest) error {
	query := `
		UPDATE lesson_plan_components
		SET subject = $1, grade_range = $2, tags = $3, injection_mode = $4,
		    display_label = $5, design_logic = $6, example_snippet = $7,
		    full_guide = $8, content = $9, scope = $10, scope_ref_id = $11,
		    status = $12, updated_at = $13
		WHERE id = $14
	`
	tags := req.Tags
	if tags == "" {
		tags = "[]"
	}
	content := req.Content
	if content == "" {
		content = "{}"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, query,
		req.Subject, req.GradeRange, tags, req.InjectionMode,
		req.DisplayLabel, req.DesignLogic, req.ExampleSnippet,
		req.FullGuide, content, req.Scope, req.ScopeRefID,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新组件失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}
	return nil
}

// DeleteComponent 删除组件（软删除：设status=archived）
func DeleteComponent(ctx context.Context, id string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plan_components SET status = 'archived', updated_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("删除组件失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}
	return nil
}

// ReviewComponent 审核组件（教研组长/骨干操作）
func ReviewComponent(ctx context.Context, id string, reviewerID string, decision string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plan_components
		 SET review_status = $1, reviewed_by = $2, reviewed_at = $3, updated_at = $3
		 WHERE id = $4`,
		decision, reviewerID, now, id,
	)
	if err != nil {
		return fmt.Errorf("审核组件失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}
	return nil
}

// ==================== 组件匹配引擎 ====================

// MatchComponents 根据学科+学段匹配组件（核心匹配接口）
// 返回按library_type分组的匹配结果，每组按quality_score降序排列
func MatchComponents(ctx context.Context, req *models.MatchComponentsRequest) ([]*models.MatchedComponentGroup, error) {
	// 构建WHERE条件
	where := " WHERE c.status = 'active' AND c.review_status = 'approved'"
	args := []interface{}{}
	argIdx := 1

	// 学科匹配（包含通用general）
	if req.Subject != "" {
		where += fmt.Sprintf(" AND (c.subject = $%d OR c.subject = 'general')", argIdx)
		args = append(args, req.Subject)
		argIdx++
	}

	// 学段匹配
	if req.GradeRange != "" {
		where += fmt.Sprintf(" AND (c.grade_range = $%d OR c.grade_range IS NULL OR c.grade_range = '')", argIdx)
		args = append(args, req.GradeRange)
		argIdx++
	}

	// 注入模式筛选
	if req.InjectionMode != "" {
		where += fmt.Sprintf(" AND c.injection_mode = $%d", argIdx)
		args = append(args, req.InjectionMode)
		argIdx++
	}

	// 指定组件库类型筛选
	if len(req.LibraryTypes) > 0 {
		where += fmt.Sprintf(" AND c.library_type = ANY($%d)", argIdx)
		args = append(args, req.LibraryTypes)
		argIdx++
	}

	// 标签匹配（JSONB包含查询）
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			where += fmt.Sprintf(" AND c.tags @> $%d::jsonb", argIdx)
			args = append(args, fmt.Sprintf(`["%s"]`, tag))
			argIdx++
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// 查询：按library_type分组，每组取前N条
	// 使用ROW_NUMBER()窗口函数实现分组取TopN
	query := fmt.Sprintf(`
		SELECT library_type, id, display_label, COALESCE(design_logic, ''), COALESCE(example_snippet, ''),
		       COALESCE(full_guide, ''), quality_score, usage_count, select_count, tags
		FROM (
			SELECT c.library_type, c.id, c.display_label, COALESCE(c.design_logic, ''), COALESCE(c.example_snippet, ''),
			       COALESCE(c.full_guide, ''), c.quality_score, c.usage_count, c.select_count, c.tags,
			       ROW_NUMBER() OVER (PARTITION BY c.library_type ORDER BY c.quality_score DESC, c.select_count DESC) AS rn
			FROM lesson_plan_components c
			%s
		) ranked
		WHERE rn <= $%d
		ORDER BY library_type, quality_score DESC
	`, where, argIdx)
	args = append(args, limit)

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("匹配组件失败: %w", err)
	}
	defer rows.Close()

	// 按library_type分组
	groupMap := make(map[string]*models.MatchedComponentGroup)
	var groupOrder []string

	for rows.Next() {
		var libraryType string
		mc := &models.MatchedComponent{}
		err := rows.Scan(
			&libraryType, &mc.ID, &mc.DisplayLabel, &mc.DesignLogic, &mc.ExampleSnippet,
			&mc.FullGuide, &mc.QualityScore, &mc.UsageCount, &mc.SelectCount, &mc.Tags,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描匹配结果行失败: %w", err)
		}

		group, exists := groupMap[libraryType]
		if !exists {
			group = &models.MatchedComponentGroup{
				LibraryType: libraryType,
				LibraryName: models.LibraryTypeNameMap[libraryType],
				Components:  []*models.MatchedComponent{},
			}
			groupMap[libraryType] = group
			groupOrder = append(groupOrder, libraryType)
		}
		group.Components = append(group.Components, mc)
	}

	// 按发现顺序返回
	var result []*models.MatchedComponentGroup
	for _, lt := range groupOrder {
		result = append(result, groupMap[lt])
	}
	return result, nil
}

// ==================== 组件统计更新 ====================

// IncrementComponentUsage 增加组件使用次数
func IncrementComponentUsage(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plan_components SET usage_count = usage_count + 1, updated_at = now() WHERE id = $1`, id)
	return err
}

// IncrementComponentSelect 增加组件选中次数
func IncrementComponentSelect(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plan_components SET select_count = select_count + 1, updated_at = now() WHERE id = $1`, id)
	return err
}

// UpdateComponentQualityScore 更新组件质量分
// quality_score = (select_count / max(usage_count, 1)) × 0.4
//               + (avg_linked_plan_score / 10) × 0.4
//               + ((like_count - dislike_count) / max(like+dislike, 1)) × 0.2
func UpdateComponentQualityScore(ctx context.Context, id string, avgLinkedPlanScore float64) error {
	query := `
		UPDATE lesson_plan_components
		SET quality_score = (
			(CAST(select_count AS NUMERIC) / GREATEST(usage_count, 1)) * 0.4
			+ ($1 / 10.0) * 0.4
			+ (CAST(like_count - dislike_count AS NUMERIC) / GREATEST(like_count + dislike_count, 1)) * 0.2
		), updated_at = now()
		WHERE id = $2
	`
	_, err := database.DB.Exec(ctx, query, avgLinkedPlanScore, id)
	if err != nil {
		return fmt.Errorf("更新组件质量分失败: %w", err)
	}
	return nil
}

// GetComponentLinkedPlanAvgScore 计算组件关联教案的平均AI评审分（Phase5新增）
// 通过 component_extractions 关联表，取萃取自的教案的 ai_review_score 均值
// 若无关联教案或分数为空，返回 0
func GetComponentLinkedPlanAvgScore(ctx context.Context, componentID string) (float64, error) {
	var avg float64
	err := database.DB.QueryRow(ctx, `
		SELECT COALESCE(AVG(lp.ai_review_score), 0)
		FROM component_extractions ce
		JOIN lesson_plans lp ON lp.id = ce.source_lesson_plan_id
		WHERE ce.extracted_component_id = $1
		  AND lp.ai_review_score IS NOT NULL
	`, componentID).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("查询组件关联教案均分失败: %w", err)
	}
	return avg, nil
}
