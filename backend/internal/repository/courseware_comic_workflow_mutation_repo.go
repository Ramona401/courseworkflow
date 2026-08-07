package repository

// courseware_comic_workflow_mutation_repo.go
//
// 本文件负责教师五步工作流中的同步CAS变更：
//   - 确认第二步分镜；
//   - 保存第三步视觉设置。
//
// 所有操作同时限定课件、项目、创建者、版本、项目状态和工作流步骤。
// 浏览器提交的ID、状态和版本不能绕过服务端归属校验。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ConfirmCoursewareComicStoryboard 确认当前AI分镜并进入第三步。
//
// narrativeMode必须与当前项目已规划的叙事方式一致。
// 更换叙事方式必须先重新规划，不能把旧分镜确认成新叙事方式。
func ConfirmCoursewareComicStoryboard(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
	narrativeMode string,
) (*models.CoursewareComicWorkflowState, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	narrativeMode =
		strings.TrimSpace(
			narrativeMode,
		)

	if coursewareID == "" ||
		projectID == "" ||
		userID == "" ||
		expectedVersion < 1 ||
		!models.IsValidCWComicNarrativeMode(
			narrativeMode,
		) {
		return nil,
			fmt.Errorf(
				"确认知识点漫画分镜参数无效",
			)
	}

	state, err :=
		scanCoursewareComicWorkflowState(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects
SET workflow_stage = $1,
    storyboard_confirmed_at = now(),
    style_confirmed_at = NULL,
    style_preview_panel_id = NULL,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND version = $5
  AND status = $6
  AND workflow_stage = $7
  AND narrative_mode = $8
RETURNING `+
					coursewareComicWorkflowSelectColumns,
				models.CWComicWorkflowStylePreview,
				projectID,
				coursewareID,
				userID,
				expectedVersion,
				models.CWComicProjectStatusPlanned,
				models.CWComicWorkflowStoryboard,
				narrativeMode,
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
				"确认知识点漫画分镜失败: %w",
				err,
			)
	}

	return state, nil
}

// UpdateCoursewareComicStyleSettings 保存第三步视觉设置。
//
// 修改画风来源或其他设置会撤销旧样张定位和确认时间。
// visual_style_source只保存courseware或selected，不保存混合模式。
// 人物设定参考图也会解除项目关联，防止旧画风污染新样张。
// 原资产仍保留在课程资产库，不执行物理删除。
//
// 第1格处于generating时拒绝修改，避免旧设置生成的图片
// 被结算成新设置对应的首格样张。
func UpdateCoursewareComicStyleSettings(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
	visualStyleSource string,
	visualStyle string,
	aspectRatio string,
	imageQuality string,
	styleInstruction string,
) (*models.CoursewareComicWorkflowState, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	visualStyleSource =
		strings.TrimSpace(
			visualStyleSource,
		)

	visualStyle =
		strings.TrimSpace(
			visualStyle,
		)

	aspectRatio =
		strings.TrimSpace(
			aspectRatio,
		)

	imageQuality =
		strings.TrimSpace(
			imageQuality,
		)

	styleInstruction =
		strings.TrimSpace(
			styleInstruction,
		)

	if coursewareID == "" ||
		projectID == "" ||
		userID == "" ||
		expectedVersion < 1 ||
		!models.IsValidCWComicVisualStyleSource(
			visualStyleSource,
		) ||
		!models.IsValidCWComicVisualStyle(
			visualStyle,
		) ||
		!models.IsValidCWComicAspectRatio(
			aspectRatio,
		) ||
		!models.IsValidCWComicImageQuality(
			imageQuality,
		) ||
		utf8.RuneCountInString(
			styleInstruction,
		) >
			models.CoursewareComicMaxStyleInstructionRunes {
		return nil,
			fmt.Errorf(
				"保存知识点漫画视觉设置参数无效",
			)
	}

	state, err :=
		scanCoursewareComicWorkflowState(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects project
SET visual_style_source = $1,
    visual_style = $2,
    aspect_ratio = $3,
    image_quality = $4,
    style_instruction = $5,
    style_confirmed_at = NULL,
    style_preview_panel_id = NULL,
    character_sheet_asset_id = NULL,
    workflow_stage = $6,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE project.id = $7
  AND project.courseware_id = $8
  AND project.created_by = $9
  AND project.version = $10
  AND project.status = $11
  AND project.workflow_stage = $12
  AND project.storyboard_confirmed_at IS NOT NULL
  AND NOT EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      WHERE panel.project_id = project.id
        AND panel.panel_no = 1
        AND panel.status = $13
  )
RETURNING `+
					coursewareComicWorkflowSelectColumns,
				visualStyleSource,
				visualStyle,
				aspectRatio,
				imageQuality,
				styleInstruction,
				models.CWComicWorkflowStylePreview,
				projectID,
				coursewareID,
				userID,
				expectedVersion,
				models.CWComicProjectStatusPlanned,
				models.CWComicWorkflowStylePreview,
				models.CWComicPanelStatusGenerating,
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
				"保存知识点漫画视觉设置失败: %w",
				err,
			)
	}

	return state, nil
}
