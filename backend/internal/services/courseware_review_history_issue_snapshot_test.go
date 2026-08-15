package services

import (
	"slices"
	"testing"

	"tedna/internal/models"
)

func TestBuildCWReviewHistoryTeacherViewSnapshotKeepsFrozenEvidence(
	t *testing.T,
) {
	expectedChecks := []string{
		"问题提出后不会立即显示最终结论",
		"课堂上能留出明确的学生思考时间",
		"教师可以根据学生回答再推进到结论",
	}

	item := &models.CoursewareReviewItem{
		EvidenceJSON: `{
			"teacher_view_schema_version": 1,
			"teacher_view_snapshot": {
				"teacher_title": "让学生先思考，再出现结论",
				"what_happened": "页面立即展示了结论。",
				"teaching_impact": "学生缺少独立判断时间。",
				"improvement_goal": "先形成判断，再逐步呈现结论。",
				"acceptance_checks": [
					"问题提出后不会立即显示最终结论",
					"课堂上能留出明确的学生思考时间",
					"教师可以根据学生回答再推进到结论"
				],
				"teacher_context": "",
				"manual_check_required": true
			}
		}`,
		Status: models.CWReviewItemStatusConfirmed,
	}

	snapshot :=
		buildCWReviewHistoryTeacherViewSnapshot(
			item,
		)

	if snapshot.TeacherTitle !=
		"让学生先思考，再出现结论" {
		t.Fatalf(
			"历史教师标题被重算: %q",
			snapshot.TeacherTitle,
		)
	}

	if !snapshot.ManualCheckRequired {
		t.Fatal(
			"历史人工检查标记丢失",
		)
	}

	if !slices.Equal(
		snapshot.AcceptanceChecks,
		expectedChecks,
	) {
		t.Fatalf(
			"历史AcceptanceChecks被当前规则重算: %+v",
			snapshot.AcceptanceChecks,
		)
	}

	if len(snapshot.AcceptanceChecks) != 3 {
		t.Fatalf(
			"冻结历史应保持3条检查项，got=%d",
			len(snapshot.AcceptanceChecks),
		)
	}
}
