package handlers

// lesson_plan_gen_handler.go — 教案生成HTTP处理器

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

type LessonPlanGenHandler struct {
	genService  *services.LessonPlanGenService
	authService *services.AuthService
}

var lpGenHandlerLog = logger.WithModule("lp_gen_handler")

func NewLessonPlanGenHandler(
	genService *services.LessonPlanGenService,
	authService *services.AuthService,
) *LessonPlanGenHandler {
	return &LessonPlanGenHandler{
		genService:  genService,
		authService: authService,
	}
}

func (h *LessonPlanGenHandler) StartConversation(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	lp, openingMsg, err := h.genService.StartConversation(
		r.Context(),
		&req,
		userID,
	)
	if err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"plan":            lp,
		"opening_message": openingMsg,
	})
}

func (h *LessonPlanGenHandler) Chat(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	if err := h.genService.ChatWithValidatedComponents(
		r.Context(),
		&req,
		userID,
	); err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"status":  "processing",
		"message": "AI正在思考，请通过SSE获取回复",
	})
}

func (h *LessonPlanGenHandler) TriggerAIReview(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	if err := h.genService.TriggerAIReview(
		r.Context(),
		planID,
		userID,
	); err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"status":  "reviewing",
		"message": "AI评审已启动，请通过SSE获取结果",
	})
}

func (h *LessonPlanGenHandler) ApplyAISuggestions(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	if err := h.genService.ApplyAISuggestions(
		r.Context(),
		&req,
		userID,
	); err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]string{
		"status":  "optimizing",
		"message": "AI优化已启动，请通过SSE获取更新",
	})
}

func (h *LessonPlanGenHandler) GetConversation(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	messages, contextCapsule, err :=
		h.genService.GetConversationWithContextCapsule(
			r.Context(),
			planID,
			userID,
		)
	if err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"messages":        messages,
		"total":           len(messages),
		"context_capsule": contextCapsule,
	})
}

// StreamPlan 建立教案实时事件连接。
//
// 同一教案允许多个标签页、窗口或设备并行订阅。
// 每个请求只注销自己的channel，不关闭同一教案的其它连接。
func (h *LessonPlanGenHandler) StreamPlan(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(w, utils.MsgMethodGetOnly, http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(
			w,
			`{"code":-1,"message":"缺少token参数"}`,
			http.StatusUnauthorized,
		)
		return
	}

	if _, err := h.authService.ValidateToken(token); err != nil {
		http.Error(
			w,
			`{"code":-1,"message":"token无效或已过期"}`,
			http.StatusUnauthorized,
		)
		return
	}

	planID := extractPlanIDForSSE(r.URL.Path)
	if planID == "" {
		http.Error(w, utils.MsgMissingLessonPlanID, http.StatusBadRequest)
		return
	}

	finishSSEHandshake, handshakeOK := beginSSEHandshake(w)
	if !handshakeOK {
		return
	}
	defer finishSSEHandshake()

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

	finishSSEHandshake()

	writeLPSSEEvent(w, flusher, models.LPSSEEvent{
		EventType: models.LPSSEConnected,
		PlanID:    planID,
	})

	lpGenHandlerLog.Debug(
		"教案SSE连接建立",
		"plan_id",
		planID,
		"remote_addr",
		r.RemoteAddr,
		"subscriber_count",
		services.GlobalLPSSEHub.SubscriberCount(planID),
	)

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			lpGenHandlerLog.Debug(
				"教案SSE客户端断开",
				"plan_id",
				planID,
			)
			return

		case event, open := <-ch:
			if !open {
				return
			}

			writeLPSSEEvent(w, flusher, event)

			if event.EventType == models.LPSSEDone ||
				event.EventType == models.LPSSEError {
				return
			}
		}
	}
}

func writeLPSSEEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	event models.LPSSEEvent,
) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(
		w,
		"event: %s\ndata: %s\n\n",
		string(event.EventType),
		string(data),
	)

	flusher.Flush()
}

func extractLPGenID(
	path string,
	suffix string,
) string {
	const prefix = "/api/v1/lesson-plans/plans/"
	return extractMiddleSegment(path, prefix, suffix)
}

func extractPlanIDForSSE(
	path string,
) string {
	streamIndex := strings.LastIndex(path, "/stream")
	if streamIndex <= 0 {
		return ""
	}

	path = path[:streamIndex]

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

func (h *LessonPlanGenHandler) handleGenError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, services.ErrLPGenServiceDraining):
		utils.Fail(w, http.StatusServiceUnavailable, err.Error())

	case errors.Is(err, services.ErrLPGenTaskRunning),
		errors.Is(err, services.ErrLPGenPublishIntent):
		utils.Fail(w, http.StatusConflict, err.Error())

	case errors.Is(err, services.ErrLPGenSubjectRequired),
		errors.Is(err, services.ErrLPGenGradeRequired),
		errors.Is(err, services.ErrLPGenTopicRequired),
		errors.Is(err, services.ErrLPGenImportContentRequired),
		errors.Is(err, services.ErrLPGenImportSourceInvalid),
		errors.Is(err, services.ErrLessonPlanWordFileRequired),
		errors.Is(err, services.ErrLessonPlanWordFileTooLarge),
		errors.Is(err, services.ErrLessonPlanWordFileInvalid),
		errors.Is(err, services.ErrLessonPlanWordParseFailed),
		errors.Is(err, services.ErrLPTextbookSelectionInvalid),
		errors.Is(err, services.ErrComponentSelectionInvalid),
		errors.Is(err, services.ErrComponentEducationDomainInvalid),
		errors.Is(err, services.ErrOutlineExactSelectionInvalid):
		utils.BadRequest(w, err.Error())

	case errors.Is(err, services.ErrLPGenUnauthorized),
		errors.Is(err, services.ErrLPGenNotEditable),
		errors.Is(err, services.ErrLPCreationEducationDomainRequired),
		errors.Is(err, services.ErrLPCreationEducationDomainConflict),
		errors.Is(err, services.ErrLPTextbookEducationDomainDenied),
		errors.Is(err, services.ErrOutlineExactSelectionForbidden),
		errors.Is(err, services.ErrOutlineEducationDomainRequired),
		errors.Is(err, services.ErrOutlineEducationDomainConflict),
		errors.Is(err, services.ErrOutlineEducationDomainMismatch):
		utils.Fail(w, http.StatusForbidden, err.Error())

	case errors.Is(err, services.ErrOutlineExactSelectionUnavailable),
		errors.Is(err, services.ErrLPGenPlanNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())

	case errors.Is(err, services.ErrLPCreationEducationDomainResolveFailed),
		errors.Is(err, services.ErrOutlineEducationDomainResolveFailed):
		lpGenHandlerLog.Error(
			"教案创建教育域或课程大纲解析失败",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"教育域或课程大纲解析失败，请稍后重试",
		)

	case errors.Is(err, services.ErrLPGenImportReviewStageRequired):
		lpGenHandlerLog.Error(
			"导入流程缺少AI评审阶段",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"导入流程配置异常，请联系管理员",
		)

	default:
		lpGenHandlerLog.Error(
			"教案生成操作失败",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"操作失败，请稍后重试",
		)
	}
}

func (h *LessonPlanGenHandler) ImportExistingPlan(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	response, err := h.genService.ImportExistingPlan(
		r.Context(),
		&req,
		userID,
	)
	if err != nil {
		h.handleGenError(w, err)
		return
	}

	utils.Success(w, response)
}
