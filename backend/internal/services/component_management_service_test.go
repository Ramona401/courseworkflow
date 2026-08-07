package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveComponentManagementListTarget(
	t *testing.T,
) {
	got, err :=
		resolveComponentManagementListTarget(
			models.EducationDomainVocational,
			models.EducationDomainAdult,
		)

	if err != nil ||
		got != "" {
		t.Fatalf(
			"普通Actor必须忽略客户端筛选域，got=%q err=%v",
			got,
			err,
		)
	}

	got, err =
		resolveComponentManagementListTarget(
			models.EducationDomainMixed,
			models.EducationDomainCommon,
		)

	if err != nil ||
		got != models.EducationDomainCommon {
		t.Fatalf(
			"mixed应允许筛选common，got=%q err=%v",
			got,
			err,
		)
	}

	_, err =
		resolveComponentManagementListTarget(
			models.EducationDomainMixed,
			"unknown",
		)

	if !errors.Is(
		err,
		ErrComponentEducationDomainInvalid,
	) {
		t.Fatalf(
			"mixed非法筛选域应被拒绝，got=%v",
			err,
		)
	}
}

func TestComponentDomainManageAllowed(
	t *testing.T,
) {
	tests := []struct {
		name            string
		currentDomain   string
		componentDomain string
		want            bool
	}{
		{
			name:            "同域允许",
			currentDomain:   models.EducationDomainAdult,
			componentDomain: models.EducationDomainAdult,
			want:            true,
		},
		{
			name:            "普通Actor不能管理common",
			currentDomain:   models.EducationDomainAdult,
			componentDomain: models.EducationDomainCommon,
			want:            false,
		},
		{
			name:            "普通Actor不能跨域",
			currentDomain:   models.EducationDomainAdult,
			componentDomain: models.EducationDomainK12,
			want:            false,
		},
		{
			name:            "mixed可管理common",
			currentDomain:   models.EducationDomainMixed,
			componentDomain: models.EducationDomainCommon,
			want:            true,
		},
		{
			name:            "mixed不能管理非法资源域",
			currentDomain:   models.EducationDomainMixed,
			componentDomain: models.EducationDomainMixed,
			want:            false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					componentDomainManageAllowed(
						test.currentDomain,
						test.componentDomain,
					)

				if got != test.want {
					t.Fatalf(
						"got=%v want=%v",
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestValidateComponentWriteRequest(
	t *testing.T,
) {
	if err := validateComponentWriteRequest(
		"",
		"组件名称",
		"",
		"",
		true,
	); !errors.Is(
		err,
		ErrComponentLibTypeRequired,
	) {
		t.Fatalf(
			"缺少library_type应报错，got=%v",
			err,
		)
	}

	if err := validateComponentWriteRequest(
		models.LibPedagogy,
		"",
		"",
		"",
		true,
	); !errors.Is(
		err,
		ErrComponentLabelRequired,
	) {
		t.Fatalf(
			"缺少display_label应报错，got=%v",
			err,
		)
	}

	if err := validateComponentWriteRequest(
		models.LibPedagogy,
		"组件名称",
		"unknown",
		"",
		true,
	); !errors.Is(
		err,
		ErrComponentInvalidInjectionMode,
	) {
		t.Fatalf(
			"非法injection_mode应报错，got=%v",
			err,
		)
	}

	if err := validateComponentWriteRequest(
		models.LibPedagogy,
		"组件名称",
		models.InjectionOnDemand,
		models.ScopeSchool,
		true,
	); err != nil {
		t.Fatalf(
			"合法请求不应报错，got=%v",
			err,
		)
	}
}
