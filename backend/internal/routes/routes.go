package routes

// routes.go — 路由注册主入口
//
// v73新增：wsStageService.SetGenService(lpGenService) 注入依赖
//   WorkshopStageService.AdvanceStage 进入review/revise阶段时自动触发Chat，
//   需要持有genService引用，通过SetGenService在routes层注入，避免循环依赖。
//
// v80新增：InitTraceWriter() 启动AI调用追踪异步写入器
//   AITraceHandler 注册到 registerAdminRoutes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
)

// ==================== 权限常量 ====================

const roleAdmin          = "admin"
const roleSeniorOperator = "senior_operator"
const roleOperator       = "operator"

func hasRole(role string, allowed ...string) bool {
	for _, r := range allowed {
		if role == r {
			return true
		}
	}
	return false
}

func forbiddenJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

func methodNotAllowedJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

// ==================== 主入口 ====================

func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// ---- v80新增：启动AI调用追踪异步写入器 ----
	repository.InitTraceWriter()

	// ---- 初始化服务层 ----
	authService     := services.NewAuthService(cfg)
	userService     := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService   := services.NewPromptService()
	edService       := services.NewExternalDataService(cfg)
	courseService   := services.NewCourseService(cfg)
	pipelineService := services.NewPipelineService(cfg)
	orgService      := services.NewOrganizationService()
	compService     := services.NewComponentService(cfg)
	lpService       := services.NewLessonPlanService(compService)
	lpGenService    := services.NewLessonPlanGenService(cfg)
	roleService     := services.NewRoleService()
	recipeService   := services.NewRecipeService()
	wsStageService  := services.NewWorkshopStageService()
	assessService   := services.NewAssessmentService(recipeService, cfg)
	tbService       := services.NewTextbookService(cfg)

	// v73：注入genService到wsStageService，使review/revise阶段可自动触发Chat
	// 必须在两个service都初始化完成后才能注入，避免循环依赖
	wsStageService.SetGenService(lpGenService)

	engine := services.NewEngine(8, 8, 100)
	pipelineService.SetEngine(engine)

	// ---- 初始化处理器层 ----
	authHandler     := handlers.NewAuthHandler(authService)
	userHandler     := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler   := handlers.NewPromptHandler(promptService)
	edHandler       := handlers.NewExternalDataHandler(edService)
	courseHandler   := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	sseHandler      := handlers.NewSSEHandler(authService)
	accountHandler  := handlers.NewAccountHandler()
	adminHandler    := handlers.NewAdminHandler(userService, orgService)
	roleHandler     := handlers.NewRoleHandler(roleService)
	orgHandler      := handlers.NewOrganizationHandler(orgService)
	compHandler     := handlers.NewComponentHandler(compService)
	lpHandler       := handlers.NewLessonPlanHandler(lpService)
	lpGenHandler    := handlers.NewLessonPlanGenHandler(lpGenService, authService)
	recipeHandler   := handlers.NewRecipeHandler(recipeService, compService)
	wsStageHandler  := handlers.NewWorkshopStageHandler(wsStageService)
	assessHandler   := handlers.NewAssessmentHandler(assessService)
	tbHandler       := handlers.NewTextbookHandler(tbService)

	// v80新增：AI调用追踪处理器
	aiTraceHandler  := handlers.NewAITraceHandler()

	authMW    := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole(roleAdmin)

	// ---- 健康检查（公开）----
	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))
	pipelineService.StartNightlyVerifyScheduler()

	// ---- 引擎状态（admin only）----
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
				"max_workers":           8,
				"max_ai_concurrency":    8,
				"queue_capacity":        100,
			},
		})
	}), authMW, adminOnly))

	// ---- 认证路由（公开）----
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// ---- 认证路由（需登录）----
	mux.Handle("/api/v1/auth/me",     middleware.Chain(http.HandlerFunc(authHandler.GetMe),    authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout),   authMW))

	// ---- 通用用户中心 ----
	mux.Handle("/api/v1/account/profile", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			accountHandler.GetProfile(w, r)
		case http.MethodPut:
			accountHandler.UpdateProfile(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT请求")
		}
	}), authMW))
	mux.Handle("/api/v1/account/password", middleware.Chain(
		http.HandlerFunc(accountHandler.ChangePassword), authMW))

	// ---- 仪表盘 ----
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(
		http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	// ---- 注册各模块路由（v80: aiTraceHandler传入registerAdminRoutes）----
	registerAdminRoutes(mux, authMW, adminOnly, adminHandler, roleHandler, userHandler, aiConfigHandler, promptHandler, edHandler, courseHandler, wsStageHandler, aiTraceHandler)
	registerPipelineRoutes(mux, authMW, pipelineHandler, sseHandler)
	registerLessonPlanRoutes(mux, authMW, orgHandler, compHandler, lpHandler, lpGenHandler, recipeHandler, wsStageHandler, assessHandler, tbHandler)

	engine.StartGracefulShutdown()
	return corsMiddleware(mux)
}

// ==================== 公共辅助函数 ====================

func hasSuffix(path string, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

func indexOf(s string, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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

func containsPagesAIFixStream(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/ai-fix-stream")
}

func containsPagesAIFix(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/ai-fix")
}

func containsPagesRollback(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/rollback")
}

func containsPagesHTML(path string) bool {
	return indexOf(path, "/pages/") >= 0 && hasSuffix(path, "/html")
}

func containsAdminMemberUID(path string) bool {
	idx := indexOf(path, "/members/")
	if idx < 0 {
		return false
	}
	rest := path[idx+len("/members/"):]
	for len(rest) > 0 && rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}
	return len(rest) > 0
}

func containsUserGroupGID(path string) bool {
	idx := indexOf(path, "/groups/")
	if idx < 0 {
		return false
	}
	rest := path[idx+len("/groups/"):]
	for len(rest) > 0 && rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}
	return len(rest) > 0
}

// ==================== CORS中间件 ====================

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

// ==================== 健康检查处理器 ====================

func makeHealthHandler(engine *services.Engine) http.HandlerFunc {
	startTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		overallStatus := "ok"
		dbStatus := "ok"
		if dbErr := database.Ping(ctx); dbErr != nil {
			dbStatus = "error: " + dbErr.Error()
			overallStatus = "degraded"
		}
		stats := engine.GetStats()
		engineStatus := "ok"
		if stats.QueueLength > 80 {
			engineStatus = "warning: queue usage high"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  overallStatus,
			"version": config.AppVersion,
			"time":    time.Now().Format(time.RFC3339),
			"uptime":  time.Since(startTime).Round(time.Second).String(),
			"database": map[string]interface{}{"status": dbStatus},
			"engine": map[string]interface{}{
				"status":                engineStatus,
				"total_submitted":       stats.TotalSubmitted,
				"total_completed":       stats.TotalCompleted,
				"total_business_failed": stats.TotalBusinessFailed,
				"total_failed":          stats.TotalFailed,
				"current_running":       stats.CurrentRunning,
				"current_ai_active":     stats.CurrentAIActive,
				"queue_length":          stats.QueueLength,
				"queue_capacity":        100,
				"max_workers":           8,
				"max_ai_concurrency":    8,
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

var _ = itoa
