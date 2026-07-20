package handlers

// 组织与教研组管理HTTP处理器
//
// 归属治理批A2改动：
//   - 组织CRUD写端点补审计；
//   - 组织主管理员变更写入审计详情。
//
// 组织列表数据范围：
//   - admin：全量；
//   - region_admin：辖区区域与学校；
//   - senior_operator：本校与上级区域；
//   - 其它或范围解析失败：空集。
//
// 上下文7新增：
//   - 创建学校缺少教育类型或提交非法教育类型时返回HTTP 400；
//   - 创建区域由Service强制写mixed；
//   - 创建审计记录数据库最终写入的education_domain。

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

var orgLog = logger.WithModule("org_handler")

// OrganizationHandler 组织管理接口处理器。
type OrganizationHandler struct {
	orgService *services.OrganizationService
}

// NewOrganizationHandler 创建组织处理器。
func NewOrganizationHandler(
	orgService *services.OrganizationService,
) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// ==================== 组织 CRUD ====================

// ListOrganizations 查询组织列表。
func (h *OrganizationHandler) ListOrganizations(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	scope := services.ResolveDataScope(
		r.Context(),
		claims.Role,
		claims.UserID,
	)

	result, err := h.orgService.ListOrganizations(
		r.Context(),
		r.URL.Query().Get("type"),
		r.URL.Query().Get("parent_id"),
		scope,
	)
	if err != nil {
		orgLog.Error(
			"获取组织列表失败",
			"error",
			err,
		)
		utils.InternalError(w, "获取组织列表失败")
		return
	}

	utils.Success(w, result)
}

// CreateOrganization 创建区域或学校。
func (h *OrganizationHandler) CreateOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
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

	org, err := h.orgService.CreateOrganization(
		r.Context(),
		&req,
	)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	// 审计必须记录数据库最终写入值：
	// 区域即使伪造其它值，最终也只能记录mixed；
	// 学校记录经过Service和数据库双重校验后的具体教学域。
	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionOrgCreate,
		map[string]interface{}{
			"org_id":           org.ID,
			"org_name":         org.Name,
			"org_type":         org.Type,
			"education_domain": org.EducationDomain,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, org)
}

// GetOrganization 查询单个组织。
func (h *OrganizationHandler) GetOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathOrgPrefix,
	)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}

	org, err := h.orgService.GetOrganization(
		r.Context(),
		id,
	)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(w, org)
}

// UpdateOrganization 编辑区域或学校。
func (h *OrganizationHandler) UpdateOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathOrgPrefix,
	)
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

	if err := h.orgService.UpdateOrganization(
		r.Context(),
		id,
		&req,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionOrgUpdate,
		map[string]interface{}{
			"org_id":        id,
			"name":          req.Name,
			"admin_user_id": req.AdminUserID,
			"status":        req.Status,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// DeleteOrganization 删除区域或学校。
func (h *OrganizationHandler) DeleteOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathOrgPrefix,
	)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingOrgID)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	if err := h.orgService.DeleteOrganization(
		r.Context(),
		id,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionOrgDelete,
		map[string]interface{}{
			"org_id": id,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// ==================== 组织多管理员 ====================

// ListOrgAdmins 列出组织全部管理员。
func (h *OrganizationHandler) ListOrgAdmins(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
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

	items, err :=
		h.orgService.ListOrgAdminsWithEducationDomain(
			r.Context(),
			orgID,
			claims.Role,
			claims.UserID,
		)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	availableDomains, err :=
		h.orgService.ListOrgAdminEducationDomains(
			r.Context(),
			orgID,
			claims.Role,
			claims.UserID,
		)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"admins":                      items,
			"total":                       len(items),
			"available_education_domains": availableDomains,
		},
	)
}

var orgAdminRoleNameMap = map[string]string{
	models.RoleAdmin:             "系统管理员",
	models.RoleRegionAdmin:       "区域管理员",
	models.RoleDistrictInspector: "区域教研员",
	models.RoleSeniorOperator:    "学校管理员",
	models.RoleOperator:          "骨干教师",
	models.RoleViewer:            "普通教师",
}

