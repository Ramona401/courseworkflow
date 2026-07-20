package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestValidateCoursewareControlMutationState(
	t *testing.T,
) {
	tests := []struct {
		name       string
		courseware *models.Courseware
		wantErr    error
	}{
		{
			name: "预览私有课件允许修改",
			courseware: &models.Courseware{
				Status:       models.CoursewareStatusPreview,
				PublishState: models.CWPublishPrivate,
			},
		},
		{
			name: "Pipeline审核锁拒绝",
			courseware: &models.Courseware{
				Status:       models.CoursewareStatusInPipeline,
				PublishState: models.CWPublishPrivate,
			},
			wantErr: ErrCoursewareControlMutationLocked,
		},
		{
			name: "发布审核锁拒绝",
			courseware: &models.Courseware{
				Status:       models.CoursewareStatusPreview,
				PublishState: models.CWPublishSubmitted,
			},
			wantErr: ErrCoursewareControlMutationLocked,
		},
		{
			name:       "空课件拒绝",
			courseware: nil,
			wantErr:    ErrCoursewareEducationDomainInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateCoursewareControlMutationState(
				testCase.courseware,
			)

			if testCase.wantErr == nil {
				if err != nil {
					t.Fatalf(
						"不期望错误，实际：%v",
						err,
					)
				}
				return
			}

			if !errors.Is(
				err,
				testCase.wantErr,
			) {
				t.Fatalf(
					"期望错误%v，实际%v",
					testCase.wantErr,
					err,
				)
			}
		})
	}
}
