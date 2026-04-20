package routes

// routes_assistant_feedback.go — AI 助手反馈路由注册(P1 收尾 · 2026-04-20)
//
// 路由清单:
//   POST   /api/v1/assistant-feedback                    创建反馈(登录即可)
//   DELETE /api/v1/assistant-feedback/{id}               删除自己的反馈
//   GET    /api/v1/assistant-feedback                    列表(admin only)
//   GET    /api/v1/assistants/{id}/feedback-stats        某助手统计(登录即可)
//
// 权限策略:
//   - 创建/删除/统计:authMW(登录即可)
//   - 列表查询:authMW + adminOnly

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerAssistantFeedbackRoutes 注册 AI 助手反馈相关路由
//
// 挂载点:routes.go 的 Setup() 函数中调用
// 参数说明:
//   - mux:    主 ServeMux
//   - authMW: 登录认证中间件
//   - adminOnly: admin 角色限制中间件
//   - feedbackHandler: 反馈处理器
func registerAssistantFeedbackRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	feedbackHandler *handlers.AssistantFeedbackHandler,
) {
	// 创建反馈 + 列表(admin)— 精确匹配不含尾斜杠的 /api/v1/assistant-feedback
	mux.Handle("/api/v1/assistant-feedback",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				// 创建反馈:登录即可
				feedbackHandler.Create(w, r)
			case http.MethodGet:
				// 列表查询:需要 admin,但 Chain 层已经通过 authMW 进来
				// 这里手动调 adminOnly 包装一下
				adminOnly(http.HandlerFunc(feedbackHandler.List)).ServeHTTP(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		}), authMW))

	// 删除某条反馈 — /api/v1/assistant-feedback/{id}
	mux.Handle("/api/v1/assistant-feedback/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				methodNotAllowedJSON(w, "仅支持DELETE请求")
				return
			}
			feedbackHandler.Delete(w, r)
		}), authMW))

	// 单助手统计 — /api/v1/assistants/{id}/feedback-stats
	//
	// 注意:这条路径和现有 /api/v1/ai-assistants/ 是不同的前缀(assistants vs ai-assistants)
	// 这是有意为之,避免和现有的 /api/v1/ai-assistants/{id}/fork 等路由冲突
	// 我们选择在 /api/v1/assistants/ 新命名空间下放统计类只读接口
	mux.Handle("/api/v1/assistants/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 仅支持 GET /feedback-stats 这一条子路径
			if r.Method != http.MethodGet {
				methodNotAllowedJSON(w, "仅支持GET请求")
				return
			}
			if !hasSuffix(r.URL.Path, "/feedback-stats") {
				methodNotAllowedJSON(w, "未知的 /assistants 子路径")
				return
			}
			feedbackHandler.Stats(w, r)
		}), authMW))
}
