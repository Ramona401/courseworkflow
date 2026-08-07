package services

// courseware_auto_assembly_image_iaoci.go — 自动装配IAOCI逐槽位图片链
//
// 本链替代旧的：
//   取第一条建议 → 生成一张主图 → 用主图填满其它占位。
//
// 当前行为：
//   - 每个槽位独立规划和生成；
//   - 每张图精确填入自己的placeholder_id；
//   - 显式R关系按需选择一张历史参考图；
//   - 无媒体需求时不生成图片；
//   - 单槽位失败只隐藏该槽位；
//   - 不调用RefinePage，不重写整页HTML。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// assembleOnePageMediaIAOCI 单页IAOCI配图和视频占位入口。
func (s *CoursewareAutoAssemblyService) assembleOnePageMediaIAOCI(
	ctx context.Context,
	pageContext *cwAssemblyPageContext,
	page *models.CoursewarePage,
) cwAssemblyPageResult {
	result := cwAssemblyPageResult{
		pageNum: page.PageNumber,
		pageID:  page.ID,
		title:   page.Title,
		htmlOK:  true,
	}

	s.assemblePageImagesIAOCI(
		ctx,
		pageContext,
		page,
		&result,
	)

	if !pageContext.skipVideo &&
		s.pageNeedsVideo(page) {
		s.assembleVideoPlaceholder(
			ctx,
			pageContext,
			page,
			&result,
		)
	} else {
		result.videoSkipped = true
	}

	return result
}

