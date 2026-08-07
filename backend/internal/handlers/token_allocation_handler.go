package handlers

// token_allocation_handler.go — Token积分分配HTTP处理器
//
// 包含合法下级查询、单笔分配和分配记录。
// 批量分配继续由token_handler_batch.go独立承载。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// GetAllocatableTargets 查询来源账户的合法下级账户。
func (h *TokenHandler) GetAllocatableTargets(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持GET请求",
			nil,
		)
		return
	}

	fromAccountID := extractTokenMiddleID(
		r.URL.Path,
		"/allocatable-targets",
	)
	if fromAccountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少来源账户ID",
			nil,
		)
		return
	}

	scope := h.resolveScope(r)

	fromAccount, err := h.tokenService.GetAccount(
		r.Context(),
		fromAccountID,
	)
	if err != nil {
		handleTokenError(
			w,
			err,
		)
		return
	}

	if !tokenSourceAllowed(
		scope,
		fromAccount.AccountType,
		fromAccount.OwnerID,
	) {
		utils.JSON(
			w,
			http.StatusForbidden,
			-1,
			"无权从该账户分配积分",
			nil,
		)
		return
	}

	items, err := h.tokenService.ListChildAccountsScoped(
		r.Context(),
		fromAccountID,
		scope,
	)
	if err != nil {
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"查询可分配下级失败",
			nil,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		map[string]interface{}{
			"items": items,
			"total": len(items),
		},
	)
}

// AllocateTokens 分配积分。
func (h *TokenHandler) AllocateTokens(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持POST请求",
			nil,
		)
		return
	}

	fromAccountID := extractTokenMiddleID(
		r.URL.Path,
		"/allocate",
	)
	if fromAccountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少来源账户ID",
			nil,
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.JSON(
			w,
			http.StatusUnauthorized,
			-1,
			"未认证",
			nil,
		)
		return
	}

	if claims.Role == models.RoleSeniorOperator ||
		claims.Role == models.RoleRegionAdmin {
		scope := h.resolveScope(r)

		fromAccount, err := h.tokenService.GetAccount(
			r.Context(),
			fromAccountID,
		)
		if err != nil {
			handleTokenError(
				w,
				err,
			)
			return
		}

		if scope == nil ||
			scope.Blocked ||
			!tokenSourceAllowed(
				scope,
				fromAccount.AccountType,
				fromAccount.OwnerID,
			) {
			utils.JSON(
				w,
				http.StatusForbidden,
				-1,
				"无权从该账户分配积分",
				nil,
			)
			return
		}
	}

	var request models.AllocateTokensRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&request,
	); err != nil {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"请求体解析失败",
			nil,
		)
		return
	}

	if err := h.tokenService.AllocateTokens(
		r.Context(),
		fromAccountID,
		&request,
		claims.UserID,
	); err != nil {
		handleTokenError(
			w,
			err,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"分配成功",
		nil,
	)
}

// ListAllocations 查询分配记录。
func (h *TokenHandler) ListAllocations(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持GET请求",
			nil,
		)
		return
	}

	query := r.URL.Query()

	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)
	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	scope := h.resolveScope(r)

	items, total, err := h.tokenService.ListAllocationsScoped(
		r.Context(),
		query.Get("from_account_id"),
		query.Get("to_account_id"),
		query.Get("exclude_monthly") == "true",
		scope,
		limit,
		offset,
	)
	if err != nil {
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"查询失败",
			nil,
		)
		return
	}

	response := map[string]interface{}{
		"items": items,
		"total": total,
	}

	if scope != nil && scope.Blocked {
		response["scope_blocked"] = true
		response["scope_message"] = scope.BlockedReason
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		response,
	)
}
