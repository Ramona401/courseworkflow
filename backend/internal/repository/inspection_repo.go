package repository

// inspection_repo.go — 抽查记录+区域教研员分配数据访问层
//
// v127 新增（多级审核体系 · 区域抽查）：
//   - 抽查记录 CRUD（inspection_records 表）
//   - 区域教研员管辖分配 CRUD（district_inspector_assignments 表）
//   - 抽样查询（按学校+比例随机抽取）
//   - 抽查统计
//
// 对应数据库表：
//   inspection_records              — 抽查记录
//   district_inspector_assignments  — 区域教研员管辖分配

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
	ErrInspectionNotFound  = errors.New("抽查记录不存在")
	ErrInspectorAssignExists = errors.New("该教研员已分配到此区域")
)

// ==================== 抽查记录 CRUD ====================

// CreateInspectionRecord 创建抽查记录
func CreateInspectionRecord(ctx context.Context, record *models.InspectionRecord) error {
	query := `
		INSERT INTO inspection_records
			(lesson_plan_id, inspector_id, sample_batch, status, priority, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	status := record.Status
	if status == "" {
		status = models.InspectionStatusPending
	}
	err := database.DB.QueryRow(ctx, query,
		record.LessonPlanID, record.InspectorID, record.SampleBatch,
		status, record.Priority, record.Comment,
	).Scan(&record.ID, &record.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建抽查记录失败: %w", err)
	}
	return nil
}

// GetInspectionByID 根据ID查询抽查记录
func GetInspectionByID(ctx context.Context, id string) (*models.InspectionRecord, error) {
	record := &models.InspectionRecord{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, lesson_plan_id, inspector_id, sample_batch, status,
		       priority, comment, inspected_at, created_at
		FROM inspection_records WHERE id = $1
	`, id).Scan(
		&record.ID, &record.LessonPlanID, &record.InspectorID,
		&record.SampleBatch, &record.Status, &record.Priority,
		&record.Comment, &record.InspectedAt, &record.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInspectionNotFound
		}
		return nil, fmt.Errorf("查询抽查记录失败: %w", err)
	}
	return record, nil
}

