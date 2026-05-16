package routes

// routes_credit_policy.go — 积分策略路由注册
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）
// v129.1 修改：所有策略接口仅admin可访问
//   - 策略列表/系统策略/学校策略（admin only）
//   - 模型单价管理（admin only）
//   - 模型积分预览/模拟计算（admin only）
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
	// ========== 策略列表（admin only）==========
	mux.Handle("/api/v1/tokens/credit-policies",
		middleware.Chain(http.HandlerFunc(handler.ListPolicies), authMW, adminOnly))

	// ========== 系统策略（admin only）==========
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
		}), authMW, adminOnly))

	// ========== 学校策略（admin only）==========
	mux.Handle("/api/v1/tokens/credit-policies/school/",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.GetSchoolPolicy(w, r)
			case http.MethodPut:
				handler.UpdateSchoolPolicy(w, r)
			case http.MethodDelete:
				// 删除仅admin
				claims, _ := middleware.GetClaims(r.Context())
				if claims == nil || claims.Role != roleAdmin {
					forbiddenJSON(w, "仅管理员可删除学校策略")
					return
				}
				handler.DeleteSchoolPolicy(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}), authMW, adminOnly))

	// ========== 模型单价管理（admin only）==========
	mux.Handle("/api/v1/tokens/model-prices",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.ListModelPrices(w, r)
			case http.MethodPost:
				claims, _ := middleware.GetClaims(r.Context())
				if claims == nil || claims.Role != roleAdmin {
					forbiddenJSON(w, "仅管理员可创建模型单价")
					return
				}
				handler.CreateModelPrice(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		}), authMW, adminOnly))

	// 单个模型单价更新/删除（admin only）
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
		}), authMW, adminOnly))

	// ========== 模型积分预览（admin only）==========
	mux.Handle("/api/v1/tokens/model-previews",
		middleware.Chain(http.HandlerFunc(handler.GetModelPreviews), authMW, adminOnly))

	// ========== 模拟计算（admin only）==========
	mux.Handle("/api/v1/tokens/simulate",
		middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				methodNotAllowedJSON(w, "仅支持POST请求")
				return
			}
			handler.Simulate(w, r)
		}), authMW, adminOnly))
}
