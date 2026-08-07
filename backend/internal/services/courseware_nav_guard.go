package services

// courseware_nav_guard.go — 课件导航栏样式依赖保留、尺寸隔离与结构识别。
//
// 根因一：模板首页走完整母版时，导航栏依赖母版<style>中的类样式；旧逻辑只保存
// NAV标记之间的HTML，批量页重新插入后丢失CSS，Logo按原图尺寸渲染并覆盖页面。
// 根因二：旧代码用“是否含NAV_END”代替真实DOM层级判断，可能把80px导航div当成
// 1920×1080主画布归一化，形成满屏遮罩。
//
// 本文件统一完成：导航相关CSS提取和作用域隔离、cw-page语义清理、80px安全壳、
// Logo尺寸上限、NAV标记兼容，以及真实“双顶层画布”识别。

import (
        "regexp"
        "strconv"
        "strings"
)

const (
        cwNavGuardStyleMarker  = "TEDNA-NAV-GUARD"
        cwNavScopedStyleMarker = "TEDNA-NAV-SCOPED"
)

var (
        // 兼容有无空格、大小写不同的NAV标记。
        cwFlexibleNavStartMarkerRe = regexp.MustCompile(`(?is)<!--\s*NAV_START\s*-->`)
        cwFlexibleNavEndMarkerRe   = regexp.MustCompile(`(?is)<!--\s*NAV_END\s*-->`)

        // 只删除平台自己注入的样式块，不碰模板其它style。
        cwNavGuardStyleBlockRe = regexp.MustCompile(
                `(?is)<style\b[^>]*>\s*/\*\s*TEDNA-NAV-GUARD\b.*?</style>`,
        )

        // HTML属性与CSS简单规则解析。输入均为平台保存的受信模板HTML。
        cwNavClassAttrRe = regexp.MustCompile(
                `(?is)\bclass\s*=\s*(?:"([^"]*)"|'([^']*)')`,
        )
        cwNavIDAttrRe = regexp.MustCompile(
                `(?is)\bid\s*=\s*(?:"([^"]*)"|'([^']*)')`,
        )
        cwNavOpeningDivRe = regexp.MustCompile(`(?is)<div\b[^>]*>`)
        cwNavStyleBlockRe = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style>`)
        cwNavSimpleCSSRuleRe = regexp.MustCompile(`(?s)([^{}]+)\{([^{}]*)\}`)
        cwNavHeightPxRe = regexp.MustCompile(`(?is)\bheight\s*:\s*([0-9]{1,4})px`)
        cwNavHTMLCommentPrefixRe = regexp.MustCompile(`(?is)^\s*<!--.*?-->\s*`)
        cwNavDivTagRe = regexp.MustCompile(`(?is)</?div\b[^>]*>`)
)

// cwNavMarkerRange 表示一对兼容空格变体的NAV标记位置。
type cwNavMarkerRange struct {
        StartMarkerStart int
        StartMarkerEnd   int
        EndMarkerStart   int
        EndMarkerEnd     int
}

// findCWNavMarkerRange 查找完整NAV标记对。
func findCWNavMarkerRange(source string) (cwNavMarkerRange, bool) {
        start := cwFlexibleNavStartMarkerRe.FindStringIndex(source)
        if len(start) != 2 {
                return cwNavMarkerRange{}, false
        }
        endRel := cwFlexibleNavEndMarkerRe.FindStringIndex(source[start[1]:])
        if len(endRel) != 2 {
                return cwNavMarkerRange{}, false
        }
        endStart, endEnd := start[1]+endRel[0], start[1]+endRel[1]
        if endStart <= start[1] {
                return cwNavMarkerRange{}, false
        }
        return cwNavMarkerRange{
                StartMarkerStart: start[0], StartMarkerEnd: start[1],
                EndMarkerStart: endStart, EndMarkerEnd: endEnd,
        }, true
}

// findCWNavMarkerPositions 分别返回首个开始和结束标记，用于只有NAV_END的历史页。
func findCWNavMarkerPositions(source string) (start []int, end []int) {
        return cwFlexibleNavStartMarkerRe.FindStringIndex(source),
                cwFlexibleNavEndMarkerRe.FindStringIndex(source)
}

// stripCWNavMarkers 去掉导航片段外层标记；完整页面只取标记内部内容。
func stripCWNavMarkers(fragment string) string {
        fragment = strings.TrimSpace(fragment)
        if fragment == "" {
                return ""
        }
        if r, ok := findCWNavMarkerRange(fragment); ok {
                return strings.TrimSpace(fragment[r.StartMarkerEnd:r.EndMarkerStart])
        }
        return fragment
}

// prepareNavTemplateForStorage 保存导航模板前补齐CSS依赖并移除错误画布语义。
// 返回值仍是不含真实页码和页面安全壳的数据库模板片段。
func prepareNavTemplateForStorage(navHTML string, coverPageHTML string) string {
        navHTML = stripCWNavMarkers(navHTML)
        navHTML = cwNavGuardStyleBlockRe.ReplaceAllLiteralString(navHTML, "")
        navHTML = removeCWPageClassFromFirstNavDiv(strings.TrimSpace(navHTML))
        if navHTML == "" || strings.Contains(navHTML, cwNavScopedStyleMarker) {
                return navHTML
        }
        scopedCSS := extractScopedNavCSS(coverPageHTML, navHTML)
        if scopedCSS == "" {
                return navHTML
        }
        return scopedCSS + "\n" + navHTML
}

// buildSafeNavBlock 为批量生成和重生构造带标记、页码和安全壳的完整导航块。
func buildSafeNavBlock(navTemplate string, pageNum int, totalPages int) string {
        navTemplate = stripCWNavMarkers(navTemplate)
        navTemplate = cwNavGuardStyleBlockRe.ReplaceAllLiteralString(navTemplate, "")
        navTemplate = removeCWPageClassFromFirstNavDiv(navTemplate)
        navTemplate = injectPageNumIntoNav(navTemplate, pageNum, totalPages)
        return buildCWNavGuardStyleTag() + "\n" +
                cwNavStartMarker + "\n" +
                wrapSafeNavShell(navTemplate) + "\n" +
                cwNavEndMarker
}

// prepareTrustedNavForPage 把微调前的权威导航转换为安全导航壳。
func prepareTrustedNavForPage(navHTML string, sourcePageHTML string) string {
        return wrapSafeNavShell(prepareNavTemplateForStorage(navHTML, sourcePageHTML))
}

// wrapSafeNavShell 增加固定80px导航壳；已有平台壳时保持原样，避免嵌套。
func wrapSafeNavShell(navHTML string) string {
        navHTML = strings.TrimSpace(navHTML)
        if navHTML == "" {
                return ""
        }
        if strings.Contains(navHTML, `data-tedna-nav-shell="1"`) ||
                strings.Contains(navHTML, `data-tedna-nav-shell='1'`) {
                return navHTML
        }
        return `<div class="tedna-nav-shell" data-tedna-nav-shell="1">` +
                "\n" + navHTML + "\n</div>"
}

// buildCWNavGuardStyleTag 返回确定性的导航尺寸与Logo保护样式。
// 使用!important压过模板或AI的宽泛img/.cw-page规则，安全边界优先于视觉规则。
func buildCWNavGuardStyleTag() string {
        return `<style>/* TEDNA-NAV-GUARD 导航栏尺寸隔离 */` +
                `.tedna-nav-shell{` +
                `position:absolute!important;top:0!important;left:0!important;right:0!important;` +
                `width:100%!important;height:80px!important;min-height:80px!important;max-height:80px!important;` +
                `z-index:1000!important;overflow:hidden!important;box-sizing:border-box!important;` +
                `transform:none!important;background-size:cover!important;background-position:center!important;}` +
                `.tedna-nav-shell,.tedna-nav-shell *{box-sizing:border-box;}` +
                `.tedna-nav-shell>div,.tedna-nav-shell>header,.tedna-nav-shell>nav{` +
                `top:0!important;bottom:auto!important;margin-top:0!important;transform:none!important;` +
                `width:auto!important;max-width:100%!important;height:80px!important;` +
                `max-height:80px!important;min-height:0!important;}` +
                `.tedna-nav-shell img{width:auto!important;height:auto!important;` +
                `max-width:180px!important;max-height:56px!important;object-fit:contain!important;` +
                `flex:0 0 auto!important;}</style>`
}

// ensureCWNavGuardStyle 确保AI微调结果只含一份导航守卫样式。
func ensureCWNavGuardStyle(source string) string {
        source = strings.TrimSpace(source)
        if source == "" {
                return source
        }
        source = cwNavGuardStyleBlockRe.ReplaceAllLiteralString(source, "")
        styleTag, lower := buildCWNavGuardStyleTag(), strings.ToLower(source)
        if headEnd := strings.Index(lower, "</head>"); headEnd >= 0 {
                return source[:headEnd] + styleTag + "\n" + source[headEnd:]
        }

        rootStart := -1
        for _, needle := range []string{"<div", "<section", "<main", "<article"} {
                if idx := strings.Index(lower, needle); idx >= 0 &&
                        (rootStart < 0 || idx < rootStart) {
                        rootStart = idx
                }
        }
        if rootStart < 0 {
                return styleTag + "\n" + source
        }
        gtRel := strings.Index(source[rootStart:], ">")
        if gtRel < 0 {
                return styleTag + "\n" + source
        }
        insertAt := rootStart + gtRel + 1
        return source[:insertAt] + "\n" + styleTag + source[insertAt:]
}

// removeCWPageClassFromFirstNavDiv 移除导航根div上的cw-page类。
// cw-page属于1920×1080内容画布，导航携带该类会被背景与画布规则误命中。
func removeCWPageClassFromFirstNavDiv(navHTML string) string {
        navHTML = strings.TrimSpace(navHTML)
        openingLoc := cwNavOpeningDivRe.FindStringIndex(navHTML)
        if navHTML == "" || len(openingLoc) != 2 {
                return navHTML
        }

        openTag := navHTML[openingLoc[0]:openingLoc[1]]
        classLoc := cwNavClassAttrRe.FindStringSubmatchIndex(openTag)
        if len(classLoc) < 6 {
                return navHTML
        }

        classValue, quote := "", `"`
        if classLoc[2] >= 0 && classLoc[3] >= 0 {
                classValue = openTag[classLoc[2]:classLoc[3]]
        } else if classLoc[4] >= 0 && classLoc[5] >= 0 {
                classValue, quote = openTag[classLoc[4]:classLoc[5]], `'`
        } else {
                return navHTML
        }

        kept := make([]string, 0)
        for _, className := range strings.Fields(classValue) {
                if !strings.EqualFold(className, "cw-page") {
                        kept = append(kept, className)
                }
        }
        replacement := `class=` + quote + strings.Join(kept, " ") + quote
        newOpenTag := openTag[:classLoc[0]] + replacement + openTag[classLoc[1]:]
        return navHTML[:openingLoc[0]] + newOpenTag + navHTML[openingLoc[1]:]
}

