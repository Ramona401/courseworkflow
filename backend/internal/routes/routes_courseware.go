package routes

// routes_courseware.go
//
// 课件工坊主路由注册。
//
// 本文件负责集合级路由、后台配置路由、模板路由和组件路由。
// 单课件实例子路由位于routes_courseware_dispatch.go。
// 通用辅助和课件审核路由位于routes_courseware_support.go。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCoursewareRoutes 注册课件工坊全部路由。
func registerCoursewareRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	cwHandler *handlers.CoursewareHandler,
	cwCompHandler *handlers.CWComponentHandler,
	cwSeedHandler *handlers.CWSeedHandler,
	cwIndexHandler *handlers.CoursewareIndexHandler,
	cwGenHandler *handlers.CoursewareGenHandler,
	cwTplHandler *handlers.CoursewareTemplateHandler,
	cwAssetHandler *handlers.CoursewareAssetHandler,
	videoEditHandler *handlers.VideoEditHandler,
	subtitleHandler *handlers.CoursewareSubtitleHandler,
) {
	draftHandler := handlers.NewVideoDraftHandler()
	backgroundHandler := handlers.NewCoursewareBackgroundHandler()
	fontHandler := handlers.NewCoursewareFontHandler()

	mux.HandleFunc(
		"/api/v1/sse/courseware/",
		cwIndexHandler.IndexStream,
	)
	mux.HandleFunc(
		"/api/v1/sse/template-refine/",
		cwTplHandler.RefineStream,
	)
	mux.HandleFunc(
		"/api/v1/sse/template-extract",
		cwTplHandler.ExtractStream,
	)

	coursewareMux := middleware.Chain(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				path := r.URL.Path

				if path == "/api/v1/coursewares/shared" ||
					path == "/api/v1/coursewares/shared/" {
					if r.Method == http.MethodGet {
						cwHandler.ListSharedCoursewares(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/collab/candidates" ||
					path == "/api/v1/coursewares/collab/candidates/" {
					if r.Method == http.MethodGet {
						cwHandler.ListCollabCandidates(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/collab/joined" ||
					path == "/api/v1/coursewares/collab/joined/" {
					if r.Method == http.MethodGet {
						cwHandler.ListJoinedCollab(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if strings.HasPrefix(
					path,
					"/api/v1/coursewares/annotations/",
				) {
					if strings.HasSuffix(path, "/resolve") {
						cwHandler.ResolveCWAnnotation(w, r)
					} else {
						cwHandler.DeleteCWAnnotation(w, r)
					}
					return
				}

				if path == "/api/v1/coursewares/from-topic" ||
					path == "/api/v1/coursewares/from-topic/" {
					if r.Method == http.MethodPost {
						cwHandler.CreateFromTopic(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/from-ppt" ||
					path == "/api/v1/coursewares/from-ppt/" {
					if r.Method == http.MethodPost {
						cwIndexHandler.CreateFromPPT(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/from-doc" ||
					path == "/api/v1/coursewares/from-doc/" {
					if r.Method == http.MethodPost {
						cwIndexHandler.CreateFromDoc(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/from-3d" ||
					path == "/api/v1/coursewares/from-3d/" {
					if r.Method == http.MethodPost {
						cwHandler.CreateFrom3D(w, r)
					} else {
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares/logo-history" ||
					path == "/api/v1/coursewares/logo-history/" {
					switch r.Method {
					case http.MethodGet:
						cwHandler.ListLogoHistory(w, r)

					case http.MethodDelete:
						cwHandler.DeleteLogoHistory(w, r)

					default:
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if path == "/api/v1/coursewares" ||
					path == "/api/v1/coursewares/" {
					switch r.Method {
					case http.MethodGet:
						cwHandler.ListCoursewares(w, r)

					case http.MethodPost:
						cwHandler.CreateCourseware(w, r)

					default:
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				if strings.Contains(path, "/subtitles") {
					dispatchSubtitleRoutes(
						w,
						r,
						subtitleHandler,
					)
					return
				}

				if strings.Contains(path, "/video-drafts") {
					draftHandler.HandleDrafts(w, r)
					return
				}

				if strings.HasSuffix(path, "/font") ||
					strings.HasSuffix(path, "/font/") {
					fontHandler.HandleCoursewareFont(w, r)
					return
				}

				if strings.HasSuffix(path, "/page-bg") &&
					strings.Contains(path, "/pages/") {
					backgroundHandler.HandlePageBackground(w, r)
					return
				}

				if strings.HasSuffix(path, "/background") ||
					strings.HasSuffix(path, "/background/") {
					backgroundHandler.HandleCoursewareBackground(w, r)
					return
				}

				dispatchCoursewareSubRoutes(
					w,
					r,
					cwHandler,
					cwIndexHandler,
					cwGenHandler,
					cwTplHandler,
					cwAssetHandler,
					videoEditHandler,
				)
			},
		),
		authMW,
	)

	mux.Handle("/api/v1/coursewares", coursewareMux)
	mux.Handle("/api/v1/coursewares/", coursewareMux)

	mux.Handle(
		"/api/v1/courseware-backgrounds",
		middleware.Chain(
			http.HandlerFunc(backgroundHandler.HandleLibrary),
			authMW,
		),
	)
	mux.Handle(
		"/api/v1/courseware-backgrounds/",
		middleware.Chain(
			http.HandlerFunc(backgroundHandler.HandleLibrary),
			authMW,
		),
	)

	mux.Handle(
		"/api/v1/courseware-fonts",
		middleware.Chain(
			http.HandlerFunc(fontHandler.ListSchemes),
			authMW,
		),
	)

	snippetHandler := handlers.NewCoursewareSnippetHandler()

	mux.Handle(
		"/api/v1/code-snippets",
		middleware.Chain(
			http.HandlerFunc(snippetHandler.HandleCollection),
			authMW,
		),
	)
	mux.Handle(
		"/api/v1/code-snippets/",
		middleware.Chain(
			http.HandlerFunc(snippetHandler.HandleItem),
			authMW,
		),
	)

	ttsConfigHandler := handlers.NewTTSConfigHandler()

	mux.Handle(
		"/api/v1/admin/tts-config",
		middleware.Chain(
			http.HandlerFunc(ttsConfigHandler.HandleTTSConfig),
			authMW,
			adminOnly,
		),
	)
	mux.Handle(
		"/api/v1/admin/tts-config/test",
		middleware.Chain(
			http.HandlerFunc(ttsConfigHandler.TestTTS),
			authMW,
			adminOnly,
		),
	)

	asrConfigHandler := handlers.NewASRConfigHandler()

	mux.Handle(
		"/api/v1/admin/asr-config",
		middleware.Chain(
			http.HandlerFunc(asrConfigHandler.HandleASRConfig),
			authMW,
			adminOnly,
		),
	)
	mux.Handle(
		"/api/v1/admin/asr-config/test",
		middleware.Chain(
			http.HandlerFunc(asrConfigHandler.TestASR),
			authMW,
			adminOnly,
		),
	)

	domesticGatewayHandler := handlers.NewDomesticGatewayHandler()

	mux.Handle(
		"/api/v1/admin/domestic-gateway",
		middleware.Chain(
			http.HandlerFunc(
				domesticGatewayHandler.HandleDomesticGateway,
			),
			authMW,
			adminOnly,
		),
	)
	mux.Handle(
		"/api/v1/admin/domestic-gateway/test",
		middleware.Chain(
			http.HandlerFunc(domesticGatewayHandler.TestDomesticGateway),
			authMW,
			adminOnly,
		),
	)
	mux.Handle(
		"/api/v1/admin/domestic-gateway/models",
		middleware.Chain(
			http.HandlerFunc(domesticGatewayHandler.ListDomesticModels),
			authMW,
			adminOnly,
		),
	)

	mux.Handle(
		"/api/v1/courseware-presets",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					cwHandler.GetSchemePresets(w, r)
				},
			),
			authMW,
		),
	)

	mux.Handle(
		"/api/v1/courseware-templates",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					cwHandler.ListTemplates(w, r)
				},
			),
			authMW,
		),
	)

	mux.Handle(
		"/api/v1/courseware-templates/",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					path := r.URL.Path

					if strings.HasSuffix(path, "/with-user") {
						cwTplHandler.ListTemplatesWithUser(w, r)
						return
					}

					if strings.Contains(path, "/personal/") &&
						r.Method == http.MethodDelete {
						cwTplHandler.DeleteMyTemplate(w, r)
						return
					}

					if strings.HasSuffix(path, "/preview") {
						cwHandler.GetTemplatePreview(w, r)
						return
					}

					http.Error(
						w,
						`{"code":-1,"message":"未找到路由"}`,
						http.StatusNotFound,
					)
				},
			),
			authMW,
		),
	)

	componentMux := middleware.Chain(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				path := r.URL.Path

				if path == "/api/v1/courseware-components" ||
					path == "/api/v1/courseware-components/" {
					switch r.Method {
					case http.MethodGet:
						cwCompHandler.ListComponents(w, r)

					case http.MethodPost:
						cwCompHandler.CreateComponent(w, r)

					default:
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
					return
				}

				dispatchCWComponentSubRoutes(
					w,
					r,
					cwCompHandler,
				)
			},
		),
		authMW,
	)

	mux.Handle(
		"/api/v1/courseware-components",
		componentMux,
	)
	mux.Handle(
		"/api/v1/courseware-components/",
		componentMux,
	)

	mux.Handle(
		"/api/v1/courseware-components/match",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					cwCompHandler.MatchComponents(w, r)
				},
			),
			authMW,
		),
	)

	mux.Handle(
		"/api/v1/admin/courseware-seed",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					cwSeedHandler.SeedAll(w, r)
				},
			),
			authMW,
			adminOnly,
		),
	)

	mux.Handle(
		"/api/v1/admin/courseware-templates",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					path := r.URL.Path

					if path == "/api/v1/admin/courseware-templates" ||
						path == "/api/v1/admin/courseware-templates/" {
						switch r.Method {
						case http.MethodPost:
							cwSeedHandler.CreateTemplate(w, r)

						default:
							http.Error(
								w,
								`{"code":-1,"message":"Method not allowed"}`,
								http.StatusMethodNotAllowed,
							)
						}
						return
					}
				},
			),
			authMW,
			adminOnly,
		),
	)

	mux.Handle(
		"/api/v1/admin/courseware-templates/",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					switch r.Method {
					case http.MethodPut:
						cwSeedHandler.UpdateTemplate(w, r)

					case http.MethodDelete:
						cwSeedHandler.DeleteTemplate(w, r)

					default:
						http.Error(
							w,
							`{"code":-1,"message":"Method not allowed"}`,
							http.StatusMethodNotAllowed,
						)
					}
				},
			),
			authMW,
			adminOnly,
		),
	)

	mux.Handle(
		"/api/v1/coursewares/templates/",
		middleware.Chain(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					dispatchTemplateAIRoutes(
						w,
						r,
						cwTplHandler,
					)
				},
			),
			authMW,
		),
	)
}
