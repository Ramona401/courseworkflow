package routes

// routes_courseware_support.go
//
// 课件工坊的小型子路由分发器与课件多级审核路由注册。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// dispatchCWComponentSubRoutes 分发课件组件子路由。
func dispatchCWComponentSubRoutes(
	w http.ResponseWriter,
	r *http.Request,
	handler *handlers.CWComponentHandler,
) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/index") {
		handler.CompressIndex(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handler.GetComponent(w, r)

	case http.MethodPut:
		handler.UpdateComponent(w, r)

	case http.MethodDelete:
		handler.DeleteComponent(w, r)

	default:
		http.Error(
			w,
			`{"code":-1,"message":"Method not allowed"}`,
			http.StatusMethodNotAllowed,
		)
	}
}

// dispatchTemplateAIRoutes 分发模板AI操作路由。
func dispatchTemplateAIRoutes(
	w http.ResponseWriter,
	r *http.Request,
	handler *handlers.CoursewareTemplateHandler,
) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/extract"):
		handler.ExtractFromHTML(w, r)

	case strings.HasSuffix(path, "/publish-targets"):
		handler.GetPublishTargets(w, r)

	case strings.HasSuffix(path, "/my-drafts"):
		handler.ListMyDrafts(w, r)

	case strings.Contains(path, "/drafts/") &&
		r.Method == http.MethodDelete:
		handler.DeleteDraft(w, r)

	case strings.HasSuffix(path, "/refine"):
		handler.RefineTemplate(w, r)

	case strings.HasSuffix(path, "/history"):
		handler.GetHistory(w, r)

	case strings.HasSuffix(path, "/rollback"):
		handler.RollbackToHistory(w, r)

	case strings.HasSuffix(path, "/unpublish"):
		handler.UnpublishTemplate(w, r)

	case strings.HasSuffix(path, "/publish"):
		handler.PublishDraft(w, r)

	default:
		http.Error(
			w,
			`{"code":-1,"message":"未找到路由"}`,
			http.StatusNotFound,
		)
	}
}

// dispatchSubtitleRoutes 分发字幕轨路由。
func dispatchSubtitleRoutes(
	w http.ResponseWriter,
	r *http.Request,
	handler *handlers.CoursewareSubtitleHandler,
) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/generate-tts"):
		handler.GenerateTTS(w, r)
		return

	case strings.HasSuffix(path, "/export-srt"):
		handler.ExportSRT(w, r)
		return

	case strings.HasSuffix(path, "/burn-in"):
		handler.BurnInSubtitle(w, r)
		return
	}

	subtitleIndex := strings.Index(path, "/subtitles/")
	if subtitleIndex >= 0 {
		rest := path[subtitleIndex+len("/subtitles/"):]
		rest = strings.TrimSuffix(rest, "/")

		if rest != "" {
			if r.Method == http.MethodDelete {
				handler.DeleteSubtitle(w, r)
				return
			}

			http.Error(
				w,
				`{"code":-1,"message":"字幕子路由仅支持DELETE"}`,
				http.StatusMethodNotAllowed,
			)
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		handler.UpsertSubtitle(w, r)

	case http.MethodGet:
		handler.ListSubtitles(w, r)

	default:
		http.Error(
			w,
			`{"code":-1,"message":"Method not allowed"}`,
			http.StatusMethodNotAllowed,
		)
	}
}

// cwReviewHandlerRef 供课件实例的submit-review子路由调用。
var cwReviewHandlerRef *handlers.CoursewareReviewHandler

// registerCoursewareReviewRoutes 注册课件多级审核路由组。
func registerCoursewareReviewRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	coursewareReviewHandler *handlers.CoursewareReviewHandler,
) {
	cwReviewHandlerRef = coursewareReviewHandler

	reviewMux := middleware.Chain(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				path := r.URL.Path
				rest := strings.TrimPrefix(
					path,
					"/api/v1/courseware-reviews/",
				)

				switch {
				case rest == "pending" &&
					r.Method == http.MethodGet:
					coursewareReviewHandler.GetPendingReviews(w, r)
					return

				case rest == "reviewed" &&
					r.Method == http.MethodGet:
					coursewareReviewHandler.GetReviewedRecords(w, r)
					return

				case rest == "stats" &&
					r.Method == http.MethodGet:
					coursewareReviewHandler.GetReviewStats(w, r)
					return
				}

				switch {
				case strings.HasSuffix(path, "/l1"):
					coursewareReviewHandler.ReviewL1(w, r)

				case strings.HasSuffix(path, "/l2"):
					coursewareReviewHandler.ReviewL2(w, r)

				case strings.HasSuffix(path, "/history"):
					coursewareReviewHandler.GetReviewHistory(w, r)

				case strings.HasSuffix(path, "/detail"):
					coursewareReviewHandler.GetReviewDetail(w, r)

				default:
					http.NotFound(w, r)
				}
			},
		),
		authMW,
	)

	mux.Handle(
		"/api/v1/courseware-reviews",
		reviewMux,
	)
	mux.Handle(
		"/api/v1/courseware-reviews/",
		reviewMux,
	)
}
