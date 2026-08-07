package services

// courseware_nav_replace.go — 把权威导航栏安全替换回页面HTML。
//
// 支持：
//  1. 标准NAV_START/NAV_END；
//  2. 无空格、大小写不同的NAV标记；
//  3. 只有NAV_END的历史页面；
//  4. 两个标记都不存在、依靠80px导航特征定位的历史页面。
//
// 本文件只替换导航区域，正文、内容样式和交互脚本保持原样。
// 替换成功后统一输出标准NAV标记，清理历史标记变体，避免嵌套标记。

import "strings"

// replaceRefinedNavInPageHTML 把newNav替换进pageHTML。
// newNav应为完整导航栏片段，不需要携带NAV标记。
// changed=false表示无法可靠定位旧导航，调用方必须拒绝写库。
func replaceRefinedNavInPageHTML(
        pageHTML string,
        newNav string,
) (
        updated string,
        changed bool,
) {
        pageHTML = strings.TrimSpace(pageHTML)
        newNav = strings.TrimSpace(newNav)
        if pageHTML == "" || newNav == "" {
                return pageHTML, false
        }

        // 防御性清理：AI或历史模板即使携带不同空格形式的NAV标记，也只取内部导航。
        if markerRange, ok := findCWNavMarkerRange(newNav); ok {
                extracted := strings.TrimSpace(
                        newNav[markerRange.StartMarkerEnd:markerRange.EndMarkerStart],
                )
                if extracted != "" {
                        newNav = extracted
                }
        }

        // 标准及变体结构：删除原标记对和其内部内容，再写入唯一一对标准标记。
        if markerRange, ok := findCWNavMarkerRange(pageHTML); ok {
                return pageHTML[:markerRange.StartMarkerStart] +
                        cwNavStartMarker + "\n" +
                        newNav + "\n" +
                        cwNavEndMarker +
                        pageHTML[markerRange.EndMarkerEnd:],
                        true
        }

        startMarker, endMarker := findCWNavMarkerPositions(pageHTML)

        // 历史常见结构：只有NAV_END。先按80px导航特征提取旧块。
        if len(startMarker) == 0 && len(endMarker) == 2 && endMarker[0] > 0 {
                oldNav := extractNavBarFromTopToEnd(
                        pageHTML,
                        endMarker[0],
                )
                if strings.TrimSpace(oldNav) == "" {
                        oldNav = ExtractNavByMarkers(pageHTML)
                }
                if strings.TrimSpace(oldNav) != "" {
                        if pos := strings.Index(
                                pageHTML[:endMarker[0]],
                                oldNav,
                        ); pos >= 0 {
                                return pageHTML[:pos] +
                                        cwNavStartMarker + "\n" +
                                        newNav + "\n" +
                                        cwNavEndMarker +
                                        pageHTML[endMarker[1]:],
                                        true
                        }
                }
        }

        // 无标记历史结构：复用兜底提取，只替换第一次出现的完整旧导航块。
        oldNav := ExtractNavByMarkers(pageHTML)
        if strings.TrimSpace(oldNav) == "" {
                return pageHTML, false
        }
        if pos := strings.Index(pageHTML, oldNav); pos >= 0 {
                return pageHTML[:pos] +
                        cwNavStartMarker + "\n" +
                        newNav + "\n" +
                        cwNavEndMarker +
                        pageHTML[pos+len(oldNav):],
                        true
        }

        return pageHTML, false
}
