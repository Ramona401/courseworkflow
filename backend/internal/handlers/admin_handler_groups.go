package handlers

// admin_handler_groups.go — 教研组成员管理 + 用户↔教研组双向分配接口
//
// 职责：
//   - 教研组成员列表/添加/角色更新/移除（通过教研组维度操作）
//   - 用户加入教研组（通过用户维度操作，v52任务六新增）
//   - 用户移出教研组（通过用户维度操作，v52任务六新增，组长保护）

import (
	"encoding/json"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 教研组成员管理（通过教研组维度）====================

// ListAdminGroupMembers GET /api/v1/admin/groups/{id}/members
// 获取教研组成员列表
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

// AddAdminGroupMember POST /api/v1/admin/groups/{id}/members
// 向教研组添加成员（通过教研组维度）
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

// UpdateAdminGroupMemberRole PUT /api/v1/admin/groups/{id}/members/{uid}
// 更新教研组成员角色（member ↔ backbone）
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

// RemoveAdminGroupMember DELETE /api/v1/admin/groups/{id}/members/{uid}
// 从教研组移除成员（通过教研组维度）
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

// ==================== 用户↔教研组双向分配（通过用户维度，v52任务六新增）====================

// AddUserToGroup POST /api/v1/admin/users/{uid}/groups
// 将指定用户加入教研组（通过用户维度操作）
//
// 请求体：{"group_id": "...", "role": "member|backbone"}
// 若用户已在该教研组，服务层返回错误
func (h *AdminHandler) AddUserToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	// 从路径提取用户ID：/api/v1/admin/users/{uid}/groups
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
	// role 默认为 member
	if body.Role != "member" && body.Role != "backbone" {
		body.Role = "member"
	}
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

// RemoveUserFromGroup DELETE /api/v1/admin/users/{uid}/groups/{gid}
// 将指定用户移出教研组（通过用户维度操作）
//
// 业务规则：教研组长不可移除，须先更换组长
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
	// 组长保护：教研组长不可通过此接口移除
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
