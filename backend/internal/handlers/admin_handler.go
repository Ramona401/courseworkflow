package handlers

// admin_handler.go — 统一用户管理中心处理器(主文件)
//
// 本批(区域管理员只读视图接入)：
//   背景: 前端 AdminPage 已为 region_admin 提供"用户 Tab"收窄视图,但后端两层全拦——
//     ① 路由中间件 adminOrSchoolAdmin=RequireRole(admin,senior) 不含 region_admin,请求 403;
//     ② resolveSchoolScope / ensureUserInScope 均无 region 分支,default 兜底"权限不足"。
//   本批规则: region_admin 在用户管理中心【只读】——
//     - 读: 用户列表改走 resolveAdminUserListScope(admin_handler_scope.go),region 按
//       "辖区学校白名单(区域双来源∪递归辖下学校∪本人本校加料)"过滤;用户详情与课程分配
//       读取经 ensureUserInScope 的 region 分支(repository.IsUserInSchools,
//       school_members∪teaching_group_members 与列表 SQL 同口径)。
//     - 写: 路由层 regionReadOnlyGate(GET-only 门,routes_admin.go)第一道拦截 +
//       本文件各写端点 ensureRegionAdminReadOnly 双保险,一律 403。
//     - resolveSchoolScope 刻意不加 region 分支: 它是"单校写语义"(批量导入/建号),
//       region 的写路径落 default"权限不足"正是预期。
//   顺手补齐: GetAdminUserAssignments / UpdateAdminUserAssignments 此前缺
//   ensureUserInScope 范围校验(双守卫架构的遗漏),本批补上(admin 恒过/senior 本校/
//   region 辖区,对既有前端零影响——UserDetailModal 不含课程分配区块)。
//
// 安全加固(历史,配合前端 UserDetailModal 补齐"资料与角色"编辑入口一并上线)：
//   - 新增 ensureSeniorTargetIsTeacher 目标账号级别守卫：
//     ensureUserInScope 只校验"目标是否本校成员",而 school_members 可能混入 admin 等
//     高级别账号(历史数据/测试保留),仅靠成员校验会让学校管理员对同级/上级账号越权操作。
//     现规则：senior 操作时,目标用户【当前角色】必须是骨干(operator)/普通教师(viewer)。
//     接入点：UpdateAdminUser / UpdateAdminUserStatus / ResetAdminUserPassword 三个写端点。
//   - CreateAdminUser / UpdateAdminUser 的 senior 角色校验收紧为白名单：
//     原实现只拦 admin/senior 两个值,region_admin/district_inspector 可穿透(学校管理员
//     可把老师提成区域管理员),现改为 IsSchoolAdminCreatableRole 白名单(仅 operator/viewer)。
//   - UpdateAdminUser 补审计日志 admin.user_update(角色变更属敏感操作必须留痕;
//     该 action 暂未登记 audit_repo.actionNameMap,日志列表会显示原始 action 码,仅外观问题)。
//
// 跨区域多校批量导入(第2步)：
//   - BatchCreateMultiSchoolUsers: POST /api/v1/admin/users/batch-multi-school
//     admin 把一个区域下【多所学校】的老师汇总成一张 Excel 一次性导入(每行自带 school_id)。
//     与单校批量(BatchCreateAdminUsers)是两条【并存】的路:
//       * 单校批量: 整批一个 school_id + 整批回滚(admin 单校 / senior 本校);
//       * 跨校批量: 每行各自 school_id + 逐行成败 + 重名自动改名(仅 admin)。
//     仅 admin 可用(路由层 adminOnly);给 service 一个 5 分钟超时 ctx 防长时间卡住;
//     底层 userService.BatchCreateUsersMultiSchool 返回 created(含改名清单)+failures 明细。
//
// 跨区域批量导入(第1步)：
//   - ListSchoolsByRegion: GET /api/v1/admin/region-schools?region_id=xxx
//     供下载 Excel 模板时使用——前端选区域,本端点返回该区域下全部 active 学校的 (id,name),
//     前端据此在模板里生成"学校清单 + 所属学校下拉列"。仅 admin(adminOnly),复用 GetSchoolsByRegion。
//
// 合并重构(废弃 SchoolAdminPage 并轨)：
//   - BatchCreateAdminUsers: POST /api/v1/admin/users/batch (单校批量,整批回滚)
//     school_id 走 resolveSchoolScope: admin 必填目标学校 / senior 强制本校。
//     角色白名单 operator/viewer;底层复用 userService.BatchCreateUsers(整批回滚 + 行号失败明细)。
//
// 迭代一 Phase 3.2 改动：
//   - CreateAdminUser: 改用 userService.CreateUserWithSchool 事务化建用户+入校。
//
// v122 方案B 历史改动(修复: 学校管理员新建老师看不见):
//   - ensureUserInScope: 从 IsUserInSchoolByGroup 换为 IsUserInSchool (school_members 主判 + 教研组兜底)
//
// v122 原改动(AdminPage 权限统一):
//   - resolveSchoolScope: 根据登录者角色决定数据范围(admin 全系统 / senior 强制本校)
//
// 职责:
//   - AdminHandler struct定义与构造
//   - 用户列表/详情/创建/批量创建(单校)/跨校批量创建/编辑/启用禁用/重置密码
//   - 课程分配查询与更新
//   - 区域学校列表查询(跨校批量导入模板用)
//   - 统计摘要
//   - 错误处理函数
//   - 路径提取工具函数

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 路径前缀常量 ====================

