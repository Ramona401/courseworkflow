package handlers

// admin_handler.go — 统一用户管理中心处理器（主文件）
//
// 职责：
//   - AdminHandler struct定义与构造
//   - 用户列表/详情/创建/编辑/启用禁用/重置密码
//   - 课程分配查询与更新
//   - 统计摘要
//   - 错误处理函数
//   - 路径提取工具函数
//
// 教研组成员管理 → admin_handler_groups.go
// 组织列表+日志查询 → admin_handler_logs.go

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

// ==================== Handler结构体 ====================

// AdminHandler 统一用户管理中心处理器
type AdminHandler struct {
	userService *services.UserService
	orgService  *services.OrganizationService
}

// NewAdminHandler 构造函数
func NewAdminHandler(userService *services.UserService, orgService *services.OrganizationService) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// ==================== 本地类型定义 ====================

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

// ==================== 统计摘要 ====================

// GetAdminStats GET /api/v1/admin/stats
// 用户管理统计（用于概览卡片：总用户/组织/教研组）
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

// ==================== 用户列表 ====================

// ListAdminUsers GET /api/v1/admin/users
// 获取用户列表（含跨表权限摘要，支持role/status/keyword/school_id/group_id多维筛选+分页）
func (h *AdminHandler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
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

	result, err := repository.ListAdminUsers(r.Context(), repository.AdminUserListParams{
		Page:     page,
		PageSize: pageSize,
		Role:     q.Get("role"),
		Status:   q.Get("status"),
		Keyword:  q.Get("keyword"),
		SchoolID: q.Get("school_id"),
		GroupID:  q.Get("group_id"),
	})
	if err != nil {
		utils.InternalError(w, "获取用户列表失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== 用户详情 ====================

// GetAdminUserDetail GET /api/v1/admin/users/{id}
// 获取用户详情（含课程分配+教研组归属的跨系统权限全貌）
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

// CreateAdminUser POST /api/v1/admin/users
// 新建用户，写入审计日志
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
	// 写入审计日志
	if claims, ok := middleware.GetClaims(r.Context()); ok {
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

// UpdateAdminUser PUT /api/v1/admin/users/{id}
// 编辑用户角色和显示名
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

// UpdateAdminUserStatus PUT /api/v1/admin/users/{id}/status
// 启用或禁用用户账户，写入审计日志
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
		map[string]interface{}{"target_user": userID, "new_status": req.Status},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "用户状态更新成功"})
}

// ==================== 重置密码 ====================

// ResetAdminUserPassword PUT /api/v1/admin/users/{id}/password
// admin直接重置用户密码（无需旧密码），写入审计日志
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
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, "admin.user_reset_password",
			map[string]interface{}{"target_user": userID},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 课程分配 ====================

// GetAdminUserAssignments GET /api/v1/admin/users/{id}/assignments
// 获取用户的课程审核权限分配列表
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

// UpdateAdminUserAssignments PUT /api/v1/admin/users/{id}/assignments
// 更新用户的课程审核权限分配（全量替换）
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

// ==================== 错误处理 ====================

// handleAdminUserError 将服务层用户操作错误映射到HTTP状态码
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

// extractAdminPathID 提取路径末尾ID
// 例：/api/v1/admin/users/abc-123 → "abc-123"
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

// extractAdminMiddleID 提取路径中间ID
// 例：/api/v1/admin/users/abc-123/status → "abc-123"（prefix="/api/v1/admin/users/", suffix="/status"）
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

// extractAdminGroupMemberPath 从教研组成员路径提取 groupID 和 userID
// 例：/api/v1/admin/groups/gid123/members/uid456 → "gid123", "uid456"
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

// extractUserGroupPath 从用户↔教研组路径提取 userID 和 groupID
// 例：/api/v1/admin/users/uid123/groups/gid456 → "uid123", "gid456"
func extractUserGroupPath(path string) (string, string) {
	prefix := "/api/v1/admin/users/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, prefix)
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

// formatRoleName 角色中文名映射
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

// isSchoolAdmin 判断当前用户是否是某学校的管理员（保留占位）
func isSchoolAdmin(_ interface{ Value(key interface{}) interface{} }, _ string) (string, bool) {
	return "", false
}

// 抑制编译器未使用警告
var _ = fmt.Sprintf
var _ = isSchoolAdmin
var _ = formatRoleName
