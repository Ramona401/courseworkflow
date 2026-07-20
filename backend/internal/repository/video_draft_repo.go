package repository

// video_draft_repo.go — 视频编辑器草稿数据访问
//
// 本文件只承担数据库读写，不判断HTTP身份或课件业务权限。
//
// 安全边界：
//   - 所有列表、读取和删除均同时绑定courseware_id与user_id；
//   - 删除不能只凭draft_id，防止通过错误课件路径操作其它资源；
//   - 保存使用事务和事务级咨询锁，串行化同一“课件+用户”的版本序列；
//   - 咨询锁使用两个独立text哈希参数，避免在PostgreSQL text中传入NUL；
//   - 先插入新草稿，确认成功后再裁剪超额旧草稿；
//   - 裁剪和主动删除草稿时，同事务删除其editor_draft字幕；
//   - 任一步失败均回滚，不留下“草稿删了但新草稿没保存”的半完成状态。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

var ErrVideoDraftNotFound = errors.New(
	"视频草稿不存在",
)

// VideoDraftItem 视频编辑器草稿列表项。
type VideoDraftItem struct {
	ID           string          `json:"id"`
	CoursewareID string          `json:"courseware_id"`
	UserID       string          `json:"user_id"`
	Name         string          `json:"name"`
	ClipsData    json.RawMessage `json:"clips_data"`
	ClipCount    int             `json:"clip_count"`
	CreatedAt    time.Time       `json:"created_at"`
}

// lockVideoDraftSequence 锁定同一“课件+用户”的草稿序列。
//
// PostgreSQL text不能包含NUL，因此不能先拼接两个ID再传给hashtext。
// 这里使用pg_advisory_xact_lock(int,int)的双键形式，
// 两个参数分别哈希，保存、删除和editor_draft字幕写入共用同一锁。
func lockVideoDraftSequence(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	userID string,
) error {
	_, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(
			hashtext($1),
			hashtext($2)
		)`,
		coursewareID,
		userID,
	)
	return err
}

// VideoDraftAssetsBelongToCourseware 确认全部片段ID属于当前课件的视频资产。
func VideoDraftAssetsBelongToCourseware(
	ctx context.Context,
	coursewareID string,
	assetIDs []string,
) (
	bool,
	error,
) {
	if coursewareID == "" ||
		len(assetIDs) == 0 {
		return false, nil
	}

	var matchedCount int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_assets
		 WHERE courseware_id::text = $1
		   AND asset_type = 'video'
		   AND id::text = ANY($2::text[])`,
		coursewareID,
		assetIDs,
	).Scan(&matchedCount); err != nil {
		return false,
			fmt.Errorf(
				"校验视频草稿资产失败: %w",
				err,
			)
	}

	return matchedCount == len(assetIDs),
		nil
}

