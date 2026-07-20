package repository

// recipe_repo_available.go — 对话模式与专家模式首屏手动可选配方查询
//
// 可见性三路并集：
//   1. 个人创建；
//   2. 所属教研组共享；
//   3. 所属学校共享。
//
// 本文件服务于老师“指定配方”的手动选择，不服务于自动挂载。
//
// 手动候选不再按学科和具体年级过滤，subject和grade只参与相关性排序：
//   1. 同学科、同具体年级；
//   2. 同学科、其它年级或学段；
//   3. 其它学科。
//
// 每个相关性分组内保持原有来源和热度顺序：
// 学校配方 → 教研组配方 → 个人配方；
// 同级按使用次数降序、更新时间降序。
//
// 平台自动匹配仍由services/recipe_resolver.go执行严格学科和具体年级规则。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// ListAvailableRecipesForDomain 查询当前用户有权手动使用的全部active配方。
//
// 组织可见性包括个人、所属教研组和所属学校；
// 教育域可见性只包括当前具体教学域和common。
//
// subject和grade不参与阻断，只用于候选相关性排序。
func ListAvailableRecipesForDomain(
	ctx context.Context,
	userID string,
	groupIDs []string,
	schoolID string,
	currentEducationDomain string,
	subject string,
	grade string,
) ([]*models.RecipeListItem, error) {
	where, args := buildVisibleActiveRecipeWhere(
		userID,
		groupIDs,
		schoolID,
		currentEducationDomain,
	)

	query := fmt.Sprintf(`
                SELECT r.id,
                       r.name,
                       COALESCE(r.description, ''),
                       r.subject,
                       r.grade_range,
                       COALESCE(jsonb_array_length(r.component_ids), 0),
                       r.scope,
                       r.author_id,
                       r.education_domain,
                       COALESCE(u.display_name, u.username, ''),
                       r.fork_count,
                       r.use_count,
                       r.version,
                       r.forked_from,
                       r.status,
                       COALESCE(r.prompt_mode, 'guided'),
                       COALESCE(r.stages_config::text, '[]'),
                       r.created_at,
                       r.updated_at
                FROM teaching_recipes r
                LEFT JOIN users u ON u.id = r.author_id
                %s
                ORDER BY
                    CASE r.scope
                        WHEN 'school' THEN 0
                        WHEN 'group' THEN 1
                        ELSE 2
                    END,
                    r.use_count DESC,
                    r.updated_at DESC
        `, where)

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询手动可选配方失败: %w",
			err,
		)
	}
	defer rows.Close()

	exactItems := make(
		[]*models.RecipeListItem,
		0,
	)
	sameSubjectItems := make(
		[]*models.RecipeListItem,
		0,
	)
	otherItems := make(
		[]*models.RecipeListItem,
		0,
	)

	normalizedSubject :=
		strings.TrimSpace(subject)

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
			return nil, fmt.Errorf(
				"扫描手动可选配方行失败: %w",
				err,
			)
		}

		item.ScopeName =
			models.RecipeScopeNameMap[item.Scope]

		sameSubject :=
			strings.TrimSpace(item.Subject) ==
				normalizedSubject

		switch {
		case sameSubject &&
			utils.IsStrictGradeMatch(
				item.GradeRange,
				grade,
			):
			exactItems = append(
				exactItems,
				item,
			)

		case sameSubject:
			sameSubjectItems = append(
				sameSubjectItems,
				item,
			)

		default:
			otherItems = append(
				otherItems,
				item,
			)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历手动可选配方失败: %w",
			err,
		)
	}

	items := make(
		[]*models.RecipeListItem,
		0,
		len(exactItems)+
			len(sameSubjectItems)+
			len(otherItems),
	)

	items = append(
		items,
		exactItems...,
	)
	items = append(
		items,
		sameSubjectItems...,
	)
	items = append(
		items,
		otherItems...,
	)

	return items, nil
}

