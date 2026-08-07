package routes

// routes_courseware_assistant.go
//
// 教师端课件教学智能体安全路由包装。
//
// 设计目标：
//   - 不修改超长routes.go和routes_courseware.go；
//   - 只拦截教学智能体插槽、方案、部署管理和教师预览路径；
//   - 所有教师端路径统一经过生产CORS和JWT认证；
//   - 其它请求原样进入知识点漫画、Style Studio与主路由；
//   - 主路由、Engine和调度器只初始化一次。
//
// 功能开关：
//
//   COURSEWARE_ASSISTANT_ENABLED=true：
//     正常构造并开放教师端教学智能体服务。
//
//   COURSEWARE_ASSISTANT_ENABLED=false：
//     所有教学智能体教师端保留路径统一返回404；
//     不构造本模块教师端服务依赖；
//     非教学智能体课件路径继续下沉，不影响现有课件功能。
//
// 路由下沉原则：
//
//   /api/v1/coursewares/{id}/pages/{page}/... 同时也是课件主路由的公共前缀，
//   generate-image、upload-image、save-html、regenerate等大量非智能体动作
//   都会经过该前缀。
//
//   本包装层只能拦截明确登记的 assistant-* 动作：
//     assistant-slot
//     assistant-context
//     assistant-plan
//     assistant-deployment
//
//   未知但以 assistant- 开头的动作按保留路径fail-closed返回404；
//   不以 assistant- 开头的普通课件页面动作必须返回“未匹配”，
//   继续交给基础路由。
//
// 教师预览：
//
//   POST /api/v1/assistant-deployments/{id}/preview-session
//
//   该入口只创建teacher_preview短时运行会话，不另写聊天实现。
//   后续读取和聊天继续复用assistant-runtime短时令牌接口。
//   教师预览扣部署所有者积分，但不占外部学生每日额度。

import (
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

const (
	coursewareAssistantRouteSlots             = "slots"
	coursewareAssistantRouteSlotItem          = "slot_item"
	coursewareAssistantRoutePageSlot          = "page_slot"
	coursewareAssistantRoutePageContext       = "page_context"
	coursewareAssistantRoutePagePlan          = "page_plan"
	coursewareAssistantRoutePublishDeployment = "publish_deployment"
	coursewareAssistantRouteDeployments       = "deployments"
	coursewareAssistantRouteVersions          = "versions"
	coursewareAssistantRoutePause             = "pause"
	coursewareAssistantRouteResume            = "resume"
	coursewareAssistantRouteRevoke            = "revoke"
	coursewareAssistantRoutePolicy            = "policy"
	coursewareAssistantRoutePreviewSession    = "preview_session"
	coursewareAssistantRouteTTS               = "tts"
	coursewareAssistantRouteInvalid           = "invalid"

	coursewareAssistantTeacherPrefix = "/api/v1/coursewares/"
	assistantDeploymentTeacherRoot   = "/api/v1/assistant-deployments"
)

// coursewareAssistantRouteMatch 是纯路径匹配结果。
//
// Matched=false表示与教学智能体模块无关，外层必须继续下沉到基础路由。
// Matched=true且Kind=invalid表示命中了智能体保留路径，
// 但路径结构或动作非法。
type coursewareAssistantRouteMatch struct {
	Kind         string
	CoursewareID string
	PageID       string
	SlotID       string
	DeploymentID string
	Matched      bool
}

// SetupWithCoursewareAssistant 在漫画、Style Studio和主路由外
// 增加教师端教学智能体入口。
func SetupWithCoursewareAssistant(
	cfg *config.Config,
) http.Handler {
	baseHandler :=
		SetupWithCoursewareComic(cfg)

	// 总开关关闭时不构造教师端教学智能体服务依赖。
	//
	// 包装层仍识别本模块保留路径并统一返回404，
	// 其余课件、漫画、Style Studio和主路由完整下沉。
	if !cfg.CoursewareAssistantEnabled {
		return buildCoursewareAssistantDisabledRouteHandler(
			baseHandler,
		)
	}

	authService :=
		services.NewAuthService(cfg)

	coursewareService :=
		services.NewCoursewareService()

	slotService :=
		services.NewCoursewareAssistantSlotService()

	contextService :=
		services.NewCoursewareAssistantContextService()

	assistantService :=
		services.NewAIAssistantService()

	planService :=
		services.NewCoursewareAssistantPlanServiceWithDependencies(
			cfg.GetAESKey(),
			cfg.AIAPIBaseURL,
			cfg.AIAPIKey,
			cfg.AIDefaultModel,
			coursewareService,
			contextService,
			assistantService,
		)

	coursewareAssistantHandler :=
		handlers.NewCoursewareAssistantHandler(
			slotService,
			contextService,
			planService,
		)

	deploymentService :=
		services.NewAssistantDeploymentServiceWithDependencies(
			coursewareService,
			slotService,
			contextService,
			assistantService,
		)

	deploymentHandler :=
		handlers.NewAssistantDeploymentHandler(
			deploymentService,
		)

	previewSessionService :=
		services.NewAssistantRuntimeSessionService(
			cfg.JWTSecret,
			cfg.AssistantRuntimePrivacySalt,
			cfg.GetAssistantRuntimeTokenTTL(),
		)

	previewHandler :=
		handlers.NewCoursewareAssistantPreviewHandler(
			previewSessionService,
		)

	ttsService :=
		services.NewCoursewareAssistantTTSService(
			cfg,
		)

	ttsHandler :=
		handlers.NewCoursewareAssistantTTSHandler(
			ttsService,
		)

	authMiddleware :=
		middleware.AuthMiddleware(
			authService,
		)

	authenticatedRoute :=
		corsMiddleware(
			middleware.Chain(
				http.HandlerFunc(
					func(
						w http.ResponseWriter,
						r *http.Request,
					) {
						dispatchCoursewareAssistantRoute(
							w,
							r,
							coursewareAssistantHandler,
							deploymentHandler,
							previewHandler,
							ttsHandler,
						)
					},
				),
				authMiddleware,
			),
		)

	return buildCoursewareAssistantRouteHandler(
		baseHandler,
		authenticatedRoute,
	)
}

// buildCoursewareAssistantDisabledRouteHandler 构造教师端功能关闭包装。
//
// 命中教学智能体保留路径时统一404；
// 与本模块无关的请求必须完整下沉到baseHandler。
func buildCoursewareAssistantDisabledRouteHandler(
	baseHandler http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			matched :=
				matchCoursewareAssistantRoute(
					r.URL.Path,
				)

			if !matched.Matched {
				baseHandler.ServeHTTP(
					w,
					r,
				)
				return
			}

			http.NotFound(w, r)
		},
	)
}

