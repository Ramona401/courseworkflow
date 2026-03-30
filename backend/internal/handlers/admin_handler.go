package handlers

/*
 * AdminHandler — 统一用户管理中心处理器
 *
 * 路由前缀：/api/v1/admin/
 *
 * 接口列表：
 *   GET  /api/v1/admin/users                      — 用户列表（含教研组/学校归属，分页+筛选）
 *   POST /api/v1/admin/users                      — 新建用户
 *   GET  /api/v1/admin/users/{id}                 — 用户详情（含跨系统权限全貌）
 *   PUT  /api/v1/admin/users/{id}                 — 编辑用户（角色+显示名）
 *   PUT  /api/v1/admin/users/{id}/status          — 启用/禁用
 *   PUT  /api/v1/admin/users/{id}/password        — 重置密码（admin直接重置）
 *   GET  /api/v1/admin/users/{id}/assignments     — 获取课程分配
 *   PUT  /api/v1/admin/users/{id}/assignments     — 更新课程分配
 *   POST /api/v1/admin/users/{id}/groups          — 将用户加入教研组（v52任务六新增）
 *   DELETE /api/v1/admin/users/{id}/groups/{gid}  — 将用户移出教研组（v52任务六新增，组长不可移除）
 *   GET  /api/v1/admin/orgs                       — 组织列表（区域+学校）
 *   GET  /api/v1/admin/groups                     — 教研组列表（含成员数）
 *   GET  /api/v1/admin/groups/{id}/members        — 教研组成员列表
 *   POST /api/v1/admin/groups/{id}/members        — 添加教研组成员
 *   PUT  /api/v1/admin/groups/{id}/members/{uid}  — 更新成员角色
 *   DELETE /api/v1/admin/groups/{id}/members/{uid}— 移除成员
 *   GET  /api/v1/admin/audit-logs                 — 操作日志（分页+筛选）
 *
 * 权限：
 *   admin → 所有操作
 */

import (
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

// AdminHandler 统一用户管理中心处理器
type AdminHandler struct {
	userService *services.UserService
	orgService  *services.OrganizationService
}

// NewAdminHandler 创建统一用户管理处理器
func NewAdminHandler(userService *services.UserService, orgService *services.OrganizationService) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// ==================== 权限判断辅助 ====================

// isSchoolAdmin 判断当前用户是否是某学校的管理员（保留占位）
func isSchoolAdmin(ctx interface{ Value(key interface{}) interface{} }, userID string) (string, bool) {
	return "", false
}

// AdminUserListItem 用户管理列表项（跨表联合，包含完整权限信息）
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

// AdminUserDetail 用户详情（含跨系统完整权限）
type AdminUserDetail struct {
	AdminUserListItem
	CourseAssignments []*models.CourseAssignment `json:"course_assignments"`
	TeachingGroups    []AdminGroupMembership     `json:"teaching_groups"`
}

// AdminGroupMembership 用户的教研组归属信息
type AdminGroupMembership struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	SchoolName string `json:"school_name"`
	Role       string `json:"role"`
	RoleName   string `json:"role_name"`
	JoinedAt   string `json:"joined_at"`
	IsLead     bool   `json:"is_lead"`
}

// ==================== 用户列表 ====================

// ListAdminUsers 获取用户列表（含跨表权限摘要）
// GET /api/v1/admin/users
func (h *AdminHandler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	roleFilter := q.Get("role")
	statusFilter := q.Get("status")
	keyword := q.Get("keyword")
	schoolID := q.Get("school_id")
	groupID := q.Get("group_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := repository.ListAdminUsers(r.Context(), repository.AdminUserListParams{
		Page:     page,
		PageSize: pageSize,
		Role:     roleFilter,
		Status:   statusFilter,
		Keyword:  keyword,
		SchoolID: schoolID,
		GroupID:  groupID,
	})
	if err != nil {
		utils.InternalError(w, "获取用户列表失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 用户详情 ====================

// GetAdminUserDetail 获取用户详情（含跨系统权限全貌）
// GET /api/v1/admin/users/{id}
func (h *AdminHandler) GetAdminUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	userID := extractAdminPathID(r.URL.Path, "/api/v1/admin/users/")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
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

// CreateAdminUser 新建用户
// POST /api/v1/admin/users
func (h *AdminHandler) CreateAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	userInfo, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}

	claims, _ := middleware.GetClaims(r.Context())
	if claims != nil {
		repository.WriteAuditLog(claims.UserID, "admin.user_create",
			map[string]interface{}{
				"target_user": userInfo.ID,
				"username":    userInfo.Username,
				"role":        userInfo.Role,
			}, repository.GetClientIP(r.RemoteAddr))
	}

	utils.Success(w, userInfo)
}

// ==================== 编辑用户 ====================

