package handlers

// admin_handler_logs.go — 组织列表查询 + 操作日志查询接口

import (
	"net/http"
	"strconv"

	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ListAdminOrgs GET /api/v1/admin/orgs
func (h *AdminHandler) ListAdminOrgs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	result, err := h.orgService.ListOrganizations(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("parent_id"))
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
	result, err := h.orgService.ListTeachingGroups(r.Context(), r.URL.Query().Get("school_id"))
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
