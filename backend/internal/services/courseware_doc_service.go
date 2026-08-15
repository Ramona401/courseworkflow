package services

// courseware_doc_service.go — Word文档(.docx)上传解析+AI索引生成服务
//
// v0.42 入口C: 从Word教案文档创建课件
// 技术方案: Go原生 archive/zip + encoding/xml 解析DOCX
// .docx本质是ZIP包，文本在 word/document.xml 的 <w:t> 标签中
//
// 从 courseware_ppt_service.go 拆分，复用 CoursewarePPTService 的方法接收器
// 因为PPT和Word共享同一个服务实例（都需要cfg和indexService）
//
// v0.43 修复（页数过少）:
//   buildDocIndexPrompt 此前未给AI任何页数下限/区间约束，导致7000+字教案
//   被模型概括成1页。本次补齐：按字数估算应展开页数 + 明确学段页数区间 +
//   结构骨架要求 + 禁止概括成数页的硬约束（与 buildTopicDirectPrompt 对齐）。
//
// v0.44 新增（Y+方案：出完方案就出脉络 + 后台补AOCI索引）:
//   GenerateIndexFromDoc 在层2直翻出页后：
//     1) 前台立即用 haiku 生成"哪几页干什么"的真脉络（GenerateOverviewFromPages），
//        替代原先的套话 overview；失败则退回套话，不阻塞。
//     2) saveAndBroadcast 成功后，go func 异步触发 TriggerPageIndexBackfillTracked，
//        对照 docx 原文 + 当前方案为每页补 AOCI 索引（page_index + CG/IL/VF），
//        供后续资源评估；失败不影响课件可用。

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// 与PPT不同，Word文档是连续文本，没有分页概念

const (
	// DocUploadDir Word文档物理存储根目录
	DocUploadDir = "/www/wwwroot/tedna/uploads/courseware-doc"

	// DocMaxSize 单个Word文件最大30MB
	DocMaxSize = 30 * 1024 * 1024
)

// Word MIME类型
var docAllowedMimeTypes = map[string]bool{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/octet-stream": true,
}

// DocExtractResult Word文档解析结果
type DocExtractResult struct {
	FileName   string   `json:"file_name"`  // 原始文件名
	WordCount  int      `json:"word_count"` // 字符数
	Paragraphs []string `json:"paragraphs"` // 段落列表
	FullText   string   `json:"full_text"`  // 完整文本
}

