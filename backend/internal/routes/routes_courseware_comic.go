package routes

// routes_courseware_comic.go — 知识点漫画作者端安全路由包装
//
// 项目路由：
//   GET/POST /api/v1/coursewares/{courseware_id}/comic-projects
//   GET  /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/plan
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/confirm-storyboard
//   PUT  /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/style-settings
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/generate-style-preview
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/confirm-style-preview
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/generate
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/insert-page
//
// 参考资源路由：
//   GET/POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/references
//   DELETE   /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/references/{reference_id}
//
// 分格路由：
//   PUT  /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/panels/{panel_id}/storyboard
//   PUT  /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/panels/{panel_id}/overlay
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/panels/{panel_id}/regenerate
//   POST /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/panels/{panel_id}/sync-page
//
// 安全收口：
//   - 不注册 /panels/{panel_id}/prompt；
//   - 分镜编辑只接受教师可见教学字段，不接受内部图片协议；
//   - 图片提示词、负面提示词、IAOCI与关系索引只由后端维护；
//   - 参考资源正文和摘要不通过GET接口返回；
//   - 对保留段中的未知漫画动作统一fail-closed返回404。

import (
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

const (
	coursewareComicRouteProjects             = "projects"
	coursewareComicRouteProject              = "project"
	coursewareComicRoutePlan                 = "plan"
	coursewareComicRouteReferences           = "references"
	coursewareComicRouteReference            = "reference"
	coursewareComicRouteConfirmStoryboard    = "confirm_storyboard"
	coursewareComicRouteStyleSettings        = "style_settings"
	coursewareComicRouteGenerateStylePreview = "generate_style_preview"
	coursewareComicRouteConfirmStylePreview  = "confirm_style_preview"
	coursewareComicRouteGenerate             = "generate"
	coursewareComicRouteInsertPage           = "insert_page"
	coursewareComicRouteStoryboard           = "storyboard"
	coursewareComicRouteOverlay              = "overlay"
	coursewareComicRouteRegenerate           = "regenerate"
	coursewareComicRouteSyncPage             = "sync_page"
	coursewareComicRouteInvalid              = "invalid"

	coursewareComicTeacherPrefix = "/api/v1/coursewares/"
)

type coursewareComicRouteMatch struct {
	Kind         string
	CoursewareID string
	ProjectID    string
	PanelID      string
	ReferenceID  string
	Matched      bool
}

// SetupWithCoursewareComic 在Style Studio和主路由外增加漫画入口。
func SetupWithCoursewareComic(cfg *config.Config) http.Handler {
	baseHandler := SetupWithCoursewareStyleStudio(cfg)

	authService := services.NewAuthService(cfg)
	coursewareService := services.NewCoursewareService()
	projectService := services.NewCoursewareComicProjectServiceWithDependencies(
		coursewareService,
	)
	assistantService := services.NewAIAssistantService()
	planService := services.NewCoursewareComicPlanServiceWithDependencies(
		cfg.GetAESKey(),
		cfg.AIAPIBaseURL,
		cfg.AIAPIKey,
		cfg.AIDefaultModel,
		coursewareService,
		assistantService,
	)
	generationService := services.NewCoursewareComicGenerationService(
		cfg,
		coursewareService,
		services.NewOSSService(cfg),
	)
	pageService := services.NewCoursewareComicPageService(coursewareService)

	comicHandler := handlers.NewCoursewareComicHandler(
		projectService,
		planService,
	)
	generationHandler := handlers.NewCoursewareComicGenerationHandler(
		generationService,
	)
	pageHandler := handlers.NewCoursewareComicPageHandler(pageService)

	authMiddleware := middleware.AuthMiddleware(authService)
	authenticatedRoute := corsMiddleware(
		middleware.Chain(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				dispatchCoursewareComicRoute(
					w,
					r,
					comicHandler,
					generationHandler,
					pageHandler,
				)
			}),
			authMiddleware,
		),
	)

	return buildCoursewareComicRouteHandler(baseHandler, authenticatedRoute)
}

