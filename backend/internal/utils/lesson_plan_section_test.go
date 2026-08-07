package utils

// lesson_plan_section_test.go — 教案目录与段落替换核心规则测试。
// 测试只覆盖纯文本解析和替换，不访问数据库、不调用AI。

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestLessonPlanSectionIgnoresMarkdownOrderedListItems(t *testing.T) {
	content := `## 教学过程
1. 教师展示图片
2. 学生观察并回答问题
3. 小组交流答案

## 作业设计
完成课后练习。`

	sections := ParseLessonPlanDocumentSections(content)
	if len(sections) != 2 {
		t.Fatalf("普通有序列表不应进入目录，期望2个标题，实际%d个：%+v", len(sections), sections)
	}

	if sections[0].Title != "教学过程" {
		t.Fatalf("第一个目录标题错误：%s", sections[0].Title)
	}

	if !strings.Contains(sections[0].BodyMarkdown, "1. 教师展示图片") {
		t.Fatalf("有序列表应保留在教学过程正文中：%q", sections[0].BodyMarkdown)
	}
}

func TestLessonPlanSectionRecognizesKnownNumberedHeadings(t *testing.T) {
	content := `1. 教学目标
理解核心概念。

2. 教学过程
开展课堂活动。

（1）导入新课
展示问题情境。`

	sections := ParseLessonPlanDocumentSections(content)
	if len(sections) != 3 {
		t.Fatalf("已知数字栏目应进入目录，期望3个标题，实际%d个：%+v", len(sections), sections)
	}

	expected := []struct {
		title string
		level int
	}{
		{title: "教学目标", level: 2},
		{title: "教学过程", level: 2},
		{title: "导入新课", level: 3},
	}

	for index, item := range expected {
		if sections[index].Title != item.title {
			t.Fatalf("第%d个标题错误，期望%s，实际%s", index, item.title, sections[index].Title)
		}
		if sections[index].Level != item.level {
			t.Fatalf("标题%s层级错误，期望%d，实际%d", item.title, item.level, sections[index].Level)
		}
	}
}

func TestLessonPlanSectionRecognizesNumberedActivityHeading(t *testing.T) {
	content := `## 教学过程

1. 活动一：观察图片
学生独立观察。

2. 教师讲解概念
教师说明定义。`

	sections := ParseLessonPlanDocumentSections(content)
	if len(sections) != 2 {
		t.Fatalf("活动标题应识别、普通步骤不应识别，期望2个标题，实际%d个：%+v", len(sections), sections)
	}

	if sections[1].Title != "活动一：观察图片" {
		t.Fatalf("活动标题识别错误：%s", sections[1].Title)
	}

	if !strings.Contains(sections[1].BodyMarkdown, "2. 教师讲解概念") {
		t.Fatalf("普通编号步骤应留在活动正文：%q", sections[1].BodyMarkdown)
	}
}

func TestLessonPlanSectionDuplicateHeadingOccurrence(t *testing.T) {
	content := `## 教学过程

### 探究活动
第一次探究。

### 探究活动
第二次探究。`

	sections := ParseLessonPlanDocumentSections(content)
	if len(sections) != 3 {
		t.Fatalf("期望3个标题，实际%d个", len(sections))
	}

	if sections[1].Occurrence != 1 || sections[2].Occurrence != 2 {
		t.Fatalf(
			"重复标题序号错误：第一次=%d，第二次=%d",
			sections[1].Occurrence,
			sections[2].Occurrence,
		)
	}

	found, ok := FindLessonPlanDocumentSection(content, models.LessonPlanSectionLocator{
		HeadingText: "### 探究活动",
		Occurrence:  2,
	})
	if !ok {
		t.Fatal("未能按第二次出现定位重复标题")
	}

	if !strings.Contains(found.BodyMarkdown, "第二次探究") {
		t.Fatalf("重复标题定位到了错误正文：%q", found.BodyMarkdown)
	}
}

func TestLessonPlanSectionReplacementPreservesOtherHeadings(t *testing.T) {
	content := `## 教学目标
原目标内容。

## 作业设计
原作业内容。
`

	section, ok := FindLessonPlanDocumentSection(content, models.LessonPlanSectionLocator{
		HeadingText: "## 教学目标",
		Occurrence:  1,
	})
	if !ok {
		t.Fatal("未找到教学目标段落")
	}

	next := ReplaceLessonPlanDocumentSectionBody(content, section, "新的教学目标。")
	expected := `## 教学目标
新的教学目标。

## 作业设计
原作业内容。
`

	if next != expected {
		t.Fatalf("段落替换结果错误：\n期望：\n%s\n实际：\n%s", expected, next)
	}
}

func TestLessonPlanSectionFallsBackToWholeDocument(t *testing.T) {
	content := "这是一份没有任何标题的旧教案正文。"
	sections := ParseLessonPlanDocumentSections(content)

	if len(sections) != 1 {
		t.Fatalf("无标题正文应返回一个虚拟节点，实际%d个", len(sections))
	}

	section := sections[0]
	if section.HeadingText != LessonPlanFullDocumentHeading {
		t.Fatalf("虚拟标题标识错误：%s", section.HeadingText)
	}
	if section.BodyMarkdown != content {
		t.Fatalf("虚拟节点未覆盖完整正文：%q", section.BodyMarkdown)
	}
}

func TestLessonPlanSectionIDsDifferByOccurrence(t *testing.T) {
	first := buildLessonPlanSectionID("### 探究活动", 1)
	second := buildLessonPlanSectionID("### 探究活动", 2)

	if first == second {
		t.Fatalf("重复标题的不同出现序号必须生成不同ID：%s", first)
	}
}
