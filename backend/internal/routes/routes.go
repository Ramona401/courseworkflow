package routes

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
)

const roleAdmin = "admin"
const roleSeniorOperator = "senior_operator"
const roleOperator = "operator"

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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

func methodNotAllowedJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	repository.InitTraceWriter()

	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	aiConfigService := services.NewAIConfigService(cfg)
	promptService := services.NewPromptService()
	edService := services.NewExternalDataService(cfg)
	courseService := services.NewCourseService(cfg)
	pipelineService := services.NewPipelineService(cfg)
	orgService := services.NewOrganizationService()
	compService := services.NewComponentService(cfg)
	lpService := services.NewLessonPlanService(compService)
	lpGenService := services.NewLessonPlanGenService(cfg)
	roleService := services.NewRoleService()
	recipeService := services.NewRecipeService()
	wsStageService := services.NewWorkshopStageService()
	assessService := services.NewAssessmentService(recipeService, cfg)
	tbService := services.NewTextbookService(cfg)
	ciService := services.NewComponentIndexService(cfg)
	liService := services.NewLessonIndexService(cfg)

	// v110(TE-DNA 3.0 P0)新增:AI 助手服务
	aiAssistantService := services.NewAIAssistantService()

	// v113(TE-DNA 3.0 P0.5)新增:AI 助手对话式创作服务(Meta-Prompt + AOCI 组件库检索)
	assistantDesignerService := services.NewAssistantDesignerService(
		cfg.AESKey, cfg.AIAPIBaseURL, cfg.AIAPIKey, cfg.AIDefaultModel,
	)

	// v110(TE-DNA 3.0 P0 STEP 3):将 AI 助手服务注入教案生成服务,使 Chat 支持 assistant_id
	lpGenService.SetAssistantService(aiAssistantService)

	wsStageService.SetGenService(lpGenService)
	wsStageService.SetAESKey(cfg.GetAESKey())

	engine := services.NewEngine(8, 8, 100)
	pipelineService.SetEngine(engine)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)
	edHandler := handlers.NewExternalDataHandler(edService)
	courseHandler := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	sseHandler := handlers.NewSSEHandler(authService)
	accountHandler := handlers.NewAccountHandler()
	adminHandler := handlers.NewAdminHandler(userService, orgService)
	roleHandler := handlers.NewRoleHandler(roleService)
	orgHandler := handlers.NewOrganizationHandler(orgService)
	compHandler := handlers.NewComponentHandler(compService)
	lpHandler := handlers.NewLessonPlanHandler(lpService)
	annotationHandler := handlers.NewAnnotationHandler(cfg)

	// v110(TE-DNA 3.0 P0)改动:NewReviewAIHandler 签名增加 aiAssistantService
	reviewAIHandler := handlers.NewReviewAIHandler(cfg, aiAssistantService)

	lpGenHandler := handlers.NewLessonPlanGenHandler(lpGenService, authService)
	recipeHandler := handlers.NewRecipeHandler(recipeService, compService)
	wsStageHandler := handlers.NewWorkshopStageHandler(wsStageService)
	assessHandler := handlers.NewAssessmentHandler(assessService)
	tbHandler := handlers.NewTextbookHandler(tbService)
	aiTraceHandler := handlers.NewAITraceHandler()

	// v110:学校管理员处理器
	schoolAdminHandler := handlers.NewSchoolAdminHandler(userService, orgService)

	// v110(TE-DNA 3.0 P0)新增:AI 助手处理器
	aiAssistantHandler := handlers.NewAIAssistantHandler(aiAssistantService)

	// v113(TE-DNA 3.0 P0.5)新增:AI 助手对话式创作处理器(SSE 流式接口)
	assistantDesignerHandler := handlers.NewAssistantDesignerHandler(assistantDesignerService)

	// v121(P1 收尾 · 2026-04-20)新增:AI 助手反馈处理器
	// 无依赖 service(反馈逻辑简单,直接调 repository)
	assistantFeedbackHandler := handlers.NewAssistantFeedbackHandler()

	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole(roleAdmin)
	seniorOperatorOnly := middleware.RequireRole(roleSeniorOperator)

	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))
	pipelineService.StartNightlyVerifyScheduler()

	mux.Handle("/api/v1/engine/stats", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w, "仅支持GET请求")
			return
		}
		stats := engine.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.GetMe), authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout), authMW))

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
	mux.Handle("/api/v1/account/password", middleware.Chain(http.HandlerFunc(accountHandler.ChangePassword), authMW))

	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	registerAdminRoutes(mux, authMW, adminOnly, adminHandler, roleHandler, userHandler, aiConfigHandler, promptHandler, edHandler, courseHandler, wsStageHandler, aiTraceHandler)
	registerPipelineRoutes(mux, authMW, pipelineHandler, sseHandler)
	registerLessonPlanRoutes(mux, authMW, orgHandler, compHandler, lpHandler, lpGenHandler, recipeHandler, wsStageHandler, assessHandler, tbHandler, annotationHandler, reviewAIHandler)

	// v110:注册学校管理员路由
	registerSchoolAdminRoutes(mux, authMW, seniorOperatorOnly, schoolAdminHandler)

	// v110(TE-DNA 3.0 P0)新增:注册 AI 助手路由
	registerAIAssistantRoutes(mux, authMW, aiAssistantHandler, assistantDesignerHandler)

	// v121(P1 收尾 · 2026-04-20)新增:注册 AI 助手反馈路由
	// - POST   /api/v1/assistant-feedback                创建反馈(登录即可)
	// - DELETE /api/v1/assistant-feedback/{id}           删除自己的反馈
	// - GET    /api/v1/assistant-feedback                列表(admin only)
	// - GET    /api/v1/assistants/{id}/feedback-stats    某助手的反馈统计
	registerAssistantFeedbackRoutes(mux, authMW, adminOnly, assistantFeedbackHandler)

	mux.Handle("/api/v1/admin/component-index/batch-compress", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w, "仅支持POST请求")
			return
		}
		go func() {
			importCtx := context.Background()
			success, failed, err := ciService.BatchCompressAllComponents(importCtx, 20, 800)
			if err != nil {
				log.Printf("批量压缩组件索引错误: %v", err)
			} else {
				log.Printf("批量压缩组件索引完成: 成功=%d 失败=%d", success, failed)
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "message": "批量压缩已开始,请查看服务日志"})
	}), authMW, adminOnly))

	mux.Handle("/api/v1/admin/lesson-index/batch-index", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w, "仅支持POST请求")
			return
		}
		go liService.BatchIndexAllLessonPlans()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "message": "批量教案索引已开始,请查看服务日志"})
	}), authMW, adminOnly))

	engine.StartGracefulShutdown()
	return corsMiddleware(mux)
}

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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   overallStatus,
			"version":  config.AppVersion,
			"time":     time.Now().Format(time.RFC3339),
			"uptime":   time.Since(startTime).Round(time.Second).String(),
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
