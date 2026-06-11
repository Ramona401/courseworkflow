package services

// courseware_gen_helpers.go — 课件HTML生成辅助函数集
//
// 本文件包含：
//   - assembleFullPage：后端硬拼接导航栏+内容区为完整页面
//   - normalizeRootCanvas / enforceCanvasDecls：根容器画布契约归一化
//     （强制压住1920×1080+剥除AI误加的transform+补cw-page类，批量/重生/微调共用）
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
// 批次3小修5：appendSamplePageReference 背景措辞改为来源感知——
//   老师在图库选了背景时，不再出现"必须沿用样例官方背景图"的矛盾指令，
//   样例参考bullet与【本页背景(硬性要求)】段均按实际背景来源动态措辞。

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
		// 闸门：先归一化根容器（强制1920×1080、剥除AI误加的transform、补cw-page），再插入导航栏
		contentTrimmed = normalizeRootCanvas(contentTrimmed)
		// 找到第一个 > 的位置（外层div开标签结束）
		firstGT := strings.Index(contentTrimmed, ">")
		if firstGT > 0 {
			// 在外层div开标签后插入导航栏
			return s.applyTemplateBackground(contentTrimmed[:firstGT+1]+"\n"+nav+"\n"+contentTrimmed[firstGT+1:], tplInfo, pageNum)
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
	return s.applyTemplateBackground(sb.String(), tplInfo, pageNum)
}

// ==================== 画布契约闸门（批量/重生/微调共用） ====================

// 根容器开标签解析用正则（只作用于最外层<div>的开标签，不碰正文）
var (
	cwRootClassRe = regexp.MustCompile(`(?i)class\s*=\s*"([^"]*)"`)
	cwRootStyleRe = regexp.MustCompile(`(?i)style\s*=\s*"([^"]*)"`)
	cwTransformRe = regexp.MustCompile(`(?i)transform\s*:[^;"]*;?`)
)

// normalizeRootCanvas 归一化最外层div的开标签，压住1920×1080画布契约
//   - 确保 class 含 cw-page（缩放与样式钩子依赖此类名）
//   - 确保 style 含 width:1920px;height:1080px;overflow:hidden;position:relative
//   - 剥除 style 中的 transform 声明（防止AI给根容器加scale导致整页缩放/背景比例收缩变形）
//
// 只动第一个 <div 开标签，正文一律不碰；非<div开头或无法解析时原样返回。
// 用于：assembleFullPage（批量生成+单页重生）与 RefinePage（单页微调）。
func normalizeRootCanvas(html string) string {
	trimmed := strings.TrimSpace(html)
	if !strings.HasPrefix(strings.ToLower(trimmed), "<div") {
		return html
	}
	gt := strings.Index(trimmed, ">")
	if gt < 0 {
		return html
	}
	openTag := trimmed[:gt+1] // 含结尾 >
	rest := trimmed[gt+1:]

	// ---- 1. 确保 class 含 cw-page ----
	if m := cwRootClassRe.FindStringSubmatch(openTag); m != nil {
		classes := m[1]
		if !strings.Contains(classes, "cw-page") {
			newClass := strings.TrimSpace(classes + " cw-page")
			openTag = cwRootClassRe.ReplaceAllLiteralString(openTag, `class="`+newClass+`"`)
		}
	} else {
		// 没有 class 属性，在 <div 后插入
		openTag = strings.Replace(openTag, "<div", `<div class="cw-page"`, 1)
	}

	// ---- 2. 规范 style：去 transform + 强制 1920×1080/overflow/position ----
	if m := cwRootStyleRe.FindStringSubmatch(openTag); m != nil {
		style := m[1]
		style = cwTransformRe.ReplaceAllLiteralString(style, "") // 剥除根容器上的 transform
		style = enforceCanvasDecls(style)
		openTag = cwRootStyleRe.ReplaceAllLiteralString(openTag, `style="`+style+`"`)
	} else {
		// 没有 style 属性，补一个最小画布 style
		openTag = strings.Replace(openTag, "<div", `<div style="width:1920px;height:1080px;overflow:hidden;position:relative"`, 1)
	}

	return openTag + rest
}