const (
	adminUsersPrefix = "/api/v1/admin/users/"
)

// ==================== Handler结构体 ====================

type AdminHandler struct {
	userService *services.UserService
	orgService  *services.OrganizationService
}

func NewAdminHandler(userService *services.UserService, orgService *services.OrganizationService) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// ==================== 本地类型定义 ====================

type AdminUserListItem struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	RoleName    string  `json:"role_name"`
	Status      string  `json:"status"`
	LoginCount  int     `json:"login_count"`
	LastLoginAt *string `json:"last_login_at"`
	CreatedAt   string  `json:"created_at"`
	SchoolName  string  `json:"school_name"`
	GroupName   string  `json:"group_name"`
	GroupRole   string  `json:"group_role"`
	GroupCount  int     `json:"group_count"`
}

type AdminUserDetail struct {
	AdminUserListItem
	CourseAssignments []*models.CourseAssignment `json:"course_assignments"`
	TeachingGroups    []AdminGroupMembership     `json:"teaching_groups"`
}

type AdminGroupMembership struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	SchoolName string `json:"school_name"`
	Role       string `json:"role"`
	RoleName   string `json:"role_name"`
	JoinedAt   string `json:"joined_at"`
	IsLead     bool   `json:"is_lead"`
}

// ==================== v122 新增:权限范围辅助 ====================

// resolveSchoolScope 根据登录者角色决定数据范围(单校写语义)
// 返回 (effectiveSchoolID, userRole, error)
//   - admin: 返回前端传的 schoolID(可以为空 → 全系统)
//   - senior_operator: 强制返回其管理的学校 ID(忽略前端传入),未绑定学校则返回错误
//   - region_admin: 【刻意不支持】——本函数是"单校写语义"(批量导入/建号等),区域管理员
//     在用户管理中心为只读:读路径走 resolveAdminUserListScope(学校白名单,见
//     admin_handler_scope.go),写路径落 default 返回"权限不足"正是预期,勿在此加 region 分支。
//   - 其他角色: 此处不应走到(中间件已拦截),保险起见返回错误
func resolveSchoolScope(ctx context.Context, requestedSchoolID string) (string, string, error) {
	claims, ok := middleware.GetClaims(ctx)
	if !ok {
		return "", "", fmt.Errorf("未登录")
	}

	switch claims.Role {
	case models.RoleAdmin:
		return requestedSchoolID, claims.Role, nil

	case models.RoleSeniorOperator:
		school, err := repository.GetSchoolByAdminUserID(ctx, claims.UserID)
		if err != nil {
			return "", claims.Role, fmt.Errorf("您尚未绑定学校,请联系系统管理员")
		}
		if school == nil || school.ID == "" {
			return "", claims.Role, fmt.Errorf("您尚未绑定学校,请联系系统管理员")
		}
		return school.ID, claims.Role, nil

	default:
		return "", claims.Role, fmt.Errorf("权限不足")
	}
}

