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
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	// ErrCWReviewNotFound 课件审核记录不存在
	ErrCWReviewNotFound = errors.New("课件审核记录不存在")
)

// buildCWReviewEducationDomainFilter 构建审核查询的教育域SQL过滤。
func buildCWReviewEducationDomainFilter(
	alias string,
	currentEducationDomain string,
	startIdx int,
) (string, []interface{}, int) {
	if alias == "" {
		alias = "c"
	}
	if startIdx <= 0 {
		startIdx = 1
	}

	domain := strings.ToLower(
		strings.TrimSpace(currentEducationDomain),
	)

	switch {
	case models.IsTeachingEducationDomain(domain):
		return fmt.Sprintf(
				" AND %s.education_domain = $%d",
				alias,
				startIdx,
			),
			[]interface{}{domain},
			startIdx + 1

	case domain == models.EducationDomainMixed:
		return fmt.Sprintf(
				" AND %s.education_domain IN ('k12', 'vocational', 'adult')",
				alias,
			),
			nil,
			startIdx

	default:
		return " AND 1 = 0", nil, startIdx
	}
}

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
func ListCWPendingReviewsL1(
	ctx context.Context,
	reviewerID string,
	currentEducationDomain string,
	limit int,
	offset int,
) ([]*models.CWPendingReviewItem, int, error) {
	memberRows, err := database.DB.Query(ctx, `
		SELECT DISTINCT m2.user_id
		FROM teaching_group_members m1
		JOIN teaching_group_members m2
			ON m2.group_id = m1.group_id
		WHERE m1.user_id = $1
			AND m1.role IN ('lead', 'backbone')`,
		reviewerID,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询审核员可管成员失败: %w",
			err,
		)
	}
	defer memberRows.Close()

	var memberIDs []string
	for memberRows.Next() {
		var userID string
		if err := memberRows.Scan(&userID); err == nil {
			memberIDs = append(memberIDs, userID)
		}
	}
	if len(memberIDs) == 0 {
		return []*models.CWPendingReviewItem{}, 0, nil
	}

	inClause, args := buildInClause(memberIDs, 1)
	domainClause, domainArgs, nextIdx :=
		buildCWReviewEducationDomainFilter(
			"c",
			currentEducationDomain,
			len(args)+1,
		)
	args = append(args, domainArgs...)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM coursewares c
		WHERE c.publish_state = 'submitted'
			AND c.review_level = 0
			AND c.user_id IN (%s)%s`,
		inClause,
		domainClause,
	)

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE c.publish_state = 'submitted'
			AND c.review_level = 0
			AND c.user_id IN (%s)%s
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`,
		inClause,
		domainClause,
		nextIdx,
		nextIdx+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*models.CWPendingReviewItem{}
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, err
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName =
			models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// ListCWPendingReviewsL1All 获取全部 L1 待审核课件（admin 场景，不限教研组）
