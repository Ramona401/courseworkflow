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
// 版本：0.22.0（P4.6-2新增verify验收路由+修复ai-fix路由）
func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// ==================== 初始化服务层 ====================
	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService := services.NewPromptService()
	edService := services.NewExternalDataService(cfg)
	courseService := services.NewCourseService(cfg)
	pipelineService := services.NewPipelineService(cfg) // P4-1新增

	// ==================== 初始化处理器层 ====================
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)
	edHandler := handlers.NewExternalDataHandler(edService)
	courseHandler := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService) // P4-1新增

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
	// GET /api/v1/dashboard/stats — 获取仪表盘统计数据（登录即可）
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

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
	// GET /api/v1/ai-config/models — 查询当前Key下可用模型列表
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

	// /api/v1/pipelines/ 子路径分发
	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// POST /pipelines/{id}/start — 启动Pipeline
		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可启动Pipeline"})
				return
			}
			pipelineHandler.StartPipeline(w, r)

		// POST /pipelines/{id}/cancel — 取消Pipeline
		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可取消Pipeline"})
				return
			}
			pipelineHandler.CancelPipeline(w, r)

		// POST /pipelines/{id}/finalize — 定稿归档（P4.5-C新增）
		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可定稿Pipeline"})
				return
			}
			pipelineHandler.FinalizePipeline(w, r)

		// POST /pipelines/{id}/mark-passed — 快捷通过（P4.5-D新增）
		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可快捷通过Pipeline"})
				return
			}
			pipelineHandler.MarkPassed(w, r)

		// POST /pipelines/{id}/verify — 手动触发验收（P4.6-2新增）
		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可触发验收"})
				return
			}
			pipelineHandler.VerifyPipeline(w, r)

		// GET /pipelines/{id}/eval-rounds — 评估轮次详情（P4.5-B）
		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)

		// POST /pipelines/{id}/pages/{n}/ai-fix — AI快修（P4.5-E-2新增）
		// 注意：ai-fix路由必须在/pages路由之前匹配，因为/pages是后缀匹配
		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可使用AI快修"})
				return
			}
			pipelineHandler.AIFixPage(w, r)

		// PUT /pipelines/{id}/pages/{n}/decision — 更新页面决策（P4.5-C新增）
		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可审核页面"})
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)

		// GET /pipelines/{id}/pages — 生成页面列表（P4.5-C新增）
		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)

		// GET /pipelines/{id}/steps/{name} — 步骤详情（必须在 /steps 之前匹配）
		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)

		// GET /pipelines/{id}/steps — 步骤列表
		case hasSuffix(path, "/steps"):
			pipelineHandler.GetSteps(w, r)

		// GET/DELETE /pipelines/{id} — 详情或删除
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
// P4.5-C新增
func containsPagesDecision(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/decision")
}

// containsPagesAIFix 检查路径是否匹配 /pages/{n}/ai-fix 模式
// P4.5-E-2新增
func containsPagesAIFix(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/ai-fix")
}

// indexOf 查找子串位置（简化版strings.Index）
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
		"version": "0.23.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}