// UpdateInspectionStatus 更新抽查记录状态
func UpdateInspectionStatus(ctx context.Context, id string, status string, comment string) error {
	now := time.Now()
	var inspectedAt *time.Time
	// 终态（passed/revoked）时记录完成时间
	if status == models.InspectionStatusPassed || status == models.InspectionStatusRevoked {
		inspectedAt = &now
	}
	result, err := database.DB.Exec(ctx, `
		UPDATE inspection_records
		SET status = $1, comment = $2, inspected_at = $3
		WHERE id = $4
	`, status, comment, inspectedAt, id)
	if err != nil {
		return fmt.Errorf("更新抽查状态失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

// AssignInspector 分配审查员到抽查记录
func AssignInspector(ctx context.Context, id string, inspectorID string) error {
	result, err := database.DB.Exec(ctx, `
		UPDATE inspection_records
		SET inspector_id = $1, status = $2
		WHERE id = $3
	`, inspectorID, models.InspectionStatusAssigned, id)
	if err != nil {
		return fmt.Errorf("分配审查员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

// ListInspections 获取抽查列表（支持状态筛选+分页）
func ListInspections(ctx context.Context, inspectorID string, status string, limit int, offset int) ([]*models.InspectionListItem, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if inspectorID != "" {
		where += fmt.Sprintf(" AND ir.inspector_id = $%d", argIdx)
		args = append(args, inspectorID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND ir.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// 统计总数
	var total int
	countQuery := "SELECT COUNT(*) FROM inspection_records ir" + where
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计抽查记录数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	listQuery := fmt.Sprintf(`
		SELECT ir.id, ir.lesson_plan_id,
		       COALESCE(lp.title, '') AS plan_title,
		       COALESCE(lp.subject, '') AS plan_subject,
		       COALESCE(lp.grade, '') AS plan_grade,
		       COALESCE(au.display_name, '') AS author_name,
		       COALESCE(o.name, '') AS school_name,
		       ir.inspector_id,
		       COALESCE(iu.display_name, '') AS inspector_name,
		       ir.sample_batch, ir.status, ir.priority,
		       ir.comment, ir.inspected_at, ir.created_at
		FROM inspection_records ir
		LEFT JOIN lesson_plans lp ON lp.id = ir.lesson_plan_id
		LEFT JOIN users au ON au.id = lp.author_id
		LEFT JOIN organizations o ON o.id = lp.review_school_id
		LEFT JOIN users iu ON iu.id = ir.inspector_id
		%s
		ORDER BY ir.priority DESC, ir.created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询抽查列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.InspectionListItem
	for rows.Next() {
		item := &models.InspectionListItem{}
		err := rows.Scan(
			&item.ID, &item.LessonPlanID, &item.PlanTitle,
			&item.PlanSubject, &item.PlanGrade, &item.AuthorName,
			&item.SchoolName, &item.InspectorID, &item.InspectorName,
			&item.SampleBatch, &item.Status, &item.Priority,
			&item.Comment, &item.InspectedAt, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描抽查记录行失败: %w", err)
		}
		item.StatusName = models.InspectionStatusNameMap[item.Status]
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 抽样查询 ====================

// SamplePublishedPlansForInspection 从已发布共享的教案中按比例随机抽样
// 排除已有抽查记录的教案，防止重复抽查
func SamplePublishedPlansForInspection(ctx context.Context, schoolID string, sampleRate float64, batchID string) (int, error) {
	if sampleRate <= 0 || sampleRate > 1.0 {
		sampleRate = 0.20
	}

	// 抽样并直接插入 inspection_records
	// 用 TABLESAMPLE 或 random() 实现随机抽样
	query := `
		INSERT INTO inspection_records (lesson_plan_id, sample_batch, status, priority)
		SELECT lp.id, $1, 'pending', 0
		FROM lesson_plans lp
		WHERE lp.status = 'published_shared'
		  AND lp.review_school_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM inspection_records ir
		    WHERE ir.lesson_plan_id = lp.id
		      AND ir.status NOT IN ('passed', 'revoked')
		  )
		ORDER BY random()
		LIMIT (
		  SELECT GREATEST(1, CEIL(COUNT(*) * $3))
		  FROM lesson_plans lp2
		  WHERE lp2.status = 'published_shared'
		    AND lp2.review_school_id = $2
		    AND NOT EXISTS (
		      SELECT 1 FROM inspection_records ir2
		      WHERE ir2.lesson_plan_id = lp2.id
		        AND ir2.status NOT IN ('passed', 'revoked')
		    )
		)
	`
	result, err := database.DB.Exec(ctx, query, batchID, schoolID, sampleRate)
	if err != nil {
		return 0, fmt.Errorf("抽样失败: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// ==================== 抽查统计 ====================

// GetInspectionStats 获取抽查统计
func GetInspectionStats(ctx context.Context, inspectorID string) (*models.InspectionStatsResponse, error) {
	stats := &models.InspectionStatsResponse{}

	where := ""
	args := []interface{}{}
	if inspectorID != "" {
		where = " WHERE inspector_id = $1"
		args = append(args, inspectorID)
	}

	_ = database.DB.QueryRow(ctx,
		"SELECT COUNT(*) FROM inspection_records"+where, args...,
	).Scan(&stats.TotalSampled)

	if inspectorID != "" {
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE inspector_id = $1 AND status IN ('pending', 'assigned', 'in_review')",
			inspectorID,
		).Scan(&stats.TotalPending)
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE inspector_id = $1 AND status = 'passed'",
			inspectorID,
		).Scan(&stats.TotalPassed)
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE inspector_id = $1 AND status = 'revoked'",
			inspectorID,
		).Scan(&stats.TotalRevoked)
	} else {
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE status IN ('pending', 'assigned', 'in_review')",
		).Scan(&stats.TotalPending)
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE status = 'passed'",
		).Scan(&stats.TotalPassed)
		_ = database.DB.QueryRow(ctx,
			"SELECT COUNT(*) FROM inspection_records WHERE status = 'revoked'",
		).Scan(&stats.TotalRevoked)
	}

	if stats.TotalSampled > 0 {
		stats.PassRate = float64(stats.TotalPassed) / float64(stats.TotalSampled)
	}
	return stats, nil
}

// ==================== 区域教研员管辖分配 ====================

// CreateDistrictInspectorAssignment 分配教研员到区域
func CreateDistrictInspectorAssignment(ctx context.Context, assign *models.DistrictInspectorAssignment) error {
	err := database.DB.QueryRow(ctx, `
		INSERT INTO district_inspector_assignments (inspector_id, region_id)
		VALUES ($1, $2)
		RETURNING id, created_at
	`, assign.InspectorID, assign.RegionID).Scan(&assign.ID, &assign.CreatedAt)
	if err != nil {
		return fmt.Errorf("分配区域教研员失败: %w", err)
	}
	return nil
}

// DeleteDistrictInspectorAssignment 取消区域教研员分配
func DeleteDistrictInspectorAssignment(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM district_inspector_assignments WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("取消区域教研员分配失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("分配记录不存在")
	}
	return nil
}

// ListDistrictInspectors 获取区域教研员列表
func ListDistrictInspectors(ctx context.Context, regionID string) ([]*models.DistrictInspectorListItem, error) {
	where := ""
	args := []interface{}{}
	if regionID != "" {
		where = " WHERE dia.region_id = $1"
		args = append(args, regionID)
	}

	query := fmt.Sprintf(`
		SELECT dia.id, dia.inspector_id,
		       COALESCE(u.display_name, '') AS inspector_name,
		       dia.region_id,
		       COALESCE(o.name, '') AS region_name,
		       dia.created_at
		FROM district_inspector_assignments dia
		LEFT JOIN users u ON u.id = dia.inspector_id
		LEFT JOIN organizations o ON o.id = dia.region_id
		%s
		ORDER BY o.name ASC, u.display_name ASC
	`, where)

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询区域教研员列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.DistrictInspectorListItem
	for rows.Next() {
		item := &models.DistrictInspectorListItem{}
		err := rows.Scan(
			&item.ID, &item.InspectorID, &item.InspectorName,
			&item.RegionID, &item.RegionName, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描区域教研员行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// GetInspectorRegionIDs 获取某教研员管辖的所有区域ID
func GetInspectorRegionIDs(ctx context.Context, inspectorID string) ([]string, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT region_id FROM district_inspector_assignments WHERE inspector_id = $1`,
		inspectorID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询教研员管辖区域失败: %w", err)
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

// QueryDistinctReviewSchoolIDs 查询所有有已发布教案的学校ID（去重）
func QueryDistinctReviewSchoolIDs(ctx context.Context) ([]string, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT DISTINCT review_school_id FROM lesson_plans
		WHERE status = 'published_shared' AND review_school_id IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("查询学校ID列表失败: %w", err)
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
