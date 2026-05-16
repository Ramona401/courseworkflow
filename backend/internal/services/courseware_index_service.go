package services

// courseware_index_service.go — 课件索引AI生成服务（Phase 3.5 两层AI架构+脉络概述）
//
// 两层AI架构：
//   层1（索引压缩）：教案全文 → AI压缩 → 课件脉络概述 + 页级AOCI索引
//     提示词: prompt_courseware_index（prompts表，后台可管理）
//     场景: courseware_index (gemini-3.1-pro)
//
//   层2（方案翻译）：AOCI索引 → AI翻译 → 用户友好的产品方案（JSON数组）
//     提示词: prompt_courseware_scheme（prompts表，后台可管理）
//     场景: scanner (haiku，低成本)
//
// 流程：
//   1-6. 获取教案→调层1 AI→解析OVERVIEW+PAGE索引
//   7-8. 调层2 AI→解析JSON→合并索引+方案
//   9-11. 写入数据库→SSE广播

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 课件索引生成服务 ====================

// CoursewareIndexService 课件索引AI生成服务
type CoursewareIndexService struct {
	cfg *config.Config
}

// NewCoursewareIndexService 创建课件索引生成服务
func NewCoursewareIndexService(cfg *config.Config) *CoursewareIndexService {
	return &CoursewareIndexService{cfg: cfg}
}

// ==================== 层2 AI输出JSON结构 ====================

// cwSchemeItem 层2 AI返回的单页方案
type cwSchemeItem struct {
	PageNumber          int    `json:"page_number"`
	Title               string `json:"title"`
	Purpose             string `json:"purpose"`
	ContentSummary      string `json:"content_summary"`
	InteractionType     string `json:"interaction_type"`
	VisualFormat        string `json:"visual_format"`
	MediaRequirements   string `json:"media_requirements"`
	EstimatedComplexity int    `json:"estimated_complexity"`
}

// ==================== 核心方法：生成课件索引（两层AI） ====================

