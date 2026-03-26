package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

// ==================== 权限常量 ====================
// 角色权限矩阵（与 models/user.go 保持同步）：
//   admin          : 全部功能
//   senior_operator: Pipeline创建/启动/审核/提交定稿/确认定稿/退回/分配/批量创建/批量启动/断点续跑/批量重跑
//   operator       : Pipeline创建/启动/审核/提交定稿/批量创建/批量启动/断点续跑（仅failed/cancelled）
//   viewer         : 只读（仅GET）

const roleAdmin = "admin"
const roleSeniorOperator = "senior_operator"
const roleOperator = "operator"

// hasRole 检查角色是否在允许列表中
func hasRole(role string, allowed ...string) bool {
	for _, r := range allowed {
		if role == r {
			return true
		}
	}
	return false
}

// forbiddenJSON 返回标准403响应
func forbiddenJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

// methodNotAllowedJSON 返回标准405响应
func methodNotAllowedJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

// Setup 注册所有路由并返回根Handler
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

	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))

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
	adminOnly := middleware.RequireRole(roleAdmin)

	// ==================== 公开路由 ====================
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// ==================== 认证路由 ====================
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.GetMe), authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout), authMW))

	// ==================== 仪表盘路由 ====================
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	// ==================== 引擎状态路由（仅admin）====================
	mux.Handle("/api/v1/engine/stats", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w, "仅支持GET请求")
			return
		}
		stats := engine.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"total_submitted":       stats.TotalSubmitted,
				"total_completed":       stats.TotalCompleted,
				"total_business_failed": stats.TotalBusinessFailed,
				"total_failed":          stats.TotalFailed,
				"current_running":       stats.CurrentRunning,
				"current_ai_active":     stats.CurrentAIActive,
				"queue_length":          stats.QueueLength,
				"max_workers":           5,
				"max_ai_concurrency":    4,
				"queue_capacity":        50,
			},
		})
	}), authMW, adminOnly))

	// ==================== 用户管理路由（仅admin）====================
	mux.Handle("/api/v1/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.List(w, r)
		case http.MethodPost:
			userHandler.Create(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
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
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		default:
			userHandler.Update(w, r)
		}
	}), authMW, adminOnly))

	// ==================== AI配置路由（仅admin）====================
	mux.Handle("/api/v1/ai-config/global", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			aiConfigHandler.GetGlobalConfig(w, r)
		case http.MethodPut:
			aiConfigHandler.UpdateGlobalConfig(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT请求")
		}
	}), authMW, adminOnly))

	mux.Handle("/api/v1/ai-config/test", middleware.Chain(http.HandlerFunc(aiConfigHandler.TestConnection), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/models", middleware.Chain(http.HandlerFunc(aiConfigHandler.ListModels), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes", middleware.Chain(http.HandlerFunc(aiConfigHandler.GetSceneConfigs), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes/", middleware.Chain(http.HandlerFunc(aiConfigHandler.UpdateSceneConfig), authMW, adminOnly))

	// ==================== 提示词路由（仅admin）====================
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
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}
	}), authMW, adminOnly))

	// ==================== 外部数据配置路由（仅admin）====================
	mux.Handle("/api/v1/external-data/configs", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			edHandler.GetConfigs(w, r)
		case http.MethodPut:
			edHandler.UpdateConfigs(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT请求")
		}
	}), authMW, adminOnly))

	// ==================== 课程管理路由 ====================
	mux.Handle("/api/v1/courses", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			courseHandler.ListCourses(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可注册课程")
				return
			}
			courseHandler.CreateCourse(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
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
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可拉取索引")
				return
			}
			courseHandler.FetchIndex(w, r)
		case hasSuffix(path, "/index-summary"):
			courseHandler.GetIndexSummary(w, r)
		case hasSuffix(path, "/index"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可查看完整索引")
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
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可创建Pipeline")
				return
			}
			pipelineHandler.CreatePipeline(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	// SSE实时推送路由（不走authMW，内部验证token）
	mux.HandleFunc("/api/v1/sse/pipelines/", sseHandler.StreamPipeline)

	// /api/v1/pipelines/ 子路径分发
	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// ===== 批量操作（必须在/{id}路由之前匹配）=====

		// POST /pipelines/batch-verify — 仅admin
		case hasSuffix(path, "/batch-verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可触发批量验收")
				return
			}
			pipelineHandler.BatchVerify(w, r)

		// POST /pipelines/batch-create — admin/senior_operator/operator
		case hasSuffix(path, "/batch-create"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量创建Pipeline")
				return
			}
			pipelineHandler.BatchCreate(w, r)

		// POST /pipelines/batch-assign — admin/senior_operator
		case hasSuffix(path, "/batch-assign"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可分配审核任务")
				return
			}
			pipelineHandler.BatchAssign(w, r)

		// v37新增：POST /pipelines/batch-restart — admin/senior_operator
		// 批量断点续跑（从指定步骤重跑多个Pipeline）
		// 注意：路由层仅允许 admin/senior_operator 调用批量重跑
		// 服务层会根据每个Pipeline的状态做进一步细粒度权限校验
		case hasSuffix(path, "/batch-restart"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可批量重跑Pipeline")
				return
			}
			pipelineHandler.BatchRestartFromStep(w, r)

		// GET /pipelines/operators — 全员（含认证）
		case hasSuffix(path, "/operators"):
			pipelineHandler.GetOperators(w, r)

		// POST /pipelines/batch-start — admin/senior_operator/operator
		case hasSuffix(path, "/batch-start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量启动Pipeline")
				return
			}
			pipelineHandler.BatchStart(w, r)

		// ===== 单个Pipeline操作 =====

		// POST /pipelines/{id}/start — admin/senior_operator/operator
		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可启动Pipeline")
				return
			}
			pipelineHandler.StartPipeline(w, r)

		// POST /pipelines/{id}/cancel — 仅admin
		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可取消Pipeline")
				return
			}
			pipelineHandler.CancelPipeline(w, r)

		// ===== 断点续跑路由 =====

		// POST /pipelines/{id}/restart-from — admin/senior_operator/operator
		// 单个Pipeline断点续跑（路由层允许三个角色，服务层做状态+角色细粒度校验）
		case hasSuffix(path, "/restart-from"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可重启Pipeline步骤")
				return
			}
			pipelineHandler.RestartFromStep(w, r)

		// ===== P7：二级审批路由 =====

		// POST /pipelines/{id}/submit-finalize — admin/senior_operator/operator
		case hasSuffix(path, "/submit-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可提交定稿申请")
				return
			}
			pipelineHandler.SubmitFinalize(w, r)

		// POST /pipelines/{id}/confirm-finalize — admin/senior_operator
		case hasSuffix(path, "/confirm-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可确认定稿")
				return
			}
			pipelineHandler.ConfirmFinalize(w, r)

		// POST /pipelines/{id}/reject-finalize — admin/senior_operator
		case hasSuffix(path, "/reject-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可退回重审")
				return
			}
			pipelineHandler.RejectFinalize(w, r)

		// POST /pipelines/{id}/finalize — 仅admin（直接定稿）
		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "直接定稿仅管理员可操作，操作员请使用提交定稿")
				return
			}
			pipelineHandler.FinalizePipeline(w, r)

		// POST /pipelines/{id}/mark-passed — admin/senior_operator/operator
		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可快捷通过Pipeline")
				return
			}
			pipelineHandler.MarkPassed(w, r)

		// POST /pipelines/{id}/verify — admin/senior_operator/operator
		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可触发验收")
				return
			}
			pipelineHandler.VerifyPipeline(w, r)

		// GET /pipelines/{id}/eval-rounds — 全员（含认证）
		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)

		// POST /pipelines/{id}/pages/{num}/ai-fix — admin/senior_operator/operator
		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可使用AI快修")
				return
			}
			pipelineHandler.AIFixPage(w, r)

		// PUT /pipelines/{id}/pages/{num}/decision — admin/senior_operator/operator
		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可审核页面")
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)

		// GET /pipelines/{id}/pages — 全员（含认证）
		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)

		// GET /pipelines/{id}/steps/{name} — 全员（含认证）
		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)

		// GET /pipelines/{id}/steps — 全员（含认证）
		case hasSuffix(path, "/steps"):
			pipelineHandler.GetSteps(w, r)

		// GET/DELETE /pipelines/{id}
		default:
			switch r.Method {
			case http.MethodGet:
				pipelineHandler.GetPipelineDetail(w, r)
			case http.MethodDelete:
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || !hasRole(claims.Role, roleAdmin) {
					forbiddenJSON(w, "仅管理员可删除Pipeline")
					return
				}
				pipelineHandler.DeletePipeline(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/DELETE请求")
			}
		}
	}), authMW))

	// 启动优雅关闭监听
	engine.StartGracefulShutdown()

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