// ==================== 统计摘要 ====================

func (h *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	stats, err := repository.GetAdminStats(r.Context())
	if err != nil {
		utils.InternalError(w, "获取统计失败: "+err.Error())
		return
	}
	utils.Success(w, stats)
}

// ==================== 区域学校列表(跨校批量导入模板用·第1步) ====================

// regionSchoolItem 精简学校项(只给前端生成模板需要的 id+name,不暴露其余组织字段)
type regionSchoolItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListSchoolsByRegion GET /api/v1/admin/region-schools?region_id=xxx
//
// 用途:
//
//	"跨区域多校批量导入"下载 Excel 模板时,前端先选一个区域,本端点返回该区域下全部
//	active 学校的 (id, name)。前端据此在模板内生成"学校清单 + 所属学校下拉列",
//	老师选自己学校,汇总上传时每行自带准确 school_id,匹配零误差。
//
// 权限: 仅 admin(路由层 adminOnly 已保证;此功能不下放 senior/region_admin)。
// 数据: 复用 repository.GetSchoolsByRegion(只返 parent_id=region 且 type='school' status='active')。
// 返回: {schools:[{id,name}], total}; region_id 为空 → 400; 该区域无学校 → 返回空数组(非错误)。
func (h *AdminHandler) ListSchoolsByRegion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		utils.BadRequest(w, "请先选择区域(缺少 region_id)")
		return
	}

	schools, err := repository.GetSchoolsByRegion(r.Context(), regionID)
	if err != nil {
		utils.InternalError(w, "获取区域学校列表失败: "+err.Error())
		return
	}

	// 只取 id+name 返给前端(精简,避免暴露 settings/logo 等无关字段)
	items := make([]regionSchoolItem, 0, len(schools))
	for _, s := range schools {
		items = append(items, regionSchoolItem{ID: s.ID, Name: s.Name})
	}
	utils.Success(w, map[string]interface{}{"schools": items, "total": len(items)})
}

// ==================== 用户列表 ====================