// assemblePageImagesIAOCI 逐槽位生成并精确填图。
func (s *CoursewareAutoAssemblyService) assemblePageImagesIAOCI(
	ctx context.Context,
	pageContext *cwAssemblyPageContext,
	page *models.CoursewarePage,
	result *cwAssemblyPageResult,
) {
	normalizedHTML, slots, changed :=
		cwEnsureImagePlaceholderIDs(
			page.HTMLContent,
		)

	// 页面方案明确不需要图片时，即使HTML误生成占位，也只隐藏不生图。
	if strings.TrimSpace(
		page.MediaRequirements,
	) == "" {
		currentHTML := normalizedHTML

		for _, slot := range slots {
			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					slot.PlaceholderID,
					"not_required",
				)
		}

		if changed ||
			currentHTML != page.HTMLContent {
			if err := repository.UpdateCWPageHTML(
				ctx,
				page.ID,
				currentHTML,
				page.PlaceholderMap,
				page.MatchedComponentIDs,
				page.Status,
			); err != nil {
				cwAssemblyLog.Warn(
					"隐藏无媒体需求占位失败",
					"page", page.PageNumber,
					"error", err,
				)
			} else {
				page.HTMLContent = currentHTML
			}
		}

		_ = repository.MarkPageImageIndexesStaleExcept(
			ctx,
			page.ID,
			nil,
		)

		result.imageSkipped = true

		cwAssemblyLog.Info(
			"跳过无图片需求页",
			"page", page.PageNumber,
			"title", page.Title,
			"hidden_slots", len(slots),
		)

		return
	}

	GlobalCWSSEHub.Broadcast(
		pageContext.coursewareID,
		CWSSEEvent{
			EventType: "assembly_page_image",
			Data: map[string]interface{}{
				"page_number": page.PageNumber,
				"stage":       "image_iaoci_plan",
				"message": fmt.Sprintf(
					"第 %d 页：正在为每个图片槽位建立IAOCI...",
					page.PageNumber,
				),
			},
		},
	)

	plans, err :=
		s.assetService.PlanPageImagesIAOCI(
			ctx,
			pageContext.coursewareID,
			page.PageNumber,
			pageContext.actor,
		)
	if err != nil {
		result.imageOK = false
		result.errMsg = fmt.Sprintf(
			"第%d页图片IAOCI规划失败: %v",
			page.PageNumber,
			err,
		)

		currentHTML := page.HTMLContent

		// 规划失败时隐藏所有真实占位，避免交付裂框。
		for _, slot := range slots {
			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					slot.PlaceholderID,
					"plan_failed",
				)
		}

		if currentHTML != page.HTMLContent {
			_ = s.persistIAOCIAssemblyHTML(
				ctx,
				page,
				currentHTML,
			)
		}

		cwAssemblyLog.Warn(
			"图片IAOCI规划失败",
			"page", page.PageNumber,
			"error", err,
		)

		return
	}

	if len(plans) == 0 {
		result.imageSkipped = true
		return
	}

	freshPage, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			pageContext.coursewareID,
			page.PageNumber,
		)
	if err != nil ||
		freshPage == nil ||
		strings.TrimSpace(
			freshPage.HTMLContent,
		) == "" {
		result.imageOK = false
		result.errMsg = fmt.Sprintf(
			"第%d页读取稳定占位HTML失败",
			page.PageNumber,
		)

		return
	}

	currentHTML := freshPage.HTMLContent
	successCount := 0
	failureCount := 0
	pendingCount := 0
	failures := make([]string, 0)

	for _, plan := range plans {
		GlobalCWSSEHub.Broadcast(
			pageContext.coursewareID,
			CWSSEEvent{
				EventType: "assembly_page_image",
				Data: map[string]interface{}{
					"page_number":    page.PageNumber,
					"stage":          "image_slot_generate",
					"placeholder_id": plan.PlaceholderID,
					"image_key":      plan.ImageKey,
					"slot_order":     plan.Order,
					"slot_total":     len(plans),
					"message": fmt.Sprintf(
						"第 %d 页：正在生成第 %d/%d 个独立图片槽位...",
						page.PageNumber,
						plan.Order,
						len(plans),
					),
				},
			},
		)

		relationReference :=
			s.resolveImageIAOCIRelationReference(
				ctx,
				pageContext.coursewareID,
				plan.AOCIText,
			)

		imageResponse, generationErr :=
			s.assetService.GenerateImageFromIAOCI(
				ctx,
				&GenerateImageIAOCIRequest{
					CoursewareID:        pageContext.coursewareID,
					PageNumber:          page.PageNumber,
					PlaceholderID:       plan.PlaceholderID,
					ImageKey:            plan.ImageKey,
					Prompt:              plan.Prompt,
					Size:                plan.Size,
					RelationRefImageURL: relationReference,
					Actor:               pageContext.actor,
				},
			)

		if generationErr != nil ||
			imageResponse == nil {
			// 同一幂等键的并发请求仍在生成时，保持原占位和generating状态。
			// 禁止把正在生产的槽位隐藏或标记为失败。
			if errors.Is(
				generationErr,
				ErrCoursewareImageBillingInProgress,
			) {
				pendingCount++

				failures = append(
					failures,
					fmt.Sprintf(
						"%s仍在生成",
						plan.PlaceholderID,
					),
				)

				cwAssemblyLog.Info(
					"IAOCI图片槽位已有同键任务正在生成",
					"page", page.PageNumber,
					"placeholder_id", plan.PlaceholderID,
					"image_key", plan.ImageKey,
				)

				continue
			}

			failureCount++

			failureMessage := fmt.Sprintf(
				"%s生成失败",
				plan.PlaceholderID,
			)

			if generationErr != nil {
				failureMessage =
					generationErr.Error()
			}

			failures = append(
				failures,
				failureMessage,
			)

			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					plan.PlaceholderID,
					"generation_failed",
				)

			if persistErr :=
				s.persistIAOCIAssemblyHTML(
					ctx,
					freshPage,
					currentHTML,
				); persistErr != nil {
				cwAssemblyLog.Warn(
					"隐藏失败图片槽位后写库失败",
					"page", page.PageNumber,
					"placeholder_id",
					plan.PlaceholderID,
					"error", persistErr,
				)
			}

			cwAssemblyLog.Warn(
				"IAOCI图片槽位生成失败",
				"page", page.PageNumber,
				"placeholder_id",
				plan.PlaceholderID,
				"image_key", plan.ImageKey,
				"error", generationErr,
			)

			continue
		}

		publicURL, uploadErr :=
			s.ossService.UploadAssetToOSS(
				imageResponse.URL,
			)

		if uploadErr != nil ||
			strings.TrimSpace(publicURL) == "" {
			publicURL = imageResponse.URL

			cwAssemblyLog.Warn(
				"IAOCI图片上云失败，降级使用本地URL",
				"page", page.PageNumber,
				"placeholder_id",
				plan.PlaceholderID,
				"asset_id",
				imageResponse.AssetID,
				"error", uploadErr,
			)
		} else {
			if updateErr :=
				repository.UpdateCWAssetPublicURL(
					ctx,
					imageResponse.AssetID,
					publicURL,
				); updateErr != nil {
				cwAssemblyLog.Warn(
					"IAOCI图片公网URL回写失败",
					"page", page.PageNumber,
					"asset_id",
					imageResponse.AssetID,
					"error", updateErr,
				)
			}
		}

		filledHTML, filled :=
			cwFillImagePlaceholder(
				currentHTML,
				plan.PlaceholderID,
				publicURL,
				plan.Caption,
			)

		if !filled {
			failureCount++

			failures = append(
				failures,
				fmt.Sprintf(
					"%s未找到对应HTML占位",
					plan.PlaceholderID,
				),
			)

			s.markImageAOCIIndexFailed(
				ctx,
				pageContext.coursewareID,
				plan.ImageKey,
				"生成成功但未找到对应HTML占位",
			)

			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					plan.PlaceholderID,
					"placeholder_missing",
				)

			_ = s.persistIAOCIAssemblyHTML(
				ctx,
				freshPage,
				currentHTML,
			)

			continue
		}

		currentHTML = filledHTML

		if persistErr :=
			s.persistIAOCIAssemblyHTML(
				ctx,
				freshPage,
				currentHTML,
			); persistErr != nil {
			failureCount++

			failures = append(
				failures,
				fmt.Sprintf(
					"%s填图落库失败: %v",
					plan.PlaceholderID,
					persistErr,
				),
			)

			s.markImageAOCIIndexFailed(
				ctx,
				pageContext.coursewareID,
				plan.ImageKey,
				"图片已生成但页面HTML写入失败",
			)

			continue
		}

		successCount++

		cwAssemblyLog.Info(
			"IAOCI图片槽位完成",
			"page", page.PageNumber,
			"placeholder_id",
			plan.PlaceholderID,
			"image_key", plan.ImageKey,
			"asset_id",
			imageResponse.AssetID,
			"relation_reference",
			relationReference != "",
		)
	}

	page.HTMLContent = currentHTML

	if successCount == 0 &&
		failureCount == 0 &&
		pendingCount == 0 {
		result.imageSkipped = true
		return
	}

	result.imageOK =
		failureCount == 0 &&
			pendingCount == 0 &&
			successCount == len(plans)

	if failureCount > 0 ||
		pendingCount > 0 {
		result.errMsg = fmt.Sprintf(
			"第%d页图片槽位完成%d个、处理中%d个、失败%d个：%s",
			page.PageNumber,
			successCount,
			pendingCount,
			failureCount,
			strings.Join(failures, "；"),
		)
	}
}

