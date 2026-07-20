package repository

// review_region_scope_repo.go
//
// 本文件集中实现区域审核、统计和抽查所需的教育域范围查询。
//
// 区域管理员审核范围必须同时满足：
//   - 学校位于其管辖区域；
//   - 学校状态为 active；
//   - 学校教育域与管理员固定教育域一致；
//   - 教案或课件的教育域快照与固定教育域一致。
//
// 学校白名单与固定教育域由 services.ResolveRegionAdminEducationScope
// 在 Service 层解析。本文件只忠实执行白名单和教育域双重过滤。
//
// 安全规则：
//   - 学校白名单为空时直接返回空集；
//   - 只接受 k12、vocational、adult；
//   - mixed、common、空值和非法值全部返回空集；
//   - 查询错误向上返回，不静默退化为全局查询；
//   - 列表和统计使用相同的学校列、状态和教育域条件。

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// normalizeReviewScopeEducationDomain 校验并规范化审核范围教育域。
func normalizeReviewScopeEducationDomain(value string) (string, bool) {
	domain := strings.ToLower(strings.TrimSpace(value))
	return domain, models.IsTeachingEducationDomain(domain)
}

// scanRegionPendingLessonReview 扫描区域管理员待审教案列表行。
func scanRegionPendingLessonReview(
	rows pgx.Rows,
	pendingForLevel int,
) (*models.PendingReviewItem, error) {
	item := &models.PendingReviewItem{}

	if err := rows.Scan(
		&item.LessonPlanID,
		&item.Title,
		&item.Subject,
		&item.Grade,
		&item.AuthorID,
		&item.AuthorName,
		&item.GroupID,
		&item.GroupName,
		&item.SchoolName,
		&item.ReviewLevel,
		&item.AIReviewScore,
		&item.SubmittedAt,
	); err != nil {
		return nil, err
	}

	item.ReviewLevel = pendingForLevel
	item.LevelName = models.ReviewLevelNameMap[pendingForLevel]

	return item, nil
}

// ListPendingLessonReviewsBySchoolsAndDomain 按学校白名单和教育域查询待审教案。
//
// pendingForLevel 表示用户界面中的待审级别：
//   - ReviewLevelL1：数据库 review_level=0；
//   - ReviewLevelL2：数据库 review_level=1。
//
// L1 和 L2 都使用 review_school_id 过滤。教案提交审核时已经写入该列，
// 因此列表和统计不依赖作者当前校籍，也不会因作者换校扩大权限。
func ListPendingLessonReviewsBySchoolsAndDomain(
	ctx context.Context,
	schoolIDs []string,
	pendingForLevel int,
	educationDomain string,
	limit int,
	offset int,
) ([]*models.PendingReviewItem, int, error) {
	if len(schoolIDs) == 0 {
		return []*models.PendingReviewItem{}, 0, nil
	}

	domain, valid := normalizeReviewScopeEducationDomain(
		educationDomain,
	)
	if !valid {
		return []*models.PendingReviewItem{}, 0, nil
	}

	if pendingForLevel != models.ReviewLevelL1 &&
		pendingForLevel != models.ReviewLevelL2 {
		return []*models.PendingReviewItem{}, 0, nil
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	dbLevel := pendingForLevel - 1

	const countQuery = `
		SELECT COUNT(*)
		FROM lesson_plans lp
		WHERE lp.status = 'submitted'
		  AND lp.review_level = $1
		  AND lp.review_school_id::text = ANY($2)
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, ''))) = $3
		  AND lp.deleted_at IS NULL
	`

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		dbLevel,
		schoolIDs,
		domain,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"统计区域同域待审教案失败: %w",
			err,
		)
	}

	const listQuery = `
		SELECT
			lp.id,
			lp.title,
			lp.subject,
			lp.grade,
			lp.author_id,
			COALESCE(u.display_name, '') AS author_name,
			lp.group_id,
			COALESCE(tg.name, '') AS group_name,
			COALESCE(o.name, '') AS school_name,
			lp.review_level,
			lp.ai_review_score,
			lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u
		  ON u.id = lp.author_id
		LEFT JOIN teaching_groups tg
		  ON tg.id = lp.group_id
		LEFT JOIN organizations o
		  ON o.id = lp.review_school_id
		 AND o.type = 'school'
		 AND o.status = 'active'
		WHERE lp.status = 'submitted'
		  AND lp.review_level = $1
		  AND lp.review_school_id::text = ANY($2)
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, ''))) = $3
		  AND lp.deleted_at IS NULL
		ORDER BY lp.updated_at ASC
		LIMIT $4 OFFSET $5
	`

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		dbLevel,
		schoolIDs,
		domain,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询区域同域待审教案失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.PendingReviewItem, 0)
	for rows.Next() {
		item, scanErr := scanRegionPendingLessonReview(
			rows,
			pendingForLevel,
		)
		if scanErr != nil {
			return nil, 0, fmt.Errorf(
				"扫描区域同域待审教案失败: %w",
				scanErr,
			)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历区域同域待审教案失败: %w",
			err,
		)
	}

	return items, total, nil
}

