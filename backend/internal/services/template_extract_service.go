package services

// template_extract_service.go — 课件风格模板AI提取服务。
//
// 当前版本的核心原则：
//
//   1. 老师提交的HTML是模板母版的唯一事实源。
//   2. AI只负责识别配色、字体、圆角、阴影、间距、风格类别和描述。
//   3. AI返回的sample_pages不再写入数据库，防止AI重画导致DOM、CSS、脚本和交互失真。
//   4. 原始母版最多保留20页、总计60万字符。
//   5. 页面较多时只抽取最多8个代表页供AI分析，原始20页仍全部完整入库。
//   6. 分析副本会省略base64图片和脚本正文，但仅影响AI输入，不影响正式母版。
//
// 同步入口和异步SSE入口共用完全相同的母版保真规则。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// 模块级日志器。
var extractLog = logger.WithModule("template_extract")

// 模板提取专用SSE事件。
const (
	CWSSEExtractStart    = "extract_start"
	CWSSEExtractProgress = "extract_progress"
	CWSSEExtractDone     = "extract_done"
	CWSSEExtractError    = "extract_error"
)

// templateExtractionPreserveContract 追加到数据库系统提示词之后。
//
// 数据库里的历史提示词可能仍要求AI生成sample_pages。
// 本补充契约明确改为只分析风格，避免AI为多页母版输出大量重写HTML。
// 即使模型仍返回sample_pages，服务层也会确定性忽略该字段。
const templateExtractionPreserveContract = `
【TE-DNA源页面保真补充契约·最高优先级】

1. 用户提供的每一页HTML都是不可重写的原始模板母版。
2. 你的任务只是分析视觉规律，不得重新设计、改写或压缩原始页面。
3. 请正常返回suggested_name、suggested_description、suggested_category、
   color_scheme、css_variables和extraction_notes。
4. sample_pages字段必须返回空数组[]。平台会确定性保存用户提交的完整原始页面。
5. 不要在输出中复述任何完整HTML，以免造成输出截断和页面损失。
`

// TemplateExtractService 课件风格模板AI提取服务。
type TemplateExtractService struct {
	cfg *config.Config
}

// NewTemplateExtractService 创建模板提取服务。
func NewTemplateExtractService(cfg *config.Config) *TemplateExtractService {
	return &TemplateExtractService{cfg: cfg}
}

// ExtractFromHTML 同步提取入口。
//
// 该入口仍保留供内部兼容调用使用；正式前端当前主要使用异步SSE入口。
func (s *TemplateExtractService) ExtractFromHTML(
	ctx context.Context,
	userID string,
	samplePages []string,
	sourceType string,
) (*models.ExtractTemplateResponse, error) {
	// 1. 准备并校验原始母版。
	cleanedPages, totalHTMLLen, err := prepareTemplateSourcePages(samplePages)
	if err != nil {
		return nil, err
	}
	if sourceType == "" {
		sourceType = "paste"
	}

	// 2. 只为AI构造代表页分析副本。
	analysisPages := selectTemplateAnalysisPages(
		cleanedPages,
		templateAnalysisMaxPages,
		templateAnalysisMaxRunesPerPage,
	)
	userPrompt := s.buildExtractUserPrompt(
		analysisPages,
		len(cleanedPages),
		totalHTMLLen,
	)

	// 3. 加载并补充系统提示词。
	sysPromptObj, err := repository.GetCurrentPromptByKey(
		"prompt_courseware_template_extract",
	)
	if err != nil {
		return nil, fmt.Errorf("加载AI提取提示词失败: %w", err)
	}
	systemPrompt := strings.TrimSpace(sysPromptObj.Content) +
		"\n\n" +
		strings.TrimSpace(templateExtractionPreserveContract)

	// 4. 获取AI配置。
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_extract",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 5. 调用AI。
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_extract",
		UserID:    &userID,
	}
	callStart := time.Now()
	result, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return nil, fmt.Errorf("AI调用失败: %w", err)
	}
	callElapsed := time.Since(callStart)

	extractLog.Info(
		"AI调用完成",
		"user", userID,
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
		"elapsed", callElapsed.String(),
		"original_pages", len(cleanedPages),
		"analysis_pages", len(analysisPages),
	)

	// 6. 解析AI风格分析结果。
	extracted, err := s.parseAIOutput(result.Content)
	if err != nil {
		extractLog.Error(
			"JSON解析失败",
			"raw_output_head", truncateForLog(result.Content, 500),
		)
		return nil, fmt.Errorf("AI输出解析失败: %w", err)
	}
	if extracted.Error != "" {
		return nil, fmt.Errorf("AI无法从输入提取风格: %s", extracted.Error)
	}
	if err := s.validateExtracted(extracted); err != nil {
		return nil, fmt.Errorf("AI提取结果不完整: %w", err)
	}

	// 7. 按保真原则构造并保存草稿。
	tpl, err := s.persistExtractedTemplate(
		ctx,
		userID,
		sourceType,
		cleanedPages,
		totalHTMLLen,
		analysisPages,
		extracted,
		result.ModelUsed,
		result.TokensUsed,
	)
	if err != nil {
		return nil, err
	}

	return &models.ExtractTemplateResponse{
		TemplateID:      tpl.ID,
		SuggestedName:   tpl.Name,
		SuggestedDesc:   tpl.Description,
		SuggestedCat:    tpl.StyleCategory,
		ExtractionNotes: extracted.ExtractionNotes,
		Message:         "AI已完成风格分析，原始模板页面已完整保留",
	}, nil
}