// enforceCanvasDecls 在style声明串中强制写入画布契约声明（已存在则覆盖值，不存在则追加）
// 保留其余声明的原有顺序，最后追加被强制的声明
func enforceCanvasDecls(style string) string {
	seen := map[string]string{}
	var order []string
	for _, d := range strings.Split(style, ";") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		idx := strings.Index(d, ":")
		if idx < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(d[:idx]))
		val := strings.TrimSpace(d[idx+1:])
		if prop == "" {
			continue
		}
		if _, ok := seen[prop]; !ok {
			order = append(order, prop)
		}
		seen[prop] = val
	}

	// 固定顺序强制画布声明
	forceOrder := []string{"width", "height", "overflow"}
	forceVal := map[string]string{"width": "1920px", "height": "1080px", "overflow": "hidden"}
	for _, prop := range forceOrder {
		if _, ok := seen[prop]; !ok {
			order = append(order, prop)
		}
		seen[prop] = forceVal[prop]
	}
	// position 缺失则补 relative（已有则尊重AI写的absolute/relative等）
	if _, ok := seen["position"]; !ok {
		order = append(order, "position")
		seen["position"] = "relative"
	}

	parts := make([]string, 0, len(order))
	for _, prop := range order {
		parts = append(parts, prop+":"+seen[prop])
	}
	return strings.Join(parts, ";")
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

	// 任务2（分页参考注入）：按页型就近选取所选模板的官方样例页注入提示词，传递布局/装饰/背景质感
	s.appendSamplePageReference(&sb, tplInfo, page, pageNum, totalPages)

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

	// 任务2（分页参考注入）：按页型就近选取所选模板的官方样例页注入提示词，传递布局/装饰/背景质感
	s.appendSamplePageReference(&sb, tplInfo, page, pageNum, totalPages)

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
		sb.WriteString("\x60\x60\x60html\n")
		sb.WriteString(code)
		sb.WriteString("\n\x60\x60\x60\n")
	}
	sb.WriteString("\n")
}

// ==================== 任务2：模板样例页参考注入（分页参考注入） ====================

// cwSampleRefMaxRunes 单个样例页注入提示词的最大字符数（rune计数，防中文截半）。
// 12000上限覆盖4套新模板最长封面样例（小画本10625字符），保证样例嵌入的背景图URL不被截掉。
const cwSampleRefMaxRunes = 12000

// pickSamplePageIndex 按"当前页页型"挑选最匹配的模板样例页下标。
//
// 5页标准模板（sample_pages 固定顺序：封面/学习目标/内容讲解/互动练习/课后作业）：
//   - 第1页 → 封面样例(0)
//   - 第2页 → 学习目标样例(1)
//   - 最后1页 → 课后作业样例(4)
//   - 互动型页（click/drag/game/quiz 或交互复杂度≥4）→ 互动练习样例(3)
//   - 其余 → 内容讲解样例(2)
//
// 非5页模板（旧单页模板/AI提取草稿等）：封面参考第1个样例，其余参考最后1个样例（单样例即同一个）。
// 返回：样例下标 + 页型中文标签（注入提示词时告知AI）。
func pickSamplePageIndex(samplesLen int, page *models.CoursewarePage, pageNum int, totalPages int) (int, string) {
	if samplesLen == 5 {
		switch {
		case pageNum == 1:
			return 0, "封面"
		case pageNum == 2:
			return 1, "学习目标"
		case pageNum == totalPages:
			return 4, "课后作业"
		default:
			it := page.InteractionType
			if it == "click" || it == "drag" || it == "game" || it == "quiz" || page.IdxInteractionLevel >= 4 {
				return 3, "互动练习"
			}
			return 2, "内容讲解"
		}
	}
	// 非5页模板的回退策略
	if pageNum == 1 || samplesLen == 1 {
		return 0, "通用样例"
	}
	return samplesLen - 1, "通用样例"
}

