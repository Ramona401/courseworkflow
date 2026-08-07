package services

// courseware_comic_page_helper_test.go
//
// 验证页面渲染侧和图片生成侧分别使用独立截断函数。
// 不连接数据库、不调用AI。

import "testing"

func TestCoursewareComicPageUnicodeTruncate(
	t *testing.T,
) {
	pageResult :=
		truncateCoursewareComicRunes(
			"  连云港西游文化旅游路线  ",
			6,
		)

	if pageResult != "连云港西游文" {
		t.Fatalf(
			"页面渲染侧Unicode截断错误: %q",
			pageResult,
		)
	}

	generationResult :=
		truncateCoursewareComicGenerationRunes(
			"  连云港西游文化旅游路线  ",
			4,
		)

	if generationResult != "连云港西" {
		t.Fatalf(
			"图片生成侧Unicode截断错误: %q",
			generationResult,
		)
	}

	if pageResult ==
		generationResult {
		t.Fatal(
			"两个职责独立的截断调用没有按各自限制执行",
		)
	}
}