func orgAdminRoleName(role string) string {
	if name, ok := orgAdminRoleNameMap[role]; ok {
		return name
	}
	return role
}

// AddOrgAdmin 任命组织管理员。
func (h *OrganizationHandler) AddOrgAdmin(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
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
		UserID          string `json:"user_id"`
		RoleType        string `json:"role_type"`
		EducationDomain string `json:"education_domain"`
		SyncRole        bool   `json:"sync_role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	result, err :=
		h.orgService.AddOrgAdminWithEducationDomain(
			r.Context(),
			orgID,
			req.UserID,
			req.RoleType,
			req.EducationDomain,
			req.SyncRole,
			claims.Role,
			claims.UserID,
		)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionOrgAdminAdd,
		map[string]interface{}{
			"org_id":           orgID,
			"target_user":      req.UserID,
			"role_type":        req.RoleType,
			"education_domain": result.EducationDomain,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	message := "任命成功"

	if req.SyncRole {
		switch {
		case result.RoleSynced:
			message = "任命成功，已同步身份为" +
				orgAdminRoleName(result.NewRole)

			repository.WriteAuditLog(
				claims.UserID,
				repository.ActionOrgAdminRoleSync,
				map[string]interface{}{
					"org_id":           orgID,
					"target_user":      req.UserID,
					"role_type":        req.RoleType,
					"education_domain": result.EducationDomain,
					"from_role":        result.TargetRole,
					"new_role":         result.NewRole,
				},
				repository.GetClientIP(r.RemoteAddr),
			)

		case result.SyncFailed:
			message = "任命成功，但身份同步失败，请到用户管理手动修改"

		default:
			message = "任命成功；该用户现有身份为" +
				orgAdminRoleName(result.TargetRole) +
				"，未变更"
		}
	}

	utils.Success(
		w,
		map[string]interface{}{
			"message":          message,
			"role_synced":      result.RoleSynced,
			"new_role":         result.NewRole,
			"education_domain": result.EducationDomain,
		},
	)
}

// RemoveOrgAdmin 移除组织管理员。
func (h *OrganizationHandler) RemoveOrgAdmin(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	orgID, targetUserID :=
		extractOrgAdminPath(r.URL.Path)
	if orgID == "" || targetUserID == "" {
		utils.BadRequest(
			w,
			"缺少组织ID或管理员用户ID",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	result, err := h.orgService.RemoveOrgAdmin(
		r.Context(),
		orgID,
		targetUserID,
		claims.Role,
		claims.UserID,
	)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionOrgAdminRemove,
		map[string]interface{}{
			"org_id":      orgID,
			"target_user": targetUserID,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	message := "移除成功"
	if result != nil {
		switch {
		case result.RoleDowngraded:
			message = "移除成功；该用户已无任何管辖，系统身份已自动调整为骨干教师（其重新登录后生效）"

			repository.WriteAuditLog(
				claims.UserID,
				repository.ActionOrgAdminRoleDowngrade,
				map[string]interface{}{
					"org_id":      orgID,
					"target_user": targetUserID,
					"from_role":   result.FromRole,
					"new_role":    result.NewRole,
				},
				repository.GetClientIP(r.RemoteAddr),
			)

		case result.DowngradeFailed:
			message = "移除成功，但身份自动降级失败，请到用户管理检查该用户的系统身份"
		}
	}

	utils.Success(
		w,
		map[string]string{
			"message": message,
		},
	)
}

// ==================== 教研组 CRUD ====================

// ListTeachingGroups 查询教研组列表。
func (h *OrganizationHandler) ListTeachingGroups(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	scope := services.ResolveDataScope(
		r.Context(),
		claims.Role,
		claims.UserID,
	)

	result, err := h.orgService.ListTeachingGroups(
		r.Context(),
		r.URL.Query().Get("school_id"),
		scope,
	)
	if err != nil {
		orgLog.Error(
			"获取教研组列表失败",
			"error",
			err,
		)
		utils.InternalError(w, "获取教研组列表失败")
		return
	}

	utils.Success(w, result)
}

// CreateTeachingGroup 创建教研组。
func (h *OrganizationHandler) CreateTeachingGroup(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	var req models.CreateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	group, err := h.orgService.CreateTeachingGroup(
		r.Context(),
		&req,
	)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(w, group)
}

// GetTeachingGroupDetail 查询教研组详情。
func (h *OrganizationHandler) GetTeachingGroupDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathGroupPrefix,
	)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	if index := strings.Index(id, "/"); index > 0 {
		id = id[:index]
	}

	detail, err := h.orgService.GetTeachingGroupDetail(
		r.Context(),
		id,
	)
	if err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(w, detail)
}

// UpdateTeachingGroup 更新教研组。
func (h *OrganizationHandler) UpdateTeachingGroup(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathGroupPrefix,
	)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	var req models.UpdateTeachingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.orgService.UpdateTeachingGroup(
		r.Context(),
		id,
		&req,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// DeleteTeachingGroup 删除教研组。
func (h *OrganizationHandler) DeleteTeachingGroup(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	id := extractIDFromPath(
		r.URL.Path,
		utils.PathGroupPrefix,
	)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	if err := h.orgService.DeleteTeachingGroup(
		r.Context(),
		id,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// ==================== 教研组成员 ====================

// AddGroupMember 添加教研组成员。
func (h *OrganizationHandler) AddGroupMember(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	groupID := extractMiddleSegment(
		r.URL.Path,
		utils.PathGroupPrefix,
		"/members",
	)
	if groupID == "" {
		utils.BadRequest(w, utils.MsgMissingGroupID)
		return
	}

	var req models.AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.orgService.AddGroupMember(
		r.Context(),
		groupID,
		&req,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "添加成功",
		},
	)
}

// RemoveGroupMember 移除教研组成员。
func (h *OrganizationHandler) RemoveGroupMember(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	if !strings.HasPrefix(
		r.URL.Path,
		utils.PathGroupPrefix,
	) {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	rest := strings.TrimPrefix(
		r.URL.Path,
		utils.PathGroupPrefix,
	)
	parts := strings.Split(rest, "/members/")
	if len(parts) != 2 ||
		parts[0] == "" ||
		parts[1] == "" {
		utils.BadRequest(
			w,
			"缺少教研组ID或成员ID",
		)
		return
	}

	groupID := parts[0]
	userID := strings.TrimSuffix(parts[1], "/")

	if err := h.orgService.RemoveGroupMember(
		r.Context(),
		groupID,
		userID,
	); err != nil {
		h.handleOrgError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "移除成功",
		},
	)
}

// GetUserTeachingGroups 查询当前用户所属教研组。
func (h *OrganizationHandler) GetUserTeachingGroups(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	groups, err := h.orgService.GetUserTeachingGroups(
		r.Context(),
		userID,
	)
	if err != nil {
		orgLog.Error(
			"获取用户教研组失败",
			"user_id",
			userID,
			"error",
			err,
		)
		utils.InternalError(w, "获取教研组失败")
		return
	}

	if groups == nil {
		groups = []*models.TeachingGroupListItem{}
	}

	utils.Success(w, groups)
}

// ==================== Logo上传 ====================

// UploadOrgLogo 上传组织Logo。
func (h *OrganizationHandler) UploadOrgLogo(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	const prefix = "/api/v1/admin/orgs/"
	const suffix = "/upload-logo"

	if !strings.HasPrefix(r.URL.Path, prefix) ||
		!strings.HasSuffix(r.URL.Path, suffix) {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	orgID := strings.TrimSuffix(
		strings.TrimPrefix(r.URL.Path, prefix),
		suffix,
	)
	if orgID == "" {
		utils.BadRequest(w, "缺少组织ID")
		return
	}

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

	if header.Size > 2*1024*1024 {
		utils.BadRequest(w, "Logo文件过大，最大支持2MB")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	allowed := map[string]string{
		"image/jpeg":    ".jpg",
		"image/jpg":     ".jpg",
		"image/png":     ".png",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
	}

	extension, allowedType := allowed[mimeType]
	if !allowedType {
		utils.BadRequest(
			w,
			"不支持的Logo格式，支持JPG/PNG/WEBP/SVG",
		)
		return
	}

	if len(orgID) < 8 {
		utils.BadRequest(w, "组织ID格式错误")
		return
	}

	baseName := fmt.Sprintf(
		"org_%s_%d",
		orgID[:8],
		time.Now().UnixMilli(),
	)
	storedName := baseName + extension

	logoDir := filepath.Join(
		"/www/wwwroot/tedna/uploads/org-logos",
		orgID,
	)
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		utils.InternalError(w, "创建Logo目录失败")
		return
	}

	fullPath := filepath.Join(logoDir, storedName)
	destination, err := os.Create(fullPath)
	if err != nil {
		utils.InternalError(w, "创建文件失败")
		return
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		_ = os.Remove(fullPath)
		utils.InternalError(w, "保存文件失败")
		return
	}

	logoURL := "/uploads/org-logos/" +
		orgID + "/" + storedName

	if err := repository.UpdateOrganizationLogo(
		r.Context(),
		orgID,
		logoURL,
	); err != nil {
		_ = os.Remove(fullPath)
		h.handleOrgError(w, err)
		return
	}

	orgLog.Info(
		"组织Logo上传成功",
		"org_id",
		orgID,
		"url",
		logoURL,
	)

	utils.Success(
		w,
		map[string]string{
			"url": logoURL,
		},
	)
}

// ==================== 错误处理 ====================

func (h *OrganizationHandler) handleOrgError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, services.ErrOrgNameRequired),
		errors.Is(err, services.ErrOrgTypeRequired),
		errors.Is(err, services.ErrOrgTypeInvalid),
		errors.Is(err, services.ErrSchoolNeedsParent),
		errors.Is(
			err,
			services.ErrSchoolEducationDomainRequired,
		),
		errors.Is(
			err,
			services.ErrSchoolEducationDomainInvalid,
		),
		errors.Is(err, services.ErrGroupNameRequired),
		errors.Is(err, services.ErrGroupSchoolRequired),
		errors.Is(err, services.ErrGroupSubjectRequired),
		errors.Is(err, services.ErrMemberUserRequired),
		errors.Is(err, services.ErrOrgAdminUserRequired),
		errors.Is(err, services.ErrOrgAdminRoleTypeInvalid),
		errors.Is(err, services.ErrOrgAdminRoleTypeMismatch),
		errors.Is(
			err,
			services.ErrOrgAdminEducationDomainRequired,
		),
		errors.Is(
			err,
			services.ErrOrgAdminEducationDomainInvalid,
		),
		errors.Is(
			err,
			services.ErrOrgAdminEducationDomainUnavailable,
		):
		utils.BadRequest(w, err.Error())

	case errors.Is(
		err,
		services.ErrOrgAdminEducationDomainConflict,
	),
		errors.Is(
			err,
			services.ErrOrgAdminEducationDomainUnconfigured,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(err, services.ErrOrgNameExists),
		errors.Is(err, services.ErrGroupNameExists),
		errors.Is(err, services.ErrMemberAlreadyExists):
		utils.BadRequest(w, err.Error())

	case errors.Is(err, services.ErrOrgHasChildren),
		errors.Is(err, services.ErrOrgHasGroups):
		utils.BadRequest(w, err.Error())

	case errors.Is(err, services.ErrOrgAdminNoPermission):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(err, services.ErrOrgNotFound),
		errors.Is(err, services.ErrGroupNotFound),
		errors.Is(err, services.ErrMemberNotFound),
		errors.Is(err, services.ErrOrgAdminTargetUserNF):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(err, services.ErrNoReviewPermission):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	default:
		orgLog.Error(
			"组织管理操作失败",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"操作失败，请稍后重试",
		)
	}
}

// ==================== 路径辅助 ====================

func extractIDFromPath(
	path string,
	prefix string,
) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}

func extractMiddleSegment(
	path string,
	prefix string,
	suffix string,
) string {
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
	return strings.TrimSuffix(id, "/")
}

func extractOrgAdminsOrgID(path string) string {
	prefix := utils.PathOrgPrefix
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")

	if !strings.HasSuffix(rest, "/admins") {
		return ""
	}

	orgID := strings.TrimSuffix(rest, "/admins")
	return strings.TrimSuffix(orgID, "/")
}

func extractOrgAdminPath(
	path string,
) (string, string) {
	prefix := utils.PathOrgPrefix
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}

	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")

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
