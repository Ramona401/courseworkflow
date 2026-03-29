package handlers

// 组件库管理HTTP处理器
// 负责组件CRUD、匹配、审核、萃取队列管理的HTTP接口
//
// Phase5新增端点：
//   GET  /api/v1/lesson-plans/extractions              — 待审萃取列表
//   POST /api/v1/lesson-plans/extractions/{id}/confirm — 确认/拒绝萃取记录

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ComponentHandler 组件库接口处理器
type ComponentHandler struct {
	compService *services.ComponentService
}

// NewComponentHandler 创建组件库处理器实例
func NewComponentHandler(compService *services.ComponentService) *ComponentHandler {
	return &ComponentHandler{compService: compService}
}

// ==================== 组件列表 ====================

// ListComponents 获取组件列表
// GET /api/v1/lesson-plans/components?library_type=&subject=&review_status=&scope=&limit=50&offset=0
func (h *ComponentHandler) ListComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	q := r.URL.Query()
	libraryType := q.Get("library_type")
	subject := q.Get("subject")
	reviewStatus := q.Get("review_status")
	scope := q.Get("scope")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	result, err := h.compService.ListComponents(r.Context(), libraryType, subject, reviewStatus, scope, limit, offset)
	if err != nil {
		log.Printf("获取组件列表失败: %v", err)
		utils.InternalError(w, "获取组件列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 创建组件 ====================

// CreateComponent 创建组件
// POST /api/v1/lesson-plans/components
func (h *ComponentHandler) CreateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CreateComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	comp, err := h.compService.CreateComponent(r.Context(), &req, userID)
	if err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, comp)
}

// ==================== 获取组件详情 ====================

// GetComponent 获取组件详情
// GET /api/v1/lesson-plans/components/{id}
func (h *ComponentHandler) GetComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractComponentID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}
	comp, err := h.compService.GetComponent(r.Context(), id)
	if err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, comp)
}

// ==================== 更新组件 ====================

// UpdateComponent 更新组件
// PUT /api/v1/lesson-plans/components/{id}
func (h *ComponentHandler) UpdateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	id := extractComponentID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	var req models.UpdateComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.compService.UpdateComponent(r.Context(), id, &req); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除组件 ====================

// DeleteComponent 删除组件（软删除）
// DELETE /api/v1/lesson-plans/components/{id}
func (h *ComponentHandler) DeleteComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	id := extractComponentID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	if err := h.compService.DeleteComponent(r.Context(), id); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 审核组件 ====================

// ReviewComponent 审核组件
// POST /api/v1/lesson-plans/components/{id}/review
func (h *ComponentHandler) ReviewComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	prefix := "/api/v1/lesson-plans/components/"
	suffix := "/review"
	id := extractMiddleSegment(r.URL.Path, prefix, suffix)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.ReviewComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.compService.ReviewComponent(r.Context(), id, userID, &req); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "审核成功"})
}

// ==================== 组件匹配 ====================

// MatchComponents 匹配组件
// POST /api/v1/lesson-plans/components/match
func (h *ComponentHandler) MatchComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	var req models.MatchComponentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	result, err := h.compService.MatchComponents(r.Context(), &req)
	if err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, result)
}

// ==================== Phase5：萃取队列管理 ====================

// ListExtractions 获取待审萃取列表（教研组长/骨干查看）
// GET /api/v1/lesson-plans/extractions?limit=50
func (h *ComponentHandler) ListExtractions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	result, err := h.compService.ListPendingExtractionItems(r.Context(), limit)
	if err != nil {
		log.Printf("获取萃取列表失败: %v", err)
		utils.InternalError(w, "获取萃取列表失败")
		return
	}
	utils.Success(w, result)
}

// ConfirmExtraction 确认或拒绝萃取记录
// POST /api/v1/lesson-plans/extractions/{id}/confirm
// body: {"decision": "confirmed" | "rejected"}
func (h *ComponentHandler) ConfirmExtraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从路径 /api/v1/lesson-plans/extractions/{id}/confirm 提取ID
	prefix := "/api/v1/lesson-plans/extractions/"
	suffix := "/confirm"
	id := extractMiddleSegment(r.URL.Path, prefix, suffix)
	if id == "" {
		utils.BadRequest(w, "缺少萃取记录ID")
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.ConfirmExtractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.compService.ConfirmExtractionByID(r.Context(), id, userID, req.Decision); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "操作成功"})
}

// ==================== 错误处理 ====================

func (h *ComponentHandler) handleCompError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrComponentLibTypeRequired),
		errors.Is(err, services.ErrComponentLibTypeInvalid),
		errors.Is(err, services.ErrComponentLabelRequired),
		errors.Is(err, services.ErrComponentReviewInvalid),
		errors.Is(err, services.ErrExtractionDecisionInvalid):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrComponentNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrExtractionNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("组件库操作失败: %v", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}

// ==================== 路径辅助函数 ====================

// extractComponentID 从组件URL路径中提取ID
func extractComponentID(path string) string {
	prefix := "/api/v1/lesson-plans/components/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if idx := strings.Index(id, "/"); idx > 0 {
		id = id[:idx]
	}
	return id
}
