package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateCoursewareReviewEducationDomain(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *CoursewareActorContext
		courseware *models.Courseware
		wantErr    error
	}{
		{
			name: "K12审核员可处理K12课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainK12,
			),
		},
		{
			name: "K12审核员不能处理职教课件",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainVocational,
			),
			wantErr: ErrCoursewareEducationDomainMismatch,
		},
		{
			name: "mixed admin可处理合法具体教学域",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainAdult,
			),
		},
		{
			name: "common课件不能进入审核",
			actor: newCoursewareAccessTestActor(
				models.RoleOperator,
				models.EducationDomainK12,
			),
			courseware: newCoursewareAccessTestResource(
				models.EducationDomainCommon,
			),
			wantErr: ErrCoursewareRuntimeDomainRequired,
		},
		{
			name: "非法课件域不能进入审核",
			actor: newCoursewareAccessTestActor(
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			courseware: newCoursewareAccessTestResource(
				"invalid",
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCoursewareReviewEducationDomain(
				test.actor,
				test.courseware,
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
				t.Fatalf("不期望错误：%v", err)
			}
		})
	}
}

func TestCoursewareReviewHistoryPolicyAllows(
	t *testing.T,
) {
	t.Parallel()

	actor := newCoursewareAccessTestActor(
		models.RoleOperator,
		models.EducationDomainK12,
	)
	courseware := newCoursewareAccessTestResource(
		models.EducationDomainK12,
	)

	local := *courseware
	local.UserID = actor.UserID

	if !coursewareReviewHistoryPolicyAllows(
		&local,
		actor,
		false,
	) {
		t.Fatal("作者应可查看自己的审核历史")
	}

	if !coursewareReviewHistoryPolicyAllows(
		courseware,
		actor,
		true,
	) {
		t.Fatal("合法审核员应可查看审核历史")
	}

	if coursewareReviewHistoryPolicyAllows(
		courseware,
		actor,
		false,
	) {
		t.Fatal("无关用户不应查看审核历史")
	}
}
