package services

// lesson_plan_context_capsule_runtime_confirmation_test.go
//
// 验证运行时确认约束：
//   - 教师明确确认的环节进入独立防重复确认区块；
//   - 内部确认条目原文不混入普通教学方向；
//   - 已确认环节统一排序、去重；
//   - AI推断不能升级为运行时确认事实；
//   - 非active的旧条目不能进入运行时确认事实；
//   - write进入review后形成的累积确认能进入运行时上下文。

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestBuildLessonPlanContextCapsuleContextTextAddsConfirmedSectionsGuard(
	t *testing.T,
) {
	internalContent :=
		"教师已确认环节二、环节一、环节二的教案内容。"

	document :=
		&models.LessonPlanContextCapsuleDocument{
			SchemaVersion: 1,
			CourseCore: []models.LessonPlanContextCapsuleItem{
				{
					Key:       "course.hainan",
					Title:     "课程核心",
					Content:   "三年级语文《海南岛》聚焦文学阅读与创意表达。",
					State:     models.LessonPlanContextCapsuleItemStateActive,
					Authority: models.LessonPlanContextCapsuleAuthoritySourceVerified,
				},
			},
			TeachingConsensus: []models.LessonPlanContextCapsuleItem{
				{
					Key:       "consensus.stable.design",
					Title:     "教学设计主线",
					Content:   "采用旅行推荐官情境和跨主题分层写作。",
					State:     models.LessonPlanContextCapsuleItemStateActive,
					Authority: models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				},
				{
					Key:            lessonPlanCapsuleConfirmedSectionsKey,
					Title:          "已确认教案环节",
					Content:        internalContent,
					State:          models.LessonPlanContextCapsuleItemStateActive,
					Authority:      models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
					DoNotReconfirm: true,
				},
			},
			Constraints: []models.LessonPlanContextCapsuleItem{
				{
					Key:       "constraint.duration",
					Title:     "课时边界",
					Content:   "本课总时长为45分钟。",
					State:     models.LessonPlanContextCapsuleItemStateActive,
					Authority: models.LessonPlanContextCapsuleAuthoritySourceVerified,
				},
			},
		}

	runtimeText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	confirmationTitle :=
		"【已确认教案进度·不得重复确认】"

	if strings.Count(
		runtimeText,
		confirmationTitle,
	) != 1 {
		t.Fatalf(
			"独立确认区块数量异常：%q",
			runtimeText,
		)
	}

	if !strings.Contains(
		runtimeText,
		"教案环节一、环节二已经由教师确认。",
	) {
		t.Fatalf(
			"运行时上下文缺少归并后的确认范围：%q",
			runtimeText,
		)
	}

	if !strings.Contains(
		runtimeText,
		"不得再次询问教师是否确认",
	) {
		t.Fatalf(
			"运行时上下文缺少防重复确认要求：%q",
			runtimeText,
		)
	}

	// 内部条目原文不能直接进入运行时。
	if strings.Contains(
		runtimeText,
		internalContent,
	) {
		t.Fatalf(
			"内部确认条目原文不应直接注入运行时：%q",
			runtimeText,
		)
	}

	teachingStart :=
		strings.Index(
			runtimeText,
			"【已经确定的教学方向】",
		)

	confirmationStart :=
		strings.Index(
			runtimeText,
			confirmationTitle,
		)

	constraintsStart :=
		strings.Index(
			runtimeText,
			"【必须遵守的边界】",
		)

	if teachingStart < 0 ||
		confirmationStart <= teachingStart ||
		constraintsStart <= confirmationStart {
		t.Fatalf(
			"运行时区块顺序异常：%q",
			runtimeText,
		)
	}

	teachingSection :=
		runtimeText[teachingStart:confirmationStart]

	if strings.Contains(
		teachingSection,
		"已经由教师确认",
	) {
		t.Fatalf(
			"教案确认进度不应混入普通教学方向：%q",
			teachingSection,
		)
	}
}

func TestBuildLessonPlanContextCapsuleContextTextRejectsAIInferredConfirmation(
	t *testing.T,
) {
	document :=
		lessonPlanCapsuleRuntimeConfirmationTestDocument(
			models.LessonPlanContextCapsuleAuthorityAIInferred,
			models.LessonPlanContextCapsuleItemStateActive,
		)

	runtimeText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	if strings.Contains(
		runtimeText,
		"【已确认教案进度·不得重复确认】",
	) {
		t.Fatalf(
			"AI推断不能升级为运行时确认事实：%q",
			runtimeText,
		)
	}
}

func TestBuildLessonPlanContextCapsuleContextTextRejectsInactiveConfirmation(
	t *testing.T,
) {
	document :=
		lessonPlanCapsuleRuntimeConfirmationTestDocument(
			models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
			models.LessonPlanContextCapsuleItemStateSuperseded,
		)

	runtimeText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	if strings.Contains(
		runtimeText,
		"【已确认教案进度·不得重复确认】",
	) {
		t.Fatalf(
			"已替代确认条目不能进入运行时：%q",
			runtimeText,
		)
	}
}

func TestWriteReviewDecisionAddsConfirmedSectionsRuntimeGuard(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleStageDecisionTestDocument()

	document :=
		lessonPlanCapsuleStageDecisionTestDocument()

	_, _, _, changed :=
		reconcileLessonPlanContextCapsuleWriteReviewDecision(
			document,
			current,
			lessonPlanCapsuleStageDecisionDetailedContent(),
			"stage_write_confirm_1785603000002",
		)

	if !changed {
		t.Fatal(
			"write进入review后应形成结构化确认变化",
		)
	}

	runtimeText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	if !strings.Contains(
		runtimeText,
		"教案环节一、环节二、环节三、环节四已经由教师确认。",
	) {
		t.Fatalf(
			"write进入review后的运行时上下文缺少完整确认范围：%q",
			runtimeText,
		)
	}
}

func lessonPlanCapsuleRuntimeConfirmationTestDocument(
	authority string,
	state string,
) *models.LessonPlanContextCapsuleDocument {
	return &models.LessonPlanContextCapsuleDocument{
		SchemaVersion: 1,
		CourseCore: []models.LessonPlanContextCapsuleItem{
			{
				Key:       "course.hainan",
				Content:   "三年级语文《海南岛》。",
				State:     models.LessonPlanContextCapsuleItemStateActive,
				Authority: models.LessonPlanContextCapsuleAuthoritySourceVerified,
			},
		},
		TeachingConsensus: []models.LessonPlanContextCapsuleItem{
			{
				Key:            lessonPlanCapsuleConfirmedSectionsKey,
				Title:          "已确认教案环节",
				Content:        "教师已确认环节一、环节二的教案内容。",
				State:          state,
				Authority:      authority,
				DoNotReconfirm: true,
			},
		},
	}
}
