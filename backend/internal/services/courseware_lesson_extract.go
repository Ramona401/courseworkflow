package services

// courseware_lesson_extract.go — 课件相关场景共用的"教案正文提取"包级公共函数
//
// 背景：courseware_index_service.go 原有私有方法 extractLessonPlanContent（按
//   content_markdown → conversation_log 最长assistant消息 → ai_review_result →
//   ai_review_history 的优先级链提取可注入的教案正文）。
//   对齐报告服务（courseware_alignment_service.go）需要同一份正文提取逻辑，故抽成
//   包级公共函数 ExtractLessonPlanContentForCW，两处共用，避免逻辑漂移。
//
// 行为与原私有方法逐字对齐（纯位置搬迁 + 导出），index_service 改为转调本函数。

import (
	"encoding/json"
	"log"
	"strings"

	"tedna/internal/models"
)

// cwLessonExtractMsg 解析 conversation_log 用的轻量消息结构（仅取 role/content）
type cwLessonExtractMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ExtractLessonPlanContentForCW 从教案对象提取可供 AI 使用的教案正文。
//
// 优先级链（与原 CoursewareIndexService.extractLessonPlanContent 完全一致）：
//   1. content_markdown —— 教案正文（Fork/导入/手动编辑的内容）
//   2. conversation_log —— 对话记录中最长的 assistant 消息（AI 备课工坊生成的教案）
//   3. ai_review_result —— AI 评审结果
//   4. ai_review_history —— 评审历史（最后兜底）
//
// 返回可能为空字符串（教案确实无可用内容时），调用方需自行判断长度。
func ExtractLessonPlanContentForCW(lp *models.LessonPlan) string {
	var parts []string

	// 优先级1: content_markdown
	if lp.ContentMarkdown != "" && len(strings.TrimSpace(lp.ContentMarkdown)) > 50 {
		parts = append(parts, lp.ContentMarkdown)
	}

	// 优先级2: conversation_log 中最长的 assistant 消息
	if len(parts) == 0 && lp.ConversationLog != "" {
		messages := cwParseLessonConversationLog(lp.ConversationLog)
		var longestMsg string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" && len(messages[i].Content) > len(longestMsg) {
				longestMsg = messages[i].Content
			}
		}
		if len(longestMsg) > 200 {
			parts = append(parts, longestMsg)
		}
	}

	// 优先级3: ai_review_result
	if len(parts) == 0 && lp.AIReviewResult != "" {
		parts = append(parts, "【AI评审结果】\n"+lp.AIReviewResult)
	}

	// 优先级4: ai_review_history（兜底）
	if len(parts) == 0 && lp.AIReviewHistory != "" {
		parts = append(parts, "【教案历史】\n"+lp.AIReviewHistory)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// cwParseLessonConversationLog 解析教案 conversation_log JSON 为消息数组（容错）
func cwParseLessonConversationLog(logJSON string) []cwLessonExtractMsg {
	if logJSON == "" || logJSON == "null" || logJSON == "[]" {
		return nil
	}
	var messages []cwLessonExtractMsg
	if err := json.Unmarshal([]byte(logJSON), &messages); err != nil {
		log.Printf("[courseware_lesson_extract] 解析对话日志失败: %v", err)
		return nil
	}
	return messages
}
