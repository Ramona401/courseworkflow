package handlers

// ==================== 教案生成HTTP处理器 ====================
// Phase3：教案对话式生成接口
//
// 路由（均在 /api/v1/lesson-plans/ 前缀下）：
//   POST /plans/start-conversation     — 开始备课会话（创建教案+AI开场白）
//   POST /plans/:id/chat               — 发送对话消息（异步AI回复，SSE推送）
//   POST /plans/:id/trigger-review     — 触发AI质量评审（异步，SSE推送结果）
//   POST /plans/:id/apply-suggestions  — 应用AI建议（异步优化+重新评审）
//   GET  /plans/:id/conversation       — 获取完整对话记录
//   GET  /sse/plans/:id/stream?token=  — SSE流（教案生成实时推送）

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// LessonPlanGenHandler 教案生成处理器
type LessonPlanGenHandler struct {
	genService  *services.LessonPlanGenService
	authService *services.AuthService // 直接使用具体类型，与SSEHandler保持一致
}

// 模块日志
var lpGenHandlerLog = logger.WithModule("lp_gen_handler")

// NewLessonPlanGenHandler 创建教案生成处理器
func NewLessonPlanGenHandler(
	genService *services.LessonPlanGenService,
	authService *services.AuthService,
) *LessonPlanGenHandler {
	return &LessonPlanGenHandler{
		genService:  genService,
		authService: authService,
	}
}

// ==================== POST /plans/start-conversation ====================

// StartConversation 开始备课会话
// 创建教案记录 + 静默注入背景组件 + AI生成开场白
// POST /api/v1/lesson-plans/plans/start-conversation
func (h *LessonPlanGenHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	lp, openingMsg, err := h.genService.StartConversation(r.Context(), &req, userID)
	if err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"plan":            lp,
		"opening_message": openingMsg,
	})
}

// ==================== POST /plans/:id/chat ====================

// Chat 处理教师对话输入
// AI异步生成回复，通过SSE推送给前端
// POST /api/v1/lesson-plans/plans/{id}/chat
func (h *LessonPlanGenHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	planID := extractLPGenID(r.URL.Path, "/chat")
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.LessonPlanChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	req.PlanID = planID

	if err := h.genService.Chat(r.Context(), &req, userID); err != nil {
		h.handleGenError(w, err)
		return
	}

	// 立即返回ACK，AI回复通过SSE异步推送
	utils.Success(w, map[string]string{
		"status":  "processing",
		"message": "AI正在思考，请通过SSE获取回复",
	})
}

// ==================== POST /plans/:id/trigger-review ====================

// TriggerAIReview 触发AI质量评审
// POST /api/v1/lesson-plans/plans/{id}/trigger-review
func (h *LessonPlanGenHandler) TriggerAIReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	planID := extractLPGenID(r.URL.Path, "/trigger-review")
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	if err := h.genService.TriggerAIReview(r.Context(), planID, userID); err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"status":  "reviewing",
		"message": "AI评审已启动，请通过SSE获取结果",
	})
}

// ==================== POST /plans/:id/apply-suggestions ====================

// ApplyAISuggestions 应用AI评审建议并重新评审
// POST /api/v1/lesson-plans/plans/{id}/apply-suggestions
func (h *LessonPlanGenHandler) ApplyAISuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	planID := extractLPGenID(r.URL.Path, "/apply-suggestions")
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.ApplyAISuggestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	req.PlanID = planID

	if err := h.genService.ApplyAISuggestions(r.Context(), &req, userID); err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"status":  "optimizing",
		"message": "AI优化已启动，请通过SSE获取更新",
	})
}

// ==================== GET /plans/:id/conversation ====================

// GetConversation 获取完整对话记录
// GET /api/v1/lesson-plans/plans/{id}/conversation
func (h *LessonPlanGenHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	planID := extractLPGenID(r.URL.Path, "/conversation")
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	msgs, err := h.genService.GetConversation(r.Context(), planID, userID)
	if err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"messages": msgs,
		"total":    len(msgs),
	})
}

// ==================== GET /sse/plans/:id/stream ====================

// StreamPlan 教案生成SSE推送流
// GET /api/v1/lesson-plans/sse/plans/{id}/stream?token=xxx
// 注意：此路由不经过authMW，在内部验证token（与Pipeline SSE保持一致）
func (h *LessonPlanGenHandler) StreamPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
		return
	}

	// 从query参数获取token（SSE不适合传Authorization header）
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}

	// 验证token（返回*JWTClaims，忽略具体值只验证有效性）
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	// 从路径提取教案ID
	planID := extractPlanIDForSSE(r.URL.Path)
	if planID == "" {
		http.Error(w, "缺少教案ID", http.StatusBadRequest)
		return
	}

	// 检查是否支持SSE Flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持SSE流", http.StatusInternalServerError)
		return
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // 禁止Nginx缓冲

	// 订阅教案SSE事件
	ch := services.GlobalLPSSEHub.Subscribe(planID)
	defer services.GlobalLPSSEHub.Unsubscribe(planID, ch)

	// 推送连接建立事件
	writeLPSSEEvent(w, flusher, models.LPSSEEvent{
		EventType: models.LPSSEConnected,
		PlanID:    planID,
	})

	lpGenHandlerLog.Debug("教案SSE连接建立", "plan_id", planID, "remote_addr", r.RemoteAddr)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// 客户端断开连接（正常事件，DEBUG级别）
			lpGenHandlerLog.Debug("教案SSE客户端断开", "plan_id", planID)
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeLPSSEEvent(w, flusher, event)
			// done或error事件后主动关闭连接
			if event.EventType == models.LPSSEDone || event.EventType == models.LPSSEError {
				return
			}
		}
	}
}

// ==================== 辅助函数 ====================

// writeLPSSEEvent 写入一条教案SSE事件到HTTP响应
func writeLPSSEEvent(w http.ResponseWriter, flusher http.Flusher, event models.LPSSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", string(event.EventType), string(data))
	flusher.Flush()
}

// extractLPGenID 从教案生成子路径中提取ID
// 示例：/api/v1/lesson-plans/plans/{id}/chat → id
func extractLPGenID(path string, suffix string) string {
	prefix := "/api/v1/lesson-plans/plans/"
	return extractMiddleSegment(path, prefix, suffix)
}

// extractPlanIDForSSE 从教案SSE路径中提取教案ID
// 示例：/api/v1/lesson-plans/sse/plans/{id}/stream → id
func extractPlanIDForSSE(path string) string {
	streamIdx := strings.LastIndex(path, "/stream")
	if streamIdx <= 0 {
		return ""
	}
	path = path[:streamIdx]
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return ""
	}
	id := path[lastSlash+1:]
	if id == "" || id == "plans" {
		return ""
	}
	return id
}

// handleGenError 教案生成操作统一错误处理
func (h *LessonPlanGenHandler) handleGenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrLPGenSubjectRequired),
		errors.Is(err, services.ErrLPGenGradeRequired),
		errors.Is(err, services.ErrLPGenTopicRequired):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrLPGenUnauthorized),
		errors.Is(err, services.ErrLPGenNotEditable):
		utils.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrLPGenPlanNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		lpGenHandlerLog.Error("教案生成操作失败", "error", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}