// buildCoursewareAssistantRouteHandler 只把命中的有效教师端路径
// 交给认证路由。
//
// 与教学智能体无关的请求必须完整下沉到baseHandler，
// 不能在本包装层提前返回404。
func buildCoursewareAssistantRouteHandler(
	baseHandler http.Handler,
	authenticatedRoute http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			matched :=
				matchCoursewareAssistantRoute(
					r.URL.Path,
				)

			if !matched.Matched {
				baseHandler.ServeHTTP(
					w,
					r,
				)
				return
			}

			if matched.Kind ==
				coursewareAssistantRouteInvalid {
				http.NotFound(w, r)
				return
			}

			if authenticatedRoute == nil {
				http.Error(
					w,
					"课件教学智能体服务未就绪",
					http.StatusServiceUnavailable,
				)
				return
			}

			authenticatedRoute.ServeHTTP(
				w,
				r,
			)
		},
	)
}

// dispatchCoursewareAssistantRoute 把已认证请求分发到插槽、部署或预览Handler。
func dispatchCoursewareAssistantRoute(
	w http.ResponseWriter,
	r *http.Request,
	assistantHandler *handlers.CoursewareAssistantHandler,
	deploymentHandler *handlers.AssistantDeploymentHandler,
	previewHandler *handlers.CoursewareAssistantPreviewHandler,
	ttsHandler *handlers.CoursewareAssistantTTSHandler,
) {
	matched :=
		matchCoursewareAssistantRoute(
			r.URL.Path,
		)

	if !matched.Matched ||
		matched.Kind ==
			coursewareAssistantRouteInvalid {
		http.NotFound(w, r)
		return
	}

	if assistantHandler == nil ||
		deploymentHandler == nil ||
		previewHandler == nil ||
		ttsHandler == nil {
		http.Error(
			w,
			"课件教学智能体服务未就绪",
			http.StatusServiceUnavailable,
		)
		return
	}

	switch matched.Kind {
	case coursewareAssistantRouteSlots:
		assistantHandler.ListSlots(
			w,
			r,
		)

	case coursewareAssistantRouteSlotItem:
		switch r.Method {
		case http.MethodPut:
			assistantHandler.UpdateSlot(
				w,
				r,
			)

		case http.MethodDelete:
			assistantHandler.DeleteSlot(
				w,
				r,
			)

		default:
			methodNotAllowedJSON(
				w,
				"仅支持PUT/DELETE请求",
			)
		}

	case coursewareAssistantRoutePageSlot:
		switch r.Method {
		case http.MethodGet:
			assistantHandler.GetPageSlot(
				w,
				r,
			)

		case http.MethodPost:
			assistantHandler.CreatePageSlot(
				w,
				r,
			)

		default:
			methodNotAllowedJSON(
				w,
				"仅支持GET/POST请求",
			)
		}

	case coursewareAssistantRoutePageContext:
		assistantHandler.GetContextPreview(
			w,
			r,
		)

	case coursewareAssistantRoutePagePlan:
		assistantHandler.GeneratePlan(
			w,
			r,
		)

	case coursewareAssistantRoutePublishDeployment:
		deploymentHandler.Publish(
			w,
			r,
			matched.CoursewareID,
			matched.PageID,
		)

	case coursewareAssistantRouteDeployments:
		deploymentHandler.List(
			w,
			r,
			matched.CoursewareID,
		)

	case coursewareAssistantRouteVersions:
		switch r.Method {
		case http.MethodGet:
			deploymentHandler.ListVersions(
				w,
				r,
				matched.DeploymentID,
			)

		case http.MethodPost:
			deploymentHandler.PublishVersion(
				w,
				r,
				matched.DeploymentID,
			)

		default:
			methodNotAllowedJSON(
				w,
				"仅支持GET/POST请求",
			)
		}

	case coursewareAssistantRoutePause:
		deploymentHandler.Pause(
			w,
			r,
			matched.DeploymentID,
		)

	case coursewareAssistantRouteResume:
		deploymentHandler.Resume(
			w,
			r,
			matched.DeploymentID,
		)

	case coursewareAssistantRouteRevoke:
		deploymentHandler.Revoke(
			w,
			r,
			matched.DeploymentID,
		)

	case coursewareAssistantRoutePolicy:
		deploymentHandler.UpdatePolicy(
			w,
			r,
			matched.DeploymentID,
		)

	case coursewareAssistantRoutePreviewSession:
		previewHandler.StartPreviewSession(
			w,
			r,
			matched.DeploymentID,
		)

	case coursewareAssistantRouteTTS:
		ttsHandler.Synthesize(
			w,
			r,
			matched.DeploymentID,
		)

	default:
		http.NotFound(w, r)
	}
}

