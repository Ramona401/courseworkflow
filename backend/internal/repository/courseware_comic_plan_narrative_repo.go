package repository

// courseware_comic_plan_narrative_repo.go
//
// 本文件负责在领取AI规划状态时原子保存本轮叙事方式。
//
// 叙事方式不能先单独写入再调用规划，否则规划前置校验失败时，
// 新叙事方式可能与旧分镜并存。
// 本入口把以下变更放在同一个版本CAS中：
//   - narrative_mode；
//   - status=planning；
//   - project.version递增。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// BeginCoursewareComicProjectPlanningWithNarrative
// 使用版本CAS领取AI规划并保存本轮叙事方式。
func BeginCoursewareComicProjectPlanningWithNarrative(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
	narrativeMode string,
) (*models.CoursewareComicProject, error) {
	narrativeMode =
		strings.TrimSpace(
			narrativeMode,
		)

	if expectedVersion < 1 ||
		!models.IsValidCWComicNarrativeMode(
			narrativeMode,
		) {
		return nil,
			fmt.Errorf(
				"漫画项目版本或叙事方式不合法",
			)
	}

	updated, err :=
		scanCoursewareComicProject(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects
SET narrative_mode = $1,
    status = $2,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5
  AND version = $6
  AND status IN ($7, $8, $9)
RETURNING `+
					coursewareComicProjectSelectColumns,
				narrativeMode,
				models.CWComicProjectStatusPlanning,
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					userID,
				),
				expectedVersion,
				models.CWComicProjectStatusDraft,
				models.CWComicProjectStatusPlanned,
				models.CWComicProjectStatusFailed,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"领取知识点漫画AI规划并保存叙事方式失败: %w",
				err,
			)
	}

	return updated, nil
}
