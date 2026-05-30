package routes

// routes_courseware.go — 课件工坊路由注册(v0.42 多入口+PPT上传+多媒体+字幕轨+3D单页)
//
// v0.42.11 新增路由:
//   - POST /api/v1/coursewares/from-3d                            — 创建3D互动单页课件
//   - POST /api/v1/coursewares/{id}/generate-3d-page              — 触发3D单页AI生成
//
// v0.42.8 字幕轨新增路由:
//   - POST   /api/v1/coursewares/{id}/subtitles                     — 创建/更新字幕轨
//   - GET    /api/v1/coursewares/{id}/subtitles                     — 查询字幕轨列表
//   - DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}            — 删除字幕轨
//   - POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt — 导出 SRT
//   - POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in    — 硬字幕烧录
//
// v0.42 多媒体新增路由:
//   - POST   /api/v1/coursewares/{id}/pages/{num}/generate-image  — AI生成图片
//   - POST   /api/v1/coursewares/{id}/pages/{num}/upload-image    — 手动上传图片
//   - POST   /api/v1/coursewares/{id}/pages/{num}/upload-video    — v0.42.5 手动上传视频
//   - GET    /api/v1/coursewares/{id}/pages/{num}/assets           — 获取页面图片列表
//   - POST   /api/v1/coursewares/{id}/pages/{num}/insert-image     — 将图片插入HTML
//   - GET    /api/v1/coursewares/{id}/assets                       — 获取课件全部图片
//   - DELETE /api/v1/coursewares/{id}/assets/{asset_id}            — 删除图片
//
// v0.42 多入口路由:
//   - POST /api/v1/coursewares/from-topic                    — 从主题直接创建课件
//   - POST /api/v1/coursewares/{id}/generate-index-topic     — 从主题生成课件索引
//   - POST /api/v1/coursewares/from-ppt                      — 上传PPT创建课件
//   - POST /api/v1/coursewares/{id}/generate-index-ppt       — 从PPT内容生成课件索引
//   - POST /api/v1/coursewares/from-doc                      — 上传Word文档创建课件
//   - POST /api/v1/coursewares/{id}/generate-index-doc       — 从Word文档生成课件索引

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerCoursewareRoutes 注册课件工坊全部路由
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
	// v0.42.5: 视频编辑器草稿处理器
	draftHandler := handlers.NewVideoDraftHandler()

	// ==================== 课件索引 SSE(内部 Token 验证,不走 authMW) ====================
	mux.HandleFunc("/api/v1/sse/courseware/", cwIndexHandler.IndexStream)

	// ==================== v139 新增:模板微调 SSE(内部 Token 验证) ====================
	mux.HandleFunc("/api/v1/sse/template-refine/", cwTplHandler.RefineStream)

	// ==================== v145 新增:模板提取 SSE(内部 Token 验证) ====================
	mux.HandleFunc("/api/v1/sse/template-extract", cwTplHandler.ExtractStream)

	// ==================== 课件 CRUD(登录即可) ====================
	cwMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// v0.42: /api/v1/coursewares/from-topic — 从主题直接创建课件
		if path == "/api/v1/coursewares/from-topic" || path == "/api/v1/coursewares/from-topic/" {
			if r.Method == http.MethodPost {
				cwHandler.CreateFromTopic(w, r)
			} else {
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		// v0.42 入口B: /api/v1/coursewares/from-ppt — 上传PPT创建课件
		if path == "/api/v1/coursewares/from-ppt" || path == "/api/v1/coursewares/from-ppt/" {
			if r.Method == http.MethodPost {
				cwIndexHandler.CreateFromPPT(w, r)
			} else {
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		// v0.42 入口C: /api/v1/coursewares/from-doc — 上传Word文档创建课件
		if path == "/api/v1/coursewares/from-doc" || path == "/api/v1/coursewares/from-doc/" {
			if r.Method == http.MethodPost {
				cwIndexHandler.CreateFromDoc(w, r)
			} else {
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		// v0.42.11 入口E: /api/v1/coursewares/from-3d — 创建3D互动单页课件
		if path == "/api/v1/coursewares/from-3d" || path == "/api/v1/coursewares/from-3d/" {
			if r.Method == http.MethodPost {
				cwHandler.CreateFrom3D(w, r)
			} else {
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		if path == "/api/v1/coursewares" || path == "/api/v1/coursewares/" {
			switch r.Method {
			case http.MethodGet:
				cwHandler.ListCoursewares(w, r)
			case http.MethodPost:
				cwHandler.CreateCourseware(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}

		// v0.42.8: 字幕轨路由（必须在 video-drafts 之前匹配）
		if strings.Contains(path, "/subtitles") {
			dispatchSubtitleRoutes(w, r, subtitleHandler)
			return
		}

		// v0.42.5: 视频编辑器草稿路由
		if strings.Contains(path, "/video-drafts") {
			draftHandler.HandleDrafts(w, r)
			return
		}

		dispatchCoursewareSubRoutes(w, r, cwHandler, cwIndexHandler, cwGenHandler, cwTplHandler, cwAssetHandler, videoEditHandler)
	}), authMW)
	mux.Handle("/api/v1/coursewares", cwMux)
	mux.Handle("/api/v1/coursewares/", cwMux)

	// ==================== v136: 方案结构预设(登录即可) ====================
	mux.Handle("/api/v1/courseware-presets", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwHandler.GetSchemePresets(w, r)
	}), authMW))

	// ==================== 风格模板查询(登录即可) ====================
	mux.Handle("/api/v1/courseware-templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwHandler.ListTemplates(w, r)
	}), authMW))

	mux.Handle("/api/v1/courseware-templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/with-user") {
			cwTplHandler.ListTemplatesWithUser(w, r)
			return
		}
		if strings.Contains(path, "/personal/") && r.Method == http.MethodDelete {
			cwTplHandler.DeleteMyTemplate(w, r)
			return
		}
		if strings.HasSuffix(path, "/preview") {
			cwHandler.GetTemplatePreview(w, r)
			return
		}
		http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
	}), authMW))

	// ==================== 课件组件库 ====================
	compMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/courseware-components" || path == "/api/v1/courseware-components/" {
			switch r.Method {
			case http.MethodGet:
				cwCompHandler.ListComponents(w, r)
			case http.MethodPost:
				cwCompHandler.CreateComponent(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		dispatchCWComponentSubRoutes(w, r, cwCompHandler)
	}), authMW)
	mux.Handle("/api/v1/courseware-components", compMux)
	mux.Handle("/api/v1/courseware-components/", compMux)

	mux.Handle("/api/v1/courseware-components/match", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCompHandler.MatchComponents(w, r)
	}), authMW))

	// ==================== 种子数据填充(admin) ====================
	mux.Handle("/api/v1/admin/courseware-seed", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwSeedHandler.SeedAll(w, r)
	}), authMW, adminOnly))

	// ==================== Admin 模板管理 ====================
	mux.Handle("/api/v1/admin/courseware-templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/admin/courseware-templates" || path == "/api/v1/admin/courseware-templates/" {
			switch r.Method {
			case http.MethodPost:
				cwSeedHandler.CreateTemplate(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
	}), authMW, adminOnly))

	mux.Handle("/api/v1/admin/courseware-templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			cwSeedHandler.UpdateTemplate(w, r)
		case http.MethodDelete:
			cwSeedHandler.DeleteTemplate(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}), authMW, adminOnly))

	// ==================== v139 新增:模板 AI 操作路由 ====================
	mux.Handle("/api/v1/coursewares/templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatchTemplateAIRoutes(w, r, cwTplHandler)
	}), authMW))
}