// GetLessonReviewStatsBySchoolsAndDomain 查询区域管理员教案审核聚合统计。
//
// 待审统计使用当前 review_school_id。
// 已审核统计在终审或退回后 review_school_id 可能被清空，因此按以下顺序恢复学校：
//   - review_school_id；
//   - lesson_plans.school_id；
//   - 教案绑定教研组的 school_id。
//
// 所有统计仍必须通过教育域快照精确过滤。
func GetLessonReviewStatsBySchoolsAndDomain(
	ctx context.Context,
	level int,
	schoolIDs []string,
	educationDomain string,
) (*models.ReviewStatsResponse, error) {
	stats := &models.ReviewStatsResponse{}

	if len(schoolIDs) == 0 {
		return stats, nil
	}

	domain, valid := normalizeReviewScopeEducationDomain(
		educationDomain,
	)
	if !valid {
		return stats, nil
	}

	if level != models.ReviewLevelL1 &&
		level != models.ReviewLevelL2 {
		return stats, nil
	}

	dbPendingLevel := level - 1

	const pendingQuery = `
		SELECT COUNT(*)
		FROM lesson_plans lp
		WHERE lp.status = 'submitted'
		  AND lp.review_level = $1
		  AND lp.review_school_id::text = ANY($2)
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, ''))) = $3
		  AND lp.deleted_at IS NULL
	`

	if err := database.DB.QueryRow(
		ctx,
		pendingQuery,
		dbPendingLevel,
		schoolIDs,
		domain,
	).Scan(&stats.TotalPending); err != nil {
		return nil, fmt.Errorf(
			"统计区域同域待审教案失败: %w",
			err,
		)
	}

	const reviewedQuery = `
		SELECT
			COUNT(*) AS total_reviewed,
			COUNT(*) FILTER (
				WHERE r.decision = 'approved'
			) AS total_approved,
			COUNT(*) FILTER (
				WHERE r.decision = 'revision'
			) AS total_revision
		FROM lesson_plan_reviews_v2 r
		JOIN lesson_plans lp
		  ON lp.id = r.lesson_plan_id
		LEFT JOIN teaching_groups tg
		  ON tg.id = lp.group_id
		WHERE r.review_level = $1
		  AND COALESCE(
				lp.review_school_id::text,
				lp.school_id::text,
				tg.school_id::text,
				''
		  ) = ANY($2)
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, ''))) = $3
		  AND lp.deleted_at IS NULL
	`

	if err := database.DB.QueryRow(
		ctx,
		reviewedQuery,
		level,
		schoolIDs,
		domain,
	).Scan(
		&stats.TotalReviewed,
		&stats.TotalApproved,
		&stats.TotalRevision,
	); err != nil {
		return nil, fmt.Errorf(
			"统计区域同域教案审核结果失败: %w",
			err,
		)
	}

	return stats, nil
}

// GetCoursewareReviewStatsBySchoolsAndDomain 查询区域管理员课件审核聚合统计。
//
// 待审阶段直接按 review_school_id 过滤。
// 退回课件可能清空 review_school_id，已审核统计使用作者当前 active 校籍作为兜底。
// 教育域始终以课件创建时快照为准。
func GetCoursewareReviewStatsBySchoolsAndDomain(
	ctx context.Context,
	level int,
	schoolIDs []string,
	educationDomain string,
) (*models.CWReviewStatsResponse, error) {
	stats := &models.CWReviewStatsResponse{}

	if len(schoolIDs) == 0 {
		return stats, nil
	}

	domain, valid := normalizeReviewScopeEducationDomain(
		educationDomain,
	)
	if !valid {
		return stats, nil
	}

	if level != models.ReviewLevelL1 &&
		level != models.ReviewLevelL2 {
		return stats, nil
	}

	dbPendingLevel := level - 1

	const pendingQuery = `
		SELECT COUNT(*)
		FROM coursewares c
		WHERE c.publish_state = 'submitted'
		  AND c.review_level = $1
		  AND c.review_school_id::text = ANY($2)
		  AND LOWER(BTRIM(COALESCE(c.education_domain, ''))) = $3
		  AND c.deleted_at IS NULL
	`

	if err := database.DB.QueryRow(
		ctx,
		pendingQuery,
		dbPendingLevel,
		schoolIDs,
		domain,
	).Scan(&stats.TotalPending); err != nil {
		return nil, fmt.Errorf(
			"统计区域同域待审课件失败: %w",
			err,
		)
	}

	const reviewedQuery = `
		SELECT
			COUNT(*) AS total_reviewed,
			COUNT(*) FILTER (
				WHERE r.decision = 'approved'
			) AS total_approved,
			COUNT(*) FILTER (
				WHERE r.decision = 'revision'
			) AS total_revision
		FROM courseware_reviews r
		JOIN coursewares c
		  ON c.id = r.courseware_id
		LEFT JOIN LATERAL (
			SELECT sm.school_id::text AS school_id
			FROM school_members sm
			JOIN organizations o
			  ON o.id = sm.school_id
			 AND o.type = 'school'
			 AND o.status = 'active'
			WHERE sm.user_id = c.user_id
			ORDER BY sm.joined_at ASC
			LIMIT 1
		) author_school ON true
		WHERE r.review_level = $1
		  AND COALESCE(
				c.review_school_id::text,
				author_school.school_id,
				''
		  ) = ANY($2)
		  AND LOWER(BTRIM(COALESCE(c.education_domain, ''))) = $3
		  AND c.deleted_at IS NULL
	`

	if err := database.DB.QueryRow(
		ctx,
		reviewedQuery,
		level,
		schoolIDs,
		domain,
	).Scan(
		&stats.TotalReviewed,
		&stats.TotalApproved,
		&stats.TotalRevision,
	); err != nil {
		return nil, fmt.Errorf(
			"统计区域同域课件审核结果失败: %w",
			err,
		)
	}

	return stats, nil
}

