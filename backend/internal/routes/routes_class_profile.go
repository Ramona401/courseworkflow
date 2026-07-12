package routes

// routes_class_profile.go — 班级学情路由注册（差异化教学·老师私有资料，独立模块）
//
//   /api/v1/class-profiles        GET 列表 / POST 新建
//   /api/v1/class-profiles/{id}   GET 详情 / PUT 更新 / DELETE 删除
//   /api/v1/class-profiles/{id}/students[...]            学生档案 CRUD（批次2a，HandleItem 内分发）
//   /api/v1/class-profiles/{id}/students/import          成绩单导入（批次2b，HandleItem 内分发）
//   /api/v1/class-profiles/{id}/students/summarize       AI 总结学情（批次2c，HandleItem 内分发）
//
// 自构造 service+handler，routes.go 的 Setup 只需加一行 registerClassProfileRoutes(mux, authMW, cfg)。
// 纯个人，登录即可；归属校验（created_by==userID）在 service 层。
// 仿 registerUnitPlanRoutes 的极简结构（签名一致：mux, authMW, cfg）。
//
// 批次2c 起 cfg 被真正用上：handler 持有 cfg 供 AI 总结（SummarizeClassProfile）取 AES 密钥与
// 兜底模型/网关。故构造 handler 时把 cfg 传入（NewClassProfileHandler(svc, cfg)）。

import (
"net/http"

"tedna/internal/config"
"tedna/internal/handlers"
"tedna/internal/middleware"
"tedna/internal/services"
)

func registerClassProfileRoutes(
mux *http.ServeMux,
authMW func(http.Handler) http.Handler,
cfg *config.Config,
) {
svc := services.NewClassProfileService()
// 批次2c：handler 持 cfg 供 AI 总结调用（取 AES 密钥 + 兜底模型/网关）
h := handlers.NewClassProfileHandler(svc, cfg)

mux.Handle("/api/v1/class-profiles", middleware.Chain(
http.HandlerFunc(h.HandleCollection), authMW))
mux.Handle("/api/v1/class-profiles/", middleware.Chain(
http.HandlerFunc(h.HandleItem), authMW))
}
