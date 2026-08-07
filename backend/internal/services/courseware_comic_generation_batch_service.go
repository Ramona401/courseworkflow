package services

// courseware_comic_generation_batch_service.go — 漫画整批四路并发生产
//
// 本文件由原超长生成主服务拆分而来，负责：
//   - 人物设定图完成后的分格生产；
//   - 每个波次最多并发4个图片任务；
//   - 部署排空时不再派发新波次，已启动任务继续安全收敛；
//   - 每格继续使用独立版本、独立计费幂等键和不可变历史；
//   - 同一波次共享最近已完成的前序图片作为视觉锚点；
//   - 单格恢复、工作流校验和项目最终状态收敛。
//
// 图片供应商调用在锁外并发执行；数据库成功与失败事务均按
// courseware_comic_projects → courseware_comic_panels 顺序加锁。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const coursewareComicGenerationConcurrency = 4

type coursewareComicPanelGenerationResult struct {
	PanelNo int
	Asset   *models.CoursewareAsset
	Err     error
}

// runProjectGeneration 执行人物设定图和漫画格最多四路并发生成。
func (s *CoursewareComicGenerationService) runProjectGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
	stopDispatch *atomic.Bool,
) error {
	coursewareService := s.coursewareService
	if coursewareService == nil {
		coursewareService = NewCoursewareService()
	}

	courseware, scopedActor, err := coursewareService.LoadCoursewareForOwnerRuntime(
		ctx,
		coursewareID,
		actor,
	)
	if err != nil {
		return err
	}

	project, err := repository.GetCoursewareComicProjectByIDForUser(
		ctx,
		coursewareID,
		projectID,
		scopedActor.UserID,
	)
	if err != nil {
		return err
	}

	workflow, err := repository.GetCoursewareComicWorkflowState(
		ctx,
		coursewareID,
		projectID,
		scopedActor.UserID,
	)
	if err != nil {
		return err
	}

	if err := validateCoursewareComicBatchGenerationWorkerState(
		project,
		workflow,
	); err != nil {
		return err
	}

	if err := validateCoursewareComicVisualStyleSourceRuntime(
		courseware,
		workflow,
	); err != nil {
		return err
	}

	panels, err := repository.ListCoursewareComicPanels(
		ctx,
		coursewareID,
		projectID,
		scopedActor.UserID,
	)
	if err != nil {
		return err
	}

	if len(panels) != project.PanelCount {
		return fmt.Errorf("漫画分格数量与项目规划不一致")
	}

	imageConfig, traceContext, err := s.loadComicImageRuntime(
		ctx,
		scopedActor.UserID,
	)
	if err != nil {
		return err
	}

	s.broadcastGeneration(
		coursewareID,
		"project_start",
		map[string]interface{}{
			"project_id":          projectID,
			"panel_total":         len(panels),
			"concurrency":         coursewareComicGenerationConcurrency,
			"visual_style_source": workflow.VisualStyleSource,
			"aspect_ratio":        workflow.AspectRatio,
			"image_quality":       workflow.ImageQuality,
			"message":             "开始生成人物设定图，并按每波最多4格并发生成漫画图片",
		},
	)

	characterSheet, sheetErr := s.ensureCharacterSheet(
		ctx,
		courseware,
		project,
		workflow,
		imageConfig,
		traceContext,
	)
	if sheetErr != nil {
		hasUsableSheet := characterSheet != nil &&
			resolveAssetPublicURL(characterSheet) != ""

		if workflow.VisualStyleSource == models.CWComicVisualStyleSourceCourseware &&
			!hasUsableSheet {
			return fmt.Errorf(
				"跟随课件画风时人物设定图必须从有效课件风格锚点生成: %w",
				sheetErr,
			)
		}

		coursewareComicGenerationLog.Warn(
			"漫画人物设定图生成或绑定未完全成功",
			"courseware_id",
			coursewareID,
			"project_id",
			projectID,
			"visual_style_source",
			workflow.VisualStyleSource,
			"has_recoverable_asset",
			hasUsableSheet,
			"error",
			sheetErr,
		)

		s.broadcastGeneration(
			coursewareID,
			"character_sheet_warning",
			map[string]interface{}{
				"project_id": projectID,
				"message": "人物设定图未完全绑定；" +
					"使用老师所选画风时可按同一预设画风和人物文字设定继续生成，" +
					"跟随课件画风则必须存在可恢复的课件锚点派生设定图",
			},
		)
	}

	assetsByPanelNo := make(map[int]*models.CoursewareAsset)
	pendingPanels := make([]*models.CoursewareComicPanel, 0, len(panels))

	for _, panel := range panels {
		if panel == nil {
			return fmt.Errorf("漫画分格数据为空")
		}

		if panel.Status == models.CWComicPanelStatusGenerated &&
			panel.CurrentAssetID != nil {
			existing, loadErr := s.loadComicImageAsset(
				ctx,
				coursewareID,
				*panel.CurrentAssetID,
			)
			if loadErr != nil {
				return fmt.Errorf(
					"第%d格已生成图片资产不可用: %w",
					panel.PanelNo,
					loadErr,
				)
			}

			assetsByPanelNo[panel.PanelNo] = existing
			continue
		}

		pendingPanels = append(pendingPanels, panel)
	}

	generationErrors := make([]error, 0)

	for waveStart := 0; waveStart < len(pendingPanels); waveStart +=
		coursewareComicGenerationConcurrency {
		if stopDispatch != nil && stopDispatch.Load() {
			return fmt.Errorf("服务正在部署排空，已停止派发后续漫画格")
		}

		waveEnd := waveStart + coursewareComicGenerationConcurrency
		if waveEnd > len(pendingPanels) {
			waveEnd = len(pendingPanels)
		}

		wave := pendingPanels[waveStart:waveEnd]
		results := make(chan coursewareComicPanelGenerationResult, len(wave))
		var waitGroup sync.WaitGroup

		for _, panel := range wave {
			panel := panel
			previousAsset := resolveCoursewareComicWaveReferenceAsset(
				panel.PanelNo,
				assetsByPanelNo,
			)

			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()

				claimed := panel
				if panel.Status != models.CWComicPanelStatusGenerating {
					claimedPanel, claimErr := repository.ClaimCoursewareComicPanelGeneration(
						ctx,
						coursewareID,
						projectID,
						panel.ID,
						scopedActor.UserID,
						panel.Version,
					)
					if claimErr != nil {
						results <- coursewareComicPanelGenerationResult{
							PanelNo: panel.PanelNo,
							Err:     claimErr,
						}
						return
					}
					claimed = claimedPanel
				}

				generationSource := models.CWComicVersionSourceInitial
				if claimed.CurrentAssetID != nil {
					generationSource = models.CWComicVersionSourceRegenerate
				}

				coursewareSnapshot := *courseware
				projectSnapshot := *project
				workflowSnapshot := *workflow
				imageConfigSnapshot := *imageConfig
				traceContextSnapshot := *traceContext

				generatedAsset, generationErr := s.generateAndCompletePanel(
					ctx,
					&coursewareSnapshot,
					&projectSnapshot,
					&workflowSnapshot,
					claimed,
					characterSheet,
					previousAsset,
					&imageConfigSnapshot,
					&traceContextSnapshot,
					scopedActor.UserID,
					generationSource,
					"",
				)

				if generationErr != nil && generatedAsset == nil {
					failErr := repository.FailCoursewareComicPanelGenerationProjectFirst(
						ctx,
						coursewareID,
						projectID,
						claimed.ID,
						scopedActor.UserID,
						generationErr,
					)
					if failErr != nil {
						generationErr = errors.Join(generationErr, failErr)
					}
				}

				if generationErr != nil && generatedAsset != nil {
					coursewareComicGenerationLog.Warn(
						"漫画格图片资产已形成但版本绑定失败，保持generating供下次恢复",
						"courseware_id",
						coursewareID,
						"project_id",
						projectID,
						"panel_id",
						claimed.ID,
						"panel_version",
						claimed.Version,
						"asset_id",
						generatedAsset.ID,
						"error",
						generationErr,
					)
				}

				results <- coursewareComicPanelGenerationResult{
					PanelNo: claimed.PanelNo,
					Asset:   generatedAsset,
					Err:     generationErr,
				}
			}()
		}

		waitGroup.Wait()
		close(results)

		for result := range results {
			if result.Err != nil {
				generationErrors = append(
					generationErrors,
					fmt.Errorf("第%d格生成失败: %w", result.PanelNo, result.Err),
				)
				continue
			}

			if result.Asset != nil {
				assetsByPanelNo[result.PanelNo] = result.Asset
			}
		}
	}

	if len(generationErrors) > 0 {
		return fmt.Errorf(
			"漫画四路并发生成有%d格未完成: %w",
			len(generationErrors),
			errors.Join(generationErrors...),
		)
	}

	if err := s.finalizeCoursewareComicProjectGeneration(
		ctx,
		coursewareID,
		projectID,
		scopedActor.UserID,
	); err != nil {
		return err
	}

	s.broadcastGeneration(
		coursewareID,
		"project_done",
		map[string]interface{}{
			"project_id":  projectID,
			"panel_total": len(panels),
			"concurrency": coursewareComicGenerationConcurrency,
			"message":     "知识点漫画全部图片已完成四路并发生成，可以编辑文字或插入课件",
		},
	)

	return nil
}

