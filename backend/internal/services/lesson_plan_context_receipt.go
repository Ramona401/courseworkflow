package services

// lesson_plan_context_receipt.go — 备课上下文回执核心装配
//
// 本文件负责：
//   - 助手解析与来源记录；
//   - 正式提示词生成后的回执总装；
//   - 配方与阶段定义的同规则复核。
//
// 课本、单元方案、原始课程大纲、active知识脉络、班级学情和组件
// 的具体回执构建拆分到lesson_plan_context_receipt_materials.go。
//
// 核心边界：
//   - 正式回执不能为了“确认状态”而偷偷读取本轮没有注入的原始大纲；
//   - 原始课程大纲和active知识脉络必须分别记录；
//   - 回执构建不新增AI调用，不写数据库。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// resolveAssistantPromptForReceipt 解析助手并保留真实来源信息。
//
// 优先级与正式助手解析一致：
//   1. 当轮手动指定；
//   2. 老师×学科偏好；
//   3. 显式系统默认纯骨架；
//   4. 技能路由自动匹配；
//   5. 无匹配或加载失败时静默降级。
//
// 手动指定和老师偏好忽略具体年级；
// 自动助手继续严格匹配具体年级。
func (s *LessonPlanGenService) resolveAssistantPromptForReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	assistantID string,
	callerID string,
) *models.AssistantPromptResolution {
	result := &models.AssistantPromptResolution{
		Receipt: &models.AssistantContextReceipt{
			Status: models.ContextReceiptNotFound,
			Reason: "本轮没有严格匹配到可用自动助手，使用系统阶段骨架",
		},
	}

	if s.assistantService == nil {
		result.Receipt.Status =
			models.ContextReceiptUnavailable
		result.Receipt.Reason =
			"助手服务当前不可用，已使用系统阶段骨架"
		return result
	}
	if lessonPlan == nil {
		return result
	}

	user, err := repository.FindUserByID(
		ctx,
		callerID,
	)
	if err != nil {
		result.Receipt.Status =
			models.ContextReceiptUnavailable
		result.Receipt.Reason =
			"无法确认助手可见范围，已使用系统阶段骨架"
		return result
	}

	actor := BuildActorFromClaims(
		ctx,
		callerID,
		user.Role,
	)

	// 回执与正式助手注入必须使用同一个教案教育域快照。
	applyLessonPlanEducationDomainToAssistantActor(
		actor,
		lessonPlan,
	)

	scene := stageCodeToAssistantScene(
		lessonPlan.CurrentStage,
	)

	setLoaded := func(
		assistant *models.AIAssistant,
		mode string,
	) {
		result.Prompt = assistant.FullPrompt
		result.Receipt.Status =
			models.ContextReceiptLoaded
		result.Receipt.SelectionMode = mode
		result.Receipt.ID = assistant.ID
		result.Receipt.Name = assistant.Name
		result.Receipt.Source = assistant.Source
		result.Receipt.Reason = ""
	}

	loadManual := func(
		id string,
		mode string,
	) error {
		assistant, loadErr :=
			s.assistantService.
				LoadActiveAssistantForManualLessonUse(
					ctx,
					actor,
					id,
					lessonPlan.Subject,
					scene,
				)
		if loadErr != nil {
			return loadErr
		}
		if strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
			return errors.New(
				"助手内容为空",
			)
		}

		setLoaded(assistant, mode)
		return nil
	}

	loadAuto := func(
		id string,
		mode string,
	) error {
		assistant, loadErr :=
			s.assistantService.
				LoadActiveAssistantForLessonUse(
					ctx,
					actor,
					id,
					lessonPlan.Subject,
					lessonPlan.Grade,
					scene,
				)
		if loadErr != nil {
			return loadErr
		}
		if strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
			return errors.New(
				"助手内容为空",
			)
		}

		setLoaded(assistant, mode)
		return nil
	}

	assistantID = strings.TrimSpace(
		assistantID,
	)

	if assistantID != "" {
		if loadErr := loadManual(
			assistantID,
			"manual",
		); loadErr != nil {
			result.Receipt.Status =
				models.ContextReceiptUnavailable
			result.Receipt.SelectionMode =
				"manual"
			result.Receipt.ID =
				assistantID
			result.Receipt.Reason =
				"老师指定的助手已停用、无权使用，或不适用于当前学科和阶段"
		}

		return result
	}

	preferenceID, found, preferenceErr :=
		repository.GetPref(
			ctx,
			callerID,
			lessonPlan.Subject,
		)

	if preferenceErr != nil {
		lpGenLog.Warn(
			"上下文回执-查询助手偏好失败，继续自动匹配",
			"caller_id", callerID,
			"subject", lessonPlan.Subject,
			"grade", lessonPlan.Grade,
			"error", preferenceErr,
		)
	} else if found {
		preferenceID = strings.TrimSpace(
			preferenceID,
		)

		if preferenceID == "" {
			result.Receipt.Status =
				models.ContextReceiptExplicitNone
			result.Receipt.SelectionMode =
				"explicit_none"
			result.Receipt.Reason =
				"老师已明确选择系统默认，不注入额外助手"
			return result
		}

		if loadErr := loadManual(
			preferenceID,
			"preference",
		); loadErr == nil {
			return result
		}

		lpGenLog.Info(
			"老师助手偏好不适用于当前学科或阶段，继续自动匹配",
			"plan_id", lessonPlan.ID,
			"pref_assistant_id", preferenceID,
			"subject", lessonPlan.Subject,
			"grade", lessonPlan.Grade,
			"scene", scene,
		)
	}

	defaultID := RouteDefaultAssistant(
		ctx,
		s.assistantService,
		actor,
		lessonPlan.CurrentStage,
		lessonPlan.Subject,
		lessonPlan.Grade,
	)
	if defaultID == "" {
		return result
	}

	if loadErr := loadAuto(
		defaultID,
		"auto",
	); loadErr != nil {
		result.Receipt.Status =
			models.ContextReceiptUnavailable
		result.Receipt.SelectionMode =
			"auto"
		result.Receipt.ID =
			defaultID
		result.Receipt.Reason =
			"自动匹配的助手当前不可用，已使用系统阶段骨架"
	}

	return result
}

