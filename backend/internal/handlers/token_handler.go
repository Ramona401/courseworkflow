package handlers

// token_handler.go — Token积分系统基础HTTP处理器
//
// 本文件只保留：
//   - TokenHandler结构与构造函数；
//   - TokenScope统一解析；
//   - 积分概览；
//   - 当前用户个人积分账户。
//
// 其它职责拆分为：
//   - token_account_handler.go：账户管理；
//   - token_allocation_handler.go：单笔分配和分配记录；
//   - token_purchase_handler.go：采购与充值；
//   - token_alert_handler.go：预警配置；
//   - token_consumption_handler.go：消费流水；
//   - token_summary_handler.go：消费汇总。

import (
	"errors"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// TokenHandler Token积分系统HTTP处理器。
type TokenHandler struct {
	tokenService *services.TokenService
}

// NewTokenHandler 创建TokenHandler实例。
func NewTokenHandler(
	tokenService *services.TokenService,
) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
	}
}

// resolveScope 从请求上下文解析当前请求者的数据可见范围。
func (h *TokenHandler) resolveScope(
	r *http.Request,
) *services.TokenScope {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		return h.tokenService.ResolveTokenScope(
			r.Context(),
			"",
			"",
		)
	}

	return h.tokenService.ResolveTokenScope(
		r.Context(),
		claims.Role,
		claims.UserID,
	)
}

// GetOverviewStats 获取Token系统概览统计。
func (h *TokenHandler) GetOverviewStats(
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

	scope := h.resolveScope(r)

	stats, err := h.tokenService.GetOverviewStatsScoped(
		r.Context(),
		scope,
	)
	if err != nil {
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"获取统计失败",
			nil,
		)
		return
	}

	utils.JSON(
		w,
		http.StatusOK,
		0,
		"",
		stats,
	)
}

// GetMyTokenAccount 获取当前用户的个人积分账户。
func (h *TokenHandler) GetMyTokenAccount(
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

	account, err := h.tokenService.GetAccountByOwner(
		r.Context(),
		models.AccountTypePersonal,
		claims.UserID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrTokenAccountNotFound,
		) {
			utils.JSON(
				w,
				http.StatusOK,
				0,
				"",
				map[string]interface{}{
					"has_account": false,
					"message":     "暂未开通积分账户",
				},
			)
			return
		}

		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			"查询失败",
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
			"has_account": true,
			"account":     account,
			"available_balance": account.Balance -
				account.FrozenAmount,
		},
	)
}
