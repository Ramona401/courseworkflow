package handlers

// assistant_designer_handler.go — AI 助手对话式创作与教学风格画像接口
//
// 接口清单：
//   POST /api/v1/ai-assistants/design/chat
//     对话式生成助手，使用 SSE 流式返回。
//
//   POST /api/v1/ai-assistants/design/profile-materials
//     一次性读取平台教案或前端提取的 Word/PDF/粘贴文字，
//     返回可编辑的教学风格与成长画像，使用普通 JSON 返回。
//
// SSE 事件协议：
//   event: connected    → {"phase":"start"}
//   event: searching    → {"reason":"为何要查库..."}
//   event: components   → {"components":[...]}
//   event: chunk        → {"chunk":"..."}
//   event: draft_update → {"draft":"...完整草稿..."}
//   event: done         → {"reply":"...","draft":"...","referenced":[...]}
//   event: error        → {"error":"..."}

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AssistantDesignerHandler AI 助手创作与教学风格画像处理器。
type AssistantDesignerHandler struct {
	designerService *services.AssistantDesignerService
}

// NewAssistantDesignerHandler 构造函数。
func NewAssistantDesignerHandler(ds *services.AssistantDesignerService) *AssistantDesignerHandler {
	return &AssistantDesignerHandler{
		designerService: ds,
	}
}

// ==================== SSE 工具函数 ====================

// writeDesignerSSEEvent 向客户端写入一条 SSE 事件。
func writeDesignerSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

// prepareDesignerSSE 切换 HTTP 响应为 SSE 流式模式。
func prepareDesignerSSE(w http.ResponseWriter) http.Flusher {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")
	return flusher
}

// ==================== 教学风格画像 ====================

// ProfileMaterials POST /api/v1/ai-assistants/design/profile-materials
//
// 普通 JSON 接口，不使用 SSE。
// 原始材料只在本次请求中使用，不写数据库。
func (h *AssistantDesignerHandler) ProfileMaterials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	if h.designerService == nil {
		utils.InternalError(w, "Designer 服务未初始化")
		return
	}

	var req services.StyleProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	resp, err := h.designerService.AnalyzeStyleProfile(
		r.Context(),
		claims.UserID,
		claims.Role,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrStyleProfileNoMaterials),
			errors.Is(err, services.ErrStyleProfileTooManyMaterials),
			errors.Is(err, services.ErrStyleProfileMaterialInvalid),
			errors.Is(err, services.ErrStyleProfileMaterialTooLong),
			errors.Is(err, services.ErrStyleProfileTotalTooLong),
			errors.Is(err, services.ErrStyleProfilePlanEmpty):
			utils.BadRequest(w, err.Error())

		case errors.Is(err, services.ErrStyleProfilePlanNotAccessible):
			utils.Forbidden(w, err.Error())

		default:
			log.Printf(
				"[designer profile] 画像生成失败 user=%s error=%v",
				claims.UserID,
				err,
			)
			utils.InternalError(w, err.Error())
		}
		return
	}

	utils.Success(w, resp)
}

// ==================== 对话式创作 ====================

// Chat POST /api/v1/ai-assistants/design/chat
//
// 流程：
//  1. 鉴权和参数校验。
//  2. 切换 SSE 响应头并发送 connected。
//  3. 调用 Designer 两阶段 AI 服务。
//  4. 通过回调推送查库、组件、流式文本、草稿和完成事件。
func (h *AssistantDesignerHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req struct {
		Message      string                     `json:"message"`
		History      []services.DesignerMessage `json:"history"`
		Subject      string                     `json:"subject"`
		Grade        string                     `json:"grade"`
		Scenes       []string                   `json:"scenes"`
		CurrentDraft string                     `json:"current_draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		utils.BadRequest(w, "消息内容不能为空")
		return
	}
	if h.designerService == nil {
		utils.InternalError(w, "Designer 服务未初始化")
		return
	}

	flusher := prepareDesignerSSE(w)
	if flusher == nil {
		utils.InternalError(w, "不支持流式响应")
		return
	}

	writeDesignerSSEEvent(w, flusher, "connected", map[string]string{
		"phase": "start",
	})

	dCtx := &services.DesignerContext{
		Subject:      strings.TrimSpace(req.Subject),
		Grade:        strings.TrimSpace(req.Grade),
		Scenes:       req.Scenes,
		CurrentDraft: req.CurrentDraft,
	}

	startTime := time.Now()

	callbacks := &services.DesignerStreamCallbacks{
		OnSearching: func(reason string) {
			writeDesignerSSEEvent(w, flusher, "searching", map[string]string{
				"reason": reason,
			})
		},
		OnComponents: func(briefs []*services.ComponentBrief) {
			writeDesignerSSEEvent(w, flusher, "components", map[string]interface{}{
				"components": briefs,
			})
		},
		OnChunk: func(text string) {
			writeDesignerSSEEvent(w, flusher, "chunk", map[string]string{
				"chunk": text,
			})
		},
		OnDone: func(reply, draft string, referenced []string) {
			if strings.TrimSpace(draft) != "" {
				writeDesignerSSEEvent(w, flusher, "draft_update", map[string]string{
					"draft": draft,
				})
			}
			writeDesignerSSEEvent(w, flusher, "done", map[string]interface{}{
				"reply":      reply,
				"draft":      draft,
				"referenced": referenced,
			})
			log.Printf(
				"[designer chat] 完成 user=%s subject=%s grade=%s ref_count=%d latency=%dms",
				claims.UserID,
				dCtx.Subject,
				dCtx.Grade,
				len(referenced),
				time.Since(startTime).Milliseconds(),
			)
		},
		OnError: func(errMsg string) {
			writeDesignerSSEEvent(w, flusher, "error", map[string]string{
				"error": errMsg,
			})
		},
	}

	if err := h.designerService.DesignChat(
		r.Context(),
		claims.UserID,
		req.Message,
		req.History,
		dCtx,
		callbacks,
	); err != nil {
		log.Printf(
			"[designer chat] DesignChat 失败: user=%s err=%v",
			claims.UserID,
			err,
		)
	}
}
