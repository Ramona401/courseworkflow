package services

// template_refine_service.go — 课件风格模板AI微调服务。
//
// 当前版本采用“结构保真微调”：
//
//   1. 原始sample_pages始终由后端持有。
//   2. AI只分析老师的修改意图，并返回配色、CSS变量和纯CSS覆盖规则。
//   3. AI不再输出或重写完整页面HTML。
//   4. 后端确定性替换旧颜色、字体和阴影令牌。
//   5. 后端向每个页面写入带固定标记的CSS覆盖层。
//   6. DOM、HTML层级、节点ID、脚本和交互函数不经过AI重写。
//   7. 每次修改前仍保存完整sample_pages历史快照，原回退机制保持有效。

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

// 模板微调专用SSE事件。
const (
	CWSSERefineStart    = "refine_start"
	CWSSERefineChunk    = "refine_chunk"
	CWSSERefineProgress = "refine_progress"
	CWSSERefineDone     = "refine_done"
	CWSSERefineError    = "refine_error"
)

// templateRefinePreserveContract 覆盖历史数据库提示词中的“重写sample_pages”要求。
const templateRefinePreserveContract = `
【TE-DNA模板微调结构保真补充契约·最高优先级】

你的任务是修改模板风格参数，而不是重新生成页面HTML。

必须只输出以下JSON结构：
{
  "color_scheme": {
    "primary": "...",
    "secondary": "...",
    "background": "...",
    "accent": "...",
    "text": "..."
  },
  "css_variables": {
    "--cw-primary": "...",
    "--cw-secondary": "...",
    "--cw-bg": "...",
    "--cw-accent": "...",
    "--cw-text": "...",
    "--cw-font-heading": "...",
    "--cw-font-body": "...",
    "--cw-radius": "...",
    "--cw-shadow": "..."
  },
  "css_overrides": "纯CSS规则字符串",
  "change_summary": "本次修改摘要",
  "suggested_category": "minimalist/playful/tech/academic/organic/immersive"
}

硬性规则：
1. 不得返回sample_pages，不得输出或重写任何完整HTML。
2. css_overrides只能包含CSS规则，不得包含style标签、HTML、JavaScript或Markdown围栏。
3. 不得使用@import、url()、expression()、javascript:或外部资源。
4. 不得通过CSS隐藏、删除或清空原页面主要内容。
5. 不得使用display:none、visibility:hidden或opacity:0批量隐藏现有结构。
6. 只落实用户明确提出的风格修改；要求模糊时选择最小改动。
7. 修改颜色时同步更新color_scheme和对应--cw-*变量。
8. 修改字体、圆角或阴影时同步更新对应CSS变量。
9. 需要覆盖原页面硬编码样式时，可在css_overrides中使用适度的!important。
10. 只输出合法JSON对象，不要解释文字。
`

// 模块级日志器。
var refineLog = logger.WithModule("template_refine")

// TemplateRefineService 课件风格模板AI微调服务。
type TemplateRefineService struct {
	cfg *config.Config
}

// NewTemplateRefineService 创建模板微调服务。
func NewTemplateRefineService(
	cfg *config.Config,
) *TemplateRefineService {
	return &TemplateRefineService{cfg: cfg}
}

