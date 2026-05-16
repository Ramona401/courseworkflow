package routes

// routes_token.go — Token积分系统路由注册
//
// v128 新增（阶段C · Token/积分系统）：
//   - 概览统计（admin + senior_operator）
//   - 账户管理（admin创建/状态更新，admin+senior查看）
//   - 积分分配（admin）
//   - 采购/充值（admin）
//   - 消费流水（admin + senior_operator）
//   - 预警配置（admin）
//   - 我的积分（登录即可）
//
// 路由前缀：/api/v1/tokens/

import (
	"net/http"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerTokenRoutes 注册Token积分系统路由
// 参数说明：
//   - authMW: 认证中间件
//   - adminOnly: 仅admin可访问
//   - adminOrSchoolAdmin: admin + senior_operator可访问
//   - tokenHandler: Token处理器
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

	// ========== 概览统计（admin + senior_operator）==========
	mux.Handle("/api/v1/tokens/overview",
		middleware.Chain(http.HandlerFunc(tokenHandler.GetOverviewStats), authMW, adminOrSchoolAdmin))

	// ========== 账户管理 ==========
	// 创建账户（admin only）
	// 列表查看（admin + senior_operator）
	mux.Handle("/api/v1/tokens/accounts", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tokenHandler.ListAccounts(w, r)
		case http.MethodPost:
			// 创建账户需admin权限，在此做二次检查
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || claims.Role != roleAdmin {
				forbiddenJSON(w, "仅管理员可创建账户")
				return
			}
			tokenHandler.CreateAccount(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, adminOrSchoolAdmin))

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

		// /accounts/{id}/alert-config → 预警配置
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

		// /accounts/{id} → 账户详情
		tokenHandler.GetAccountDetail(w, r)
	}), authMW, adminOrSchoolAdmin))

	// ========== 分配记录（admin + senior_operator）==========
	mux.Handle("/api/v1/tokens/allocations",
		middleware.Chain(http.HandlerFunc(tokenHandler.ListAllocations), authMW, adminOrSchoolAdmin))

	// ========== 采购/充值（admin only）==========
	mux.Handle("/api/v1/tokens/purchases", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tokenHandler.ListPurchases(w, r)
		case http.MethodPost:
			tokenHandler.PurchaseTokens(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, adminOnly))

	// ========== 消费流水（admin + senior_operator）==========
	mux.Handle("/api/v1/tokens/consumption",
		middleware.Chain(http.HandlerFunc(tokenHandler.ListConsumptionLogs), authMW, adminOrSchoolAdmin))
}
