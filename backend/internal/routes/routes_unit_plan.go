package routes

// routes_unit_plan.go — 单元方案路由注册（大单元备课·独立模块）
//
//   /api/v1/unit-plans         GET 列表 / POST 开始会话
//   /api/v1/unit-plans/{id}    GET 详情 / DELETE 删除
//   /api/v1/unit-plans/{id}/chat  POST 对话一轮
//   /api/v1/unit-plans/{id}/save  POST 定稿保存
//
// 自构造 service+handler，routes.go 的 Setup 只需加一行 registerUnitPlanRoutes(mux, authMW, cfg)。
// 写权限的归属校验在 service 层，路由层只要求登录态（仿 registerCourseOutlineRoutes）。

import (
	"net/http"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

func registerUnitPlanRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	cfg *config.Config,
) {
	svc := services.NewUnitPlanService(cfg)
	h := handlers.NewUnitPlanHandler(svc)

	mux.Handle("/api/v1/unit-plans", middleware.Chain(
		http.HandlerFunc(h.HandleCollection), authMW))
	mux.Handle("/api/v1/unit-plans/", middleware.Chain(
		http.HandlerFunc(h.HandleItem), authMW))
}
