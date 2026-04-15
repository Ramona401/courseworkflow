package services

// recipe_flow_service.go — 备课配方流程校验与预设模板
//
// 从 recipe_service.go 拆分而来（v92重构）
// v97重构：ValidateStageFlow拆分为独立规则函数，降低认知复杂度

import (
	"fmt"

	"tedna/internal/models"
)

// ==================== 流程校验 ====================

// ValidateStageFlow 校验流程配置的完整性
// 返回校验结果，包含三级提示（info/warning/error）
func (s *RecipeService) ValidateStageFlow(stages []models.StageFlowItem) *models.FlowValidationResult {
	result := &models.FlowValidationResult{Valid: true, Messages: []models.FlowValidationMessage{}}

	// 筛选出已启用的阶段
	var enabled []models.StageFlowItem
	for _, st := range stages {
		if st.Enabled {
			enabled = append(enabled, st)
		}
	}

	// 规则1（阻断）：流程不能为空
	if len(enabled) == 0 {
		result.Valid = false
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgError, Code: "flow_empty",
			Message: "流程不能为空，至少需要保留「教案撰写」和「修订定稿」",
		})
		return result
	}

	// 构建辅助数据
	enabledSet := make(map[string]bool)
	orderMap := make(map[string]int)
	for _, st := range enabled {
		enabledSet[st.StageCode] = true
		orderMap[st.StageCode] = st.Order
	}

	// 依次执行各条阻断规则（规则2-5）
	checkRequiredStages(result, enabledSet)
	checkReviseIsLast(result, enabled, enabledSet, orderMap)
	checkReviewAfterWrite(result, enabledSet, orderMap)
	checkNoDuplicateStages(result, stages)

	// 执行警告规则（规则6-8）
	checkSkipAnalyze(result, enabledSet)
	checkSkipDesign(result, enabledSet)
	checkSkipReview(result, enabledSet)

	// 执行信息提示（规则9）
	checkMinimalFlow(result, enabled)

	return result
}

// ==================== 阻断规则（error级别） ====================

// checkRequiredStages 规则2：不可移除的阶段必须保留（write和revise）
func checkRequiredStages(result *models.FlowValidationResult, enabledSet map[string]bool) {
	for code, rule := range models.SystemStageFlowRules {
		if !rule.Removable && !enabledSet[code] {
			result.Valid = false
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "required_stage_missing",
				Message: fmt.Sprintf("「%s」是必须保留的阶段，不可移除", rule.StageName),
			})
		}
	}
}

// checkReviseIsLast 规则3：修订定稿必须在最后
func checkReviseIsLast(result *models.FlowValidationResult, enabled []models.StageFlowItem, enabledSet map[string]bool, orderMap map[string]int) {
	if !enabledSet["revise"] {
		return
	}
	reviseOrder := orderMap["revise"]
	for _, st := range enabled {
		if st.StageCode != "revise" && st.Order > reviseOrder {
			result.Valid = false
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "revise_not_last",
				Message: "「修订定稿」必须是最后一个阶段",
			})
			return
		}
	}
}

// checkReviewAfterWrite 规则4：review必须在write之后
func checkReviewAfterWrite(result *models.FlowValidationResult, enabledSet map[string]bool, orderMap map[string]int) {
	if enabledSet["review"] && enabledSet["write"] && orderMap["review"] < orderMap["write"] {
		result.Valid = false
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgError, Code: "review_before_write",
			Message: "「AI评审」必须在「教案撰写」之后",
		})
	}
}

// checkNoDuplicateStages 规则5：阶段不能重复
func checkNoDuplicateStages(result *models.FlowValidationResult, stages []models.StageFlowItem) {
	codeCount := make(map[string]int)
	for _, st := range stages {
		codeCount[st.StageCode]++
	}
	for code, count := range codeCount {
		if count > 1 {
			result.Valid = false
			rule := models.SystemStageFlowRules[code]
			name := code
			if rule.StageName != "" {
				name = rule.StageName
			}
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "stage_duplicate",
				Message: fmt.Sprintf("阶段「%s」重复出现", name),
			})
		}
	}
}

// ==================== 警告规则（warning级别） ====================

// checkSkipAnalyze 规则6：跳过分析阶段
func checkSkipAnalyze(result *models.FlowValidationResult, enabledSet map[string]bool) {
	if !enabledSet["analyze"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_analyze",
			Message: "跳过「教学分析」后，AI缺少学情和课标依据，教案质量可能下降",
		})
	}
}

// checkSkipDesign 规则7：跳过设计阶段
func checkSkipDesign(result *models.FlowValidationResult, enabledSet map[string]bool) {
	if !enabledSet["design"] && enabledSet["write"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_design",
			Message: "跳过「教学设计」后，AI将直接撰写教案，缺少系统化的教学设计环节",
		})
	}
}

// checkSkipReview 规则8：跳过评审阶段
func checkSkipReview(result *models.FlowValidationResult, enabledSet map[string]bool) {
	if !enabledSet["review"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_review",
			Message: "跳过「AI评审」后，教案将不经过自动质量检查",
		})
	}
}

// ==================== 信息提示（info级别） ====================

// checkMinimalFlow 规则9：极简模式提示
func checkMinimalFlow(result *models.FlowValidationResult, enabled []models.StageFlowItem) {
	if len(enabled) <= 2 {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgInfo, Code: "minimal_flow",
			Message: "极简流程适合经验丰富的老师快速出稿，新手建议使用更完整的流程",
		})
	}
}

// ==================== 预设流程模板 ====================

// GetFlowPresets 返回预设流程模板列表
func (s *RecipeService) GetFlowPresets() []*models.FlowPreset {
	return []*models.FlowPreset{
		{
			Key: "full_guided", Name: "完整引导", Description: "5步全开，逐步引导，适合新手或重要课程",
			Duration: "15-25分钟", Icon: "🎓", PromptMode: models.PromptModeGuided,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: true, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: true, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "standard", Name: "标准协作", Description: "跳过AI评审，分析+设计+撰写+修订",
			Duration: "10-15分钟", Icon: "🤝", PromptMode: models.PromptModeGuided,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: true, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "quick_draft", Name: "快速出稿", Description: "跳过设计和评审，分析+撰写+修订",
			Duration: "5-10分钟", Icon: "⚡", PromptMode: models.PromptModeEfficient,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: false, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "express", Name: "极速模式", Description: "仅撰写+修订，适合老手快速完成",
			Duration: "3-5分钟", Icon: "🚀", PromptMode: models.PromptModeEfficient,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: false, Order: 1},
				{StageCode: "design", Enabled: false, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
	}
}
