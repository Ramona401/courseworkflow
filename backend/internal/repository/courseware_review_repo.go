package repository

// courseware_review_repo.go — 课件多级审核数据访问层（阶段3）
//
// 镜像 review_v2_repo.go，但有三处关键改写：
//   1. 审核记录表 lesson_plan_reviews_v2 → courseware_reviews。
//   2. 待审核条件从教案 status='submitted' 改为课件 publish_state='submitted'
//      （课件审核态走与 status 正交的 publish_state 维度）。
//   3. 课件无 group_id 列——课件归属教研组靠"作者所属教研组"间接判定。
//      因此 L1 待审核列表的"组内课件"过滤，不能像教案那样直接 lp.group_id IN(...)，
//      而是改为"课件作者 user_id ∈ 这些教研组的成员"。本文件用 teaching_group_members
//      关联 coursewares.user_id 实现。
//
// 审核流程配置（review_flow_configs）的读写【复用】review_v2_repo.go 里已有的
// GetReviewFlowConfig / UpsertReviewFlowConfig，本文件不重复实现。
//
// buildInClause 为 repository 包内已有辅助（review_v2_repo.go 等处使用），本文件直接复用。
//
// ★ B3 修复（账户与权限·第一批）★
//   1. 新增 ListCWPendingReviewsBySchools：按"审核学校白名单"查待审课件（L1/L2 通用），
//      供区域管理员辖区视图使用。过滤列统一用 review_school_id——SubmitForReview 提交时
//      即写入作者学校，L1/L2 待审阶段该列都有值，与既有 L2 单校查询同列同口径。
//   2. GetCWReviewStats 堵住待审计数泄漏：历史实现里"非 admin 且无教研组"的 L1、
//      以及所有非 admin 的 L2，都落到无过滤的全局计数（区域/学校管理员看到全系统
//      所有学校的待审数）。现改为：admin 全局；有教研组按组员；有学校白名单按
//      review_school_id；三者皆无 → 计 0（fail-closed），口径与各角色待审列表严格一致。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	// ErrCWReviewNotFound 课件审核记录不存在
	ErrCWReviewNotFound = errors.New("课件审核记录不存在")
)

// ==================== 审核记录 CRUD ====================

// CreateCoursewareReview 创建课件多级审核记录
func CreateCoursewareReview(ctx context.Context, review *models.CoursewareReview) error {
	query := `
		INSERT INTO courseware_reviews
			(courseware_id, review_level, reviewer_id, decision, score, comment, dimensions, review_round)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	dimensions := review.Dimensions
	if dimensions == "" {
		dimensions = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		review.CoursewareID, review.ReviewLevel, review.ReviewerID,
		review.Decision, review.Score, review.Comment, dimensions, review.ReviewRound,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建课件审核记录失败: %w", err)
	}
	return nil
}

// ListCoursewareReviewsByCourseware 获取某课件的所有审核记录（含审核员名称）
func ListCoursewareReviewsByCourseware(ctx context.Context, coursewareID string) ([]*models.CWReviewListItem, error) {
	query := `
		SELECT r.id, r.courseware_id, r.review_level, r.reviewer_id,
		       COALESCE(u.display_name, u.username, '') AS reviewer_name,
		       r.decision, r.score, r.comment, r.review_round, r.created_at
		FROM courseware_reviews r
		LEFT JOIN users u ON u.id = r.reviewer_id
		WHERE r.courseware_id = $1
		ORDER BY r.review_round ASC, r.review_level ASC, r.created_at ASC`
	rows, err := database.DB.Query(ctx, query, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("查询课件审核记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWReviewListItem
	for rows.Next() {
		item := &models.CWReviewListItem{}
		err := rows.Scan(
			&item.ID, &item.CoursewareID, &item.ReviewLevel, &item.ReviewerID,
			&item.ReviewerName, &item.Decision, &item.Score, &item.Comment,
			&item.ReviewRound, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描课件审核记录行失败: %w", err)
		}
		item.LevelName = models.ReviewLevelNameMap[item.ReviewLevel]
		items = append(items, item)
	}
	return items, nil
}

// CountCoursewareReviewsByLevel 统计某课件在某审核级别的记录数（用于计算轮次）
func CountCoursewareReviewsByLevel(ctx context.Context, coursewareID string, level int) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM courseware_reviews
		 WHERE courseware_id = $1 AND review_level = $2`,
		coursewareID, level,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计课件审核记录数失败: %w", err)
	}
	return count, nil
}