// RefineTemplate 对模板执行结构保真AI微调。
func (s *TemplateRefineService) RefineTemplate(
	ctx context.Context,
	templateID string,
	userID string,
	instruction string,
) error {
	// 1. 加载模板和校验所有权。
	tpl, err := repository.GetTemplateForRefine(ctx, templateID)
	if err != nil {
		s.broadcastError(templateID, "模板不存在: "+err.Error())
		return fmt.Errorf("加载模板失败: %w", err)
	}
	if tpl.UserID == nil || *tpl.UserID != userID {
		s.broadcastError(templateID, "无权微调此模板")
		return fmt.Errorf("无权操作")
	}

	currentPages, err := decodeTemplateSamplePages(tpl.SamplePages)
	if err != nil {
		s.broadcastError(templateID, err.Error())
		return err
	}

	currentColors, err := parseTemplateStringMap(tpl.ColorScheme)
	if err != nil {
		s.broadcastError(templateID, "模板配色数据损坏")
		return fmt.Errorf("解析模板配色失败: %w", err)
	}

	currentVariables, err := parseTemplateStringMap(tpl.CSSVariables)
	if err != nil {
		s.broadcastError(templateID, "模板CSS变量数据损坏")
		return fmt.Errorf("解析模板CSS变量失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineStart,
		Data: map[string]interface{}{
			"template_id": templateID,
			"instruction": instruction,
			"page_count":  len(currentPages),
			"message":     "正在分析修改意图，原始页面结构和交互将完整保留...",
		},
	})

	// 2. 只抽取代表页的分析副本给AI观察。
	analysisPages := selectTemplateAnalysisPages(
		currentPages,
		templateAnalysisMaxPages,
		templateAnalysisMaxRunesPerPage,
	)

	// 3. 加载系统提示词并追加最高优先级结构保真契约。
	sysPromptObj, err := repository.GetCurrentPromptByKey(
		"prompt_courseware_template_refine",
	)
	if err != nil {
		s.broadcastError(templateID, "加载微调提示词失败: "+err.Error())
		return fmt.Errorf("加载提示词失败: %w", err)
	}
	systemPrompt := strings.TrimSpace(sysPromptObj.Content) +
		"\n\n" +
		strings.TrimSpace(templateRefinePreserveContract)

	userPrompt := s.buildRefineUserPrompt(
		tpl,
		currentColors,
		currentVariables,
		analysisPages,
		len(currentPages),
		instruction,
	)

	// 4. 获取AI配置。
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_refine",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(templateID, "获取AI配置失败: "+err.Error())
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data: map[string]interface{}{
			"message": fmt.Sprintf(
				"AI正在分析%d个代表页的视觉规律，不会重写原页面...",
				len(analysisPages),
			),
		},
	})

	// 5. 流式调用AI。
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_refine",
		UserID:    &userID,
	}

	chunkCount := 0
	callStart := time.Now()

	onChunk := func(chunk string) error {
		chunkCount++

		if chunkCount == 1 {
			refineLog.Info(
				"模板微调首字节到达",
				"latency", time.Since(callStart).String(),
				"template", templateID,
			)
		}

		if chunkCount%5 != 0 {
			return nil
		}

		GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
			EventType: CWSSERefineChunk,
			Data: map[string]interface{}{
				"chunk_no": chunkCount,
				"message":  s.statusMessageByProgress(chunkCount),
			},
		})
		return nil
	}

	streamResult, err := ai.CallAIStream(
		aiCfg,
		systemPrompt,
		userPrompt,
		onChunk,
		traceCtx,
	)
	if err != nil {
		s.broadcastError(templateID, "AI调用失败: "+err.Error())
		return fmt.Errorf("AI流式调用失败: %w", err)
	}

	callElapsed := time.Since(callStart)
	refineLog.Info(
		"模板风格微调AI调用完成",
		"template", templateID,
		"model", streamResult.ModelUsed,
		"tokens", streamResult.TokensUsed,
		"chunks", chunkCount,
		"elapsed", callElapsed.String(),
		"original_pages", len(currentPages),
		"analysis_pages", len(analysisPages),
	)

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data: map[string]interface{}{
			"message": "AI风格参数已返回，正在执行确定性样式替换...",
		},
	})

	// 6. 解析和校验AI返回的风格协议。
	refined, err := s.parseRefineOutput(streamResult.Content)
	if err != nil {
		refineLog.Error(
			"模板微调JSON解析失败",
			"raw_output_head", truncateForLog(streamResult.Content, 500),
		)
		s.broadcastError(templateID, "解析AI输出失败: "+err.Error())
		return fmt.Errorf("解析AI输出失败: %w", err)
	}

	if refined.Error != "" {
		s.broadcastError(
			templateID,
			"AI无法处理此指令: "+refined.Error,
		)
		return fmt.Errorf("AI返回错误: %s", refined.Error)
	}

	if err := validateTemplateStyleRefineResult(refined); err != nil {
		s.broadcastError(templateID, "AI输出不完整: "+err.Error())
		return fmt.Errorf("AI输出校验失败: %w", err)
	}

	safeOverrides, err := sanitizeTemplateCSSOverrides(
		refined.CSSOverrides,
	)
	if err != nil {
		s.broadcastError(templateID, "CSS覆盖规则校验失败: "+err.Error())
		return err
	}

	// 7. 后端确定性应用风格，不让AI重写任何页面结构。
	refinedPages, err := applyTemplateStyleRefinement(
		currentPages,
		currentColors,
		refined.ColorScheme,
		currentVariables,
		refined.CSSVariables,
		safeOverrides,
	)
	if err != nil {
		s.broadcastError(templateID, "应用模板风格失败: "+err.Error())
		return err
	}

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data: map[string]interface{}{
			"message": fmt.Sprintf(
				"已保持全部%d页DOM和脚本，正在保存样式结果与历史快照...",
				len(refinedPages),
			),
		},
	})

	// 8. 构造修改前的完整历史快照。
	historyEntry := models.RefineHistoryEntry{
		Timestamp:          time.Now().Format(time.RFC3339),
		UserInstruction:    instruction,
		SamplePagesBefore:  currentPages,
		CSSVariablesBefore: tpl.CSSVariables,
		ColorSchemeBefore:  tpl.ColorScheme,
		ChangeSummary:      strings.TrimSpace(refined.ChangeSummary),
	}

	// 9. 序列化新的风格参数和后端确定性处理后的完整页面。
	newColorSchemeJSON, err := json.Marshal(refined.ColorScheme)
	if err != nil {
		s.broadcastError(templateID, "序列化配色方案失败")
		return err
	}

	newCSSVariablesJSON, err := json.Marshal(refined.CSSVariables)
	if err != nil {
		s.broadcastError(templateID, "序列化CSS变量失败")
		return err
	}

	newSamplePagesJSON, err := json.Marshal(refinedPages)
	if err != nil {
		s.broadcastError(templateID, "序列化模板页面失败")
		return err
	}

	newCategory := strings.TrimSpace(refined.SuggestedCategory)
	if !models.IsValidCWStyleCategory(newCategory) {
		newCategory = ""
	}

	// 10. 写入数据库并保留原有历史回退机制。
	err = repository.UpdateTemplateRefined(
		ctx,
		templateID,
		string(newSamplePagesJSON),
		string(newCSSVariablesJSON),
		string(newColorSchemeJSON),
		newCategory,
		historyEntry,
	)
	if err != nil {
		s.broadcastError(templateID, "保存微调结果失败: "+err.Error())
		return fmt.Errorf("数据库更新失败: %w", err)
	}

	finalCategory := newCategory
	if finalCategory == "" {
		finalCategory = tpl.StyleCategory
	}

	refineLog.Info(
		"模板结构保真微调成功",
		"template", templateID,
		"instruction", truncateForLog(instruction, 80),
		"summary", truncateForLog(refined.ChangeSummary, 120),
		"pages", len(refinedPages),
		"css_override_len", len(safeOverrides),
	)

	// 11. 返回前端当前完整页面，兼容现有微调弹窗刷新逻辑。
	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineDone,
		Data: map[string]interface{}{
			"template_id":       templateID,
			"color_scheme":      refined.ColorScheme,
			"css_variables":     refined.CSSVariables,
			"sample_pages":      refinedPages,
			"style_category":    finalCategory,
			"change_summary":    refined.ChangeSummary,
			"page_count":        len(refinedPages),
			"structure_saved":   true,
			"interaction_saved": true,
			"message":           "✨ 模板样式已更新，页面结构和交互代码保持不变",
		},
	})

	return nil
}

