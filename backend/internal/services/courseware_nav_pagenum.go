package services

// courseware_nav_pagenum.go — 导航栏页码管理（剥除+注入）
//
// 设计思路：导航栏模板（nav_template_html）不存页码，页码由后端在拼接时确定性追加。
// 根治两类问题：
//   1. ReplaceNavPageNumbers 正则把Logo URL中的数字串误当页码替换，导致Logo裂掉
//   2. AI生成的页码div样式不一致，导致右侧页码截断/格式飘
//
// 本文件提供：
//   - StripNavPageNumbers：从导航栏中剥除页码元素（保存模板时调用）
//   - injectPageNumIntoNav：在导航栏末尾追加页码div（拼接时调用）
//   - buildNavPageNumDiv：构建固定样式的页码div
//
// 调用关系：
//   SaveNavTemplate / RefineNav → StripNavPageNumbers（保存时剥页码）
//   assembleFullPage / restoreAuthoritativeNav → injectPageNumIntoNav（拼接时加页码）

import (
	"fmt"
	"regexp"
	"strings"
)

// cwNavPageNumDivRe 匹配导航栏中独立的页码容器div。
// 匹配模式：<div ...> 数字/数字 或 占位符 </div>（div内只含页码文本，无嵌套子元素）
var cwNavPageNumDivRe = regexp.MustCompile(
	`(?is)<div[^>]*>\s*(?:\d+\s*/\s*\d+|\{\{PAGE_NUM\}\}\s*/\s*\{\{TOTAL_PAGES\}\})\s*</div>`,
)

// StripNavPageNumbers 从导航栏HTML中剥除页码元素（保存模板时调用）。
//
// 策略（由精到粗，命中即止）：
//  1. 优先剥整个页码div容器（匹配含"数字/数字"或占位符的独立div）——最干净；
//  2. div剥除未命中时，回退剥裸文本页码（先保护URL再正则替换）——兼容非div包裹的页码。
//
// 剥除后模板只含logo+机构名等结构元素，页码由 injectPageNumIntoNav 拼接时追加。

var (
	// 页码三段：分子、原始分隔符、分母。用于保留01/18、1 / 18等模板原格式。
	cwNavPagePartsRe = regexp.MustCompile(`(\d+)(\s*/\s*)(\d+)`)

	// 新模板占位符支持记录前导零宽度，如{{PAGE_NUM_2}}。
	cwNavPagePlaceholderRe  = regexp.MustCompile(`\{\{PAGE_NUM(?:_(\d+))?\}\}`)
	cwNavTotalPlaceholderRe = regexp.MustCompile(`\{\{TOTAL_PAGES(?:_(\d+))?\}\}`)
)

func StripNavPageNumbers(navHTML string) string {
	nav := strings.TrimSpace(navHTML)
	if nav == "" {
		return nav
	}

	// 保护src/href/url()，避免资源URL里的数字被误当成页码。
	guards := make([]string, 0, 8)
	protected := cwNavURLGuardRe.ReplaceAllStringFunc(nav, func(m string) string {
		token := fmt.Sprintf("\x07GUARD%d\x07", len(guards))
		guards = append(guards, m)
		return token
	})

	// 保留原页码DOM、class、位置、分隔符和前导零，仅替换第一处数字。
	replaced := false
	protected = cwNavPagePartsRe.ReplaceAllStringFunc(protected, func(m string) string {
		if replaced {
			return m
		}
		parts := cwNavPagePartsRe.FindStringSubmatch(m)
		if len(parts) != 4 {
			return m
		}

		pageToken := "{{PAGE_NUM}}"
		if len(parts[1]) > 1 && strings.HasPrefix(parts[1], "0") {
			pageToken = fmt.Sprintf("{{PAGE_NUM_%d}}", len(parts[1]))
		}

		totalToken := "{{TOTAL_PAGES}}"
		if len(parts[3]) > 1 && strings.HasPrefix(parts[3], "0") {
			totalToken = fmt.Sprintf("{{TOTAL_PAGES_%d}}", len(parts[3]))
		}

		replaced = true
		return pageToken + parts[2] + totalToken
	})

	for i, original := range guards {
		token := fmt.Sprintf("\x07GUARD%d\x07", i)
		protected = strings.Replace(protected, token, original, 1)
	}

	return strings.TrimSpace(protected)
}

