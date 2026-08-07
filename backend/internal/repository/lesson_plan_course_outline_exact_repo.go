package repository

// lesson_plan_course_outline_exact_repo.go — 教案精确课程大纲快照读写
//
// 数据库触发器负责：
//   - 校验大纲active、教育域、学科和具体年级；
//   - 固化出版社、册次和学制快照；
//   - 解除精确挂载时清空精确快照。
//
// 本文件只提供最小读写入口，不进行模糊匹配。

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

func scanLessonPlanCourseOutlineSnapshot(
	row pgx.Row,
) (*models.LessonPlanCourseOutlineSnapshot, error) {
	var (
		id        sql.NullString
		publisher sql.NullString
		volume    sql.NullString
		system    sql.NullString
	)

	err := row.Scan(
		&id,
		&publisher,
		&volume,
		&system,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}

		return nil, fmt.Errorf(
			"扫描教案精确课程大纲快照失败: %w",
			err,
		)
	}

	snapshot :=
		&models.LessonPlanCourseOutlineSnapshot{}

	if id.Valid &&
		strings.TrimSpace(id.String) != "" {
		value := strings.TrimSpace(id.String)
		snapshot.CourseOutlineID = &value
	}

	if publisher.Valid {
		value := strings.TrimSpace(
			publisher.String,
		)
		snapshot.CourseOutlinePublisher = &value
	}

	if volume.Valid &&
		strings.TrimSpace(volume.String) != "" {
		value := strings.TrimSpace(volume.String)
		snapshot.CourseOutlineVolume = &value
	}

	if system.Valid &&
		strings.TrimSpace(system.String) != "" {
		value := strings.TrimSpace(system.String)
		snapshot.SchoolSystem = &value
	}

	return snapshot, nil
}

// GetLessonPlanCourseOutlineSnapshot
// 读取教案精确大纲ID及数据库固化快照。
func GetLessonPlanCourseOutlineSnapshot(
	ctx context.Context,
	lessonPlanID string,
) (*models.LessonPlanCourseOutlineSnapshot, error) {
	return scanLessonPlanCourseOutlineSnapshot(
		database.DB.QueryRow(
			ctx,
			`
                                SELECT
                                    course_outline_id::text,
                                    course_outline_publisher,
                                    course_outline_volume,
                                    school_system
                                FROM lesson_plans
                                WHERE id = $1
                                  AND deleted_at IS NULL
                        `,
			strings.TrimSpace(
				lessonPlanID,
			),
		),
	)
}

// UpdateLessonPlanCourseOutlineID
// 设置或解除教案精确课程大纲ID，并返回数据库最终快照。
//
// outlineID为nil或空字符串时解除挂载；
// 对legacy publisher-only教案解除时，SQL显式清空旧publisher字段。
func UpdateLessonPlanCourseOutlineID(
	ctx context.Context,
	lessonPlanID string,
	outlineID *string,
) (*models.LessonPlanCourseOutlineSnapshot, error) {
	var normalizedID interface{}

	if outlineID != nil &&
		strings.TrimSpace(*outlineID) != "" {
		normalizedID = strings.TrimSpace(
			*outlineID,
		)
	}

	return scanLessonPlanCourseOutlineSnapshot(
		database.DB.QueryRow(
			ctx,
			`
                                UPDATE lesson_plans
                                SET
                                    course_outline_id =
                                        $1::uuid,
                                    course_outline_publisher =
                                        CASE
                                            WHEN $1::uuid IS NULL
                                            THEN NULL
                                            ELSE course_outline_publisher
                                        END,
                                    updated_at = NOW()
                                WHERE id = $2
                                  AND deleted_at IS NULL
                                RETURNING
                                    course_outline_id::text,
                                    course_outline_publisher,
                                    course_outline_volume,
                                    school_system
                        `,
			normalizedID,
			strings.TrimSpace(
				lessonPlanID,
			),
		),
	)
}
