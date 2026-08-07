package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestResolveCoursewareComicPlanNarrativeMode(
	t *testing.T,
) {
	t.Run(
		"valid override",
		func(t *testing.T) {
			value, err :=
				resolveCoursewareComicPlanNarrativeMode(
					"  role_dialogue  ",
					models.CWComicNarrativeKnowledgeStory,
				)

			if err != nil {
				t.Fatalf(
					"合法叙事覆盖被拒绝：%v",
					err,
				)
			}

			if value !=
				models.CWComicNarrativeRoleDialogue {
				t.Fatalf(
					"叙事覆盖未规范化：%q",
					value,
				)
			}
		},
	)

	t.Run(
		"empty uses current",
		func(t *testing.T) {
			value, err :=
				resolveCoursewareComicPlanNarrativeMode(
					"",
					"  inquiry_mystery  ",
				)

			if err != nil {
				t.Fatalf(
					"沿用当前叙事被拒绝：%v",
					err,
				)
			}

			if value !=
				models.CWComicNarrativeInquiryMystery {
				t.Fatalf(
					"未沿用当前叙事：%q",
					value,
				)
			}
		},
	)

	t.Run(
		"invalid override",
		func(t *testing.T) {
			_, err :=
				resolveCoursewareComicPlanNarrativeMode(
					"unknown_story",
					models.CWComicNarrativeKnowledgeStory,
				)

			if !errors.Is(
				err,
				ErrCoursewareComicPlanInvalidRequest,
			) {
				t.Fatalf(
					"非法请求错误类型不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"invalid stored context",
		func(t *testing.T) {
			_, err :=
				resolveCoursewareComicPlanNarrativeMode(
					"",
					"legacy_unknown",
				)

			if !errors.Is(
				err,
				ErrCoursewareComicPlanContextInvalid,
			) {
				t.Fatalf(
					"非法项目上下文错误类型不正确：%v",
					err,
				)
			}
		},
	)
}