// GenerateIndex 生成课件索引（异步执行，通过SSE推送进度）
func (s *CoursewareIndexService) GenerateIndex(ctx context.Context, coursewareID string, userID string) error {
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

	// ---- 2. 获取关联教案全部内容 ----
	lp, err := repository.GetLessonPlanByID(ctx, cw.LessonPlanID)
	if err != nil {
		s.broadcastError(coursewareID, "关联教案不存在: "+err.Error())
		return fmt.Errorf("关联教案不存在: %w", err)
	}
	lessonContent := s.extractLessonPlanContent(lp)
	if len(lessonContent) < 50 {
		s.broadcastError(coursewareID, "教案内容过少，无法生成课件方案")
		return fmt.Errorf("教案内容过少")
	}

	// ---- 3. 更新课件状态为 indexing ----
	if cw.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusIndexing)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"lesson_plan":  lp.Title,
			"message":      "正在分析教案内容，生成课件方案...",
		},
	})

	// ==================== 层1：AOCI索引压缩 ====================

	// ---- 4. 加载层1提示词 ----
	dictPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_index")
	if err != nil {
		s.broadcastError(coursewareID, "加载索引字典失败: "+err.Error())
		return fmt.Errorf("加载索引字典失败: %w", err)
	}

	// ---- 5. 调用层1 AI（courseware_index场景） ----
	aiCfg1, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_index",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(coursewareID, "获取AI配置失败: "+err.Error())
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	userPrompt1 := s.buildLayer1UserPrompt(lp, lessonContent)
	traceCtx1 := &ai.TraceContext{SceneCode: "courseware_index", UserID: &userID}
	callResult1, err := ai.CallAI(aiCfg1, dictPrompt.Content, userPrompt1, traceCtx1)
	if err != nil {
		s.broadcastError(coursewareID, "AI索引压缩失败: "+err.Error())
		return fmt.Errorf("层1 AI调用失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "索引压缩完成，正在生成详细方案..."},
	})

	// ---- 6. 解析层1输出 ----
	overview, pageText := s.splitOverviewAndPages(callResult1.Content)
	rawPages, err := s.parseAOCIIndexOutput(pageText)
	if err != nil {
		s.broadcastError(coursewareID, "解析索引输出失败: "+err.Error())
		return fmt.Errorf("解析层1输出失败: %w", err)
	}
	if len(rawPages) == 0 {
		s.broadcastError(coursewareID, "AI未生成任何页面索引")
		return fmt.Errorf("层1未生成任何页面")
	}

	// ==================== 层2：AI方案翻译 ====================

	// ---- 7. 加载层2提示词 ----
	schemePrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	if err != nil {
		// 层2提示词缺失时降级为规则翻译
		log.Printf("[courseware_index] 层2提示词缺失，降级为规则翻译: %v", err)
		pages := s.fallbackTranslateToPages(rawPages, coursewareID)
		return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
	}

	// ---- 8. 构建层2输入（将所有页面的AOCI索引拼接） ----
	var indexBuf strings.Builder
	indexBuf.WriteString(fmt.Sprintf("课件标题：%s\n学科：%s\n年级：%s\n总页数：%d\n\n", lp.Title, lp.Subject, lp.Grade, len(rawPages)))
	for _, rp := range rawPages {
		indexBuf.WriteString(rp.RawIndex)
		indexBuf.WriteString("\n\n")
	}

	// ---- 9. 调用层2 AI（scanner场景，Haiku低成本） ----
	aiCfg2, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		log.Printf("[courseware_index] 层2 AI配置失败，降级规则翻译: %v", err)
		pages := s.fallbackTranslateToPages(rawPages, coursewareID)
		return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
	}

	traceCtx2 := &ai.TraceContext{SceneCode: "scanner", UserID: &userID}
	callResult2, err := ai.CallAI(aiCfg2, schemePrompt.Content, indexBuf.String(), traceCtx2)
	if err != nil {
		log.Printf("[courseware_index] 层2 AI调用失败，降级规则翻译: %v", err)
		pages := s.fallbackTranslateToPages(rawPages, coursewareID)
		return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "方案生成完成，正在整理..."},
	})

	// ---- 10. 解析层2 JSON输出 ----
	schemes, err := s.parseSchemeJSON(callResult2.Content)
	if err != nil {
		log.Printf("[courseware_index] 层2 JSON解析失败，降级规则翻译: %v", err)
		pages := s.fallbackTranslateToPages(rawPages, coursewareID)
		return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
	}

	// ---- 11. 合并层1索引+层2方案 → CoursewarePage ----
	pages := s.mergeIndexAndScheme(rawPages, schemes, coursewareID)

	log.Printf("[courseware_index] 两层AI完成: cw=%s pages=%d overview=%d字 L1=%s/%dtok L2=%s/%dtok",
		coursewareID, len(pages), len([]rune(overview)),
		callResult1.ModelUsed, callResult1.TokensUsed,
		callResult2.ModelUsed, callResult2.TokensUsed)

	return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
}

// ==================== 保存并广播（统一出口） ====================

func (s *CoursewareIndexService) saveAndBroadcast(ctx context.Context, coursewareID string, overview string, pages []*models.CoursewarePage) error {
	// 删除旧页面
	if err := repository.DeleteAllCoursewarePages(ctx, coursewareID); err != nil {
		log.Printf("[courseware_index] 删除旧页面失败: %v", err)
	}
	// 批量创建新页面
	if err := repository.BatchCreateCoursewarePages(ctx, pages); err != nil {
		s.broadcastError(coursewareID, "保存页面失败: "+err.Error())
		return fmt.Errorf("批量创建页面失败: %w", err)
	}
	_ = repository.UpdateCoursewarePageCount(ctx, coursewareID, len(pages))

	// 保存脉络概述
	if overview != "" {
		_ = repository.UpdateCoursewareOverview(ctx, coursewareID, overview)
	}

	// 逐页广播
	for _, page := range pages {
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEIndexPage, Data: page,
		})
	}

	// 广播完成
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexDone,
		Data: map[string]interface{}{
			"courseware_id":   coursewareID,
			"page_count":     len(pages),
			"index_overview":  overview,
			"message":        fmt.Sprintf("课件方案生成完成，共 %d 页", len(pages)),
		},
	})
	return nil
}

// ==================== 层1 提示词构建 ====================