// appendSamplePageReference 把"与本页页型最接近的模板官方样例页HTML"追加进AI生成提示词。
//
// 设计要点（对齐课件UI提升PRD第五节·分页参考注入）：
//   - 样例页是模板作者精心设计的视觉基准（含布局骨架、装饰语言、嵌入的OSS背景图与可读性蒙版做法），
//     注入后AI生成的页面质感向官方样例靠拢，而不是只拿到几个CSS变量"凭空发挥"。
//   - 只注入1页（页型就近匹配），控制提示词体积；超长按rune安全截断。
//   - 模板无样例（SamplePages为空）时静默跳过，行为与旧版完全一致——零回归风险。
//   - 明确约束AI：参考视觉、绝不照抄占位文字；输出结构仍按系统提示词要求（<div>而非样例的<section>）。
//
// 批次3小修5：背景指令统一指向【本页背景(硬性要求)】段，按来源（老师图库选择/模板官方背景）
// 动态措辞，根治"老师选了背景"与"必须沿用样例官方背景"的自相矛盾。
func (s *CoursewareGenService) appendSamplePageReference(
	sb *strings.Builder,
	tplInfo *cwTemplateInfo,
	page *models.CoursewarePage,
	pageNum int, totalPages int,
) {
	if tplInfo == nil || page == nil {
		return
	}
	// 批次1：先解析老师图库选择（三级优先级第一级），样例段与硬约束段共用
	userBgDecls := resolveUserBgDecls(tplInfo, pageNum)
	if len(tplInfo.SamplePages) == 0 {
		// 模板无样例页（旧/个人模板）：无样例可参考；老师选了背景仍须硬约束告知AI
		if userBgDecls != "" {
			s.appendBackgroundHardRule(sb, userBgDecls, "老师在背景图库中为本课件选定的背景图")
		}
		return
	}
	idx, label := pickSamplePageIndex(len(tplInfo.SamplePages), page, pageNum, totalPages)
	if idx < 0 || idx >= len(tplInfo.SamplePages) {
		return
	}
	sample := strings.TrimSpace(tplInfo.SamplePages[idx])
	if sample == "" {
		return
	}

	// rune 安全截断，防中文截半
	truncated := false
	runes := []rune(sample)
	if len(runes) > cwSampleRefMaxRunes {
		sample = string(runes[:cwSampleRefMaxRunes])
		truncated = true
	}

	sb.WriteString(fmt.Sprintf("## 模板官方样例页参考（页型：%s）\n", label))
	sb.WriteString("下面是所选风格模板中与本页页型最匹配的官方样例页HTML，请把它当作本页的视觉基准：\n")
	sb.WriteString("- 严格沿用其布局骨架、装饰语言、卡片质感、圆角阴影与留白节奏；\n")
	sb.WriteString("- 本页背景以下方【本页背景（硬性要求）】段为准（系统也会在后端强制注入）；样例自带的背景仅作质感参考，若与硬性要求不一致请以硬性要求为准；\n")
	sb.WriteString("- 正文内容必须按上方【本页方案】重写，绝不照抄样例中的占位文字；\n")
	sb.WriteString("- 配色以上方CSS变量为准；样例外层的<section>仅为展示包装，你的输出仍须按系统提示词要求输出<div>结构；\n")
	sb.WriteString("- 若本任务要求只生成内容区，则只参考样例的内容区部分，忽略其整页框架。\n")
	sb.WriteString("\x60\x60\x60html\n")
	sb.WriteString(sample)
	if truncated {
		sb.WriteString("\n<!-- ...样例过长已截断，参考以上部分即可 -->")
	}
	sb.WriteString("\n\x60\x60\x60\n\n")

	// 背景硬约束（来源感知）：三级优先级——老师图库选择 > 模板样例提取 > 无。
	// 把背景从"AI建议"升级为明示的硬性要求，并告知后端 applyTemplateBackground 会兜底强制注入，
	// AI据此按"内容压在背景图上"的前提设计半透明卡片+backdrop-filter，避免注入后可读性冲突。
	bgDecls := userBgDecls
	bgSource := "老师在背景图库中为本课件选定的背景图"
	if bgDecls == "" {
		bgDecls = extractSampleBackgroundDecls(tplInfo.SamplePages, pageNum)
		bgSource = "风格模板自带的官方背景图"
	}
	if bgDecls != "" {
		s.appendBackgroundHardRule(sb, bgDecls, bgSource)
	}
}

