package utils

import "testing"

func TestNormalizeGradeToSpecific(
	t *testing.T,
) {
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

		{
			"中职一规范值",
			"中职Ⅰ年级",
			"vocational:1",
			true,
		},
		{
			"中职一简称",
			"职一",
			"vocational:1",
			true,
		},
		{
			"中职二中文写法",
			"中职二年级",
			"vocational:2",
			true,
		},
		{
			"中职三",
			"中职Ⅲ年级",
			"vocational:3",
			true,
		},

		{
			"成人入门",
			"成人入门",
			"adult:entry",
			true,
		},
		{
			"成人进阶简称",
			"进阶",
			"adult:advanced",
			true,
		},
		{
			"成人高级",
			"成人高级",
			"adult:senior",
			true,
		},
		{
			"成人管理者",
			"成人管理者",
			"adult:manager",
			true,
		},

		{"高中拒绝", "高中", "", false},
		{"高中范围拒绝", "10-12", "", false},
		{"初中范围拒绝", "7-9", "", false},
		{"小学低段拒绝", "小学低段", "", false},
		{
			"中职不限年级拒绝自动匹配",
			"中职不限年级",
			"",
			false,
		},
		{
			"成人不限层级拒绝自动匹配",
			"成人不限层级",
			"",
			false,
		},
		{"空值拒绝", "", "", false},
		{"无法识别拒绝", "幼儿园", "", false},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got, ok :=
					NormalizeGradeToSpecific(
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
			},
		)
	}
}

func TestIsStrictGradeMatch(
	t *testing.T,
) {
	tests := []struct {
		name      string
		resource  string
		requested string
		want      bool
	}{
		{"高三匹配高三", "高三", "高三", true},
		{"十二年级匹配高三", "十二年级", "高三", true},
		{"数字12匹配高三", "12", "高三", true},
		{"高中不能匹配高三", "高中", "高三", false},
		{"高一不能匹配高三", "高一", "高三", false},

		{
			"职一匹配中职一",
			"职一",
			"中职Ⅰ年级",
			true,
		},
		{
			"职一不能匹配职二",
			"中职Ⅰ年级",
			"中职Ⅱ年级",
			false,
		},
		{
			"中职一不能匹配K12一年级",
			"中职Ⅰ年级",
			"一年级",
			false,
		},
		{
			"中职不限不能自动匹配职一",
			"中职不限年级",
			"中职Ⅰ年级",
			false,
		},

		{
			"成人进阶匹配简称",
			"成人进阶",
			"进阶",
			true,
		},
		{
			"成人入门不能匹配进阶",
			"成人入门",
			"成人进阶",
			false,
		},
		{
			"成人入门不能匹配K12一年级",
			"成人入门",
			"一年级",
			false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					IsStrictGradeMatch(
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
			},
		)
	}
}

func TestIsStrictSubjectGradeMatch(
	t *testing.T,
) {
	if !IsStrictSubjectGradeMatch(
		"语文",
		"十二年级",
		"语文",
		"高三",
	) {
		t.Fatal("同课程同K12具体年级应匹配")
	}

	if !IsStrictSubjectGradeMatch(
		"数控车削",
		"职二",
		"数控车削",
		"中职Ⅱ年级",
	) {
		t.Fatal("同课程同中职具体年级应匹配")
	}

	if !IsStrictSubjectGradeMatch(
		"客户沟通",
		"成人进阶",
		"客户沟通",
		"进阶",
	) {
		t.Fatal("同课程同成人具体层级应匹配")
	}

	if IsStrictSubjectGradeMatch(
		"数学",
		"中职Ⅰ年级",
		"数学",
		"一年级",
	) {
		t.Fatal("跨教育域同序号层级不应匹配")
	}

	if IsStrictSubjectGradeMatch(
		"",
		"高三",
		"语文",
		"高三",
	) {
		t.Fatal("空课程不应匹配")
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
		{"初一", "初一", "七年级", true},
		{"十二年级", "十二年级", "高三", true},

		{
			"职一标准化",
			"职一",
			VocationalGradeOne,
			true,
		},
		{
			"中职二标准化",
			"中职二年级",
			VocationalGradeTwo,
			true,
		},
		{
			"中职三标准化",
			"中职Ⅲ年级",
			VocationalGradeThree,
			true,
		},

		{
			"成人入门标准化",
			"入门",
			AdultLevelEntry,
			true,
		},
		{
			"成人管理者标准化",
			"成人管理者",
			AdultLevelManager,
			true,
		},

		{"高中拒绝", "高中", "", false},
		{
			"中职不限拒绝具体标准化",
			"中职不限年级",
			"",
			false,
		},
		{
			"成人不限拒绝具体标准化",
			"成人不限层级",
			"",
			false,
		},
		{"范围拒绝", "10-12", "", false},
		{"空值拒绝", "", "", false},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got, ok :=
					NormalizeGradeToStandardLabel(
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
			},
		)
	}
}

func TestNormalizeBroadLearningLevel(
	t *testing.T,
) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"小学", "小学", true},
		{"初中", "初中", true},
		{"高中", "高中", true},
		{
			"中职不限",
			VocationalGradeAll,
			true,
		},
		{
			"成人不限",
			AdultLevelAll,
			true,
		},
		{"中职Ⅰ年级", "", false},
		{"成人入门", "", false},
		{"", "", false},
	}

	for _, testCase := range tests {
		got, ok :=
			NormalizeBroadLearningLevel(
				testCase.input,
			)

		if got != testCase.want ||
			ok != testCase.ok {
			t.Fatalf(
				"input=%q got=(%q,%v) want=(%q,%v)",
				testCase.input,
				got,
				ok,
				testCase.want,
				testCase.ok,
			)
		}
	}
}
