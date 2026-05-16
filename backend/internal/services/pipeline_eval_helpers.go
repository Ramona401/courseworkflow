package services

// pipeline_eval_helpers.go — 一审上下文构建+评估辅助函数
//
// 从 pipeline_eval_meta.go 拆分

import (
	"encoding/json"
	"fmt"
	"strings"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 一审上下文构建（v68新增）====================

// buildFirstRoundContextForEval 为二审Evaluator构建一审上下文摘要
// 读取一审meta步骤的step_data，提取修改方案的raw_output摘要
// 让二审evaluator理解"一审为什么做了这些修改"，避免给出与一审方向相悖的评分
//
// 返回格式化的上下文字符串，如果读取失败返回空字符串（降级为无上下文，不阻断流程）
func buildFirstRoundContextForEval(pipelineID string) string {
	// 读取一审meta步骤的step_data
	metaStep, err := repository.GetStepByNameAndRound(pipelineID, models.StepMeta, 1)
	if err != nil || metaStep.Status != models.StepStatusDone || metaStep.StepData == "" {
		return ""
	}

	// 提取meta的raw_output（包含完整修改方案+修改后索引）
	metaRawOutput := extractMetaRawOutput(metaStep.StepData)
	if metaRawOutput == "" {
		return ""
	}

	// 提取meta的评分信息（总分+各维度）
	metaScoreInfo := extractMetaScoreSummary(metaStep.StepData)

	// 读取一审translator步骤的最终结果摘要
	transStep, _ := repository.GetStepByNameAndRound(pipelineID, models.StepTranslator, 1)
	transSummary := ""
	if transStep != nil && transStep.Status == models.StepStatusDone && transStep.StepData != "" {
		transSummary = extractTranslatorFinalSummary(transStep.StepData)
	}

	// 组装上下文
	var parts []string
	parts = append(parts,
		"【一审修改背景（重要：请仔细阅读）】",
		"以下是一审对本课件的评估和修改方案。当前你正在进行二审评估，",
		"请基于一审已做的修改进行增量评估，而非推翻一审的修改方向。",
		"如果一审的某项修改是合理的（如增加交互、增加能力评估、调整页面数），",
		"二审不应建议撤销这些修改，除非有明确的质量问题。",
		"",
	)

	// 一审评分摘要
	if metaScoreInfo != "" {
		parts = append(parts, "【一审Meta评分】", metaScoreInfo, "")
	}

	// 一审修改方案摘要（截取关键部分，避免过长）
	metaSummary := truncateForContext(metaRawOutput, 6000)
	parts = append(parts, "【一审修改方案摘要】", metaSummary, "")

	// 一审translator翻译方案摘要
	if transSummary != "" {
		parts = append(parts, "【一审逐页修改指令摘要】", transSummary, "")
	}

	return strings.Join(parts, "\n")
}

// buildFirstRoundContextForMeta 为二审Meta构建一审上下文摘要
// 读取一审的meta修改方案 + translator最终翻译方案 + reviewer审核意见
// 让二审meta在仲裁时能延续一审方向
//
// 返回格式化的上下文字符串，如果读取失败返回空字符串
func buildFirstRoundContextForMeta(pipelineID string) string {
	// 读取一审meta步骤
	metaStep, err := repository.GetStepByNameAndRound(pipelineID, models.StepMeta, 1)
	if err != nil || metaStep.Status != models.StepStatusDone || metaStep.StepData == "" {
		return ""
	}
	metaRawOutput := extractMetaRawOutput(metaStep.StepData)

	// 读取一审translator步骤
	transStep, _ := repository.GetStepByNameAndRound(pipelineID, models.StepTranslator, 1)
	transFinalOutput := ""
	transReviewOutput := ""
	if transStep != nil && transStep.Status == models.StepStatusDone && transStep.StepData != "" {
		transFinalOutput = extractTranslatorFinalOutput(transStep.StepData)
		transReviewOutput = extractTranslatorFinalReview(transStep.StepData)
	}

	// 如果所有数据都为空，降级为无上下文
	if metaRawOutput == "" && transFinalOutput == "" {
		return ""
	}

	// 组装上下文
	var parts []string
	parts = append(parts,
		"【一审修改背景（重要：二审必读）】",
		"以下是一审的完整修改方案和审核结果。当前你正在进行二审元评估，",
		"请在一审修改的基础上进行增量优化，延续一审已确认的修改方向。",
		"重点关注：一审已增加的交互、能力评估、页面调整等修改应当保留，",
		"除非存在明确的质量缺陷。二审应聚焦于一审遗漏的问题或可进一步提升的点。",
		"",
	)

	// 一审meta修改方案
	if metaRawOutput != "" {
		metaSummary := truncateForContext(metaRawOutput, 5000)
		parts = append(parts, "【一审Meta修改方案】", metaSummary, "")
	}

	// 一审translator最终逐页修改指令
	if transFinalOutput != "" {
		transSummary := truncateForContext(transFinalOutput, 5000)
		parts = append(parts, "【一审Translator逐页修改指令】", transSummary, "")
	}

	// 一审reviewer最终审核意见
	if transReviewOutput != "" {
		reviewSummary := truncateForContext(transReviewOutput, 3000)
		parts = append(parts, "【一审Reviewer审核意见】", reviewSummary, "")
	}

	return strings.Join(parts, "\n")
}

// extractMetaScoreSummary 从meta步骤的step_data中提取评分摘要
// 返回格式如 "总分: 7.5 | E1:7.8 E2:7.2 E3:7.5 E4:7.3 | GRADE:B"
func extractMetaScoreSummary(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}

	// 提取各评分字段
	getFloat := func(key string) float64 {
		raw, ok := data[key]
		if !ok {
			return 0
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			return 0
		}
		return v
	}
	getString := func(key string) string {
		raw, ok := data[key]
		if !ok {
			return ""
		}
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return ""
		}
		return v
	}

	totalFinal := getFloat("total_final")
	e1 := getFloat("e1_final")
	e2 := getFloat("e2_final")
	e3 := getFloat("e3_final")
	e4 := getFloat("e4_final")
	grade := getString("grade")

	if totalFinal <= 0 {
		return ""
	}

	result := fmt.Sprintf("总分: %.1f | E1:%.1f E2:%.1f E3:%.1f E4:%.1f", totalFinal, e1, e2, e3, e4)
	if grade != "" {
		result += " | GRADE:" + grade
	}
	return result
}

// extractTranslatorFinalSummary 从translator步骤的step_data中提取最终翻译方案的摘要
// 返回截取后的最终translator输出
func extractTranslatorFinalSummary(stepData string) string {
	output := extractTranslatorFinalOutput(stepData)
	if output == "" {
		return ""
	}
	return truncateForContext(output, 4000)
}

// extractTranslatorFinalOutput 从translator步骤的step_data中提取final_trans_output字段
func extractTranslatorFinalOutput(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}
	raw, ok := data["final_trans_output"]
	if !ok || string(raw) == "null" || string(raw) == `""` {
		return ""
	}
	var output string
	if err := json.Unmarshal(raw, &output); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return output
}

// extractTranslatorFinalReview 从translator步骤的step_data中提取final_review_output字段
func extractTranslatorFinalReview(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}
	raw, ok := data["final_review_output"]
	if !ok || string(raw) == "null" || string(raw) == `""` {
		return ""
	}
	var output string
	if err := json.Unmarshal(raw, &output); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return output
}

// truncateForContext 为上下文注入截取字符串
// 使用rune计算长度，确保中文字符不被截断
func truncateForContext(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "\n...（已截取，完整内容请参考一审步骤数据）"
}

