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
//
// 注：层2 JSON 解析（parseSchemeJSON 及其多重兜底、cwExtractJSONArray /
//     cwExtractJSONObjects / cwExtractSchemeByFields / cwTruncate）已拆分至
//     同包文件 courseware_index_json.go，保持本文件聚焦业务编排。
//
// v0.43 修复（RefineIndex 修改方案不看原文/doc来源被硬拦）：
//   - RefineIndex 去除 LessonPlanID==nil 硬拦截，改为按 source_type 注入原文上下文：
//       lesson_plan → 教案正文；doc_upload → 重新读docx原文；其余 → 用课件基本信息
//   - RefineIndex 内置 docx 原文读取（archive/zip + encoding/xml，不依赖 PPT 服务，
//     避免与 courseware_ppt_service / courseware_doc_service 形成循环依赖）
//   - 修改提示词补页数下限约束，避免修改后被概括成极少页
//
// v0.44 新增（doc/ppt 直翻路径补脉络与索引，体验Y+方案）：
//   - GenerateOverviewFromPages：方案出页后用 haiku 快速生成"哪几页干什么"脉络（前台，几秒）
//   - BackfillPageIndexAsync：后台异步对照"教案原文+当前方案"为每页生成 AOCI 索引并回填
//     （haiku，整批，复用 parseAOCIIndexOutput 解析，失败不影响主流程）
//
// v0.44.1 修复（后台补索引在"改过方案"时可能静默贴错索引）：
//   BackfillPageIndexAsync 原按"解析顺序第i个 ↔ 当前页第i个"对齐。若老师在后台
//   任务运行的窗口内改了方案（删页/调序），会导致索引贴错页（比留空更危险）。
//   本次加双保险：
//     A. 页数守卫——记录喂给AI的页数，回填前重新查当前页，页数不一致则整体放弃
//        本次回填（留给夜间轮询补），绝不在不确定对齐时硬写。
//     B. 标题锚点匹配——不再纯靠顺序，用 AI 回显的页标题(TT) 与当前页标题匹配定位
//        具体 page_number 再写；匹配不上的块跳过。宁可留空也不贴错。

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"path/filepath"
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
func (s *CoursewareIndexService) GenerateIndex(ctx context.Context, coursewareID string, userID string, preset string) error {
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
	if cw.LessonPlanID == nil || *cw.LessonPlanID == "" {
		s.broadcastError(coursewareID, "课件未关联教案，无法生成方案")
		return fmt.Errorf("课件未关联教案")
	}
	lp, err := repository.GetLessonPlanByID(ctx, *cw.LessonPlanID)
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
			"lesson_plan":   lp.Title,
			"message":       "正在分析教案内容，生成课件方案...",
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

	userPrompt1 := s.buildLayer1UserPrompt(lp, lessonContent, preset)
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
			"courseware_id":  coursewareID,
			"page_count":     len(pages),
			"index_overview": overview,
			"message":        fmt.Sprintf("课件方案生成完成，共 %d 页", len(pages)),
		},
	})
	return nil
}

// ==================== 层1 提示词构建 ====================

