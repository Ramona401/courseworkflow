package handlers

// 组织与教研组管理HTTP处理器
// 负责组织CRUD、教研组CRUD、成员管理的HTTP接口

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// OrganizationHandler 组织管理接口处理器
type OrganizationHandler struct {
	orgService *services.OrganizationService
}

// NewOrganizationHandler 创建组织管理处理器实例
func NewOrganizationHandler(orgService *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgService: orgService}
}

// ==================== 组织 CRUD ====================

// ListOrganizations 获取组织列表
// GET /api/v1/lesson-plans/organizations?type=school&parent_id=xxx
func (h *OrganizationHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	orgType := r.URL.Query().Get("type")
	parentID := r.URL.Query().Get("parent_id")

	result, err := h.orgService.ListOrganizations(r.Context(), orgType, parentID)
	if err != nil {
		log.Printf("获取组织列表失败: %v", err)
		utils.InternalError(w, "获取组织列表失败")
		return
	}
	utils.Success(w, result)
}

// CreateOrganization 创建组织
// POST /api/v1/lesson-plans/organizations
func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	var req models.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	org, err := h.orgService.CreateOrganization(r.Context(), &req)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, org)
}

// GetOrganization 获取组织详情
// GET /api/v1/lesson-plans/organizations/{id}
func (h *OrganizationHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/organizations/")
	if id == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}
	org, err := h.orgService.GetOrganization(r.Context(), id)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, org)
}

// UpdateOrganization 更新组织
// PUT /api/v1/lesson-plans/organizations/{id}
func (h *OrganizationHandler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/organizations/")
	if id == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}
	var req models.UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if err := h.orgService.UpdateOrganization(r.Context(), id, &req); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteOrganization 删除组织
// DELETE /api/v1/lesson-plans/organizations/{id}
func (h *OrganizationHandler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/organizations/")
	if id == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}
	if err := h.orgService.DeleteOrganization(r.Context(), id); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教研组 CRUD ====================

// ListTeachingGroups 获取教研组列表
// GET /api/v1/lesson-plans/teaching-groups?school_id=xxx
func (h *OrganizationHandler) ListTeachingGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	schoolID := r.URL.Query().Get("school_id")
	result, err := h.orgService.ListTeachingGroups(r.Context(), schoolID)
	if err != nil {
		log.Printf("获取教研组列表失败: %v", err)
		utils.InternalError(w, "获取教研组列表失败")
		return
	}
	utils.Success(w, result)
}

// CreateTeachingGroup 创建教研组
// POST /api/v1/lesson-plans/teaching-groups
func (h *OrganizationHandler) CreateTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	var req models.CreateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	tg, err := h.orgService.CreateTeachingGroup(r.Context(), &req)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, tg)
}

// GetTeachingGroupDetail 获取教研组详情
// GET /api/v1/lesson-plans/teaching-groups/{id}
func (h *OrganizationHandler) GetTeachingGroupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/teaching-groups/")
	if id == "" {
		utils.BadRequest(w, "缺少教研组ID")
		return
	}
	// 截断子路径（如 /members）
	if idx := strings.Index(id, "/"); idx > 0 {
		id = id[:idx]
	}
	detail, err := h.orgService.GetTeachingGroupDetail(r.Context(), id)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, detail)
}

// UpdateTeachingGroup 更新教研组
// PUT /api/v1/lesson-plans/teaching-groups/{id}
func (h *OrganizationHandler) UpdateTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/teaching-groups/")
	if id == "" {
		utils.BadRequest(w, "缺少教研组ID")
		return
	}
	var req models.UpdateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if err := h.orgService.UpdateTeachingGroup(r.Context(), id, &req); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteTeachingGroup 删除教研组
// DELETE /api/v1/lesson-plans/teaching-groups/{id}
func (h *OrganizationHandler) DeleteTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	id := extractIDFromPath(r.URL.Path, "/api/v1/lesson-plans/teaching-groups/")
	if id == "" {
		utils.BadRequest(w, "缺少教研组ID")
		return
	}
	if err := h.orgService.DeleteTeachingGroup(r.Context(), id); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教研组成员管理 ====================

// AddGroupMember 添加教研组成员
// POST /api/v1/lesson-plans/teaching-groups/{id}/members
func (h *OrganizationHandler) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	groupID := extractMiddleSegment(r.URL.Path, "/api/v1/lesson-plans/teaching-groups/", "/members")
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
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

// RemoveGroupMember 移除教研组成员
// DELETE /api/v1/lesson-plans/teaching-groups/{id}/members/{user_id}
func (h *OrganizationHandler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	// 从路径提取 groupID 和 userID
	path := r.URL.Path
	prefix := "/api/v1/lesson-plans/teaching-groups/"
	if !strings.HasPrefix(path, prefix) {
		utils.BadRequest(w, "路径格式错误")
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/members/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.BadRequest(w, "缺少教研组ID或成员ID")
		return
	}
	groupID := parts[0]
	userID := strings.TrimSuffix(parts[1], "/")

	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "移除成功"})
}

// GetUserTeachingGroups 获取当前用户所属教研组列表
// GET /api/v1/lesson-plans/my-groups
func (h *OrganizationHandler) GetUserTeachingGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}
	groups, err := h.orgService.GetUserTeachingGroups(r.Context(), userID)
	if err != nil {
		log.Printf("获取用户教研组失败: %v", err)
		utils.InternalError(w, "获取教研组失败")
		return
	}
	if groups == nil {
		groups = []*models.TeachingGroupListItem{}
	}
	utils.Success(w, groups)
}

// ==================== 错误处理 ====================

func (h *OrganizationHandler) handleOrgError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrOrgNameRequired),
		errors.Is(err, services.ErrOrgTypeRequired),
		errors.Is(err, services.ErrOrgTypeInvalid),
		errors.Is(err, services.ErrSchoolNeedsParent),
		errors.Is(err, services.ErrGroupNameRequired),
		errors.Is(err, services.ErrGroupSchoolRequired),
		errors.Is(err, services.ErrGroupSubjectRequired),
		errors.Is(err, services.ErrMemberUserRequired):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgNameExists),
		errors.Is(err, services.ErrGroupNameExists),
		errors.Is(err, services.ErrMemberAlreadyExists):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgHasChildren),
		errors.Is(err, services.ErrOrgHasGroups):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgNotFound),
		errors.Is(err, services.ErrGroupNotFound),
		errors.Is(err, services.ErrMemberNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrNoReviewPermission):
		utils.Fail(w, http.StatusForbidden, err.Error())
	default:
		log.Printf("组织管理操作失败: %v", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}

// ==================== 辅助函数 ====================

// extractIDFromPath 从URL路径中提取末尾ID
// 示例："/api/v1/lesson-plans/organizations/xxx-yyy" → "xxx-yyy"
func extractIDFromPath(path string, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

// extractMiddleSegment 从含子路径的URL中提取中间段
// 示例："/api/v1/.../xxx-yyy/members" → "xxx-yyy"
func extractMiddleSegment(path string, prefix string, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(rest, suffix) {
		// 可能还有尾部斜杠
		rest = strings.TrimSuffix(rest, "/")
		if !strings.HasSuffix(rest, suffix) {
			return ""
		}
	}
	id := strings.TrimSuffix(rest, suffix)
	id = strings.TrimSuffix(id, "/")
	return id
}
