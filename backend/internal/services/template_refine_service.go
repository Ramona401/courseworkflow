package services

// template_refine_service.go — 课件风格模板 AI 微调服务(v139.1-fix)
//
// 功能:
//   用户对已有模板提出修改意见,AI 通过 SSE 流式生成更新后的样例页面等,
//   自动写入 refine_history 快照供历史回退。
//
// v139.1-fix 修改:
//   - log.Printf 全部替换为 logger.WithModule 结构化日志
//   - JSON 解析前复用 sanitizeNonASCIIForJSON 预处理

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

// ==================== SSE 事件类型常量(v139 模板微调专用) ====================

const (
	CWSSERefineStart    = "refine_start"    // 微调任务开始
	CWSSERefineChunk    = "refine_chunk"    // AI 输出片段(纯进度提示)
	CWSSERefineProgress = "refine_progress" // 阶段状态消息
	CWSSERefineDone     = "refine_done"     // 微调完成,含新模板内容
	CWSSERefineError    = "refine_error"    // 微调失败
)

// 模块级日志器
var refineLog = logger.WithModule("template_refine")

// ==================== 服务结构体 ====================

// TemplateRefineService 课件风格模板 AI 微调服务
type TemplateRefineService struct {
	cfg *config.Config
}

// NewTemplateRefineService 构造函数
func NewTemplateRefineService(cfg *config.Config) *TemplateRefineService {
	return &TemplateRefineService{cfg: cfg}
}

// ==================== 入口方法:RefineTemplate(异步执行,SSE 推送)====================

// RefineTemplate 对模板执行 AI 微调
// 此方法被 handler 异步调用(go func + 800ms 延迟等前端 SSE 连接建立)
func (s *TemplateRefineService) RefineTemplate(
	ctx context.Context,
	templateID string,
	userID string,
	instruction string,
) error {
	// -------- 1. 加载当前模板 --------
	tpl, err := repository.GetTemplateForRefine(ctx, templateID)
	if err != nil {
		s.broadcastError(templateID, "模板不存在: "+err.Error())
		return fmt.Errorf("加载模板失败: %w", err)
	}

	// 权限校验:只能微调自己的草稿或个人模板
	if tpl.UserID == nil || *tpl.UserID != userID {
		s.broadcastError(templateID, "无权微调此模板")
		return fmt.Errorf("无权操作")
	}

	// 广播开始事件
	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineStart,
		Data: map[string]interface{}{
			"template_id": templateID,
			"instruction": instruction,
			"message":     "正在根据您的意见调整模板...",
		},
	})

	// -------- 2. 构建 AI 提示词 --------
	sysPromptObj, err := repository.GetCurrentPromptByKey("prompt_courseware_template_refine")
	if err != nil {
		s.broadcastError(templateID, "加载微调提示词失败: "+err.Error())
		return fmt.Errorf("加载提示词失败: %w", err)
	}

	userPrompt := s.buildRefineUserPrompt(tpl, instruction)

	// -------- 3. 获取 AI 配置 --------
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_template_refine",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(templateID, "获取 AI 配置失败: "+err.Error())
		return fmt.Errorf("获取 AI 配置失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data:      map[string]interface{}{"message": "AI 正在生成调整方案..."},
	})

	// -------- 4. SSE 流式调用 AI --------
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_template_refine",
		UserID:    &userID,
	}

	chunkCount := 0
	var firstChunkTime time.Time
	callStart := time.Now()

	// onChunk 回调:每收到 AI 输出片段时,广播给前端做进度提示
	onChunk := func(chunk string) error {
		chunkCount++
		if chunkCount == 1 {
			firstChunkTime = time.Now()
			refineLog.Info("首字节延迟", "latency", firstChunkTime.Sub(callStart).String(), "template", templateID)
		}

		// 每 5 个 chunk 推送一次进度,避免 SSE 推送过频
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

	// 调用 SSE 流式 AI
	streamResult, err := ai.CallAIStream(aiCfg, sysPromptObj.Content, userPrompt, onChunk, traceCtx)
	if err != nil {
		s.broadcastError(templateID, "AI 调用失败: "+err.Error())
		return fmt.Errorf("AI 流式调用失败: %w", err)
	}

	callElapsed := time.Since(callStart)
	refineLog.Info("AI 流式完成", "template", templateID, "model", streamResult.ModelUsed, "tokens", streamResult.TokensUsed, "chunks", chunkCount, "elapsed", callElapsed.String())

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data:      map[string]interface{}{"message": "AI 输出完成,正在解析..."},
	})

	// -------- 5. 解析完整 JSON 输出 --------
	refined, err := s.parseRefineOutput(streamResult.Content)
	if err != nil {
		refineLog.Error("JSON 解析失败", "raw_output_head", truncateForLog(streamResult.Content, 500))
		s.broadcastError(templateID, "解析 AI 输出失败: "+err.Error())
		return fmt.Errorf("解析 AI 输出失败: %w", err)
	}

	// AI 自己表示无法处理
	if refined.Error != "" {
		s.broadcastError(templateID, "AI 无法处理此指令: "+refined.Error)
		return fmt.Errorf("AI 返回错误: %s", refined.Error)
	}

	// -------- 6. 校验关键字段完整性 --------
	if err := s.validateRefined(refined); err != nil {
		s.broadcastError(templateID, "AI 输出不完整: "+err.Error())
		return fmt.Errorf("AI 输出校验失败: %w", err)
	}

	// 样例页面数量一致性检查(宽容策略,只 WARN 不阻塞)
	currentPagesCount := s.countSamplePagesInJSON(tpl.SamplePages)
	if len(refined.SamplePages) != currentPagesCount && currentPagesCount > 0 {
		refineLog.Warn("AI 返回样例页面数不一致,接受新数量", "ai_count", len(refined.SamplePages), "current_count", currentPagesCount)
	}

	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineProgress,
		Data:      map[string]interface{}{"message": "正在保存修改并备份历史快照..."},
	})

	// -------- 7. 构造历史快照(修改前的内容)--------
	var currentSamplePagesArr []string
	if tpl.SamplePages != "" {
		_ = json.Unmarshal([]byte(tpl.SamplePages), &currentSamplePagesArr)
	}

	historyEntry := models.RefineHistoryEntry{
		Timestamp:          time.Now().Format(time.RFC3339),
		UserInstruction:    instruction,
		SamplePagesBefore:  currentSamplePagesArr,
		CSSVariablesBefore: tpl.CSSVariables,
		ColorSchemeBefore:  tpl.ColorScheme,
		ChangeSummary:      strings.TrimSpace(refined.ChangeSummary),
	}

	// -------- 8. 序列化新内容 --------
	newColorSchemeJSON, _ := json.Marshal(refined.ColorScheme)
	newCSSVarsJSON, _ := json.Marshal(refined.CSSVariables)
	newSamplePagesJSON, _ := json.Marshal(refined.SamplePages)

	// 风格类别变更兜底
	newCategory := refined.SuggestedCategory
	if !models.IsValidCWStyleCategory(newCategory) {
		newCategory = "" // 传空字符串到 repository,表示不更新 style_category
	}

	// -------- 9. 写入数据库 --------
	err = repository.UpdateTemplateRefined(
		ctx,
		templateID,
		string(newSamplePagesJSON),
		string(newCSSVarsJSON),
		string(newColorSchemeJSON),
		newCategory,
		historyEntry,
	)
	if err != nil {
		s.broadcastError(templateID, "保存微调结果失败: "+err.Error())
		return fmt.Errorf("数据库更新失败: %w", err)
	}

	refineLog.Info("微调成功", "template", templateID, "instruction", truncateForLog(instruction, 50), "summary", truncateForLog(refined.ChangeSummary, 80))

	// -------- 10. 广播完成事件 --------
	finalCategory := newCategory
	if finalCategory == "" {
		finalCategory = tpl.StyleCategory
	}
	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineDone,
		Data: map[string]interface{}{
			"template_id":    templateID,
			"color_scheme":   refined.ColorScheme,
			"css_variables":  refined.CSSVariables,
			"sample_pages":   refined.SamplePages,
			"style_category": finalCategory,
			"change_summary": refined.ChangeSummary,
			"message":        "✨ 模板已更新",
		},
	})

	return nil
}

