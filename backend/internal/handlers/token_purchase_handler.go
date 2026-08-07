package handlers

// token_purchase_handler.go — Token积分采购与充值HTTP处理器
//
// POST写操作由路由层限制为超级管理员；
// GET采购记录仍按TokenScope进行数据收窄。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// PurchaseTokens 采购或充值积分。
func (h *TokenHandler) PurchaseTokens(
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

	var request models.PurchaseTokensRequest

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

	if request.AccountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"account_id不能为空",
			nil,
		)
		return
	}

	if err := h.tokenService.PurchaseTokens(
		r.Context(),
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
		"充值成功",
		nil,
	)
}

// ListPurchases 查询采购记录。
func (h *TokenHandler) ListPurchases(
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

	items, total, err := h.tokenService.ListPurchasesScoped(
		r.Context(),
		query.Get("account_id"),
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