// resolveCoursewareComicWaveReferenceAsset
// 为同一并发波次选择最近已经完成的前序漫画格作为视觉锚点。
//
// 同波次内尚未完成的图片不会互相等待，避免重新退化为串行；
// 找不到前序已完成图片时返回nil，由生成函数回退到人物设定图。
func resolveCoursewareComicWaveReferenceAsset(
	panelNo int,
	assetsByPanelNo map[int]*models.CoursewareAsset,
) *models.CoursewareAsset {
	for candidateNo := panelNo - 1; candidateNo >= 1; candidateNo-- {
		if asset := assetsByPanelNo[candidateNo]; asset != nil {
			return asset
		}
	}

	return nil
}

// runPanelRegeneration 执行一个漫画格的重新生成或同键恢复。
//
// 第一个返回值表示图片供应商成本和课程资产是否已经形成。
// 形成后即使漫画格事务失败，也不得把分格推进为failed。
func (s *CoursewareComicGenerationService) runPanelRegeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panel *models.CoursewareComicPanel,
	regenerationInstruction string,
	actor *CoursewareActorContext,
) (bool, error) {
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
		return false, err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return false, err
	}

	workflow, err :=
		repository.GetCoursewareComicWorkflowState(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return false, err
	}

	if err :=
		validateCoursewareComicVisualStyleSourceRuntime(
			courseware,
			workflow,
		); err != nil {
		return false, err
	}

	panels, err :=
		repository.ListCoursewareComicPanels(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return false, err
	}

	imageConfig, traceContext, err :=
		s.loadComicImageRuntime(
			ctx,
			scopedActor.UserID,
		)
	if err != nil {
		return false, err
	}

	characterSheet :=
		s.loadProjectCharacterSheet(
			ctx,
			coursewareID,
			project,
		)

	if characterSheet == nil {
		characterSheet, err =
			s.ensureCharacterSheet(
				ctx,
				courseware,
				project,
				workflow,
				imageConfig,
				traceContext,
			)

		if err != nil &&
			workflow.VisualStyleSource ==
				models.CWComicVisualStyleSourceCourseware {
			return characterSheet != nil,
				err
		}
	}

	previousAsset :=
		s.findPreviousPanelAsset(
			ctx,
			coursewareID,
			panels,
			panel.PanelNo,
		)

	generatedAsset, err :=
		s.generateAndCompletePanel(
			ctx,
			courseware,
			project,
			workflow,
			panel,
			characterSheet,
			previousAsset,
			imageConfig,
			traceContext,
			scopedActor.UserID,
			models.CWComicVersionSourceRegenerate,
			regenerationInstruction,
		)

	return generatedAsset != nil, err
}

