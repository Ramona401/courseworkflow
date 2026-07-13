package routes

// routes_unit_plan.go — 单元方案及其参考资料路由注册
//
// 单元方案：
//   /api/v1/unit-plans
//   /api/v1/unit-plans/{id}
//   /api/v1/unit-plans/{id}/chat
//   /api/v1/unit-plans/{id}/save
//   /api/v1/unit-plans/mountable
//
// 大单元参考资料：
//   GET/POST /api/v1/unit-plan-materials?unit_plan_id={id}
//   DELETE   /api/v1/unit-plan-materials/{material_id}?unit_plan_id={id}

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
	unitPlanService := services.NewUnitPlanService(cfg)
	unitPlanHandler := handlers.NewUnitPlanHandler(unitPlanService)

	materialService := services.NewUnitPlanMaterialService()
	materialHandler := handlers.NewUnitPlanMaterialHandler(materialService)

	mux.Handle("/api/v1/unit-plans", middleware.Chain(
		http.HandlerFunc(unitPlanHandler.HandleCollection),
		authMW,
	))

	mux.Handle("/api/v1/unit-plans/", middleware.Chain(
		http.HandlerFunc(unitPlanHandler.HandleItem),
		authMW,
	))

	mux.Handle("/api/v1/unit-plan-materials", middleware.Chain(
		http.HandlerFunc(materialHandler.HandleCollection),
		authMW,
	))

	mux.Handle("/api/v1/unit-plan-materials/", middleware.Chain(
		http.HandlerFunc(materialHandler.HandleItem),
		authMW,
	))
}
