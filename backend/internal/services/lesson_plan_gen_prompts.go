package services

// lesson_plan_gen_prompts.go — 教案生成提示词构建 + 解析辅助函数
//
// 职责：
//   - 各阶段提示词构建函数（buildDefaultSystemPrompt/buildChatPrompt/...）
//   - AI回复解析函数（parseAIReviewResult/extractContentFromReply/...）
//   - 组件格式转换工具（convertGroupsToConvComponents/...）
//   - 消息格式化工具（formatSelectedOptions/generateMsgID/...）
//
// 所有函数均为纯函数（无状态），供 lesson_plan_gen_service.go 调用

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
)


// ==================== 系统提示词 ====================



// buildDefaultReviewRules 默认评审规则（含学科专属维度）
func buildDefaultReviewRules(subject string) string {
	base := `通用评审维度（各10分）：
T1 目标清晰度：三维目标是否具体、可观察、可评估
T2 结构完整性：环节是否齐全、时间分配是否合理
T3 学生参与度：学生主动参与vs被动接收，讲授占比
T4 评估对齐度：评估方式能否检验目标达成
T5 可操作性：活动步骤清晰、材料可获得`

	if subject == "AI" || subject == "人工智能" {
		base += `

学科维度（各10分）：
S1 技术体验真实性：学生是否真正操作了AI工具
S2 概念准确性：AI相关概念是否准确、适龄
S3 批判性思维：是否引导学生思考AI的局限
S4 跨学科连接：是否与已有学科知识关联
S5 工具可用性：所用AI工具是否免费、无需翻墙`
	}
	return base
}

// buildReviewSystemPrompt 评审专用系统提示词（要求严格JSON输出格式）
func buildReviewSystemPrompt(subject string) string {
	return fmt.Sprintf(`你是一位经验丰富的%s课教案评审专家。
请对教案进行专业评审，输出格式严格按照以下JSON结构：

{
  "total_score": 8.5,
  "summary": "整体来说这份教案...(对话口吻，100-150字)",
  "good_points": ["做得好的1", "做得好的2"],
  "improvements": [
    {
      "id": "imp_1",
      "issue": "问题描述",
      "suggestion": "具体改进方案（对话口吻，如：试试把讲解时间从10分钟压缩到5分钟？）",
      "section": "涉及环节（可选）"
    }
  ],
  "dimensions": [
    {"code": "T1", "name": "目标清晰度", "score": 9, "comment": "...", "good": true}
  ]
}

评分原则：
- 总分为各维度平均分（0-10分制）
- 6分以下：明显问题  7-8分：可以改进  9-10分：优秀
- "做得好的"和"可以更好"各至少2-3条
- 所有描述使用对话口吻，如"这里可以试试..."而非"应该..."`, subject)
}

// ==================== 对话提示词 ====================


// buildReviewPrompt 组装评审用户提示词
func buildReviewPrompt(lp *models.LessonPlan, reviewRules string) string {
	return fmt.Sprintf(
		"请评审以下%s课教案：\n\n**基本信息**\n年级：%s\n课题：%s\n课时：%d分钟\n\n**教案内容**\n%s\n\n**评审维度参考**\n%s",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes,
		lp.ContentMarkdown, reviewRules,
	)
}

// buildOptimizePrompt 组装教案优化提示词
func buildOptimizePrompt(content string, suggestions []string) string {
	var sb strings.Builder
	sb.WriteString("请根据以下评审建议优化教案，保持Markdown格式，重点改进被指出的问题：\n\n")
	sb.WriteString("**改进建议：**\n")
	for i, s := range suggestions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	sb.WriteString("\n**原教案：**\n")
	sb.WriteString(content)
	sb.WriteString("\n\n**输出优化后的完整教案（Markdown格式）：**")
	return sb.String()
}

