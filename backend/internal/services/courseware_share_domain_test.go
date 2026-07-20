package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareForkEducationDomain(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *CoursewareActorContext
		source     *models.Courseware
		wantDomain string
		wantErr    error
	}{
		{
			name: "K12用户复制K12课件并继承K12",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantDomain: models.EducationDomainK12,
		},
		{
			name: "职教用户复制职教课件并继承职教",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainVocational,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
			wantDomain: models.EducationDomainVocational,
		},
		{
			name: "成教用户复制成教课件并继承成教",
			actor: newCoursewareAccessTestActor(
				models.RoleSeniorOperator,
				models.EducationDomainAdult,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainAdult,
			),
			wantDomain: models.EducationDomainAdult,
		},
		{
			name: "K12用户不能复制职教课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
			wantErr: ErrCoursewareEducationDomainMismatch,
		},
		{
			name: "mixed管理员不能Fork个人课件",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "mixed区域管理员不能Fork个人课件",
			actor: newCoursewareAccessTestActor(
				models.RoleRegionAdmin,
				models.EducationDomainMixed,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name: "common课件可查看但不能直接Fork",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainCommon,
			),
			wantErr: ErrCoursewareForkSourceDomainUnsupported,
		},
		{
			name: "非法来源课件域fail-closed",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			source: newCoursewareAccessTestResource(
				"invalid",
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "当前教育域为空不能Fork",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				"",
			),
			source: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareCreationDomainRequired,
		},
		{
			name:  "nil Actor拒绝",
			actor: nil,
			source: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareActorRequired,
		},
		{
			name: "nil来源课件拒绝",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			source:  nil,
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotDomain, err :=
				ResolveCoursewareForkEducationDomain(
					test.actor,
					test.source,
				)

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf(
						"期望错误%v，实际错误%v",
						test.wantErr,
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

			if gotDomain != test.wantDomain {
				t.Fatalf(
					"期望教育域%s，实际%s",
					test.wantDomain,
					gotDomain,
				)
			}
		})
	}
}
