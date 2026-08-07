package models

import "testing"

func TestCoursewareComicWorkflowContract(
	t *testing.T,
) {
	t.Run(
		"default state",
		func(t *testing.T) {
			state :=
				NewDefaultCoursewareComicWorkflowState(
					" project-1 ",
				)

			if state.ProjectID != "project-1" {
				t.Fatalf(
					"项目ID未规范化：%q",
					state.ProjectID,
				)
			}

			if state.Stage !=
				CWComicWorkflowSource {
				t.Fatalf(
					"新项目步骤错误：%q",
					state.Stage,
				)
			}

			if state.AspectRatio !=
				CWComicAspectRatioCourseware {
				t.Fatalf(
					"默认比例错误：%q",
					state.AspectRatio,
				)
			}

			if state.ImageQuality !=
				CWComicImageQualityHigh {
				t.Fatalf(
					"默认清晰度错误：%q",
					state.ImageQuality,
				)
			}

			if state.InsertionMode !=
				CWComicInsertionSinglePage {
				t.Fatalf(
					"默认使用方式错误：%q",
					state.InsertionMode,
				)
			}

			if state.StyleInstruction != "" {
				t.Fatal(
					"默认风格补充要求应为空",
				)
			}
		},
	)

	t.Run(
		"valid enumerations",
		func(t *testing.T) {
			stages := []string{
				CWComicWorkflowSource,
				CWComicWorkflowStoryboard,
				CWComicWorkflowStylePreview,
				CWComicWorkflowBatchGeneration,
				CWComicWorkflowRefinement,
			}

			for _, value := range stages {
				if !IsValidCWComicWorkflowStage(
					value,
				) {
					t.Fatalf(
						"合法工作流步骤被拒绝：%q",
						value,
					)
				}
			}

			ratios := []string{
				CWComicAspectRatioCourseware,
				CWComicAspectRatio16x9,
				CWComicAspectRatio4x3,
				CWComicAspectRatio1x1,
				CWComicAspectRatio3x4,
				CWComicAspectRatio9x16,
			}

			for _, value := range ratios {
				if !IsValidCWComicAspectRatio(
					value,
				) {
					t.Fatalf(
						"合法图片比例被拒绝：%q",
						value,
					)
				}
			}

			qualities := []string{
				CWComicImageQualityStandard,
				CWComicImageQualityHigh,
			}

			for _, value := range qualities {
				if !IsValidCWComicImageQuality(
					value,
				) {
					t.Fatalf(
						"合法清晰度被拒绝：%q",
						value,
					)
				}
			}

			modes := []string{
				CWComicInsertionSinglePage,
				CWComicInsertionSmartPages,
				CWComicInsertionOnePanelPerPage,
				CWComicInsertionLibraryOnly,
			}

			for _, value := range modes {
				if !IsValidCWComicInsertionMode(
					value,
				) {
					t.Fatalf(
						"合法使用方式被拒绝：%q",
						value,
					)
				}
			}
		},
	)

	t.Run(
		"invalid enumerations",
		func(t *testing.T) {
			if IsValidCWComicWorkflowStage(
				"generate_everything",
			) {
				t.Fatal(
					"非法工作流步骤被接受",
				)
			}

			if IsValidCWComicAspectRatio(
				"2:1",
			) {
				t.Fatal(
					"非法图片比例被接受",
				)
			}

			if IsValidCWComicImageQuality(
				"ultra",
			) {
				t.Fatal(
					"非法清晰度被接受",
				)
			}

			if IsValidCWComicInsertionMode(
				"overwrite_courseware",
			) {
				t.Fatal(
					"非法使用方式被接受",
				)
			}
		},
	)

	t.Run(
		"normalization",
		func(t *testing.T) {
			emptyPanelID := "  "

			normalized, ok :=
				NormalizeCoursewareComicWorkflowState(
					&CoursewareComicWorkflowState{
						ProjectID:           " project-2 ",
						Stage:               " ",
						StylePreviewPanelID: &emptyPanelID,
						AspectRatio:         " ",
						ImageQuality:        " ",
						InsertionMode:       " ",
						StyleInstruction:    "  颜色更明亮，人物更像中学生  ",
					},
				)

			if !ok || normalized == nil {
				t.Fatal(
					"工作流状态规范化失败",
				)
			}

			if normalized.ProjectID !=
				"project-2" {
				t.Fatalf(
					"项目ID规范化错误：%q",
					normalized.ProjectID,
				)
			}

			if normalized.Stage !=
				CWComicWorkflowSource {
				t.Fatalf(
					"默认步骤规范化错误：%q",
					normalized.Stage,
				)
			}

			if normalized.AspectRatio !=
				CWComicAspectRatioCourseware {
				t.Fatalf(
					"默认比例规范化错误：%q",
					normalized.AspectRatio,
				)
			}

			if normalized.ImageQuality !=
				CWComicImageQualityHigh {
				t.Fatalf(
					"默认清晰度规范化错误：%q",
					normalized.ImageQuality,
				)
			}

			if normalized.InsertionMode !=
				CWComicInsertionSinglePage {
				t.Fatalf(
					"默认使用方式规范化错误：%q",
					normalized.InsertionMode,
				)
			}

			if normalized.StylePreviewPanelID !=
				nil {
				t.Fatal(
					"空白样张分格ID应规范化为nil",
				)
			}

			if normalized.StyleInstruction !=
				"颜色更明亮，人物更像中学生" {
				t.Fatalf(
					"风格补充要求规范化错误：%q",
					normalized.StyleInstruction,
				)
			}
		},
	)

	t.Run(
		"invalid state",
		func(t *testing.T) {
			normalized, ok :=
				NormalizeCoursewareComicWorkflowState(
					&CoursewareComicWorkflowState{
						ProjectID: "project-invalid",
						Stage:     "invalid-stage",
					},
				)

			if ok || normalized != nil {
				t.Fatal(
					"非法工作流状态不应通过规范化",
				)
			}
		},
	)
}