// extractScopedNavCSS 从封面style块提取导航依赖规则，并限定到tedna-nav-shell。
// 只复制与导航真实class/id或明确导航语义相关的简单规则，不复制整页背景布局。
func extractScopedNavCSS(coverPageHTML string, navHTML string) string {
        coverPageHTML, navHTML = strings.TrimSpace(coverPageHTML), strings.TrimSpace(navHTML)
        if coverPageHTML == "" || navHTML == "" {
                return ""
        }
        classTokens, idTokens := collectNavSelectorTokens(navHTML)
        seen, scopedRules := make(map[string]struct{}), make([]string, 0, 24)

        for _, styleMatch := range cwNavStyleBlockRe.FindAllStringSubmatch(coverPageHTML, -1) {
                if len(styleMatch) < 2 {
                        continue
                }
                for _, rule := range cwNavSimpleCSSRuleRe.FindAllStringSubmatch(styleMatch[1], -1) {
                        if len(rule) < 3 {
                                continue
                        }
                        selector, declarations := strings.TrimSpace(rule[1]), strings.TrimSpace(rule[2])
                        if selector == "" || declarations == "" ||
                                strings.HasPrefix(selector, "@") ||
                                !isNavRelevantSelector(selector, classTokens, idTokens) {
                                continue
                        }
                        scopedSelector := scopeNavSelector(selector)
                        if scopedSelector == "" {
                                continue
                        }
                        normalized := scopedSelector + "{" + declarations + "}"
                        if _, exists := seen[normalized]; exists {
                                continue
                        }
                        seen[normalized] = struct{}{}
                        scopedRules = append(scopedRules, normalized)
                }
        }
        if len(scopedRules) == 0 {
                return ""
        }
        return `<style>/* TEDNA-NAV-SCOPED 模板导航依赖样式 */` +
                strings.Join(scopedRules, "") + `</style>`
}

