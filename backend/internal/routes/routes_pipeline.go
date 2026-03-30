package routes

// routes_pipeline.go — Pipeline相关路由注册
//
// 注册的路由：
//   GET  /api/v1/sse/pipelines/{id}/stream  — SSE实时推送
//   GET/POST /api/v1/pipelines              — 列表/创建
//   /api/v1/pipelines/{id}/*               — 详情/各操作（含批量）

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerPipelineRoutes 注册Pipeline相关所有路由
func registerPipelineRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	pipelineHandler *handlers.PipelineHandler,
	sseHandler *handlers.SSEHandler,
) {
	// SSE实时推送（内部JWT验证，不走authMW）
	mux.HandleFunc("/api/v1/sse/pipelines/", sseHandler.StreamPipeline)

	// Pipeline列表/创建
	mux.Handle("/api/v1/pipelines", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pipelineHandler.ListPipelines(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可创建Pipeline")
				return
			}
			pipelineHandler.CreatePipeline(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	// Pipeline子路由（含所有批量操作和步骤操作）
	mux.Handle("/api/v1/pipelines/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// ---- 批量操作 ----
		case hasSuffix(path, "/batch-verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可触发批量验收")
				return
			}
			pipelineHandler.BatchVerify(w, r)

		case hasSuffix(path, "/batch-create"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量创建Pipeline")
				return
			}
			pipelineHandler.BatchCreate(w, r)

		case hasSuffix(path, "/batch-assign"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可分配审核任务")
				return
			}
			pipelineHandler.BatchAssign(w, r)

		case hasSuffix(path, "/batch-restart"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可批量重跑Pipeline")
				return
			}
			pipelineHandler.BatchRestartFromStep(w, r)

		case hasSuffix(path, "/batch-start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可批量启动Pipeline")
				return
			}
			pipelineHandler.BatchStart(w, r)

		case hasSuffix(path, "/operators"):
			pipelineHandler.GetOperators(w, r)

		// ---- 单Pipeline操作 ----
		case hasSuffix(path, "/start"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可启动Pipeline")
				return
			}
			pipelineHandler.StartPipeline(w, r)

		case hasSuffix(path, "/cancel"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅管理员和高级操作员可取消Pipeline")
				return
			}
			pipelineHandler.CancelPipeline(w, r)

		case hasSuffix(path, "/restart-from"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可重启Pipeline步骤")
				return
			}
			pipelineHandler.RestartFromStep(w, r)

		case hasSuffix(path, "/force-proceed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可强制推进Pipeline")
				return
			}
			pipelineHandler.ForceProceed(w, r)

		// ---- 定稿操作 ----
		case hasSuffix(path, "/submit-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可提交定稿申请")
				return
			}
			pipelineHandler.SubmitFinalize(w, r)

		case hasSuffix(path, "/confirm-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可确认定稿")
				return
			}
			pipelineHandler.ConfirmFinalize(w, r)

		case hasSuffix(path, "/reject-finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
				forbiddenJSON(w, "仅高级操作员和管理员可退回重审")
				return
			}
			pipelineHandler.RejectFinalize(w, r)

		case hasSuffix(path, "/finalize"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "直接定稿仅管理员可操作")
				return
			}
			pipelineHandler.FinalizePipeline(w, r)

		case hasSuffix(path, "/mark-passed"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可快捷通过Pipeline")
				return
			}
			pipelineHandler.MarkPassed(w, r)

		case hasSuffix(path, "/verify"):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可触发验收")
				return
			}
			pipelineHandler.VerifyPipeline(w, r)

		// ---- 评估/页面/步骤查询 ----
		case hasSuffix(path, "/eval-rounds"):
			pipelineHandler.GetEvalRounds(w, r)

		case containsPagesAIFix(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可使用AI快修")
				return
			}
			pipelineHandler.AIFixPage(w, r)

		case containsPagesDecision(path):
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator, roleOperator) {
				forbiddenJSON(w, "仅管理员和操作员可审核页面")
				return
			}
			pipelineHandler.UpdatePageDecision(w, r)

		case hasSuffix(path, "/pages"):
			pipelineHandler.GetGeneratedPages(w, r)

		case containsStepsWithName(path):
			pipelineHandler.GetStepDetail(w, r)

		case hasSuffix(path, "/steps"):
			pipelineHandler.GetSteps(w, r)

		// ---- Pipeline详情/删除 ----
		default:
			switch r.Method {
			case http.MethodGet:
				pipelineHandler.GetPipelineDetail(w, r)
			case http.MethodDelete:
				claims, ok := middleware.GetClaims(r.Context())
				if !ok || !hasRole(claims.Role, roleAdmin, roleSeniorOperator) {
					forbiddenJSON(w, "仅管理员和高级操作员可删除Pipeline")
					return
				}
				pipelineHandler.DeletePipeline(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/DELETE请求")
			}
		}
	}), authMW))
}
