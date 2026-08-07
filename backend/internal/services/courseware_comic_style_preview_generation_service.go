package services

// courseware_comic_style_preview_generation_service.go
//
// 本文件负责第三步首格样张后台任务的登记和启动。
//
// 核心边界：
//   - 只允许planned项目在style_preview步骤启动；
//   - 必须已经确认分镜；
//   - 只领取panel_no=1；
//   - 项目和分格版本均通过仓储事务递增；
//   - 真实图片生成使用后台Context；
//   - 失败只收敛首格样张，不把整个项目改为failed。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const backgroundTaskTypeCoursewareComicStylePreview = "courseware_comic_style_preview"

// StartStylePreviewGeneration 启动第三步首格完整样张生成。
func (s *CoursewareComicGenerationService) StartStylePreviewGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	expectedProjectVersion int,
	actor *CoursewareActorContext,
) (*CoursewareComicGenerationStartResult, error) {
	if s == nil ||
		s.cfg == nil ||
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

	if coursewareID == "" ||
		projectID == "" {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	coursewareService :=
		s.coursewareService

	if coursewareService == nil {
		coursewareService =
			NewCoursewareService()
	}

	courseware, scopedActor, err :=
		coursewareService.
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

	panels, err :=
		repository.ListCoursewareComicPanels(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	previewPanel, err :=
		findCoursewareComicStylePreviewPanel(
			panels,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareComicStylePreviewStart(
			project,
			workflow,
			previewPanel,
			expectedProjectVersion,
		); err != nil {
		return nil, err
	}

	asyncActor :=
		CloneCoursewareActorContext(
			scopedActor,
		)

	if asyncActor == nil {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	task, startResult :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewareComicStylePreview,
			project.ID,
			BackgroundTaskCritical,
			nil,
		)

	if err :=
		mapCoursewareComicTaskStartResult(
			startResult,
		); err != nil {
		return nil, err
	}

	claimedPanel, err :=
		repository.ClaimCoursewareComicStylePreview(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
			expectedProjectVersion,
		)
	if err != nil {
		task.Done()
		return nil, err
	}

	go func() {
		runErr :=
			task.Run(
				func() error {
					return s.runStylePreviewGeneration(
						context.Background(),
						courseware.ID,
						project.ID,
						claimedPanel,
						asyncActor,
					)
				},
			)

		if runErr == nil {
			return
		}

		_ =
			repository.FailCoursewareComicStylePreview(
				context.Background(),
				courseware.ID,
				project.ID,
				claimedPanel.ID,
				asyncActor.UserID,
				runErr,
			)

		s.broadcastGeneration(
			courseware.ID,
			"style_preview_failed",
			map[string]interface{}{
				"project_id": project.ID,
				"panel_id":   claimedPanel.ID,
				"panel_no":   claimedPanel.PanelNo,
				"message":    "首格完整样张生成失败，可以调整视觉设置后重新生成",
			},
		)
	}()

	return &CoursewareComicGenerationStartResult{
		Status: string(
			startResult,
		),
		CoursewareID: courseware.ID,
		ProjectID:    project.ID,
		PanelID:      claimedPanel.ID,
		Message:      "首格完整样张已进入后台生成",
	}, nil
}

func findCoursewareComicStylePreviewPanel(
	panels []*models.CoursewareComicPanel,
) (*models.CoursewareComicPanel, error) {
	for _, panel := range panels {
		if panel != nil &&
			panel.PanelNo == 1 {
			return panel, nil
		}
	}

	return nil,
		repository.ErrCoursewareComicPanelNotFound
}

func validateCoursewareComicStylePreviewStart(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
	panel *models.CoursewareComicPanel,
	expectedProjectVersion int,
) error {
	if project == nil ||
		workflow == nil ||
		panel == nil ||
		expectedProjectVersion < 1 {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	if project.Version !=
		expectedProjectVersion {
		return repository.
			ErrCoursewareComicProjectConflict
	}

	if project.Status !=
		models.CWComicProjectStatusPlanned {
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if workflow.ProjectID !=
		project.ID ||
		workflow.Stage !=
			models.CWComicWorkflowStylePreview ||
		workflow.StoryboardConfirmedAt == nil ||
		workflow.StyleConfirmedAt != nil {
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if panel.ProjectID !=
		project.ID ||
		panel.PanelNo != 1 {
		return repository.
			ErrCoursewareComicPanelNotFound
	}

	switch panel.Status {
	case models.CWComicPanelStatusPlanned,
		models.CWComicPanelStatusGenerated,
		models.CWComicPanelStatusFailed,
		models.CWComicPanelStatusStale:

	case models.CWComicPanelStatusGenerating:
		return repository.
			ErrCoursewareComicPanelNotGeneratable

	default:
		return repository.
			ErrCoursewareComicPanelNotGeneratable
	}

	_, valid :=
		buildCoursewareComicStylePreviewRenderPlan(
			project,
			panel,
			workflow,
		)

	if !valid {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	return nil
}
