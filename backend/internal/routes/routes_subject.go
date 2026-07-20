package routes

// routes_subject.go — 学科字典与组织教育域基础数据路由
//
// 学科字典：
//   GET /api/v1/subjects
//       登录用户按自身教育域和教学组织读取课程目录；
//
//   GET/POST /api/v1/admin/subjects
//   PUT/DELETE /api/v1/admin/subjects/{id}
//       admin管理统一课程定义。
//
// 组织教育域：
//   GET /api/v1/admin/organization-education-domains
//       admin只读查看全部组织当前教育域；
//
//   PUT /api/v1/admin/organization-education-domains/{id}
//       保留旧客户端兼容路由；
//       仅超级管理员可以到达；
//       对真实组织统一返回409 Conflict，不执行任何修改。
//
// 处理器和服务在本注册函数中自构造，保持routes.go主初始化零改动。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

func registerSubjectRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	subjectHandler *handlers.SubjectHandler,
) {
	// ==================== 公开课程目录 ====================

	mux.Handle(
		"/api/v1/subjects",
		middleware.Chain(
			http.HandlerFunc(subjectHandler.ListPublic),
			authMW,
		),
	)

	// ==================== 统一课程定义管理 ====================

	subjectAdminMux := middleware.Chain(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			path := strings.TrimRight(r.URL.Path, "/")

			if strings.HasPrefix(
				path,
				"/api/v1/admin/subjects/",
			) {
				switch r.Method {
				case http.MethodPut:
					subjectHandler.Update(w, r)
				case http.MethodDelete:
					subjectHandler.Delete(w, r)
				default:
					http.Error(
						w,
						`{"code":-1,"message":"Method not allowed"}`,
						http.StatusMethodNotAllowed,
					)
				}
				return
			}

			if path == "/api/v1/admin/subjects" {
				switch r.Method {
				case http.MethodGet:
					subjectHandler.ListAdmin(w, r)
				case http.MethodPost:
					subjectHandler.Create(w, r)
				default:
					http.Error(
						w,
						`{"code":-1,"message":"Method not allowed"}`,
						http.StatusMethodNotAllowed,
					)
				}
				return
			}

			http.Error(
				w,
				`{"code":-1,"message":"未找到路由"}`,
				http.StatusNotFound,
			)
		}),
		authMW,
		adminOnly,
	)

	mux.Handle(
		"/api/v1/admin/subjects",
		subjectAdminMux,
	)
	mux.Handle(
		"/api/v1/admin/subjects/",
		subjectAdminMux,
	)

	// ==================== 组织教育域只读管理 ====================

	educationDomainService :=
		services.NewOrganizationEducationDomainService()

	educationDomainHandler :=
		handlers.NewOrganizationEducationDomainHandler(
			educationDomainService,
		)

	// 查看：所有admin均可读取。
	mux.Handle(
		"/api/v1/admin/organization-education-domains",
		middleware.Chain(
			http.HandlerFunc(educationDomainHandler.List),
			authMW,
			adminOnly,
		),
	)

	// 旧修改路由：超级管理员可到达，但Handler固定返回409。
	//
	// 继续保留权限墙，避免普通admin利用错误响应探测随机组织ID。
	superAdminOnly := middleware.SuperAdminOnly()

	mux.Handle(
		"/api/v1/admin/organization-education-domains/",
		middleware.Chain(
			http.HandlerFunc(educationDomainHandler.Update),
			authMW,
			adminOnly,
			superAdminOnly,
		),
	)
}
