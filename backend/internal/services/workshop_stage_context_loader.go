package services

// workshop_stage_context_loader.go — 阶段提示词与资料上下文装配
//
// 本文件从workshop_stage_service.go拆出，集中负责：
//   - 配方三态解析；
//   - 系统阶段、配方覆盖阶段和自定义阶段解析；
//   - 前序阶段产出与组件选择；
//   - 课本、单元方案、课程大纲和班级学情装配；
//   - 单轮确定性上下文计划的Use*开关执行。
//
// WorkshopStageService主文件只保留阶段状态机和流转编排，避免继续膨胀。
//
// 当前资料装配规则保持兼容：
//   1. 普通讨论由lessonPlanTurnContextPlan按需加载；
//   2. 课本全文仅在正式任务或老师明确询问课本时读取；
//   3. 课程大纲原文仅在老师明确查询时读取；
//   4. 后续阶段优先使用active知识脉络短版；
//   5. 班级学情只在analyze、design和write使用；
//   6. 数据库错误和非法课本关联必须阻断，非active单元方案和学情卡静默降级。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// LoadStagePromptContext 保留旧调用签名。
func (s *WorkshopStageService) LoadStagePromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
) (string, error) {
	return s.LoadStagePromptContextV2(
		ctx,
		lessonPlan,
		stageCode,
		"",
		"",
	)
}

// LoadStagePromptContextV2 构建当前阶段完整系统提示词。
//
// assistantPrompt只作为阶段原生骨架之后的教学风格补充，不能替换流程骨架。
// recentUserText供组件知识路由根据教师当前问题做轻量精排。
func (s *WorkshopStageService) LoadStagePromptContextV2(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	assistantPrompt string,
	recentUserText string,
) (string, error) {
	if lessonPlan == nil {
		return "", errors.New("加载阶段提示词失败：教案为空")
	}

	turnPlan := lessonPlanTurnContextPlanFromContext(ctx)

	recipe, validRecipeID := s.loadLessonPlanPromptRecipe(
		ctx,
		lessonPlan,
		stageCode,
	)

	stage, err := s.loadLessonPlanPromptStage(
		ctx,
		lessonPlan,
		stageCode,
		recipe,
		validRecipeID,
	)
	if err != nil {
		return "", err
	}

	promptMode, lessonStructure := resolveLessonPlanPromptPreferences(
		lessonPlan,
		stageCode,
		recipe,
	)

	promptRecipe := recipe
	if turnPlan != nil && !turnPlan.UseRecipe {
		promptRecipe = nil
		promptMode = models.PromptModeGuided
		lessonStructure = ""
	}

	priorOutputs := loadLessonPlanPromptPriorOutputs(
		ctx,
		lessonPlan.ID,
		stageCode,
		turnPlan,
	)

	selectedComponentIDs := s.loadLessonPlanPromptSelectedComponents(
		ctx,
		lessonPlan.ID,
		stageCode,
		turnPlan,
	)

	promptStage := stage
	if turnPlan != nil && !turnPlan.UseComponents {
		stageCopy := *stage
		stageCopy.ComponentTypes = "[]"
		promptStage = &stageCopy
	}

	componentContext := withLessonComponentDomain(
		ctx,
		lessonPlan.EducationDomain,
	)

	basePrompt := BuildStageSystemPromptV2(
		componentContext,
		promptStage,
		promptRecipe,
		priorOutputs,
		lessonPlan.Subject,
		lessonPlan.Grade,
		promptMode,
		lessonStructure,
		selectedComponentIDs,
		assistantPrompt,
		recentUserText,
	)

	var capsuleInjected bool
	basePrompt, capsuleInjected =
		appendLessonPlanContextCapsulePromptContext(
			ctx,
			lessonPlan,
			stageCode,
			basePrompt,
		)

	if turnPlan != nil {
		turnPlan.UseContextCapsule = capsuleInjected

		// active胶囊已经包含课程大纲知识脉络的稳定短版时，
		// 本轮不再重复注入同一份active知识脉络。
		// 教师明确查询原始课程大纲时仍保留原文读取。
		if capsuleInjected && !turnPlan.UseRawCourseOutline {
			turnPlan.UseKnowledgeLineage = false
			turnPlan.UseCourseOutline = false
		}

		// 正式产物即使没有其它原始资料，只要使用了active胶囊，
		// 仍进入正式一致性Harness。
		if capsuleInjected && turnPlan.FormalArtifact {
			turnPlan.BlockingEvidenceHarness = true
		}
	}

	basePrompt, err = appendLessonPlanTextbookPromptContext(
		ctx,
		lessonPlan,
		stageCode,
		turnPlan,
		basePrompt,
	)
	if err != nil {
		return "", err
	}

	var unitPlanInjected bool
	basePrompt, unitPlanInjected = appendLessonPlanUnitPlanPromptContext(
		ctx,
		lessonPlan,
		stageCode,
		turnPlan,
		basePrompt,
	)

	basePrompt, err = appendLessonPlanCourseOutlinePromptContext(
		ctx,
		lessonPlan,
		stageCode,
		turnPlan,
		unitPlanInjected,
		basePrompt,
	)
	if err != nil {
		return "", err
	}

	basePrompt = appendLessonPlanClassProfilePromptContext(
		ctx,
		lessonPlan,
		stageCode,
		turnPlan,
		basePrompt,
	)

	return basePrompt, nil
}

