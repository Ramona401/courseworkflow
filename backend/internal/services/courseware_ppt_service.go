package services

// courseware_ppt_service.go — PPT上传解析+AI索引生成服务
//
// v0.42 入口B: 从PPT创建课件
// 技术方案: Go原生 archive/zip + encoding/xml 解析PPTX
// PPTX本质是ZIP包，幻灯片文本在 ppt/slides/slide*.xml 的 <a:t> 标签中
//
// 流程:
//   1. 用户上传.pptx文件 → 存储到磁盘
//   2. Go解析ZIP中的slide*.xml → 提取每页标题+正文
//   3. 将PPT内容作为上下文 → 调AI生成课件索引（复用层2方案翻译）
//   4. 写入数据库并SSE广播

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 常量 ====================

const (
	// PPTUploadDir PPT文件物理存储根目录
	PPTUploadDir = "/www/wwwroot/tedna/uploads/courseware-ppt"

	// PPTMaxSize 单个PPT文件最大50MB
	PPTMaxSize = 50 * 1024 * 1024

	// PPTMaxSlides 最多解析的幻灯片数量（防止超大PPT耗时过久）
	PPTMaxSlides = 100
)

// 允许的PPT MIME类型
var pptAllowedMimeTypes = map[string]bool{
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/octet-stream": true, // 某些浏览器上传.pptx时MIME为此
}

var pptServiceLog = logger.WithModule("courseware_ppt_service")

// ==================== PPT解析服务 ====================

// CoursewarePPTService PPT上传解析服务
type CoursewarePPTService struct {
	cfg          *config.Config
	indexService *CoursewareIndexService
}

// NewCoursewarePPTService 创建PPT解析服务
func NewCoursewarePPTService(cfg *config.Config, indexService *CoursewareIndexService) *CoursewarePPTService {
	return &CoursewarePPTService{
		cfg:          cfg,
		indexService: indexService,
	}
}

// ==================== PPT页面提取结果 ====================

// PPTSlide 从PPT中提取的单页内容
type PPTSlide struct {
	SlideNumber int    `json:"slide_number"` // 页码（从1开始）
	Title       string `json:"title"`        // 标题（如果有）
	BodyText    string `json:"body_text"`    // 正文文本
	Notes       string `json:"notes"`        // 备注（演讲者备注）
}

// PPTExtractResult PPT解析完整结果
type PPTExtractResult struct {
	SlideCount int        `json:"slide_count"` // 总页数
	Slides     []PPTSlide `json:"slides"`      // 每页内容
	FileName   string     `json:"file_name"`   // 原始文件名
}

// ==================== 上传+创建课件 ====================

