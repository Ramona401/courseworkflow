package handlers

// school_admin_handler.go — 学校管理员专属处理器
//
// v122 方案B 改动(修复: 学校管理员新建老师看不见):
//   - CreateSchoolUser: 创建成功后自动 AddSchoolMember,写入 school_members
//   - 所有 IsUserInSchoolByGroup 换为 IsUserInSchool
//     (school_members 主判 + teaching_group_members 兜底)
//
// v110 原设计:
//   - senior_operator 作为学校管理员
//   - 仅当 organizations.admin_user_id = 当前用户 时才可使用
//   - 不依赖 users.school_id 字段,完全基于现有组织体系

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// SchoolAdminHandler 学校管理员处理器
type SchoolAdminHandler struct {
	userService *services.UserService
	orgService  *services.OrganizationService
}

func NewSchoolAdminHandler(userService *services.UserService, orgService *services.OrganizationService) *SchoolAdminHandler {
	return &SchoolAdminHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// getCurrentSchoolAdminContext 获取当前学校管理员的用户ID和所管理的学校
func getCurrentSchoolAdminContext(r *http.Request) (string, *models.Organization, error) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		return "", nil, errors.New(utils.MsgUnauthorized)
	}
	if claims.Role != models.RoleSeniorOperator {
		return "", nil, errors.New("仅学校管理员可访问")
	}
	school, err := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
	if err != nil {
		return "", nil, errors.New("当前账号未绑定学校管理员身份")
	}
	return claims.UserID, school, nil
}

// ==================== 学校信息 ====================

func (h *SchoolAdminHandler) GetMySchool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"id":      school.ID,
		"name":    school.Name,
		"type":    school.Type,
		"status":  school.Status,
		"created": school.CreatedAt,
	})
}

// ==================== 学校用户管理 ====================

// ListSchoolUsers GET /api/v1/school-admin/users
// v122 方案B: 通过 school_members (∪ teaching_group_members 兜底) 查本校用户
func (h *SchoolAdminHandler) ListSchoolUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	result, err := repository.ListAdminUsers(r.Context(), repository.AdminUserListParams{
		Page:     1,
		PageSize: 500,
		SchoolID: school.ID,
	})
	if err != nil {
		utils.InternalError(w, "获取学校用户列表失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// CreateSchoolUser POST /api/v1/school-admin/users
// v122 方案B: 创建成功后自动写入 school_members
func (h *SchoolAdminHandler) CreateSchoolUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	adminUserID, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if !models.IsSchoolAdminCreatableRole(req.Role) {
		utils.BadRequest(w, "学校管理员仅可创建骨干教师(operator)或普通教师(viewer)账号")
		return
	}

	userInfo, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		handleSchoolAdminUserError(w, err)
		return
	}

	// v122 方案B: 创建成功后自动写入 school_members
	// 失败时不阻断响应(用户已创建),但写 warning 日志
	if addErr := repository.AddSchoolMember(r.Context(), school.ID, userInfo.ID, "school_admin_create"); addErr != nil {
		fmt.Printf("[WARN] school_admin_handler.CreateSchoolUser: 写入 school_members 失败 user_id=%s school_id=%s err=%v\n",
			userInfo.ID, school.ID, addErr)
	}

	repository.WriteAuditLog(adminUserID, "school_admin.user_create",
		map[string]interface{}{
			"target_user_id": userInfo.ID,
			"username":       userInfo.Username,
			"role":           userInfo.Role,
			"school_id":      school.ID,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, userInfo)
}

// GetSchoolUserDetail GET /api/v1/school-admin/users/{id}
func (h *SchoolAdminHandler) GetSchoolUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	userID := extractSchoolAdminPathID(r.URL.Path, "/api/v1/school-admin/users/")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	// v122 方案B: 换用 IsUserInSchool
	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验用户归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "只能查看本校用户")
		return
	}

	detail, err := repository.GetAdminUserDetail(r.Context(), userID)
	if err != nil {
		utils.InternalError(w, "获取用户详情失败: "+err.Error())
		return
	}
	utils.Success(w, detail)
}

// UpdateSchoolUser PUT /api/v1/school-admin/users/{id}
func (h *SchoolAdminHandler) UpdateSchoolUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	adminUserID, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	userID := extractSchoolAdminPathID(r.URL.Path, "/api/v1/school-admin/users/")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	// v122 方案B
	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验用户归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "只能管理本校用户")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if !models.IsSchoolAdminCreatableRole(req.Role) {
		utils.BadRequest(w, "学校管理员仅可设置骨干教师或普通教师角色")
		return
	}

	userInfo, err := h.userService.UpdateUser(r.Context(), userID, adminUserID, &req)
	if err != nil {
		handleSchoolAdminUserError(w, err)
		return
	}
	utils.Success(w, userInfo)
}

