package services

// courseware_image_iaoci_planner.go — 页面图片槽位IAOCI规划主流程
//
// 本文件负责：
//   - 加载课件和页面；
//   - 为图片槽位补稳定placeholder_id；
//   - 调AI生成与真实槽位一一对应的IAOCI；
//   - 校验稳定图片键和R关系引用边界；
//   - 保存图片索引和关系；
//   - 返回自动装配使用的逐槽位计划。
//
// HTML槽位识别、IAOCI块解析和提示词编译等纯函数位于
// courseware_image_iaoci_planner_helpers.go。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const cwImageIAOCIPlanPromptKey = "prompt_courseware_image_iaoci_plan"

// PlanPageImagesIAOCI 为一页的所有真实图片槽位建立IAOCI计划。
func (s *CoursewareAssetService) PlanPageImagesIAOCI(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	actor *CoursewareActorContext,
) ([]CoursewareImageAOCIPlanItem, error) {
	courseware, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	page, err := repository.GetCoursewarePageByNumber(
		ctx,
		coursewareID,
		pageNumber,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			coursewareID,
			pageNumber,
		)
	}

	normalizedHTML, slots, changed :=
		cwEnsureImagePlaceholderIDs(page.HTMLContent)

	if changed {
		if err := repository.UpdateCWPageHTML(
			ctx,
			page.ID,
			normalizedHTML,
			page.PlaceholderMap,
			page.MatchedComponentIDs,
			page.Status,
		); err != nil {
			return nil, fmt.Errorf(
				"保存图片占位稳定ID失败: %w",
				err,
			)
		}

		page.HTMLContent = normalizedHTML
	}

	// 没有媒体需求或没有真实图片槽位时，不调用AI。
	if strings.TrimSpace(page.MediaRequirements) == "" ||
		len(slots) == 0 {
		_ = repository.UpdatePageImageSuggestions(
			ctx,
			coursewareID,
			pageNumber,
			"",
		)

		_ = repository.MarkPageImageIndexesStaleExcept(
			ctx,
			page.ID,
			nil,
		)

		return []CoursewareImageAOCIPlanItem{}, nil
	}

	expectedOrder := make(map[string]int, len(slots))
	expectedSlot := make(
		map[string]cwImagePlaceholderSlot,
		len(slots),
	)
	allowedPlaceholderIDs := make(
		[]string,
		0,
		len(slots),
	)

	for index := range slots {
		imageKey, keyErr := utils.BuildImageAOCIKey(
			page.ID,
			slots[index].PlaceholderID,
		)
		if keyErr != nil {
			return nil, keyErr
		}

		slots[index].ImageKey = imageKey
		expectedOrder[imageKey] = slots[index].Order
		expectedSlot[imageKey] = slots[index]

		allowedPlaceholderIDs = append(
			allowedPlaceholderIDs,
			slots[index].PlaceholderID,
		)
	}

	if err := repository.MarkPageImageIndexesStaleExcept(
		ctx,
		page.ID,
		allowedPlaceholderIDs,
	); err != nil {
		return nil, err
	}

	anchorAOCI := cwParseCoursewareAnchorAOCI(
		courseware,
	)

	historyIndexes, historyErr :=
		repository.ListCoursewareImageIndexesByCourseware(
			ctx,
			coursewareID,
		)
	if historyErr != nil {
		cwAssetLog.Warn(
			"读取历史图片IAOCI失败，本页不允许跨页R引用",
			"courseware_id", coursewareID,
			"page_number", pageNumber,
			"error", historyErr,
		)

		historyIndexes =
			[]*models.CoursewareImageIndex{}
	}

	historyKeySet := cwGeneratedImageHistoryKeySet(
		historyIndexes,
		page.ID,
	)

	// 当前页已经成功并绑定资产的稳定槽位必须复用。
	// 人工智能补配模式还会保留“内容审核失败”的原IAOCI，让后续生成链先按该错误做安全重述，
	// 而不是重新规划后丢掉真实失败原因。
	repairMode := coursewareImageRepairModeFromContext(ctx)
	currentPreservedByKey := make(
		map[string]*models.CoursewareImageIndex,
	)
	for _, index := range historyIndexes {
		if index == nil ||
			index.PageID == nil ||
			*index.PageID != page.ID ||
			index.IndexType !=
				models.CWImageIndexTypeImage {
			continue
		}

		generatedReusable :=
			index.Status == models.CWImageIndexStatusGenerated &&
				index.AssetID != nil &&
				strings.TrimSpace(*index.AssetID) != ""
		contentReviewReusable :=
			repairMode &&
				index.Status == models.CWImageIndexStatusFailed &&
				isCoursewareImageContentReviewFailureText(
					index.LastError,
				)

		if generatedReusable || contentReviewReusable {
			currentPreservedByKey[index.ImageKey] = index
		}
	}

	// 如果当前每个稳定槽位都有可复用事实，直接返回计划，不再调用文本AI重新规划。
	// 这使“内容审核失败”的人工补配真正只调整失败提示词，也让全成功页面断点恢复零规划成本。
	if preservedItems, complete :=
		cwBuildPreservedImageIAOCIPlanItems(
			slots,
			currentPreservedByKey,
		); complete {
		_ = repository.UpdatePageImageSuggestions(
			ctx,
			coursewareID,
			pageNumber,
			"",
		)

		cwAssetLog.Info(
			"页面图片IAOCI直接复用现有稳定槽位计划",
			"courseware_id", coursewareID,
			"page_number", pageNumber,
			"slot_count", len(slots),
			"repair_mode", repairMode,
		)
		return preservedItems, nil
	}

	userInput := cwBuildImageIAOCIPlanInput(
		courseware,
		page,
		slots,
		anchorAOCI,
		historyIndexes,
	)

	systemPrompt, err :=
		repository.GetCurrentPromptByKey(
			cwImageIAOCIPlanPromptKey,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"加载图片IAOCI规划提示词失败: %w",
			err,
		)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"获取图片IAOCI规划模型失败: %w",
			err,
		)
	}

	userID := scopedActor.UserID
	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		userID,
	)

	traceContext := &ai.TraceContext{
		SceneCode: sceneCWMediaPrompt,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, callErr := ai.CallAI(
		aiConfig,
		systemPrompt.Content,
		userInput,
		traceContext,
	)
	if callErr != nil {
		return nil, fmt.Errorf(
			"AI规划图片IAOCI失败: %w",
			callErr,
		)
	}
	if result == nil ||
		strings.TrimSpace(result.Content) == "" {
		return nil, fmt.Errorf(
			"AI未返回图片IAOCI计划",
		)
	}

	parsedAOCIs, validationErr :=
		cwParseImageAOCIBlocks(result.Content)
	if validationErr == nil {
		validationErr =
			cwValidatePlannedImageAOCIs(
				parsedAOCIs,
				slots,
				expectedOrder,
				expectedSlot,
				historyKeySet,
			)
	}

	if validationErr != nil {
		cwAssetLog.Warn(
			"图片IAOCI首次规划未通过严格校验，准备自动修复一次",
			"courseware_id", coursewareID,
			"page_number", pageNumber,
			"error", validationErr,
		)

		parsedAOCIs, err =
			s.repairImageIAOCIPlanOnce(
				ctx,
				aiConfig,
				systemPrompt.Content,
				userInput,
				result.Content,
				validationErr,
				traceContext,
				slots,
				expectedOrder,
				expectedSlot,
				historyKeySet,
			)
		if err != nil {
			if errors.Is(
				err,
				errCoursewareImageIAOCIPlanRepairCallFailed,
			) {
				return nil, err
			}
			return nil, newCoursewareImageIAOCIPlanRepairableError(
				err,
			)
		}
	}

	// 第一阶段：先保存全部图片索引。
	// 这样同一页面后续槽位的R关系目标一定已经存在。
	planItems := make(
		[]CoursewareImageAOCIPlanItem,
		0,
		len(parsedAOCIs),
	)
	preservedExisting :=
		make(map[string]bool)

	for _, imageAOCI := range parsedAOCIs {
		slot := expectedSlot[imageAOCI.ImageKey]

		if existing := currentPreservedByKey[imageAOCI.ImageKey]; existing != nil {
			existingAOCI, parseErr :=
				utils.ParseImageAOCI(
					existing.AOCIText,
				)
			if parseErr == nil {
				preservedExisting[imageAOCI.ImageKey] = true

				planItems = append(
					planItems,
					CoursewareImageAOCIPlanItem{
						IndexID:       existing.ID,
						PlaceholderID: slot.PlaceholderID,
						ImageKey:      existing.ImageKey,
						AOCIText:      existing.AOCIText,
						Caption:       existing.FocusText,
						Prompt:        existing.GenerationPrompt,
						Size: cwImageAOCISize(
							existingAOCI,
						),
						Order: slot.Order,
					},
				)
				continue
			}
		}

		formattedAOCI, formatErr :=
			utils.FormatImageAOCI(imageAOCI)
		if formatErr != nil {
			return nil, formatErr
		}

		generationPrompt :=
			cwCompileImageGenerationPrompt(
				imageAOCI,
				anchorAOCI,
			)

		indexModel, modelErr :=
			utils.BuildCoursewareImageIndexFromAOCI(
				coursewareID,
				&page.ID,
				slot.PlaceholderID,
				slot.Order,
				imageAOCI,
			)
		if modelErr != nil {
			return nil, modelErr
		}

		indexModel.GenerationPrompt =
			generationPrompt
		indexModel.Status =
			models.CWImageIndexStatusPlanned
		indexModel.AssetID = nil
		indexModel.LastError = ""

		if err := repository.UpsertCoursewarePageImageIndex(
			ctx,
			indexModel,
		); err != nil {
			return nil, fmt.Errorf(
				"保存图片IAOCI索引失败(%s): %w",
				imageAOCI.ImageKey,
				err,
			)
		}

		persistedIndex, err :=
			repository.GetCoursewareImageIndexByKey(
				ctx,
				coursewareID,
				imageAOCI.ImageKey,
			)
		if err != nil {
			return nil, err
		}

		planItems = append(
			planItems,
			CoursewareImageAOCIPlanItem{
				IndexID:       persistedIndex.ID,
				PlaceholderID: slot.PlaceholderID,
				ImageKey:      imageAOCI.ImageKey,
				AOCIText:      formattedAOCI,
				Caption:       imageAOCI.FocusText,
				Prompt:        generationPrompt,
				Size:          cwImageAOCISize(imageAOCI),
				Order:         slot.Order,
			},
		)
	}

	// 第二阶段：全部索引存在后，再保存R关系。
	// 已经生成成功的槽位保持原IAOCI与原关系，禁止仅因补配其它失败槽位而改写成功事实。
	for _, imageAOCI := range parsedAOCIs {
		if preservedExisting[imageAOCI.ImageKey] {
			continue
		}

		if _, err :=
			repository.ReplaceCoursewareImageRelationsByKeys(
				ctx,
				coursewareID,
				imageAOCI.ImageKey,
				imageAOCI.Relations,
			); err != nil {
			return nil, fmt.Errorf(
				"保存图片R关系失败(%s): %w",
				imageAOCI.ImageKey,
				err,
			)
		}
	}

	sort.Slice(
		planItems,
		func(left int, right int) bool {
			return planItems[left].Order <
				planItems[right].Order
		},
	)

	// 新IAOCI已经成为自动装配的事实源，清除旧JSON建议缓存。
	_ = repository.UpdatePageImageSuggestions(
		ctx,
		coursewareID,
		pageNumber,
		"",
	)

	cwAssetLog.Info(
		"页面图片IAOCI规划成功",
		"courseware_id", coursewareID,
		"page_number", pageNumber,
		"slot_count", len(slots),
		"iaoci_count", len(planItems),
		"reused_existing",
		len(preservedExisting),
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
	)

	return planItems, nil
}

