package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateExtractionLessonDomain(
	t *testing.T,
) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "K12",
			input: " K12 ",
			want:  models.EducationDomainK12,
		},
		{
			name:  "职教",
			input: models.EducationDomainVocational,
			want:  models.EducationDomainVocational,
		},
		{
			name:  "成教",
			input: models.EducationDomainAdult,
			want:  models.EducationDomainAdult,
		},
		{
			name:    "common拒绝",
			input:   models.EducationDomainCommon,
			wantErr: ErrComponentEducationDomainInvalid,
		},
		{
			name:    "mixed拒绝",
			input:   models.EducationDomainMixed,
			wantErr: ErrComponentEducationDomainInvalid,
		},
		{
			name:    "空值拒绝",
			input:   "",
			wantErr: ErrComponentEducationDomainInvalid,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got, err :=
					validateExtractionLessonDomain(
						test.input,
					)

				if test.wantErr != nil {
					if !errors.Is(
						err,
						test.wantErr,
					) {
						t.Fatalf(
							"got err=%v want=%v",
							err,
							test.wantErr,
						)
					}
					return
				}

				if err != nil {
					t.Fatalf(
						"unexpected err=%v",
						err,
					)
				}

				if got != test.want {
					t.Fatalf(
						"got=%q want=%q",
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestExtractionActorCanAccessDomain(
	t *testing.T,
) {
	tests := []struct {
		name         string
		actorDomain  string
		lessonDomain string
		want         bool
	}{
		{
			name:         "同域允许",
			actorDomain:  models.EducationDomainVocational,
			lessonDomain: models.EducationDomainVocational,
			want:         true,
		},
		{
			name:         "普通Actor跨域拒绝",
			actorDomain:  models.EducationDomainVocational,
			lessonDomain: models.EducationDomainAdult,
			want:         false,
		},
		{
			name:         "mixed允许具体教学域",
			actorDomain:  models.EducationDomainMixed,
			lessonDomain: models.EducationDomainAdult,
			want:         true,
		},
		{
			name:         "mixed拒绝common来源",
			actorDomain:  models.EducationDomainMixed,
			lessonDomain: models.EducationDomainCommon,
			want:         false,
		},
		{
			name:         "空Actor域拒绝",
			actorDomain:  "",
			lessonDomain: models.EducationDomainK12,
			want:         false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					extractionActorCanAccessDomain(
						test.actorDomain,
						test.lessonDomain,
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

func TestIsAutoExtractionLibraryType(
	t *testing.T,
) {
	validTypes := []string{
		models.LibActivityDesign,
		models.LibQuestioningStrategy,
		models.LibPedagogy,
		models.LibAssessmentStrategy,
		models.LibCrossSubject,
		models.LibScenarioMaterial,
	}

	for _, libraryType := range validTypes {
		if !isAutoExtractionLibraryType(
			libraryType,
		) {
			t.Fatalf(
				"应允许自动萃取类型：%s",
				libraryType,
			)
		}
	}

	if isAutoExtractionLibraryType(
		models.LibCurriculumStandard,
	) {
		t.Fatal(
			"课程标准库不应由自动萃取通道创建",
		)
	}
}