// UpdateAdminUser 编辑用户（角色+显示名）
// PUT /api/v1/admin/users/{id}
func (h *AdminHandler) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	userID := extractAdminPathID(r.URL.Path, "/api/v1/admin/users/")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	userInfo, err := h.userService.UpdateUser(r.Context(), userID, claims.UserID, &req)
	if err != nil {
		handleAdminUserError(w, err)
		return
	}

	utils.Success(w, userInfo)
}

// ==================== 启用/禁用 ====================

// UpdateAdminUserStatus 启用/禁用用户
// PUT /api/v1/admin/users/{id}/status
func (h *AdminHandler) UpdateAdminUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	userID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/users/", "/status")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.userService.UpdateStatus(r.Context(), userID, claims.UserID, &req); err != nil {
		handleAdminUserError(w, err)
		return
	}

	repository.WriteAuditLog(claims.UserID, "admin.user_status",
		map[string]interface{}{
			"target_user": userID,
			"new_status":  req.Status,
		}, repository.GetClientIP(r.RemoteAddr))

	utils.Success(w, map[string]string{"message": "用户状态更新成功"})
}

// ==================== 重置密码 ====================

// ResetAdminUserPassword admin重置用户密码
// PUT /api/v1/admin/users/{id}/password
func (h *AdminHandler) ResetAdminUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	userID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/users/", "/password")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.userService.ResetPassword(r.Context(), userID, &req); err != nil {
		handleAdminUserError(w, err)
		return
	}

	claims, _ := middleware.GetClaims(r.Context())
	if claims != nil {
		repository.WriteAuditLog(claims.UserID, "admin.user_reset_password",
			map[string]interface{}{"target_user": userID},
			repository.GetClientIP(r.RemoteAddr))
	}

	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 课程分配 ====================

// GetAdminUserAssignments 获取用户课程分配
// GET /api/v1/admin/users/{id}/assignments
func (h *AdminHandler) GetAdminUserAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/users/", "/assignments")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
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

// UpdateAdminUserAssignments 更新用户课程分配
// PUT /api/v1/admin/users/{id}/assignments
func (h *AdminHandler) UpdateAdminUserAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	userID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/users/", "/assignments")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req models.UpdateAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	result, err := h.userService.UpdateAssignments(r.Context(), userID, claims.UserID, &req)
	if err != nil {
		utils.InternalError(w, "更新课程分配失败")
		return
	}

	utils.Success(w, result)
}

// ==================== 用户↔教研组双向分配（v52任务六新增）====================

// AddUserToGroup 将用户加入指定教研组
//
// POST /api/v1/admin/users/{uid}/groups
// Body: {"group_id": "...", "role": "member|backbone"}
//
// 业务规则：
//   - 若用户已在该教研组，服务层返回错误
//   - role 默认为 member
func (h *AdminHandler) AddUserToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从 URL 路径提取用户ID：/api/v1/admin/users/{uid}/groups
	userID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/users/", "/groups")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	var body struct {
		GroupID string `json:"group_id"`
		Role    string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if body.GroupID == "" {
		utils.BadRequest(w, "请选择教研组")
		return
	}
	// role 默认值
	if body.Role != "member" && body.Role != "backbone" {
		body.Role = "member"
	}

	// 构造 AddGroupMemberRequest（UserID=路径中的用户，加入 body.GroupID）
	req := &models.AddGroupMemberRequest{
		UserID: userID,
		Role:   body.Role,
	}
	if err := h.orgService.AddGroupMember(r.Context(), body.GroupID, req); err != nil {
		utils.InternalError(w, "加入教研组失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]string{"message": "已成功加入教研组"})
}

// RemoveUserFromGroup 将用户移出指定教研组
//
// DELETE /api/v1/admin/users/{uid}/groups/{gid}
//
// 业务规则：
//   - 若该用户是该教研组的组长（teaching_groups.lead_user_id = uid），拒绝移除并返回 400
//   - 非组长成员可正常移除
func (h *AdminHandler) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	// 从路径提取 uid 和 gid：/api/v1/admin/users/{uid}/groups/{gid}
	userID, groupID := extractUserGroupPath(r.URL.Path)
	if userID == "" || groupID == "" {
		utils.BadRequest(w, "缺少用户ID或教研组ID")
		return
	}

	// 检查是否是教研组长（组长不可通过此接口移除）
	isLead, err := repository.IsGroupLead(r.Context(), groupID, userID)
	if err != nil {
		utils.InternalError(w, "权限检查失败: "+err.Error())
		return
	}
	if isLead {
		utils.BadRequest(w, "教研组长不能被移除，请先在教研组管理中更换组长后再操作")
		return
	}

	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		utils.InternalError(w, "移出教研组失败: "+err.Error())
		return
	}

	utils.Success(w, map[string]string{"message": "已移出教研组"})
}

// ==================== 组织列表 ====================

