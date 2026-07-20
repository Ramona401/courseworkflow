package services

import (
	"fmt"
	"strings"
	"testing"

	"tedna/internal/models"
)

// TestCoursewareTemplateSourceKeepsAllPages 验证20页以内按原顺序全部保存。
func TestCoursewareTemplateSourceKeepsAllPages(t *testing.T) {
	pages := make([]*models.CoursewarePage, 0, 7)

	for number := 1; number <= 7; number++ {
		pages = append(pages, &models.CoursewarePage{
			PageNumber: number,
			HTMLContent: fmt.Sprintf(
				`<div id="page-%d">第%d页</div>`,
				number,
				number,
			),
		})
	}

	result := BuildCoursewareTemplateSourceSelection(pages)

	if result.SourcePageCount != 7 {
		t.Fatalf(
			"来源页数错误: got=%d want=7",
			result.SourcePageCount,
		)
	}
	if len(result.SamplePages) != 7 {
		t.Fatalf(
			"保存页数错误: got=%d want=7",
			len(result.SamplePages),
		)
	}

	for index, pageNumber := range result.SelectedPageNumbers {
		expected := index + 1
		if pageNumber != expected {
			t.Fatalf(
				"页面顺序错误: index=%d got=%d want=%d",
				index,
				pageNumber,
				expected,
			)
		}
	}
}

// TestCoursewareTemplateSourceSelectsTwentyPages 验证超20页均匀选择，并保留首末页。
func TestCoursewareTemplateSourceSelectsTwentyPages(t *testing.T) {
	pages := make([]*models.CoursewarePage, 0, 37)

	for number := 37; number >= 1; number-- {
		pages = append(pages, &models.CoursewarePage{
			PageNumber: number,
			HTMLContent: fmt.Sprintf(
				`<div id="page-%d">第%d页</div>`,
				number,
				number,
			),
		})
	}

	result := BuildCoursewareTemplateSourceSelection(pages)

	if result.SourcePageCount != 37 {
		t.Fatalf(
			"来源页数错误: got=%d want=37",
			result.SourcePageCount,
		)
	}
	if len(result.SamplePages) != templateSourceMaxPages {
		t.Fatalf(
			"保存页数错误: got=%d want=%d",
			len(result.SamplePages),
			templateSourceMaxPages,
		)
	}
	if result.SelectedPageNumbers[0] != 1 {
		t.Fatalf(
			"首页未被保留: got=%d",
			result.SelectedPageNumbers[0],
		)
	}
	if result.SelectedPageNumbers[len(result.SelectedPageNumbers)-1] != 37 {
		t.Fatalf(
			"末页未被保留: got=%d",
			result.SelectedPageNumbers[len(result.SelectedPageNumbers)-1],
		)
	}

	for index := 1; index < len(result.SelectedPageNumbers); index++ {
		if result.SelectedPageNumbers[index] <=
			result.SelectedPageNumbers[index-1] {
			t.Fatalf(
				"选中页码未保持递增: %v",
				result.SelectedPageNumbers,
			)
		}
	}
}

// TestCoursewareTemplateSourceSkipsEmptyPages 验证空HTML页面不会进入模板。
func TestCoursewareTemplateSourceSkipsEmptyPages(t *testing.T) {
	pages := []*models.CoursewarePage{
		{
			PageNumber:  1,
			HTMLContent: "<div>第一页</div>",
		},
		{
			PageNumber:  2,
			HTMLContent: "   ",
		},
		nil,
		{
			PageNumber:  4,
			HTMLContent: "<div>第四页</div>",
		},
	}

	result := BuildCoursewareTemplateSourceSelection(pages)

	if result.SourcePageCount != 2 {
		t.Fatalf(
			"有效来源页数错误: got=%d want=2",
			result.SourcePageCount,
		)
	}
	if len(result.SamplePages) != 2 {
		t.Fatalf(
			"有效保存页数错误: got=%d want=2",
			len(result.SamplePages),
		)
	}
	if strings.Contains(
		strings.Join(result.SamplePages, "\n"),
		"第二页",
	) {
		t.Fatal("空页面被错误保存")
	}
}

// TestTemplateSampleMatcherSupportsArbitraryPageCount 验证任意页数模板能命中互动页。
func TestTemplateSampleMatcherSupportsArbitraryPageCount(t *testing.T) {
	samples := []string{
		`<div>课程封面</div>`,
		`<div>目录与学习路径</div>`,
		`<div>学习目标</div>`,
		`<div>知识讲解一</div>`,
		`<div>知识讲解二</div>`,
		`<button onclick="nextStep()">互动练习</button>`,
		`<div>课堂小结与知识回顾</div>`,
		`<div>课后作业</div>`,
	}

	page := &models.CoursewarePage{
		PageNumber:          5,
		InteractionType:     "click",
		IdxInteractionLevel: 4,
		Title:               "互动探究",
	}

	index, label := pickTemplateSamplePageIndex(
		samples,
		page,
		5,
		10,
	)

	if index != 5 {
		t.Fatalf(
			"未命中互动模板页: got=%d want=5",
			index,
		)
	}
	if label != "互动练习" {
		t.Fatalf(
			"页型标签错误: got=%s",
			label,
		)
	}
}
