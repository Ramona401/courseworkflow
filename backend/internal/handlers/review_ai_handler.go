package handlers

// review_ai_handler.go — 审核员AI辅助接口
//
// API列表：
//   POST /api/v1/lesson-plans/review-ai/overview — 生成教案整体概览（供审核员快速了解）
//   POST /api/v1/lesson-plans/review-ai/chat     — 对话式AI评审（审核员与AI就教案内容对话）
//
// 设计思路：
//   - overview：AI一段话总结教案整体教学设计，帮审核员快速建立全局印象
//   - chat：审核员看到疑问可随时与AI聊，AI带着完整教案正文回答

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/utils"
)

// ReviewAIHandler 审核员AI辅助处理器
type ReviewAIHandler struct {
	cfg *config.Config
}

// NewReviewAIHandler 创建审核员AI辅助处理器
func NewReviewAIHandler(cfg *config.Config) *ReviewAIHandler {
	return &ReviewAIHandler{cfg: cfg}
}

// ==================== 教案整体概览 ====================

// ReviewAIOverviewRequest 生成教案概览请求体
type ReviewAIOverviewRequest struct {
	PlanMeta    string `json:"plan_meta"`    // 教案基本信息（学科/年级/课题/课时）
	PlanContent string `json:"plan_content"` // 教案正文内容
}

const reviewOverviewSystemPrompt = `你是一位经验丰富的教研员，正在帮助同事快速了解一份待评审的教案。

请用3-5句话简明扼要地描述这份教案的整体教学设计，重点说明：
1. 教学目标是否清晰、教学重难点是否合理
2. 教学流程的亮点（如有特色的导入/活动设计）
3. 需要重点关注的评审维度（基于教案内容预判）

语气专业但口语化，像在向同事介绍一样，不要用标题和列表，用自然段落表达。`

// Overview POST /lesson-plans/review-ai/overview
func (h *ReviewAIHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	// 鉴权
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req ReviewAIOverviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.PlanContent) == "" {
		utils.BadRequest(w, "教案内容不能为空")
		return
	}

	// 获取AI配置
	aiCfg, err := ai.GetEffectiveConfig(
		h.cfg.AESKey, "lesson_plan",
		h.cfg.AIAPIBaseURL, h.cfg.AIAPIKey, h.cfg.AIDefaultModel,
	)
	if err != nil {
		utils.InternalError(w, "获取AI配置失败")
		return
	}

	userPrompt := fmt.Sprintf(
		"【教案基本信息】\n%s\n\n【教案正文】\n%s",
		strings.TrimSpace(req.PlanMeta),
		strings.TrimSpace(req.PlanContent),
	)

	result, callErr := ai.CallAI(aiCfg, reviewOverviewSystemPrompt, userPrompt, nil)
	if callErr != nil {
		log.Printf("[review-ai overview] AI调用失败: %v", callErr)
		utils.InternalError(w, "AI概览生成失败，请稍后重试")
		return
	}

	utils.Success(w, map[string]string{"overview": result.Content})
}

// ==================== 对话式审核 ====================

// ReviewAIChatRequest 对话式审核请求体
type ReviewAIChatRequest struct {
	PlanMeta    string                   `json:"plan_meta"`    // 教案基本信息
	PlanContent string                   `json:"plan_content"` // 教案正文内容
	History     []map[string]string      `json:"history"`      // 对话历史 [{role,content},...]
	Message     string                   `json:"message"`      // 当前问题
}

const reviewChatSystemPrompt = `你是一位经验丰富的教研员，正在协助同事评审一份教案。

你已经仔细阅读了完整的教案内容，可以就教案的任何方面回答问题：
- 具体教学环节的设计是否合理
- 教学目标表述是否规范
- 时间分配和活动安排是否科学
- 与同年级/同学科优秀教案的对比分析
- 需要作者改进的具体建议

回答时：
- 基于教案实际内容给出有依据的分析，引用教案中的具体细节
- 语气专业但友好，像与同事探讨一样
- 如发现教案的问题，给出建设性的改进建议
- 回答简明扼要，通常3-6句话即可，复杂问题可以更长`

// Chat POST /lesson-plans/review-ai/chat
func (h *ReviewAIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	// 鉴权
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req ReviewAIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		utils.BadRequest(w, "问题内容不能为空")
		return
	}

	// 获取AI配置
	aiCfg, err := ai.GetEffectiveConfig(
		h.cfg.AESKey, "lesson_plan",
		h.cfg.AIAPIBaseURL, h.cfg.AIAPIKey, h.cfg.AIDefaultModel,
	)
	if err != nil {
		utils.InternalError(w, "获取AI配置失败")
		return
	}

	// 构建用户提示词：包含教案全文 + 对话历史 + 当前问题
	var promptBuilder strings.Builder
	promptBuilder.WriteString("【我正在评审的教案】\n")
	promptBuilder.WriteString(strings.TrimSpace(req.PlanMeta))
	promptBuilder.WriteString("\n\n")
	promptBuilder.WriteString(strings.TrimSpace(req.PlanContent))
	promptBuilder.WriteString("\n\n")

	// 加入对话历史上下文（最近5轮）
	if len(req.History) > 0 {
		promptBuilder.WriteString("【之前的对话】\n")
		start := 0
		if len(req.History) > 10 {
			start = len(req.History) - 10 // 最多保留5轮对话
		}
		for _, msg := range req.History[start:] {
			role := "评审员"
			if msg["role"] == "assistant" {
				role = "AI助手"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s：%s\n", role, msg["content"]))
		}
		promptBuilder.WriteString("\n")
	}

	promptBuilder.WriteString("【评审员当前问题】\n")
	promptBuilder.WriteString(strings.TrimSpace(req.Message))

	result, callErr := ai.CallAI(aiCfg, reviewChatSystemPrompt, promptBuilder.String(), nil)
	if callErr != nil {
		log.Printf("[review-ai chat] AI调用失败: %v", callErr)
		utils.InternalError(w, "AI回答失败，请稍后重试")
		return
	}

	utils.Success(w, map[string]string{"reply": result.Content})
}