// CreateVideoDraftCapped 原子创建草稿，并只保留最近maxKeep条。
//
// 同一课件、同一用户的草稿序列通过事务级咨询锁串行化，
// 防止两个并发保存请求同时突破版本上限。
func CreateVideoDraftCapped(
	ctx context.Context,
	coursewareID string,
	userID string,
	name string,
	clipsJSON string,
	clipCount int,
	maxKeep int,
) (
	*VideoDraftItem,
	error,
) {
	if maxKeep <= 0 {
		maxKeep = 20
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开始视频草稿事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockVideoDraftSequence(
		ctx,
		tx,
		coursewareID,
		userID,
	); err != nil {
		return nil,
			fmt.Errorf(
				"锁定视频草稿序列失败: %w",
				err,
			)
	}

	item := &VideoDraftItem{
		CoursewareID: coursewareID,
		UserID:       userID,
		Name:         name,
		ClipsData:    json.RawMessage(clipsJSON),
		ClipCount:    clipCount,
	}

	if err := tx.QueryRow(
		ctx,
		`INSERT INTO video_editor_drafts (
			id,
			courseware_id,
			user_id,
			name,
			clips_data,
			clip_count
		)
		VALUES (
			gen_random_uuid(),
			$1,
			$2,
			$3,
			$4::jsonb,
			$5
		)
		RETURNING
			id::text,
			created_at`,
		coursewareID,
		userID,
		name,
		clipsJSON,
		clipCount,
	).Scan(
		&item.ID,
		&item.CreatedAt,
	); err != nil {
		return nil,
			fmt.Errorf(
				"创建视频草稿失败: %w",
				err,
			)
	}

	// 新记录写入成功后，找出超过上限的旧草稿。
	rows, err := tx.Query(
		ctx,
		`SELECT id::text
		 FROM video_editor_drafts
		 WHERE courseware_id::text = $1
		   AND user_id::text = $2
		 ORDER BY created_at DESC, id DESC
		 OFFSET $3
		 FOR UPDATE`,
		coursewareID,
		userID,
		maxKeep,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"读取超额视频草稿失败: %w",
				err,
			)
	}

	doomedIDs := make([]string, 0)

	for rows.Next() {
		var draftID string

		if err := rows.Scan(&draftID); err != nil {
			rows.Close()

			return nil,
				fmt.Errorf(
					"扫描超额视频草稿失败: %w",
					err,
				)
		}

		doomedIDs = append(
			doomedIDs,
			draftID,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()

		return nil,
			fmt.Errorf(
				"遍历超额视频草稿失败: %w",
				err,
			)
	}

	rows.Close()

	if len(doomedIDs) > 0 {
		if _, err := tx.Exec(
			ctx,
			`DELETE FROM courseware_subtitles
			 WHERE courseware_id::text = $1
			   AND scope_type = 'editor_draft'
			   AND scope_id::text = ANY($2::text[])`,
			coursewareID,
			doomedIDs,
		); err != nil {
			return nil,
				fmt.Errorf(
					"清理旧草稿字幕失败: %w",
					err,
				)
		}

		if _, err := tx.Exec(
			ctx,
			`DELETE FROM video_editor_drafts
			 WHERE courseware_id::text = $1
			   AND user_id::text = $2
			   AND id::text = ANY($3::text[])`,
			coursewareID,
			userID,
			doomedIDs,
		); err != nil {
			return nil,
				fmt.Errorf(
					"清理超额视频草稿失败: %w",
					err,
				)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交视频草稿事务失败: %w",
				err,
			)
	}

	return item, nil
}

// ListVideoDrafts 获取指定课件和用户自己的草稿列表。
func ListVideoDrafts(
	ctx context.Context,
	coursewareID string,
	userID string,
) (
	[]*VideoDraftItem,
	error,
) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT
			id::text,
			courseware_id::text,
			user_id::text,
			COALESCE(name, ''),
			clips_data,
			clip_count,
			created_at
		 FROM video_editor_drafts
		 WHERE courseware_id::text = $1
		   AND user_id::text = $2
		 ORDER BY created_at DESC, id DESC
		 LIMIT 20`,
		coursewareID,
		userID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询视频草稿列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	drafts := make(
		[]*VideoDraftItem,
		0,
	)

	for rows.Next() {
		draft := &VideoDraftItem{}

		if err := rows.Scan(
			&draft.ID,
			&draft.CoursewareID,
			&draft.UserID,
			&draft.Name,
			&draft.ClipsData,
			&draft.ClipCount,
			&draft.CreatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"扫描视频草稿失败: %w",
					err,
				)
		}

		drafts = append(
			drafts,
			draft,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历视频草稿失败: %w",
				err,
			)
	}

	return drafts, nil
}

// GetVideoDraftForCoursewareUser 使用三层边界读取草稿。
func GetVideoDraftForCoursewareUser(
	ctx context.Context,
	coursewareID string,
	draftID string,
	userID string,
) (
	*VideoDraftItem,
	error,
) {
	draft := &VideoDraftItem{}

	err := database.DB.QueryRow(
		ctx,
		`SELECT
			id::text,
			courseware_id::text,
			user_id::text,
			COALESCE(name, ''),
			clips_data,
			clip_count,
			created_at
		 FROM video_editor_drafts
		 WHERE id::text = $1
		   AND courseware_id::text = $2
		   AND user_id::text = $3`,
		draftID,
		coursewareID,
		userID,
	).Scan(
		&draft.ID,
		&draft.CoursewareID,
		&draft.UserID,
		&draft.Name,
		&draft.ClipsData,
		&draft.ClipCount,
		&draft.CreatedAt,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrVideoDraftNotFound
		}

		return nil,
			fmt.Errorf(
				"查询视频草稿失败: %w",
				err,
			)
	}

	return draft, nil
}

// DeleteVideoDraftForCoursewareUser 原子删除草稿及其绑定字幕。
func DeleteVideoDraftForCoursewareUser(
	ctx context.Context,
	coursewareID string,
	draftID string,
	userID string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始删除视频草稿事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockVideoDraftSequence(
		ctx,
		tx,
		coursewareID,
		userID,
	); err != nil {
		return fmt.Errorf(
			"锁定视频草稿序列失败: %w",
			err,
		)
	}

	var lockedID string

	if err := tx.QueryRow(
		ctx,
		`SELECT id::text
		 FROM video_editor_drafts
		 WHERE id::text = $1
		   AND courseware_id::text = $2
		   AND user_id::text = $3
		 FOR UPDATE`,
		draftID,
		coursewareID,
		userID,
	).Scan(&lockedID); err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrVideoDraftNotFound
		}

		return fmt.Errorf(
			"锁定待删除视频草稿失败: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM courseware_subtitles
		 WHERE courseware_id::text = $1
		   AND scope_type = 'editor_draft'
		   AND scope_id::text = $2`,
		coursewareID,
		lockedID,
	); err != nil {
		return fmt.Errorf(
			"删除视频草稿字幕失败: %w",
			err,
		)
	}

	tag, err := tx.Exec(
		ctx,
		`DELETE FROM video_editor_drafts
		 WHERE id::text = $1
		   AND courseware_id::text = $2
		   AND user_id::text = $3`,
		lockedID,
		coursewareID,
		userID,
	)
	if err != nil {
		return fmt.Errorf(
			"删除视频草稿失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrVideoDraftNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交删除视频草稿事务失败: %w",
			err,
		)
	}

	return nil
}

// GetVideoDraftByID 按ID读取草稿。
//
// 仅保留给已经在Service层完成额外归属校验的兼容调用。
// 新的editor_draft字幕链优先使用GetVideoDraftForCoursewareUser。
func GetVideoDraftByID(
	ctx context.Context,
	draftID string,
) (
	*VideoDraftItem,
	error,
) {
	draft := &VideoDraftItem{}

	err := database.DB.QueryRow(
		ctx,
		`SELECT
			id::text,
			courseware_id::text,
			user_id::text,
			COALESCE(name, ''),
			clips_data,
			clip_count,
			created_at
		 FROM video_editor_drafts
		 WHERE id::text = $1`,
		draftID,
	).Scan(
		&draft.ID,
		&draft.CoursewareID,
		&draft.UserID,
		&draft.Name,
		&draft.ClipsData,
		&draft.ClipCount,
		&draft.CreatedAt,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrVideoDraftNotFound
		}

		return nil,
			fmt.Errorf(
				"查询视频草稿失败: %w",
				err,
			)
	}

	return draft, nil
}
