package routes

// routes_course_outline.go — 课程大纲路由注册（大单元备课能力·批次一）
//
//   /api/v1/course-outlines        GET 列表(全员) / POST 创建(归属者)
//   /api/v1/course-outlines/{id}   GET 详情(全员) / PUT 更新 / DELETE 删除(归属者)
//
// 写权限的归属校验在 service 层做（组长/校管/admin），路由层只要求登录态。
// 仿照 registerCurriculumRoutes 的极简注册风格。

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCourseOutlineRoutes 注册课程大纲路由
func registerCourseOutlineRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	coHandler *handlers.CourseOutlineHandler,
) {
	// 集合路径（精确匹配，无尾斜杠）：GET 列表 / POST 创建
	mux.Handle("/api/v1/course-outlines", middleware.Chain(
		http.HandlerFunc(coHandler.HandleCollection), authMW))

	// 单条路径（带尾斜杠通配）：GET 详情 / PUT 更新 / DELETE 删除
	mux.Handle("/api/v1/course-outlines/", middleware.Chain(
		http.HandlerFunc(coHandler.HandleItem), authMW))
}
