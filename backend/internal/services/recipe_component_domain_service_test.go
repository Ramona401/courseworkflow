package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestRecipeComponentDomainMatches(
	t *testing.T,
) {
	tests := []struct {
		name            string
		componentDomain string
		recipeDomain    string
		want            bool
	}{
		{
			name:            "K12配方接受K12组件",
			componentDomain: models.EducationDomainK12,
			recipeDomain:    models.EducationDomainK12,
			want:            true,
		},
		{
			name:            "K12配方接受common组件",
			componentDomain: models.EducationDomainCommon,
			recipeDomain:    models.EducationDomainK12,
			want:            true,
		},
		{
			name:            "K12配方拒绝职教组件",
			componentDomain: models.EducationDomainVocational,
			recipeDomain:    models.EducationDomainK12,
			want:            false,
		},
		{
			name:            "common配方接受common组件",
			componentDomain: models.EducationDomainCommon,
			recipeDomain:    models.EducationDomainCommon,
			want:            true,
		},
		{
			name:            "common配方拒绝具体域组件",
			componentDomain: models.EducationDomainK12,
			recipeDomain:    models.EducationDomainCommon,
			want:            false,
		},
		{
			name:            "mixed不能作为配方资源域",
			componentDomain: models.EducationDomainCommon,
			recipeDomain:    models.EducationDomainMixed,
			want:            false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				got :=
					recipeComponentDomainMatches(
						testCase.componentDomain,
						testCase.recipeDomain,
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

func TestValidateRecipeComponentAccessRecords(
	t *testing.T,
) {
	validRecords := []*repository.ComponentAccessRecord{
		{
			ID:              "component-k12",
			EducationDomain: models.EducationDomainK12,
			LibraryType:     models.LibPedagogy,
			Status:          "active",
			ReviewStatus:    models.ComponentReviewApproved,
		},
		{
			ID:              "component-common",
			EducationDomain: models.EducationDomainCommon,
			LibraryType:     models.LibActivityDesign,
			Status:          "active",
			ReviewStatus:    models.ComponentReviewApproved,
		},
	}

	t.Run(
		"具体域同域加common整组通过",
		func(t *testing.T) {
			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-k12",
						"component-common",
					},
					validRecords,
					models.EducationDomainK12,
				)

			if err != nil {
				t.Fatalf(
					"合法组件不应失败：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"缺失组件整组拒绝",
		func(t *testing.T) {
			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-missing",
					},
					validRecords,
					models.EducationDomainK12,
				)

			if !errors.Is(
				err,
				ErrComponentSelectionInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)

	t.Run(
		"异域组件整组拒绝",
		func(t *testing.T) {
			records :=
				[]*repository.ComponentAccessRecord{
					{
						ID:              "component-adult",
						EducationDomain: models.EducationDomainAdult,
						LibraryType:     models.LibPedagogy,
						Status:          "active",
						ReviewStatus:    models.ComponentReviewApproved,
					},
				}

			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-adult",
					},
					records,
					models.EducationDomainK12,
				)

			if !errors.Is(
				err,
				ErrComponentSelectionInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)

	t.Run(
		"未审核组件整组拒绝",
		func(t *testing.T) {
			records :=
				[]*repository.ComponentAccessRecord{
					{
						ID:              "component-pending",
						EducationDomain: models.EducationDomainK12,
						LibraryType:     models.LibPedagogy,
						Status:          "active",
						ReviewStatus:    models.ComponentReviewPending,
					},
				}

			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-pending",
					},
					records,
					models.EducationDomainK12,
				)

			if !errors.Is(
				err,
				ErrComponentSelectionInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)

	t.Run(
		"common配方不能绑定具体域组件",
		func(t *testing.T) {
			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-k12",
					},
					validRecords,
					models.EducationDomainCommon,
				)

			if !errors.Is(
				err,
				ErrComponentSelectionInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)

	t.Run(
		"非法配方资源域拒绝",
		func(t *testing.T) {
			err :=
				validateRecipeComponentAccessRecords(
					[]string{
						"component-common",
					},
					validRecords,
					models.EducationDomainMixed,
				)

			if !errors.Is(
				err,
				ErrRecipeComponentConfigInvalid,
			) {
				t.Fatalf(
					"got=%v",
					err,
				)
			}
		},
	)
}
