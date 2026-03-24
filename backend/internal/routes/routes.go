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

// Setup 注册所有路由并返回根Handler
// 版本：0.28.0（P5-4新增SSE实时进度推送+批量创建+批量启动+并发引擎+AI限流）
func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// ==================== 初始化服务层 ====================
	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService := services.NewPromptService()
	edService := services.NewExternalDataService(cfg)
	courseService := services.NewCourseService(cfg)
	pipelineService := services.NewPipelineService(cfg)

	// ==================== P5-2新增：创建并发引擎并注入PipelineService ====================
	// 参数：maxWorkers=3（同时执行3个Pipeline），maxAIConcurrency=2（同时2个AI调用），queueSize=50
	engine := services.NewEngine(5, 4, 50)
	pipelineService.SetEngine(engine)

	// ==================== P4.6-4新增：启动夜间批量验收定时任务 ====================
	pipelineService.StartNightlyVerifyScheduler()

	// ==================== 初始化处理器层 ====================
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)
	edHandler := handlers.NewExternalDataHandler(edService)
	courseHandler := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	sseHandler := handlers.NewSSEHandler(authService) // P5-4修复：传入authService用于token验证

	// ==================== 中间件 ====================
	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole("admin")

	// ==================== 公开路由（无需认证） ====================
	mux.HandleFunc("/api/v1/health", healthHandler)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// ==================== 认证路由 ====================
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.GetMe), authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout), authMW))

	// ==================== 仪表盘路由（P4.5-D新增） ====================
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	// ==================== P5-2新增：引擎状态查询路由 ====================
	// GET /api/v1/engine/stats — 获取并发引擎运行统计（仅admin）
	mux.Handle("/api/v1/engine/stats", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET请求"})
			return
		}
		stats := engine.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"total_submitted":   stats.TotalSubmitted,
				"total_completed":   stats.TotalCompleted,
				"total_failed":      stats.TotalFailed,
				"current_running":   stats.CurrentRunning,
				"current_ai_active": stats.CurrentAIActive,
				"queue_length":      stats.QueueLength,
				"max_workers":       5,
				"max_ai_concurrency": 4,
				"queue_capacity":    50,
			},
		})
	}), authMW, adminOnly))

	// ==================== 用户管理路由（仅admin） ====================
	mux.Handle("/api/v1/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.List(w, r)
		case http.MethodPost:
			userHandler.Create(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/POST请求"})
		}
	}), authMW, adminOnly))

	mux.Handle("/api/v1/users/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case len(path) > len("/api/v1/users/") && hasSuffix(path, "/password"):
			userHandler.ResetPassword(w, r)
		case len(path) > len("/api/v1/users/") && hasSuffix(path, "/status"):
			userHandler.UpdateStatus(w, r)
		case len(path) > len("/api/v1/users/") && hasSuffix(path, "/assignments"):
			switch r.Method {
			case http.MethodGet:
				userHandler.GetAssignments(w, r)
			case http.MethodPut:
				userHandler.UpdateAssignments(w, r)
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/PUT请求"})
			}
		default:
			userHandler.Update(w, r)
		}
	}), authMW, adminOnly))

	// ==================== AI配置路由（仅admin） ====================
	mux.Handle("/api/v1/ai-config/global", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			aiConfigHandler.GetGlobalConfig(w, r)
		case http.MethodPut:
			aiConfigHandler.UpdateGlobalConfig(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/PUT请求"})
		}
	}), authMW, adminOnly))

	mux.Handle("/api/v1/ai-config/test", middleware.Chain(http.HandlerFunc(aiConfigHandler.TestConnection), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/models", middleware.Chain(http.HandlerFunc(aiConfigHandler.ListModels), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes", middleware.Chain(http.HandlerFunc(aiConfigHandler.GetSceneConfigs), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes/", middleware.Chain(http.HandlerFunc(aiConfigHandler.UpdateSceneConfig), authMW, adminOnly))

	// ==================== 提示词路由（仅admin） ====================
	mux.Handle("/api/v1/prompts", middleware.Chain(http.HandlerFunc(promptHandler.ListPrompts), authMW, adminOnly))
	mux.Handle("/api/v1/prompts/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/versions"):
			promptHandler.GetVersionHistory(w, r)
		case hasSuffix(path, "/rollback"):
			promptHandler.RollbackVersion(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				promptHandler.GetPrompt(w, r)
			case http.MethodPut:
				promptHandler.UpdatePrompt(w, r)
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/PUT请求"})
			}
		}
	}), authMW, adminOnly))

	// ==================== 外部数据配置路由（仅admin） ====================
	mux.Handle("/api/v1/external-data/configs", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			edHandler.GetConfigs(w, r)
		case http.MethodPut:
			edHandler.UpdateConfigs(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/PUT请求"})
		}
	}), authMW, adminOnly))

	// ==================== 课程管理路由 ====================
	mux.Handle("/api/v1/courses", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			courseHandler.ListCourses(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可注册课程"})
				return
			}
			courseHandler.CreateCourse(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/POST请求"})
		}
	}), authMW))

	mux.Handle("/api/v1/courses/oss-catalog", middleware.Chain(http.HandlerFunc(courseHandler.GetOSSCatalog), authMW, adminOnly))
	mux.Handle("/api/v1/courses/register-fetch", middleware.Chain(http.HandlerFunc(courseHandler.RegisterAndFetch), authMW, adminOnly))
	mux.Handle("/api/v1/courses/batch-register", middleware.Chain(http.HandlerFunc(courseHandler.BatchRegisterAndFetch), authMW, adminOnly))
	mux.Handle("/api/v1/courses/batch-fetch", middleware.Chain(http.HandlerFunc(courseHandler.BatchFetchIndexes), authMW, adminOnly))

	mux.Handle("/api/v1/courses/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/fetch-index"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可拉取索引"})
				return
			}
			courseHandler.FetchIndex(w, r)
		case hasSuffix(path, "/index-summary"):
			courseHandler.GetIndexSummary(w, r)
		case hasSuffix(path, "/index"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可查看完整索引"})
				return
			}
			courseHandler.GetIndexFull(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "未知的课程子路径"})
		}
	}), authMW))

	// ==================== Pipeline路由（P4-1新增） ====================
	mux.Handle("/api/v1/pipelines", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pipelineHandler.ListPipelines(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可创建Pipeline"})
				return
			}
			pipelineHandler.CreatePipeline(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/POST请求"})
		}
	}), authMW))

	// ==================== P5-4新增：SSE实时推送路由（不走authMW，内部验证token） ====================
	// GET /api/v1/pipelines/{id}/stream?token=xxx — EventSource不支持自定义header，token通过query传递
	mux.HandleFunc("/api/v1/sse/pipelines/", sseHandler.StreamPipeline)

	// /api/v1/pipelines/ 子路径分发
	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// POST /pipelines/batch-verify — 批量验收（必须在/{id}路由之前匹配）
		case hasSuffix(path, "/batch-verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可触发批量验收"})
				return
			}
			pipelineHandler.BatchVerify(w, r)

			// POST /pipelines/batch-create — 批量创建Pipeline（P5-3新增）
			case hasSuffix(path, "/batch-create"):
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || (claims.Role != "admin" && claims.Role != "operator") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可批量创建Pipeline"})
					return
				}
				pipelineHandler.BatchCreate(w, r)

			// POST /pipelines/batch-start — 批量启动Pipeline（P5-3新增）
			case hasSuffix(path, "/batch-start"):
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || (claims.Role != "admin" && claims.Role != "operator") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可批量启动Pipeline"})
					return
				}
				pipelineHandler.BatchStart(w, r)

		// POST /pipelines/{id}/start
		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可启动Pipeline"})
				return
			}
			pipelineHandler.StartPipeline(w, r)

		// POST /pipelines/{id}/cancel
		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可取消Pipeline"})
				return
			}
			pipelineHandler.CancelPipeline(w, r)

		// POST /pipelines/{id}/finalize
		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可定稿Pipeline"})
				return
			}
			pipelineHandler.FinalizePipeline(w, r)

		// POST /pipelines/{id}/mark-passed
		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可快捷通过Pipeline"})
				return
			}
			pipelineHandler.MarkPassed(w, r)

		// POST /pipelines/{id}/verify
		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可触发验收"})
				return
			}
			pipelineHandler.VerifyPipeline(w, r)

		// GET /pipelines/{id}/eval-rounds
		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)

		// POST /pipelines/{id}/pages/{n}/ai-fix
		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可使用AI快修"})
				return
			}
			pipelineHandler.AIFixPage(w, r)

		// PUT /pipelines/{id}/pages/{n}/decision
		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可审核页面"})
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)

		// GET /pipelines/{id}/pages
		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)

		// GET /pipelines/{id}/steps/{name}
		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)

		// GET /pipelines/{id}/steps
		case hasSuffix(path, "/steps"):
			pipelineHandler.GetSteps(w, r)

		// GET/DELETE /pipelines/{id}
		default:
			switch r.Method {
			case http.MethodGet:
				pipelineHandler.GetPipelineDetail(w, r)
			case http.MethodDelete:
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || claims.Role != "admin" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可删除Pipeline"})
					return
				}
				pipelineHandler.DeletePipeline(w, r)
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅支持GET/DELETE请求"})
			}
		}
	}), authMW))

	return corsMiddleware(mux)
}

// ==================== 辅助函数 ====================

// hasSuffix 检查路径是否以指定后缀结尾
func hasSuffix(path string, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

// containsStepsWithName 检查路径是否匹配 /steps/{name} 模式
func containsStepsWithName(path string) bool {
	idx := indexOf(path, "/steps/")
	if idx < 0 {
		return false
	}
	remaining := path[idx+len("/steps/"):]
	return len(remaining) > 0 && remaining != "/"
}

// containsPagesDecision 检查路径是否匹配 /pages/{n}/decision 模式
func containsPagesDecision(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/decision")
}

// containsPagesAIFix 检查路径是否匹配 /pages/{n}/ai-fix 模式
func containsPagesAIFix(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/ai-fix")
}

// indexOf 查找子串位置
func indexOf(s string, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// corsMiddleware CORS跨域中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// healthHandler 健康检查接口
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.29.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}
