package routes

// routes_courseware_style_studio.go — AI美术风格工作室安全路由包装
//
// 设计目标：
//   - 不修改超长的routes.go和routes_courseware.go；
//   - 只优先拦截/coursewares/{id}/style-studio子树；
//   - 风格工作室独立经过CORS和JWT认证；
//   - 其它所有请求原样交给既有Setup路由；
//   - 不重复启动主路由中的Engine、调度器或其它服务。

import (
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/middleware"
	"tedna/internal/services"
)

// SetupWithCoursewareStyleStudio 构建包含风格工作室的完整HTTP入口。
func SetupWithCoursewareStyleStudio(
	cfg *config.Config,
) http.Handler {
	// 主业务路由只初始化一次。
	baseHandler := Setup(cfg)

	authService :=
		services.NewAuthService(cfg)

	assetService :=
		services.NewCoursewareAssetService(
			cfg,
		)

	ossService :=
		services.NewOSSService(cfg)

	styleStudioService :=
		services.NewCoursewareStyleStudioService(
			cfg,
			assetService,
			ossService,
		)

	styleStudioHandler :=
		handlers.NewCoursewareStyleStudioHandler(
			styleStudioService,
		)

	authMiddleware :=
		middleware.AuthMiddleware(
			authService,
		)

	// 风格工作室独立套用认证和CORS。
	// OPTIONS会先由corsMiddleware响应，不会被认证中间件阻断。
	styleStudioRoute :=
		corsMiddleware(
			middleware.Chain(
				http.HandlerFunc(
					styleStudioHandler.Handle,
				),
				authMiddleware,
			),
		)

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			if isCoursewareStyleStudioRequest(
				r.URL.Path,
			) {
				styleStudioRoute.ServeHTTP(
					w,
					r,
				)
				return
			}

			baseHandler.ServeHTTP(
				w,
				r,
			)
		},
	)
}

// isCoursewareStyleStudioRequest 判断请求是否属于风格工作室子树。
func isCoursewareStyleStudioRequest(
	path string,
) bool {
	const prefix = "/api/v1/coursewares/"

	path = strings.TrimSpace(path)
	path = strings.TrimRight(path, "/")

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return false
	}

	remaining :=
		strings.TrimPrefix(
			path,
			prefix,
		)

	parts :=
		strings.Split(
			remaining,
			"/",
		)

	return len(parts) >= 2 &&
		strings.TrimSpace(
			parts[0],
		) != "" &&
		parts[1] == "style-studio"
}
