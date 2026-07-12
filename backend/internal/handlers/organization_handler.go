package handlers

// 组织与教研组管理HTTP处理器
//
// 归属治理批A2(2026-07-04)改动：
//   组织 CRUD 三个写端点补审计（此前零留痕）：
//     CreateOrganization → admin.org_create
//     UpdateOrganization → admin.org_update（detail 含 admin_user_id——
//       "编辑学校/区域弹窗改主管理员单字段"正是 lichao01 事件中 school_admin
//       身份消失却查无记录的那条路径,本次收口）
//     DeleteOrganization → admin.org_delete
//   审计动作码使用 repository.ActionOrgCreate/Update/Delete 常量。
//
// 迭代一 Phase 5 改动：
//   新增组织多管理员端点（挂在 /api/v1/lesson-plans/organizations/{id}/admins 下）：
//     - ListOrgAdmins   GET    .../{id}/admins
//     - AddOrgAdmin     POST   .../{id}/admins         body: {user_id, role_type, sync_role}
//     - RemoveOrgAdmin  DELETE .../{id}/admins/{user_id}
//   权限分层（admin 任何组织 / region_admin 仅辖区学校 / 其它拒绝）在 service 层判定，
//   handler 仅从 JWT claims 取 callerRole+callerID 透传。
//
// 组织列表越权修复（Phase 6 验收期补漏）：
//   ListOrganizations / ListTeachingGroups 此前无任何数据范围收窄，任何能进 /admin 的角色
//   （含 senior_operator）都能拿到全库所有区域/学校/教研组——属真实越权（前端隐藏挡不住）。
//   现改为：handler 取 claims.Role+UserID 调 services.ResolveDataScope（全系统唯一数据范围解析器），
//   把 scope 传给 service 按范围过滤：
//     - admin           → 全量（IsAdmin）
//     - region_admin    → 仅辖区区域 + 辖区学校
//     - senior_operator → 仅本校 + 本校所属父区域（保证组织三栏可用，区域栏只读展示上级）
//     - 其它/Blocked    → 空集
//   与教案/审核数据隔离共用同一套 ResolveDataScope，杜绝散落各处的 scope 判断遗漏。
//
// B13（任命即同步身份）改动：
//   AddOrgAdmin 请求体新增 sync_role（前端默认 true）；调用 service 新签名拿
//   OrgAdminAddResult，按四种结果拼中文 message（前端原样 toast），响应扩为
//   {message, role_synced, new_role}；RoleSynced=true 时额外写
//   admin.org_admin_role_sync 审计（region_admin 在用户域唯一写例外，必须留痕）。
//   审计动作码收敛为 repository 常量（值与旧字面量逐字一致，历史日志兼容）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// orgLog 组织管理处理器模块级日志器
var orgLog = logger.WithModule("org_handler")

// OrganizationHandler 组织管理接口处理器
type OrganizationHandler struct {
	orgService *services.OrganizationService
}

func NewOrganizationHandler(orgService *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgService: orgService}
}

// ==================== 组织 CRUD ====================

// ListOrganizations 组织列表（按调用者数据范围收窄，防跨区域/跨校越权）
//
// 流程：取 claims → ResolveDataScope → 传 service 按范围过滤。
// 未登录 → 401；其余错误 → 500。
func (h *OrganizationHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	// 解析调用者数据范围（全系统唯一数据范围解析器）
	scope := services.ResolveDataScope(r.Context(), claims.Role, claims.UserID)
	result, err := h.orgService.ListOrganizations(
		r.Context(),
		r.URL.Query().Get("type"),
		r.URL.Query().Get("parent_id"),
		scope,
	)
	if err != nil {
		orgLog.Error("获取组织列表失败", "error", err)
		utils.InternalError(w, "获取组织列表失败")
		return
	}
	utils.Success(w, result)
}

