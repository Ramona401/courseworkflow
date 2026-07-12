package routes

// routes_admin.go — Admin/用户/AI配置/提示词/课程路由注册
//
// 超管收口(本批)：把"模型配置/AI统计/审计日志"等敏感入口从 adminOnly 升级为
//   仅超管(is_super=true)可访问，二线管理员(admin 但 is_super=false)被拦。
//   收口方式：
//     - 纯 admin 的敏感路由 → 在 Chain 末尾追加 middleware.SuperAdminOnly()。
//     - 审计日志(/admin/audit-logs) 特殊：原挂 adminOrSchoolAdmin(senior 可看本校)，
//       需求为"senior 仍可看、只拦二线 admin"，故不能用 SuperAdminOnly(会连 senior
//       一起拦)。改为在闭包内判定：若 claims.Role==admin 则要求 claims.IsSuper==true，
//       否则(senior)放行——把二线 admin 精确挑出拦掉。
//   收口清单：
//     - AI追踪 /admin/ai-traces/dashboard
//     - 学校境外授权 /admin/school-model-policies(+ /)
//     - 网关命名 /admin/gateway-naming
//     - 模型别名 /admin/model-alias/rules(+ / )、/fallback、/preview
//     - AI配置 /ai-config/global、/test、/models、/scenes(+ / )
//     - 提示词 /prompts(+ / )
//     - 审计日志 /admin/audit-logs(特殊：仅拦二线 admin)
//   不收口(二线 admin 仍可用)：用户/组织/教研组/角色权限/阶段管理/外部数据/课程。
//
// 归属治理批A(2026-07-04)：
//   - /api/v1/admin/users/ 分支新增 containsUserSchoolSID 匹配:
//     DELETE /api/v1/admin/users/{uid}/schools/{sid} → adminHandler.RemoveUserFromSchool
//     (移出本校=R3: 单事务连带退出该校全部教研组+删校籍行;
//      region_admin 被本组顶部 regionReadOnlyGate 拦截,senior 仅本校由 handler 校验)。
//     匹配器 containsUserSchoolSID 定义于本文件末尾,与 /groups/、/batch 等既有段不冲突。
//
// 本批(区域管理员只读视图接入)：
//   - 新增 adminOrScopedView = RequireRole(admin, senior_operator, region_admin),
//     仅用于三处【读为主】路由: /admin/stats、/admin/users、/admin/users/。
//     修复: region_admin 打开 /admin 用户 Tab 被中间件直接 403(此前 adminOrSchoolAdmin
//     不含 region_admin);数据收窄在 handler 层(resolveAdminUserListScope 辖区学校白名单)。
//   - /admin/users 与 /admin/users/ 两闭包顶部加 regionReadOnlyGate:
//     region_admin 非 GET 一律 403,一道门覆盖该组下全部写入口(创建/编辑/状态/密码/
//     单校批量/跨校批量/教研组归属增删/课程分配更新),免逐个修改分散在
//     admin_handler.go 与 admin_handler_groups.go 的写处理器;主文件写端点内另有
//     ensureRegionAdminReadOnly 双保险。
//   - /admin/stats 对 region 放行但数据未按辖区收窄(GetAdminStats 是全局口径,
//     senior 现状亦然);前端 region 视图不渲染概览 Tab 故实际不消费,放行仅防预取报错。
//   - 其余(audit-logs/orgs/groups/logo)保持 adminOrSchoolAdmin 不放开:
//     audit-logs 查询无范围收窄,放开 region 会扩大暴露面;组织架构 Tab 的组织/成员
//     数据前端走 /lesson-plans/organizations 路由(service 层已按 ResolveDataScope 收窄),
//     不依赖本文件这几条历史遗留路由(前端已不调用,仅堵直连)。
//
// 跨区域多校批量导入(第2步)：
//   - /api/v1/admin/users/ 分支内 /batch-multi-school 精确匹配(紧跟 /batch 之后):
//     POST /api/v1/admin/users/batch-multi-school → adminHandler.BatchCreateMultiSchoolUsers
//     (跨校批量:每行自带 school_id + 逐行成败 + 重名自动改名,仅 admin)。
//     注意:本路由也带 /users/ 前缀,必须置于 default(用户详情)之前,否则被当成
//     userID=batch-multi-school 误解析;且必须与 /batch 区分(用 hasSuffix 精确匹配各自后缀)。
//     权限:此段中间件放行 admin/senior/region,但跨校是 admin 专属——
//     handler 内已加 claims.Role != admin → 403 的双保险;region 另被 regionReadOnlyGate 拦。
//
// 跨区域批量导入(第1步)：
//   - GET /api/v1/admin/region-schools → adminHandler.ListSchoolsByRegion（authMW + adminOnly）
//     供下载 Excel 模板用:前端选区域,返回该区域下全部 active 学校的 (id,name)。仅 admin。
//
// 合并重构(废弃 SchoolAdminPage 并轨)：
//   - /api/v1/admin/users/ 分支最前 /batch 精确匹配:
//     POST /api/v1/admin/users/batch → adminHandler.BatchCreateAdminUsers
//     (单校批量导入教师统一入口;admin/senior 可用,region 被只读门拦)。
//     /batch 与 /batch-multi-school 必须置于 /status、/password、/assignments、用户详情等所有
//     case 之前,否则会被 default 当成用户详情误解析。
//
// v122 改动(AdminPage 权限统一):
//   签名新增 adminOrSchoolAdmin 中间件,对以下路由放开 senior_operator:
//     - /admin/stats / /admin/users(含子路由) / /admin/groups(含子路由) / /admin/orgs / /admin/audit-logs
//   保留 adminOnly 的:
//     - /admin/ai-traces / /admin/region-schools(跨校模板,admin 专属) / /admin/roles
//     - /admin/workshop-stages / /ai-config / /prompts / /external-data / /users(旧版) / /courses 写操作

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/models"
)

