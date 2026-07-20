package services

// courseware_template_master.go — 模板首页硬母版生成。
//
// 核心原则：
//   1. sample_pages[0] 是不可重写的首页母版；
//   2. AI只能返回“可见文字槽位替换表”，不能返回或重写整页HTML；
//   3. 模板已有导航栏时原样保留其DOM、CSS、高度和排版；
//   4. 模板确实没有导航栏时，才由AI生成一段与模板视觉语言一致的导航栏；
//   5. 页码只替换数字，原页码DOM和样式由courseware_nav_pagenum.go保留。

import (
	"context"
	"encoding/json"
	"fmt"
	htmlstd "html"
	"regexp"
	"strings"
	"unicode"

	"tedna/internal/ai"
	"tedna/internal/models"
)

const (
	cwMasterMaxSlots       = 120
	cwMasterMaxSlotRunes   = 260
	cwMasterMaxNewRunes    = 420
	cwMasterNavSampleRunes = 9000
)

type cwMasterSegment struct {
	Raw  string
	Slot int
}

type cwMasterSlotView struct {
	Slot    int    `json:"slot"`
	Text    string `json:"text"`
	Context string `json:"context,omitempty"`
}

type cwMasterReplacement struct {
	Slot int    `json:"slot"`
	Text string `json:"text"`
}

type cwMasterReplacementEnvelope struct {
	Replacements []cwMasterReplacement `json:"replacements"`
}

var cwTemplateNavLikeOpenRe = regexp.MustCompile(
	`(?is)<(nav|div)\b[^>]*(?:class|id)\s*=\s*["'][^"']*(?:nav|navbar|main-header|page-header|topbar|top-bar|ct-top|header)[^"']*["'][^>]*>`,
)

// buildTemplateMasterPreview 使用模板第1页作为不可变母版生成封面。
func (s *CoursewareGenService) buildTemplateMasterPreview(
	ctx context.Context,
	cw *models.Courseware,
	page *models.CoursewarePage,
	tplInfo *cwTemplateInfo,
	logoURL string,
	orgName string,
	totalPages int,
	userID string,
	schoolID string,
	aiCfg *ai.EffectiveConfig,
) (string, string, int, error) {
	if cw == nil || page == nil || tplInfo == nil {
		return "", "", 0, fmt.Errorf("模板母版生成参数不完整")
	}
	if len(tplInfo.SamplePages) == 0 {
		return "", "", 0, fmt.Errorf("所选模板没有首页样例")
	}

	masterHTML := strings.TrimSpace(tplInfo.SamplePages[0])
	if masterHTML == "" {
		return "", "", 0, fmt.Errorf("所选模板首页样例为空")
	}

	modelUsed := ""
	totalTokens := 0

	// 先识别模板是否已经定义了真实导航栏。
	navStart, navEnd, hasTemplateNav := findTemplateNavRegion(masterHTML)
	if hasTemplateNav {
		if !templateHasNavMarkers(masterHTML) {
			masterHTML = wrapTemplateNavRegion(masterHTML, navStart, navEnd)
		}
	} else {
		// 模板确实没有导航栏时，AI只补导航栏片段，不重写母版。
		generatedNav, navModel, navTokens, navErr := s.generateTemplateMatchedNav(
			ctx,
			cw,
			page,
			tplInfo,
			logoURL,
			orgName,
			totalPages,
			userID,
			schoolID,
			aiCfg,
		)
		if navErr != nil {
			return "", "", 0, fmt.Errorf("模板未定义导航栏，AI补充导航栏失败: %w", navErr)
		}
		masterHTML = insertGeneratedNavIntoTemplate(masterHTML, generatedNav)
		modelUsed = navModel
		totalTokens += navTokens
	}

	// AI只读取可见文字槽位并返回替换表。
	segments, slotViews := scanTemplateVisibleText(masterHTML)
	if len(slotViews) > 0 {
		replacements, slotModel, slotTokens, slotErr := s.planTemplateMasterTextReplacements(
			ctx,
			cw,
			page,
			orgName,
			slotViews,
			userID,
			schoolID,
			aiCfg,
		)
		if slotErr != nil {
			return "", "", 0, fmt.Errorf("模板首页文字适配失败: %w", slotErr)
		}

		masterHTML = applyTemplateMasterReplacements(segments, replacements)
		if slotModel != "" {
			modelUsed = slotModel
		}
		totalTokens += slotTokens
	}

	// 对常见历史占位文字做确定性兜底。
	masterHTML = applyCommonTemplateMasterFallbacks(masterHTML, cw, page)

	// 此时必须已经存在可提取导航栏。
	currentNav := ExtractNavByMarkers(masterHTML)
	if strings.TrimSpace(currentNav) == "" {
		return "", "", 0, fmt.Errorf("模板母版未能形成可识别导航栏")
	}

	// 保留模板原有页码DOM，只把数字转换后再注入当前页码。
	navTemplate := StripNavPageNumbers(currentNav)
	previewNav := injectPageNumIntoNav(navTemplate, 1, totalPages)

	updatedHTML, replaced := replaceRefinedNavInPageHTML(masterHTML, previewNav)
	if !replaced {
		return "", "", 0, fmt.Errorf("无法把导航栏写回模板首页母版")
	}

	if modelUsed == "" {
		modelUsed = "template-master-deterministic"
	}

	return updatedHTML, modelUsed, totalTokens, nil
}

