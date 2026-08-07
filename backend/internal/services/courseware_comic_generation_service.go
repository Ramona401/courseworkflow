package services

// courseware_comic_generation_service.go — 漫画图片生产主编排
//
// 本文件负责：
//   - 创建图片生产服务；
//   - 登记整批生成和单格重画后台任务；
//   - 重新执行课件作者与状态校验；
//   - 只允许已确认首格样张的项目启动整批生成；
//   - 人物设定图完成后按稳定波次最多4路并发生成漫画格；
//   - 部署排空时停止派发新的漫画格；
//   - 使用数据库状态和统一图片计费记录恢复中断任务；
//   - 完成项目最终状态收敛。
//
// 图片下载、课程级资产保存、参考图和图片计费位于：
// courseware_comic_generation_asset.go
//
// 提示词、任务错误映射和SSE广播位于：
// courseware_comic_generation_helper.go

import (
	"context"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// CoursewareComicGenerationStartResult 是异步任务启动响应。
type CoursewareComicGenerationStartResult struct {
	Status       string `json:"status"`
	CoursewareID string `json:"courseware_id"`
	ProjectID    string `json:"project_id"`
	PanelID      string `json:"panel_id,omitempty"`
	Message      string `json:"message"`
}

// CoursewareComicGenerationService 漫画图片生产服务。
type CoursewareComicGenerationService struct {
	cfg               *config.Config
	coursewareService *CoursewareService
	ossService        *OSSService
}

// NewCoursewareComicGenerationService 创建漫画图片生产服务。
func NewCoursewareComicGenerationService(
	cfg *config.Config,
	coursewareService *CoursewareService,
	ossService *OSSService,
) *CoursewareComicGenerationService {
	if coursewareService == nil {
		coursewareService =
			NewCoursewareService()
	}

	if ossService == nil {
		ossService =
			NewOSSService(
				cfg,
			)
	}

	return &CoursewareComicGenerationService{
		cfg:               cfg,
		coursewareService: coursewareService,
		ossService:        ossService,
	}
}

// StartProjectGeneration 登记并启动整批最多4路并发生成或中断恢复。
//
// 已确认首格样张并进入batch_generation步骤的项目才能启动。
// planned、failed以及进程重启后遗留的generating项目均可被领取。
// 同一进程内的重复请求仍由GlobalBackgroundTasks任务键阻止。
func (s *CoursewareComicGenerationService) StartProjectGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	expectedVersion int,
	actor *CoursewareActorContext,
) (*CoursewareComicGenerationStartResult, error) {
	if s == nil ||
		s.cfg == nil ||
		expectedVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

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

	if err :=
		validateCoursewareComicBatchGenerationStart(
			project,
			workflow,
			expectedVersion,
		); err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareComicVisualStyleSourceRuntime(
			courseware,
			workflow,
		); err != nil {
		return nil, err
	}

	asyncActor :=
		CloneCoursewareActorContext(
			scopedActor,
		)

	if asyncActor == nil {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	stopDispatch :=
		&atomic.Bool{}

	task, startResult :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewareComicGenerate,
			project.ID,
			BackgroundTaskCritical,
			func() {
				stopDispatch.Store(
					true,
				)
			},
		)

	if err :=
		mapCoursewareComicTaskStartResult(
			startResult,
		); err != nil {
		return nil, err
	}

	startedProject, err :=
		repository.BeginCoursewareComicProjectGeneration(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
			expectedVersion,
		)
	if err != nil {
		task.Done()
		return nil, err
	}

	go func() {
		runErr :=
			task.Run(
				func() error {
					return s.runProjectGeneration(
						context.Background(),
						courseware.ID,
						startedProject.ID,
						asyncActor,
						stopDispatch,
					)
				},
			)

		if runErr == nil {
			return
		}

		s.markProjectGenerationFailed(
			courseware.ID,
			startedProject.ID,
			asyncActor.UserID,
			runErr,
		)

		s.broadcastGeneration(
			courseware.ID,
			"project_failed",
			map[string]interface{}{
				"project_id": startedProject.ID,
				"message":    "知识点漫画图片生成未完成，已保留成功图片，可以稍后继续生成",
			},
		)
	}()

	return &CoursewareComicGenerationStartResult{
		Status: string(
			startResult,
		),
		CoursewareID: courseware.ID,
		ProjectID:    project.ID,
		Message:      "已按确认样张进入后台最多4路并发生成或断点恢复其余漫画图片",
	}, nil
}