// SamplePublishedPlansForInspectionSameDomain 从指定学校抽取同域已发布教案。
//
// 候选必须同时满足：
//   - 学校真实存在、类型为 school、状态为 active；
//   - 学校教育域是具体教学域；
//   - 教案教育域快照与学校教育域完全一致；
//   - 教案未软删除；
//   - 教案不存在未结束的抽查记录。
func SamplePublishedPlansForInspectionSameDomain(
	ctx context.Context,
	schoolID string,
	sampleRate float64,
	batchID string,
) (int, error) {
	if strings.TrimSpace(schoolID) == "" {
		return 0, nil
	}

	if sampleRate <= 0 || sampleRate > 1.0 {
		sampleRate = 0.20
	}

	const query = `
		INSERT INTO inspection_records (
			lesson_plan_id,
			sample_batch,
			status,
			priority
		)
		SELECT
			lp.id,
			$1,
			'pending',
			0
		FROM lesson_plans lp
		JOIN organizations o
		  ON o.id = lp.review_school_id
		 AND o.id = $2
		 AND o.type = 'school'
		 AND o.status = 'active'
		WHERE lp.status = 'published_shared'
		  AND lp.deleted_at IS NULL
		  AND LOWER(BTRIM(COALESCE(o.education_domain, '')))
				IN ('k12', 'vocational', 'adult')
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, '')))
				= LOWER(BTRIM(COALESCE(o.education_domain, '')))
		  AND NOT EXISTS (
			SELECT 1
			FROM inspection_records ir
			WHERE ir.lesson_plan_id = lp.id
			  AND ir.status NOT IN ('passed', 'revoked')
		  )
		ORDER BY random()
		LIMIT (
			SELECT GREATEST(
				1,
				CEIL(COUNT(*) * $3)
			)
			FROM lesson_plans lp2
			JOIN organizations o2
			  ON o2.id = lp2.review_school_id
			 AND o2.id = $2
			 AND o2.type = 'school'
			 AND o2.status = 'active'
			WHERE lp2.status = 'published_shared'
			  AND lp2.deleted_at IS NULL
			  AND LOWER(BTRIM(COALESCE(o2.education_domain, '')))
					IN ('k12', 'vocational', 'adult')
			  AND LOWER(BTRIM(COALESCE(lp2.education_domain, '')))
					= LOWER(BTRIM(COALESCE(o2.education_domain, '')))
			  AND NOT EXISTS (
				SELECT 1
				FROM inspection_records ir2
				WHERE ir2.lesson_plan_id = lp2.id
				  AND ir2.status NOT IN ('passed', 'revoked')
			  )
		)
	`

	result, err := database.DB.Exec(
		ctx,
		query,
		batchID,
		schoolID,
		sampleRate,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"同域抽样失败: %w",
			err,
		)
	}

	return int(result.RowsAffected()), nil
}

// QueryDistinctSameDomainReviewSchoolIDs 返回存在同域已发布教案的 active 学校。
func QueryDistinctSameDomainReviewSchoolIDs(
	ctx context.Context,
) ([]string, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT DISTINCT o.id::text
		FROM lesson_plans lp
		JOIN organizations o
		  ON o.id = lp.review_school_id
		 AND o.type = 'school'
		 AND o.status = 'active'
		WHERE lp.status = 'published_shared'
		  AND lp.deleted_at IS NULL
		  AND LOWER(BTRIM(COALESCE(o.education_domain, '')))
				IN ('k12', 'vocational', 'adult')
		  AND LOWER(BTRIM(COALESCE(lp.education_domain, '')))
				= LOWER(BTRIM(COALESCE(o.education_domain, '')))
		ORDER BY o.id::text
	`)
	if err != nil {
		return nil, fmt.Errorf(
			"查询同域抽查学校失败: %w",
			err,
		)
	}
	defer rows.Close()

	schoolIDs := make([]string, 0)
	for rows.Next() {
		var schoolID string
		if err := rows.Scan(&schoolID); err != nil {
			return nil, fmt.Errorf(
				"扫描同域抽查学校失败: %w",
				err,
			)
		}
		schoolIDs = append(schoolIDs, schoolID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历同域抽查学校失败: %w",
			err,
		)
	}

	return schoolIDs, nil
}
