package repository

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestNormalizeLessonPlanExplicitEducationDomain(
	t *testing.T,
) {
	tests := []struct {
		name       string
		input      string
		wantDomain string
		wantErr    bool
	}{
		{
			name:       "K12允许",
			input:      models.EducationDomainK12,
			wantDomain: models.EducationDomainK12,
		},
		{
			name:       "职业教育规范化",
			input:      " VOCATIONAL ",
			wantDomain: models.EducationDomainVocational,
		},
		{
			name:       "成人教育允许",
			input:      models.EducationDomainAdult,
			wantDomain: models.EducationDomainAdult,
		},
		{
			name:    "空值拒绝",
			input:   "",
			wantErr: true,
		},
		{
			name:    "mixed拒绝",
			input:   models.EducationDomainMixed,
			wantErr: true,
		},
		{
			name:    "common拒绝",
			input:   models.EducationDomainCommon,
			wantErr: true,
		},
		{
			name:    "非法字符串拒绝",
			input:   "general",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			domain, err :=
				normalizeLessonPlanExplicitEducationDomain(
					testCase.input,
				)

			if testCase.wantErr {
				if !errors.Is(
					err,
					ErrLessonPlanExplicitEducationDomainRequired,
				) {
					t.Fatalf(
						"期望错误%v，实际=%v",
						ErrLessonPlanExplicitEducationDomainRequired,
						err,
					)
				}
				if domain != "" {
					t.Fatalf(
						"失败时不应返回教育域，实际=%q",
						domain,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("不期望错误，实际=%v", err)
			}
			if domain != testCase.wantDomain {
				t.Fatalf(
					"教育域=%q，期望=%q",
					domain,
					testCase.wantDomain,
				)
			}
		})
	}
}

func TestLessonPlanCreateAuditActionRegistered(
	t *testing.T,
) {
	if ActionLessonPlanCreate != "lesson_plan.create" {
		t.Fatalf(
			"普通教案创建审计动作=%q",
			ActionLessonPlanCreate,
		)
	}

	if actionNameMap[ActionLessonPlanCreate] != "创建教案" {
		t.Fatalf(
			"普通教案创建审计中文名=%q",
			actionNameMap[ActionLessonPlanCreate],
		)
	}
}