// matchCoursewareAssistantRoute 严格匹配教师端教学智能体路径。
func matchCoursewareAssistantRoute(
	path string,
) coursewareAssistantRouteMatch {
	normalized :=
		normalizeCoursewareAssistantRoutePath(
			path,
		)

	if normalized ==
		assistantDeploymentTeacherRoot {
		return invalidCoursewareAssistantRoute()
	}

	if strings.HasPrefix(
		normalized,
		assistantDeploymentTeacherRoot+"/",
	) {
		return matchAssistantDeploymentTeacherRoute(
			normalized,
		)
	}

	if !strings.HasPrefix(
		normalized,
		coursewareAssistantTeacherPrefix,
	) {
		return coursewareAssistantRouteMatch{}
	}

	rest :=
		strings.TrimPrefix(
			normalized,
			coursewareAssistantTeacherPrefix,
		)

	parts := strings.Split(rest, "/")

	switch {
	case len(parts) == 2 &&
		validCoursewareAssistantRouteID(
			parts[0],
		) &&
		parts[1] == "assistant-slots":
		return coursewareAssistantRouteMatch{
			Kind:         coursewareAssistantRouteSlots,
			CoursewareID: parts[0],
			Matched:      true,
		}

	case len(parts) == 3 &&
		validCoursewareAssistantRouteID(
			parts[0],
		) &&
		parts[1] == "assistant-slots" &&
		validCoursewareAssistantRouteID(
			parts[2],
		):
		return coursewareAssistantRouteMatch{
			Kind:         coursewareAssistantRouteSlotItem,
			CoursewareID: parts[0],
			SlotID:       parts[2],
			Matched:      true,
		}

	case len(parts) == 2 &&
		validCoursewareAssistantRouteID(
			parts[0],
		) &&
		parts[1] == "assistant-deployments":
		return coursewareAssistantRouteMatch{
			Kind:         coursewareAssistantRouteDeployments,
			CoursewareID: parts[0],
			Matched:      true,
		}

	case len(parts) == 4 &&
		validCoursewareAssistantRouteID(
			parts[0],
		) &&
		parts[1] == "pages" &&
		validCoursewareAssistantRouteID(
			parts[2],
		):
		return matchCoursewareAssistantPageRoute(
			parts[0],
			parts[2],
			parts[3],
		)

	default:
		if containsCoursewareAssistantReservedSegment(
			parts,
		) {
			return invalidCoursewareAssistantRoute()
		}

		return coursewareAssistantRouteMatch{}
	}
}

