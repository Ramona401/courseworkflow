package handlers

// notification_handler.go — 通用通知中心 HTTP 处理器
//
// 四端点（全部登录即可，强制按 claims.UserID 做收件人隔离，无越权面）：
//   GET /api/v1/notifications              列表（?unread_only=true&limit=&offset=）
//   GET /api/v1/notifications/unread-count 未读数（顶栏红点轮询用，极轻）
//   PUT /api/v1/notifications/{id}/read    标单条已读
//   PUT /api/v1/notifications/read-all     全部标已读
//
// 响应风格对齐 courseware_collab_handler：utils.Success/Fail/BadRequest/Unauthorized/InternalError
//   + middleware.GetClaims 双返回值。路径解析自包含（extractNotificationID）。

import (
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// NotificationHandler 通知处理器，持服务引用。
type NotificationHandler struct {
	svc *services.NotificationService
}

// NewNotificationHandler 构造。
func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// HandleList GET /api/v1/notifications — 通知列表。
func (h *NotificationHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	resp, err := h.svc.List(r.Context(), claims.UserID, unreadOnly, limit, offset)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// HandleUnreadCount GET /api/v1/notifications/unread-count — 未读数。
func (h *NotificationHandler) HandleUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	n, err := h.svc.CountUnread(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]int{"unread_count": n})
}

// HandleMarkRead PUT /api/v1/notifications/{id}/read — 标单条已读。
func (h *NotificationHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractNotificationID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少通知ID")
		return
	}
	if err := h.svc.MarkRead(r.Context(), id, claims.UserID); err != nil {
		if err == repository.ErrNotificationNotFound {
			utils.Fail(w, http.StatusNotFound, "通知不存在")
			return
		}
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已标记为已读"})
}

// HandleMarkAllRead PUT /api/v1/notifications/read-all — 全部标已读。
func (h *NotificationHandler) HandleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	n, err := h.svc.MarkAllRead(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]int{"marked": n})
}

// ==================== 路径解析与辅助 ====================

// extractNotificationID 从 /api/v1/notifications/{id}/read 提取通知ID。
func extractNotificationID(p string) string {
	const prefix = "/api/v1/notifications/"
	if !strings.HasPrefix(p, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix)
	rest = strings.TrimSuffix(rest, "/read") // 去掉 /read 末段
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" || strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// parseIntDefault 解析 query 整数，失败/为空返默认值。
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
