package services

// courseware_auto_assembly_service.go — 课件「全自动一键装配」主编排服务
//
// ============================ 设计目标 ============================
// 老师选定"全自动"交付模式后，本服务把已有的多条独立能力串成一条"总装线"：
//   ① 逐页 HTML 生成（复用 CoursewareGenService 的单页生成范式，与批量生成一致）
//   ② AI 写配图提示词（CoursewareAssetService.SuggestImagePrompt）
//   ③ 图片生成（GenerateImage，内部已自动套用课件风格锚点：图生图 + VAOCI）
//   ④ 上传 OSS 云盘（OSSService.UploadAssetToOSS(本地URL) + 回写 public_oss_url）
//   ⑤ 图片插入页面（InsertImageToPage，按 assetID，内部据 asset 占位/追加落库）
//   ⑥ 视频首帧占位（关键词命中页：SuggestVideoPrompt → 首帧图 → 落分镜）
// 一次性交付图文（+视频占位）齐备的完整课件。
//
// ============================ 交付模式（三档）====================
// 前端"交付模式"选择三档，全自动/中间档两档都调本服务，靠 skipVideo 参数区分：
//   · 纯手动          —— 不走本服务（走 GenerateRemainingPages，只生成 HTML）
//   · 全自动装配      —— skipVideo=false：HTML + 配图 + 视频首帧占位（关键词命中页）
//   · HTML+配图不做视频 —— skipVideo=true ：HTML + 配图，所有页一律跳过视频占位
// skipVideo 一路透传进 cwAssemblyPageContext（只读上下文），单页视频链据此判定是否跳过。
//
// ============================ 关键设计决策 ============================
// 1. 前置强约束：必须已设风格锚点（style_anchor_asset_id 非空），否则拦截。
//    锚点保证每页生图自动带 VAOCI 风格 + 人物一致（由 GenerateImage 内部实现）。
// 2. 双流水线并行（工程核心）：
//    - HTML 流水线：并发 = cfg.CoursewareGenConcurrency（文本网关）
//    - 配图流水线：并发 = cfg.CoursewareAssemblyImgConcurrency（图片网关豆包）
//    两条独立限流、物理隔离；某页 HTML 落库成功即投递配图流水线，
//    HTML 的 goroutine 不等配图完成即释放去下一页 → 生成与配图重叠进行。
// 3. 多图策略：每页只出主图（SuggestImagePrompt 数组的 items[0]），成本可控。
// 4. 图片上云：每页图生成后自动上云拿公网 URL 回写 public_oss_url 再插页面。
// 5. 视频首帧占位：仅方案关键词命中"视频/动画/演示"等的页才生成首帧占位图；
//    且 skipVideo=true 时所有页一律跳过（中间档"不做视频"）。
// 6. best-effort 逐页隔离：任一页任一步失败只记该页失败，绝不拖累其他页。
// 7. 独立端点 + 独立 SSE 事件（字符串 "assembly_*"）+ 独立编排 service，
//    底层 100% 复用现有已验证方法，与"纯手动"路径物理隔离，回滚干净。
//
// ============================ SSE 约定（对齐 courseware_sse_hub.go 真实实现）====
//   发布：GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{EventType: "assembly_xxx", Data: {...}})
//   CWSSEEvent.EventType 为字符串；本项目事件标识即字符串字面量，assembly 事件用 "assembly_*"。
//   错误统一用 EventType "error"（对齐现有 gen 流程 CWSSEError 值 = "error"）。
//
// 关联文件：
//   - courseware_auto_assembly_media.go：单页三条干活链（HTML生成/配图/视频占位）的具体步骤

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwAssemblyLog 全自动装配模块级结构化日志器
var cwAssemblyLog = logger.WithModule("cw_assembly")

// cwAssemblyRunning 标记 coursewareID 是否有全自动装配进行中（防并发重复触发）
var cwAssemblyRunning sync.Map

// ==================== 服务定义 ====================

// CoursewareAutoAssemblyService 全自动装配主编排服务
// 持有三个已有 service 实例（同包，可调其私有方法）+ 全局配置。
type CoursewareAutoAssemblyService struct {
	cfg          *config.Config
	genService   *CoursewareGenService
	assetService *CoursewareAssetService
	ossService   *OSSService
}