// CreateOrganization 创建区域/学校（批A2：补审计 admin.org_create）
func (h *OrganizationHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
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
	repository.WriteAuditLog(claims.UserID, repository.ActionOrgCreate,
		map[string]interface{}{
			"org_id":   org.ID,
			"org_name": req.Name,
			"org_type": req.Type,
		}, repository.GetClientIP(r.RemoteAddr))
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

// UpdateOrganization 编辑区域/学校（批A2：补审计 admin.org_update）
//
// 本端点覆盖"编辑弹窗改主管理员单字段(admin_user_id)"——lichao01 事件中
// school_admin 身份消失却零留痕的路径。detail 记录请求携带的 name/admin_user_id/
// status（admin_user_id 为请求原始值，null=未动/置空由前端语义决定，完整留痕即可追溯）。
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
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
	repository.WriteAuditLog(claims.UserID, repository.ActionOrgUpdate,
		map[string]interface{}{
			"org_id":        id,
			"name":          req.Name,
			"admin_user_id": req.AdminUserID,
			"status":        req.Status,
		}, repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteOrganization 删除区域/学校（批A2：补审计 admin.org_delete）
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
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	if err := h.orgService.DeleteOrganization(r.Context(), id); err != nil {
		h.handleOrgError(w, err)
		return
	}
	repository.WriteAuditLog(claims.UserID, repository.ActionOrgDelete,
		map[string]interface{}{"org_id": id},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 迭代一 Phase 5 新增：组织多管理员端点 ====================

// ListOrgAdmins GET /api/v1/lesson-plans/organizations/{id}/admins
// 列出某组织的全部管理员。权限在 service 层判定（admin/辖区 region_admin）。
func (h *OrganizationHandler) ListOrgAdmins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	orgID := extractOrgAdminsOrgID(r.URL.Path)
	if orgID == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	items, err := h.orgService.ListOrgAdmins(r.Context(), orgID, claims.Role, claims.UserID)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"admins": items, "total": len(items)})
}

// orgAdminRoleNameMap 系统身份→中文名（仅供本 handler 拼任命响应文案）
// 与前端 adminShared.RoleBadge 的 nameMap 口径一致
var orgAdminRoleNameMap = map[string]string{
	models.RoleAdmin:             "系统管理员",
	models.RoleRegionAdmin:       "区域管理员",
	models.RoleDistrictInspector: "区域教研员",
	models.RoleSeniorOperator:    "学校管理员",
	models.RoleOperator:          "骨干教师",
	models.RoleViewer:            "普通教师",
}

// orgAdminRoleName 取身份中文名，未登记的返回原始码兜底
func orgAdminRoleName(role string) string {
	if name, ok := orgAdminRoleNameMap[role]; ok {
		return name
	}
	return role
}

// AddOrgAdmin POST /api/v1/lesson-plans/organizations/{id}/admins
// body: {"user_id":"...", "role_type":"region_admin|school_admin", "sync_role":true|false}
//
// B13：sync_role=true（前端默认勾选）时，若目标当前身份是 operator/viewer，
// 任命成功后同步升级 users.role（region 任命→region_admin / school 任命→senior_operator）。
// 响应 {message, role_synced, new_role}，message 按四种结果拼好中文供前端原样 toast。
// RoleSynced=true 时额外写 admin.org_admin_role_sync 审计。
func (h *OrganizationHandler) AddOrgAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	orgID := extractOrgAdminsOrgID(r.URL.Path)
	if orgID == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req struct {
		UserID   string `json:"user_id"`
		RoleType string `json:"role_type"`
		SyncRole bool   `json:"sync_role"` // B13：是否同步升级系统身份（前端默认 true）
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	// 调 service：任命 +（可选）同步身份。error 非 nil = 任命本身失败
	result, err := h.orgService.AddOrgAdmin(r.Context(), orgID, req.UserID, req.RoleType, req.SyncRole, claims.Role, claims.UserID)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	// 任命本体审计（与旧字面量逐字一致的常量，历史日志兼容）
	repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminAdd,
		map[string]interface{}{"org_id": orgID, "target_user": req.UserID, "role_type": req.RoleType},
		repository.GetClientIP(r.RemoteAddr))

	// ---- 按四种结果拼响应文案 ----
	message := "任命成功"
	if req.SyncRole {
		switch {
		case result.RoleSynced:
			// 同步成功：额外写身份同步审计（region_admin 在用户域唯一写例外，必须留痕）
			message = "任命成功，已同步身份为" + orgAdminRoleName(result.NewRole)
			repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminRoleSync,
				map[string]interface{}{
					"org_id":      orgID,
					"target_user": req.UserID,
					"role_type":   req.RoleType,
					"from_role":   result.TargetRole,
					"new_role":    result.NewRole,
				},
				repository.GetClientIP(r.RemoteAddr))
		case result.SyncFailed:
			// 任命已成功但 users.role 更新失败：明示手工修复路径
			message = "任命成功，但身份同步失败，请到用户管理手动修改"
		default:
			// 目标身份不在升级白名单（已是管理身份等）：任命成功，身份原样
			message = "任命成功；该用户现有身份为" + orgAdminRoleName(result.TargetRole) + "，未变更"
		}
	}

	utils.Success(w, map[string]interface{}{
		"message":     message,
		"role_synced": result.RoleSynced,
		"new_role":    result.NewRole,
	})
}