// regionReadOnlyGate region_admin 在用户管理中心为只读——非 GET 请求一律 403(第一道门)。
//
// 放在 /admin/users 与 /admin/users/ 两路由闭包顶部统一拦截,一次性覆盖该组下全部
// 写入口(创建/编辑/状态/密码/单校批量/跨校批量/教研组归属增删/移出本校/课程分配更新),
// 避免逐个修改分散在 admin_handler.go 与 admin_handler_groups.go 的处理器;
// admin_handler.go 各写端点内另有 ensureRegionAdminReadOnly 双保险,防将来路由重排
// 导致本门失效。返回 true=放行继续分发,false=已写 403 响应,调用方直接 return。
func regionReadOnlyGate(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.GetClaims(r.Context())
	if ok && claims.Role == models.RoleRegionAdmin && r.Method != http.MethodGet {
		forbiddenJSON(w, "区域管理员在用户管理中心为只读")
		return false
	}
	return true
}

// auditLogSuperGate 审计日志专用收口门：只拦二线 admin，senior 照常放行。
//
// 需求：审计日志对超管、学校管理员(senior)开放，但对二线 admin(admin 且 is_super=false)
// 关闭。因不能用 SuperAdminOnly(会把 senior 一起拦)，故在此单独判定：
//   - claims.Role == admin 时：要求 IsSuper==true，否则拦(二线 admin 被挑出)；
//   - 其余角色(此路由中间件只放行 admin+senior，故实际是 senior)：放行。
// 返回 true=放行继续，false=已写 403 响应，调用方直接 return。
func auditLogSuperGate(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		forbiddenJSON(w, "未找到认证信息")
		return false
	}
	// 只针对 admin 角色收口：admin 必须是超管才放行；senior 走到这里直接放行
	if claims.Role == models.RoleAdmin && !claims.IsSuper {
		forbiddenJSON(w, "审计日志仅超级管理员可访问")
		return false
	}
	return true
}

// containsUserSchoolSID 判断路径是否形如 /api/v1/admin/users/{uid}/schools/{sid}
// （归属治理批A：移出本校路由匹配器；与 /groups/、/batch 等既有段互斥不冲突）
func containsUserSchoolSID(path string) bool {
	if !strings.HasPrefix(path, "/api/v1/admin/users/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/api/v1/admin/users/")
	parts := strings.Split(rest, "/schools/")
	if len(parts) != 2 {
		return false
	}
	uid := strings.TrimSuffix(parts[0], "/")
	sid := strings.TrimSuffix(parts[1], "/")
	return uid != "" && sid != ""
}

