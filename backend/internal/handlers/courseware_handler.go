package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件工坊HTTP处理器 ====================
// 课件CRUD + 页面操作 + 状态流转 + 风格模板

// CoursewareHandler 课件工坊处理器
type CoursewareHandler struct {
	cwService *services.CoursewareService
}

// NewCoursewareHandler 创建课件工坊处理器
func NewCoursewareHandler(cwService *services.CoursewareService) *CoursewareHandler {
	return &CoursewareHandler{cwService: cwService}
}

// ==================== 课件CRUD ====================

// ListCoursewares GET /api/v1/coursewares — 我的课件列表
func (h *CoursewareHandler) ListCoursewares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	status := r.URL.Query().Get("status")
	subject := r.URL.Query().Get("subject")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	resp, err := h.cwService.ListCoursewares(r.Context(), claims.UserID, status, subject, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询课件列表失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

// CreateCourseware POST /api/v1/coursewares — 创建课件
func (h *CoursewareHandler) CreateCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CreateCoursewareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	cw, err := h.cwService.CreateCourseware(r.Context(), claims.UserID, &req)
	if err != nil {
		utils.InternalError(w, "创建课件失败: "+err.Error())
		return
	}
	utils.Success(w, cw)
}

// GetCourseware GET /api/v1/coursewares/{id} — 课件详情
func (h *CoursewareHandler) GetCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	resp, err := h.cwService.GetCourseware(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "获取课件详情失败: "+err.Error())
		return
	}
	utils.Success(w, resp)
}

// UpdateCourseware PUT /api/v1/coursewares/{id} — 更新课件标题
func (h *CoursewareHandler) UpdateCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.UpdateCoursewareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.UpdateCoursewareTitle(r.Context(), id, claims.UserID, req.Title); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteCourseware DELETE /api/v1/coursewares/{id} — 删除课件
func (h *CoursewareHandler) DeleteCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if err := h.cwService.DeleteCourseware(r.Context(), id, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 课件页面操作 ====================

// GetCoursewarePages GET /api/v1/coursewares/{id}/pages — 获取全部页面
func (h *CoursewareHandler) GetCoursewarePages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/pages")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	pages, err := h.cwService.GetPages(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "获取页面列表失败: "+err.Error())
		return
	}
	utils.Success(w, pages)
}

// UpdatePageIndex PUT /api/v1/coursewares/{id}/pages/{num} — 更新单页索引
func (h *CoursewareHandler) UpdatePageIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	cwID, pageNum := extractCoursewarePagePath(r.URL.Path)
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}

	var req models.UpdateCWPageIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.UpdatePageIndex(r.Context(), cwID, pageNum, claims.UserID, &req); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// AddPage POST /api/v1/coursewares/{id}/pages — 手动添加页面
func (h *CoursewareHandler) AddPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/pages")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.AddCWPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	page, err := h.cwService.AddPage(r.Context(), id, claims.UserID, &req)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, page)
}

// ReorderPages PUT /api/v1/coursewares/{id}/pages/reorder — 页面排序
func (h *CoursewareHandler) ReorderPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/pages/reorder")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.ReorderCWPagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.ReorderPages(r.Context(), id, claims.UserID, req.PageIDs); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "排序成功"})
}

// ==================== 状态流转 ====================

// ConfirmIndex POST /api/v1/coursewares/{id}/confirm-index — 确认索引
func (h *CoursewareHandler) ConfirmIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/confirm-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if err := h.cwService.ConfirmIndex(r.Context(), id, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "索引确认成功，请选择风格"})
}

// SaveStyle PUT /api/v1/coursewares/{id}/style — 保存风格
func (h *CoursewareHandler) SaveStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/style")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req models.UpdateCoursewareStyleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.cwService.SaveStyle(r.Context(), id, claims.UserID, req.StyleConfig); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "风格保存成功"})
}

// ConfirmCourseware POST /api/v1/coursewares/{id}/confirm — 确认课件
func (h *CoursewareHandler) ConfirmCourseware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/confirm")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if err := h.cwService.ConfirmCourseware(r.Context(), id, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "课件确认成功"})
}

// ==================== 风格模板查询 ====================

// ListTemplates GET /api/v1/courseware-templates — 获取风格模板列表
func (h *CoursewareHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	templates, err := services.ListCWTemplates(r.Context(), true)
	if err != nil {
		utils.InternalError(w, "获取风格模板失败: "+err.Error())
		return
	}
	utils.Success(w, templates)
}

// GetTemplatePreview GET /api/v1/courseware-templates/{id}/preview — 模板样例预览
func (h *CoursewareHandler) GetTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCWTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}

	t, err := services.GetCWTemplateByID(r.Context(), id)
	if err != nil {
		utils.InternalError(w, "模板不存在: "+err.Error())
		return
	}
	utils.Success(w, t)
}

// ==================== 路径解析辅助函数 ====================

// extractCoursewareID 从 /api/v1/coursewares/{id} 提取ID
func extractCoursewareID(path string) string {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	rest = strings.TrimRight(rest, "/")
	if strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// extractCoursewareMiddleID 从 /api/v1/coursewares/{id}/{suffix} 提取ID
func extractCoursewareMiddleID(path string, suffix string) string {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	idx := strings.Index(rest, suffix)
	if idx <= 0 {
		return ""
	}
	return rest[:idx]
}

// extractCoursewarePagePath 从 /api/v1/coursewares/{id}/pages/{num} 提取ID和页码
func extractCoursewarePagePath(path string) (string, int) {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return "", 0
	}
	rest := path[len(prefix):]
	idx := strings.Index(rest, "/pages/")
	if idx <= 0 {
		return "", 0
	}
	cwID := rest[:idx]
	numStr := rest[idx+len("/pages/"):]
	numStr = strings.TrimRight(numStr, "/")
	if slashIdx := strings.Index(numStr, "/"); slashIdx >= 0 {
		numStr = numStr[:slashIdx]
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return cwID, 0
	}
	return cwID, num
}

// extractCWTemplateID 从 /api/v1/courseware-templates/{id}/... 提取ID
func extractCWTemplateID(path string) string {
	const prefix = "/api/v1/courseware-templates/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return strings.TrimRight(rest, "/")
}
