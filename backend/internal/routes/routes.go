package routes

// routes.go — 主路由注册
//
// v0.42 多媒体: 新增 CoursewareAssetService + CoursewareAssetHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/handlers"
	"tedna/internal/logger"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
)

// 模块日志
var routeLog = logger.WithModule("routes")

const roleAdmin = "admin"
const roleSeniorOperator = "senior_operator"
const roleOperator = "operator"
const roleDistrictInspector = "district_inspector"

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
	assetService := services.NewLessonPlanAssetService()
	ciService := services.NewComponentIndexService(cfg)
	liService := services.NewLessonIndexService(cfg)

	// v110(TE-DNA 3.0 P0)新增:AI 助手服务
	aiAssistantService := services.NewAIAssistantService()

	// v113(TE-DNA 3.0 P0.5)新增:AI 助手对话式创作服务
	assistantDesignerService := services.NewAssistantDesignerService(
		cfg.AESKey, cfg.AIAPIBaseURL, cfg.AIAPIKey, cfg.AIDefaultModel,
	)

	// v110:将 AI 助手服务注入教案生成服务
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

	// v110:NewReviewAIHandler 签名增加 aiAssistantService
	reviewAIHandler := handlers.NewReviewAIHandler(cfg, aiAssistantService)

	lpGenHandler := handlers.NewLessonPlanGenHandler(lpGenService, authService)
	recipeHandler := handlers.NewRecipeHandler(recipeService, compService)
	wsStageHandler := handlers.NewWorkshopStageHandler(wsStageService)
	assessHandler := handlers.NewAssessmentHandler(assessService)
	tbHandler := handlers.NewTextbookHandler(tbService)
	assetHandler := handlers.NewLessonPlanAssetHandler(assetService)
	// v125新增：教案互动服务（点赞/收藏）
	interactionService := services.NewLessonPlanInteractionService()

	// v127新增：多级审核 + 抽查服务
	reviewV2Service := services.NewReviewV2Service(compService)

	// v128新增：Token积分系统
	// v129改造：启用积分检查 + 注入精确积分计算钩子
	tokenService := services.NewTokenService()
	creditPolicyService := services.NewCreditPolicyService()
	tokenGuard := services.NewTokenGuard(true)

	// v129新增：注入AI调用积分回调钩子
	ai.SetCreditHook(
		// 消费回调
		func(traceCtx *ai.TraceContext, modelUsed string, inputTokens int, outputTokens int, totalTokens int, latencyMs int64) {
			if traceCtx == nil || traceCtx.UserID == nil || *traceCtx.UserID == "" {
				return
			}
			ctx := context.Background()
			calc := creditPolicyService.CalculateCredits(ctx, modelUsed, inputTokens, outputTokens, totalTokens, traceCtx.SchoolID, latencyMs)
			if calc == nil || calc.CreditsConsumed <= 0 {
				return
			}
			req := &models.TokenConsumeRequest{
				UserID:       *traceCtx.UserID,
				SceneCode:    traceCtx.SceneCode,
				TokensUsed:   totalTokens,
				ModelUsed:    modelUsed,
				LessonPlanID: traceCtx.LessonPlanID,
				PipelineID:   traceCtx.PipelineID,
				Calculation:  calc,
			}
			_ = tokenService.ConsumeTokens(ctx, req)
		},
		// 前置检查回调
		func(traceCtx *ai.TraceContext) (bool, string) {
			if traceCtx == nil || traceCtx.UserID == nil || *traceCtx.UserID == "" {
				return true, ""
			}
			ctx := context.Background()
			result := tokenGuard.CheckBalance(ctx, *traceCtx.UserID)
			if result.HasBalance {
				return true, ""
			}
			return false, result.Message
		},
	)
	inspectionService := services.NewInspectionService()
	interactionHandler := handlers.NewLessonPlanInteractionHandler(interactionService)
	aiTraceHandler := handlers.NewAITraceHandler()

	// v127新增：多级审核 + 抽查处理器
	reviewV2Handler := handlers.NewReviewV2Handler(reviewV2Service)
	inspectionHandler := handlers.NewInspectionHandler(inspectionService)

	// v128新增：Token处理器
	tokenHandler := handlers.NewTokenHandler(tokenService)
	// v129新增：积分策略处理器
	creditPolicyHandler := handlers.NewCreditPolicyHandler(creditPolicyService)

	// v110:学校管理员处理器
	schoolAdminHandler := handlers.NewSchoolAdminHandler(userService, orgService)

	// v130(课件工坊 Phase 1)新增:课件工坊服务+处理器
	cwService := services.NewCoursewareService()
	cwHandler := handlers.NewCoursewareHandler(cwService)
	cwCompHandler := handlers.NewCWComponentHandler()

	// v130(课件工坊 Phase 2)新增:种子数据+模板管理处理器
	cwSeedService := services.NewCoursewareSeedService()
	cwSeedHandler := handlers.NewCWSeedHandler(cwSeedService)

	// v131(课件工坊 Phase 3)新增:课件索引AI生成服务+处理器
	cwIndexService := services.NewCoursewareIndexService(cfg)
	cwIndexHandler := handlers.NewCoursewareIndexHandler(cwIndexService, cwService, authService)

	// v0.42(入口B)新增:PPT上传解析服务
	cwPPTService := services.NewCoursewarePPTService(cfg, cwIndexService)
	cwIndexHandler.SetPPTService(cwPPTService)

	// v134(课件工坊 Phase 4B)新增:课件HTML逐页AI生成服务+处理器
	cwGenService := services.NewCoursewareGenService(cfg)
	cwGenHandler := handlers.NewCoursewareGenHandler(cwGenService, cwService)

	// v0.42 多媒体:课件多媒体资产服务+处理器
	cwAssetService := services.NewCoursewareAssetService(cfg)
	// v0.42.10: 创建OSS上传服务实例（复用已有配置体系）
        ossService := services.NewOSSService(cfg)
        cwAssetHandler := handlers.NewCoursewareAssetHandler(cwAssetService, ossService)

	// v0.42.1 视频编辑:服务+处理器
	videoEditService := services.NewVideoEditService(cfg)
	videoEditHandler := handlers.NewVideoEditHandler(videoEditService)

	// v0.42.8 字幕轨:服务+处理器
	subtitleService := services.NewCoursewareSubtitleService(cfg)
	subtitleHandler := handlers.NewCoursewareSubtitleHandler(subtitleService)

	// v110(TE-DNA 3.0 P0)新增:AI 助手处理器
	aiAssistantHandler := handlers.NewAIAssistantHandler(aiAssistantService)

	// v113(TE-DNA 3.0 P0.5)新增:AI 助手对话式创作处理器
	assistantDesignerHandler := handlers.NewAssistantDesignerHandler(assistantDesignerService)

	// v121新增:AI 助手反馈处理器
	assistantFeedbackHandler := handlers.NewAssistantFeedbackHandler()

	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole(roleAdmin)
	seniorOperatorOnly := middleware.RequireRole(roleSeniorOperator)
	adminOrSchoolAdmin := middleware.RequireRole(roleAdmin, roleSeniorOperator)
	adminOrInspector := middleware.RequireRole(roleAdmin, roleDistrictInspector)

	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))
	// v142优化：测试环境跳过调度器启动
	if !cfg.DisableSchedulers {
		pipelineService.StartNightlyVerifyScheduler()
		tokenService.StartMonthlyQuotaScheduler()
		tokenService.StartAlertCheckScheduler()
		courseService.StartNightlyIndexSyncScheduler()
	}

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

	registerAdminRoutes(mux, authMW, adminOnly, adminOrSchoolAdmin, adminHandler, orgHandler, roleHandler, userHandler, aiConfigHandler, promptHandler, edHandler, courseHandler, wsStageHandler, aiTraceHandler)
	registerPipelineRoutes(mux, authMW, pipelineHandler, sseHandler)
	registerLessonPlanRoutes(mux, authMW, orgHandler, compHandler, lpHandler, lpGenHandler, recipeHandler, wsStageHandler, assessHandler, tbHandler, annotationHandler, reviewAIHandler, assetHandler, interactionHandler)

	// v127新增：注册多级审核 + 抽查路由
	registerReviewV2Routes(mux, authMW, adminOnly, adminOrInspector, adminOrSchoolAdmin, reviewV2Handler, inspectionHandler)

	// v128新增：注册Token积分系统路由
	registerTokenRoutes(mux, authMW, adminOnly, adminOrSchoolAdmin, tokenHandler)
	// v129新增：注册积分策略路由
	registerCreditPolicyRoutes(mux, authMW, adminOnly, creditPolicyHandler)

	// v110:注册学校管理员路由
	registerSchoolAdminRoutes(mux, authMW, seniorOperatorOnly, schoolAdminHandler)

	// v110(TE-DNA 3.0 P0)新增:注册 AI 助手路由
	registerAIAssistantRoutes(mux, authMW, aiAssistantHandler, assistantDesignerHandler)

	// v121:注册 AI 助手反馈路由
	registerAssistantFeedbackRoutes(mux, authMW, adminOnly, assistantFeedbackHandler)

	// v130(课件工坊 Phase 1)新增:注册课件工坊路由
	// v139(模板 AI 提取+微调)新增:模板提取和微调服务
	templateExtractService := services.NewTemplateExtractService(cfg)
	templateRefineService := services.NewTemplateRefineService(cfg)

	// v139:使用 V139 完整构造函数注入提取/微调/认证 3 个服务
	cwTplHandler := handlers.NewCoursewareTemplateHandlerV139(
		templateExtractService,
		templateRefineService,
		authService,
	)
	// v0.42 多媒体: registerCoursewareRoutes 新增 cwAssetHandler 参数
	registerCoursewareRoutes(mux, authMW, adminOnly, cwHandler, cwCompHandler, cwSeedHandler, cwIndexHandler, cwGenHandler, cwTplHandler, cwAssetHandler, videoEditHandler, subtitleHandler)

	// v0.42.9新增：TTS音色列表（登录即可）
	mux.Handle("/api/v1/tts-voices", middleware.Chain(http.HandlerFunc(subtitleHandler.ListTTSVoices), authMW))

	mux.Handle("/api/v1/admin/component-index/batch-compress", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w, "仅支持POST请求")
			return
		}
		go func() {
			importCtx := context.Background()
			success, failed, err := ciService.BatchCompressAllComponents(importCtx, 20, 800)
			if err != nil {
				routeLog.Error("批量压缩组件索引错误", "error", err)
			} else {
				routeLog.Info("批量压缩组件索引完成", "success", success, "failed", failed)
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
