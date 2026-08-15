package services

import (
	"strings"
	"testing"
)

func TestNormalizeLessonPlanEvidenceVerdictPolicyIgnoresSurfaceIssues(t *testing.T) {
	t.Run("markdown blank lines", func(t *testing.T) {
		verdict := &lessonPlanEvidenceVerdict{
			Pass: false,
			SourceConflicts: []string{
				"教案正文段落之间存在多处空行，违反Markdown格式规范",
			},
			Reasons: []string{
				"存在空行",
			},
			RepairInstruction: "删除空行",
		}

		normalizeLessonPlanEvidenceVerdictPolicy(verdict)

		if !verdict.Pass {
			t.Fatalf("surface-only issue should not block evidence harness: %+v", verdict)
		}
		if len(verdict.SourceConflicts) != 0 {
			t.Fatalf("surface issue should be removed from source conflicts: %+v", verdict.SourceConflicts)
		}
	})

	t.Run("mixed language typo", func(t *testing.T) {
		verdict := &lessonPlanEvidenceVerdict{
			Pass: false,
			UnsupportedModelAdditions: []string{
				`教学准备中的“相同规格 of 玻璃杯6个”中英文混杂`,
			},
		}

		normalizeLessonPlanEvidenceVerdictPolicy(verdict)

		if !verdict.Pass {
			t.Fatalf("surface-only language issue should not block evidence harness: %+v", verdict)
		}
		if len(verdict.UnsupportedModelAdditions) != 0 {
			t.Fatalf("surface issue should be removed from unsupported additions: %+v", verdict.UnsupportedModelAdditions)
		}
	})
}

func TestNormalizeLessonPlanEvidenceVerdictPolicyKeepsRealEvidenceIssue(t *testing.T) {
	verdict := &lessonPlanEvidenceVerdict{
		Pass: false,
		SourceConflicts: []string{
			"课本明确要求45分钟，候选教案写成90分钟",
		},
		RepairInstruction: "改回45分钟",
	}

	normalizeLessonPlanEvidenceVerdictPolicy(verdict)

	if verdict.Pass {
		t.Fatal("real evidence conflict must remain blocking")
	}
	if len(verdict.SourceConflicts) != 1 {
		t.Fatalf("real evidence conflict must be preserved: %+v", verdict.SourceConflicts)
	}
}

func TestLessonPlanEvidenceHarnessRejectedPublicMessageIncludesReason(t *testing.T) {
	err := errorsForLessonPlanEvidencePublicMessageTest()

	message := lessonPlanEvidenceHarnessRejectedPublicMessage(err)

	if !strings.Contains(message, "具体原因") {
		t.Fatalf("public message should expose safe business reason: %q", message)
	}
	if !strings.Contains(message, "课本明确要求45分钟") {
		t.Fatalf("public message should preserve actionable reason: %q", message)
	}
}

func errorsForLessonPlanEvidencePublicMessageTest() error {
	return &lessonPlanEvidencePublicMessageTestError{}
}

type lessonPlanEvidencePublicMessageTestError struct{}

func (*lessonPlanEvidencePublicMessageTestError) Error() string {
	return ErrLessonPlanEvidenceHarnessRejected.Error() +
		": 来源冲突：课本明确要求45分钟，候选教案写成90分钟"
}
