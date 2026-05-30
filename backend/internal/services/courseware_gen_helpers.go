package services

// courseware_gen_helpers.go — 课件HTML生成辅助函数集
//
// 本文件包含：
//   - assembleFullPage：后端硬拼接导航栏+内容区为完整页面
//   - buildCSSVarsString：CSS变量内联字符串构建
//   - ExtractNavByMarkers / extractNavFallback：导航栏标记提取+兜底
//   - ReplaceNavPageNumbers：页码占位符替换
//   - buildPreviewUserPrompt / buildBatchUserPrompt：AI提示词构建
//   - appendStyleConfig / appendMatchedComponents：提示词片段追加
//   - matchComponentsForPage：组件匹配
//   - extractHTMLFromAIOutput / cwGenStripCodeFences：HTML提取
//   - parseStyleConfig / loadTemplateInfo / defaultTemplateInfo：风格配置
//   - buildMatchedComponentIDs：匹配组件ID列表
//   - resolveLogoAndOrg：Logo和机构名优先级链解析
//
// 拆分自原 courseware_gen_service.go（v142 结构化日志迁移+模块化拆分）

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== P0-1: 后端硬拼接完整页面 ====================

// assembleFullPage 将导航栏模板和AI生成的内容区HTML拼接成完整的1920×1080页面
// 导航栏模板中的 {{PAGE_NUM}} / {{TOTAL_PAGES}} 替换为实际页码
// AI生成的内容区可能是完整的<div>（含最外层），也可能只是内容区片段
func (s *CoursewareGenService) assembleFullPage(contentHTML string, navTemplate string, pageNum int, totalPages int, tplInfo *cwTemplateInfo) string {
	// 替换导航栏中的页码占位符
	nav := strings.ReplaceAll(navTemplate, "{{PAGE_NUM}}", fmt.Sprintf("%d", pageNum))
	nav = strings.ReplaceAll(nav, "{{TOTAL_PAGES}}", fmt.Sprintf("%d", totalPages))

	// 构建CSS变量字符串
	cssVars := s.buildCSSVarsString(tplInfo)

	// 检查AI输出是否已经是完整的1920×1080外层div
	contentTrimmed := strings.TrimSpace(contentHTML)
	if strings.HasPrefix(contentTrimmed, "<div") && strings.Contains(contentTrimmed[:min(200, len(contentTrimmed))], "1920") {
		// AI输出了完整的外层div，需要在其第一个子元素位置插入导航栏
		// 找到第一个 > 的位置（外层div开标签结束）
		firstGT := strings.Index(contentTrimmed, ">")
		if firstGT > 0 {
			// 在外层div开标签后插入导航栏
			return contentTrimmed[:firstGT+1] + "\n" + nav + "\n" + contentTrimmed[firstGT+1:]
		}
	}

	// AI只输出了内容区片段，构建完整的外层div包裹
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="cw-page" style="width:1920px;height:1080px;overflow:hidden;position:relative;background:var(--cw-bg,#F8FAFC);color:var(--cw-text,#1E293B);font-family:var(--cw-font-body,'Inter',system-ui,sans-serif);%s">`, cssVars))
	sb.WriteString("\n")
	sb.WriteString(nav)
	sb.WriteString("\n")
	// 内容区包裹div，从top:80px开始
	sb.WriteString(`<div style="position:absolute;top:80px;left:0;right:0;bottom:0;overflow:hidden">`)
	sb.WriteString("\n")
	sb.WriteString(contentTrimmed)
	sb.WriteString("\n")
	sb.WriteString("</div>")
	sb.WriteString("\n")
	sb.WriteString("</div>")
	return sb.String()
}

// buildCSSVarsString 从模板信息构建CSS变量内联字符串
func (s *CoursewareGenService) buildCSSVarsString(tplInfo *cwTemplateInfo) string {
	if tplInfo == nil || len(tplInfo.CSSVariables) == 0 {
		return ""
	}
	var parts []string
	for k, v := range tplInfo.CSSVariables {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v))
	}
	return strings.Join(parts, ";")
}

// ==================== P0-1: 导航栏标记提取 ====================

// ExtractNavByMarkers 按 <!-- NAV_START --> / <!-- NAV_END --> 标记提取导航栏HTML
// 返回标记之间的内容（不含标记本身），如果没找到标记则尝试兜底提取
func ExtractNavByMarkers(html string) string {
	const startMarker = "<!-- NAV_START -->"
	const endMarker = "<!-- NAV_END -->"

	startIdx := strings.Index(html, startMarker)
	endIdx := strings.Index(html, endMarker)

	if startIdx >= 0 && endIdx > startIdx {
		// 精确提取标记之间的内容
		navContent := html[startIdx+len(startMarker) : endIdx]
		navContent = strings.TrimSpace(navContent)
		if navContent != "" {
			return navContent
		}
	}

	// 兜底：找第一个高度80px的div（兼容AI偶尔忘记标记的情况）
	cwGenLog.Warn("未找到NAV_START/NAV_END标记，尝试兜底提取导航栏")
	return extractNavFallback(html)
}

// extractNavFallback 兜底导航栏提取：找最外层div的第一个子元素（高度约80px）
func extractNavFallback(html string) string {
	// 简单策略：找到第一个包含 height:80px 或 height: 80px 的div块
	// 先找最外层div的开标签结束位置
	outerDivStart := strings.Index(html, "<div")
	if outerDivStart < 0 {
		return ""
	}
	firstGT := strings.Index(html[outerDivStart:], ">")
	if firstGT < 0 {
		return ""
	}
	afterOuterOpen := outerDivStart + firstGT + 1

	// 在外层div内部找第一个子div
	innerContent := html[afterOuterOpen:]
	childDivStart := strings.Index(innerContent, "<div")
	if childDivStart < 0 {
		return ""
	}

	// 提取这个子div的完整HTML（简单的标签匹配）
	childHTML := innerContent[childDivStart:]
	depth := 0
	i := 0
	for i < len(childHTML) {
		if strings.HasPrefix(childHTML[i:], "<div") {
			depth++
			i += 4
		} else if strings.HasPrefix(childHTML[i:], "</div>") {
			depth--
			if depth == 0 {
				return strings.TrimSpace(childHTML[:i+6])
			}
			i += 6
		} else {
			i++
		}
	}
	return ""
}

// ReplaceNavPageNumbers P0-1: 将导航栏HTML中的硬编码页码替换为占位符
// 匹配模式："数字 / 数字" → "{{PAGE_NUM}} / {{TOTAL_PAGES}}"
// 例如："1 / 15" → "{{PAGE_NUM}} / {{TOTAL_PAGES}}"
func ReplaceNavPageNumbers(navHTML string) string {
	// 匹配 "数字 / 数字" 或 "数字/数字" 模式（页码显示）
	re := regexp.MustCompile(`(\d{1,3})\s*/\s*(\d{1,3})`)
	result := re.ReplaceAllString(navHTML, "{{PAGE_NUM}} / {{TOTAL_PAGES}}")
	return result
}

// ==================== 预览模式提示词构建 ====================

// buildPreviewUserPrompt 构建预览模式的AI用户提示词
// 预览模式：AI自由生成导航栏（用NAV标记包裹），生成完整页面
func (s *CoursewareGenService) buildPreviewUserPrompt(
	page *models.CoursewarePage,
	pageNum int, totalPages int,
	tplInfo *cwTemplateInfo,
	logoURL string, orgName string,
	matchedComps []*models.MatchedCWComponent,
	cw *models.Courseware,
) string {
	var sb strings.Builder

	// 课件基本信息
	sb.WriteString("## 课件基本信息\n")
	sb.WriteString(fmt.Sprintf("- 课件标题：%s\n", cw.Title))
	sb.WriteString(fmt.Sprintf("- 学科：%s\n", cw.Subject))
	sb.WriteString(fmt.Sprintf("- 年级：%s\n", cw.Grade))
	sb.WriteString(fmt.Sprintf("- 当前页码：第 %d 页 / 共 %d 页\n", pageNum, totalPages))
	sb.WriteString("\n")

	// 页面方案
	sb.WriteString("## 本页方案\n")
	sb.WriteString(fmt.Sprintf("- 页面标题：%s\n", page.Title))
	sb.WriteString(fmt.Sprintf("- 教学目的：%s\n", page.Purpose))
	sb.WriteString(fmt.Sprintf("- 内容概要：%s\n", page.ContentSummary))
	sb.WriteString(fmt.Sprintf("- 交互类型：%s\n", page.InteractionType))
	sb.WriteString(fmt.Sprintf("- 视觉形式：%s\n", page.VisualFormat))
	if page.MediaRequirements != "" {
		sb.WriteString(fmt.Sprintf("- 多媒体需求：%s\n", page.MediaRequirements))
	}
	sb.WriteString(fmt.Sprintf("- 预估复杂度：%d/5\n", page.EstimatedComplexity))
	sb.WriteString("\n")

	// 封面页提示
	sb.WriteString("⚠️ 这是封面页（第1页），请生成大标题居中的封面设计，突出课件标题、学科年级和机构品牌。\n\n")

	// 导航栏配置（预览模式：AI自由生成导航栏，用标记包裹）
	sb.WriteString("## 导航栏配置\n")
	sb.WriteString("请生成一个80px高的导航栏，并用 <!-- NAV_START --> 和 <!-- NAV_END --> 标记包裹。\n")
	if logoURL != "" {
		sb.WriteString(fmt.Sprintf("- Logo图片URL：%s （用<img src=\"%s\" style=\"max-height:32px;max-width:32px;object-fit:contain;border-radius:6px\">）\n", logoURL, logoURL))
	} else {
		firstChar := "L"
		if orgName != "" {
			runes := []rune(orgName)
			firstChar = string(runes[0])
		}
		sb.WriteString(fmt.Sprintf("- 无Logo图片，使用首字母方块：%s（用主色背景+白色文字的圆角方块）\n", firstChar))
	}
	if orgName != "" {
		sb.WriteString(fmt.Sprintf("- 机构名称：%s\n", orgName))
	}
	sb.WriteString(fmt.Sprintf("- 页码显示：%d / %d\n", pageNum, totalPages))
	sb.WriteString("- 导航栏样式要求：左侧Logo+机构名，右侧页码，底部1px分隔线，背景使用风格模板主色调\n")
	sb.WriteString("\n")

	// 风格配置
	s.appendStyleConfig(&sb, tplInfo)

	// 参考组件
	s.appendMatchedComponents(&sb, matchedComps)

	sb.WriteString("请根据以上信息生成本页的完整HTML代码。严格遵守系统提示词中的画布规格(1920×1080)和字号硬约束。\n")
	sb.WriteString("导航栏必须用 <!-- NAV_START --> 和 <!-- NAV_END --> 标记包裹。\n")

	return sb.String()
}

// ==================== 批量模式提示词构建 ====================

// buildBatchUserPrompt 构建批量生成模式的AI用户提示词
// 批量模式：AI只生成内容区HTML（不含导航栏），后端自动拼接导航栏
func (s *CoursewareGenService) buildBatchUserPrompt(
	page *models.CoursewarePage,
	pageNum int, totalPages int,
	tplInfo *cwTemplateInfo,
	logoURL string, orgName string,
	matchedComps []*models.MatchedCWComponent,
	cw *models.Courseware,
) string {
	var sb strings.Builder

	// 课件基本信息
	sb.WriteString("## 课件基本信息\n")
	sb.WriteString(fmt.Sprintf("- 课件标题：%s\n", cw.Title))
	sb.WriteString(fmt.Sprintf("- 学科：%s\n", cw.Subject))
	sb.WriteString(fmt.Sprintf("- 年级：%s\n", cw.Grade))
	sb.WriteString(fmt.Sprintf("- 当前页码：第 %d 页 / 共 %d 页\n", pageNum, totalPages))
	sb.WriteString("\n")

	// 页面方案
	sb.WriteString("## 本页方案\n")
	sb.WriteString(fmt.Sprintf("- 页面标题：%s\n", page.Title))
	sb.WriteString(fmt.Sprintf("- 教学目的：%s\n", page.Purpose))
	sb.WriteString(fmt.Sprintf("- 内容概要：%s\n", page.ContentSummary))
	sb.WriteString(fmt.Sprintf("- 交互类型：%s\n", page.InteractionType))
	sb.WriteString(fmt.Sprintf("- 视觉形式：%s\n", page.VisualFormat))
	if page.MediaRequirements != "" {
		sb.WriteString(fmt.Sprintf("- 多媒体需求：%s\n", page.MediaRequirements))
	}
	sb.WriteString(fmt.Sprintf("- 预估复杂度：%d/5\n", page.EstimatedComplexity))
	sb.WriteString("\n")

	// 页面位置提示
	if pageNum == 2 {
		sb.WriteString("💡 这是目标页（第2页），请生成清晰的学习目标列表。\n\n")
	} else if pageNum == totalPages-1 {
		sb.WriteString("💡 这是小结页（倒数第2页），请生成本节要点回顾/思维导图式总结。\n\n")
	} else if pageNum == totalPages {
		sb.WriteString("💡 这是作业页（最后1页），请生成课后任务和拓展思考。\n\n")
	}

	// P0-1核心：告诉AI只生成内容区，不含导航栏
	sb.WriteString("## ⚠️ 仅生成内容区（导航栏由系统自动拼接）\n")
	sb.WriteString("重要：你只需生成内容区HTML，不要生成导航栏。导航栏（顶部80px）由系统自动添加。\n")
	sb.WriteString("你的内容区可用高度为1000px（1080-80导航栏）。\n")
	sb.WriteString("请输出一个完整的1920×1080最外层div，内部直接放内容区（从top:80px开始），不要放任何导航栏元素。\n")
	sb.WriteString("\n")

	// 风格配置
	s.appendStyleConfig(&sb, tplInfo)

	// 参考组件
	s.appendMatchedComponents(&sb, matchedComps)

	sb.WriteString("请根据以上信息生成本页的内容区HTML代码。严格遵守系统提示词中的画布规格和字号硬约束。\n")
	sb.WriteString("不要生成导航栏，系统会自动添加。\n")

	return sb.String()
}

// ==================== 公共提示词片段 ====================

// appendStyleConfig 追加风格配置到提示词
func (s *CoursewareGenService) appendStyleConfig(sb *strings.Builder, tplInfo *cwTemplateInfo) {
	sb.WriteString("## 风格配置\n")
	sb.WriteString(fmt.Sprintf("- 风格模板：%s（%s）\n", tplInfo.Name, tplInfo.StyleCategory))
	sb.WriteString("- CSS变量（必须使用）：\n")
	for k, v := range tplInfo.CSSVariables {
		sb.WriteString(fmt.Sprintf("  %s: %s;\n", k, v))
	}
	sb.WriteString("\n")
}

// appendMatchedComponents 追加参考组件到提示词
func (s *CoursewareGenService) appendMatchedComponents(sb *strings.Builder, matchedComps []*models.MatchedCWComponent) {
	if len(matchedComps) == 0 {
		return
	}
	sb.WriteString("## 参考组件（可参考其布局和交互模式，但必须用风格模板的配色）\n")
	for i, comp := range matchedComps {
		sb.WriteString(fmt.Sprintf("\n### 参考组件 %d：%s（%s）\n", i+1, comp.Name, comp.ComponentType))
		// 只注入代码片段的前2000字符，避免提示词过长
		code := comp.CodeContent
		if len(code) > 2000 {
			code = code[:2000] + "\n<!-- ... 代码截断 -->"
		}
		sb.WriteString("```html\n")
		sb.WriteString(code)
		sb.WriteString("\n```\n")
	}
	sb.WriteString("\n")
}

