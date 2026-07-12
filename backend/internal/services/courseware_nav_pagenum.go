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
func StripNavPageNumbers(navHTML string) string {
	result := navHTML

	// 策略1：剥整个页码div容器
	stripped := cwNavPageNumDivRe.ReplaceAllString(result, "")
	if strings.TrimSpace(stripped) != strings.TrimSpace(result) {
		return strings.TrimSpace(stripped)
	}

	// 策略2：回退剥裸文本页码（先保护URL上下文，再剥页码文本，再还原URL）
	guards := make([]string, 0, 8)
	guardIdx := 0
	protected := cwNavURLGuardRe.ReplaceAllStringFunc(result, func(m string) string {
		// 用 ASCII BEL 字符做占位（Go源码合法，不会出现在HTML中）
		token := fmt.Sprintf("\x07GUARD%d\x07", guardIdx)
		guards = append(guards, m)
		guardIdx++
		return token
	})
	protected = cwNavPageNumRe.ReplaceAllString(protected, "")
	for i, g := range guards {
		token := fmt.Sprintf("\x07GUARD%d\x07", i)
		protected = strings.Replace(protected, token, g, 1)
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
func injectPageNumIntoNav(navTemplate string, pageNum int, totalPages int) string {
	nav := strings.TrimSpace(navTemplate)
	if nav == "" {
		return nav
	}
	// 清理可能残留的旧占位符文本（兼容存量已保存含占位符的模板）
	nav = strings.ReplaceAll(nav, "{{PAGE_NUM}}", "")
	nav = strings.ReplaceAll(nav, "{{TOTAL_PAGES}}", "")

	pageDiv := buildNavPageNumDiv(pageNum, totalPages)

	// 找最后一个 </div>，在其前面插入页码div
	lastClose := strings.LastIndex(nav, "</div>")
	if lastClose < 0 {
		return nav + "\n" + pageDiv
	}
	return nav[:lastClose] + "\n  " + pageDiv + "\n" + nav[lastClose:]
}
