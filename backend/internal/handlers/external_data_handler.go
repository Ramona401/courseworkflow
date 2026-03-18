package handlers

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ExternalDataHandler 外部数据配置接口处理器
type ExternalDataHandler struct {
	edService *services.ExternalDataService
}

// NewExternalDataHandler 创建外部数据配置处理器实例
func NewExternalDataHandler(edService *services.ExternalDataService) *ExternalDataHandler {
	return &ExternalDataHandler{edService: edService}
}

// ==================== 配置读取接口 ====================

// GetConfigs 获取所有外部数据配置
// GET /api/v1/external-data/configs
func (h *ExternalDataHandler) GetConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	result, err := h.edService.GetAllConfigs()
	if err != nil {
		utils.InternalError(w, "获取外部数据配置失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 配置更新接口 ====================

// UpdateConfigs 批量更新外部数据配置
// PUT /api/v1/external-data/configs
func (h *ExternalDataHandler) UpdateConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 解析请求体
	var req models.UpdateExternalDataConfigsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 获取当前操作者ID（GetClaims返回两个值）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "无法获取用户信息")
		return
	}

	// 执行批量更新
	if err := h.edService.UpdateConfigs(&req, claims.UserID); err != nil {
		handleExternalDataError(w, err)
		return
	}

	// 返回更新后的配置列表
	result, err := h.edService.GetAllConfigs()
	if err != nil {
		utils.InternalError(w, "获取更新后配置失败")
		return
	}

	utils.Success(w, result)
}

// ==================== 错误处理 ====================

// handleExternalDataError 统一外部数据配置错误分发
func handleExternalDataError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrEDConfigsRequired:
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}
