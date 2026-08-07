package services

// courseware_auto_assembly_service.go — 课件全自动一键装配主编排
//
// 双流水线：
//   - HTML流水线按文本模型并发生成页面；
//   - 图片流水线按图片模型并发处理页面。
//
// 断点续装：
//   - 只为html_content为空的页面生成HTML；
//   - 已有HTML页面先重新执行背景与画布保护，再进入媒体恢复链；
//   - 每个页面和图片槽位完成后立即写入数据库；
//   - 进程重启后再次调用本服务，只继续数据库中的未完成内容。
//
// 快速部署：
//   - 服务关停时CancelAutoAssembly关闭任务取消信号；
//   - 尚未派发的HTML和图片任务不再派发；
//   - 已在调用远端AI的任务不阻塞部署；
//   - 尚未落库的当前工作由新进程重新执行。
//
// 排版与背景治理：
//   - 正式数据库提示词不在本服务内修改；
//   - prepareAssembly克隆本轮提示词并追加自动装配专用画布硬约束；
//   - 新生成页面和断点续装已有页面均调用ensureAutoAssemblyPagePresentation；
//   - 背景与画布保护失败时只影响当前页，不破坏其他页面。

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

var cwAssemblyLog = logger.WithModule("cw_assembly")

var cwAssemblyRunning sync.Map

// cwAssemblyCancelSignal 是并发安全的一次性取消信号。
type cwAssemblyCancelSignal struct {
	channel chan struct{}
	once    sync.Once
}

func newCWAssemblyCancelSignal() *cwAssemblyCancelSignal {
	return &cwAssemblyCancelSignal{
		channel: make(chan struct{}),
	}
}

func (signal *cwAssemblyCancelSignal) Cancel() {
	if signal == nil {
		return
	}

	signal.once.Do(
		func() {
			close(signal.channel)
		},
	)
}

// cwAssemblyCancelMap 保存当前进程中的装配停止信号。
var cwAssemblyCancelMap sync.Map

func isCWAssemblyCancelled(
	cancelChannel <-chan struct{},
) bool {
	if cancelChannel == nil {
		return false
	}

	select {
	case <-cancelChannel:
		return true
	default:
		return false
	}
}

// CoursewareAutoAssemblyService 全自动装配服务。
type CoursewareAutoAssemblyService struct {
	cfg          *config.Config
	genService   *CoursewareGenService
	assetService *CoursewareAssetService
	ossService   *OSSService
}

// NewCoursewareAutoAssemblyService 创建全自动装配服务。
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

// cwAssemblyPageResult 单页装配结果。
type cwAssemblyPageResult struct {
	pageNum      int
	pageID       string
	title        string
	htmlOK       bool
	imageOK      bool
	imageSkipped bool
	videoOK      bool
	videoSkipped bool
	errMsg       string
}

// cwAssemblyPageContext 单页装配只读上下文。
type cwAssemblyPageContext struct {
	coursewareID  string
	userID        string
	actor         *CoursewareActorContext
	schoolID      string
	cw            *models.Courseware
	styleCfg      *cwStyleConfig
	tplInfo       *cwTemplateInfo
	aiCfg         *ai.EffectiveConfig
	genPrompt     *models.Prompt
	logoURL       string
	orgName       string
	navHTML       string
	lessonContext string
	totalPages    int
	skipVideo     bool
}

