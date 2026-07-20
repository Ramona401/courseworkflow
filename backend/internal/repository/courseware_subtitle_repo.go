package repository

// courseware_subtitle_repo.go — 课件字幕轨通用数据访问
//
// 表：courseware_subtitles
//
// 安全边界：
//   - 通用字幕按courseware_id + subtitle_id复合读写；
//   - page、video_asset以及兼容空scope_id的editor_draft使用通用UPSERT；
//   - 带真实draft_id的editor_draft字幕专用原子读写拆到
//     courseware_subtitle_editor_draft_repo.go。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 创建/更新（UPSERT） ====================

// UpsertCoursewareSubtitle 创建或更新通用字幕轨。
func UpsertCoursewareSubtitle(
	ctx context.Context,
	sub *models.CoursewareSubtitle,
) error {
	query := `
		INSERT INTO courseware_subtitles (
			courseware_id,
			scope_type,
			scope_id,
			language,
			segments,
			style_config,
			tts_config,
			created_by,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
		ON CONFLICT (
			courseware_id,
			scope_type,
			scope_id,
			language
		)
		DO UPDATE SET
			segments = EXCLUDED.segments,
			style_config = EXCLUDED.style_config,
			tts_config = EXCLUDED.tts_config,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`

	now := time.Now()

	return database.DB.QueryRow(
		ctx,
		query,
		sub.CoursewareID,
		sub.ScopeType,
		sub.ScopeID,
		sub.Language,
		sub.Segments,
		sub.StyleConfig,
		sub.TTSConfig,
		sub.CreatedBy,
		now,
	).Scan(
		&sub.ID,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
}

// ==================== 查询 ====================

// 统一的SELECT列。
const subtitleSelectColumns = `
	id,
	courseware_id,
	scope_type,
	scope_id,
	language,
	segments,
	style_config,
	tts_config,
	created_by,
	created_at,
	updated_at
`

// 带s别名的SELECT列，供JOIN其它表时避免列名歧义。
const subtitleSelectColumnsS = `
	s.id,
	s.courseware_id,
	s.scope_type,
	s.scope_id,
	s.language,
	s.segments,
	s.style_config,
	s.tts_config,
	s.created_by,
	s.created_at,
	s.updated_at
`

// scanSubtitle 统一扫描行到模型。
func scanSubtitle(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareSubtitle, error) {
	s := &models.CoursewareSubtitle{}

	err := scanner.Scan(
		&s.ID,
		&s.CoursewareID,
		&s.ScopeType,
		&s.ScopeID,
		&s.Language,
		&s.Segments,
		&s.StyleConfig,
		&s.TTSConfig,
		&s.CreatedBy,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// scanSubtitleRows 扫描字幕列表并统一处理rows.Err。
func scanSubtitleRows(
	rows pgx.Rows,
) (
	[]*models.CoursewareSubtitle,
	error,
) {
	defer rows.Close()

	results := make(
		[]*models.CoursewareSubtitle,
		0,
	)

	for rows.Next() {
		subtitle, err := scanSubtitle(rows)
		if err != nil {
			return nil,
				fmt.Errorf(
					"扫描字幕行失败: %w",
					err,
				)
		}

		results = append(
			results,
			subtitle,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历字幕列表失败: %w",
				err,
			)
	}

	return results, nil
}

// GetCoursewareSubtitleByID 按ID查询单条字幕轨。
func GetCoursewareSubtitleByID(
	ctx context.Context,
	id string,
) (
	*models.CoursewareSubtitle,
	error,
) {
	query := fmt.Sprintf(
		"SELECT %s FROM courseware_subtitles WHERE id = $1",
		subtitleSelectColumns,
	)

	row := database.DB.QueryRow(
		ctx,
		query,
		id,
	)

	return scanSubtitle(row)
}

// ListCoursewareSubtitles 按课件和可选范围查询通用字幕列表。
func ListCoursewareSubtitles(
	ctx context.Context,
	coursewareID string,
	scopeType string,
	scopeID string,
) (
	[]*models.CoursewareSubtitle,
	error,
) {
	query := fmt.Sprintf(
		"SELECT %s FROM courseware_subtitles WHERE courseware_id::text = $1",
		subtitleSelectColumns,
	)

	args := []interface{}{
		coursewareID,
	}
	argIdx := 2

	if scopeType != "" {
		query += fmt.Sprintf(
			" AND scope_type = $%d",
			argIdx,
		)
		args = append(
			args,
			scopeType,
		)
		argIdx++
	}

	if scopeID != "" {
		query += fmt.Sprintf(
			" AND scope_id::text = $%d",
			argIdx,
		)
		args = append(
			args,
			scopeID,
		)
	}

	query += " ORDER BY language ASC, created_at ASC"

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询字幕列表失败: %w",
				err,
			)
	}

	return scanSubtitleRows(rows)
}

// ==================== 删除 ====================

// DeleteCoursewareSubtitle 按ID删除字幕轨。
func DeleteCoursewareSubtitle(
	ctx context.Context,
	id string,
) error {
	tag, err := database.DB.Exec(
		ctx,
		"DELETE FROM courseware_subtitles WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"删除字幕失败: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"字幕不存在: %s",
			id,
		)
	}

	return nil
}

// DeleteCoursewareSubtitlesByScope 按范围批量删除。
func DeleteCoursewareSubtitlesByScope(
	ctx context.Context,
	coursewareID string,
	scopeType string,
	scopeID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`DELETE FROM courseware_subtitles
		 WHERE courseware_id::text = $1
		   AND scope_type = $2
		   AND scope_id::text = $3`,
		coursewareID,
		scopeType,
		scopeID,
	)
	return err
}

// GetCoursewareSubtitleForCourseware 按课件ID和字幕ID复合读取字幕。
func GetCoursewareSubtitleForCourseware(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
) (
	*models.CoursewareSubtitle,
	error,
) {
	query := fmt.Sprintf(
		`SELECT %s
		 FROM courseware_subtitles
		 WHERE courseware_id::text = $1
		   AND id::text = $2`,
		subtitleSelectColumns,
	)

	row := database.DB.QueryRow(
		ctx,
		query,
		coursewareID,
		subtitleID,
	)

	return scanSubtitle(row)
}

// DeleteCoursewareSubtitleForCourseware 按课件ID和字幕ID复合删除。
func DeleteCoursewareSubtitleForCourseware(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
) error {
	tag, err := database.DB.Exec(
		ctx,
		`DELETE FROM courseware_subtitles
		 WHERE courseware_id::text = $1
		   AND id::text = $2`,
		coursewareID,
		subtitleID,
	)
	if err != nil {
		return fmt.Errorf(
			"删除字幕失败: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"字幕不存在或不属于当前课件",
		)
	}

	return nil
}

// UpdateCoursewareSubtitleSegmentsIfUnchanged 使用updated_at乐观锁更新字幕片段。
func UpdateCoursewareSubtitleSegmentsIfUnchanged(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	expectedUpdatedAt time.Time,
	segments string,
) (
	bool,
	error,
) {
	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_subtitles
		 SET segments = $1,
			 updated_at = $2
		 WHERE courseware_id::text = $3
		   AND id::text = $4
		   AND updated_at = $5`,
		segments,
		time.Now(),
		coursewareID,
		subtitleID,
		expectedUpdatedAt,
	)
	if err != nil {
		return false,
			fmt.Errorf(
				"更新TTS字幕结果失败: %w",
				err,
			)
	}

	return tag.RowsAffected() == 1,
		nil
}
