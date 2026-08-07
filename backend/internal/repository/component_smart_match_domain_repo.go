package repository

// component_smart_match_domain_repo.go — 画像加权教育域组件匹配。
//
// 域过滤与普通匹配完全一致：只允许具体教学域，并只读取同域或common。
// 画像标签只影响排序分数，不得扩大教育域候选集。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// SmartMatchComponentsForEducationDomain 按教育域和教师画像加权匹配。
func SmartMatchComponentsForEducationDomain(
	ctx context.Context,
	req *models.MatchComponentsRequest,
	currentDomain string,
	profileTags []string,
) ([]*models.MatchedComponentGroup, error) {
	if len(profileTags) == 0 {
		return MatchComponentsForEducationDomain(
			ctx,
			req,
			currentDomain,
		)
	}

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
				"编码组件筛选标签失败: %w",
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

	bonusExpression := "0"

	for _, rawTag := range profileTags {
		tag := strings.TrimSpace(rawTag)
		weight := ""

		switch {
		case strings.HasPrefix(
			tag,
			"style:",
		):
			weight = "2.0"

		case strings.HasPrefix(
			tag,
			"collab:",
		):
			weight = "1.5"

		case strings.HasPrefix(
			tag,
			"priority:",
		):
			weight = "0.5"
		}

		if tag == "" || weight == "" {
			continue
		}

		tagJSON, err := componentTagJSON(tag)
		if err != nil {
			return nil, fmt.Errorf(
				"编码组件画像标签失败: %w",
				err,
			)
		}

		bonusExpression += fmt.Sprintf(
			" + CASE WHEN c.tags @> $%d::jsonb THEN %s ELSE 0 END",
			argIndex,
			weight,
		)

		args = append(args, tagJSON)
		argIndex++
	}

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
				final_score,
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
					LEAST(
						c.quality_score + (%s),
						10.0
					) AS final_score,
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
							LEAST(
								c.quality_score + (%s),
								10.0
							) DESC,
							c.select_count DESC
					) AS row_number
				FROM lesson_plan_components c
				%s
			) ranked
			WHERE row_number <= $%d
			ORDER BY
				library_type,
				final_score DESC,
				select_count DESC
		`,
		bonusExpression,
		bonusExpression,
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
			"画像加权教育域组件匹配失败: %w",
			err,
		)
	}
	defer rows.Close()

	return scanDomainMatchedComponentGroups(rows)
}
