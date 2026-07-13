package services

// lesson_plan_context_receipt.go — 备课上下文回执采集
//
// 本文件不改变既有提示词拼接规则，而是在同一轮调用中确定性记录：
//   - 最终使用了哪个助手，以及来源；
//   - 配方、课本、单元方案、课程大纲和班级学情是否真正注入；
//   - 第3层专业组件的来源与最终条目；
//   - 本轮参考资料和最终system prompt规模。
//
// 第一版采用“既有提示词照常生成 + 同条件复核”的兼容方式，避免一次重写六层引擎。
// 复核过程只执行轻量数据库读取和确定性匹配，不新增AI调用、不写数据库。

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// resolveAssistantPromptForReceipt 解析助手并保留来源信息。
//
// 优先级与原resolveAssistantPrompt完全一致：
//  1. 当轮手动指定；
//  2. 老师×学科偏好；
//  3. 显式系统默认纯骨架；
//  4. 技能路由自动匹配；
//  5. 无匹配或加载失败时静默降级。
func (s *LessonPlanGenService) resolveAssistantPromptForReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	assistantID string,
	callerID string,
) *models.AssistantPromptResolution {
	result := &models.AssistantPromptResolution{
		Receipt: &models.AssistantContextReceipt{
			Status: models.ContextReceiptNotFound,
			Reason: "本轮未匹配到同学科同具体年级的可用助手，使用系统阶段骨架",
		},
	}

	if s.assistantService == nil {
		result.Receipt.Status = models.ContextReceiptUnavailable
		result.Receipt.Reason = "助手服务当前不可用，已使用系统阶段骨架"
		return result
	}
	if lp == nil {
		return result
	}

	user, err := repository.FindUserByID(ctx, callerID)
	if err != nil {
		result.Receipt.Status = models.ContextReceiptUnavailable
		result.Receipt.Reason = "无法确认助手可见范围，已使用系统阶段骨架"
		return result
	}

	actor := BuildActorFromClaims(
		ctx,
		callerID,
		user.Role,
	)
	scene := stageCodeToAssistantScene(
		lp.CurrentStage,
	)

	load := func(
		id string,
		mode string,
	) (*models.AIAssistant, error) {
		assistant, loadErr :=
			s.assistantService.LoadActiveAssistantForLessonUse(
				ctx,
				actor,
				id,
				lp.Subject,
				lp.Grade,
				scene,
			)
		if loadErr != nil {
			return nil, loadErr
		}

		if strings.TrimSpace(assistant.FullPrompt) == "" {
			return nil, errors.New("助手内容为空")
		}

		result.Prompt = assistant.FullPrompt
		result.Receipt.Status = models.ContextReceiptLoaded
		result.Receipt.SelectionMode = mode
		result.Receipt.ID = assistant.ID
		result.Receipt.Name = assistant.Name
		result.Receipt.Source = assistant.Source
		result.Receipt.Reason = ""

		lpGenLog.Info(
			"上下文回执-助手已解析",
			"plan_id", lp.ID,
			"assistant_id", assistant.ID,
			"assistant_name", assistant.Name,
			"selection_mode", mode,
			"source", assistant.Source,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"scene", scene,
		)

		return assistant, nil
	}

	assistantID = strings.TrimSpace(assistantID)

	// 当轮显式选择拥有最高优先级。
	// 若不匹配当前课程，直接使用系统骨架，不能偷偷换成其它助手。
	if assistantID != "" {
		if _, loadErr := load(
			assistantID,
			"manual",
		); loadErr != nil {
			result.Receipt.Status = models.ContextReceiptUnavailable
			result.Receipt.SelectionMode = "manual"
			result.Receipt.ID = assistantID
			result.Receipt.Reason = "老师指定的助手不适用于当前学科、具体年级或阶段，已使用系统阶段骨架"
		}
		return result
	}

	prefID, found, prefErr := repository.GetPref(
		ctx,
		callerID,
		lp.Subject,
	)
	if prefErr != nil {
		lpGenLog.Warn(
			"上下文回执-查询助手偏好失败，继续自动匹配",
			"caller_id", callerID,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"error", prefErr,
		)
	} else if found {
		prefID = strings.TrimSpace(prefID)

		if prefID == "" {
			result.Receipt.Status = models.ContextReceiptExplicitNone
			result.Receipt.SelectionMode = "explicit_none"
			result.Receipt.Reason = "老师已明确选择系统默认，不注入额外助手"
			return result
		}

		if _, loadErr := load(
			prefID,
			"preference",
		); loadErr == nil {
			return result
		}

		// 存量偏好在当前年级或阶段不适用时不删除，
		// 继续寻找当前课程真正适用的自动助手。
		lpGenLog.Info(
			"老师助手偏好不适用于当前学科、具体年级或阶段，继续自动匹配",
			"plan_id", lp.ID,
			"pref_assistant_id", prefID,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"scene", scene,
		)
	}

	defaultID := RouteDefaultAssistant(
		ctx,
		s.assistantService,
		actor,
		lp.CurrentStage,
		lp.Subject,
		lp.Grade,
	)
	if defaultID == "" {
		return result
	}

	if _, loadErr := load(
		defaultID,
		"auto",
	); loadErr != nil {
		result.Receipt.Status = models.ContextReceiptUnavailable
		result.Receipt.SelectionMode = "auto"
		result.Receipt.ID = defaultID
		result.Receipt.Reason = "自动匹配的助手当前不可用，已使用系统阶段骨架"
	}

	return result
}