// matchCoursewareAssistantPageRoute 匹配稳定page_id下的教学智能体动作。
//
// 已登记的assistant-*动作正常匹配；
// 未登记但仍以assistant-开头时fail-closed；
// 不以assistant-开头时下沉到普通课件页面路由。
func matchCoursewareAssistantPageRoute(
	coursewareID string,
	pageID string,
	action string,
) coursewareAssistantRouteMatch {
	kind := ""

	switch action {
	case "assistant-slot":
		kind =
			coursewareAssistantRoutePageSlot

	case "assistant-context":
		kind =
			coursewareAssistantRoutePageContext

	case "assistant-plan":
		kind =
			coursewareAssistantRoutePagePlan

	case "assistant-deployment":
		kind =
			coursewareAssistantRoutePublishDeployment

	default:
		if strings.HasPrefix(
			action,
			"assistant-",
		) {
			return invalidCoursewareAssistantRoute()
		}

		return coursewareAssistantRouteMatch{}
	}

	return coursewareAssistantRouteMatch{
		Kind:         kind,
		CoursewareID: coursewareID,
		PageID:       pageID,
		Matched:      true,
	}
}

// matchAssistantDeploymentTeacherRoute 匹配内部部署ID下的管理和预览动作。
func matchAssistantDeploymentTeacherRoute(
	normalized string,
) coursewareAssistantRouteMatch {
	rest :=
		strings.TrimPrefix(
			normalized,
			assistantDeploymentTeacherRoot+"/",
		)

	parts := strings.Split(rest, "/")

	if len(parts) != 2 ||
		!validCoursewareAssistantRouteID(
			parts[0],
		) {
		return invalidCoursewareAssistantRoute()
	}

	kind := ""

	switch parts[1] {
	case "versions":
		kind =
			coursewareAssistantRouteVersions

	case "pause":
		kind =
			coursewareAssistantRoutePause

	case "resume":
		kind =
			coursewareAssistantRouteResume

	case "revoke":
		kind =
			coursewareAssistantRouteRevoke

	case "policy":
		kind =
			coursewareAssistantRoutePolicy

	case "preview-session":
		kind =
			coursewareAssistantRoutePreviewSession

	case "tts":
		kind =
			coursewareAssistantRouteTTS

	default:
		return invalidCoursewareAssistantRoute()
	}

	return coursewareAssistantRouteMatch{
		Kind:         kind,
		DeploymentID: parts[0],
		Matched:      true,
	}
}

// normalizeCoursewareAssistantRoutePath 允许教师端REST路径使用可选尾斜杠。
func normalizeCoursewareAssistantRoutePath(
	path string,
) string {
	path = strings.TrimSpace(path)

	if path == "/" {
		return path
	}

	return strings.TrimRight(
		path,
		"/",
	)
}

// containsCoursewareAssistantReservedSegment 判断是否命中本模块保留段。
func containsCoursewareAssistantReservedSegment(
	parts []string,
) bool {
	for _, part := range parts {
		if strings.HasPrefix(
			part,
			"assistant-",
		) {
			return true
		}
	}

	return false
}

// validCoursewareAssistantRouteID 校验单段内部资源ID。
func validCoursewareAssistantRouteID(
	value string,
) bool {
	return value != "" &&
		len(value) <= 256 &&
		strings.TrimSpace(value) ==
			value &&
		value != "." &&
		value != ".." &&
		!strings.Contains(
			value,
			"\\",
		)
}

// invalidCoursewareAssistantRoute 返回命中模块前缀但结构非法的结果。
func invalidCoursewareAssistantRoute() coursewareAssistantRouteMatch {
	return coursewareAssistantRouteMatch{
		Kind:    coursewareAssistantRouteInvalid,
		Matched: true,
	}
}
