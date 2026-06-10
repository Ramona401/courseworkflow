package routes

// routes_kb.go — 知识库课标压缩入库系统路由注册
//
// 访问控制：业务路由挂 authMW + RequireKBAuthorized()（admin 恒通过+名单内通过）；
//          SSE 路由不走 Chain（EventSource 不能设 header），在 handler 内部验 token + 查白名单；
//          白名单管理路由挂 authMW + adminOnly（仅 admin 增删查白名单）。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerKBRoutes 注册知识库压缩全部路由
func registerKBRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	compressHandler *handlers.KBCompressHandler,
	reviewHandler *handlers.KBReviewHandler,
	adminHandler *handlers.KBAdminHandler,
) {
	kbGuard := middleware.RequireKBAuthorized()

	// ==================== SSE 压缩进度（内部验 token + 查白名单，不走 authMW）====================
	mux.HandleFunc("/api/v1/sse/kb/", compressHandler.ProgressStream)

	// ==================== KB 业务路由（authMW + 白名单守卫）====================
	kbMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 蓝绿切换：/api/v1/kb/switch-batch
		if path == "/api/v1/kb/switch-batch" || path == "/api/v1/kb/switch-batch/" {
			reviewHandler.SwitchBatch(w, r)
			return
		}

		// 单元审核动作：/api/v1/kb/items/{id}/review
		if strings.HasPrefix(path, "/api/v1/kb/items/") && strings.HasSuffix(strings.TrimRight(path, "/"), "/review") {
			reviewHandler.ReviewAction(w, r)
			return
		}

		// 任务相关：/api/v1/kb/jobs...
		if strings.HasPrefix(path, "/api/v1/kb/jobs") {
			trimmed := strings.TrimRight(path, "/")
			// 审核队列：/jobs/{id}/review-queue
			if strings.HasSuffix(trimmed, "/review-queue") {
				reviewHandler.GetReviewQueue(w, r)
				return
			}
			// 灌入：/jobs/{id}/commit
			if strings.HasSuffix(trimmed, "/commit") {
				reviewHandler.CommitBatch(w, r)
				return
			}
			// 列表/创建：/jobs（无尾段）
			if trimmed == "/api/v1/kb/jobs" {
				switch r.Method {
				case http.MethodGet:
					compressHandler.ListJobs(w, r)
				case http.MethodPost:
					compressHandler.CreateJob(w, r)
				default:
					http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
				}
				return
			}
			// 详情：/jobs/{id}
			compressHandler.GetJob(w, r)
			return
		}

		http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
	}), authMW, kbGuard)

	mux.Handle("/api/v1/kb/jobs", kbMux)
	mux.Handle("/api/v1/kb/jobs/", kbMux)
	mux.Handle("/api/v1/kb/items/", kbMux)
	mux.Handle("/api/v1/kb/switch-batch", kbMux)

	// ==================== 白名单管理（admin only，挂 /admin 下）====================
	kbAdminMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimRight(r.URL.Path, "/")
		// 移除：DELETE /api/v1/admin/kb-authorized/{user_id}
		if strings.HasPrefix(path, "/api/v1/admin/kb-authorized/") && r.Method == http.MethodDelete {
			adminHandler.RemoveAuthorized(w, r)
			return
		}
		// 列表/添加：/api/v1/admin/kb-authorized
		if path == "/api/v1/admin/kb-authorized" {
			switch r.Method {
			case http.MethodGet:
				adminHandler.ListAuthorized(w, r)
			case http.MethodPost:
				adminHandler.AddAuthorized(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
	}), authMW, adminOnly)

	mux.Handle("/api/v1/admin/kb-authorized", kbAdminMux)
	mux.Handle("/api/v1/admin/kb-authorized/", kbAdminMux)
}
