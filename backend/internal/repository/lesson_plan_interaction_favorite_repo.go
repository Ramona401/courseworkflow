package repository

// lesson_plan_interaction_favorite_repo.go — 安全收藏列表查询
//
// 本文件从lesson_plan_interaction_repo.go拆出，避免原文件超过600行。
// COUNT和分页SELECT共用相同的共享候选、作者白名单和教育域条件，
// 历史草稿、提交中、个人发布、软删除或异域收藏均不会进入total或items。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListUserFavoritesWithSharedAccess 查询当前用户仍有权访问的共享收藏。
func ListUserFavoritesWithSharedAccess(
	ctx context.Context,
	userID string,
	limit int,
	offset int,
	visibleAuthorIDs []string,
	educationDomain string,
) ([]*models.FavoriteListItem, int, error) {
	userID = strings.TrimSpace(userID)
	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if userID == "" ||
		len(visibleAuthorIDs) == 0 ||
		!models.IsTeachingEducationDomain(educationDomain) {
		return []*models.FavoriteListItem{}, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const sharedWhere = `
		FROM lesson_plan_interactions i
		JOIN lesson_plans lp
		  ON lp.id = i.lesson_plan_id
		WHERE i.user_id = $1
		  AND i.interaction_type = 'favorite'
		  AND lp.deleted_at IS NULL
		  AND lp.status IN ('approved', 'published_shared')
		  AND lp.visibility IN ('group', 'school', 'region', 'public')
		  AND lp.author_id::text = ANY($2)
		  AND lp.education_domain IN ($3, 'common')
	`

	var total int
	if err := database.DB.QueryRow(
		ctx,
		"SELECT COUNT(*) "+sharedWhere,
		userID,
		visibleAuthorIDs,
		educationDomain,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"查询可见收藏总数失败: %w",
			err,
		)
	}

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
			COALESCE((
				SELECT COUNT(*)
				FROM lesson_plan_interactions x
				WHERE x.lesson_plan_id = lp.id
				  AND x.interaction_type = 'like'
			), 0) AS like_count,
			COALESCE((
				SELECT COUNT(*)
				FROM lesson_plan_interactions x
				WHERE x.lesson_plan_id = lp.id
				  AND x.interaction_type = 'favorite'
			), 0) AS favorite_count,
			i.created_at AS favorited_at
		FROM lesson_plan_interactions i
		JOIN lesson_plans lp
		  ON lp.id = i.lesson_plan_id
		LEFT JOIN users u
		  ON u.id = lp.author_id
		WHERE i.user_id = $1
		  AND i.interaction_type = 'favorite'
		  AND lp.deleted_at IS NULL
		  AND lp.status IN ('approved', 'published_shared')
		  AND lp.visibility IN ('group', 'school', 'region', 'public')
		  AND lp.author_id::text = ANY($2)
		  AND lp.education_domain IN ($3, 'common')
		ORDER BY i.created_at DESC
		LIMIT $4 OFFSET $5
	`

	rows, err := database.DB.Query(
		ctx,
		query,
		userID,
		visibleAuthorIDs,
		educationDomain,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询可见收藏列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.FavoriteListItem,
		0,
	)
	for rows.Next() {
		item := &models.FavoriteListItem{}
		var favoritedAt time.Time
		if err := rows.Scan(
			&item.InteractionID,
			&item.LessonPlanID,
			&item.Title,
			&item.Subject,
			&item.Grade,
			&item.Topic,
			&item.AuthorName,
			&item.AIReviewScore,
			&item.Status,
			&item.LikeCount,
			&item.FavoriteCount,
			&favoritedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描可见收藏行失败: %w",
				err,
			)
		}

		item.FavoritedAt = &favoritedAt
		item.StatusName =
			models.LPStatusNameMap[item.Status]
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历可见收藏列表失败: %w",
			err,
		)
	}

	return items, total, nil
}
