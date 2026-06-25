package services

// courseware_index_refine.go — 课件方案 AI 修改（RefineIndex）
//
// 从 courseware_index_service.go 拆出（超 600 行红线拆分）。
// 与主文件同 package 同 *CoursewareIndexService 接收器，复用主文件的
// broadcastError/parseSchemeJSON/saveAndBroadcast/extractLessonPlanContent
// 及包级 cwClamp/cwTruncate/cwRecommendPageRange/DocUploadDir，调用方零改动。
//
// 含 docx 原文读取（readDocxFullText/readAllFromReader）——RefineIndex 在 doc 来源
// 时内部读 docx，不调 PPT/Doc 服务以避免循环依赖，故 archive/zip + encoding/xml +
// path/filepath 这三个 import 随本块迁来本文件（主文件已移除）。
//
// v0.43 修复（RefineIndex 修改方案不看原文/doc来源被硬拦）：
//   - 去除 LessonPlanID==nil 硬拦截，按 source_type 注入原文上下文
//     （lesson_plan→教案正文；doc_upload→重读docx原文；其余→课件基本信息）
//   - 修改提示词补页数下限约束，避免修改后被概括成极少页

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== v136新增：AI修改方案 ====================

// RefineIndex 根据用户反馈修改课件方案（异步执行，通过SSE推送进度）
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

	// v198：解析操作者所属学校ID，供模型境内/境外分流判定
	refSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "courseware_scheme", UserID: &userID, SchoolID: schoolIDPtr(refSchoolID)}
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

	// ---- 6. 构建新的CoursewarePage列表（尽量保留原有页面的层1索引信息）----
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

	// ---- 7. 保存并广播（保留原有概述不变）----
	return s.saveAndBroadcast(ctx, coursewareID, cw.IndexOverview, newPages)
}

// buildRefineSourceContext 为"修改方案"按课件来源装配 标题/学科/年级 与原文上下文
// 返回 (title, subject, grade, sourceContext)
//   - lesson_plan：取教案，sourceContext 取教案正文
//   - doc_upload  ：取课件，sourceContext 读取docx原文
//   - 其它(topic/ppt/3d/html)：取课件基本信息，sourceContext 为空
//
// 本函数在 index 服务内部完成 docx 读取，不调用 PPT/Doc 服务，避免循环依赖。
func (s *CoursewareIndexService) buildRefineSourceContext(ctx context.Context, cw *models.Courseware) (string, string, string, string) {
	title := cw.Title
	subject := cw.Subject
	grade := cw.Grade
	sourceContext := ""

	switch cw.SourceType {
	case models.CWSourceLessonPlan:
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

	return title, subject, grade, sourceContext
}

// readDocxFullText 读取.docx文件的全部正文文本（index服务内部独立实现，无服务间依赖）
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

// readAllFromReader 读取io.Reader全部内容（避免仅为一次使用引入io包的额外import面）
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
			if n == 0 {
				return buf, err
			}
		}
		if n == 0 {
			return buf, nil
		}
	}
}
