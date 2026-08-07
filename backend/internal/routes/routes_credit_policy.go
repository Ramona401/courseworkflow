package routes

// routes_credit_policy.go — 积分策略及价格同步路由。
//
// 全部接口属于积分成本和模型价格配置能力，仅超级管理员可访问。
//
// 价格同步服务使用主路由传入的同一份运行配置构造，避免再次加载.env。
// 调度器启动仍严格受以下条件控制：
//   - 主配置DisableSchedulers必须为false；
//   - 数据库price_sync_enabled必须为true；
//   - 单个价格目标auto_sync_enabled必须为true。
//
// 本文件不参与图片、视频或TTS业务结算。

import (
	"net/http"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

// registerCreditPolicyRoutes 注册积分策略及价格同步路由。
func registerCreditPolicyRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	handler *handlers.CreditPolicyHandler,
	cfg *config.Config,
) {
	superAdmin := middleware.SuperAdminOnly()

	secure := func(
		handler http.Handler,
	) http.Handler {
		return middleware.Chain(
			handler,
			authMW,
			adminOnly,
			superAdmin,
		)
	}

	// cfg由Setup统一加载并注入。价格同步Handler和调度器共享同一个
	// PriceSyncService实例，确保AES密钥和DisableSchedulers口径一致。
	priceSyncService := services.NewPriceSyncService(cfg)
	priceSyncHandler := handlers.NewPriceSyncHandler(
		priceSyncService,
	)

	if cfg != nil && !cfg.DisableSchedulers {
		priceSyncService.StartScheduler()
	}

	// ==================== 积分策略 ====================

	mux.Handle(
		"/api/v1/tokens/credit-policies",
		secure(
			http.HandlerFunc(
				handler.ListPolicies,
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/credit-policies/system",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					switch request.Method {
					case http.MethodGet:
						handler.GetSystemPolicy(
							writer,
							request,
						)

					case http.MethodPut:
						handler.UpdateSystemPolicy(
							writer,
							request,
						)

					default:
						methodNotAllowedJSON(
							writer,
							"仅支持GET/PUT请求",
						)
					}
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/credit-policies/school/",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					switch request.Method {
					case http.MethodGet:
						handler.GetSchoolPolicy(
							writer,
							request,
						)

					case http.MethodPut:
						handler.UpdateSchoolPolicy(
							writer,
							request,
						)

					case http.MethodDelete:
						claims, _ :=
							middleware.GetClaims(
								request.Context(),
							)

						if claims == nil ||
							claims.Role != roleAdmin ||
							!claims.IsSuper {
							forbiddenJSON(
								writer,
								"仅超级管理员可删除学校策略",
							)
							return
						}

						handler.DeleteSchoolPolicy(
							writer,
							request,
						)

					default:
						methodNotAllowedJSON(
							writer,
							"仅支持GET/PUT/DELETE请求",
						)
					}
				},
			),
		),
	)

	// ==================== 模型价格管理 ====================

	mux.Handle(
		"/api/v1/tokens/model-prices",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					switch request.Method {
					case http.MethodGet:
						handler.ListModelPrices(
							writer,
							request,
						)

					case http.MethodPost:
						handler.CreateModelPrice(
							writer,
							request,
						)

					default:
						methodNotAllowedJSON(
							writer,
							"仅支持GET/POST请求",
						)
					}
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/model-prices/",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					switch request.Method {
					case http.MethodPut:
						handler.UpdateModelPrice(
							writer,
							request,
						)

					case http.MethodDelete:
						handler.DeleteModelPrice(
							writer,
							request,
						)

					default:
						methodNotAllowedJSON(
							writer,
							"仅支持PUT/DELETE请求",
						)
					}
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/model-previews",
		secure(
			http.HandlerFunc(
				handler.GetModelPreviews,
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/simulate",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodPost {
						methodNotAllowedJSON(
							writer,
							"仅支持POST请求",
						)
						return
					}

					handler.Simulate(
						writer,
						request,
					)
				},
			),
		),
	)

	// ==================== 价格同步配置 ====================

	mux.Handle(
		"/api/v1/tokens/price-sync/settings",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					switch request.Method {
					case http.MethodGet:
						priceSyncHandler.GetSettings(
							writer,
							request,
						)

					case http.MethodPut:
						priceSyncHandler.UpdateSettings(
							writer,
							request,
						)

					default:
						methodNotAllowedJSON(
							writer,
							"仅支持GET/PUT请求",
						)
					}
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/price-sync/targets/",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodPut {
						methodNotAllowedJSON(
							writer,
							"仅支持PUT请求",
						)
						return
					}

					priceSyncHandler.UpdateTarget(
						writer,
						request,
					)
				},
			),
		),
	)

	// ==================== 价格同步执行 ====================

	mux.Handle(
		"/api/v1/tokens/price-sync/preview",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodPost {
						methodNotAllowedJSON(
							writer,
							"仅支持POST请求",
						)
						return
					}

					priceSyncHandler.Preview(
						writer,
						request,
					)
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/price-sync/apply",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodPost {
						methodNotAllowedJSON(
							writer,
							"仅支持POST请求",
						)
						return
					}

					priceSyncHandler.Apply(
						writer,
						request,
					)
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/price-sync/runs",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodGet {
						methodNotAllowedJSON(
							writer,
							"仅支持GET请求",
						)
						return
					}

					priceSyncHandler.ListRuns(
						writer,
						request,
					)
				},
			),
		),
	)

	mux.Handle(
		"/api/v1/tokens/price-sync/runs/",
		secure(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if request.Method != http.MethodGet {
						methodNotAllowedJSON(
							writer,
							"仅支持GET请求",
						)
						return
					}

					priceSyncHandler.GetRunDetail(
						writer,
						request,
					)
				},
			),
		),
	)
}