// loadLessonPlanPromptRecipe 按教案保存的配方三态重新验证配方。
func (s *WorkshopStageService) loadLessonPlanPromptRecipe(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
) (*models.TeachingRecipe, string) {
	if lessonPlan.RecipeID == nil ||
		strings.TrimSpace(*lessonPlan.RecipeID) == "" {
		return nil, ""
	}

	candidate, selectionMode, err := loadRecipeForPlanUse(
		ctx,
		lessonPlan,
	)
	if err != nil {
		wsLog.Warn(
			"教案关联配方当前不可用，本轮忽略",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"recipe_id", *lessonPlan.RecipeID,
			"recipe_mode", selectionMode,
			"subject", lessonPlan.Subject,
			"grade", lessonPlan.Grade,
			"error", err,
		)
		return nil, ""
	}

	if candidate == nil {
		return nil, ""
	}

	return candidate, candidate.ID
}

// loadLessonPlanPromptStage 解析本轮使用的正式阶段定义。
func (s *WorkshopStageService) loadLessonPlanPromptStage(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	recipe *models.TeachingRecipe,
	validRecipeID string,
) (*models.WorkshopStage, error) {
	isCustomStage := false

	if recipe != nil &&
		lessonPlan.StageConfig != "" &&
		lessonPlan.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot

		if json.Unmarshal(
			[]byte(lessonPlan.StageConfig),
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
	) (*models.WorkshopStage, error) {
		stage, err := repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			code,
		)
		if err == nil && stage != nil {
			return stage, nil
		}

		if code == "write" {
			return nil, err
		}

		wsLog.Warn(
			"系统不存在当前阶段定义，退回系统教案撰写阶段",
			"plan_id", lessonPlan.ID,
			"requested_stage", code,
			"fallback_stage", "write",
			"error", err,
		)

		return repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			"write",
		)
	}

	var (
		stage *models.WorkshopStage
		err   error
	)

	switch {
	case isCustomStage:
		stage, err = repository.GetRecipeStageByCode(
			ctx,
			validRecipeID,
			stageCode,
		)
		if err != nil || stage == nil {
			wsLog.Warn(
				"有效配方的自定义阶段加载失败，退回系统阶段",
				"plan_id", lessonPlan.ID,
				"recipe_id", validRecipeID,
				"stage", stageCode,
				"error", err,
			)
			stage, err = loadSystemStage(stageCode)
		}

	case recipe != nil:
		stage, err = repository.GetStageByCode(
			ctx,
			models.StageSourceRecipe,
			stageCode,
		)
		if err != nil || stage == nil {
			stage, err = loadSystemStage(stageCode)
		}

	default:
		stage, err = loadSystemStage(stageCode)
	}

	if err != nil {
		return nil, fmt.Errorf("加载阶段定义失败: %w", err)
	}
	if stage == nil {
		return nil, errors.New("加载阶段定义失败：阶段定义为空")
	}

	return stage, nil
}

// resolveLessonPlanPromptPreferences 解析配方和阶段级提示模式。
func resolveLessonPlanPromptPreferences(
	lessonPlan *models.LessonPlan,
	stageCode string,
	recipe *models.TeachingRecipe,
) (string, string) {
	promptMode := models.PromptModeGuided
	lessonStructure := ""

	if recipe != nil {
		if recipe.PromptMode != "" {
			promptMode = recipe.PromptMode
		}
		if recipe.LessonStructure != "" &&
			recipe.LessonStructure != "[]" {
			lessonStructure = recipe.LessonStructure
		}
	}

	if recipe == nil ||
		lessonPlan.StageConfig == "" ||
		lessonPlan.StageConfig == "[]" {
		return promptMode, lessonStructure
	}

	var snapshots []models.StageConfigSnapshot
	if json.Unmarshal(
		[]byte(lessonPlan.StageConfig),
		&snapshots,
	) != nil {
		return promptMode, lessonStructure
	}

	for _, snapshot := range snapshots {
		if snapshot.StageCode == stageCode &&
			snapshot.PromptModeOverride != "" {
			promptMode = snapshot.PromptModeOverride
			break
		}
	}

	return promptMode, lessonStructure
}