func (s *CoursewareIndexService) buildLayer1UserPrompt(lp *models.LessonPlan, content string, preset string) string {
	var sb strings.Builder
	sb.WriteString("请根据以下教案内容，先输出课件脉络概述（OVERVIEW:），再为每一页生成AOCI压缩索引。\n\n")
	sb.WriteString("## 教案基本信息\n")
	sb.WriteString(fmt.Sprintf("- 标题：%s\n", lp.Title))
	sb.WriteString(fmt.Sprintf("- 学科：%s\n", lp.Subject))
	sb.WriteString(fmt.Sprintf("- 年级：%s\n", lp.Grade))
	sb.WriteString("\n## 教案完整内容\n\n")
	sb.WriteString(content)
	// v136: 注入方案结构预设提示
	if preset != "" {
		presetObj := models.GetSchemePresetByKey(preset)
		if presetObj != nil && presetObj.PromptHint != "" {
			sb.WriteString("\n\n")
			sb.WriteString(presetObj.PromptHint)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n\n请严格按照字典格式输出（先OVERVIEW:概述，再PAGE:页面索引，不要任何格式之外的说明文字）：")
	return sb.String()
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

	// 优先级1: content_markdown — 教案正文（Fork/导入/手动编辑的教案内容存储在此字段）
	if lp.ContentMarkdown != "" && len(strings.TrimSpace(lp.ContentMarkdown)) > 50 {
		parts = append(parts, lp.ContentMarkdown)
	}

	// 优先级2: conversation_log — 对话记录中最长的assistant消息（AI备课工坊生成的教案）
	if len(parts) == 0 && lp.ConversationLog != "" {
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

	// 优先级3: ai_review_result — AI评审结果
	if len(parts) == 0 && lp.AIReviewResult != "" {
		parts = append(parts, "【AI评审结果】\n"+lp.AIReviewResult)
	}

	// 优先级4: ai_review_history — 评审历史（最后兜底）
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

// cwStripCodeFences 剥离Markdown代码围栏
// 注：此函数为跨服务共享公共工具（template_extract_service / template_refine_service
//
//	及其测试均依赖），保持在本文件，请勿随意改动其行为。
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

// cwCleanChinesePunctuation 清理JSON字符串中的中文标点符号
// AI在生成JSON时经常使用中文标点，导致JSON解析失败
// 关键策略：中文引号必须删除（不能替换为英文引号，否则破坏JSON结构）
//
// 注：此函数为跨服务共享公共工具（template_extract_service / template_refine_service
//
//	及其测试均依赖），保持在本文件，请勿随意改动其行为。
func cwCleanChinesePunctuation(s string) string {
	replacer := strings.NewReplacer(
		"\u201c", "", "\u201d", "", // 中文双引号 " " → 删除
		"\u2018", "", "\u2019", "", // 中文单引号 ' ' → 删除
		"\u3001", ",", "\uff0c", ",", // 顿号、全角逗号
		"\uff1a", ":", "\uff1b", ";", // 全角冒号、分号
		"\uff08", "(", "\uff09", ")", // 全角括号
		"\u300a", "", "\u300b", "", // 书名号 《 》→ 删除
		"\u3008", "", "\u3009", "", // 尖括号 〈 〉→ 删除
		"\u2014\u2014", "-", // 破折号 ——
		"\u2014", "-", // 单个破折号 —
		"\u2026", "...", // 省略号 …
		"\uff01", "!", "\uff1f", "?", // 全角感叹号、问号
	)
	return replacer.Replace(s)
}

// cwFixJSONQuotes 修复JSON值内部的未转义双引号
// 策略：在JSON字符串值内部，如果遇到"不是跟在\后面的"，且后面不是,:]}等JSON分隔符，则转义它
//
// 注：此函数为跨服务共享公共工具（template_extract_service / template_refine_service
//
//	及其测试均依赖），保持在本文件，请勿随意改动其行为。
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

// ==================== v136新增：AI修改方案 ====================

// RefineIndex 根据用户反馈修改课件方案（异步执行，通过SSE推送进度）
//
// v0.43 修复：
//   - 去除 LessonPlanID==nil 硬拦截，doc_upload / topic_direct / ppt_upload 也可修改
//   - 按 source_type 注入原文上下文（教案正文 / docx原文 / 课件基本信息）
//   - 补页数下限约束，避免修改后被概括成极少页
//
// 流程：
//  1. 获取课件 + 当前全部页面方案
//  2. 按来源取原文上下文
//  3. 拼接 原文上下文 + 当前方案 + 用户反馈
//  4. 调用AI（courseware_scheme场景，降级scanner）重新生成方案JSON
//  5. 解析JSON，尽量保留层1索引，仅更新层2用户字段
//  6. 写入数据库并SSE广播
func (s *CoursewareIndexService) RefineIndex(ctx context.Context, coursewareID string, userID string, feedback string) error {
	// ---- 1. 获取课件 ----
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		s.broadcastError(coursewareID, "课件不存在: "+err.Error())
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		s.broadcastError(coursewareID, "无权操作此课件")
		return fmt.Errorf("无权操作此课件")
	}

	// 获取当前全部页面（修改方案的基础）
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		s.broadcastError(coursewareID, "当前没有可修改的方案页面")
		return fmt.Errorf("当前没有可修改的方案页面")
	}

	// ---- 2. 按来源取原文上下文 + 课件基本信息 ----
	// 注意：不再因 LessonPlanID 为空而拒绝；doc/topic/ppt 来源同样支持修改方案
	title, subject, grade, sourceContext := s.buildRefineSourceContext(ctx, cw)

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"message":       "正在根据您的意见修改方案...",
		},
	})

	// ---- 3. 构建修改提示词 ----
	var promptBuf strings.Builder
	promptBuf.WriteString("你是课件方案修改专家。用户对当前课件方案提出了修改意见，请结合下方的教案/原始内容，按意见调整方案。\n\n")
	promptBuf.WriteString(fmt.Sprintf("## 课件基本信息\n- 标题：%s\n- 学科：%s\n- 年级：%s\n\n", title, subject, grade))

	// 原文上下文（教案正文 / docx原文）——有则注入，供AI据实修改而非凭空发挥
	if strings.TrimSpace(sourceContext) != "" {
		promptBuf.WriteString("## 教案/原始内容（修改时须依据此内容）\n\n")
		// 截断，避免超出上下文
		srcText := sourceContext
		if len([]rune(srcText)) > 24000 {
			srcText = string([]rune(srcText)[:24000]) + "\n\n[内容过长，已截取前24000字]"
		}
		promptBuf.WriteString(srcText)
		promptBuf.WriteString("\n\n")
	}

	promptBuf.WriteString("## 当前方案（需要修改）\n")
	for _, p := range pages {
		promptBuf.WriteString(fmt.Sprintf("第%d页 | 标题：%s | 目的：%s | 概要：%s | 交互：%s | 视觉：%s | 复杂度：%d\n",
			p.PageNumber, p.Title, p.Purpose, p.ContentSummary, p.InteractionType, p.VisualFormat, p.EstimatedComplexity))
	}
	promptBuf.WriteString(fmt.Sprintf("\n## 用户修改意见\n%s\n\n", feedback))

	// 页数下限约束：若用户要求增加页数，给出明确区间，避免AI缩水
	minPages, rangeDesc := cwRecommendPageRange(grade, len([]rune(sourceContext)))
	promptBuf.WriteString("## 篇幅与页数要求\n")
	promptBuf.WriteString(fmt.Sprintf("- 若需扩充内容，目标页数区间参考：%s；一般情况下不应少于 %d 页（除非用户明确要求精简）。\n", rangeDesc, minPages))
	promptBuf.WriteString("- 严禁把方案概括成一页或极少数几页。\n\n")

	promptBuf.WriteString("## 输出要求\n")
	promptBuf.WriteString("请输出修改后的完整方案，格式为JSON数组。每个元素包含以下字段：\n")
	promptBuf.WriteString("page_number(int), title(string), purpose(string), content_summary(string), interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n")
	promptBuf.WriteString("\n可以增加、删除或修改页面。page_number从1开始连续编号。\n")
	promptBuf.WriteString("交互类型可选：static/click/drag/input/animation/video/game/quiz\n")
	promptBuf.WriteString("视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n")
	promptBuf.WriteString("\n请只输出JSON数组，不要有任何额外说明文字。")

	// ---- 4. 调用AI ----
	schemePromptObj, sErr := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	systemPrompt := ""
	if sErr == nil {
		systemPrompt = schemePromptObj.Content
	} else {
		systemPrompt = "你是课件方案设计专家，请按要求输出JSON格式的课件方案。"
	}

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_scheme",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		// courseware_scheme场景不存在时降级到scanner
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
		Data:      map[string]interface{}{"message": "AI正在修改方案..."},
	})

	traceCtx := &ai.TraceContext{SceneCode: "courseware_scheme", UserID: &userID}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, promptBuf.String(), traceCtx)
	if err != nil {
		s.broadcastError(coursewareID, "AI修改方案失败: "+err.Error())
		return fmt.Errorf("AI调用失败: %w", err)
	}

	// ---- 5. 解析AI输出的JSON ----
	schemes, err := s.parseSchemeJSON(callResult.Content)
	if err != nil {
		s.broadcastError(coursewareID, "解析修改后的方案失败: "+err.Error())
		return fmt.Errorf("解析方案失败: %w", err)
	}
	if len(schemes) == 0 {
		s.broadcastError(coursewareID, "AI未返回有效方案")
		return fmt.Errorf("AI未返回有效方案")
	}

	// ---- 6. 构建新的CoursewarePage列表 ----
	// 尽量保留原有页面的层1索引信息
	oldPageMap := make(map[int]*models.CoursewarePage)
	for _, p := range pages {
		oldPageMap[p.PageNumber] = p
	}

	var newPages []*models.CoursewarePage
	for _, sc := range schemes {
		pn := sc.PageNumber
		if pn <= 0 {
			pn = len(newPages) + 1
		}
		page := &models.CoursewarePage{
			CoursewareID:        coursewareID,
			PageNumber:          pn,
			Title:               strings.TrimSpace(sc.Title),
			Purpose:             strings.TrimSpace(sc.Purpose),
			ContentSummary:      strings.TrimSpace(sc.ContentSummary),
			InteractionType:     strings.TrimSpace(sc.InteractionType),
			VisualFormat:        strings.TrimSpace(sc.VisualFormat),
			MediaRequirements:   strings.TrimSpace(sc.MediaRequirements),
			EstimatedComplexity: cwClamp(sc.EstimatedComplexity, 1, 5),
			Status:              models.CWPageStatusPending,
		}
		// 如果原有相同页码的页面有层1索引，保留
		if oldPage, ok := oldPageMap[pn]; ok {
			page.PageIndex = oldPage.PageIndex
			page.IdxCognitiveLevel = oldPage.IdxCognitiveLevel
			page.IdxInteractionLevel = oldPage.IdxInteractionLevel
			page.IdxVisualFormat = oldPage.IdxVisualFormat
		}
		if page.InteractionType == "" {
			page.InteractionType = "static"
		}
		if page.VisualFormat == "" {
			page.VisualFormat = "text_heavy"
		}
		newPages = append(newPages, page)
	}

	// 重新编号（确保连续）
	for i, p := range newPages {
		p.PageNumber = i + 1
	}

	log.Printf("[courseware_index] RefineIndex完成: cw=%s source=%s oldPages=%d newPages=%d model=%s tokens=%d feedback=%s",
		coursewareID, cw.SourceType, len(pages), len(newPages), callResult.ModelUsed, callResult.TokensUsed, cwTruncate(feedback, 50))

	// ---- 7. 保存并广播 ----
	// 保留原有概述不变
	return s.saveAndBroadcast(ctx, coursewareID, cw.IndexOverview, newPages)
}