// appendBackgroundHardRule 追加【本页背景(硬性要求)】段（小修5抽出的公共函数，来源感知措辞）
// bgSource 为背景来源的中文描述：老师图库选择 / 模板官方背景，措辞随来源变化不再互相矛盾
func (s *CoursewareGenService) appendBackgroundHardRule(sb *strings.Builder, bgDecls string, bgSource string) {
	sb.WriteString("## 本页背景（硬性要求）\n")
	sb.WriteString(fmt.Sprintf("本页根容器背景必须使用下列指定的背景图与蒙版（来源：%s），声明如下。请勿用纯色 var(--cw-bg) 替代（系统也会在后端强制注入此背景）：\n", bgSource))
	sb.WriteString("\x60\x60\x60css\n.cw-page{" + bgDecls + "}\n\x60\x60\x60\n")
	sb.WriteString(fmt.Sprintf("说明：此背景图是%s，属于系统约束\"不依赖外部资源\"的唯一允许例外；", bgSource))
	sb.WriteString("正文卡片请用半透明底+backdrop-filter:blur浮于背景之上，保证文字可读。\n\n")
}

// ==================== 模板官方背景兜底注入（确定性，不依赖AI采纳） ====================

// 样例CSS规则提取正则：.cw-page.cover{...} / .cw-page.inner{...}（[^}]字符类天然跨行）
var (
	cwSampleCoverRuleRe = regexp.MustCompile(`\.cw-page\.cover\s*\{([^}]*)\}`)
	cwSampleInnerRuleRe = regexp.MustCompile(`\.cw-page\.inner\s*\{([^}]*)\}`)
)

// extractSampleBackgroundDecls 从模板样例页CSS中提取官方背景声明（仅background*系列，分号连接）。
// 封面(pageNum==1)取封面样例(下标0)的 .cw-page.cover 规则；其余页取内容样例(下标2)的 .cw-page.inner 规则。
// 规则不存在、或规则内不含 url((即无背景图，旧模板/个人模板/纯色模板)时返回空串 → 调用方零注入零回归。
func extractSampleBackgroundDecls(samplePages []string, pageNum int) string {
	if len(samplePages) == 0 {
		return ""
	}
	var sample string
	var re *regexp.Regexp
	if pageNum == 1 {
		sample = samplePages[0]
		re = cwSampleCoverRuleRe
	} else {
		idx := 2
		if idx >= len(samplePages) {
			idx = len(samplePages) - 1
		}
		sample = samplePages[idx]
		re = cwSampleInnerRuleRe
	}
	m := re.FindStringSubmatch(sample)
	if m == nil {
		return ""
	}
	body := m[1]
	if !strings.Contains(body, "url(") {
		return "" // 无背景图的模板不注入，避免画蛇添足
	}
	var out []string
	for _, d := range strings.Split(body, ";") {
		d = strings.TrimSpace(d)
		if strings.HasPrefix(strings.ToLower(d), "background") {
			out = append(out, d)
		}
	}
	return strings.Join(out, ";")
}

