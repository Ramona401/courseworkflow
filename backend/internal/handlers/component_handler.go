package handlers

// 组件库管理HTTP处理器
// 负责组件CRUD、匹配、审核、萃取队列管理的HTTP接口

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

func (h *ComponentHandler) ListComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *ComponentHandler) CreateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.CreateComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *ComponentHandler) GetComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *ComponentHandler) UpdateComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractComponentID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少组件ID")
		return
	}
	var req models.UpdateComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.compService.UpdateComponent(r.Context(), id, &req); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除组件 ====================

func (h *ComponentHandler) DeleteComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
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

func (h *ComponentHandler) ReviewComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
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
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.ReviewComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.compService.ReviewComponent(r.Context(), id, userID, &req); err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "审核成功"})
}

// ==================== 组件匹配 ====================

func (h *ComponentHandler) MatchComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.MatchComponentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	result, err := h.compService.MatchComponents(r.Context(), &req)
	if err != nil {
		h.handleCompError(w, err)
		return
	}
	utils.Success(w, result)
}

// ==================== 萃取队列管理 ====================

func (h *ComponentHandler) ListExtractions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *ComponentHandler) ConfirmExtraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	prefix := "/api/v1/lesson-plans/extractions/"
	suffix := "/confirm"
	id := extractMiddleSegment(r.URL.Path, prefix, suffix)
	if id == "" {
		utils.BadRequest(w, "缺少萃取记录ID")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.ConfirmExtractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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
