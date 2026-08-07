package services

// courseware_comic_workflow_service.go — 五步漫画工作流读取服务
//
// 本服务负责：
//   - 通过正式课件作者运行域重新授权；
//   - 返回浏览器安全工作流视图；
//   - 批量加载课件内项目工作流；
//   - 在AI规划成功后进入确认分镜步骤。
//
// 本文件不确认分镜、不生成样张、不启动整批生图。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func buildCoursewareComicWorkflowView(
	state *models.CoursewareComicWorkflowState,
) (*models.CoursewareComicWorkflowView, error) {
	normalized, valid :=
		models.NormalizeCoursewareComicWorkflowState(
			state,
		)
	if !valid ||
		normalized == nil {
		return nil,
			fmt.Errorf(
				"知识点漫画工作流数据无效",
			)
	}

	return &models.CoursewareComicWorkflowView{
		Stage:                 normalized.Stage,
		StoryboardConfirmedAt: normalized.StoryboardConfirmedAt,
		StyleConfirmedAt:      normalized.StyleConfirmedAt,
		StylePreviewPanelID:   normalized.StylePreviewPanelID,
		VisualStyleSource:     normalized.VisualStyleSource,
		AspectRatio:           normalized.AspectRatio,
		ImageQuality:          normalized.ImageQuality,
		InsertionMode:         normalized.InsertionMode,
		StyleInstruction:      normalized.StyleInstruction,
	}, nil
}

// GetProjectWorkflowForBrowser 返回单个项目浏览器安全工作流。
func (s *CoursewareComicProjectService) GetProjectWorkflowForBrowser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil ||
		strings.TrimSpace(
			coursewareID,
		) == "" ||
		strings.TrimSpace(
			projectID,
		) == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	_, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				actor,
			)
	if err != nil {
		return nil, err
	}

	state, err :=
		repository.GetCoursewareComicWorkflowState(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicWorkflowView(
		state,
	)
}

// ListProjectWorkflowsForBrowser 返回课件内项目ID到工作流视图的映射。
func (s *CoursewareComicProjectService) ListProjectWorkflowsForBrowser(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (map[string]*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil ||
		strings.TrimSpace(
			coursewareID,
		) == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	_, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
				actor,
			)
	if err != nil {
		return nil, err
	}

	states, err :=
		repository.ListCoursewareComicWorkflowStates(
			ctx,
			coursewareID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	result :=
		make(
			map[string]*models.CoursewareComicWorkflowView,
			len(states),
		)

	for projectID, state := range states {
		view, viewErr :=
			buildCoursewareComicWorkflowView(
				state,
			)
		if viewErr != nil {
			return nil, viewErr
		}

		result[projectID] =
			view
	}

	return result, nil
}

// AdvanceWorkflowAfterPlanning 在规划成功后进入第二步。
func (s *CoursewareComicProjectService) AdvanceWorkflowAfterPlanning(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
) (*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil ||
		strings.TrimSpace(
			coursewareID,
		) == "" ||
		strings.TrimSpace(
			projectID,
		) == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				strings.TrimSpace(
					coursewareID,
				),
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

	state, err :=
		repository.AdvanceCoursewareComicWorkflowAfterPlanning(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicWorkflowView(
		state,
	)
}
