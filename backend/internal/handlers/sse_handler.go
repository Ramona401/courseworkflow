package handlers

// SSE实时推送处理器
// Phase8日志升级：
//   - 客户端正常断开 → DEBUG（高频事件，不需要在生产日志中出现）

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/services"
)

// SSEHandler SSE推送处理器
type SSEHandler struct {
	authService *services.AuthService
}

// 模块日志
var sseHandlerLog = logger.WithModule("sse_handler")

// NewSSEHandler 创建SSE处理器实例
func NewSSEHandler(authService *services.AuthService) *SSEHandler {
	return &SSEHandler{authService: authService}
}

// StreamPipeline GET /api/v1/pipelines/{id}/stream?token=xxx
func (h *SSEHandler) StreamPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
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

	pipelineID := extractPipelineIDForSSE(r.URL.Path)
	if pipelineID == "" {
		http.Error(w, "缺少Pipeline ID", http.StatusBadRequest)
		return
	}

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
	w.Header().Set("X-Accel-Buffering", "no")

	ch := services.GlobalSSEHub.Subscribe(pipelineID)
	defer services.GlobalSSEHub.Unsubscribe(pipelineID, ch)

	connectEvent := services.SSEEvent{
		EventType:  "connected",
		PipelineID: pipelineID,
		Message:    "SSE连接已建立",
	}
	writeSSEEvent(w, flusher, connectEvent)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// DEBUG：客户端断开是高频正常事件，不需要在INFO级别出现
			sseHandlerLog.Debug("SSE客户端断开",
				"pipeline_id", pipelineID,
				"remote_addr", r.RemoteAddr,
			)
			return

		case event, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, event)

			if event.EventType == "pipeline_done" || event.EventType == "pipeline_error" {
				return
			}
		}
	}
}

// writeSSEEvent 写入一条SSE事件到HTTP响应
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event services.SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.EventType, string(data))
	flusher.Flush()
}

// extractPipelineIDForSSE 从SSE路径中提取Pipeline ID
func extractPipelineIDForSSE(path string) string {
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
	if id == "" || id == "pipelines" {
		return ""
	}
	return id
}