// ==================== 待审核列表查询 ====================

// cwPendingSelectColumns 待审核列表统一查询列（避免列顺序漂移）
// 课件无 group_id 列，故不查教研组名；学校名通过作者 school_members 关联取一条。
const cwPendingSelectColumns = `
	c.id, c.title, c.subject, c.grade, c.page_count,
	COALESCE(c.source_type, 'lesson_plan'),
	c.user_id, COALESCE(u.display_name, u.username, ''),
	COALESCE(sch.name, ''),
	c.review_level, c.updated_at`

// cwPendingFromJoin 待审核列表统一 FROM+JOIN（作者名 + 作者学校名）
const cwPendingFromJoin = `
	FROM coursewares c
	LEFT JOIN users u ON u.id = c.user_id
	LEFT JOIN LATERAL (
		SELECT o.name
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id AND o.status = 'active'
		WHERE sm.user_id = c.user_id
		LIMIT 1
	) sch ON true`

// scanCWPendingItem 扫描一行待审核项（顺序须与 cwPendingSelectColumns 一致）
func scanCWPendingItem(rows pgx.Rows) (*models.CWPendingReviewItem, error) {
	item := &models.CWPendingReviewItem{}
	err := rows.Scan(
		&item.CoursewareID, &item.Title, &item.Subject, &item.Grade, &item.PageCount,
		&item.SourceType,
		&item.AuthorID, &item.AuthorName,
		&item.SchoolName,
		&item.ReviewLevel, &item.SubmittedAt,
	)
	if err != nil {
		return nil, err
	}
	item.SourceName = models.CWSourceNameMap[item.SourceType]
	return item, nil
}

// ListCWPendingReviewsL1 获取 L1 待审核课件列表（教研组长/骨干可见"本组成员"提交的课件）
//
// 与教案差异：课件无 group_id，"组内课件"= 课件作者 ∈ 审核员所在(lead/backbone)教研组的成员。
// 流程：先查审核员的 lead/backbone 教研组 → 取这些组的全部成员 user_id → 课件作者 ∈ 这些成员 且 publish_state='submitted' 且 review_level=0。
func ListCWPendingReviewsL1(ctx context.Context, reviewerID string, limit int, offset int) ([]*models.CWPendingReviewItem, int, error) {
	// 1) 审核员作为 lead/backbone 的教研组成员（即"本组所有人"）
	memberQuery := `
		SELECT DISTINCT m2.user_id
		FROM teaching_group_members m1
		JOIN teaching_group_members m2 ON m2.group_id = m1.group_id
		WHERE m1.user_id = $1 AND m1.role IN ('lead', 'backbone')`
	memberRows, err := database.DB.Query(ctx, memberQuery, reviewerID)
	if err != nil {
		return nil, 0, fmt.Errorf("查询审核员可管成员失败: %w", err)
	}
	defer memberRows.Close()

	var memberIDs []string
	for memberRows.Next() {
		var uid string
		if err := memberRows.Scan(&uid); err == nil {
			memberIDs = append(memberIDs, uid)
		}
	}
	if len(memberIDs) == 0 {
		return []*models.CWPendingReviewItem{}, 0, nil
	}

	inClause, args := buildInClause(memberIDs, 1)

	// 2) 统计总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM coursewares c
		WHERE c.publish_state = 'submitted' AND c.review_level = 0 AND c.user_id IN (%s)`, inClause)
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课件L1待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	nextIdx := len(args) + 1
	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE c.publish_state = 'submitted' AND c.review_level = 0 AND c.user_id IN (%s)
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`, inClause, nextIdx, nextIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件L1待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWPendingReviewItem
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描课件L1待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}
	return items, total, nil
}

