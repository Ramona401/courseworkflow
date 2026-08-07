package services

// courseware_page_resync.go — 课件页码、总页数和导航栏显示的严格校准服务。
//
// 本服务负责把数据库页面顺序和每页导航栏显示统一校准为同一个事实：
//   - 页面按给定顺序编号为1至N；
//   - coursewares.page_count写为N；
//   - 各页NAV标记区间内的“当前页 / 总页数”写为真实值。
//
// 与旧实现不同，正式校准不会吞掉错误：
//   仓储层会在一个数据库事务中完成页号、HTML和总数更新，
//   任一步失败全部回滚并向上返回错误。
//
// 并发保护：
//   服务层计算校准HTML时记录每页updated_at。
//   仓储层取得行锁后核对快照；页面如果已被源码编辑、AI微调或重构更新，
//   本次校准整体回滚，避免旧HTML覆盖新结果。
//
// 兼容策略：
//   AddPage和DeletePage等历史入口仍可忽略返回值，把校准作为增强动作；
//   拖拽排序和前端“一键校准页码”必须使用ResyncCWPageNumbersByOrder，
//   将错误明确返回给用户。
//
// 安全范围：
//   页码替换只发生在NAV_START/NAV_END标记区间，正文中的数学分数不会被修改。
//   同时兼容标记大小写、额外空格、真实数字和旧占位符。

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var cwResyncLog = cwServiceLog

// cwNavBlockRe定位NAV起止标记中间的导航栏内容。
//
// 标记正则片段统一定义在courseware_nav_markers.go：
//   - (?is)表示忽略大小写，并允许点号跨行匹配；
//   - 起止标记允许注释内部存在不同数量的空格；
//   - 第一个捕获组是需要检查和替换页码的导航栏正文。
var cwNavBlockRe = regexp.MustCompile(
	`(?is)` +
		cwNavStartMarkerPattern +
		`(.*?)` +
		cwNavEndMarkerPattern,
)

// cwNavPageNumRe匹配导航栏中的页码表达式，兼容：
//   - 3 / 12
//   - 3/12
//   - {{PAGE_NUM}} / {{TOTAL_PAGES}}
var cwNavPageNumRe = regexp.MustCompile(
	`(?i)(?:\{\{\s*PAGE_NUM\s*\}\}|\d+)\s*/\s*(?:\{\{\s*TOTAL_PAGES\s*\}\}|\d+)`,
)

// ResyncCWPageNumbers按数据库当前page_number顺序执行严格页码校准。
//
// 常用于插页和删页后补齐连续页号。
// 返回错误表示校准事务未成功提交。
func (s *CoursewareService) ResyncCWPageNumbers(
	ctx context.Context,
	coursewareID string,
) error {
	pages, err := repository.ListCoursewarePages(
		ctx,
		coursewareID,
	)
	if err != nil {
		return fmt.Errorf(
			"页码校准读取页面列表失败: %w",
			err,
		)
	}

	orderedPageIDs := make(
		[]string,
		len(pages),
	)

	for index, page := range pages {
		orderedPageIDs[index] =
			page.ID
	}

	return s.applyCWPageNumberCalibration(
		ctx,
		coursewareID,
		pages,
		orderedPageIDs,
	)
}

// ResyncCWPageNumbersByOrder按照指定页面ID顺序执行严格校准。
//
// 该方法同时服务于：
//   - 用户拖拽调整页面顺序；
//   - 用户确认页面顺序无误后点击“一键校准页码”。
//
// 一键校准时，前端只需按当前视觉顺序提交全部页面ID。
// 即使顺序没有变化，本方法也会重新校准总数与每页导航页码。
func (s *CoursewareService) ResyncCWPageNumbersByOrder(
	ctx context.Context,
	coursewareID string,
	orderedPageIDs []string,
) error {
	pages, err := repository.ListCoursewarePages(
		ctx,
		coursewareID,
	)
	if err != nil {
		return fmt.Errorf(
			"页码校准读取页面列表失败: %w",
			err,
		)
	}

	if len(orderedPageIDs) != len(pages) {
		return fmt.Errorf(
			"页面顺序数据不完整：当前共%d页，请求包含%d页",
			len(pages),
			len(orderedPageIDs),
		)
	}

	pageByID := make(
		map[string]*models.CoursewarePage,
		len(pages),
	)

	for _, page := range pages {
		pageByID[page.ID] = page
	}

	seen := make(
		map[string]struct{},
		len(orderedPageIDs),
	)

	orderedPages := make(
		[]*models.CoursewarePage,
		0,
		len(orderedPageIDs),
	)

	normalizedPageIDs := make(
		[]string,
		0,
		len(orderedPageIDs),
	)

	for _, rawPageID :=
		range orderedPageIDs {
		pageID := strings.TrimSpace(
			rawPageID,
		)

		if pageID == "" {
			return fmt.Errorf(
				"页面顺序数据包含空页面ID",
			)
		}

		if _, duplicated :=
			seen[pageID]; duplicated {
			return fmt.Errorf(
				"页面顺序数据包含重复页面ID: %s",
				pageID,
			)
		}

		page, exists :=
			pageByID[pageID]
		if !exists {
			return fmt.Errorf(
				"页面不属于当前课件或已经不存在: %s",
				pageID,
			)
		}

		seen[pageID] = struct{}{}

		orderedPages = append(
			orderedPages,
			page,
		)

		normalizedPageIDs = append(
			normalizedPageIDs,
			pageID,
		)
	}

	return s.applyCWPageNumberCalibration(
		ctx,
		coursewareID,
		orderedPages,
		normalizedPageIDs,
	)
}

