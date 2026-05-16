package handlers

// token_handler.go — Token积分系统HTTP处理器
//
// v128 新增（阶段C · Token/积分系统）：
//   - 账户管理：创建/列表/详情/状态更新
//   - 积分分配：上级→下级
//   - 采购/充值
//   - 消费流水查询
//   - 概览统计
//   - 预警配置
//
// 权限设计：
//   - 概览统计/账户列表/详情：admin + senior_operator
//   - 创建账户/采购/充值/分配：admin
//   - 消费流水：admin + senior_operator（senior_operator限本校）
//   - 预警配置：admin
//   - 我的积分：登录即可

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// TokenHandler Token积分系统HTTP处理器
type TokenHandler struct {
	tokenService *services.TokenService
}

// NewTokenHandler 创建TokenHandler实例
func NewTokenHandler(tokenService *services.TokenService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService}
}

// ==================== 概览统计 ====================

// GetOverviewStats 获取Token系统概览统计
// GET /api/v1/tokens/overview
func (h *TokenHandler) GetOverviewStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	stats, err := h.tokenService.GetOverviewStats(r.Context())
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "获取统计失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", stats)
}

// ==================== 账户管理 ====================

// CreateAccount 创建积分账户
// POST /api/v1/tokens/accounts
func (h *TokenHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
		return
	}
	var req models.CreateTokenAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	if req.OwnerID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "owner_id不能为空", nil)
		return
	}
	acc, err := h.tokenService.CreateAccount(r.Context(), &req)
	if err != nil {
		handleTokenError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "创建成功", acc)
}

// ListAccounts 查询账户列表
// GET /api/v1/tokens/accounts?type=&parent_id=&status=&limit=&offset=
func (h *TokenHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	q := r.URL.Query()
	accountType := q.Get("type")
	parentID := q.Get("parent_id")
	status := q.Get("status")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	// v128.2：senior_operator 只看本校相关账户（本校+本校子账户）
	claims, _ := middleware.GetClaims(r.Context())
	if claims != nil && claims.Role == models.RoleSeniorOperator {
		schoolAcc, err := h.tokenService.GetSchoolAccountByAdmin(r.Context(), claims.UserID)
		if err == nil && schoolAcc != nil {
			parentID = schoolAcc.ID // 强制过滤为本校的子账户
		}
	}

	items, total, err := h.tokenService.ListAccounts(r.Context(), accountType, parentID, status, limit, offset)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// GetAccountDetail 获取账户详情
// GET /api/v1/tokens/accounts/{id}
func (h *TokenHandler) GetAccountDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	accountID := extractTokenPathID(r.URL.Path)
	if accountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
		return
	}
	detail, err := h.tokenService.GetAccount(r.Context(), accountID)
	if err != nil {
		handleTokenError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", detail)
}

// UpdateAccountStatus 更新账户状态
// PUT /api/v1/tokens/accounts/{id}/status
func (h *TokenHandler) UpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持PUT请求", nil)
		return
	}
	accountID := extractTokenMiddleID(r.URL.Path, "/status")
	if accountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	if err := h.tokenService.UpdateAccountStatus(r.Context(), accountID, req.Status); err != nil {
		handleTokenError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "状态更新成功", nil)
}

// ==================== 积分分配 ====================

// AllocateTokens 分配积分
// POST /api/v1/tokens/accounts/{id}/allocate
func (h *TokenHandler) AllocateTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
		return
	}
	fromAccountID := extractTokenMiddleID(r.URL.Path, "/allocate")
	if fromAccountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少来源账户ID", nil)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.AllocateTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	if err := h.tokenService.AllocateTokens(r.Context(), fromAccountID, &req, claims.UserID); err != nil {
		handleTokenError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "分配成功", nil)
}

// ListAllocations 查询分配记录
// GET /api/v1/tokens/allocations?from_account_id=&to_account_id=&limit=&offset=
func (h *TokenHandler) ListAllocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	q := r.URL.Query()
	fromID := q.Get("from_account_id")
	toID := q.Get("to_account_id")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, total, err := h.tokenService.ListAllocations(r.Context(), fromID, toID, limit, offset)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// ==================== 采购/充值 ====================

// PurchaseTokens 采购/充值积分
// POST /api/v1/tokens/purchases
func (h *TokenHandler) PurchaseTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	var req models.PurchaseTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	if req.AccountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "account_id不能为空", nil)
		return
	}
	if err := h.tokenService.PurchaseTokens(r.Context(), &req, claims.UserID); err != nil {
		handleTokenError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "充值成功", nil)
}

