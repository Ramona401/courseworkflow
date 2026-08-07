package services

import (
	"strings"
	"testing"
)

func TestBuildLessonPlanWordFormatSlotsKeepsFixedStructure(t *testing.T) {
	baseline := strings.Join(
		[]string{
			"表格1 · 第1行",
			"第1列：原有教师活动",
			"课前预习阶段：",
			"[图片：image9.wmf]",
			"【提问】原有问题",
		},
		"\n",
	)

	slots :=
		buildLessonPlanWordFormatSlots(
			baseline,
		)

	if len(slots) != 3 {
		t.Fatalf(
			"可编辑槽位数=%d, want 3",
			len(slots),
		)
	}

	if slots[0].Prefix != "第1列：" ||
		slots[0].Original != "原有教师活动" {
		t.Fatalf(
			"第1列槽位拆分异常: %#v",
			slots[0],
		)
	}
	if slots[1].Prefix != "课前预习阶段：" ||
		slots[1].Original != "" {
		t.Fatalf(
			"空反思槽位拆分异常: %#v",
			slots[1],
		)
	}
	if slots[2].Prefix != "【提问】" ||
		slots[2].Original != "原有问题" {
		t.Fatalf(
			"活动标签槽位拆分异常: %#v",
			slots[2],
		)
	}
}

func TestBuildLessonPlanWordFormatSlotsKeepsHeadingPrefix(t *testing.T) {
	baseline := strings.Join(
		[]string{
			"## 教学目标",
			"表格1 · 第1行",
			"第1列：原有内容",
		},
		"\n",
	)

	slots :=
		buildLessonPlanWordFormatSlots(
			baseline,
		)

	if len(slots) != 2 {
		t.Fatalf(
			"可编辑槽位数=%d, want 2",
			len(slots),
		)
	}
	if slots[0].Prefix != "## " ||
		slots[0].Original != "教学目标" {
		t.Fatalf(
			"Markdown标题前缀拆分异常: %#v",
			slots[0],
		)
	}
}

func TestApplyLessonPlanWordFormatReplacementsPreservesLinesAndImages(t *testing.T) {
	baseline := strings.Join(
		[]string{
			"表格1 · 第1行",
			"第1列：原有教师活动",
			"课前预习阶段：",
			"[图片：image9.wmf]",
			"【提问】原有问题",
		},
		"\n",
	)

	slots :=
		buildLessonPlanWordFormatSlots(
			baseline,
		)
	projected,
		changedCount,
		err :=
		applyLessonPlanWordFormatReplacements(
			baseline,
			slots,
			[]lessonPlanWordFormatReplacement{
				{
					ID:   1,
					Text: "改进后的教师活动",
				},
				{
					ID:   2,
					Text: "学生已经完成基础预习。",
				},
				{
					ID:   3,
					Text: "新的问题",
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"应用槽位替换失败: %v",
			err,
		)
	}

	if changedCount != 3 {
		t.Fatalf(
			"实际修改槽位=%d, want 3",
			changedCount,
		)
	}
	if len(strings.Split(projected, "\n")) !=
		len(strings.Split(baseline, "\n")) {
		t.Fatal(
			"格式投影不得改变原Word语义行数",
		)
	}
	for _, expected := range []string{
		"表格1 · 第1行",
		"第1列：改进后的教师活动",
		"课前预习阶段：学生已经完成基础预习。",
		"[图片：image9.wmf]",
		"【提问】新的问题",
	} {
		if !strings.Contains(
			projected,
			expected,
		) {
			t.Fatalf(
				"投影结果缺少: %s",
				expected,
			)
		}
	}
}

func TestParseLessonPlanWordFormatRepairResponse(t *testing.T) {
	raw := "```json\n" +
		`{"replacements":[{"id":2,"text":"补充后的原段落文字"}]}` +
		"\n```"

	replacements, err :=
		parseLessonPlanWordFormatRepairResponse(
			raw,
		)
	if err != nil {
		t.Fatalf(
			"解析格式投影JSON失败: %v",
			err,
		)
	}
	if len(replacements) != 1 ||
		replacements[0].ID != 2 ||
		replacements[0].Text !=
			"补充后的原段落文字" {
		t.Fatalf(
			"格式投影JSON结果异常: %#v",
			replacements,
		)
	}
}

func TestApplyLessonPlanWordFormatReplacementsRejectsUnknownSlot(t *testing.T) {
	baseline :=
		"表格1 · 第1行\n第1列：原文"
	slots :=
		buildLessonPlanWordFormatSlots(
			baseline,
		)

	_, _, err :=
		applyLessonPlanWordFormatReplacements(
			baseline,
			slots,
			[]lessonPlanWordFormatReplacement{
				{
					ID:   999,
					Text: "不能写入",
				},
			},
		)
	if err == nil {
		t.Fatal(
			"未知槽位必须被拒绝",
		)
	}
}