// ==================== 组件匹配 ====================

// matchComponentsForPage 为单页匹配最合适的课件组件（top 2）
func (s *CoursewareGenService) matchComponentsForPage(ctx context.Context, page *models.CoursewarePage, subject string, grade string) []*models.MatchedCWComponent {
	req := &models.MatchCWComponentsRequest{
		SubjectScope:     subject,
		GradeScope:       grade,
		InteractionLevel: page.IdxInteractionLevel,
		VisualFormat:     page.IdxVisualFormat,
		Limit:            2,
	}
	// 如果页面没有索引维度，用方案字段推断
	if req.InteractionLevel <= 0 {
		req.InteractionLevel = page.EstimatedComplexity
	}

	matched, err := repository.MatchCWComponents(ctx, req)
	if err != nil {
		cwGenLog.Warn("组件匹配失败", "page_num", page.PageNumber, "error", err)
		return nil
	}
	return matched
}

// ==================== HTML提取 ====================

// extractHTMLFromAIOutput 从AI输出中提取HTML代码
// AI可能输出markdown代码块包裹或直接HTML
func (s *CoursewareGenService) extractHTMLFromAIOutput(aiOutput string) string {
	text := strings.TrimSpace(aiOutput)
	if text == "" {
		return ""
	}

	// 去除markdown代码块标记
	text = cwGenStripCodeFences(text)

	// 查找第一个<div开始的位置
	divStart := strings.Index(text, "<div")
	if divStart < 0 {
		// 可能包含完整HTML文档结构
		htmlStart := strings.Index(text, "<html")
		if htmlStart >= 0 {
			return text[htmlStart:]
		}
		// 最后尝试返回全部文本（可能就是纯HTML）
		if strings.Contains(text, "<") && strings.Contains(text, ">") {
			return text
		}
		return ""
	}

	// 从<div开始提取到最后一个匹配的</div>
	htmlPart := text[divStart:]

	// 简单验证：至少有一个闭合的div
	if !strings.Contains(htmlPart, "</div>") {
		return ""
	}

	return strings.TrimSpace(htmlPart)
}

