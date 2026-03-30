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
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

func methodNotAllowedJSON(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": message})
}

// Setup 注册所有路由并返回根Handler
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
	orgService := services.NewOrganizationService()
	compService := services.NewComponentService(cfg)
	lpService := services.NewLessonPlanService(compService)
	lpGenService := services.NewLessonPlanGenService(cfg)
	// v52: 角色权限服务
	roleService := services.NewRoleService()

	engine := services.NewEngine(8, 8, 100)
	pipelineService.SetEngine(engine)

	mux.HandleFunc("/api/v1/health", makeHealthHandler(engine))
	pipelineService.StartNightlyVerifyScheduler()

	// ==================== 初始化处理器层 ====================
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	aiConfigHandler := handlers.NewAIConfigHandler(aiConfigService)
	promptHandler := handlers.NewPromptHandler(promptService)
	edHandler := handlers.NewExternalDataHandler(edService)
	courseHandler := handlers.NewCourseHandler(courseService)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService)
	sseHandler := handlers.NewSSEHandler(authService)
	// 通用用户中心（所有已登录用户，自助管理个人账户）
	accountHandler := handlers.NewAccountHandler()
	// 统一用户管理中心（admin+分层权限）
	adminHandler := handlers.NewAdminHandler(userService, orgService)
	// v52: 角色权限管理（admin 专属）
	roleHandler := handlers.NewRoleHandler(roleService)

	orgHandler := handlers.NewOrganizationHandler(orgService)
	compHandler := handlers.NewComponentHandler(compService)
	lpHandler := handlers.NewLessonPlanHandler(lpService)
	lpGenHandler := handlers.NewLessonPlanGenHandler(lpGenService, authService)

	authMW := middleware.AuthMiddleware(authService)
	adminOnly := middleware.RequireRole(roleAdmin)

	// ==================== 公开路由 ====================
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// ==================== 认证路由 ====================
	mux.Handle("/api/v1/auth/me", middleware.Chain(http.HandlerFunc(authHandler.GetMe), authMW))
	mux.Handle("/api/v1/auth/logout", middleware.Chain(http.HandlerFunc(authHandler.Logout), authMW))

	// ==================== 通用用户中心（所有已登录用户）====================
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

	// ==================== 统一用户管理中心（admin+分层权限）====================
	// 接口说明：
	//   /admin/users/*      用户管理（含 /users/{uid}/groups/* 教研组分配，v52任务六新增）
	//   /admin/orgs         组织列表
	//   /admin/groups/*     教研组管理
	//   /admin/roles/*      角色权限管理（v52任务五新增）
	//   /admin/audit-logs   操作日志
	//   /admin/stats        统计摘要

	// 统计摘要（admin only）
	mux.Handle("/api/v1/admin/stats", middleware.Chain(
		http.HandlerFunc(adminHandler.GetAdminStats), authMW, adminOnly))

	// 操作日志（admin only）
	mux.Handle("/api/v1/admin/audit-logs", middleware.Chain(
		http.HandlerFunc(adminHandler.ListAdminAuditLogs), authMW, adminOnly))

	// 组织列表（admin only）
	mux.Handle("/api/v1/admin/orgs", middleware.Chain(
		http.HandlerFunc(adminHandler.ListAdminOrgs), authMW, adminOnly))

	// 教研组列表
	mux.Handle("/api/v1/admin/groups", middleware.Chain(
		http.HandlerFunc(adminHandler.ListAdminGroups), authMW, adminOnly))

	// 教研组成员管理（/admin/groups/{id}/members 和 /admin/groups/{id}/members/{uid}）
	mux.Handle("/api/v1/admin/groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// /admin/groups/{id}/members/{uid} — 更新角色或移除成员
		if containsAdminMemberUID(path) {
			switch r.Method {
			case http.MethodPut:
				adminHandler.UpdateAdminGroupMemberRole(w, r)
			case http.MethodDelete:
				adminHandler.RemoveAdminGroupMember(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持PUT/DELETE请求")
			}
			return
		}
		// /admin/groups/{id}/members — 列表或添加成员
		if hasSuffix(path, "/members") {
			switch r.Method {
			case http.MethodGet:
				adminHandler.ListAdminGroupMembers(w, r)
			case http.MethodPost:
				adminHandler.AddAdminGroupMember(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
			return
		}
		methodNotAllowedJSON(w, "未知的教研组子路径")
	}), authMW, adminOnly))

	// ==================== 角色权限管理（v52任务五，admin only）====================
	// GET  /api/v1/admin/roles         — 角色列表
	// POST /api/v1/admin/roles         — 新建自定义角色
	mux.Handle("/api/v1/admin/roles", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			roleHandler.ListRoles(w, r)
		case http.MethodPost:
			roleHandler.CreateRole(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, adminOnly))

	// 角色子操作（/admin/roles/{id}/*）
	mux.Handle("/api/v1/admin/roles/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/status"):
			roleHandler.UpdateRoleStatus(w, r)
		case hasSuffix(path, "/permissions"):
			switch r.Method {
			case http.MethodGet:
				roleHandler.GetRolePermissions(w, r)
			case http.MethodPut:
				roleHandler.UpdateRolePermissions(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		default:
			switch r.Method {
			case http.MethodGet:
				roleHandler.GetRole(w, r)
			case http.MethodPut:
				roleHandler.UpdateRole(w, r)
			case http.MethodDelete:
				roleHandler.DeleteRole(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW, adminOnly))

	// ==================== 用户管理（/admin/users 列表+创建）====================
	mux.Handle("/api/v1/admin/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListAdminUsers(w, r)
		case http.MethodPost:
			adminHandler.CreateAdminUser(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, adminOnly))

	// 用户详情+子操作（/admin/users/{id}/*）
	// 包含 v52任务六新增的 /users/{uid}/groups 和 /users/{uid}/groups/{gid}
	mux.Handle("/api/v1/admin/users/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/status"):
			// PUT /admin/users/{uid}/status
			adminHandler.UpdateAdminUserStatus(w, r)
		case hasSuffix(path, "/password"):
			// PUT /admin/users/{uid}/password
			adminHandler.ResetAdminUserPassword(w, r)
		case hasSuffix(path, "/assignments"):
			// GET/PUT /admin/users/{uid}/assignments
			switch r.Method {
			case http.MethodGet:
				adminHandler.GetAdminUserAssignments(w, r)
			case http.MethodPut:
				adminHandler.UpdateAdminUserAssignments(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		// v52任务六：用户↔教研组双向分配
		// 优先匹配含 /groups/{gid} 的路径（避免被 /groups 后缀捕获）
		case containsUserGroupGID(path):
			// DELETE /admin/users/{uid}/groups/{gid} — 移出教研组
			if r.Method == http.MethodDelete {
				adminHandler.RemoveUserFromGroup(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持DELETE请求")
			}
		case hasSuffix(path, "/groups"):
			// POST /admin/users/{uid}/groups — 加入教研组
			if r.Method == http.MethodPost {
				adminHandler.AddUserToGroup(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持POST请求")
			}
		default:
			// GET/PUT /admin/users/{uid} — 详情/编辑
			switch r.Method {
			case http.MethodGet:
				adminHandler.GetAdminUserDetail(w, r)
			case http.MethodPut:
				adminHandler.UpdateAdminUser(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}
	}), authMW, adminOnly))

	// ==================== 仪表盘 ====================
	mux.Handle("/api/v1/dashboard/stats", middleware.Chain(http.HandlerFunc(pipelineHandler.GetDashboardStats), authMW))

	// ==================== 引擎状态（仅admin）====================
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

	// ==================== 用户管理（旧接口，仅admin，保留兼容）====================
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

	// ==================== AI配置（仅admin）====================
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

	// ==================== 提示词（仅admin）====================
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

	// ==================== 外部数据配置（仅admin）====================
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

	// ==================== 课程管理 ====================
	mux.Handle("/api/v1/courses", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			courseHandler.ListCourses(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
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

	mux.HandleFunc("/api/v1/sse/pipelines/", sseHandler.StreamPipeline)

	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/batch-verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可触发批量验收")
				return
			}
			pipelineHandler.BatchVerify(w, r)
		case hasSuffix(path, "/batch-create"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量创建Pipeline")
				return
			}
			pipelineHandler.BatchCreate(w, r)
		case hasSuffix(path, "/batch-assign"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可分配审核任务")
				return
			}
			pipelineHandler.BatchAssign(w, r)
		case hasSuffix(path, "/batch-restart"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可批量重跑Pipeline")
				return
			}
			pipelineHandler.BatchRestartFromStep(w, r)
		case hasSuffix(path, "/operators"):
			pipelineHandler.GetOperators(w, r)
		case hasSuffix(path, "/batch-start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量启动Pipeline")
				return
			}
			pipelineHandler.BatchStart(w, r)
		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可启动Pipeline")
				return
			}
			pipelineHandler.StartPipeline(w, r)
		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可取消Pipeline")
				return
			}
			pipelineHandler.CancelPipeline(w, r)
		case hasSuffix(path, "/restart-from"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可重启Pipeline步骤")
				return
			}
			pipelineHandler.RestartFromStep(w, r)
		case hasSuffix(path, "/force-proceed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可强制推进Pipeline")
				return
			}
			pipelineHandler.ForceProceed(w, r)
		case hasSuffix(path, "/submit-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可提交定稿申请")
				return
			}
			pipelineHandler.SubmitFinalize(w, r)
		case hasSuffix(path, "/confirm-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可确认定稿")
				return
			}
			pipelineHandler.ConfirmFinalize(w, r)
		case hasSuffix(path, "/reject-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可退回重审")
				return
			}
			pipelineHandler.RejectFinalize(w, r)
		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "直接定稿仅管理员可操作")
				return
			}
			pipelineHandler.FinalizePipeline(w, r)
		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可快捷通过Pipeline")
				return
			}
			pipelineHandler.MarkPassed(w, r)
		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可触发验收")
				return
			}
			pipelineHandler.VerifyPipeline(w, r)
		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)
		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可使用AI快修")
				return
			}
			pipelineHandler.AIFixPage(w, r)
		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可审核页面")
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)
		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)
		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)
		case hasSuffix(path, "/steps"):
			pipelineHandler.GetSteps(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				pipelineHandler.GetPipelineDetail(w, r)
			case http.MethodDelete:
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
					forbiddenJSON(w, "仅管理员和高级操作员可删除Pipeline")
					return
				}
				pipelineHandler.DeletePipeline(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/DELETE请求")
			}
		}
	}), authMW))

	// ==================== 教案系统路由 ====================
	mux.HandleFunc("/api/v1/lesson-plans/sse/", lpGenHandler.StreamPlan)
	mux.Handle("/api/v1/lesson-plans/plans/start-conversation",
		middleware.Chain(http.HandlerFunc(lpGenHandler.StartConversation), authMW))

	mux.Handle("/api/v1/lesson-plans/organizations", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.ListOrganizations(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可创建组织")
				return
			}
			orgHandler.CreateOrganization(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/organizations/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.GetOrganization(w, r)
		case http.MethodPut:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可更新组织")
				return
			}
			orgHandler.UpdateOrganization(w, r)
		case http.MethodDelete:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可删除组织")
				return
			}
			orgHandler.DeleteOrganization(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/teaching-groups", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.ListTeachingGroups(w, r)
		case http.MethodPost:
			orgHandler.CreateTeachingGroup(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/teaching-groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/members") && r.Method == http.MethodPost:
			orgHandler.AddGroupMember(w, r)
		case indexOf(path, "/members/") >= 0 && r.Method == http.MethodDelete:
			orgHandler.RemoveGroupMember(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				orgHandler.GetTeachingGroupDetail(w, r)
			case http.MethodPut:
				orgHandler.UpdateTeachingGroup(w, r)
			case http.MethodDelete:
				orgHandler.DeleteTeachingGroup(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/my-groups", middleware.Chain(
		http.HandlerFunc(orgHandler.GetUserTeachingGroups), authMW))

	mux.Handle("/api/v1/lesson-plans/components", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			compHandler.ListComponents(w, r)
		case http.MethodPost:
			compHandler.CreateComponent(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/components/match", middleware.Chain(
		http.HandlerFunc(compHandler.MatchComponents), authMW))

	mux.Handle("/api/v1/lesson-plans/components/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/review"):
			compHandler.ReviewComponent(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				compHandler.GetComponent(w, r)
			case http.MethodPut:
				compHandler.UpdateComponent(w, r)
			case http.MethodDelete:
				compHandler.DeleteComponent(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/extractions", middleware.Chain(
		http.HandlerFunc(compHandler.ListExtractions), authMW))

	mux.Handle("/api/v1/lesson-plans/extractions/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if hasSuffix(path, "/confirm") {
			compHandler.ConfirmExtraction(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "未知的萃取子路径"})
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/plans", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lpHandler.ListLessonPlans(w, r)
		case http.MethodPost:
			lpHandler.CreateLessonPlan(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/plans/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/chat"):
			lpGenHandler.Chat(w, r)
		case hasSuffix(path, "/trigger-review"):
			lpGenHandler.TriggerAIReview(w, r)
		case hasSuffix(path, "/apply-suggestions"):
			lpGenHandler.ApplyAISuggestions(w, r)
		case hasSuffix(path, "/conversation"):
			lpGenHandler.GetConversation(w, r)
		case hasSuffix(path, "/publish-personal"):
			lpHandler.PublishPersonal(w, r)
		case hasSuffix(path, "/submit-review"):
			lpHandler.SubmitForReview(w, r)
		case hasSuffix(path, "/review"):
			lpHandler.ReviewLessonPlan(w, r)
		case hasSuffix(path, "/publish-shared"):
			lpHandler.PublishShared(w, r)
		case hasSuffix(path, "/start-development"):
			lpHandler.StartDevelopment(w, r)
		case hasSuffix(path, "/fork"):
			lpHandler.ForkLessonPlan(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				lpHandler.GetLessonPlan(w, r)
			case http.MethodPut:
				lpHandler.UpdateLessonPlan(w, r)
			case http.MethodDelete:
				lpHandler.DeleteLessonPlan(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lpHandler.ListPromptTemplates(w, r)
		case http.MethodPost:
			lpHandler.CreatePromptTemplate(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/resolved"):
			lpHandler.ResolvePromptTemplate(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				lpHandler.GetPromptTemplate(w, r)
			case http.MethodPut:
				lpHandler.UpdatePromptTemplate(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}
	}), authMW))

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

// containsAdminMemberUID 判断路径是否包含 /members/{uid}（有uid则为操作单个成员）
func containsAdminMemberUID(path string) bool {
	idx := indexOf(path, "/members/")
	if idx < 0 {
		return false
	}
	rest := path[idx+len("/members/"):]
	rest = func(s string) string {
		for i := len(s) - 1; i >= 0; i-- {
			if s[i] == '/' {
				return s[:i]
			}
		}
		return s
	}(rest)
	return len(rest) > 0
}

// containsUserGroupGID 判断路径是否为 /users/{uid}/groups/{gid} 格式
// 即路径中包含 /groups/ 且其后有非空内容（gid）
// 用于区分 /users/{uid}/groups（加入）和 /users/{uid}/groups/{gid}（移出）
func containsUserGroupGID(path string) bool {
	idx := indexOf(path, "/groups/")
	if idx < 0 {
		return false
	}
	// /groups/ 之后必须有非空内容（即 gid）
	rest := path[idx+len("/groups/"):]
	// 去除末尾斜杠后应有内容
	for len(rest) > 0 && rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}
	return len(rest) > 0
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
