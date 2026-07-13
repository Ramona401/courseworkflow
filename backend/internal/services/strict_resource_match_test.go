package services

import (
	"testing"

	"tedna/internal/models"
)

func TestStrictAssistantMatchesEntity(t *testing.T) {
	exact := &models.AIAssistant{
		Subject:    "语文",
		GradeRange: "十二年级",
		Scenes:     `["workshop_analyze","workshop_design"]`,
	}

	if !strictAssistantMatchesEntity(
		exact,
		"语文",
		"高三",
		"workshop_analyze",
	) {
		t.Fatal("同学科、同具体年级、同场景助手应匹配")
	}

	broadGrade := &models.AIAssistant{
		Subject:    "语文",
		GradeRange: "高中",
		Scenes:     `["workshop_analyze"]`,
	}

	if strictAssistantMatchesEntity(
		broadGrade,
		"语文",
		"高三",
		"workshop_analyze",
	) {
		t.Fatal("高中学段助手不应匹配高三")
	}

	wrongGrade := &models.AIAssistant{
		Subject:    "语文",
		GradeRange: "高二",
		Scenes:     `["workshop_analyze"]`,
	}

	if strictAssistantMatchesEntity(
		wrongGrade,
		"语文",
		"高三",
		"workshop_analyze",
	) {
		t.Fatal("高二助手不应匹配高三")
	}

	wrongScene := &models.AIAssistant{
		Subject:    "语文",
		GradeRange: "高三",
		Scenes:     `["workshop_write"]`,
	}

	if strictAssistantMatchesEntity(
		wrongScene,
		"语文",
		"高三",
		"workshop_analyze",
	) {
		t.Fatal("不包含当前场景的助手不应匹配")
	}
}

func TestStrictRecipeMatchesEntity(t *testing.T) {
	exact := &models.TeachingRecipe{
		Subject:    "数学",
		GradeRange: "12年级",
	}

	if !strictRecipeMatchesEntity(
		exact,
		"数学",
		"高三",
	) {
		t.Fatal("同学科同具体年级配方应匹配")
	}

	broad := &models.TeachingRecipe{
		Subject:    "数学",
		GradeRange: "高中",
	}

	if strictRecipeMatchesEntity(
		broad,
		"数学",
		"高三",
	) {
		t.Fatal("高中通用配方不应匹配高三")
	}

	otherGrade := &models.TeachingRecipe{
		Subject:    "数学",
		GradeRange: "高二",
	}

	if strictRecipeMatchesEntity(
		otherGrade,
		"数学",
		"高三",
	) {
		t.Fatal("高二配方不应匹配高三")
	}
}

func TestStrictAssistantMatchesListItem(t *testing.T) {
	item := &models.AIAssistantListItem{
		Subject:    "语文",
		GradeRange: "12年级",
		Scenes: []string{
			models.SceneWorkshopDesign,
		},
	}

	if !strictAssistantMatchesListItem(
		item,
		"语文",
		"高三",
		models.SceneWorkshopDesign,
	) {
		t.Fatal("同学科同具体年级同场景列表项应匹配")
	}

	if strictAssistantMatchesListItem(
		item,
		"语文",
		"高三",
		models.SceneWorkshopAnalyze,
	) {
		t.Fatal("不同阶段场景不应匹配")
	}

	if strictAssistantMatchesListItem(
		item,
		"语文",
		"高中",
		models.SceneWorkshopDesign,
	) {
		t.Fatal("学段请求不应通过具体年级严格匹配")
	}
}

func TestStrictRecipeRejectsRangeAndEmptyGrade(
	t *testing.T,
) {
	tests := []struct {
		name        string
		recipeGrade string
	}{
		{
			name:        "高中学段",
			recipeGrade: "高中",
		},
		{
			name:        "高中范围",
			recipeGrade: "10-12",
		},
		{
			name:        "空年级",
			recipeGrade: "",
		},
		{
			name:        "高二具体年级",
			recipeGrade: "高二",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recipe := &models.TeachingRecipe{
				Subject:    "语文",
				GradeRange: testCase.recipeGrade,
			}

			if strictRecipeMatchesEntity(
				recipe,
				"语文",
				"高三",
			) {
				t.Fatalf(
					"资源年级%q不应匹配高三",
					testCase.recipeGrade,
				)
			}
		})
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
			name:        "高三标准化",
			subject:     " 语文 ",
			grade:       "十二年级",
			wantSubject: "语文",
			wantGrade:   "高三",
			wantOK:      true,
		},
		{
			name:        "初一标准化",
			subject:     "数学",
			grade:       "初一",
			wantSubject: "数学",
			wantGrade:   "七年级",
			wantOK:      true,
		},
		{
			name:    "空学科拒绝",
			subject: " ",
			grade:   "高三",
			wantOK:  false,
		},
		{
			name:    "高中学段拒绝",
			subject: "语文",
			grade:   "高中",
			wantOK:  false,
		},
		{
			name:    "范围年级拒绝",
			subject: "语文",
			grade:   "10-12",
			wantOK:  false,
		},
		{
			name:    "空年级拒绝",
			subject: "语文",
			grade:   "",
			wantOK:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			subject, grade, ok :=
				normalizeStrictResourceScope(
					testCase.subject,
					testCase.grade,
				)

			if subject != testCase.wantSubject ||
				grade != testCase.wantGrade ||
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
		})
	}
}