// UploadAndCreateCourseware 上传PPT并创建课件记录
// 流程: 校验文件 → 存储到磁盘 → 解析PPT文本 → 创建课件记录（draft状态）
// 索引生成由前端触发 GenerateIndexFromPPT 异步执行
func (s *CoursewarePPTService) UploadAndCreateCourseware(
	ctx context.Context,
	userID string,
	file multipart.File,
	header *multipart.FileHeader,
	subject string,
	grade string,
	title string,
) (*models.Courseware, *PPTExtractResult, error) {
	// ---- 1. 校验文件大小 ----
	if header.Size > PPTMaxSize {
		return nil, nil, fmt.Errorf("PPT文件过大，最大支持50MB（当前%.1fMB）", float64(header.Size)/(1024*1024))
	}

	// ---- 2. 校验文件扩展名 ----
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pptx" {
		return nil, nil, fmt.Errorf("仅支持.pptx格式的PPT文件（当前: %s）", ext)
	}

	// ---- 3. 校验MIME类型（宽松：某些浏览器不准确） ----
	mimeType := header.Header.Get("Content-Type")
	if mimeType != "" && !pptAllowedMimeTypes[mimeType] {
		// 宽松处理：扩展名是.pptx就放行，不完全依赖MIME
		pptServiceLog.Warn("PPT上传MIME类型非标准",
			"mime", mimeType,
			"filename", header.Filename,
		)
	}

	// ---- 4. 保存文件到磁盘 ----
	storedName := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), sanitizePPTFileName(header.Filename))
	dirPath := filepath.Join(PPTUploadDir, userID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建PPT存储目录失败: %w", err)
	}
	fullPath := filepath.Join(dirPath, storedName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return nil, nil, fmt.Errorf("保存PPT文件失败: %w", err)
	}

	pptServiceLog.Info("PPT文件保存成功",
		"path", fullPath,
		"size", header.Size,
		"user_id", userID,
	)

	// ---- 5. 解析PPT内容 ----
	extractResult, err := s.ExtractPPTContent(fullPath)
	if err != nil {
		// 解析失败不删除文件（可能后续手动处理）
		pptServiceLog.Warn("PPT解析失败，课件仍会创建",
			"path", fullPath,
			"error", err.Error(),
		)
		// 创建空的解析结果，允许用户手动输入
		extractResult = &PPTExtractResult{
			SlideCount: 0,
			Slides:     nil,
			FileName:   header.Filename,
		}
	} else {
		extractResult.FileName = header.Filename
	}

	// ---- 6. 确定课件标题 ----
	if title == "" {
		// 尝试用PPT文件名（去掉扩展名）
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	// ---- 7. 创建课件记录 ----
	// source_file_path 存储相对路径（相对于uploads根目录）
	relPath := filepath.Join(userID, storedName)
	cw := &models.Courseware{
		LessonPlanID:   nil, // 无教案关联
		UserID:         userID,
		Title:          title,
		Subject:        subject,
		Grade:          grade,
		Status:         models.CoursewareStatusDraft,
		SourceType:     models.CWSourcePPTUpload,
		SourceFilePath: relPath,
		PageCount:      0,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		return nil, nil, fmt.Errorf("创建课件记录失败: %w", err)
	}

	pptServiceLog.Info("从PPT创建课件成功",
		"courseware_id", cw.ID,
		"slide_count", extractResult.SlideCount,
		"subject", subject,
		"grade", grade,
		"user_id", userID,
	)

	return cw, extractResult, nil
}

// ==================== PPT内容解析（Go原生ZIP+XML） ====================

// ExtractPPTContent 从.pptx文件提取每页文本内容
// PPTX结构:
//
//	ppt/slides/slide1.xml, slide2.xml, ... — 幻灯片内容
//	ppt/notesSlides/notesSlide1.xml, ...   — 演讲者备注
//
// 每页XML中的文本在 <a:t> 标签（属于 drawingML 命名空间）
func (s *CoursewarePPTService) ExtractPPTContent(pptxPath string) (*PPTExtractResult, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("打开PPTX文件失败: %w", err)
	}
	defer r.Close()

	// ---- 1. 收集所有 slide*.xml 文件 ----
	type slideFile struct {
		Number int
		File   *zip.File
	}
	var slideFiles []slideFile

	// 同时收集 notesSlide*.xml
	noteFiles := make(map[int]*zip.File)

	slideRe := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	noteRe := regexp.MustCompile(`^ppt/notesSlides/notesSlide(\d+)\.xml$`)

	for _, f := range r.File {
		if m := slideRe.FindStringSubmatch(f.Name); m != nil {
			num, _ := strconv.Atoi(m[1])
			slideFiles = append(slideFiles, slideFile{Number: num, File: f})
		}
		if m := noteRe.FindStringSubmatch(f.Name); m != nil {
			num, _ := strconv.Atoi(m[1])
			noteFiles[num] = f
		}
	}

	// 按页码排序
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Number < slideFiles[j].Number
	})

	if len(slideFiles) == 0 {
		return nil, fmt.Errorf("PPTX文件中未找到幻灯片内容")
	}

	// 限制最大解析页数
	if len(slideFiles) > PPTMaxSlides {
		slideFiles = slideFiles[:PPTMaxSlides]
	}

	// ---- 2. 逐页解析文本 ----
	var slides []PPTSlide
	for _, sf := range slideFiles {
		slide, err := s.parseSlideXML(sf.File, sf.Number)
		if err != nil {
			pptServiceLog.Warn("解析单页失败,跳过",
				"slide", sf.Number,
				"error", err.Error(),
			)
			continue
		}

		// 尝试解析对应的备注
		if noteFile, ok := noteFiles[sf.Number]; ok {
			notes, _ := s.extractTextFromXML(noteFile)
			slide.Notes = strings.TrimSpace(notes)
		}

		slides = append(slides, *slide)
	}

	return &PPTExtractResult{
		SlideCount: len(slides),
		Slides:     slides,
	}, nil
}

