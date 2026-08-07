package routes

// routes_courseware_dispatch.go
//
// 单个课件实例及其页面、资产、审核、协作子路由分发。
//
// 本文件由routes_courseware.go拆出，完整保留原分发顺序。
// 长后缀和更具体路径必须优先于短后缀与通用/pages/路径。

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
)

// dispatchCoursewareSubRoutes 分发课件实例子路由。
func dispatchCoursewareSubRoutes(
	w http.ResponseWriter,
	r *http.Request,
	h *handlers.CoursewareHandler,
	indexHandler *handlers.CoursewareIndexHandler,
	genHandler *handlers.CoursewareGenHandler,
	templateHandler *handlers.CoursewareTemplateHandler,
	assetHandler *handlers.CoursewareAssetHandler,
	videoEditHandler *handlers.VideoEditHandler,
) {
	path := r.URL.Path

	if strings.HasSuffix(
		path,
		"/export-bundle",
	) {
		h.ExportBundle(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/videos/mix-narration",
	) {
		videoEditHandler.MixNarration(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/advanced-concat",
	) {
		videoEditHandler.AdvancedConcat(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/concat",
	) {
		videoEditHandler.ConcatVideos(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/trim",
	) {
		videoEditHandler.TrimVideo(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/trim-audio",
	) {
		videoEditHandler.TrimAudio(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/mute",
	) {
		videoEditHandler.MuteVideo(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/videos/extract-audio",
	) {
		videoEditHandler.ExtractAudio(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/save-as-template",
	) {
		templateHandler.SaveAsMyTemplate(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/generate-video-first-frame",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.GenerateVideoFirstFrame(
			w,
			r,
		)
		return
	}

	if strings.HasSuffix(
		path,
		"/generate-image",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.GenerateImage(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/upload-image",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.UploadImage(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/upload-video",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.UploadVideo(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/upload-audio",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.UploadAudio(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/insert-image",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.InsertImage(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/suggest-image-prompt",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.SuggestImagePrompt(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/suggest-video-prompt",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.SuggestVideoPrompt(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/image-suggestions",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.GetStoredImageSuggestions(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/video-storyboards",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		switch r.Method {
		case http.MethodGet:
			assetHandler.GetStoredVideoStoryboards(
				w,
				r,
			)
		case http.MethodPost:
			assetHandler.SaveVideoStoryboards(
				w,
				r,
			)
		default:
			http.Error(
				w,
				`{"code":-1,"message":"Method not allowed"}`,
				http.StatusMethodNotAllowed,
			)
		}
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-video",
	) &&
		strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.GenerateVideo(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/video-status",
	) &&
		strings.Contains(
			path,
			"/assets/",
		) {
		assetHandler.QueryVideoStatus(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/upload-oss",
	) &&
		strings.Contains(
			path,
			"/assets/",
		) {
		assetHandler.UploadToOSS(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/style-anchor",
	) {
		switch r.Method {
		case http.MethodPost:
			assetHandler.SetStyleAnchor(w, r)
		case http.MethodDelete:
			assetHandler.ClearStyleAnchor(w, r)
		default:
			http.Error(
				w,
				`{"code":-1,"message":"Method not allowed"}`,
				http.StatusMethodNotAllowed,
			)
		}
		return
	}

	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/assets",
		) {
		assetHandler.ListPageAssets(w, r)
		return
	}

	if strings.Contains(
		path,
		"/assets/",
	) &&
		r.Method ==
			http.MethodDelete {
		assetHandler.DeleteAsset(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/assets",
	) &&
		!strings.Contains(
			path,
			"/pages/",
		) {
		assetHandler.ListCoursewareAssets(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/generate-index-doc",
	) {
		indexHandler.GenerateIndexFromDocTracked(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-index-ppt",
	) {
		indexHandler.GenerateIndexFromPPTTracked(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-index-topic",
	) {
		indexHandler.GenerateIndexFromTopicTracked(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-index",
	) {
		indexHandler.GenerateIndexWithPresetTracked(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/refine-index",
	) {
		indexHandler.RefineIndexTracked(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/alignment-report",
	) {
		indexHandler.GetAlignmentReport(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/lesson-plan-content",
	) {
		indexHandler.GetLessonPlanContent(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/recheck-alignment",
	) {
		indexHandler.RecheckAlignmentTracked(
			w,
			r,
		)
		return
	}
	if strings.HasSuffix(
		path,
		"/rollback-status",
	) {
		h.RollbackStatus(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-3d-page",
	) {
		genHandler.Generate3DPageTracked(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-preview",
	) {
		genHandler.GeneratePreviewTracked(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/save-nav-template",
	) {
		genHandler.SaveNavTemplate(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/auto-assemble",
	) {
		genHandler.AutoAssembleTracked(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/generate-pages",
	) {
		genHandler.GeneratePagesTracked(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/refine-nav",
	) {
		genHandler.RefineNav(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/cancel-generate",
	) {
		genHandler.CancelGenerate(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/index-stream",
	) {
		indexHandler.IndexStream(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/confirm-index",
	) {
		h.ConfirmIndex(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/upload-logo",
	) {
		h.UploadLogo(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/save-style",
	) {
		h.SaveStyleFull(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/confirm-style",
	) {
		h.ConfirmStyle(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/style",
	) {
		h.SaveStyle(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/confirm",
	) &&
		!strings.HasSuffix(
			path,
			"/confirm-index",
		) &&
		!strings.HasSuffix(
			path,
			"/confirm-style",
		) {
		h.ConfirmCourseware(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/pages/reorder",
	) {
		h.ReorderPages(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/pages",
	) ||
		strings.HasSuffix(
			path,
			"/pages/",
		) {
		switch r.Method {
		case http.MethodGet:
			h.GetCoursewarePages(w, r)
		case http.MethodPost:
			h.AddPage(w, r)
		default:
			http.Error(
				w,
				`{"code":-1,"message":"Method not allowed"}`,
				http.StatusMethodNotAllowed,
			)
		}
		return
	}

	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/rebuild-discussion",
		) {
		genHandler.RebuildDiscussion(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/regenerate",
		) {
		genHandler.RegeneratePage(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/refine",
		) {
		genHandler.RefinePage(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/save-html",
		) {
		genHandler.SavePageHTML(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/import-html",
		) {
		genHandler.ImportPageHTML(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.Contains(
			path,
			"/versions/",
		) {
		genHandler.GetPageVersionDetail(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/versions",
		) {
		genHandler.ListPageVersions(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) &&
		strings.HasSuffix(
			path,
			"/rollback",
		) {
		genHandler.RollbackPage(w, r)
		return
	}
	if strings.Contains(
		path,
		"/pages/",
	) {
		switch r.Method {
		case http.MethodPut:
			h.UpdatePageIndex(w, r)
		case http.MethodDelete:
			indexHandler.DeletePage(w, r)
		default:
			http.Error(
				w,
				`{"code":-1,"message":"Method not allowed"}`,
				http.StatusMethodNotAllowed,
			)
		}
		return
	}

	if strings.HasSuffix(
		path,
		"/annotations",
	) ||
		strings.HasSuffix(
			path,
			"/annotations/",
		) {
		switch r.Method {
		case http.MethodPost:
			h.CreateCWAnnotation(w, r)
		case http.MethodGet:
			h.ListCWAnnotations(w, r)
		default:
			http.Error(
				w,
				`{"code":-1,"message":"Method not allowed"}`,
				http.StatusMethodNotAllowed,
			)
		}
		return
	}

	if strings.HasSuffix(
		path,
		"/publish",
	) {
		h.PublishCourseware(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/code-share-scope",
	) {
		h.SetCodeShareScope(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/fork",
	) {
		h.ForkCourseware(w, r)
		return
	}

	if strings.HasSuffix(
		path,
		"/submit-review",
	) {
		if cwReviewHandlerRef != nil {
			cwReviewHandlerRef.SubmitForReview(
				w,
				r,
			)
		} else {
			http.Error(
				w,
				`{"code":-1,"message":"审核服务未就绪"}`,
				http.StatusServiceUnavailable,
			)
		}
		return
	}

	if strings.HasSuffix(
		path,
		"/collab/start",
	) {
		h.StartCollab(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/collab/end",
	) {
		h.EndCollab(w, r)
		return
	}
	if strings.Contains(
		path,
		"/collab/members/",
	) {
		h.RemoveCollabMember(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/collab/members",
	) {
		h.AddCollabMember(w, r)
		return
	}
	if strings.HasSuffix(
		path,
		"/collab",
	) ||
		strings.HasSuffix(
			path,
			"/collab/",
		) {
		h.GetCollabStatus(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetCourseware(w, r)
	case http.MethodPut:
		h.UpdateCourseware(w, r)
	case http.MethodDelete:
		h.DeleteCourseware(w, r)
	default:
		http.Error(
			w,
			`{"code":-1,"message":"Method not allowed"}`,
			http.StatusMethodNotAllowed,
		)
	}
}
