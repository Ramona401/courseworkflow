package repository

// courseware_annotation_repo.go — 课件页级批注数据访问层(阶段2)
//
// 镜像 annotation_repo.go,挂载点为 page_number(非段落号),不含 review_round。
// 提供:创建 / 按课件列出 / 按ID查 / 更新状态 / 删除 / 按课件级联删。
// 表上已配 ON DELETE CASCADE(随课件删),DeleteCWAnnotationsByCoursewareID 仅供显式批量清理用。

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCWAnnotationNotFound 课件批注不存在错误
var ErrCWAnnotationNotFound = errors.New("批注不存在")

// cwAnnotationSelectColumns 统一列清单,避免各查询列顺序漂移
const cwAnnotationSelectColumns = `
	id, courseware_id, page_number, reviewer_id, reviewer_name,
	content, status, created_at, updated_at`

// scanCWAnnotation 统一扫描一行批注(Scan 顺序须与 cwAnnotationSelectColumns 一致)
func scanCWAnnotation(row pgx.Row) (*models.CoursewareAnnotation, error) {
	a := &models.CoursewareAnnotation{}
	err := row.Scan(
		&a.ID, &a.CoursewareID, &a.PageNumber, &a.ReviewerID, &a.ReviewerName,
		&a.Content, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ==================== 创建批注 ====================

// CreateCWAnnotation 创建课件页级批注
// status 固定初始为 pending,page_number 由调用方校验(须为该课件已存在页号)
func CreateCWAnnotation(ctx context.Context, a *models.CoursewareAnnotation) error {
	query := `
		INSERT INTO courseware_annotations
			(courseware_id, page_number, reviewer_id, reviewer_name, content, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, query,
		a.CoursewareID,
		a.PageNumber,
		a.ReviewerID,
		a.ReviewerName,
		a.Content,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// ==================== 查询批注 ====================

// ListCWAnnotationsByCoursewareID 查询课件全部批注,按页号→时间排序
// 前端收到后可按 page_number 分组,在胶片条对应页挂气泡
func ListCWAnnotationsByCoursewareID(ctx context.Context, coursewareID string) ([]*models.CoursewareAnnotation, error) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations
		WHERE courseware_id = $1
		ORDER BY page_number ASC, created_at ASC`
	rows, err := database.DB.Query(ctx, query, coursewareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.CoursewareAnnotation
	for rows.Next() {
		a, err := scanCWAnnotation(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetCWAnnotationByID 按ID查询单条批注(供删除/标记前的归属与存在性校验)
func GetCWAnnotationByID(ctx context.Context, id string) (*models.CoursewareAnnotation, error) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations
		WHERE id = $1`
	a, err := scanCWAnnotation(database.DB.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCWAnnotationNotFound
		}
		return nil, err
	}
	return a, nil
}

// ==================== 更新批注 ====================

// UpdateCWAnnotationStatus 更新批注处理状态(pending/resolved/archived)
func UpdateCWAnnotationStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE courseware_annotations
		SET status = $1, updated_at = $2
		WHERE id = $3`
	tag, err := database.DB.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCWAnnotationNotFound
	}
	return nil
}

// ==================== 删除批注 ====================

// DeleteCWAnnotation 删除单条批注(归属校验在 service 层做)
func DeleteCWAnnotation(ctx context.Context, id string) error {
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM courseware_annotations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCWAnnotationNotFound
	}
	return nil
}

// DeleteCWAnnotationsByCoursewareID 删除某课件全部批注
// 表上已 ON DELETE CASCADE,本函数仅供显式批量清理用(当前无调用方,留作完整对称)
func DeleteCWAnnotationsByCoursewareID(ctx context.Context, coursewareID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM courseware_annotations WHERE courseware_id = $1`, coursewareID)
	return err
}