// UploadDocAndCreateCourseware 上传Word文档并创建课件记录
func (s *CoursewarePPTService) UploadDocAndCreateCourseware(
	ctx context.Context,
	actor *CoursewareActorContext,
	file multipart.File,
	header *multipart.FileHeader,
	subject string,
	grade string,
	title string,
) (*models.Courseware, *DocExtractResult, error) {
	domain, err := ResolveCoursewareCreationEducationDomain(actor)
	if err != nil {
		return nil, nil, err
	}
	userID := actor.UserID

	// ---- 1. 校验文件大小 ----
	if header.Size > DocMaxSize {
		return nil, nil, newCoursewareSourceUploadError(
			CoursewareSourceUploadTooLarge,
			fmt.Sprintf(
				"Word文件过大，最大支持30MB（当前%.1fMB）",
				float64(header.Size)/(1024*1024),
			),
			nil,
		)
	}
	if header.Size <= 0 {
		return nil, nil, newCoursewareSourceUploadError(
			CoursewareSourceUploadInvalidFormat,
			"这个Word文件为空，请重新选择有效的.docx文件。",
			nil,
		)
	}

	// ---- 2. 校验文件扩展名 ----
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".docx" {
		return nil, nil, newCoursewareSourceUploadError(
			CoursewareSourceUploadInvalidExtension,
			fmt.Sprintf(
				"仅支持.docx格式的Word文件（当前: %s）",
				ext,
			),
			nil,
		)
	}

	// ---- 3. 保存文件到磁盘 ----
	storedName := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), sanitizePPTFileName(header.Filename))
	dirPath := filepath.Join(DocUploadDir, userID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建文档存储目录失败: %w", err)
	}
	fullPath := filepath.Join(dirPath, storedName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(fullPath)
		return nil, nil, fmt.Errorf("保存Word文件失败: %w", err)
	}

	pptServiceLog.Info("Word文件保存成功",
		"path", fullPath,
		"size", header.Size,
		"user_id", userID,
	)

	// ---- 4. 解析Word内容 ----
	extractResult, err := s.ExtractDocContent(fullPath)
	if err != nil {
		pptServiceLog.Warn(
			"Word上传文件结构无效，已拒绝创建空课件",
			"path", fullPath,
			"error", err.Error(),
		)
		_ = os.Remove(fullPath)
		return nil, nil, newCoursewareSourceUploadError(
			CoursewareSourceUploadInvalidFormat,
			"无法读取这个Word文件：文件扩展名虽然是.docx，但内部不是有效的Word DOCX文档。请用Word/WPS打开原文件，选择“另存为 Word 文档（.docx）”后重新上传。",
			err,
		)
	}
	extractResult.FileName = header.Filename

	if extractResult.WordCount < 50 {
		pptServiceLog.Warn(
			"Word上传未提取到足够文字，已拒绝创建空课件",
			"path", fullPath,
			"word_count", extractResult.WordCount,
		)
		_ = os.Remove(fullPath)
		return nil, nil, newCoursewareSourceUploadError(
			CoursewareSourceUploadNoUsableContent,
			fmt.Sprintf(
				"这个Word文件可以打开，但只提取到%d字，无法生成课件。请确认文档中包含可选择的正文文字；如果主要内容是扫描图片，请先转换为可提取文字的DOCX后重新上传。",
				extractResult.WordCount,
			),
			nil,
		)
	}

	// ---- 5. 确定课件标题 ----
	if title == "" {
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	// ---- 6. 创建课件记录 ----
	relPath := filepath.Join(userID, storedName)
	cw := &models.Courseware{
		LessonPlanID:    nil,
		UserID:          userID,
		Title:           title,
		Subject:         subject,
		EducationDomain: domain,
		Grade:           grade,
		Status:          models.CoursewareStatusDraft,
		SourceType:      models.CWSourceDocUpload,
		SourceFilePath:  relPath,
		PageCount:       0,
	}

	if err := repository.CreateCourseware(ctx, cw); err != nil {
		_ = os.Remove(fullPath)
		return nil, nil, fmt.Errorf("创建课件记录失败: %w", err)
	}

	pptServiceLog.Info("从Word文档创建课件成功",
		"courseware_id", cw.ID,
		"word_count", extractResult.WordCount,
		"subject", subject,
		"grade", grade,
		"user_id", userID,
	)

	return cw, extractResult, nil
}

// ExtractDocContent 从.docx文件提取文本内容
// docx结构: word/document.xml 中的 <w:t> 标签包含所有文本
// <w:p> 标签分隔段落
func (s *CoursewarePPTService) ExtractDocContent(docxPath string) (*DocExtractResult, error) {
	r, err := zip.OpenReader(docxPath)
	if err != nil {
		return nil, fmt.Errorf("打开docx文件失败: %w", err)
	}
	defer r.Close()

	// 找到 word/document.xml
	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return nil, fmt.Errorf("docx文件中未找到 word/document.xml")
	}

	rc, err := docFile.Open()
	if err != nil {
		return nil, fmt.Errorf("打开document.xml失败: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("读取document.xml失败: %w", err)
	}

	// 解析XML: 按 <w:p> 分段，每段内的 <w:t> 拼接为段落文本
	paragraphs := extractDocParagraphs(data)

	// 组装完整文本
	var nonEmpty []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	fullText := strings.Join(nonEmpty, "\n\n")

	return &DocExtractResult{
		WordCount:  len([]rune(fullText)),
		Paragraphs: nonEmpty,
		FullText:   fullText,
	}, nil
}

