package services

// courseware_nav_replace.go — 把微调后的导航栏安全替换回第1页HTML。
//
// 支持三种历史页面结构：
//  1. 同时存在 NAV_START / NAV_END；
//  2. AI遗漏NAV_START、只存在NAV_END；
//  3. 两个标记均不存在，依靠既有导航栏兜底提取能力定位。
//
// 本文件只替换导航栏区域，正文、样式块和交互脚本均原样保留。

import "strings"

// replaceNavInPageHTML 把newNav替换进pageHTML。
// newNav应为完整导航栏片段，不需要携带NAV标记。
// 返回值changed=false表示无法可靠定位旧导航栏，调用方必须拒绝写库。
func replaceRefinedNavInPageHTML(pageHTML string, newNav string) (updated string, changed bool) {
	const startMarker = "<!-- NAV_START -->"
	const endMarker = "<!-- NAV_END -->"

	pageHTML = strings.TrimSpace(pageHTML)
	newNav = strings.TrimSpace(newNav)

	if pageHTML == "" || newNav == "" {
		return pageHTML, false
	}

	// 防御性清理：若AI仍返回了NAV标记，只取标记内部真实导航栏。
	if strings.Contains(newNav, startMarker) && strings.Contains(newNav, endMarker) {
		if extracted := ExtractNavByMarkers(newNav); strings.TrimSpace(extracted) != "" {
			newNav = strings.TrimSpace(extracted)
		}
	}

	startIdx := strings.Index(pageHTML, startMarker)
	endIdx := strings.Index(pageHTML, endMarker)

	// 标准结构：精确替换两个标记之间的内容。
	if startIdx >= 0 && endIdx > startIdx {
		contentStart := startIdx + len(startMarker)
		return pageHTML[:contentStart] + "\n" + newNav + "\n" + pageHTML[endIdx:], true
	}

	// 历史常见结构：只有NAV_END。先按80px导航栏特征提取旧块。
	if startIdx < 0 && endIdx > 0 {
		oldNav := extractNavBarFromTopToEnd(pageHTML, endIdx)
		if strings.TrimSpace(oldNav) == "" {
			oldNav = ExtractNavByMarkers(pageHTML)
		}
		if strings.TrimSpace(oldNav) != "" {
			if pos := strings.Index(pageHTML[:endIdx], oldNav); pos >= 0 {
				afterEnd := endIdx + len(endMarker)
				return pageHTML[:pos] +
					startMarker + "\n" + newNav + "\n" + endMarker +
					pageHTML[afterEnd:], true
			}
		}
	}

	// 两个标记均不存在：复用既有兜底提取，只替换第一次出现的完整旧导航栏块。
	oldNav := ExtractNavByMarkers(pageHTML)
	if strings.TrimSpace(oldNav) == "" {
		return pageHTML, false
	}
	if pos := strings.Index(pageHTML, oldNav); pos >= 0 {
		return pageHTML[:pos] +
			startMarker + "\n" + newNav + "\n" + endMarker +
			pageHTML[pos+len(oldNav):], true
	}

	return pageHTML, false
}
