package handlers

// inspection_handler.go — 区域抽查HTTP处理器
//
// v127 新增（多级审核体系 · 区域抽查）：
//   - 抽查列表与详情
//   - 提交抽查结果（通过/撤回）
//   - 手动触发抽样
//   - 分配审查员
//   - 抽查统计
//   - 区域教研员管辖管理（CRUD）
//
// 路由前缀：
//   /api/v1/inspections/   — 抽查操作
//   /api/v1/district-inspectors/ — 区域教研员管理

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// InspectionHandler 抽查处理器
type InspectionHandler struct {
	inspService *services.InspectionService
}

// NewInspectionHandler 创建抽查处理器实例
func NewInspectionHandler(inspService *services.InspectionService) *InspectionHandler {
	return &InspectionHandler{inspService: inspService}
}

// ==================== 抽查列表 ====================

// ListInspections GET /api/v1/inspections
// 获取抽查列表（区域教研员看自己分配的，admin看全部）
func (h *InspectionHandler) ListInspections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	q := r.URL.Query()
	status := q.Get("status")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	// 非admin只能看自己分配的
	inspectorID := ""
	if claims.Role != models.RoleAdmin {
		inspectorID = claims.UserID
	}

	result, err := h.inspService.ListInspections(r.Context(), inspectorID, status, limit, offset)
	if err != nil {
		utils.InternalError(w, "获取抽查列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 抽查详情 ====================

// GetInspection GET /api/v1/inspections/{id}
func (h *InspectionHandler) GetInspection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	id := extractInspectionID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少抽查记录ID")
		return
	}

	record, err := h.inspService.GetInspection(r.Context(), id)
	if err != nil {
		h.handleInspectionError(w, err)
		return
	}
	utils.Success(w, record)
}

// ==================== 提交抽查结果 ====================

// ReviewInspection POST /api/v1/inspections/{id}/review
func (h *InspectionHandler) ReviewInspection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	id := extractInspectionMiddleID(r.URL.Path, "/review")
	if id == "" {
		utils.BadRequest(w, "缺少抽查记录ID")
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req models.InspectionReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.inspService.ReviewInspection(r.Context(), id, userID, &req); err != nil {
		h.handleInspectionError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "抽查结果已提交"})
}

// ==================== 手动触发抽样 ====================

// BatchSample POST /api/v1/inspections/batch-sample
func (h *InspectionHandler) BatchSample(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	var req models.BatchSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	count, err := h.inspService.BatchSample(r.Context(), &req)
	if err != nil {
		utils.InternalError(w, "抽样失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"message":       "抽样完成",
		"sampled_count": count,
	})
}

// ==================== 分配审查员 ====================

// AssignInspector PUT /api/v1/inspections/{id}/assign
func (h *InspectionHandler) AssignInspector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}

	id := extractInspectionMiddleID(r.URL.Path, "/assign")
	if id == "" {
		utils.BadRequest(w, "缺少抽查记录ID")
		return
	}

	var req models.AssignInspectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.inspService.AssignInspector(r.Context(), id, req.InspectorID); err != nil {
		h.handleInspectionError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "审查员分配成功"})
}

// ==================== 抽查统计 ====================

// GetInspectionStats GET /api/v1/inspections/stats
func (h *InspectionHandler) GetInspectionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 非admin只看自己的统计
	inspectorID := ""
	if claims.Role != models.RoleAdmin {
		inspectorID = claims.UserID
	}

	result, err := h.inspService.GetInspectionStats(r.Context(), inspectorID)
	if err != nil {
		utils.InternalError(w, "获取抽查统计失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 区域教研员管理 ====================

// ListDistrictInspectors GET /api/v1/district-inspectors
func (h *InspectionHandler) ListDistrictInspectors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	regionID := r.URL.Query().Get("region_id")
	items, err := h.inspService.ListDistrictInspectors(r.Context(), regionID)
	if err != nil {
		utils.InternalError(w, "获取区域教研员列表失败")
		return
	}
	utils.Success(w, items)
}

// CreateDistrictInspector POST /api/v1/district-inspectors
func (h *InspectionHandler) CreateDistrictInspector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	var req models.CreateDistrictInspectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	assign, err := h.inspService.CreateDistrictInspector(r.Context(), &req)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, assign)
}

// DeleteDistrictInspector DELETE /api/v1/district-inspectors/{id}
func (h *InspectionHandler) DeleteDistrictInspector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}

	id := extractDistrictInspectorID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少分配记录ID")
		return
	}

	if err := h.inspService.DeleteDistrictInspector(r.Context(), id); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已取消分配"})
}

// ==================== 错误处理 ====================

func (h *InspectionHandler) handleInspectionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInspectionNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrInspectionNoPermission):
		utils.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrInspectionAlreadyDone),
		errors.Is(err, services.ErrInspectionNotAssigned),
		errors.Is(err, services.ErrInspectionInvalidDecision):
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, "抽查操作失败，请稍后重试")
	}
}

// ==================== 路径解析辅助 ====================

// extractInspectionID 从 /api/v1/inspections/{id} 提取 id
func extractInspectionID(path string) string {
	prefix := "/api/v1/inspections/"
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

// extractInspectionMiddleID 从 /api/v1/inspections/{id}/{suffix} 提取 id
func extractInspectionMiddleID(path string, suffix string) string {
	prefix := "/api/v1/inspections/"
	return extractMiddleSegment(path, prefix, suffix)
}

// extractDistrictInspectorID 从 /api/v1/district-inspectors/{id} 提取 id
func extractDistrictInspectorID(path string) string {
	prefix := "/api/v1/district-inspectors/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}