// NewCoursewareAutoAssemblyService 创建全自动装配服务
// 参数顺序：cfg, genService, assetService, ossService（routes.go 注入处须与此一致）。
func NewCoursewareAutoAssemblyService(
	cfg *config.Config,
	genService *CoursewareGenService,
	assetService *CoursewareAssetService,
	ossService *OSSService,
) *CoursewareAutoAssemblyService {
	return &CoursewareAutoAssemblyService{
		cfg:          cfg,
		genService:   genService,
		assetService: assetService,
		ossService:   ossService,
	}
}

// ==================== 单页装配结果（用于计数汇总）====================

// cwAssemblyPageResult 单页装配的结果快照
type cwAssemblyPageResult struct {
	pageNum      int
	pageID       string
	title        string
	htmlOK       bool
	imageOK      bool
	imageSkipped bool // 无图片占位/无有效提示词，跳过配图（非失败）
	videoOK      bool
	videoSkipped bool // 无视频需求 或 中间档 skipVideo，跳过视频（非失败）
	errMsg       string
}

// ==================== 单页装配上下文（只读依赖打包，多 goroutine 共享安全）====================

// cwAssemblyPageContext 单页装配需要的只读依赖打包。
// 入口一次性构建，之后各页 goroutine 只读，天然并发安全。
type cwAssemblyPageContext struct {
	coursewareID  string
	userID        string
	schoolID      string
	cw            *models.Courseware
	styleCfg      *cwStyleConfig // 真实类型（gen_service 内小写 cwStyleConfig）
	tplInfo       *cwTemplateInfo
	aiCfg         *ai.EffectiveConfig
	genPrompt     *models.Prompt
	logoURL       string
	orgName       string
	navHTML       string
	lessonContext string
	totalPages    int
	// skipVideo 交付模式"HTML+配图不做视频"（中间档）时为 true：
	//   单页配图链正常走，视频首帧占位链一律跳过（assembleOnePageMedia 内据此判定）。
	//   全自动档 skipVideo=false，视频占位按关键词命中逐页决定。
	skipVideo bool
}

// ==================== 主编排入口 ====================

