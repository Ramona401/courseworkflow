package routes

// routes_token.go — Token积分系统路由注册
//
// v128 新增（阶段C · Token/积分系统）
// v172 改造（积分管理三级数据权限隔离）：查询类端点下放 authMW，数据层 TokenScope 收窄
// v172.2 改造（采购记录隔离）：purchases GET 下放 authMW，POST 仍 admin
// 究极彻底版·批次1：/accounts/{id}/allocatable-targets
// region_admin 区域分配 batch：allocate / allocatable-targets 白名单加 roleRegionAdmin
//
// 积分消费汇总报告 batch 新增（本次）：
//   - /api/v1/tokens/consumption-summary（GET）→ 消费汇总报告（登录即可，TokenScope 收窄）。
//     5 维度(school/user/model/scene/time) + 时间范围 + 下钻过滤,数据层按角色自动收窄。
//
// 写操作收口：
//   创建账户/状态/预警配置 → admin（handler 二次校验）
//   充值（purchases POST） → admin（handler 二次校验）
//   积分分配 / 可分配下级查询 → admin + senior_operator + region_admin
//
// 路由前缀：/api/v1/tokens/

import (
	"net/http"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// roleRegionAdmin 区域管理员角色码（与 models.RoleRegionAdmin 一致；
// 包内局部常量，与 routes.go 既有 roleAdmin/roleSeniorOperator 风格一致，避免引入 models import）。
const roleRegionAdmin = "region_admin"

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

	// ========== 消费汇总报告（登录即可，TokenScope 收窄）==========
	mux.Handle("/api/v1/tokens/consumption-summary",
		middleware.Chain(http.HandlerFunc(tokenHandler.GetConsumptionSummary), authMW))

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

	// 账户详情 + 状态更新 + 分配 + 可分配下级 + 预警配置（通配分发）
	mux.Handle("/api/v1/tokens/accounts/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// /accounts/{id}/allocate → 分配积分（admin + senior_operator + region_admin）
		if hasSuffix(path, "/allocate") {
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleRegionAdmin) {
				forbiddenJSON(w, "仅管理员、学校管理员或区域管理员可分配积分")
				return
			}
			tokenHandler.AllocateTokens(w, r)
			return
		}

		// /accounts/{id}/allocatable-targets → 查询可分配下级（admin + senior_operator + region_admin）
		// 注意：必须置于兜底 GetAccountDetail 之前，否则会被当作账户详情请求。
		if hasSuffix(path, "/allocatable-targets") {
			claims, _ := middleware.GetClaims(r.Context())
			if claims == nil || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleRegionAdmin) {
				forbiddenJSON(w, "仅管理员、学校管理员或区域管理员可查询可分配下级")
				return
			}
			tokenHandler.GetAllocatableTargets(w, r)
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
	// GET 查询采购记录（登录即可，TokenScope 收窄）；POST 充值（admin only，handler 二次校验）
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