func templateHasNavMarkers(source string) bool {
	start := strings.Index(source, cwNavStartMarker)
	end := strings.Index(source, cwNavEndMarker)
	return start >= 0 && end > start
}

// findTemplateNavRegion 识别模板首页已经存在的顶部导航结构。
// 优先级：NAV标记 > header元素 > nav元素/常见导航class或id。
func findTemplateNavRegion(source string) (int, int, bool) {
	if source == "" {
		return 0, 0, false
	}

	startMarker := strings.Index(source, cwNavStartMarker)
	endMarker := strings.Index(source, cwNavEndMarker)
	if startMarker >= 0 && endMarker > startMarker {
		contentStart := startMarker + len(cwNavStartMarker)
		return contentStart, endMarker, true
	}

	lower := strings.ToLower(source)

	if headerStart := strings.Index(lower, "<header"); headerStart >= 0 {
		if headerEnd, ok := findBalancedElementEnd(source, headerStart, "header"); ok {
			return headerStart, headerEnd, true
		}
	}

	if navStart := strings.Index(lower, "<nav"); navStart >= 0 {
		if navEnd, ok := findBalancedElementEnd(source, navStart, "nav"); ok {
			return navStart, navEnd, true
		}
	}

	match := cwTemplateNavLikeOpenRe.FindStringSubmatchIndex(source)
	if len(match) >= 4 {
		elementStart := match[0]
		tagName := strings.ToLower(source[match[2]:match[3]])
		if elementEnd, ok := findBalancedElementEnd(source, elementStart, tagName); ok {
			return elementStart, elementEnd, true
		}
	}

	return 0, 0, false
}

func findBalancedElementEnd(source string, start int, tagName string) (int, bool) {
	if start < 0 || start >= len(source) || tagName == "" {
		return 0, false
	}

	tagRe := regexp.MustCompile(`(?is)</?` + regexp.QuoteMeta(tagName) + `\b[^>]*>`)
	matches := tagRe.FindAllStringIndex(source[start:], -1)
	if len(matches) == 0 {
		return 0, false
	}

	depth := 0
	for _, m := range matches {
		token := source[start+m[0] : start+m[1]]
		lowerToken := strings.ToLower(strings.TrimSpace(token))

		if strings.HasPrefix(lowerToken, "</") {
			depth--
			if depth == 0 {
				return start + m[1], true
			}
			continue
		}

		if strings.HasSuffix(lowerToken, "/>") {
			continue
		}
		depth++
	}

	return 0, false
}

func wrapTemplateNavRegion(source string, start, end int) string {
	if start < 0 || end <= start || end > len(source) {
		return source
	}
	return source[:start] +
		cwNavStartMarker + "\n" +
		source[start:end] + "\n" +
		cwNavEndMarker +
		source[end:]
}

