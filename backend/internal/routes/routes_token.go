package routes

// routes_token.go — Token积分系统路由注册
//
// v128 新增（阶段C · Token/积分系统）
// v172 改造（积分管理三级数据权限隔离）：查询类端点下放 authMW，数据层 TokenScope 收窄
// v172.2 改造（采购记录隔离）：
//   - purchases 端点拆分方法：GET（查询采购记录）下放到 authMW（按 TokenScope 收窄），
//     POST（充值）仍要求 admin（handler 内 claims 二次校验）
//   - 这样 senior/operator 能看到本校/本人采购记录，但只有 admin 能充值
//
// 写操作收口：
//   创建账户/状态/预警配置 → admin（handler 二次校验）
//   充值（purchases POST） → admin（handler 二次校验）
//   积分分配 → admin + senior_operator（handler 二次校验 + senior 来源账户范围校验）
//
// 路由前缀：/api/v1/tokens/

import (
	"net/http"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerTokenRoutes 注册Token积分系统路由
func registerTokenRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	adminOrSchoolAdmin func(http.Handler) http.Handler,
	tokenHandler *handlers.TokenHandler,
) {
	// ========== 我的积分（登录即可）==========
	mux.Handle("/api/v1/tokens/my-account",
		middleware.Chain(http.HandlerFunc(tokenHandler.GetMyTokenAccount), authMW))

	// ========== 概览统计（登录即可，数据按 TokenScope 收窄）==========
	mux.Handle("/api/v1/tokens/overview",
		middleware.Chain(http.HandlerFunc(tokenHandler.GetOverviewStats), authMW))

	// ========== 账户管理 ==========
	// GET 列表（登录即可，TokenScope 收窄）；POST 创建（admin only，handler 二次校验）
	mux.Handle("/api/v1/tokens/accounts", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tokenHandler.ListAccounts(w, r)
		case http.MethodPost:
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || claims.Role != roleAdmin {
				forbiddenJSON(w, "仅管理员可创建账户")
				return
			}
			tokenHandler.CreateAccount(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	// 账户详情 + 状态更新 + 分配 + 预警配置（通配分发）
	mux.Handle("/api/v1/tokens/accounts/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// /accounts/{id}/allocate → 分配积分（admin + senior_operator）
		if hasSuffix(path, "/allocate") {
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员或学校管理员可分配积分")
				return
			}
			tokenHandler.AllocateTokens(w, r)
			return
		}

		// /accounts/{id}/status → 更新状态（admin only）
		if hasSuffix(path, "/status") {
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || claims.Role != roleAdmin {
				forbiddenJSON(w, "仅管理员可更新账户状态")
				return
			}
			tokenHandler.UpdateAccountStatus(w, r)
			return
		}

		// /accounts/{id}/alert-config → 预警配置（admin only）
		if hasSuffix(path, "/alert-config") {
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || claims.Role != roleAdmin {
				forbiddenJSON(w, "仅管理员可管理预警配置")
				return
			}
			switch r.Method {
			case http.MethodGet:
				tokenHandler.GetAlertConfig(w, r)
			case http.MethodPut:
				tokenHandler.UpdateAlertConfig(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
			return
		}

		// /accounts/{id} → 账户详情（登录即可，handler 内做范围校验 + region 硬拒绝）
		tokenHandler.GetAccountDetail(w, r)
	}), authMW))

	// ========== 分配记录（登录即可，TokenScope 收窄）==========
	mux.Handle("/api/v1/tokens/allocations",
		middleware.Chain(http.HandlerFunc(tokenHandler.ListAllocations), authMW))

	// ========== 采购/充值 ==========
	// v172.2：GET 查询采购记录（登录即可，TokenScope 收窄）；POST 充值（admin only，handler 二次校验）
	mux.Handle("/api/v1/tokens/purchases", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tokenHandler.ListPurchases(w, r)
		case http.MethodPost:
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || claims.Role != roleAdmin {
				forbiddenJSON(w, "仅管理员可充值")
				return
			}
			tokenHandler.PurchaseTokens(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	// ========== 消费流水（登录即可，TokenScope 收窄）==========
	mux.Handle("/api/v1/tokens/consumption",
		middleware.Chain(http.HandlerFunc(tokenHandler.ListConsumptionLogs), authMW))
}
