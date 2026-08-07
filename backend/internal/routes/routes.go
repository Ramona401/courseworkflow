package routes

// routes.go — 主路由注册
//
// v0.42 多媒体: 新增 CoursewareAssetService + CoursewareAssetHandler
// 合并重构: 移除学校管理员独立体系(school_admin_handler/routes_school_admin)，
//          senior_operator 统一走 /admin（registerAdminRoutes 的 adminOrSchoolAdmin 中间件）。
// 助手轻量选择入口 Phase 1: 新增 TeacherAssistantPrefHandler，注册老师×学科助手偏好读写与可选助手列表路由。
// 班级学情(差异化教学·老师私有资料): 新增 registerClassProfileRoutes，注册 /api/v1/class-profiles 系列。
// 课件审核阶段3: 新增 CoursewareReviewService + CoursewareReviewHandler，注册 /api/v1/courseware-reviews 路由组。
// 批次A(数学图形AI定制): 新增 registerMathGraphRoutes，AI 生成/改编 JSXGraph 构造代码（/api/v1/math-graph/generate）。

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
	lpSectionRewriteService := services.NewLessonPlanSectionRewriteService(cfg)
	lpGenService := services.NewLessonPlanGenService(cfg)
	// 参考资料附件(PDF/Word)压缩服务：把长参考资料 AI 压成结构化要点(复用 lesson_plan 场景)
	lpRefService := services.NewLessonPlanRefService(cfg)
	roleService := services.NewRoleService()
	recipeService := services.NewRecipeService()
	wsStageService := services.NewWorkshopStageService()
	assessService := services.NewAssessmentService(recipeService, cfg)
	tbService := services.NewTextbookService(cfg)
	assetService := services.NewLessonPlanAssetService()
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
	// 迭代7B：注入课本服务，使备课工坊各阶段提示词能拼入老师勾选的课本原文
	wsStageService.SetTextbookService(tbService)
	// 迭代7B修复：对话流程用的是 lpGenService 内部独立的 stageService 实例，需单独注入课本服务
	lpGenService.SetTextbookServiceForStage(tbService)

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
	speechHandler := handlers.NewSpeechHandler(cfg, authService, services.GlobalSpeechConnections)
	accountHandler := handlers.NewAccountHandler()
	accountOrgHandler := handlers.NewAccountOrgHandler()
	adminHandler := handlers.NewAdminHandler(userService, orgService)
	roleHandler := handlers.NewRoleHandler(roleService)
	orgHandler := handlers.NewOrganizationHandler(orgService)
	compHandler := handlers.NewComponentHandler(compService)
	lpHandler := handlers.NewLessonPlanHandler(lpService)
	lpHandler.SetSectionRewriteService(lpSectionRewriteService)
	annotationHandler := handlers.NewAnnotationHandler(cfg)

	// v110:NewReviewAIHandler 签名增加 aiAssistantService
	reviewAIHandler := handlers.NewReviewAIHandler(cfg, aiAssistantService)

	lpGenHandler := handlers.NewLessonPlanGenHandler(lpGenService, authService)
	// 参考资料附件压缩处理器（单端点 POST /ref-material/compress）
	lpRefHandler := handlers.NewLessonPlanRefHandler(lpRefService)
	recipeHandler := handlers.NewRecipeHandler(recipeService)
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

	// v197新增：注入模型分流所需的 AES 密钥（用于解密境内通道 key）
	ai.SetModelPolicyConfig(cfg.GetAESKey())

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

	// 课程知识库（平台级公共只读查询，课件/教案两处复用）
	curriculumHandler := handlers.NewCurriculumHandler()
	// v231 学科字典：单一真相源，供全平台学科下拉统一消费
	subjectHandler := handlers.NewSubjectHandler()

	// 课程大纲（大单元备课能力·批次一）：服务 + 处理器
	courseOutlineService := services.NewCourseOutlineService()
	courseOutlineHandler := handlers.NewCourseOutlineHandler(courseOutlineService)

	// 知识库课标压缩入库系统（迭代一：课标先行）
	kbCompressService := services.NewKBCompressService(cfg)
	kbReviewService := services.NewKBReviewService()
	kbCompressHandler := handlers.NewKBCompressHandler(kbCompressService, authService)
	kbReviewHandler := handlers.NewKBReviewHandler(kbReviewService)
	kbAdminHandler := handlers.NewKBAdminHandler()

	// v130(课件工坊 Phase 1)新增:课件工坊服务+处理器
	cwService := services.NewCoursewareService()
	cwHandler := handlers.NewCoursewareHandler(cwService)
	cwCompHandler := handlers.NewCWComponentHandler()

	// 课件审核阶段3新增:课件多级审核服务+处理器（复用 cwService 装配审核详情含页面）
	cwReviewService := services.NewCoursewareReviewService()
	cwReviewHandler := handlers.NewCoursewareReviewHandler(cwReviewService, cwService)

	// 课件AI审核助手：
	//   - 准备服务负责权限校验、教案/大纲装配、页面互动索引和分批规划；
	//   - 执行器负责顺序批次AI调用、连续性账本继承和最终综合报告；
	//   - AI结果只辅助人工审核，不自动改变课件发布或审核状态。
	cwAIReviewService := services.NewCoursewareAIReviewService(
		cwReviewService,
		cwService,
		aiAssistantService,
	)
	cwAIReviewRunner := services.NewCoursewareAIReviewRunner(
		cfg,
		cwReviewService,
	)
	cwAIReviewHandler := handlers.NewCoursewareAIReviewHandler(
		cwAIReviewService,
		cwAIReviewRunner,
	)

	// v130(课件工坊 Phase 2)新增:种子数据+模板管理处理器
	cwSeedService := services.NewCoursewareSeedService()
	cwSeedHandler := handlers.NewCWSeedHandler(cwSeedService)

	// v131(课件工坊 Phase 3)新增:课件索引AI生成服务+处理器
	cwIndexService := services.NewCoursewareIndexService(cfg)
	cwIndexHandler := handlers.NewCoursewareIndexHandler(cwIndexService, cwService, authService)

	// v0.42(入口B)新增:PPT上传解析服务
	cwPPTService := services.NewCoursewarePPTService(cfg, cwIndexService)
	cwIndexHandler.SetPPTService(cwPPTService)

	// 课件↔教案对齐报告：构造对齐校验服务，双向注入
	//   - 注入 cwIndexService：方案落库(saveAndBroadcast)后自动异步触发对齐校验
	//   - 注入 cwIndexHandler：暴露 GET 查询 / POST 手动重算 两端点
	cwAlignmentService := services.NewCoursewareAlignmentService(cfg)
	cwIndexService.SetAlignmentService(cwAlignmentService)
	cwIndexHandler.SetAlignmentService(cwAlignmentService)

	// 教案规整服务：构造并注入 cwIndexService。
	//   方案落库(saveAndBroadcast)后异步规整教案原文，把又长又乱的教案
	//   规整成去噪保核、预置清单一字不差的干净教案，存库供逐页生成注入，
	//   提升课件对教案的还原度、保证跨页共享案例一致。best-effort 不阻断生成。
	//   仅注入 service（纯后台触发、无对外查询端点，故无需 handler 侧注入）。
	cwNormalizeService := services.NewCoursewareLessonNormalizeService(cfg)
	cwIndexService.SetNormalizeService(cwNormalizeService)

	// v134(课件工坊 Phase 4B)新增:课件HTML逐页AI生成服务+处理器
	cwGenService := services.NewCoursewareGenService(cfg)

	// v0.42 多媒体:课件多媒体资产服务+处理器
	cwAssetService := services.NewCoursewareAssetService(cfg)
	// v0.42.10: 创建OSS上传服务实例（复用已有配置体系）
	ossService := services.NewOSSService(cfg)
	cwAssetHandler := handlers.NewCoursewareAssetHandler(cwAssetService, ossService)

	// 全自动一键装配:主编排服务(依赖 cwGenService/cwAssetService/ossService 三个已有实例)
	cwAutoAssemblyService := services.NewCoursewareAutoAssemblyService(cfg, cwGenService, cwAssetService, ossService)
	// v134 课件HTML生成处理器(下移至此:需注入装配服务;装配服务依赖 asset/oss 故须在其后构造)
	cwGenHandler := handlers.NewCoursewareGenHandler(cwGenService, cwService, cwAutoAssemblyService)

	// v0.42.1 视频编辑:服务+处理器
	videoEditService := services.NewVideoEditService(cfg)
	videoEditHandler := handlers.NewVideoEditHandler(videoEditService)

	// v0.42.8 字幕轨:服务+处理器
	subtitleService := services.NewCoursewareSubtitleService(cfg)
	subtitleHandler := handlers.NewCoursewareSubtitleHandler(subtitleService)

	// v110(TE-DNA 3.0 P0)新增:AI 助手处理器
	aiAssistantHandler := handlers.NewAIAssistantHandler(aiAssistantService)

	// 助手轻量选择入口 Phase 1 新增:老师×学科助手偏好处理器(复用 aiAssistantService 做可选列表与可见性校验)
	taPrefHandler := handlers.NewTeacherAssistantPrefHandler(aiAssistantService)

	// v113(TE-DNA 3.0 P0.5)新增:AI 助手对话式创作处理器
	assistantDesignerHandler := handlers.NewAssistantDesignerHandler(assistantDesignerService)

	// v121新增:AI 助手反馈处理器
	assistantFeedbackHandler := handlers.NewAssistantFeedbackHandler()

	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole(roleAdmin)
	adminOrSchoolAdmin := middleware.RequireRole(roleAdmin, roleSeniorOperator)
	adminOrInspector := middleware.RequireRole(roleAdmin, roleDistrictInspector)

	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))
	// 语音WebSocket由Handler内部校验query token和生产Origin，不套普通authMW。
	mux.HandleFunc("/api/v1/speech/stream", speechHandler.Stream)
	// v142优化：测试环境跳过调度器启动
	// 初始化回收站服务（路由注册与调度器共用）
	trashService := services.NewTrashService()
	trashHandler := handlers.NewTrashHandler(trashService)

	if !cfg.DisableSchedulers {
		pipelineService.StartNightlyVerifyScheduler()
		tokenService.StartMonthlyQuotaScheduler()
		tokenService.StartAlertCheckScheduler()
		tokenService.StartMonthEndTopUpScheduler() // Token自动分配·月初补足(每月1号04:30)
		courseService.StartNightlyIndexSyncScheduler()
		trashService.StartTrashScheduler() // 回收站30天自动清理
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
	// 个人中心「我的组织」——查询当前用户的区域/学校/教研组归属及职位角色（登录即可，只查自己）
	mux.Handle("/api/v1/account/organization", middleware.Chain(http.HandlerFunc(accountOrgHandler.GetMyOrganization), authMW))

	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	registerAdminRoutes(mux, authMW, adminOnly, adminOrSchoolAdmin, adminHandler, orgHandler, roleHandler, userHandler, aiConfigHandler, promptHandler, edHandler, courseHandler, wsStageHandler, aiTraceHandler)
	registerPipelineRoutes(mux, authMW, pipelineHandler, sseHandler)
	registerLessonPlanRoutes(mux, authMW, orgHandler, compHandler, lpHandler, lpGenHandler, recipeHandler, wsStageHandler, assessHandler, tbHandler, annotationHandler, reviewAIHandler, assetHandler, interactionHandler, taPrefHandler, lpRefHandler)

	// v127新增：注册多级审核 + 抽查路由
	registerReviewV2Routes(mux, authMW, adminOnly, adminOrInspector, adminOrSchoolAdmin, reviewV2Handler, inspectionHandler)

	// v128新增：注册Token积分系统路由
	registerTokenRoutes(mux, authMW, adminOnly, adminOrSchoolAdmin, tokenHandler)
	// v129新增：注册积分策略路由
	registerCreditPolicyRoutes(mux, authMW, adminOnly, creditPolicyHandler, cfg)

	// 课程知识库公共只读路由（/api/v1/curriculum/*）
	registerCurriculumRoutes(mux, authMW, curriculumHandler)
	// v231 学科字典路由（公开 GET /api/v1/subjects + admin CRUD /api/v1/admin/subjects）
	registerSubjectRoutes(mux, authMW, adminOnly, subjectHandler)

	// 课程大纲路由（/api/v1/course-outlines）
	registerCourseOutlineRoutes(mux, authMW, courseOutlineHandler)

	// 单元方案路由（大单元备课·独立模块，/api/v1/unit-plans）
	registerUnitPlanRoutes(mux, authMW, cfg)

	// 班级学情路由（差异化教学·老师私有资料，/api/v1/class-profiles）
	registerClassProfileRoutes(mux, authMW, cfg)

	// 通知中心路由（站内信，/api/v1/notifications，阶段5a）
	registerNotificationRoutes(mux, authMW)

	// 知识库课标压缩入库路由（/api/v1/kb/* + /api/v1/sse/kb/* + /api/v1/admin/kb-authorized）
	registerKBRoutes(mux, authMW, adminOnly, kbCompressHandler, kbReviewHandler, kbAdminHandler)

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

	// 课件审核阶段3:注册课件多级审核路由组（/api/v1/courseware-reviews/*）
	// 同时注入 cwReviewHandlerRef，使 cwMux 子路由的 submit-review 能调用审核处理器
	registerCoursewareReviewRoutes(mux, authMW, cwReviewHandler)

	// 课件评审体验使用事件：
	// 严格白名单，只记录枚举、数量和页码，不接收正文、整改要求或搜索关键词。
	registerCoursewareReviewUsageRoutes(
		mux,
		authMW,
	)

	// 课件AI审核助手独立路由组。
	registerCoursewareAIReviewRoutes(
		mux,
		authMW,
		cwAIReviewHandler,
	)

	// 批次A(数学图形AI定制):注册 AI 生成/改编 JSXGraph 构造代码路由（/api/v1/math-graph/generate）
	registerMathGraphRoutes(mux, authMW, cfg)

	// v0.42.9新增：TTS音色列表（登录即可）
	mux.Handle("/api/v1/tts-voices", middleware.Chain(http.HandlerFunc(subtitleHandler.ListTTSVoices), authMW))

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
	// 注册回收站路由
	RegisterTrashRoutes(mux, authMW, trashHandler)

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
