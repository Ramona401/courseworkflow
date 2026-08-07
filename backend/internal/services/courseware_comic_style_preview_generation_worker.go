package services

// courseware_comic_style_preview_generation_worker.go
//
// 本文件执行第三步首格完整样张的真实图片生产：
//   - 重新加载可信课件、项目和工作流；
//   - 使用第三步比例、清晰度和画风构建提示词；
//   - 只调用一次图片模型；
//   - 保存课程级图片资产；
//   - 使用样张独立事务创建不可变历史并绑定第1格；
//   - 通过SSE广播样张开始和完成事件。
//
// 图片模型只生成无文字视觉底图。
// 浏览器继续使用panel.overlay_document渲染中文对白、旁白和教学卡片。

import (
	"context"
	"fmt"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

func (s *CoursewareComicGenerationService) runStylePreviewGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	claimedPanel *models.CoursewareComicPanel,
	actor *CoursewareActorContext,
) error {
	if s == nil ||
		s.cfg == nil ||
		claimedPanel == nil ||
		actor == nil {
		return ErrCoursewareComicWorkflowInvalidRequest
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
		return err
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			courseware.ID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return err
	}

	workflow, err :=
		repository.GetCoursewareComicWorkflowState(
			ctx,
			courseware.ID,
			project.ID,
			scopedActor.UserID,
		)
	if err != nil {
		return err
	}

	if err :=
		validateCoursewareComicStylePreviewWorkerState(
			project,
			workflow,
			claimedPanel,
		); err != nil {
		return err
	}

	renderPlan, valid :=
		buildCoursewareComicStylePreviewRenderPlan(
			project,
			claimedPanel,
			workflow,
		)

	if !valid ||
		renderPlan == nil {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	imageConfig, traceContext, err :=
		s.loadComicImageRuntime(
			ctx,
			scopedActor.UserID,
		)
	if err != nil {
		return err
	}

	generationSource :=
		coursewareComicStylePreviewGenerationSource(
			claimedPanel,
		)

	s.broadcastGeneration(
		courseware.ID,
		"style_preview_generating",
		map[string]interface{}{
			"project_id":    project.ID,
			"panel_id":      claimedPanel.ID,
			"panel_no":      claimedPanel.PanelNo,
			"image_size":    renderPlan.ImageSize,
			"aspect_ratio":  workflow.AspectRatio,
			"image_quality": workflow.ImageQuality,
			"message":       "正在生成首格完整样张",
		},
	)

	_, asset, err :=
		executeBilledCoursewareImage(
			ctx,
			&coursewareImageBillingInput{
				UserID:          scopedActor.UserID,
				SchoolID:        traceContext.SchoolID,
				BillingNodeCode: "comic_style_preview",
				CoursewareID:    courseware.ID,
				ModelName:       imageConfig.Model,
				IdempotencyKey: fmt.Sprintf(
					"courseware-image:comic-style-preview:%s:v%d",
					claimedPanel.ID,
					claimedPanel.Version,
				),
				Metadata: map[string]interface{}{
					"comic_project_id":     project.ID,
					"comic_panel_id":       claimedPanel.ID,
					"panel_number":         claimedPanel.PanelNo,
					"panel_version":        claimedPanel.Version,
					"generation_source":    generationSource,
					"aspect_ratio":         workflow.AspectRatio,
					"image_quality":        workflow.ImageQuality,
					"requested_image_size": renderPlan.ImageSize,
				},
			},
			func() (*ai.ImageGenerateResult, error) {
				return ai.GenerateImage(
					ctx,
					imageConfig,
					renderPlan.Prompt,
					renderPlan.ImageSize,
					1,
					"",
					traceContext,
				)
			},
			func(
				generated *ai.ImageGenerateResult,
			) (*models.CoursewareAsset, error) {
				return s.saveComicGeneratedImage(
					ctx,
					courseware.ID,
					project.ID,
					claimedPanel,
					"style_preview",
					generated.URLs[0],
					renderPlan.Prompt,
					generationSource,
					nil,
				)
			},
		)
	if err != nil {
		return fmt.Errorf(
			"首格完整样张生成或资产保存失败: %w",
			err,
		)
	}

	updatedPanel, err :=
		repository.CompleteCoursewareComicStylePreview(
			ctx,
			courseware.ID,
			project.ID,
			claimedPanel.ID,
			scopedActor.UserID,
			asset.ID,
			claimedPanel.AOCIText,
			generationSource,
		)
	if err != nil {
		// 图片调用已经产生费用，课程资产不删除。
		return err
	}

	s.broadcastGeneration(
		courseware.ID,
		"style_preview_done",
		map[string]interface{}{
			"project_id":    project.ID,
			"panel_id":      updatedPanel.ID,
			"panel_no":      updatedPanel.PanelNo,
			"panel_version": updatedPanel.Version,
			"asset_id":      asset.ID,
			"asset_url":     asset.OssURL,
			"public_url": resolveAssetPublicURL(
				asset,
			),
			"aspect_ratio":  workflow.AspectRatio,
			"image_quality": workflow.ImageQuality,
			"message":       "首格完整样张已生成，请确认画风后继续",
		},
	)

	return nil
}

func validateCoursewareComicStylePreviewWorkerState(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
	panel *models.CoursewareComicPanel,
) error {
	if project == nil ||
		workflow == nil ||
		panel == nil {
		return ErrCoursewareComicWorkflowInvalidRequest
	}

	if project.Status !=
		models.CWComicProjectStatusPlanned ||
		workflow.ProjectID !=
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

	if panel.Status !=
		models.CWComicPanelStatusGenerating {
		return repository.
			ErrCoursewareComicPanelConflict
	}

	return nil
}

func coursewareComicStylePreviewGenerationSource(
	panel *models.CoursewareComicPanel,
) string {
	if panel != nil &&
		panel.CurrentAssetID != nil &&
		*panel.CurrentAssetID != "" {
		return models.
			CWComicVersionSourceRegenerate
	}

	return models.
		CWComicVersionSourceInitial
}
