package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

// Setup 注册所有路由，返回带 CORS 处理的 http.Handler
func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// 创建服务层实例（全局共享）
	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService := services.NewPromptService()

	// 创建处理器实例
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)

	// 创建中间件实例
	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole("admin")

	// ========== 公共接口（无需认证） ==========

	// 健康检查
	mux.HandleFunc("/api/v1/health", healthHandler)

	// ========== 认证接口 ==========

	// 登录（公开，无需 token）
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// 获取当前用户（需要认证）
	mux.Handle("/api/v1/auth/me",
		middleware.Chain(
			http.HandlerFunc(authHandler.GetMe),
			authMW,
		),
	)

	// 登出（需要认证）
	mux.Handle("/api/v1/auth/logout",
		middleware.Chain(
			http.HandlerFunc(authHandler.Logout),
			authMW,
		),
	)

	// ========== 用户管理接口（仅admin）P1-4 ==========

	// GET /api/v1/users — 用户列表
	mux.Handle("/api/v1/users",
		middleware.Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					userHandler.List(w, r)
				case http.MethodPost:
					userHandler.Create(w, r)
				default:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusMethodNotAllowed)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": -1, "message": "仅支持GET/POST请求",
					})
				}
			}),
			authMW,
			adminOnly,
		),
	)

	// PUT /api/v1/users/{id} — 编辑用户（通过路径前缀匹配）
	mux.Handle("/api/v1/users/",
		middleware.Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				// 路由分发：根据子路径后缀决定处理器
				switch {
				// PUT /api/v1/users/{id}/password — 重置密码
				case len(path) > len("/api/v1/users/") && hasSuffix(path, "/password"):
					userHandler.ResetPassword(w, r)

				// PUT /api/v1/users/{id}/status — 启用/禁用
				case len(path) > len("/api/v1/users/") && hasSuffix(path, "/status"):
					userHandler.UpdateStatus(w, r)

				// GET/PUT /api/v1/users/{id}/assignments — 课程分配
				case len(path) > len("/api/v1/users/") && hasSuffix(path, "/assignments"):
					switch r.Method {
					case http.MethodGet:
						userHandler.GetAssignments(w, r)
					case http.MethodPut:
						userHandler.UpdateAssignments(w, r)
					default:
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusMethodNotAllowed)
						json.NewEncoder(w).Encode(map[string]interface{}{
							"code": -1, "message": "仅支持GET/PUT请求",
						})
					}

				// PUT /api/v1/users/{id} — 编辑用户基本信息
				default:
					userHandler.Update(w, r)
				}
			}),
			authMW,
			adminOnly,
		),
	)

	// ========== AI配置接口（仅admin）P2-1 + P2-2 ==========

	// GET /api/v1/ai-config/global — 获取全局配置
	// PUT /api/v1/ai-config/global — 更新全局配置
	mux.Handle("/api/v1/ai-config/global",
		middleware.Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					aiConfigHandler.GetGlobalConfig(w, r)
				case http.MethodPut:
					aiConfigHandler.UpdateGlobalConfig(w, r)
				default:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusMethodNotAllowed)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": -1, "message": "仅支持GET/PUT请求",
					})
				}
			}),
			authMW,
			adminOnly,
		),
	)

	// POST /api/v1/ai-config/test — AI连通性测试（P2-2新增）
	mux.Handle("/api/v1/ai-config/test",
		middleware.Chain(
			http.HandlerFunc(aiConfigHandler.TestConnection),
			authMW,
			adminOnly,
		),
	)

	// GET /api/v1/ai-config/scenes — 获取所有场景配置
	mux.Handle("/api/v1/ai-config/scenes",
		middleware.Chain(
			http.HandlerFunc(aiConfigHandler.GetSceneConfigs),
			authMW,
			adminOnly,
		),
	)

	// PUT /api/v1/ai-config/scenes/{code} — 更新指定场景配置
	mux.Handle("/api/v1/ai-config/scenes/",
		middleware.Chain(
			http.HandlerFunc(aiConfigHandler.UpdateSceneConfig),
			authMW,
			adminOnly,
		),
	)

	// ========== 提示词管理接口（仅admin）P2-3 ==========

	// GET /api/v1/prompts — 获取所有提示词（当前生效版本）
	mux.Handle("/api/v1/prompts",
		middleware.Chain(
			http.HandlerFunc(promptHandler.ListPrompts),
			authMW,
			adminOnly,
		),
	)

	// /api/v1/prompts/ — 子路径分发
	// GET  /api/v1/prompts/{key} — 获取指定提示词
	// PUT  /api/v1/prompts/{key} — 更新提示词（创建新版本）
	// GET  /api/v1/prompts/{key}/versions — 版本历史
	// POST /api/v1/prompts/{key}/rollback — 回滚到指定版本
	mux.Handle("/api/v1/prompts/",
		middleware.Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				// 路由分发：根据子路径后缀决定处理器
				switch {
				// GET /api/v1/prompts/{key}/versions — 版本历史
				case hasSuffix(path, "/versions"):
					promptHandler.GetVersionHistory(w, r)

				// POST /api/v1/prompts/{key}/rollback — 回滚
				case hasSuffix(path, "/rollback"):
					promptHandler.RollbackVersion(w, r)

				// GET /api/v1/prompts/{key} — 获取指定提示词
				// PUT /api/v1/prompts/{key} — 更新提示词
				default:
					switch r.Method {
					case http.MethodGet:
						promptHandler.GetPrompt(w, r)
					case http.MethodPut:
						promptHandler.UpdatePrompt(w, r)
					default:
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusMethodNotAllowed)
						json.NewEncoder(w).Encode(map[string]interface{}{
							"code": -1, "message": "仅支持GET/PUT请求",
						})
					}
				}
			}),
			authMW,
			adminOnly,
		),
	)

	// ========== TODO: 后续 Phase 路由 ==========
	// P3: 课程管理路由（admin + operator）
	// P4: Pipeline 路由（admin + operator）
	// P5: 引擎控制路由
	// P6: 审核中心路由

	// 包裹 CORS 中间件后返回
	return corsMiddleware(mux)
}

// hasSuffix 检查路径是否以指定后缀结尾
func hasSuffix(path string, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

// corsMiddleware 处理跨域请求（开发阶段允许所有来源）
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 响应头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 预检请求直接返回
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// healthHandler 健康检查接口
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.7.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}
