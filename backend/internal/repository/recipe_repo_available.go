package repository

// recipe_repo_available.go — 对话模式与专家模式首屏可用配方查询
//
// 可见性三路并集：
//   1. 个人创建；
//   2. 所属教研组共享；
//   3. 所属学校共享。
//
// 严格匹配：
//   - 学科必须精确一致；
//   - 年级必须是同一个具体年级；
//   - 空年级、学段和跨年级范围不参与首屏备课匹配。
//
// 排序：学校配方 → 教研组配方 → 个人配方；
// 同级按使用次数降序、更新时间降序。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// ListAvailableRecipes 查询当前用户在指定学科、具体年级下可见的 active 配方。
func ListAvailableRecipes(
	ctx context.Context,
	userID string,
	groupIDs []string,
	schoolID string,
	subject string,
	grade string,
) ([]*models.RecipeListItem, error) {
	where := " WHERE r.status = 'active' AND ("
	args := []interface{}{}
	argIdx := 1

	where += fmt.Sprintf("r.author_id = $%d", argIdx)
	args = append(args, userID)
	argIdx++

	if len(groupIDs) > 0 {
		where += fmt.Sprintf(
			" OR (r.scope = 'group' AND r.scope_ref_id = ANY($%d))",
			argIdx,
		)
		args = append(args, groupIDs)
		argIdx++
	}

	if schoolID != "" {
		where += fmt.Sprintf(
			" OR (r.scope = 'school' AND r.scope_ref_id = $%d)",
			argIdx,
		)
		args = append(args, schoolID)
		argIdx++
	}

	where += ")"

	if subject != "" {
		where += fmt.Sprintf(" AND r.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}

	// 年级使用Go层统一规范化后严格过滤。
	// 不在SQL里直接比较，是为了兼容“高三/十二年级/12年级/12”等同义表达。
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
			&item.Version,
			&item.ForkedFrom,
			&item.Status,
			&item.PromptMode,
			&item.StagesConfig,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描可用配方行失败: %w", err)
		}

		if !utils.IsStrictGradeMatch(
			item.GradeRange,
			grade,
		) {
			continue
		}

		item.ScopeName = models.RecipeScopeNameMap[item.Scope]
		items = append(items, item)

		if len(items) >= 50 {
			break
		}
	}

	if items == nil {
		items = []*models.RecipeListItem{}
	}

	return items, nil
}
