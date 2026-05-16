package repository

// lesson_plan_interaction_repo.go — 教案互动数据访问层
//
// 职责：
//   - 点赞/收藏的 Toggle（有则删、无则插）
//   - 按教案ID查询互动计数 + 当前用户状态
//   - 批量查询互动计数（用于列表页）
//   - 收藏列表查询
//   - 按教案ID获取互动计数（不含用户状态，用于Toggle后返回新计数）

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== Toggle 互动 ====================

// ToggleInteraction 切换互动状态
// 如果已存在则删除（取消），如果不存在则创建（添加）
// 返回 active=true 表示添加成功，active=false 表示取消成功
func ToggleInteraction(ctx context.Context, userID, planID, interactionType string) (bool, error) {
	// 先尝试删除
	result, err := database.DB.Exec(ctx,
		`DELETE FROM lesson_plan_interactions
		 WHERE user_id = $1 AND lesson_plan_id = $2 AND interaction_type = $3`,
		userID, planID, interactionType,
	)
	if err != nil {
		return false, fmt.Errorf("删除互动记录失败: %w", err)
	}

	// 如果删除了行，说明之前存在 → 现在是"取消"
	if result.RowsAffected() > 0 {
		return false, nil
	}

	// 没有删除到 → 之前不存在 → 插入新记录
	_, err = database.DB.Exec(ctx,
		`INSERT INTO lesson_plan_interactions (user_id, lesson_plan_id, interaction_type)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, lesson_plan_id, interaction_type) DO NOTHING`,
		userID, planID, interactionType,
	)
	if err != nil {
		return false, fmt.Errorf("创建互动记录失败: %w", err)
	}
	return true, nil
}

// ==================== 查询单个教案的互动计数 + 用户状态 ====================

// GetInteractionCounts 查询指定教案的互动统计（含当前用户状态）
// 如果 currentUserID 为空字符串，is_liked / is_favorited 永远返回 false
func GetInteractionCounts(ctx context.Context, planID string, currentUserID string) (*models.InteractionCounts, error) {
	counts := &models.InteractionCounts{}

	// 聚合查询：一次扫描出 like_count 和 favorite_count
	err := database.DB.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN interaction_type = 'like' THEN 1 ELSE 0 END), 0) AS like_count,
			COALESCE(SUM(CASE WHEN interaction_type = 'favorite' THEN 1 ELSE 0 END), 0) AS favorite_count
		FROM lesson_plan_interactions
		WHERE lesson_plan_id = $1
	`, planID).Scan(&counts.LikeCount, &counts.FavoriteCount)
	if err != nil {
		return nil, fmt.Errorf("查询互动计数失败: %w", err)
	}

	// 查询当前用户状态
	if currentUserID != "" {
		rows, err := database.DB.Query(ctx,
			`SELECT interaction_type FROM lesson_plan_interactions
			 WHERE user_id = $1 AND lesson_plan_id = $2`,
			currentUserID, planID,
		)
		if err != nil {
			return nil, fmt.Errorf("查询用户互动状态失败: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				continue
			}
			switch t {
			case models.InteractionTypeLike:
				counts.IsLiked = true
			case models.InteractionTypeFavorite:
				counts.IsFavorited = true
			}
		}
	}

	return counts, nil
}

// ==================== 按教案ID获取单种互动计数（Toggle后返回） ====================

// GetInteractionCount 获取指定教案某种互动的总计数
func GetInteractionCount(ctx context.Context, planID string, interactionType string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM lesson_plan_interactions
		 WHERE lesson_plan_id = $1 AND interaction_type = $2`,
		planID, interactionType,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询互动计数失败: %w", err)
	}
	return count, nil
}

// ==================== 批量查询互动计数（教案列表页使用） ====================

// BatchInteractionCounts 一次查出的互动数据
type BatchInteractionCounts struct {
	LikeCount     int
	FavoriteCount int
	IsLiked       bool
	IsFavorited   bool
}

