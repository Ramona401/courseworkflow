package services

// template_extract_service.go — 课件风格模板 AI 提取服务(v139.1-fix)
//
// 功能:
//   用户粘贴一页或多页 HTML 代码,AI 分析后提取一套可复用的视觉风格模板,
//   入库为草稿(is_draft=true, scope=personal)。
//
// v139.1-fix 修改:
//   - log.Printf 全部替换为 logger.WithModule 结构化日志
//   - 新增 sanitizeNonASCIIForJSON 在 JSON 解析前预处理非 ASCII 字符(防 é/′/¨ 等导致解析失败)

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

// 模块级日志器
var extractLog = logger.WithModule("template_extract")

// ==================== SSE 事件类型常量(模板提取专用, v145) ====================

const (
	CWSSEExtractStart    = "extract_start"    // 提取任务开始
	CWSSEExtractProgress = "extract_progress" // 提取阶段进度
	CWSSEExtractDone     = "extract_done"     // 提取完成,含草稿数据
	CWSSEExtractError    = "extract_error"    // 提取失败
)

// HTML 总长度上限(包级常量,供同步/异步方法共用)
const maxTotalHTMLLen = 200000

// ==================== 服务结构体 ====================

// TemplateExtractService 课件风格模板 AI 提取服务
type TemplateExtractService struct {
	cfg *config.Config
}

// NewTemplateExtractService 构造函数
func NewTemplateExtractService(cfg *config.Config) *TemplateExtractService {
	return &TemplateExtractService{cfg: cfg}
}

// ==================== 入口方法:ExtractFromHTML ====================

// ExtractFromHTML 从用户粘贴的 HTML 代码中提取风格模板
func (s *TemplateExtractService) ExtractFromHTML(
	ctx context.Context,
	userID string,
	samplePages []string,
	sourceType string,
) (*models.ExtractTemplateResponse, error) {

	// -------- 1. 输入校验 --------
	if len(samplePages) == 0 {
		return nil, fmt.Errorf("请至少提供一页 HTML 代码")
	}
	cleanedPages := s.cleanSamplePages(samplePages)
	if len(cleanedPages) == 0 {
		return nil, fmt.Errorf("提供的 HTML 内容为空,请粘贴有效的 HTML 代码")
	}
	totalHTMLLen := 0
	for _, p := range cleanedPages {
		totalHTMLLen += len(p)
	}
	if totalHTMLLen > maxTotalHTMLLen {
		return nil, fmt.Errorf("HTML 总长度 %d 字符超出上限 %d 字符,请精简后再试", totalHTMLLen, maxTotalHTMLLen)
	}

	if sourceType == "" {
		sourceType = "paste"
	}

	// -------- 2. 构建 AI 用户提示词 --------
	userPrompt := s.buildExtractUserPrompt(cleanedPages)

	// -------- 3. 加载系统提示词 --------
	sysPromptObj, err := repository.GetCurrentPromptByKey("prompt_courseware_template_extract")
	if err != nil {
		return nil, fmt.Errorf("加载 AI 提取提示词失败: %w", err)
	}

	// -------- 4. 获取 AI 配置(courseware_template_extract 场景) --------
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_extract",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("获取 AI 配置失败: %w", err)
	}

	// -------- 5. 调用 AI(非流式,等待完整 JSON) --------
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_extract",
		UserID:    &userID,
	}
	callStart := time.Now()
	result, err := ai.CallAI(aiCfg, sysPromptObj.Content, userPrompt, traceCtx)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}
	callElapsed := time.Since(callStart)
	extractLog.Info("AI 调用完成", "user", userID, "model", result.ModelUsed, "tokens", result.TokensUsed, "elapsed", callElapsed.String())

	// -------- 6. 解析 AI 输出 JSON(四重兜底) --------
	extracted, err := s.parseAIOutput(result.Content)
	if err != nil {
		extractLog.Error("JSON 解析失败", "raw_output_head", truncateForLog(result.Content, 500))
		return nil, fmt.Errorf("AI 输出解析失败: %w", err)
	}

	// AI 自己表示无法分析
	if extracted.Error != "" {
		return nil, fmt.Errorf("AI 无法从输入提取风格: %s", extracted.Error)
	}

	// -------- 7. 校验关键字段 --------
	if err := s.validateExtracted(extracted); err != nil {
		extractLog.Warn("校验失败", "error", err, "name", extracted.SuggestedName, "category", extracted.SuggestedCategory, "css_vars_count", len(extracted.CSSVariables), "sample_pages_count", len(extracted.SamplePages))
		return nil, fmt.Errorf("AI 提取结果不完整: %w", err)
	}

	// -------- 8. 序列化为 JSON 字符串,准备入库 --------
	colorSchemeJSON, _ := json.Marshal(extracted.ColorScheme)
	cssVarsJSON, _ := json.Marshal(extracted.CSSVariables)
	samplePagesJSON, _ := json.Marshal(extracted.SamplePages)

	// 风格类别兜底:AI 给的值不合法时,设为 minimalist
	category := extracted.SuggestedCategory
	if !models.IsValidCWStyleCategory(category) {
		extractLog.Warn("AI 返回的 category 不合法,兜底为 minimalist", "category", category)
		category = models.CWStyleMinimalist
	}

	// 名称兜底:AI 没给名称时用时间戳
	name := strings.TrimSpace(extracted.SuggestedName)
	if name == "" {
		name = fmt.Sprintf("AI 提取草稿 %s", time.Now().Format("01-02 15:04"))
	}

	// -------- 9. 构造草稿模板对象 --------
	tpl := &models.CoursewareTemplate{
		Name:          name,
		Description:   strings.TrimSpace(extracted.SuggestedDescription),
		StyleCategory: category,
		ColorScheme:   string(colorSchemeJSON),
		CSSVariables:  string(cssVarsJSON),
		SamplePages:   string(samplePagesJSON),
		UserID:        &userID,
	}

	// 来源元信息
	sourceMeta := map[string]interface{}{
		"source_type":          sourceType,
		"original_html_length": totalHTMLLen,
		"sample_pages_count":   len(cleanedPages),
		"extracted_at":         time.Now().Format(time.RFC3339),
		"ai_model_used":        result.ModelUsed,
		"ai_tokens_used":       result.TokensUsed,
		"ai_extraction_notes":  extracted.ExtractionNotes,
	}

	// -------- 10. 入库为草稿 --------
	if err := repository.CreateDraftTemplate(ctx, tpl, sourceMeta); err != nil {
		return nil, fmt.Errorf("草稿入库失败: %w", err)
	}

	extractLog.Info("草稿创建成功", "id", tpl.ID, "name", tpl.Name, "category", tpl.StyleCategory, "pages", len(cleanedPages), "user", userID)

	// -------- 11. 构造响应 --------
	return &models.ExtractTemplateResponse{
		TemplateID:      tpl.ID,
		SuggestedName:   tpl.Name,
		SuggestedDesc:   tpl.Description,
		SuggestedCat:    tpl.StyleCategory,
		ExtractionNotes: extracted.ExtractionNotes,
		Message:         "AI 已提取风格模板,您可以继续微调或保存",
	}, nil
}

