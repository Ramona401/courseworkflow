package handlers

// review_ai_handler.go — 审核员AI辅助接口(流式版)
//
// API列表:
//   POST /api/v1/lesson-plans/review-ai/overview — 生成教案整体概览(SSE流式)
//   POST /api/v1/lesson-plans/review-ai/chat     — 对话式AI评审(SSE流式)
//
// v110(TE-DNA 3.0 P0)改造:
//   - 请求体新增可选字段 assistant_id:前端若传,则用该助手的 full_prompt 作为 system prompt
//   - 未传 assistant_id 时回退到原硬编码 prompt(完全向后兼容)
//   - 使用助手时通过 AIAssistantService.LoadActiveAssistantForUse 自动做可见性校验 + 使用量埋点
//
// v113(本次改造)核心:
//   - 两个接口从非流式 ai.CallAI 切换为流式 ai.CallAIStream
//   - SSE 事件协议与 annotation ai-fix 接口保持一致:
//       event: connected  → {"plan_info":"..."}
//       event: chunk      → {"chunk":"..."}      (多次)
//       event: done       → {"full_content":"...", "tokens_used":N}
//       event: error      → {"error":"..."}
//   - 前置错误(鉴权/请求体格式)仍走普通 JSON 响应(非 SSE),保持前端 fetch 可统一处理
//   - 进入 AI 调用后切换 SSE Header,逐 chunk 推送,最后以 done 事件收尾

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ReviewAIHandler 审核员AI辅助处理器
// v110 改动:持有 assistantService 以支持 assistant_id 参数
type ReviewAIHandler struct {
	cfg              *config.Config
	assistantService *services.AIAssistantService
}

// NewReviewAIHandler 创建审核员AI辅助处理器
// v110 改动:签名变化,新增 assistantService 依赖
func NewReviewAIHandler(cfg *config.Config, assistantService *services.AIAssistantService) *ReviewAIHandler {
	return &ReviewAIHandler{
		cfg:              cfg,
		assistantService: assistantService,
	}
}

// ==================== 硬编码默认 Prompt(未选择助手时使用) ====================

const reviewOverviewDefaultPrompt = `你是一位经验丰富的教研员,正在帮助同事快速了解一份待评审的教案。

请用3-5句话简明扼要地描述这份教案的整体教学设计,重点说明:
1. 教学目标是否清晰、教学重难点是否合理
2. 教学流程的亮点(如有特色的导入/活动设计)
3. 需要重点关注的评审维度(基于教案内容预判)

语气专业但口语化,像在向同事介绍一样,不要用标题和列表,用自然段落表达。`

const reviewChatDefaultPrompt = `你是一位经验丰富的教研员,正在协助同事评审一份教案。

你已经仔细阅读了完整的教案内容,可以就教案的任何方面回答问题:
- 具体教学环节的设计是否合理
- 教学目标表述是否规范
- 时间分配和活动安排是否科学
- 与同年级/同学科优秀教案的对比分析
- 需要作者改进的具体建议

回答时:
- 基于教案实际内容给出有依据的分析,引用教案中的具体细节
- 语气专业但友好,像与同事探讨一样
- 如发现教案的问题,给出建设性的改进建议
- 回答简明扼要,通常3-6句话即可,复杂问题可以更长`

// ==================== SSE 工具函数 ====================

// writeReviewAISSEEvent 向客户端写入一条 SSE 事件
// 与 annotation_handler.writeAnnotationSSEEvent 结构一致,便于前端统一消费
func writeReviewAISSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

// prepareSSEResponse 将响应切换为 SSE 流式模式,返回 flusher(不支持时返回 nil)
func prepareSSEResponse(w http.ResponseWriter) http.Flusher {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // 关键:禁用 Nginx 对 SSE 的缓冲
	return flusher
}

// ==================== 教案整体概览(流式) ====================

// ReviewAIOverviewRequest 生成教案概览请求体
// v110 新增字段:AssistantID(可选)
type ReviewAIOverviewRequest struct {
	PlanMeta    string `json:"plan_meta"`    // 教案基本信息(学科/年级/课题/课时)
	PlanContent string `json:"plan_content"` // 教案正文内容
	AssistantID string `json:"assistant_id"` // v110 新增:可选的 AI 助手 ID
}

