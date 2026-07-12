package routes

// routes_courseware.go — 课件工坊路由注册(v0.42 多入口+PPT上传+多媒体+字幕轨+3D单页)
//
// 阶段1（课件审核与协作·发布与共享）新增路由:
//   - GET  /api/v1/coursewares/shared                 — 共享课件库列表（集合级，cwMux 精确拦截）
//   - POST /api/v1/coursewares/{id}/publish           — 发布/撤回
//   - PUT  /api/v1/coursewares/{id}/code-share-scope  — 设源代码开放范围
//   - POST /api/v1/coursewares/{id}/fork              — 复制共享课件到我的
//
// 阶段3（课件审核与协作·多级审核）新增路由:
//   - POST /api/v1/coursewares/{id}/submit-review     — 作者提交课件进入审核流（挂在 cwMux 子路由）
//   - /api/v1/courseware-reviews 路由组               — L1/L2 审核、历史、详情、待审核、已审核、统计
//     （独立注册函数 registerCoursewareReviewRoutes，见文件末尾）
//
// v0.42.11 新增路由:
//   - POST /api/v1/coursewares/from-3d                            — 创建3D互动单页课件
//   - POST /api/v1/coursewares/{id}/generate-3d-page              — 触发3D单页AI生成
//
// 就地文字编辑新增路由:
//   - POST /api/v1/coursewares/{id}/pages/{num}/save-html         — 保存就地改过的整页HTML（不调AI，存旧版+写新版）
//
// 批次B（粘贴HTML建页）新增路由:
//   - POST /api/v1/coursewares/{id}/pages/{num}/import-html       — 导入外部HTML到指定页（画布归一+导航重编号+背景补注+快照）
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

        // 批次1: 课件背景图库处理器（图集列表 + 课件选择/清除背景并秒换已生成页）
        bgHandler := handlers.NewCoursewareBackgroundHandler()

        // 字体F1: 课件字体方案处理器（方案列表 + 课件选择/清除字体并秒换已生成页）
        fontHandler := handlers.NewCoursewareFontHandler()

        // ==================== 课件索引 SSE(内部 Token 验证,不走 authMW) ====================
        mux.HandleFunc("/api/v1/sse/courseware/", cwIndexHandler.IndexStream)

        // ==================== v139 新增:模板微调 SSE(内部 Token 验证) ====================
        mux.HandleFunc("/api/v1/sse/template-refine/", cwTplHandler.RefineStream)

        // ==================== v145 新增:模板提取 SSE(内部 Token 验证) ====================
        mux.HandleFunc("/api/v1/sse/template-extract", cwTplHandler.ExtractStream)

        // ==================== 课件 CRUD(登录即可) ====================
        cwMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                path := r.URL.Path

                // 阶段1: /api/v1/coursewares/shared — 共享课件库列表（集合级路径，必须在此拦截，
                // 不能落入 dispatchCoursewareSubRoutes，否则 "shared" 会被当成 courseware_id）
                if path == "/api/v1/coursewares/shared" || path == "/api/v1/coursewares/shared/" {
                        if r.Method == http.MethodGet {
                                cwHandler.ListSharedCoursewares(w, r)
                        } else {
                                http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                        }
                        return
                }

                // 阶段4: /api/v1/coursewares/collab/candidates — 集体备课候选成员（同校同组，集合级）
                // 必须在此拦截，否则 "collab" 会被 dispatchCoursewareSubRoutes 当成 courseware_id
                if path == "/api/v1/coursewares/collab/candidates" || path == "/api/v1/coursewares/collab/candidates/" {
                        if r.Method == http.MethodGet {
                                cwHandler.ListCollabCandidates(w, r)
                        } else {
                                http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                        }
                        return
                }

                // 阶段4: /api/v1/coursewares/collab/joined — 我参与的集体备课（集合级，参与者入口）
                if path == "/api/v1/coursewares/collab/joined" || path == "/api/v1/coursewares/collab/joined/" {
                        if r.Method == http.MethodGet {
                                cwHandler.ListJoinedCollab(w, r)
                        } else {
                                http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                        }
                        return
                }

                // 阶段2: /api/v1/coursewares/annotations/{aid}[/resolve] — 批注标记/删除(集合级路径)
                // 必须在此拦截,否则 "annotations" 会被 dispatchCoursewareSubRoutes 当成 courseware_id
                if strings.HasPrefix(path, "/api/v1/coursewares/annotations/") {
                        if strings.HasSuffix(path, "/resolve") {
                                cwHandler.ResolveCWAnnotation(w, r) // PUT 标记已处理/待处理
                        } else {
                                cwHandler.DeleteCWAnnotation(w, r) // DELETE 删除批注
                        }
                        return
                }

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

                // 需求2: /api/v1/coursewares/logo-history — 查询当前用户历史用过的 Logo（去重、最近优先，免重传）
                if path == "/api/v1/coursewares/logo-history" || path == "/api/v1/coursewares/logo-history/" {
                        switch r.Method {
                        case http.MethodGet:
                                cwHandler.ListLogoHistory(w, r)
                        case http.MethodDelete:
                                cwHandler.DeleteLogoHistory(w, r)
                        default:
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

                // 字体F1: 课件字体 /api/v1/coursewares/{id}/font — GET=查当前选择 PUT=选择/清除并秒换
                if strings.HasSuffix(path, "/font") || strings.HasSuffix(path, "/font/") {
                        fontHandler.HandleCoursewareFont(w, r)
                        return
                }

                        // 页级背景覆盖: /api/v1/coursewares/{id}/pages/{num}/page-bg（在课件级 /background 之前匹配）
                        if strings.HasSuffix(path, "/page-bg") && strings.Contains(path, "/pages/") {
                                bgHandler.HandlePageBackground(w, r)
                                return
                        }

                // 批次1: 课件背景 /api/v1/coursewares/{id}/background — GET=查当前选择 PUT=选择/清除并秒换
                if strings.HasSuffix(path, "/background") || strings.HasSuffix(path, "/background/") {
                        bgHandler.HandleCoursewareBackground(w, r)
                        return
                }

                dispatchCoursewareSubRoutes(w, r, cwHandler, cwIndexHandler, cwGenHandler, cwTplHandler, cwAssetHandler, videoEditHandler)
        }), authMW)
        mux.Handle("/api/v1/coursewares", cwMux)
        mux.Handle("/api/v1/coursewares/", cwMux)

        // ==================== 批次1+3: 背景图库子树(列表/AI生成/上传/归档删除/升级系统库, 登录即可, 权限在handler内细分) ====================
        mux.Handle("/api/v1/courseware-backgrounds", middleware.Chain(http.HandlerFunc(bgHandler.HandleLibrary), authMW))
        mux.Handle("/api/v1/courseware-backgrounds/", middleware.Chain(http.HandlerFunc(bgHandler.HandleLibrary), authMW))

        // 字体F1: 字体方案列表(5套系统预设常量, 登录即可)
        mux.Handle("/api/v1/courseware-fonts", middleware.Chain(http.HandlerFunc(fontHandler.ListSchemes), authMW))

	// ==================== 批次C: 代码收藏库(打星收藏课件页HTML, 登录即可, 归属校验在handler内) ====================
	// GET/POST   /api/v1/code-snippets       — 我的收藏列表 / 收藏某课件某页
	// GET/DELETE /api/v1/code-snippets/{id}  — 单条完整HTML / 删除收藏
	snippetHandler := handlers.NewCoursewareSnippetHandler()
	mux.Handle("/api/v1/code-snippets", middleware.Chain(http.HandlerFunc(snippetHandler.HandleCollection), authMW))
	mux.Handle("/api/v1/code-snippets/", middleware.Chain(http.HandlerFunc(snippetHandler.HandleItem), authMW))

        // ==================== S-V1.5: TTS语音合成配置(admin专属) ====================
        // GET/PUT /api/v1/admin/tts-config — 查看/更新TTS provider与火山v3鉴权
        // POST    /api/v1/admin/tts-config/test — 服务端直连合成测试音频验证链路
        ttsCfgHandler := handlers.NewTTSConfigHandler()
        mux.Handle("/api/v1/admin/tts-config", middleware.Chain(http.HandlerFunc(ttsCfgHandler.HandleTTSConfig), authMW, adminOnly))
        mux.Handle("/api/v1/admin/tts-config/test", middleware.Chain(http.HandlerFunc(ttsCfgHandler.TestTTS), authMW, adminOnly))

        // ==================== 批一: 境内文本网关(双网关分流的降级通道)连接配置(admin专属) ====================
        // GET/PUT /api/v1/admin/domestic-gateway      — 查看/更新 domestic_text_base_url/key_enc/model 三键
        // POST    /api/v1/admin/domestic-gateway/test — 服务端直连 dashscope 发一句测试请求验证境内链路
        // PUT 成功后 handler 内调 ai.InvalidateDomesticChannelCache() 让 5 分钟缓存即时失效
        dgHandler := handlers.NewDomesticGatewayHandler()
        mux.Handle("/api/v1/admin/domestic-gateway", middleware.Chain(http.HandlerFunc(dgHandler.HandleDomesticGateway), authMW, adminOnly))
        mux.Handle("/api/v1/admin/domestic-gateway/test", middleware.Chain(http.HandlerFunc(dgHandler.TestDomesticGateway), authMW, adminOnly))
        // 查询境内网关可用模型（GET，admin专属）：用三键调 dashscope /models 列真实模型名
        mux.Handle("/api/v1/admin/domestic-gateway/models", middleware.Chain(http.HandlerFunc(dgHandler.ListDomesticModels), authMW, adminOnly))
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
        // S-V1 配音混入成片: /api/v1/coursewares/{id}/videos/mix-narration
        // 把字幕轨中已生成的TTS配音按时间轴混入指定视频（迭代3.5子专项S）
        if strings.HasSuffix(path, "/videos/mix-narration") {
                videoEditH.MixNarration(w, r)
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
        // 音频裁剪: /api/v1/coursewares/{id}/videos/trim-audio
        if strings.HasSuffix(path, "/videos/trim-audio") {
                videoEditH.TrimAudio(w, r)
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
        // 手动上传音频: /api/v1/coursewares/{id}/pages/{num}/upload-audio
        if strings.HasSuffix(path, "/upload-audio") && strings.Contains(path, "/pages/") {
                assetH.UploadAudio(w, r)
                return
        }
        // 插入图片到HTML: /api/v1/coursewares/{id}/pages/{num}/insert-image
        if strings.HasSuffix(path, "/insert-image") && strings.Contains(path, "/pages/") {
                assetH.InsertImage(w, r)
                return
        }

        // 批次4c+: AI 写详细生图提示词: /api/v1/coursewares/{id}/pages/{num}/suggest-image-prompt
        if strings.HasSuffix(path, "/suggest-image-prompt") && strings.Contains(path, "/pages/") {
                assetH.SuggestImagePrompt(w, r)
                return
        }
        // 批次4c+: AI 写详细视频物料: /api/v1/coursewares/{id}/pages/{num}/suggest-video-prompt
        if strings.HasSuffix(path, "/suggest-video-prompt") && strings.Contains(path, "/pages/") {
                assetH.SuggestVideoPrompt(w, r)
                return
        }
        // 物料存储·读已存生图建议(GET, 不调AI): /api/v1/coursewares/{id}/pages/{num}/image-suggestions
        if strings.HasSuffix(path, "/image-suggestions") && strings.Contains(path, "/pages/") {
                assetH.GetStoredImageSuggestions(w, r)
                return
        }
        // 物料存储·视频分镜(GET=读已存不调AI, POST=保存编辑): /api/v1/coursewares/{id}/pages/{num}/video-storyboards
        if strings.HasSuffix(path, "/video-storyboards") && strings.Contains(path, "/pages/") {
                switch r.Method {
                case http.MethodGet:
                        assetH.GetStoredVideoStoryboards(w, r)
                case http.MethodPost:
                        assetH.SaveVideoStoryboards(w, r)
                default:
                        http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                }
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

        // 风格锚点（轮2）: /api/v1/coursewares/{id}/style-anchor
        //   POST=设锚点(取URL→提取VAOCI→落库)  DELETE=清锚点
        if strings.HasSuffix(path, "/style-anchor") {
                switch r.Method {
                case http.MethodPost:
                        assetH.SetStyleAnchor(w, r)
                case http.MethodDelete:
                        assetH.ClearStyleAnchor(w, r)
                default:
                        http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                }
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
        // 课件↔教案对齐报告：查询（GET）/ 手动重算（POST）
        if strings.HasSuffix(path, "/alignment-report") {
                idxH.GetAlignmentReport(w, r)
                return
        }
        // 断裂B: 取课件关联教案正文（对照抽屉）
        if strings.HasSuffix(path, "/lesson-plan-content") {
                idxH.GetLessonPlanContent(w, r)
                return
        }
        if strings.HasSuffix(path, "/recheck-alignment") {
                idxH.RecheckAlignment(w, r)
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
        // 全自动一键装配: /api/v1/coursewares/{id}/auto-assemble
        // HTML生成+配图+视频占位总装线，异步执行经SSE推 assembly_* 进度。挂在 /generate-pages 之前（长后缀优先惯例）。
        if strings.HasSuffix(path, "/auto-assemble") {
                genH.AutoAssemble(w, r)
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
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/regenerate") {
                genH.RegeneratePage(w, r)
                return
        }
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/refine") {
                genH.RefinePage(w, r)
                return
        }
        // 就地文字编辑保存: /api/v1/coursewares/{id}/pages/{num}/save-html（POST）
        //   前端「就地改文字」编辑器改完文字/字号/颜色后，把整页纯净HTML回传本端点落库（不调AI，只存旧版+写新版）。
        //   必须在通用 /pages/ 分支之前匹配（POST 带 /pages/，否则会落到只认 PUT/DELETE 的通用分支返 405）。
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/save-html") {
                genH.SavePageHTML(w, r)
                return
        }
        // 粘贴HTML导入: /api/v1/coursewares/{id}/pages/{num}/import-html（POST，批次B）
        //   Step5「＋添加页面 → 📋 粘贴HTML」模式：addPage 建页后把粘贴的外部HTML经本端点导入
        //   （service 层做画布归一/导航栏重编号/背景补注/版本快照）。同 save-html 口径，
        //   必须在通用 /pages/ 分支之前匹配（POST 带 /pages/，否则落入只认 PUT/DELETE 的通用分支返 405）。
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/import-html") {
                genH.ImportPageHTML(w, r)
                return
        }
        // 页面级版本对比：取单个历史版本完整HTML（GET）/api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
        //   必须在 HasSuffix "/versions"（列表）分支之前匹配：
        //   本路径以 versionId 结尾（非 /versions 结尾），用 Contains "/versions/" 识别；
        //   若排在列表分支之后，会因两者都不 HasSuffix 而落到通用 /pages/ 分支被当 PUT/DELETE 返 405。
        if strings.Contains(path, "/pages/") && strings.Contains(path, "/versions/") {
                genH.GetPageVersionDetail(w, r)
                return
        }
        // 页面级版本与回退：版本列表（GET）/api/v1/coursewares/{id}/pages/{num}/versions
        //   必须在通用 /pages/ 分支之前匹配（GET 带 /pages/，否则会落到只认 PUT/DELETE 的通用分支返回 405）
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/versions") {
                genH.ListPageVersions(w, r)
                return
        }
        // 页面级版本与回退：回退到指定版本（POST）/api/v1/coursewares/{id}/pages/{num}/rollback
        if strings.Contains(path, "/pages/") && strings.HasSuffix(path, "/rollback") {
                genH.RollbackPage(w, r)
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

        // ==================== 阶段2: 页级批注(创建/列表,带真实课件ID,置于兜底 switch 之前)====================
        // /api/v1/coursewares/{id}/annotations — POST 创建批注 / GET 列出全部批注
        if strings.HasSuffix(path, "/annotations") || strings.HasSuffix(path, "/annotations/") {
                switch r.Method {
                case http.MethodPost:
                        h.CreateCWAnnotation(w, r)
                case http.MethodGet:
                        h.ListCWAnnotations(w, r)
                default:
                        http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
                }
                return
        }

        // ==================== 阶段1: 发布与共享 + 产权分级（置于兜底 switch 之前）====================
        // 发布/撤回: /api/v1/coursewares/{id}/publish
        if strings.HasSuffix(path, "/publish") {
                h.PublishCourseware(w, r)
                return
        }
        // 设源代码开放范围: /api/v1/coursewares/{id}/code-share-scope
        if strings.HasSuffix(path, "/code-share-scope") {
                h.SetCodeShareScope(w, r)
                return
        }
        // 复制共享课件到我的: /api/v1/coursewares/{id}/fork
        if strings.HasSuffix(path, "/fork") {
                h.ForkCourseware(w, r)
                return
        }

        // ==================== 阶段3: 提交审核（作者发起，置于兜底 switch 之前）====================
        // 提交审核: POST /api/v1/coursewares/{id}/submit-review
        // 由课件审核处理器接管；handler 在包内通过包级变量 cwReviewHandlerRef 引用，
        // 在 registerCoursewareReviewRoutes 中注入（见文件末尾）。
        if strings.HasSuffix(path, "/submit-review") {
                if cwReviewHandlerRef != nil {
                        cwReviewHandlerRef.SubmitForReview(w, r)
                } else {
                        http.Error(w, `{"code":-1,"message":"审核服务未就绪"}`, http.StatusServiceUnavailable)
                }
                return
        }

        // ==================== 阶段4: 集体备课（置于兜底 switch 之前，长后缀优先）====================
        // 发起集体备课: POST /api/v1/coursewares/{id}/collab/start
        if strings.HasSuffix(path, "/collab/start") {
                h.StartCollab(w, r)
                return
        }
        // 结束集体备课: POST /api/v1/coursewares/{id}/collab/end
        if strings.HasSuffix(path, "/collab/end") {
                h.EndCollab(w, r)
                return
        }
        // 移除参与者: DELETE /api/v1/coursewares/{id}/collab/members/{uid}
        //   （path 含 /collab/members/ 说明后面还带 uid，必须在 POST /collab/members 之前判断）
        if strings.Contains(path, "/collab/members/") {
                h.RemoveCollabMember(w, r)
                return
        }
        // 加参与者: POST /api/v1/coursewares/{id}/collab/members
        if strings.HasSuffix(path, "/collab/members") {
                h.AddCollabMember(w, r)
                return
        }
        // 查集体备课状态: GET /api/v1/coursewares/{id}/collab（后缀最短，放最后）
        if strings.HasSuffix(path, "/collab") || strings.HasSuffix(path, "/collab/") {
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
//
//	POST   /api/v1/coursewares/{id}/subtitles                     — 创建/更新字幕轨
//	GET    /api/v1/coursewares/{id}/subtitles                     — 查询字幕轨列表
//	DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}            — 删除字幕轨
//	POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt — 导出 SRT 文件
//	POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in    — FFmpeg 硬字幕烧录
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

// ==================== 阶段3: 课件多级审核路由组 ====================

// cwReviewHandlerRef 课件审核处理器的包级引用。
// 因 dispatchCoursewareSubRoutes 的函数签名固定（不便新增参数），
// submit-review 子路由通过此包级变量调用审核处理器；
// 由 registerCoursewareReviewRoutes 在启动注册时注入。
var cwReviewHandlerRef *handlers.CoursewareReviewHandler

// registerCoursewareReviewRoutes 注册课件多级审核路由组（阶段3，登录即可，细粒度权限在 service 层裁决）。
//
// 路由前缀 /api/v1/courseware-reviews/ 与教案 /api/v1/reviews/ 物理隔离。
// 同时注入 cwReviewHandlerRef，使 cwMux 子路由的 submit-review 能调用审核处理器。
//
// 路由映射（镜像 routes_review_v2.go 的 Chain 通配分发，固定路径优先）：
//
//	GET  /api/v1/courseware-reviews/pending          待审核列表
//	GET  /api/v1/courseware-reviews/reviewed         已审核记录列表
//	GET  /api/v1/courseware-reviews/stats            审核统计
//	POST /api/v1/courseware-reviews/{id}/l1          L1 教研组审核
//	POST /api/v1/courseware-reviews/{id}/l2          L2 学校审核
//	GET  /api/v1/courseware-reviews/{id}/history     审核历史
//	GET  /api/v1/courseware-reviews/{id}/detail      审核详情（课件+批注+历史）
func registerCoursewareReviewRoutes(
        mux *http.ServeMux,
        authMW func(http.Handler) http.Handler,
        cwReviewHandler *handlers.CoursewareReviewHandler,
) {
        // 注入包级引用，供 dispatchCoursewareSubRoutes 的 submit-review 分支使用
        cwReviewHandlerRef = cwReviewHandler

        reviewMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                path := r.URL.Path
                rest := strings.TrimPrefix(path, "/api/v1/courseware-reviews/")

                // 固定路径优先匹配
                switch {
                case rest == "pending" && r.Method == http.MethodGet:
                        cwReviewHandler.GetPendingReviews(w, r)
                        return
                case rest == "reviewed" && r.Method == http.MethodGet:
                        cwReviewHandler.GetReviewedRecords(w, r)
                        return
                case rest == "stats" && r.Method == http.MethodGet:
                        cwReviewHandler.GetReviewStats(w, r)
                        return
                }

                // 动态路径：{id}/l1, {id}/l2, {id}/history, {id}/detail
                if strings.HasSuffix(path, "/l1") {
                        cwReviewHandler.ReviewL1(w, r)
                        return
                }
                if strings.HasSuffix(path, "/l2") {
                        cwReviewHandler.ReviewL2(w, r)
                        return
                }
                if strings.HasSuffix(path, "/history") {
                        cwReviewHandler.GetReviewHistory(w, r)
                        return
                }
                if strings.HasSuffix(path, "/detail") {
                        cwReviewHandler.GetReviewDetail(w, r)
                        return
                }

                http.NotFound(w, r)
        }), authMW)

        mux.Handle("/api/v1/courseware-reviews", reviewMux)
        mux.Handle("/api/v1/courseware-reviews/", reviewMux)
}