// resolveImageIAOCIRelationReference 解析第一张可用的显式R参考图。
func (s *CoursewareAutoAssemblyService) resolveImageIAOCIRelationReference(
	ctx context.Context,
	coursewareID string,
	aociText string,
) string {
	imageAOCI, err :=
		utils.ParseImageAOCI(aociText)
	if err != nil {
		return ""
	}

	for _, relation := range imageAOCI.Relations {
		// 仅继承A艺术风格时不使用图片参考，避免污染本图场景。
		if !strings.ContainsAny(
			relation.InheritMask,
			"CSOL",
		) {
			continue
		}

		targetIndex, err :=
			repository.GetCoursewareImageIndexByKey(
				ctx,
				coursewareID,
				relation.TargetImageKey,
			)
		if err != nil ||
			targetIndex == nil ||
			targetIndex.AssetID == nil ||
			strings.TrimSpace(
				*targetIndex.AssetID,
			) == "" {
			continue
		}

		targetAsset, err :=
			repository.GetCWAssetByID(
				ctx,
				*targetIndex.AssetID,
			)
		if err != nil ||
			targetAsset == nil {
			continue
		}

		referenceURL :=
			resolveAssetPublicURL(
				targetAsset,
			)

		if referenceURL != "" {
			return referenceURL
		}
	}

	return ""
}

// persistIAOCIAssemblyHTML 保存每个槽位后的最新HTML。
func (s *CoursewareAutoAssemblyService) persistIAOCIAssemblyHTML(
	ctx context.Context,
	page *models.CoursewarePage,
	pageHTML string,
) error {
	if page == nil {
		return fmt.Errorf("页面对象为空")
	}

	if err := repository.UpdateCWPageHTML(
		ctx,
		page.ID,
		pageHTML,
		page.PlaceholderMap,
		page.MatchedComponentIDs,
		page.Status,
	); err != nil {
		return err
	}

	page.HTMLContent = pageHTML
	return nil
}

func (s *CoursewareAutoAssemblyService) markImageAOCIIndexFailed(
	ctx context.Context,
	coursewareID string,
	imageKey string,
	message string,
) {
	index, err :=
		repository.GetCoursewareImageIndexByKey(
			ctx,
			coursewareID,
			imageKey,
		)
	if err != nil ||
		index == nil {
		return
	}

	_ = repository.UpdateCoursewareImageIndexAssetStatus(
		ctx,
		index.ID,
		nil,
		models.CWImageIndexStatusFailed,
		message,
	)
}