// ==================== 辅助:阶段进度文案 ====================

func (s *TemplateRefineService) statusMessageByProgress(chunkCount int) string {
	switch {
	case chunkCount < 20:
		return "✨ AI 正在分析您的修改意图..."
	case chunkCount < 60:
		return "🎨 正在调整配色和样式变量..."
	case chunkCount < 150:
		return "📝 正在更新样例页面 HTML..."
	case chunkCount < 300:
		return "🔧 正在精细调整细节..."
	default:
		return "⏳ 即将完成,请稍候..."
	}
}

// ==================== AI 提示词构建 ====================

func (s *TemplateRefineService) buildRefineUserPrompt(tpl *models.CoursewareTemplate, instruction string) string {
	var sb strings.Builder

	sb.WriteString("# 当前模板的完整内容\n\n")
	sb.WriteString(fmt.Sprintf("模板名称: %s\n", tpl.Name))
	sb.WriteString(fmt.Sprintf("风格类别: %s\n\n", tpl.StyleCategory))

	sb.WriteString("## color_scheme(当前配色)\n```json\n")
	if tpl.ColorScheme != "" {
		sb.WriteString(tpl.ColorScheme)
	} else {
		sb.WriteString("{}")
	}
	sb.WriteString("\n```\n\n")

	sb.WriteString("## css_variables(当前 CSS 变量)\n```json\n")
	if tpl.CSSVariables != "" {
		sb.WriteString(tpl.CSSVariables)
	} else {
		sb.WriteString("{}")
	}
	sb.WriteString("\n```\n\n")

	sb.WriteString("## sample_pages(当前样例页面)\n```json\n")
	if tpl.SamplePages != "" {
		sb.WriteString(tpl.SamplePages)
	} else {
		sb.WriteString("[]")
	}
	sb.WriteString("\n```\n\n")

	sb.WriteString("# 用户的修改指令\n\n")
	sb.WriteString(instruction)
	sb.WriteString("\n\n")

	sb.WriteString("# 请输出修改后的完整 JSON\n\n")
	sb.WriteString("严格按照系统提示词中约定的 5 个字段输出(color_scheme / css_variables / sample_pages / change_summary / suggested_category)。\n")
	sb.WriteString("只输出 JSON 对象,不要任何说明文字或代码块标记。")

	return sb.String()
}

