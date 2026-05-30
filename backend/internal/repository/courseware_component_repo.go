package repository

// courseware_component_repo.go — 课件组件库数据访问层（v139 拆分版）
//
// v139 变更：
//   - 模板相关函数（ListCWTemplates / GetCWTemplateByID / CreateCWTemplate / UpdateCWTemplate /
//     DeleteCWTemplate / ListCWTemplatesWithUser / CreatePersonalTemplate / DeletePersonalTemplate）
//     全部拆出到 courseware_template_repo.go,职责分离更清晰
//   - 本文件只保留组件库（courseware_components 表）CRUD 和匹配引擎
//
// 保留的全部函数（向后兼容,签名不变）：
//   - CreateCWComponent / GetCWComponentByID / ListCWComponents
//   - UpdateCWComponent / UpdateCWComponentIndex / DeleteCWComponent
//   - MatchCWComponents（AOCI 匹配引擎,逻辑保持原样）

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 课件组件库 CRUD ====================

// CreateCWComponent 创建课件组件
func CreateCWComponent(ctx context.Context, comp *models.CoursewareComponent) error {
	sql := `INSERT INTO courseware_components (id, name, description, component_type,
		code_content, preview_image_url, preview_html, subject_scope, grade_scope,
		component_index, idx_interaction_level, idx_visual_format, idx_tech_tag,
		tech_dependencies, tags, is_active, review_status)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14::jsonb, $15, $16)
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, sql,
		comp.Name, comp.Description, comp.ComponentType,
		comp.CodeContent, comp.PreviewImageURL, comp.PreviewHTML,
		comp.SubjectScope, comp.GradeScope,
		comp.ComponentIndex, comp.IdxInteractionLevel, comp.IdxVisualFormat, comp.IdxTechTag,
		nullIfEmpty(comp.TechDependencies), nullIfEmpty(comp.Tags),
		comp.IsActive, comp.ReviewStatus,
	).Scan(&comp.ID, &comp.CreatedAt, &comp.UpdatedAt)
}

// GetCWComponentByID 根据ID获取课件组件详情
func GetCWComponentByID(ctx context.Context, id string) (*models.CoursewareComponent, error) {
	sql := `SELECT id, name, COALESCE(description,''), component_type,
		code_content, COALESCE(preview_image_url,''), COALESCE(preview_html,''),
		COALESCE(subject_scope,'ALL'), COALESCE(grade_scope,'ALL'),
		COALESCE(component_index,''), idx_interaction_level,
		COALESCE(idx_visual_format,''), COALESCE(idx_tech_tag,''),
		COALESCE(tech_dependencies::text,''), COALESCE(tags::text,''),
		is_active, review_status, created_at, updated_at
		FROM courseware_components WHERE id = $1`
	comp := &models.CoursewareComponent{}
	err := database.DB.QueryRow(ctx, sql, id).Scan(
		&comp.ID, &comp.Name, &comp.Description, &comp.ComponentType,
		&comp.CodeContent, &comp.PreviewImageURL, &comp.PreviewHTML,
		&comp.SubjectScope, &comp.GradeScope,
		&comp.ComponentIndex, &comp.IdxInteractionLevel,
		&comp.IdxVisualFormat, &comp.IdxTechTag,
		&comp.TechDependencies, &comp.Tags,
		&comp.IsActive, &comp.ReviewStatus,
		&comp.CreatedAt, &comp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return comp, nil
}

// ListCWComponents 查询课件组件列表（支持类型+学科+学段+状态筛选+分页）
func ListCWComponents(ctx context.Context, componentType string, subjectScope string, gradeScope string, isActive *bool, limit int, offset int) ([]*models.CWComponentListItem, int, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if componentType != "" {
		conditions = append(conditions, fmt.Sprintf("component_type = $%d", argIdx))
		args = append(args, componentType)
		argIdx++
	}
	if subjectScope != "" {
		conditions = append(conditions, fmt.Sprintf("(subject_scope = $%d OR subject_scope = 'ALL')", argIdx))
		args = append(args, subjectScope)
		argIdx++
	}
	if gradeScope != "" {
		conditions = append(conditions, fmt.Sprintf("(grade_scope = $%d OR grade_scope = 'ALL')", argIdx))
		args = append(args, gradeScope)
		argIdx++
	}
	if isActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *isActive)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM courseware_components WHERE %s", whereClause)
	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询课件组件总数失败: %w", err)
	}

	listSQL := fmt.Sprintf(`SELECT id, name, COALESCE(description,''), component_type,
		COALESCE(preview_image_url,''), COALESCE(subject_scope,'ALL'), COALESCE(grade_scope,'ALL'),
		COALESCE(component_index,''), idx_interaction_level, is_active, review_status, created_at
		FROM courseware_components
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件组件列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWComponentListItem
	for rows.Next() {
		item := &models.CWComponentListItem{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.ComponentType,
			&item.PreviewImageURL, &item.SubjectScope, &item.GradeScope,
			&item.ComponentIndex, &item.IdxInteractionLevel,
			&item.IsActive, &item.ReviewStatus, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描课件组件列表行失败: %w", err)
		}
		item.ComponentTypeName = models.CWComponentTypeNameMap[item.ComponentType]
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateCWComponent 更新课件组件
func UpdateCWComponent(ctx context.Context, id string, req *models.UpdateCWComponentRequest) error {
	sql := `UPDATE courseware_components SET name = $1, description = $2, component_type = $3,
		code_content = $4, preview_image_url = $5, preview_html = $6,
		subject_scope = $7, grade_scope = $8,
		tech_dependencies = $9::jsonb, tags = $10::jsonb,
		updated_at = $11
		WHERE id = $12`
	tag, err := database.DB.Exec(ctx, sql,
		req.Name, req.Description, req.ComponentType,
		req.CodeContent, req.PreviewImageURL, req.PreviewHTML,
		req.SubjectScope, req.GradeScope,
		nullIfEmpty(req.TechDependencies), nullIfEmpty(req.Tags),
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("更新课件组件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件组件不存在: %s", id)
	}
	if req.IsActive != nil {
		_, _ = database.DB.Exec(ctx, `UPDATE courseware_components SET is_active = $1 WHERE id = $2`, *req.IsActive, id)
	}
	if req.ReviewStatus != "" {
		_, _ = database.DB.Exec(ctx, `UPDATE courseware_components SET review_status = $1 WHERE id = $2`, req.ReviewStatus, id)
	}
	return nil
}

// UpdateCWComponentIndex 更新组件的AOCI索引（索引压缩后回写）
func UpdateCWComponentIndex(ctx context.Context, id string, indexStr string, interactionLevel *int, visualFormat string, techTag string) error {
	sql := `UPDATE courseware_components SET component_index = $1,
		idx_interaction_level = $2, idx_visual_format = $3, idx_tech_tag = $4,
		updated_at = $5 WHERE id = $6`
	_, err := database.DB.Exec(ctx, sql, indexStr, interactionLevel, visualFormat, techTag, time.Now(), id)
	return err
}

// DeleteCWComponent 删除课件组件（物理删除）
func DeleteCWComponent(ctx context.Context, id string) error {
	sql := `DELETE FROM courseware_components WHERE id = $1`
	tag, err := database.DB.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("删除课件组件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件组件不存在: %s", id)
	}
	return nil
}

// ==================== 课件组件匹配引擎 ====================

// MatchCWComponents 根据交互类型+视觉形式+学科+学段匹配课件组件
// 按交互复杂度 ±1 浮动匹配,视觉形式精确匹配或通配('')
// 按学科+学段精度排序(精确匹配优先于 ALL 通配)
func MatchCWComponents(ctx context.Context, req *models.MatchCWComponentsRequest) ([]*models.MatchedCWComponent, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	conditions := []string{"is_active = true", "review_status = 'approved'"}
	args := []interface{}{}
	argIdx := 1

	if req.ComponentType != "" {
		conditions = append(conditions, fmt.Sprintf("component_type = $%d", argIdx))
		args = append(args, req.ComponentType)
		argIdx++
	}
	if req.SubjectScope != "" {
		conditions = append(conditions, fmt.Sprintf("(subject_scope = $%d OR subject_scope = 'ALL')", argIdx))
		args = append(args, req.SubjectScope)
		argIdx++
	}
	if req.GradeScope != "" {
		conditions = append(conditions, fmt.Sprintf("(grade_scope = $%d OR grade_scope = 'ALL')", argIdx))
		args = append(args, req.GradeScope)
		argIdx++
	}
	if req.InteractionLevel > 0 {
		conditions = append(conditions, fmt.Sprintf("(idx_interaction_level IS NULL OR ABS(idx_interaction_level - $%d) <= 1)", argIdx))
		args = append(args, req.InteractionLevel)
		argIdx++
	}
	if req.VisualFormat != "" {
		conditions = append(conditions, fmt.Sprintf("(idx_visual_format = '' OR idx_visual_format = $%d)", argIdx))
		args = append(args, req.VisualFormat)
		argIdx++
	}
	if req.TechTag != "" {
		conditions = append(conditions, fmt.Sprintf("(idx_tech_tag = '' OR idx_tech_tag = $%d)", argIdx))
		args = append(args, req.TechTag)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")
	orderClause := `ORDER BY
		CASE WHEN subject_scope != 'ALL' AND grade_scope != 'ALL' THEN 0
		     WHEN subject_scope != 'ALL' OR grade_scope != 'ALL' THEN 1
		     ELSE 2 END ASC,
		created_at DESC`

	sql := fmt.Sprintf(`SELECT id, name, component_type, code_content,
		COALESCE(preview_html,''), COALESCE(component_index,''), idx_interaction_level
		FROM courseware_components
		WHERE %s %s LIMIT $%d`, whereClause, orderClause, argIdx)
	args = append(args, limit)

	rows, err := database.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("匹配课件组件失败: %w", err)
	}
	defer rows.Close()

	var matched []*models.MatchedCWComponent
	for rows.Next() {
		m := &models.MatchedCWComponent{}
		if err := rows.Scan(
			&m.ID, &m.Name, &m.ComponentType, &m.CodeContent,
			&m.PreviewHTML, &m.ComponentIndex, &m.InteractionLevel,
		); err != nil {
			return nil, fmt.Errorf("扫描匹配结果行失败: %w", err)
		}
		matched = append(matched, m)
	}
	return matched, nil
}
