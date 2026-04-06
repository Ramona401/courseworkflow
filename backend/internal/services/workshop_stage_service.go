package services

// workshop_stage_service.go — 阶段化备课工坊业务逻辑层
//
// v74重大改动：
//   1. 所有阶段进入时都自动发AI开场白（不仅限review）
//      AI自我介绍角色、说明拿到了什么前序成果、告诉用户接下来要做什么
//   2. review阶段仍然自动触发评审（开场白即为评审报告）
//   3. 其他阶段（design/write/revise）的开场白是引导性的，用户看到后自然对话

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrStageNotInitialized = errors.New("教案尚未初始化阶段配置")
	ErrStageAlreadyFirst   = errors.New("已经是第一个阶段，无法回退")
	ErrStageAlreadyLast    = errors.New("已经是最后一个阶段")
	ErrStageNotSkippable   = errors.New("当前阶段不可跳过")
	ErrStageInvalidTarget  = errors.New("目标阶段不存在")
	ErrCustomStageLimit    = errors.New("自定义阶段数量已达上限（最多10个）")
)

// autoTriggerStages 进入后自动触发Chat的阶段及对应的触发消息
//
// v74改动：所有非首阶段都自动触发，让AI主动打招呼
//   - design/write/revise：AI发一条友好的开场白，介绍自己、说明前序成果、引导用户
//   - review：AI直接输出评审报告（保持原有行为）
//
// 触发消息会作为"用户消息"发给AI，AI基于系统提示词中的角色设定和前序产出物来回复
var autoTriggerStages = map[string]string{
	"design": "我们进入教学设计阶段了。请先简要介绍你是谁、你拿到了哪些前序阶段的分析成果，然后告诉我接下来你会带我做什么。用友好的口吻，不超过200字。",
	"write":  "我们进入教案撰写阶段了。请先简要介绍你是谁、你拿到了哪些前序阶段的设计方案，然后告诉我接下来你会怎么帮我写教案。用友好的口吻，不超过200字。",
	"review": "请对上一阶段完成的教案进行全面专业评审，直接输出评审报告，包含各维度评分和改进建议。",
	"revise": "我们进入修订定稿阶段了。请先简要介绍你是谁、你拿到了评审报告中的哪些改进建议，然后告诉我接下来你会怎么帮我修订教案。用友好的口吻，不超过200字。",
}

// ==================== 服务结构体 ====================

// WorkshopStageService 阶段化备课工坊服务
type WorkshopStageService struct {
	recipeService *RecipeService
	// genService 在运行时注入，避免循环依赖（workshop_stage_service ↔ lesson_plan_gen_service）
	genService interface {
		Chat(ctx context.Context, req *models.LessonPlanChatRequest, callerID string) error
	}
}

var wsLog = logger.WithModule("workshop_stage")

// NewWorkshopStageService 创建阶段服务实例
func NewWorkshopStageService() *WorkshopStageService {
	return &WorkshopStageService{
		recipeService: NewRecipeService(),
	}
}

// SetGenService 注入生成服务（由routes层调用，避免循环依赖）
func (s *WorkshopStageService) SetGenService(gs interface {
	Chat(ctx context.Context, req *models.LessonPlanChatRequest, callerID string) error
}) {
	s.genService = gs
}

// ==================== 1. 获取系统默认阶段 ====================

func (s *WorkshopStageService) GetDefaultStages(ctx context.Context) (*models.DefaultStagesResponse, error) {
	stages, err := repository.GetSystemDefaultStages(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取默认阶段失败: %w", err)
	}

	var items []*models.DefaultStageItem
	for _, st := range stages {
		items = append(items, &models.DefaultStageItem{
			StageCode:      st.StageCode,
			StageName:      st.StageName,
			StageOrder:     st.StageOrder,
			AIRole:         st.AIRole,
			GateMode:       st.GateMode,
			Skippable:      st.Skippable,
			ComponentTypes: st.ComponentTypes,
		})
	}
	if items == nil {
		items = []*models.DefaultStageItem{}
	}
	return &models.DefaultStagesResponse{Stages: items}, nil
}

// ==================== 2. 获取教案阶段进度 ====================

