package repository

import (
	"testing"
	"time"

	"tedna/internal/models"
)

func TestImportedStageCompletedAt(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.August,
		5,
		9,
		30,
		0,
		0,
		time.FixedZone("CST", 8*60*60),
	)

	completedAt := importedStageCompletedAt(
		string(models.StageOutputSkipped),
		now,
	)

	if completedAt == nil {
		t.Fatal(
			"skipped阶段必须生成completed_at",
		)
	}

	if !completedAt.Equal(now) {
		t.Fatalf(
			"completed_at不一致：actual=%v expected=%v",
			*completedAt,
			now,
		)
	}

	inProgressAt := importedStageCompletedAt(
		string(models.StageOutputInProgress),
		now,
	)

	if inProgressAt != nil {
		t.Fatalf(
			"in_progress阶段不应生成completed_at：%v",
			*inProgressAt,
		)
	}
}

func TestIsEmptyImportedAIReviewJSON(
	t *testing.T,
) {
	emptyCases := []string{
		"",
		"   ",
		"null",
		"{}",
		"{ }",
	}

	for _, raw := range emptyCases {
		if !isEmptyImportedAIReviewJSON(raw) {
			t.Fatalf(
				"应识别为空评审：%q",
				raw,
			)
		}
	}

	nonEmptyCases := []string{
		`{"score": 88}`,
		`[]`,
		`"reviewed"`,
		`invalid-json`,
	}

	for _, raw := range nonEmptyCases {
		if isEmptyImportedAIReviewJSON(raw) {
			t.Fatalf(
				"不应识别为空评审：%q",
				raw,
			)
		}
	}
}