// AutoAssemble 全自动装配课件。
func (s *CoursewareAutoAssemblyService) AutoAssemble(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	skipVideo bool,
) error {
	startTime := time.Now()

	courseware, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		s.pushError(
			coursewareID,
			err.Error(),
		)
		return err
	}

	if _, busy := cwAssemblyRunning.LoadOrStore(
		coursewareID,
		struct{}{},
	); busy {
		s.pushError(
			coursewareID,
			"该课件正在装配中，请勿重复触发",
		)

		return fmt.Errorf(
			"课件正在装配中: %s",
			coursewareID,
		)
	}
	defer cwAssemblyRunning.Delete(
		coursewareID,
	)

	cancelSignal :=
		newCWAssemblyCancelSignal()

	cwAssemblyCancelMap.Store(
		coursewareID,
		cancelSignal,
	)
	defer cwAssemblyCancelMap.Delete(
		coursewareID,
	)

	cancelChannel := cancelSignal.channel

	pageContext, pages, err :=
		s.prepareAssembly(
			ctx,
			courseware,
			scopedActor,
			skipVideo,
		)
	if err != nil {
		return err
	}

	// 断点续装页可能来自旧版本生成、手工导入或此前被AI复制了错误背景标记。
	// 在区分“待生成HTML页”和“已有HTML页”之前，先统一修复所有已有HTML页面。
	for _, page := range pages {
		if page == nil ||
			strings.TrimSpace(
				page.HTMLContent,
			) == "" {
			continue
		}

		repairedHTML, repairErr :=
			s.ensureAutoAssemblyPagePresentation(
				ctx,
				pageContext,
				page,
			)

		if repairErr != nil {
			cwAssemblyLog.Warn(
				"断点续装页面背景与画布保护失败，保留原HTML继续处理",
				"courseware_id", coursewareID,
				"page_number", page.PageNumber,
				"page_id", page.ID,
				"error", repairErr,
			)
			continue
		}

		page.HTMLContent =
			repairedHTML
	}

	totalPages := len(pages)

	remainingPages := make(
		[]*models.CoursewarePage,
		0,
		len(pages),
	)

	for _, page := range pages {
		if strings.TrimSpace(
			page.HTMLContent,
		) == "" {
			remainingPages = append(
				remainingPages,
				page,
			)
		}
	}

	alreadyHTMLPages :=
		s.collectAlreadyHtmlPagesForImage(
			pages,
			remainingPages,
		)

	htmlConcurrency :=
		s.cfg.CoursewareGenConcurrency
	if htmlConcurrency < 1 {
		htmlConcurrency = 1
	}

	imageConcurrency :=
		s.cfg.CoursewareAssemblyImgConcurrency
	if imageConcurrency < 1 {
		imageConcurrency = 1
	}

	startMessage := fmt.Sprintf(
		"开始全自动装配或断点续装（共 %d 页，HTML待生成 %d 页）...",
		totalPages,
		len(remainingPages),
	)

	if skipVideo {
		startMessage = fmt.Sprintf(
			"开始装配或断点续装（HTML+配图，不做视频；共 %d 页，HTML待生成 %d 页）...",
			totalPages,
			len(remainingPages),
		)
	}

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: "assembly_start",
			Data: map[string]interface{}{
				"courseware_id": coursewareID,
				"total_pages":   totalPages,
				"html_pending":  len(remainingPages),
				"image_pipeline": len(remainingPages) +
					len(alreadyHTMLPages),
				"html_concurrency":  htmlConcurrency,
				"image_concurrency": imageConcurrency,
				"skip_video":        skipVideo,
				"resume_mode":       len(alreadyHTMLPages) > 0,
				"message":           startMessage,
			},
		},
	)

	var (
		mutex          sync.Mutex
		htmlSuccess    int
		htmlFail       int
		imageSuccess   int
		imageFail      int
		imageSkip      int
		videoSuccess   int
		videoSkip      int
		assemblyErrors []string
	)

	imageSemaphore := make(
		chan struct{},
		imageConcurrency,
	)

	var imageWaitGroup sync.WaitGroup

	// dispatchImage返回false表示任务已取消，调用方应停止继续派发。
	dispatchImage := func(
		page *models.CoursewarePage,
	) bool {
		if isCWAssemblyCancelled(
			cancelChannel,
		) {
			return false
		}

		select {
		case imageSemaphore <- struct{}{}:
		case <-cancelChannel:
			return false
		}

		imageWaitGroup.Add(1)

		go func(
			currentPage *models.CoursewarePage,
		) {
			defer imageWaitGroup.Done()
			defer func() {
				<-imageSemaphore
			}()

			if isCWAssemblyCancelled(
				cancelChannel,
			) {
				return
			}

			result :=
				s.assembleOnePageMediaIAOCI(
					ctx,
					pageContext,
					currentPage,
				)

			mutex.Lock()

			if result.imageSkipped {
				imageSkip++
			} else if result.imageOK {
				imageSuccess++
			} else {
				imageFail++

				if result.errMsg != "" {
					assemblyErrors = append(
						assemblyErrors,
						result.errMsg,
					)
				}
			}

			if result.videoSkipped {
				videoSkip++
			} else if result.videoOK {
				videoSuccess++
			}

			mutex.Unlock()

			GlobalCWSSEHub.Broadcast(
				coursewareID,
				CWSSEEvent{
					EventType: "assembly_page_done",
					Data: map[string]interface{}{
						"page_number":   currentPage.PageNumber,
						"page_id":       currentPage.ID,
						"title":         currentPage.Title,
						"image_ok":      result.imageOK,
						"image_skipped": result.imageSkipped,
						"video_ok":      result.videoOK,
						"video_skipped": result.videoSkipped,
						"message": s.buildPageDoneMessage(
							currentPage.PageNumber,
							result,
						),
					},
				},
			)
		}(page)

		return true
	}

	for _, page := range alreadyHTMLPages {
		if !dispatchImage(page) {
			break
		}
	}

	htmlSemaphore := make(
		chan struct{},
		htmlConcurrency,
	)

	var htmlWaitGroup sync.WaitGroup