func (s *WorkshopStageService) GetStageStatus(ctx context.Context, lessonPlanID string, callerID string) (*models.StageStatusResponse, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}
	if len(snapshots) == 0 {
		return nil, ErrStageNotInitialized
	}

	outputs, _ := repository.ListStageOutputs(ctx, lessonPlanID)
	outputMap := make(map[string]*models.WorkshopStageOutput)
	for _, out := range outputs {
		outputMap[out.StageCode] = out
	}

	var items []*models.StageProgressItem
	for _, snap := range snapshots {
		item := &models.StageProgressItem{
			StageCode:  snap.StageCode,
			StageName:  snap.StageName,
			StageOrder: snap.StageOrder,
			AIRole:     snap.AIRole,
			GateMode:   snap.GateMode,
			Skippable:  snap.Skippable,
			Status:     "pending",
			IsCustom:   snap.IsCustom,
		}
		if out, ok := outputMap[snap.StageCode]; ok {
			item.Status = out.Status
			item.HasOutput = out.StructuredOutput != "" && out.StructuredOutput != "{}"
			item.CompletedAt = out.CompletedAt
		}
		items = append(items, item)
	}

	return &models.StageStatusResponse{
		CurrentStage: lp.CurrentStage,
		TotalStages:  len(snapshots),
		Stages:       items,
	}, nil
}

// ==================== 3. 获取阶段产出物 ====================

func (s *WorkshopStageService) GetStageOutput(ctx context.Context, lessonPlanID string, stageCode string, callerID string) (*models.StageOutputResponse, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	out, err := repository.GetStageOutput(ctx, lessonPlanID, stageCode)
	if err != nil {
		return nil, err
	}

	return &models.StageOutputResponse{
		StageCode:        out.StageCode,
		StageName:        stageCodeToName(out.StageCode),
		StructuredOutput: out.StructuredOutput,
		NarrativeOutput:  out.NarrativeOutput,
		Status:           out.Status,
		ModelUsed:        out.ModelUsed,
		TokensUsed:       out.TokensUsed,
	}, nil
}

// ==================== 4. 阶段合并算法 ====================

func (s *WorkshopStageService) MergeStages(ctx context.Context, recipeStagesConfig string, recipeID string) ([]models.StageConfigSnapshot, error) {
	defaults, err := repository.GetSystemDefaultStages(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载默认阶段失败: %w", err)
	}

	defaultSnapshots := buildDefaultSnapshots(defaults)

	if recipeStagesConfig == "" || recipeStagesConfig == "[]" {
		return defaultSnapshots, nil
	}

	format := detectStagesConfigFormat(recipeStagesConfig)

	switch format {
	case "new":
		return s.mergeStagesNewFormat(defaultSnapshots, defaults, recipeStagesConfig, recipeID, ctx)
	case "legacy":
		return s.mergeStagesLegacyFormat(defaultSnapshots, recipeStagesConfig)
	default:
		wsLog.Warn("无法识别的stages_config格式，使用默认阶段", "config", recipeStagesConfig)
		return defaultSnapshots, nil
	}
}

func (s *WorkshopStageService) mergeStagesNewFormat(
	defaultSnapshots []models.StageConfigSnapshot,
	defaults []*models.WorkshopStage,
	configJSON string,
	recipeID string,
	ctx context.Context,
) ([]models.StageConfigSnapshot, error) {
	var flowItems []models.StageFlowItem
	if err := json.Unmarshal([]byte(configJSON), &flowItems); err != nil {
		wsLog.Warn("解析新格式stages_config失败，使用默认阶段", "error", err)
		return defaultSnapshots, nil
	}

	defaultMap := make(map[string]*models.WorkshopStage)
	for _, d := range defaults {
		defaultMap[d.StageCode] = d
	}

	customMap := make(map[string]*models.WorkshopStage)
	if recipeID != "" {
		customStages, err := repository.GetRecipeStages(ctx, recipeID)
		if err == nil {
			for _, cs := range customStages {
				customMap[cs.StageCode] = cs
			}
		}
	}

	var merged []models.StageConfigSnapshot
	for _, item := range flowItems {
		if !item.Enabled {
			continue
		}

		if item.IsCustom {
			cs, ok := customMap[item.StageCode]
			if !ok {
				wsLog.Warn("配方引用了不存在的自定义阶段，跳过", "stage_code", item.StageCode)
				continue
			}
			snap := models.StageConfigSnapshot{
				StageCode:  cs.StageCode,
				StageName:  cs.StageName,
				StageOrder: item.Order,
				AIRole:     cs.AIRole,
				GateMode:   cs.GateMode,
				Skippable:  cs.Skippable,
				IsCustom:   true,
			}
			if item.PromptMode != "" {
				snap.PromptModeOverride = item.PromptMode
			}
			merged = append(merged, snap)
		} else {
			d, ok := defaultMap[item.StageCode]
			if !ok {
				wsLog.Warn("配方引用了未知系统阶段，跳过", "stage_code", item.StageCode)
				continue
			}
			snap := models.StageConfigSnapshot{
				StageCode:  d.StageCode,
				StageName:  d.StageName,
				StageOrder: item.Order,
				AIRole:     d.AIRole,
				GateMode:   d.GateMode,
				Skippable:  d.Skippable,
				IsCustom:   false,
			}
			if item.PromptMode != "" {
				snap.PromptModeOverride = item.PromptMode
			}
			merged = append(merged, snap)
		}
	}

	if len(merged) == 0 {
		wsLog.Warn("新格式stages_config过滤后无启用阶段，使用默认阶段")
		return defaultSnapshots, nil
	}

	for i := 0; i < len(merged); i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[j].StageOrder < merged[i].StageOrder {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}

	for i := range merged {
		merged[i].StageOrder = i + 1
	}

	return merged, nil
}

