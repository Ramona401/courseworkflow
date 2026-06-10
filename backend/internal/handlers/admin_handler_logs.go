package handlers

// admin_handler_logs.go — 组织列表查询 + 操作日志查询接口
//
// 组织列表越权修复（Phase 6 验收期补漏）：
//   ListAdminOrgs / ListAdminGroups 这两个 /admin/* 端点经 adminOrSchoolAdmin 中间件，
//   admin 与 senior_operator 都可达，故必须按真实 claims 解析数据范围后传 service 过滤，
//   不能假定调用者一定是 admin。与 /lesson-plans/organizations 共用同一套 ResolveDataScope。
//   （注：当前前端组织架构 Tab 走 /lesson-plans/organizations；本两端点为历史遗留，
//     仍一并堵住数据范围，防止绕过前端直接调用 API 越权。）

import (
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ListAdminOrgs GET /api/v1/admin/orgs
func (h *AdminHandler) ListAdminOrgs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	scope := services.ResolveDataScope(r.Context(), claims.Role, claims.UserID)
	result, err := h.orgService.ListOrganizations(
		r.Context(),
		r.URL.Query().Get("type"),
		r.URL.Query().Get("parent_id"),
		scope,
	)
	if err != nil {
		utils.InternalError(w, "获取组织列表失败")
		return
	}
	utils.Success(w, result)
}

// ListAdminGroups GET /api/v1/admin/groups
func (h *AdminHandler) ListAdminGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	scope := services.ResolveDataScope(r.Context(), claims.Role, claims.UserID)
	result, err := h.orgService.ListTeachingGroups(r.Context(), r.URL.Query().Get("school_id"), scope)
	if err != nil {
		utils.InternalError(w, "获取教研组列表失败")
		return
	}
	utils.Success(w, result)
}

// ListAdminAuditLogs GET /api/v1/admin/audit-logs
func (h *AdminHandler) ListAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	result, err := repository.ListAuditLogs(r.Context(), repository.AuditLogQueryParams{
		UserID:    q.Get("user_id"),
		Username:  q.Get("username"),
		Action:    q.Get("action"),
		StartDate: q.Get("start_date"),
		EndDate:   q.Get("end_date"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		utils.InternalError(w, "查询操作日志失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}
