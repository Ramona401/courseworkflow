package repository

// review_v2_repo.go — 多级审核数据访问层
//
// v127.2 修复：
//   - ListPendingReviewsL1All：admin全量查所有L1待审核（不限教研组）
//   - ListReviewedRecords：已审核记录列表（按级别+审核员+决策类型过滤）
//   - GetReviewStats：admin统计不限reviewer_id

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrReviewV2NotFound     = errors.New("审核记录不存在")
	ErrReviewConfigNotFound = errors.New("审核配置不存在")
)

// ==================== 审核记录 CRUD ====================

// CreateReviewV2 创建多级审核记录
func CreateReviewV2(ctx context.Context, review *models.ReviewV2) error {
	query := `
		INSERT INTO lesson_plan_reviews_v2
			(lesson_plan_id, review_level, reviewer_id, decision, score, comment, dimensions, review_round)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	dimensions := review.Dimensions
	if dimensions == "" {
		dimensions = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		review.LessonPlanID, review.ReviewLevel, review.ReviewerID,
		review.Decision, review.Score, review.Comment, dimensions, review.ReviewRound,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建多级审核记录失败: %w", err)
	}
	return nil
}

// ListReviewsV2ByPlan 获取某教案的所有多级审核记录（含审核员名称）
func ListReviewsV2ByPlan(ctx context.Context, planID string) ([]*models.ReviewV2ListItem, error) {
	query := `
		SELECT r.id, r.lesson_plan_id, r.review_level, r.reviewer_id,
		       COALESCE(u.display_name, '') AS reviewer_name,
		       r.decision, r.score, r.comment, r.review_round, r.created_at
		FROM lesson_plan_reviews_v2 r
		LEFT JOIN users u ON u.id = r.reviewer_id
		WHERE r.lesson_plan_id = $1
		ORDER BY r.review_round ASC, r.review_level ASC, r.created_at ASC
	`
	rows, err := database.DB.Query(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("查询多级审核记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ReviewV2ListItem
	for rows.Next() {
		item := &models.ReviewV2ListItem{}
		err := rows.Scan(
			&item.ID, &item.LessonPlanID, &item.ReviewLevel, &item.ReviewerID,
			&item.ReviewerName, &item.Decision, &item.Score, &item.Comment,
			&item.ReviewRound, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描审核记录行失败: %w", err)
		}
		item.LevelName = models.ReviewLevelNameMap[item.ReviewLevel]
		items = append(items, item)
	}
	return items, nil
}

// CountReviewsV2ByPlanAndLevel 统计某教案在某审核级别的审核记录数（用于计算轮次）
func CountReviewsV2ByPlanAndLevel(ctx context.Context, planID string, level int) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM lesson_plan_reviews_v2
		 WHERE lesson_plan_id = $1 AND review_level = $2`,
		planID, level,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计审核记录数失败: %w", err)
	}
	return count, nil
}

// ==================== 待审核列表查询 ====================