// LoadStagePromptContextWithReceipt 调用既有提示词生成，并同步生成确定性回执。
// 回执构建失败不得阻断备课，最坏只返回较少的回执项。
func (s *WorkshopStageService) LoadStagePromptContextWithReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
	assistantPrompt string,
	recentUserText string,
	assistantReceipt *models.AssistantContextReceipt,
) (string, *models.ContextReceipt, error) {
	prompt, err := s.LoadStagePromptContextV2(
		ctx,
		lp,
		stageCode,
		assistantPrompt,
		recentUserText,
	)
	if err != nil {
		return "", nil, err
	}

	receipt := s.buildContextReceipt(
		ctx,
		lp,
		stageCode,
		recentUserText,
		assistantReceipt,
	)
	receipt.SystemPromptRunes = len([]rune(prompt))

	return prompt, receipt, nil
}

func (s *WorkshopStageService) buildContextReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
	recentUserText string,
	assistantReceipt *models.AssistantContextReceipt,
) *models.ContextReceipt {
	receipt := &models.ContextReceipt{
		Version:   models.ContextReceiptVersion,
		StageCode: stageCode,
		Assistant: assistantReceipt,
		RefMaterial: &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有附加参考资料",
		},
	}

	stage, recipe := s.loadReceiptStageAndRecipe(ctx, lp, stageCode)

	recipeSelectionMode := resolveRecipeSelectionModeForReceipt(
		ctx,
		lp,
	)
	receipt.Recipe = buildRecipeReceipt(
		lp,
		recipe,
		recipeSelectionMode,
	)
	receipt.Textbook = s.buildTextbookReceipt(ctx, lp)
	receipt.UnitPlan = buildUnitPlanReceipt(ctx, lp)

	unitLoaded := receipt.UnitPlan != nil &&
		receipt.UnitPlan.Status == models.ContextReceiptLoaded

	receipt.CourseOutline = buildCourseOutlineReceipt(
		ctx,
		lp,
		stageCode,
		unitLoaded,
	)
	receipt.ClassProfile = buildClassProfileReceipt(
		ctx,
		lp,
		stageCode,
	)
	receipt.Components = s.buildComponentsReceipt(
		ctx,
		lp,
		stage,
		recipe,
		stageCode,
		recentUserText,
	)

	return receipt
}