// LoadStagePromptContextWithReceipt 调用正式提示词生成，随后生成确定性回执。
//
// 提示词先生成非常重要：
// BuildLessonPlanCourseOutlineContext可能在analyze尚未完成确认时，
// 把当前turnPlan中的UseKnowledgeLineage改为false；回执必须读取该最终状态。
func (s *WorkshopStageService) LoadStagePromptContextWithReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	assistantPrompt string,
	recentUserText string,
	assistantReceipt *models.AssistantContextReceipt,
) (
	string,
	*models.ContextReceipt,
	error,
) {
	prompt, err := s.LoadStagePromptContextV2(
		ctx,
		lessonPlan,
		stageCode,
		assistantPrompt,
		recentUserText,
	)
	if err != nil {
		return "", nil, err
	}

	receipt := s.buildContextReceipt(
		ctx,
		lessonPlan,
		stageCode,
		recentUserText,
		assistantReceipt,
	)

	receipt.SystemPromptRunes =
		len([]rune(prompt))

	return prompt, receipt, nil
}

// buildContextReceipt 组装本轮正式上下文回执。
func (s *WorkshopStageService) buildContextReceipt(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
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

	if lessonPlan == nil {
		receipt.CourseOutline =
			&models.MaterialContextReceipt{
				Status: models.ContextReceiptUnavailable,
				Reason: "教案数据为空，无法确认课程大纲来源",
			}
		receipt.KnowledgeLineage =
			&models.MaterialContextReceipt{
				Status: models.ContextReceiptUnavailable,
				Reason: "教案数据为空，无法确认知识脉络",
			}
		return receipt
	}

	turnPlan :=
		lessonPlanTurnContextPlanFromContext(
			ctx,
		)

	stage, recipe := s.loadReceiptStageAndRecipe(
		ctx,
		lessonPlan,
		stageCode,
	)

	recipeSelectionMode :=
		resolveRecipeSelectionModeForReceipt(
			ctx,
			lessonPlan,
		)

	receipt.Recipe = buildRecipeReceipt(
		lessonPlan,
		recipe,
		recipeSelectionMode,
	)
	receipt.Textbook =
		s.buildTextbookReceipt(
			ctx,
			lessonPlan,
		)
	receipt.UnitPlan =
		buildUnitPlanReceipt(
			ctx,
			lessonPlan,
		)

	// 原始课程大纲与知识脉络分别构建。
	// buildCourseOutlineReceipt只有UseRawCourseOutline=true时才允许读取大纲。
	receipt.CourseOutline =
		buildCourseOutlineReceipt(
			ctx,
			lessonPlan,
			turnPlan,
		)

	receipt.KnowledgeLineage =
		buildKnowledgeLineageReceipt(
			ctx,
			lessonPlan,
			turnPlan,
		)

	receipt.ClassProfile =
		buildClassProfileReceipt(
			ctx,
			lessonPlan,
			stageCode,
		)

	receipt.Components =
		s.buildComponentsReceipt(
			ctx,
			lessonPlan,
			stage,
			recipe,
			stageCode,
			recentUserText,
		)

	return receipt
}

