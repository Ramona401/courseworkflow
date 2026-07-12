package repository

// review_v2_repo_school.go — 教案多级审核·按学校口径的数据访问（方案B，2026-07-03）
//
// 从 review_v2_repo.go 独立成文件的原因：主文件已接近 600 行红线，本次"学校管理员
// 本校 L1 可见+可审"改造新增的按学校查询单独放置，主文件仅改 GetReviewStats。
//
// ListPendingReviewsL1BySchool：按学校取全部 L1 待审教案，供 senior_operator（学校管理员）
// 的待审列表使用——此前 senior 的 L1 待审走"组内口径"（仅其兼任 lead/backbone 的教研组），
// 不兼任者完全看不到本校 L1 待审教案，形成管理盲区。
//
// 过滤口径选择说明：教案侧不用 review_school_id 而用 tg.school_id（教案绑定教研组所属学校）——
//   1. 教案 L1 待审必然有 group_id（提交评审时强制选组），组→学校关系结构性成立；
//   2. 不依赖 review_school_id 的历史回填情况，对任何存量待审数据都成立；
//   3. 与既有 L1 查询（ListPendingReviewsL1 / L1All）同一张 JOIN 结构，仅多一个学校过滤条件。
// （课件侧无 group_id 列，故课件用 review_school_id 白名单，两侧实现不同但语义一致。）

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListPendingReviewsL1BySchool 按学校获取全部 L1 待审核教案列表（学校管理员本校视图）
// 查询条件：status='submitted' AND review_level=0 AND 教研组所属学校 = schoolID
// （JOIN teaching_groups 为内连接，group_id 为 NULL 的行天然被排除）
// schoolID 为空时直接返回空列表（fail-closed，绝不退化为全局查询）。
func ListPendingReviewsL1BySchool(ctx context.Context, schoolID string, limit int, offset int) ([]*models.PendingReviewItem, int, error) {
	if schoolID == "" {
		return []*models.PendingReviewItem{}, 0, nil
	}

	// 统计总数（JOIN teaching_groups 过滤学校）
	var total int
	if err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM lesson_plans lp
		JOIN teaching_groups tg ON tg.id = lp.group_id
		WHERE lp.status = 'submitted' AND lp.review_level = 0 AND tg.school_id = $1
	`, schoolID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("按学校统计L1待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}
	rows, err := database.DB.Query(ctx, `
		SELECT lp.id, lp.title, lp.subject, lp.grade, lp.author_id,
		       COALESCE(u.display_name, '') AS author_name,
		       lp.group_id,
		       COALESCE(tg.name, '') AS group_name,
		       COALESCE(o.name, '') AS school_name,
		       lp.review_level, lp.ai_review_score, lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u ON u.id = lp.author_id
		JOIN teaching_groups tg ON tg.id = lp.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE lp.status = 'submitted' AND lp.review_level = 0 AND tg.school_id = $1
		ORDER BY lp.updated_at ASC
		LIMIT $2 OFFSET $3
	`, schoolID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("按学校查询L1待审核列表失败: %w", err)
	}
	defer rows.Close()

	// 初始化为空切片（非 nil），前端拿到 items 恒可安全遍历
	items := []*models.PendingReviewItem{}
	for rows.Next() {
		item := &models.PendingReviewItem{}
		err := rows.Scan(
			&item.LessonPlanID, &item.Title, &item.Subject, &item.Grade,
			&item.AuthorID, &item.AuthorName, &item.GroupID, &item.GroupName,
			&item.SchoolName, &item.ReviewLevel, &item.AIReviewScore, &item.SubmittedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描按学校L1待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}
	return items, total, nil
}