// ListAdminUsers GET /api/v1/admin/users
//
// 数据范围经 resolveAdminUserListScope(admin_handler_scope.go)解析:
//   - admin  : 单校 SchoolID(前端传参,可空=全系统);
//   - senior : 单校 SchoolID(强制本校);
//   - region : 学校白名单 SchoolIDs(辖区学校;前端另传 school_id 时须在白名单内否则 403)。
func (h *AdminHandler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	scope, scopeErr := resolveAdminUserListScope(r.Context(), q.Get("school_id"))
	if scopeErr != nil {
		utils.Forbidden(w, scopeErr.Error())
		return
	}

	result, err := repository.ListAdminUsers(r.Context(), repository.AdminUserListParams{
		Page:      page,
		PageSize:  pageSize,
		Role:      q.Get("role"),
		Status:    q.Get("status"),
		Keyword:   q.Get("keyword"),
		SchoolID:  scope.SchoolID,
		SchoolIDs: scope.SchoolIDs,
		GroupID:   q.Get("group_id"),
	})
	if err != nil {
		utils.InternalError(w, "获取用户列表失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== 用户详情 ====================

func (h *AdminHandler) GetAdminUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	userID := extractAdminPathID(r.URL.Path, adminUsersPrefix)
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	detail, err := repository.GetAdminUserDetail(r.Context(), userID)
	if err != nil {
		utils.InternalError(w, "获取用户详情失败: "+err.Error())
		return
	}
	utils.Success(w, detail)
}

// ==================== 创建用户 ====================

// CreateAdminUser POST /api/v1/admin/users
// 迭代一 Phase 3.2: 事务化建用户+(senior分支)入校，删除原 WARN 降级
//   - senior_operator: targetSchoolID 非空 → 事务内建用户+入校原子完成
//   - admin          : targetSchoolID 为空 → 只建用户，不自动入校(与历史一致)
//
// 安全加固: senior 角色校验从"拦 admin/senior 两个值"收紧为白名单(仅 operator/viewer),
// 堵住学校管理员创建 region_admin/district_inspector 账号的穿透。
// 本批: region_admin 只读双保险(路由层 GET-only 门已拦,此处防中间件配置疏漏)。
func (h *AdminHandler) CreateAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	// 安全加固: senior_operator 角色白名单校验(仅可创建骨干/普通教师)
	if claims.Role == models.RoleSeniorOperator {
		if !models.IsSchoolAdminCreatableRole(req.Role) {
			utils.Forbidden(w, "学校管理员仅可创建骨干教师或普通教师账号")
			return
		}
	}

	// 单用户建号的学校归属规则：
	//   - 学校管理员：学校只取服务端实时解析的管理学校，忽略客户端 school_id；
	//   - 系统管理员创建 operator/viewer 教学账号：必须显式选择启用且教育域合法的学校；
	//   - 系统管理员创建 admin/district_inspector 管理账号：不写普通校籍。
	//
	// 这里先完成学校真实性校验，再把学校ID交给 UserService。UserService 仍会在
	// 正式事务前二次校验，并把 users + school_members + personal token_accounts
	// 放进同一事务，Handler 校验不能替代 Service 的最终安全边界。
	var targetSchoolID string
	var source string

	switch claims.Role {
	case models.RoleSeniorOperator:
		school, err := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
		if err != nil || school == nil || school.ID == "" {
			utils.Forbidden(w, "您尚未绑定学校,无法创建本校用户")
			return
		}
		targetSchoolID = school.ID
		source = "school_admin_create"
		req.SchoolID = targetSchoolID

	case models.RoleAdmin:
		req.SchoolID = strings.TrimSpace(req.SchoolID)

		if models.IsSchoolAdminCreatableRole(req.Role) {
			if req.SchoolID == "" {
				utils.BadRequest(w, "创建骨干教师或普通教师时必须选择所属学校")
				return
			}

			school, err := repository.GetOrganizationByID(r.Context(), req.SchoolID)
			if err != nil ||
				school == nil ||
				school.Type != models.OrgTypeSchool ||
				school.Status != models.StatusActive ||
				!models.IsTeachingEducationDomain(school.EducationDomain) {
				utils.BadRequest(w, "所选学校不存在、已停用或教育类型未正确配置")
				return
			}

			targetSchoolID = school.ID
			source = "admin_create"
			req.SchoolID = targetSchoolID
		} else {
			// 管理身份不作为普通教学账号入校，防止把平台管理账号混入学校成员范围。
			req.SchoolID = ""
		}

	default:
		utils.Forbidden(w, "权限不足")
		return
	}

	// 用户、校籍和个人积分账户在 UserService 内按事务闭环创建。
	userInfo, err := h.userService.CreateUserWithSchool(
		r.Context(),
		&req,
		targetSchoolID,
		source,
	)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}

	repository.WriteAuditLog(claims.UserID, "admin.user_create",
		map[string]interface{}{
			"target_user": userInfo.ID,
			"username":    userInfo.Username,
			"role":        userInfo.Role,
			"school_id":   targetSchoolID,
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, userInfo)
}

// ==================== 批量创建用户·单校(合并重构新增) ====================

// BatchCreateAdminUsers POST /api/v1/admin/users/batch
//
// 单校批量导入教师统一入口(替代旧 /school-admin/users/batch)。整批一个 school_id + 整批回滚。
//
// 请求体: {role, school_id?, users:[{username,display_name,password}]}
//
// 数据范围(resolveSchoolScope,与列表/详情一致):
//   - admin          : effectiveSchoolID = 前端传入的 school_id(必须非空,否则 400);
//   - senior_operator: effectiveSchoolID 强制本校(忽略前端)。
//   - region_admin   : 只读——handler 双保险 + resolveSchoolScope default 双双拒绝。
//
// 角色白名单 operator/viewer; source 按角色区分审计语义。
//
// 返回: 整批校验失败 → 200 + Success=false + Failures; 系统级异常 → 500; 全成功 → 200 + Success=true。
//
// ⚠️ 这是"单校批量"。"跨区域多校批量"(逐行 school_id + 逐行成败 + 重名自动改名)是另一条
//
//	并存的路,见下方 BatchCreateMultiSchoolUsers,本端点不动。
func (h *AdminHandler) BatchCreateAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	// 仅解析前端需要提供的字段(role + school_id + users);source 不信任前端,后端按角色强制
	var body struct {
		Role     string                   `json:"role"`
		SchoolID string                   `json:"school_id"`
		Users    []services.BatchUserItem `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	// 数据范围:admin 用前端传入的 school_id,senior 强制本校(忽略前端)
	effectiveSchoolID, role, scopeErr := resolveSchoolScope(r.Context(), body.SchoolID)
	if scopeErr != nil {
		utils.Forbidden(w, scopeErr.Error())
		return
	}

	// admin 批量导入必须指定目标学校(为空则会产生不入校的孤儿账号)
	if effectiveSchoolID == "" {
		utils.BadRequest(w, "批量导入请先指定目标学校")
		return
	}

	// 角色白名单(仅可批量建 operator/viewer)
	if !models.IsSchoolAdminCreatableRole(body.Role) {
		utils.BadRequest(w, "仅可批量创建骨干教师(operator)或普通教师(viewer)账号")
		return
	}

	// source 按角色区分,保留审计语义(senior 与旧端点一致)
	source := "admin_batch_create"
	if role == models.RoleSeniorOperator {
		source = "school_admin_batch_create"
	}

	svcReq := &services.BatchCreateUsersRequest{
		Role:     body.Role,
		SchoolID: effectiveSchoolID,
		Source:   source,
		Users:    body.Users,
	}

	result, err := h.userService.BatchCreateUsers(r.Context(), svcReq)
	if err != nil {
		// 系统级异常(开事务失败/查重失败等):返回 500
		utils.InternalError(w, "批量创建用户失败: "+err.Error())
		return
	}

	// 业务级结果(成功 or 整批校验失败带明细)统一 200 + result 返回
	if result.Success {
		repository.WriteAuditLog(claims.UserID, "admin.user_batch_create",
			map[string]interface{}{
				"role":          body.Role,
				"school_id":     effectiveSchoolID,
				"created_count": result.CreatedCount,
				"source":        source,
			}, repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, result)
}

// ==================== 批量创建用户·跨校(第2步) ====================

// BatchCreateMultiSchoolUsers POST /api/v1/admin/users/batch-multi-school
//
// 跨区域多校批量导入(仅 admin):一张 Excel 汇总一个区域下多所学校的老师,每行自带 school_id。
// 与单校批量(BatchCreateAdminUsers)并存:本端点逐行成败 + 重名自动改名,不整批回滚。
//
// 请求体: {role, users:[{username,display_name,password,school_id}]}
//   - role : 批次级统一角色(仅 operator/viewer);
//   - 每行 school_id : 前端由"学校名→ID"反查填入,后端会一次性批量校验有效性。
//
// 数据范围/权限: 仅 admin(路由层 adminOnly 已挡;此处再确认一次以防直连)。
//
//	senior/region_admin 不走此端点(跨校是 admin 专属能力)。
//
// 超时: 给 service 一个 5 分钟超时 ctx,防 2000 人逐行事务长时间卡住;
//
//	超时则已建成的保留、剩余行在 result.failures 标"超时未处理"。
//
// 返回:
//   - 前置异常(行数超限/角色非法) → service 返 error → 本 handler 400;
//   - 学校批量校验查库失败 → service 返 error → 500;
//   - 正常逐行处理 → 200 + result(created 含改名清单 + failures 明细),error 恒 nil。
func (h *AdminHandler) BatchCreateMultiSchoolUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 跨校批量仅 admin(路由 adminOnly 已挡,这里双保险防中间件配置疏漏)
	if claims.Role != models.RoleAdmin {
		utils.Forbidden(w, "跨校批量导入仅系统管理员可用")
		return
	}

	var body struct {
		Role  string                         `json:"role"`
		Users []services.MultiSchoolUserItem `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	// 角色白名单(仅可批量建 operator/viewer)——提前在 handler 拦一次给更友好的 400
	if !models.IsSchoolAdminCreatableRole(body.Role) {
		utils.BadRequest(w, "仅可批量创建骨干教师(operator)或普通教师(viewer)账号")
		return
	}

	// 5 分钟超时 ctx:2000 人逐行小事务估计 1-2 分钟内,5 分钟很宽裕;超时由 service 优雅收尾
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	svcReq := &services.MultiSchoolBatchRequest{
		Role:   body.Role,
		Source: "admin_multi_school_batch_create",
		Users:  body.Users,
	}

	result, err := h.userService.BatchCreateUsersMultiSchool(ctx, svcReq)
	if err != nil {
		// service 的 error 仅用于前置/系统级异常:
		//   行数超限/角色非法 → 当 400(请求数据问题);其余(查库失败) → 500。
		msg := err.Error()
		if strings.Contains(msg, "请分批导入") ||
			strings.Contains(msg, "无效的批次角色") ||
			strings.Contains(msg, "仅可批量创建") ||
			strings.Contains(msg, "用户列表为空") {
			utils.BadRequest(w, msg)
		} else {
			utils.InternalError(w, "跨校批量创建用户失败: "+msg)
		}
		return
	}

	// 逐行处理完成(无论成败明细):写审计(记总数/成功/失败,改名清单在 result 里返前端)
	repository.WriteAuditLog(claims.UserID, "admin.user_batch_create",
		map[string]interface{}{
			"mode":          "multi_school",
			"role":          body.Role,
			"total":         result.TotalCount,
			"created_count": result.CreatedCount,
			"failed_count":  result.FailedCount,
		}, repository.GetClientIP(r.RemoteAddr))

	utils.Success(w, result)
}

// ==================== 编辑用户 ====================

// UpdateAdminUser PUT /api/v1/admin/users/{id}
// 编辑用户"显示名称 + 系统角色"(前端 UserDetailModal「资料与角色」区块调用)。
//
// 安全加固:
//  1. senior 目标级别守卫: ensureSeniorTargetIsTeacher——目标当前角色必须是骨干/普通教师;
//  2. senior 角色白名单收紧: 仅可授予 operator/viewer(原实现只拦 admin/senior,
//     region_admin/district_inspector 可穿透);
//  3. 补审计日志 admin.user_update(角色变更敏感操作留痕)。
//  4. 本批: region_admin 只读双保险。
//
// service 层另有"不能修改自己的角色"(ErrCannotChangeOwnRole)保护。
func (h *AdminHandler) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractAdminPathID(r.URL.Path, adminUsersPrefix)
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if claims.Role == models.RoleSeniorOperator {
		// 守卫一: 目标账号级别——目标当前角色必须是骨干/普通教师
		if err := ensureSeniorTargetIsTeacher(r.Context(), userID); err != nil {
			utils.Forbidden(w, err.Error())
			return
		}
		// 守卫二: 角色白名单——仅可授予 operator/viewer
		if !models.IsSchoolAdminCreatableRole(req.Role) {
			utils.Forbidden(w, "学校管理员仅可将用户角色设置为骨干教师或普通教师")
			return
		}
	}

	userInfo, err := h.userService.UpdateUser(r.Context(), userID, claims.UserID, &req)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}

	// 审计: 资料与角色变更留痕
	repository.WriteAuditLog(claims.UserID, "admin.user_update",
		map[string]interface{}{
			"target_user":  userID,
			"display_name": req.DisplayName,
			"new_role":     req.Role,
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, userInfo)
}