// parseSlideXML 解析单个slide*.xml文件,提取标题和正文
// PPTX的slide XML结构:
//
//	<p:sld> → <p:cSld> → <p:spTree> → <p:sp> (shape)
//	  每个shape有 <p:nvSpPr> → <p:nvPr> → <p:ph type="title|body|...">
//	  文本在 <p:txBody> → <a:p> → <a:r> → <a:t>
func (s *CoursewarePPTService) parseSlideXML(f *zip.File, slideNum int) (*PPTSlide, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("打开slide文件失败: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("读取slide文件失败: %w", err)
	}

	// 使用XML decoder逐token解析,提取所有 <a:t> 内容
	// 同时识别shape类型（title vs body）
	slide := &PPTSlide{SlideNumber: slideNum}

	// 简化策略: 提取所有文本,第一个非空文本块作为标题,其余为正文
	// 更精确的方式需要解析placeholder type属性,但增加复杂度
	allTexts := extractAllTextBlocks(data)

	if len(allTexts) > 0 {
		slide.Title = allTexts[0]
		if len(allTexts) > 1 {
			slide.BodyText = strings.Join(allTexts[1:], "\n")
		}
	}

	return slide, nil
}

// extractAllTextBlocks 从slide XML中提取所有文本块
// 策略: 按 <p:sp> (shape) 分组,每个shape内的 <a:t> 拼接为一个文本块
// 这样能区分标题shape和内容shape
func extractAllTextBlocks(data []byte) []string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	// 宽松模式:忽略命名空间不匹配等问题
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose

	var blocks []string
	var currentTexts []string
	inShape := false
	depth := 0

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			localName := t.Name.Local
			// 进入一个shape
			if localName == "sp" {
				inShape = true
				depth = 1
				currentTexts = nil
			} else if inShape {
				depth++
			}

		case xml.EndElement:
			localName := t.Name.Local
			if inShape {
				depth--
				// 退出shape时收集文本
				if localName == "sp" || depth <= 0 {
					inShape = false
					text := strings.TrimSpace(strings.Join(currentTexts, ""))
					if text != "" {
						blocks = append(blocks, text)
					}
					currentTexts = nil
				}
			}

		case xml.CharData:
			if inShape {
				text := strings.TrimSpace(string(t))
				if text != "" {
					currentTexts = append(currentTexts, text)
				}
			}
		}
	}

	return blocks
}

// extractTextFromXML 从XML文件提取所有纯文本（用于备注等简单场景）
func (s *CoursewarePPTService) extractTextFromXML(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	blocks := extractAllTextBlocks(data)
	return strings.Join(blocks, "\n"), nil
}

// ==================== 从PPT内容生成课件索引 ====================