// buildExtractUserPrompt 构建只用于风格分析的AI输入。
//
// originalPageCount是实际完整入库页数；analysisPages只是送给AI观察的代表页。
func (s *TemplateExtractService) buildExtractUserPrompt(
	analysisPages []templateAnalysisPage,
	originalPageCount int,
	totalHTMLLen int,
) string {
	var sb strings.Builder

	sb.WriteString("# 模板风格分析任务\n\n")
	sb.WriteString(fmt.Sprintf(
		"用户共提交%d页原始HTML，总计%d字符。\n",
		originalPageCount,
		totalHTMLLen,
	))
	sb.WriteString(fmt.Sprintf(
		"平台已完整保存全部原始页面。本次只提供其中%d个代表页供你分析风格。\n",
		len(analysisPages),
	))
	sb.WriteString("不要重写页面，不要复述HTML，不要生成新的页面结构。\n\n")

	for _, page := range analysisPages {
		sb.WriteString(fmt.Sprintf(
			"=== 原模板第%d页代表样例（推断页型：%s）开始 ===\n",
			page.SourcePageNumber,
			page.RoleLabel,
		))
		sb.WriteString(page.HTML)
		sb.WriteString(fmt.Sprintf(
			"\n=== 原模板第%d页代表样例结束 ===\n\n",
			page.SourcePageNumber,
		))
	}

	sb.WriteString("# 输出要求\n\n")
	sb.WriteString("请只提取统一视觉规律，包括配色、字体、圆角、阴影、间距、装饰语言和风格类别。\n")
	sb.WriteString("sample_pages必须返回空数组[]，原始母版页由平台直接保存。\n")
	sb.WriteString("严格按照系统提示词约定的JSON格式输出，不要代码围栏或解释文字。")

	return sb.String()
}