// StartPanelRegeneration 登记并启动教师主动单格重新生成或中断恢复。
//
// 浏览器入口必须提交明确的画面微调要求。
// 整批生成内部对已有图片分格的恢复不经过本入口，
// 允许继续使用基础渲染计划而不附加教师本次微调文字。
func (s *CoursewareComicGenerationService) StartPanelRegeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	expectedPanelVersion int,
	regenerationInstruction string,
	actor *CoursewareActorContext,
) (*CoursewareComicGenerationStartResult, error) {
	if s == nil ||
		s.cfg == nil ||
		expectedPanelVersion < 1 {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

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

	regenerationInstruction =
		strings.TrimSpace(
			regenerationInstruction,
		)

	if regenerationInstruction == "" {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	if utf8.RuneCountInString(
		regenerationInstruction,
	) >
		coursewareComicPanelRegenerationInstructionMaxRunes {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
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

	if err :=
		validateCoursewareComicVisualStyleSourceRuntime(
			courseware,
			workflow,
		); err != nil {
		return nil, err
	}

	panel, err :=
		repository.GetCoursewareComicPanelByIDForProject(
			ctx,
			courseware.ID,
			projectID,
			panelID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if panel.Version !=
		expectedPanelVersion {
		return nil,
			repository.ErrCoursewareComicPanelConflict
	}

	asyncActor :=
		CloneCoursewareActorContext(
			scopedActor,
		)

	if asyncActor == nil {
		return nil,
			ErrCoursewareComicProjectInvalidRequest
	}

	regenerationInstructionSnapshot :=
		regenerationInstruction

	task, startResult :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewareComicPanelRegenerate,
			panel.ID,
			BackgroundTaskCritical,
			nil,
		)

	if err :=
		mapCoursewareComicTaskStartResult(
			startResult,
		); err != nil {
		return nil, err
	}

	claimedPanel :=
		panel

	// generating表示上一进程可能在供应商成功后、业务绑定完成前退出。
	// 此时保留原version，直接执行同一计费幂等键恢复，不能再次领取并增加version。
	if panel.Status !=
		models.CWComicPanelStatusGenerating {
		claimedPanel, err =
			repository.ClaimCoursewareComicPanelRegeneration(
				ctx,
				courseware.ID,
				projectID,
				panel.ID,
				scopedActor.UserID,
				expectedPanelVersion,
			)

		if err != nil {
			task.Done()
			return nil, err
		}
	}

	go func() {
		assetPersisted := false

		runErr :=
			task.Run(
				func() error {
					var workerErr error

					assetPersisted,
						workerErr =
						s.runPanelRegeneration(
							context.Background(),
							courseware.ID,
							projectID,
							claimedPanel,
							regenerationInstructionSnapshot,
							asyncActor,
						)

					return workerErr
				},
			)

		if runErr == nil {
			return
		}

		if !assetPersisted {
			_ =
				repository.FailCoursewareComicPanelGeneration(
					context.Background(),
					courseware.ID,
					projectID,
					claimedPanel.ID,
					asyncActor.UserID,
					runErr,
				)

			s.broadcastGeneration(
				courseware.ID,
				"panel_failed",
				map[string]interface{}{
					"project_id": projectID,
					"panel_id":   claimedPanel.ID,
					"panel_no":   claimedPanel.PanelNo,
					"message":    "该漫画格重新生成失败，可以再次重试",
				},
			)

			return
		}

		// 图片资产和费用已经形成，但业务绑定未完成。
		// 保留generating和原version，下次请求继续恢复同一资产。
		coursewareComicGenerationLog.Warn(
			"漫画格图片资产已形成但业务绑定未完成，保留原任务供恢复",
			"courseware_id",
			courseware.ID,
			"project_id",
			projectID,
			"panel_id",
			claimedPanel.ID,
			"panel_version",
			claimedPanel.Version,
			"error",
			runErr,
		)

		s.broadcastGeneration(
			courseware.ID,
			"panel_recovery_needed",
			map[string]interface{}{
				"project_id": projectID,
				"panel_id":   claimedPanel.ID,
				"panel_no":   claimedPanel.PanelNo,
				"message":    "图片已经生成，但漫画格绑定尚未完成；再次点击重新生成将直接恢复原图片，不会重复扣费",
			},
		)
	}()

	return &CoursewareComicGenerationStartResult{
		Status: string(
			startResult,
		),
		CoursewareID: courseware.ID,
		ProjectID:    projectID,
		PanelID:      panel.ID,
		Message:      "该漫画格已进入后台重新生成或原任务恢复",
	}, nil
}
