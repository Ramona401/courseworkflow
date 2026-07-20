package services

import (
	"testing"

	"tedna/internal/models"
)

// TestApplyLessonPlanEducationDomainToAssistantActor
// 锁定具体教案运行时的教育域覆盖规则。
//
// 教案education_domain是资源创建时快照，是具体教学运行的事实依据。
// 登录用户当前组织域或mixed管理域都不能覆盖这份资源快照。
func TestApplyLessonPlanEducationDomainToAssistantActor(
	t *testing.T,
) {
	tests := []struct {
		name          string
		initialDomain string
		lessonDomain  string
		wantDomain    string
	}{
		{
			name:          "mixed管理员进入K12教案后收敛到K12",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  models.EducationDomainK12,
			wantDomain:    models.EducationDomainK12,
		},
		{
			name:          "mixed管理员进入职教教案后收敛到职教",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  models.EducationDomainVocational,
			wantDomain:    models.EducationDomainVocational,
		},
		{
			name:          "mixed管理员进入成教教案后收敛到成教",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  models.EducationDomainAdult,
			wantDomain:    models.EducationDomainAdult,
		},
		{
			name:          "登录K12域不能覆盖职教教案快照",
			initialDomain: models.EducationDomainK12,
			lessonDomain:  models.EducationDomainVocational,
			wantDomain:    models.EducationDomainVocational,
		},
		{
			name:          "登录职教域不能覆盖成教教案快照",
			initialDomain: models.EducationDomainVocational,
			lessonDomain:  models.EducationDomainAdult,
			wantDomain:    models.EducationDomainAdult,
		},
		{
			name:          "教案域大小写和空格会被规范化",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  " Vocational ",
			wantDomain:    models.EducationDomainVocational,
		},
		{
			name:          "空教案域严格清空Actor",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  "",
			wantDomain:    "",
		},
		{
			name:          "非法教案域严格清空Actor",
			initialDomain: models.EducationDomainK12,
			lessonDomain:  "unknown",
			wantDomain:    "",
		},
		{
			name:          "common不能作为具体教案运行域",
			initialDomain: models.EducationDomainMixed,
			lessonDomain:  models.EducationDomainCommon,
			wantDomain:    "",
		},
		{
			name:          "mixed不能作为具体教案资源域",
			initialDomain: models.EducationDomainK12,
			lessonDomain:  models.EducationDomainMixed,
			wantDomain:    "",
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				actor := &AssistantActorContext{
					UserID:          "user-1",
					Role:            models.RoleOperator,
					EducationDomain: testCase.initialDomain,
				}

				lp := &models.LessonPlan{
					ID:              "lesson-plan-1",
					EducationDomain: testCase.lessonDomain,
				}

				applyLessonPlanEducationDomainToAssistantActor(
					actor,
					lp,
				)

				if actor.EducationDomain != testCase.wantDomain {
					t.Fatalf(
						"initial=%q lesson=%q got=%q want=%q",
						testCase.initialDomain,
						testCase.lessonDomain,
						actor.EducationDomain,
						testCase.wantDomain,
					)
				}
			},
		)
	}
}

// TestApplyLessonPlanEducationDomainToAssistantActorNilPlan
// nil教案不能沿用登录Actor原来的mixed或教学域，必须主动清空并fail-closed。
func TestApplyLessonPlanEducationDomainToAssistantActorNilPlan(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	applyLessonPlanEducationDomainToAssistantActor(
		actor,
		nil,
	)

	if actor.EducationDomain != "" {
		t.Fatalf(
			"nil教案后Actor教育域应被清空，实际为%q",
			actor.EducationDomain,
		)
	}
}

// TestApplyLessonPlanEducationDomainToAssistantActorNilActor
// nil Actor应安全返回，不能发生panic。
func TestApplyLessonPlanEducationDomainToAssistantActorNilActor(
	t *testing.T,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf(
				"nil Actor不应触发panic：%v",
				recovered,
			)
		}
	}()

	applyLessonPlanEducationDomainToAssistantActor(
		nil,
		&models.LessonPlan{
			ID:              "lesson-plan-1",
			EducationDomain: models.EducationDomainK12,
		},
	)
}

// TestLessonPlanEducationDomainControlsAssistantResources
// 组合锁定“教案快照覆盖Actor”与“助手资源域校验”两层防线。
func TestLessonPlanEducationDomainControlsAssistantResources(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "admin-1",
		Role:            models.RoleAdmin,
		EducationDomain: models.EducationDomainMixed,
	}

	lp := &models.LessonPlan{
		ID:              "vocational-plan",
		EducationDomain: models.EducationDomainVocational,
	}

	applyLessonPlanEducationDomainToAssistantActor(
		actor,
		lp,
	)

	if actor.EducationDomain != models.EducationDomainVocational {
		t.Fatalf(
			"Actor应收敛到职教域，实际为%q",
			actor.EducationDomain,
		)
	}

	vocationalAssistant := &models.AIAssistant{
		ID:              "vocational-assistant",
		EducationDomain: models.EducationDomainVocational,
	}
	commonAssistant := &models.AIAssistant{
		ID:              "common-assistant",
		EducationDomain: models.EducationDomainCommon,
	}
	k12Assistant := &models.AIAssistant{
		ID:              "k12-assistant",
		EducationDomain: models.EducationDomainK12,
	}
	adultAssistant := &models.AIAssistant{
		ID:              "adult-assistant",
		EducationDomain: models.EducationDomainAdult,
	}

	if !assistantResourceEducationDomainMatches(
		actor,
		vocationalAssistant,
	) {
		t.Fatal("职教教案应允许职教助手")
	}

	if !assistantResourceEducationDomainMatches(
		actor,
		commonAssistant,
	) {
		t.Fatal("职教教案应允许common助手")
	}

	if assistantResourceEducationDomainMatches(
		actor,
		k12Assistant,
	) {
		t.Fatal("职教教案不应允许K12助手")
	}

	if assistantResourceEducationDomainMatches(
		actor,
		adultAssistant,
	) {
		t.Fatal("职教教案不应允许成教助手")
	}
}

// TestInvalidLessonPlanDomainBlocksAllAssistantResources
// 异常教案快照必须清空Actor教育域，并拒绝所有助手资源，不能回退到登录域。
func TestInvalidLessonPlanDomainBlocksAllAssistantResources(
	t *testing.T,
) {
	actor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainK12,
	}

	lp := &models.LessonPlan{
		ID:              "invalid-plan",
		EducationDomain: "unknown",
	}

	applyLessonPlanEducationDomainToAssistantActor(
		actor,
		lp,
	)

	if actor.EducationDomain != "" {
		t.Fatalf(
			"异常教案快照后Actor教育域应为空，实际为%q",
			actor.EducationDomain,
		)
	}

	for _, resourceDomain := range []string{
		models.EducationDomainK12,
		models.EducationDomainVocational,
		models.EducationDomainAdult,
		models.EducationDomainCommon,
	} {
		assistant := &models.AIAssistant{
			ID:              "assistant-" + resourceDomain,
			EducationDomain: resourceDomain,
		}

		if assistantResourceEducationDomainMatches(
			actor,
			assistant,
		) {
			t.Fatalf(
				"异常教案快照不应允许%q助手",
				resourceDomain,
			)
		}
	}
}