// GenerateIndexFromPPT 从PPT内容生成课件索引（异步执行,通过SSE推送进度）
// 流程:
//  1. 读取已存储的PPT文件 → 解析内容
//  2. 构建PPT内容提示词 → 调AI生成方案JSON
//  3. 写入数据库并SSE广播
func (s *CoursewarePPTService) GenerateIndexFromPPT(ctx context.Context, coursewareID string, userID string, preset string) error {
	// ---- 1. 获取课件信息 ----
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		s.broadcastError(coursewareID, "课件不存在: "+err.Error())
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		s.broadcastError(coursewareID, "无权操作此课件")
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status != models.CoursewareStatusDraft && cw.Status != models.CoursewareStatusIndexing {
		s.broadcastError(coursewareID, "当前状态不允许生成方案: "+cw.Status)
		return fmt.Errorf("当前状态不允许生成方案: %s", cw.Status)
	}
	if cw.SourceType != models.CWSourcePPTUpload {
		s.broadcastError(coursewareID, "非PPT来源的课件不能使用此接口")
		return fmt.Errorf("非PPT来源: %s", cw.SourceType)
	}

	// ---- 2. 解析已存储的PPT文件 ----
	if cw.SourceFilePath == "" {
		s.broadcastError(coursewareID, "PPT文件路径为空")
		return fmt.Errorf("PPT文件路径为空")
	}
	pptFullPath := filepath.Join(PPTUploadDir, cw.SourceFilePath)
	extractResult, err := s.ExtractPPTContent(pptFullPath)
	if err != nil {
		s.broadcastError(coursewareID, "PPT文件解析失败: "+err.Error())
		return fmt.Errorf("PPT解析失败: %w", err)
	}
	if extractResult.SlideCount == 0 {
		s.broadcastError(coursewareID, "PPT文件中未提取到任何内容")
		return fmt.Errorf("PPT无内容")
	}

	// ---- 3. 更新课件状态为 indexing ----
	if cw.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusIndexing)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"slide_count":  extractResult.SlideCount,
			"message":      fmt.Sprintf("已解析PPT(%d页)，正在生成课件方案...", extractResult.SlideCount),
		},
	})

	// ---- 4. 构建提示词（PPT内容作为上下文） ----
	userPrompt := s.buildPPTIndexPrompt(cw, extractResult, preset)

	// ---- 5. 加载提示词模板 ----
	schemePrompt, sErr := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	systemPrompt := ""
	if sErr == nil {
		systemPrompt = schemePrompt.Content
	} else {
		systemPrompt = "你是K12课件规划专家，请按要求输出JSON格式的课件方案。"
	}

	// ---- 6. 调用AI（courseware_scheme场景,降级到scanner） ----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_scheme",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		aiCfg, err = ai.GetEffectiveConfig(
			s.cfg.GetAESKey(), "scanner",
			s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
		)
		if err != nil {
			s.broadcastError(coursewareID, "获取AI配置失败")
			return fmt.Errorf("获取AI配置失败: %w", err)
		}
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "AI正在分析PPT内容，规划课件结构..."},
	})

	traceCtx := &ai.TraceContext{SceneCode: "courseware_scheme", UserID: &userID}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		s.broadcastError(coursewareID, "AI规划失败: "+err.Error())
		return fmt.Errorf("AI调用失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "方案生成完成，正在整理..."},
	})

	// ---- 7. 解析JSON输出（复用 indexService 的解析逻辑） ----
	schemes, err := s.indexService.parseSchemeJSON(callResult.Content)
	if err != nil {
		s.broadcastError(coursewareID, "解析方案失败: "+err.Error())
		return fmt.Errorf("解析方案失败: %w", err)
	}
	if len(schemes) == 0 {
		s.broadcastError(coursewareID, "AI未返回有效方案")
		return fmt.Errorf("AI未返回有效方案")
	}

	// ---- 8. 构建CoursewarePage列表 ----
	var pages []*models.CoursewarePage
	for i, sc := range schemes {
		page := &models.CoursewarePage{
			CoursewareID:        coursewareID,
			PageNumber:          i + 1,
			Title:               strings.TrimSpace(sc.Title),
			Purpose:             strings.TrimSpace(sc.Purpose),
			ContentSummary:      strings.TrimSpace(sc.ContentSummary),
			InteractionType:     strings.TrimSpace(sc.InteractionType),
			VisualFormat:        strings.TrimSpace(sc.VisualFormat),
			MediaRequirements:   strings.TrimSpace(sc.MediaRequirements),
			EstimatedComplexity: cwClamp(sc.EstimatedComplexity, 1, 5),
			Status:              models.CWPageStatusPending,
		}
		if page.InteractionType == "" {
			page.InteractionType = "static"
		}
		if page.VisualFormat == "" {
			page.VisualFormat = "text_heavy"
		}
		pages = append(pages, page)
	}

	// 生成概述
	overview := fmt.Sprintf("来源：PPT上传（%s，%d页），%s·%s，共%d页课件方案。",
		extractResult.FileName, extractResult.SlideCount,
		cw.Subject, cw.Grade, len(pages))

	pptServiceLog.Info("PPT索引生成完成",
		"courseware_id", coursewareID,
		"ppt_slides", extractResult.SlideCount,
		"cw_pages", len(pages),
		"model", callResult.ModelUsed,
		"tokens", callResult.TokensUsed,
	)

	return s.indexService.saveAndBroadcast(ctx, coursewareID, overview, pages)
}