dispatchHTMLLoop:
	for _, page := range remainingPages {
		if isCWAssemblyCancelled(
			cancelChannel,
		) {
			break
		}

		select {
		case htmlSemaphore <- struct{}{}:
		case <-cancelChannel:
			break dispatchHTMLLoop
		}

		htmlWaitGroup.Add(1)

		go func(
			currentPage *models.CoursewarePage,
		) {
			defer htmlWaitGroup.Done()
			defer func() {
				<-htmlSemaphore
			}()

			if isCWAssemblyCancelled(
				cancelChannel,
			) {
				return
			}

			fullHTML, generationErr :=
				s.generateOnePageHTML(
					ctx,
					pageContext,
					currentPage,
				)

			// generateOnePageHTML完成首次落库后，再统一覆盖真实背景和画布保护。
			// 该步骤使用数据库最新页面元数据写回，不会把页面状态回退。
			if generationErr == nil {
				repairedHTML, repairErr :=
					s.ensureAutoAssemblyPagePresentation(
						ctx,
						pageContext,
						currentPage,
					)

				if repairErr != nil {
					generationErr =
						fmt.Errorf(
							"页面背景与画布保护失败: %w",
							repairErr,
						)
				} else {
					fullHTML =
						repairedHTML
				}
			}

			mutex.Lock()

			if generationErr != nil {
				htmlFail++

				assemblyErrors = append(
					assemblyErrors,
					fmt.Sprintf(
						"第%d页HTML生成失败: %v",
						currentPage.PageNumber,
						generationErr,
					),
				)

				mutex.Unlock()

				GlobalCWSSEHub.Broadcast(
					coursewareID,
					CWSSEEvent{
						EventType: "assembly_progress",
						Data: map[string]interface{}{
							"page_number": currentPage.PageNumber,
							"page_title":  currentPage.Title,
							"stage":       "html",
							"error":       generationErr.Error(),
							"message": fmt.Sprintf(
								"⚠️ 第 %d 页 HTML 生成失败，跳过该页配图",
								currentPage.PageNumber,
							),
						},
					},
				)

				return
			}

			htmlSuccess++
			mutex.Unlock()

			GlobalCWSSEHub.Broadcast(
				coursewareID,
				CWSSEEvent{
					EventType: "assembly_page_html",
					Data: map[string]interface{}{
						"page_number":  currentPage.PageNumber,
						"page_id":      currentPage.ID,
						"title":        currentPage.Title,
						"html_content": fullHTML,
						"message": fmt.Sprintf(
							"第 %d 页 HTML 已生成并落库，开始逐槽位配图...",
							currentPage.PageNumber,
						),
					},
				},
			)

			currentPage.HTMLContent = fullHTML

			// 部署关停后不再派发新的图片任务。
			_ = dispatchImage(
				currentPage,
			)
		}(page)
	}

	// 正常运行时等待全部任务完成。
	// 快速部署时主进程不会等待本后台任务，本进程会由systemd在短期限内结束。
	htmlWaitGroup.Wait()
	imageWaitGroup.Wait()

	elapsed := time.Since(startTime)

	mutex.Lock()

	finalHTMLSuccess := htmlSuccess
	finalHTMLFail := htmlFail
	finalImageSuccess := imageSuccess
	finalImageFail := imageFail
	finalImageSkip := imageSkip
	finalVideoSuccess := videoSuccess
	finalVideoSkip := videoSkip
	finalErrors := append(
		[]string(nil),
		assemblyErrors...,
	)

	mutex.Unlock()

	wasCancelled :=
		isCWAssemblyCancelled(
			cancelChannel,
		)

	if finalHTMLSuccess > 0 ||
		len(alreadyHTMLPages) > 0 {
		_ = repository.UpdateCoursewareStatus(
			ctx,
			coursewareID,
			models.CoursewareStatusPreview,
		)
	}

	cwAssemblyLog.Info(
		"全自动装配任务收敛",
		"courseware_id", coursewareID,
		"skip_video", skipVideo,
		"cancelled", wasCancelled,
		"html_success", finalHTMLSuccess,
		"html_fail", finalHTMLFail,
		"image_success", finalImageSuccess,
		"image_fail", finalImageFail,
		"image_skip", finalImageSkip,
		"video_success", finalVideoSuccess,
		"video_skip", finalVideoSkip,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	var doneMessage string

	if wasCancelled {
		doneMessage = fmt.Sprintf(
			"装配因服务更新暂停：本次已完成HTML %d页、配图 %d页。刷新后再次点击即可从数据库断点续装。",
			finalHTMLSuccess,
			finalImageSuccess,
		)
	} else if skipVideo {
		doneMessage = fmt.Sprintf(
			"装配完成！HTML成功 %d页，配图成功 %d页（%d页无需配图，%d页存在槽位失败）。本次未生成视频占位。",
			finalHTMLSuccess,
			finalImageSuccess,
			finalImageSkip,
			finalImageFail,
		)
	} else {
		doneMessage = fmt.Sprintf(
			"全自动装配完成！HTML成功 %d页，配图成功 %d页（%d页无需配图，%d页存在槽位失败），视频占位 %d页。",
			finalHTMLSuccess,
			finalImageSuccess,
			finalImageSkip,
			finalImageFail,
			finalVideoSuccess,
		)
	}

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: "assembly_done",
			Data: map[string]interface{}{
				"courseware_id": coursewareID,
				"skip_video":    skipVideo,
				"cancelled":     wasCancelled,
				"html_success":  finalHTMLSuccess,
				"html_fail":     finalHTMLFail,
				"image_success": finalImageSuccess,
				"image_fail":    finalImageFail,
				"image_skip":    finalImageSkip,
				"video_success": finalVideoSuccess,
				"video_skip":    finalVideoSkip,
				"total_pages":   totalPages,
				"elapsed_ms":    elapsed.Milliseconds(),
				"errors":        finalErrors,
				"message":       doneMessage,
			},
		},
	)

	return nil
}

