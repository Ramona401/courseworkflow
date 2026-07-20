package services

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
)

func subtitleStringPtr(
	value string,
) *string {
	return &value
}

func TestValidateCoursewareEditorDraftSubtitleOwnerLegacy(
	t *testing.T,
) {
	courseware := &models.Courseware{
		ID:     "courseware-1",
		UserID: "owner-1",
	}

	t.Run(
		"创建者本人可以管理历史空scope字幕",
		func(t *testing.T) {
			err := validateCoursewareEditorDraftSubtitleOwner(
				context.Background(),
				courseware,
				&CoursewareActorContext{
					UserID: "user-1",
				},
				&models.CoursewareSubtitle{
					CoursewareID: "courseware-1",
					ScopeType:    models.SubScopeEditorDraft,
					CreatedBy:    subtitleStringPtr("user-1"),
				},
			)
			if err != nil {
				t.Fatalf(
					"本人字幕不应被拒绝: %v",
					err,
				)
			}
		},
	)

	t.Run(
		"协作者不能管理他人历史空scope字幕",
		func(t *testing.T) {
			err := validateCoursewareEditorDraftSubtitleOwner(
				context.Background(),
				courseware,
				&CoursewareActorContext{
					UserID: "user-2",
				},
				&models.CoursewareSubtitle{
					CoursewareID: "courseware-1",
					ScopeType:    models.SubScopeEditorDraft,
					CreatedBy:    subtitleStringPtr("user-1"),
				},
			)
			if !errors.Is(
				err,
				ErrCoursewareSubtitleScopeTargetMismatch,
			) {
				t.Fatalf(
					"他人字幕应被拒绝: %v",
					err,
				)
			}
		},
	)

	t.Run(
		"课件作者兼容管理created_by为空的最早期字幕",
		func(t *testing.T) {
			err := validateCoursewareEditorDraftSubtitleOwner(
				context.Background(),
				courseware,
				&CoursewareActorContext{
					UserID: "owner-1",
				},
				&models.CoursewareSubtitle{
					CoursewareID: "courseware-1",
					ScopeType:    models.SubScopeEditorDraft,
				},
			)
			if err != nil {
				t.Fatalf(
					"课件作者兼容路径不应失败: %v",
					err,
				)
			}
		},
	)

	t.Run(
		"非作者不能管理created_by为空的遗留字幕",
		func(t *testing.T) {
			err := validateCoursewareEditorDraftSubtitleOwner(
				context.Background(),
				courseware,
				&CoursewareActorContext{
					UserID: "user-2",
				},
				&models.CoursewareSubtitle{
					CoursewareID: "courseware-1",
					ScopeType:    models.SubScopeEditorDraft,
				},
			)
			if !errors.Is(
				err,
				ErrCoursewareSubtitleScopeTargetMismatch,
			) {
				t.Fatalf(
					"非作者遗留字幕应被拒绝: %v",
					err,
				)
			}
		},
	)
}
