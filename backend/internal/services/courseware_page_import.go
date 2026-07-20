package services

// courseware_page_import.go — 外部HTML导入服务
//
// 外部HTML属于作者私有源码覆盖能力：
//   - Handler在解析JSON前完成作者域预检；
//   - Service重新加载正式课件并二次授权；
//   - 生成中、自动装配中和审核提交后禁止导入；
//   - 页面必须真实属于路径课件；
//   - 写库前必须成功保存旧版快照；
//   - 外来HTML必须通过大小、画布和结构完整性校验。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	importNavStartMarker = "<!-- NAV_START -->"
	importNavEndMarker   = "<!-- NAV_END -->"
)

// ImportPageHTML 把老师粘贴的外部完整HTML导入指定页面。
func (s *CoursewareGenService) ImportPageHTML(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	pageNum int,
	pastedHTML string,
) (string, error) {
	courseware, page, err :=
		s.loadOwnedCoursewarePageForMutation(
			ctx,
			coursewareID,
			actor,
			pageNum,
		)
	if err != nil {
		return "", err
	}

	if err := validateCoursewarePageHTMLPayload(
		pastedHTML,
	); err != nil {
		return "", err
	}

	normalized := normalizeRootCanvas(
		pastedHTML,
	)
	if err := validateCoursewarePageHTMLPayload(
		normalized,
	); err != nil {
		return "", err
	}

	// 来源页面带平台导航栏标记时，替换为当前课件已经确认的导航栏，
	// 并由后端重新注入本页页码和总页数。
	if strings.TrimSpace(
		courseware.NavTemplateHTML,
	) != "" {
		startIndex := strings.Index(
			normalized,
			importNavStartMarker,
		)
		endIndex := strings.Index(
			normalized,
			importNavEndMarker,
		)

		if startIndex >= 0 &&
			endIndex > startIndex {
			pages, listErr :=
				repository.ListCoursewarePages(
					ctx,
					coursewareID,
				)
			if listErr != nil {
				return "", fmt.Errorf(
					"读取课件总页数失败: %w",
					listErr,
				)
			}

			newNav := injectPageNumIntoNav(
				courseware.NavTemplateHTML,
				pageNum,
				len(pages),
			)
			newNav = strings.ReplaceAll(
				newNav,
				importNavStartMarker,
				"",
			)
			newNav = strings.ReplaceAll(
				newNav,
				importNavEndMarker,
				"",
			)

			normalized =
				normalized[:startIndex] +
					importNavStartMarker +
					newNav +
					importNavEndMarker +
					normalized[endIndex+
						len(importNavEndMarker):]

			cwGenLog.Info(
				"粘贴导入：已替换为当前课件导航栏",
				"courseware_id",
				coursewareID,
				"page_num",
				pageNum,
				"total_pages",
				len(pages),
			)
		}
	}

	// 背景和字体继续通过后端确定性出口补注。
	styleConfig := s.parseStyleConfig(
		courseware.StyleConfig,
	)
	if templateInfo, templateErr :=
		s.loadTemplateInfo(
			ctx,
			styleConfig.TemplateID,
		); templateErr == nil {
		s.attachUserBackground(
			ctx,
			courseware,
			templateInfo,
		)
		normalized =
			s.applyTemplateBackground(
				normalized,
				templateInfo,
				pageNum,
			)
	}

	// 外部HTML可以整体更换结构，因此采用重生模式结构校验，
	// 只要求画布和HTML结构完整，不要求继承旧页面DOM或资产。
	validation := validateRefinedPageHTML(
		page.HTMLContent,
		normalized,
		"外部HTML导入",
		true,
	)
	if !validation.OK {
		return "", fmt.Errorf(
			"%w: %s",
			ErrCoursewarePageHTMLInvalid,
			validation.Reason,
		)
	}
	if validation.FixedHTML != "" {
		normalized = validation.FixedHTML
	}

	if err := validateCoursewarePageHTMLPayload(
		normalized,
	); err != nil {
		return "", err
	}

	// 已有页面重复导入时必须保证旧版快照成功；新建空页会自动跳过。
	if err := s.SavePageVersionBeforeOverwriteStrict(
		ctx,
		page.ID,
		coursewareID,
		page.HTMLContent,
		models.CWPageVersionSourceManual,
		"粘贴HTML导入前",
	); err != nil {
		return "", err
	}

	if err := repository.UpdateCWPageHTML(
		ctx,
		page.ID,
		normalized,
		"",
		page.MatchedComponentIDs,
		models.CWPageStatusGenerated,
	); err != nil {
		return "", fmt.Errorf(
			"保存导入内容失败: %w",
			err,
		)
	}

	cwGenLog.Info(
		"粘贴HTML导入完成",
		"courseware_id",
		coursewareID,
		"page_num",
		pageNum,
		"page_id",
		page.ID,
		"html_len",
		len(normalized),
	)

	return normalized, nil
}