// CancelAutoAssembly 停止继续派发全自动装配的新页面和新图片任务。
//
// 已经成功落库的数据不回滚；
// 新进程再次执行AutoAssemble时从数据库断点续装。
func (s *CoursewareAutoAssemblyService) CancelAutoAssembly(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return err
	}

	value, exists :=
		cwAssemblyCancelMap.Load(
			coursewareID,
		)
	if !exists {
		cwAssemblyLog.Info(
			"当前没有运行中的全自动装配任务",
			"courseware_id",
			coursewareID,
		)
		return nil
	}

	signal, ok :=
		value.(*cwAssemblyCancelSignal)
	if !ok || signal == nil {
		return nil
	}

	signal.Cancel()

	cwAssemblyLog.Info(
		"已通知全自动装配停止继续派发",
		"courseware_id",
		coursewareID,
	)

	return nil
}

// prepareAssembly 执行装配前置检查。
func (s *CoursewareAutoAssemblyService) prepareAssembly(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	skipVideo bool,
) (
	*cwAssemblyPageContext,
	[]*models.CoursewarePage,
	error,
) {
	if courseware == nil {
		return nil, nil,
			ErrCoursewareEducationDomainInvalid
	}

	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil, nil,
			ErrCoursewareActorRequired
	}

	coursewareID := courseware.ID
	userID := actor.UserID

	if courseware.Status !=
		models.CoursewareStatusGenerating &&
		courseware.Status !=
			models.CoursewareStatusPreview {
		s.pushError(
			coursewareID,
			"当前状态不允许全自动装配: "+
				courseware.Status,
		)

		return nil, nil, fmt.Errorf(
			"当前状态不允许全自动装配: %s",
			courseware.Status,
		)
	}

	if strings.TrimSpace(
		courseware.NavTemplateHTML,
	) == "" {
		s.pushError(
			coursewareID,
			"请先确认导航栏样式再启用全自动装配",
		)

		return nil, nil,
			fmt.Errorf("导航栏模板未保存")
	}

	if courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(
			*courseware.StyleAnchorAssetID,
		) == "" {
		s.pushError(
			coursewareID,
			"全自动装配需先设置风格锚点。锚点只锁定艺术风格及明确复用的主体，不会锁定教室、课桌或构图。",
		)

		return nil, nil,
			fmt.Errorf("未设置风格锚点")
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil ||
		len(pages) == 0 {
		s.pushError(
			coursewareID,
			"课件没有页面方案，请先生成索引",
		)

		return nil, nil,
			fmt.Errorf("课件页面为空")
	}

	styleConfig :=
		s.genService.parseStyleConfig(
			courseware.StyleConfig,
		)

	templateInfo, templateErr :=
		s.genService.loadTemplateInfo(
			ctx,
			styleConfig.TemplateID,
		)
	if templateErr != nil {
		cwAssemblyLog.Warn(
			"加载模板失败，使用默认风格",
			"error", templateErr,
			"courseware_id", coursewareID,
		)

		templateInfo =
			s.genService.defaultTemplateInfo()
	}

	logoURL, orgName :=
		s.genService.resolveLogoAndOrg(
			ctx,
			courseware,
			styleConfig,
		)

	s.genService.attachUserBackground(
		ctx,
		courseware,
		templateInfo,
	)

	lessonContext :=
		loadLessonPlanContextForGen(
			ctx,
			courseware,
		)

	generationPrompt, err :=
		repository.GetCurrentPromptByKey(
			"prompt_courseware_generate",
		)
	if err != nil {
		s.pushError(
			coursewareID,
			"加载生成提示词失败: "+err.Error(),
		)

		return nil, nil, fmt.Errorf(
			"加载生成提示词失败: %w",
			err,
		)
	}

	// 只克隆本次自动装配运行使用的提示词对象，
	// 不修改数据库提示词，也不影响手工生成、微调、重生或其他课件功能。
	generationPrompt =
		cloneCoursewarePromptWithAutoAssemblyLayoutRules(
			generationPrompt,
		)

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_generate",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.pushError(
			coursewareID,
			"获取AI配置失败: "+err.Error(),
		)

		return nil, nil, fmt.Errorf(
			"获取AI配置失败: %w",
			err,
		)
	}

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	pageContext := &cwAssemblyPageContext{
		coursewareID:  coursewareID,
		userID:        userID,
		actor:         actor,
		schoolID:      schoolID,
		cw:            courseware,
		styleCfg:      styleConfig,
		tplInfo:       templateInfo,
		aiCfg:         aiConfig,
		genPrompt:     generationPrompt,
		logoURL:       logoURL,
		orgName:       orgName,
		navHTML:       courseware.NavTemplateHTML,
		lessonContext: lessonContext,
		totalPages:    len(pages),
		skipVideo:     skipVideo,
	}

	return pageContext, pages, nil
}

