package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveCWComponentReadDomain(
	t *testing.T,
) {
	tests := []struct {
		name        string
		actor       *AssistantActorContext
		wantDomain  string
		wantErrorIs error
	}{
		{
			name: "k12教学Actor",
			actor: &AssistantActorContext{
				UserID:          "user-k12",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainK12,
			},
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "vocational教学Actor",
			actor: &AssistantActorContext{
				UserID:          "user-vocational",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainVocational,
			},
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "mixed系统管理员",
			actor: &AssistantActorContext{
				UserID:          "admin",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			wantDomain: models.EducationDomainMixed,
		},
		{
			name: "mixed区域管理员",
			actor: &AssistantActorContext{
				UserID:          "region-admin",
				Role:            models.RoleRegionAdmin,
				EducationDomain: models.EducationDomainMixed,
			},
			wantDomain: models.EducationDomainMixed,
		},
		{
			name: "普通角色伪造mixed",
			actor: &AssistantActorContext{
				UserID:          "operator",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainMixed,
			},
			wantErrorIs: ErrCWComponentEducationDomainForbidden,
		},
		{
			name: "空教育域fail closed",
			actor: &AssistantActorContext{
				UserID: "operator",
				Role:   models.RoleOperator,
			},
			wantErrorIs: ErrCWComponentEducationDomainForbidden,
		},
		{
			name:        "nil Actor",
			wantErrorIs: ErrCWComponentEducationDomainForbidden,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				domain, err :=
					resolveCWComponentReadDomain(
						test.actor,
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

func TestResolveCWComponentCreationDomain(
	t *testing.T,
) {
	mixedAdmin := &AssistantActorContext{
		UserID:          "admin",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	tests := []struct {
		name        string
		actor       *AssistantActorContext
		requested   string
		wantDomain  string
		wantErrorIs error
	}{
		{
			name:       "mixed管理员创建K12",
			actor:      mixedAdmin,
			requested:  models.EducationDomainK12,
			wantDomain: models.EducationDomainK12,
		},
		{
			name:       "mixed管理员创建common",
			actor:      mixedAdmin,
			requested:  models.EducationDomainCommon,
			wantDomain: models.EducationDomainCommon,
		},
		{
			name:        "mixed管理员必须显式选域",
			actor:       mixedAdmin,
			requested:   "",
			wantErrorIs: ErrCWComponentEducationDomainRequired,
		},
		{
			name:        "mixed不能写入资源",
			actor:       mixedAdmin,
			requested:   models.EducationDomainMixed,
			wantErrorIs: ErrCWComponentEducationDomainInvalid,
		},
		{
			name: "具体域管理员忽略伪造域",
			actor: &AssistantActorContext{
				UserID:          "teaching-admin",
				Role:            models.RoleAdmin,
				EducationDomain: models.EducationDomainVocational,
			},
			requested:  models.EducationDomainK12,
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "普通教师不能创建",
			actor: &AssistantActorContext{
				UserID:          "teacher",
				Role:            models.RoleOperator,
				EducationDomain: models.EducationDomainK12,
			},
			requested:   models.EducationDomainK12,
			wantErrorIs: ErrCWComponentEducationDomainForbidden,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				domain, err :=
					resolveCWComponentCreationDomain(
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

func TestResolveCWComponentListTarget(
	t *testing.T,
) {
	t.Run(
		"普通Actor忽略客户端筛选",
		func(t *testing.T) {
			target, err :=
				resolveCWComponentListTarget(
					models.EducationDomainK12,
					models.EducationDomainVocational,
				)

			if err != nil {
				t.Fatalf(
					"普通Actor筛选不应报错：%v",
					err,
				)
			}

			if target != "" {
				t.Fatalf(
					"普通Actor筛选应被忽略，实际%s",
					target,
				)
			}
		},
	)

	t.Run(
		"mixed可筛选adult",
		func(t *testing.T) {
			target, err :=
				resolveCWComponentListTarget(
					models.EducationDomainMixed,
					models.EducationDomainAdult,
				)

			if err != nil {
				t.Fatalf(
					"mixed合法筛选不应报错：%v",
					err,
				)
			}

			if target != models.EducationDomainAdult {
				t.Fatalf(
					"期望adult，实际%s",
					target,
				)
			}
		},
	)

	t.Run(
		"mixed可筛选common",
		func(t *testing.T) {
			target, err :=
				resolveCWComponentListTarget(
					models.EducationDomainMixed,
					models.EducationDomainCommon,
				)

			if err != nil {
				t.Fatalf(
					"mixed筛选common不应报错：%v",
					err,
				)
			}

			if target != models.EducationDomainCommon {
				t.Fatalf(
					"期望common，实际%s",
					target,
				)
			}
		},
	)

	t.Run(
		"mixed不能作为资源筛选值",
		func(t *testing.T) {
			_, err :=
				resolveCWComponentListTarget(
					models.EducationDomainMixed,
					models.EducationDomainMixed,
				)

			if !errors.Is(
				err,
				ErrCWComponentEducationDomainInvalid,
			) {
				t.Fatalf(
					"mixed资源筛选应被拒绝，实际错误：%v",
					err,
				)
			}
		},
	)
}

func TestCWComponentCanMutate(
	t *testing.T,
) {
	tests := []struct {
		name           string
		currentDomain  string
		resourceDomain string
		want           bool
	}{
		{
			name:           "K12可修改K12",
			currentDomain:  models.EducationDomainK12,
			resourceDomain: models.EducationDomainK12,
			want:           true,
		},
		{
			name:           "K12不能修改职教",
			currentDomain:  models.EducationDomainK12,
			resourceDomain: models.EducationDomainVocational,
			want:           false,
		},
		{
			name:           "具体域不能修改common",
			currentDomain:  models.EducationDomainK12,
			resourceDomain: models.EducationDomainCommon,
			want:           false,
		},
		{
			name:           "mixed可治理职教",
			currentDomain:  models.EducationDomainMixed,
			resourceDomain: models.EducationDomainVocational,
			want:           true,
		},
		{
			name:           "mixed可受控治理common",
			currentDomain:  models.EducationDomainMixed,
			resourceDomain: models.EducationDomainCommon,
			want:           true,
		},
		{
			name:           "mixed资源非法",
			currentDomain:  models.EducationDomainMixed,
			resourceDomain: models.EducationDomainMixed,
			want:           false,
		},
		{
			name:           "空资源域fail closed",
			currentDomain:  models.EducationDomainMixed,
			resourceDomain: "",
			want:           false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				actual :=
					cwComponentCanMutate(
						test.currentDomain,
						test.resourceDomain,
					)

				if actual != test.want {
					t.Fatalf(
						"期望%v，实际%v",
						test.want,
						actual,
					)
				}
			},
		)
	}
}

func TestNormalizeCWComponentScope(
	t *testing.T,
) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "ALL"},
		{input: "   ", want: "ALL"},
		{input: " ALL ", want: "ALL"},
		{input: " 数学 ", want: "数学"},
		{input: "七年级", want: "七年级"},
	}

	for _, test := range tests {
		actual :=
			normalizeCWComponentScope(
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
