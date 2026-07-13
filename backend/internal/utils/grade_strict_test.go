package utils

import "testing"

func TestNormalizeGradeToSpecific(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"高三", "高三", "12", true},
		{"十二年级", "十二年级", "12", true},
		{"12年级", "12年级", "12", true},
		{"纯数字12", "12", "12", true},
		{"高二", "高二", "11", true},
		{"高中拒绝", "高中", "", false},
		{"高中范围拒绝", "10-12", "", false},
		{"初中范围拒绝", "7-9", "", false},
		{"小学低段拒绝", "小学低段", "", false},
		{"空值拒绝", "", "", false},
		{"无法识别拒绝", "幼儿园", "", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := NormalizeGradeToSpecific(
				testCase.input,
			)
			if ok != testCase.ok {
				t.Fatalf(
					"ok不一致：got=%v want=%v",
					ok,
					testCase.ok,
				)
			}
			if got != testCase.want {
				t.Fatalf(
					"结果不一致：got=%q want=%q",
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestIsStrictGradeMatch(t *testing.T) {
	tests := []struct {
		name      string
		resource  string
		requested string
		want      bool
	}{
		{"高三匹配高三", "高三", "高三", true},
		{"十二年级匹配高三", "十二年级", "高三", true},
		{"12年级匹配高三", "12年级", "高三", true},
		{"数字12匹配高三", "12", "高三", true},
		{"高中不能匹配高三", "高中", "高三", false},
		{"高一不能匹配高三", "高一", "高三", false},
		{"高二不能匹配高三", "高二", "高三", false},
		{"10到12不能匹配高三", "10-12", "高三", false},
		{"空年级不能匹配高三", "", "高三", false},
		{"资源高三不能匹配高中请求", "高三", "高中", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := IsStrictGradeMatch(
				testCase.resource,
				testCase.requested,
			)
			if got != testCase.want {
				t.Fatalf(
					"匹配结果不一致：got=%v want=%v",
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestIsStrictSubjectGradeMatch(t *testing.T) {
	if !IsStrictSubjectGradeMatch(
		"语文",
		"十二年级",
		"语文",
		"高三",
	) {
		t.Fatal("同学科同具体年级应匹配")
	}

	if IsStrictSubjectGradeMatch(
		"",
		"高三",
		"语文",
		"高三",
	) {
		t.Fatal("空学科通用资源不应匹配")
	}

	if IsStrictSubjectGradeMatch(
		"数学",
		"高三",
		"语文",
		"高三",
	) {
		t.Fatal("跨学科资源不应匹配")
	}

	if IsStrictSubjectGradeMatch(
		"语文",
		"高中",
		"语文",
		"高三",
	) {
		t.Fatal("学段级资源不应匹配具体年级")
	}
}

func TestNormalizeGradeToStandardLabel(
	t *testing.T,
) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"数字1", "1", "一年级", true},
		{"一年级", "一年级", "一年级", true},
		{"初一", "初一", "七年级", true},
		{"数字7", "7", "七年级", true},
		{"十年级", "十年级", "高一", true},
		{"十二年级", "十二年级", "高三", true},
		{"数字12", "12", "高三", true},
		{"高三", "高三", "高三", true},
		{"高中拒绝", "高中", "", false},
		{"范围拒绝", "10-12", "", false},
		{"空值拒绝", "", "", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := NormalizeGradeToStandardLabel(
				testCase.input,
			)

			if got != testCase.want ||
				ok != testCase.ok {
				t.Fatalf(
					"got=(%q,%v) want=(%q,%v)",
					got,
					ok,
					testCase.want,
					testCase.ok,
				)
			}
		})
	}
}