func (s *WorkshopStageService) mergeStagesLegacyFormat(
	merged []models.StageConfigSnapshot,
	configJSON string,
) ([]models.StageConfigSnapshot, error) {
	var overrides []models.StageOverride
	if err := json.Unmarshal([]byte(configJSON), &overrides); err != nil {
		wsLog.Warn("解析旧格式阶段覆盖失败，使用默认阶段", "error", err)
		return merged, nil
	}

	for _, ov := range overrides {
		switch ov.Action {
		case models.StageActionSkip:
			merged = removeStage(merged, ov.StageCode)

		case models.StageActionOverride, models.StageActionReplacePrompt:
			for i := range merged {
				if merged[i].StageCode == ov.StageCode {
					if ov.StageName != "" {
						merged[i].StageName = ov.StageName
					}
					if ov.AIRole != "" {
						merged[i].AIRole = ov.AIRole
					}
					if ov.GateMode != "" {
						merged[i].GateMode = ov.GateMode
					}
					if ov.Skippable != nil {
						merged[i].Skippable = *ov.Skippable
					}
					break
				}
			}

		case models.StageActionInsertAfter:
			insertIdx := -1
			for i := range merged {
				if merged[i].StageCode == ov.InsertAfter {
					insertIdx = i + 1
					break
				}
			}
			if insertIdx >= 0 {
				newStage := models.StageConfigSnapshot{
					StageCode: ov.StageCode,
					StageName: ov.StageName,
					AIRole:    ov.AIRole,
					GateMode:  ov.GateMode,
					Skippable: true,
				}
				if ov.Skippable != nil {
					newStage.Skippable = *ov.Skippable
				}
				if newStage.GateMode == "" {
					newStage.GateMode = models.StageGateSuggest
				}
				merged = append(merged, models.StageConfigSnapshot{})
				copy(merged[insertIdx+1:], merged[insertIdx:])
				merged[insertIdx] = newStage
			}
		}
	}

	for i := range merged {
		merged[i].StageOrder = i + 1
	}

	return merged, nil
}

// ==================== 5. 为教案初始化阶段配置 ====================

func (s *WorkshopStageService) InitStagesForPlan(ctx context.Context, lessonPlanID string, recipeStagesConfig string, recipeID string) ([]models.StageConfigSnapshot, error) {
	snapshots, err := s.MergeStages(ctx, recipeStagesConfig, recipeID)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, errors.New("合并后阶段列表为空")
	}

	configJSON, _ := json.Marshal(snapshots)
	if err := repository.UpdateLessonPlanStageConfig(ctx, lessonPlanID, string(configJSON)); err != nil {
		return nil, fmt.Errorf("写入阶段配置失败: %w", err)
	}

	firstStage := snapshots[0]
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, firstStage.StageCode); err != nil {
		return nil, fmt.Errorf("设置初始阶段失败: %w", err)
	}

	output := &models.WorkshopStageOutput{
		LessonPlanID:         lessonPlanID,
		StageCode:            firstStage.StageCode,
		StageOrder:           firstStage.StageOrder,
		StructuredOutput:     "{}",
		NarrativeOutput:      "",
		ConversationSnapshot: "[]",
		Status:               models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		return nil, fmt.Errorf("创建初始阶段产出记录失败: %w", err)
	}

	wsLog.Info("教案阶段初始化完成",
		"plan_id", lessonPlanID,
		"stages_count", len(snapshots),
		"first_stage", firstStage.StageCode,
	)
	return snapshots, nil
}

