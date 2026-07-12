package handlers

// token_handler_batch.go — Token积分批量分配HTTP处理器（一键分配 batch 新增）
//
// 独立文件挂既有 TokenHandler（token_handler.go 已近千行守 600 红线，
// 与 token_handler_region.go 同一拆分范式）。
//
// 端点：POST /api/v1/tokens/accounts/{id}/batch-allocate
//   请求体 {to_account_ids:[...], amount_each: 每户金额, memo}
//   响应   {success_count, fail_count, total_allocated, failures:[{to_account_id,reason}]}
//
// 权限口径与单笔 /allocate 逐字一致：
//   - 路由层：admin + senior_operator + region_admin（见 routes_token.go）
//   - handler 层：senior / region_admin 须过 tokenSourceAllowed 校验来源账户
//     （region 账户须 ∈ AllowedRegionOwnerIDs，school/personal 须 ∈ OwnerIDs），
//     admin 跳过（service 层每笔的父子关系校验兜底）。
//   - 目标合法性：每笔在 service.AllocateTokens 内做父子关系校验，非下级目标
//     该笔自然失败进 failures（配合前端只从 getAllocatableTargets 勾选，双保险）。
//
// 错误语义：
//   - 预检失败（金额非法/目标超限/余额不足等）→ 一笔未分，400 带中文原因
//   - 进入逐笔后 → 恒 200，成败明细在响应体（部分失败不算请求失败）

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// BatchAllocateTokens 批量分配积分
// POST /api/v1/tokens/accounts/{id}/batch-allocate
func (h *TokenHandler) BatchAllocateTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持POST请求", nil)
		return
	}

	// 解析路径中的来源账户ID（复用同包 extractTokenMiddleID）
	fromAccountID := extractTokenMiddleID(r.URL.Path, "/batch-allocate")
	if fromAccountID == "" {
		utils.JSON(w, http.StatusBadRequest, -1, "缺少来源账户ID", nil)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.JSON(w, http.StatusUnauthorized, -1, "未认证", nil)
		return
	}

	// senior / region_admin：校验"来源账户可作分配来源"（与单笔 AllocateTokens 逐字同口径；
	// admin 跳过，由 service 层每笔父子校验兜底）。
	if claims.Role == models.RoleSeniorOperator || claims.Role == models.RoleRegionAdmin {
		scope := h.resolveScope(r)
		fromAcc, accErr := h.tokenService.GetAccount(r.Context(), fromAccountID)
		if accErr != nil {
			handleTokenError(w, accErr)
			return
		}
		if scope != nil && scope.Blocked {
			utils.JSON(w, http.StatusForbidden, -1, "无权从该账户分配积分", nil)
			return
		}
		if !tokenSourceAllowed(scope, fromAcc.AccountType, fromAcc.OwnerID) {
			utils.JSON(w, http.StatusForbidden, -1, "无权从该账户分配积分", nil)
			return
		}
	}

	// 解析请求体
	var req services.BatchAllocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, -1, "请求体解析失败", nil)
		return
	}

	// 执行批量分配：返回 err = 预检失败（一笔未分）；err=nil = 已逐笔执行（成败在result）
	result, err := h.tokenService.BatchAllocateTokens(r.Context(), fromAccountID, &req, claims.UserID)
	if err != nil {
		// 预检类错误统一 400 带中文原因（余额不足/目标超限等对用户是可修正的输入问题）
		utils.JSON(w, http.StatusBadRequest, -1, err.Error(), nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "批量分配完成", result)
}
