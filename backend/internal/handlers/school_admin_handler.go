package handlers

// school_admin_handler.go — 学校管理员专属处理器
//
// 设计目标：
// 1) senior_operator 作为学校管理员
// 2) 仅当该用户是 organizations.admin_user_id 且 type=school 时，才可使用本模块
// 3) 学校管理员仅可创建低级账号：operator / viewer
// 4) 教师创建后可手动分配到教研组
// 5) 学校管理员可查看本校全部教师（users.school_id = 本校ID）

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	schoolAdminUsersPrefix  = "/api/v1/school-admin/users/"
	schoolAdminGroupsPrefix = "/api/v1/school-admin/groups/"
)

// SchoolAdminHandler 学校管理员处理器
type SchoolAdminHandler struct {
	userService *services.UserService
	orgService  *services.OrganizationService
}

// NewSchoolAdminHandler 构造函数
func NewSchoolAdminHandler(userService *services.UserService, orgService *services.OrganizationService) *SchoolAdminHandler {
	return &SchoolAdminHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// getCurrentSchoolAdminContext 获取当前学校管理员上下文（用户ID+学校）
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

// GetMySchool GET /api/v1/school-admin/my-school
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

// ==================== 学校教师管理 ====================

// ListSchoolUsers GET /api/v1/school-admin/users
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
	users, err := repository.ListUsersBySchool(r.Context(), school.ID)
	if err != nil {
		utils.InternalError(w, "获取学校教师列表失败: "+err.Error())
		return
	}
	resp := make([]*models.UserInfo, 0, len(users))
	for _, u := range users {
		resp = append(resp, u.ToUserInfo())
	}
	utils.Success(w, &models.UserListResponse{Users: resp, Total: len(resp)})
}

// CreateSchoolUser POST /api/v1/school-admin/users
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

	// 学校管理员只能创建比自己低的角色
	if !models.IsSchoolAdminCreatableRole(req.Role) {
		utils.BadRequest(w, "学校管理员仅可创建 operator / viewer 账号")
		return
	}

	req.SchoolID = &school.ID
	userInfo, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		handleSchoolAdminUserError(w, err)
		return
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
	userID := extractSchoolAdminPathID(r.URL.Path, schoolAdminUsersPrefix)
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验教师归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "仅可查看本校教师")
		return
	}

	detail, err := repository.GetAdminUserDetail(r.Context(), userID)
	if err != nil {
		utils.InternalError(w, "获取教师详情失败: "+err.Error())
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
	userID := extractSchoolAdminPathID(r.URL.Path, schoolAdminUsersPrefix)
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验教师归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "仅可编辑本校教师")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if !models.IsSchoolAdminCreatableRole(req.Role) {
		utils.BadRequest(w, "学校管理员仅可设置 operator / viewer 角色")
		return
	}

	// 强制归属当前学校
	req.SchoolID = &school.ID

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
	userID := extractSchoolAdminMiddleID(r.URL.Path, schoolAdminUsersPrefix, "/status")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验教师归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "仅可操作本校教师")
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
	userID := extractSchoolAdminMiddleID(r.URL.Path, schoolAdminUsersPrefix, "/password")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}

	inSchool, err := repository.IsUserInSchool(r.Context(), userID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验教师归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "仅可操作本校教师")
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

// ListSchoolGroups GET /api/v1/school-admin/groups
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

// CreateSchoolGroup POST /api/v1/school-admin/groups
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

// UpdateSchoolGroup PUT /api/v1/school-admin/groups/{id}
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
	groupID := extractSchoolAdminPathID(r.URL.Path, schoolAdminGroupsPrefix)
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
		utils.Forbidden(w, "仅可管理本校教研组")
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

// DeleteSchoolGroup DELETE /api/v1/school-admin/groups/{id}
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
	groupID := extractSchoolAdminPathID(r.URL.Path, schoolAdminGroupsPrefix)
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
		utils.Forbidden(w, "仅可管理本校教研组")
		return
	}
	if err := h.orgService.DeleteTeachingGroup(r.Context(), groupID); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ListSchoolGroupMembers GET /api/v1/school-admin/groups/{id}/members
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
	groupID := extractSchoolAdminMiddleID(r.URL.Path, schoolAdminGroupsPrefix, "/members")
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
		utils.Forbidden(w, "仅可管理本校教研组")
		return
	}
	members, err := repository.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		utils.InternalError(w, "获取成员失败: "+err.Error())
		return
	}
	utils.Success(w, members)
}

// AddSchoolGroupMember POST /api/v1/school-admin/groups/{id}/members
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
	groupID := extractSchoolAdminMiddleID(r.URL.Path, schoolAdminGroupsPrefix, "/members")
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
		utils.Forbidden(w, "仅可管理本校教研组")
		return
	}

	var req models.AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	// 仅允许添加本校教师
	inSchool, err := repository.IsUserInSchool(r.Context(), req.UserID, school.ID)
	if err != nil {
		utils.InternalError(w, "校验教师归属失败: "+err.Error())
		return
	}
	if !inSchool {
		utils.Forbidden(w, "仅可添加本校教师")
		return
	}

	if err := h.orgService.AddGroupMember(r.Context(), groupID, &req); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

// UpdateSchoolGroupMemberRole PUT /api/v1/school-admin/groups/{id}/members/{uid}
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
		utils.Forbidden(w, "仅可管理本校教研组")
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

// RemoveSchoolGroupMember DELETE /api/v1/school-admin/groups/{id}/members/{uid}
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
		utils.Forbidden(w, "仅可管理本校教研组")
		return
	}
	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		handleSchoolAdminOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "移除成功"})
}

// ==================== 错误映射 ====================

func handleSchoolAdminUserError(w http.ResponseWriter, err error) {
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
	case errors.Is(err, services.ErrGroupNotFound),
		errors.Is(err, services.ErrMemberNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, "组织操作失败: "+err.Error())
	}
}

// ==================== 路径解析辅助 ====================

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

// extractSchoolAdminGroupMemberPath 解析 /api/v1/school-admin/groups/{gid}/members/{uid}
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