// ==================== 6. 进入下一阶段 ====================

// AdvanceStage 进入下一阶段
//
// v74改动：所有非首阶段都自动触发AI开场白
//   进入后异步调用genService.Chat发送触发消息，
//   AI基于阶段系统提示词和前序产出物生成开场白。
func (s *WorkshopStageService) AdvanceStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}

	var targetIdx int
	if targetStageCode != "" {
		targetIdx = findStageIndex(snapshots, targetStageCode)
		if targetIdx == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else {
		targetIdx = currentIdx + 1
		if targetIdx >= len(snapshots) {
			return nil, ErrStageAlreadyLast
		}
	}

	// 完成当前阶段产出
	_ = repository.CompleteStageOutput(ctx, lessonPlanID, lp.CurrentStage, "[]")

	// 创建目标阶段产出记录
	targetStage := snapshots[targetIdx]
	output := &models.WorkshopStageOutput{
		LessonPlanID:         lessonPlanID,
		StageCode:            targetStage.StageCode,
		StageOrder:           targetStage.StageOrder,
		StructuredOutput:     "{}",
		NarrativeOutput:      "",
		ConversationSnapshot: "[]",
		Status:               models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn("创建阶段产出记录失败（可能已存在）", "error", err)
	}

	// 更新current_stage
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}

	wsLog.Info("进入下一阶段",
		"plan_id", lessonPlanID,
		"from", lp.CurrentStage,
		"to", targetStage.StageCode,
	)

	// v74：自动触发机制——所有配置了触发消息的阶段都自动发Chat
	if triggerMsg, needsTrigger := autoTriggerStages[targetStage.StageCode]; needsTrigger && s.genService != nil {
		wsLog.Info("自动触发阶段AI开场白",
			"plan_id", lessonPlanID,
			"stage", targetStage.StageCode,
		)
		go func() {
			// 等待100ms让SSE连接先建立
			time.Sleep(100 * time.Millisecond)
			bgCtx := context.Background()
			req := &models.LessonPlanChatRequest{
				PlanID:  lessonPlanID,
				Message: triggerMsg,
			}
			if err := s.genService.Chat(bgCtx, req, callerID); err != nil {
				wsLog.Warn("自动触发阶段AI开场白失败",
					"plan_id", lessonPlanID,
					"stage", targetStage.StageCode,
					"error", err,
				)
			}
		}()
	}

	return &targetStage, nil
}

// ==================== 7. 跳过当前阶段 ====================

func (s *WorkshopStageService) SkipStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}

	if !snapshots[currentIdx].Skippable {
		return nil, ErrStageNotSkippable
	}

	_ = repository.SkipStageOutput(ctx, lessonPlanID, lp.CurrentStage)

	var targetIdx int
	if targetStageCode != "" {
		targetIdx = findStageIndex(snapshots, targetStageCode)
		if targetIdx == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else {
		targetIdx = currentIdx + 1
		if targetIdx >= len(snapshots) {
			return nil, ErrStageAlreadyLast
		}
	}

	targetStage := snapshots[targetIdx]
	output := &models.WorkshopStageOutput{
		LessonPlanID:         lessonPlanID,
		StageCode:            targetStage.StageCode,
		StageOrder:           targetStage.StageOrder,
		StructuredOutput:     "{}",
		NarrativeOutput:      "",
		ConversationSnapshot: "[]",
		Status:               models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn("创建阶段产出记录失败（可能已存在）", "error", err)
	}

	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}

	wsLog.Info("跳过阶段",
		"plan_id", lessonPlanID,
		"skipped", lp.CurrentStage,
		"to", targetStage.StageCode,
	)

	// 跳过时也触发自动AI开场白（如果目标阶段需要）
	if triggerMsg, needsTrigger := autoTriggerStages[targetStage.StageCode]; needsTrigger && s.genService != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			bgCtx := context.Background()
			req := &models.LessonPlanChatRequest{
				PlanID:  lessonPlanID,
				Message: triggerMsg,
			}
			if err := s.genService.Chat(bgCtx, req, callerID); err != nil {
				wsLog.Warn("跳过后自动触发AI开场白失败", "plan_id", lessonPlanID, "error", err)
			}
		}()
	}

	return &targetStage, nil
}

