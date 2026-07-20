package services

import (
	"strings"
	"testing"
)

// TestCWContinuityReferenceExtract 验证页码协议解析及内部标记移除。
func TestCWContinuityReferenceExtract(t *testing.T) {
	instruction := `延续前几页人物形象和逐步点击逻辑，开发下一阶段任务。

<!-- TEDNA_COURSEWARE_PAGE_REFS {"page_numbers":[2,4,5]} -->`

	cleaned, request, err :=
		extractCWCoursewarePageReferences(instruction)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if request == nil {
		t.Fatal("未解析到本课前页引用")
	}

	if len(request.PageNumbers) != 3 {
		t.Fatalf(
			"页码数量错误: got=%d want=3",
			len(request.PageNumbers),
		)
	}

	if strings.Contains(
		cleaned,
		"TEDNA_COURSEWARE_PAGE_REFS",
	) {
		t.Fatal("内部协议标记未被删除")
	}

	if !strings.Contains(cleaned, "下一阶段任务") {
		t.Fatal("老师原始指令丢失")
	}
}

// TestCWContinuityReferenceNormalize 验证升序整理。
func TestCWContinuityReferenceNormalize(t *testing.T) {
	pageNumbers, err := normalizeCWContinuityPageNumbers(
		[]int{5, 2, 4},
		7,
	)
	if err != nil {
		t.Fatalf("校验失败: %v", err)
	}

	expected := []int{2, 4, 5}
	for index, pageNumber := range expected {
		if pageNumbers[index] != pageNumber {
			t.Fatalf(
				"页码顺序错误: got=%v want=%v",
				pageNumbers,
				expected,
			)
		}
	}
}

// TestCWContinuityReferenceRejectsInvalid 验证数量、重复页和当前/后续页限制。
func TestCWContinuityReferenceRejectsInvalid(t *testing.T) {
	cases := []struct {
		name        string
		pageNumbers []int
		currentPage int
	}{
		{
			name:        "超过五页",
			pageNumbers: []int{1, 2, 3, 4, 5, 6},
			currentPage: 8,
		},
		{
			name:        "重复页码",
			pageNumbers: []int{2, 2},
			currentPage: 5,
		},
		{
			name:        "包含当前页",
			pageNumbers: []int{2, 5},
			currentPage: 5,
		},
		{
			name:        "包含后续页",
			pageNumbers: []int{2, 6},
			currentPage: 5,
		},
		{
			name:        "第一页无前页",
			pageNumbers: []int{1},
			currentPage: 1,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := normalizeCWContinuityPageNumbers(
				testCase.pageNumbers,
				testCase.currentPage,
			); err == nil {
				t.Fatal("非法前页引用未被拒绝")
			}
		})
	}
}

// TestCWContinuityReferenceRemovesNav 验证前页导航栏被移除。
func TestCWContinuityReferenceRemovesNav(t *testing.T) {
	source := `<div>
<!-- NAV_START -->
<header>旧导航栏</header>
<!-- NAV_END -->
<main>连续性正文</main>
</div>`

	result := stripCWContinuityReferenceNav(source)

	if strings.Contains(result, "旧导航栏") {
		t.Fatal("前页导航栏未被移除")
	}
	if !strings.Contains(result, "连续性正文") {
		t.Fatal("前页正文被误删")
	}
}

// TestCWContinuityReferenceTruncateKeepsHeadAndTail 验证超长页保留代码头尾。
func TestCWContinuityReferenceTruncateKeepsHeadAndTail(
	t *testing.T,
) {
	head := `<style>.hero{color:red}</style>`
	middle := strings.Repeat("中间内容", 6000)
	tail := `<script>function continueStory(){return true}</script>`

	result, truncated := truncateCWContinuityReferenceHTML(
		head+middle+tail,
		6000,
	)

	if !truncated {
		t.Fatal("超长页面未执行截断")
	}
	if !strings.Contains(result, head) {
		t.Fatal("HTML/CSS头部丢失")
	}
	if !strings.Contains(result, "continueStory") {
		t.Fatal("JavaScript尾部丢失")
	}
}

// TestCWContinuityReferencePrompt 验证连续性规则与页面顺序进入提示词。
func TestCWContinuityReferencePrompt(t *testing.T) {
	references := []cwResolvedCoursewarePageReference{
		{
			PageNumber:     2,
			Title:          "任务导入",
			Purpose:        "建立人物和任务背景",
			ContentSummary: "学生点击人物了解任务",
			HTML:           `<div class="hero">小明</div>`,
		},
		{
			PageNumber:     4,
			Title:          "第一阶段互动",
			Purpose:        "完成第一阶段操作",
			ContentSummary: "通过三步点击推进任务",
			HTML:           `<button onclick="nextStep()">下一步</button>`,
		},
	}

	prompt := buildCWCoursewareContinuityPrompt(references)

	required := []string{
		"第2页、第4页",
		"按页码顺序理解",
		"人物状态",
		"继续发展，而不是重复前页",
		"页码较后的状态",
		"当前页教学内容",
		"nextStep",
	}

	for _, fragment := range required {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf(
				"连续性提示词缺少关键内容: %s",
				fragment,
			)
		}
	}
}
