package services

import (
	"testing"
	"time"

	"tedna/internal/models"
)

func stringPointer(value string) *string {
	return &value
}

func TestCWReviewHistoryUUID(t *testing.T) {
	valid :=
		"123e4567-e89b-12d3-a456-426614174000"

	if !isCWReviewHistoryUUID(valid) {
		t.Fatalf("expected valid UUID")
	}

	for _, invalid := range []string{
		"",
		"records",
		"123e4567-e89b-12d3-a456-42661417400",
		"123e4567-e89b-12d3-a456-42661417400z",
		"123e4567/e89b-12d3-a456-426614174000",
	} {
		if isCWReviewHistoryUUID(invalid) {
			t.Fatalf(
				"unexpected valid UUID: %q",
				invalid,
			)
		}
	}
}

func TestCWReviewHistoryActualLessonMaterialUsageNoLesson(
	t *testing.T,
) {
	session :=
		&models.CoursewareAIReviewSession{
			ContextManifestJSON: `{}`,
		}

	config :=
		&CWAIReviewConfigSnapshot{
			LessonReferenceMode: models.CWAIReviewLessonReferenceNoLesson,
		}

	got :=
		cwReviewHistoryActualLessonMaterialUsage(
			session,
			config,
		)

	if got == nil || *got {
		t.Fatalf(
			"no_lesson must prove lesson materials were not used",
		)
	}
}

func TestCWReviewHistoryActualLessonMaterialUsageIncluded(
	t *testing.T,
) {
	session :=
		&models.CoursewareAIReviewSession{
			ContextManifestJSON: `{
				"lesson_material_usage": {
					"lesson_content_included": true,
					"course_outline_included": false,
					"alignment_report_included": false
				},
				"lesson_plan": {
					"available": true,
					"used": true
				}
			}`,
		}

	config :=
		&CWAIReviewConfigSnapshot{
			LessonReferenceMode: models.CWAIReviewLessonReferenceCurrentCompatible,
		}

	got :=
		cwReviewHistoryActualLessonMaterialUsage(
			session,
			config,
		)

	if got == nil || !*got {
		t.Fatalf(
			"immutable manifest proves lesson material was used",
		)
	}
}

func TestCWReviewHistoryActualLessonMaterialUsageAvailableButUnused(
	t *testing.T,
) {
	session :=
		&models.CoursewareAIReviewSession{
			ContextManifestJSON: `{
				"lesson_material_usage": {
					"lesson_content_included": false,
					"course_outline_included": false,
					"alignment_report_included": false
				},
				"lesson_plan": {
					"available": false,
					"used": true
				}
			}`,
		}

	config :=
		&CWAIReviewConfigSnapshot{
			LessonReferenceMode: models.CWAIReviewLessonReferenceCurrentCompatible,
		}

	got :=
		cwReviewHistoryActualLessonMaterialUsage(
			session,
			config,
		)

	if got == nil {
		t.Fatalf(
			"manifest has explicit material usage facts",
		)
	}

	if *got {
		t.Fatalf(
			"allowing lesson materials must not be reported as actual use",
		)
	}
}

func TestCWReviewHistoryActualLessonMaterialUsageUnknownLegacy(
	t *testing.T,
) {
	session :=
		&models.CoursewareAIReviewSession{
			ContextManifestJSON: `{}`,
		}

	config :=
		&CWAIReviewConfigSnapshot{
			LessonReferenceMode: models.CWAIReviewLessonReferenceCurrentCompatible,
		}

	got :=
		cwReviewHistoryActualLessonMaterialUsage(
			session,
			config,
		)

	if got != nil {
		t.Fatalf(
			"legacy manifest without facts must remain unknown",
		)
	}
}

