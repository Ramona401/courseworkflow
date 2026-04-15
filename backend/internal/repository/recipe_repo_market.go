package repository

// recipe_repo_market.go — 备课配方市场功能数据访问
//
// 从 recipe_repo.go 拆分而来（v92重构）
//
// 本文件包含：
//   - RecipeStatsRow / RecipeUsageRow — 效果统计类型
//   - GetRecipeStats — 配方效果统计聚合查询
//   - MarketRecipeItem — 市场列表项类型
//   - ListMarketRecipes — 配方市场排行榜查询（综合分排序）
//
// 依赖：
//   - database.DB（pgxpool连接池）
//   - models.RecipeScopeNameMap（scope名称映射）

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 配方效果统计 ====================

// RecipeStatsRow 配方统计数据行
type RecipeStatsRow struct {
	TotalUsage   int              // 总使用次数
	TotalPlans   int              // 产出教案总数
	AvgScore     float64          // 教案平均分
	RecentUsages []RecipeUsageRow // 最近使用记录
}

// RecipeUsageRow 单条使用记录
type RecipeUsageRow struct {
	LessonPlanID  *string  `json:"lesson_plan_id"`
	UserName      string   `json:"user_name"`
	AIReviewScore *float64 `json:"ai_review_score"`
	CreatedAt     string   `json:"created_at"`
}

// GetRecipeStats 获取配方效果统计数据
// 从 recipe_usage_log 表聚合使用次数、教案数、平均分，并返回最近20条使用记录
func GetRecipeStats(ctx context.Context, recipeID string) (*RecipeStatsRow, error) {
	stats := &RecipeStatsRow{}

	// 1. 聚合统计（使用次数、有教案的记录数、平均分）
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(lesson_plan_id),
		       COALESCE(AVG(ai_review_score) FILTER (WHERE ai_review_score IS NOT NULL), 0)
		FROM recipe_usage_log
		WHERE recipe_id = $1
	`, recipeID).Scan(&stats.TotalUsage, &stats.TotalPlans, &stats.AvgScore)
	if err != nil {
		return nil, fmt.Errorf("查询配方统计失败: %w", err)
	}

	// 2. 最近20条使用记录（含用户名和教案分数）
	rows, err := database.DB.Query(ctx, `
		SELECT rul.lesson_plan_id,
		       COALESCE(u.display_name, u.username, '未知'),
		       rul.ai_review_score,
		       TO_CHAR(rul.created_at, 'YYYY-MM-DD HH24:MI')
		FROM recipe_usage_log rul
		LEFT JOIN users u ON u.id = rul.user_id
		WHERE rul.recipe_id = $1
		ORDER BY rul.created_at DESC
		LIMIT 20
	`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("查询配方使用记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row RecipeUsageRow
		if err := rows.Scan(&row.LessonPlanID, &row.UserName, &row.AIReviewScore, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描使用记录行失败: %w", err)
		}
		stats.RecentUsages = append(stats.RecentUsages, row)
	}
	if stats.RecentUsages == nil {
		stats.RecentUsages = []RecipeUsageRow{}
	}

	return stats, nil
}

// ==================== 配方市场排行榜 ====================

// MarketRecipeItem 市场配方列表项（含统计数据）
type MarketRecipeItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Subject        string  `json:"subject"`
	GradeRange     string  `json:"grade_range"`
	ComponentCount int     `json:"component_count"`
	Scope          string  `json:"scope"`
	ScopeName      string  `json:"scope_name"`
	AuthorID       string  `json:"author_id"`
	AuthorName     string  `json:"author_name"`
	ForkCount      int     `json:"fork_count"`
	UseCount       int     `json:"use_count"`
	AvgScore       float64 `json:"avg_score"`
	PlanCount      int     `json:"plan_count"`
	PromptMode     string  `json:"prompt_mode"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// ListMarketRecipes 查询配方市场排行榜
// 条件：scope为group或school的已共享配方，按综合分排序
// 综合分 = avg_score * 0.5 + log2(use_count+1) * 0.3 + log2(fork_count+1) * 0.2
func ListMarketRecipes(ctx context.Context, subject string, gradeRange string, sortBy string, limit int, offset int) ([]*MarketRecipeItem, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// 构建WHERE条件
	where := " WHERE r.status = 'active' AND r.scope IN ('group','school')"
	args := []interface{}{}
	argIdx := 1

	if subject != "" {
		where += fmt.Sprintf(" AND r.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if gradeRange != "" {
		where += fmt.Sprintf(" AND r.grade_range = $%d", argIdx)
		args = append(args, gradeRange)
		argIdx++
	}

	// 查总数
	countQuery := "SELECT COUNT(*) FROM teaching_recipes r" + where
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询市场配方总数失败: %w", err)
	}

	// 排序方式
	orderClause := "ORDER BY composite_score DESC"
	switch sortBy {
	case "use_count":
		orderClause = "ORDER BY r.use_count DESC"
	case "fork_count":
		orderClause = "ORDER BY r.fork_count DESC"
	case "avg_score":
		orderClause = "ORDER BY avg_score DESC"
	case "newest":
		orderClause = "ORDER BY r.created_at DESC"
	}

	// 查列表
	listQuery := fmt.Sprintf(`
		SELECT r.id, r.name, COALESCE(r.description, ''), r.subject, r.grade_range,
		       COALESCE(jsonb_array_length(r.component_ids), 0),
		       r.scope, r.author_id, COALESCE(u.display_name, u.username, ''),
		       r.fork_count, r.use_count,
		       COALESCE(stats.avg_score, 0),
		       COALESCE(stats.plan_count, 0),
		       COALESCE(r.prompt_mode, 'guided'),
		       TO_CHAR(r.created_at, 'YYYY-MM-DD'),
		       TO_CHAR(r.updated_at, 'YYYY-MM-DD'),
		       (COALESCE(stats.avg_score, 0) * 0.5 + LN(r.use_count + 1)/LN(2) * 0.3 + LN(r.fork_count + 1)/LN(2) * 0.2) AS composite_score
		FROM teaching_recipes r
		LEFT JOIN users u ON u.id = r.author_id
		LEFT JOIN LATERAL (
		        SELECT AVG(ai_review_score) FILTER (WHERE ai_review_score IS NOT NULL) AS avg_score,
		               COUNT(lesson_plan_id) AS plan_count
		        FROM recipe_usage_log WHERE recipe_id = r.id
		) stats ON true
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, where, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询市场配方列表失败: %w", err)
	}
	defer rows.Close()

	var items []*MarketRecipeItem
	for rows.Next() {
		item := &MarketRecipeItem{}
		var compositeScore float64
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Subject, &item.GradeRange,
			&item.ComponentCount,
			&item.Scope, &item.AuthorID, &item.AuthorName,
			&item.ForkCount, &item.UseCount,
			&item.AvgScore, &item.PlanCount,
			&item.PromptMode,
			&item.CreatedAt, &item.UpdatedAt,
			&compositeScore,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描市场配方行失败: %w", err)
		}
		item.ScopeName = models.RecipeScopeNameMap[item.Scope]
		items = append(items, item)
	}
	if items == nil {
		items = []*MarketRecipeItem{}
	}
	return items, total, nil
}