// ListPurchases 查询采购记录
// GET /api/v1/tokens/purchases?account_id=&limit=&offset=
func (h *TokenHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	q := r.URL.Query()
	accountID := q.Get("account_id")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, total, err := h.tokenService.ListPurchases(r.Context(), accountID, limit, offset)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// ==================== 消费流水 ====================

// ListConsumptionLogs 查询消费流水
// GET /api/v1/tokens/consumption?account_id=&user_id=&scene_code=&limit=&offset=
func (h *TokenHandler) ListConsumptionLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	q := r.URL.Query()
	accountID := q.Get("account_id")
	userID := q.Get("user_id")
	sceneCode := q.Get("scene_code")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, total, err := h.tokenService.ListConsumptionLogs(r.Context(), accountID, userID, sceneCode, limit, offset)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// ==================== 我的积分 ====================

// GetMyTokenAccount 获取当前用户的积分账户
// GET /api/v1/tokens/my-account
func (h *TokenHandler) GetMyTokenAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}
	acc, err := h.tokenService.GetAccountByOwner(r.Context(), models.AccountTypePersonal, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrTokenAccountNotFound) {
			utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
				"has_account": false,
				"message":     "暂未开通积分账户",
			})
			return
		}
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", map[string]interface{}{
		"has_account":       true,
		"account":           acc,
		"available_balance": acc.Balance - acc.FrozenAmount,
	})
}

// ==================== 预警配置 ====================

// GetAlertConfig 获取预警配置
// GET /api/v1/tokens/accounts/{id}/alert-config
func (h *TokenHandler) GetAlertConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	accountID := extractTokenMiddleID(r.URL.Path, "/alert-config")
	if accountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
		return
	}
	cfg, err := h.tokenService.GetAlertConfig(r.Context(), accountID)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "查询失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", cfg)
}

// UpdateAlertConfig 更新预警配置
// PUT /api/v1/tokens/accounts/{id}/alert-config
func (h *TokenHandler) UpdateAlertConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持PUT请求", nil)
		return
	}
	accountID := extractTokenMiddleID(r.URL.Path, "/alert-config")
	if accountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少账户ID", nil)
		return
	}
	var req models.UpdateAlertConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}
	if err := h.tokenService.UpdateAlertConfig(r.Context(), accountID, &req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "更新成功", nil)
}

// ==================== 辅助函数 ====================

// extractTokenPathID 从路径末尾提取ID
// /api/v1/tokens/accounts/{id} → id
func extractTokenPathID(path string) string {
	// 找最后一个 /
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			id := path[i+1:]
			// 去掉尾部斜杠
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

// extractTokenMiddleID 从路径中提取中间的ID
// /api/v1/tokens/accounts/{id}/suffix → id
func extractTokenMiddleID(path string, suffix string) string {
	// 去掉suffix
	if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
		path = path[:len(path)-len(suffix)]
	}
	return extractTokenPathID(path)
}

// handleTokenError 统一Token错误响应
func handleTokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrTokenAccountNotFound):
		utils.JSON(w, http.StatusNotFound, -1, "账户不存在", nil)
	case errors.Is(err, repository.ErrInsufficientBalance):
		utils.JSON(w, http.StatusBadRequest, -1, "积分余额不足", nil)
	case errors.Is(err, repository.ErrAccountSuspended):
		utils.JSON(w, http.StatusBadRequest, -1, "账户已冻结", nil)
	case errors.Is(err, repository.ErrDuplicateAccount):
		utils.JSON(w, http.StatusConflict, -1, "该实体已存在同类型账户", nil)
	case errors.Is(err, services.ErrTokenInvalidAmount):
		utils.JSON(w, http.StatusBadRequest, -1, "积分数量必须大于0", nil)
	case errors.Is(err, services.ErrTokenSelfAllocate):
		utils.JSON(w, http.StatusBadRequest, -1, "不能分配给自己", nil)
	case errors.Is(err, services.ErrTokenNotParentChild):
		utils.JSON(w, http.StatusBadRequest, -1, "只能向下级账户分配积分", nil)
	case errors.Is(err, services.ErrTokenAccountNotActive):
		utils.JSON(w, http.StatusBadRequest, -1, "账户不在活跃状态", nil)
	default:
		utils.JSON(w, http.StatusInternalServerError, -1, err.Error(), nil)
	}
}
