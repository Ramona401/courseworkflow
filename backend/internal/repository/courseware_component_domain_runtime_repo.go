package repository

// courseware_component_domain_runtime_repo.go — 课件组件运行时域仓储。
//
// 本文件只承载两类运行时读取：
//   1. 按具体课件education_domain快照匹配同域或common组件；
//   2. 读取页面matched_component_ids的最小复核快照。
//
// mixed、common、空值和非法当前域不能进入课件运行时。
// Repository在非法当前域下返回空候选，不执行任何K12回退。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CWComponentAccessRecord 是页面组件ID复核所需的最小数据库快照。
type CWComponentAccessRecord struct {
	ID              string
	EducationDomain string
	IsActive        bool
	ReviewStatus    string
}

// MatchCWComponentsForEducationDomain 按具体课件快照域匹配组件。
func MatchCWComponentsForEducationDomain(
	ctx context.Context,
	request *models.MatchCWComponentsRequest,
	currentDomain string,
) ([]*models.MatchedCWComponentResource, error) {
	if request == nil {
		return []*models.MatchedCWComponentResource{}, nil
	}

	currentDomain = normalizeCWComponentRepositoryDomain(
		currentDomain,
	)

	if !models.IsTeachingEducationDomain(currentDomain) {
		return []*models.MatchedCWComponentResource{}, nil
	}

	limit := request.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 50 {
		limit = 50
	}

	conditions := []string{
		"is_active = true",
		"review_status = 'approved'",
		`(
			education_domain = $1
			OR education_domain = 'common'
		)`,
	}
	args := []interface{}{currentDomain}
	argIndex := 2

	if request.ComponentType != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"component_type = $%d",
				argIndex,
			),
		)
		args = append(args, request.ComponentType)
		argIndex++
	}

	if request.SubjectScope != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(subject_scope = $%d OR subject_scope = 'ALL')",
				argIndex,
			),
		)
		args = append(args, request.SubjectScope)
		argIndex++
	}

	if request.GradeScope != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(grade_scope = $%d OR grade_scope = 'ALL')",
				argIndex,
			),
		)
		args = append(args, request.GradeScope)
		argIndex++
	}

	if request.InteractionLevel > 0 {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(idx_interaction_level IS NULL OR ABS(idx_interaction_level - $%d) <= 1)",
				argIndex,
			),
		)
		args = append(args, request.InteractionLevel)
		argIndex++
	}

	if request.VisualFormat != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(idx_visual_format = '' OR idx_visual_format = $%d)",
				argIndex,
			),
		)
		args = append(args, request.VisualFormat)
		argIndex++
	}

	if request.TechTag != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(idx_tech_tag = '' OR idx_tech_tag = $%d)",
				argIndex,
			),
		)
		args = append(args, request.TechTag)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(
		`
			SELECT
				id,
				education_domain,
				name,
				component_type,
				code_content,
				COALESCE(preview_html, ''),
				COALESCE(component_index, ''),
				idx_interaction_level
			FROM courseware_components
			WHERE %s
			ORDER BY
				CASE
					WHEN subject_scope != 'ALL'
					 AND grade_scope != 'ALL'
					THEN 0
					WHEN subject_scope != 'ALL'
					  OR grade_scope != 'ALL'
					THEN 1
					ELSE 2
				END ASC,
				CASE
					WHEN education_domain = $1
					THEN 0
					ELSE 1
				END ASC,
				created_at DESC
			LIMIT $%d
		`,
		whereClause,
		argIndex,
	)

	args = append(args, limit)

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按教育域匹配课件组件失败: %w",
			err,
		)
	}
	defer rows.Close()

	matched := make(
		[]*models.MatchedCWComponentResource,
		0,
	)

	for rows.Next() {
		item := &models.MatchedCWComponentResource{
			MatchedCWComponent: &models.MatchedCWComponent{},
		}

		if err := rows.Scan(
			&item.ID,
			&item.EducationDomain,
			&item.Name,
			&item.ComponentType,
			&item.CodeContent,
			&item.PreviewHTML,
			&item.ComponentIndex,
			&item.InteractionLevel,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教育域课件组件匹配结果失败: %w",
				err,
			)
		}

		matched = append(matched, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教育域课件组件匹配结果失败: %w",
			err,
		)
	}

	return matched, nil
}

// GetCWComponentAccessRecordsByIDs 读取页面组件ID复核快照。
//
// 使用id::text匹配，历史JSON即使含非UUID字符串也不会触发CAST错误。
func GetCWComponentAccessRecordsByIDs(
	ctx context.Context,
	componentIDs []string,
) ([]*CWComponentAccessRecord, error) {
	if len(componentIDs) == 0 {
		return []*CWComponentAccessRecord{}, nil
	}

	rows, err := database.DB.Query(
		ctx,
		`
			SELECT
				id::text,
				education_domain,
				is_active,
				review_status
			FROM courseware_components
			WHERE id::text = ANY($1::text[])
		`,
		componentIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件组件ID复核快照失败: %w",
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]*CWComponentAccessRecord,
		0,
		len(componentIDs),
	)

	for rows.Next() {
		record := &CWComponentAccessRecord{}

		if err := rows.Scan(
			&record.ID,
			&record.EducationDomain,
			&record.IsActive,
			&record.ReviewStatus,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课件组件ID复核快照失败: %w",
				err,
			)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件组件ID复核快照失败: %w",
			err,
		)
	}

	return records, nil
}
