package routes

// routes_lessonplan.go — 教案系统路由注册
//
// 迭代4B-2 新增：/api/v1/lesson-plans/recipes/smart-recommend — 画像感知智能推荐
// 迭代5 新增：/api/v1/lesson-plans/recipes/{id}/custom-stages — 自定义阶段CRUD
// 迭代6 新增：/api/v1/lesson-plans/recipes/market — 配方市场排行榜
// 迭代6 新增：/api/v1/lesson-plans/recipes/{id}/stats — 配方效果统计
// 迭代7 新增：/api/v1/lesson-plans/textbooks — 课本上传管理

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
	recipeHandler *handlers.RecipeHandler,
	wsStageHandler *handlers.WorkshopStageHandler,
	assessHandler *handlers.AssessmentHandler,
	tbHandler *handlers.TextbookHandler,
) {
	// ---- 教案SSE推送（内部JWT验证）----
	mux.HandleFunc("/api/v1/lesson-plans/sse/", lpGenHandler.StreamPlan)

	// ---- 开始对话（特殊路径，需在/plans/前注册）----
	mux.Handle("/api/v1/lesson-plans/plans/start-conversation",
		middleware.Chain(http.HandlerFunc(lpGenHandler.StartConversation), authMW))

	// ==================== 阶段化备课工坊（Phase 7B 新增）====================

	mux.Handle("/api/v1/lesson-plans/workshop/stages/defaults",
		middleware.Chain(http.HandlerFunc(wsStageHandler.GetDefaultStages), authMW))

	// ==================== 教学风格前测（迭代3新增）====================

	mux.Handle("/api/v1/lesson-plans/assessment/start",
		middleware.Chain(http.HandlerFunc(assessHandler.StartAssessment), authMW))

	mux.Handle("/api/v1/lesson-plans/assessment/chat",
		middleware.Chain(http.HandlerFunc(assessHandler.ChatAssessment), authMW))

	mux.Handle("/api/v1/lesson-plans/assessment/submit",
		middleware.Chain(http.HandlerFunc(assessHandler.SubmitAssessment), authMW))

	mux.Handle("/api/v1/lesson-plans/assessment/skip",
		middleware.Chain(http.HandlerFunc(assessHandler.SkipAssessment), authMW))

	mux.Handle("/api/v1/lesson-plans/assessment/result",
		middleware.Chain(http.HandlerFunc(assessHandler.GetAssessmentResult), authMW))

	mux.Handle("/api/v1/lesson-plans/assessment/auto-recipe",
		middleware.Chain(http.HandlerFunc(assessHandler.AutoGenerateRecipe), authMW))

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

	mux.Handle("/api/v1/lesson-plans/my-groups",
		middleware.Chain(http.HandlerFunc(orgHandler.GetUserTeachingGroups), authMW))

	// ==================== 迭代7：课本上传 ====================

	// 上传接口（multipart，需在通配符路由前注册）
	mux.Handle("/api/v1/lesson-plans/textbooks/upload",
		middleware.Chain(http.HandlerFunc(tbHandler.UploadTextbook), authMW))

	// 课本列表
	mux.Handle("/api/v1/lesson-plans/textbooks", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tbHandler.ListTextbooks(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET请求")
		}
	}), authMW))

	// 课本子路由：详情/更新/删除/OCR
	mux.Handle("/api/v1/lesson-plans/textbooks/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case hasSuffix(path, "/ocr") && r.Method == http.MethodPost:
			tbHandler.TriggerOCR(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				tbHandler.GetTextbook(w, r)
			case http.MethodPut:
				tbHandler.UpdateTextbook(w, r)
			case http.MethodDelete:
				tbHandler.DeleteTextbook(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

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

	// ==================== 备课配方 ====================

	// 迭代2：预设流程模板（需在 /recipes/ 和 /recipes/recommend 前注册）
	mux.Handle("/api/v1/lesson-plans/recipes/flow-presets",
		middleware.Chain(http.HandlerFunc(recipeHandler.GetFlowPresets), authMW))

	// 迭代2：校验流程完整性
	mux.Handle("/api/v1/lesson-plans/recipes/validate-flow",
		middleware.Chain(http.HandlerFunc(recipeHandler.ValidateFlow), authMW))

	// 迭代4B-2新增：画像感知智能推荐（需在 /recipes/ 和 /recipes/recommend 前注册）
	mux.Handle("/api/v1/lesson-plans/recipes/smart-recommend",
		middleware.Chain(http.HandlerFunc(recipeHandler.SmartRecommendComponents), authMW))

	// 迭代6新增：配方市场排行榜
	mux.Handle("/api/v1/lesson-plans/recipes/market",
		middleware.Chain(http.HandlerFunc(recipeHandler.ListMarketRecipes), authMW))

	// 智能推荐（原始版本）
	mux.Handle("/api/v1/lesson-plans/recipes/recommend",
		middleware.Chain(http.HandlerFunc(recipeHandler.RecommendComponents), authMW))

	// 配方列表 + 创建
	mux.Handle("/api/v1/lesson-plans/recipes", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			recipeHandler.ListRecipes(w, r)
		case http.MethodPost:
			recipeHandler.CreateRecipe(w, r)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}
	}), authMW))

	// 配方子路由（迭代5：custom-stages必须在通用/{id}路由之前匹配）
	mux.Handle("/api/v1/lesson-plans/recipes/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// ---- 迭代5新增：自定义阶段CRUD ----
		// /recipes/{id}/custom-stages/{code} (PUT/DELETE)
		case indexOf(path, "/custom-stages/") >= 0:
			switch r.Method {
			case http.MethodPut:
				wsStageHandler.UpdateCustomStage(w, r)
			case http.MethodDelete:
				wsStageHandler.DeleteCustomStage(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持PUT/DELETE请求")
			}
		// /recipes/{id}/custom-stages (GET/POST)
		case hasSuffix(path, "/custom-stages"):
			switch r.Method {
			case http.MethodGet:
				wsStageHandler.ListCustomStages(w, r)
			case http.MethodPost:
				wsStageHandler.CreateCustomStage(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/POST请求")
			}
		// ---- 原有配方子路由 ----
		case hasSuffix(path, "/fork") && r.Method == http.MethodPost:
			recipeHandler.ForkRecipe(w, r)
		case hasSuffix(path, "/share") && r.Method == http.MethodPut:
			recipeHandler.ShareRecipe(w, r)
		case hasSuffix(path, "/student-profile") && r.Method == http.MethodPut:
			recipeHandler.UpdateStudentProfile(w, r)
		case hasSuffix(path, "/preview-context") && r.Method == http.MethodGet:
			recipeHandler.PreviewContext(w, r)
		// 迭代6新增：配方效果统计
		case hasSuffix(path, "/stats") && r.Method == http.MethodGet:
			recipeHandler.GetRecipeStats(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				recipeHandler.GetRecipe(w, r)
			case http.MethodPut:
				recipeHandler.UpdateRecipe(w, r)
			case http.MethodDelete:
				recipeHandler.DeleteRecipe(w, r)
			default:
				methodNotAllowedJSON(w, "仅支持GET/PUT/DELETE请求")
			}
		}
	}), authMW))

	// ==================== 教案CRUD + 阶段操作 ====================

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
				case hasSuffix(path, "/stages/switch") && r.Method == http.MethodPost:
					wsStageHandler.SwitchToStage(w, r)
		case hasSuffix(path, "/stages/reset") && r.Method == http.MethodPost:
				wsStageHandler.ResetStage(w, r)
			case hasSuffix(path, "/stages/advance") && r.Method == http.MethodPost:
			wsStageHandler.AdvanceStage(w, r)
		case hasSuffix(path, "/stages/skip") && r.Method == http.MethodPost:
			wsStageHandler.SkipStage(w, r)
		case hasSuffix(path, "/stages/back") && r.Method == http.MethodPost:
			wsStageHandler.BackStage(w, r)
		// 迭代12新增：阶段推荐组件
			case hasSuffix(path, "/recommended-components") && indexOf(path, "/stages/") >= 0 && r.Method == http.MethodGet:
				wsStageHandler.GetStageRecommendedComponents(w, r)
			case hasSuffix(path, "/output") && indexOf(path, "/stages/") >= 0 && r.Method == http.MethodGet:
				wsStageHandler.GetStageOutput(w, r)
		case hasSuffix(path, "/stages") && r.Method == http.MethodGet:
			wsStageHandler.GetStageStatus(w, r)
		case hasSuffix(path, "/chat"):
			lpGenHandler.Chat(w, r)
		case hasSuffix(path, "/trigger-review"):
			lpGenHandler.TriggerAIReview(w, r)
		case hasSuffix(path, "/apply-suggestions"):
			lpGenHandler.ApplyAISuggestions(w, r)
		case hasSuffix(path, "/conversation"):
			lpGenHandler.GetConversation(w, r)
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
