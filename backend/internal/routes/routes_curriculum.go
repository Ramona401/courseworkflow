package routes

// routes_curriculum.go — 课程知识库公共路由注册
//
// 平台级公共接口（/api/v1/curriculum/*），登录即可访问：
//   GET /api/v1/curriculum/knowledge-points  — 按学科+年级查知识点清单
//   GET /api/v1/curriculum/textbook-units    — 查教材单元
//   GET /api/v1/curriculum/publishers        — 查某学科某年级的教材版本
//
// 设计：故意不挂在 /coursewares/ 下，因为知识库是学科无关、场景无关的公共底座，
//       课件工坊与备课工坊（教案撰写）将来都从这里取数。

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCurriculumRoutes 注册课程知识库公共只读路由
func registerCurriculumRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	curriculumHandler *handlers.CurriculumHandler,
) {
	mux.Handle("/api/v1/curriculum/knowledge-points", middleware.Chain(
		http.HandlerFunc(curriculumHandler.ListKnowledgePoints), authMW))

	mux.Handle("/api/v1/curriculum/textbook-units", middleware.Chain(
		http.HandlerFunc(curriculumHandler.ListTextbookUnits), authMW))

	mux.Handle("/api/v1/curriculum/publishers", middleware.Chain(
		http.HandlerFunc(curriculumHandler.ListPublishers), authMW))
}