// persistExtractedTemplate 将AI分析结果与完整原始母版组装后写入数据库。
func (s *TemplateExtractService) persistExtractedTemplate(
	ctx context.Context,
	userID string,
	sourceType string,
	cleanedPages []string,
	totalHTMLLen int,
	analysisPages []templateAnalysisPage,
	extracted *models.AIExtractedTemplate,
	modelUsed string,
	tokensUsed int,
) (*models.CoursewareTemplate, error) {
	colorSchemeJSON, err := json.Marshal(extracted.ColorScheme)
	if err != nil {
		return nil, fmt.Errorf("序列化配色方案失败: %w", err)
	}

	cssVarsJSON, err := json.Marshal(extracted.CSSVariables)
	if err != nil {
		return nil, fmt.Errorf("序列化CSS变量失败: %w", err)
	}

	// 关键改变：正式sample_pages永远来自老师提交的cleanedPages，
	// 不使用AI输出的extracted.SamplePages。
	samplePagesJSON, err := json.Marshal(cleanedPages)
	if err != nil {
		return nil, fmt.Errorf("序列化原始母版页面失败: %w", err)
	}

	category := extracted.SuggestedCategory
	if !models.IsValidCWStyleCategory(category) {
		extractLog.Warn(
			"AI返回的风格类别不合法，兜底为minimalist",
			"category", category,
		)
		category = models.CWStyleMinimalist
	}

	name := strings.TrimSpace(extracted.SuggestedName)
	if name == "" {
		name = fmt.Sprintf(
			"AI提取草稿 %s",
			time.Now().Format("01-02 15:04"),
		)
	}

	tpl := &models.CoursewareTemplate{
		Name:          name,
		Description:   strings.TrimSpace(extracted.SuggestedDescription),
		StyleCategory: category,
		ColorScheme:   string(colorSchemeJSON),
		CSSVariables:  string(cssVarsJSON),
		SamplePages:   string(samplePagesJSON),
		UserID:        &userID,
	}

	sourceMeta := map[string]interface{}{
		"source_type":                  sourceType,
		"original_html_length":         totalHTMLLen,
		"sample_pages_count":           len(cleanedPages),
		"analysis_pages_count":         len(analysisPages),
		"page_roles":                   buildTemplatePageRoleMeta(cleanedPages),
		"extracted_at":                 time.Now().Format(time.RFC3339),
		"ai_model_used":                modelUsed,
		"ai_tokens_used":               tokensUsed,
		"ai_extraction_notes":          extracted.ExtractionNotes,
		"ai_output_sample_pages_count": len(extracted.SamplePages),
		"source_pages_preserved":       true,
	}

	if err := repository.CreateDraftTemplate(ctx, tpl, sourceMeta); err != nil {
		return nil, fmt.Errorf("草稿入库失败: %w", err)
	}

	extractLog.Info(
		"原始母版保真草稿创建成功",
		"id", tpl.ID,
		"name", tpl.Name,
		"category", tpl.StyleCategory,
		"pages", len(cleanedPages),
		"total_html_len", totalHTMLLen,
		"user", userID,
	)

	return tpl, nil
}

// parseAIOutput 解析AI返回的JSON，执行多重容错。
func (s *TemplateExtractService) parseAIOutput(
	aiOutput string,
) (*models.AIExtractedTemplate, error) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	var jsonStr string
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		jsonStr = text
	} else if extracted, ok := ai.ExtractJSON(text); ok && extracted != "" {
		jsonStr = extracted
	} else {
		return nil, fmt.Errorf("未能从AI输出中找到JSON对象")
	}

	jsonStr = sanitizeNonASCIIForJSON(jsonStr)

	var extracted models.AIExtractedTemplate
	if err := json.Unmarshal([]byte(jsonStr), &extracted); err == nil {
		return &extracted, nil
	}

	cleaned := cwCleanChinesePunctuation(jsonStr)
	if err := json.Unmarshal([]byte(cleaned), &extracted); err == nil {
		extractLog.Info("JSON经中文标点清理后解析成功")
		return &extracted, nil
	}

	fixed := cwFixJSONQuotes(cleaned)
	if err := json.Unmarshal([]byte(fixed), &extracted); err == nil {
		extractLog.Info("JSON经引号修复后解析成功")
		return &extracted, nil
	}

	extracted = models.AIExtractedTemplate{
		SuggestedName:        extractStringField(cleaned, "suggested_name"),
		SuggestedDescription: extractStringField(cleaned, "suggested_description"),
		SuggestedCategory:    extractStringField(cleaned, "suggested_category"),
		ExtractionNotes:      extractStringField(cleaned, "extraction_notes"),
		Error:                extractStringField(cleaned, "error"),
	}

	if value := extractObjectField(cleaned, "color_scheme"); value != "" {
		_ = json.Unmarshal([]byte(value), &extracted.ColorScheme)
	}
	if value := extractObjectField(cleaned, "css_variables"); value != "" {
		_ = json.Unmarshal([]byte(value), &extracted.CSSVariables)
	}
	if value := extractArrayField(cleaned, "sample_pages"); value != "" {
		_ = json.Unmarshal([]byte(value), &extracted.SamplePages)
	}

	if extracted.SuggestedName == "" && len(extracted.ColorScheme) == 0 {
		return nil, fmt.Errorf("多重兜底解析全部失败，AI输出可能严重畸形")
	}

	extractLog.Warn("JSON使用字段级宽容提取成功")
	return &extracted, nil
}

