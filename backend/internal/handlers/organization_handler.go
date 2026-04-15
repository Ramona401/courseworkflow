package handlers

// 组织与教研组管理HTTP处理器

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

func NewOrganizationHandler(orgService *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgService: orgService}
}

// ==================== 组织 CRUD ====================

func (h *OrganizationHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	result, err := h.orgService.ListOrganizations(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("parent_id"))
	if err != nil {
		log.Printf("获取组织列表失败: %v", err)
		utils.InternalError(w, "获取组织列表失败")
		return
	}
	utils.Success(w, result)
}

func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	org, err := h.orgService.CreateOrganization(r.Context(), &req)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, org)
}

func (h *OrganizationHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathOrgPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}
	org, err := h.orgService.GetOrganization(r.Context(), id)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, org)
}

func (h *OrganizationHandler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathOrgPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}
	var req models.UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.orgService.UpdateOrganization(r.Context(), id, &req); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

func (h *OrganizationHandler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathOrgPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}
	if err := h.orgService.DeleteOrganization(r.Context(), id); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教研组 CRUD ====================

func (h *OrganizationHandler) ListTeachingGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	result, err := h.orgService.ListTeachingGroups(r.Context(), r.URL.Query().Get("school_id"))
	if err != nil {
		log.Printf("获取教研组列表失败: %v", err)
		utils.InternalError(w, "获取教研组列表失败")
		return
	}
	utils.Success(w, result)
}

func (h *OrganizationHandler) CreateTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	tg, err := h.orgService.CreateTeachingGroup(r.Context(), &req)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, tg)
}

func (h *OrganizationHandler) GetTeachingGroupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathGroupPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}
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

func (h *OrganizationHandler) UpdateTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathGroupPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}
	var req models.UpdateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.orgService.UpdateTeachingGroup(r.Context(), id, &req); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

func (h *OrganizationHandler) DeleteTeachingGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	id := extractIDFromPath(r.URL.Path, utils.PathGroupPrefix)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}
	if err := h.orgService.DeleteTeachingGroup(r.Context(), id); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教研组成员管理 ====================

func (h *OrganizationHandler) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	groupID := extractMiddleSegment(r.URL.Path, utils.PathGroupPrefix, "/members")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}
	var req models.AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.orgService.AddGroupMember(r.Context(), groupID, &req); err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

func (h *OrganizationHandler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, utils.PathGroupPrefix) {
		utils.BadRequest(w, "路径格式错误")
		return
	}
	rest := strings.TrimPrefix(path, utils.PathGroupPrefix)
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

func (h *OrganizationHandler) GetUserTeachingGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
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

func extractIDFromPath(path string, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

func extractMiddleSegment(path string, prefix string, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(rest, suffix) {
		rest = strings.TrimSuffix(rest, "/")
		if !strings.HasSuffix(rest, suffix) {
			return ""
		}
	}
	id := strings.TrimSuffix(rest, suffix)
	id = strings.TrimSuffix(id, "/")
	return id
}