// extractDocParagraphs 从document.xml中按段落提取文本
// <w:p> 标签界定段落边界，<w:t> 标签包含文本内容
func extractDocParagraphs(data []byte) []string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose

	var paragraphs []string
	var currentParaTexts []string
	inParagraph := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			// <w:p> 开始新段落
			if t.Name.Local == "p" && (t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" || t.Name.Space == "" || t.Name.Space == "w") {
				inParagraph = true
				currentParaTexts = nil
			}

		case xml.EndElement:
			// </w:p> 结束段落
			if t.Name.Local == "p" && inParagraph {
				inParagraph = false
				paraText := strings.Join(currentParaTexts, "")
				paragraphs = append(paragraphs, paraText)
				currentParaTexts = nil
			}

		case xml.CharData:
			if inParagraph {
				text := string(t)
				if text != "" {
					currentParaTexts = append(currentParaTexts, text)
				}
			}
		}
	}

	return paragraphs
}

// GenerateIndexFromDoc 从Word文档内容生成课件索引
func (s *CoursewarePPTService) GenerateIndexFromDoc(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	preset string,
	customHint string,
) error {
	// ---- 1. 可信Actor二次授权并重新加载正式课件 ----
	cw, scopedActor, err :=
		loadOwnedCoursewareForSchemeMutation(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		s.broadcastError(
			coursewareID,
			"课件授权失败: "+err.Error(),
		)
		return err
	}

	userID := scopedActor.UserID

	if cw.Status != models.CoursewareStatusDraft &&
		cw.Status != models.CoursewareStatusIndexing {
		s.broadcastError(
			coursewareID,
			"当前状态不允许生成方案: "+cw.Status,
		)
		return fmt.Errorf(
			"当前状态不允许生成方案: %s",
			cw.Status,
		)
	}
	if cw.SourceType != models.CWSourceDocUpload {
		s.broadcastError(
			coursewareID,
			"非文档来源的课件不能使用此接口",
		)
		return fmt.Errorf(
			"非文档来源: %s",
			cw.SourceType,
		)
	}

	// ---- 2. 解析已存储的Word文件 ----
	if cw.SourceFilePath == "" {
		s.broadcastError(coursewareID, "文档文件路径为空")
		return fmt.Errorf("文档文件路径为空")
	}
	docFullPath := filepath.Join(DocUploadDir, cw.SourceFilePath)
	extractResult, err := s.ExtractDocContent(docFullPath)
	if err != nil {
		s.broadcastError(coursewareID, "文档解析失败: "+err.Error())
		return fmt.Errorf("文档解析失败: %w", err)
	}
	if extractResult.WordCount < 50 {
		s.broadcastError(coursewareID, "文档内容过少（少于50字），无法生成课件方案")
		return fmt.Errorf("文档内容过少")
	}

	// ---- 3. 更新状态 ----
	if cw.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusIndexing)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"word_count":    extractResult.WordCount,
			"message":       fmt.Sprintf("已解析文档(%d字)，正在生成课件方案...", extractResult.WordCount),
		},
	})

	// ---- 4. 构建提示词 ----
	userPrompt := s.buildDocIndexPrompt(cw, extractResult, preset, customHint)

	// ---- 5. 加载系统提示词 ----
	schemePrompt, sErr := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	systemPrompt := ""
	if sErr == nil {
		systemPrompt = schemePrompt.Content
	} else {
		systemPrompt = "你是K12课件规划专家，请按要求输出JSON格式的课件方案。"
	}

	// ---- 6. 调用AI ----
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
		Data:      map[string]interface{}{"message": "AI正在分析教案文档，规划课件结构..."},
	})

	// v198：解析操作者所属学校ID，供模型境内/境外分流判定（Word文档直翻方案，操作者=userID）
	docSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "courseware_scheme", UserID: &userID, SchoolID: schoolIDPtr(docSchoolID)}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		s.broadcastError(coursewareID, "AI规划失败: "+err.Error())
		return fmt.Errorf("AI调用失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "方案生成完成，正在整理..."},
	})

	// ---- 7. 解析JSON ----
	schemes, err := s.indexService.parseSchemeJSON(callResult.Content)
	if err != nil {
		s.broadcastError(coursewareID, "解析方案失败: "+err.Error())
		return fmt.Errorf("解析方案失败: %w", err)
	}
	if len(schemes) == 0 {
		s.broadcastError(coursewareID, "AI未返回有效方案")
		return fmt.Errorf("AI未返回有效方案")
	}

	// ---- 8. 构建CoursewarePage ----
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

	// ---- 9. 出完方案就出脉络（前台快速，haiku） ----
	// 用已生成的页面方案让 haiku 快速概括"哪几页干什么"，替代套话 overview。
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "正在生成课件脉络..."},
	})
	overview := s.indexService.GenerateOverviewFromPages(ctx, userID, cw.Title, cw.Subject, cw.Grade, pages)
	if strings.TrimSpace(overview) == "" {
		// 脉络生成失败，退回套话（不阻塞主流程）
		overview = fmt.Sprintf("来源：教案文档上传（%s，%d字），%s·%s，共%d页课件方案。",
			extractResult.FileName, extractResult.WordCount,
			cw.Subject, cw.Grade, len(pages))
	}

	pptServiceLog.Info("文档索引生成完成",
		"courseware_id", coursewareID,
		"word_count", extractResult.WordCount,
		"cw_pages", len(pages),
		"model", callResult.ModelUsed,
		"tokens", callResult.TokensUsed,
	)

	// ---- 10. 保存并广播（方案+脉络，前台立即可见） ----
	if err := s.indexService.saveAndBroadcast(ctx, coursewareID, scopedActor, overview, pages); err != nil {
		return err
	}

	// ---- 11. 后台异步补 AOCI 索引（对照 docx 原文 + 当前方案，不阻塞前台） ----
	// 供后续资源评估使用；失败不影响课件可用。
	rawText := extractResult.FullText

	backfillResult, backfillErr :=
		s.indexService.TriggerPageIndexBackfillTracked(
			ctx,
			coursewareID,
			scopedActor,
			rawText,
		)

	switch {
	case backfillErr != nil:
		pptServiceLog.Warn(
			"Word页索引回填任务授权失败，课件方案主体不受影响",
			"courseware_id", coursewareID,
			"error", backfillErr,
		)

	case backfillResult ==
		BackgroundRejectedDraining:
		pptServiceLog.Info(
			"服务正在排空，跳过Word页索引回填",
			"courseware_id", coursewareID,
		)

	case backfillResult ==
		BackgroundAlreadyRunning:
		pptServiceLog.Info(
			"Word页索引回填任务已经运行，无需重复启动",
			"courseware_id", coursewareID,
		)

	case backfillResult ==
		BackgroundInvalid:
		pptServiceLog.Warn(
			"Word页索引回填任务参数无效，课件方案主体不受影响",
			"courseware_id", coursewareID,
		)
	}

	return nil
}