// validateExtracted 只校验风格数据。
//
// sample_pages不再由AI负责生成，因此不再要求AI输出非空页面数组。
func (s *TemplateExtractService) validateExtracted(
	extracted *models.AIExtractedTemplate,
) error {
	requiredColorKeys := []string{
		"primary",
		"secondary",
		"background",
		"accent",
		"text",
	}
	for _, key := range requiredColorKeys {
		if strings.TrimSpace(extracted.ColorScheme[key]) == "" {
			return fmt.Errorf("color_scheme.%s缺失", key)
		}
	}

	requiredCSSKeys := []string{
		"--cw-primary",
		"--cw-secondary",
		"--cw-bg",
		"--cw-accent",
		"--cw-text",
		"--cw-font-heading",
		"--cw-font-body",
		"--cw-radius",
		"--cw-shadow",
	}
	for _, key := range requiredCSSKeys {
		if strings.TrimSpace(extracted.CSSVariables[key]) == "" {
			return fmt.Errorf("css_variables.%s缺失", key)
		}
	}

	return nil
}

// extractStringField 从可能畸形的JSON中宽容提取字符串字段。
func extractStringField(jsonStr string, key string) string {
	keyPattern := fmt.Sprintf("\"%s\"", key)
	idx := strings.Index(jsonStr, keyPattern)
	if idx < 0 {
		return ""
	}

	rest := jsonStr[idx+len(keyPattern):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return ""
	}

	rest = strings.TrimSpace(rest[colonIdx+1:])
	if !strings.HasPrefix(rest, "\"") {
		return ""
	}
	rest = rest[1:]

	var sb strings.Builder
	for i := 0; i < len(rest); i++ {
		if rest[i] == '\\' && i+1 < len(rest) {
			sb.WriteByte(rest[i])
			sb.WriteByte(rest[i+1])
			i++
			continue
		}
		if rest[i] == '"' {
			return sb.String()
		}
		sb.WriteByte(rest[i])
	}

	return ""
}

