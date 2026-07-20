package repository

// recipe_repo_market_domain.go — 配方市场可信范围与教育域查询
//
// 本文件不修改旧recipe_repo_market.go，而是为登录用户提供
// 与详情权限一致的市场查询：
//
//   - 市场只包含active的group或school共享配方；
//   - 普通用户只能看自己所属教研组和当前学校的共享资源；
//   - admin可以跨组织管理合法市场资源；
//   - 具体教学域只允许同域或common；
//   - mixed管理上下文允许k12、vocational、adult和common；
//   - 空值、非法域和current=common返回空结果。
//
// subject与grade_range只负责收窄结果，sort_by只决定排序。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// RecipeMarketDomainParams 市场可信查询参数。
type RecipeMarketDomainParams struct {
	CurrentRole            string
	CurrentSchoolID        string
	CurrentEducationDomain string
	CurrentGroupIDs        []string

	Subject    string
	GradeRange string
	SortBy     string

	Limit  int
	Offset int
}

// ListMarketRecipesForActorDomain 查询当前Actor可见的市场配方。
func ListMarketRecipesForActorDomain(
	ctx context.Context,
	params *RecipeMarketDomainParams,
) (
	[]*MarketRecipeItem,
	int,
	error,
) {
	if params == nil {
		return []*MarketRecipeItem{}, 0, nil
	}

	params.CurrentRole = strings.TrimSpace(
		params.CurrentRole,
	)
	params.CurrentSchoolID = strings.TrimSpace(
		params.CurrentSchoolID,
	)
	params.CurrentEducationDomain = strings.ToLower(
		strings.TrimSpace(
			params.CurrentEducationDomain,
		),
	)
	params.Subject = strings.TrimSpace(
		params.Subject,
	)
	params.GradeRange = strings.TrimSpace(
		params.GradeRange,
	)
	params.SortBy = strings.TrimSpace(
		params.SortBy,
	)

	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	whereParts := []string{
		"r.status = 'active'",
		"r.scope IN ('group', 'school')",
	}
	args := make([]interface{}, 0)

	addArg := func(value interface{}) string {
		args = append(args, value)
		return fmt.Sprintf(
			"$%d",
			len(args),
		)
	}

	// 非admin仍须遵守当前教研组和学校可见范围。
	if params.CurrentRole != models.RoleAdmin {
		visibleParts := make(
			[]string,
			0,
			2,
		)

		if len(params.CurrentGroupIDs) > 0 {
			groupArg := addArg(
				params.CurrentGroupIDs,
			)
			visibleParts = append(
				visibleParts,
				fmt.Sprintf(
					"(r.scope = 'group' AND r.scope_ref_id::text = ANY(%s))",
					groupArg,
				),
			)
		}

		if params.CurrentSchoolID != "" {
			schoolArg := addArg(
				params.CurrentSchoolID,
			)
			visibleParts = append(
				visibleParts,
				fmt.Sprintf(
					"(r.scope = 'school' AND r.scope_ref_id::text = %s)",
					schoolArg,
				),
			)
		}

		if len(visibleParts) == 0 {
			whereParts = append(
				whereParts,
				"1 = 0",
			)
		} else {
			whereParts = append(
				whereParts,
				"("+
					strings.Join(
						visibleParts,
						" OR ",
					)+
					")",
			)
		}
	}

	// 教育域独立硬过滤。
	switch {
	case models.IsTeachingEducationDomain(
		params.CurrentEducationDomain,
	):
		currentDomainArg := addArg(
			params.CurrentEducationDomain,
		)
		commonDomainArg := addArg(
			models.EducationDomainCommon,
		)

		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"(r.education_domain = %s OR r.education_domain = %s)",
				currentDomainArg,
				commonDomainArg,
			),
		)

	case params.CurrentEducationDomain ==
		models.EducationDomainMixed:
		legalDomainsArg := addArg(
			[]string{
				models.EducationDomainK12,
				models.EducationDomainVocational,
				models.EducationDomainAdult,
				models.EducationDomainCommon,
			},
		)

		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.education_domain = ANY(%s)",
				legalDomainsArg,
			),
		)

	default:
		whereParts = append(
			whereParts,
			"1 = 0",
		)
	}

	if params.Subject != "" {
		subjectArg := addArg(
			params.Subject,
		)
		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.subject = %s",
				subjectArg,
			),
		)
	}

	if params.GradeRange != "" {
		gradeArg := addArg(
			params.GradeRange,
		)
		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.grade_range = %s",
				gradeArg,
			),
		)
	}

	whereClause := " WHERE " +
		strings.Join(
			whereParts,
			" AND ",
		)

	countQuery :=
		"SELECT COUNT(*) FROM teaching_recipes r" +
			whereClause

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"查询可信市场配方总数失败: %w",
			err,
		)
	}

	orderClause :=
		"ORDER BY composite_score DESC"

	switch params.SortBy {
	case "use_count":
		orderClause =
			"ORDER BY r.use_count DESC"

	case "fork_count":
		orderClause =
			"ORDER BY r.fork_count DESC"

	case "avg_score":
		orderClause =
			"ORDER BY avg_score DESC"

	case "newest":
		orderClause =
			"ORDER BY r.created_at DESC"
	}

	limitArgIndex := len(args) + 1
	offsetArgIndex := len(args) + 2

	listQuery := fmt.Sprintf(`
		SELECT
			r.id,
			r.name,
			COALESCE(r.description, ''),
			r.subject,
			r.grade_range,
			COALESCE(
				jsonb_array_length(r.component_ids),
				0
			),
			r.scope,
			r.author_id,
			COALESCE(
				u.display_name,
				u.username,
				''
			),
			r.fork_count,
			r.use_count,
			COALESCE(stats.avg_score, 0),
			COALESCE(stats.plan_count, 0),
			COALESCE(r.prompt_mode, 'guided'),
			TO_CHAR(r.created_at, 'YYYY-MM-DD'),
			TO_CHAR(r.updated_at, 'YYYY-MM-DD'),
			(
				COALESCE(stats.avg_score, 0) * 0.5
				+ LN(r.use_count + 1) / LN(2) * 0.3
				+ LN(r.fork_count + 1) / LN(2) * 0.2
			) AS composite_score
		FROM teaching_recipes r
		LEFT JOIN users u
		  ON u.id = r.author_id
		LEFT JOIN LATERAL (
			SELECT
				AVG(ai_review_score)
					FILTER (
						WHERE ai_review_score IS NOT NULL
					) AS avg_score,
				COUNT(lesson_plan_id) AS plan_count
			FROM recipe_usage_log
			WHERE recipe_id = r.id
		) stats ON true
		%s
		%s
		LIMIT $%d
		OFFSET $%d
	`,
		whereClause,
		orderClause,
		limitArgIndex,
		offsetArgIndex,
	)

	listArgs := make(
		[]interface{},
		0,
		len(args)+2,
	)
	listArgs = append(
		listArgs,
		args...,
	)
	listArgs = append(
		listArgs,
		params.Limit,
		params.Offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询可信市场配方失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*MarketRecipeItem,
		0,
	)

	for rows.Next() {
		item := &MarketRecipeItem{}
		var compositeScore float64

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Subject,
			&item.GradeRange,
			&item.ComponentCount,
			&item.Scope,
			&item.AuthorID,
			&item.AuthorName,
			&item.ForkCount,
			&item.UseCount,
			&item.AvgScore,
			&item.PlanCount,
			&item.PromptMode,
			&item.CreatedAt,
			&item.UpdatedAt,
			&compositeScore,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描可信市场配方失败: %w",
				err,
			)
		}

		item.ScopeName =
			models.RecipeScopeNameMap[item.Scope]

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历可信市场配方失败: %w",
			err,
		)
	}

	return items, total, nil
}
