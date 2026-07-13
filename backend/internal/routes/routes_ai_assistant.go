package routes

// routes_ai_assistant.go — AI 助手路由注册
//
// 路由清单：
//   GET    /api/v1/ai-assistants
//   POST   /api/v1/ai-assistants
//   GET    /api/v1/ai-assistants/my-groups
//   GET    /api/v1/ai-assistants/{id}
//   PUT    /api/v1/ai-assistants/{id}
//   DELETE /api/v1/ai-assistants/{id}
//   POST   /api/v1/ai-assistants/{id}/fork
//   POST   /api/v1/ai-assistants/design/chat
//   POST   /api/v1/ai-assistants/design/profile-materials
//
// 所有路由均需登录，助手归属、查看原文、编辑、复制和平台教案读取权限
// 分别在对应 Service 层执行最终校验。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerAIAssistantRoutes 注册 AI 助手相关路由。
func registerAIAssistantRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	aiAssistantHandler *handlers.AIAssistantHandler,
	designerHandler *handlers.AssistantDesignerHandler,
) {
	// 列表 + 创建。
	mux.Handle(
		"/api/v1/ai-assistants",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				aiAssistantHandler.List(w, r)
			case http.MethodPost:
				aiAssistantHandler.Create(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		}), authMW),
	)

	// 单条操作与固定子路径。
	//
	// 路径优先级：
	//   1. /design/profile-materials
	//   2. /design/chat
	//   3. /my-groups
	//   4. /{id}/fork
	//   5. /{id}
	mux.Handle(
		"/api/v1/ai-assistants/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if path == "/api/v1/ai-assistants/design/profile-materials" {
				if designerHandler == nil {
					methodNotAllowedJSON(w, "Designer 服务未启用")
					return
				}
				if r.Method != http.MethodPost {
					methodNotAllowedJSON(w, "仅支持POST请求")
					return
				}
				designerHandler.ProfileMaterials(w, r)
				return
			}

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

			if path == "/api/v1/ai-assistants/my-groups" {
				aiAssistantHandler.MyPublishGroups(w, r)
				return
			}

			if strings.HasSuffix(path, "/fork") {
				if r.Method != http.MethodPost {
					methodNotAllowedJSON(w, "仅支持POST请求")
					return
				}
				aiAssistantHandler.Fork(w, r)
				return
			}

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
		}), authMW),
	)
}