// ==================== 8. 回退到上一阶段 ====================

func (s *WorkshopStageService) BackStage(ctx context.Context, lessonPlanID string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}

	if currentIdx <= 0 {
		return nil, ErrStageAlreadyFirst
	}

	targetStage := snapshots[currentIdx-1]

	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("回退阶段失败: %w", err)
	}

	wsLog.Info("回退阶段",
		"plan_id", lessonPlanID,
		"from", lp.CurrentStage,
		"to", targetStage.StageCode,
	)
	return &targetStage, nil
}

// ==================== 9. 保存阶段产出物 ====================

func (s *WorkshopStageService) SaveStageOutput(ctx context.Context, lessonPlanID string, stageCode string, structuredJSON string, narrative string, modelUsed string, tokensUsed int) error {
	return repository.UpdateStageOutputContent(ctx, lessonPlanID, stageCode, structuredJSON, narrative, modelUsed, tokensUsed)
}

// ==================== 10. 加载阶段完整提示词上下文 ====================

func (s *WorkshopStageService) LoadStagePromptContext(ctx context.Context, lp *models.LessonPlan, stageCode string) (string, error) {
	isCustomStage := false
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal([]byte(lp.StageConfig), &snapshots) == nil {
			for _, snap := range snapshots {
				if snap.StageCode == stageCode && snap.IsCustom {
					isCustomStage = true
					break
				}
			}
		}
	}

	var stage *models.WorkshopStage
	var err error

	if isCustomStage && lp.RecipeID != nil && *lp.RecipeID != "" {
		stage, err = repository.GetRecipeStageByCode(ctx, *lp.RecipeID, stageCode)
		if err != nil {
			wsLog.Warn("自定义阶段加载失败，尝试系统默认", "stage_code", stageCode, "error", err)
			stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
		}
	} else if lp.RecipeID != nil && *lp.RecipeID != "" {
		stage, err = repository.GetStageByCode(ctx, models.StageSourceRecipe, stageCode)
		if err != nil {
			stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
		}
	} else {
		stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
	}
	if err != nil {
		return "", fmt.Errorf("加载阶段定义失败: %w", err)
	}

	var recipe *models.TeachingRecipe
	promptMode := models.PromptModeGuided
	lessonStructure := ""

	if lp.RecipeID != nil && *lp.RecipeID != "" {
		recipe, _ = repository.GetRecipeByID(ctx, *lp.RecipeID)
		if recipe != nil {
			if recipe.PromptMode != "" {
				promptMode = recipe.PromptMode
			}
			if recipe.LessonStructure != "" && recipe.LessonStructure != "[]" {
				lessonStructure = recipe.LessonStructure
			}
		}
	}

	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal([]byte(lp.StageConfig), &snapshots) == nil {
			for _, snap := range snapshots {
				if snap.StageCode == stageCode && snap.PromptModeOverride != "" {
					promptMode = snap.PromptModeOverride
					break
				}
			}
		}
	}

	allOutputs, _ := repository.ListStageOutputs(ctx, lp.ID)
	var priorOutputs []*models.WorkshopStageOutput
	for _, out := range allOutputs {
		if out.StageCode == stageCode {
			break
		}
		priorOutputs = append(priorOutputs, out)
	}

	return BuildStageSystemPrompt(ctx, stage, recipe, priorOutputs, lp.Subject, lp.Grade, promptMode, lessonStructure), nil
}

// ==================== 自定义阶段 CRUD ====================

