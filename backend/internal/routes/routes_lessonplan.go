package routes

// routes_lessonplan.go — 教案系统路由注册
//
// 注册的路由：
//   /api/v1/lesson-plans/sse/*           — 教案SSE推送
//   /api/v1/lesson-plans/organizations/* — 组织管理
//   /api/v1/lesson-plans/teaching-groups/* — 教研组管理
//   /api/v1/lesson-plans/my-groups       — 当前用户教研组
//   /api/v1/lesson-plans/components/*    — 组件库
//   /api/v1/lesson-plans/extractions/*   — 萃取队列
//   /api/v1/lesson-plans/plans/*         — 教案CRUD+状态操作+生成对话
//   /api/v1/lesson-plans/templates/*     — 提示词模板

import (
	"encoding/json"
	"net/http"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerLessonPlanRoutes 注册教案系统所有路由
func registerLessonPlanRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	orgHandler *handlers.OrganizationHandler,
	compHandler *handlers.ComponentHandler,
	lpHandler *handlers.LessonPlanHandler,
	lpGenHandler *handlers.LessonPlanGenHandler,
) {
	// ---- 教案SSE推送（内部JWT验证）----
	mux.HandleFunc("/api/v1/lesson-plans/sse/", lpGenHandler.StreamPlan)

	// ---- 开始对话（特殊路径，需在/plans/前注册）----
	mux.Handle("/api/v1/lesson-plans/plans/start-conversation",
		middleware.Chain(http.HandlerFunc(lpGenHandler.StartConversation), authMW))

	// ==================== 组织管理 ====================

	mux.Handle("/api/v1/lesson-plans/organizations", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.ListOrganizations(w, r)
		case http.MethodPost:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可创建组织")
				return
			}
			orgHandler.CreateOrganization(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/organizations/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.GetOrganization(w, r)
		case http.MethodPut:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可更新组织")
				return
			}
			orgHandler.UpdateOrganization(w, r)
		case http.MethodDelete:
			claims, ok := middleware.GetClaims(r.Context())
			if !ok || !hasRole(claims.Role, roleAdmin) {
				forbiddenJSON(w, "仅管理员可删除组织")
				return
			}
			orgHandler.DeleteOrganization(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
		}
	}), authMW))

	// ==================== 教研组管理 ====================

	mux.Handle("/api/v1/lesson-plans/teaching-groups", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			orgHandler.ListTeachingGroups(w, r)
		case http.MethodPost:
			orgHandler.CreateTeachingGroup(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/teaching-groups/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/members") && r.Method == http.MethodPost:
			orgHandler.AddGroupMember(w, r)
		case indexOf(path, "/members/") >= 0 && r.Method == http.MethodDelete:
			orgHandler.RemoveGroupMember(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				orgHandler.GetTeachingGroupDetail(w, r)
			case http.MethodPut:
				orgHandler.UpdateTeachingGroup(w, r)
			case http.MethodDelete:
				orgHandler.DeleteTeachingGroup(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	// 当前用户所属教研组
	mux.Handle("/api/v1/lesson-plans/my-groups",
		middleware.Chain(http.HandlerFunc(orgHandler.GetUserTeachingGroups), authMW))

	// ==================== 组件库 ====================

	mux.Handle("/api/v1/lesson-plans/components", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			compHandler.ListComponents(w, r)
		case http.MethodPost:
			compHandler.CreateComponent(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/components/match",
		middleware.Chain(http.HandlerFunc(compHandler.MatchComponents), authMW))

	mux.Handle("/api/v1/lesson-plans/components/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/review"):
			compHandler.ReviewComponent(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				compHandler.GetComponent(w, r)
			case http.MethodPut:
				compHandler.UpdateComponent(w, r)
			case http.MethodDelete:
				compHandler.DeleteComponent(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	// ==================== 萃取队列 ====================

	mux.Handle("/api/v1/lesson-plans/extractions",
		middleware.Chain(http.HandlerFunc(compHandler.ListExtractions), authMW))

	mux.Handle("/api/v1/lesson-plans/extractions/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if hasSuffix(path, "/confirm") {
			compHandler.ConfirmExtraction(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": "未知的萃取子路径"})
	}), authMW))

	// ==================== 教案CRUD ====================

	mux.Handle("/api/v1/lesson-plans/plans", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lpHandler.ListLessonPlans(w, r)
		case http.MethodPost:
			lpHandler.CreateLessonPlan(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/plans/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// 生成对话类
		case hasSuffix(path, "/chat"):
			lpGenHandler.Chat(w, r)
		case hasSuffix(path, "/trigger-review"):
			lpGenHandler.TriggerAIReview(w, r)
		case hasSuffix(path, "/apply-suggestions"):
			lpGenHandler.ApplyAISuggestions(w, r)
		case hasSuffix(path, "/conversation"):
			lpGenHandler.GetConversation(w, r)
		// 状态操作类
		case hasSuffix(path, "/publish-personal"):
			lpHandler.PublishPersonal(w, r)
		case hasSuffix(path, "/submit-review"):
			lpHandler.SubmitForReview(w, r)
		case hasSuffix(path, "/review"):
			lpHandler.ReviewLessonPlan(w, r)
		case hasSuffix(path, "/publish-shared"):
			lpHandler.PublishShared(w, r)
		case hasSuffix(path, "/start-development"):
			lpHandler.StartDevelopment(w, r)
		case hasSuffix(path, "/fork"):
			lpHandler.ForkLessonPlan(w, r)
		// CRUD
		default:
			switch r.Method {
			case http.MethodGet:
				lpHandler.GetLessonPlan(w, r)
			case http.MethodPut:
				lpHandler.UpdateLessonPlan(w, r)
			case http.MethodDelete:
				lpHandler.DeleteLessonPlan(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	// ==================== 提示词模板 ====================

	mux.Handle("/api/v1/lesson-plans/templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lpHandler.ListPromptTemplates(w, r)
		case http.MethodPost:
			lpHandler.CreatePromptTemplate(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	mux.Handle("/api/v1/lesson-plans/templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/resolved"):
			lpHandler.ResolvePromptTemplate(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				lpHandler.GetPromptTemplate(w, r)
			case http.MethodPut:
				lpHandler.UpdatePromptTemplate(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT请求")
			}
		}
	}), authMW))
}
