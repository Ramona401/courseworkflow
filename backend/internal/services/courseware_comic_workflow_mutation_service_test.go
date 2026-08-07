package services

import (
	"errors"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestNormalizeCoursewareComicWorkflowMutationRequests(
	t *testing.T,
) {
	t.Run(
		"valid narrative confirmation",
		func(t *testing.T) {
			narrativeMode, err :=
				normalizeCoursewareComicNarrativeConfirmation(
					&models.ConfirmCoursewareComicStoryboardRequest{
						ExpectedVersion: 7,
						NarrativeMode:   "  role_dialogue  ",
					},
				)

			if err != nil {
				t.Fatalf(
					"合法叙事确认被拒绝：%v",
					err,
				)
			}

			if narrativeMode !=
				models.CWComicNarrativeRoleDialogue {
				t.Fatalf(
					"叙事方式未规范化：%q",
					narrativeMode,
				)
			}
		},
	)

	t.Run(
		"invalid narrative confirmation",
		func(t *testing.T) {
			requests :=
				[]*models.ConfirmCoursewareComicStoryboardRequest{
					nil,
					{
						ExpectedVersion: 0,
						NarrativeMode:   models.CWComicNarrativeKnowledgeStory,
					},
					{
						ExpectedVersion: 1,
						NarrativeMode:   "unsupported_story",
					},
				}

			for index, request := range requests {
				_, err :=
					normalizeCoursewareComicNarrativeConfirmation(
						request,
					)

				if !errors.Is(
					err,
					ErrCoursewareComicWorkflowInvalidRequest,
				) {
					t.Fatalf(
						"第%d个非法叙事请求错误类型不正确：%v",
						index+1,
						err,
					)
				}
			}
		},
	)

	t.Run(
		"valid style settings",
		func(t *testing.T) {
			settings, err :=
				normalizeCoursewareComicStyleSettings(
					&models.UpdateCoursewareComicStyleSettingsRequest{
						ExpectedVersion:  9,
						VisualStyle:      "  modern_flat  ",
						AspectRatio:      " 4:3 ",
						ImageQuality:     " high ",
						StyleInstruction: "  色彩更明亮，人物保持中学生特征  ",
					},
				)

			if err != nil {
				t.Fatalf(
					"合法视觉设置被拒绝：%v",
					err,
				)
			}

			if settings == nil {
				t.Fatal(
					"合法视觉设置返回空结果",
				)
			}

			if settings.VisualStyle !=
				models.CWComicVisualModernFlat {
				t.Fatalf(
					"美术风格未规范化：%q",
					settings.VisualStyle,
				)
			}

			if settings.AspectRatio !=
				models.CWComicAspectRatio4x3 {
				t.Fatalf(
					"图片比例未规范化：%q",
					settings.AspectRatio,
				)
			}

			if settings.ImageQuality !=
				models.CWComicImageQualityHigh {
				t.Fatalf(
					"图片清晰度未规范化：%q",
					settings.ImageQuality,
				)
			}

			if settings.StyleInstruction !=
				"色彩更明亮，人物保持中学生特征" {
				t.Fatalf(
					"风格补充要求未规范化：%q",
					settings.StyleInstruction,
				)
			}
		},
	)

	t.Run(
		"empty style instruction allowed",
		func(t *testing.T) {
			settings, err :=
				normalizeCoursewareComicStyleSettings(
					&models.UpdateCoursewareComicStyleSettingsRequest{
						ExpectedVersion:  1,
						VisualStyle:      models.CWComicVisualScienceEncyclopedia,
						AspectRatio:      models.CWComicAspectRatioCourseware,
						ImageQuality:     models.CWComicImageQualityStandard,
						StyleInstruction: "   ",
					},
				)

			if err != nil {
				t.Fatalf(
					"空白风格补充要求不应被拒绝：%v",
					err,
				)
			}

			if settings.StyleInstruction != "" {
				t.Fatalf(
					"空白补充要求应规范化为空：%q",
					settings.StyleInstruction,
				)
			}
		},
	)

	t.Run(
		"invalid style enumerations",
		func(t *testing.T) {
			requests :=
				[]*models.UpdateCoursewareComicStyleSettingsRequest{
					nil,
					{
						ExpectedVersion: 0,
						VisualStyle:     models.CWComicVisualModernFlat,
						AspectRatio:     models.CWComicAspectRatio16x9,
						ImageQuality:    models.CWComicImageQualityHigh,
					},
					{
						ExpectedVersion: 1,
						VisualStyle:     "unknown_style",
						AspectRatio:     models.CWComicAspectRatio16x9,
						ImageQuality:    models.CWComicImageQualityHigh,
					},
					{
						ExpectedVersion: 1,
						VisualStyle:     models.CWComicVisualModernFlat,
						AspectRatio:     "2:1",
						ImageQuality:    models.CWComicImageQualityHigh,
					},
					{
						ExpectedVersion: 1,
						VisualStyle:     models.CWComicVisualModernFlat,
						AspectRatio:     models.CWComicAspectRatio16x9,
						ImageQuality:    "ultra",
					},
				}

			for index, request := range requests {
				_, err :=
					normalizeCoursewareComicStyleSettings(
						request,
					)

				if !errors.Is(
					err,
					ErrCoursewareComicWorkflowInvalidRequest,
				) {
					t.Fatalf(
						"第%d个非法视觉设置错误类型不正确：%v",
						index+1,
						err,
					)
				}
			}
		},
	)

	t.Run(
		"style instruction exact limit",
		func(t *testing.T) {
			instruction :=
				strings.Repeat(
					"画",
					models.CoursewareComicMaxStyleInstructionRunes,
				)

			settings, err :=
				normalizeCoursewareComicStyleSettings(
					&models.UpdateCoursewareComicStyleSettingsRequest{
						ExpectedVersion:  1,
						VisualStyle:      models.CWComicVisualWarmStorybook,
						AspectRatio:      models.CWComicAspectRatio1x1,
						ImageQuality:     models.CWComicImageQualityHigh,
						StyleInstruction: instruction,
					},
				)

			if err != nil {
				t.Fatalf(
					"恰好达到字符上限不应失败：%v",
					err,
				)
			}

			if settings.StyleInstruction !=
				instruction {
				t.Fatal(
					"字符上限内的补充要求被错误修改",
				)
			}
		},
	)

	t.Run(
		"style instruction over limit",
		func(t *testing.T) {
			instruction :=
				strings.Repeat(
					"画",
					models.CoursewareComicMaxStyleInstructionRunes+1,
				)

			_, err :=
				normalizeCoursewareComicStyleSettings(
					&models.UpdateCoursewareComicStyleSettingsRequest{
						ExpectedVersion:  1,
						VisualStyle:      models.CWComicVisualWarmStorybook,
						AspectRatio:      models.CWComicAspectRatio1x1,
						ImageQuality:     models.CWComicImageQualityHigh,
						StyleInstruction: instruction,
					},
				)

			if !errors.Is(
				err,
				ErrCoursewareComicStyleInstructionTooLong,
			) {
				t.Fatalf(
					"超限补充要求错误类型不正确：%v",
					err,
				)
			}
		},
	)
}