func (s *WorkshopStageService) CreateCustomStage(ctx context.Context, recipeID string, req *models.CreateCustomStageRequest, callerID string) (*models.CustomStageResponse, error) {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return nil, ErrRecipeUnauthorized
	}

	if strings.TrimSpace(req.StageCode) == "" {
		return nil, errors.New("阶段代码不能为空")
	}
	if strings.TrimSpace(req.StageName) == "" {
		return nil, errors.New("阶段名称不能为空")
	}
	if strings.TrimSpace(req.AIRole) == "" {
		return nil, errors.New("AI角色不能为空")
	}

	for _, ch := range req.StageCode {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return nil, errors.New("阶段代码仅允许小写英文、数字和下划线")
		}
	}

	count, err := repository.CountRecipeStages(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, ErrCustomStageLimit
	}

	stage, err := repository.CreateRecipeStage(ctx, recipeID, req)
	if err != nil {
		if errors.Is(err, repository.ErrStageCodeConflict) {
			return nil, errors.New("阶段代码已存在，请使用其他代码")
		}
		return nil, err
	}

	wsLog.Info("创建自定义阶段成功",
		"recipe_id", recipeID,
		"stage_code", stage.StageCode,
		"stage_name", stage.StageName,
	)

	return &models.CustomStageResponse{
		StageCode: stage.StageCode,
		StageName: stage.StageName,
		AIRole:    stage.AIRole,
		GateMode:  stage.GateMode,
		Skippable: stage.Skippable,
		HasPrompt: stage.SystemPrompt != "",
	}, nil
}

func (s *WorkshopStageService) UpdateCustomStage(ctx context.Context, recipeID string, stageCode string, req *models.UpdateCustomStageRequest, callerID string) error {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}

	if strings.TrimSpace(req.StageName) == "" {
		return errors.New("阶段名称不能为空")
	}
	if strings.TrimSpace(req.AIRole) == "" {
		return errors.New("AI角色不能为空")
	}

	if err := repository.UpdateRecipeStage(ctx, recipeID, stageCode, req); err != nil {
		return err
	}

	wsLog.Info("更新自定义阶段成功", "recipe_id", recipeID, "stage_code", stageCode)
	return nil
}

func (s *WorkshopStageService) DeleteCustomStage(ctx context.Context, recipeID string, stageCode string, callerID string) error {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}

	if err := repository.DeleteRecipeStage(ctx, recipeID, stageCode); err != nil {
		return err
	}

	wsLog.Info("删除自定义阶段成功", "recipe_id", recipeID, "stage_code", stageCode)
	return nil
}

func (s *WorkshopStageService) ListCustomStages(ctx context.Context, recipeID string) ([]*models.CustomStageResponse, error) {
	stages, err := repository.GetRecipeStages(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	var items []*models.CustomStageResponse
	for _, st := range stages {
		items = append(items, &models.CustomStageResponse{
			StageCode: st.StageCode,
			StageName: st.StageName,
			AIRole:    st.AIRole,
			GateMode:  st.GateMode,
			Skippable: st.Skippable,
			HasPrompt: st.SystemPrompt != "",
		})
	}
	if items == nil {
		items = []*models.CustomStageResponse{}
	}
	return items, nil
}

// ==================== 内部辅助函数 ====================

func (s *WorkshopStageService) resolveStages(lp *models.LessonPlan) ([]models.StageConfigSnapshot, int, error) {
	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}
	if len(snapshots) == 0 {
		return nil, -1, ErrStageNotInitialized
	}

	currentIdx := findStageIndex(snapshots, lp.CurrentStage)
	if currentIdx == -1 {
		return nil, -1, fmt.Errorf("当前阶段 %s 不在配置中", lp.CurrentStage)
	}

	return snapshots, currentIdx, nil
}

func findStageIndex(snapshots []models.StageConfigSnapshot, stageCode string) int {
	for i, s := range snapshots {
		if s.StageCode == stageCode {
			return i
		}
	}
	return -1
}

func removeStage(snapshots []models.StageConfigSnapshot, stageCode string) []models.StageConfigSnapshot {
	result := make([]models.StageConfigSnapshot, 0, len(snapshots))
	for _, s := range snapshots {
		if s.StageCode != stageCode {
			result = append(result, s)
		}
	}
	return result
}

func buildDefaultSnapshots(defaults []*models.WorkshopStage) []models.StageConfigSnapshot {
	merged := make([]models.StageConfigSnapshot, 0, len(defaults))
	for _, d := range defaults {
		merged = append(merged, models.StageConfigSnapshot{
			StageCode:  d.StageCode,
			StageName:  d.StageName,
			StageOrder: d.StageOrder,
			AIRole:     d.AIRole,
			GateMode:   d.GateMode,
			Skippable:  d.Skippable,
		})
	}
	return merged
}

func detectStagesConfigFormat(configJSON string) string {
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil || len(raw) == 0 {
		return "unknown"
	}

	first := raw[0]

	if _, hasEnabled := first["enabled"]; hasEnabled {
		return "new"
	}

	if _, hasAction := first["action"]; hasAction {
		return "legacy"
	}

	return "unknown"
}
