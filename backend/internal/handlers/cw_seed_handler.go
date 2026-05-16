package handlers

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件工坊种子数据+模板管理HTTP处理器 ====================

// CWSeedHandler 种子数据+模板管理处理器
type CWSeedHandler struct {
	seedService *services.CoursewareSeedService
}

// NewCWSeedHandler 创建种子数据处理器
func NewCWSeedHandler(seedService *services.CoursewareSeedService) *CWSeedHandler {
	return &CWSeedHandler{seedService: seedService}
}

// ==================== 种子数据接口 ====================

// SeedAll POST /api/v1/admin/courseware-seed — 一键填充种子数据（admin）
func (h *CWSeedHandler) SeedAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil || claims.Role != "admin" {
		utils.Forbidden(w, "仅管理员可执行种子数据填充")
		return
	}

	// 解析force参数
	var req struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // 允许空body

	result, err := h.seedService.SeedAll(r.Context(), req.Force)
	if err != nil {
		utils.InternalError(w, "种子数据填充失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== Admin模板管理接口 ====================

// CreateTemplate POST /api/v1/admin/courseware-templates — 创建模板（admin）
func (h *CWSeedHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil || claims.Role != "admin" {
		utils.Forbidden(w, "仅管理员可创建模板")
		return
	}

	var req models.CreateCWTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.StyleCategory == "" {
		utils.BadRequest(w, "名称和风格类别为必填项")
		return
	}

	t := &models.CoursewareTemplate{
		Name:            req.Name,
		Description:     req.Description,
		StyleCategory:   req.StyleCategory,
		PreviewImageURL: req.PreviewImageURL,
		ColorScheme:     req.ColorScheme,
		CSSVariables:    req.CSSVariables,
		SamplePages:     req.SamplePages,
		IsActive:        true,
		SortOrder:       0,
	}
	if err := repository.CreateCWTemplate(r.Context(), t); err != nil {
		utils.InternalError(w, "创建模板失败: "+err.Error())
		return
	}
	utils.Success(w, t)
}

// UpdateTemplate PUT /api/v1/admin/courseware-templates/{id} — 更新模板（admin）
func (h *CWSeedHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
return
}
claims, ok := middleware.GetClaims(r.Context())
if !ok || claims == nil || claims.Role != "admin" {
utils.Forbidden(w, "仅管理员可更新模板")
return
}
id := extractAdminCWTemplateID(r.URL.Path)
if id == "" {
utils.BadRequest(w, "缺少模板ID")
return
}
var req models.UpdateCWTemplateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	utils.BadRequest(w, "请求参数格式错误")
	return
}

if err := repository.UpdateCWTemplate(r.Context(), id, &req); err != nil {
	utils.InternalError(w, err.Error())
	return
}
utils.Success(w, map[string]string{"message": "模板更新成功"})
}
// DeleteTemplate DELETE /api/v1/admin/courseware-templates/{id} — 删除模板（admin）
func (h *CWSeedHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodDelete {
utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
return
}
claims, ok := middleware.GetClaims(r.Context())
if !ok || claims == nil || claims.Role != "admin" {
utils.Forbidden(w, "仅管理员可删除模板")
return
}
id := extractAdminCWTemplateID(r.URL.Path)
if id == "" {
utils.BadRequest(w, "缺少模板ID")
return
}
if err := repository.DeleteCWTemplate(r.Context(), id); err != nil {
	utils.InternalError(w, err.Error())
	return
}
utils.Success(w, map[string]string{"message": "模板删除成功"})
}
// ==================== 路径解析 ====================
// extractAdminCWTemplateID 从 /api/v1/admin/courseware-templates/{id} 提取ID
func extractAdminCWTemplateID(path string) string {
const prefix = "/api/v1/admin/courseware-templates/"
if len(path) <= len(prefix) {
return ""
}
rest := path[len(prefix):]
// 去掉末尾斜杠
for len(rest) > 0 && rest[len(rest)-1] == '/' {
rest = rest[:len(rest)-1]
}
return rest
}
