package repository

// courseware_comic_style_preview_complete_repo.go
//
// 本文件负责第三步首格样张生成成功后的独立结算：
//   - 校验项目仍处于planned和style_preview；
//   - 校验第1格仍处于generating；
//   - 校验图片资产属于当前课件；
//   - 创建不可变分格历史版本；
//   - 把图片绑定为第1格当前资产；
//   - 保存style_preview_panel_id。
//
// 本事务不会：
//   - 把项目改为generating、ready或failed；
//   - 确认视觉风格；
//   - 自动生成其余分格；
//   - 删除任何旧资产或旧历史版本。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CompleteCoursewareComicStylePreview 完成第1格样张生成。
func CompleteCoursewareComicStylePreview(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	assetID string,
	aociText string,
	generationSource string,
) (*models.CoursewareComicPanel, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	panelID =
		strings.TrimSpace(
			panelID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	assetID =
		strings.TrimSpace(
			assetID,
		)

	aociText =
		strings.TrimSpace(
			aociText,
		)

	generationSource =
		strings.TrimSpace(
			generationSource,
		)

	if coursewareID == "" ||
		projectID == "" ||
		panelID == "" ||
		userID == "" ||
		assetID == "" {
		return nil,
			fmt.Errorf(
				"首格样张完成参数无效",
			)
	}

	switch generationSource {
	case models.CWComicVersionSourceInitial,
		models.CWComicVersionSourceRegenerate:
	default:
		return nil,
			fmt.Errorf(
				"首格样张版本来源不合法",
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启首格样张完成事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	var lockedProjectID string

	err =
		tx.QueryRow(
			ctx,
			`SELECT id
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
  AND status = $4
  AND workflow_stage = $5
  AND storyboard_confirmed_at IS NOT NULL
FOR UPDATE`,
			projectID,
			coursewareID,
			userID,
			models.CWComicProjectStatusPlanned,
			models.CWComicWorkflowStylePreview,
		).Scan(
			&lockedProjectID,
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
				"锁定首格样张项目失败: %w",
				err,
			)
	}

	panel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`SELECT `+
					coursewareComicPanelSelectColumns+
					` FROM courseware_comic_panels
WHERE id = $1
  AND project_id = $2
  AND panel_no = 1
FOR UPDATE`,
				panelID,
				projectID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicPanelNotFound
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"锁定首格样张分格失败: %w",
				err,
			)
	}

	if panel.Status !=
		models.CWComicPanelStatusGenerating {
		return nil,
			ErrCoursewareComicPanelConflict
	}

	var validAsset bool

	err =
		tx.QueryRow(
			ctx,
			`SELECT EXISTS (
    SELECT 1
    FROM courseware_assets
    WHERE id = $1
      AND courseware_id = $2
      AND asset_type = 'image'
)`,
			assetID,
			coursewareID,
		).Scan(
			&validAsset,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"校验首格样张资产失败: %w",
				err,
			)
	}

	if !validAsset {
		return nil,
			ErrCoursewareComicAssetInvalid
	}

	if aociText == "" {
		aociText =
			strings.TrimSpace(
				panel.AOCIText,
			)
	}

	if aociText == "" {
		return nil,
			fmt.Errorf(
				"首格样张缺少IAOCI",
			)
	}

	var nextVersionNo int

	err =
		tx.QueryRow(
			ctx,
			`SELECT COALESCE(MAX(version_no), 0) + 1
FROM courseware_comic_panel_versions
WHERE panel_id = $1`,
			panelID,
		).Scan(
			&nextVersionNo,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"计算首格样张历史版本号失败: %w",
				err,
			)
	}

	_, err =
		tx.Exec(
			ctx,
			`INSERT INTO courseware_comic_panel_versions (
panel_id,
version_no,
prompt_snapshot,
aoci_snapshot,
overlay_document_snapshot,
asset_id,
generation_source,
created_by
)
VALUES (
$1, $2, $3, $4,
$5::jsonb, $6, $7, $8
)`,
			panelID,
			nextVersionNo,
			panel.VisualPrompt,
			aociText,
			panel.OverlayDocumentJSON,
			assetID,
			generationSource,
			userID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"保存首格样张历史版本失败: %w",
				err,
			)
	}

	updatedPanel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels
SET current_asset_id = $1,
    aoci_text = $2,
    status = $3,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $4
  AND project_id = $5
  AND panel_no = 1
  AND status = $6
RETURNING `+
					coursewareComicPanelSelectColumns,
				assetID,
				aociText,
				models.CWComicPanelStatusGenerated,
				panelID,
				projectID,
				models.CWComicPanelStatusGenerating,
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
				"绑定首格样张资产失败: %w",
				err,
			)
	}

	projectTag, err :=
		tx.Exec(
			ctx,
			`UPDATE courseware_comic_projects
SET style_preview_panel_id = $1,
    style_confirmed_at = NULL,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND status = $5
  AND workflow_stage = $6
  AND storyboard_confirmed_at IS NOT NULL`,
			panelID,
			projectID,
			coursewareID,
			userID,
			models.CWComicProjectStatusPlanned,
			models.CWComicWorkflowStylePreview,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"保存首格样张定位失败: %w",
				err,
			)
	}

	if projectTag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return nil,
			fmt.Errorf(
				"提交首格样张完成事务失败: %w",
				err,
			)
	}

	return updatedPanel, nil
}
