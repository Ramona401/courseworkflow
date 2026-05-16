package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 课件组件库HTTP处理器 ====================

// CWComponentHandler 课件组件库处理器
type CWComponentHandler struct{}

// NewCWComponentHandler 创建课件组件库处理器
func NewCWComponentHandler() *CWComponentHandler {
	return &CWComponentHandler{}
}

// ListComponents GET /api/v1/courseware-components — 组件列表
func (h *CWComponentHandler) ListComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	componentType := r.URL.Query().Get("component_type")
	subjectScope := r.URL.Query().Get("subject_scope")
	gradeScope := r.URL.Query().Get("grade_scope")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	activeTrue := true
	items, total, err := repository.ListCWComponents(r.Context(), componentType, subjectScope, gradeScope, &activeTrue, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询组件列表失败: "+err.Error())
		return
	}
	utils.Success(w, &models.CWComponentListResponse{
		Components: items,
		Total:      total,
	})
}

// CreateComponent POST /api/v1/courseware-components — 创建组件（admin）
func (h *CWComponentHandler) CreateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil || claims.Role != "admin" {
		utils.Forbidden(w, "仅管理员可创建组件")
		return
	}

	var req models.CreateCWComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.Name == "" || req.ComponentType == "" || req.CodeContent == "" {
		utils.BadRequest(w, "名称、类型、代码内容为必填项")
		return
	}
	if !models.IsValidCWComponentType(req.ComponentType) {
		utils.BadRequest(w, "无效的组件类型: "+req.ComponentType)
		return
	}

	comp := &models.CoursewareComponent{
		Name:             req.Name,
		Description:      req.Description,
		ComponentType:    req.ComponentType,
		CodeContent:      req.CodeContent,
		PreviewImageURL:  req.PreviewImageURL,
		PreviewHTML:      req.PreviewHTML,
		SubjectScope:     req.SubjectScope,
		GradeScope:       req.GradeScope,
		TechDependencies: req.TechDependencies,
		Tags:             req.Tags,
		IsActive:         true,
		ReviewStatus:     models.CWCompReviewDraft,
	}
	if comp.SubjectScope == "" {
		comp.SubjectScope = "ALL"
	}
	if comp.GradeScope == "" {
		comp.GradeScope = "ALL"
	}

	if err := repository.CreateCWComponent(r.Context(), comp); err != nil {
		utils.InternalError(w, "创建组件失败: "+err.Error())
		return
	}
	utils.Success(w, comp)
}

// GetComponent GET /api/v1/courseware-components/{id} — 组件详情
func (h *CWComponentHandler) GetComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCWCompID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	comp, err := repository.GetCWComponentByID(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "组件不存在: "+err.Error())
		return
	}
	utils.Success(w, comp)
}

// UpdateComponent PUT /api/v1/courseware-components/{id} — 更新组件（admin）
func (h *CWComponentHandler) UpdateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil || claims.Role != "admin" {
		utils.Forbidden(w, "仅管理员可更新组件")
		return
	}
	id := extractCWCompID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	var req models.UpdateCWComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := repository.UpdateCWComponent(r.Context(), id, &req); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteComponent DELETE /api/v1/courseware-components/{id} — 删除组件（admin）
func (h *CWComponentHandler) DeleteComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil || claims.Role != "admin" {
		utils.Forbidden(w, "仅管理员可删除组件")
		return
	}
	id := extractCWCompID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	if err := repository.DeleteCWComponent(r.Context(), id); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// MatchComponents POST /api/v1/courseware-components/match — 组件匹配
func (h *CWComponentHandler) MatchComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	var req models.MatchCWComponentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	matched, err := repository.MatchCWComponents(r.Context(), &req)
	if err != nil {
		utils.InternalError(w, "组件匹配失败: "+err.Error())
		return
	}
	utils.Success(w, matched)
}

// CompressIndex POST /api/v1/courseware-components/{id}/index — 触发索引压缩
// TODO: Phase 2实现，当前返回占位响应
func (h *CWComponentHandler) CompressIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	utils.Success(w, map[string]string{"message": "索引压缩功能将在Phase 2实现"})
}

// ==================== 路径解析 ====================

// extractCWCompID 从 /api/v1/courseware-components/{id} 或 /{id}/xxx 提取ID
func extractCWCompID(path string) string {
	const prefix = "/api/v1/courseware-components/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	rest = strings.TrimRight(rest, "/")
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return rest
}
