package routes

// routes_credit_policy.go — 积分策略路由注册
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）
// v129.1 修改：所有策略接口仅admin可访问
//   - 策略列表/系统策略/学校策略（admin only）
//   - 模型单价管理（admin only）
//   - 模型积分预览/模拟计算（admin only）
//
// 超管收口（本批）：本文件全部路由属"积分金融命脉配置"（策略/模型单价/预览/模拟），
//   从 adminOnly 整体收紧为超管专属——每条 Chain 末尾追加 superAdmin(SuperAdminOnly)，
//   二线管理员(admin 但 is_super=false)被拦。学校策略 DELETE 分支内层原有的
//   Role != roleAdmin 判定同步收紧为 || !claims.IsSuper，保持与外层中间件语义一致。
//   本文件无"查询下放低权限"的情况（全部本就 adminOnly），故整条收口不会误伤任何角色。
//
// 路由前缀：/api/v1/tokens/credit-policies/ 和 /api/v1/tokens/model-prices/

import (
	"net/http"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCreditPolicyRoutes 注册积分策略路由
func registerCreditPolicyRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	handler *handlers.CreditPolicyHandler,
) {
	// 超管收口：超管专属中间件（在 adminOnly 之上再收一层 is_super=true）。
	// 函数内自建，不改函数签名（routes.go 调用处零改动）。
	superAdmin := middleware.SuperAdminOnly()

	// ========== 策略列表（超管 only）==========
	mux.Handle("/api/v1/tokens/credit-policies",
		middleware.Chain(http.HandlerFunc(handler.ListPolicies), authMW, adminOnly, superAdmin))

	// ========== 系统策略（超管 only）==========
	mux.Handle("/api/v1/tokens/credit-policies/system",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.GetSystemPolicy(w, r)
			case http.MethodPut:
				handler.UpdateSystemPolicy(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}), authMW, adminOnly, superAdmin))

	// ========== 学校策略（超管 only）==========
	mux.Handle("/api/v1/tokens/credit-policies/school/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.GetSchoolPolicy(w, r)
			case http.MethodPut:
				handler.UpdateSchoolPolicy(w, r)
			case http.MethodDelete:
				// 删除仅超管（原 Role==admin，收紧为 is_super；与外层 superAdmin 中间件语义一致）
				claims, _ := middleware.GetClaims(r.Context())
				if claims == nil || claims.Role != roleAdmin || !claims.IsSuper {
					forbiddenJSON(w, "仅超级管理员可删除学校策略")
					return
				}
				handler.DeleteSchoolPolicy(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}), authMW, adminOnly, superAdmin))

	// ========== 模型单价管理（超管 only）==========
	mux.Handle("/api/v1/tokens/model-prices",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.ListModelPrices(w, r)
			case http.MethodPost:
				handler.CreateModelPrice(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		}), authMW, adminOnly, superAdmin))

	// ========== 模型单价更新/删除（超管 only）==========
	mux.Handle("/api/v1/tokens/model-prices/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				handler.UpdateModelPrice(w, r)
			case http.MethodDelete:
				handler.DeleteModelPrice(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持PUT/DELETE请求")
			}
		}), authMW, adminOnly, superAdmin))

	// ========== 模型积分预览（超管 only）==========
	mux.Handle("/api/v1/tokens/model-previews",
		middleware.Chain(http.HandlerFunc(handler.GetModelPreviews), authMW, adminOnly, superAdmin))

	// ========== 积分模拟计算（超管 only）==========
	mux.Handle("/api/v1/tokens/simulate",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				methodNotAllowedJSON(w, "仅支持POST请求")
				return
			}
			handler.Simulate(w, r)
		}), authMW, adminOnly, superAdmin))
}