func (s *CoursewareIndexService) buildLayer1UserPrompt(lp *models.LessonPlan, content string) string {
	var sb strings.Builder
	sb.WriteString("请根据以下教案内容，先输出课件脉络概述（OVERVIEW:），再为每一页生成AOCI压缩索引。\n\n")
	sb.WriteString("## 教案基本信息\n")
	sb.WriteString(fmt.Sprintf("- 标题：%s\n", lp.Title))
	sb.WriteString(fmt.Sprintf("- 学科：%s\n", lp.Subject))
	sb.WriteString(fmt.Sprintf("- 年级：%s\n", lp.Grade))
	sb.WriteString("\n## 教案完整内容\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\n请严格按照字典格式输出（先OVERVIEW:概述，再PAGE:页面索引，不要任何格式之外的说明文字）：")
	return sb.String()
}

// ==================== 层2 JSON解析 ====================

// parseSchemeJSON 解析层2 AI返回的JSON数组
// parseSchemeJSON 解析层2 AI返回的JSON数组
// 使用宽容策略：先尝试直接解析，失败则修复JSON后重试
func (s *CoursewareIndexService) parseSchemeJSON(aiOutput string) ([]cwSchemeItem, error) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	jsonStr := cwExtractJSONArray(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("层2输出中未找到JSON数组")
	}

	// 第一次尝试：直接解析
	var schemes []cwSchemeItem
	if err := json.Unmarshal([]byte(jsonStr), &schemes); err == nil && len(schemes) > 0 {
		return schemes, nil
	}

	// 第二次尝试：清理中文标点后解析
	cleaned := cwCleanChinesePunctuation(jsonStr)
	if err := json.Unmarshal([]byte(cleaned), &schemes); err == nil && len(schemes) > 0 {
		return schemes, nil
	}

	// 第三次尝试：修复JSON值内部的未转义引号
	fixed := cwFixJSONQuotes(cleaned)
	if err := json.Unmarshal([]byte(fixed), &schemes); err == nil && len(schemes) > 0 {
		log.Printf("[courseware_index] 层2 JSON修复后解析成功")
		return schemes, nil
	}

	// 第四次尝试：逐个对象提取（最宽容）
	schemes = cwExtractJSONObjects(cleaned)
	if len(schemes) > 0 {
		log.Printf("[courseware_index] 层2通过逐对象提取成功: %d个", len(schemes))
		return schemes, nil
	}

	return nil, fmt.Errorf("层2 JSON解析失败(四重兜底均失败), 前200字: %s", cwTruncate(jsonStr, 200))
}

// cwFixJSONQuotes 修复JSON值内部的未转义双引号
// 策略：在JSON字符串值内部，如果遇到"不是跟在\后面的"，且后面不是,:]}等JSON分隔符，则转义它
func cwFixJSONQuotes(s string) string {
	var result strings.Builder
	inString := false
	result.Grow(len(s))

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '"' {
			if !inString {
				// 进入字符串
				inString = true
				result.WriteByte(c)
			} else {
				// 在字符串内遇到引号——判断是结束引号还是内嵌引号
				// 检查后面的字符：如果是 , : ] } 或空白后跟这些，则是结束引号
				isEnd := false
				for j := i + 1; j < len(s); j++ {
					next := s[j]
					if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
						continue
					}
					if next == ',' || next == ':' || next == ']' || next == '}' || next == '"' {
						isEnd = true
					}
					break
				}
				// 检查前面是否是反斜杠转义
				if i > 0 && s[i-1] == '\\' {
					result.WriteByte(c)
					continue
				}
				if isEnd {
					inString = false
					result.WriteByte(c)
				} else {
					// 内嵌引号，转义为空（直接删除）
					// 不写入任何东西，等于删除这个引号
				}
			}
		} else {
			result.WriteByte(c)
		}
	}
	return result.String()
}

// cwExtractJSONObjects 逐个提取JSON对象（最宽容的兜底方案）
// 按page_number/title/purpose等关键字段逐个提取
func cwExtractJSONObjects(text string) []cwSchemeItem {
	var items []cwSchemeItem

	// 按 "page_number" 分割
	parts := strings.Split(text, "\"page_number\"")
	if len(parts) < 2 {
		return nil
	}

	for i := 1; i < len(parts); i++ {
		chunk := "\"page_number\"" + parts[i]
		// 尝试找到这个对象的结束位置
		braceEnd := strings.Index(chunk, "}")
		if braceEnd < 0 {
			continue
		}
		objStr := "{" + chunk[:braceEnd+1]
		
		var item cwSchemeItem
		if err := json.Unmarshal([]byte(objStr), &item); err == nil && item.Title != "" {
			items = append(items, item)
		}
	}
	return items
}

