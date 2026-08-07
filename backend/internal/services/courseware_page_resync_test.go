package services

// courseware_page_resync_test.go — 页码导航替换的纯函数聚焦测试。
//
// 本测试不连接数据库、不调用AI、不启动HTTP服务。
// 只验证refreshNavPageNumInHTML的安全边界：
//   - 正确替换导航栏中的真实数字页码；
//   - 正确替换历史占位符；
//   - 只替换NAV区间第一处，不改正文中的数学分数；
//   - 兼容标记大小写和额外空格；
//   - 无导航标记时保持原HTML不变；
//   - 页码已经正确时不产生无意义写入。

import (
	"strings"
	"testing"
)

func TestRefreshNavPageNumInHTMLUpdatesNumericPageNumber(
	t *testing.T,
) {
	source := `<div class="cw-page">
<!-- NAV_START -->
<nav><span>2 / 8</span></nav>
<!-- NAV_END -->
<section>正文中的比例 3 / 4 不应修改</section>
</div>`

	got, changed := refreshNavPageNumInHTML(
		source,
		5,
		12,
	)

	if !changed {
		t.Fatal("期望导航栏页码发生变化，但changed=false")
	}

	if !strings.Contains(
		got,
		"<span>5 / 12</span>",
	) {
		t.Fatalf(
			"导航栏页码未正确替换，结果为：%s",
			got,
		)
	}

	if !strings.Contains(
		got,
		"正文中的比例 3 / 4 不应修改",
	) {
		t.Fatalf(
			"正文内容被意外修改，结果为：%s",
			got,
		)
	}
}

func TestRefreshNavPageNumInHTMLUpdatesTemplatePlaceholders(
	t *testing.T,
) {
	source := `<!-- NAV_START -->
<div>{{PAGE_NUM}} / {{TOTAL_PAGES}}</div>
<!-- NAV_END -->`

	got, changed := refreshNavPageNumInHTML(
		source,
		1,
		9,
	)

	if !changed {
		t.Fatal("期望导航栏占位符发生变化，但changed=false")
	}

	if !strings.Contains(
		got,
		"<div>1 / 9</div>",
	) {
		t.Fatalf(
			"导航栏占位符未正确替换，结果为：%s",
			got,
		)
	}
}

func TestRefreshNavPageNumInHTMLUsesFirstPageExpressionOnly(
	t *testing.T,
) {
	source := `<!-- NAV_START -->
<div class="primary">1 / 7</div>
<div class="secondary">99 / 100</div>
<!-- NAV_END -->`

	got, changed := refreshNavPageNumInHTML(
		source,
		3,
		7,
	)

	if !changed {
		t.Fatal("期望导航栏第一处页码发生变化，但changed=false")
	}

	if !strings.Contains(
		got,
		`class="primary">3 / 7`,
	) {
		t.Fatalf(
			"第一处导航页码未正确替换，结果为：%s",
			got,
		)
	}

	if !strings.Contains(
		got,
		`class="secondary">99 / 100`,
	) {
		t.Fatalf(
			"第二处数字表达式不应被修改，结果为：%s",
			got,
		)
	}
}

func TestRefreshNavPageNumInHTMLAcceptsFlexibleMarkerFormatting(
	t *testing.T,
) {
	source := `<!--nav_start-->
<div>4/10</div>
<!-- nav_end -->`

	got, changed := refreshNavPageNumInHTML(
		source,
		6,
		11,
	)

	if !changed {
		t.Fatal("期望兼容格式的导航标记被识别，但changed=false")
	}

	if !strings.Contains(
		got,
		"<div>6 / 11</div>",
	) {
		t.Fatalf(
			"兼容格式标记中的页码未正确替换，结果为：%s",
			got,
		)
	}
}

func TestRefreshNavPageNumInHTMLWithoutNavigationMarkers(
	t *testing.T,
) {
	source := `<div class="cw-page">
<section>正文 2 / 8</section>
</div>`

	got, changed := refreshNavPageNumInHTML(
		source,
		3,
		9,
	)

	if changed {
		t.Fatal("无导航标记时不应报告发生变化")
	}

	if got != source {
		t.Fatalf(
			"无导航标记时HTML必须保持不变，结果为：%s",
			got,
		)
	}
}

func TestRefreshNavPageNumInHTMLAlreadyCorrect(
	t *testing.T,
) {
	source := `<!-- NAV_START -->
<div>3 / 9</div>
<!-- NAV_END -->`

	got, changed := refreshNavPageNumInHTML(
		source,
		3,
		9,
	)

	if changed {
		t.Fatal("页码已经正确时不应报告变化")
	}

	if got != source {
		t.Fatalf(
			"页码已经正确时HTML必须保持不变，结果为：%s",
			got,
		)
	}
}