// collectNavSelectorTokens 收集导航实际使用的class/id，排除平台画布类与导航壳类。
func collectNavSelectorTokens(navHTML string) (map[string]struct{}, map[string]struct{}) {
        classes, ids := make(map[string]struct{}), make(map[string]struct{})
        for _, match := range cwNavClassAttrRe.FindAllStringSubmatch(navHTML, -1) {
                if len(match) < 3 {
                        continue
                }
                value := match[1]
                if value == "" {
                        value = match[2]
                }
                for _, className := range strings.Fields(value) {
                        className = strings.TrimSpace(className)
                        if className != "" && !strings.EqualFold(className, "cw-page") &&
                                !strings.EqualFold(className, "tedna-nav-shell") {
                                classes[strings.ToLower(className)] = struct{}{}
                        }
                }
        }
        for _, match := range cwNavIDAttrRe.FindAllStringSubmatch(navHTML, -1) {
                if len(match) < 3 {
                        continue
                }
                value := match[1]
                if value == "" {
                        value = match[2]
                }
                if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
                        ids[value] = struct{}{}
                }
        }
        return classes, ids
}

// isNavRelevantSelector 只接受导航真实class/id、明确导航语义词或独立nav/header元素。
func isNavRelevantSelector(
        selector string,
        classTokens map[string]struct{},
        idTokens map[string]struct{},
) bool {
        lower := strings.ToLower(selector)
        for className := range classTokens {
                if strings.Contains(lower, "."+className) {
                        return true
                }
        }
        for idValue := range idTokens {
                if strings.Contains(lower, "#"+idValue) {
                        return true
                }
        }
        for _, keyword := range []string{
                "navbar", "topbar", "top-bar", "main-header", "page-header",
                "nav-", "-nav", "header-", "-header", "brand", "logo",
                "page-num", "page-number", "page-counter",
        } {
                if strings.Contains(lower, keyword) {
                        return true
                }
        }
        trimmed := strings.TrimSpace(lower)
        return trimmed == "nav" || strings.HasPrefix(trimmed, "nav:") ||
                trimmed == "header" || strings.HasPrefix(trimmed, "header:")
}

