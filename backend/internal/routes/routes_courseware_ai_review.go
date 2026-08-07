package routes

// routes_courseware_ai_review.go
//
// 课件AI审核助手独立路由组。
//
// 与人工课件审核路由分开：
//
//   人工审核：
//     /api/v1/courseware-reviews/*
//
//   AI辅助审核：
//     /api/v1/courseware-ai-reviews/*
//
// 所有路由均要求登录；课件、学校、教育域、审核级别和助手权限
// 由服务层继续细分。

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCoursewareAIReviewRoutes 注册课件AI审核助手路由。
func registerCoursewareAIReviewRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	handler *handlers.CoursewareAIReviewHandler,
) {
	collectionHandler :=
		middleware.Chain(
			http.HandlerFunc(
				handler.HandleCollection,
			),
			authMW,
		)

	itemHandler :=
		middleware.Chain(
			http.HandlerFunc(
				handler.HandleItem,
			),
			authMW,
		)

	// Go ServeMux按最长路径匹配。
	// 更具体的/items/入口只接管指令版本和旧confirm路径，
	// 其他整改项路径由Handler内部委托回原HandleItem。
	instructionVersionHandler :=
		middleware.Chain(
			http.HandlerFunc(
				handler.HandleReviewInstructionVersionRoute,
			),
			authMW,
		)

	mux.Handle(
		"/api/v1/courseware-ai-reviews",
		collectionHandler,
	)

	mux.Handle(
		"/api/v1/courseware-ai-reviews/items/",
		instructionVersionHandler,
	)

	mux.Handle(
		"/api/v1/courseware-ai-reviews/",
		itemHandler,
	)
}
