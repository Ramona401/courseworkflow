package repository

// component_match_domain_repo.go — 组件教育域普通匹配仓储。
//
// 运行时匹配规则：
//   - currentDomain必须是k12、vocational或adult；
//   - 只返回与当前域相同或common的组件；
//   - mixed、common、空值和非法当前域均返回空结果；
//   - 只返回active且approved的组件；
//   - 返回结果显式携带education_domain。
//
// 年级匹配兼容两类语义：
//   - 纯数字或数字范围：继续执行原有范围包含匹配；
//   - 职教、成教等非数字层级：执行trim后的精确文本匹配；
//   - 组件grade_range为空：视为不限层级。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// buildDomainGradeRangeCondition 构建跨教育域安全年级条件。
//
// 只有组件值和请求值都满足纯数字或数字范围格式时才执行整数转换，
// 避免“第一学期”“初级班”等非K12文本触发数据库CAST错误。
func buildDomainGradeRangeCondition(
	argIndex int,
) string {
	return fmt.Sprintf(
		`
			AND (
				c.grade_range IS NULL
				OR btrim(c.grade_range) = ''
				OR btrim(c.grade_range) = btrim($%d)
				OR (
					btrim(c.grade_range) ~ '^[0-9]+(-[0-9]+)?$'
					AND btrim($%d) ~ '^[0-9]+(-[0-9]+)?$'
					AND
					split_part(
						btrim(c.grade_range),
						'-',
						1
					)::integer
					<=
					split_part(
						btrim($%d),
						'-',
						1
					)::integer
					AND
					CASE
						WHEN position(
							'-' IN btrim(c.grade_range)
						) > 0
						THEN split_part(
							btrim(c.grade_range),
							'-',
							2
						)::integer
						ELSE btrim(c.grade_range)::integer
					END
					>=
					split_part(
						btrim($%d),
						'-',
						1
					)::integer
				)
			)
		`,
		argIndex,
		argIndex,
		argIndex,
		argIndex,
	)
}

// componentTagJSON 把单个标签安全编码成JSONB包含查询参数。
func componentTagJSON(
	tag string,
) (string, error) {
	encoded, err := json.Marshal(
		[]string{tag},
	)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

// MatchComponentsForEducationDomain 按具体教学域普通匹配组件。
func MatchComponentsForEducationDomain(
	ctx context.Context,
	req *models.MatchComponentsRequest,
	currentDomain string,
) ([]*models.MatchedComponentGroup, error) {
	where := `
		WHERE c.status = 'active'
		  AND c.review_status = 'approved'
		  AND $1 IN (
			'k12',
			'vocational',
			'adult'
		  )
		  AND (
			c.education_domain = $1
			OR c.education_domain = 'common'
		  )
	`

	args := []interface{}{
		currentDomain,
	}
	argIndex := 2

	if strings.TrimSpace(req.Subject) != "" {
		where += fmt.Sprintf(
			" AND (c.subject = $%d OR c.subject = 'general')",
			argIndex,
		)
		args = append(
			args,
			strings.TrimSpace(req.Subject),
		)
		argIndex++
	}

	if strings.TrimSpace(req.GradeRange) != "" {
		where += buildDomainGradeRangeCondition(
			argIndex,
		)
		args = append(
			args,
			strings.TrimSpace(req.GradeRange),
		)
		argIndex++
	}

	if strings.TrimSpace(req.InjectionMode) != "" {
		where += fmt.Sprintf(
			" AND c.injection_mode = $%d",
			argIndex,
		)
		args = append(
			args,
			strings.TrimSpace(req.InjectionMode),
		)
		argIndex++
	}

	if len(req.LibraryTypes) > 0 {
		where += fmt.Sprintf(
			" AND c.library_type = ANY($%d)",
			argIndex,
		)
		args = append(
			args,
			req.LibraryTypes,
		)
		argIndex++
	}

	for _, rawTag := range req.Tags {
		tag := strings.TrimSpace(rawTag)
		if tag == "" {
			continue
		}

		tagJSON, err := componentTagJSON(tag)
		if err != nil {
			return nil, fmt.Errorf(
				"编码组件标签失败: %w",
				err,
			)
		}

		where += fmt.Sprintf(
			" AND c.tags @> $%d::jsonb",
			argIndex,
		)
		args = append(args, tagJSON)
		argIndex++
	}

	indexConditions, updatedArgs, updatedIndex :=
		buildIndexColumnConditions(
			req,
			args,
			argIndex,
		)

	where += indexConditions
	args = updatedArgs
	argIndex = updatedIndex

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	query := fmt.Sprintf(
		`
			SELECT
				library_type,
				id,
				education_domain,
				display_label,
				design_logic,
				example_snippet,
				full_guide,
				quality_score,
				usage_count,
				select_count,
				tags,
				component_index
			FROM (
				SELECT
					c.library_type,
					c.id,
					c.education_domain,
					c.display_label,
					COALESCE(
						c.design_logic,
						''
					) AS design_logic,
					COALESCE(
						c.example_snippet,
						''
					) AS example_snippet,
					COALESCE(
						c.full_guide,
						''
					) AS full_guide,
					c.quality_score,
					c.usage_count,
					c.select_count,
					COALESCE(
						c.tags::text,
						'[]'
					) AS tags,
					COALESCE(
						c.component_index,
						''
					) AS component_index,
					ROW_NUMBER() OVER (
						PARTITION BY c.library_type
						ORDER BY
							c.quality_score DESC,
							c.select_count DESC
					) AS row_number
				FROM lesson_plan_components c
				%s
			) ranked
			WHERE row_number <= $%d
			ORDER BY
				library_type,
				quality_score DESC,
				select_count DESC
		`,
		where,
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
			"按教育域匹配组件失败: %w",
			err,
		)
	}
	defer rows.Close()

	return scanDomainMatchedComponentGroups(rows)
}

// scanDomainMatchedComponentGroups 扫描教育域组件匹配分组。
func scanDomainMatchedComponentGroups(
	rows interface {
		Next() bool
		Scan(dest ...interface{}) error
		Err() error
	},
) ([]*models.MatchedComponentGroup, error) {
	groupMap := make(
		map[string]*models.MatchedComponentGroup,
	)
	groupOrder := make([]string, 0)

	for rows.Next() {
		var libraryType string
		component := &models.MatchedComponent{}

		if err := rows.Scan(
			&libraryType,
			&component.ID,
			&component.EducationDomain,
			&component.DisplayLabel,
			&component.DesignLogic,
			&component.ExampleSnippet,
			&component.FullGuide,
			&component.QualityScore,
			&component.UsageCount,
			&component.SelectCount,
			&component.Tags,
			&component.ComponentIndex,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教育域组件匹配结果失败: %w",
				err,
			)
		}

		group, exists := groupMap[libraryType]
		if !exists {
			group = &models.MatchedComponentGroup{
				LibraryType: libraryType,
				LibraryName: models.LibraryTypeNameMap[libraryType],
				Components:  []*models.MatchedComponent{},
			}

			groupMap[libraryType] = group
			groupOrder = append(
				groupOrder,
				libraryType,
			)
		}

		group.Components = append(
			group.Components,
			component,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教育域组件匹配结果失败: %w",
			err,
		)
	}

	result := make(
		[]*models.MatchedComponentGroup,
		0,
		len(groupOrder),
	)

	for _, libraryType := range groupOrder {
		result = append(
			result,
			groupMap[libraryType],
		)
	}

	return result, nil
}
