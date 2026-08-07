package repository

// courseware_comic_workshop_edit_repo.go — 漫画工坊持续编辑仓储
//
// 允许planned、ready、inserted、failed项目继续保存：
//   - 文字、题目和气泡覆盖层；
//   - 单格图片提示词与IAOCI。
//
// generating和archived项目仍拒绝编辑。
// 全部写操作继续使用panel.version CAS。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// UpdateCoursewareComicPanelOverlayForWorkshopIfUnchanged
// 保存工坊中的文字、题目和气泡排版。
func UpdateCoursewareComicPanelOverlayForWorkshopIfUnchanged(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
	narrationText string,
	dialoguesJSON string,
	overlayDocumentJSON string,
) (*models.CoursewareComicPanel, error) {
	var err error

	dialoguesJSON, err =
		cwComicNormalizeJSON(
			dialoguesJSON,
			"[]",
			"array",
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"漫画对白无效: %w",
				err,
			)
	}

	overlayDocumentJSON, err =
		cwComicNormalizeJSON(
			overlayDocumentJSON,
			"{}",
			"object",
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"漫画覆盖层无效: %w",
				err,
			)
	}

	updated, err :=
		scanCoursewareComicPanel(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels panel
SET narration_text = $1,
    dialogues_json = $2::jsonb,
    overlay_document_json = $3::jsonb,
    overlay_version = overlay_version + 1,
    version = version + 1,
    updated_at = now()
WHERE panel.id = $4
  AND panel.project_id = $5
  AND panel.version = $6
  AND panel.status IN ($7, $8, $9, $10)
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $11
        AND project.created_by = $12
        AND project.status IN ($13, $14, $15, $16)
  )
RETURNING `+coursewareComicPanelSelectColumns,
				strings.TrimSpace(narrationText),
				dialoguesJSON,
				overlayDocumentJSON,
				strings.TrimSpace(panelID),
				strings.TrimSpace(projectID),
				expectedVersion,
				models.CWComicPanelStatusPlanned,
				models.CWComicPanelStatusGenerated,
				models.CWComicPanelStatusFailed,
				models.CWComicPanelStatusStale,
				strings.TrimSpace(coursewareID),
				strings.TrimSpace(userID),
				models.CWComicProjectStatusPlanned,
				models.CWComicProjectStatusReady,
				models.CWComicProjectStatusInserted,
				models.CWComicProjectStatusFailed,
			),
		)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"保存漫画文字和气泡排版失败: %w",
				err,
			)
	}

	return updated, nil
}

// UpdateCoursewareComicPanelPromptForWorkshopIfUnchanged
// 保存工坊中的单格图片提示词与IAOCI。
func UpdateCoursewareComicPanelPromptForWorkshopIfUnchanged(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
	visualPrompt string,
	negativePrompt string,
	aociText string,
	relationsJSON string,
) (*models.CoursewareComicPanel, error) {
	relationsJSON, err :=
		cwComicNormalizeJSON(
			relationsJSON,
			"[]",
			"array",
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"漫画格关系无效: %w",
				err,
			)
	}

	visualPrompt =
		strings.TrimSpace(
			visualPrompt,
		)
	aociText =
		strings.TrimSpace(
			aociText,
		)

	if visualPrompt == "" ||
		aociText == "" {
		return nil,
			fmt.Errorf(
				"漫画格提示词和IAOCI不能为空",
			)
	}

	updated, err :=
		scanCoursewareComicPanel(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels panel
SET visual_prompt = $1,
    negative_prompt = $2,
    aoci_text = $3,
    relations_json = $4::jsonb,
    status = CASE
        WHEN panel.status = $5 THEN $6
        ELSE panel.status
    END,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE panel.id = $7
  AND panel.project_id = $8
  AND panel.version = $9
  AND panel.status IN ($10, $11, $12, $13)
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $14
        AND project.created_by = $15
        AND project.status IN ($16, $17, $18, $19)
  )
RETURNING `+coursewareComicPanelSelectColumns,
				visualPrompt,
				strings.TrimSpace(negativePrompt),
				aociText,
				relationsJSON,
				models.CWComicPanelStatusGenerated,
				models.CWComicPanelStatusStale,
				strings.TrimSpace(panelID),
				strings.TrimSpace(projectID),
				expectedVersion,
				models.CWComicPanelStatusPlanned,
				models.CWComicPanelStatusGenerated,
				models.CWComicPanelStatusFailed,
				models.CWComicPanelStatusStale,
				strings.TrimSpace(coursewareID),
				strings.TrimSpace(userID),
				models.CWComicProjectStatusPlanned,
				models.CWComicProjectStatusReady,
				models.CWComicProjectStatusInserted,
				models.CWComicProjectStatusFailed,
			),
		)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"保存漫画格提示词失败: %w",
				err,
			)
	}

	return updated, nil
}