// applyTemplateBackground 后端兜底：把模板官方背景声明强制注入页面根容器。
//
// 背景（本次修复的根因）：系统提示词明确指令"背景色:var(--cw-bg)"+"不依赖外部资源"，
// AI严格服从，导致样例参考里建议性的"可沿用背景图"被无视。本函数把背景从"AI建议"
// 升级为"后端确定性注入"——生成完成后在根容器开标签后插入
// <style>.cw-page{background...!important}</style>，!important 压过AI写在根容器
// 内联样式上的 background:var(--cw-bg)。
//
// 安全性：
//   - 模板无背景声明（旧/个人/纯色模板）→ 原样返回，零回归；
//   - 先过 normalizeRootCanvas 画布闸门（幂等）确保根容器带 .cw-page 类，注入样式按类选择器生效；
//   - TEDNA-TPL-BG 标记做幂等保护，微调/重生回流不重复注入；
//   - 注入的<style>随 html_content 持久化，离线ZIP与edu运行时同样生效。
//
// 调用点：封面预览(GeneratePreviewPages)、批量生成与单页重生(assembleFullPage两个出口)。
func (s *CoursewareGenService) applyTemplateBackgroundOnly(html string, tplInfo *cwTemplateInfo, pageNum int) string {
	if tplInfo == nil || strings.TrimSpace(html) == "" {
		return html
	}
	// 批次1三级优先级：老师图库选择(课件级) > 模板自带背景(样例提取) > 无
	bgDecls := resolveUserBgDecls(tplInfo, pageNum)
	if bgDecls == "" {
		bgDecls = extractSampleBackgroundDecls(tplInfo.SamplePages, pageNum)
	}
	if bgDecls == "" {
		return html
	}
	if strings.Contains(html, "TEDNA-TPL-BG") {
		return html // 已注入过，幂等跳过
	}
	out := normalizeRootCanvas(html)
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(strings.ToLower(trimmed), "<div") {
		return html
	}
	gt := strings.Index(trimmed, ">")
	if gt < 0 {
		return html
	}
	var parts []string
	for _, d := range strings.Split(bgDecls, ";") {
		d = strings.TrimSpace(d)
		if d != "" {
			parts = append(parts, d+" !important")
		}
	}
	styleTag := "<style>/* TEDNA-TPL-BG 模板官方背景兜底注入 */.cw-page{" + strings.Join(parts, ";") + "}</style>"
	cwGenLog.Info("模板官方背景已兜底注入", "page_num", pageNum)
	return trimmed[:gt+1] + styleTag + trimmed[gt+1:]
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

	// 从<div开始提取
	htmlPart := text[divStart:]

	// 简单验证：至少有一个闭合的div
	if !strings.Contains(htmlPart, "</div>") {
		return ""
	}

	// 截断到最后一个 </div>，剥掉 AI 在 HTML 之后追加的解释文字 / 代码围栏残留
	// 修复：微调时 AI 常在 HTML 后补一句"我已经改好了…"或再贴一段代码围栏，
	// 旧逻辑取 text[divStart:] 直到字符串末尾，导致这些尾巴被当作页面内容存库并渲染到预览。
	if last := strings.LastIndex(htmlPart, "</div>"); last >= 0 {
		htmlPart = htmlPart[:last+len("</div>")]
	}

	return strings.TrimSpace(htmlPart)
}

// cwGenStripCodeFences 去除AI输出中的markdown代码块标记
func cwGenStripCodeFences(text string) string {
	// 处理 \x60\x60\x60html 或 \x60\x60\x60 开头
	if strings.HasPrefix(text, "\x60\x60\x60") {
		idx := strings.Index(text, "\n")
		if idx >= 0 {
			text = text[idx+1:]
		}
	}
	text = strings.TrimSpace(text)
	// 处理末尾的 \x60\x60\x60
	if strings.HasSuffix(text, "\x60\x60\x60") {
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
	// 任务2（分页参考注入）：解析模板样例页数组，供生成提示词按页型注入官方样例参考。
	// 解析失败不致命——SamplePages 留空时 appendSamplePageReference 静默跳过，行为退回旧版。
	if tpl.SamplePages != "" {
		_ = json.Unmarshal([]byte(tpl.SamplePages), &info.SamplePages)
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

// ==================== 字体F1：背景+字体统一注入出口 ====================

// applyTemplateBackground 包装函数：原背景兜底注入逻辑整体更名为 applyTemplateBackgroundOnly（零改动），
// 本包装让所有既有调用点（封面预览 / assembleFullPage两个出口 / 单页微调补注 / 单页重生）
// 自动同时获得字体方案注入，无需逐处修改调用方。
// applyFontInjection 自带 TEDNA-TPL-FONT 幂等保护与未选字体跳过逻辑（见 courseware_font_service.go）。
func (s *CoursewareGenService) applyTemplateBackground(html string, tplInfo *cwTemplateInfo, pageNum int) string {
	return s.applyFontInjection(s.applyTemplateBackgroundOnly(html, tplInfo, pageNum), tplInfo)
}
