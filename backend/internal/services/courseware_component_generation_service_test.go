package services

import (
	"errors"
	"reflect"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareComponentRuntimeDomain(
	t *testing.T,
) {
	tests := []struct {
		name        string
		courseware  *models.Courseware
		wantDomain  string
		wantErrorIs error
	}{
		{
			name: "K12课件",
			courseware: &models.Courseware{
				ID:              "cw-k12",
				EducationDomain: models.EducationDomainK12,
			},
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "职教课件",
			courseware: &models.Courseware{
				ID:              "cw-vocational",
				EducationDomain: " Vocational ",
			},
			wantDomain:
				models.EducationDomainVocational,
		},
		{
			name: "成人教育课件",
			courseware: &models.Courseware{
				ID:              "cw-adult",
				EducationDomain: models.EducationDomainAdult,
			},
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "common不能进入具体课件运行时",
			courseware: &models.Courseware{
				ID:              "cw-common",
				EducationDomain: models.EducationDomainCommon,
			},
			wantErrorIs:
				ErrCWComponentEducationDomainInvalid,
		},
		{
			name: "mixed不能进入具体课件运行时",
			courseware: &models.Courseware{
				ID:              "cw-mixed",
				EducationDomain: models.EducationDomainMixed,
			},
			wantErrorIs:
				ErrCWComponentEducationDomainInvalid,
		},
		{
			name: "空教育域fail closed",
			courseware: &models.Courseware{
				ID: "cw-empty",
			},
			wantErrorIs:
				ErrCWComponentEducationDomainInvalid,
		},
		{
			name: "空课件ID拒绝",
			courseware: &models.Courseware{
				EducationDomain:
					models.EducationDomainK12,
			},
			wantErrorIs:
				ErrCWComponentRuntimeCoursewareRequired,
		},
		{
			name:        "nil课件拒绝",
			wantErrorIs: ErrCWComponentRuntimeCoursewareRequired,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				domain, err :=
					resolveCoursewareComponentRuntimeDomain(
						test.courseware,
					)

				if test.wantErrorIs != nil {
					if !errors.Is(
						err,
						test.wantErrorIs,
					) {
						t.Fatalf(
							"期望错误%v，实际%v",
							test.wantErrorIs,
							err,
						)
					}

					return
				}

				if err != nil {
					t.Fatalf(
						"不期望错误，实际%v",
						err,
					)
				}

				if domain != test.wantDomain {
					t.Fatalf(
						"期望%s，实际%s",
						test.wantDomain,
						domain,
					)
				}
			},
		)
	}
}

func TestUnwrapMatchedCWComponentResources(
	t *testing.T,
) {
	first := &models.MatchedCWComponent{
		ID:   "component-1",
		Name: "组件一",
	}
	second := &models.MatchedCWComponent{
		ID:   "component-2",
		Name: "组件二",
	}

	resources := []*models.MatchedCWComponentResource{
		nil,
		{},
		{
			MatchedCWComponent: first,
			EducationDomain:
				models.EducationDomainK12,
		},
		{
			MatchedCWComponent: second,
			EducationDomain:
				models.EducationDomainCommon,
		},
	}

	actual :=
		unwrapMatchedCWComponentResources(
			resources,
		)

	expected := []*models.MatchedCWComponent{
		first,
		second,
	}

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
