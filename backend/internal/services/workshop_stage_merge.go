package services

// workshop_stage_merge.go — 阶段合并算法与教案阶段初始化
//
// v84拆分自 workshop_stage_service.go
//
// 包含：
//   - MergeStages: 系统默认阶段 + 配方覆盖 → 最终阶段列表
//   - mergeStagesNewFormat: 新格式（flow_items）合并
//   - mergeStagesLegacyFormat: 旧格式（overrides）合并
//   - InitStagesForPlan: 为教案初始化阶段配置
//   - 辅助函数: buildDefaultSnapshots / detectStagesConfigFormat / findStageIndex / removeStage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 阶段合并算法 ====================

// MergeStages 合并系统默认阶段与配方阶段覆盖，生成最终阶段列表
//
// 逻辑：
//   1. 加载系统默认阶段（source='system', status='active'）
//   2. 如果配方有 stages_config，检测格式（new/legacy）
//   3. 按格式执行合并：启用/禁用/插入/覆盖/替换提示词
//   4. 返回合并后的阶段快照列表
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

// mergeStagesNewFormat 新格式合并（flow_items数组，含enabled/order/is_custom等字段）
//
// 新格式由配方编辑器的流程搭建器生成，每个item包含：
//   - stage_code: 阶段代码
//   - enabled: 是否启用
//   - order: 排序序号
//   - is_custom: 是否自定义阶段
//   - prompt_mode: 阶段级对话模式覆盖
func (s *WorkshopStageService) mergeStagesNewFormat(defaultSnapshots []models.StageConfigSnapshot, defaults []*models.WorkshopStage, configJSON string, recipeID string, ctx context.Context) ([]models.StageConfigSnapshot, error) {
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
				StageCode: cs.StageCode, StageName: cs.StageName, StageOrder: item.Order,
				AIRole: cs.AIRole, GateMode: cs.GateMode, Skippable: cs.Skippable, IsCustom: true,
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
				StageCode: d.StageCode, StageName: d.StageName, StageOrder: item.Order,
				AIRole: d.AIRole, GateMode: d.GateMode, Skippable: d.Skippable, IsCustom: false,
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
	// 按order排序（冒泡排序，阶段数量少）
	for i := 0; i < len(merged); i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[j].StageOrder < merged[i].StageOrder {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}
	// 重新编号为连续序号
	for i := range merged {
		merged[i].StageOrder = i + 1
	}
	return merged, nil
}

// mergeStagesLegacyFormat 旧格式合并（overrides数组，含action/stage_code等字段）
//
// 旧格式支持的action类型：
//   - skip: 移除指定阶段
//   - override/replace_prompt: 覆盖阶段属性（名称/角色/门控/可跳过）
//   - insert_after: 在指定阶段之后插入新阶段
func (s *WorkshopStageService) mergeStagesLegacyFormat(merged []models.StageConfigSnapshot, configJSON string) ([]models.StageConfigSnapshot, error) {
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
					StageCode: ov.StageCode, StageName: ov.StageName, AIRole: ov.AIRole, GateMode: ov.GateMode, Skippable: true,
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
	// 重新编号
	for i := range merged {
		merged[i].StageOrder = i + 1
	}
	return merged, nil
}

// ==================== 为教案初始化阶段配置 ====================

// InitStagesForPlan 为教案初始化阶段配置
//
// 逻辑：
//   1. 调用MergeStages合并阶段
//   2. 将阶段配置快照写入lesson_plans.stage_config
//   3. 设置current_stage为第一个阶段
//   4. 创建第一个阶段的产出物记录（status=in_progress）
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
		LessonPlanID: lessonPlanID, StageCode: firstStage.StageCode, StageOrder: firstStage.StageOrder,
		StructuredOutput: "{}", NarrativeOutput: "", ConversationSnapshot: "[]", Status: models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		return nil, fmt.Errorf("创建初始阶段产出记录失败: %w", err)
	}
	wsLog.Info("教案阶段初始化完成", "plan_id", lessonPlanID, "stages_count", len(snapshots), "first_stage", firstStage.StageCode)
	return snapshots, nil
}

// ==================== 内部辅助函数 ====================

// findStageIndex 在阶段快照列表中查找指定阶段代码的索引位置
// 返回 -1 表示未找到
func findStageIndex(snapshots []models.StageConfigSnapshot, stageCode string) int {
	for i, s := range snapshots {
		if s.StageCode == stageCode {
			return i
		}
	}
	return -1
}

// removeStage 从阶段快照列表中移除指定阶段代码
func removeStage(snapshots []models.StageConfigSnapshot, stageCode string) []models.StageConfigSnapshot {
	result := make([]models.StageConfigSnapshot, 0, len(snapshots))
	for _, s := range snapshots {
		if s.StageCode != stageCode {
			result = append(result, s)
		}
	}
	return result
}

// buildDefaultSnapshots 从系统默认阶段定义构建快照列表
func buildDefaultSnapshots(defaults []*models.WorkshopStage) []models.StageConfigSnapshot {
	merged := make([]models.StageConfigSnapshot, 0, len(defaults))
	for _, d := range defaults {
		merged = append(merged, models.StageConfigSnapshot{
			StageCode: d.StageCode, StageName: d.StageName, StageOrder: d.StageOrder,
			AIRole: d.AIRole, GateMode: d.GateMode, Skippable: d.Skippable,
		})
	}
	return merged
}

// detectStagesConfigFormat 检测配方阶段配置的格式（new/legacy/unknown）
//
// new格式：数组元素包含 "enabled" 字段
// legacy格式：数组元素包含 "action" 字段
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