// buildRefineSourceContext 为"修改方案"按课件来源装配 标题/学科/年级 与原文上下文
// 返回 (title, subject, grade, sourceContext)
//   - lesson_plan：title/subject/grade 取教案，sourceContext 取教案正文
//   - doc_upload  ：title/subject/grade 取课件，sourceContext 读取docx原文
//   - 其它(topic/ppt/3d/html)：取课件基本信息，sourceContext 为空（用当前方案即可修改）
//
// 设计要点：本函数在 index_service 内部完成 docx 读取，不调用 PPT/Doc 服务，
// 避免与 courseware_ppt_service / courseware_doc_service 形成循环依赖。
func (s *CoursewareIndexService) buildRefineSourceContext(ctx context.Context, cw *models.Courseware) (string, string, string, string) {
	title := cw.Title
	subject := cw.Subject
	grade := cw.Grade
	sourceContext := ""

	switch cw.SourceType {
	case models.CWSourceLessonPlan:
		// 关联教案：优先用教案正文作为修改依据
		if cw.LessonPlanID != nil && *cw.LessonPlanID != "" {
			lp, err := repository.GetLessonPlanByID(ctx, *cw.LessonPlanID)
			if err == nil && lp != nil {
				if strings.TrimSpace(lp.Title) != "" {
					title = lp.Title
				}
				if strings.TrimSpace(lp.Subject) != "" {
					subject = lp.Subject
				}
				if strings.TrimSpace(lp.Grade) != "" {
					grade = lp.Grade
				}
				sourceContext = s.extractLessonPlanContent(lp)
			} else {
				log.Printf("[courseware_index] RefineIndex 读取关联教案失败: cw=%s err=%v", cw.ID, err)
			}
		}

	case models.CWSourceDocUpload:
		// Word文档来源：重新读取已存储的docx原文
		if cw.SourceFilePath != "" {
			docFullPath := filepath.Join(DocUploadDir, cw.SourceFilePath)
			text, err := readDocxFullText(docFullPath)
			if err == nil && strings.TrimSpace(text) != "" {
				sourceContext = text
			} else {
				log.Printf("[courseware_index] RefineIndex 读取docx原文失败: cw=%s path=%s err=%v", cw.ID, docFullPath, err)
			}
		}
	}

	// topic_direct / ppt_upload / 3d_single / html_import：sourceContext 留空，
	// 仅依据当前方案 + 用户意见进行修改（这些来源原文价值有限或已体现在当前方案中）
	return title, subject, grade, sourceContext
}

