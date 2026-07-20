package repository

// courseware_subtitle_editor_draft_repo.go — 视频编辑器草稿字幕专用数据访问
//
// 本文件只处理scope_type=editor_draft的个人草稿边界：
//   - 带真实draft_id写入时绑定courseware_id + draft_id + user_id；
//   - 写入与草稿保存、删除、自动裁剪共用事务级咨询锁；
//   - 查询真实draft_id字幕时只返回当前用户自己的草稿字幕；
//   - 查询历史空scope_id字幕时按created_by收窄；
//   - created_by为空的最早期遗留字幕只允许课件作者读取。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// UpsertCoursewareSubtitleForEditorDraft 原子写入带真实draft_id的字幕。
func UpsertCoursewareSubtitleForEditorDraft(
	ctx context.Context,
	sub *models.CoursewareSubtitle,
	userID string,
) error {
	if sub == nil ||
		sub.ScopeID == nil ||
		*sub.ScopeID == "" ||
		sub.CoursewareID == "" ||
		userID == "" {
		return ErrVideoDraftNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始编辑器字幕事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockVideoDraftSequence(
		ctx,
		tx,
		sub.CoursewareID,
		userID,
	); err != nil {
		return fmt.Errorf(
			"锁定视频草稿序列失败: %w",
			err,
		)
	}

	now := time.Now()

	err = tx.QueryRow(
		ctx,
		`INSERT INTO courseware_subtitles (
			courseware_id,
			scope_type,
			scope_id,
			language,
			segments,
			style_config,
			tts_config,
			created_by,
			updated_at
		)
		SELECT
			d.courseware_id,
			'editor_draft',
			d.id,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		FROM video_editor_drafts d
		WHERE d.id::text = $1
		  AND d.courseware_id::text = $2
		  AND d.user_id::text = $3
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
		RETURNING id, created_at, updated_at`,
		*sub.ScopeID,
		sub.CoursewareID,
		userID,
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
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrVideoDraftNotFound
		}

		return fmt.Errorf(
			"保存编辑器草稿字幕失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交编辑器字幕事务失败: %w",
			err,
		)
	}

	return nil
}

// ListCoursewareEditorDraftSubtitles 查询当前用户自己的编辑器字幕。
//
// scopeID非空：只返回属于当前用户该真实草稿的字幕。
// scopeID为空：只返回当前用户创建的历史空scope_id字幕；对于created_by
// 为空的最早期遗留记录，仅课件作者本人可以读取。
func ListCoursewareEditorDraftSubtitles(
	ctx context.Context,
	coursewareID string,
	userID string,
	scopeID string,
) (
	[]*models.CoursewareSubtitle,
	error,
) {
	var (
		query string
		args  []interface{}
	)

	if scopeID != "" {
		query = fmt.Sprintf(
			`SELECT %s
			 FROM courseware_subtitles s
			 JOIN video_editor_drafts d
			   ON d.id = s.scope_id
			  AND d.courseware_id = s.courseware_id
			 WHERE s.courseware_id::text = $1
			   AND s.scope_type = 'editor_draft'
			   AND s.scope_id::text = $2
			   AND d.user_id::text = $3
			 ORDER BY s.language ASC, s.created_at ASC`,
			subtitleSelectColumnsS,
		)

		args = []interface{}{
			coursewareID,
			scopeID,
			userID,
		}
	} else {
		query = fmt.Sprintf(
			`SELECT %s
			 FROM courseware_subtitles s
			 WHERE s.courseware_id::text = $1
			   AND s.scope_type = 'editor_draft'
			   AND s.scope_id IS NULL
			   AND (
				s.created_by::text = $2
				OR (
					s.created_by IS NULL
					AND EXISTS (
						SELECT 1
						FROM coursewares c
						WHERE c.id = s.courseware_id
						  AND c.user_id::text = $2
						  AND c.deleted_at IS NULL
					)
				)
			   )
			 ORDER BY s.language ASC, s.created_at ASC`,
			subtitleSelectColumnsS,
		)

		args = []interface{}{
			coursewareID,
			userID,
		}
	}

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询编辑器字幕列表失败: %w",
				err,
			)
	}

	return scanSubtitleRows(rows)
}