// ==================== 输入清理 ====================

// cleanSamplePages 清理用户输入的样例页面
func (s *TemplateExtractService) cleanSamplePages(pages []string) []string {
	var result []string
	for _, p := range pages {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ==================== AI 提示词构建 ====================

// buildExtractUserPrompt 构建 AI 用户提示词
func (s *TemplateExtractService) buildExtractUserPrompt(samplePages []string) string {
	var sb strings.Builder
	sb.WriteString("请从以下 HTML 代码中提取统一的视觉风格模板。\n\n")
	sb.WriteString(fmt.Sprintf("共 %d 页 HTML,逐页分析后给出统一风格:\n\n", len(samplePages)))

	for i, page := range samplePages {
		sb.WriteString(fmt.Sprintf("=== 第 %d 页 HTML 开始 ===\n", i+1))
		sb.WriteString(page)
		sb.WriteString("\n=== 第 ")
		sb.WriteString(fmt.Sprintf("%d 页 HTML 结束 ===\n\n", i+1))
	}

	sb.WriteString("请按照系统提示词约定的 JSON 格式输出,只输出 JSON 对象,不要任何说明文字或代码块标记。")
	return sb.String()
}

// ==================== AI 输出解析(四重兜底) ====================

// parseAIOutput 解析 AI 返回的 JSON,四重兜底容错
//
// 在尝试 JSON 解析前,先对字符串做非 ASCII 预处理(sanitizeNonASCIIForJSON),
// 将 é/′/¨/™ 等拉丁特殊字符转义为 \uXXXX 格式,防止 json.Unmarshal 失败
func (s *TemplateExtractService) parseAIOutput(aiOutput string) (*models.AIExtractedTemplate, error) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	// AI 输出含大量嵌套 HTML 时,ai.ExtractJSON 的括号配对会被 HTML 中的 { } 误导
	// 直接判断:若开头是 { 结尾是 },就当作纯 JSON 用,不再"提取"
	var jsonStr string
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		jsonStr = text
	} else if extracted, ok := ai.ExtractJSON(text); ok && extracted != "" {
		jsonStr = extracted
	} else {
		return nil, fmt.Errorf("未能从 AI 输出中找到 JSON 对象")
	}

	// 非 ASCII 预处理:将 JSON 字符串值中的非 ASCII 拉丁字符转义为 \uXXXX
	// 这是对付 AI 不听话输出 é/′/¨ 等特殊字符的根治方案
	jsonStr = sanitizeNonASCIIForJSON(jsonStr)

	extractLog.Info("JSON 字符串长度(预处理后)", "length", len(jsonStr))

	// 尝试 1: 直接 Unmarshal
	var extracted models.AIExtractedTemplate
	if err1 := json.Unmarshal([]byte(jsonStr), &extracted); err1 == nil {
		return &extracted, nil
	}

	// 尝试 2: 清理中文标点(仅对外层结构,不动 HTML 字符串值)
	cleaned := cwCleanChinesePunctuation(jsonStr)
	if err2 := json.Unmarshal([]byte(cleaned), &extracted); err2 == nil {
		extractLog.Info("JSON 解析第2重(中文标点清理后)成功")
		return &extracted, nil
	}

	// 尝试 3: 修复未转义引号
	fixed := cwFixJSONQuotes(cleaned)
	if err3 := json.Unmarshal([]byte(fixed), &extracted); err3 == nil {
		extractLog.Info("JSON 解析第3重(引号修复后)成功")
		return &extracted, nil
	}

	// 尝试 4: 字段级宽容提取(只取核心字段)
	extracted = models.AIExtractedTemplate{}
	extracted.SuggestedName = extractStringField(cleaned, "suggested_name")
	extracted.SuggestedDescription = extractStringField(cleaned, "suggested_description")
	extracted.SuggestedCategory = extractStringField(cleaned, "suggested_category")
	extracted.ExtractionNotes = extractStringField(cleaned, "extraction_notes")

	if cs := extractObjectField(cleaned, "color_scheme"); cs != "" {
		_ = json.Unmarshal([]byte(cs), &extracted.ColorScheme)
	}
	if cv := extractObjectField(cleaned, "css_variables"); cv != "" {
		_ = json.Unmarshal([]byte(cv), &extracted.CSSVariables)
	}
	if sp := extractArrayField(cleaned, "sample_pages"); sp != "" {
		_ = json.Unmarshal([]byte(sp), &extracted.SamplePages)
	}

	if extracted.SuggestedName == "" && len(extracted.ColorScheme) == 0 {
		return nil, fmt.Errorf("四重兜底解析全部失败,AI 输出可能严重畸形")
	}
	extractLog.Warn("JSON 解析:字段级宽容提取成功(部分字段可能缺失)")
	return &extracted, nil
}

