package services

import (
	"testing"

	"tedna/internal/models"
)

func TestBuildCWReviewCarryoverItemsUsesTeacherView(t *testing.T) {
	item := &models.CoursewareReviewItem{
		ID:                     "carryover-item-1",
		ReviewLevel:            1,
		ReviewRound:            2,
		ResubmittedReviewLevel: 1,
		ResubmittedReviewRound: 3,
		PageNumberSnapshot:     4,
		PageTitleSnapshot:      "课堂练习",
		PageHTMLHash:           "internal-page-hash",
		AppliedPageHash:        "internal-applied-hash",
		Severity:               models.CWReviewSeverityHigh,
		Dimension:              "technical_implementation",
		Title:                  "JavaScript selector failed",
		Description:            "raw technical description",
		OriginalSuggestion:     "raw technical suggestion",
		ConfirmedInstruction:   "保持课堂目标不变，修正本页操作反馈。",
		Status:                 models.CWReviewItemStatusApplied,
	}

	expected := BuildCWReviewItemTeacherView(item)
	result := buildCWReviewCarryoverItems(
		[]*models.CoursewareReviewItem{
			item,
		},
	)

	if len(result) != 1 {
		t.Fatalf(
			"复审教师视图数量错误: got=%d want=1",
			len(result),
		)
	}

	got := result[0]
	if got == nil {
		t.Fatal("复审教师视图不能为空")
	}

	if got.TeacherTitle != expected.TeacherTitle {
		t.Fatalf(
			"teacher_title未复用统一教师视图: got=%q want=%q",
			got.TeacherTitle,
			expected.TeacherTitle,
		)
	}

	if got.WhatHappened != expected.WhatHappened {
		t.Fatalf(
			"what_happened未复用统一教师视图: got=%q want=%q",
			got.WhatHappened,
			expected.WhatHappened,
		)
	}

	if got.TeachingImpact != expected.TeachingImpact {
		t.Fatalf(
			"teaching_impact未复用统一教师视图: got=%q want=%q",
			got.TeachingImpact,
			expected.TeachingImpact,
		)
	}

	if got.ImprovementGoal != expected.ImprovementGoal {
		t.Fatalf(
			"improvement_goal未复用统一教师视图: got=%q want=%q",
			got.ImprovementGoal,
			expected.ImprovementGoal,
		)
	}

	if got.TeacherContext != expected.TeacherContext {
		t.Fatalf(
			"teacher_context未复用统一教师视图: got=%q want=%q",
			got.TeacherContext,
			expected.TeacherContext,
		)
	}

	if got.ManualCheckRequired != expected.ManualCheckRequired {
		t.Fatalf(
			"manual_check_required未复用统一教师视图: got=%v want=%v",
			got.ManualCheckRequired,
			expected.ManualCheckRequired,
		)
	}

	if got.Title != got.TeacherTitle {
		t.Fatalf(
			"旧title必须是教师化兼容映射: title=%q teacher_title=%q",
			got.Title,
			got.TeacherTitle,
		)
	}

	if got.Description != got.WhatHappened {
		t.Fatalf(
			"旧description必须是教师化兼容映射: description=%q what_happened=%q",
			got.Description,
			got.WhatHappened,
		)
	}

	if got.PageHTMLHash != "" {
		t.Fatalf(
			"浏览器复审响应不应暴露page_html_hash: %q",
			got.PageHTMLHash,
		)
	}

	if got.AppliedPageHash != "" {
		t.Fatalf(
			"浏览器复审响应不应暴露applied_page_hash: %q",
			got.AppliedPageHash,
		)
	}

	if len(got.AcceptanceChecks) !=
		len(expected.AcceptanceChecks) {
		t.Fatalf(
			"acceptance_checks数量错误: got=%d want=%d",
			len(got.AcceptanceChecks),
			len(expected.AcceptanceChecks),
		)
	}

	for index := range expected.AcceptanceChecks {
		if got.AcceptanceChecks[index] !=
			expected.AcceptanceChecks[index] {
			t.Fatalf(
				"acceptance_checks[%d]不一致: got=%q want=%q",
				index,
				got.AcceptanceChecks[index],
				expected.AcceptanceChecks[index],
			)
		}
	}

	if len(got.AcceptanceChecks) > 0 {
		original :=
			expected.AcceptanceChecks[0]

		got.AcceptanceChecks[0] =
			"mutated-by-test"

		if expected.AcceptanceChecks[0] !=
			original {
			t.Fatal(
				"复审AcceptanceChecks必须复制切片，不能共享底层数组",
			)
		}
	}
}
