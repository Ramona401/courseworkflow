package routes

// routes_admin.go — Admin/用户/AI配置/提示词/课程路由注册
//
// v122 改动(AdminPage 权限统一):
//   签名新增 adminOrSchoolAdmin 中间件,对以下路由放开 senior_operator:
//     - /admin/stats               学校管理员能看概览(数据在 handler 层过滤)
//     - /admin/users (含子路由)    学校管理员能管本校用户
//     - /admin/groups (含子路由)   学校管理员能管本校教研组成员
//     - /admin/orgs                学校管理员能看学校列表(用于自己学校筛选)
//     - /admin/audit-logs          学校管理员能看本校相关日志
//   保留 adminOnly 的:
//     - /admin/ai-traces           AI 调用统计(admin 专属)
//     - /admin/roles (角色权限)    角色体系管理(admin 专属)
//     - /admin/workshop-stages     系统阶段管理(admin 专属)
//     - /ai-config /prompts        AI 和提示词配置(admin 专属)
//     - /external-data             外部数据配置(admin 专属)
//     - /users (旧版)              旧路由废弃中,保留 admin 专属
//     - /courses 的写操作          课程管理(admin 专属)

import (
        "encoding/json"
        "net/http"

        "tedna/internal/handlers"
        "tedna/internal/middleware"
)

// registerAdminRoutes 注册Admin及系统配置相关所有路由
// v80变更:新增aiTraceHandler参数
// v122变更:新增 adminOrSchoolAdmin 参数(用户/教研组/组织/日志/stats 对 senior_operator 放开)
func registerAdminRoutes(
        mux *http.ServeMux,
        authMW func(http.Handler) http.Handler,
        adminOnly func(http.Handler) http.Handler,
        adminOrSchoolAdmin func(http.Handler) http.Handler,
        adminHandler *handlers.AdminHandler,
        orgHandler *handlers.OrganizationHandler,
        roleHandler *handlers.RoleHandler,
        userHandler *handlers.UserHandler,
        aiConfigHandler *handlers.AIConfigHandler,
        promptHandler *handlers.PromptHandler,
        edHandler *handlers.ExternalDataHandler,
        courseHandler *handlers.CourseHandler,
        wsStageHandler *handlers.WorkshopStageHandler,
        aiTraceHandler *handlers.AITraceHandler,
) {
        // ==================== AI调用追踪仪表盘(admin only)====================

        mux.Handle("/api/v1/admin/ai-traces/dashboard",
                middleware.Chain(http.HandlerFunc(aiTraceHandler.GetDashboard), authMW, adminOnly))

        // ==================== 统一用户管理中心(v122:对学校管理员放开)====================

        // 统计摘要 — 学校管理员可看(handler 层按 school_id 过滤数据)
        mux.Handle("/api/v1/admin/stats",
                middleware.Chain(http.HandlerFunc(adminHandler.GetAdminStats), authMW, adminOrSchoolAdmin))

        // 操作日志 — 学校管理员可看(handler 层过滤本校相关日志)
        mux.Handle("/api/v1/admin/audit-logs",
                middleware.Chain(http.HandlerFunc(adminHandler.ListAdminAuditLogs), authMW, adminOrSchoolAdmin))

        // 组织列表 — 学校管理员可看(用于组织架构 Tab 筛选)
        mux.Handle("/api/v1/admin/orgs",
                middleware.Chain(http.HandlerFunc(adminHandler.ListAdminOrgs), authMW, adminOrSchoolAdmin))

        // 组织Logo上传 — admin和senior_operator都可操作
        mux.Handle("/api/v1/admin/orgs/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if hasSuffix(r.URL.Path, "/upload-logo") {
                        orgHandler.UploadOrgLogo(w, r)
                        return
                }
                http.Error(w, `{"code":-1,"message":"未知路径"}`, http.StatusNotFound)
        }), authMW, adminOrSchoolAdmin))

        // 教研组列表 — 学校管理员可看
        mux.Handle("/api/v1/admin/groups",
                middleware.Chain(http.HandlerFunc(adminHandler.ListAdminGroups), authMW, adminOrSchoolAdmin))

        // 教研组成员管理 — 学校管理员可管本校教研组成员
        mux.Handle("/api/v1/admin/groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                path := r.URL.Path
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
        }), authMW, adminOrSchoolAdmin))

        // ==================== 角色权限管理(admin only 专属)====================

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

        // ==================== 用户管理(v122:对学校管理员放开)====================

        // GET/POST /api/v1/admin/users — 列表/创建(handler 层做 school_id 过滤和角色白名单校验)
        mux.Handle("/api/v1/admin/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                switch r.Method {
                case http.MethodGet:
                        adminHandler.ListAdminUsers(w, r)
                case http.MethodPost:
                        adminHandler.CreateAdminUser(w, r)
                default:
                        methodNotAllowedJSON(w, "仅支持GET/POST请求")
                }
        }), authMW, adminOrSchoolAdmin))

        // 用户详情+子操作 — 学校管理员可管本校用户
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
                case containsUserGroupGID(path):
                        if r.Method == http.MethodDelete {
                                adminHandler.RemoveUserFromGroup(w, r)
                        } else {
                                methodNotAllowedJSON(w, "仅支持DELETE请求")
                        }
                case hasSuffix(path, "/groups"):
                        if r.Method == http.MethodPost {
                                adminHandler.AddUserToGroup(w, r)
                        } else {
                                methodNotAllowedJSON(w, "仅支持POST请求")
                        }
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
        }), authMW, adminOrSchoolAdmin))

        // ==================== 旧版用户管理(保留兼容,admin only)====================

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

        // ==================== AI配置(admin only)====================

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

        // ==================== 提示词管理(admin only)====================

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

        // ==================== 阶段管理(admin only)====================

        mux.Handle("/api/v1/admin/workshop-stages", middleware.Chain(
                http.HandlerFunc(wsStageHandler.ListAllSystemStages), authMW, adminOnly))

        mux.Handle("/api/v1/admin/workshop-stages/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPut {
                        methodNotAllowedJSON(w, "仅支持PUT请求")
                        return
                }
                wsStageHandler.UpdateSystemStage(w, r)
        }), authMW, adminOnly))

        // ==================== 外部数据配置(admin only)====================

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

        // ==================== 课程管理(读:全员,写:admin)====================

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
                        _ = json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "未知的课程子路径"})
                }
        }), authMW))
}