// cwRecommendPageRange 根据年级学段与教案字数，给出建议页数下限与区间文字
// 返回 (最小页数, 区间描述)；用于在提示词中对AI施加明确的页数约束，
// 避免长教案被概括成极少页数。
func cwRecommendPageRange(grade string, wordCount int) (int, string) {
	// 先按学段定基础区间
	seg := cwGradeSegment(grade)
	var baseMin, baseMax int
	switch seg {
	case "primary": // 小学
		baseMin, baseMax = 15, 25
	case "senior": // 高中
		baseMin, baseMax = 22, 35
	default: // 初中及未知，取初中区间
		baseMin, baseMax = 20, 30
	}

	// 再按字数动态托底：经验值约每 350~400 字至少展开 1 页
	// （教案含目标/重难点/流程/活动/作业等，信息密度高）
	byWord := wordCount / 380
	if byWord > baseMin {
		// 字数要求更高时抬高下限，但不超过该学段上限太多
		if byWord > baseMax {
			baseMin = baseMax
		} else {
			baseMin = byWord
		}
	}
	if baseMin < 8 {
		baseMin = 8
	}

	desc := fmt.Sprintf("%d-%d页", baseMin, baseMax)
	return baseMin, desc
}

// cwGradeSegment 将中文年级名粗略归一化为学段（primary/junior/senior）
// 仅用于页数区间判断，无需精确
func cwGradeSegment(grade string) string {
	g := strings.TrimSpace(grade)
	// 小学关键词
	for _, k := range []string{"小学", "一年级", "二年级", "三年级", "四年级", "五年级", "六年级", "低段", "高段", "中段"} {
		if strings.Contains(g, k) {
			return "primary"
		}
	}
	// 高中关键词
	for _, k := range []string{"高中", "高一", "高二", "高三"} {
		if strings.Contains(g, k) {
			return "senior"
		}
	}
	// 初中关键词
	for _, k := range []string{"初中", "初一", "初二", "初三", "七年级", "八年级", "九年级"} {
		if strings.Contains(g, k) {
			return "junior"
		}
	}
	return "junior"
}

