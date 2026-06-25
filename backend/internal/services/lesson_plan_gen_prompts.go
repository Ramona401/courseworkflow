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
//
// v198 变更（评审侧·教学逻辑内核观测点接入·旁路评审）：
//   - buildDefaultReviewRules 的 T2/T5 维度补入三个可观测检查点——学科逻辑严谨性、
//     例子连贯贯穿、真实生活化情景——使手动触发的旁路 AI 评审(buildReviewSystemPrompt
//     + buildReviewPrompt 走的这条)也按写教案侧同一套底层标准给分。
//   - 注意：对话模式 review 阶段走的是数据库 workshop_stages.system_prompt(另行处理),
//     本文件只覆盖手动触发评审这条旁路;两处口径将保持一致。

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
//
// v198：T2/T5 补入「教学逻辑内核」三个可观测检查点(学科逻辑严谨性 / 例子连贯贯穿 /
//
//	真实生活化情景),并在通用维度后补一条总括说明,使旁路评审与写教案侧同源约束。
func buildDefaultReviewRules(subject string) string {
	base := `通用评审维度（各10分）：
T1 目标清晰度：三维目标是否具体、可观察、可评估
T2 结构与逻辑严谨性：环节是否齐全、时间分配是否合理；更重要的是——整节课是否围绕一条清晰的学科核心逻辑/原理展开，各环节是否层层为这条主线服务、彼此咬合，而非互不关联地堆叠；对学科原理的讲解（含具象化比喻）是否站得住脚，有没有为了通俗而编造违背学科真实机制、会让学生形成错误认知的伪过程
T3 学生参与度：学生主动参与vs被动接收，讲授占比
T4 评估对齐度：评估方式能否检验目标达成
T5 可操作性与情景真实性：活动步骤是否清晰、材料是否可获得；导入与核心情境是否取自学生真实生活经验或当下真实问题（2025新课标要求与学生实际生活相关联）；例子是否尽量单线贯穿、层层升华，而非每个环节频繁更换新例增加学生（尤其低年级）的认知负荷

评审取向（重要）：判断一个环节/活动/例子好不好，标准是"它是否真的在为本课核心逻辑服务、机制是否严谨、是否贴近学生真实生活"，而不是"它读起来像不像一个漂亮的教学设计"。对于堆砌互不关联的前沿术语、华丽但偏离主线的活动、或违背学科本质的伪逻辑，应在改进建议中明确指出。`

	if subject == "AI" || subject == "人工智能" {
		base += `

学科维度（各10分）：
S1 技术体验真实性：学生是否真正操作了AI工具
S2 概念准确性：AI相关概念是否准确、适龄；尤其是对AI工作机制的讲解有没有歪曲本质（例如把AI描述成"先识别颜色再识别形状再组合判断"这类人类预设的串行流水线，这违背了AI从大量样本中自己学习特征的本质）
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
