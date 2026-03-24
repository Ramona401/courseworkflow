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
// 版本：0.30.0（P7新增：二级审批流程 submit-finalize/confirm-finalize/reject-finalize）
func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService := services.NewPromptService()
	edService := services.NewExternalDataService(cfg)
	courseService := services.NewCourseService(cfg)
	pipelineService := services.NewPipelineService(cfg)

	engine := services.NewEngine(5, 4, 50)
	pipelineService.SetEngine(engine)

	pipelineService.StartNightlyVerifyScheduler()

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)
	edHandler := handlers.NewExternalDataHandler(edService)
	courseHandler := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	sseHandler := handlers.NewSSEHandler(authService)

	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole("admin")

	// ==================== 公开路由 ====================
	mux.HandleFunc("/api/v1/health", healthHandler)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// ==================== 认证路由 ====================
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.GetMe), authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout), authMW))

	// ==================== 仪表盘路由 ====================
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	// ==================== 引擎状态路由 ====================
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
				"total_submitted":    stats.TotalSubmitted,
				"total_completed":    stats.TotalCompleted,
				"total_failed":       stats.TotalFailed,
				"current_running":    stats.CurrentRunning,
				"current_ai_active":  stats.CurrentAIActive,
				"queue_length":       stats.QueueLength,
				"max_workers":        5,
				"max_ai_concurrency": 4,
				"queue_capacity":     50,
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

	// ==================== Pipeline路由 ====================
	mux.Handle("/api/v1/pipelines", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pipelineHandler.ListPipelines(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
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

	// SSE实时推送路由（不走authMW，内部验证token）
	mux.HandleFunc("/api/v1/sse/pipelines/", sseHandler.StreamPipeline)

	// /api/v1/pipelines/ 子路径分发
	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// ===== 批量操作（必须在/{id}路由之前匹配） =====

		case hasSuffix(path, "/batch-verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可触发批量验收"})
				return
			}
			pipelineHandler.BatchVerify(w, r)

		case hasSuffix(path, "/batch-create"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可批量创建Pipeline"})
				return
			}
			pipelineHandler.BatchCreate(w, r)

		case hasSuffix(path, "/batch-assign"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和高级操作员可分配审核任务"})
				return
			}
			pipelineHandler.BatchAssign(w, r)

		case hasSuffix(path, "/operators"):
			pipelineHandler.GetOperators(w, r)

		case hasSuffix(path, "/batch-start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可批量启动Pipeline"})
				return
			}
			pipelineHandler.BatchStart(w, r)

		// ===== 单个Pipeline操作 =====

		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可启动Pipeline"})
				return
			}
			pipelineHandler.StartPipeline(w, r)

		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员可取消Pipeline"})
				return
			}
			pipelineHandler.CancelPipeline(w, r)

		// ===== P7新增：二级审批路由 =====

		// POST /pipelines/{id}/submit-finalize — 审核员提交定稿申请
		// 权限：operator / senior_operator / admin（所有可操作角色）
		case hasSuffix(path, "/submit-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可提交定稿申请"})
				return
			}
			pipelineHandler.SubmitFinalize(w, r)

		// POST /pipelines/{id}/confirm-finalize — 超级审核员确认定稿
		// 权限：senior_operator / admin
		case hasSuffix(path, "/confirm-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅高级操作员和管理员可确认定稿"})
				return
			}
			pipelineHandler.ConfirmFinalize(w, r)

		// POST /pipelines/{id}/reject-finalize — 超级审核员退回重审
		// 权限：senior_operator / admin
		case hasSuffix(path, "/reject-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅高级操作员和管理员可退回重审"})
				return
			}
			pipelineHandler.RejectFinalize(w, r)

		// POST /pipelines/{id}/finalize — 直接定稿（admin跳过二级审批）
		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || claims.Role != "admin" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "直接定稿仅管理员可操作，操作员请使用提交定稿"})
				return
			}
			pipelineHandler.FinalizePipeline(w, r)

		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可快捷通过Pipeline"})
				return
			}
			pipelineHandler.MarkPassed(w, r)

		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可触发验收"})
				return
			}
			pipelineHandler.VerifyPipeline(w, r)

		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)

		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可使用AI快修"})
				return
			}
			pipelineHandler.AIFixPage(w, r)

		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || (claims.Role != "admin" && claims.Role != "operator" && claims.Role != "senior_operator") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "仅管理员和操作员可审核页面"})
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)

		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)

		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)

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

func hasSuffix(path string, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

func containsStepsWithName(path string) bool {
	idx := indexOf(path, "/steps/")
	if idx < 0 {
		return false
	}
	remaining := path[idx+len("/steps/"):]
	return len(remaining) > 0 && remaining != "/"
}

func containsPagesDecision(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/decision")
}

func containsPagesAIFix(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/ai-fix")
}

func indexOf(s string, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.30.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}
