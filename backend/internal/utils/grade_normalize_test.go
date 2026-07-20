package utils

// grade_normalize_test.go — 基础层级归一化单元测试
//
// 覆盖：
//   - K12数字与中文年级；
//   - K12范围和学段；
//   - 职业教育与成人教育文本不得误转为K12数字；
//   - 跨教育域手动候选分组。

import "testing"

func TestNormalizeGradeToNumber(
	t *testing.T,
) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"纯数字", "7", "7"},
		{"数字加年级", "7年级", "7"},
		{"数字加其它文字", "第5年级", "5"},
		{"两位数字", "12年级", "12"},

		{"范围3到6", "3-6", "3-6"},
		{"范围1到9", "1-9", "1-9"},
		{"范围7到9", "7-9", "7-9"},

		{"小学低段", "小学低段", "1-2"},
		{"小学中段", "小学中段", "3-4"},
		{"小学高段", "小学高段", "5-6"},

		{"初一", "初一", "7"},
		{"初二", "初二", "8"},
		{"初三", "初三", "9"},
		{"高一", "高一", "10"},
		{"高二", "高二", "11"},
		{"高三", "高三", "12"},

		{"一年级", "一年级", "1"},
		{"二年级", "二年级", "2"},
		{"三年级", "三年级", "3"},
		{"七年级", "七年级", "7"},
		{"九年级", "九年级", "9"},
		{"十年级", "十年级", "10"},
		{"十一年级", "十一年级", "11"},
		{"十二年级", "十二年级", "12"},

		{
			"职业教育罗马数字不转K12",
			"中职Ⅰ年级",
			"中职Ⅰ年级",
		},
		{
			"职业教育中文数字不转K12",
			"中职一年级",
			"中职一年级",
		},
		{
			"成人教育不转K12",
			"成人进阶",
			"成人进阶",
		},

		{"空字符串", "", ""},
		{"纯空格", "  ", "  "},
		{"无法识别", "幼儿园", "幼儿园"},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					NormalizeGradeToNumber(
						testCase.input,
					)

				if got !=
					testCase.expected {
					t.Fatalf(
						"NormalizeGradeToNumber(%q)=%q want=%q",
						testCase.input,
						got,
						testCase.expected,
					)
				}
			},
		)
	}
}

func TestNormalizeGradeToSegment(
	t *testing.T,
) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"K12小学", "三年级", SegmentPrimary},
		{"K12初中", "八年级", SegmentJunior},
		{"K12高中", "高二", SegmentSenior},

		{
			"职业教育具体年级",
			"中职Ⅱ年级",
			SegmentVocational,
		},
		{
			"职业教育简称",
			"职三",
			SegmentVocational,
		},
		{
			"职业教育不限年级",
			"中职不限年级",
			SegmentVocational,
		},

		{
			"成人教育具体层级",
			"成人高级",
			SegmentAdult,
		},
		{
			"成人教育不限层级",
			"成人不限层级",
			SegmentAdult,
		},

		{"空值", "", SegmentAll},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					NormalizeGradeToSegment(
						testCase.input,
					)

				if got != testCase.want {
					t.Fatalf(
						"got=%q want=%q",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}