// AutoAssemble 全自动一键装配课件（异步执行，进度经 SSE "assembly_*" 事件推送）。
// 由 handler 在 go func 内以 context.Background() 调用。
//
// skipVideo：交付模式区分——
//   false = 全自动装配（HTML + 配图 + 视频首帧占位，视频按关键词命中页决定）
//   true  = HTML+配图不做视频（中间档，所有页一律跳过视频占位）
func (s *CoursewareAutoAssemblyService) AutoAssemble(ctx context.Context, coursewareID string, userID string, skipVideo bool) error {
	startTime := time.Now()

	// ---- 防并发：同一课件同时只允许一个装配任务 ----
	if _, busy := cwAssemblyRunning.LoadOrStore(coursewareID, struct{}{}); busy {
		s.pushError(coursewareID, "该课件正在装配中，请勿重复触发")
		return fmt.Errorf("课件正在装配中: %s", coursewareID)
	}
	defer cwAssemblyRunning.Delete(coursewareID)

	// ---- 前置校验（含风格锚点强约束）+ 资源加载（skipVideo 一并写入上下文）----
	pc, pages, err := s.prepareAssembly(ctx, coursewareID, userID, skipVideo)
	if err != nil {
		return err // prepareAssembly 内部已推送具体错误事件
	}

	totalPages := len(pages)

	// ---- 找出待生成 HTML 的页（HTMLContent 为空）----
	var remainingPages []*models.CoursewarePage
	for _, p := range pages {
		if strings.TrimSpace(p.HTMLContent) == "" {
			remainingPages = append(remainingPages, p)
		}
	}
	// 已有 HTML 的页也纳入配图流水线（老师可能先手动生成了部分 HTML）
	alreadyHTMLPages := s.collectAlreadyHtmlPagesForImage(pages, remainingPages)

	// ---- 解析并发数（config 已保证 ≥1，此处再防御一次）----
	htmlConcurrency := s.cfg.CoursewareGenConcurrency
	if htmlConcurrency < 1 {
		htmlConcurrency = 1
	}
	imgConcurrency := s.cfg.CoursewareAssemblyImgConcurrency
	if imgConcurrency < 1 {
		imgConcurrency = 1
	}

	// 装配开始消息按交付模式区分措辞，供前端进度视图显示当前模式
	startMsg := fmt.Sprintf("开始全自动装配（共 %d 页，HTML 待生成 %d 页）...", totalPages, len(remainingPages))
	if skipVideo {
		startMsg = fmt.Sprintf("开始装配（HTML+配图，不做视频；共 %d 页，HTML 待生成 %d 页）...", totalPages, len(remainingPages))
	}

	// ---- 广播装配开始（带 skip_video 供前端进度视图标注模式）----
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: "assembly_start",
		Data: map[string]interface{}{
			"courseware_id":     coursewareID,
			"total_pages":       totalPages,
			"html_pending":      len(remainingPages),
			"image_pipeline":    len(remainingPages) + len(alreadyHTMLPages),
			"html_concurrency":  htmlConcurrency,
			"image_concurrency": imgConcurrency,
			"skip_video":        skipVideo,
			"message":           startMsg,
		},
	})

	// ==================== 双流水线并行 ====================
	var (
		mu             sync.Mutex
		htmlSuccess    int
		htmlFail       int
		imageSuccess   int
		imageFail      int
		imageSkip      int
		videoSuccess   int
		videoSkip      int
		assemblyErrors []string
	)

	// 配图流水线：独立信号量 + WaitGroup
	imgSem := make(chan struct{}, imgConcurrency)
	var imgWG sync.WaitGroup

	// dispatchImage 投递一页到配图流水线（不阻塞主流程）
	dispatchImage := func(page *models.CoursewarePage) {
		imgWG.Add(1)
		imgSem <- struct{}{}
		go func(p *models.CoursewarePage) {
			defer imgWG.Done()
			defer func() { <-imgSem }()

			res := s.assembleOnePageMedia(ctx, pc, p)

			mu.Lock()
			if res.imageSkipped {
				imageSkip++
			} else if res.imageOK {
				imageSuccess++
			} else {
				imageFail++
				if res.errMsg != "" {
					assemblyErrors = append(assemblyErrors, res.errMsg)
				}
			}
			if res.videoSkipped {
				videoSkip++
			} else if res.videoOK {
				videoSuccess++
			}
			mu.Unlock()

			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: "assembly_page_done",
				Data: map[string]interface{}{
					"page_number":   p.PageNumber,
					"page_id":       p.ID,
					"title":         p.Title,
					"image_ok":      res.imageOK,
					"image_skipped": res.imageSkipped,
					"video_ok":      res.videoOK,
					"video_skipped": res.videoSkipped,
					"message":       s.buildPageDoneMessage(p.PageNumber, res),
				},
			})
		}(page)
	}

	// 先把已有 HTML 的页直接投递配图流水线
	for _, p := range alreadyHTMLPages {
		dispatchImage(p)
	}

	// HTML 流水线：逐页生成 HTML，落库成功立即投递配图
	htmlSem := make(chan struct{}, htmlConcurrency)
	var htmlWG sync.WaitGroup

	for _, page := range remainingPages {
		htmlWG.Add(1)
		htmlSem <- struct{}{}
		go func(p *models.CoursewarePage) {
			defer htmlWG.Done()
			defer func() { <-htmlSem }()

			fullHTML, genErr := s.generateOnePageHTML(ctx, pc, p)

			mu.Lock()
			if genErr != nil {
				htmlFail++
				assemblyErrors = append(assemblyErrors, fmt.Sprintf("第%d页HTML生成失败: %v", p.PageNumber, genErr))
				mu.Unlock()
				GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
					EventType: "assembly_progress",
					Data: map[string]interface{}{
						"page_number": p.PageNumber,
						"page_title":  p.Title,
						"stage":       "html",
						"error":       genErr.Error(),
						"message":     fmt.Sprintf("⚠️ 第 %d 页 HTML 生成失败，跳过该页配图", p.PageNumber),
					},
				})
				return
			}
			htmlSuccess++
			mu.Unlock()

			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: "assembly_page_html",
				Data: map[string]interface{}{
					"page_number":  p.PageNumber,
					"page_id":      p.ID,
					"title":        p.Title,
					"html_content": fullHTML,
					"message":      fmt.Sprintf("第 %d 页 HTML 已生成，开始配图...", p.PageNumber),
				},
			})

			// 该页 HTML 已落库；把最新 HTML 写回内存对象，供配图链读 <img> 占位
			p.HTMLContent = fullHTML
			dispatchImage(p)
		}(page)
	}

	htmlWG.Wait() // 等 HTML 全部生成完（且各自已投递配图）
	imgWG.Wait()  // 再等配图流水线全部完成

	// ==================== 完成汇总 ====================
	elapsed := time.Since(startTime)

	mu.Lock()
	fHTMLSuccess, fHTMLFail := htmlSuccess, htmlFail
	fImgSuccess, fImgFail, fImgSkip := imageSuccess, imageFail, imageSkip
	fVideoSuccess, fVideoSkip := videoSuccess, videoSkip
	fErrors := append([]string(nil), assemblyErrors...)
	mu.Unlock()

	// 只要有页成功生成 HTML（或本就有已生成页），推进课件状态到 preview
	if fHTMLSuccess > 0 || len(alreadyHTMLPages) > 0 {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusPreview)
	}

	cwAssemblyLog.Info("全自动装配完成",
		"courseware_id", coursewareID,
		"skip_video", skipVideo,
		"html_success", fHTMLSuccess, "html_fail", fHTMLFail,
		"image_success", fImgSuccess, "image_fail", fImgFail, "image_skip", fImgSkip,
		"video_success", fVideoSuccess, "video_skip", fVideoSkip,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	// 完成消息按交付模式区分：中间档不提视频成功数（本就不做视频）
	var doneMsg string
	if skipVideo {
		doneMsg = fmt.Sprintf(
			"装配完成！HTML 成功 %d 页，配图成功 %d 页（%d 页无需配图，%d 页配图失败）。本次未生成视频占位。",
			fHTMLSuccess, fImgSuccess, fImgSkip, fImgFail,
		)
	} else {
		doneMsg = fmt.Sprintf(
			"全自动装配完成！HTML 成功 %d 页，配图成功 %d 页（%d 页无需配图，%d 页配图失败），视频占位 %d 页。",
			fHTMLSuccess, fImgSuccess, fImgSkip, fImgFail, fVideoSuccess,
		)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: "assembly_done",
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"skip_video":    skipVideo,
			"html_success":  fHTMLSuccess, "html_fail": fHTMLFail,
			"image_success": fImgSuccess, "image_fail": fImgFail, "image_skip": fImgSkip,
			"video_success": fVideoSuccess, "video_skip": fVideoSkip,
			"total_pages": totalPages,
			"elapsed_ms":  elapsed.Milliseconds(),
			"errors":      fErrors,
			"message":     doneMsg,
		},
	})

	return nil
}

