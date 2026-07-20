package services

import (
	"testing"

	"tedna/internal/models"
)

func TestStrictAssistantMatchesEntity(
	t *testing.T,
) {
	tests := []struct {
		name      string
		assistant *models.AIAssistant
		subject   string
		grade     string
		scene     string
		want      bool
	}{
		{
			name: "K12同课程同年级同场景",
			assistant: &models.AIAssistant{
				Subject:    "语文",
				GradeRange: "十二年级",
				Scenes: `[
					"workshop_analyze",
					"workshop_design"
				]`,
			},
			subject: "语文",
			grade:   "高三",
			scene:   "workshop_analyze",
			want:    true,
		},
		{
			name: "中职同课程同具体年级",
			assistant: &models.AIAssistant{
				Subject:    "数控车削",
				GradeRange: "职二",
				Scenes:     `["workshop_write"]`,
			},
			subject: "数控车削",
			grade:   "中职Ⅱ年级",
			scene:   "workshop_write",
			want:    true,
		},
		{
			name: "中职不同年级拒绝",
			assistant: &models.AIAssistant{
				Subject:    "数控车削",
				GradeRange: "中职Ⅰ年级",
				Scenes:     `["workshop_write"]`,
			},
			subject: "数控车削",
			grade:   "中职Ⅱ年级",
			scene:   "workshop_write",
			want:    false,
		},
		{
			name: "成人同课程同具体层级",
			assistant: &models.AIAssistant{
				Subject:    "客户沟通",
				GradeRange: "成人进阶",
				Scenes:     `["workshop_design"]`,
			},
			subject: "客户沟通",
			grade:   "进阶",
			scene:   "workshop_design",
			want:    true,
		},
		{
			name: "不限层级不自动匹配",
			assistant: &models.AIAssistant{
				Subject:    "数控车削",
				GradeRange: "中职不限年级",
				Scenes:     `["workshop_write"]`,
			},
			subject: "数控车削",
			grade:   "中职Ⅰ年级",
			scene:   "workshop_write",
			want:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					strictAssistantMatchesEntity(
						testCase.assistant,
						testCase.subject,
						testCase.grade,
						testCase.scene,
					)

				if got != testCase.want {
					t.Fatalf(
						"got=%v want=%v",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

func TestStrictRecipeMatchesEntity(
	t *testing.T,
) {
	tests := []struct {
		name      string
		recipe    *models.TeachingRecipe
		subject   string
		grade     string
		wantMatch bool
	}{
		{
			name: "K12严格匹配",
			recipe: &models.TeachingRecipe{
				Subject:    "数学",
				GradeRange: "12年级",
			},
			subject:   "数学",
			grade:     "高三",
			wantMatch: true,
		},
		{
			name: "中职严格匹配",
			recipe: &models.TeachingRecipe{
				Subject:    "电工基础",
				GradeRange: "中职Ⅲ年级",
			},
			subject:   "电工基础",
			grade:     "职三",
			wantMatch: true,
		},
		{
			name: "中职不同年级拒绝",
			recipe: &models.TeachingRecipe{
				Subject:    "电工基础",
				GradeRange: "中职Ⅱ年级",
			},
			subject:   "电工基础",
			grade:     "中职Ⅲ年级",
			wantMatch: false,
		},
		{
			name: "中职不限拒绝自动匹配",
			recipe: &models.TeachingRecipe{
				Subject:    "电工基础",
				GradeRange: "中职不限年级",
			},
			subject:   "电工基础",
			grade:     "中职Ⅱ年级",
			wantMatch: false,
		},
		{
			name: "成人严格匹配",
			recipe: &models.TeachingRecipe{
				Subject:    "项目复盘",
				GradeRange: "成人高级",
			},
			subject:   "项目复盘",
			grade:     "高级",
			wantMatch: true,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					strictRecipeMatchesEntity(
						testCase.recipe,
						testCase.subject,
						testCase.grade,
					)

				if got != testCase.wantMatch {
					t.Fatalf(
						"got=%v want=%v",
						got,
						testCase.wantMatch,
					)
				}
			},
		)
	}
}

func TestStrictAssistantMatchesListItem(
	t *testing.T,
) {
	item := &models.AIAssistantListItem{
		Subject:    "机械制图",
		GradeRange: "中职1年级",
		Scenes: []string{
			models.SceneWorkshopDesign,
		},
	}

	if !strictAssistantMatchesListItem(
		item,
		"机械制图",
		"中职Ⅰ年级",
		models.SceneWorkshopDesign,
	) {
		t.Fatal("中职同课程同具体年级同场景列表项应匹配")
	}

	if strictAssistantMatchesListItem(
		item,
		"机械制图",
		"中职Ⅱ年级",
		models.SceneWorkshopDesign,
	) {
		t.Fatal("中职不同具体年级不应匹配")
	}

	if strictAssistantMatchesListItem(
		item,
		"机械制图",
		"中职Ⅰ年级",
		models.SceneWorkshopAnalyze,
	) {
		t.Fatal("不同场景不应匹配")
	}
}

func TestNormalizeStrictResourceScope(
	t *testing.T,
) {
	tests := []struct {
		name        string
		subject     string
		grade       string
		wantSubject string
		wantGrade   string
		wantOK      bool
	}{
		{
			name:        "K12高三标准化",
			subject:     " 语文 ",
			grade:       "十二年级",
			wantSubject: "语文",
			wantGrade:   "高三",
			wantOK:      true,
		},
		{
			name:        "中职一年级标准化",
			subject:     " 数控车削 ",
			grade:       "职一",
			wantSubject: "数控车削",
			wantGrade:   "中职Ⅰ年级",
			wantOK:      true,
		},
		{
			name:        "中职三年级标准化",
			subject:     "机械制图",
			grade:       "中职三年级",
			wantSubject: "机械制图",
			wantGrade:   "中职Ⅲ年级",
			wantOK:      true,
		},
		{
			name:        "成人进阶标准化",
			subject:     "客户沟通",
			grade:       "进阶",
			wantSubject: "客户沟通",
			wantGrade:   "成人进阶",
			wantOK:      true,
		},
		{
			name:    "中职不限拒绝配方",
			subject: "机械制图",
			grade:   "中职不限年级",
			wantOK:  false,
		},
		{
			name:    "成人不限拒绝配方",
			subject: "客户沟通",
			grade:   "成人不限层级",
			wantOK:  false,
		},
		{
			name:    "空课程拒绝",
			subject: " ",
			grade:   "中职Ⅰ年级",
			wantOK:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				subject, grade, ok :=
					normalizeStrictResourceScope(
						testCase.subject,
						testCase.grade,
					)

				if subject !=
					testCase.wantSubject ||
					grade !=
						testCase.wantGrade ||
					ok != testCase.wantOK {
					t.Fatalf(
						"got=(%q,%q,%v) want=(%q,%q,%v)",
						subject,
						grade,
						ok,
						testCase.wantSubject,
						testCase.wantGrade,
						testCase.wantOK,
					)
				}
			},
		)
	}
}

func TestNormalizeAssistantResourceScope(
	t *testing.T,
) {
	tests := []struct {
		name        string
		subject     string
		grade       string
		wantSubject string
		wantGrade   string
		wantOK      bool
	}{
		{
			name:        "中职具体年级允许",
			subject:     "数控车削",
			grade:       "职二",
			wantSubject: "数控车削",
			wantGrade:   "中职Ⅱ年级",
			wantOK:      true,
		},
		{
			name:        "中职不限年级允许手动",
			subject:     "数控车削",
			grade:       "中职不限",
			wantSubject: "数控车削",
			wantGrade:   "中职不限年级",
			wantOK:      true,
		},
		{
			name:        "成人具体层级允许",
			subject:     "客户沟通",
			grade:       "高级",
			wantSubject: "客户沟通",
			wantGrade:   "成人高级",
			wantOK:      true,
		},
		{
			name:        "成人不限层级允许手动",
			subject:     "客户沟通",
			grade:       "成人不限",
			wantSubject: "客户沟通",
			wantGrade:   "成人不限层级",
			wantOK:      true,
		},
		{
			name:        "K12学段允许",
			subject:     "语文",
			grade:       "高中",
			wantSubject: "语文",
			wantGrade:   "高中",
			wantOK:      true,
		},
		{
			name:        "历史空值允许",
			subject:     "语文",
			grade:       "",
			wantSubject: "语文",
			wantGrade:   "",
			wantOK:      true,
		},
		{
			name:    "模糊范围拒绝",
			subject: "语文",
			grade:   "10-12",
			wantOK:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				subject, grade, ok :=
					normalizeAssistantResourceScope(
						testCase.subject,
						testCase.grade,
					)

				if subject !=
					testCase.wantSubject ||
					grade !=
						testCase.wantGrade ||
					ok != testCase.wantOK {
					t.Fatalf(
						"got=(%q,%q,%v) want=(%q,%q,%v)",
						subject,
						grade,
						ok,
						testCase.wantSubject,
						testCase.wantGrade,
						testCase.wantOK,
					)
				}
			},
		)
	}
}

func TestManualAssistantMatchesEntity(
	t *testing.T,
) {
	assistant := &models.AIAssistant{
		Subject:    "数控车削",
		GradeRange: "中职不限年级",
		Scenes:     `["workshop_analyze"]`,
	}

	if !manualAssistantMatchesEntity(
		assistant,
		"数控车削",
		"workshop_analyze",
	) {
		t.Fatal("中职不限年级助手应允许老师手动选择")
	}

	if manualAssistantMatchesEntity(
		assistant,
		"机械制图",
		"workshop_analyze",
	) {
		t.Fatal("不同课程助手仍应拒绝")
	}

	if manualAssistantMatchesEntity(
		assistant,
		"数控车削",
		"workshop_write",
	) {
		t.Fatal("不同场景助手仍应拒绝")
	}
}

func TestManualAssistantGradeRank(
	t *testing.T,
) {
	tests := []struct {
		name           string
		candidateGrade string
		currentGrade   string
		want           int
	}{
		{
			name:           "K12精确具体年级第一档",
			candidateGrade: "十二年级",
			currentGrade:   "高三",
			want:           0,
		},
		{
			name:           "K12同学段第二档",
			candidateGrade: "高二",
			currentGrade:   "高三",
			want:           1,
		},
		{
			name:           "中职精确年级第一档",
			candidateGrade: "职二",
			currentGrade:   "中职Ⅱ年级",
			want:           0,
		},
		{
			name:           "中职其它年级同组第二档",
			candidateGrade: "中职Ⅰ年级",
			currentGrade:   "中职Ⅱ年级",
			want:           1,
		},
		{
			name:           "中职不限年级第二档",
			candidateGrade: "中职不限年级",
			currentGrade:   "中职Ⅱ年级",
			want:           1,
		},
		{
			name:           "成人同组其它层级第二档",
			candidateGrade: "成人入门",
			currentGrade:   "成人高级",
			want:           1,
		},
		{
			name:           "跨教育域第三档",
			candidateGrade: "高中",
			currentGrade:   "中职Ⅱ年级",
			want:           2,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					manualAssistantGradeRank(
						testCase.candidateGrade,
						testCase.currentGrade,
					)

				if got != testCase.want {
					t.Fatalf(
						"got=%d want=%d",
						got,
						testCase.want,
					)
				}
			},
		)
	}
}
