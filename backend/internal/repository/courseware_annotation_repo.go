package repository

// courseware_annotation_repo.go — 课件页级批注数据访问层
//
// 页面关联规则：
//   - 创建时通过courseware_id + 当前page_number确定稳定page_id；
//   - page_number_snapshot保存批注创建时页码；
//   - 查询时通过page_id解析页面当前页码；
//   - 页面已删除时，当前页码回退为page_number_snapshot。
//
// 安全边界：
//   - 创建绑定和批注写入在同一条SQL中完成；
//   - 读取使用课件和批注复合条件；
//   - 状态更新和删除使用updated_at乐观锁。

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

// cwAnnotationSelectColumns 返回稳定页面ID、当前页码和历史页码快照。
//
// 页面仍存在时使用courseware_pages.page_number作为当前页码；
// 页面已删除并导致page_id被清空时，回退到创建时页码快照。
const cwAnnotationSelectColumns = `
	annotation.id,
	annotation.courseware_id,
	annotation.page_id,
	COALESCE(
		page.page_number,
		annotation.page_number_snapshot
	) AS page_number,
	annotation.page_number_snapshot,
	annotation.reviewer_id,
	annotation.reviewer_name,
	annotation.content,
	annotation.status,
	annotation.created_at,
	annotation.updated_at`

const cwAnnotationPageJoin = `
	LEFT JOIN courseware_pages AS page
		ON page.id = annotation.page_id
		AND page.courseware_id =
			annotation.courseware_id`

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
		&annotation.PageID,
		&annotation.PageNumber,
		&annotation.PageNumberSnapshot,
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
// INSERT ... SELECT使页面存在性检查、稳定page_id解析、
// 历史页码快照保存和批注写入在同一条SQL中完成，避免并发漂移。
func CreateCWAnnotation(
	ctx context.Context,
	annotation *models.CoursewareAnnotation,
) error {
	query := `
		INSERT INTO courseware_annotations (
			courseware_id,
			page_number,
			page_id,
			page_number_snapshot,
			reviewer_id,
			reviewer_name,
			content,
			status
		)
		SELECT
			page.courseware_id,
			page.page_number,
			page.id,
			page.page_number,
			$3,
			$4,
			$5,
			'pending'
		FROM courseware_pages AS page
		WHERE page.courseware_id = $1
			AND page.page_number = $2
		RETURNING
			id,
			page_id,
			page_number,
			page_number_snapshot,
			created_at,
			updated_at`

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
		&annotation.PageID,
		&annotation.PageNumber,
		&annotation.PageNumberSnapshot,
		&annotation.CreatedAt,
		&annotation.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCWAnnotationPageNotFound
	}

	return err
}

// ListCWAnnotationsByCoursewareID 查询课件全部批注。
//
// 按页面当前页码排序；页面已删除时按创建时页码快照排序。
func ListCWAnnotationsByCoursewareID(
	ctx context.Context,
	coursewareID string,
) (
	[]*models.CoursewareAnnotation,
	error,
) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations AS annotation` +
		cwAnnotationPageJoin + `
		WHERE annotation.courseware_id = $1
		ORDER BY
			COALESCE(
				page.page_number,
				annotation.page_number_snapshot
			) ASC,
			annotation.created_at ASC`

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

// GetCWAnnotationByID 按批注ID读取正式批注。
func GetCWAnnotationByID(
	ctx context.Context,
	annotationID string,
) (
	*models.CoursewareAnnotation,
	error,
) {
	query := `
		SELECT` + cwAnnotationSelectColumns + `
		FROM courseware_annotations AS annotation` +
		cwAnnotationPageJoin + `
		WHERE annotation.id = $1`

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
		FROM courseware_annotations AS annotation` +
		cwAnnotationPageJoin + `
		WHERE annotation.courseware_id = $1
			AND annotation.id = $2`

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
