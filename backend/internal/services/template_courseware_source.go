package services

// template_courseware_source.go — 从现有课件构建多页个人模板来源。
//
// 适用入口：
//   - PPT导入后生成的课件；
//   - Word或主题生成的课件；
//   - HTML导入页面组成的课件；
//   - 老师已经反复编辑完成的正式课件。
//
// 选择规则：
//   1. 只选取html_content非空的已生成页面。
//   2. 不超过20页时按原page_number顺序全部保存。
//   3. 超过20页时均匀选取20页。
//   4. 首页和末页必定入选。
//   5. 每个页面HTML原样保存，仅去除首尾空白。
//   6. 不调用AI，不改写DOM、CSS或JavaScript。

import (
	"sort"
	"strings"

	"tedna/internal/models"
)

// CoursewareTemplateSourceSelection 描述从现有课件中选出的模板母版页面。
type CoursewareTemplateSourceSelection struct {
	SamplePages         []string
	SourcePageCount     int
	SelectedPageNumbers []int
}

// BuildCoursewareTemplateSourceSelection 从现有课件页面中构建模板母版选择结果。
func BuildCoursewareTemplateSourceSelection(
	pages []*models.CoursewarePage,
) CoursewareTemplateSourceSelection {
	validPages := make([]*models.CoursewarePage, 0, len(pages))

	for _, page := range pages {
		if page == nil || strings.TrimSpace(page.HTMLContent) == "" {
			continue
		}
		validPages = append(validPages, page)
	}

	sort.SliceStable(
		validPages,
		func(left int, right int) bool {
			return validPages[left].PageNumber <
				validPages[right].PageNumber
		},
	)

	result := CoursewareTemplateSourceSelection{
		SamplePages:         []string{},
		SourcePageCount:     len(validPages),
		SelectedPageNumbers: []int{},
	}

	if len(validPages) == 0 {
		return result
	}

	selectedIndices := evenlySelectTemplateSourceIndices(
		len(validPages),
		templateSourceMaxPages,
	)

	result.SamplePages = make(
		[]string,
		0,
		len(selectedIndices),
	)
	result.SelectedPageNumbers = make(
		[]int,
		0,
		len(selectedIndices),
	)

	for _, index := range selectedIndices {
		page := validPages[index]

		result.SamplePages = append(
			result.SamplePages,
			strings.TrimSpace(page.HTMLContent),
		)
		result.SelectedPageNumbers = append(
			result.SelectedPageNumbers,
			page.PageNumber,
		)
	}

	return result
}

// evenlySelectTemplateSourceIndices 按顺序均匀选取页面下标。
//
// total<=limit时返回全部下标；total>limit时在[0,total-1]上均匀取点。
// 使用整数四舍五入，确保首下标0和末下标total-1均被保留。
func evenlySelectTemplateSourceIndices(
	total int,
	limit int,
) []int {
	if total <= 0 || limit <= 0 {
		return []int{}
	}

	if total <= limit {
		result := make([]int, total)

		for index := 0; index < total; index++ {
			result[index] = index
		}

		return result
	}

	if limit == 1 {
		return []int{0}
	}

	result := make([]int, 0, limit)
	denominator := limit - 1
	lastIndex := total - 1

	for position := 0; position < limit; position++ {
		// 拆为两个普通算式，避免Go解析器拒绝跨行括号表达式。
		numerator := position*lastIndex + denominator/2
		index := numerator / denominator

		if len(result) == 0 ||
			result[len(result)-1] != index {
			result = append(result, index)
		}
	}

	return result
}
