package services

// courseware_comic_workflow_mutation_service.go
//
// 本文件负责教师五步工作流中的同步业务操作：
//   - 确认第二步AI分镜；
//   - 保存第三步视觉设置。
//
// 服务层先重新执行课件作者授权和课件可变更状态检查，
// 再读取当前项目与工作流，最后调用仓储CAS操作。
// 仓储CAS仍是并发冲突的最终判定边界。

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCoursewareComicWorkflowInvalidRequest = errors.New(
		"知识点漫画工作流请求无效",
	)

	ErrCoursewareComicNarrativeReplanRequired = errors.New(
		"叙事方式已改变，需要重新生成分镜",
	)

	ErrCoursewareComicStyleInstructionTooLong = errors.New(
		"漫画风格补充要求长度超过上限",
	)
)

type normalizedCoursewareComicStyleSettings struct {
	VisualStyleSource string
	VisualStyle       string
	AspectRatio       string
	ImageQuality      string
	StyleInstruction  string
}

func normalizeCoursewareComicNarrativeConfirmation(
	request *models.ConfirmCoursewareComicStoryboardRequest,
) (string, error) {
	if request == nil ||
		request.ExpectedVersion < 1 {
		return "",
			ErrCoursewareComicWorkflowInvalidRequest
	}

	narrativeMode :=
		strings.TrimSpace(
			request.NarrativeMode,
		)

	if !models.IsValidCWComicNarrativeMode(
		narrativeMode,
	) {
		return "",
			ErrCoursewareComicWorkflowInvalidRequest
	}

	return narrativeMode, nil
}

func normalizeCoursewareComicStyleSettings(
	request *models.UpdateCoursewareComicStyleSettingsRequest,
) (*normalizedCoursewareComicStyleSettings, error) {
	if request == nil ||
		request.ExpectedVersion < 1 {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	normalized :=
		&normalizedCoursewareComicStyleSettings{
			VisualStyleSource: strings.TrimSpace(
				request.VisualStyleSource,
			),
			VisualStyle: strings.TrimSpace(
				request.VisualStyle,
			),
			AspectRatio: strings.TrimSpace(
				request.AspectRatio,
			),
			ImageQuality: strings.TrimSpace(
				request.ImageQuality,
			),
			StyleInstruction: strings.TrimSpace(
				request.StyleInstruction,
			),
		}

	if !models.IsValidCWComicVisualStyleSource(
		normalized.VisualStyleSource,
	) ||
		!models.IsValidCWComicVisualStyle(
			normalized.VisualStyle,
		) ||
		!models.IsValidCWComicAspectRatio(
			normalized.AspectRatio,
		) ||
		!models.IsValidCWComicImageQuality(
			normalized.ImageQuality,
		) {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	if utf8.RuneCountInString(
		normalized.StyleInstruction,
	) >
		models.CoursewareComicMaxStyleInstructionRunes {
		return nil,
			ErrCoursewareComicStyleInstructionTooLong
	}

	return normalized, nil
}

// ConfirmProjectStoryboard 确认当前项目的AI分镜。
//
// 浏览器提交的NarrativeMode必须与数据库中的当前规划一致。
// 老师更换叙事方式后必须重新调用规划接口，
// 不能继续确认旧叙事方式生成的分镜。
func (s *CoursewareComicProjectService) ConfirmProjectStoryboard(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
	request *models.ConfirmCoursewareComicStoryboardRequest,
) (*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil {
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

	narrativeMode, err :=
		normalizeCoursewareComicNarrativeConfirmation(
			request,
		)
	if err != nil {
		return nil, err
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
		request.ExpectedVersion {
		return nil,
			repository.ErrCoursewareComicProjectConflict
	}

	if project.Status !=
		models.CWComicProjectStatusPlanned {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	if strings.TrimSpace(
		project.NarrativeMode,
	) != narrativeMode {
		return nil,
			ErrCoursewareComicNarrativeReplanRequired
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
		models.CWComicWorkflowStoryboard ||
		workflow.StoryboardConfirmedAt != nil {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	updated, err :=
		repository.ConfirmCoursewareComicStoryboard(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
			request.ExpectedVersion,
			narrativeMode,
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicWorkflowView(
		updated,
	)
}

// UpdateProjectStyleSettings 保存第三步视觉设置。
//
// 修改画风来源或任何视觉设置都会撤销旧样张确认和样张分格定位。
// visual_style_source严格二选一，不允许在服务层回退或合并。
// 之后教师需要重新生成并确认首格样张。
func (s *CoursewareComicProjectService) UpdateProjectStyleSettings(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
	request *models.UpdateCoursewareComicStyleSettingsRequest,
) (*models.CoursewareComicWorkflowView, error) {
	if s == nil ||
		actor == nil {
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

	settings, err :=
		normalizeCoursewareComicStyleSettings(
			request,
		)
	if err != nil {
		return nil, err
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
		request.ExpectedVersion {
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
		workflow.StoryboardConfirmedAt == nil {
		return nil,
			repository.ErrCoursewareComicProjectNotEditable
	}

	updated, err :=
		repository.UpdateCoursewareComicStyleSettings(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
			request.ExpectedVersion,
			settings.VisualStyleSource,
			settings.VisualStyle,
			settings.AspectRatio,
			settings.ImageQuality,
			settings.StyleInstruction,
		)
	if err != nil {
		return nil, err
	}

	return buildCoursewareComicWorkflowView(
		updated,
	)
}