// ==================== 提示词构建 ====================

// buildPPTIndexPrompt 构建PPT内容→课件方案的提示词
func (s *CoursewarePPTService) buildPPTIndexPrompt(cw *models.Courseware, ppt *PPTExtractResult, preset string) string {
	var sb strings.Builder
	sb.WriteString("你是K12课件规划专家。\n")
	sb.WriteString("用户提供了一份PPT演示文稿的内容（逐页文本），\n")
	sb.WriteString("请将其转化为结构化的交互式课件方案。\n\n")

	sb.WriteString(fmt.Sprintf("## 基本信息\n"))
	sb.WriteString(fmt.Sprintf("- 学科: %s\n", cw.Subject))
	sb.WriteString(fmt.Sprintf("- 年级: %s\n", cw.Grade))
	sb.WriteString(fmt.Sprintf("- 课件标题: %s\n", cw.Title))
	sb.WriteString(fmt.Sprintf("- PPT原始页数: %d\n\n", ppt.SlideCount))

	sb.WriteString("## PPT内容（逐页）\n\n")
	for _, slide := range ppt.Slides {
		sb.WriteString(fmt.Sprintf("### 第%d页", slide.SlideNumber))
		if slide.Title != "" {
			sb.WriteString(fmt.Sprintf(" — %s", slide.Title))
		}
		sb.WriteString("\n")
		if slide.BodyText != "" {
			sb.WriteString(slide.BodyText)
			sb.WriteString("\n")
		}
		if slide.Notes != "" {
			sb.WriteString(fmt.Sprintf("[演讲者备注] %s\n", slide.Notes))
		}
		sb.WriteString("\n")
	}

	// 注入方案结构预设
	if preset != "" {
		presetObj := models.GetSchemePresetByKey(preset)
		if presetObj != nil && presetObj.PromptHint != "" {
			sb.WriteString("\n")
			sb.WriteString(presetObj.PromptHint)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n## 转化原则\n")
	sb.WriteString("1. 保留PPT的知识结构和教学逻辑\n")
	sb.WriteString("2. 将静态PPT内容升级为交互式课件方案（适度增加交互，不过度）\n")
	sb.WriteString("3. PPT中信息密度高的页面可拆分为多个课件页面\n")
	sb.WriteString("4. PPT中信息少的连续页面可合并为一个课件页面\n")
	sb.WriteString("5. 补充必要的导入页、练习页和总结页（如PPT中缺失）\n\n")

	sb.WriteString("## 输出要求\n")
	sb.WriteString("请输出JSON数组格式。每个元素包含以下字段：\n")
	sb.WriteString("page_number(int), title(string), purpose(string), content_summary(string), ")
	sb.WriteString("interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n\n")
	sb.WriteString("交互类型可选：static/click/drag/input/animation/video/game/quiz\n")
	sb.WriteString("视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n\n")
	sb.WriteString("请只输出JSON数组，不要有任何额外说明文字。")

	return sb.String()
}

// ==================== 辅助函数 ====================

// broadcastError 广播错误事件
func (s *CoursewarePPTService) broadcastError(coursewareID string, message string) {
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEError,
		Data:      map[string]interface{}{"message": message},
	})
}

// sanitizePPTFileName 安全化PPT文件名（去除特殊字符）
func sanitizePPTFileName(name string) string {
	// 保留扩展名
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	// 替换所有非字母数字汉字的字符为下划线
	reg := regexp.MustCompile(`[^\p{L}\p{N}_.-]`)
	base = reg.ReplaceAllString(base, "_")

	// 压缩连续下划线
	for strings.Contains(base, "__") {
		base = strings.ReplaceAll(base, "__", "_")
	}
	base = strings.Trim(base, "_")

	// 限制长度
	if len(base) > 80 {
		base = base[:80]
	}
	if base == "" {
		base = "upload"
	}

	return base + ext
}