// cwGenStripCodeFences 去除AI输出中的markdown代码块标记
func cwGenStripCodeFences(text string) string {
	// 处理 ```html 或 ``` 开头
	if strings.HasPrefix(text, "```") {
		idx := strings.Index(text, "\n")
		if idx >= 0 {
			text = text[idx+1:]
		}
	}
	text = strings.TrimSpace(text)
	// 处理末尾的 ```
	if strings.HasSuffix(text, "```") {
		text = text[:len(text)-3]
	}
	return strings.TrimSpace(text)
}

// ==================== 风格配置解析 ====================

// parseStyleConfig 从课件的style_config JSON解析风格配置
func (s *CoursewareGenService) parseStyleConfig(styleConfigJSON string) *cwStyleConfig {
	cfg := &cwStyleConfig{}
	if styleConfigJSON == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(styleConfigJSON), cfg)
	return cfg
}

// loadTemplateInfo 加载风格模板的关键信息
func (s *CoursewareGenService) loadTemplateInfo(ctx context.Context, templateID string) (*cwTemplateInfo, error) {
	if templateID == "" {
		return nil, fmt.Errorf("模板ID为空")
	}
	tpl, err := repository.GetCWTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}
	info := &cwTemplateInfo{
		Name:          tpl.Name,
		StyleCategory: tpl.StyleCategory,
		CSSVariables:  make(map[string]string),
		ColorScheme:   make(map[string]string),
	}
	// 解析CSS变量
	if tpl.CSSVariables != "" {
		_ = json.Unmarshal([]byte(tpl.CSSVariables), &info.CSSVariables)
	}
	// 解析配色方案
	if tpl.ColorScheme != "" {
		_ = json.Unmarshal([]byte(tpl.ColorScheme), &info.ColorScheme)
	}
	return info, nil
}

