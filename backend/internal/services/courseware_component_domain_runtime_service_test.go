package services

import (
	"errors"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestResolveCWComponentMatchDomain(
	t *testing.T,
) {
	tests := []struct {
		name        string
		actor       *AssistantActorContext
		requested   string
		wantDomain  string
		wantErrorIs error
	}{
		{
			name: "普通K12 Actor忽略职教伪造值",
			actor: &AssistantActorContext{
				UserID:          "teacher-k12",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainK12,
			},
			requested:  models.EducationDomainVocational,
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "普通职教Actor使用可信职教域",
			actor: &AssistantActorContext{
				UserID:          "teacher-vocational",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainVocational,
			},
			requested:  "",
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "mixed管理员显式预览成人域",
			actor: &AssistantActorContext{
				UserID:          "admin",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			requested:  models.EducationDomainAdult,
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "mixed管理员未指定目标域",
			actor: &AssistantActorContext{
				UserID:          "admin",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			requested:   "",
			wantErrorIs: ErrCWComponentEducationDomainRequired,
		},
		{
			name: "common不能作为运行时当前域",
			actor: &AssistantActorContext{
				UserID:          "admin",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			requested:   models.EducationDomainCommon,
			wantErrorIs: ErrCWComponentEducationDomainInvalid,
		},
		{
			name:        "空Actor fail closed",
			actor:       nil,
			wantErrorIs: ErrCWComponentEducationDomainForbidden,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				domain, err :=
					resolveCWComponentMatchDomain(
						test.actor,
						test.requested,
					)

				if test.wantErrorIs != nil {
					if !errors.Is(
						err,
						test.wantErrorIs,
					) {
						t.Fatalf(
							"期望错误%v，实际错误%v",
							test.wantErrorIs,
							err,
						)
					}
					return
				}

				if err != nil {
					t.Fatalf(
						"不期望错误，实际错误：%v",
						err,
					)
				}

				if domain != test.wantDomain {
					t.Fatalf(
						"期望教育域%s，实际%s",
						test.wantDomain,
						domain,
					)
				}
			},
		)
	}
}

func TestNormalizeUniqueCWComponentIDs(
	t *testing.T,
) {
	input := []string{
		" component-a ",
		"",
		"component-b",
		"component-a",
		"   ",
		"component-c",
		"component-b",
	}

	expected := []string{
		"component-a",
		"component-b",
		"component-c",
	}

	actual := normalizeUniqueCWComponentIDs(
		input,
	)

	if !reflect.DeepEqual(
		actual,
		expected,
	) {
		t.Fatalf(
			"期望%v，实际%v",
			expected,
			actual,
		)
	}
}

func TestNormalizeCWComponentDomainDoesNotFallback(
	t *testing.T,
) {
	tests := []struct {
		input string
		want  string
	}{
		{input: " K12 ", want: models.EducationDomainK12},
		{input: "COMMON", want: models.EducationDomainCommon},
		{input: " invalid ", want: "invalid"},
		{input: "", want: ""},
	}

	for _, test := range tests {
		actual := normalizeCWComponentDomain(
			test.input,
		)

		if actual != test.want {
			t.Fatalf(
				"输入%q期望%q，实际%q",
				test.input,
				test.want,
				actual,
			)
		}
	}
}
