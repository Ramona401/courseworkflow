package routes

// routes_courseware_review_usage.go
//
// 课件评审体验使用事件独立路由。
//
// 该端点只允许登录用户提交严格白名单事件；
// Handler不接收用户身份、自由文本、搜索关键词、问题正文或教案正文。

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

func registerCoursewareReviewUsageRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
) {
	handler :=
		handlers.NewCoursewareReviewUsageHandler()

	mux.Handle(
		"/api/v1/courseware-review-usage",
		middleware.Chain(
			http.HandlerFunc(
				handler.Record,
			),
			authMW,
		),
	)
}
