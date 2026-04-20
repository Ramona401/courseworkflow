package handlers

// assistant_designer_handler.go — AI 助手对话式创作 HTTP 接口(TE-DNA 3.0 P0.5)
//
// 对应 Service:services/assistant_designer_service.go
//
// 接口清单:
//   POST /api/v1/ai-assistants/design/chat  — 对话式生成(SSE 流式)
//
// SSE 事件协议:
//   event: connected    → {"phase":"start"}              流建立
//   event: searching    → {"reason":"为何要查库..."}        AI 决定调组件库时
//   event: components   → {"components":[{id,name,library_type,...}]}  查到组件
//   event: chunk        → {"chunk":"..."}                 AI 最终回复流式文本(多次)
//   event: draft_update → {"draft":"...完整草稿..."}       草稿更新(一次)
//   event: done         → {"reply":"...","draft":"...","referenced":[id,id]}
//   event: error        → {"error":"..."}                 任何错误
//
// 错误处理:
//   - 鉴权失败 / 请求体格式错误:走普通 JSON 400/401(非 SSE)
//   - 进入 SSE 后的错误走 error 事件

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AssistantDesignerHandler AI 助手对话式创作处理器
type AssistantDesignerHandler struct {
	designerService *services.AssistantDesignerService
}

// NewAssistantDesignerHandler 构造函数
func NewAssistantDesignerHandler(ds *services.AssistantDesignerService) *AssistantDesignerHandler {
	return &AssistantDesignerHandler{
		designerService: ds,
	}
}

// ==================== 请求体定义 ====================

// DesignChatRequest 对话式创作请求
type DesignChatRequest struct {
	Message      string                       `json:"message"`       // 老师本轮消息
	History      []services.DesignerMessage   `json:"history"`       // 对话历史(可空)
	Subject      string                       `json:"subject"`       // Modal 当前学科
	Grade        string                       `json:"grade"`         // Modal 当前年级
	Scenes       []string                     `json:"scenes"`        // Modal 勾选的场景
	CurrentDraft string                       `json:"current_draft"` // 当前草稿(可空)
}

// ==================== SSE 工具函数 ====================

// writeDesignerSSEEvent 向客户端写入一条 SSE 事件
// 格式与 review_ai_handler/annotation_handler 保持一致,便于前端统一消费
func writeDesignerSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

// prepareDesignerSSE 切换 HTTP 响应为 SSE 流式模式
// 返回 flusher(不支持时返回 nil)
func prepareDesignerSSE(w http.ResponseWriter) http.Flusher {
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

// ==================== 对话式创作接口 ====================

// Chat POST /api/v1/ai-assistants/design/chat (SSE 流式)
//
// 请求体:DesignChatRequest
// 响应:SSE 事件流(见文件头部 SSE 事件协议注释)
//
// 流程:
//   1. 鉴权 + 参数校验(走普通 JSON 错误响应)
//   2. 切换 SSE 响应头,发 connected 事件
//   3. 调 designerService.DesignChat(传入 callbacks 把每个阶段的事件转成 SSE)
//   4. 老师看到:AI 思考 → [若需要]查库进度 → 流式回复 → 草稿更新 → 完成
func (h *AssistantDesignerHandler) Chat(w http.ResponseWriter, r *http.Request) {
	// ========== 前置错误阶段:走普通 JSON 响应 ==========

	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req DesignChatRequest
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

	// ========== 切换 SSE 响应头 ==========

	flusher := prepareDesignerSSE(w)
	if flusher == nil {
		utils.InternalError(w, "不支持流式响应")
		return
	}

	// 发送 connected 事件(让前端知道流已建立)
	writeDesignerSSEEvent(w, flusher, "connected", map[string]string{
		"phase": "start",
	})

	// ========== 构造 Designer 上下文 + SSE 回调 ==========

	dCtx := &services.DesignerContext{
		Subject:      strings.TrimSpace(req.Subject),
		Grade:        strings.TrimSpace(req.Grade),
		Scenes:       req.Scenes,
		CurrentDraft: req.CurrentDraft,
	}

	startTime := time.Now()

	// SSE 回调:每个阶段把事件转成 SSE 推给前端
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
			// draft 如果非空,先单独发一个 draft_update 事件,方便前端"草稿区"单独订阅
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
			log.Printf("[designer chat] 完成 user=%s subject=%s grade=%s ref_count=%d latency=%dms",
				claims.UserID, dCtx.Subject, dCtx.Grade, len(referenced),
				time.Since(startTime).Milliseconds())
		},
		OnError: func(errMsg string) {
			writeDesignerSSEEvent(w, flusher, "error", map[string]string{
				"error": errMsg,
			})
		},
	}

	// ========== 调用 Service ==========

	if err := h.designerService.DesignChat(r.Context(), req.Message, req.History, dCtx, callbacks); err != nil {
		// 此时已经进入 SSE 模式,错误已通过 OnError 回调推给前端,这里只记日志
		log.Printf("[designer chat] DesignChat 失败: user=%s err=%v", claims.UserID, err)
	}
}