// 条件：publish_state='submitted' AND review_level=0。
func ListCWPendingReviewsL1All(
	ctx context.Context,
	currentEducationDomain string,
	limit int,
	offset int,
) ([]*models.CWPendingReviewItem, int, error) {
	domainClause, args, nextIdx :=
		buildCWReviewEducationDomainFilter(
			"c",
			currentEducationDomain,
			1,
		)

	countQuery := `
		SELECT COUNT(*)
		FROM coursewares c
		WHERE c.publish_state = 'submitted'
			AND c.review_level = 0` +
		domainClause

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE c.publish_state = 'submitted'
			AND c.review_level = 0%s
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`,
		domainClause,
		nextIdx,
		nextIdx+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*models.CWPendingReviewItem{}
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, err
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName =
			models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// ListCWPendingReviewsL2 获取 L2 待审核课件列表
// schoolID 非空只查该校（按 review_school_id 匹配）；为空查全部（admin 场景）。
// 条件：publish_state='submitted' AND review_level=1。
func ListCWPendingReviewsL2(
	ctx context.Context,
	schoolID string,
	currentEducationDomain string,
	limit int,
	offset int,
) ([]*models.CWPendingReviewItem, int, error) {
	conditions := []string{
		"c.publish_state = 'submitted'",
		"c.review_level = 1",
	}
	args := []interface{}{}
	argIdx := 1

	if schoolID != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"c.review_school_id = $%d",
				argIdx,
			),
		)
		args = append(args, schoolID)
		argIdx++
	}

	domainClause, domainArgs, nextIdx :=
		buildCWReviewEducationDomainFilter(
			"c",
			currentEducationDomain,
			argIdx,
		)
	args = append(args, domainArgs...)

	whereClause := strings.Join(
		conditions,
		" AND ",
	) + domainClause

	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM coursewares c WHERE %s",
		whereClause,
	)

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE %s
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`,
		whereClause,
		nextIdx,
		nextIdx+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*models.CWPendingReviewItem{}
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, err
		}
		item.ReviewLevel = models.ReviewLevelL2
		item.LevelName =
			models.ReviewLevelNameMap[models.ReviewLevelL2]
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// ListCWPendingReviewsBySchools 按"审核学校白名单"获取待审核课件列表（B3 新增，区域管理员辖区视图）
//
// 参数 pendingForLevel：待审核的目标级别——
//
//	models.ReviewLevelL1(=1) → 查 DB 中 review_level=0 的"待L1"课件；
//	models.ReviewLevelL2(=2) → 查 DB 中 review_level=1 的"待L2"课件。
//	（DB 的 review_level 语义是"已通过到第几级"，待审级别 = 已通过级别 + 1，故 dbLevel = pendingForLevel - 1）
//
// 过滤列统一用 review_school_id：SubmitForReview 提交时即写入作者学校，
// L1/L2 待审阶段该列都有值，与既有 L2 单校查询同列同口径。
// 学校白名单为空时直接返回空列表（fail-closed，绝不退化为全局查询）。
func ListCWPendingReviewsBySchools(
	ctx context.Context,
	schoolIDs []string,
	pendingForLevel int,
	currentEducationDomain string,
	limit int,
	offset int,
) ([]*models.CWPendingReviewItem, int, error) {
	if len(schoolIDs) == 0 {
		return []*models.CWPendingReviewItem{}, 0, nil
	}

	dbLevel := pendingForLevel - 1
	inClause, schoolArgs := buildInClause(schoolIDs, 2)
	args := append(
		[]interface{}{dbLevel},
		schoolArgs...,
	)

	domainClause, domainArgs, nextIdx :=
		buildCWReviewEducationDomainFilter(
			"c",
			currentEducationDomain,
			len(args)+1,
		)
	args = append(args, domainArgs...)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM coursewares c
		WHERE c.publish_state = 'submitted'
			AND c.review_level = $1
			AND c.review_school_id IN (%s)%s`,
		inClause,
		domainClause,
	)

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT`+cwPendingSelectColumns+cwPendingFromJoin+`
		WHERE c.publish_state = 'submitted'
			AND c.review_level = $1
			AND c.review_school_id IN (%s)%s
		ORDER BY c.updated_at ASC
		LIMIT $%d OFFSET $%d`,
		inClause,
		domainClause,
		nextIdx,
		nextIdx+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*models.CWPendingReviewItem{}
	for rows.Next() {
		item, err := scanCWPendingItem(rows)
		if err != nil {
			return nil, 0, err
		}
		item.ReviewLevel = pendingForLevel
		item.LevelName =
			models.ReviewLevelNameMap[pendingForLevel]
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// ==================== 已审核记录查询 ====================

// ListCWReviewedRecords 查询课件已审核记录列表（按级别 + 审核员 + 决策过滤）
// isAdmin=true 时不限 reviewerID（查所有人的）。
func ListCWReviewedRecords(
	ctx context.Context,
	reviewerID string,
	level int,
	decision string,
	isAdmin bool,
	currentEducationDomain string,
	limit int,
	offset int,
) ([]*models.CWReviewedListItem, int, error) {
	where := "r.review_level = $1"
	args := []interface{}{level}
	argIdx := 2

	if !isAdmin {
		if reviewerID == "" {
			where += " AND 1 = 0"
		} else {
			where += fmt.Sprintf(
				" AND r.reviewer_id = $%d",
				argIdx,
			)
			args = append(args, reviewerID)
			argIdx++
		}
	}

	if decision != "" {
		where += fmt.Sprintf(
			" AND r.decision = $%d",
			argIdx,
		)
		args = append(args, decision)
		argIdx++
	}

	domainClause, domainArgs, nextIdx :=
		buildCWReviewEducationDomainFilter(
			"c",
			currentEducationDomain,
			argIdx,
		)
	where += domainClause
	args = append(args, domainArgs...)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM courseware_reviews r
		JOIN coursewares c ON c.id = r.courseware_id
		WHERE %s`,
		where,
	)

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(`
		SELECT
			r.id,
			r.courseware_id,
			COALESCE(c.title, ''),
			COALESCE(c.subject, ''),
			COALESCE(c.grade, ''),
			COALESCE(author.display_name, author.username, ''),
			r.review_level,
			COALESCE(reviewer.display_name, reviewer.username, ''),
			r.decision,
			r.score,
			r.comment,
			r.created_at
		FROM courseware_reviews r
		JOIN coursewares c ON c.id = r.courseware_id
		LEFT JOIN users author ON author.id = c.user_id
		LEFT JOIN users reviewer ON reviewer.id = r.reviewer_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d`,
		where,
		nextIdx,
		nextIdx+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*models.CWReviewedListItem{}
	for rows.Next() {
		item := &models.CWReviewedListItem{}
		if err := rows.Scan(
			&item.ID,
			&item.CoursewareID,
			&item.CoursewareTitle,
			&item.Subject,
			&item.Grade,
			&item.AuthorName,
			&item.ReviewLevel,
			&item.ReviewerName,
			&item.Decision,
			&item.Score,
			&item.Comment,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.LevelName =
			models.ReviewLevelNameMap[item.ReviewLevel]
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// ==================== 审核统计 ====================

// GetCWReviewStats 获取课件审核统计（B3 修复版：待审计数按调用方白名单收窄，fail-closed）
//
// 待审核数（TotalPending）口径必须与该角色的待审列表严格一致，按以下优先级：
//  1. isAdmin=true                       → 全局计数（对应 admin 的全量列表）；
//  2. level=L1 且 memberIDs 非空          → 按"教研组成员作者"计数（对应 lead/backbone 的 L1 列表）；
//  3. schoolIDs 非空                     → 按 review_school_id ∈ 白名单计数
//     （senior 传 [本校] 对应其 L2 列表；region_admin 传辖区学校对应其 L1+L2 列表）；
//  4. 三者皆无                           → 计 0（fail-closed）。
//
// ⚠ 历史泄漏（B3 堵住）：旧实现里"非 admin 且无教研组"的 L1、以及所有非 admin 的 L2，
//
//	都落到无过滤的全局计数——区域管理员/学校管理员的统计卡显示全系统所有学校的待审数。
//
// 已审核/已通过/已退回三项为"审核员个人产出"口径：admin 看全局、非 admin 看本人 reviewer_id，
// 不涉及跨校泄漏，维持原逻辑不变。
func GetCWReviewStats(
	ctx context.Context,
	reviewerID string,
	level int,
	isAdmin bool,
	memberIDs []string,
	schoolIDs []string,
	currentEducationDomain string,
) (*models.CWReviewStatsResponse, error) {
	stats := &models.CWReviewStatsResponse{}
	dbPendingLevel := level - 1

	if isAdmin {
		domainClause, domainArgs, _ :=
			buildCWReviewEducationDomainFilter(
				"c",
				currentEducationDomain,
				2,
			)
		args := append(
			[]interface{}{dbPendingLevel},
			domainArgs...,
		)
		query := `
			SELECT COUNT(*)
			FROM coursewares c
			WHERE c.publish_state = 'submitted'
				AND c.review_level = $1` +
			domainClause
		_ = database.DB.QueryRow(
			ctx,
			query,
			args...,
		).Scan(&stats.TotalPending)

	} else if level == models.ReviewLevelL1 &&
		len(memberIDs) > 0 {
		inClause, args := buildInClause(memberIDs, 1)
		domainClause, domainArgs, _ :=
			buildCWReviewEducationDomainFilter(
				"c",
				currentEducationDomain,
				len(args)+1,
			)
		args = append(args, domainArgs...)

		query := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM coursewares c
			WHERE c.publish_state = 'submitted'
				AND c.review_level = 0
				AND c.user_id IN (%s)%s`,
			inClause,
			domainClause,
		)
		_ = database.DB.QueryRow(
			ctx,
			query,
			args...,
		).Scan(&stats.TotalPending)

	} else if len(schoolIDs) > 0 {
		inClause, schoolArgs := buildInClause(schoolIDs, 2)
		args := append(
			[]interface{}{dbPendingLevel},
			schoolArgs...,
		)
		domainClause, domainArgs, _ :=
			buildCWReviewEducationDomainFilter(
				"c",
				currentEducationDomain,
				len(args)+1,
			)
		args = append(args, domainArgs...)

		query := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM coursewares c
			WHERE c.publish_state = 'submitted'
				AND c.review_level = $1
				AND c.review_school_id IN (%s)%s`,
			inClause,
			domainClause,
		)
		_ = database.DB.QueryRow(
			ctx,
			query,
			args...,
		).Scan(&stats.TotalPending)
	}

	if isAdmin {
		domainClause, domainArgs, _ :=
			buildCWReviewEducationDomainFilter(
				"c",
				currentEducationDomain,
				2,
			)
		args := append(
			[]interface{}{level},
			domainArgs...,
		)
		base := `
			FROM courseware_reviews r
			JOIN coursewares c ON c.id = r.courseware_id
			WHERE r.review_level = $1` +
			domainClause

		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base,
			args...,
		).Scan(&stats.TotalReviewed)
		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base+
				" AND r.decision = 'approved'",
			args...,
		).Scan(&stats.TotalApproved)
		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base+
				" AND r.decision = 'revision'",
			args...,
		).Scan(&stats.TotalRevision)

	} else {
		domainClause, domainArgs, _ :=
			buildCWReviewEducationDomainFilter(
				"c",
				currentEducationDomain,
				3,
			)
		args := append(
			[]interface{}{reviewerID, level},
			domainArgs...,
		)
		base := `
			FROM courseware_reviews r
			JOIN coursewares c ON c.id = r.courseware_id
			WHERE r.reviewer_id = $1
				AND r.review_level = $2` +
			domainClause

		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base,
			args...,
		).Scan(&stats.TotalReviewed)
		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base+
				" AND r.decision = 'approved'",
			args...,
		).Scan(&stats.TotalApproved)
		_ = database.DB.QueryRow(
			ctx,
			"SELECT COUNT(*) "+base+
				" AND r.decision = 'revision'",
			args...,
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