// loadLessonPlanPromptPriorOutputs 按单轮计划读取前序阶段产出。
func loadLessonPlanPromptPriorOutputs(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
) []*models.WorkshopStageOutput {
	if turnPlan != nil && !turnPlan.UsePriorOutputs {
		return nil
	}

	outputs, err := repository.ListStageOutputs(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		wsLog.Warn(
			"读取前序阶段产出失败，本轮使用当前阶段上下文继续",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
		return nil
	}

	priorOutputs := make(
		[]*models.WorkshopStageOutput,
		0,
		len(outputs),
	)

	for _, output := range outputs {
		if output == nil {
			continue
		}
		if output.StageCode == stageCode {
			break
		}
		priorOutputs = append(priorOutputs, output)
	}

	return priorOutputs
}

// loadLessonPlanPromptSelectedComponents 读取教师为当前阶段选择的组件。
func (s *WorkshopStageService) loadLessonPlanPromptSelectedComponents(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
) []string {
	if turnPlan != nil && !turnPlan.UseComponents {
		return nil
	}

	componentIDs := s.getSelectedComponentIDsFromOutput(
		ctx,
		lessonPlanID,
		stageCode,
	)

	if len(componentIDs) > 0 {
		wsLog.Info(
			"检测到用户选中的阶段组件",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"selected_count", len(componentIDs),
		)
	}

	return componentIDs
}

// appendLessonPlanContextCapsulePromptContext 装配active核心共识胶囊。
//
// 该读取只有一次数据库查询，不调用AI、不读取原文，也不等待旁路更新。
// 胶囊读取失败时记录日志并继续走现有知识脉络兜底，不能阻塞主回复。
func appendLessonPlanContextCapsulePromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	basePrompt string,
) (string, bool) {
	capsuleContext, capsule, err :=
		BuildLessonPlanContextCapsuleRuntime(
			ctx,
			lessonPlan,
		)
	if err != nil {
		wsLog.Warn(
			"active核心共识胶囊读取失败，本轮继续使用现有上下文兜底",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"error", err,
		)
		return basePrompt, false
	}

	if capsule == nil ||
		strings.TrimSpace(capsuleContext) == "" {
		return basePrompt, false
	}

	basePrompt += capsuleContext

	wsLog.Info(
		"已注入active备课核心共识胶囊",
		"plan_id", lessonPlan.ID,
		"stage", stageCode,
		"capsule_version", capsule.Version,
		"context_runes", len([]rune(capsuleContext)),
	)

	return basePrompt, true
}

// appendLessonPlanTextbookPromptContext 装配课本原文。
func appendLessonPlanTextbookPromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
	basePrompt string,
) (string, error) {
	useTextbook := strings.TrimSpace(
		lessonPlan.TextbookPageIDs,
	) != "" && lessonPlan.TextbookPageIDs != "[]"

	if turnPlan != nil {
		useTextbook = useTextbook && turnPlan.UseTextbook
	}
	if !useTextbook {
		return basePrompt, nil
	}

	textbookContext, err := BuildLessonPlanTextbookContext(
		ctx,
		lessonPlan,
	)
	if err != nil {
		return "", fmt.Errorf(
			"加载课本原文上下文失败: %w",
			err,
		)
	}

	if strings.TrimSpace(textbookContext) == "" {
		return basePrompt, nil
	}

	basePrompt += "\n" + textbookContext

	wsLog.Info(
		"已通过K12运行时硬闸注入课本原文上下文",
		"plan_id", lessonPlan.ID,
		"stage", stageCode,
		"education_domain", lessonPlan.EducationDomain,
	)

	return basePrompt, nil
}

// appendLessonPlanUnitPlanPromptContext 装配显式挂载的active单元方案。
func appendLessonPlanUnitPlanPromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
	basePrompt string,
) (string, bool) {
	useUnitPlan := lessonPlan.UnitPlanID != nil &&
		strings.TrimSpace(*lessonPlan.UnitPlanID) != ""

	if turnPlan != nil {
		useUnitPlan = useUnitPlan && turnPlan.UseUnitPlan
	}
	if !useUnitPlan {
		return basePrompt, false
	}

	unitPlan, err := repository.GetUnitPlanByID(
		ctx,
		*lessonPlan.UnitPlanID,
	)
	if err != nil {
		wsLog.Warn(
			"已挂载单元方案但查询失败，跳过单元方案注入",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"unit_plan_id", *lessonPlan.UnitPlanID,
			"error", err,
		)
		return basePrompt, false
	}

	if unitPlan.Status != models.UnitPlanStatusActive {
		wsLog.Info(
			"已挂载单元方案但非active状态，跳过单元方案注入",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"unit_plan_id", *lessonPlan.UnitPlanID,
			"status", unitPlan.Status,
		)
		return basePrompt, false
	}

	unitPlanContext := BuildUnitPlanContext(unitPlan)
	if strings.TrimSpace(unitPlanContext) == "" {
		return basePrompt, false
	}

	basePrompt += unitPlanContext

	wsLog.Info(
		"已注入单元方案上下文",
		"plan_id", lessonPlan.ID,
		"stage", stageCode,
		"unit_plan_id", unitPlan.ID,
		"unit", unitPlan.Unit,
		"unit_theme", unitPlan.UnitTheme,
	)

	return basePrompt, true
}