func (s *WorkshopStageService) loadReceiptStageAndRecipe(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
) (*models.WorkshopStage, *models.TeachingRecipe) {
	if lp == nil {
		return nil, nil
	}

	// 与正式提示词装配使用完全相同的严格配方规则。
	// 配方不适用当前学科或具体年级时，recipe保持nil。
	var recipe *models.TeachingRecipe
	validRecipeID := ""

	if lp.RecipeID != nil &&
		strings.TrimSpace(*lp.RecipeID) != "" {
		candidate, recipeErr := loadRecipeForLesson(
			ctx,
			*lp.RecipeID,
			lp.Subject,
			lp.Grade,
		)
		if recipeErr == nil {
			recipe = candidate
			validRecipeID = candidate.ID
		}
	}

	// 错误配方留下的自定义阶段快照不能继续生效。
	isCustomStage := false
	if recipe != nil &&
		strings.TrimSpace(lp.StageConfig) != "" &&
		lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal(
			[]byte(lp.StageConfig),
			&snapshots,
		) == nil {
			for _, snapshot := range snapshots {
				if snapshot.StageCode == stageCode &&
					snapshot.IsCustom {
					isCustomStage = true
					break
				}
			}
		}
	}

	loadSystemStage := func(
		code string,
	) *models.WorkshopStage {
		stage, _ := repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			code,
		)
		if stage != nil {
			return stage
		}

		if code == "write" {
			return nil
		}

		stage, _ = repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			"write",
		)
		return stage
	}

	var stage *models.WorkshopStage
	var err error

	switch {
	case isCustomStage:
		stage, err = repository.GetRecipeStageByCode(
			ctx,
			validRecipeID,
			stageCode,
		)
		if err != nil || stage == nil {
			stage = loadSystemStage(stageCode)
		}

	case recipe != nil:
		stage, err = repository.GetStageByCode(
			ctx,
			models.StageSourceRecipe,
			stageCode,
		)
		if err != nil || stage == nil {
			stage = loadSystemStage(stageCode)
		}

	default:
		stage = loadSystemStage(stageCode)
	}

	return stage, recipe
}

func buildRecipeReceipt(
	lp *models.LessonPlan,
	recipe *models.TeachingRecipe,
	selectionMode models.RecipeSelectionMode,
) *models.MaterialContextReceipt {
	modeText := string(selectionMode)

	if selectionMode == models.RecipeSelectionModeNone {
		return &models.MaterialContextReceipt{
			Status:        models.ContextReceiptExplicitNone,
			SelectionMode: modeText,
			Reason:        "老师已明确选择不使用配方，本轮使用系统阶段骨架",
		}
	}

	if lp.RecipeID == nil ||
		strings.TrimSpace(*lp.RecipeID) == "" {
		if selectionMode == models.RecipeSelectionModeAuto {
			return &models.MaterialContextReceipt{
				Status:        models.ContextReceiptNotFound,
				SelectionMode: modeText,
				Reason:        "平台已自动匹配，但没有找到可用配方，本轮使用系统阶段骨架",
			}
		}

		return &models.MaterialContextReceipt{
			Status:        models.ContextReceiptUnavailable,
			SelectionMode: modeText,
			Reason:        "老师选择了配方，但没有读取到有效的配方标识",
		}
	}

	recipeID := strings.TrimSpace(*lp.RecipeID)
	if recipe == nil {
		return &models.MaterialContextReceipt{
			Status:        models.ContextReceiptUnavailable,
			SelectionMode: modeText,
			ID:            recipeID,
			Reason:        "已关联配方，但当前无法读取或不适用于本教案的学科和具体年级，本轮已忽略",
		}
	}

	reason := ""
	switch selectionMode {
	case models.RecipeSelectionModeAuto:
		reason = "平台根据学校、教研组和学科规则自动选择"
	case models.RecipeSelectionModeSelected:
		reason = "老师在开始备课时明确选择"
	}

	return &models.MaterialContextReceipt{
		Status:        models.ContextReceiptLoaded,
		SelectionMode: modeText,
		ID:            recipe.ID,
		Name:          recipe.Name,
		Reason:        reason,
	}
}

