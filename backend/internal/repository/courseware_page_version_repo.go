package repository

// courseware_page_version_repo.go — 课件页面完整版本快照仓储
//
// 版本兼容策略：
//   - 迁移前历史记录metadata_snapshot_complete=false，只保证HTML快照；
//   - 迁移后新记录metadata_snapshot_complete=true，同时保存页面JSON元数据和状态；
//   - 页面与课件使用组合外键，数据库层禁止归属错配；
//   - page_id+version_no使用唯一约束；
//   - 每页保留最近20个版本。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrCoursewarePageVersionNotFound = errors.New(
	"课件页面版本不存在",
)

const cwPageVersionMaxKeep = 20

// CreatePageVersion 保存覆盖前的页面版本。
//
// HTML由调用方传入；JSON元数据和页面状态从正式页面行重新读取，
// 防止调用方遗漏这些字段。版本创建、编号和裁剪在同一事务内完成。
func CreatePageVersion(
	ctx context.Context,
	pageID string,
	coursewareID string,
	html string,
	source string,
	note string,
) (*models.CoursewarePageVersion, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始页面版本事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 同一页面版本序列必须串行计算。
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`,
		pageID,
	); err != nil {
		return nil, fmt.Errorf(
			"锁定页面版本序列失败: %w",
			err,
		)
	}

	version := &models.CoursewarePageVersion{
		PageID:                   pageID,
		CoursewareID:             coursewareID,
		HTMLContent:              html,
		Source:                   source,
		Note:                     note,
		MetadataSnapshotComplete: true,
	}

	// 从正式页面重新读取元数据，并锁定页面行直到版本事务完成。
	err = tx.QueryRow(
		ctx,
		`SELECT
			COALESCE(placeholder_map::text, ''),
			COALESCE(matched_component_ids::text, ''),
			status
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR SHARE`,
		pageID,
		coursewareID,
	).Scan(
		&version.PlaceholderMap,
		&version.MatchedComponentIDs,
		&version.PageStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewarePageVersionNotFound
		}
		return nil, fmt.Errorf(
			"读取页面版本元数据失败: %w",
			err,
		)
	}

	if err := tx.QueryRow(
		ctx,
		`SELECT COALESCE(MAX(version_no), 0) + 1
		 FROM courseware_page_versions
		 WHERE page_id = $1
		   AND courseware_id = $2`,
		pageID,
		coursewareID,
	).Scan(&version.VersionNo); err != nil {
		return nil, fmt.Errorf(
			"计算页面版本号失败: %w",
			err,
		)
	}

	if err := tx.QueryRow(
		ctx,
		`INSERT INTO courseware_page_versions (
			id,
			page_id,
			courseware_id,
			version_no,
			html_content,
			placeholder_map,
			matched_component_ids,
			page_status,
			metadata_snapshot_complete,
			source,
			note
		)
		VALUES (
			gen_random_uuid(),
			$1,
			$2,
			$3,
			$4,
			$5::jsonb,
			$6::jsonb,
			$7,
			true,
			$8,
			$9
		)
		RETURNING
			id,
			created_at`,
		pageID,
		coursewareID,
		version.VersionNo,
		html,
		nullIfEmpty(version.PlaceholderMap),
		nullIfEmpty(version.MatchedComponentIDs),
		version.PageStatus,
		source,
		nullIfEmpty(note),
	).Scan(
		&version.ID,
		&version.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf(
			"写入页面完整版本快照失败: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM courseware_page_versions
		 WHERE page_id = $1
		   AND courseware_id = $2
		   AND id NOT IN (
				SELECT id
				FROM courseware_page_versions
				WHERE page_id = $1
				  AND courseware_id = $2
				ORDER BY version_no DESC
				LIMIT $3
		   )`,
		pageID,
		coursewareID,
		cwPageVersionMaxKeep,
	); err != nil {
		return nil, fmt.Errorf(
			"裁剪页面历史版本失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交页面版本事务失败: %w",
			err,
		)
	}

	return version, nil
}

// ListPageVersions 返回指定课件页面的版本列表。
func ListPageVersions(
	ctx context.Context,
	pageID string,
	coursewareID string,
) ([]*models.CoursewarePageVersionListItem, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT
			id,
			version_no,
			source,
			COALESCE(note, ''),
			metadata_snapshot_complete,
			created_at
		 FROM courseware_page_versions
		 WHERE page_id = $1
		   AND courseware_id = $2
		 ORDER BY version_no DESC`,
		pageID,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询页面版本列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewarePageVersionListItem,
		0,
	)

	for rows.Next() {
		item :=
			&models.CoursewarePageVersionListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.VersionNo,
			&item.Source,
			&item.Note,
			&item.MetadataSnapshotComplete,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描页面版本列表失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历页面版本列表失败: %w",
			err,
		)
	}

	return items, nil
}

// GetPageVersion 使用version、page和courseware三层边界读取版本。
func GetPageVersion(
	ctx context.Context,
	versionID string,
	pageID string,
	coursewareID string,
) (*models.CoursewarePageVersion, error) {
	version := &models.CoursewarePageVersion{}

	err := database.DB.QueryRow(
		ctx,
		`SELECT
			id,
			page_id,
			courseware_id,
			version_no,
			html_content,
			COALESCE(placeholder_map::text, ''),
			COALESCE(matched_component_ids::text, ''),
			COALESCE(page_status, ''),
			metadata_snapshot_complete,
			source,
			COALESCE(note, ''),
			created_at
		 FROM courseware_page_versions
		 WHERE id = $1
		   AND page_id = $2
		   AND courseware_id = $3`,
		versionID,
		pageID,
		coursewareID,
	).Scan(
		&version.ID,
		&version.PageID,
		&version.CoursewareID,
		&version.VersionNo,
		&version.HTMLContent,
		&version.PlaceholderMap,
		&version.MatchedComponentIDs,
		&version.PageStatus,
		&version.MetadataSnapshotComplete,
		&version.Source,
		&version.Note,
		&version.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewarePageVersionNotFound
		}

		return nil, fmt.Errorf(
			"查询页面版本失败: %w",
			err,
		)
	}

	return version, nil
}

// CountPageVersions 统计某页现有版本数。
func CountPageVersions(
	ctx context.Context,
	pageID string,
) (int, error) {
	var count int

	err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_page_versions
		 WHERE page_id = $1`,
		pageID,
	).Scan(&count)

	return count, err
}