// insertGeneratedNavIntoTemplate 只负责把AI生成的导航栏插入模板母版。
// 不改变母版其余HTML、CSS和布局节点。
func insertGeneratedNavIntoTemplate(source string, nav string) string {
	source = strings.TrimSpace(source)
	nav = strings.TrimSpace(nav)
	if source == "" || nav == "" {
		return source
	}

	block := "\n" + cwNavStartMarker + "\n" + nav + "\n" + cwNavEndMarker + "\n"

	lower := strings.ToLower(source)

	// 完整HTML文档：插到body开标签之后。
	if bodyStart := strings.Index(lower, "<body"); bodyStart >= 0 {
		if gtRel := strings.Index(source[bodyStart:], ">"); gtRel >= 0 {
			insertAt := bodyStart + gtRel + 1
			return source[:insertAt] + block + source[insertAt:]
		}
	}

	// HTML片段：插到最外层section/div开标签之后。
	rootStart := -1
	for _, needle := range []string{"<section", "<div"} {
		if idx := strings.Index(lower, needle); idx >= 0 && (rootStart < 0 || idx < rootStart) {
			rootStart = idx
		}
	}
	if rootStart >= 0 {
		if gtRel := strings.Index(source[rootStart:], ">"); gtRel >= 0 {
			insertAt := rootStart + gtRel + 1
			return source[:insertAt] + block + source[insertAt:]
		}
	}

	return block + source
}

// scanTemplateVisibleText 把HTML拆为原样片段，只给可见文字节点分配槽位。
// style/script/head/svg等内容永远不进入AI，也不会被替换。
func scanTemplateVisibleText(source string) ([]cwMasterSegment, []cwMasterSlotView) {
	segments := make([]cwMasterSegment, 0, 128)
	slots := make([]cwMasterSlotView, 0, 32)

	skipStack := make([]string, 0, 4)
	lastTag := ""
	slotNo := 0

	for i := 0; i < len(source); {
		if source[i] == '<' {
			if strings.HasPrefix(source[i:], "<!--") {
				endRel := strings.Index(source[i:], "-->")
				if endRel < 0 {
					segments = append(segments, cwMasterSegment{Raw: source[i:]})
					break
				}
				end := i + endRel + 3
				raw := source[i:end]
				segments = append(segments, cwMasterSegment{Raw: raw})
				lastTag = cwMasterTruncate(raw, 120)
				i = end
				continue
			}

			endRel := strings.IndexByte(source[i:], '>')
			if endRel < 0 {
				segments = append(segments, cwMasterSegment{Raw: source[i:]})
				break
			}

			end := i + endRel + 1
			raw := source[i:end]
			segments = append(segments, cwMasterSegment{Raw: raw})
			lastTag = cwMasterTruncate(raw, 160)

			tagName, closing, selfClosing := parseHTMLTagName(raw)
			if closing {
				for len(skipStack) > 0 {
					last := skipStack[len(skipStack)-1]
					skipStack = skipStack[:len(skipStack)-1]
					if last == tagName {
						break
					}
				}
			} else if !selfClosing && isTemplateMasterSkipTag(tagName) {
				skipStack = append(skipStack, tagName)
			}

			i = end
			continue
		}

		nextRel := strings.IndexByte(source[i:], '<')
		end := len(source)
		if nextRel >= 0 {
			end = i + nextRel
		}

		raw := source[i:end]
		trimmed := strings.TrimSpace(raw)

		if len(skipStack) == 0 && trimmed != "" && slotNo < cwMasterMaxSlots {
			slotNo++
			segments = append(segments, cwMasterSegment{Raw: raw, Slot: slotNo})
			slots = append(slots, cwMasterSlotView{
				Slot:    slotNo,
				Text:    cwMasterTruncate(htmlstd.UnescapeString(trimmed), cwMasterMaxSlotRunes),
				Context: lastTag,
			})
		} else {
			segments = append(segments, cwMasterSegment{Raw: raw})
		}

		i = end
	}

	return segments, slots
}