// ==================== 合并层1索引+层2方案 ====================

// mergeIndexAndScheme 合并层1的AOCI索引和层2的用户方案为CoursewarePage
// 按页码对齐：以层1为主骨架，层2方案覆盖用户字段
func (s *CoursewareIndexService) mergeIndexAndScheme(rawPages []*cwRawPageIndex, schemes []cwSchemeItem, coursewareID string) []*models.CoursewarePage {
	// 建立层2方案的页码索引
	schemeMap := make(map[int]*cwSchemeItem)
	for i := range schemes {
		schemeMap[schemes[i].PageNumber] = &schemes[i]
	}

	var pages []*models.CoursewarePage
	for _, rp := range rawPages {
		// 层1基础数据
		il := cwClamp(rp.IL, 1, 5)
		cg := cwClamp(rp.CG, 1, 6)

		page := &models.CoursewarePage{
			CoursewareID:        coursewareID,
			PageNumber:          rp.PageNumber,
			PageIndex:           rp.RawIndex,
			IdxCognitiveLevel:   cg,
			IdxInteractionLevel: il,
			IdxVisualFormat:     rp.VF,
			Status:              models.CWPageStatusPending,
		}

		// 层2方案覆盖用户字段
		if sc, ok := schemeMap[rp.PageNumber]; ok {
			page.Title = strings.TrimSpace(sc.Title)
			page.Purpose = strings.TrimSpace(sc.Purpose)
			page.ContentSummary = strings.TrimSpace(sc.ContentSummary)
			page.InteractionType = strings.TrimSpace(sc.InteractionType)
			page.VisualFormat = strings.TrimSpace(sc.VisualFormat)
			page.MediaRequirements = strings.TrimSpace(sc.MediaRequirements)
			page.EstimatedComplexity = cwClamp(sc.EstimatedComplexity, 1, 5)
		} else {
			// 层2未覆盖此页，用层1数据兜底
			page.Title = rp.Title
			page.Purpose = cwJoinNonEmpty("；", "知识目标："+rp.Knowledge, "能力目标："+rp.Ability)
			page.ContentSummary = rp.Content
			page.InteractionType = cwILToInteractionType[strconv.Itoa(rp.IL)]
			page.VisualFormat = cwVFToVisualFormat[rp.VF]
			page.EstimatedComplexity = il
			if page.InteractionType == "" {
				page.InteractionType = "static"
			}
			if page.VisualFormat == "" {
				page.VisualFormat = "text_heavy"
			}
		}

		pages = append(pages, page)
	}
	return pages
}

// ==================== 降级：规则翻译（层2 AI失败时兜底） ====================

var cwVFToVisualFormat = map[string]string{
	"TH": "text_heavy", "IT": "image_text", "DG": "diagram", "CT": "chart",
	"TL": "timeline", "CP": "comparison", "GL": "gallery", "FM": "fullscreen_media",
}
var cwILToInteractionType = map[string]string{
	"1": "static", "2": "click", "3": "input", "4": "drag", "5": "game",
}

func (s *CoursewareIndexService) fallbackTranslateToPages(rawPages []*cwRawPageIndex, coursewareID string) []*models.CoursewarePage {
	var pages []*models.CoursewarePage
	for _, rp := range rawPages {
		il := cwClamp(rp.IL, 1, 5)
		cg := cwClamp(rp.CG, 1, 6)

		visualFormat := cwVFToVisualFormat[rp.VF]
		if visualFormat == "" {
			visualFormat = "text_heavy"
		}
		interactionType := cwILToInteractionType[strconv.Itoa(rp.IL)]
		if interactionType == "" {
			interactionType = "static"
		}

		mediaReq := ""
		if strings.Contains(rp.Interaction, "视频") || strings.Contains(rp.Interaction, "动画") || rp.TG == "V" {
			mediaReq = rp.Interaction
		}

		page := &models.CoursewarePage{
			CoursewareID:        coursewareID,
			PageNumber:          rp.PageNumber,
			Title:               rp.Title,
			Purpose:             cwJoinNonEmpty("；", "知识目标："+rp.Knowledge, "能力目标："+rp.Ability),
			ContentSummary:      rp.Content,
			InteractionType:     interactionType,
			VisualFormat:        visualFormat,
			MediaRequirements:   mediaReq,
			EstimatedComplexity: il,
			PageIndex:           rp.RawIndex,
			IdxCognitiveLevel:   cg,
			IdxInteractionLevel: il,
			IdxVisualFormat:     rp.VF,
			Status:              models.CWPageStatusPending,
		}
		pages = append(pages, page)
	}
	return pages
}