// buildNavPageNumDiv 构建后端确定性的页码div。
// color:inherit 继承导航栏文字色（深色/浅色导航栏都适配），white-space:nowrap 防换行截断。
func buildNavPageNumDiv(pageNum int, totalPages int) string {
	return fmt.Sprintf(
		`<div style="font-size:22px;font-weight:700;color:inherit;opacity:0.85;letter-spacing:0.1em;white-space:nowrap;">%d / %d</div>`,
		pageNum, totalPages,
	)
}

// injectPageNumIntoNav 在导航栏HTML的最后一个 </div> 之前插入页码div。
//
// 导航栏典型结构：<div style="...flex;justify-content:space-between;...">左侧logo...</div>
// 页码div插入到外层闭合 </div> 之前，flex + space-between 自动推到右端。
// 同时清理可能残留的旧占位符（兼容存量数据）。
func cwNavPlaceholderWidth(token string) int {
	underscore := strings.LastIndex(token, "_")
	end := strings.LastIndex(token, "}}")
	if underscore < 0 || end <= underscore+1 {
		return 0
	}

	width := 0
	for _, ch := range token[underscore+1 : end] {
		if ch < '0' || ch > '9' {
			return 0
		}
		width = width*10 + int(ch-'0')
	}
	return width
}

func cwFormatNavNumber(value int, width int) string {
	if width > 0 {
		return fmt.Sprintf("%0*d", width, value)
	}
	return fmt.Sprintf("%d", value)
}

func cwInsertFallbackPageDiv(nav string, pageDiv string) string {
	trimmed := strings.TrimSpace(nav)
	lower := strings.ToLower(trimmed)

	// 按导航根标签插入，避免header/nav结构中的页码被塞进最后一个内部div。
	for _, tagName := range []string{"header", "nav", "div"} {
		openPrefix := "<" + tagName
		if !strings.HasPrefix(lower, openPrefix) {
			continue
		}

		closeTag := "</" + tagName + ">"
		closeIndex := strings.LastIndex(lower, closeTag)
		if closeIndex >= 0 {
			return trimmed[:closeIndex] +
				"\n  " + pageDiv + "\n" +
				trimmed[closeIndex:]
		}
	}

	return trimmed + "\n" + pageDiv
}

func injectPageNumIntoNav(navTemplate string, pageNum int, totalPages int) string {
	nav := strings.TrimSpace(navTemplate)
	if nav == "" {
		return nav
	}

	hadPlaceholder :=
		cwNavPagePlaceholderRe.MatchString(nav) ||
			cwNavTotalPlaceholderRe.MatchString(nav)

	if hadPlaceholder {
		nav = cwNavPagePlaceholderRe.ReplaceAllStringFunc(nav, func(token string) string {
			return cwFormatNavNumber(pageNum, cwNavPlaceholderWidth(token))
		})
		nav = cwNavTotalPlaceholderRe.ReplaceAllStringFunc(nav, func(token string) string {
			return cwFormatNavNumber(totalPages, cwNavPlaceholderWidth(token))
		})
		return nav
	}

	// 兼容没有占位符的存量模板：替换第一处真实页码，同时保留分隔与补零风格。
	guards := make([]string, 0, 8)
	protected := cwNavURLGuardRe.ReplaceAllStringFunc(nav, func(m string) string {
		token := fmt.Sprintf("\x07GUARD%d\x07", len(guards))
		guards = append(guards, m)
		return token
	})

	replaced := false
	protected = cwNavPagePartsRe.ReplaceAllStringFunc(protected, func(m string) string {
		if replaced {
			return m
		}
		parts := cwNavPagePartsRe.FindStringSubmatch(m)
		if len(parts) != 4 {
			return m
		}

		pageWidth := 0
		if len(parts[1]) > 1 && strings.HasPrefix(parts[1], "0") {
			pageWidth = len(parts[1])
		}

		totalWidth := 0
		if len(parts[3]) > 1 && strings.HasPrefix(parts[3], "0") {
			totalWidth = len(parts[3])
		}

		replaced = true
		return cwFormatNavNumber(pageNum, pageWidth) +
			parts[2] +
			cwFormatNavNumber(totalPages, totalWidth)
	})

	for i, original := range guards {
		token := fmt.Sprintf("\x07GUARD%d\x07", i)
		protected = strings.Replace(protected, token, original, 1)
	}

	if replaced {
		return strings.TrimSpace(protected)
	}

	// 模板根本没有页码区域时，才追加平台兜底页码。
	return cwInsertFallbackPageDiv(nav, buildNavPageNumDiv(pageNum, totalPages))
}