// extractObjectField 从可能畸形的JSON中提取对象字段。
func extractObjectField(jsonStr string, key string) string {
	keyPattern := fmt.Sprintf("\"%s\"", key)
	idx := strings.Index(jsonStr, keyPattern)
	if idx < 0 {
		return ""
	}

	rest := jsonStr[idx+len(keyPattern):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return ""
	}

	rest = strings.TrimSpace(rest[colonIdx+1:])
	if !strings.HasPrefix(rest, "{") {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(rest); i++ {
		char := rest[i]

		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch char {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}

	return ""
}

// extractArrayField 从可能畸形的JSON中提取数组字段。
func extractArrayField(jsonStr string, key string) string {
	keyPattern := fmt.Sprintf("\"%s\"", key)
	idx := strings.Index(jsonStr, keyPattern)
	if idx < 0 {
		return ""
	}

	rest := jsonStr[idx+len(keyPattern):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return ""
	}

	rest = strings.TrimSpace(rest[colonIdx+1:])
	if !strings.HasPrefix(rest, "[") {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(rest); i++ {
		char := rest[i]

		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch char {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}

	return ""
}

// sanitizeNonASCIIForJSON 转义JSON字符串值中的非ASCII非CJK字符。
func sanitizeNonASCIIForJSON(source string) string {
	runes := []rune(source)

	var sb strings.Builder
	sb.Grow(len(source) + len(source)/10)

	inString := false
	escaped := false

	for _, char := range runes {
		if escaped {
			sb.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' && inString {
			sb.WriteRune(char)
			escaped = true
			continue
		}
		if char == '"' {
			inString = !inString
			sb.WriteRune(char)
			continue
		}

		if inString && char > 0x7F && !isCJKRune(char) {
			if char <= 0xFFFF {
				sb.WriteString(fmt.Sprintf("\\u%04X", char))
			} else {
				char -= 0x10000
				high := 0xD800 + (char>>10)&0x3FF
				low := 0xDC00 + char&0x3FF
				sb.WriteString(fmt.Sprintf("\\u%04X\\u%04X", high, low))
			}
			continue
		}

		sb.WriteRune(char)
	}

	return sb.String()
}

// isCJKRune 判断是否为中日韩文字或常见全角符号。
func isCJKRune(char rune) bool {
	return (char >= 0x3000 && char <= 0x9FFF) ||
		(char >= 0xF900 && char <= 0xFAFF) ||
		(char >= 0xFE30 && char <= 0xFE4F) ||
		(char >= 0xFF00 && char <= 0xFFEF) ||
		(char >= 0x20000 && char <= 0x2FA1F)
}

// truncateForLog 截断日志文本。
func truncateForLog(source string, maxLen int) string {
	if len(source) <= maxLen {
		return source
	}
	return source[:maxLen] + "...(truncated)"
}

// ExtractFromHTMLAsync 异步SSE模板提取入口。
func (s *TemplateExtractService) ExtractFromHTMLAsync(
	ctx context.Context,
	userID string,
	samplePages []string,
	sourceType string,
) {
	sseKey := "extract_" + userID

	broadcast := func(message string) {
		GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
			EventType: CWSSEExtractProgress,
			Data: map[string]interface{}{
				"message": message,
			},
		})
	}

	broadcastErr := func(message string) {
		extractLog.Error(
			"模板提取失败",
			"key", sseKey,
			"error", message,
		)
		GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
			EventType: CWSSEExtractError,
			Data: map[string]interface{}{
				"message": message,
			},
		})
	}

	GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
		EventType: CWSSEExtractStart,
		Data: map[string]interface{}{
			"message": "🔍 正在校验并保留原始模板页面...",
		},
	})

	// 1. 准备完整原始母版。
	cleanedPages, totalHTMLLen, err := prepareTemplateSourcePages(samplePages)
	if err != nil {
		broadcastErr(err.Error())
		return
	}
	if sourceType == "" {
		sourceType = "paste"
	}

	// 2. 生成代表页分析副本。
	analysisPages := selectTemplateAnalysisPages(
		cleanedPages,
		templateAnalysisMaxPages,
		templateAnalysisMaxRunesPerPage,
	)

	broadcast(fmt.Sprintf(
		"📚 已完整保留%d页原始母版，正在选择%d个代表页分析...",
		len(cleanedPages),
		len(analysisPages),
	))

	userPrompt := s.buildExtractUserPrompt(
		analysisPages,
		len(cleanedPages),
		totalHTMLLen,
	)

	sysPromptObj, err := repository.GetCurrentPromptByKey(
		"prompt_courseware_template_extract",
	)
	if err != nil {
		broadcastErr("加载AI提取提示词失败: " + err.Error())
		return
	}
	systemPrompt := strings.TrimSpace(sysPromptObj.Content) +
		"\n\n" +
		strings.TrimSpace(templateExtractionPreserveContract)

	// 3. 获取AI配置。
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_extract",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		broadcastErr("获取AI配置失败: " + err.Error())
		return
	}

	// 4. AI只分析风格。
	broadcast("🤖 AI正在分析代表页的配色、字体、间距和视觉规律...")

	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_extract",
		UserID:    &userID,
	}
	callStart := time.Now()
	result, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		broadcastErr("AI调用失败: " + err.Error())
		return
	}
	callElapsed := time.Since(callStart)

	extractLog.Info(
		"AI风格分析完成",
		"user", userID,
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
		"elapsed", callElapsed.String(),
		"original_pages", len(cleanedPages),
		"analysis_pages", len(analysisPages),
	)

	// 5. 解析风格数据。
	broadcast(fmt.Sprintf(
		"🔧 AI分析完成（耗时%d秒），正在解析风格参数...",
		int(callElapsed.Seconds()),
	))

	extracted, err := s.parseAIOutput(result.Content)
	if err != nil {
		extractLog.Error(
			"异步JSON解析失败",
			"raw_output_head", truncateForLog(result.Content, 500),
		)
		broadcastErr("AI输出解析失败: " + err.Error())
		return
	}
	if extracted.Error != "" {
		broadcastErr("AI无法从输入提取风格: " + extracted.Error)
		return
	}
	if err := s.validateExtracted(extracted); err != nil {
		broadcastErr("AI提取结果不完整: " + err.Error())
		return
	}

	// 6. 保存AI风格参数和完整原始母版。
	broadcast(fmt.Sprintf(
		"💾 正在保存风格参数和全部%d页原始母版...",
		len(cleanedPages),
	))

	tpl, err := s.persistExtractedTemplate(
		ctx,
		userID,
		sourceType,
		cleanedPages,
		totalHTMLLen,
		analysisPages,
		extracted,
		result.ModelUsed,
		result.TokensUsed,
	)
	if err != nil {
		broadcastErr(err.Error())
		return
	}

	GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
		EventType: CWSSEExtractDone,
		Data: map[string]interface{}{
			"template_id":        tpl.ID,
			"suggested_name":     tpl.Name,
			"suggested_desc":     tpl.Description,
			"suggested_category": tpl.StyleCategory,
			"extraction_notes":   extracted.ExtractionNotes,
			"page_count":         len(cleanedPages),
			"source_preserved":   true,
			"message":            "✨ 风格分析完成，原始模板页面已完整保留",
		},
	})
}
