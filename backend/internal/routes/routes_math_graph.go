package routes

// routes_math_graph.go — 批次A·老师自定义数学图形路由注册(2026-07-08)
//
// 自构造范式:本文件内部构造 MathGraphAIService 与 MathGraphHandler,
// routes.go 的 Setup 只需一行 registerMathGraphRoutes(mux, authMW, cfg) 调用,
// 保持主路由文件的增量最小化。
//
// 端点:
//   POST /api/v1/math-graph/generate — AI 生成/改编 JSXGraph 构造代码(登录即可)

import (
	"net/http"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/services"
)

// registerMathGraphRoutes 注册数学图形 AI 定制路由。
// authMW 形参用底层函数类型声明,与 middleware.AuthMiddleware 返回值天然兼容。
func registerMathGraphRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler, cfg *config.Config) {
	svc := services.NewMathGraphAIService(cfg)
	h := handlers.NewMathGraphHandler(svc)

	// AI 生成/改编构造代码(登录即可;积分前置检查与消费经 CallAI 积分钩子天然生效)
	mux.Handle("/api/v1/math-graph/generate", authMW(http.HandlerFunc(h.Generate)))
}