// ListPendingReviewsL1 获取L1待审核列表（教研组长/骨干可见本组提交的教案）
// 查询条件：status='submitted' AND review_level=0 AND group_id IN (用户lead/backbone的教研组)
func ListPendingReviewsL1(ctx context.Context, reviewerID string, limit int, offset int) ([]*models.PendingReviewItem, int, error) {
	// 先查询该审核员所在教研组（角色为lead或backbone）的ID列表
	groupQuery := `
		SELECT group_id FROM teaching_group_members
		WHERE user_id = $1 AND role IN ('lead', 'backbone')
	`
	groupRows, err := database.DB.Query(ctx, groupQuery, reviewerID)
	if err != nil {
		return nil, 0, fmt.Errorf("查询审核员教研组失败: %w", err)
	}
	defer groupRows.Close()

	var groupIDs []string
	for groupRows.Next() {
		var gid string
		if err := groupRows.Scan(&gid); err == nil {
			groupIDs = append(groupIDs, gid)
		}
	}
	if len(groupIDs) == 0 {
		return []*models.PendingReviewItem{}, 0, nil
	}

	// 构建 IN 子句
	inClause, args := buildInClause(groupIDs, 1)

	// 统计总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM lesson_plans
		WHERE status = 'submitted' AND review_level = 0 AND group_id IN (%s)
	`, inClause)
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计L1待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	nextIdx := len(args) + 1
	listQuery := fmt.Sprintf(`
		SELECT lp.id, lp.title, lp.subject, lp.grade, lp.author_id,
		       COALESCE(u.display_name, '') AS author_name,
		       lp.group_id,
		       COALESCE(tg.name, '') AS group_name,
		       COALESCE(o.name, '') AS school_name,
		       lp.review_level, lp.ai_review_score, lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u ON u.id = lp.author_id
		LEFT JOIN teaching_groups tg ON tg.id = lp.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE lp.status = 'submitted' AND lp.review_level = 0 AND lp.group_id IN (%s)
		ORDER BY lp.updated_at ASC
		LIMIT $%d OFFSET $%d
	`, inClause, nextIdx, nextIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询L1待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PendingReviewItem
	for rows.Next() {
		item := &models.PendingReviewItem{}
		err := rows.Scan(
			&item.LessonPlanID, &item.Title, &item.Subject, &item.Grade,
			&item.AuthorID, &item.AuthorName, &item.GroupID, &item.GroupName,
			&item.SchoolName, &item.ReviewLevel, &item.AIReviewScore, &item.SubmittedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描L1待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}
	return items, total, nil
}

// ListPendingReviewsL1All 获取全部L1待审核列表（admin场景，不限教研组）
// 查询条件：status='submitted' AND review_level=0 AND group_id IS NOT NULL
func ListPendingReviewsL1All(ctx context.Context, limit int, offset int) ([]*models.PendingReviewItem, int, error) {
	var total int
	if err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM lesson_plans
		 WHERE status = 'submitted' AND review_level = 0 AND group_id IS NOT NULL`,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计全量L1待审核数失败: %w", err)
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
		LEFT JOIN teaching_groups tg ON tg.id = lp.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE lp.status = 'submitted' AND lp.review_level = 0 AND lp.group_id IS NOT NULL
		ORDER BY lp.updated_at ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询全量L1待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PendingReviewItem
	for rows.Next() {
		item := &models.PendingReviewItem{}
		err := rows.Scan(
			&item.LessonPlanID, &item.Title, &item.Subject, &item.Grade,
			&item.AuthorID, &item.AuthorName, &item.GroupID, &item.GroupName,
			&item.SchoolName, &item.ReviewLevel, &item.AIReviewScore, &item.SubmittedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描全量L1待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL1
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL1]
		items = append(items, item)
	}
	return items, total, nil
}