// applyCWPageNumberCalibration根据目标顺序计算HTML页码，并调用仓储事务一次性落库。
func (s *CoursewareService) applyCWPageNumberCalibration(
	ctx context.Context,
	coursewareID string,
	orderedPages []*models.CoursewarePage,
	orderedPageIDs []string,
) error {
	totalPages := len(
		orderedPages,
	)

	htmlByPageID := make(
		map[string]repository.CoursewarePageCalibrationHTML,
	)

	updatedHTMLPages := 0

	for index, page :=
		range orderedPages {
		if page == nil ||
			page.ID == "" {
			return fmt.Errorf(
				"页码校准遇到无效页面记录",
			)
		}

		if strings.TrimSpace(
			page.HTMLContent,
		) == "" {
			continue
		}

		newHTML, changed :=
			refreshNavPageNumInHTML(
				page.HTMLContent,
				index+1,
				totalPages,
			)

		if !changed {
			continue
		}

		if page.UpdatedAt == nil {
			return fmt.Errorf(
				"页码校准缺少页面更新时间(page_id=%s)",
				page.ID,
			)
		}

		htmlByPageID[page.ID] =
			repository.CoursewarePageCalibrationHTML{
				HTMLContent:
					newHTML,
				ExpectedUpdatedAt:
					*page.UpdatedAt,
			}

		updatedHTMLPages++
	}

	if err := repository.ApplyCoursewarePageCalibration(
		ctx,
		coursewareID,
		orderedPageIDs,
		htmlByPageID,
	); err != nil {
		cwResyncLog.Warn(
			"课件页码严格校准失败",
			"courseware_id",
			coursewareID,
			"total_pages",
			totalPages,
			"error",
			err,
		)

		return err
	}

	cwResyncLog.Info(
		"课件页码严格校准完成",
		"courseware_id",
		coursewareID,
		"total_pages",
		totalPages,
		"updated_html_pages",
		updatedHTMLPages,
	)

	return nil
}

// refreshNavPageNumInHTML只修改NAV标记区间内的第一处页码表达式。
//
// 无NAV标记或NAV区间内无页码表达式时，原样返回changed=false。
// 页面正文和脚本中的其他数字不会被触碰。
func refreshNavPageNumInHTML(
	html string,
	pageNumber int,
	totalPages int,
) (string, bool) {
	match := cwNavBlockRe.FindStringSubmatchIndex(
		html,
	)
	if len(match) < 4 {
		return html, false
	}

	navContentStart := match[2]
	navContentEnd := match[3]

	if navContentStart < 0 ||
		navContentEnd < navContentStart {
		return html, false
	}

	navContent :=
		html[navContentStart:navContentEnd]

	location :=
		cwNavPageNumRe.FindStringIndex(
			navContent,
		)
	if len(location) != 2 {
		return html, false
	}

	expectedPageNumber := fmt.Sprintf(
		"%d / %d",
		pageNumber,
		totalPages,
	)

	currentPageNumber :=
		navContent[location[0]:location[1]]

	if currentPageNumber ==
		expectedPageNumber {
		return html, false
	}

	newNavContent :=
		navContent[:location[0]] +
			expectedPageNumber +
			navContent[location[1]:]

	return html[:navContentStart] +
			newNavContent +
			html[navContentEnd:],
		true
}