// readDocxFullText 读取.docx文件的全部正文文本（index_service内部独立实现，无外部依赖）
// 与 courseware_doc_service.ExtractDocContent 逻辑等价，但不引入服务间依赖，
// 仅供 RefineIndex 在doc来源时读取原文使用。
func readDocxFullText(docxPath string) (string, error) {
	r, err := zip.OpenReader(docxPath)
	if err != nil {
		return "", fmt.Errorf("打开docx文件失败: %w", err)
	}
	defer r.Close()

	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("docx文件中未找到 word/document.xml")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("打开document.xml失败: %w", err)
	}
	defer rc.Close()

	data, err := readAllFromReader(rc)
	if err != nil {
		return "", fmt.Errorf("读取document.xml失败: %w", err)
	}

	// 按 <w:p> 段落边界提取，<w:t> 内文本拼接
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose

	var paragraphs []string
	var cur []string
	inP := false
	for {
		tok, e := decoder.Token()
		if e != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" {
				inP = true
				cur = nil
			}
		case xml.EndElement:
			if t.Name.Local == "p" && inP {
				inP = false
				para := strings.TrimSpace(strings.Join(cur, ""))
				if para != "" {
					paragraphs = append(paragraphs, para)
				}
				cur = nil
			}
		case xml.CharData:
			if inP {
				cur = append(cur, string(t))
			}
		}
	}
	return strings.Join(paragraphs, "\n\n"), nil
}

