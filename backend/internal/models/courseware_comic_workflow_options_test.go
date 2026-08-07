package models

import "testing"

func TestCoursewareComicWorkflowOptions(t *testing.T) {
	narratives := []string{
		CWComicNarrativeKnowledgeStory,
		CWComicNarrativeInquiryMystery,
		CWComicNarrativeRoleDialogue,
		CWComicNarrativeTravelAdventure,
		CWComicNarrativeCivicCase,
	}

	for _, value := range narratives {
		if !IsValidCWComicNarrativeMode(value) {
			t.Fatalf(
				"合法叙事方式被拒绝：%q",
				value,
			)
		}
	}

	visualStyles := []string{
		CWComicVisualScienceEncyclopedia,
		CWComicVisualWarmStorybook,
		CWComicVisualModernFlat,
		CWComicVisualChineseInk,
		CWComicVisualCinematic3D,
		CWComicVisualRealisticIllustration,
	}

	for _, value := range visualStyles {
		if !IsValidCWComicVisualStyle(value) {
			t.Fatalf(
				"合法美术风格被拒绝：%q",
				value,
			)
		}

		if CoursewareComicVisualStyleInstruction(value) == "" {
			t.Fatalf(
				"美术风格缺少服务端说明：%q",
				value,
			)
		}
	}

	if IsValidCWComicNarrativeMode("random_story") {
		t.Fatal(
			"非法叙事方式被接受",
		)
	}

	if IsValidCWComicVisualStyle("unknown_visual") {
		t.Fatal(
			"非法美术风格被接受",
		)
	}

	if CoursewareComicVisualStyleInstruction(
		"unknown_visual",
	) != "" {
		t.Fatal(
			"非法美术风格不应产生系统说明",
		)
	}

	if CoursewareComicMaxStyleInstructionRunes != 1200 {
		t.Fatalf(
			"教师风格补充要求上限错误：%d",
			CoursewareComicMaxStyleInstructionRunes,
		)
	}
}