// defaultTemplateInfo 默认风格模板信息（加载失败时兜底）
func (s *CoursewareGenService) defaultTemplateInfo() *cwTemplateInfo {
	return &cwTemplateInfo{
		Name:          "默认风格",
		StyleCategory: "minimalist",
		CSSVariables: map[string]string{
			"--cw-primary":      "#2563EB",
			"--cw-secondary":    "#60A5FA",
			"--cw-bg":           "#F8FAFC",
			"--cw-text":         "#1E293B",
			"--cw-accent":       "#F59E0B",
			"--cw-radius":       "12px",
			"--cw-shadow":       "0 4px 24px rgba(0,0,0,0.06)",
			"--cw-font-heading": "'Inter',system-ui,sans-serif",
			"--cw-font-body":    "'Inter',system-ui,sans-serif",
		},
		ColorScheme: map[string]string{
			"primary": "#2563EB", "secondary": "#60A5FA",
			"background": "#F8FAFC", "text": "#1E293B", "accent": "#F59E0B",
		},
	}
}

// ==================== 辅助函数 ====================

// buildMatchedComponentIDs 构建匹配组件ID的JSON数组
func (s *CoursewareGenService) buildMatchedComponentIDs(comps []*models.MatchedCWComponent) string {
	if len(comps) == 0 {
		return ""
	}
	ids := make([]string, len(comps))
	for i, c := range comps {
		ids[i] = c.ID
	}
	data, _ := json.Marshal(ids)
	return string(data)
}