// ==================== 启用/禁用 ====================

func (h *AdminHandler) UpdateAdminUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, adminUsersPrefix, "/status")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	// 安全加固: senior 目标级别守卫(不能启禁用同级或更高级别账号)
	if claims.Role == models.RoleSeniorOperator {
		if err := ensureSeniorTargetIsTeacher(r.Context(), userID); err != nil {
			utils.Forbidden(w, err.Error())
			return
		}
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.UpdateStatus(r.Context(), userID, claims.UserID, &req); err != nil {
		handleAdminUserError(w, err)
		return
	}
	repository.WriteAuditLog(claims.UserID, "admin.user_status",
		map[string]interface{}{"target_user": userID, "new_status": req.Status},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "用户状态更新成功"})
}

// ==================== 重置密码 ====================

func (h *AdminHandler) ResetAdminUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, adminUsersPrefix, "/password")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	// 安全加固: senior 目标级别守卫(不能重置同级或更高级别账号的密码——
	// 如混入本校成员名单的 admin 账号,原实现仅靠成员校验会被放行)
	if claims.Role == models.RoleSeniorOperator {
		if err := ensureSeniorTargetIsTeacher(r.Context(), userID); err != nil {
			utils.Forbidden(w, err.Error())
			return
		}
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.ResetPassword(r.Context(), userID, &req); err != nil {
		handleAdminUserError(w, err)
		return
	}
	repository.WriteAuditLog(claims.UserID, "admin.user_reset_password",
		map[string]interface{}{"target_user": userID},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 课程分配 ====================

// GetAdminUserAssignments GET /api/v1/admin/users/{id}/assignments
// 本批补齐: 增加 ensureUserInScope 范围校验(此前缺失,属双守卫架构的遗漏)——
// admin 恒过 / senior 仅本校成员 / region 仅辖区学校成员。
func (h *AdminHandler) GetAdminUserAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, adminUsersPrefix, "/assignments")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	// 数据范围校验(与用户详情同口径)
	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	assignments, err := h.userService.GetAssignments(r.Context(), userID)
	if err != nil {
		utils.InternalError(w, "获取课程分配失败")
		return
	}
	if assignments == nil {
		assignments = []*models.CourseAssignment{}
	}
	utils.Success(w, assignments)
}

