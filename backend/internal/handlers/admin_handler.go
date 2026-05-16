package handlers

// admin_handler.go — 统一用户管理中心处理器(主文件)
//
// v122 方案B 改动(修复: 学校管理员新建老师看不见):
//   - CreateAdminUser: senior_operator 创建成功后自动调 AddSchoolMember 写入 school_members
//   - ensureUserInScope: 从 IsUserInSchoolByGroup 换为 IsUserInSchool (school_members 主判 + 教研组兜底)
//   - resolveSchoolScope: 逻辑不变(仍用 GetSchoolByAdminUserID 反查学校)
//
// v122 原改动(AdminPage 权限统一):
//   - resolveSchoolScope: 根据登录者角色决定数据范围
//     * admin 可看全系统,传入的 school_id 照常使用
//     * senior_operator 强制过滤为自己管理的学校,忽略前端传的 school_id
//   - ListAdminUsers / GetAdminUserDetail 调用 resolveSchoolScope 过滤数据
//   - CreateAdminUser / UpdateAdminUser 校验操作者角色:
//     * senior_operator 不能创建/编辑 role=admin 或 senior_operator 的用户
//
// 职责:
//   - AdminHandler struct定义与构造
//   - 用户列表/详情/创建/编辑/启用禁用/重置密码
//   - 课程分配查询与更新
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

// resolveSchoolScope 根据登录者角色决定数据范围
// 返回 (effectiveSchoolID, userRole, error)
//   - admin: 返回前端传的 schoolID(可以为空 → 全系统)
//   - senior_operator: 强制返回其管理的学校 ID(忽略前端传入),未绑定学校则返回错误
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

// ==================== 用户列表 ====================

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

	effectiveSchoolID, _, scopeErr := resolveSchoolScope(r.Context(), q.Get("school_id"))
	if scopeErr != nil {
		utils.Forbidden(w, scopeErr.Error())
		return
	}

	result, err := repository.ListAdminUsers(r.Context(), repository.AdminUserListParams{
		Page:     page,
		PageSize: pageSize,
		Role:     q.Get("role"),
		Status:   q.Get("status"),
		Keyword:  q.Get("keyword"),
		SchoolID: effectiveSchoolID,
		GroupID:  q.Get("group_id"),
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
// v122 方案B: senior_operator 创建成功后自动调 AddSchoolMember 写入 school_members
// 修复: 学校管理员新建老师后在列表看不见的 bug
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

	// v122: senior_operator 角色白名单校验
	if claims.Role == models.RoleSeniorOperator {
		if req.Role == models.RoleAdmin {
			utils.Forbidden(w, "学校管理员不能创建系统管理员账号")
			return
		}
		if req.Role == models.RoleSeniorOperator {
			utils.Forbidden(w, "学校管理员不能创建其他学校管理员账号")
			return
		}
	}

	// v122 方案B: senior_operator 创建前,先反查其管理的学校(供创建成功后自动入校用)
	// admin 创建用户不自动入校(admin 可能为任意学校创建,意图不明确,交给后续手动或教研组绑定)
	var targetSchoolID string
	if claims.Role == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
		if err != nil || school == nil || school.ID == "" {
			utils.Forbidden(w, "您尚未绑定学校,无法创建本校用户")
			return
		}
		targetSchoolID = school.ID
	}

	// 调 service 创建用户
	userInfo, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}

	// v122 方案B: 如果是 senior_operator 创建,自动写入 school_members
	// 失败时不阻断响应(用户已创建成功),但写 warning 日志
	if targetSchoolID != "" {
		if addErr := repository.AddSchoolMember(r.Context(), targetSchoolID, userInfo.ID, "admin_create"); addErr != nil {
			// 降级处理:不阻断,避免用户已建但响应失败造成二次创建
			fmt.Printf("[WARN] admin_handler.CreateAdminUser: 写入 school_members 失败 user_id=%s school_id=%s err=%v\n",
				userInfo.ID, targetSchoolID, addErr)
		}
	}

	repository.WriteAuditLog(claims.UserID, "admin.user_create",
		map[string]interface{}{
			"target_user": userInfo.ID,
			"username":    userInfo.Username,
			"role":        userInfo.Role,
			"school_id":   targetSchoolID, // v122 方案B: 记录入校学校
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, userInfo)
}

// ==================== 编辑用户 ====================

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
		if req.Role == models.RoleAdmin || req.Role == models.RoleSeniorOperator {
			utils.Forbidden(w, "学校管理员不能授予 admin 或 senior_operator 角色")
			return
		}
	}

	userInfo, err := h.userService.UpdateUser(r.Context(), userID, claims.UserID, &req)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}
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

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
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

	if err := ensureUserInScope(r.Context(), userID); err != nil {
		utils.Forbidden(w, err.Error())
		return
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
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, "admin.user_reset_password",
			map[string]interface{}{"target_user": userID},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 课程分配 ====================

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
//   (school_members 主判 + teaching_group_members 兜底)
// - admin: 总是允许
// - senior_operator: 目标用户必须属于其管理的学校(通过 school_members 或 教研组)
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

	return fmt.Errorf("权限不足")
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
		services.ErrCannotChangeOwnRole:
		utils.BadRequest(w, err.Error())
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

func isSchoolAdmin(_ interface{ Value(key interface{}) interface{} }, _ string) (string, bool) {
	return "", false
}

var _ = fmt.Sprintf
var _ = isSchoolAdmin
var _ = formatRoleName
