package services

// courseware_comic_page_fragment_test.go
//
// 验证单格替换只修改指定稳定标记区间。
// 不连接数据库、不调用AI。

import (
	"strings"
	"testing"
)

func TestCoursewareComicPanelFragmentReplacement(
	t *testing.T,
) {
	projectID := "project-1"
	panelID := "panel-2"

	start :=
		coursewareComicPanelStartMarker(
			projectID,
			panelID,
		)

	end :=
		coursewareComicPanelEndMarker(
			projectID,
			panelID,
		)

	oldHTML :=
		"<div>前置内容</div>" +
			start +
			`<article data-version="old">旧图</article>` +
			end +
			"<div>后置内容</div>"

	newFragment :=
		start +
			`<article data-version="new">新图</article>` +
			end

	updated, err :=
		replaceCoursewareComicPanelFragment(
			oldHTML,
			projectID,
			panelID,
			newFragment,
		)
	if err != nil {
		t.Fatalf(
			"稳定分格替换失败: %v",
			err,
		)
	}

	required := []string{
		"前置内容",
		"后置内容",
		`data-version="new"`,
	}

	for _, value := range required {
		if !strings.Contains(
			updated,
			value,
		) {
			t.Fatalf(
				"替换后缺少内容: %s",
				value,
			)
		}
	}

	if strings.Contains(
		updated,
		`data-version="old"`,
	) {
		t.Fatal(
			"旧漫画格内容没有被替换",
		)
	}

	if strings.Count(
		updated,
		start,
	) != 1 ||
		strings.Count(
			updated,
			end,
		) != 1 {
		t.Fatal(
			"替换后稳定标记数量异常",
		)
	}
}

func TestCoursewareComicPanelFragmentReplacementRejectsMissingOrDuplicateMarkers(
	t *testing.T,
) {
	projectID := "project-1"
	panelID := "panel-2"

	start :=
		coursewareComicPanelStartMarker(
			projectID,
			panelID,
		)

	end :=
		coursewareComicPanelEndMarker(
			projectID,
			panelID,
		)

	cases := []string{
		"<div>没有标记</div>",
		start + "内容",
		start + "A" + end + start + "B" + end,
	}

	for _, pageHTML := range cases {
		if _, err :=
			replaceCoursewareComicPanelFragment(
				pageHTML,
				projectID,
				panelID,
				start+"新内容"+end,
			); err == nil {
			t.Fatalf(
				"异常标记结构应被拒绝: %s",
				pageHTML,
			)
		}
	}
}