// UpdateAdminUserAssignments PUT /api/v1/admin/users/{id}/assignments
// 本批补齐: region_admin 只读双保险 + ensureUserInScope 范围校验(此前缺失)。
func (h *AdminHandler) UpdateAdminUserAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, adminUsersPrefix, "/assignments")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}

	// region_admin 只读双保险
	if err := ensureRegionAdminReadOnly(claims.Role); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	// 数据范围校验(与用户详情同口径)
	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	var req models.UpdateAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	result, err := h.userService.UpdateAssignments(r.Context(), userID, claims.UserID, &req)
	if err != nil {
		utils.InternalError(w, "更新课程分配失败")
		return
	}
	utils.Success(w, result)
}

// ==================== v122 新增:范围校验辅助 ====================

// ensureUserInScope 校验目标用户是否在登录者的数据范围内
// v122 方案B: senior 校验从 IsUserInSchoolByGroup 换为 IsUserInSchool
//
//	(school_members 主判 + teaching_group_members 兜底)
//   - admin: 总是允许
//   - senior_operator: 目标用户必须属于其管理的学校(通过 school_members 或 教研组)
//   - region_admin(本批新增): 目标用户必须属于其辖区学校
//     (repository.IsUserInSchools, school_members ∪ teaching_group_members 与用户列表 SQL 同口径,
//     保证"列表里看得到的人,详情一定点得开"——若用 data_scope 的 ListSchoolMemberIDs 口径
//     会漏掉只在教研组的历史用户,列表可见详情却 403)
//
// ⚠️ 本函数只回答"目标是否范围内成员",不回答"目标级别是否可管"——后者由
// ensureSeniorTargetIsTeacher 负责,senior 的写端点须两道守卫都过;
// region_admin 无写权限(ensureRegionAdminReadOnly 已拦),本函数对其仅服务读路径。
func ensureUserInScope(ctx context.Context, targetUserID string) error {
	claims, ok := middleware.GetClaims(ctx)
	if !ok {
		return fmt.Errorf("未登录")
	}

	if claims.Role == models.RoleAdmin {
		return nil
	}

	if claims.Role == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(ctx, claims.UserID)
		if err != nil || school == nil || school.ID == "" {
			return fmt.Errorf("您尚未绑定学校,请联系系统管理员")
		}
		// v122 方案B: 使用 IsUserInSchool(school_members ∪ teaching_group_members)
		inSchool, err := repository.IsUserInSchool(ctx, targetUserID, school.ID)
		if err != nil {
			return fmt.Errorf("校验用户所属学校失败")
		}
		if !inSchool {
			return fmt.Errorf("该用户不属于您管理的学校")
		}
		return nil
	}

	// 本批新增: region_admin 读路径——目标须为辖区学校成员
	if claims.Role == models.RoleRegionAdmin {
		schoolIDs, err := resolveRegionScopeSchoolIDs(ctx, claims.UserID)
		if err != nil {
			return err
		}
		inScope, err := repository.IsUserInSchools(ctx, targetUserID, schoolIDs)
		if err != nil {
			return fmt.Errorf("校验用户所属学校失败")
		}
		if !inScope {
			return fmt.Errorf("该用户不属于您管辖的区域")
		}
		return nil
	}

	return fmt.Errorf("权限不足")
}

