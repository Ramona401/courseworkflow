package handlers

// credit_policy_handler.go — 积分策略HTTP处理器
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）：
//   - 策略管理：系统策略查看/更新（admin）+ 学校策略查看/更新/删除（admin+senior）
//   - 模型单价管理：列表/创建/更新/删除（admin）
//   - 策略列表：所有策略（admin+senior）
//   - 模型积分预览：所有活跃模型的积分预览
//   - 模拟计算器：输入token数→预估积分消耗

import (
	"encoding/json"
	"errors"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CreditPolicyHandler 积分策略HTTP处理器
type CreditPolicyHandler struct {
	service *services.CreditPolicyService
}

// NewCreditPolicyHandler 创建CreditPolicyHandler实例
func NewCreditPolicyHandler(service *services.CreditPolicyService) *CreditPolicyHandler {
	return &CreditPolicyHandler{service: service}
}

// ==================== 策略管理 ====================

// GetSystemPolicy 获取系统级策略
// GET /api/v1/tokens/credit-policies/system
func (h *CreditPolicyHandler) GetSystemPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.service.GetSystemPolicy(r.Context())
	if err != nil {
		// 不存在返回默认值（对齐AOCI）
		utils.JSON(w, http.StatusOK, 0, "", &models.CreditPolicy{
			Scope:        models.PolicyScopeSystem,
			ExchangeRate: models.DefaultExchangeRate,
			Multiplier:   models.DefaultMultiplier,
			Description:  "系统默认策略（未自定义）",
		})
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", policy)
}

// UpdateSystemPolicy 更新系统级策略
// PUT /api/v1/tokens/credit-policies/system
func (h *CreditPolicyHandler) UpdateSystemPolicy(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.UpdateCreditPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	policy, err := h.service.UpdateSystemPolicy(r.Context(), &req, claims.UserID)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "系统策略已更新", policy)
}

// GetSchoolPolicy 获取学校级策略
// GET /api/v1/tokens/credit-policies/school/{id}
func (h *CreditPolicyHandler) GetSchoolPolicy(w http.ResponseWriter, r *http.Request) {
	schoolID := extractCPPathID(r.URL.Path)
	if schoolID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少学校ID", nil)
		return
	}
	policy, err := h.service.GetSchoolPolicy(r.Context(), schoolID)
	if err != nil {
		// 不存在返回默认值（提示使用系统策略）
		utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
			"using_system_default": true,
			"message":              "该学校未自定义策略，使用系统默认策略",
		})
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", policy)
}

// UpdateSchoolPolicy 更新学校级策略
// PUT /api/v1/tokens/credit-policies/school/{id}
func (h *CreditPolicyHandler) UpdateSchoolPolicy(w http.ResponseWriter, r *http.Request) {
	schoolID := extractCPPathID(r.URL.Path)
	if schoolID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少学校ID", nil)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.UpdateCreditPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	policy, err := h.service.UpdateSchoolPolicy(r.Context(), schoolID, &req, claims.UserID)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "学校策略已更新", policy)
}

// DeleteSchoolPolicy 删除学校级策略
// DELETE /api/v1/tokens/credit-policies/school/{id}
func (h *CreditPolicyHandler) DeleteSchoolPolicy(w http.ResponseWriter, r *http.Request) {
	schoolID := extractCPPathID(r.URL.Path)
	if schoolID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少学校ID", nil)
		return
	}
	if err := h.service.DeleteSchoolPolicy(r.Context(), schoolID); err != nil {
		if errors.Is(err, repository.ErrCreditPolicyNotFound) {
			utils.JSON(w, http.StatusNotFound, -1, "学校策略不存在", nil)
			return
		}
		utils.JSON(w, http.StatusInternalServerError, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "学校策略已删除", nil)
}

// ListPolicies 列出所有策略
// GET /api/v1/tokens/credit-policies
func (h *CreditPolicyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPolicies(r.Context())
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", items)
}

// ==================== 模型单价管理 ====================

// ListModelPrices 列出模型单价
// GET /api/v1/tokens/model-prices
func (h *CreditPolicyHandler) ListModelPrices(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	items, err := h.service.ListModelPrices(r.Context(), includeInactive)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", items)
}

// CreateModelPrice 创建模型单价
// POST /api/v1/tokens/model-prices
func (h *CreditPolicyHandler) CreateModelPrice(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.CreateModelPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	mp, err := h.service.CreateModelPrice(r.Context(), &req, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrModelPriceDuplicate) {
			utils.JSON(w, http.StatusConflict, -1, "模型单价已存在", nil)
			return
		}
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "创建成功", mp)
}

// UpdateModelPrice 更新模型单价
// PUT /api/v1/tokens/model-prices/{id}
func (h *CreditPolicyHandler) UpdateModelPrice(w http.ResponseWriter, r *http.Request) {
	priceID := extractCPPathID(r.URL.Path)
	if priceID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少ID", nil)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.UpdateModelPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	mp, err := h.service.UpdateModelPrice(r.Context(), priceID, &req, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrModelPriceNotFound) {
			utils.JSON(w, http.StatusNotFound, -1, "模型单价不存在", nil)
			return
		}
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "更新成功", mp)
}

// DeleteModelPrice 删除模型单价
// DELETE /api/v1/tokens/model-prices/{id}
func (h *CreditPolicyHandler) DeleteModelPrice(w http.ResponseWriter, r *http.Request) {
	priceID := extractCPPathID(r.URL.Path)
	if priceID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少ID", nil)
		return
	}
	if err := h.service.DeleteModelPrice(r.Context(), priceID); err != nil {
		if errors.Is(err, repository.ErrModelPriceNotFound) {
			utils.JSON(w, http.StatusNotFound, -1, "模型单价不存在", nil)
			return
		}
		utils.JSON(w, http.StatusInternalServerError, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "删除成功", nil)
}

// ==================== 预览 + 模拟 ====================

// GetModelPreviews 获取模型积分预览
// GET /api/v1/tokens/model-previews
func (h *CreditPolicyHandler) GetModelPreviews(w http.ResponseWriter, r *http.Request) {
	previews, err := h.service.GetModelPreviews(r.Context())
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", previews)
}

// Simulate 模拟积分计算
// POST /api/v1/tokens/simulate
func (h *CreditPolicyHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	var req models.SimulateCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	calc, err := h.service.Simulate(r.Context(), &req)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", calc)
}

// ==================== 辅助函数 ====================

// extractCPPathID 从路径末尾提取ID
func extractCPPathID(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			id := path[i+1:]
			for len(id) > 0 && id[len(id)-1] == '/' {
				id = id[:len(id)-1]
			}
			if len(id) > 0 {
				return id
			}
		}
	}
	return ""
}
