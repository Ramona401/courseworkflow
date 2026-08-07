package repository

// recipe_component_domain_repo.go — 配方绑定组件的教育域读取仓储。
//
// 本文件仅服务teaching_recipes.component_ids：
//   - 配方资源域为k12、vocational或adult时，允许同域组件和common组件；
//   - 配方资源域为common时，只允许common组件；
//   - mixed、空值和非法资源域返回空结果；
//   - 摘要查询保留同域历史组件状态，供配方编辑页识别失效组件；
//   - AI上下文查询只返回active且approved组件；
//   - 异域组件的ID、标题和正文均不返回。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// GetRecipeComponentBriefsForResourceDomain 按配方资源域读取组件摘要。
func GetRecipeComponentBriefsForResourceDomain(
	ctx context.Context,
	componentIDs []string,
	resourceDomain string,
) ([]*models.RecipeComponentBrief, error) {
	if len(componentIDs) == 0 {
		return []*models.RecipeComponentBrief{}, nil
	}

	rows, err := database.DB.Query(
		ctx,
		`
			SELECT
				c.id,
				c.library_type,
				c.display_label,
				c.quality_score,
				c.status
			FROM lesson_plan_components c
			WHERE c.id = ANY($1::uuid[])
			  AND $2 IN (
					'k12',
					'vocational',
					'adult',
					'common'
			  )
			  AND (
					(
						$2 = 'common'
						AND c.education_domain = 'common'
					)
					OR
					(
						$2 IN (
							'k12',
							'vocational',
							'adult'
						)
						AND (
							c.education_domain = $2
							OR c.education_domain = 'common'
						)
					)
			  )
			ORDER BY
				c.library_type,
				c.quality_score DESC,
				c.updated_at DESC
		`,
		componentIDs,
		resourceDomain,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按配方教育域查询组件摘要失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.RecipeComponentBrief,
		0,
		len(componentIDs),
	)

	for rows.Next() {
		item := &models.RecipeComponentBrief{}

		if err := rows.Scan(
			&item.ID,
			&item.LibraryType,
			&item.DisplayLabel,
			&item.QualityScore,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描配方教育域组件摘要失败: %w",
				err,
			)
		}

		item.LibraryName =
			models.LibraryTypeNameMap[item.LibraryType]

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历配方教育域组件摘要失败: %w",
			err,
		)
	}

	return items, nil
}

// GetRecipeComponentContentsForResourceDomain 按配方资源域读取AI上下文组件。
func GetRecipeComponentContentsForResourceDomain(
	ctx context.Context,
	componentIDs []string,
	resourceDomain string,
) ([]*models.MatchedComponentGroup, error) {
	if len(componentIDs) == 0 {
		return []*models.MatchedComponentGroup{}, nil
	}

	rows, err := database.DB.Query(
		ctx,
		`
			SELECT
				c.library_type,
				c.id,
				c.education_domain,
				c.display_label,
				COALESCE(c.design_logic, ''),
				COALESCE(c.example_snippet, ''),
				COALESCE(c.full_guide, ''),
				c.quality_score,
				c.usage_count,
				c.select_count,
				COALESCE(c.tags::text, '[]'),
				COALESCE(c.component_index, '')
			FROM lesson_plan_components c
			WHERE c.id = ANY($1::uuid[])
			  AND c.status = 'active'
			  AND c.review_status = 'approved'
			  AND $2 IN (
					'k12',
					'vocational',
					'adult',
					'common'
			  )
			  AND (
					(
						$2 = 'common'
						AND c.education_domain = 'common'
					)
					OR
					(
						$2 IN (
							'k12',
							'vocational',
							'adult'
						)
						AND (
							c.education_domain = $2
							OR c.education_domain = 'common'
						)
					)
			  )
			ORDER BY
				c.library_type,
				c.quality_score DESC,
				c.select_count DESC
		`,
		componentIDs,
		resourceDomain,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按配方教育域查询组件内容失败: %w",
			err,
		)
	}
	defer rows.Close()

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
				"扫描配方教育域组件内容失败: %w",
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
			"遍历配方教育域组件内容失败: %w",
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
