package services

// courseware_comic_style_preview_confirm_service.go
//
// 本文件负责第三步样张确认的业务校验：
//   - 重新授权课件作者；
//   - 校验课件当前允许修改；
//   - 校验项目版本、状态和工作流；
//   - 校验样张ID与项目保存的样张定位一致；
//   - 校验样张是第1格且已经绑定图片；
//   - 调用仓储CAS推进到batch_generation。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ConfirmProjectStylePreview 确认首格完整样张。
func (s *CoursewareComicProjectService) ConfirmProjectStylePreview(
	ctx context.Context,
	coursewareID string,
	projectID string,
	previewPanelID string,
	expectedProjectVersion int,
	actor *CoursewareActorContext,
) (*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil ||
		expectedProjectVersion < 1 {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

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

	if coursewareID == "" ||
		projectID == "" ||
		previewPanelID == "" {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if project.Version !=
		expectedProjectVersion {
		return nil,
			repository.ErrCoursewareComicProjectConflict
	}

	if project.Status !=
		models.CWComicProjectStatusPlanned {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	workflow, err :=
		repository.GetCoursewareComicWorkflowState(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if workflow.Stage !=
		models.CWComicWorkflowStylePreview ||
		workflow.StoryboardConfirmedAt == nil ||
		workflow.StyleConfirmedAt != nil {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	if workflow.StylePreviewPanelID == nil ||
		strings.TrimSpace(
			*workflow.StylePreviewPanelID,
		) != previewPanelID {
		return nil,
			repository.ErrCoursewareComicProjectConflict
	}

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			courseware.ID,
			project.ID,
			previewPanelID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.PanelNo != 1 ||
		panel.Status !=
			models.CWComicPanelStatusGenerated ||
		panel.CurrentAssetID == nil ||
		strings.TrimSpace(
			*panel.CurrentAssetID,
		) == "" {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	state, err :=
		repository.ConfirmCoursewareComicStylePreview(
			ctx,
			courseware.ID,
			project.ID,
			previewPanelID,
			scopedActor.UserID,
			expectedProjectVersion,
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicWorkflowView(
		state,
	)
}
