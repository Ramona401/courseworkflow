package routes

// routes_notification.go — 通用通知中心路由注册（自构造模块）
//
//   GET /api/v1/notifications              列表（?unread_only=true&limit=&offset=）
//   GET /api/v1/notifications/unread-count 未读数（顶栏红点轮询用，极轻）
//   PUT /api/v1/notifications/{id}/read    标单条已读
//   PUT /api/v1/notifications/read-all     全部标已读
//
// 自构造 service+handler（NotificationService 无状态、无 cfg 依赖），routes.go 的 Setup 只需加一行
//   registerNotificationRoutes(mux, authMW)
// 仿 registerClassProfileRoutes 的极简结构，但比它少一个 cfg 参数（通知无 AES/AI 调用）。
// 全部登录即可，数据隔离在 handler 按 claims.UserID 收口。
//
// 分发顺序坑：unread-count 与 read-all 是固定路径，必须在 {id}/read 通配之前判定，否则会被
//   extractNotificationID 当作通知ID。故带尾斜杠的总入口按 path 后缀分流（固定后缀先匹配），
//   与课件路由 dispatchCoursewareSubRoutes 同思路。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

func registerNotificationRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
) {
	svc := services.NewNotificationService()
	h := handlers.NewNotificationHandler(svc)

	// 无尾斜杠精确路径：GET 列表
	mux.Handle("/api/v1/notifications", middleware.Chain(
		http.HandlerFunc(h.HandleList), authMW))

	// 带尾斜杠总入口：unread-count / read-all / {id}/read 在此按后缀分流
	mux.Handle("/api/v1/notifications/", middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path
			switch {
			case strings.HasSuffix(p, "/unread-count"):
				h.HandleUnreadCount(w, r)
			case strings.HasSuffix(p, "/read-all"):
				h.HandleMarkAllRead(w, r)
			case strings.HasSuffix(p, "/read"):
				h.HandleMarkRead(w, r)
			default:
				http.NotFound(w, r)
			}
		}), authMW))
}