// ListAdminOrgs 获取组织列表（区域+学校树形结构）
// GET /api/v1/admin/orgs?type=school
func (h *AdminHandler) ListAdminOrgs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	orgType := r.URL.Query().Get("type")
	parentID := r.URL.Query().Get("parent_id")

	result, err := h.orgService.ListOrganizations(r.Context(), orgType, parentID)
	if err != nil {
		utils.InternalError(w, "获取组织列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 教研组列表 ====================

// ListAdminGroups 获取所有教研组列表
// GET /api/v1/admin/groups?school_id=xxx
func (h *AdminHandler) ListAdminGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	schoolID := r.URL.Query().Get("school_id")
	result, err := h.orgService.ListTeachingGroups(r.Context(), schoolID)
	if err != nil {
		utils.InternalError(w, "获取教研组列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 教研组成员管理 ====================

// ListAdminGroupMembers 获取教研组成员列表
// GET /api/v1/admin/groups/{id}/members
func (h *AdminHandler) ListAdminGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	groupID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/groups/", "/members")
	if groupID == "" {
		utils.BadRequest(w, "缺少教研组ID")
		return
	}
	members, err := repository.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		utils.InternalError(w, "获取成员列表失败")
		return
	}
	if members == nil {
		members = []*models.GroupMemberItem{}
	}
	utils.Success(w, members)
}

// AddAdminGroupMember 添加教研组成员
// POST /api/v1/admin/groups/{id}/members
func (h *AdminHandler) AddAdminGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	groupID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/groups/", "/members")
	if groupID == "" {
		utils.BadRequest(w, "缺少教研组ID")
		return
	}

	var req models.AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.orgService.AddGroupMember(r.Context(), groupID, &req); err != nil {
		utils.InternalError(w, "添加成员失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

// UpdateAdminGroupMemberRole 更新教研组成员角色
// PUT /api/v1/admin/groups/{id}/members/{uid}
func (h *AdminHandler) UpdateAdminGroupMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	groupID, userID := extractAdminGroupMemberPath(r.URL.Path)
	if groupID == "" || userID == "" {
		utils.BadRequest(w, "缺少教研组ID或用户ID")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Role != "member" && req.Role != "backbone" {
		utils.BadRequest(w, "角色只能是 member 或 backbone")
		return
	}

	if err := repository.UpdateGroupMemberRole(r.Context(), groupID, userID, req.Role); err != nil {
		utils.InternalError(w, "更新成员角色失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "角色更新成功"})
}

// RemoveAdminGroupMember 移除教研组成员
// DELETE /api/v1/admin/groups/{id}/members/{uid}
func (h *AdminHandler) RemoveAdminGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	groupID, userID := extractAdminGroupMemberPath(r.URL.Path)
	if groupID == "" || userID == "" {
		utils.BadRequest(w, "缺少教研组ID或用户ID")
		return
	}

	if err := repository.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		utils.InternalError(w, "移除成员失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "移除成功"})
}

// ==================== 操作日志 ====================

// ListAdminAuditLogs 查询操作日志（分页+多维筛选）
// GET /api/v1/admin/audit-logs
func (h *AdminHandler) ListAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	userID := q.Get("user_id")
	username := q.Get("username")
	action := q.Get("action")
	startDate := q.Get("start_date")
	endDate := q.Get("end_date")

	result, err := repository.ListAuditLogs(r.Context(), repository.AuditLogQueryParams{
		UserID:    userID,
		Username:  username,
		Action:    action,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		utils.InternalError(w, "查询操作日志失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 统计摘要 ====================

// GetAdminStats 用户管理统计（用于概览卡片）
// GET /api/v1/admin/stats
func (h *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	stats, err := repository.GetAdminStats(r.Context())
	if err != nil {
		utils.InternalError(w, "获取统计失败: "+err.Error())
		return
	}
	utils.Success(w, stats)
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

// ==================== 路径提取工具 ====================

// extractAdminPathID 提取末尾ID
// /api/v1/admin/users/xxx-yyy → xxx-yyy
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

// extractAdminMiddleID 提取中间ID
// /api/v1/admin/users/xxx-yyy/status → xxx-yyy
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

// extractAdminGroupMemberPath 从 /api/v1/admin/groups/{gid}/members/{uid} 提取 gid 和 uid
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

// extractUserGroupPath 从 /api/v1/admin/users/{uid}/groups/{gid} 提取 uid 和 gid
// 用于用户↔教研组双向分配接口
func extractUserGroupPath(path string) (string, string) {
	prefix := "/api/v1/admin/users/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, prefix)
	// rest = "{uid}/groups/{gid}"
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

// formatRoleName 角色中文名
func formatRoleName(role string) string {
	names := map[string]string{
		"admin":           "管理员",
		"senior_operator": "高级操作员",
		"operator":        "操作员",
		"viewer":          "查看者",
	}
	if n, ok := names[role]; ok {
		return n
	}
	return role
}

// 抑制未使用警告
var _ = fmt.Sprintf
var _ = isSchoolAdmin