// ==================== 前置准备 ====================

// prepareAssembly 前置校验（含风格锚点强约束）+ 资源加载，返回单页装配上下文与页面列表。
// 复用 gen_service 的私有方法（同包可调），与批量生成保持完全一致的风格/模板/教案上下文。
// skipVideo 一并写入返回的上下文，供单页视频链判定是否跳过。
func (s *CoursewareAutoAssemblyService) prepareAssembly(
	ctx context.Context, coursewareID string, userID string, skipVideo bool,
) (*cwAssemblyPageContext, []*models.CoursewarePage, error) {
	// 1. 课件存在 + 归属
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		s.pushError(coursewareID, "课件不存在: "+err.Error())
		return nil, nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		s.pushError(coursewareID, "无权操作此课件")
		return nil, nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 状态校验：generating / preview 才允许装配
	if cw.Status != models.CoursewareStatusGenerating && cw.Status != models.CoursewareStatusPreview {
		s.pushError(coursewareID, "当前状态不允许全自动装配: "+cw.Status)
		return nil, nil, fmt.Errorf("当前状态不允许全自动装配: %s", cw.Status)
	}

	// 3. 导航栏模板已保存（assembleFullPage 需要）
	if strings.TrimSpace(cw.NavTemplateHTML) == "" {
		s.pushError(coursewareID, "请先确认导航栏样式再启用全自动装配")
		return nil, nil, fmt.Errorf("导航栏模板未保存")
	}

	// 4. 【风格锚点强约束】必须已设风格锚点
	if cw.StyleAnchorAssetID == nil || strings.TrimSpace(*cw.StyleAnchorAssetID) == "" {
		s.pushError(coursewareID, "全自动装配需先设置风格锚点，以保证全课件配图风格与人物一致。请先在多媒体中设置一张锚点图。")
		return nil, nil, fmt.Errorf("未设置风格锚点")
	}

	// 5. 页面方案
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		s.pushError(coursewareID, "课件没有页面方案，请先生成索引")
		return nil, nil, fmt.Errorf("课件页面为空")
	}

	// 6. 风格配置 + 模板加载（复用 gen_service 私有方法；loadTemplateInfo 真实签名传 TemplateID 字符串）
	styleCfg := s.genService.parseStyleConfig(cw.StyleConfig)
	tplInfo, tErr := s.genService.loadTemplateInfo(ctx, styleCfg.TemplateID)
	if tErr != nil {
		cwAssemblyLog.Warn("加载模板失败，使用默认风格", "error", tErr, "courseware_id", coursewareID)
		tplInfo = s.genService.defaultTemplateInfo()
	}
	logoURL, orgName := s.genService.resolveLogoAndOrg(ctx, cw, styleCfg)
	// 背景图库：把课件级老师选择的背景URL挂进生成上下文（与批量生成一致）
	s.genService.attachUserBackground(ctx, cw, tplInfo)

	// 7. 导航栏 HTML（assembleFullPage 的 navTemplate 参数）——用课件已存的导航栏模板
	navHTML := cw.NavTemplateHTML

	// 8. 教案正文（非教案来源返空串，行为与批量生成一致）
	lessonContext := loadLessonPlanContextForGen(ctx, cw)

	// 9. 生成提示词
	genPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_generate")
	if err != nil {
		s.pushError(coursewareID, "加载生成提示词失败: "+err.Error())
		return nil, nil, fmt.Errorf("加载生成提示词失败: %w", err)
	}

	// 10. AI 配置（场景码 courseware_generate，与批量生成一致）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_generate",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.pushError(coursewareID, "获取AI配置失败: "+err.Error())
		return nil, nil, fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 11. 操作者学校ID（模型境内/境外分流）
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)

	pc := &cwAssemblyPageContext{
		coursewareID:  coursewareID,
		userID:        userID,
		schoolID:      schoolID,
		cw:            cw,
		styleCfg:      styleCfg,
		tplInfo:       tplInfo,
		aiCfg:         aiCfg,
		genPrompt:     genPrompt,
		logoURL:       logoURL,
		orgName:       orgName,
		navHTML:       navHTML,
		lessonContext: lessonContext,
		totalPages:    len(pages),
		skipVideo:     skipVideo,
	}
	return pc, pages, nil
}