func validateCoursewareComicBatchGenerationStart(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
	expectedVersion int,
) error {
	if project == nil ||
		workflow == nil ||
		expectedVersion < 1 {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	if project.Version !=
		expectedVersion {
		return repository.
			ErrCoursewareComicProjectConflict
	}

	switch project.Status {
	case models.CWComicProjectStatusPlanned,
		models.CWComicProjectStatusFailed,
		models.CWComicProjectStatusGenerating:

	default:
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if workflow.ProjectID !=
		project.ID ||
		workflow.Stage !=
			models.CWComicWorkflowBatchGeneration ||
		workflow.StoryboardConfirmedAt == nil ||
		workflow.StyleConfirmedAt == nil ||
		workflow.StylePreviewPanelID == nil ||
		strings.TrimSpace(
			*workflow.StylePreviewPanelID,
		) == "" {
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if !models.IsValidCWComicVisualStyleSource(
		workflow.VisualStyleSource,
	) ||
		!models.IsValidCWComicVisualStyle(
			project.VisualStyle,
		) ||
		!models.IsValidCWComicAspectRatio(
			workflow.AspectRatio,
		) ||
		!models.IsValidCWComicImageQuality(
			workflow.ImageQuality,
		) ||
		utf8.RuneCountInString(
			strings.TrimSpace(
				workflow.StyleInstruction,
			),
		) >
			models.CoursewareComicMaxStyleInstructionRunes {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	return nil
}

func validateCoursewareComicBatchGenerationWorkerState(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
) error {
	if project == nil ||
		workflow == nil {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	if project.Status !=
		models.CWComicProjectStatusGenerating {
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if workflow.ProjectID !=
		project.ID ||
		workflow.Stage !=
			models.CWComicWorkflowBatchGeneration ||
		workflow.StoryboardConfirmedAt == nil ||
		workflow.StyleConfirmedAt == nil ||
		workflow.StylePreviewPanelID == nil ||
		strings.TrimSpace(
			*workflow.StylePreviewPanelID,
		) == "" {
		return repository.
			ErrCoursewareComicProjectNotEditable
	}

	if !models.IsValidCWComicVisualStyleSource(
		workflow.VisualStyleSource,
	) ||
		!models.IsValidCWComicVisualStyle(
			project.VisualStyle,
		) ||
		!models.IsValidCWComicAspectRatio(
			workflow.AspectRatio,
		) ||
		!models.IsValidCWComicImageQuality(
			workflow.ImageQuality,
		) {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	return nil
}

func validateCoursewareComicVisualStyleSourceRuntime(
	courseware *models.Courseware,
	workflow *models.CoursewareComicWorkflowState,
) error {
	if courseware == nil ||
		workflow == nil ||
		!models.IsValidCWComicVisualStyleSource(
			workflow.VisualStyleSource,
		) {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	if workflow.VisualStyleSource ==
		models.CWComicVisualStyleSourceSelected {
		return nil
	}

	if courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(
			*courseware.StyleAnchorAssetID,
		) == "" {
		return fmt.Errorf(
			"%w: 已选择跟随课件画风，但课件尚未设置风格锚点",
			ErrCoursewareComicWorkflowInvalidRequest,
		)
	}

	return nil
}

// finalizeCoursewareComicProjectGeneration 处理恢复生成时全部格已存在的情况。
func (s *CoursewareComicGenerationService) finalizeCoursewareComicProjectGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) error {
	latest, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			coursewareID,
			projectID,
			userID,
		)

	if err != nil {
		return err
	}

	if latest.Status ==
		models.CWComicProjectStatusReady {
		return nil
	}

	if latest.Status !=
		models.CWComicProjectStatusGenerating {
		return fmt.Errorf(
			"漫画图片已完成，但项目状态异常: %s",
			latest.Status,
		)
	}

	_, err =
		repository.TransitionCoursewareComicProjectStatus(
			ctx,
			coursewareID,
			projectID,
			userID,
			[]string{
				models.CWComicProjectStatusGenerating,
			},
			models.CWComicProjectStatusReady,
			"",
		)

	return err
}
