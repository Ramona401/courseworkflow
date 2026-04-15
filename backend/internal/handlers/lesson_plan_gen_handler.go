package handlers

// ==================== 教案生成HTTP处理器 ====================

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
	authService *services.AuthService
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

func (h *LessonPlanGenHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *LessonPlanGenHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	planID := extractLPGenID(r.URL.Path, "/chat")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.LessonPlanChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	req.PlanID = planID
	if err := h.genService.Chat(r.Context(), &req, userID); err != nil {
		h.handleGenError(w, err)
		return
	}
	utils.Success(w, map[string]string{
		"status":  "processing",
		"message": "AI正在思考，请通过SSE获取回复",
	})
}

// ==================== POST /plans/:id/trigger-review ====================

func (h *LessonPlanGenHandler) TriggerAIReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	planID := extractLPGenID(r.URL.Path, "/trigger-review")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
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

func (h *LessonPlanGenHandler) ApplyAISuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	planID := extractLPGenID(r.URL.Path, "/apply-suggestions")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.ApplyAISuggestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *LessonPlanGenHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	planID := extractLPGenID(r.URL.Path, "/conversation")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
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

func (h *LessonPlanGenHandler) StreamPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, utils.MsgMethodGetOnly, http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}
	planID := extractPlanIDForSSE(r.URL.Path)
	if planID == "" {
		http.Error(w, utils.MsgMissingLessonPlanID, http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持SSE流", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := services.GlobalLPSSEHub.Subscribe(planID)
	defer services.GlobalLPSSEHub.Unsubscribe(planID, ch)

	writeLPSSEEvent(w, flusher, models.LPSSEEvent{
		EventType: models.LPSSEConnected,
		PlanID:    planID,
	})
	lpGenHandlerLog.Debug("教案SSE连接建立", "plan_id", planID, "remote_addr", r.RemoteAddr)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			lpGenHandlerLog.Debug("教案SSE客户端断开", "plan_id", planID)
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeLPSSEEvent(w, flusher, event)
			if event.EventType == models.LPSSEDone || event.EventType == models.LPSSEError {
				return
			}
		}
	}
}

// ==================== 辅助函数 ====================

func writeLPSSEEvent(w http.ResponseWriter, flusher http.Flusher, event models.LPSSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", string(event.EventType), string(data))
	flusher.Flush()
}

func extractLPGenID(path string, suffix string) string {
	prefix := "/api/v1/lesson-plans/plans/"
	return extractMiddleSegment(path, prefix, suffix)
}

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

// ==================== POST /plans/import-existing（v108新增）====================

// ImportExistingPlan 导入已有教案
// 前端负责解析Word/PDF，将纯文本+元信息POST到此接口
func (h *LessonPlanGenHandler) ImportExistingPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.ImportExistingPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	resp, err := h.genService.ImportExistingPlan(r.Context(), &req, userID)
	if err != nil {
		h.handleGenError(w, err)
		return
	}
	utils.Success(w, resp)
}