// ==================== 教案内容提取 ====================

func (s *CoursewareIndexService) extractLessonPlanContent(lp *models.LessonPlan) string {
	var parts []string
	if lp.ConversationLog != "" {
		messages := s.parseConversationLog(lp.ConversationLog)
		var longestMsg string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" && len(messages[i].Content) > len(longestMsg) {
				longestMsg = messages[i].Content
			}
		}
		if len(longestMsg) > 200 {
			parts = append(parts, longestMsg)
		}
	}
	if lp.AIReviewResult != "" && len(parts) == 0 {
		parts = append(parts, "【AI评审结果】\n"+lp.AIReviewResult)
	}
	if len(parts) == 0 && lp.AIReviewHistory != "" {
		parts = append(parts, "【教案历史】\n"+lp.AIReviewHistory)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

type cwConversationMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *CoursewareIndexService) parseConversationLog(logJSON string) []cwConversationMsg {
	if logJSON == "" || logJSON == "null" || logJSON == "[]" {
		return nil
	}
	var messages []cwConversationMsg
	if err := json.Unmarshal([]byte(logJSON), &messages); err != nil {
		log.Printf("[courseware_index] 解析对话日志失败: %v", err)
		return nil
	}
	return messages
}

// ==================== 概述与页面分离 ====================

func (s *CoursewareIndexService) splitOverviewAndPages(aiOutput string) (overview string, pageText string) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	overviewIdx := strings.Index(text, "OVERVIEW:")
	pageIdx := strings.Index(text, "PAGE:")

	if overviewIdx >= 0 && pageIdx > overviewIdx {
		overviewRaw := text[overviewIdx+len("OVERVIEW:") : pageIdx]
		overview = strings.TrimSpace(overviewRaw)
		pageText = strings.TrimSpace(text[pageIdx:])
	} else if pageIdx >= 0 {
		pageText = strings.TrimSpace(text[pageIdx:])
	} else {
		pageText = text
	}
	return
}

// ==================== 层1：AOCI索引输出解析 ====================

type cwRawPageIndex struct {
	PageNumber  int
	Title       string
	RawIndex    string
	KT          string
	CG          int
	IL          int
	VF          string
	TG          string
	Knowledge   string
	Ability     string
	Interaction string
	Recovery    string
	Content     string
}

func (s *CoursewareIndexService) parseAOCIIndexOutput(pageText string) ([]*cwRawPageIndex, error) {
	text := strings.TrimSpace(pageText)
	if text == "" {
		return nil, fmt.Errorf("页面索引文本为空")
	}
	blocks := cwSplitBlocks(text)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("未找到有效的页面索引块")
	}

	var pages []*cwRawPageIndex
	for _, block := range blocks {
		page := s.parseSinglePageBlock(block)
		if page != nil {
			pages = append(pages, page)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("解析后无有效页面（原始块数=%d）", len(blocks))
	}
	for i, p := range pages {
		p.PageNumber = i + 1
	}
	return pages, nil
}