// ListPendingReviewsL2 获取L2待审核列表
// 当 schoolID 非空时只查该学校的；为空时查全部（admin场景）
func ListPendingReviewsL2(ctx context.Context, schoolID string, limit int, offset int) ([]*models.PendingReviewItem, int, error) {
	var countQuery string
	var listQuery string
	var countArgs []interface{}
	var listArgs []interface{}

	if schoolID != "" {
		countQuery = `
			SELECT COUNT(*) FROM lesson_plans
			WHERE status = 'submitted' AND review_level = 1 AND review_school_id = $1
		`
		countArgs = []interface{}{schoolID}
	} else {
		countQuery = `
			SELECT COUNT(*) FROM lesson_plans
			WHERE status = 'submitted' AND review_level = 1
		`
		countArgs = []interface{}{}
	}

	var total int
	if err := database.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计L2待审核数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	if schoolID != "" {
		listQuery = `
			SELECT lp.id, lp.title, lp.subject, lp.grade, lp.author_id,
			       COALESCE(u.display_name, '') AS author_name,
			       lp.group_id,
			       COALESCE(tg.name, '') AS group_name,
			       COALESCE(o.name, '') AS school_name,
			       lp.review_level, lp.ai_review_score, lp.updated_at
			FROM lesson_plans lp
			LEFT JOIN users u ON u.id = lp.author_id
			LEFT JOIN teaching_groups tg ON tg.id = lp.group_id
			LEFT JOIN organizations o ON o.id = lp.review_school_id
			WHERE lp.status = 'submitted' AND lp.review_level = 1 AND lp.review_school_id = $1
			ORDER BY lp.updated_at ASC
			LIMIT $2 OFFSET $3
		`
		listArgs = []interface{}{schoolID, limit, offset}
	} else {
		listQuery = `
			SELECT lp.id, lp.title, lp.subject, lp.grade, lp.author_id,
			       COALESCE(u.display_name, '') AS author_name,
			       lp.group_id,
			       COALESCE(tg.name, '') AS group_name,
			       COALESCE(o.name, '') AS school_name,
			       lp.review_level, lp.ai_review_score, lp.updated_at
			FROM lesson_plans lp
			LEFT JOIN users u ON u.id = lp.author_id
			LEFT JOIN teaching_groups tg ON tg.id = lp.group_id
			LEFT JOIN organizations o ON o.id = lp.review_school_id
			WHERE lp.status = 'submitted' AND lp.review_level = 1
			ORDER BY lp.updated_at ASC
			LIMIT $1 OFFSET $2
		`
		listArgs = []interface{}{limit, offset}
	}

	rows, err := database.DB.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询L2待审核列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.PendingReviewItem
	for rows.Next() {
		item := &models.PendingReviewItem{}
		err := rows.Scan(
			&item.LessonPlanID, &item.Title, &item.Subject, &item.Grade,
			&item.AuthorID, &item.AuthorName, &item.GroupID, &item.GroupName,
			&item.SchoolName, &item.ReviewLevel, &item.AIReviewScore, &item.SubmittedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描L2待审核行失败: %w", err)
		}
		item.ReviewLevel = models.ReviewLevelL2
		item.LevelName = models.ReviewLevelNameMap[models.ReviewLevelL2]
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 已审核记录查询（v127.2新增） ====================

// ListReviewedRecords 查询已审核记录列表
// 支持按级别、审核员、决策类型过滤
// isAdmin=true 时不限 reviewerID（查所有人的）
func ListReviewedRecords(ctx context.Context, reviewerID string, level int, decision string, isAdmin bool, limit int, offset int) ([]*models.ReviewedListItem, int, error) {
	// 构建动态WHERE条件
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

	// 统计总数
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM lesson_plan_reviews_v2 r WHERE %s`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计已审核记录数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	// 分页查询（关联教案和用户表获取标题、作者、审核员名称）
	listQuery := fmt.Sprintf(`
		SELECT r.id, r.lesson_plan_id,
		       COALESCE(lp.title, '') AS plan_title,
		       COALESCE(lp.subject, '') AS plan_subject,
		       COALESCE(lp.grade, '') AS plan_grade,
		       COALESCE(author.display_name, '') AS author_name,
		       r.review_level,
		       COALESCE(reviewer.display_name, '') AS reviewer_name,
		       r.decision, r.score, r.comment, r.created_at
		FROM lesson_plan_reviews_v2 r
		LEFT JOIN lesson_plans lp ON lp.id = r.lesson_plan_id
		LEFT JOIN users author ON author.id = lp.author_id
		LEFT JOIN users reviewer ON reviewer.id = r.reviewer_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询已审核记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ReviewedListItem
	for rows.Next() {
		item := &models.ReviewedListItem{}
		err := rows.Scan(
			&item.ID, &item.LessonPlanID, &item.PlanTitle, &item.PlanSubject,
			&item.PlanGrade, &item.AuthorName, &item.ReviewLevel,
			&item.ReviewerName, &item.Decision, &item.Score, &item.Comment, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描已审核记录行失败: %w", err)
		}
		item.LevelName = models.ReviewLevelNameMap[item.ReviewLevel]
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 教案审核级别更新 ====================

// UpdateLessonPlanReviewLevel 更新教案的审核级别和关联学校
func UpdateLessonPlanReviewLevel(ctx context.Context, planID string, level int, schoolID *string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET review_level = $1, review_school_id = $2, updated_at = $3
		WHERE id = $4
	`, level, schoolID, now, planID)
	if err != nil {
		return fmt.Errorf("更新教案审核级别失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ==================== 审核流程配置 CRUD ====================

// GetReviewFlowConfig 获取学校的审核流程配置
func GetReviewFlowConfig(ctx context.Context, schoolID string) (*models.ReviewFlowConfig, error) {
	cfg := &models.ReviewFlowConfig{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, school_id, l2_enabled, l3_sample_rate, auto_publish_on_approved,
		       updated_by, updated_at
		FROM review_flow_configs WHERE school_id = $1
	`, schoolID).Scan(
		&cfg.ID, &cfg.SchoolID, &cfg.L2Enabled, &cfg.L3SampleRate,
		&cfg.AutoPublishOnApproved, &cfg.UpdatedBy, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewConfigNotFound
		}
		return nil, fmt.Errorf("查询审核配置失败: %w", err)
	}
	return cfg, nil
}

// UpsertReviewFlowConfig 创建或更新学校的审核流程配置
func UpsertReviewFlowConfig(ctx context.Context, schoolID string, req *models.UpdateReviewFlowConfigRequest, updatedBy string) error {
	now := time.Now()
	_, err := database.DB.Exec(ctx, `
		INSERT INTO review_flow_configs
			(school_id, l2_enabled, l3_sample_rate, auto_publish_on_approved, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (school_id) DO UPDATE SET
			l2_enabled = EXCLUDED.l2_enabled,
			l3_sample_rate = EXCLUDED.l3_sample_rate,
			auto_publish_on_approved = EXCLUDED.auto_publish_on_approved,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
	`, schoolID, req.L2Enabled, req.L3SampleRate, req.AutoPublishOnApproved, updatedBy, now)
	if err != nil {
		return fmt.Errorf("更新审核配置失败: %w", err)
	}
	return nil
}

// ==================== 审核统计 ====================

// GetReviewStats 获取审核统计
//
// v127.2 修复：isAdmin=true 时"已审核/已通过/已退回"不限 reviewer_id
func GetReviewStats(ctx context.Context, reviewerID string, level int, isAdmin bool, groupIDs []string) (*models.ReviewStatsResponse, error) {
	stats := &models.ReviewStatsResponse{}

	// 待审核数：按级别查
	// v127.3: 非admin的L1统计限定到自己的教研组，避免统计数字和列表不一致
	if level == models.ReviewLevelL1 && !isAdmin && len(groupIDs) > 0 {
		// 非admin：只统计自己教研组内的L1待审核
		inClause, args := buildInClause(groupIDs, 1)
		q := fmt.Sprintf(`SELECT COUNT(*) FROM lesson_plans
			WHERE status = 'submitted' AND review_level = 0 AND group_id IN (%s)`, inClause)
		_ = database.DB.QueryRow(ctx, q, args...).Scan(&stats.TotalPending)
	} else if level == models.ReviewLevelL1 {
		// admin：全局L1待审核
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plans
			 WHERE status = 'submitted' AND review_level = 0 AND group_id IS NOT NULL`,
		).Scan(&stats.TotalPending)
	} else if level == models.ReviewLevelL2 {
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plans
			 WHERE status = 'submitted' AND review_level = 1`,
		).Scan(&stats.TotalPending)
	}

	if isAdmin {
		// admin看全局统计
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2 WHERE review_level = $1`,
			level,
		).Scan(&stats.TotalReviewed)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2 WHERE review_level = $1 AND decision = 'approved'`,
			level,
		).Scan(&stats.TotalApproved)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2 WHERE review_level = $1 AND decision = 'revision'`,
			level,
		).Scan(&stats.TotalRevision)
	} else {
		// 普通审核员看个人统计
		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2
			 WHERE reviewer_id = $1 AND review_level = $2`,
			reviewerID, level,
		).Scan(&stats.TotalReviewed)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2
			 WHERE reviewer_id = $1 AND review_level = $2 AND decision = 'approved'`,
			reviewerID, level,
		).Scan(&stats.TotalApproved)

		_ = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM lesson_plan_reviews_v2
			 WHERE reviewer_id = $1 AND review_level = $2 AND decision = 'revision'`,
			reviewerID, level,
		).Scan(&stats.TotalRevision)
	}

	return stats, nil
}

// ==================== 辅助函数 ====================

// GetUserLeadOrBackboneGroupIDs 获取用户作为lead/backbone的教研组ID列表
func GetUserLeadOrBackboneGroupIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT group_id FROM teaching_group_members
		 WHERE user_id = $1 AND role IN ('lead', 'backbone')`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户教研组ID失败: %w", err)
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
