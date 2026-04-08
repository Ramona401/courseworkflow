package services

// workshop_stage_summary.go — 阶段摘要生成（分层记忆Episodic层）
//
// v84新增，拆分自 workshop_stage_extract.go
//
// 包含：
//   - GenerateStageSummary: 从阶段对话中生成结构化摘要（规则提取，不调AI）
//   - generateAnalyzeStageSummary: 教学分析阶段摘要
//   - generateDesignStageSummary: 教学设计阶段摘要
//   - generateWriteStageSummary: 教案撰写/修订阶段摘要
//   - generateReviewStageSummary: AI评审阶段摘要
//   - generateGenericStageSummary: 通用阶段摘要
//   - 辅助函数: getLastAIReply / countDialogueRounds / stageCodeToNameForSummary

import (
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// ==================== 阶段摘要生成主入口 ====================

// GenerateStageSummary 从阶段对话消息中生成结构化摘要
//
// v84新增：分层记忆架构的Episodic Memory层摘要生成函数
//
// 设计思路：
//   不调用AI（避免额外token消耗和延迟），而是从对话内容中规则提取关键信息
//   生成≤500字的结构化摘要，供后续阶段的AI上下文使用
//
// 提取策略（按阶段类型）：
//   analyze: 提取教师描述的学情、需求、重点难点等关键词
//   design:  提取讨论的教学策略、活动设计、目标等
//   write:   返回教案生成状态（"已生成X字教案"）
//   review:  返回评审摘要（"评审总分X分，N条改进建议"）
//   其他:    提取最后几条AI回复的核心内容
//
// 参数：
//   - stageCode: 阶段代码
//   - messages: 当前阶段的对话消息列表
//   - structuredOutput: 该阶段的structured_output（可能已包含有用信息）
//
// 返回值：
//   - summary: ≤500字的摘要文本
func GenerateStageSummary(stageCode string, messages []*models.ConversationMessage, structuredOutput string) string {
	// 如果对话消息为空，尝试从structured_output提取
	if len(messages) == 0 {
		return extractSummaryFromStructuredOutput(stageCode, structuredOutput)
	}

	switch stageCode {
	case "analyze":
		return generateAnalyzeStageSummary(messages)
	case "design":
		return generateDesignStageSummary(messages)
	case "write", "revise":
		return generateWriteStageSummary(stageCode, messages, structuredOutput)
	case "review":
		return generateReviewStageSummary(messages, structuredOutput)
	default:
		return generateGenericStageSummary(messages)
	}
}

// ==================== 各阶段摘要生成实现 ====================

// generateAnalyzeStageSummary 生成教学分析阶段的摘要
//
// 提取策略：收集教师消息中的关键信息（学情描述、需求表达等）
// 加上AI最后一轮分析的核心结论
func generateAnalyzeStageSummary(messages []*models.ConversationMessage) string {
	var sb strings.Builder

	// 收集教师的输入要点（可能包含学情、需求等关键信息）
	var teacherInputs []string
	for _, msg := range messages {
		if msg.Role == models.ConvRoleUser {
			content := strings.TrimSpace(msg.Content)
			if content != "" && len(content) > 10 {
				// 截取每条教师消息的前100字作为要点
				if len([]rune(content)) > 100 {
					content = string([]rune(content)[:100]) + "..."
				}
				teacherInputs = append(teacherInputs, content)
			}
		}
	}

	// 提取AI最后一条回复的核心内容（通常是分析总结）
	lastAIReply := getLastAIReply(messages)

	sb.WriteString("教师提供的信息：")
	if len(teacherInputs) > 0 {
		// 最多取3条教师输入
		limit := 3
		if len(teacherInputs) < limit {
			limit = len(teacherInputs)
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(teacherInputs[i])
			if i < limit-1 {
				sb.WriteString("；")
			}
		}
	} else {
		sb.WriteString("（未提供详细学情信息）")
	}

	if lastAIReply != "" {
		sb.WriteString("。AI分析结论：")
		// 截取AI回复的前200字
		if len([]rune(lastAIReply)) > 200 {
			lastAIReply = string([]rune(lastAIReply)[:200]) + "..."
		}
		sb.WriteString(lastAIReply)
	}

	result := sb.String()
	// 总长度限制500字
	if len([]rune(result)) > 500 {
		result = string([]rune(result)[:500]) + "..."
	}
	return result
}

// generateDesignStageSummary 生成教学设计阶段的摘要
//
// 提取策略：提取AI最后的设计方案总结 + 教师确认的关键决策
func generateDesignStageSummary(messages []*models.ConversationMessage) string {
	var sb strings.Builder

	// 收集教师的关键决策（选择的方案、确认的设计等）
	var teacherDecisions []string
	for _, msg := range messages {
		if msg.Role == models.ConvRoleUser {
			content := strings.TrimSpace(msg.Content)
			// 过滤掉简单确认语（"好的""可以""继续"等）
			if len(content) > 15 {
				if len([]rune(content)) > 80 {
					content = string([]rune(content)[:80]) + "..."
				}
				teacherDecisions = append(teacherDecisions, content)
			}
		}
	}

	// 提取AI最后一条回复（通常包含确定的设计方案）
	lastAIReply := getLastAIReply(messages)

	if lastAIReply != "" {
		sb.WriteString("设计方案：")
		if len([]rune(lastAIReply)) > 300 {
			lastAIReply = string([]rune(lastAIReply)[:300]) + "..."
		}
		sb.WriteString(lastAIReply)
	}

	if len(teacherDecisions) > 0 {
		sb.WriteString("。教师决策要点：")
		limit := 2
		if len(teacherDecisions) < limit {
			limit = len(teacherDecisions)
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(teacherDecisions[i])
			if i < limit-1 {
				sb.WriteString("；")
			}
		}
	}

	result := sb.String()
	if len([]rune(result)) > 500 {
		result = string([]rune(result)[:500]) + "..."
	}
	return result
}

// generateWriteStageSummary 生成教案撰写/修订阶段的摘要
//
// 策略：写作阶段的核心产出是教案正文，摘要主要记录状态
func generateWriteStageSummary(stageCode string, messages []*models.ConversationMessage, structuredOutput string) string {
	stageName := "撰写"
	if stageCode == "revise" {
		stageName = "修订"
	}

	// 尝试从structured_output获取教案长度
	if structuredOutput != "" && structuredOutput != "{}" {
		var data map[string]interface{}
		if json.Unmarshal([]byte(structuredOutput), &data) == nil {
			if content, ok := data["content_markdown"]; ok {
				if c, ok := content.(string); ok && len(c) > 0 {
					return fmt.Sprintf("教案%s完成，共%d字符。对话轮次：%d轮", stageName, len(c), countDialogueRounds(messages))
				}
			}
		}
	}

	rounds := countDialogueRounds(messages)
	return fmt.Sprintf("教案%s阶段进行了%d轮对话", stageName, rounds)
}

// generateReviewStageSummary 生成AI评审阶段的摘要
func generateReviewStageSummary(messages []*models.ConversationMessage, structuredOutput string) string {
	// 优先从structured_output提取评审关键数据
	if structuredOutput != "" && structuredOutput != "{}" {
		var data map[string]interface{}
		if json.Unmarshal([]byte(structuredOutput), &data) == nil {
			var parts []string

			if score, ok := data["total_score"]; ok {
				parts = append(parts, fmt.Sprintf("评审总分：%.1f/10", score))
			}
			if improvements, ok := data["improvements"]; ok {
				if impList, ok := improvements.([]interface{}); ok && len(impList) > 0 {
					parts = append(parts, fmt.Sprintf("提出%d条改进建议", len(impList)))
				}
			}
			if summary, ok := data["summary"]; ok {
				if s, ok := summary.(string); ok && len(s) > 0 {
					if len([]rune(s)) > 200 {
						s = string([]rune(s)[:200]) + "..."
					}
					parts = append(parts, "总评："+s)
				}
			}

			if len(parts) > 0 {
				return strings.Join(parts, "。")
			}
		}
	}

	// 兜底：从AI回复中提取
	lastAIReply := getLastAIReply(messages)
	if lastAIReply != "" {
		if len([]rune(lastAIReply)) > 500 {
			lastAIReply = string([]rune(lastAIReply)[:500]) + "..."
		}
		return "评审报告：" + lastAIReply
	}

	return "评审阶段已完成"
}

// generateGenericStageSummary 生成通用阶段的摘要（自定义阶段等）
func generateGenericStageSummary(messages []*models.ConversationMessage) string {
	lastAIReply := getLastAIReply(messages)
	if lastAIReply == "" {
		return fmt.Sprintf("阶段对话%d轮", countDialogueRounds(messages))
	}

	if len([]rune(lastAIReply)) > 400 {
		lastAIReply = string([]rune(lastAIReply)[:400]) + "..."
	}
	return lastAIReply
}

// ==================== 兜底提取 ====================

// extractSummaryFromStructuredOutput 从structured_output中提取摘要（对话为空时的兜底）
func extractSummaryFromStructuredOutput(stageCode string, structuredOutput string) string {
	if structuredOutput == "" || structuredOutput == "{}" {
		return ""
	}

	var data map[string]interface{}
	if json.Unmarshal([]byte(structuredOutput), &data) != nil {
		return ""
	}

	// 通用提取逻辑：按优先级尝试各字段
	if summary, ok := data["summary"]; ok {
		if s, ok := summary.(string); ok && s != "" {
			return s
		}
	}
	if content, ok := data["content_markdown"]; ok {
		if c, ok := content.(string); ok && len(c) > 0 {
			return fmt.Sprintf("已生成%s内容（%d字符）", stageCodeToNameForSummary(stageCode), len(c))
		}
	}
	if score, ok := data["total_score"]; ok {
		return fmt.Sprintf("评审总分：%.1f/10", score)
	}

	return ""
}

// ==================== 辅助函数 ====================

// getLastAIReply 获取消息列表中最后一条AI回复的内容
func getLastAIReply(messages []*models.ConversationMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.ConvRoleAssistant {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

// countDialogueRounds 统计对话轮次（一轮 = 一条用户消息）
func countDialogueRounds(messages []*models.ConversationMessage) int {
	rounds := 0
	for _, msg := range messages {
		if msg.Role == models.ConvRoleUser {
			rounds++
		}
	}
	return rounds
}

// stageCodeToNameForSummary 阶段代码转中文名（summary文件内部使用）
func stageCodeToNameForSummary(code string) string {
	nameMap := map[string]string{
		"analyze": "教学分析",
		"design":  "教学设计",
		"write":   "教案撰写",
		"review":  "AI评审",
		"revise":  "修订定稿",
	}
	if name, ok := nameMap[code]; ok {
		return name
	}
	return code
}