// appendLessonPlanCourseOutlinePromptContext 装配原始大纲或active知识脉络。
func appendLessonPlanCourseOutlinePromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
	unitPlanInjected bool,
	basePrompt string,
) (string, error) {
	useCourseOutline := (stageCode == "analyze" ||
		stageCode == "design") &&
		!unitPlanInjected

	if turnPlan != nil {
		useCourseOutline = turnPlan.UseCourseOutline
	}

	if !useCourseOutline {
		if lessonPlan.CourseOutlinePublisher != nil {
			wsLog.Info(
				"课程大纲已关联但本轮按需跳过",
				"plan_id", lessonPlan.ID,
				"stage", stageCode,
			)
		}
		return basePrompt, nil
	}

	if lessonPlan.CourseOutlinePublisher == nil {
		wsLog.Info(
			"本轮规划要求课程大纲，但教案未关联",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
		)
		return basePrompt, nil
	}

	outlineContext, outlines, err := BuildLessonPlanCourseOutlineContext(
		ctx,
		lessonPlan,
	)
	if err != nil {
		return "", fmt.Errorf(
			"加载课程大纲上下文失败: %w",
			err,
		)
	}

	if strings.TrimSpace(outlineContext) == "" {
		return basePrompt, nil
	}

	basePrompt += outlineContext

	titles := make([]string, 0, len(outlines))
	for _, outline := range outlines {
		if outline != nil &&
			strings.TrimSpace(outline.Title) != "" {
			titles = append(titles, outline.Title)
		}
	}

	wsLog.Info(
		"已按单轮上下文规划注入课程层级上下文",
		"plan_id", lessonPlan.ID,
		"stage", stageCode,
		"subject", lessonPlan.Subject,
		"plan_grade", lessonPlan.Grade,
		"education_domain", lessonPlan.EducationDomain,
		"outline_count", len(outlines),
		"outline_titles", strings.Join(titles, " | "),
	)

	return basePrompt, nil
}

// appendLessonPlanClassProfilePromptContext 装配匿名群体学情。
func appendLessonPlanClassProfilePromptContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
	basePrompt string,
) string {
	useClassProfile := (stageCode == "analyze" ||
		stageCode == "design" ||
		stageCode == "write") &&
		lessonPlan.ClassProfileID != nil &&
		strings.TrimSpace(*lessonPlan.ClassProfileID) != ""

	if turnPlan != nil {
		useClassProfile = turnPlan.UseClassProfile &&
			lessonPlan.ClassProfileID != nil &&
			strings.TrimSpace(*lessonPlan.ClassProfileID) != ""
	}
	if !useClassProfile {
		return basePrompt
	}

	classProfile, err := repository.GetClassProfileByID(
		ctx,
		*lessonPlan.ClassProfileID,
	)
	if err != nil {
		wsLog.Warn(
			"已挂载班级学情卡但查询失败，跳过班级学情注入",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"class_profile_id", *lessonPlan.ClassProfileID,
			"error", err,
		)
		return basePrompt
	}

	if classProfile.Status != models.ClassProfileStatusActive {
		wsLog.Info(
			"已挂载班级学情卡但非active状态，跳过班级学情注入",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"class_profile_id", classProfile.ID,
			"status", classProfile.Status,
		)
		return basePrompt
	}

	if classProfile.CreatedBy != lessonPlan.AuthorID {
		wsLog.Warn(
			"挂载的班级学情卡归属与教案作者不一致",
			"plan_id", lessonPlan.ID,
			"stage", stageCode,
			"class_profile_id", classProfile.ID,
			"card_owner", classProfile.CreatedBy,
			"plan_author", lessonPlan.AuthorID,
		)
		return basePrompt
	}

	classProfileContext := BuildClassProfileContext(classProfile)
	if strings.TrimSpace(classProfileContext) == "" {
		return basePrompt
	}

	basePrompt += classProfileContext

	wsLog.Info(
		"已注入班级学情上下文",
		"plan_id", lessonPlan.ID,
		"stage", stageCode,
		"class_profile_id", classProfile.ID,
		"class_name", classProfile.ClassName,
	)

	return basePrompt
}