// ==================== AI 输出校验 ====================

// validateExtracted 校验 AI 提取结果的关键字段完整性
func (s *TemplateExtractService) validateExtracted(e *models.AIExtractedTemplate) error {
	requiredColorKeys := []string{"primary", "secondary", "background", "accent", "text"}
	for _, k := range requiredColorKeys {
		if e.ColorScheme[k] == "" {
			return fmt.Errorf("color_scheme.%s 缺失", k)
		}
	}

	requiredCSSKeys := []string{
		"--cw-primary", "--cw-secondary", "--cw-bg", "--cw-accent", "--cw-text",
		"--cw-font-heading", "--cw-font-body", "--cw-radius", "--cw-shadow",
	}
	for _, k := range requiredCSSKeys {
		if e.CSSVariables[k] == "" {
			return fmt.Errorf("css_variables.%s 缺失", k)
		}
	}

	validPages := 0
	for _, p := range e.SamplePages {
		if strings.TrimSpace(p) != "" {
			validPages++
		}
	}
	if validPages == 0 {
		return fmt.Errorf("sample_pages 数组为空或全部为空字符串")
	}

	return nil
}

// ==================== 字段级宽容提取辅助函数 ====================

// extractStringField 从 JSON 字符串中宽容提取一个字符串字段
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

// extractObjectField 从 JSON 字符串中提取一个对象字段
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
	for i := 0; i < len(rest); i++ {
		if rest[i] == '{' {
			depth++
		} else if rest[i] == '}' {
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}
	return ""
}

// extractArrayField 从 JSON 字符串中提取一个数组字段
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
	escape := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}
	return ""
}