// Overview POST /lesson-plans/review-ai/overview (SSE)
// v113 改造:改为 SSE 流式响应,逐 chunk 推送 AI 生成的概览
func (h *ReviewAIHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	// ========== 前置错误阶段:走普通 JSON 响应 ==========

	// 鉴权
	claims, ok := middleware.GetClaims(r.Context())
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

	// v110 新增:根据 assistant_id 决定 system prompt
	systemPrompt, err := h.resolveSystemPrompt(r, claims.UserID, claims.Role, req.AssistantID, reviewOverviewDefaultPrompt)
	if err != nil {
		utils.BadRequest(w, err.Error())
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

	// ========== 流式响应阶段:切换为 SSE ==========

	flusher := prepareSSEResponse(w)
	if flusher == nil {
		utils.InternalError(w, "不支持流式响应")
		return
	}

	// 发送 connected 事件(前端借此知道流已建立)
	writeReviewAISSEEvent(w, flusher, "connected", map[string]string{
		"phase": "overview",
	})

	// 流式调用 AI,逐 token 推送 chunk 事件
	startTime := time.Now()
	callResult, callErr := ai.CallAIStream(
		aiCfg,
		systemPrompt,
		userPrompt,
		func(chunk string) error {
			writeReviewAISSEEvent(w, flusher, "chunk", map[string]string{"chunk": chunk})
			return nil
		},
		nil,
	)

	if callErr != nil {
		log.Printf("[review-ai overview stream] AI调用失败 user=%s assistant=%s err=%v",
			claims.UserID, req.AssistantID, callErr)
		writeReviewAISSEEvent(w, flusher, "error", map[string]string{
			"error": "AI概览生成失败,请稍后重试",
		})
		return
	}

	latencyMs := time.Since(startTime).Milliseconds()
	log.Printf("[review-ai overview stream] 完成 user=%s assistant=%s tokens=%d latency=%dms",
		claims.UserID, req.AssistantID, callResult.TokensUsed, latencyMs)

	// 发送 done 事件,携带完整内容(供前端兜底/记录)
	writeReviewAISSEEvent(w, flusher, "done", map[string]interface{}{
		"full_content": callResult.Content,
		"tokens_used":  callResult.TokensUsed,
	})
}

// ==================== 对话式审核(流式) ====================

// ReviewAIChatRequest 对话式审核请求体
// v110 新增字段:AssistantID(可选)
type ReviewAIChatRequest struct {
	PlanMeta    string              `json:"plan_meta"`    // 教案基本信息
	PlanContent string              `json:"plan_content"` // 教案正文内容
	History     []map[string]string `json:"history"`      // 对话历史 [{role,content},...]
	Message     string              `json:"message"`      // 当前问题
	AssistantID string              `json:"assistant_id"` // v110 新增:可选的 AI 助手 ID
}

// Chat POST /lesson-plans/review-ai/chat (SSE)
// v113 改造:改为 SSE 流式响应,逐 chunk 推送 AI 回答
func (h *ReviewAIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	// ========== 前置错误阶段:走普通 JSON 响应 ==========

	// 鉴权
	claims, ok := middleware.GetClaims(r.Context())
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

	// v110 新增:根据 assistant_id 决定 system prompt
	systemPrompt, err := h.resolveSystemPrompt(r, claims.UserID, claims.Role, req.AssistantID, reviewChatDefaultPrompt)
	if err != nil {
		utils.BadRequest(w, err.Error())
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

	// 构建用户提示词:包含教案全文 + 对话历史 + 当前问题
	var promptBuilder strings.Builder
	promptBuilder.WriteString("【我正在评审的教案】\n")
	promptBuilder.WriteString(strings.TrimSpace(req.PlanMeta))
	promptBuilder.WriteString("\n\n")
	promptBuilder.WriteString(strings.TrimSpace(req.PlanContent))
	promptBuilder.WriteString("\n\n")

	// 加入对话历史上下文(最近 10 条 = 5 轮)
	if len(req.History) > 0 {
		promptBuilder.WriteString("【之前的对话】\n")
		start := 0
		if len(req.History) > 10 {
			start = len(req.History) - 10
		}
		for _, msg := range req.History[start:] {
			role := "评审员"
			if msg["role"] == "assistant" {
				role = "AI助手"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s:%s\n", role, msg["content"]))
		}
		promptBuilder.WriteString("\n")
	}

	promptBuilder.WriteString("【评审员当前问题】\n")
	promptBuilder.WriteString(strings.TrimSpace(req.Message))

	// ========== 流式响应阶段:切换为 SSE ==========

	flusher := prepareSSEResponse(w)
	if flusher == nil {
		utils.InternalError(w, "不支持流式响应")
		return
	}

	writeReviewAISSEEvent(w, flusher, "connected", map[string]string{
		"phase": "chat",
	})

	startTime := time.Now()
	callResult, callErr := ai.CallAIStream(
		aiCfg,
		systemPrompt,
		promptBuilder.String(),
		func(chunk string) error {
			writeReviewAISSEEvent(w, flusher, "chunk", map[string]string{"chunk": chunk})
			return nil
		},
		nil,
	)

	if callErr != nil {
		log.Printf("[review-ai chat stream] AI调用失败 user=%s assistant=%s err=%v",
			claims.UserID, req.AssistantID, callErr)
		writeReviewAISSEEvent(w, flusher, "error", map[string]string{
			"error": "AI回答失败,请稍后重试",
		})
		return
	}

	latencyMs := time.Since(startTime).Milliseconds()
	log.Printf("[review-ai chat stream] 完成 user=%s assistant=%s tokens=%d latency=%dms",
		claims.UserID, req.AssistantID, callResult.TokensUsed, latencyMs)

	writeReviewAISSEEvent(w, flusher, "done", map[string]interface{}{
		"full_content": callResult.Content,
		"tokens_used":  callResult.TokensUsed,
	})
}

// ==================== 辅助:解析 system prompt ====================

// resolveSystemPrompt 根据 assistant_id 决定使用的 system prompt
//   - assistant_id 为空 → 返回 defaultPrompt(向后兼容)
//   - assistant_id 非空 → 加载助手,校验可见性+激活状态,返回其 full_prompt
//
// 加载失败时返回清晰的用户可见错误;由 caller 决定 HTTP 状态码
func (h *ReviewAIHandler) resolveSystemPrompt(
	r *http.Request,
	userID, role, assistantID, defaultPrompt string,
) (string, error) {
	if strings.TrimSpace(assistantID) == "" {
		return defaultPrompt, nil
	}
	if h.assistantService == nil {
		// 服务未注入的兜底(理论上不会发生)
		log.Printf("[review-ai] AIAssistantService 未初始化,降级使用默认 prompt")
		return defaultPrompt, nil
	}

	actor := services.BuildActorFromClaims(r.Context(), userID, role)
	a, err := h.assistantService.LoadActiveAssistantForUse(r.Context(), actor, assistantID)
	if err != nil {
		// 用户友好错误
		switch {
		case errors.Is(err, repository.ErrAIAssistantNotFound):
			return "", errors.New("选择的 AI 助手不存在")
		case errors.Is(err, repository.ErrAIAssistantInactive):
			return "", errors.New("选择的 AI 助手已停用,请切换其他助手")
		case errors.Is(err, services.ErrAssistantPermDenied):
			return "", errors.New("无权使用该 AI 助手")
		default:
			log.Printf("[review-ai] 加载助手失败 assistant_id=%s err=%v", assistantID, err)
			return "", errors.New("加载 AI 助手失败,请稍后重试")
		}
	}
	if strings.TrimSpace(a.FullPrompt) == "" {
		log.Printf("[review-ai] 助手 %s full_prompt 为空,降级使用默认 prompt", assistantID)
		return defaultPrompt, nil
	}
	return a.FullPrompt, nil
}
