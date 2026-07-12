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

	// ---- 6. 构建新的CoursewarePage列表（双重保留策略：问题页码集+字段比对）----
	// 双重保留策略确保最高保留率：
	//   策略A（优先）：前端修正指令含"⚠️ 【严格要求】仅修改以下页面"时，从中提取问题页码集，
	//     不在集合中的页直接保留全部旧数据（含HTML），无需字段比对，不受AI微调措辞影响。
	//   策略B（兜底）：无法提取页码集时（手动输入修改意见等场景），回退到三字段精确比对。
	// 两策略对问题页的处理相同：用AI新方案字段，但不保留HTML（需重新生成）。

	// 从修正指令中提取问题页码集（前端 buildFixInstruction 格式："仅修改以下页面的方案：P3、P5、P8"）
	affectedPages := extractAffectedPageNums(feedback)
	usePageNumStrategy := len(affectedPages) > 0 // 策略A可用

	oldPageMap := make(map[int]*models.CoursewarePage)
	for _, p := range pages {
		oldPageMap[p.PageNumber] = p
	}

	var newPages []*models.CoursewarePage
	htmlPreserved := 0 // 统计：保留了HTML的页数
	htmlCleared := 0   // 统计：方案变更需重新生成的页数
	for _, sc := range schemes {
		pn := sc.PageNumber
		if pn <= 0 {
			pn = len(newPages) + 1
		}
		newTitle := strings.TrimSpace(sc.Title)
		newPurpose := strings.TrimSpace(sc.Purpose)
		newSummary := strings.TrimSpace(sc.ContentSummary)

		// 检查旧页是否存在
		oldPage, hasOld := oldPageMap[pn]

		// 判定该页是否属于"不需要改"——满足任一即为"应保留"：
		//   策略A：有页码集且该页不在集合中
		//   策略B：无页码集且三字段精确相同
		shouldPreserve := false
		if hasOld && strings.TrimSpace(oldPage.HTMLContent) != "" {
			if usePageNumStrategy {
				// 策略A：不在问题页码集中 → 强制保留（AI即使微调了措辞也用旧数据）
				_, isAffected := affectedPages[pn]
				shouldPreserve = !isAffected
			} else {
				// 策略B兜底：三字段精确比对
				shouldPreserve = oldPage.Title == newTitle &&
					oldPage.Purpose == newPurpose &&
					oldPage.ContentSummary == newSummary
			}
		}

		if shouldPreserve && hasOld {
			// 保留旧页全部数据（含HTML、方案字段、状态、元数据）——逐字不动
			page := &models.CoursewarePage{
				CoursewareID:        coursewareID,
				PageNumber:          pn,
				Title:               oldPage.Title,    // 用旧标题（策略A下AI可能微调了措辞，但该页不在问题集，应保留原样）
				Purpose:             oldPage.Purpose,
				ContentSummary:      oldPage.ContentSummary,
				InteractionType:     oldPage.InteractionType,
				VisualFormat:        oldPage.VisualFormat,
				MediaRequirements:   oldPage.MediaRequirements,
				EstimatedComplexity: oldPage.EstimatedComplexity,
				Status:              oldPage.Status,
				HTMLContent:         oldPage.HTMLContent,
				PageIndex:           oldPage.PageIndex,
				IdxCognitiveLevel:   oldPage.IdxCognitiveLevel,
				IdxInteractionLevel: oldPage.IdxInteractionLevel,
				IdxVisualFormat:     oldPage.IdxVisualFormat,
				PlaceholderMap:      oldPage.PlaceholderMap,
				MatchedComponentIDs: oldPage.MatchedComponentIDs,
			}
			newPages = append(newPages, page)
			htmlPreserved++
		} else {
			// 新页 或 问题页 或 方案变了的页 → 用AI新方案，不保留HTML
			page := &models.CoursewarePage{
				CoursewareID:        coursewareID,
				PageNumber:          pn,
				Title:               newTitle,
				Purpose:             newPurpose,
				ContentSummary:      newSummary,
				InteractionType:     strings.TrimSpace(sc.InteractionType),
				VisualFormat:        strings.TrimSpace(sc.VisualFormat),
				MediaRequirements:   strings.TrimSpace(sc.MediaRequirements),
				EstimatedComplexity: cwClamp(sc.EstimatedComplexity, 1, 5),
				Status:              models.CWPageStatusPending,
			}
			// 层1索引仍保留（与方案变更无关）
			if hasOld {
				page.PageIndex = oldPage.PageIndex
				page.IdxCognitiveLevel = oldPage.IdxCognitiveLevel
				page.IdxInteractionLevel = oldPage.IdxInteractionLevel
				page.IdxVisualFormat = oldPage.IdxVisualFormat
				if strings.TrimSpace(oldPage.HTMLContent) != "" {
					htmlCleared++
				}
			}
			if page.InteractionType == "" {
				page.InteractionType = "static"
			}
			if page.VisualFormat == "" {
				page.VisualFormat = "text_heavy"
			}
			newPages = append(newPages, page)
		}
	}

	// 重新编号（确保连续）
	for i, p := range newPages {
		p.PageNumber = i + 1
	}

	log.Printf("[courseware_index] RefineIndex完成: cw=%s source=%s oldPages=%d newPages=%d htmlPreserved=%d htmlCleared=%d pageNumStrategy=%v model=%s tokens=%d feedback=%s",
		coursewareID, cw.SourceType, len(pages), len(newPages), htmlPreserved, htmlCleared, usePageNumStrategy, callResult.ModelUsed, callResult.TokensUsed, cwTruncate(feedback, 50))

	// ---- 7. 保存并广播（保留原有概述不变）----
	if err := s.saveAndBroadcast(ctx, coursewareID, cw.IndexOverview, newPages); err != nil {
		return err
	}

	// ---- 8. 状态智能回退：如果有页面HTML被清空（方案变更），需要回退状态让老师重新生成 ----
	// 规则：只要有任何一页的HTML因方案变更被清空，就把课件status回退到generating
	//（保留导航栏nav_template_html不丢，老师直接在Step4对变更页重新生成，无需从头来）。
	// 如果全部页HTML都保留了（方案微调未影响任何页），则不回退，保持原status。
	if htmlCleared > 0 {
		prevStatus := cw.Status
		if err := repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusGenerating); err != nil {
			log.Printf("[courseware_index] RefineIndex状态回退失败: cw=%s err=%v", coursewareID, err)
		} else {
			log.Printf("[courseware_index] RefineIndex状态回退: cw=%s %s→generating (htmlCleared=%d, htmlPreserved=%d)",
				coursewareID, prevStatus, htmlCleared, htmlPreserved)
		}
	}

	return nil
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

// extractAffectedPageNums 从修正指令中提取问题页码集合。
// 前端 buildFixInstruction 格式："仅修改以下页面的方案：P3、P5、P8。"
// 返回 map[int]struct{} 页码集合，空集合表示未找到（回退策略B）。
func extractAffectedPageNums(feedback string) map[int]struct{} {
	result := make(map[int]struct{})

	// 查找"仅修改以下页面"标记
	marker := "仅修改以下页面"
	idx := strings.Index(feedback, marker)
	if idx < 0 {
		return result
	}

	// 从标记位置往后取到句号或换行，在这段内找所有 P+数字
	segment := feedback[idx:]
	if dotIdx := strings.IndexAny(segment, "。\n"); dotIdx > 0 {
		segment = segment[:dotIdx]
	}

	// 提取 P 后面的数字（P3、P5、P8 等）
	for i := 0; i < len(segment)-1; i++ {
		if segment[i] == 'P' || segment[i] == 'p' {
			j := i + 1
			num := 0
			for j < len(segment) && segment[j] >= '0' && segment[j] <= '9' {
				num = num*10 + int(segment[j]-'0')
				j++
			}
			if num > 0 {
				result[num] = struct{}{}
			}
		}
	}

	return result
}