// loadReceiptStageAndRecipe 按正式提示词相同规则读取阶段和配方。
func (s *WorkshopStageService) loadReceiptStageAndRecipe(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
) (
	*models.WorkshopStage,
	*models.TeachingRecipe,
) {
	if lessonPlan == nil {
		return nil, nil
	}

	var recipe *models.TeachingRecipe
	validRecipeID := ""

	if lessonPlan.RecipeID != nil &&
		strings.TrimSpace(
			*lessonPlan.RecipeID,
		) != "" {
		candidate, _, recipeErr :=
			loadRecipeForPlanUse(
				ctx,
				lessonPlan,
			)
		if recipeErr == nil &&
			candidate != nil {
			recipe = candidate
			validRecipeID = candidate.ID
		}
	}

	isCustomStage := false
	if recipe != nil &&
		strings.TrimSpace(
			lessonPlan.StageConfig,
		) != "" &&
		lessonPlan.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal(
			[]byte(lessonPlan.StageConfig),
			&snapshots,
		) == nil {
			for _, snapshot := range snapshots {
				if snapshot.StageCode ==
					stageCode &&
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

	var (
		stage *models.WorkshopStage
		err   error
	)

	switch {
	case isCustomStage:
		stage, err =
			repository.GetRecipeStageByCode(
				ctx,
				validRecipeID,
				stageCode,
			)
		if err != nil || stage == nil {
			stage = loadSystemStage(
				stageCode,
			)
		}

	case recipe != nil:
		stage, err =
			repository.GetStageByCode(
				ctx,
				models.StageSourceRecipe,
				stageCode,
			)
		if err != nil || stage == nil {
			stage = loadSystemStage(
				stageCode,
			)
		}

	default:
		stage = loadSystemStage(
			stageCode,
		)
	}

	return stage, recipe
}

// buildRecipeReceipt 构建配方回执。
func buildRecipeReceipt(
	lessonPlan *models.LessonPlan,
	recipe *models.TeachingRecipe,
	selectionMode models.RecipeSelectionMode,
) *models.MaterialContextReceipt {
	modeText := string(selectionMode)

	if selectionMode ==
		models.RecipeSelectionModeNone {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptExplicitNone,
			SelectionMode: modeText,
			Reason: "老师已明确选择不使用配方，本轮使用系统阶段骨架",
		}
	}

	if lessonPlan == nil {
		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			SelectionMode: modeText,
			Reason: "教案数据为空，无法确认配方",
		}
	}

	if lessonPlan.RecipeID == nil ||
		strings.TrimSpace(
			*lessonPlan.RecipeID,
		) == "" {
		if selectionMode ==
			models.RecipeSelectionModeAuto {
			return &models.MaterialContextReceipt{
				Status: models.ContextReceiptNotFound,
				SelectionMode: modeText,
				Reason: "平台已自动匹配，但没有找到可用配方，本轮使用系统阶段骨架",
			}
		}

		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			SelectionMode: modeText,
			Reason: "老师选择了配方，但没有读取到有效的配方标识",
		}
	}

	recipeID := strings.TrimSpace(
		*lessonPlan.RecipeID,
	)

	if recipe == nil {
		reason :=
			"自动关联的配方当前无法读取，或已不再严格匹配本教案的学科和具体年级，本轮已忽略"

		if selectionMode ==
			models.RecipeSelectionModeSelected {
			reason =
				"老师选择的配方已停用、删除或当前账号已无权使用，本轮未读取"
		}

		return &models.MaterialContextReceipt{
			Status: models.ContextReceiptUnavailable,
			SelectionMode: modeText,
			ID: recipeID,
			Reason: reason,
		}
	}

	reason := ""
	switch selectionMode {
	case models.RecipeSelectionModeAuto:
		reason =
			"平台根据学校、教研组和学科规则自动选择"
	case models.RecipeSelectionModeSelected:
		reason =
			"老师在开始备课时明确选择"
	}

	return &models.MaterialContextReceipt{
		Status: models.ContextReceiptLoaded,
		SelectionMode: modeText,
		ID: recipe.ID,
		Name: recipe.Name,
		Reason: reason,
	}
}