// dispatchCoursewareSubRoutes 课件子路由分发
// v0.42 多媒体: 新增 /generate-image, /upload-image, /upload-video, /insert-image, /assets 路由
// 匹配顺序：长suffix先匹配，避免被短suffix误匹配
func dispatchCoursewareSubRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CoursewareHandler, idxH *handlers.CoursewareIndexHandler, genH *handlers.CoursewareGenHandler, cwTplHandler *handlers.CoursewareTemplateHandler, assetH *handlers.CoursewareAssetHandler, videoEditH *handlers.VideoEditHandler) {
	path := r.URL.Path

	// 离线打包下载: /api/v1/coursewares/{id}/export-bundle
	if strings.HasSuffix(path, "/export-bundle") {
		h.ExportBundle(w, r)
		return
	}
	// v0.42.1 高级视频拼接: /api/v1/coursewares/{id}/videos/advanced-concat
	if strings.HasSuffix(path, "/videos/advanced-concat") {
		videoEditH.AdvancedConcat(w, r)
		return
	}
	// v0.42.1 视频编辑路由: /api/v1/coursewares/{id}/videos/concat
	if strings.HasSuffix(path, "/videos/concat") {
		videoEditH.ConcatVideos(w, r)
		return
	}
	// v0.42.1 视频裁剪: /api/v1/coursewares/{id}/videos/trim
	if strings.HasSuffix(path, "/videos/trim") {
		videoEditH.TrimVideo(w, r)
		return
	}
	// v0.42.4 视频静音: /api/v1/coursewares/{id}/videos/mute
	if strings.HasSuffix(path, "/videos/mute") {
		videoEditH.MuteVideo(w, r)
		return
	}
	// v0.42.4 音轨分离: /api/v1/coursewares/{id}/videos/extract-audio
	if strings.HasSuffix(path, "/videos/extract-audio") {
		videoEditH.ExtractAudio(w, r)
		return
	}

	if strings.HasSuffix(path, "/save-as-template") {
		cwTplHandler.SaveAsMyTemplate(w, r)
		return
	}

	// ==================== v0.42 多媒体: 图片操作路由（必须在 /pages/{num} 通用处理之前） ====================

	// AI生成图片: /api/v1/coursewares/{id}/pages/{num}/generate-image
	if strings.HasSuffix(path, "/generate-image") && strings.Contains(path, "/pages/") {
		assetH.GenerateImage(w, r)
		return
	}
	// 手动上传图片: /api/v1/coursewares/{id}/pages/{num}/upload-image
	if strings.HasSuffix(path, "/upload-image") && strings.Contains(path, "/pages/") {
		assetH.UploadImage(w, r)
		return
	}
	// v0.42.5 手动上传视频: /api/v1/coursewares/{id}/pages/{num}/upload-video
	if strings.HasSuffix(path, "/upload-video") && strings.Contains(path, "/pages/") {
		assetH.UploadVideo(w, r)
		return
	}
	// 插入图片到HTML: /api/v1/coursewares/{id}/pages/{num}/insert-image
	if strings.HasSuffix(path, "/insert-image") && strings.Contains(path, "/pages/") {
		assetH.InsertImage(w, r)
		return
	}

	// v0.42.1 AI生成视频: /api/v1/coursewares/{id}/pages/{num}/generate-video
	if strings.HasSuffix(path, "/generate-video") && strings.Contains(path, "/pages/") {
		assetH.GenerateVideo(w, r)
		return
	}
	// v0.42.1 查询视频状态: /api/v1/coursewares/{id}/assets/{asset_id}/video-status
	if strings.HasSuffix(path, "/video-status") && strings.Contains(path, "/assets/") {
		assetH.QueryVideoStatus(w, r)
		return
	}
	// v0.42.10 上传资产到阿里云OSS: /api/v1/coursewares/{id}/assets/{asset_id}/upload-oss
	if strings.HasSuffix(path, "/upload-oss") && strings.Contains(path, "/assets/") {
		assetH.UploadToOSS(w, r)
		return
	}

	// 页面图片列表: /api/v1/coursewares/{id}/pages/{num}/assets
	if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/assets") {
		assetH.ListPageAssets(w, r)
		return
	}

	// 课件全部图片: /api/v1/coursewares/{id}/assets (不含 /pages/)
	// 注意：必须在 /assets/{asset_id} DELETE 之前判断
	if strings.Contains(path, "/assets/") && r.Method == http.MethodDelete {
		assetH.DeleteAsset(w, r)
		return
	}
	if strings.HasSuffix(path, "/assets") && !strings.Contains(path, "/pages/") {
		assetH.ListCoursewareAssets(w, r)
		return
	}

	// ==================== 原有路由 ====================

	// v0.42 入口C: 从Word文档生成索引（必须在其他generate-index之前匹配）
	if strings.HasSuffix(path, "/generate-index-doc") {
		idxH.GenerateIndexFromDoc(w, r)
		return
	}
	// v0.42 入口B: 从PPT生成索引
	if strings.HasSuffix(path, "/generate-index-ppt") {
		idxH.GenerateIndexFromPPT(w, r)
		return
	}
	// v0.42: 从主题生成索引
	if strings.HasSuffix(path, "/generate-index-topic") {
		idxH.GenerateIndexFromTopic(w, r)
		return
	}
	if strings.HasSuffix(path, "/generate-index") {
		idxH.GenerateIndexWithPreset(w, r)
		return
	}
	if strings.HasSuffix(path, "/refine-index") {
		idxH.RefineIndex(w, r)
		return
	}
	if strings.HasSuffix(path, "/rollback-status") {
		h.RollbackStatus(w, r)
		return
	}
	// v0.42.11: 3D互动单页生成
	if strings.HasSuffix(path, "/generate-3d-page") {
		genH.Generate3DPage(w, r)
		return
	}
	if strings.HasSuffix(path, "/generate-preview") {
		genH.GeneratePreview(w, r)
		return
	}
	if strings.HasSuffix(path, "/save-nav-template") {
		genH.SaveNavTemplate(w, r)
		return
	}
	if strings.HasSuffix(path, "/generate-pages") {
		genH.GeneratePages(w, r)
		return
	}
	if strings.HasSuffix(path, "/refine-nav") {
		genH.RefineNav(w, r)
		return
	}
	if strings.HasSuffix(path, "/cancel-generate") {
		genH.CancelGenerate(w, r)
		return
	}
	if strings.HasSuffix(path, "/index-stream") {
		idxH.IndexStream(w, r)
		return
	}
	if strings.HasSuffix(path, "/confirm-index") {
		h.ConfirmIndex(w, r)
		return
	}
	if strings.HasSuffix(path, "/upload-logo") {
		h.UploadLogo(w, r)
		return
	}
	if strings.HasSuffix(path, "/save-style") {
		h.SaveStyleFull(w, r)
		return
	}
	if strings.HasSuffix(path, "/confirm-style") {
		h.ConfirmStyle(w, r)
		return
	}
	if strings.HasSuffix(path, "/style") {
		h.SaveStyle(w, r)
		return
	}
	if strings.HasSuffix(path, "/confirm") && !strings.HasSuffix(path, "/confirm-index") && !strings.HasSuffix(path, "/confirm-style") {
		h.ConfirmCourseware(w, r)
		return
	}
	if strings.HasSuffix(path, "/pages/reorder") {
		h.ReorderPages(w, r)
		return
	}
	if strings.HasSuffix(path, "/pages") || strings.HasSuffix(path, "/pages/") {
		switch r.Method {
		case http.MethodGet:
			h.GetCoursewarePages(w, r)
		case http.MethodPost:
			h.AddPage(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/refine") {
		genH.RefinePage(w, r)
		return
	}
	if strings.Contains(path, "/pages/") {
		switch r.Method {
		case http.MethodPut:
			h.UpdatePageIndex(w, r)
		case http.MethodDelete:
			idxH.DeletePage(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
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
		http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// dispatchCWComponentSubRoutes 课件组件子路由分发
func dispatchCWComponentSubRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CWComponentHandler) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/index") {
		h.CompressIndex(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.GetComponent(w, r)
	case http.MethodPut:
		h.UpdateComponent(w, r)
	case http.MethodDelete:
		h.DeleteComponent(w, r)
	default:
		http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// dispatchTemplateAIRoutes v139 模板 AI 操作路由分发
func dispatchTemplateAIRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CoursewareTemplateHandler) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/extract") {
		h.ExtractFromHTML(w, r)
		return
	}
	if strings.HasSuffix(path, "/publish-targets") {
		h.GetPublishTargets(w, r)
		return
	}
	if strings.HasSuffix(path, "/my-drafts") {
		h.ListMyDrafts(w, r)
		return
	}
	if strings.Contains(path, "/drafts/") && r.Method == http.MethodDelete {
		h.DeleteDraft(w, r)
		return
	}
	if strings.HasSuffix(path, "/refine") {
		h.RefineTemplate(w, r)
		return
	}
	if strings.HasSuffix(path, "/history") {
		h.GetHistory(w, r)
		return
	}
	if strings.HasSuffix(path, "/rollback") {
		h.RollbackToHistory(w, r)
		return
	}
	if strings.HasSuffix(path, "/unpublish") {
		h.UnpublishTemplate(w, r)
		return
	}
	if strings.HasSuffix(path, "/publish") {
		h.PublishDraft(w, r)
		return
	}
	http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
}

// dispatchSubtitleRoutes v0.42.8 字幕轨路由分发
//
// 路由映射:
//   POST   /api/v1/coursewares/{id}/subtitles                     — 创建/更新字幕轨
//   GET    /api/v1/coursewares/{id}/subtitles                     — 查询字幕轨列表
//   DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}            — 删除字幕轨
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt — 导出 SRT 文件
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in    — FFmpeg 硬字幕烧录
func dispatchSubtitleRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CoursewareSubtitleHandler) {
	path := r.URL.Path

	// v0.42.9 TTS 配音: .../subtitles/{sub_id}/generate-tts
	if strings.HasSuffix(path, "/generate-tts") {
		h.GenerateTTS(w, r)
		return
	}
	// 导出 SRT: .../subtitles/{sub_id}/export-srt
	if strings.HasSuffix(path, "/export-srt") {
		h.ExportSRT(w, r)
		return
	}
	// 硬字幕烧录: .../subtitles/{sub_id}/burn-in
	if strings.HasSuffix(path, "/burn-in") {
		h.BurnInSubtitle(w, r)
		return
	}

	// 判断是否有 subtitles/{sub_id}（路径中 /subtitles/ 后还有内容）
	idx := strings.Index(path, "/subtitles/")
	if idx >= 0 {
		rest := path[idx+len("/subtitles/"):]
		rest = strings.TrimSuffix(rest, "/")
		if len(rest) > 0 {
			// DELETE /subtitles/{sub_id}
			if r.Method == http.MethodDelete {
				h.DeleteSubtitle(w, r)
				return
			}
			http.Error(w, `{"code":-1,"message":"字幕子路由仅支持DELETE"}`, http.StatusMethodNotAllowed)
			return
		}
	}

	// /subtitles 根路径: POST=创建/更新, GET=列表
	switch r.Method {
	case http.MethodPost:
		h.UpsertSubtitle(w, r)
	case http.MethodGet:
		h.ListSubtitles(w, r)
	default:
		http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