// buildDocIndexPrompt 构建Word文档→课件方案的提示词
// v0.43: 补齐页数下限/区间约束 + 结构骨架 + 禁止概括成数页的硬约束
func (s *CoursewarePPTService) buildDocIndexPrompt(cw *models.Courseware, doc *DocExtractResult, preset string, customHint string) string {
	var sb strings.Builder
	sb.WriteString("你是K12课件规划专家。\n")
	sb.WriteString("用户提供了一份教案文档的完整内容，\n")
	sb.WriteString("请将其转化为结构化的交互式课件方案（逐页规划）。\n\n")

	sb.WriteString("## 基本信息\n")
	sb.WriteString(fmt.Sprintf("- 学科: %s\n", cw.Subject))
	sb.WriteString(fmt.Sprintf("- 年级: %s\n", cw.Grade))
	sb.WriteString(fmt.Sprintf("- 课件标题: %s\n", cw.Title))
	sb.WriteString(fmt.Sprintf("- 教案字数: %d\n\n", doc.WordCount))

	sb.WriteString("## 教案完整内容\n\n")
	// 截断过长的文档（避免超出AI上下文限制）
	fullText := doc.FullText
	if len([]rune(fullText)) > 30000 {
		fullText = string([]rune(fullText)[:30000]) + "\n\n[文档内容过长，已截取前30000字]"
	}
	sb.WriteString(fullText)

	// 注入预设
	if hint := models.ResolveSchemePromptHint(preset, customHint); hint != "" {
		sb.WriteString("\n")
		sb.WriteString(hint)
		sb.WriteString("\n")
	}

	// ---- 关键修复：明确页数下限/区间，防止长教案被概括成1页 ----
	minPages, rangeDesc := cwRecommendPageRange(cw.Grade, doc.WordCount)
	sb.WriteString("\n\n## 篇幅与页数要求（必须遵守）\n")
	sb.WriteString(fmt.Sprintf("- 本教案约 %d 字，信息量充足，必须充分展开。\n", doc.WordCount))
	sb.WriteString(fmt.Sprintf("- 目标页数区间：%s，**不得少于 %d 页**。\n", rangeDesc, minPages))
	sb.WriteString("- 严禁把整份教案概括成一页或极少数几页；教案中的每个教学环节、每个知识点、每个活动都应有独立或拆分的课件页面承载。\n")
	sb.WriteString("- 若内容确实丰富，可超出区间上限，但绝不可低于下限。\n")

	sb.WriteString("\n## 转化原则\n")
	sb.WriteString("1. 保留教案的知识结构和教学设计，逐环节落到课件页面\n")
	sb.WriteString("2. 将文字描述转化为视觉呈现方案（图文、图表、动画等）\n")
	sb.WriteString("3. 在关键知识点处添加交互环节（选择题、拖拽、填空等）\n")
	sb.WriteString("4. 信息密度高的教学环节应拆分为多页，而非压缩为一页\n")
	sb.WriteString("5. 结构骨架：封面(1页) → 学习目标(1页) → 知识讲授(主体，按知识点逐页) → 练习/活动(2-3页) → 课堂小结(1页) → 作业布置(1页)\n")
	sb.WriteString("6. 交互类型分布：纯展示≤40%，简单交互30-40%，复杂交互≤20%\n")
	sb.WriteString("7. 难度递进：前1/3基础 → 中1/3进阶 → 后1/3综合\n\n")

	sb.WriteString("## 输出要求\n")
	sb.WriteString("请输出JSON数组格式。每个元素包含以下字段：\n")
	sb.WriteString("page_number(int), title(string), purpose(string), content_summary(string), ")
	sb.WriteString("interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n\n")
	sb.WriteString("page_number 从1开始连续编号。\n")
	sb.WriteString("交互类型可选：static/click/drag/input/animation/video/game/quiz\n")
	sb.WriteString("视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n\n")
	sb.WriteString("请只输出JSON数组，不要有任何额外说明文字。")

	return sb.String()
}
