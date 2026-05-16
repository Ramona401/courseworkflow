package handlers

// courseware_index_handler.go — 课件索引生成HTTP处理器
//
// Phase 3: 提供两个接口
//   1. POST /api/v1/coursewares/{id}/generate-index — 触发AI生成索引（异步）
//   2. GET  /api/v1/coursewares/{id}/index-stream    — SSE订阅索引生成进度
//
// 另外新增:
//   3. DELETE /api/v1/coursewares/{id}/pages/{num}    — 删除单页（补充Phase 1缺失）

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件索引处理器 ====================

// CoursewareIndexHandler 课件索引生成处理器
type CoursewareIndexHandler struct {
	indexService *services.CoursewareIndexService
	cwService    *services.CoursewareService
	authService  *services.AuthService
}

// NewCoursewareIndexHandler 创建课件索引处理器
func NewCoursewareIndexHandler(
	indexService *services.CoursewareIndexService,
	cwService *services.CoursewareService,
	authService *services.AuthService,
) *CoursewareIndexHandler {
	return &CoursewareIndexHandler{
		indexService: indexService,
		cwService:    cwService,
		authService:  authService,
	}
}

// ==================== 触发索引生成 ====================

// GenerateIndex POST /api/v1/coursewares/{id}/generate-index — 触发AI生成课件索引
// 异步执行：立即返回200，通过SSE推送进度
func (h *CoursewareIndexHandler) GenerateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 从路径提取课件ID: /api/v1/coursewares/{id}/generate-index
	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 异步执行索引生成（使用独立context，不受HTTP请求取消影响）
	userID := claims.UserID
	go func() {
		asyncCtx := context.Background()
		if err := h.indexService.GenerateIndex(asyncCtx, id, userID); err != nil {
			fmt.Printf("[courseware_index_handler] 索引生成失败: courseware=%s err=%v\n", id, err)
		}
	}()

	utils.Success(w, map[string]interface{}{
		"message":      "课件索引生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// ==================== SSE订阅索引生成进度 ====================

// IndexStream GET /api/v1/coursewares/{id}/index-stream?token=xxx — SSE订阅课件索引生成进度
// 注意：SSE使用EventSource，不支持设置Authorization Header
// 因此token通过URL参数传递，在handler内部验证（不走authMW中间件）
func (h *CoursewareIndexHandler) IndexStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// SSE内部Token验证（从URL参数读取，与Pipeline SSE方式一致）
	token := extractTokenFromQuery(r)
	if token == "" {
		http.Error(w, `{"code":-1,"message":"缺少token参数"}`, http.StatusUnauthorized)
		return
	}
	_, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"code":-1,"message":"token无效或已过期"}`, http.StatusUnauthorized)
		return
	}

	// 从路径提取课件ID: /api/v1/coursewares/{id}/index-stream
	// 从路径提取课件ID: /api/v1/sse/courseware/{id} 或 /api/v1/coursewares/{id}/index-stream
	id := extractCWSSEID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 禁止Nginx缓冲

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	// 订阅SSE事件
	ch := services.GlobalCWSSEHub.Subscribe(id)
	defer services.GlobalCWSSEHub.Unsubscribe(id, ch)

	// 发送连接建立事件
	writeCWSSEEvent(w, flusher, services.CWSSEConnected, map[string]string{
		"courseware_id": id,
		"message":      "SSE连接已建立",
	})

	// 监听事件（带超时，5分钟无事件自动断开）
	timeout := time.After(5 * time.Minute)
	for {
		select {
		case event, open := <-ch:
			if !open {
				// channel被关闭（被新连接替代或主动取消）
				return
			}
			writeCWSSEEvent(w, flusher, event.EventType, event.Data)
			// 完成或错误时结束SSE
			if event.EventType == services.CWSSEIndexDone || event.EventType == services.CWSSEError {
				return
			}
		case <-r.Context().Done():
			// 客户端断开
			return
		case <-timeout:
			// 超时断开
			writeCWSSEEvent(w, flusher, "timeout", map[string]string{
				"message": "SSE连接超时",
			})
			return
		}
	}
}

// ==================== 删除单页（补充Phase 1缺失） ====================

// DeletePage DELETE /api/v1/coursewares/{id}/pages/{num} — 删除课件单页
func (h *CoursewareIndexHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCoursewarePagePath(r.URL.Path)
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	if err := h.cwService.DeletePage(r.Context(), cwID, pageNum, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "页面删除成功"})
}

// ==================== SSE辅助函数 ====================

// writeCWSSEEvent 写入一条SSE事件
func writeCWSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(dataBytes))
	flusher.Flush()
}

// ==================== 路径解析（复用courseware_handler.go中的函数） ====================
// extractCoursewareMiddleID 和 extractCoursewarePagePath
// 已在 courseware_handler.go 中定义，同一个 handlers 包内可直接使用

// extractCWIndexStreamID 从 /api/v1/coursewares/{id}/index-stream 提取ID
// 实际使用 extractCoursewareMiddleID(path, "/index-stream")

// extractCWGenerateIndexID 从 /api/v1/coursewares/{id}/generate-index 提取ID
// 实际使用 extractCoursewareMiddleID(path, "/generate-index")

// ==================== Token认证解析（SSE场景） ====================

// extractCWSSEID 从SSE路径中提取课件ID
// 支持两种路径格式:
//   /api/v1/sse/courseware/{id}
//   /api/v1/coursewares/{id}/index-stream
func extractCWSSEID(path string) string {
	// 格式1: /api/v1/sse/courseware/{id}
	const ssePrefix = "/api/v1/sse/courseware/"
	if strings.HasPrefix(path, ssePrefix) {
		rest := path[len(ssePrefix):]
		rest = strings.TrimRight(rest, "/")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	// 格式2: /api/v1/coursewares/{id}/index-stream
	return extractCoursewareMiddleID(path, "/index-stream")
}

// extractTokenFromQuery 从URL参数获取Token（SSE不支持Header传Token）
func extractTokenFromQuery(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}
	// 兜底：从Authorization header获取
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