// collectAlreadyHtmlPagesForImage 收集已有HTML页面。
func (s *CoursewareAutoAssemblyService) collectAlreadyHtmlPagesForImage(
	allPages []*models.CoursewarePage,
	remainingPages []*models.CoursewarePage,
) []*models.CoursewarePage {
	remainingSet := make(
		map[string]bool,
		len(remainingPages),
	)

	for _, page := range remainingPages {
		remainingSet[page.ID] = true
	}

	result := make(
		[]*models.CoursewarePage,
		0,
	)

	for _, page := range allPages {
		if strings.TrimSpace(
			page.HTMLContent,
		) != "" &&
			!remainingSet[page.ID] {
			result = append(
				result,
				page,
			)
		}
	}

	return result
}

// pushError 推送装配错误。
func (s *CoursewareAutoAssemblyService) pushError(
	coursewareID string,
	message string,
) {
	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: CWSSEError,
			Data: map[string]interface{}{
				"message": message,
			},
		},
	)
}

// buildPageDoneMessage 构造单页完成消息。
func (s *CoursewareAutoAssemblyService) buildPageDoneMessage(
	pageNumber int,
	result cwAssemblyPageResult,
) string {
	parts := []string{
		fmt.Sprintf(
			"第 %d 页装配完成",
			pageNumber,
		),
	}

	switch {
	case result.imageSkipped:
		parts = append(
			parts,
			"无需配图",
		)

	case result.imageOK:
		parts = append(
			parts,
			"全部图片槽位完成✓",
		)

	default:
		parts = append(
			parts,
			"部分或全部图片槽位失败",
		)
	}

	if result.videoOK {
		parts = append(
			parts,
			"视频占位✓",
		)
	}

	return strings.Join(
		parts,
		"，",
	)
}