// scopeNavSelector 把模板导航规则限定在.tedna-nav-shell内。
// 移除html/body/cw-page整页祖先，保留导航节点及后代选择器。
func scopeNavSelector(selector string) string {
        scopedParts := make([]string, 0)
        for _, part := range strings.Split(selector, ",") {
                part = strings.TrimSpace(part)
                if part == "" {
                        continue
                }
                if strings.Contains(strings.ToLower(part), ":root") {
                        scopedParts = append(scopedParts,
                                strings.ReplaceAll(part, ":root", ".tedna-nav-shell"))
                        continue
                }
                for _, ancestor := range []string{
                        "html ", "body ", ".cw-page.cover ", ".cw-page.inner ", ".cw-page ",
                } {
                        part = strings.ReplaceAll(part, ancestor, "")
                }
                part = strings.TrimSpace(part)
                if part == "" {
                        continue
                }
                if strings.HasPrefix(part, ".tedna-nav-shell") {
                        scopedParts = append(scopedParts, part)
                } else {
                        scopedParts = append(scopedParts, ".tedna-nav-shell "+part)
                }
        }
        return strings.Join(scopedParts, ",")
}

// isDetachedCWNavCanvas 判断是否真的是“导航div + 内容div”两个顶层兄弟节点。
// NAV标记只作为辅助信号；普通单根页面即使有NAV标记也不会被误判。
func isDetachedCWNavCanvas(source string) bool {
        source = strings.TrimSpace(source)
        if source == "" || !strings.HasPrefix(strings.ToLower(source), "<div") {
                return false
        }
        firstOpenEnd := strings.Index(source, ">")
        firstDivEnd, ok := findBalancedTopLevelDivEnd(source)
        if firstOpenEnd < 0 || !ok || firstDivEnd >= len(source) {
                return false
        }

        rest := strings.TrimSpace(source[firstDivEnd:])
        for {
                before := rest
                rest = strings.TrimSpace(
                        cwNavHTMLCommentPrefixRe.ReplaceAllLiteralString(rest, ""),
                )
                if rest == before {
                        break
                }
        }
        lowerRest := strings.ToLower(rest)
        hasSibling := strings.HasPrefix(lowerRest, "<div") ||
                strings.HasPrefix(lowerRest, "<section") ||
                strings.HasPrefix(lowerRest, "<main") ||
                strings.HasPrefix(lowerRest, "<article")
        if !hasSibling {
                return false
        }

        openTag, openLower := source[:firstOpenEnd+1], strings.ToLower(source[:firstOpenEnd+1])
        navLike := strings.Contains(openLower, "nav") ||
                strings.Contains(openLower, "header") ||
                strings.Contains(openLower, "topbar") ||
                strings.Contains(openLower, "top-bar") ||
                strings.Contains(openLower, "z-index") ||
                strings.Contains(openLower, "cw-page")
        if match := cwNavHeightPxRe.FindStringSubmatch(openTag); len(match) >= 2 {
                if height, err := strconv.Atoi(match[1]); err == nil && height > 0 && height <= 200 {
                        navLike = true
                }
        }
        firstBlock := source[:firstDivEnd]
        if cwFlexibleNavStartMarkerRe.MatchString(firstBlock) ||
                cwFlexibleNavEndMarkerRe.MatchString(firstBlock) {
                navLike = true
        }
        return navLike
}

// findBalancedTopLevelDivEnd 返回首个顶层div闭合标签后的下标。
func findBalancedTopLevelDivEnd(source string) (int, bool) {
        tags := cwNavDivTagRe.FindAllStringIndex(source, -1)
        if len(tags) == 0 || tags[0][0] != 0 {
                return 0, false
        }
        depth := 0
        for _, loc := range tags {
                token := strings.ToLower(strings.TrimSpace(source[loc[0]:loc[1]]))
                if strings.HasPrefix(token, "</div") {
                        depth--
                        if depth == 0 {
                                return loc[1], true
                        }
                } else if !strings.HasSuffix(token, "/>") {
                        depth++
                }
        }
        return 0, false
}
