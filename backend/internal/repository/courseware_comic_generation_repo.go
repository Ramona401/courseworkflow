package repository

// courseware_comic_generation_repo.go — 漫画图片生产状态仓储
//
// 本文件负责：
//   1. 使用项目版本CAS领取整批生成或中断恢复；
//   2. 保存项目人物设定图资产ID；
//   3. 在图片资产已经形成但项目版本CAS失败时补绑人物设定图；
//   4. 领取单格重新生成任务，并把ready或inserted项目重新置为generating。
//
// 整批生成领取必须同时满足：
//   - 工作流为batch_generation；
//   - style_confirmed_at非空；
//   - style_preview_panel_id指向本项目第1格；
//   - 第1格已经生成并绑定有效图片资产。
//
// planned、failed和进程重启后遗留的generating项目均可领取。
// 同一进程内的并发重复由GlobalBackgroundTasks任务键阻止。
//
// 锁顺序统一为：
//   courseware_comic_projects → courseware_comic_panels。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// BeginCoursewareComicProjectGeneration 使用项目版本领取整批图片生成或恢复。
func BeginCoursewareComicProjectGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
) (*models.CoursewareComicProject, error) {
	if expectedVersion < 1 {
		return nil,
			fmt.Errorf(
				"漫画项目版本号不合法",
			)
	}

	item, err :=
		scanCoursewareComicProject(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects project
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE project.id = $2
  AND project.courseware_id = $3
  AND project.created_by = $4
  AND project.version = $5
  AND project.status IN ($6, $7, $8)
  AND project.workflow_stage = $9
  AND project.style_confirmed_at IS NOT NULL
  AND project.style_preview_panel_id IS NOT NULL
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      JOIN courseware_assets asset
        ON asset.id = panel.current_asset_id
      WHERE panel.id = project.style_preview_panel_id
        AND panel.project_id = project.id
        AND panel.panel_no = 1
        AND panel.status = $10
        AND panel.current_asset_id IS NOT NULL
        AND asset.courseware_id = project.courseware_id
        AND asset.asset_type = 'image'
  )
RETURNING `+
					coursewareComicProjectSelectColumns,
				models.CWComicProjectStatusGenerating,
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
				models.CWComicProjectStatusPlanned,
				models.CWComicProjectStatusFailed,
				models.CWComicProjectStatusGenerating,
				models.CWComicWorkflowBatchGeneration,
				models.CWComicPanelStatusGenerated,
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
				"领取知识点漫画整批生成失败: %w",
				err,
			)
	}

	return item, nil
}

// UpdateCoursewareComicProjectCharacterSheet 保存人物设定图。
func UpdateCoursewareComicProjectCharacterSheet(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	assetID string,
	expectedVersion int,
) (*models.CoursewareComicProject, error) {
	assetID =
		strings.TrimSpace(
			assetID,
		)

	if assetID == "" ||
		expectedVersion < 1 {
		return nil,
			fmt.Errorf(
				"人物设定图资产或项目版本无效",
			)
	}

	var validAsset bool

	err :=
		database.DB.QueryRow(
			ctx,
			`SELECT EXISTS (
    SELECT 1
    FROM courseware_assets
    WHERE id = $1
      AND courseware_id = $2
      AND asset_type = 'image'
)`,
			assetID,
			strings.TrimSpace(
				coursewareID,
			),
		).Scan(
			&validAsset,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"校验人物设定图资产失败: %w",
				err,
			)
	}

	if !validAsset {
		return nil,
			ErrCoursewareComicAssetInvalid
	}

	item, err :=
		scanCoursewareComicProject(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects
SET character_sheet_asset_id = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND version = $5
  AND status = $6
RETURNING `+
					coursewareComicProjectSelectColumns,
				assetID,
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
				models.CWComicProjectStatusGenerating,
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
				"保存漫画人物设定图失败: %w",
				err,
			)
	}

	return item, nil
}

// AttachCoursewareComicProjectCharacterSheetIfMissing
// 在图片资产和计费记录已经形成、但项目版本CAS绑定失败时执行补绑。
//
// 本操作不会覆盖已经存在的其他人物设定图。
// 已经绑定同一个asset_id时直接返回当前项目，保证重复恢复幂等。
func AttachCoursewareComicProjectCharacterSheetIfMissing(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	assetID string,
) (*models.CoursewareComicProject, error) {
	assetID =
		strings.TrimSpace(
			assetID,
		)

	if assetID == "" {
		return nil,
			fmt.Errorf(
				"人物设定图资产无效",
			)
	}

	var validAsset bool

	err :=
		database.DB.QueryRow(
			ctx,
			`SELECT EXISTS (
    SELECT 1
    FROM courseware_assets
    WHERE id = $1
      AND courseware_id = $2
      AND asset_type = 'image'
)`,
			assetID,
			strings.TrimSpace(
				coursewareID,
			),
		).Scan(
			&validAsset,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"校验待恢复人物设定图失败: %w",
				err,
			)
	}

	if !validAsset {
		return nil,
			ErrCoursewareComicAssetInvalid
	}

	item, err :=
		scanCoursewareComicProject(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects
SET character_sheet_asset_id = CASE
        WHEN character_sheet_asset_id IS NULL THEN $1
        ELSE character_sheet_asset_id
    END,
    version = CASE
        WHEN character_sheet_asset_id IS NULL THEN version + 1
        ELSE version
    END,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND status = $5
  AND (
      character_sheet_asset_id IS NULL
      OR character_sheet_asset_id = $1
  )
RETURNING `+
					coursewareComicProjectSelectColumns,
				assetID,
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					userID,
				),
				models.CWComicProjectStatusGenerating,
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
				"恢复绑定漫画人物设定图失败: %w",
				err,
			)
	}

	return item, nil
}

// ClaimCoursewareComicPanelRegeneration 领取单格重新生成。
func ClaimCoursewareComicPanelRegeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedPanelVersion int,
) (*models.CoursewareComicPanel, error) {
	if expectedPanelVersion < 1 {
		return nil,
			fmt.Errorf(
				"漫画格版本号不合法",
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启漫画格重画事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	var projectStatus string

	err =
		tx.QueryRow(
			ctx,
			`SELECT status
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
FOR UPDATE`,
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				userID,
			),
		).Scan(
			&projectStatus,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"锁定漫画项目失败: %w",
				err,
			)
	}

	switch projectStatus {
	case models.CWComicProjectStatusPlanned,
		models.CWComicProjectStatusFailed,
		models.CWComicProjectStatusReady,
		models.CWComicProjectStatusInserted:

	default:
		return nil,
			ErrCoursewareComicProjectNotEditable
	}

	panel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND project_id = $3
  AND version = $4
  AND status IN ($5, $6, $7, $8)
RETURNING `+
					coursewareComicPanelSelectColumns,
				models.CWComicPanelStatusGenerating,
				strings.TrimSpace(
					panelID,
				),
				strings.TrimSpace(
					projectID,
				),
				expectedPanelVersion,
				models.CWComicPanelStatusPlanned,
				models.CWComicPanelStatusGenerated,
				models.CWComicPanelStatusFailed,
				models.CWComicPanelStatusStale,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"领取漫画格重新生成失败: %w",
				err,
			)
	}

	tag, err :=
		tx.Exec(
			ctx,
			`UPDATE courseware_comic_projects
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4`,
			models.CWComicProjectStatusGenerating,
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				userID,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"更新漫画项目重画状态失败: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return nil,
			fmt.Errorf(
				"提交漫画格重画事务失败: %w",
				err,
			)
	}

	return panel, nil
}
