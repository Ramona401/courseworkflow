package handlers

// admin_handler_groups.go — 教研组成员管理 + 用户↔教研组双向分配 + 移出本校接口
//
// 归属治理批A（2026-07-04）改动：
//   1. 组归属四个写端点补审计（此前全部零留痕，lichao01 退组无迹可查的根因之一）：
//        admin.group_member_add    加入教研组（教研组面板 / 用户详情两入口）
//        admin.group_member_remove 移出教研组（同上两入口，detail 里 entry 区分）
//        admin.group_member_role   变更组内角色
//   2. RemoveAdminGroupMember 收口：原实现绕过 service 直调 repository.RemoveGroupMember，
//      无业务校验、无 journald 日志、无审计——现改走 h.orgService.RemoveGroupMember，
//      与用户详情入口(RemoveUserFromGroup)同一条路。
//   3. 新增 RemoveUserFromSchool（归属三规则之 R3）：
//        DELETE /api/v1/admin/users/{uid}/schools/{sid}
//      单事务连带退出该校全部教研组 + 删校籍行，审计 admin.school_member_remove 记完整明细。
//      权限：admin 任意学校；senior 仅本校（GetSchoolByAdminUserID 匹配）且目标须教师级
//      （ensureSeniorTargetIsTeacher）；region 被路由层 regionReadOnlyGate + 本文件
//      ensureRegionAdminReadOnly 双保险拦截。
//   归属三规则：R1 加组⇒自动入校（service 内既有）；R2 退组只退组（不碰校籍）；
//               R3 退校⇒连带退光该校组（本文件新接口）。
//
// 审计 action 说明：admin.group_member_* / admin.school_member_remove 暂未登记
// audit_repo.actionNameMap（批A2 补），日志列表会显示原始 action 码，仅外观问题
// （与既有 admin.user_update 同状况），详情 JSON 完整可查。

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

// AddAdminGroupMember 教研组面板入口：向组内添加成员（R1：service 内自动入校）
// 批A：补审计 admin.group_member_add
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
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
	repository.WriteAuditLog(claims.UserID, "admin.group_member_add",
		map[string]interface{}{
			"target_user": req.UserID,
			"group_id":    groupID,
			"role":        req.Role,
			"entry":       "group_panel",
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "添加成功"})
}

// UpdateAdminGroupMemberRole 变更组内角色（member/backbone/lead 三态）
// 批A：补审计 admin.group_member_role
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
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
	repository.WriteAuditLog(claims.UserID, "admin.group_member_role",
		map[string]interface{}{
			"target_user": userID,
			"group_id":    groupID,
			"new_role":    req.Role,
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "角色更新成功"})
}

// RemoveAdminGroupMember 教研组面板入口：移出组成员（R2：只退组，不碰校籍）
// 批A收口：原直调 repository（零日志零审计的绕过路径），改走 service + 补审计
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		utils.InternalError(w, "移除成员失败: "+err.Error())
		return
	}
	repository.WriteAuditLog(claims.UserID, "admin.group_member_remove",
		map[string]interface{}{
			"target_user": userID,
			"group_id":    groupID,
			"entry":       "group_panel",
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "移除成功"})
}

// ==================== 用户↔教研组双向分配 ====================

// AddUserToGroup 用户详情入口：把用户加入某教研组（R1：service 内自动入校）
// 批A：补审计 admin.group_member_add
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
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
	repository.WriteAuditLog(claims.UserID, "admin.group_member_add",
		map[string]interface{}{
			"target_user": userID,
			"group_id":    body.GroupID,
			"role":        body.Role,
			"entry":       "user_detail",
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "已成功加入教研组"})
}

// RemoveUserFromGroup 用户详情入口：把用户移出某教研组（R2：只退组，不碰校籍）
// 批A：补审计 admin.group_member_remove
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	// v109多组长：允许直接移除组长成员，无需先更换组长
	if err := h.orgService.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
		utils.InternalError(w, "移出教研组失败: "+err.Error())
		return
	}
	repository.WriteAuditLog(claims.UserID, "admin.group_member_remove",
		map[string]interface{}{
			"target_user": userID,
			"group_id":    groupID,
			"entry":       "user_detail",
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "已移出教研组"})
}

// ==================== 移出本校（归属三规则之 R3，批A新增；批C2 增可选连带移除任命）====================