// ==================== AI 输出解析(四重兜底)====================

// parseRefineOutput 解析 AI 微调输出 JSON
// 复用 sanitizeNonASCIIForJSON 预处理非 ASCII 字符
func (s *TemplateRefineService) parseRefineOutput(aiOutput string) (*models.AIRefinedTemplate, error) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	jsonStr, ok := ai.ExtractJSON(text)
	if !ok {
		if strings.HasPrefix(text, "{") {
			jsonStr = text
			ok = true
		}
	}
	if !ok || jsonStr == "" {
		return nil, fmt.Errorf("未能从 AI 输出中找到 JSON 对象")
	}

	// 非 ASCII 预处理
	jsonStr = sanitizeNonASCIIForJSON(jsonStr)

	// 尝试 1: 直接 Unmarshal
	var refined models.AIRefinedTemplate
	if err := json.Unmarshal([]byte(jsonStr), &refined); err == nil {
		return &refined, nil
	}

	// 尝试 2: 清理中文标点
	cleaned := cwCleanChinesePunctuation(jsonStr)
	if err := json.Unmarshal([]byte(cleaned), &refined); err == nil {
		refineLog.Info("JSON 解析:中文标点清理后成功")
		return &refined, nil
	}

	// 尝试 3: 修复未转义引号
	fixed := cwFixJSONQuotes(cleaned)
	if err := json.Unmarshal([]byte(fixed), &refined); err == nil {
		refineLog.Info("JSON 解析:引号修复后成功")
		return &refined, nil
	}

	// 尝试 4: 字段级宽容提取
	refined = models.AIRefinedTemplate{}
	refined.ChangeSummary = extractStringField(cleaned, "change_summary")
	refined.SuggestedCategory = extractStringField(cleaned, "suggested_category")
	if cs := extractObjectField(cleaned, "color_scheme"); cs != "" {
		_ = json.Unmarshal([]byte(cs), &refined.ColorScheme)
	}
	if cv := extractObjectField(cleaned, "css_variables"); cv != "" {
		_ = json.Unmarshal([]byte(cv), &refined.CSSVariables)
	}
	if sp := extractArrayField(cleaned, "sample_pages"); sp != "" {
		_ = json.Unmarshal([]byte(sp), &refined.SamplePages)
	}

	if len(refined.ColorScheme) == 0 && len(refined.SamplePages) == 0 {
		return nil, fmt.Errorf("四重兜底解析全部失败")
	}
	refineLog.Warn("JSON 解析:字段级宽容提取成功")
	return &refined, nil
}

// ==================== AI 输出校验 ====================

func (s *TemplateRefineService) validateRefined(r *models.AIRefinedTemplate) error {
	requiredColorKeys := []string{"primary", "secondary", "background", "accent", "text"}
	for _, k := range requiredColorKeys {
		if r.ColorScheme[k] == "" {
			return fmt.Errorf("color_scheme.%s 缺失", k)
		}
	}

	requiredCSSKeys := []string{
		"--cw-primary", "--cw-secondary", "--cw-bg", "--cw-accent", "--cw-text",
		"--cw-font-heading", "--cw-font-body", "--cw-radius", "--cw-shadow",
	}
	for _, k := range requiredCSSKeys {
		if r.CSSVariables[k] == "" {
			return fmt.Errorf("css_variables.%s 缺失", k)
		}
	}

	validPages := 0
	for _, p := range r.SamplePages {
		if strings.TrimSpace(p) != "" {
			validPages++
		}
	}
	if validPages == 0 {
		return fmt.Errorf("sample_pages 数组为空")
	}
	return nil
}

// ==================== 辅助函数 ====================

func (s *TemplateRefineService) countSamplePagesInJSON(jsonStr string) int {
	if jsonStr == "" || jsonStr == "[]" {
		return 0
	}
	var pages []string
	if err := json.Unmarshal([]byte(jsonStr), &pages); err != nil {
		return 0
	}
	return len(pages)
}

func (s *TemplateRefineService) broadcastError(templateID string, message string) {
	GlobalCWSSEHub.Broadcast(templateID, CWSSEEvent{
		EventType: CWSSERefineError,
		Data:      map[string]interface{}{"message": message},
	})
}

// ==================== 回退到历史快照 ====================

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

	if err := repository.RollbackToHistory(ctx, templateID, historyIndex); err != nil {
		return nil, fmt.Errorf("回退失败: %w", err)
	}

	updated, err := repository.GetTemplateForRefine(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("回退后重新加载失败: %w", err)
	}

	refineLog.Info("回退成功", "template", templateID, "index", historyIndex, "user", userID)
	return updated, nil
}

// ==================== 历史读取 ====================

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
