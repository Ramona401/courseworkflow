package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func newOwnerRuntimeActor(
	userID string,
	role string,
	domain string,
) *CoursewareActorContext {
	return &CoursewareActorContext{
		UserID:          userID,
		Role:            role,
		EducationDomain: domain,
	}
}

func newOwnerRuntimeCourseware(
	ownerID string,
	domain string,
) *models.Courseware {
	return &models.Courseware{
		ID:              "cw-owner-runtime-test",
		UserID:          ownerID,
		EducationDomain: domain,
		Status:          models.CoursewareStatusPreview,
	}
}

func TestCanOperateOwnedCourseware(t *testing.T) {
	service := &CoursewareService{}

	tests := []struct {
		name        string
		courseware  *models.Courseware
		actor       *CoursewareActorContext
		wantAllowed bool
		wantErr     error
	}{
		{
			name: "same domain owner allowed",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainK12,
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantAllowed: true,
		},
		{
			name: "historical snapshot owner allowed after domain change",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainK12,
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainVocational,
			),
			wantAllowed: true,
		},
		{
			name: "admin is not owner",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainK12,
			),
			actor: newOwnerRuntimeActor(
				"admin-1",
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			wantAllowed: false,
		},
		{
			name: "collaborator is not owner",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainK12,
			),
			actor: newOwnerRuntimeActor(
				"collab-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantAllowed: false,
		},
		{
			name: "common resource cannot run",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainCommon,
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareRuntimeDomainRequired,
		},
		{
			name: "mixed resource cannot run",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainMixed,
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				models.RoleAdmin,
				models.EducationDomainMixed,
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "empty resource domain is invalid",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				"",
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "unknown resource domain is invalid",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				"unknown-domain",
			),
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
		{
			name: "nil actor rejected",
			courseware: newOwnerRuntimeCourseware(
				"owner-1",
				models.EducationDomainK12,
			),
			actor:   nil,
			wantErr: ErrCoursewareActorRequired,
		},
		{
			name:       "nil courseware rejected",
			courseware: nil,
			actor: newOwnerRuntimeActor(
				"owner-1",
				"teacher",
				models.EducationDomainK12,
			),
			wantErr: ErrCoursewareEducationDomainInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			allowed, err :=
				service.CanOperateOwnedCourseware(
					testCase.courseware,
					testCase.actor,
				)

			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf(
						"expected error %v, got %v",
						testCase.wantErr,
						err,
					)
				}
				if allowed {
					t.Fatalf(
						"expected denied when error occurs",
					)
				}
				return
			}

			if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}
			if allowed != testCase.wantAllowed {
				t.Fatalf(
					"expected allowed=%v, got %v",
					testCase.wantAllowed,
					allowed,
				)
			}
		})
	}
}

func TestCoursewareOwnerRuntimePolicyAllows(t *testing.T) {
	owner := newOwnerRuntimeActor(
		"owner-1",
		"teacher",
		models.EducationDomainAdult,
	)

	resource := newOwnerRuntimeCourseware(
		"owner-1",
		models.EducationDomainK12,
	)

	if !coursewareOwnerRuntimePolicyAllows(
		resource,
		owner,
	) {
		t.Fatalf(
			"historical owner snapshot should be allowed",
		)
	}

	resource.UserID = "other-owner"
	if coursewareOwnerRuntimePolicyAllows(
		resource,
		owner,
	) {
		t.Fatalf(
			"non-owner must not enter owner runtime channel",
		)
	}

	resource.UserID = "owner-1"
	resource.EducationDomain =
		models.EducationDomainCommon
	if coursewareOwnerRuntimePolicyAllows(
		resource,
		owner,
	) {
		t.Fatalf(
			"common resource must not enter runtime channel",
		)
	}
}