func parseHTMLTagName(raw string) (string, bool, bool) {
	t := strings.TrimSpace(raw)
	if len(t) < 3 || t[0] != '<' {
		return "", false, false
	}
	if strings.HasPrefix(t, "<!") || strings.HasPrefix(t, "<?") {
		return "", false, true
	}

	closing := strings.HasPrefix(t, "</")
	selfClosing := strings.HasSuffix(t, "/>")

	start := 1
	if closing {
		start = 2
	}
	for start < len(t) && unicode.IsSpace(rune(t[start])) {
		start++
	}
	end := start
	for end < len(t) {
		c := t[end]
		if !(c == '-' || c == ':' || c == '_' ||
			(c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9')) {
			break
		}
		end++
	}
	if end <= start {
		return "", closing, selfClosing
	}
	return strings.ToLower(t[start:end]), closing, selfClosing
}

func isTemplateMasterSkipTag(tagName string) bool {
	switch tagName {
	case "head", "style", "script", "noscript", "svg":
		return true
	default:
		return false
	}
}

func (s *CoursewareGenService) planTemplateMasterTextReplacements(
	ctx context.Context,
	cw *models.Courseware,
	page *models.CoursewarePage,
	orgName string,
	slots []cwMasterSlotView,
	userID string,
	schoolID string,
	aiCfg *ai.EffectiveConfig,
) ([]cwMasterReplacement, string, int, error) {
	if len(slots) == 0 {
		return nil, "", 0, nil
	}

	slotJSON, err := json.Marshal(slots)
	if err != nil {
		return nil, "", 0, err
	}

	systemPrompt := `你是课件模板首页的文字适配器。

你不能输出HTML，也不能改变模板结构。你只能根据给出的可见文字槽位，决定哪些文字需要替换。

【硬性规则】
1. 只输出JSON对象，不要输出代码围栏和解释。
2. JSON格式必须是：
{"replacements":[{"slot":1,"text":"替换后的纯文字"}]}
3. slot必须来自给定列表。
4. text只能是纯文字，不能包含HTML标签、脚本或CSS。
5. 保留模板的功能性标签、装饰短语、英文栏目名和系列品牌文字，除非它明显是占位示例。
6. 把“主标题、课程标题、副标题、学科、年级、任务主题”等示例文字替换为本课件真实内容。
7. 新文字长度要适配原区域，标题简洁，不能写成长段。
8. 不得删除结构性文字槽位；不需要替换的槽位不要返回。`

	userPrompt := fmt.Sprintf(`## 当前课件
课件标题：%s
本页标题：%s
学科：%s
年级：%s
机构名称：%s
教学目的：%s
内容概要：%s

## 模板可见文字槽位
%s

请返回需要替换的槽位JSON。`,
		cw.Title,
		page.Title,
		cw.Subject,
		cw.Grade,
		orgName,
		page.Purpose,
		page.ContentSummary,
		string(slotJSON),
	)

	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_generate",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if aiErr != nil {
		return nil, "", 0, aiErr
	}

	envelope, parseErr := parseTemplateReplacementEnvelope(result.Content)
	if parseErr != nil {
		return nil, "", 0, parseErr
	}

	validSlots := make(map[int]struct{}, len(slots))
	for _, slot := range slots {
		validSlots[slot.Slot] = struct{}{}
	}

	seen := make(map[int]struct{}, len(envelope.Replacements))
	cleaned := make([]cwMasterReplacement, 0, len(envelope.Replacements))

	for _, item := range envelope.Replacements {
		if _, ok := validSlots[item.Slot]; !ok {
			continue
		}
		if _, dup := seen[item.Slot]; dup {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		text = cwMasterTruncate(text, cwMasterMaxNewRunes)
		seen[item.Slot] = struct{}{}
		cleaned = append(cleaned, cwMasterReplacement{
			Slot: item.Slot,
			Text: text,
		})
	}

	return cleaned, result.ModelUsed, result.TokensUsed, nil
}

func parseTemplateReplacementEnvelope(content string) (*cwMasterReplacementEnvelope, error) {
	cleaned := strings.TrimSpace(content)
	cleaned = cwGenStripCodeFences(cleaned)

	var envelope cwMasterReplacementEnvelope
	if err := json.Unmarshal([]byte(cleaned), &envelope); err == nil {
		return &envelope, nil
	}

	first := strings.Index(cleaned, "{")
	last := strings.LastIndex(cleaned, "}")
	if first >= 0 && last > first {
		if err := json.Unmarshal([]byte(cleaned[first:last+1]), &envelope); err == nil {
			return &envelope, nil
		}
	}

	return nil, fmt.Errorf("AI未返回合法的文字槽位JSON")
}

func applyTemplateMasterReplacements(
	segments []cwMasterSegment,
	replacements []cwMasterReplacement,
) string {
	replacementMap := make(map[int]string, len(replacements))
	for _, item := range replacements {
		replacementMap[item.Slot] = item.Text
	}

	var sb strings.Builder
	for _, segment := range segments {
		if segment.Slot <= 0 {
			sb.WriteString(segment.Raw)
			continue
		}
		newText, ok := replacementMap[segment.Slot]
		if !ok {
			sb.WriteString(segment.Raw)
			continue
		}
		sb.WriteString(replaceVisibleTextPreservingWhitespace(segment.Raw, newText))
	}
	return sb.String()
}

func replaceVisibleTextPreservingWhitespace(raw string, newText string) string {
	leftTrimmed := strings.TrimLeftFunc(raw, unicode.IsSpace)
	prefixLen := len(raw) - len(leftTrimmed)

	rightTrimmed := strings.TrimRightFunc(leftTrimmed, unicode.IsSpace)
	suffix := leftTrimmed[len(rightTrimmed):]

	return raw[:prefixLen] + htmlstd.EscapeString(newText) + suffix
}

func applyCommonTemplateMasterFallbacks(
	source string,
	cw *models.Courseware,
	page *models.CoursewarePage,
) string {
	title := strings.TrimSpace(cw.Title)
	if title == "" {
		title = strings.TrimSpace(page.Title)
	}
	subtitle := strings.TrimSpace(page.ContentSummary)
	if subtitle == "" {
		subtitle = strings.TrimSpace(page.Purpose)
	}

	title = cwMasterTruncate(title, 80)
	subtitle = cwMasterTruncate(subtitle, 140)

	replacements := map[string]string{
		"主标题文字":   title,
		"课程主标题":   title,
		"页面主标题":   title,
		"副标题文字说明": subtitle,
		"课程副标题":   subtitle,
	}

	out := source
	for oldText, newText := range replacements {
		if oldText != "" && newText != "" {
			out = strings.ReplaceAll(out, oldText, htmlstd.EscapeString(newText))
		}
	}
	return out
}

func (s *CoursewareGenService) generateTemplateMatchedNav(
	ctx context.Context,
	cw *models.Courseware,
	page *models.CoursewarePage,
	tplInfo *cwTemplateInfo,
	logoURL string,
	orgName string,
	totalPages int,
	userID string,
	schoolID string,
	aiCfg *ai.EffectiveConfig,
) (string, string, int, error) {
	sample := strings.TrimSpace(tplInfo.SamplePages[0])
	sample = cwMasterTruncate(sample, cwMasterNavSampleRunes)

	cssJSON, _ := json.Marshal(tplInfo.CSSVariables)

	systemPrompt := `你是课件模板导航栏设计助手。

模板首页本身没有导航栏，你只需要补充一段导航栏HTML。

【硬性规则】
1. 只输出导航栏HTML，用<!-- NAV_START -->和<!-- NAV_END -->包裹。
2. 不得输出首页正文、完整页面、html/body/head标签。
3. 导航栏高度固定80px。
4. 导航栏的字体、配色、圆角、边框、阴影和装饰语言必须与模板首页一致。
5. 左侧放机构或课程名称，右侧放年级和页码。
6. 有Logo URL时使用img；没有Logo时不强行添加字母Logo方块。
7. 页码显示为“1 / 总页数”。
8. 输出必须是完整、闭合、可直接插入页面的HTML。`

	userPrompt := fmt.Sprintf(`## 模板信息
模板名称：%s
模板风格：%s
CSS变量：%s

## 当前课件
课件标题：%s
本页标题：%s
学科：%s
年级：%s
机构名称：%s
Logo URL：%s
总页数：%d

## 模板首页参考
%s

请只生成与该模板视觉语言一致的导航栏。`,
		tplInfo.Name,
		tplInfo.StyleCategory,
		string(cssJSON),
		cw.Title,
		page.Title,
		cw.Subject,
		cw.Grade,
		orgName,
		logoURL,
		totalPages,
		sample,
	)

	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_generate",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if aiErr != nil {
		return "", "", 0, aiErr
	}

	nav := ExtractNavByMarkers(result.Content)
	if strings.TrimSpace(nav) == "" {
		nav = s.extractHTMLFromAIOutput(result.Content)
	}
	if strings.TrimSpace(nav) == "" {
		return "", "", 0, fmt.Errorf("AI未返回有效导航栏HTML")
	}

	return strings.TrimSpace(nav), result.ModelUsed, result.TokensUsed, nil
}

func cwMasterTruncate(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes])
}