// corsMiddleware CORS中间件（仅允许生产域名）
func corsMiddleware(next http.Handler) http.Handler {
	const allowedOrigin = "https://workflow.pkuailab.com"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// makeHealthHandler 健康检查接口（增强版）
func makeHealthHandler(engine *services.Engine) http.HandlerFunc {
	startTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		overallStatus := "ok"

		dbStatus := "ok"
		dbErr := database.Ping(ctx)
		if dbErr != nil {
			dbStatus = "error: " + dbErr.Error()
			overallStatus = "degraded"
		}

		stats := engine.GetStats()
		engineStatus := "ok"
		queueUsagePct := 0
		if stats.QueueLength > 0 {
			queueUsagePct = stats.QueueLength * 100 / 50
		}
		if queueUsagePct >= 80 {
			engineStatus = "warning: queue usage " + itoa(queueUsagePct) + "%"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  overallStatus,
			"version": config.AppVersion,
			"time":    time.Now().Format(time.RFC3339),
			"uptime":  time.Since(startTime).Round(time.Second).String(),
			"database": map[string]interface{}{
				"status": dbStatus,
			},
			"engine": map[string]interface{}{
				"status":                engineStatus,
				"total_submitted":       stats.TotalSubmitted,
				"total_completed":       stats.TotalCompleted,
				"total_business_failed": stats.TotalBusinessFailed,
				"total_failed":          stats.TotalFailed,
				"current_running":       stats.CurrentRunning,
				"current_ai_active":     stats.CurrentAIActive,
				"queue_length":          stats.QueueLength,
				"queue_capacity":        50,
				"max_workers":           5,
				"max_ai_concurrency":    4,
			},
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
