package routes

// routes_ai_assistant.go — AI 助手路由注册(TE-DNA 3.0 P0 + P0.5)
//
// 路由清单:
//   GET    /api/v1/ai-assistants                   列表(按场景/学科/年级筛选)
//   POST   /api/v1/ai-assistants                   创建
//   GET    /api/v1/ai-assistants/{id}              详情
//   PUT    /api/v1/ai-assistants/{id}              编辑
//   DELETE /api/v1/ai-assistants/{id}              删除
//   POST   /api/v1/ai-assistants/{id}/fork         复制到"我的"
//   POST   /api/v1/ai-assistants/design/chat       P0.5 新增:对话式创作(SSE 流式)
//
// 权限策略:所有路由需登录,业务权限在 Service 层根据 actor.Role + source 细粒度控制

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerAIAssistantRoutes 注册 AI 助手相关路由
//
// v113(P0.5)改造:新增 designerHandler 参数,拦截 /design/chat 路径走对话式创作
// 其它原有路由保持不变
func registerAIAssistantRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	aiAssistantHandler *handlers.AIAssistantHandler,
	designerHandler *handlers.AssistantDesignerHandler,
) {
	// 列表 + 创建(/api/v1/ai-assistants 精确匹配,不含尾斜杠)
	mux.Handle("/api/v1/ai-assistants",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				aiAssistantHandler.List(w, r)
			case http.MethodPost:
				aiAssistantHandler.Create(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		}), authMW))

	// 单条操作 + 其他子路径(/api/v1/ai-assistants/ 通配所有子路径)
	//
	// 路径分发优先级(从特殊到通用):
	//   1. /design/chat      → P0.5 对话式创作(SSE)
	//   2. /{id}/fork        → 复制到"我的"
	//   3. /{id}             → 详情/编辑/删除
	mux.Handle("/api/v1/ai-assistants/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// v113 P0.5 新增:对话式创作路由(优先匹配,避免被当成 /{id}="design" 处理)
			if path == "/api/v1/ai-assistants/design/chat" {
				if designerHandler == nil {
					methodNotAllowedJSON(w, "Designer 服务未启用")
					return
				}
				if r.Method != http.MethodPost {
					methodNotAllowedJSON(w, "仅支持POST请求")
					return
				}
				designerHandler.Chat(w, r)
				return
			}

			// /{id}/fork
			if strings.HasSuffix(path, "/fork") {
				if r.Method != http.MethodPost {
					methodNotAllowedJSON(w, "仅支持POST请求")
					return
				}
				aiAssistantHandler.Fork(w, r)
				return
			}

			// /{id}
			switch r.Method {
			case http.MethodGet:
				aiAssistantHandler.Get(w, r)
			case http.MethodPut:
				aiAssistantHandler.Update(w, r)
			case http.MethodDelete:
				aiAssistantHandler.Delete(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}), authMW))
}