// readAllFromReader 读取io.Reader全部内容（避免在本文件引入io包仅用一次的额外import面）
func readAllFromReader(rc interface {
	Read(p []byte) (int, error)
}) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 32*1024)
	for {
		n, err := rc.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			// io.EOF 的字符串也是 "EOF"，上面已处理；其余错误返回
			if n == 0 {
				return buf, err
			}
		}
		if n == 0 {
			return buf, nil
		}
	}
}

// ==================== v0.42新增：从主题直接生成课件索引 ====================

// GenerateIndexFromTopic 从主题直接生成课件索引（无教案，纯AI规划）
// 流程：
//  1. 校验课件状态和权限
//  2. 用主题信息构建提示词，跳过层1（无教案内容可压缩）
//  3. 直接调层2 AI生成方案JSON
//  4. 写入数据库并SSE广播
func (s *CoursewareIndexService) GenerateIndexFromTopic(ctx context.Context, coursewareID string, userID string, req *models.CreateCoursewareFromTopicRequest, preset string) error {
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

	// ---- 2. 更新课件状态为 indexing ----
	if cw.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusIndexing)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"message":       "正在根据主题规划课件方案...",
		},
	})

	// ---- 3. 构建主题直接生成的提示词 ----
	// 课程知识库轮：若前端传了知识点编码，先查 curriculum_standards 构建难度适配约束段落
	// 为空/查询失败/查不到时 constraint 为空串，buildTopicDirectPrompt 退回原有纯主题规划逻辑
	curriculumConstraint := BuildCurriculumConstraint(ctx, req.KPCodes)
	userPrompt := s.buildTopicDirectPrompt(req, preset, curriculumConstraint)

	// ---- 4. 加载提示词模板（复用 courseware_scheme 场景） ----
	schemePrompt, sErr := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	systemPrompt := ""
	if sErr == nil {
		systemPrompt = schemePrompt.Content
	} else {
		systemPrompt = "你是K12课件规划专家，请按要求输出JSON格式的课件方案。"
	}

	// ---- 5. 调用AI（courseware_scheme场景，降级到scanner） ----
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
		Data:      map[string]interface{}{"message": "AI正在规划课件结构..."},
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

	// ---- 6. 解析JSON输出 ----
	schemes, err := s.parseSchemeJSON(callResult.Content)
	if err != nil {
		s.broadcastError(coursewareID, "解析方案失败: "+err.Error())
		return fmt.Errorf("解析方案失败: %w", err)
	}
	if len(schemes) == 0 {
		s.broadcastError(coursewareID, "AI未返回有效方案")
		return fmt.Errorf("AI未返回有效方案")
	}

	// ---- 7. 构建CoursewarePage列表（无层1索引，全部来自层2方案） ----
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

	// 生成简要概述
	overview := fmt.Sprintf("主题：%s（%s·%s），共%d页课件方案，由AI根据主题直接规划。",
		req.Topic, req.Subject, req.Grade, len(pages))

	log.Printf("[courseware_index] TopicDirect完成: cw=%s pages=%d model=%s tokens=%d topic=%s",
		coursewareID, len(pages), callResult.ModelUsed, callResult.TokensUsed, req.Topic)

	return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
}