// RemoveOrgAdmin DELETE /api/v1/lesson-plans/organizations/{id}/admins/{user_id}
//
// 批C（任命唯一事实源·末任命自动降级）：service 返回 OrgAdminRemoveResult——
// 目标任命归零且身份为任命制身份(senior_operator/region_admin)时已自动降级为骨干教师。
// 本 handler：移除本体审计照旧；降级成功额外写 admin.org_admin_role_downgrade 审计，
// 并在响应 message 明示（目标需重新登录后生效）；降级失败明示手工处理路径。
// 响应保持 {message: string} 形态，前端 OrgAdminsPanel 零改动兼容。
func (h *OrganizationHandler) RemoveOrgAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	orgID, targetUserID := extractOrgAdminPath(r.URL.Path)
	if orgID == "" || targetUserID == "" {
		utils.BadRequest(w, "缺少组织ID或管理员用户ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	result, err := h.orgService.RemoveOrgAdmin(r.Context(), orgID, targetUserID, claims.Role, claims.UserID)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}
	// 移除本体审计（与历史逐字一致的常量）
	repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminRemove,
		map[string]interface{}{"org_id": orgID, "target_user": targetUserID},
		repository.GetClientIP(r.RemoteAddr))

	// 批C：按降级结果拼文案 + 降级审计
	message := "移除成功"
	if result != nil {
		switch {
		case result.RoleDowngraded:
			message = "移除成功；该用户已无任何管辖，系统身份已自动调整为骨干教师（其重新登录后生效）"
			repository.WriteAuditLog(claims.UserID, repository.ActionOrgAdminRoleDowngrade,
				map[string]interface{}{
					"org_id":      orgID,
					"target_user": targetUserID,
					"from_role":   result.FromRole,
					"new_role":    result.NewRole,
				}, repository.GetClientIP(r.RemoteAddr))
		case result.DowngradeFailed:
			message = "移除成功，但身份自动降级失败，请到用户管理检查该用户的系统身份"
		}
	}
	utils.Success(w, map[string]string{"message": message})
}

// ==================== 教研组 CRUD ====================

// ListTeachingGroups 教研组列表（按调用者数据范围校验 school_id 归属，防跨校越权）
//
// 流程：取 claims → ResolveDataScope → 传 service。
// service 会校验请求的 school_id 是否在调用者可见学校集内，越权返空集。
func (h *OrganizationHandler) ListTeachingGroups(w http.ResponseWriter, r *http.Request) {
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
		orgLog.Error("获取教研组列表失败", "error", err)
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
		orgLog.Error("获取用户教研组失败", "user_id", userID, "error", err)
		utils.InternalError(w, "获取教研组失败")
		return
	}
	if groups == nil {
		groups = []*models.TeachingGroupListItem{}
	}
	utils.Success(w, groups)
}

// ==================== Logo上传 ====================

