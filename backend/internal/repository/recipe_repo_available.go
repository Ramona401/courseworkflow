package repository

// recipe_repo_available.go — 对话模式可用配方查询
//
// 为对话模式首屏提供"当前老师在指定学科下可见的全部 active 配方"列表。
// 可见性三路并集：
//   1. 个人创建（author_id = userID）
//   2. 教研组共享（scope='group' AND scope_ref_id IN 用户所属教研组）
//   3. 学校共享（scope='school' AND scope_ref_id = 用户所属学校）
//
// 排序：学校配方排最前（教研共识优先级最高）→ 教研组配方 → 个人配方；
//       同级按使用次数降序 → 更新时间降序。
//
// 供 handlers/recipe_available_handler.go 消费，与现有 ListRecipes 互不干扰。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListAvailableRecipes 查询当前用户在指定学科下可见的全部 active 配方
//
// 参数：
//   - userID：当前登录用户（查个人配方）
//   - groupIDs：用户所属的全部教研组ID（查教研组共享配方）
//   - schoolID：用户所属学校ID（查学校共享配方）
//   - subject：学科过滤（空串=不过滤）
//
// 返回按 scope 优先级（school > group > personal）+ use_count 降序排列，上限50条。
func ListAvailableRecipes(ctx context.Context, userID string, groupIDs []string, schoolID string, subject string) ([]*models.RecipeListItem, error) {
	// 构建可见性 OR 子句：个人 ∪ 教研组共享 ∪ 学校共享
	where := " WHERE r.status = 'active' AND ("
	args := []interface{}{}
	argIdx := 1

	// 条件1：我创建的（个人配方）
	where += fmt.Sprintf("r.author_id = $%d", argIdx)
	args = append(args, userID)
	argIdx++

	// 条件2：我所在教研组共享的配方
	if len(groupIDs) > 0 {
		where += fmt.Sprintf(" OR (r.scope = 'group' AND r.scope_ref_id = ANY($%d))", argIdx)
		args = append(args, groupIDs)
		argIdx++
	}

	// 条件3：我所在学校共享的配方
	if schoolID != "" {
		where += fmt.Sprintf(" OR (r.scope = 'school' AND r.scope_ref_id = $%d)", argIdx)
		args = append(args, schoolID)
		argIdx++
	}

	where += ")"

	// 学科过滤（非空时精确匹配）
	if subject != "" {
		where += fmt.Sprintf(" AND r.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}

	// 查询：与 ListRecipes 相同的列集合，排序按 scope 优先级 + 使用量 + 更新时间
	query := fmt.Sprintf(`
		SELECT r.id, r.name, COALESCE(r.description, ''), r.subject, r.grade_range,
		       COALESCE(jsonb_array_length(r.component_ids), 0),
		       r.scope, r.author_id, COALESCE(u.display_name, u.username, ''),
		       r.fork_count, r.use_count, r.version, r.forked_from, r.status,
		       COALESCE(r.prompt_mode, 'guided'),
		       COALESCE(r.stages_config::text, '[]'),
		       r.created_at, r.updated_at
		FROM teaching_recipes r
		LEFT JOIN users u ON u.id = r.author_id
		%s
		ORDER BY
		    CASE r.scope WHEN 'school' THEN 0 WHEN 'group' THEN 1 ELSE 2 END,
		    r.use_count DESC,
		    r.updated_at DESC
		LIMIT 50
	`, where)

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询可用配方失败: %w", err)
	}
	defer rows.Close()

	var items []*models.RecipeListItem
	for rows.Next() {
		item := &models.RecipeListItem{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Subject, &item.GradeRange,
			&item.ComponentCount,
			&item.Scope, &item.AuthorID, &item.AuthorName,
			&item.ForkCount, &item.UseCount, &item.Version, &item.ForkedFrom, &item.Status,
			&item.PromptMode,
			&item.StagesConfig,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描可用配方行失败: %w", err)
		}
		item.ScopeName = models.RecipeScopeNameMap[item.Scope]
		items = append(items, item)
	}
	if items == nil {
		items = []*models.RecipeListItem{}
	}
	return items, nil
}