// ListCWPendingReviewsL1All 获取全部 L1 待审核课件（admin 场景，不限教研组）
// 条件：publish_state='submitted' AND review_level=0。
func ListCWPendingReviewsL1All(ctx context.Context, limit int, offset int) ([]*models.CWPendingReviewItem, int, error) {
	var total int
	if err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM coursewares c
		 WHERE c.publish_state = 'submitted' AND c.review_level = 0`,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课件全量L1待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}
	listQuery := `
		SELECT` + cwPendingSelectColumns + cwPendingFromJoin + `
		WHERE c.publish_state = 'submitted' AND c.review_level = 0
		ORDER BY c.updated_at ASC
		LIMIT $1 OFFSET $2`
	rows, err := database.DB.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件全量L1待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWPendingReviewItem
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描课件全量L1待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}
	return items, total, nil
}

// ListCWPendingReviewsL2 获取 L2 待审核课件列表
// schoolID 非空只查该校（按 review_school_id 匹配）；为空查全部（admin 场景）。
// 条件：publish_state='submitted' AND review_level=1。
func ListCWPendingReviewsL2(ctx context.Context, schoolID string, limit int, offset int) ([]*models.CWPendingReviewItem, int, error) {
	var countQuery, listQuery string
	var countArgs, listArgs []interface{}

	if schoolID != "" {
		countQuery = `SELECT COUNT(*) FROM coursewares c
			WHERE c.publish_state = 'submitted' AND c.review_level = 1 AND c.review_school_id = $1`
		countArgs = []interface{}{schoolID}
	} else {
		countQuery = `SELECT COUNT(*) FROM coursewares c
			WHERE c.publish_state = 'submitted' AND c.review_level = 1`
		countArgs = []interface{}{}
	}

	var total int
	if err := database.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课件L2待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	if schoolID != "" {
		listQuery = `SELECT` + cwPendingSelectColumns + cwPendingFromJoin + `
			WHERE c.publish_state = 'submitted' AND c.review_level = 1 AND c.review_school_id = $1
			ORDER BY c.updated_at ASC
			LIMIT $2 OFFSET $3`
		listArgs = []interface{}{schoolID, limit, offset}
	} else {
		listQuery = `SELECT` + cwPendingSelectColumns + cwPendingFromJoin + `
			WHERE c.publish_state = 'submitted' AND c.review_level = 1
			ORDER BY c.updated_at ASC
			LIMIT $1 OFFSET $2`
		listArgs = []interface{}{limit, offset}
	}

	rows, err := database.DB.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件L2待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWPendingReviewItem
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描课件L2待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL2
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL2]
		items = append(items, item)
	}
	return items, total, nil
}

// ListCWPendingReviewsBySchools 按"审核学校白名单"获取待审核课件列表（B3 新增，区域管理员辖区视图）
//
// 参数 pendingForLevel：待审核的目标级别——
//   models.ReviewLevelL1(=1) → 查 DB 中 review_level=0 的"待L1"课件；
//   models.ReviewLevelL2(=2) → 查 DB 中 review_level=1 的"待L2"课件。
//   （DB 的 review_level 语义是"已通过到第几级"，待审级别 = 已通过级别 + 1，故 dbLevel = pendingForLevel - 1）
//
// 过滤列统一用 review_school_id：SubmitForReview 提交时即写入作者学校，
// L1/L2 待审阶段该列都有值，与既有 L2 单校查询同列同口径。
// 学校白名单为空时直接返回空列表（fail-closed，绝不退化为全局查询）。
func ListCWPendingReviewsBySchools(ctx context.Context, schoolIDs []string, pendingForLevel int, limit int, offset int) ([]*models.CWPendingReviewItem, int, error) {
	if len(schoolIDs) == 0 {
		return []*models.CWPendingReviewItem{}, 0, nil
	}
	dbLevel := pendingForLevel - 1

	// $1 = dbLevel，学校白名单占位符从 $2 起
	inClause, schoolArgs := buildInClause(schoolIDs, 2)
	args := append([]interface{}{dbLevel}, schoolArgs...)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM coursewares c
		WHERE c.publish_state = 'submitted' AND c.review_level = $1 AND c.review_school_id IN (%s)`, inClause)
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("按学校统计课件待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}
	nextIdx := len(args) + 1
	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE c.publish_state = 'submitted' AND c.review_level = $1 AND c.review_school_id IN (%s)
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`, inClause, nextIdx, nextIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("按学校查询课件待审核列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.CWPendingReviewItem{}
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描按学校待审核行失败: %w", err)
		}
		item.ReviewLevel = pendingForLevel
		item.LevelName = models.ReviewLevelNameMap[pendingForLevel]
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 已审核记录查询 ====================

// ListCWReviewedRecords 查询课件已审核记录列表（按级别 + 审核员 + 决策过滤）
// isAdmin=true 时不限 reviewerID（查所有人的）。
func ListCWReviewedRecords(ctx context.Context, reviewerID string, level int, decision string, isAdmin bool, limit int, offset int) ([]*models.CWReviewedListItem, int, error) {
	where := "r.review_level = $1"
	args := []interface{}{level}
	argIdx := 2

	if !isAdmin && reviewerID != "" {
		where += fmt.Sprintf(" AND r.reviewer_id = $%d", argIdx)
		args = append(args, reviewerID)
		argIdx++
	}
	if decision != "" {
		where += fmt.Sprintf(" AND r.decision = $%d", argIdx)
		args = append(args, decision)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM courseware_reviews r WHERE %s`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课件已审核记录数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	listQuery := fmt.Sprintf(`
		SELECT r.id, r.courseware_id,
		       COALESCE(c.title, '') AS courseware_title,
		       COALESCE(c.subject, '') AS subject,
		       COALESCE(c.grade, '') AS grade,
		       COALESCE(author.display_name, author.username, '') AS author_name,
		       r.review_level,
		       COALESCE(reviewer.display_name, reviewer.username, '') AS reviewer_name,
		       r.decision, r.score, r.comment, r.created_at
		FROM courseware_reviews r
		LEFT JOIN coursewares c ON c.id = r.courseware_id
		LEFT JOIN users author ON author.id = c.user_id
		LEFT JOIN users reviewer ON reviewer.id = r.reviewer_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件已审核记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CWReviewedListItem
	for rows.Next() {
		item := &models.CWReviewedListItem{}
		err := rows.Scan(
			&item.ID, &item.CoursewareID, &item.CoursewareTitle, &item.Subject,
			&item.Grade, &item.AuthorName, &item.ReviewLevel,
			&item.ReviewerName, &item.Decision, &item.Score, &item.Comment, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描课件已审核记录行失败: %w", err)
		}
		item.LevelName = models.ReviewLevelNameMap[item.ReviewLevel]
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 审核统计 ====================

// GetCWReviewStats 获取课件审核统计（B3 修复版：待审计数按调用方白名单收窄，fail-closed）
//
// 待审核数（TotalPending）口径必须与该角色的待审列表严格一致，按以下优先级：
//   1. isAdmin=true                       → 全局计数（对应 admin 的全量列表）；
//   2. level=L1 且 memberIDs 非空          → 按"教研组成员作者"计数（对应 lead/backbone 的 L1 列表）；
//   3. schoolIDs 非空                     → 按 review_school_id ∈ 白名单计数
//                                           （senior 传 [本校] 对应其 L2 列表；region_admin 传辖区学校对应其 L1+L2 列表）；
//   4. 三者皆无                           → 计 0（fail-closed）。
//
// ⚠ 历史泄漏（B3 堵住）：旧实现里"非 admin 且无教研组"的 L1、以及所有非 admin 的 L2，
//   都落到无过滤的全局计数——区域管理员/学校管理员的统计卡显示全系统所有学校的待审数。
//
// 已审核/已通过/已退回三项为"审核员个人产出"口径：admin 看全局、非 admin 看本人 reviewer_id，
// 不涉及跨校泄漏，维持原逻辑不变。
func GetCWReviewStats(ctx context.Context, reviewerID string, level int, isAdmin bool, memberIDs []string, schoolIDs []string) (*models.CWReviewStatsResponse, error) {
	stats := &models.CWReviewStatsResponse{}

	// dbPendingLevel：DB 的 review_level 语义是"已通过到第几级"，待审级别 = 已通过级别 + 1
	dbPendingLevel := level - 1

	// ---------- 待审核数（口径与列表一致，fail-closed）----------
	if isAdmin {
		// admin：全局计数
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM coursewares c
			 WHERE c.publish_state = 'submitted' AND c.review_level = $1`, dbPendingLevel,
		).Scan(&stats.TotalPending)
	} else if level == models.ReviewLevelL1 && len(memberIDs) > 0 {
		// lead/backbone 的 L1：按教研组成员作者计数
		inClause, args := buildInClause(memberIDs, 1)
		q := fmt.Sprintf(`SELECT COUNT(*) FROM coursewares c
			WHERE c.publish_state = 'submitted' AND c.review_level = 0 AND c.user_id IN (%s)`, inClause)
		_ = database.DB.QueryRow(ctx, q, args...).Scan(&stats.TotalPending)
	} else if len(schoolIDs) > 0 {
		// senior（本校 L2）/ region_admin（辖区 L1+L2）：按 review_school_id 白名单计数
		inClause, schoolArgs := buildInClause(schoolIDs, 2)
		q := fmt.Sprintf(`SELECT COUNT(*) FROM coursewares c
			WHERE c.publish_state = 'submitted' AND c.review_level = $1 AND c.review_school_id IN (%s)`, inClause)
		args := append([]interface{}{dbPendingLevel}, schoolArgs...)
		_ = database.DB.QueryRow(ctx, q, args...).Scan(&stats.TotalPending)
	}
	// 以上分支均不匹配（非 admin、无教研组、无学校白名单）→ TotalPending 保持 0（fail-closed）

	// ---------- 已审核/已通过/已退回（审核员个人产出口径，原逻辑不变）----------
	if isAdmin {
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE review_level = $1`, level,
		).Scan(&stats.TotalReviewed)
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE review_level = $1 AND decision = 'approved'`, level,
		).Scan(&stats.TotalApproved)
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE review_level = $1 AND decision = 'revision'`, level,
		).Scan(&stats.TotalRevision)
	} else {
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE reviewer_id = $1 AND review_level = $2`,
			reviewerID, level,
		).Scan(&stats.TotalReviewed)
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE reviewer_id = $1 AND review_level = $2 AND decision = 'approved'`,
			reviewerID, level,
		).Scan(&stats.TotalApproved)
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM courseware_reviews WHERE reviewer_id = $1 AND review_level = $2 AND decision = 'revision'`,
			reviewerID, level,
		).Scan(&stats.TotalRevision)
	}

	return stats, nil
}

// ==================== 辅助：审核员可管教研组成员 ====================

// GetCWReviewableMemberIDs 获取审核员作为 lead/backbone 的教研组的全部成员 user_id（含本人）
// 供 L1 待审核列表与统计共用同一口径（"我能审本组所有人的课件"）。
func GetCWReviewableMemberIDs(ctx context.Context, reviewerID string) ([]string, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT DISTINCT m2.user_id
		FROM teaching_group_members m1
		JOIN teaching_group_members m2 ON m2.group_id = m1.group_id
		WHERE m1.user_id = $1 AND m1.role IN ('lead', 'backbone')`, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("查询审核员可管成员失败: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ==================== 课件审核态联合更新（复用阶段1的 UpdateCoursewarePublishState）====================
// 说明：课件发布态/审核层级/审核学校的写入，统一走阶段1已有的
//   repository.UpdateCoursewarePublishState(ctx, id, publishState, reviewLevel, reviewSchoolID)
// 本文件不重复实现，service 层直接调用该函数即可。