func (s *WorkshopStageService) buildTextbookReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
) *models.MaterialContextReceipt {
	raw := strings.TrimSpace(lp.TextbookPageIDs)
	if raw == "" || raw == "[]" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联课本页面",
		}
	}

	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil || len(ids) == 0 {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "课本页面关联数据无法解析",
		}
	}

	if s.textbookService == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Count:  len(ids),
			Reason: "课本服务当前不可用，本轮未读取课本原文",
		}
	}

	pages, err := repository.GetTextbookPagesByIDs(ctx, ids)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Count:  len(ids),
			Reason: "关联课本页面读取失败",
		}
	}

	readable := 0
	titles := make([]string, 0, len(pages))
	for _, page := range pages {
		if strings.TrimSpace(page.OCRText) != "" {
			readable++
		}
		label := strings.TrimSpace(page.Chapter)
		if label == "" {
			label = strings.TrimSpace(page.TextbookName)
		}
		if label != "" {
			titles = appendUniqueReceiptTitle(titles, label)
		}
	}

	// 回执层已经通过GetTextbookPagesByIDs读取了与正式提示词装配相同的课本页面。
	// 这里禁止再次调用BuildTextbookContext：
	// BuildTextbookContext除生成上下文文本外，还会异步递增每页usage_count；
	// 若回执再次调用，会导致同一轮备课把课本使用次数重复增加。
	//
	// 正式提示词装配中的BuildTextbookContext调用保持不变，本回执只根据已读取页面、
	// OCR可读数量和标题生成教师可理解的状态，不产生任何额外业务副作用。
	if len(pages) == 0 {
		return &models.MaterialContextReceipt{
			Status:          models.ContextReceiptUnavailable,
			Count:           len(ids),
			ReadableCount:   readable,
			UnreadableCount: maxReceiptInt(0, len(ids)-readable),
			Titles:          limitReceiptTitles(titles, 5),
			Reason:          "课本已关联，但没有读取到可用页面",
		}
	}

	return &models.MaterialContextReceipt{
		Status:          models.ContextReceiptLoaded,
		Count:           len(ids),
		ReadableCount:   readable,
		UnreadableCount: maxReceiptInt(0, len(ids)-readable),
		Titles:          limitReceiptTitles(titles, 5),
	}
}

func buildUnitPlanReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
) *models.MaterialContextReceipt {
	if lp.UnitPlanID == nil || strings.TrimSpace(*lp.UnitPlanID) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联单元方案",
		}
	}

	id := strings.TrimSpace(*lp.UnitPlanID)
	up, err := repository.GetUnitPlanByID(ctx, id)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     id,
			Reason: "已关联单元方案，但当前无法读取",
		}
	}

	name := strings.TrimSpace(up.UnitTheme)
	if name == "" {
		name = strings.TrimSpace(up.Unit)
	}

	if up.Status != models.UnitPlanStatusActive {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     up.ID,
			Name:   name,
			Reason: "单元方案不是已启用状态，本轮未读取",
		}
	}

	if strings.TrimSpace(BuildUnitPlanContext(up)) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     up.ID,
			Name:   name,
			Reason: "单元方案没有可注入内容",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID:     up.ID,
		Name:   name,
	}
}

func buildCourseOutlineReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
	unitPlanLoaded bool,
) *models.MaterialContextReceipt {
	if stageCode != "analyze" && stageCode != "design" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			Reason: "课程大纲只在教学分析和教学设计阶段读取",
		}
	}

	if unitPlanLoaded {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptSuperseded,
			Reason: "已使用更具体的单元方案，课程大纲本轮让位",
		}
	}

	if lp.CourseOutlinePublisher == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "老师没有选择课程大纲教材版本",
		}
	}

	candidates, err := repository.ListActiveOutlinesBySubject(
		ctx,
		lp.Subject,
	)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "课程大纲候选读取失败",
		}
	}

	hits := MatchOutlinesByPublisher(
		lp.Grade,
		*lp.CourseOutlinePublisher,
		candidates,
	)
	if len(hits) == 0 {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "所选教材版本与学段没有匹配到可用大纲",
		}
	}

	if strings.TrimSpace(BuildCourseOutlinesContext(hits)) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			Reason: "匹配到课程大纲，但没有可注入内容",
		}
	}

	titles := make([]string, 0, len(hits))
	for _, item := range hits {
		titles = appendUniqueReceiptTitle(titles, item.Title)
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		Count:  len(hits),
		Titles: limitReceiptTitles(titles, 5),
	}
}

func buildClassProfileReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
) *models.MaterialContextReceipt {
	if lp.ClassProfileID == nil ||
		strings.TrimSpace(*lp.ClassProfileID) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotLinked,
			Reason: "本轮没有关联班级学情",
		}
	}

	id := strings.TrimSpace(*lp.ClassProfileID)

	if stageCode != "analyze" &&
		stageCode != "design" &&
		stageCode != "write" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			ID:     id,
			Reason: "班级学情只在教学分析、教学设计和教案撰写阶段读取",
		}
	}

	cp, err := repository.GetClassProfileByID(ctx, id)
	if err != nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     id,
			Reason: "已关联班级学情，但当前无法读取",
		}
	}

	if cp.Status != models.ClassProfileStatusActive {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     cp.ID,
			Name:   cp.ClassName,
			Reason: "班级学情不是已启用状态，本轮未读取",
		}
	}

	if cp.CreatedBy != lp.AuthorID {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptForbidden,
			ID:     cp.ID,
			Name:   cp.ClassName,
			Reason: "班级学情归属与教案作者不一致，本轮未读取",
		}
	}

	if strings.TrimSpace(BuildClassProfileContext(cp)) == "" {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			ID:     cp.ID,
			Name:   cp.ClassName,
			Reason: "班级学情没有可注入内容",
		}
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		ID:     cp.ID,
		Name:   cp.ClassName,
	}
}

func (s *WorkshopStageService) buildComponentsReceipt(
	ctx context.Context,
	lp *models.LessonPlan,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	stageCode string,
	recentUserText string,
) *models.ComponentsContextReceipt {
	if stage == nil ||
		strings.TrimSpace(stage.ComponentTypes) == "" ||
		stage.ComponentTypes == "[]" {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			Reason: "当前阶段没有配置专业组件类型",
		}
	}

	selectedIDs := s.getSelectedComponentIDsFromOutput(
		ctx,
		lp.ID,
		stageCode,
	)
	if len(selectedIDs) > 0 {
		groups, err := repository.GetRecipeComponentContents(
			ctx,
			selectedIDs,
		)
		if err != nil || len(groups) == 0 {
			return &models.ComponentsContextReceipt{
				Status:        models.ContextReceiptUnavailable,
				SelectionMode: "manual",
				Reason:        "老师已选择组件，但当前无法读取组件内容",
			}
		}
		return componentReceiptFromGroups(
			groups,
			"manual",
			len(selectedIDs),
			false,
		)
	}

	if recipe != nil {
		var recipeIDs []string
		_ = json.Unmarshal([]byte(recipe.ComponentIDs), &recipeIDs)
		if len(recipeIDs) > 0 {
			groups, err := repository.GetRecipeComponentContents(
				ctx,
				recipeIDs,
			)
			if err == nil && len(groups) > 0 {
				var stageTypes []string
				_ = json.Unmarshal(
					[]byte(stage.ComponentTypes),
					&stageTypes,
				)
				typeSet := make(map[string]bool, len(stageTypes))
				for _, item := range stageTypes {
					typeSet[item] = true
				}

				filtered := make(
					[]*models.MatchedComponentGroup,
					0,
					len(groups),
				)
				for _, group := range groups {
					if typeSet[group.LibraryType] {
						filtered = append(filtered, group)
					}
				}
				if len(filtered) > 0 {
					return componentReceiptFromGroups(
						filtered,
						"recipe",
						len(recipeIDs),
						false,
					)
				}
			}
		}
	}

	return buildAutoComponentsReceipt(
		ctx,
		stage.ComponentTypes,
		lp.Subject,
		lp.Grade,
		stageCode,
		recentUserText,
	)
}