func TestFreezeCWReviewItemAtDeliveryDoesNotMutateCurrentItem(
	t *testing.T,
) {
	deliveredID := "version-1"
	currentID := "version-2"
	appliedID := "version-3"
	resolvedBy := "reviewer-2"
	resolvedReviewID := "review-2"

	now := time.Now()

	item :=
		&models.CoursewareReviewItem{
			Status: models.CWReviewItemStatusResolved,

			CurrentInstructionVersionID: &currentID,

			DeliveredInstructionVersionID: &deliveredID,

			AppliedInstructionVersionID: &appliedID,

			AppliedPageHash: "new-page-hash",
			AppliedAt:       &now,

			ResubmittedAt:          &now,
			ResubmittedReviewLevel: 1,
			ResubmittedReviewRound: 2,

			ResolvedBy:          &resolvedBy,
			ResolvedReviewID:    &resolvedReviewID,
			ResolvedReviewLevel: 1,
			ResolvedReviewRound: 2,
			ResolutionNote:      "later resolution",
			ResolvedAt:          &now,
		}

	frozen :=
		freezeCWReviewItemAtDelivery(
			item,
		)

	if frozen == item {
		t.Fatalf(
			"delivery view must use a copy",
		)
	}

	if frozen.Status !=
		models.CWReviewItemStatusConfirmed {
		t.Fatalf(
			"expected delivery-time confirmed state, got %s",
			frozen.Status,
		)
	}

	if frozen.CurrentInstructionVersionID == nil ||
		*frozen.CurrentInstructionVersionID !=
			deliveredID {
		t.Fatalf(
			"history teacher view must anchor current version to delivered version",
		)
	}

	if frozen.AppliedInstructionVersionID != nil ||
		frozen.AppliedAt != nil ||
		frozen.AppliedPageHash != "" ||
		frozen.ResubmittedAt != nil ||
		frozen.ResolvedBy != nil ||
		frozen.ResolvedReviewID != nil ||
		frozen.ResolvedAt != nil ||
		frozen.ResolutionNote != "" {
		t.Fatalf(
			"later remediation state leaked into delivery-time copy",
		)
	}

	if item.Status !=
		models.CWReviewItemStatusResolved {
		t.Fatalf(
			"freezing history mutated current database model copy",
		)
	}

	if item.CurrentInstructionVersionID == nil ||
		*item.CurrentInstructionVersionID !=
			currentID {
		t.Fatalf(
			"freezing history mutated current instruction version",
		)
	}
}

func TestCWReviewHistoryExecutionNoteMarker(t *testing.T) {
	if !isCWReviewHistoryExecutionNote(
		`{"event":"owner_execution_note"}`,
	) {
		t.Fatalf(
			"owner execution note marker not recognized",
		)
	}

	for _, raw := range []string{
		`{}`,
		`{"event":"confirmed"}`,
		`{"event":"owner_execution_note_other"}`,
		`not-json`,
	} {
		if isCWReviewHistoryExecutionNote(raw) {
			t.Fatalf(
				"unexpected execution note marker: %s",
				raw,
			)
		}
	}
}

func TestValidateCWReviewHistoryFeedback(t *testing.T) {
	review :=
		&models.CoursewareReview{
			ID:           "review-1",
			CoursewareID: "courseware-1",
			ReviewLevel:  1,
			ReviewRound:  2,
			Decision:     models.ReviewDecisionRevision,
		}

	feedback :=
		&models.CoursewareReviewFeedback{
			CoursewareReviewID: "review-1",
			CoursewareID:       "courseware-1",
			ReviewLevel:        1,
			ReviewRound:        2,
			Decision:           models.ReviewDecisionRevision,
		}

	if err :=
		validateCWReviewHistoryFeedback(
			review,
			feedback,
		); err != nil {
		t.Fatalf(
			"matching feedback rejected: %v",
			err,
		)
	}

	feedback.CoursewareID =
		"other-courseware"

	if err :=
		validateCWReviewHistoryFeedback(
			review,
			feedback,
		); err == nil {
		t.Fatalf(
			"mismatched feedback relation must fail closed",
		)
	}
}

func TestValidateCWReviewHistoryDeliveredItem(t *testing.T) {
	reviewID := "review-1"
	feedbackID := "feedback-1"
	sessionID := "session-1"

	courseware :=
		&models.Courseware{
			ID:     "courseware-1",
			UserID: "author-1",
		}

	review :=
		&models.CoursewareReview{
			ID:           reviewID,
			CoursewareID: courseware.ID,
			ReviewerID:   "reviewer-1",
			ReviewLevel:  1,
			ReviewRound:  2,
		}

	feedback :=
		&models.CoursewareReviewFeedback{
			ID:                 feedbackID,
			CoursewareReviewID: reviewID,
			CoursewareID:       courseware.ID,
			AIReviewSessionID:  &sessionID,
			ReviewLevel:        1,
			ReviewRound:        2,
		}

	item :=
		&models.CoursewareReviewItem{
			ID:                 "item-1",
			CoursewareID:       courseware.ID,
			SourceSessionID:    sessionID,
			CoursewareReviewID: &reviewID,
			FeedbackID:         &feedbackID,
			SourceType:         models.CWReviewItemSourceFormal,
			ReviewLevel:        1,
			ReviewRound:        2,
			CreatedBy:          "reviewer-1",
			OwnerID:            "author-1",
		}

	if err :=
		validateCWReviewHistoryDeliveredItem(
			courseware,
			review,
			feedback,
			item,
		); err != nil {
		t.Fatalf(
			"valid delivered item rejected: %v",
			err,
		)
	}

	item.CreatedBy =
		"other-reviewer"

	if err :=
		validateCWReviewHistoryDeliveredItem(
			courseware,
			review,
			feedback,
			item,
		); err == nil {
		t.Fatalf(
			"cross-reviewer item relation must fail closed",
		)
	}
}
