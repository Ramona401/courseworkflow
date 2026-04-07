package routes

// routes_admin.go — Admin/用户/AI配置/提示词/课程路由注册
//
// 注册的路由：
//   /api/v1/admin/*          — 统一用户管理中心（admin only）
//   /api/v1/users/*          — 旧版用户管理（保留兼容，admin only）
//   /api/v1/ai-config/*      — AI配置（admin only）
//   /api/v1/prompts/*        — 提示词管理（admin only）
//   /api/v1/external-data/*  — 外部数据配置（admin only）
//   /api/v1/courses/*        — 课程管理（读：全员，写：admin）
//   /api/v1/admin/ai-traces/* — AI调用追踪仪表盘（v80新增，admin only）

import (
	"encoding/json"
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerAdminRoutes 注册Admin及系统配置相关所有路由
// v80变更：新增aiTraceHandler参数，注册AI调用追踪仪表盘路由
func registerAdminRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	adminHandler *handlers.AdminHandler,
	roleHandler *handlers.RoleHandler,
	userHandler *handlers.UserHandler,
	aiConfigHandler *handlers.AIConfigHandler,
	promptHandler *handlers.PromptHandler,
	edHandler *handlers.ExternalDataHandler,
	courseHandler *handlers.CourseHandler,
	wsStageHandler *handlers.WorkshopStageHandler,
	aiTraceHandler *handlers.AITraceHandler,
) {
	// ==================== AI调用追踪仪表盘（v80新增，admin only）====================

	// GET /api/v1/admin/ai-traces/dashboard — 获取AI调用统计数据
	mux.Handle("/api/v1/admin/ai-traces/dashboard",
		middleware.Chain(http.HandlerFunc(aiTraceHandler.GetDashboard), authMW, adminOnly))

	// ==================== 统一用户管理中心（admin only）====================

	// 统计摘要
	mux.Handle("/api/v1/admin/stats",
		middleware.Chain(http.HandlerFunc(adminHandler.GetAdminStats), authMW, adminOnly))

	// 操作日志
	mux.Handle("/api/v1/admin/audit-logs",
		middleware.Chain(http.HandlerFunc(adminHandler.ListAdminAuditLogs), authMW, adminOnly))

	// 组织列表
	mux.Handle("/api/v1/admin/orgs",
		middleware.Chain(http.HandlerFunc(adminHandler.ListAdminOrgs), authMW, adminOnly))

	// 教研组列表
	mux.Handle("/api/v1/admin/groups",
		middleware.Chain(http.HandlerFunc(adminHandler.ListAdminGroups), authMW, adminOnly))

	// 教研组成员管理（/admin/groups/{id}/members 和 /admin/groups/{id}/members/{uid}）
	mux.Handle("/api/v1/admin/groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if containsAdminMemberUID(path) {
			// /admin/groups/{id}/members/{uid} — 更新角色或移除成员
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
		if hasSuffix(path, "/members") {
			// /admin/groups/{id}/members — 列表或添加成员
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

	// GET/POST /api/v1/admin/roles — 角色列表/新建
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

	// ==================== 用户管理（/admin/users）====================

	// GET/POST /api/v1/admin/users — 列表/创建
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

	// 用户详情+子操作（含v52任务六的教研组分配）
	mux.Handle("/api/v1/admin/users/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/status"):
			adminHandler.UpdateAdminUserStatus(w, r)
		case hasSuffix(path, "/password"):
			adminHandler.ResetAdminUserPassword(w, r)
		case hasSuffix(path, "/assignments"):
			switch r.Method {
			case http.MethodGet:
				adminHandler.GetAdminUserAssignments(w, r)
			case http.MethodPut:
				adminHandler.UpdateAdminUserAssignments(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		// v52任务六：优先匹配 /users/{uid}/groups/{gid}（移出）
		case containsUserGroupGID(path):
			if r.Method == http.MethodDelete {
				adminHandler.RemoveUserFromGroup(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持DELETE请求")
			}
		// v52任务六：/users/{uid}/groups（加入）
		case hasSuffix(path, "/groups"):
			if r.Method == http.MethodPost {
				adminHandler.AddUserToGroup(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持POST请求")
			}
		// 用户详情/编辑
		default:
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

	// ==================== 旧版用户管理（保留兼容，admin only）====================

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

	// ==================== AI配置（admin only）====================

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
	mux.Handle("/api/v1/ai-config/test",   middleware.Chain(http.HandlerFunc(aiConfigHandler.TestConnection), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/models", middleware.Chain(http.HandlerFunc(aiConfigHandler.ListModels), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes", middleware.Chain(http.HandlerFunc(aiConfigHandler.GetSceneConfigs), authMW, adminOnly))
	mux.Handle("/api/v1/ai-config/scenes/", middleware.Chain(http.HandlerFunc(aiConfigHandler.UpdateSceneConfig), authMW, adminOnly))

	// ==================== 提示词管理（admin only）====================

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

	// ==================== 阶段管理（admin only）====================

	// GET /api/v1/admin/workshop-stages — 全部系统阶段列表（含disabled）
	mux.Handle("/api/v1/admin/workshop-stages", middleware.Chain(
		http.HandlerFunc(wsStageHandler.ListAllSystemStages), authMW, adminOnly))

	// PUT /api/v1/admin/workshop-stages/{code} — 更新系统阶段
	mux.Handle("/api/v1/admin/workshop-stages/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			methodNotAllowedJSON(w, "仅支持PUT请求")
			return
		}
		wsStageHandler.UpdateSystemStage(w, r)
	}), authMW, adminOnly))

	// ==================== 外部数据配置（admin only）====================

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

	// ==================== 课程管理（读：全员，写：admin）====================

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

	mux.Handle("/api/v1/courses/oss-catalog",    middleware.Chain(http.HandlerFunc(courseHandler.GetOSSCatalog), authMW, adminOnly))
	mux.Handle("/api/v1/courses/register-fetch", middleware.Chain(http.HandlerFunc(courseHandler.RegisterAndFetch), authMW, adminOnly))
	mux.Handle("/api/v1/courses/batch-register", middleware.Chain(http.HandlerFunc(courseHandler.BatchRegisterAndFetch), authMW, adminOnly))
	mux.Handle("/api/v1/courses/batch-fetch",    middleware.Chain(http.HandlerFunc(courseHandler.BatchFetchIndexes), authMW, adminOnly))

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
}
