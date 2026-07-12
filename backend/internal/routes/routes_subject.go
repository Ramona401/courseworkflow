package routes

// routes_subject.go — 学科字典路由注册（v231）
//
// 两组：
//   公开只读（authMW，登录即可）：GET /api/v1/subjects — 前端各下拉统一消费。
//   管理 CRUD（authMW + adminOnly）：/api/v1/admin/subjects 与 /api/v1/admin/subjects/{id}
//     GET 列表(含停用) / POST 新建 / PUT 编辑 / DELETE 删除（内置学科禁删）。
//
// 风格对齐 routes_curriculum.go（公开只读）与 routes_kb.go 的 kbAdminMux（admin 按 path+method 分发）。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerSubjectRoutes 注册学科字典公开只读 + admin 管理路由
func registerSubjectRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	subjectHandler *handlers.SubjectHandler,
) {
	// ==================== 公开只读（登录即可）====================
	mux.Handle("/api/v1/subjects", middleware.Chain(
		http.HandlerFunc(subjectHandler.ListPublic), authMW))

	// ==================== 管理 CRUD（admin only）====================
	adminMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimRight(r.URL.Path, "/")

		// 带 {id} 的编辑/删除：/api/v1/admin/subjects/{id}
		if strings.HasPrefix(path, "/api/v1/admin/subjects/") {
			switch r.Method {
			case http.MethodPut:
				subjectHandler.Update(w, r)
			case http.MethodDelete:
				subjectHandler.Delete(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		// 集合级：/api/v1/admin/subjects
		if path == "/api/v1/admin/subjects" {
			switch r.Method {
			case http.MethodGet:
				subjectHandler.ListAdmin(w, r)
			case http.MethodPost:
				subjectHandler.Create(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
	}), authMW, adminOnly)

	mux.Handle("/api/v1/admin/subjects", adminMux)
	mux.Handle("/api/v1/admin/subjects/", adminMux)
}