// statusMessageByProgress 根据流式片段数量生成进度文案。
func (s *TemplateRefineService) statusMessageByProgress(
	chunkCount int,
) string {
	switch {
	case chunkCount < 20:
		return "✨ AI正在理解您的风格修改要求..."
	case chunkCount < 60:
		return "🎨 正在规划新的配色、字体和视觉令牌..."
	case chunkCount < 120:
		return "🧩 正在生成安全的CSS覆盖规则..."
	case chunkCount < 240:
		return "🔧 正在检查样式一致性和最小改动边界..."
	default:
		return "⏳ 风格参数即将完成，请稍候..."
	}
}

// buildRefineUserPrompt 构建模板风格分析提示词。
//
// 不再把全部sample_pages JSON直接发送给AI，只发送最多8个代表页分析副本。
func (s *TemplateRefineService) buildRefineUserPrompt(
	tpl *models.CoursewareTemplate,
	currentColors map[string]string,
	currentVariables map[string]string,
	analysisPages []templateAnalysisPage,
	originalPageCount int,
	instruction string,
) string {
	var sb strings.Builder

	colorJSON, _ := json.Marshal(currentColors)
	variableJSON, _ := json.Marshal(currentVariables)

	sb.WriteString("# 当前模板\n\n")
	sb.WriteString(fmt.Sprintf("模板名称：%s\n", tpl.Name))
	sb.WriteString(fmt.Sprintf("当前风格类别：%s\n", tpl.StyleCategory))
	sb.WriteString(fmt.Sprintf("原始母版页数：%d页\n", originalPageCount))
	sb.WriteString("原始页面已由平台完整持有，不需要也不允许重新输出HTML。\n\n")

	sb.WriteString("## 当前配色\n```json\n")
	sb.Write(colorJSON)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## 当前CSS变量\n```json\n")
	sb.Write(variableJSON)
	sb.WriteString("\n```\n\n")

	sb.WriteString(fmt.Sprintf(
		"## 代表页分析副本（共%d页）\n\n",
		len(analysisPages),
	))

	for _, page := range analysisPages {
		sb.WriteString(fmt.Sprintf(
			"=== 原模板第%d页（推断页型：%s）开始 ===\n",
			page.SourcePageNumber,
			page.RoleLabel,
		))
		sb.WriteString(page.HTML)
		sb.WriteString(fmt.Sprintf(
			"\n=== 原模板第%d页结束 ===\n\n",
			page.SourcePageNumber,
		))
	}

	sb.WriteString("# 老师的修改指令\n\n")
	sb.WriteString(strings.TrimSpace(instruction))
	sb.WriteString("\n\n")

	sb.WriteString("# 输出要求\n\n")
	sb.WriteString("只返回color_scheme、css_variables、css_overrides、change_summary和suggested_category。\n")
	sb.WriteString("不得输出sample_pages，不得复述HTML，不得重写页面结构。\n")
	sb.WriteString("css_overrides必须是JSON字符串中的纯CSS内容，不能包含style标签或Markdown围栏。\n")
	sb.WriteString("只输出合法JSON对象，不要任何解释文字。")

	return sb.String()
}