// UpdateSchoolUserStatus PUT /api/v1/school-admin/users/{id}/status
func (h *SchoolAdminHandler) UpdateSchoolUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	adminUserID, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	userID := extractSchoolAdminMiddleID(r.URL.Path, "/api/v1/school-admin/users/", "/status")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	// v122 方案B
	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验用户归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "只能管理本校用户")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.UpdateStatus(r.Context(), userID, adminUserID, &req); err != nil {
		handleSchoolAdminUserError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "状态更新成功"})
}

// ResetSchoolUserPassword PUT /api/v1/school-admin/users/{id}/password
func (h *SchoolAdminHandler) ResetSchoolUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	userID := extractSchoolAdminMiddleID(r.URL.Path, "/api/v1/school-admin/users/", "/password")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	// v122 方案B
	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验用户归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "只能管理本校用户")
		return
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.ResetPassword(r.Context(), userID, &req); err != nil {
		handleSchoolAdminUserError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 学校教研组管理 ====================

func (h *SchoolAdminHandler) ListSchoolGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	result, err := h.orgService.ListTeachingGroups(r.Context(), school.ID)
	if err != nil {
		utils.InternalError(w, "获取教研组列表失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *SchoolAdminHandler) CreateSchoolGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	var req models.CreateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	req.SchoolID = school.ID

	group, err := h.orgService.CreateTeachingGroup(r.Context(), &req)
	if err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, group)
}

func (h *SchoolAdminHandler) UpdateSchoolGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID := extractSchoolAdminPathID(r.URL.Path, "/api/v1/school-admin/groups/")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}

	var req models.UpdateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.orgService.UpdateTeachingGroup(r.Context(), groupID, &req); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

func (h *SchoolAdminHandler) DeleteSchoolGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID := extractSchoolAdminPathID(r.URL.Path, "/api/v1/school-admin/groups/")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}
	if err := h.orgService.DeleteTeachingGroup(r.Context(), groupID); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

func (h *SchoolAdminHandler) ListSchoolGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID := extractSchoolAdminMiddleID(r.URL.Path, "/api/v1/school-admin/groups/", "/members")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}

	members, err := repository.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		utils.InternalError(w, "获取成员失败: "+err.Error())
		return
	}
	utils.Success(w, members)
}

func (h *SchoolAdminHandler) AddSchoolGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID := extractSchoolAdminMiddleID(r.URL.Path, "/api/v1/school-admin/groups/", "/members")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}

	var req models.AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.orgService.AddGroupMember(r.Context(), groupID, &req); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

func (h *SchoolAdminHandler) UpdateSchoolGroupMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID, userID := extractSchoolAdminGroupMemberPath(r.URL.Path)
	if groupID == "" || userID == "" {
		utils.BadRequest(w, "缺少教研组ID或成员ID")
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.orgService.UpdateGroupMemberRole(r.Context(), groupID, userID, req.Role); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "角色更新成功"})
}

func (h *SchoolAdminHandler) RemoveSchoolGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	_, school, err := getCurrentSchoolAdminContext(r)
	if err != nil {
		utils.Forbidden(w, err.Error())
		return
	}

	groupID, userID := extractSchoolAdminGroupMemberPath(r.URL.Path)
	if groupID == "" || userID == "" {
		utils.BadRequest(w, "缺少教研组ID或成员ID")
		return
	}

	group, err := repository.GetTeachingGroupByID(r.Context(), groupID)
	if err != nil {
		utils.Fail(w, http.StatusNotFound, "教研组不存在")
		return
	}
	if group.SchoolID != school.ID {
		utils.Forbidden(w, "只能管理本校教研组")
		return
	}
	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "移除成功"})
}

// ==================== 错误处理 ====================

func handleSchoolAdminUserError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrUsernameRequired, services.ErrDisplayNameRequired,
		services.ErrPasswordTooShort, services.ErrInvalidRole,
		services.ErrInvalidStatus, services.ErrUsernameExists,
		services.ErrCannotDisableSelf, services.ErrCannotChangeOwnRole:
		utils.BadRequest(w, err.Error())
	case services.ErrUserNotFound:
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, "用户操作失败: "+err.Error())
	}
}

func handleSchoolAdminOrgError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrGroupNameRequired),
		errors.Is(err, services.ErrGroupSubjectRequired),
		errors.Is(err, services.ErrGroupNameExists),
		errors.Is(err, services.ErrMemberUserRequired),
		errors.Is(err, services.ErrMemberAlreadyExists):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrGroupNotFound), errors.Is(err, services.ErrMemberNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, "组织操作失败: "+err.Error())
	}
}

// ==================== 路径解析 ====================

func extractSchoolAdminPathID(path, prefix string) string {
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

func extractSchoolAdminMiddleID(path, prefix, suffix string) string {
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

func extractSchoolAdminGroupMemberPath(path string) (string, string) {
	prefix := "/api/v1/school-admin/groups/"
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
