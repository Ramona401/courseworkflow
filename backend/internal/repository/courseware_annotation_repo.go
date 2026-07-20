package repository

// courseware_annotation_repo.go — 课件页级批注数据访问层
//
// 安全边界：
//   - 创建时在同一SQL中确认courseware_id+page_number真实存在；
//   - 读取可使用courseware_id+annotation_id复合条件；
//   - 状态更新和删除使用courseware_id+annotation_id+updated_at乐观锁。

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrCWAnnotationNotFound 课件批注不存在。
	ErrCWAnnotationNotFound = errors.New(
		"批注不存在",
	)

	// ErrCWAnnotationPageNotFound 目标课件页面不存在。
	ErrCWAnnotationPageNotFound = errors.New(
		"课件页面不存在",
	)
)

const cwAnnotationSelectColumns = `
	id, courseware_id, page_number, reviewer_id, reviewer_name,
	content, status, created_at, updated_at`

func scanCWAnnotation(
	row pgx.Row,
) (
	*models.CoursewareAnnotation,
	error,
) {
	annotation := &models.CoursewareAnnotation{}

	err := row.Scan(
		&annotation.ID,
		&annotation.CoursewareID,
		&annotation.PageNumber,
		&annotation.ReviewerID,
		&annotation.ReviewerName,
		&annotation.Content,
		&annotation.Status,
		&annotation.CreatedAt,
		&annotation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

// CreateCWAnnotation 创建课件页级批注。
//
// INSERT ... SELECT ... WHERE EXISTS使页面存在性检查与批注写入处于同一条SQL中，
// 防止Service校验页面后、正式写入前页面被并发删除。
func CreateCWAnnotation(
	ctx context.Context,
	annotation *models.CoursewareAnnotation,
) error {
	query := `
		INSERT INTO courseware_annotations (
			courseware_id,
			page_number,
			reviewer_id,
			reviewer_name,
			content,
			status
		)
		SELECT
			$1,
			$2,
			$3,
			$4,
			$5,
			'pending'
		WHERE EXISTS (
			SELECT 1
			FROM courseware_pages
			WHERE courseware_id = $1
				AND page_number = $2
		)
		RETURNING id, created_at, updated_at`

	err := database.DB.QueryRow(
		ctx,
		query,
		annotation.CoursewareID,
		annotation.PageNumber,
		annotation.ReviewerID,
		annotation.ReviewerName,
		annotation.Content,
	).Scan(
		&annotation.ID,
		&annotation.CreatedAt,
		&annotation.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCWAnnotationPageNotFound
	}

	return err
}

// ListCWAnnotationsByCoursewareID 查询课件全部批注。
func ListCWAnnotationsByCoursewareID(
	ctx context.Context,
	coursewareID string,
) (
	[]*models.CoursewareAnnotation,
	error,
) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations
		WHERE courseware_id = $1
		ORDER BY page_number ASC, created_at ASC`

	rows, err := database.DB.Query(
		ctx,
		query,
		coursewareID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	annotations := make(
		[]*models.CoursewareAnnotation,
		0,
	)

	for rows.Next() {
		annotation, err :=
			scanCWAnnotation(rows)
		if err != nil {
			return nil, err
		}

		annotations = append(
			annotations,
			annotation,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return annotations, nil
}

// GetCWAnnotationByID 按ID读取正式批注。
func GetCWAnnotationByID(
	ctx context.Context,
	annotationID string,
) (
	*models.CoursewareAnnotation,
	error,
) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations
		WHERE id = $1`

	annotation, err :=
		scanCWAnnotation(
			database.DB.QueryRow(
				ctx,
				query,
				annotationID,
			),
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCWAnnotationNotFound
	}

	return annotation, err
}

// GetCWAnnotationForCourseware 按课件ID和批注ID复合读取。
func GetCWAnnotationForCourseware(
	ctx context.Context,
	coursewareID string,
	annotationID string,
) (
	*models.CoursewareAnnotation,
	error,
) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations
		WHERE courseware_id = $1
			AND id = $2`

	annotation, err :=
		scanCWAnnotation(
			database.DB.QueryRow(
				ctx,
				query,
				coursewareID,
				annotationID,
			),
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCWAnnotationNotFound
	}

	return annotation, err
}

// UpdateCWAnnotationStatusIfUnchanged 使用复合路径和updated_at乐观锁更新状态。
func UpdateCWAnnotationStatusIfUnchanged(
	ctx context.Context,
	coursewareID string,
	annotationID string,
	expectedUpdatedAt time.Time,
	status string,
) (
	bool,
	error,
) {
	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_annotations
		SET status = $1,
			updated_at = $2
		WHERE courseware_id = $3
			AND id = $4
			AND updated_at = $5`,
		status,
		time.Now(),
		coursewareID,
		annotationID,
		expectedUpdatedAt,
	)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() == 1,
		nil
}

// DeleteCWAnnotationIfUnchanged 使用复合路径和updated_at乐观锁删除批注。
func DeleteCWAnnotationIfUnchanged(
	ctx context.Context,
	coursewareID string,
	annotationID string,
	expectedUpdatedAt time.Time,
) (
	bool,
	error,
) {
	tag, err := database.DB.Exec(
		ctx,
		`DELETE FROM courseware_annotations
		WHERE courseware_id = $1
			AND id = $2
			AND updated_at = $3`,
		coursewareID,
		annotationID,
		expectedUpdatedAt,
	)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() == 1,
		nil
}

// DeleteCWAnnotationsByCoursewareID 显式删除某课件全部批注。
func DeleteCWAnnotationsByCoursewareID(
	ctx context.Context,
	coursewareID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`DELETE FROM courseware_annotations
		WHERE courseware_id = $1`,
		coursewareID,
	)

	return err
}
