package handlers

// admin_handler_groups.go — 教研组成员管理 + 用户↔教研组双向分配接口

import (
	"encoding/json"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 教研组成员管理（通过教研组维度）====================

func (h *AdminHandler) ListAdminGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	groupID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/groups/", "/members")
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
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

func (h *AdminHandler) AddAdminGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	groupID := extractAdminMiddleID(r.URL.Path, "/api/v1/admin/groups/", "/members")
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
		utils.InternalError(w, "添加成员失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "添加成功"})
}

func (h *AdminHandler) UpdateAdminGroupMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
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
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.Role != "member" && req.Role != "backbone" && req.Role != "lead" {
		utils.BadRequest(w, "角色只能是 member、backbone 或 lead")
		return
	}
	if err := repository.UpdateGroupMemberRole(r.Context(), groupID, userID, req.Role); err != nil {
		utils.InternalError(w, "更新成员角色失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "角色更新成功"})
}

func (h *AdminHandler) RemoveAdminGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
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

// ==================== 用户↔教研组双向分配 ====================

func (h *AdminHandler) AddUserToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := extractAdminMiddleID(r.URL.Path, adminUsersPrefix, "/groups")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	var body struct {
		GroupID string `json:"group_id"`
		Role    string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if body.GroupID == "" {
		utils.BadRequest(w, "请选择教研组")
		return
	}
	if body.Role != "member" && body.Role != "backbone" && body.Role != "lead" {
		body.Role = "member"
	}
	req := &models.AddGroupMemberRequest{UserID: userID, Role: body.Role}
	if err := h.orgService.AddGroupMember(r.Context(), body.GroupID, req); err != nil {
		utils.InternalError(w, "加入教研组失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已成功加入教研组"})
}

func (h *AdminHandler) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	userID, groupID := extractUserGroupPath(r.URL.Path)
	if userID == "" || groupID == "" {
		utils.BadRequest(w, "缺少用户ID或教研组ID")
		return
	}
	// v109多组长：允许直接移除组长成员，无需先更换组长
	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		utils.InternalError(w, "移出教研组失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已移出教研组"})
}