// ensureSeniorTargetIsTeacher 学校管理员"目标账号级别"守卫(安全加固)
//
// 背景: ensureUserInScope 只校验"目标是否本校成员",而 school_members 里可能混入
// admin 等高级别账号(历史数据/测试保留),仅靠成员校验会让学校管理员对同级或
// 上级账号做出编辑资料/改角色/重置密码/启禁用等越权操作。
//
// 规则: 目标用户【当前角色】必须是骨干教师(operator)或普通教师(viewer),否则拒绝。
// 仅在操作者为 senior_operator 时调用;admin 不受此限制。
func ensureSeniorTargetIsTeacher(ctx context.Context, targetUserID string) error {
	target, err := repository.FindUserByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("获取目标用户信息失败")
	}
	if !models.IsSchoolAdminCreatableRole(target.Role) {
		return fmt.Errorf("学校管理员不能管理同级或更高级别的账号")
	}
	return nil
}

// ==================== 错误处理 ====================

func handleAdminUserError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrUsernameRequired,
		services.ErrDisplayNameRequired,
		services.ErrPasswordTooShort,
		services.ErrInvalidRole,
		services.ErrInvalidStatus,
		services.ErrUsernameExists,
		services.ErrCannotDisableSelf,
		services.ErrCannotChangeOwnRole,
		services.ErrRoleAppointmentOnly,
		services.ErrSchoolRequired,
		services.ErrSchoolUnavailable:
		utils.BadRequest(w, err.Error())
	case services.ErrSchoolTokenAccountUnavailable:
		utils.Fail(w, http.StatusConflict, err.Error())
	case services.ErrUserNotFound:
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}