// ==================== 非 ASCII 预处理(v139.1-fix 新增) ====================

// sanitizeNonASCIIForJSON 对 JSON 字符串中的非 ASCII 拉丁字符做 \uXXXX 转义
//
// 问题背景:
//   AI 输出的 JSON 中偶尔包含 é(U+00E9)、′(U+2032)、¨(U+00A8)、™(U+2122) 等字符,
//   这些字符本身是合法 UTF-8,但在部分 JSON 解析场景下(特别是嵌套 HTML 的 JSON 字符串值中)
//   可能导致 json.Unmarshal 失败(如 "invalid character 'é' after array element")。
//
// 策略:
//   遍历 JSON 字符串,在 JSON 字符串值内部(引号之间),将非 ASCII 且非 CJK 的字符
//   转义为 \uXXXX 格式。CJK 字符(中日韩)保持原样(提示词要求"中文保留")。
//
// 范围:
//   - 转义: U+0080 ~ U+2FFF 中非 CJK 的拉丁/符号字符
//   - 保留: ASCII(U+0000~U+007F)、CJK(U+3000~U+9FFF, U+F900~U+FAFF)、
//           CJK扩展(U+20000+)、已有 \uXXXX 转义
func sanitizeNonASCIIForJSON(s string) string {
	runes := []rune(s)
	var sb strings.Builder
	sb.Grow(len(s) + len(s)/10) // 预分配略大空间

	inString := false
	escaped := false

	for _, r := range runes {
		if escaped {
			sb.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inString {
			sb.WriteRune(r)
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			sb.WriteRune(r)
			continue
		}

		// 只在 JSON 字符串值内部处理
		if inString && r > 0x7F && !isCJKRune(r) {
			// 非 ASCII 且非 CJK → 转义为 \uXXXX
			// 对于 BMP 之外的字符(>0xFFFF),用 surrogate pair
			if r <= 0xFFFF {
				sb.WriteString(fmt.Sprintf("\\u%04X", r))
			} else {
				// UTF-16 surrogate pair
				r -= 0x10000
				hi := 0xD800 + (r>>10)&0x3FF
				lo := 0xDC00 + r&0x3FF
				sb.WriteString(fmt.Sprintf("\\u%04X\\u%04X", hi, lo))
			}
			continue
		}

		sb.WriteRune(r)
	}
	return sb.String()
}

// isCJKRune 判断是否为 CJK 字符(中日韩统一表意文字及常见标点)
// 这些字符在 JSON 中合法且应保持原样输出
func isCJKRune(r rune) bool {
	return (r >= 0x3000 && r <= 0x9FFF) || // CJK 统一表意文字 + 符号标点
		(r >= 0xF900 && r <= 0xFAFF) || // CJK 兼容表意文字
		(r >= 0xFE30 && r <= 0xFE4F) || // CJK 兼容形式
		(r >= 0xFF00 && r <= 0xFFEF) || // 全角ASCII/半角片假名
		(r >= 0x20000 && r <= 0x2FA1F) // CJK 扩展B~F + 兼容补充
}

// ==================== 日志辅助 ====================

// truncateForLog 截断字符串供日志使用
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// ==================== 异步提取入口(v145 新增, SSE 推送) ====================

// ExtractFromHTMLAsync 异步版 AI 风格模板提取,通过 SSE 广播进度
//
// 此方法被 handler 异步调用(go func + 800ms 延迟等前端 SSE 连接建立)
// SSE Key 格式: "extract_" + userID (提取时还没有 templateID,用户维度天然唯一)
func (s *TemplateExtractService) ExtractFromHTMLAsync(
	ctx context.Context,
	userID string,
	samplePages []string,
	sourceType string,
) {
	sseKey := "extract_" + userID

	// 便捷广播函数
	broadcast := func(msg string) {
		GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
			EventType: CWSSEExtractProgress,
			Data:      map[string]interface{}{"message": msg},
		})
	}
	broadcastErr := func(msg string) {
		extractLog.Error("提取失败(异步)", "key", sseKey, "error", msg)
		GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
			EventType: CWSSEExtractError,
			Data:      map[string]interface{}{"message": msg},
		})
	}

	// -------- 广播开始 --------
	GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
		EventType: CWSSEExtractStart,
		Data:      map[string]interface{}{"message": "\U0001f50d 正在预处理 HTML 输入..."},
	})

	// -------- 1. 输入校验 --------
	cleanedPages := s.cleanSamplePages(samplePages)
	if len(cleanedPages) == 0 {
		broadcastErr("提供的 HTML 内容为空,请粘贴有效的 HTML 代码")
		return
	}
	totalHTMLLen := 0
	for _, p := range cleanedPages {
		totalHTMLLen += len(p)
	}
	if totalHTMLLen > maxTotalHTMLLen {
		broadcastErr(fmt.Sprintf("HTML 总长度 %d 字符超出上限 %d 字符", totalHTMLLen, maxTotalHTMLLen))
		return
	}
	if sourceType == "" {
		sourceType = "paste"
	}

	// -------- 2. 构建提示词 --------
	broadcast("\U0001f4dd 正在构建 AI 分析指令...")
	userPrompt := s.buildExtractUserPrompt(cleanedPages)
	sysPromptObj, err := repository.GetCurrentPromptByKey("prompt_courseware_template_extract")
	if err != nil {
		broadcastErr("加载 AI 提取提示词失败: " + err.Error())
		return
	}

	// -------- 3. 获取 AI 配置 --------
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_extract",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		broadcastErr("获取 AI 配置失败: " + err.Error())
		return
	}

	// -------- 4. 调用 AI(耗时最长的步骤) --------
	broadcast("\U0001f916 AI 深度分析中,正在识别配色、字体、布局风格...(约 3-8 分钟)")
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_extract",
		UserID:    &userID,
	}
	callStart := time.Now()
	result, err := ai.CallAI(aiCfg, sysPromptObj.Content, userPrompt, traceCtx)
	if err != nil {
		broadcastErr("AI 调用失败: " + err.Error())
		return
	}
	callElapsed := time.Since(callStart)
	extractLog.Info("AI 调用完成(异步)", "user", userID, "model", result.ModelUsed,
		"tokens", result.TokensUsed, "elapsed", callElapsed.String())

	// -------- 5. 解析 AI 输出 --------
	broadcast(fmt.Sprintf("\U0001f527 AI 输出完成(耗时 %d 秒),正在解析风格数据...", int(callElapsed.Seconds())))
	extracted, err := s.parseAIOutput(result.Content)
	if err != nil {
		extractLog.Error("JSON 解析失败(异步)", "raw_output_head", truncateForLog(result.Content, 500))
		broadcastErr("AI 输出解析失败: " + err.Error())
		return
	}
	if extracted.Error != "" {
		broadcastErr("AI 无法从输入提取风格: " + extracted.Error)
		return
	}

	// -------- 6. 校验关键字段 --------
	if err := s.validateExtracted(extracted); err != nil {
		extractLog.Warn("校验失败(异步)", "error", err)
		broadcastErr("AI 提取结果不完整: " + err.Error())
		return
	}

	// -------- 7. 序列化+入库 --------
	broadcast("\U0001f4be 正在保存草稿模板...")
	colorSchemeJSON, _ := json.Marshal(extracted.ColorScheme)
	cssVarsJSON, _ := json.Marshal(extracted.CSSVariables)
	samplePagesJSON, _ := json.Marshal(extracted.SamplePages)

	category := extracted.SuggestedCategory
	if !models.IsValidCWStyleCategory(category) {
		category = models.CWStyleMinimalist
	}
	name := strings.TrimSpace(extracted.SuggestedName)
	if name == "" {
		name = fmt.Sprintf("AI 提取草稿 %s", time.Now().Format("01-02 15:04"))
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
		"source_type":          sourceType,
		"original_html_length": totalHTMLLen,
		"sample_pages_count":   len(cleanedPages),
		"extracted_at":         time.Now().Format(time.RFC3339),
		"ai_model_used":        result.ModelUsed,
		"ai_tokens_used":       result.TokensUsed,
		"ai_extraction_notes":  extracted.ExtractionNotes,
	}

	if err := repository.CreateDraftTemplate(ctx, tpl, sourceMeta); err != nil {
		broadcastErr("草稿入库失败: " + err.Error())
		return
	}

	extractLog.Info("草稿创建成功(异步)", "id", tpl.ID, "name", tpl.Name,
		"category", tpl.StyleCategory, "user", userID)

	// -------- 8. 广播完成 --------
	GlobalCWSSEHub.Broadcast(sseKey, CWSSEEvent{
		EventType: CWSSEExtractDone,
		Data: map[string]interface{}{
			"template_id":        tpl.ID,
			"suggested_name":     tpl.Name,
			"suggested_desc":     tpl.Description,
			"suggested_category": tpl.StyleCategory,
			"extraction_notes":   extracted.ExtractionNotes,
			"message":            "\u2728 AI 已提取风格模板,您可以继续微调或保存",
		},
	})
}