// registerAdminRoutes 注册Admin及系统配置相关所有路由
// v80变更:新增aiTraceHandler参数
// v122变更:新增 adminOrSchoolAdmin 参数(用户/教研组/组织/日志/stats 对 senior_operator 放开)
// 本批变更:函数内自建 adminOrScopedView(admin+senior+region 三角色),仅用于 stats/users
//          两组读路由;不改函数签名(routes.go 零改动),region 角色常量直接取
//          models.RoleRegionAdmin(routes.go 未定义 roleRegionAdmin 局部常量,本文件
//          已因只读门 import models,复用之)。
// 超管收口:函数内敏感路由 Chain 末尾追加 middleware.SuperAdminOnly()(不改签名)。
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
	// 本批新增: 读视图中间件——admin/senior/region 三角色放行。
	// 仅挂 /admin/stats、/admin/users、/admin/users/ 三条;region 的数据收窄与只读
	// 分别由 handler 层(resolveAdminUserListScope/ensureUserInScope)与 regionReadOnlyGate 保证。
	adminOrScopedView := middleware.RequireRole(roleAdmin, roleSeniorOperator, models.RoleRegionAdmin)

	// 超管收口: 超管专属中间件（在 adminOnly 之上再收一层 is_super=true）。
	superAdmin := middleware.SuperAdminOnly()

	// ==================== AI调用追踪仪表盘(超管专属)====================

	mux.Handle("/api/v1/admin/ai-traces/dashboard",
		middleware.Chain(http.HandlerFunc(aiTraceHandler.GetDashboard), authMW, adminOnly, superAdmin))

	// ==================== 区域学校列表(跨校批量导入模板用·admin only·第1步)====================
	// GET /api/v1/admin/region-schools?region_id=xxx — 返回该区域下全部 active 学校的 (id,name)
	// 仅 admin 可用(跨校批量导入是 admin 专属功能,不下放 senior/region_admin)。
	// 不收超管: 属用户管理配套(跨校建号),二线 admin 也应可用。
	mux.Handle("/api/v1/admin/region-schools",
		middleware.Chain(http.HandlerFunc(adminHandler.ListSchoolsByRegion), authMW, adminOnly))

	// ==================== 统一用户管理中心(v122:对学校管理员放开;本批:对区域管理员放开只读)====================

	// 统计摘要 — 学校/区域管理员可达(数据为全局口径,region 前端不渲染概览 Tab 实际不消费)
	mux.Handle("/api/v1/admin/stats",
		middleware.Chain(http.HandlerFunc(adminHandler.GetAdminStats), authMW, adminOrScopedView))

	// 操作日志 — 超管专属收口: 中间件放行 admin+senior,闭包内 auditLogSuperGate 拦掉二线 admin
	// (senior 仍可看本校日志; admin 必须 is_super=true)。
	mux.Handle("/api/v1/admin/audit-logs",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auditLogSuperGate(w, r) {
				return
			}
			adminHandler.ListAdminAuditLogs(w, r)
		}), authMW, adminOrSchoolAdmin))

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

	// ==================== 学校境外模型授权策略(超管专属)====================
	// 平台级境外放行,不下放给 senior/region_admin。默认所有学校境内,仅超管显式授权某校走境外。
	// GET    /api/v1/admin/school-model-policies            — 列出全部已授权/已登记学校
	// GET    /api/v1/admin/school-model-policies/{schoolID} — 查单校当前策略(无记录返默认境内)
	// PUT    /api/v1/admin/school-model-policies/{schoolID} — 授权/取消授权(body:{overseas_enabled,note})
	// DELETE /api/v1/admin/school-model-policies/{schoolID} — 删除记录(=回到默认境内)
	smpHandler := handlers.NewSchoolModelPolicyHandler()
	mux.Handle("/api/v1/admin/school-model-policies",
		middleware.Chain(http.HandlerFunc(smpHandler.ListPolicies), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/admin/school-model-policies/",
		middleware.Chain(http.HandlerFunc(smpHandler.HandlePolicyByID), authMW, adminOnly, superAdmin))

	// ==================== 双网关展示名(超管专属)====================
	// 给境外/境内两网关各起业务展示名,供配置界面与将来老师侧渲染读取(老师侧公开读接口在批三-3接)。
	// GET /api/v1/admin/gateway-naming — 查看两网关展示名
	// PUT /api/v1/admin/gateway-naming — 更新(overseas_label/domestic_label,留空不修改)
	gnHandler := handlers.NewGatewayNamingHandler()
	mux.Handle("/api/v1/admin/gateway-naming",
		middleware.Chain(http.HandlerFunc(gnHandler.HandleGatewayNaming), authMW, adminOnly, superAdmin))

	// ==================== 模型别名映射规则(超管专属)====================
	// 真实模型名→业务别名映射(exact精确/prefix前缀,精确优先)。老师侧据此渲染替换在批三-3接。
	// GET/POST   /api/v1/admin/model-alias/rules        — 列表/新增规则
	// PUT/DELETE /api/v1/admin/model-alias/rules/{id}   — 更新/删除规则
	// GET/PUT    /api/v1/admin/model-alias/fallback     — 查/改兜底别名
	// POST       /api/v1/admin/model-alias/preview      — 预览(输入模型名→返回别名,自测)
	maHandler := handlers.NewModelAliasHandler()
	mux.Handle("/api/v1/admin/model-alias/rules",
		middleware.Chain(http.HandlerFunc(maHandler.HandleRules), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/admin/model-alias/rules/",
		middleware.Chain(http.HandlerFunc(maHandler.HandleRuleByID), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/admin/model-alias/fallback",
		middleware.Chain(http.HandlerFunc(maHandler.HandleFallback), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/admin/model-alias/preview",
		middleware.Chain(http.HandlerFunc(maHandler.PreviewAlias), authMW, adminOnly, superAdmin))

	// ==================== 角色权限管理(admin only,不收超管:二线 admin 也应可管角色)====================

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

	// ==================== 用户管理(v122:对学校管理员放开;本批:对区域管理员放开只读)====================

	// GET/POST /api/v1/admin/users — 列表/创建
	// (列表 handler 层做数据范围过滤: admin 全系统/senior 本校/region 辖区学校白名单;
	//  创建对 region 被 regionReadOnlyGate + handler 双保险拦截)
	mux.Handle("/api/v1/admin/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// region_admin 只读门: 非 GET 一律 403(第一道拦截)
		if !regionReadOnlyGate(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListAdminUsers(w, r)
		case http.MethodPost:
			adminHandler.CreateAdminUser(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, adminOrScopedView))

	// 用户详情+子操作 — 学校管理员可管本校用户;区域管理员只读(详情/课程分配查询)
	// 合并重构:/batch 与 /batch-multi-school 精确匹配置于最前,避免被当成用户详情误解析
	mux.Handle("/api/v1/admin/users/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// region_admin 只读门: 非 GET 一律 403,一道门覆盖本组全部写入口
		// (创建在无尾斜杠路由;此处覆盖编辑/状态/密码/两种批量/组归属增删/移出本校/课程分配更新)
		if !regionReadOnlyGate(w, r) {
			return
		}
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/batch-multi-school"):
			// 跨校批量(每行自带 school_id + 逐行成败 + 改名,仅 admin,handler 内双保险拦 senior)
			if r.Method == http.MethodPost {
				adminHandler.BatchCreateMultiSchoolUsers(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持POST请求")
			}
		case hasSuffix(path, "/batch"):
			// 单校批量(整批一个 school_id + 整批回滚)
			if r.Method == http.MethodPost {
				adminHandler.BatchCreateAdminUsers(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持POST请求")
			}
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
		case containsUserSchoolSID(path):
			// 归属治理批A: 移出本校(R3, 单事务连带退出该校全部教研组+删校籍)
			// DELETE /api/v1/admin/users/{uid}/schools/{sid}
			if r.Method == http.MethodDelete {
				adminHandler.RemoveUserFromSchool(w, r)
			} else {
				methodNotAllowedJSON(w, "仅支持DELETE请求")
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
	}), authMW, adminOrScopedView))

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

	// ==================== AI配置(超管专属)====================

	mux.Handle("/api/v1/ai-config/global", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			aiConfigHandler.GetGlobalConfig(w, r)
		case http.MethodPut:
			aiConfigHandler.UpdateGlobalConfig(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT请求")
		}
	}), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/ai-config/test", middleware.Chain(http.HandlerFunc(aiConfigHandler.TestConnection), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/ai-config/models", middleware.Chain(http.HandlerFunc(aiConfigHandler.ListModels), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/ai-config/scenes", middleware.Chain(http.HandlerFunc(aiConfigHandler.GetSceneConfigs), authMW, adminOnly, superAdmin))
	mux.Handle("/api/v1/ai-config/scenes/", middleware.Chain(http.HandlerFunc(aiConfigHandler.UpdateSceneConfig), authMW, adminOnly, superAdmin))

	// ==================== 提示词管理(超管专属)====================

	mux.Handle("/api/v1/prompts", middleware.Chain(http.HandlerFunc(promptHandler.ListPrompts), authMW, adminOnly, superAdmin))
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
	}), authMW, adminOnly, superAdmin))

	// ==================== 阶段管理(admin only,不收超管:属备课工坊基础配置)====================

	mux.Handle("/api/v1/admin/workshop-stages", middleware.Chain(
		http.HandlerFunc(wsStageHandler.ListAllSystemStages), authMW, adminOnly))

	mux.Handle("/api/v1/admin/workshop-stages/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			methodNotAllowedJSON(w, "仅支持PUT请求")
			return
		}
		wsStageHandler.UpdateSystemStage(w, r)
	}), authMW, adminOnly))

	// ==================== 外部数据配置(超管专属:含OSS密钥等敏感凭证)====================

	mux.Handle("/api/v1/external-data/configs", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			edHandler.GetConfigs(w, r)
		case http.MethodPut:
			edHandler.UpdateConfigs(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT请求")
		}
	}), authMW, adminOnly, superAdmin))

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
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "未知的课程子路径"})
		}
	}), authMW))
}