// RemoveUserFromSchool DELETE /api/v1/admin/users/{uid}/schools/{sid}[?remove_admin=1]
//
// 语义：把用户从某学校【彻底】移出——单事务连带退出该校全部教研组 + 删除校籍行。
// 这是系统里唯一的"退校"入口（R3）；退组入口（R2）永不触碰校籍。
//
// 批C2（2026-07-04）：新增可选参数 remove_admin=1——目标同时是该校管理员时，
// 先走 orgService.RemoveOrgAdmin 完整链路（主字段补位/置空 + 末任命自动降级 + 审计），
// 再执行移出本校，一个动作彻底清干净。不带该参数则保留管辖（支持"派驻管理员"场景：
// 管本校但不是本校成员）。此前"移出本校"与"移除任命"两入口割裂，管理员点了前者
// 以为人已清走，实际任命与身份仍挂着（lichao01 二次实测暴露）。
//
// 权限：
//   - admin           ：任意学校；remove_admin 由 RemoveOrgAdmin 的权限层二次校验（admin 恒过）
//   - senior_operator ：仅本校 + 目标须教师级——教师级目标不可能持有本校任命，
//                       故 senior 实际到不了 remove_admin 分支（带了参数也会因目标校验先被拦）
//   - region_admin    ：只读（路由层 regionReadOnlyGate 已拦非 GET，此处双保险）
//
// 审计：admin.school_member_remove（含 with_admin_removal 标志）；连带移任命时另写
// admin.org_admin_remove（entry=school_member_remove），触发降级再写 role_downgrade。
func (h *AdminHandler) RemoveUserFromSchool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	userID, schoolID := extractUserSchoolPath(r.URL.Path)
	if userID == "" || schoolID == "" {
		utils.BadRequest(w, "缺少用户ID或学校ID")
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

	// 批C2：是否连带移除该校管理员任命
	removeAdmin := r.URL.Query().Get("remove_admin") == "1"

	switch claims.Role {
	case models.RoleAdmin:
		// admin：任意学校，直接放行
	case models.RoleSeniorOperator:
		// senior：仅本校 + 目标账号级别守卫
		school, sErr := repository.GetSchoolByAdminUserID(r.Context(), claims.UserID)
		if sErr != nil || school == nil || school.ID == "" {
			utils.Forbidden(w, "您尚未绑定学校,请联系系统管理员")
			return
		}
		if school.ID != schoolID {
			utils.Forbidden(w, "只能将成员移出您管理的学校")
			return
		}
		if err := ensureSeniorTargetIsTeacher(r.Context(), userID); err != nil {
			utils.Forbidden(w, err.Error())
			return
		}
	default:
		utils.Forbidden(w, "权限不足")
		return
	}

	// ==================== 批C2：先移任命（可选，走完整降级链路）====================
	adminNote := ""
	if removeAdmin {
		res, aErr := h.orgService.RemoveOrgAdmin(r.Context(), schoolID, userID, claims.Role, claims.UserID)
		switch {
		case aErr == nil:
			repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminRemove,
				map[string]interface{}{"org_id": schoolID, "target_user": userID, "entry": "school_member_remove"},
				repository.GetClientIP(r.RemoteAddr))
			adminNote = "已移除其本校管理员任命"
			if res != nil {
				switch {
				case res.RoleDowngraded:
					adminNote += "（已无任何管辖，身份已自动降级为骨干教师，其重新登录后生效）"
					repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminRoleDowngrade,
						map[string]interface{}{
							"org_id":      schoolID,
							"target_user": userID,
							"from_role":   res.FromRole,
							"new_role":    res.NewRole,
						}, repository.GetClientIP(r.RemoteAddr))
				case res.DowngradeFailed:
					adminNote += "（身份自动降级失败，请到用户管理检查其系统身份）"
				}
			}
			adminNote += "；"
		case errors.Is(aErr, services.ErrMemberNotFound):
			// 目标本就没有该校任命（前端标志过期/并发已移除）：静默跳过，继续移出本校
		case errors.Is(aErr, services.ErrOrgAdminNoPermission):
			utils.Forbidden(w, "您没有权限移除该用户的管理员任命")
			return
		default:
			utils.InternalError(w, "移除管理员任命失败: "+aErr.Error())
			return
		}
	}

	result, err := h.orgService.RemoveUserFromSchool(r.Context(), schoolID, userID)
	if err != nil {
		prefix := ""
		if adminNote != "" {
			// 任命已被移除但校籍移除失败：明示已完成的部分，避免误以为整体未生效
			prefix = "（注意：其管理员任命已被移除）"
		}
		switch {
		case errors.Is(err, services.ErrOrgNotFound):
			utils.Fail(w, http.StatusNotFound, prefix+"学校不存在")
		case errors.Is(err, services.ErrMemberNotFound):
			utils.BadRequest(w, prefix+"该用户不是此学校的成员（既无校籍也不在该校任何教研组）")
		default:
			utils.InternalError(w, prefix+"移出本校失败: "+err.Error())
		}
		return
	}

	// 审计：完整记录联动明细（含是否连带移除任命）
	repository.WriteAuditLog(claims.UserID, "admin.school_member_remove",
		map[string]interface{}{
			"target_user":         userID,
			"school_id":           schoolID,
			"school_name":         result.SchoolName,
			"removed_group_count": len(result.RemovedGroupIDs),
			"removed_group_ids":   result.RemovedGroupIDs,
			"school_row_removed":  result.SchoolRowRemoved,
			"with_admin_removal":  removeAdmin,
		}, repository.GetClientIP(r.RemoteAddr))

	msg := adminNote + fmt.Sprintf("已将该用户移出学校「%s」", result.SchoolName)
	if n := len(result.RemovedGroupIDs); n > 0 {
		msg += fmt.Sprintf("，并连带退出该校 %d 个教研组", n)
	}
	utils.Success(w, map[string]string{"message": msg})
}

// extractUserSchoolPath 解析 /api/v1/admin/users/{uid}/schools/{sid} 路径
// （与 extractUserGroupPath 同构，分隔段为 /schools/）
func extractUserSchoolPath(path string) (string, string) {
	if !strings.HasPrefix(path, adminUsersPrefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, adminUsersPrefix)
	parts := strings.SplitN(rest, "/schools/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	uid := strings.TrimSuffix(parts[0], "/")
	sid := strings.TrimSuffix(parts[1], "/")
	if uid == "" || sid == "" {
		return "", ""
	}
	return uid, sid
}