// GetBatchInteractionCounts 批量查询多个教案的互动计数 + 当前用户状态
// 返回 map[planID]*BatchInteractionCounts
func GetBatchInteractionCounts(ctx context.Context, planIDs []string, currentUserID string) (map[string]*BatchInteractionCounts, error) {
	result := make(map[string]*BatchInteractionCounts)
	if len(planIDs) == 0 {
		return result, nil
	}

	// 初始化全部
	for _, pid := range planIDs {
		result[pid] = &BatchInteractionCounts{}
	}

	// 1. 聚合计数
	// 动态构建 IN 子句
	inClause, args := buildInClause(planIDs, 1)
	countQuery := fmt.Sprintf(`
		SELECT lesson_plan_id, interaction_type, COUNT(*) AS cnt
		FROM lesson_plan_interactions
		WHERE lesson_plan_id IN (%s)
		GROUP BY lesson_plan_id, interaction_type
	`, inClause)

	rows, err := database.DB.Query(ctx, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询互动计数失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var planID, iType string
		var cnt int
		if err := rows.Scan(&planID, &iType, &cnt); err != nil {
			continue
		}
		if item, ok := result[planID]; ok {
			switch iType {
			case models.InteractionTypeLike:
				item.LikeCount = cnt
			case models.InteractionTypeFavorite:
				item.FavoriteCount = cnt
			}
		}
	}

	// 2. 当前用户状态
	if currentUserID != "" {
		argIdx := len(args) + 1
		userArgs := append(args, currentUserID)
		userQuery := fmt.Sprintf(`
			SELECT lesson_plan_id, interaction_type
			FROM lesson_plan_interactions
			WHERE lesson_plan_id IN (%s) AND user_id = $%d
		`, inClause, argIdx)

		uRows, err := database.DB.Query(ctx, userQuery, userArgs...)
		if err != nil {
			// 用户状态查询失败不阻断，仅计数正常返回
			return result, nil
		}
		defer uRows.Close()
		for uRows.Next() {
			var planID, iType string
			if err := uRows.Scan(&planID, &iType); err != nil {
				continue
			}
			if item, ok := result[planID]; ok {
				switch iType {
				case models.InteractionTypeLike:
					item.IsLiked = true
				case models.InteractionTypeFavorite:
					item.IsFavorited = true
				}
			}
		}
	}

	return result, nil
}

// buildInClause 构建 PostgreSQL IN($1, $2, ...) 子句
// startIdx: 参数起始编号
func buildInClause(ids []string, startIdx int) (string, []interface{}) {
	placeholders := ""
	args := make([]interface{}, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", startIdx+i)
		args = append(args, id)
	}
	return placeholders, args
}

// ==================== 收藏列表 ====================

// ListUserFavorites 查询用户的收藏列表（含教案摘要+互动计数）
func ListUserFavorites(ctx context.Context, userID string, limit, offset int) ([]*models.FavoriteListItem, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// 总数
	var total int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM lesson_plan_interactions
		 WHERE user_id = $1 AND interaction_type = 'favorite'`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询收藏总数失败: %w", err)
	}

	// 列表
	query := `
		SELECT
			i.id AS interaction_id,
			lp.id AS lesson_plan_id,
			lp.title,
			lp.subject,
			lp.grade,
			lp.topic,
			COALESCE(u.display_name, '') AS author_name,
			lp.ai_review_score,
			lp.status,
			COALESCE(
				(SELECT COUNT(*) FROM lesson_plan_interactions x WHERE x.lesson_plan_id = lp.id AND x.interaction_type = 'like'),
			0) AS like_count,
			COALESCE(
				(SELECT COUNT(*) FROM lesson_plan_interactions x WHERE x.lesson_plan_id = lp.id AND x.interaction_type = 'favorite'),
			0) AS favorite_count,
			i.created_at AS favorited_at
		FROM lesson_plan_interactions i
		JOIN lesson_plans lp ON lp.id = i.lesson_plan_id
		LEFT JOIN users u ON u.id = lp.author_id
		WHERE i.user_id = $1 AND i.interaction_type = 'favorite'
		ORDER BY i.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := database.DB.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询收藏列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.FavoriteListItem
	for rows.Next() {
		item := &models.FavoriteListItem{}
		var favAt time.Time
		if err := rows.Scan(
			&item.InteractionID, &item.LessonPlanID,
			&item.Title, &item.Subject, &item.Grade, &item.Topic,
			&item.AuthorName, &item.AIReviewScore, &item.Status,
			&item.LikeCount, &item.FavoriteCount,
			&favAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描收藏行失败: %w", err)
		}
		item.FavoritedAt = &favAt
		item.StatusName = models.LPStatusNameMap[item.Status]
		items = append(items, item)
	}
	if items == nil {
		items = []*models.FavoriteListItem{}
	}
	return items, total, nil
}