func (s *CoursewareIndexService) parseSinglePageBlock(block string) *cwRawPageIndex {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) < 2 {
		return nil
	}
	page := &cwRawPageIndex{RawIndex: block}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "PAGE:") {
			parts := strings.SplitN(line, "|", 2)
			pageStr := strings.TrimPrefix(parts[0], "PAGE:")
			page.PageNumber, _ = strconv.Atoi(strings.TrimSpace(pageStr))
			if len(parts) > 1 && strings.HasPrefix(parts[1], "TT:") {
				page.Title = strings.TrimSpace(strings.TrimPrefix(parts[1], "TT:"))
			}
			continue
		}
		if strings.HasPrefix(line, "KT:") && strings.Contains(line, "|") {
			s.parseEncodingLine(page, line)
			continue
		}
		if len(line) >= 3 && line[0] == '[' && line[2] == ']' {
			tag := string(line[1])
			content := strings.TrimSpace(line[3:])
			switch tag {
			case "K":
				page.Knowledge = content
			case "A":
				page.Ability = content
			case "I":
				page.Interaction = content
			case "R":
				page.Recovery = content
			case "C":
				page.Content = content
			}
		}
	}
	if page.Title == "" && page.Knowledge == "" {
		return nil
	}
	return page
}

func (s *CoursewareIndexService) parseEncodingLine(page *cwRawPageIndex, line string) {
	for _, part := range strings.Split(line, "|") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "KT":
			page.KT = strings.TrimSpace(kv[1])
		case "CG":
			page.CG, _ = strconv.Atoi(strings.TrimSpace(kv[1]))
		case "IL":
			page.IL, _ = strconv.Atoi(strings.TrimSpace(kv[1]))
		case "VF":
			page.VF = strings.TrimSpace(kv[1])
		case "TG":
			page.TG = strings.TrimSpace(kv[1])
		}
	}
}

// ==================== 辅助函数 ====================

func (s *CoursewareIndexService) broadcastError(coursewareID string, message string) {
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEError,
		Data:      map[string]interface{}{"message": message},
	})
}

// cwClamp 数值钳位
func cwClamp(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// cwJoinNonEmpty 拼接非空字符串
func cwJoinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		// 跳过仅有前缀的空内容（如"知识目标："）
		if trimmed != "" && !strings.HasSuffix(trimmed, "：") && !strings.HasSuffix(trimmed, ":") {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	return strings.Join(nonEmpty, sep)
}

// cwExtractJSONArray 从文本中提取JSON数组
func cwExtractJSONArray(text string) string {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return ""
}

func cwStripCodeFences(text string) string {
	if strings.HasPrefix(text, "```") {
		idx := strings.Index(text, "\n")
		if idx >= 0 {
			text = text[idx+1:]
		}
	}
	text = strings.TrimSpace(text)
	if strings.HasSuffix(text, "```") {
		text = text[:len(text)-3]
	}
	return strings.TrimSpace(text)
}

func cwSplitBlocks(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var blocks []string
	lines := strings.Split(text, "\n")
	var currentBlock []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PAGE:") && len(currentBlock) > 0 {
			blocks = append(blocks, strings.Join(currentBlock, "\n"))
			currentBlock = nil
		}
		if trimmed != "" {
			currentBlock = append(currentBlock, line)
		}
	}
	if len(currentBlock) > 0 {
		blocks = append(blocks, strings.Join(currentBlock, "\n"))
	}
	return blocks
}

func cwTruncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// cwCleanChinesePunctuation 清理JSON字符串中的中文标点符号
// AI在生成JSON时经常使用中文标点，导致JSON解析失败
// 关键策略：中文引号必须删除（不能替换为英文引号，否则破坏JSON结构）
func cwCleanChinesePunctuation(s string) string {
	replacer := strings.NewReplacer(
		"\u201c", "", "\u201d", "",         // 中文双引号 " " → 删除
		"\u2018", "", "\u2019", "",         // 中文单引号 ' ' → 删除
		"\u3001", ",", "\uff0c", ",",       // 顿号、全角逗号
		"\uff1a", ":", "\uff1b", ";",       // 全角冒号、分号
		"\uff08", "(", "\uff09", ")",       // 全角括号
		"\u300a", "", "\u300b", "",         // 书名号 《 》→ 删除
		"\u3008", "", "\u3009", "",         // 尖括号 〈 〉→ 删除
		"\u2014\u2014", "-",               // 破折号 ——
		"\u2014", "-",                       // 单个破折号 —
		"\u2026", "...",                     // 省略号 …
		"\uff01", "!", "\uff1f", "?",       // 全角感叹号、问号
	)
	return replacer.Replace(s)
}