// buildTopicDirectPrompt 构建主题直接生成的用户提示词
func (s *CoursewareIndexService) buildTopicDirectPrompt(req *models.CreateCoursewareFromTopicRequest, preset string, curriculumConstraint string) string {
	var sb strings.Builder
	sb.WriteString("你是K12课件规划专家。\n根据以下信息，设计一份完整的课件大纲（每页详细说明）。\n\n")
	sb.WriteString(fmt.Sprintf("学科: %s\n", req.Subject))
	sb.WriteString(fmt.Sprintf("年级: %s\n", req.Grade))
	sb.WriteString(fmt.Sprintf("主题: %s\n", req.Topic))
	if req.PageRange != "" {
		sb.WriteString(fmt.Sprintf("期望页数: %s\n", req.PageRange))
	} else {
		sb.WriteString("期望页数: 按学段默认（小学15-25页，初中20-30页，高中22-35页）\n")
	}
	if req.ExtraNotes != "" {
		sb.WriteString(fmt.Sprintf("额外说明: %s\n", req.ExtraNotes))
	}

	// 课程知识库轮：注入课标知识点与难度适配约束（非空时启用"难度自动适配"）
	if strings.TrimSpace(curriculumConstraint) != "" {
		sb.WriteString(curriculumConstraint)
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

	sb.WriteString("\n请输出JSON数组格式。每个元素包含以下字段：\n")
	sb.WriteString("page_number(int), title(string), purpose(string), content_summary(string), ")
	sb.WriteString("interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n\n")
	sb.WriteString("设计原则：\n")
	sb.WriteString("1. 遵循课程标准，知识点覆盖完整\n")
	sb.WriteString("2. 结构：封面(1页) → 学习目标(1页) → 知识讲授(主体) → 练习(2-3页) → 总结(1页)\n")
	sb.WriteString("3. 交互类型分布：纯展示≤40%，简单交互30-40%，复杂交互≤20%\n")
	sb.WriteString("4. 难度递进：前1/3基础 → 中1/3进阶 → 后1/3综合\n\n")
	sb.WriteString("交互类型可选：static/click/drag/input/animation/video/game/quiz\n")
	sb.WriteString("视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n\n")
	sb.WriteString("请只输出JSON数组，不要有任何额外说明文字。")
	return sb.String()
}

// ==================== v0.44新增：直翻路径补脉络（前台快速）与补索引（后台异步） ====================

// GenerateOverviewFromPages 根据已生成的页面方案，用 haiku 快速生成"哪几页干什么"的脉络概述
//
// 用途：doc/ppt 直翻路径出页后，前台立即调用此方法补一段真脉络（替代套话overview）。
// 输入小（仅页码+标题+目的）、模型便宜（scanner/haiku）、几秒返回，不显著拖慢体验。
//
// 返回脉络字符串；失败或为空时返回空串，调用方应退回套话overview，不阻塞主流程。
func (s *CoursewareIndexService) GenerateOverviewFromPages(
	ctx context.Context, userID string,
	title string, subject string, grade string,
	pages []*models.CoursewarePage,
) string {
	if len(pages) == 0 {
		return ""
	}

	// 构建提示词：喂页码+标题+目的，要求输出连贯脉络
	var sb strings.Builder
	sb.WriteString("你是课件结构分析专家。下面是一份课件的逐页方案，请你用一段连贯的中文概括这份课件的整体脉络。\n\n")
	sb.WriteString(fmt.Sprintf("课件标题：%s\n学科：%s\n年级：%s\n总页数：%d\n\n", title, subject, grade, len(pages)))
	sb.WriteString("## 逐页方案\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("第%d页：%s —— %s\n", p.PageNumber, p.Title, p.Purpose))
	}
	sb.WriteString("\n## 输出要求\n")
	sb.WriteString("1. 用80-150字概括，说明这份课件分几个部分、哪几页讲什么，例如\"第1-3页为情境导入，第4-8页讲解核心概念，第9-12页为分组练习，第13页课堂小结\"。\n")
	sb.WriteString("2. 按页码顺序归并相邻的同类页面，体现教学的递进逻辑。\n")
	sb.WriteString("3. 只输出这段脉络文字，不要任何标题、前缀、markdown 或额外说明。\n")

	systemPrompt := "你是课件结构分析专家，擅长用简洁连贯的语言概括课件的教学脉络。"

	// 用 scanner 场景（haiku，便宜快）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		log.Printf("[courseware_index] 生成脉络-获取AI配置失败，退回套话: %v", err)
		return ""
	}

	traceCtx := &ai.TraceContext{SceneCode: "scanner", UserID: &userID}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, sb.String(), traceCtx)
	if err != nil {
		log.Printf("[courseware_index] 生成脉络-AI调用失败，退回套话: %v", err)
		return ""
	}

	overview := strings.TrimSpace(cwStripCodeFences(callResult.Content))
	// 防御：若AI输出异常长（跑题），截断到合理长度
	if len([]rune(overview)) > 400 {
		overview = string([]rune(overview)[:400])
	}
	return overview
}

