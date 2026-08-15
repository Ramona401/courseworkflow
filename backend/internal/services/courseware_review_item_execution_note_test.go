package services

import (
	"errors"
	"strings"
	"testing"

	"tedna/internal/models"
)

func deliveredFormalExecutionNoteItem() *models.CoursewareReviewItem {
	reviewID := "review-1"
	feedbackID := "feedback-1"
	deliveredVersionID := "version-1"

	return &models.CoursewareReviewItem{
		ID:                            "item-1",
		OwnerID:                       "owner-1",
		SourceType:                    models.CWReviewItemSourceFormal,
		CoursewareReviewID:            &reviewID,
		FeedbackID:                    &feedbackID,
		DeliveredInstructionVersionID: &deliveredVersionID,
		Status:                        models.CWReviewItemStatusConfirmed,
	}
}

func TestNormalizeCWReviewItemExecutionNote(t *testing.T) {
	normalized, err :=
		normalizeCWReviewItemExecutionNote(
			"  已按审核要求调整，并补充了课堂操作检查。  ",
		)
	if err != nil {
		t.Fatalf("有效执行补充不应失败: %v", err)
	}
	if normalized !=
		"已按审核要求调整，并补充了课堂操作检查。" {
		t.Fatalf("执行补充没有正确去除首尾空白: %q", normalized)
	}

	if _, err :=
		normalizeCWReviewItemExecutionNote("   "); !errors.Is(
		err,
		ErrCWReviewItemExecutionNoteInvalid,
	) {
		t.Fatalf("空执行补充应被拒绝: %v", err)
	}

	oversized :=
		strings.Repeat(
			"补",
			cwReviewItemMaxExecutionNoteRunes+1,
		)

	if _, err :=
		normalizeCWReviewItemExecutionNote(oversized); !errors.Is(
		err,
		ErrCWReviewItemExecutionNoteInvalid,
	) {
		t.Fatalf("超长执行补充应被拒绝: %v", err)
	}
}

func TestValidateCWReviewItemExecutionNote(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.CoursewareReviewItem)
		actorID string
		wantErr error
	}{
		{
			name:    "delivered formal owner allowed",
			actorID: "owner-1",
		},
		{
			name:    "non owner forbidden",
			actorID: "reviewer-1",
			wantErr: ErrCWReviewItemExecutionNoteForbidden,
		},
		{
			name: "self item unavailable",
			mutate: func(item *models.CoursewareReviewItem) {
				item.SourceType =
					models.CWReviewItemSourceSelf
			},
			actorID: "owner-1",
			wantErr: ErrCWReviewItemExecutionNoteUnavailable,
		},
		{
			name: "undelivered formal unavailable",
			mutate: func(item *models.CoursewareReviewItem) {
				item.DeliveredInstructionVersionID = nil
			},
			actorID: "owner-1",
			wantErr: ErrCWReviewItemExecutionNoteUnavailable,
		},
		{
			name: "resolved history frozen",
			mutate: func(item *models.CoursewareReviewItem) {
				item.Status =
					models.CWReviewItemStatusResolved
			},
			actorID: "owner-1",
			wantErr: ErrCWReviewItemExecutionNoteUnavailable,
		},
		{
			name: "stale item still allows explanation",
			mutate: func(item *models.CoursewareReviewItem) {
				item.Status =
					models.CWReviewItemStatusStale
			},
			actorID: "owner-1",
		},
		{
			name: "orphaned item still allows explanation",
			mutate: func(item *models.CoursewareReviewItem) {
				item.Status =
					models.CWReviewItemStatusOrphaned
			},
			actorID: "owner-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := deliveredFormalExecutionNoteItem()

			if test.mutate != nil {
				test.mutate(item)
			}

			err :=
				validateCWReviewItemExecutionNote(
					item,
					test.actorID,
				)

			if test.wantErr == nil {
				if err != nil {
					t.Fatalf(
						"预期允许执行补充，实际错误: %v",
						err,
					)
				}
				return
			}

			if !errors.Is(err, test.wantErr) {
				t.Fatalf(
					"错误类型不符合预期: got=%v want=%v",
					err,
					test.wantErr,
				)
			}
		})
	}
}