// collectAlreadyHtmlPagesForImage 收集"已有 HTML 但需配图"的页（不在待生成列表内的已生成页）
func (s *CoursewareAutoAssemblyService) collectAlreadyHtmlPagesForImage(
	allPages []*models.CoursewarePage, remainingPages []*models.CoursewarePage,
) []*models.CoursewarePage {
	remainingSet := make(map[string]bool, len(remainingPages))
	for _, p := range remainingPages {
		remainingSet[p.ID] = true
	}
	var result []*models.CoursewarePage
	for _, p := range allPages {
		if strings.TrimSpace(p.HTMLContent) != "" && !remainingSet[p.ID] {
			result = append(result, p)
		}
	}
	return result
}

// ==================== SSE / 消息辅助 ====================

// pushError 推送错误事件（EventType "error"，对齐现有 gen 流程 CWSSEError 值）
func (s *CoursewareAutoAssemblyService) pushError(coursewareID string, message string) {
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEError,
		Data:      map[string]interface{}{"message": message},
	})
}

// buildPageDoneMessage 构造单页装配完成的人话消息
func (s *CoursewareAutoAssemblyService) buildPageDoneMessage(pageNum int, res cwAssemblyPageResult) string {
	parts := []string{fmt.Sprintf("第 %d 页装配完成", pageNum)}
	switch {
	case res.imageSkipped:
		parts = append(parts, "无需配图")
	case res.imageOK:
		parts = append(parts, "配图✓")
	default:
		parts = append(parts, "配图失败")
	}
	if res.videoOK {
		parts = append(parts, "视频占位✓")
	}
	return strings.Join(parts, "，")
}