// BackfillPageIndexAsync 后台异步：对照"教案原文 + 当前页面方案"，为每页生成 AOCI 索引并回填
//
// 用途：doc/ppt 直翻路径下，页面创建时 page_index 及 CG/IL/VF 索引列为空，
// 导致后续无法做资源评估。本方法在方案保存后由调用方以 go func 异步触发，
// 用 haiku 整批对照原文+方案逐页编码 AOCI 索引，回填 4 个索引列。
//
// 设计要点：
//   - 使用独立 context.Background()，不受原请求 ctx 取消影响（后台任务需跑完）
//   - 回填前重新 ListCoursewarePages 拿"当前"页，对齐老师可能的方案改动
//   - 复用层1解析 parseAOCIIndexOutput + 编码映射，与教案库索引同构
//   - 全程失败不 panic、不影响课件可用，仅记日志（索引是增强项）
//   - 调用 repository.UpdateCWPageIndexFields 只更新索引列，不碰方案/HTML
//
// v0.44.1 防贴错双保险：
//
//	A. 页数守卫：记录构建提示词时的页数 promptPageCount，回填前重新查当前页，
//	   若页数与 promptPageCount 不一致（说明老师在窗口期改了方案），整体放弃
//	   本次回填，留给夜间轮询补——绝不在不确定对齐时硬写。
//	B. 标题锚点匹配：不靠纯顺序，用 AI 回显的页标题(TT) 与当前页标题匹配定位
//	   具体 page_number；先精确（去空白相等）后模糊（互相包含）；匹配不上则跳过。
//	   宁可留空也不贴错。
//
// 参数 rawText：教案/文档原文（doc 传 docx 全文，ppt 传各页文本拼接）
func (s *CoursewareIndexService) BackfillPageIndexAsync(
	coursewareID string, userID string,
	title string, subject string, grade string, rawText string,
) {
	// 独立后台上下文（不随请求取消而中断）
	ctx := context.Background()

	defer func() {
		// 兜底：后台任务绝不因 panic 影响进程
		if r := recover(); r != nil {
			log.Printf("[courseware_index] 后台补索引 panic 已恢复: cw=%s r=%v", coursewareID, r)
		}
	}()

	// ---- 1. 取当前页面（构建提示词所依据的快照）----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		log.Printf("[courseware_index] 后台补索引-取当前页面失败或为空: cw=%s err=%v", coursewareID, err)
		return
	}
	promptPageCount := len(pages) // 保险A：记录喂给AI的页数

	// ---- 2. 加载层1提示词（AOCI索引字典）----
	dictPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_index")
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-加载索引字典失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 3. 构建"对照原文+方案"的用户提示词 ----
	var sb strings.Builder
	sb.WriteString("请为下面这份课件的每一页生成AOCI压缩索引（仅输出PAGE索引，不需要OVERVIEW概述）。\n")
	sb.WriteString("索引必须对照【教案/文档原文】与【逐页方案】两者：方案告诉你每页讲什么，原文帮你准确判断知识点、认知层次、能力目标。\n\n")
	sb.WriteString(fmt.Sprintf("## 课件基本信息\n- 标题：%s\n- 学科：%s\n- 年级：%s\n- 总页数：%d\n\n", title, subject, grade, len(pages)))

	// 原文（截断避免超上下文）
	srcText := rawText
	if len([]rune(srcText)) > 20000 {
		srcText = string([]rune(srcText)[:20000]) + "\n\n[原文过长，已截取前20000字]"
	}
	sb.WriteString("## 教案/文档原文\n\n")
	sb.WriteString(srcText)
	sb.WriteString("\n\n## 逐页方案（共" + strconv.Itoa(len(pages)) + "页，请严格按此页数与顺序逐页输出索引）\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("第%d页｜标题：%s｜目的：%s｜概要：%s｜交互：%s｜视觉：%s\n",
			p.PageNumber, p.Title, p.Purpose, p.ContentSummary, p.InteractionType, p.VisualFormat))
	}

	sb.WriteString("\n## 输出要求\n")
	sb.WriteString("严格为上面每一页输出一个AOCI索引块，顺序、页数与上面的方案完全一致。\n")
	sb.WriteString("特别注意：每个索引块的 PAGE 行必须原样回显该页的标题（TT字段），以便系统按标题对齐回填。\n")
	sb.WriteString("每块格式如下（PAGE行 + 编码行 + 语义行）：\n")
	sb.WriteString("PAGE:页码|TT:页面标题（与上面方案该页标题保持一致）\n")
	sb.WriteString("KT:知识类型|CG:认知层次(1-6)|IL:交互复杂度(1-5)|VF:视觉形式编码|TG:类型标记\n")
	sb.WriteString("[K]知识目标（这一页要让学生掌握的核心知识点，依据原文准确归纳）\n")
	sb.WriteString("[A]能力目标（这一页训练的能力）\n")
	sb.WriteString("[I]交互说明（这一页的交互/活动形式）\n")
	sb.WriteString("[C]内容要点（这一页承载的具体内容）\n\n")
	sb.WriteString("编码取值说明：\n")
	sb.WriteString("- CG认知层次：1记忆 2理解 3应用 4分析 5评价 6创造\n")
	sb.WriteString("- IL交互复杂度：1静态展示 2点击 3输入 4拖拽 5游戏\n")
	sb.WriteString("- VF视觉形式编码：TH纯文字 IT图文 DG图示 CT图表 TL时间线 CP对比 GL画廊 FM全屏媒体\n\n")
	sb.WriteString("只输出这些PAGE索引块，不要OVERVIEW、不要JSON、不要任何额外说明文字。")

	// ---- 4. 调用 AI（scanner场景，haiku，整批）----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-获取AI配置失败: cw=%s err=%v", coursewareID, err)
		return
	}

	traceCtx := &ai.TraceContext{SceneCode: "scanner", UserID: &userID}
	callResult, err := ai.CallAI(aiCfg, dictPrompt.Content, sb.String(), traceCtx)
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-AI调用失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 5. 解析 AOCI 索引输出 ----
	// 注意：splitOverviewAndPages 兼容"无OVERVIEW、直接PAGE"的输出
	_, pageText := s.splitOverviewAndPages(callResult.Content)
	rawPages, err := s.parseAOCIIndexOutput(pageText)
	if err != nil || len(rawPages) == 0 {
		log.Printf("[courseware_index] 后台补索引-解析索引失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 6. 保险A：页数守卫——回填前重查当前页，页数变了就整体放弃 ----
	curPages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(curPages) == 0 {
		log.Printf("[courseware_index] 后台补索引-回填前重查页面失败或为空: cw=%s err=%v", coursewareID, err)
		return
	}
	if len(curPages) != promptPageCount {
		log.Printf("[courseware_index] 后台补索引-页数已变化(喂AI时=%d 当前=%d)，疑似方案被修改，放弃本次回填，留待夜间轮询: cw=%s",
			promptPageCount, len(curPages), coursewareID)
		return
	}

	// ---- 7. 保险B：按标题锚点匹配回填 ----
	// 建立"标题(规整后) → 当前页"映射，标题为空的页不进映射（无法锚定）。
	titleMap := make(map[string]*models.CoursewarePage)
	for _, p := range curPages {
		key := cwNormalizeTitle(p.Title)
		if key != "" {
			// 同名标题仅保留首个（极少见），避免覆盖
			if _, exists := titleMap[key]; !exists {
				titleMap[key] = p
			}
		}
	}

	filled := 0
	matchedPageNums := make(map[int]bool) // 防止两个解析块匹配到同一页
	for _, rp := range rawPages {
		rpTitleKey := cwNormalizeTitle(rp.Title)
		if rpTitleKey == "" {
			continue // AI未回显标题，无法锚定，跳过
		}

		// 先精确匹配
		target, ok := titleMap[rpTitleKey]
		if !ok {
			// 再模糊匹配：互相包含（AI可能轻微改写标题）
			for k, p := range titleMap {
				if matchedPageNums[p.PageNumber] {
					continue
				}
				if strings.Contains(k, rpTitleKey) || strings.Contains(rpTitleKey, k) {
					target = p
					ok = true
					break
				}
			}
		}
		if !ok || target == nil {
			continue // 匹配不上，宁可留空也不贴错
		}
		if matchedPageNums[target.PageNumber] {
			continue // 该页已被其它块匹配过，跳过
		}

		cg := cwClamp(rp.CG, 1, 6)
		il := cwClamp(rp.IL, 1, 5)
		vf := cwNormalizeVF(rp.VF)
		if err := repository.UpdateCWPageIndexFields(ctx, coursewareID, target.PageNumber, rp.RawIndex, cg, il, vf); err != nil {
			log.Printf("[courseware_index] 后台补索引-回填第%d页失败: cw=%s err=%v", target.PageNumber, coursewareID, err)
			continue
		}
		matchedPageNums[target.PageNumber] = true
		filled++
	}

	log.Printf("[courseware_index] 后台补索引完成: cw=%s 当前页=%d 解析索引=%d 标题匹配回填=%d model=%s tokens=%d",
		coursewareID, len(curPages), len(rawPages), filled, callResult.ModelUsed, callResult.TokensUsed)
}

// cwNormalizeTitle 规整页标题用于锚点匹配：去首尾空白 + 去全部内部空白
// （AI回显标题可能在空格/标点上有细微差异，做轻量归一以提高匹配率）
func cwNormalizeTitle(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	// 去掉所有空白字符（空格/制表/换行），保留其余原貌
	var b strings.Builder
	for _, r := range t {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\u3000' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// cwNormalizeVF 规整视觉形式编码：非法/空值归一为 TH（纯文字）
// 仅接受 8 个合法编码，避免非法值进库导致前端映射不到
func cwNormalizeVF(vf string) string {
	v := strings.ToUpper(strings.TrimSpace(vf))
	switch v {
	case "TH", "IT", "DG", "CT", "TL", "CP", "GL", "FM":
		return v
	default:
		return "TH"
	}
}
