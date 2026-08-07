package repository

// courseware_comic_style_preview_confirm_repo.go
//
// 本文件负责教师确认第三步首格样张。
//
// 确认条件：
//   - 项目版本必须匹配；
//   - 项目status必须为planned；
//   - 工作流必须为style_preview；
//   - 分镜已经确认；
//   - 样张尚未确认；
//   - 提交的样张分格必须等于style_preview_panel_id；
//   - 样张必须是当前项目第1格；
//   - 第1格必须为generated并绑定同课件图片资产。
//
// 确认成功后只推进教师工作流到batch_generation。
// 项目status仍保持planned，等待整批生成接口领取。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ConfirmCoursewareComicStylePreview 确认首格完整样张。
func ConfirmCoursewareComicStylePreview(
	ctx context.Context,
	coursewareID string,
	projectID string,
	previewPanelID string,
	userID string,
	expectedProjectVersion int,
) (*models.CoursewareComicWorkflowState, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	previewPanelID =
		strings.TrimSpace(
			previewPanelID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		previewPanelID == "" ||
		userID == "" ||
		expectedProjectVersion < 1 {
		return nil,
			fmt.Errorf(
				"确认首格样张参数无效",
			)
	}

	state, err :=
		scanCoursewareComicWorkflowState(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects project
SET workflow_stage = $1,
    style_confirmed_at = now(),
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE project.id = $2
  AND project.courseware_id = $3
  AND project.created_by = $4
  AND project.version = $5
  AND project.status = $6
  AND project.workflow_stage = $7
  AND project.storyboard_confirmed_at IS NOT NULL
  AND project.style_confirmed_at IS NULL
  AND project.style_preview_panel_id = $8
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      JOIN courseware_assets asset
        ON asset.id = panel.current_asset_id
      WHERE panel.id = project.style_preview_panel_id
        AND panel.id = $8
        AND panel.project_id = project.id
        AND panel.panel_no = 1
        AND panel.status = $9
        AND panel.current_asset_id IS NOT NULL
        AND asset.courseware_id = project.courseware_id
        AND asset.asset_type = 'image'
  )
RETURNING `+
					coursewareComicWorkflowSelectColumns,
				models.CWComicWorkflowBatchGeneration,
				projectID,
				coursewareID,
				userID,
				expectedProjectVersion,
				models.CWComicProjectStatusPlanned,
				models.CWComicWorkflowStylePreview,
				previewPanelID,
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
				"确认首格完整样张失败: %w",
				err,
			)
	}

	return state, nil
}