// buildDefaultOpeningMessage 构建默认开场消息（AI调用失败时降级使用）
func buildDefaultOpeningMessage(req *models.StartConversationRequest) *models.ConversationMessage {
	content := fmt.Sprintf(
		"你好！我是你的AI备课助手 ✨\n\n我看到你要备一节**%s年级 %s课**，课题是「%s」，%d分钟课时。\n\n让我先了解一下你的学生情况，这样我能给你更精准的建议：\n\n1. 学生之前有没有接触过相关内容？\n2. 班级同学的整体接受能力怎么样？",
		req.Grade, req.Subject, req.Topic, req.DurationMinutes,
	)
	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// ==================== AI回复解析函数 ====================

// parseAIReviewResult 解析AI评审JSON结果
// 先用ExtractJSON提取JSON块，再反序列化
func parseAIReviewResult(content string) (*models.AIReviewResult, error) {
	jsonStr, ok := aiClient.ExtractJSON(content)
	if !ok {
		return nil, fmt.Errorf("AI回复中未找到JSON")
	}
	var result models.AIReviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析评审JSON失败: %w", err)
	}
	result.ReviewedAt = time.Now()
	return &result, nil
}

// buildFallbackReview 解析失败时的降级评审结果
func buildFallbackReview(rawContent string) *models.AIReviewResult {
	return &models.AIReviewResult{
		TotalScore: 7.0,
		Summary:    "AI评审已完成，请查看详细内容。",
		GoodPoints: []string{"教案结构基本完整"},
		Improvements: []models.AIReviewImprovement{{
			ID:         "imp_fallback",
			Issue:      "评审解析异常",
			Suggestion: rawContent,
		}},
		ReviewedAt: time.Now(),
	}
}

// appendReviewToHistory 将新评审结果追加到历史记录JSON
func appendReviewToHistory(oldHistory string, review *models.AIReviewResult) string {
	var history []models.AIReviewResult
	if err := json.Unmarshal([]byte(oldHistory), &history); err != nil {
		history = []models.AIReviewResult{}
	}
	history = append(history, *review)
	b, _ := json.Marshal(history)
	return string(b)
}

// extractSuggestionsByIDs 从评审结果JSON中提取指定ID的建议文本
// ids为空时返回全部建议
func extractSuggestionsByIDs(reviewResultJSON string, ids []string) []string {
	var result models.AIReviewResult
	if err := json.Unmarshal([]byte(reviewResultJSON), &result); err != nil {
		return nil
	}
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	var suggestions []string
	for _, imp := range result.Improvements {
		if len(ids) == 0 || idSet[imp.ID] {
			suggestions = append(suggestions, imp.Suggestion)
		}
	}
	return suggestions
}


// ==================== 组件格式转换 ====================

// convertGroupsToConvComponents 将组件组转为对话消息中的组件卡片格式
func convertGroupsToConvComponents(groups []*models.MatchedComponentGroup) []models.ConvComponent {
	var result []models.ConvComponent
	for _, g := range groups {
		for _, c := range g.Components {
			result = append(result, models.ConvComponent{
				ID:             c.ID,
				LibraryType:    g.LibraryType,
				DisplayLabel:   c.DisplayLabel,
				DesignLogic:    c.DesignLogic,
				ExampleSnippet: c.ExampleSnippet,
				QualityScore:   c.QualityScore,
				UsageCount:     c.UsageCount,
			})
		}
	}
	return result
}

// cleanComponentMarkers 清除AI回复中的组件触发标记
func cleanComponentMarkers(content string) string {
	content = strings.ReplaceAll(content, "【推荐组件】", "")
	content = strings.ReplaceAll(content, "推荐以下教学方案", "根据你的情况，我推荐以下教学方案")
	return strings.TrimSpace(content)
}

// ==================== 消息格式化工具 ====================

// formatSelectedOptions 将选项key列表转为可读文本
func formatSelectedOptions(keys []string, originalMsg string) string {
	if originalMsg != "" {
		return originalMsg
	}
	return "我选择：" + strings.Join(keys, "、")
}

// formatSelectedComponents 将已选组件ID数量转为提示文本
func formatSelectedComponents(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("\n（已选择%d个教学组件）", len(ids))
}

// generateMsgID 生成基于时间戳的唯一消息ID
func generateMsgID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// ==================== Phase 7A：带配方上下文的对话提示词 ====================