// UploadOrgLogo POST /api/v1/admin/orgs/{id}/upload-logo — 上传组织Logo
// admin和senior_operator都可操作（senior只能操作自己的学校，由前端控制）
func (h *OrganizationHandler) UploadOrgLogo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从路径提取组织ID: /api/v1/admin/orgs/{id}/upload-logo
	path := r.URL.Path
	prefix := "/api/v1/admin/orgs/"
	suffix := "/upload-logo"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		utils.BadRequest(w, "路径格式错误")
		return
	}
	orgID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if orgID == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}

	// 解析multipart
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		utils.BadRequest(w, "文件过大，最大支持4MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequest(w, "请上传Logo文件")
		return
	}
	defer file.Close()

	// 校验大小（2MB）
	if header.Size > 2*1024*1024 {
		utils.BadRequest(w, "Logo文件过大，最大支持2MB")
		return
	}

	// 校验MIME
	mimeType := header.Header.Get("Content-Type")
	allowed := map[string]string{
		"image/jpeg": ".jpg", "image/jpg": ".jpg", "image/png": ".png",
		"image/webp": ".webp", "image/svg+xml": ".svg",
	}
	ext, ok := allowed[mimeType]
	if !ok {
		utils.BadRequest(w, "不支持的Logo格式，支持JPG/PNG/WEBP/SVG")
		return
	}

	// 生成安全文件名
	baseName := fmt.Sprintf("org_%s_%d", orgID[:8], time.Now().UnixMilli())
	storedName := baseName + ext
	logoDir := "/www/wwwroot/tedna/uploads/org-logos/" + orgID
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		utils.InternalError(w, "创建Logo目录失败")
		return
	}

	// 保存物理文件
	fullPath := filepath.Join(logoDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		utils.InternalError(w, "创建文件失败")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		utils.InternalError(w, "保存文件失败")
		return
	}

	// 构建URL并更新数据库
	logoURL := "/uploads/org-logos/" + orgID + "/" + storedName
	if err := repository.UpdateOrganizationLogo(r.Context(), orgID, logoURL); err != nil {
		_ = os.Remove(fullPath)
		h.handleOrgError(w, err)
		return
	}

	orgLog.Info("组织Logo上传成功", "org_id", orgID, "url", logoURL)
	utils.Success(w, map[string]string{"url": logoURL})
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
		errors.Is(err, services.ErrMemberUserRequired),
		errors.Is(err, services.ErrOrgAdminUserRequired),
		errors.Is(err, services.ErrOrgAdminRoleTypeInvalid),
		errors.Is(err, services.ErrOrgAdminRoleTypeMismatch):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgNameExists),
		errors.Is(err, services.ErrGroupNameExists),
		errors.Is(err, services.ErrMemberAlreadyExists):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgHasChildren),
		errors.Is(err, services.ErrOrgHasGroups):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrOrgAdminNoPermission):
		utils.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrOrgNotFound),
		errors.Is(err, services.ErrGroupNotFound),
		errors.Is(err, services.ErrMemberNotFound),
		errors.Is(err, services.ErrOrgAdminTargetUserNF):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrNoReviewPermission):
		utils.Fail(w, http.StatusForbidden, err.Error())
	default:
		orgLog.Error("组织管理操作失败", "error", err)
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

// extractOrgAdminsOrgID 从 /api/v1/lesson-plans/organizations/{id}/admins 提取 orgID
// 用于 ListOrgAdmins(GET) 与 AddOrgAdmin(POST)——路径以 /admins 结尾
func extractOrgAdminsOrgID(path string) string {
	prefix := utils.PathOrgPrefix // "/api/v1/lesson-plans/organizations/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	// rest 形如 "{id}/admins"
	if !strings.HasSuffix(rest, "/admins") {
		return ""
	}
	orgID := strings.TrimSuffix(rest, "/admins")
	orgID = strings.TrimSuffix(orgID, "/")
	return orgID
}

// extractOrgAdminPath 从 /api/v1/lesson-plans/organizations/{id}/admins/{user_id} 提取 (orgID, userID)
// 用于 RemoveOrgAdmin(DELETE)
func extractOrgAdminPath(path string) (string, string) {
	prefix := utils.PathOrgPrefix
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	// rest 形如 "{id}/admins/{user_id}"
	parts := strings.Split(rest, "/admins/")
	if len(parts) != 2 {
		return "", ""
	}
	orgID := strings.TrimSuffix(parts[0], "/")
	userID := strings.TrimSuffix(parts[1], "/")
	if orgID == "" || userID == "" {
		return "", ""
	}
	return orgID, userID
}