// resolveLogoAndOrg 解析课件生成时的Logo和机构名（优先级链）
// 优先级：课件手动上传 > 学校Logo > 区域Logo > 无
func (s *CoursewareGenService) resolveLogoAndOrg(ctx context.Context, cw *models.Courseware, styleCfg *cwStyleConfig) (string, string) {
	logoURL := cw.LogoURL
	if logoURL == "" {
		logoURL = styleCfg.LogoURL
	}
	orgName := cw.OrgName
	if orgName == "" {
		orgName = styleCfg.OrgName
	}

	// 如果课件没有Logo，尝试从用户所属学校获取
	if logoURL == "" {
		// 查用户所在的学校
		school, err := repository.GetSchoolByAdminUserID(ctx, cw.UserID)
		if err == nil && school != nil {
			if school.LogoURL != "" {
				logoURL = school.LogoURL
			}
			if orgName == "" {
				orgName = school.Name
			}
			// 如果学校也没有Logo，尝试从所属区域获取
			if logoURL == "" && school.ParentID != nil && *school.ParentID != "" {
				region, rErr := repository.GetOrganizationByID(ctx, *school.ParentID)
				if rErr == nil && region != nil && region.LogoURL != "" {
					logoURL = region.LogoURL
				}
			}
		}
	}

	return logoURL, orgName
}
