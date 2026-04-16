package routes

// routes_school_admin.go — 学校管理员路由注册
//
// 权限策略：
// 1) 必须登录
// 2) 角色必须为 senior_operator
// 3) 是否真正绑定学校管理员身份由 handler 内二次校验（organizations.admin_user_id）

import (
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerSchoolAdminRoutes 注册学校管理员路由
func registerSchoolAdminRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	seniorOperatorOnly func(http.Handler) http.Handler,
	schoolAdminHandler *handlers.SchoolAdminHandler,
) {
	// 学校信息
	mux.Handle("/api/v1/school-admin/my-school",
		middleware.Chain(http.HandlerFunc(schoolAdminHandler.GetMySchool), authMW, seniorOperatorOnly))

	// 教师列表/创建
	mux.Handle("/api/v1/school-admin/users", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			schoolAdminHandler.ListSchoolUsers(w, r)
		case http.MethodPost:
			schoolAdminHandler.CreateSchoolUser(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, seniorOperatorOnly))

	// 教师详情/编辑/状态/密码
	mux.Handle("/api/v1/school-admin/users/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/status"):
			schoolAdminHandler.UpdateSchoolUserStatus(w, r)
		case hasSuffix(path, "/password"):
			schoolAdminHandler.ResetSchoolUserPassword(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				schoolAdminHandler.GetSchoolUserDetail(w, r)
			case http.MethodPut:
				schoolAdminHandler.UpdateSchoolUser(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}
	}), authMW, seniorOperatorOnly))

	// 教研组列表/创建
	mux.Handle("/api/v1/school-admin/groups", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			schoolAdminHandler.ListSchoolGroups(w, r)
		case http.MethodPost:
			schoolAdminHandler.CreateSchoolGroup(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW, seniorOperatorOnly))

	// 教研组详情操作 + 成员管理
	mux.Handle("/api/v1/school-admin/groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if containsAdminMemberUID(path) {
			switch r.Method {
			case http.MethodPut:
				schoolAdminHandler.UpdateSchoolGroupMemberRole(w, r)
			case http.MethodDelete:
				schoolAdminHandler.RemoveSchoolGroupMember(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持PUT/DELETE请求")
			}
			return
		}
		if hasSuffix(path, "/members") {
			switch r.Method {
			case http.MethodGet:
				schoolAdminHandler.ListSchoolGroupMembers(w, r)
			case http.MethodPost:
				schoolAdminHandler.AddSchoolGroupMember(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
			return
		}
		switch r.Method {
		case http.MethodPut:
			schoolAdminHandler.UpdateSchoolGroup(w, r)
		case http.MethodDelete:
			schoolAdminHandler.DeleteSchoolGroup(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持PUT/DELETE请求")
		}
	}), authMW, seniorOperatorOnly))
}