// GetVisibleActiveRecipeByIDForDomain 按ID读取一份当前用户有权手动使用的active配方。
//
// 本方法是手动选择的服务端最终权限防线：
//   - 验证个人、教研组或学校组织可见范围；
//   - 验证active状态；
//   - 验证配方教育域为当前域或common；
//   - 不限制学科和年级，保留老师手动选择权。
func GetVisibleActiveRecipeByIDForDomain(
	ctx context.Context,
	userID string,
	groupIDs []string,
	schoolID string,
	currentEducationDomain string,
	recipeID string,
) (
	*models.TeachingRecipe,
	bool,
	error,
) {
	recipeID = strings.TrimSpace(recipeID)

	if recipeID == "" ||
		strings.TrimSpace(userID) == "" {
		return nil, false, nil
	}

	visibleWhere, visibleArgs :=
		buildVisibleActiveRecipeWhereFromIndex(
			userID,
			groupIDs,
			schoolID,
			currentEducationDomain,
			2,
		)

	query := fmt.Sprintf(`
                SELECT EXISTS(
                        SELECT 1
                        FROM teaching_recipes r
                        %s
                          AND r.id = $1
                )
        `, visibleWhere)

	args := make(
		[]interface{},
		0,
		len(visibleArgs)+1,
	)

	args = append(
		args,
		recipeID,
	)
	args = append(
		args,
		visibleArgs...,
	)

	var allowed bool

	if err := database.DB.QueryRow(
		ctx,
		query,
		args...,
	).Scan(&allowed); err != nil {
		return nil, false, fmt.Errorf(
			"校验手动配方使用权限失败: %w",
			err,
		)
	}

	if !allowed {
		return nil, false, nil
	}

	recipe, err := GetRecipeByID(
		ctx,
		recipeID,
	)
	if err != nil {
		return nil, false, err
	}

	// SQL过滤后再次调用统一函数复核，防止未来SQL调整形成旁路。
	if recipe == nil ||
		recipe.Status != "active" ||
		!models.ResourceEducationDomainMatches(
			recipe.EducationDomain,
			currentEducationDomain,
		) {
		return nil, false, nil
	}

	return recipe, true, nil
}

// buildVisibleActiveRecipeWhere 构造从$1开始的active配方可见性WHERE。
func buildVisibleActiveRecipeWhere(
	userID string,
	groupIDs []string,
	schoolID string,
	currentEducationDomain string,
) (
	string,
	[]interface{},
) {
	return buildVisibleActiveRecipeWhereFromIndex(
		userID,
		groupIDs,
		schoolID,
		currentEducationDomain,
		1,
	)
}

// buildVisibleActiveRecipeWhereFromIndex 构造指定参数起始序号的可见性WHERE。
func buildVisibleActiveRecipeWhereFromIndex(
	userID string,
	groupIDs []string,
	schoolID string,
	currentEducationDomain string,
	startIndex int,
) (
	string,
	[]interface{},
) {
	where := " WHERE r.status = 'active' AND ("
	args := []interface{}{}
	argIdx := startIndex

	where += fmt.Sprintf(
		"r.author_id = $%d",
		argIdx,
	)
	args = append(
		args,
		userID,
	)
	argIdx++

	if len(groupIDs) > 0 {
		where += fmt.Sprintf(
			" OR (r.scope = 'group' AND r.scope_ref_id::text = ANY($%d))",
			argIdx,
		)
		args = append(
			args,
			groupIDs,
		)
		argIdx++
	}

	if strings.TrimSpace(schoolID) != "" {
		where += fmt.Sprintf(
			" OR (r.scope = 'school' AND r.scope_ref_id::text = $%d)",
			argIdx,
		)
		args = append(
			args,
			schoolID,
		)
		argIdx++
	}

	where += ")"

	normalizedDomain :=
		strings.ToLower(
			strings.TrimSpace(
				currentEducationDomain,
			),
		)

	// mixed只允许管理查看，不能作为具体教学候选域。
	// common只允许作为资源域，不能作为当前教学域。
	if !models.IsTeachingEducationDomain(
		normalizedDomain,
	) {
		where += " AND 1 = 0"
		return where, args
	}

	where += fmt.Sprintf(
		" AND (r.education_domain = $%d OR r.education_domain = $%d)",
		argIdx,
		argIdx+1,
	)

	args = append(
		args,
		normalizedDomain,
		models.EducationDomainCommon,
	)

	return where, args
}
