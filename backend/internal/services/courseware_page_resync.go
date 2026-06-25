package services

// courseware_page_resync.go — 增删页后的页号连续性 + 导航栏页码刷新（② 导航栏分母烧死专题）
//
// 背景（病根）：
//   导航栏页码 "{{PAGE_NUM}} / {{TOTAL_PAGES}}" 在 assembleFullPage 生成那一刻被替换成
//   真值（如 "1 / 9"）烧进每页 html_content。之后增删页：
//     - DeletePage 只 DELETE 单页、不重排 page_number → 留下页号空洞（1,2,...,7,9,10）；
//     - 各页导航栏里烧死的分母不更新 → 出现 "8 / 9" 实际只剩 8 页这类 x/N 错位。
//   本文件提供 ResyncCWPageNumbers，在增删页后一次性根治两件事：
//     (1) 页号补洞：按 page_number 升序重排成 1..N 连续；
//     (2) 导航栏分子分母刷新：仅在 <!-- NAV_START -->/<!-- NAV_END --> 区间内，
//         把 "数字 / 数字" 替换成 "新页号 / N"，正文零触碰。
//
// 安全设计：
//   - 正则只作用于 NAV 标记之间的子串，正文里的 "3/4" 等绝不被误伤；
//   - 无 NAV 标记的页跳过页码刷新（页号补洞仍照做）；
//   - HTML/页号未变化的页不写库，避免无谓 UPDATE；
//   - 本函数为"增强动作"，失败只记 warn、不阻断增删页主流程（主操作已成功落库）。
//
// 存量策略：老课件本次不主动批量刷；但老师一旦对老课件做增删页，该课件即被动刷正确，
//   等于给存量开了一条被动修复路径，覆盖面优于"仅新课件干净"。

import (
        "context"
        "fmt"
        "regexp"
        "strings"

        "tedna/internal/repository"
)

var cwResyncLog = cwServiceLog // 复用 courseware_service.go 的结构化日志器

// NAV 标记之间页码 "数字 / 数字"（含两侧可选空格）匹配；只在标记区间内使用，正文不碰。
var cwNavPageNumRe = regexp.MustCompile(`\d+\s*/\s*\d+`)

const (
        cwNavStartMarker = "<!-- NAV_START -->"
        cwNavEndMarker   = "<!-- NAV_END -->"
)

// ResyncCWPageNumbers 增删页后重排页号 + 刷新各页导航栏分子分母。
// 在 AddPage / DeletePage / ReorderPages 成功改动后调用。返回 error 仅供日志，调用方可忽略。
func (s *CoursewareService) ResyncCWPageNumbers(ctx context.Context, coursewareID string) error {
        // 1. 拉全部页（已按 page_number 升序）
        pages, err := repository.ListCoursewarePages(ctx, coursewareID)
        if err != nil {
                cwResyncLog.Warn("页码重排：拉取页面列表失败", "courseware_id", coursewareID, "error", err)
                return err
        }
        total := len(pages)
        if total == 0 {
                return nil
        }

        	// (1) 页号补洞：按升序收集 pageID 列表，一次性两阶段避撞重排为 1..N
	//     （pages 已按 page_number 升序，其 ID 顺序即目标顺序；删页留下的洞自然填平）
	orderedIDs := make([]string, len(pages))
	for i, p := range pages {
		orderedIDs[i] = p.ID
	}
	if e := repository.ResequenceCoursewarePagesByIDs(ctx, coursewareID, orderedIDs); e != nil {
		cwResyncLog.Warn("页码重排：重排 page_number 失败",
			"courseware_id", coursewareID, "error", e)
	}

	// (2) 导航栏分子分母刷新：仅在 NAV 标记区间内替换 "数字 / 数字" → "newNum / total"
	for i, p := range pages {
		newNum := i + 1
		if p.HTMLContent == "" {
			continue
		}
		newHTML, changed := refreshNavPageNumInHTML(p.HTMLContent, newNum, total)
		if changed {
			if e := repository.UpdateCWPageHTMLOnly(ctx, p.ID, newHTML); e != nil {
				cwResyncLog.Warn("页码重排：更新导航栏页码失败",
					"courseware_id", coursewareID, "page_id", p.ID, "error", e)
			}
		}
	}

	        // 同步课件页数计数（与实际页数一致）
        _ = repository.UpdateCoursewarePageCount(ctx, coursewareID, total)
        cwResyncLog.Info("页码重排完成", "courseware_id", coursewareID, "total_pages", total)
        return nil
}

// refreshNavPageNumInHTML 仅在 <!-- NAV_START -->/<!-- NAV_END --> 区间内，
// 把第一个 "数字 / 数字" 页码替换为 "pageNum / total"。返回新 HTML 与是否发生改动。
// 无 NAV 标记、或标记区间内无页码时，原样返回 changed=false（不写库）。
func refreshNavPageNumInHTML(html string, pageNum, total int) (string, bool) {
        start := strings.Index(html, cwNavStartMarker)
        if start < 0 {
                return html, false
        }
        end := strings.Index(html, cwNavEndMarker)
        if end < 0 || end <= start {
                return html, false
        }
        // 区间 = [标记内容起点, NAV_END 之前)
        segStart := start + len(cwNavStartMarker)
        segEnd := end
        nav := html[segStart:segEnd]

        want := fmt.Sprintf("%d / %d", pageNum, total)
        // 只替换导航栏区间内的第一处页码（导航栏通常仅一处页码显示）
        replaced := false
        newNav := cwNavPageNumRe.ReplaceAllStringFunc(nav, func(m string) string {
                if replaced {
                        return m // 只动第一处，避免误伤导航栏内可能存在的其它"数字/数字"
                }
                replaced = true
                return want
        })
        if !replaced || newNav == nav {
                return html, false
        }
        return html[:segStart] + newNav + html[segEnd:], true
}
