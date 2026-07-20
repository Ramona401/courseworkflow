package repository

// recipe_repo_list_domain.go — 配方普通列表可信范围查询
//
// 本文件替代HTTP普通列表对旧ListRecipes的直接调用。
// 核心原则：前端查询参数只能收窄结果，不能决定用户可见范围。
//
// 基础可见范围由服务端Actor确定：
//   - personal：当前用户本人创建；
//   - group：scope_ref_id属于Actor.MyGroupIDs；
//   - school：scope_ref_id等于Actor.SchoolID；
//   - admin：可跨组织查看合法资源。
//
// 教育域规则：
//   - k12只列k12或common；
//   - vocational只列vocational或common；
//   - adult只列adult或common；
//   - mixed允许跨域查看k12、vocational、adult和common；
//   - 空值、非法域和current=common返回空列表。
//
// 前端scope、scope_ref_id、subject和grade_range均在基础权限之后
// 作为附加AND条件，只能进一步收窄结果。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// RecipeDomainListParams 配方可信列表参数。
type RecipeDomainListParams struct {
	CurrentUserID          string
	CurrentRole            string
	CurrentSchoolID        string
	CurrentEducationDomain string
	CurrentGroupIDs        []string

	RequestedScope      string
	RequestedScopeRefID string
	Subject             string
	GradeRange          string

	Limit  int
	Offset int
}

// ListRecipesForActorDomain 按可信Actor范围查询配方列表。
func ListRecipesForActorDomain(
	ctx context.Context,
	params *RecipeDomainListParams,
) (
	[]*models.RecipeListItem,
	int,
	error,
) {
	if params == nil {
		return []*models.RecipeListItem{}, 0, nil
	}

	params.CurrentUserID = strings.TrimSpace(
		params.CurrentUserID,
	)
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

	params.RequestedScope = strings.TrimSpace(
		params.RequestedScope,
	)
	params.RequestedScopeRefID = strings.TrimSpace(
		params.RequestedScopeRefID,
	)
	params.Subject = strings.TrimSpace(
		params.Subject,
	)
	params.GradeRange = strings.TrimSpace(
		params.GradeRange,
	)

	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	whereParts := []string{
		"r.status = 'active'",
	}
	args := make([]interface{}, 0)

	addArg := func(value interface{}) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	// admin允许跨组织管理合法资源。
	//
	// 其它角色必须在本人、所属教研组和当前学校三类可信范围内。
	if params.CurrentRole != models.RoleAdmin {
		visibleParts := make([]string, 0, 3)

		if params.CurrentUserID != "" {
			placeholder := addArg(
				params.CurrentUserID,
			)
			visibleParts = append(
				visibleParts,
				fmt.Sprintf(
					"(r.scope = 'personal' AND r.author_id::text = %s)",
					placeholder,
				),
			)
		}

		if len(params.CurrentGroupIDs) > 0 {
			placeholder := addArg(
				params.CurrentGroupIDs,
			)
			visibleParts = append(
				visibleParts,
				fmt.Sprintf(
					"(r.scope = 'group' AND r.scope_ref_id::text = ANY(%s))",
					placeholder,
				),
			)
		}

		if params.CurrentSchoolID != "" {
			placeholder := addArg(
				params.CurrentSchoolID,
			)
			visibleParts = append(
				visibleParts,
				fmt.Sprintf(
					"(r.scope = 'school' AND r.scope_ref_id::text = %s)",
					placeholder,
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

	// 教育域是独立硬门槛。
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
		legalDomains := []string{
			models.EducationDomainK12,
			models.EducationDomainVocational,
			models.EducationDomainAdult,
			models.EducationDomainCommon,
		}

		domainArgs := make(
			[]string,
			0,
			len(legalDomains),
		)
		for _, domain := range legalDomains {
			domainArgs = append(
				domainArgs,
				addArg(domain),
			)
		}

		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.education_domain IN (%s)",
				strings.Join(
					domainArgs,
					", ",
				),
			),
		)

	default:
		// 空值、非法域或current=common严格返回空列表。
		whereParts = append(
			whereParts,
			"1 = 0",
		)
	}

	// 前端scope只能附加收窄。
	if params.RequestedScope != "" {
		switch params.RequestedScope {
		case models.RecipeScopePersonal,
			models.RecipeScopeGroup,
			models.RecipeScopeSchool:
			placeholder := addArg(
				params.RequestedScope,
			)
			whereParts = append(
				whereParts,
				fmt.Sprintf(
					"r.scope = %s",
					placeholder,
				),
			)

		default:
			whereParts = append(
				whereParts,
				"1 = 0",
			)
		}
	}

	// 前端scope_ref_id只能在基础权限结果上进一步收窄。
	if params.RequestedScopeRefID != "" {
		placeholder := addArg(
			params.RequestedScopeRefID,
		)
		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.scope_ref_id::text = %s",
				placeholder,
			),
		)
	}

	if params.Subject != "" {
		placeholder := addArg(
			params.Subject,
		)
		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.subject = %s",
				placeholder,
			),
		)
	}

	if params.GradeRange != "" {
		placeholder := addArg(
			params.GradeRange,
		)
		whereParts = append(
			whereParts,
			fmt.Sprintf(
				"r.grade_range = %s",
				placeholder,
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
			"查询可信配方总数失败: %w",
			err,
		)
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
			r.education_domain,
			COALESCE(
				u.display_name,
				u.username,
				''
			),
			r.fork_count,
			r.use_count,
			r.version,
			r.forked_from,
			r.status,
			COALESCE(r.prompt_mode, 'guided'),
			COALESCE(
				r.stages_config::text,
				'[]'
			),
			r.created_at,
			r.updated_at
		FROM teaching_recipes r
		LEFT JOIN users u
		  ON u.id = r.author_id
		%s
		ORDER BY r.updated_at DESC
		LIMIT $%d
		OFFSET $%d
	`,
		whereClause,
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
			"查询可信配方列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.RecipeListItem,
		0,
	)

	for rows.Next() {
		item := &models.RecipeListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Subject,
			&item.GradeRange,
			&item.ComponentCount,
			&item.Scope,
			&item.AuthorID,
			&item.EducationDomain,
			&item.AuthorName,
			&item.ForkCount,
			&item.UseCount,
			&item.Version,
			&item.ForkedFrom,
			&item.Status,
			&item.PromptMode,
			&item.StagesConfig,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描可信配方列表失败: %w",
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
			"遍历可信配方列表失败: %w",
			err,
		)
	}

	return items, total, nil
}