func buildAutoComponentsReceipt(
	ctx context.Context,
	componentTypesJSON string,
	subject string,
	grade string,
	stageCode string,
	recentUserText string,
) *models.ComponentsContextReceipt {
	var stageTypes []string
	if err := json.Unmarshal(
		[]byte(componentTypesJSON),
		&stageTypes,
	); err != nil || len(stageTypes) == 0 {
		return &models.ComponentsContextReceipt{
			Status: models.ContextReceiptNotApplicable,
			Reason: "当前阶段没有可匹配的组件类型",
		}
	}

	normalizedGrade := utils.NormalizeGradeToNumber(grade)

	useRerank := skillRouterRerankEnabled &&
		strings.TrimSpace(recentUserText) != ""

	limit := 2
	if useRerank {
		limit = skillRouterCandidateLimitPerType
	}

	request := &models.MatchComponentsRequest{
		Subject:      subject,
		GradeRange:   normalizedGrade,
		LibraryTypes: stageTypes,
		Limit:        limit,
	}
	if timings, ok := stageTimingMap[stageCode]; ok {
		request.StageTiming = timings
	}

	groups, err := repository.MatchComponents(ctx, request)
	if err != nil || len(groups) == 0 {
		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptNotFound,
			SelectionMode: "auto",
			Reason:        "本轮没有匹配到可用专业组件",
		}
	}

	if !useRerank {
		return componentReceiptFromGroups(
			groups,
			"auto",
			countMatchedReceiptComponents(groups),
			false,
		)
	}

	candidates := make([]*rerankCandidate, 0)
	rank := 0
	for _, group := range groups {
		for _, component := range group.Components {
			candidates = append(candidates, &rerankCandidate{
				libraryType:  group.LibraryType,
				libraryName:  group.LibraryName,
				component:    component,
				originalRank: rank,
			})
			rank++
		}
	}

	keywords := extractRerankKeywords(recentUserText)
	for _, candidate := range candidates {
		candidate.score = scoreCandidateLexical(
			candidate.component,
			keywords,
		)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].originalRank <
			candidates[j].originalRank
	})

	topN := skillRouterTopN
	if topN > len(candidates) {
		topN = len(candidates)
	}
	chosen := candidates[:topN]

	items := make(
		[]models.ComponentContextReceiptItem,
		0,
		len(chosen),
	)
	for _, candidate := range chosen {
		items = append(items, models.ComponentContextReceiptItem{
			ID:           candidate.component.ID,
			LibraryType:  candidate.libraryType,
			LibraryName:  candidate.libraryName,
			DisplayLabel: candidate.component.DisplayLabel,
			QualityScore: candidate.component.QualityScore,
		})
	}

	return &models.ComponentsContextReceipt{
		Status:         models.ContextReceiptLoaded,
		SelectionMode:  "reranked",
		CandidateCount: len(candidates),
		Reranked:       len(keywords) > 0,
		Items:          items,
	}
}

func componentReceiptFromGroups(
	groups []*models.MatchedComponentGroup,
	mode string,
	candidateCount int,
	reranked bool,
) *models.ComponentsContextReceipt {
	items := make([]models.ComponentContextReceiptItem, 0)

	for _, group := range groups {
		for _, component := range group.Components {
			items = append(items, models.ComponentContextReceiptItem{
				ID:           component.ID,
				LibraryType:  group.LibraryType,
				LibraryName:  group.LibraryName,
				DisplayLabel: component.DisplayLabel,
				QualityScore: component.QualityScore,
			})
		}
	}

	if len(items) == 0 {
		return &models.ComponentsContextReceipt{
			Status:        models.ContextReceiptNotFound,
			SelectionMode: mode,
			Reason:        "本轮没有实际读取专业组件",
		}
	}

	return &models.ComponentsContextReceipt{
		Status:         models.ContextReceiptLoaded,
		SelectionMode:  mode,
		CandidateCount: candidateCount,
		Reranked:       reranked,
		Items:          items,
	}
}

func countMatchedReceiptComponents(
	groups []*models.MatchedComponentGroup,
) int {
	total := 0
	for _, group := range groups {
		total += len(group.Components)
	}
	return total
}

func appendUniqueReceiptTitle(
	items []string,
	value string,
) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func limitReceiptTitles(
	items []string,
	limit int,
) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func maxReceiptInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