// cwBuildPreservedImageIAOCIPlanItems 把当前数据库中的稳定槽位事实转换为装配计划。
// 只有所有真实槽位都能安全解析时才返回complete=true；任一槽位缺失或IAOCI损坏则回到正式规划链。
func cwBuildPreservedImageIAOCIPlanItems(
	slots []cwImagePlaceholderSlot,
	preserved map[string]*models.CoursewareImageIndex,
) (
	[]CoursewareImageAOCIPlanItem,
	bool,
) {
	if len(slots) == 0 || len(preserved) == 0 {
		return nil, false
	}

	items := make(
		[]CoursewareImageAOCIPlanItem,
		0,
		len(slots),
	)
	for _, slot := range slots {
		existing := preserved[slot.ImageKey]
		if existing == nil {
			return nil, false
		}

		imageAOCI, err := utils.ParseImageAOCI(
			existing.AOCIText,
		)
		if err != nil || imageAOCI.ImageKey != slot.ImageKey {
			return nil, false
		}

		items = append(
			items,
			CoursewareImageAOCIPlanItem{
				IndexID:       existing.ID,
				PlaceholderID: slot.PlaceholderID,
				ImageKey:      existing.ImageKey,
				AOCIText:      existing.AOCIText,
				Caption:       existing.FocusText,
				Prompt:        existing.GenerationPrompt,
				Size:          cwImageAOCISize(imageAOCI),
				Order:         slot.Order,
			},
		)
	}

	sort.Slice(
		items,
		func(left int, right int) bool {
			return items[left].Order < items[right].Order
		},
	)
	return items, true
}