// ==================== 路径提取工具函数 ====================

func extractAdminPathID(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if idx := strings.Index(id, "/"); idx > 0 {
		id = id[:idx]
	}
	return id
}

func extractAdminMiddleID(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(rest, "/"); idx > 0 {
		candidate := rest[:idx]
		tail := rest[idx:]
		if strings.HasPrefix(tail, suffix) {
			return candidate
		}
	}
	return ""
}

func extractAdminGroupMemberPath(path string) (string, string) {
	prefix := "/api/v1/admin/groups/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/members/")
	if len(parts) != 2 {
		return "", ""
	}
	gid := strings.TrimSuffix(parts[0], "/")
	uid := strings.TrimSuffix(parts[1], "/")
	if gid == "" || uid == "" {
		return "", ""
	}
	return gid, uid
}

func extractUserGroupPath(path string) (string, string) {
	if !strings.HasPrefix(path, adminUsersPrefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, adminUsersPrefix)
	parts := strings.SplitN(rest, "/groups/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	uid := strings.TrimSuffix(parts[0], "/")
	gid := strings.TrimSuffix(parts[1], "/")
	if uid == "" || gid == "" {
		return "", ""
	}
	return uid, gid
}

// ==================== 格式化辅助 ====================

func formatRoleName(role string) string {
	names := map[string]string{
		"admin":           "系统管理员",
		"senior_operator": "学校管理员",
		"operator":        "骨干教师",
		"viewer":          "普通教师",
	}
	if n, ok := names[role]; ok {
		return n
	}
	return role
}

func isSchoolAdmin(_ interface {
	Value(key interface{}) interface{}
}, _ string) (string, bool) {
	return "", false
}

var _ = fmt.Sprintf
var _ = isSchoolAdmin
var _ = formatRoleName