// parseRefineOutput 解析AI模板风格微调输出。
func (s *TemplateRefineService) parseRefineOutput(
	aiOutput string,
) (*templateStyleRefineResult, error) {
	text := strings.TrimSpace(cwStripCodeFences(aiOutput))

	var jsonText string
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		jsonText = text
	} else if extracted, ok := ai.ExtractJSON(text); ok {
		jsonText = extracted
	} else {
		return nil, fmt.Errorf("未能从AI输出中找到JSON对象")
	}

	jsonText = sanitizeNonASCIIForJSON(jsonText)

	var result templateStyleRefineResult
	if err := json.Unmarshal([]byte(jsonText), &result); err == nil {
		return &result, nil
	}

	cleaned := cwCleanChinesePunctuation(jsonText)
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		refineLog.Info("模板微调JSON经中文标点清理后解析成功")
		return &result, nil
	}

	fixed := cwFixJSONQuotes(cleaned)
	if err := json.Unmarshal([]byte(fixed), &result); err == nil {
		refineLog.Info("模板微调JSON经引号修复后解析成功")
		return &result, nil
	}

	// 字段级宽容兜底。
	result = templateStyleRefineResult{
		CSSOverrides: decodeExtractedJSONString(
			extractStringField(cleaned, "css_overrides"),
		),
		ChangeSummary: decodeExtractedJSONString(
			extractStringField(cleaned, "change_summary"),
		),
		SuggestedCategory: decodeExtractedJSONString(
			extractStringField(cleaned, "suggested_category"),
		),
		Error: decodeExtractedJSONString(
			extractStringField(cleaned, "error"),
		),
	}

	if value := extractObjectField(cleaned, "color_scheme"); value != "" {
		_ = json.Unmarshal([]byte(value), &result.ColorScheme)
	}
	if value := extractObjectField(cleaned, "css_variables"); value != "" {
		_ = json.Unmarshal([]byte(value), &result.CSSVariables)
	}

	if len(result.ColorScheme) == 0 &&
		len(result.CSSVariables) == 0 {
		return nil, fmt.Errorf("多重兜底解析全部失败")
	}

	refineLog.Warn("模板微调JSON使用字段级宽容提取成功")
	return &result, nil
}

// broadcastError 广播模板微调错误。
func (s *TemplateRefineService) broadcastError(
	templateID string,
	message string,
) {
	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineError,
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// RollbackToHistory 回退到指定模板微调历史。
func (s *TemplateRefineService) RollbackToHistory(
	ctx context.Context,
	templateID string,
	userID string,
	historyIndex int,
) (*models.CoursewareTemplate, error) {
	tpl, err := repository.GetTemplateForRefine(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}
	if tpl.UserID == nil || *tpl.UserID != userID {
		return nil, fmt.Errorf("无权操作此模板")
	}

	if err := repository.RollbackToHistory(
		ctx,
		templateID,
		historyIndex,
	); err != nil {
		return nil, fmt.Errorf("回退失败: %w", err)
	}

	updated, err := repository.GetTemplateForRefine(
		ctx,
		templateID,
	)
	if err != nil {
		return nil, fmt.Errorf("回退后重新加载失败: %w", err)
	}

	refineLog.Info(
		"模板历史回退成功",
		"template", templateID,
		"index", historyIndex,
		"user", userID,
	)
	return updated, nil
}

// GetRefineHistory 读取模板微调历史。
func (s *TemplateRefineService) GetRefineHistory(
	ctx context.Context,
	templateID string,
	userID string,
) ([]models.RefineHistoryEntry, error) {
	tpl, err := repository.GetTemplateForRefine(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}
	if tpl.UserID == nil || *tpl.UserID != userID {
		return nil, fmt.Errorf("无权操作此模板")
	}

	return repository.GetRefineHistory(ctx, templateID)
}
