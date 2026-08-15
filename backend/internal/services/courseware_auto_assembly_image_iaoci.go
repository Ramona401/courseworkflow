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

		repairablePlanFailure :=
			isCoursewareImageIAOCIPlanRepairableError(err)
		placeholderState := "plan_error"
		if repairablePlanFailure {
			placeholderState = "plan_failed"
			result.errMsg = fmt.Sprintf(
				"第%d页图片规划格式自动修复后仍未通过",
				page.PageNumber,
			)
		} else {
			result.errMsg = fmt.Sprintf(
				"第%d页图片规划暂时失败",
				page.PageNumber,
			)
		}

		currentHTML := page.HTMLContent

		// 只有“原规划+一次协议修复”仍失败才标成可智能补配的plan_failed。
		// 网络、配置、鉴权等其它规划错误使用plan_error，避免教师按钮给出错误修复语义。
		for _, slot := range slots {
			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					slot.PlaceholderID,
					placeholderState,
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
			"repairable", repairablePlanFailure,
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
		reusedHTML, reused, reuseErr :=
			s.reuseGeneratedIAOCIPlan(
				ctx,
				pageContext.coursewareID,
				freshPage,
				plan,
				currentHTML,
			)
		if reuseErr != nil {
			cwAssemblyLog.Warn(
				"复用已成功IAOCI图片失败，继续按当前槽位生成链处理",
				"page", page.PageNumber,
				"placeholder_id", plan.PlaceholderID,
				"image_key", plan.ImageKey,
				"error", reuseErr,
			)
		} else if reused {
			currentHTML = reusedHTML
			successCount++

			GlobalCWSSEHub.Broadcast(
				pageContext.coursewareID,
				CWSSEEvent{
					EventType: "assembly_page_image",
					Data: map[string]interface{}{
						"page_number":    page.PageNumber,
						"stage":          "image_slot_reused",
						"placeholder_id": plan.PlaceholderID,
						"image_key":      plan.ImageKey,
						"message": fmt.Sprintf(
							"第 %d 页：已复用成功配图 %s，不重复生图。",
							page.PageNumber,
							plan.PlaceholderID,
						),
					},
				},
			)

			continue
		}

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
			s.generateImageFromIAOCIWithAutoRepair(
				ctx,
				pageContext,
				page,
				plan,
				relationReference,
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

			failureState := "generation_failed"
			if errors.Is(
				generationErr,
				ErrCoursewareImageContentReviewRejected,
			) {
				failureState = "content_review_failed"
			}

			currentHTML, _ =
				cwHideImagePlaceholder(
					currentHTML,
					plan.PlaceholderID,
					failureState,
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

// reuseGeneratedIAOCIPlan 复用同一稳定image_key已经成功绑定的图片资产。
//
// 该入口只填回当前HTML占位并落库，不调用图片供应商、不新建资产、不产生新的媒体计费。
func (s *CoursewareAutoAssemblyService) reuseGeneratedIAOCIPlan(
	ctx context.Context,
	coursewareID string,
	page *models.CoursewarePage,
	plan CoursewareImageAOCIPlanItem,
	currentHTML string,
) (
	string,
	bool,
	error,
) {
	index, err :=
		repository.GetCoursewareImageIndexByKey(
			ctx,
			coursewareID,
			plan.ImageKey,
		)
	if err != nil ||
		index == nil ||
		index.Status !=
			models.CWImageIndexStatusGenerated ||
		index.AssetID == nil ||
		strings.TrimSpace(
			*index.AssetID,
		) == "" {
		return currentHTML, false, nil
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			*index.AssetID,
		)
	if err != nil {
		return currentHTML,
			false,
			fmt.Errorf(
				"已生成图片资产不可读取: %w",
				err,
			)
	}
	if asset == nil {
		return currentHTML,
			false,
			fmt.Errorf(
				"已生成图片资产不存在",
			)
	}

	imageURL :=
		resolveAssetPublicURL(asset)
	if strings.TrimSpace(imageURL) == "" {
		return currentHTML,
			false,
			fmt.Errorf(
				"已生成图片资产缺少可用URL",
			)
	}

	filledHTML, filled :=
		cwFillImagePlaceholder(
			currentHTML,
			plan.PlaceholderID,
			imageURL,
			index.FocusText,
		)
	if !filled {
		return currentHTML,
			false,
			fmt.Errorf(
				"已生成图片未找到对应HTML占位",
			)
	}

	if err :=
		s.persistIAOCIAssemblyHTML(
			ctx,
			page,
			filledHTML,
		); err != nil {
		return currentHTML,
			false,
			fmt.Errorf(
				"复用已生成图片写回HTML失败: %w",
				err,
			)
	}

	cwAssemblyLog.Info(
		"IAOCI图片槽位复用已成功资产",
		"courseware_id", coursewareID,
		"page", page.PageNumber,
		"placeholder_id", plan.PlaceholderID,
		"image_key", plan.ImageKey,
		"asset_id", *index.AssetID,
	)

	return filledHTML, true, nil
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
