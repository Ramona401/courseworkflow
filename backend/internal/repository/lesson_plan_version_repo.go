package repository

// lesson_plan_version_repo.go — 教案正文版本历史数据访问层
//
// 职责：
//   1. 查询某份教案的历史版本列表。
//   2. 查询单个历史版本的完整正文。
//   3. 提供版本恢复服务所需的精确版本定位。
//
// 快照创建本身位于lesson_plan_repo.go的UpdateLessonPlanContent事务中，
// 以保证“保存旧版”和“覆盖当前正文”要么同时成功，要么同时回滚。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrLessonPlanVersionNotFound 教案正文历史版本不存在。
var ErrLessonPlanVersionNotFound = errors.New("教案历史版本不存在")

// ListLessonPlanContentVersions 查询版本列表。
func ListLessonPlanContentVersions(
	ctx context.Context,
	lessonPlanID string,
	limit int,
	offset int,
) ([]*models.LessonPlanContentVersionListItem, int, int, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := database.DB.QueryRow(
		ctx,
		`
SELECT COUNT(*)
FROM lesson_plan_content_versions
WHERE lesson_plan_id = $1
`,
		lessonPlanID,
	).Scan(&total); err != nil {
		return nil, 0, 0, fmt.Errorf("查询教案版本总数失败: %w", err)
	}

	var currentVersion int
	if err := database.DB.QueryRow(
		ctx,
		`
SELECT version
FROM lesson_plans
WHERE id = $1
  AND deleted_at IS NULL
`,
		lessonPlanID,
	).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, 0, ErrLessonPlanNotFound
		}
		return nil, 0, 0, fmt.Errorf("查询教案当前版本失败: %w", err)
	}

	rows, err := database.DB.Query(
		ctx,
		`
SELECT
	v.id,
	v.version_number,
	v.title,
	LEFT(
		REGEXP_REPLACE(v.content_markdown, E'[\\n\\r\\t ]+', ' ', 'g'),
		160
	) AS content_preview,
	CHAR_LENGTH(v.content_markdown) AS character_count,
	v.duration_minutes,
	v.change_source,
	v.changed_by,
	COALESCE(u.display_name, '') AS changed_by_name,
	v.change_summary,
	v.created_at
FROM lesson_plan_content_versions v
LEFT JOIN users u ON u.id = v.changed_by
WHERE v.lesson_plan_id = $1
ORDER BY v.version_number DESC, v.created_at DESC
LIMIT $2 OFFSET $3
`,
		lessonPlanID,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("查询教案版本列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]*models.LessonPlanContentVersionListItem, 0)

	for rows.Next() {
		item := &models.LessonPlanContentVersionListItem{}
		if err := rows.Scan(
			&item.ID,
			&item.VersionNumber,
			&item.Title,
			&item.ContentPreview,
			&item.CharacterCount,
			&item.DurationMinutes,
			&item.ChangeSource,
			&item.ChangedBy,
			&item.ChangedByName,
			&item.ChangeSummary,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, 0, fmt.Errorf("扫描教案版本列表失败: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("遍历教案版本列表失败: %w", err)
	}

	return items, total, currentVersion, nil
}

// GetLessonPlanContentVersion 查询单个完整历史版本。
// lessonPlanID同时参与过滤，防止拿其它教案的versionID越权读取正文。
func GetLessonPlanContentVersion(
	ctx context.Context,
	lessonPlanID string,
	versionID string,
) (*models.LessonPlanContentVersion, error) {
	item := &models.LessonPlanContentVersion{}

	err := database.DB.QueryRow(
		ctx,
		`
SELECT
	v.id,
	v.lesson_plan_id,
	v.version_number,
	v.title,
	v.content_markdown,
	v.content_structured::text,
	v.duration_minutes,
	v.change_source,
	v.changed_by,
	COALESCE(u.display_name, '') AS changed_by_name,
	v.change_summary,
	v.created_at
FROM lesson_plan_content_versions v
LEFT JOIN users u ON u.id = v.changed_by
WHERE v.id = $1
  AND v.lesson_plan_id = $2
`,
		versionID,
		lessonPlanID,
	).Scan(
		&item.ID,
		&item.LessonPlanID,
		&item.VersionNumber,
		&item.Title,
		&item.ContentMarkdown,
		&item.ContentStructured,
		&item.DurationMinutes,
		&item.ChangeSource,
		&item.ChangedBy,
		&item.ChangedByName,
		&item.ChangeSummary,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanVersionNotFound
		}
		return nil, fmt.Errorf("查询教案历史版本失败: %w", err)
	}

	return item, nil
}