func buildCoursewareComicRouteHandler(
	baseHandler http.Handler,
	authenticatedRoute http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matched := matchCoursewareComicRoute(r.URL.Path)
		if !matched.Matched {
			baseHandler.ServeHTTP(w, r)
			return
		}

		if matched.Kind == coursewareComicRouteInvalid {
			http.NotFound(w, r)
			return
		}

		if authenticatedRoute == nil {
			http.Error(
				w,
				"知识点漫画服务未就绪",
				http.StatusServiceUnavailable,
			)
			return
		}

		authenticatedRoute.ServeHTTP(w, r)
	})
}

func dispatchCoursewareComicRoute(
	w http.ResponseWriter,
	r *http.Request,
	comicHandler *handlers.CoursewareComicHandler,
	generationHandler *handlers.CoursewareComicGenerationHandler,
	pageHandler *handlers.CoursewareComicPageHandler,
) {
	matched := matchCoursewareComicRoute(r.URL.Path)
	if !matched.Matched || matched.Kind == coursewareComicRouteInvalid {
		http.NotFound(w, r)
		return
	}

	if comicHandler == nil || generationHandler == nil || pageHandler == nil {
		http.Error(
			w,
			"知识点漫画服务未就绪",
			http.StatusServiceUnavailable,
		)
		return
	}

	switch matched.Kind {
	case coursewareComicRouteProjects:
		switch r.Method {
		case http.MethodGet:
			comicHandler.ListProjects(w, r, matched.CoursewareID)
		case http.MethodPost:
			comicHandler.CreateProject(w, r, matched.CoursewareID)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}

	case coursewareComicRouteProject:
		comicHandler.GetProject(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRoutePlan:
		comicHandler.PlanProject(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteReferences:
		switch r.Method {
		case http.MethodGet:
			comicHandler.ListProjectReferences(
				w,
				r,
				matched.CoursewareID,
				matched.ProjectID,
			)
		case http.MethodPost:
			comicHandler.CreateProjectReference(
				w,
				r,
				matched.CoursewareID,
				matched.ProjectID,
			)
		default:
			methodNotAllowedJSON(w, "仅支持GET/POST请求")
		}

	case coursewareComicRouteReference:
		if r.Method != http.MethodDelete {
			methodNotAllowedJSON(w, "仅支持DELETE请求")
			return
		}
		comicHandler.DeleteProjectReference(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
			matched.ReferenceID,
		)

	case coursewareComicRouteConfirmStoryboard:
		comicHandler.ConfirmStoryboard(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteStyleSettings:
		comicHandler.UpdateStyleSettings(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteGenerateStylePreview:
		generationHandler.GenerateStylePreview(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteConfirmStylePreview:
		comicHandler.ConfirmStylePreview(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteGenerate:
		generationHandler.GenerateProject(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteInsertPage:
		pageHandler.InsertPage(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
		)

	case coursewareComicRouteStoryboard:
		comicHandler.UpdatePanelStoryboard(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
			matched.PanelID,
		)

	case coursewareComicRouteOverlay:
		comicHandler.UpdatePanelOverlay(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
			matched.PanelID,
		)

	case coursewareComicRouteRegenerate:
		generationHandler.RegeneratePanel(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
			matched.PanelID,
		)

	case coursewareComicRouteSyncPage:
		pageHandler.SyncPanelPage(
			w,
			r,
			matched.CoursewareID,
			matched.ProjectID,
			matched.PanelID,
		)

	default:
		http.NotFound(w, r)
	}
}

func matchCoursewareComicRoute(path string) coursewareComicRouteMatch {
	normalized := normalizeCoursewareComicRoutePath(path)
	if !strings.HasPrefix(normalized, coursewareComicTeacherPrefix) {
		return coursewareComicRouteMatch{}
	}

	rest := strings.TrimPrefix(normalized, coursewareComicTeacherPrefix)
	parts := strings.Split(rest, "/")

	switch {
	case len(parts) == 2 &&
		validCoursewareComicRouteID(parts[0]) &&
		parts[1] == "comic-projects":
		return coursewareComicRouteMatch{
			Kind:         coursewareComicRouteProjects,
			CoursewareID: parts[0],
			Matched:      true,
		}

	case len(parts) == 3 &&
		validCoursewareComicRouteID(parts[0]) &&
		parts[1] == "comic-projects" &&
		validCoursewareComicRouteID(parts[2]):
		return coursewareComicRouteMatch{
			Kind:         coursewareComicRouteProject,
			CoursewareID: parts[0],
			ProjectID:    parts[2],
			Matched:      true,
		}

	case len(parts) == 4 &&
		validCoursewareComicRouteID(parts[0]) &&
		parts[1] == "comic-projects" &&
		validCoursewareComicRouteID(parts[2]):
		return matchCoursewareComicProjectActionRoute(
			parts[0],
			parts[2],
			parts[3],
		)

	case len(parts) == 5 &&
		validCoursewareComicRouteID(parts[0]) &&
		parts[1] == "comic-projects" &&
		validCoursewareComicRouteID(parts[2]) &&
		parts[3] == "references" &&
		validCoursewareComicRouteID(parts[4]):
		return coursewareComicRouteMatch{
			Kind:         coursewareComicRouteReference,
			CoursewareID: parts[0],
			ProjectID:    parts[2],
			ReferenceID:  parts[4],
			Matched:      true,
		}

	case len(parts) == 6 &&
		validCoursewareComicRouteID(parts[0]) &&
		parts[1] == "comic-projects" &&
		validCoursewareComicRouteID(parts[2]) &&
		parts[3] == "panels" &&
		validCoursewareComicRouteID(parts[4]):
		return matchCoursewareComicPanelRoute(
			parts[0],
			parts[2],
			parts[4],
			parts[5],
		)

	default:
		if containsCoursewareComicReservedSegment(parts) {
			return invalidCoursewareComicRoute()
		}
		return coursewareComicRouteMatch{}
	}
}

func matchCoursewareComicProjectActionRoute(
	coursewareID string,
	projectID string,
	action string,
) coursewareComicRouteMatch {
	kind := ""

	switch action {
	case "plan":
		kind = coursewareComicRoutePlan
	case "references":
		kind = coursewareComicRouteReferences
	case "confirm-storyboard":
		kind = coursewareComicRouteConfirmStoryboard
	case "style-settings":
		kind = coursewareComicRouteStyleSettings
	case "generate-style-preview":
		kind = coursewareComicRouteGenerateStylePreview
	case "confirm-style-preview":
		kind = coursewareComicRouteConfirmStylePreview
	case "generate":
		kind = coursewareComicRouteGenerate
	case "insert-page":
		kind = coursewareComicRouteInsertPage
	default:
		return invalidCoursewareComicRoute()
	}

	return coursewareComicRouteMatch{
		Kind:         kind,
		CoursewareID: coursewareID,
		ProjectID:    projectID,
		Matched:      true,
	}
}

func matchCoursewareComicPanelRoute(
	coursewareID string,
	projectID string,
	panelID string,
	action string,
) coursewareComicRouteMatch {
	kind := ""

	switch action {
	case "storyboard":
		kind = coursewareComicRouteStoryboard
	case "overlay":
		kind = coursewareComicRouteOverlay
	case "regenerate":
		kind = coursewareComicRouteRegenerate
	case "sync-page":
		kind = coursewareComicRouteSyncPage
	default:
		return invalidCoursewareComicRoute()
	}

	return coursewareComicRouteMatch{
		Kind:         kind,
		CoursewareID: coursewareID,
		ProjectID:    projectID,
		PanelID:      panelID,
		Matched:      true,
	}
}

func normalizeCoursewareComicRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "/" {
		return path
	}
	return strings.TrimRight(path, "/")
}

func containsCoursewareComicReservedSegment(parts []string) bool {
	for _, part := range parts {
		if part == "comic-projects" || strings.HasPrefix(part, "comic-") {
			return true
		}
	}
	return false
}

func validCoursewareComicRouteID(value string) bool {
	return value != "" &&
		len(value) <= 256 &&
		strings.TrimSpace(value) == value &&
		value != "." &&
		value != ".." &&
		!strings.Contains(value, "\\")
}

func invalidCoursewareComicRoute() coursewareComicRouteMatch {
	return coursewareComicRouteMatch{
		Kind:    coursewareComicRouteInvalid,
		Matched: true,
	}
}
