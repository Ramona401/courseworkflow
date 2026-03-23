package handlers

// ==================== P5-4 SSE实时推送处理器 ====================
// GET /api/v1/pipelines/{id}/stream?token=xxx — SSE长连接
// 由于浏览器EventSource不支持自定义header，通过URL query参数传递JWT token
// 此接口不走authMW中间件，内部手动验证token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/services"
)

// SSEHandler SSE推送处理器
type SSEHandler struct {
	authService *services.AuthService // 用于验证JWT token
}

// NewSSEHandler 创建SSE处理器实例
// P5-4修复：需要注入AuthService用于手动验证token
func NewSSEHandler(authService *services.AuthService) *SSEHandler {
	return &SSEHandler{authService: authService}
}

// StreamPipeline GET /api/v1/pipelines/{id}/stream?token=xxx
// SSE长连接：推送Pipeline执行进度事件
// token通过URL query参数传递（EventSource不支持自定义header）
func (h *SSEHandler) StreamPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
		return
	}

	// 从query参数获取token并验证
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}

	// 验证JWT token
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	// 从路径提取Pipeline ID
	pipelineID := extractPipelineIDForSSE(r.URL.Path)
	if pipelineID == "" {
		http.Error(w, "缺少Pipeline ID", http.StatusBadRequest)
		return
	}

	// 检查是否支持Flush（SSE必须）
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
	w.Header().Set("X-Accel-Buffering", "no") // 告诉Nginx不要缓冲

	// 订阅该Pipeline的事件
	ch := services.GlobalSSEHub.Subscribe(pipelineID)
	defer services.GlobalSSEHub.Unsubscribe(pipelineID, ch)

	// 发送初始连接成功事件
	connectEvent := services.SSEEvent{
		EventType:  "connected",
		PipelineID: pipelineID,
		Message:    "SSE连接已建立",
	}
	writeSSEEvent(w, flusher, connectEvent)

	// 监听事件或客户端断开
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[SSE Handler] 客户端断开: pipeline=%s\n", pipelineID)
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
