package routes

// routes_courseware_assembly_runtime.go — 自动装配生命周期外层路由
//
// 该包装层只拦截两个精确路径：
//   GET  /api/v1/coursewares/{id}/assembly-state
//   POST /api/v1/coursewares/{id}/cancel-auto-assemble
//
// 其它请求全部原样下沉到SetupWithAssistantRuntime，主路由、Engine、
// 调度器和既有课件服务仍只初始化一次。

import (
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

const (
	coursewareAssemblyRouteState  = "state"
	coursewareAssemblyRouteCancel = "cancel"

	coursewareAssemblyRoutePrefix = "/api/v1/coursewares/"
)

// coursewareAssemblyRouteMatch 是自动装配生命周期路径匹配结果。
type coursewareAssemblyRouteMatch struct {
	Kind         string
	CoursewareID string
	Matched      bool
}

// SetupWithCoursewareAssemblyRuntime 在完整应用路由外增加装配状态与取消端点。
func SetupWithCoursewareAssemblyRuntime(
	cfg *config.Config,
) http.Handler {
	baseHandler :=
		SetupWithAssistantRuntime(
			cfg,
		)

	authService :=
		services.NewAuthService(
			cfg,
		)

	assemblyHandler :=
		handlers.NewCoursewareAssemblyHandler(
			services.NewCoursewareService(),
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
						dispatchCoursewareAssemblyRoute(
							w,
							r,
							assemblyHandler,
						)
					},
				),
				authMiddleware,
			),
		)

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			matched :=
				matchCoursewareAssemblyRoute(
					r.URL.Path,
				)

			if !matched.Matched {
				baseHandler.ServeHTTP(
					w,
					r,
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

// dispatchCoursewareAssemblyRoute 分发生命周期端点。
func dispatchCoursewareAssemblyRoute(
	w http.ResponseWriter,
	r *http.Request,
	handler *handlers.CoursewareAssemblyHandler,
) {
	matched :=
		matchCoursewareAssemblyRoute(
			r.URL.Path,
		)

	if !matched.Matched ||
		handler == nil {
		http.NotFound(
			w,
			r,
		)
		return
	}

	switch matched.Kind {
	case coursewareAssemblyRouteState:
		handler.GetState(
			w,
			r,
			matched.CoursewareID,
		)

	case coursewareAssemblyRouteCancel:
		handler.Cancel(
			w,
			r,
			matched.CoursewareID,
		)

	default:
		http.NotFound(
			w,
			r,
		)
	}
}

// matchCoursewareAssemblyRoute 只匹配两个精确生命周期动作。
func matchCoursewareAssemblyRoute(
	path string,
) coursewareAssemblyRouteMatch {
	path = strings.TrimSpace(
		path,
	)
	path = strings.TrimRight(
		path,
		"/",
	)

	if !strings.HasPrefix(
		path,
		coursewareAssemblyRoutePrefix,
	) {
		return coursewareAssemblyRouteMatch{}
	}

	rest :=
		strings.TrimPrefix(
			path,
			coursewareAssemblyRoutePrefix,
		)

	parts :=
		strings.Split(
			rest,
			"/",
		)

	if len(parts) != 2 {
		return coursewareAssemblyRouteMatch{}
	}

	coursewareID :=
		validCoursewareAssemblyRouteID(
			parts[0],
		)
	if coursewareID == "" {
		return coursewareAssemblyRouteMatch{
			Matched: true,
		}
	}

	switch parts[1] {
	case "assembly-state":
		return coursewareAssemblyRouteMatch{
			Kind:         coursewareAssemblyRouteState,
			CoursewareID: coursewareID,
			Matched:      true,
		}

	case "cancel-auto-assemble":
		return coursewareAssemblyRouteMatch{
			Kind:         coursewareAssemblyRouteCancel,
			CoursewareID: coursewareID,
			Matched:      true,
		}

	default:
		return coursewareAssemblyRouteMatch{}
	}
}

// validCoursewareAssemblyRouteID 校验单段课件ID。
func validCoursewareAssemblyRouteID(
	value string,
) string {
	value = strings.TrimSpace(
		value,
	)

	if value == "" ||
		value == "." ||
		value == ".." ||
		len(value) > 256 ||
		strings.Contains(
			value,
			"/",
		) ||
		strings.Contains(
			value,
			"\\",
		) {
		return ""
	}

	return value
}
