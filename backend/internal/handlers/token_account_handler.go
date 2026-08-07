package handlers

// token_account_handler.go — Token积分账户管理HTTP处理器
//
// 包含账户创建、列表、详情和状态更新。
// 路由层负责超级管理员写权限；本文件负责参数与数据范围校验。
//
// 本次修复：
//   ListAccounts改用ListAccountsSearchableScoped，使个人账户可按
//   登录用户名搜索，并在列表中显示“显示名 (@用户名)”。

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// CreateAccount 创建积分账户。
func (h *TokenHandler) CreateAccount(
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

	var request models.CreateTokenAccountRequest

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

	if request.OwnerID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"owner_id不能为空",
			nil,
		)
		return
	}

	account, err := h.tokenService.CreateAccount(
		r.Context(),
		&request,
	)
	if err != nil {
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
		"创建成功",
		account,
	)
}

// ListAccounts 查询账户列表。
//
// keyword支持：
//   - 账户显示名称；
//   - 个人账户登录用户名；
//   - owner_id。
func (h *TokenHandler) ListAccounts(
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

	items, total, err :=
		h.tokenService.ListAccountsSearchableScoped(
			r.Context(),
			query.Get("type"),
			query.Get("parent_id"),
			query.Get("status"),
			query.Get("keyword"),
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
		response["scope_message"] =
			scope.BlockedReason
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		response,
	)
}

// GetAccountDetail 获取账户详情。
func (h *TokenHandler) GetAccountDetail(
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

	accountID := extractTokenPathID(
		r.URL.Path,
	)
	if accountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少账户ID",
			nil,
		)
		return
	}

	detail, err := h.tokenService.GetAccount(
		r.Context(),
		accountID,
	)
	if err != nil {
		handleTokenError(
			w,
			err,
		)
		return
	}

	scope := h.resolveScope(r)

	if scope != nil && !scope.IsAdmin {
		if detail.AccountType ==
			models.AccountTypeRegion ||
			!tokenOwnerInScope(
				scope,
				detail.OwnerID,
			) {
			utils.JSON(
				w,
				http.StatusForbidden,
				-1,
				"无权查看该账户",
				nil,
			)
			return
		}
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		detail,
	)
}

// UpdateAccountStatus 更新账户状态。
func (h *TokenHandler) UpdateAccountStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.JSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			"仅支持PUT请求",
			nil,
		)
		return
	}

	accountID := extractTokenMiddleID(
		r.URL.Path,
		"/status",
	)
	if accountID == "" {
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"缺少账户ID",
			nil,
		)
		return
	}

	var request struct {
		Status string `json:"status"`
	}

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

	if err := h.tokenService.UpdateAccountStatus(
		r.Context(),
		accountID,
		request.Status,
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
		"状态更新成功",
		nil,
	)
}
