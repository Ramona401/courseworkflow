package services

import (
	"testing"

	"tedna/internal/models"
)

// TestAssistantResourceEducationDomainMatches 锁定AI助手按ID加载时的资源域规则。
//
// 规则：
//   - k12、vocational、adult教学上下文只允许同域或common；
//   - mixed管理上下文可以跨域查看全部合法资源域；
//   - common不能作为用户当前教学上下文；
//   - mixed不能作为教学资源所属域；
//   - 空值或非法值一律fail-closed。
func TestAssistantResourceEducationDomainMatches(
	t *testing.T,
) {
	tests := []struct {
		name           string
		actorDomain    string
		resourceDomain string
		want           bool
	}{
		{
			name:           "K12上下文允许K12助手",
			actorDomain:    models.EducationDomainK12,
			resourceDomain: models.EducationDomainK12,
			want:           true,
		},
		{
			name:           "职教上下文允许职教助手",
			actorDomain:    models.EducationDomainVocational,
			resourceDomain: models.EducationDomainVocational,
			want:           true,
		},
		{
			name:           "成教上下文允许成教助手",
			actorDomain:    models.EducationDomainAdult,
			resourceDomain: models.EducationDomainAdult,
			want:           true,
		},
		{
			name:           "K12上下文允许common助手",
			actorDomain:    models.EducationDomainK12,
			resourceDomain: models.EducationDomainCommon,
			want:           true,
		},
		{
			name:           "职教上下文允许common助手",
			actorDomain:    models.EducationDomainVocational,
			resourceDomain: models.EducationDomainCommon,
			want:           true,
		},
		{
			name:           "成教上下文允许common助手",
			actorDomain:    models.EducationDomainAdult,
			resourceDomain: models.EducationDomainCommon,
			want:           true,
		},
		{
			name:           "职教上下文拒绝K12助手",
			actorDomain:    models.EducationDomainVocational,
			resourceDomain: models.EducationDomainK12,
			want:           false,
		},
		{
			name:           "K12上下文拒绝职教助手",
			actorDomain:    models.EducationDomainK12,
			resourceDomain: models.EducationDomainVocational,
			want:           false,
		},
		{
			name:           "职教上下文拒绝成教助手",
			actorDomain:    models.EducationDomainVocational,
			resourceDomain: models.EducationDomainAdult,
			want:           false,
		},
		{
			name:           "mixed管理上下文允许K12助手",
			actorDomain:    models.EducationDomainMixed,
			resourceDomain: models.EducationDomainK12,
			want:           true,
		},
		{
			name:           "mixed管理上下文允许职教助手",
			actorDomain:    models.EducationDomainMixed,
			resourceDomain: models.EducationDomainVocational,
			want:           true,
		},
		{
			name:           "mixed管理上下文允许成教助手",
			actorDomain:    models.EducationDomainMixed,
			resourceDomain: models.EducationDomainAdult,
			want:           true,
		},
		{
			name:           "mixed管理上下文允许common助手",
			actorDomain:    models.EducationDomainMixed,
			resourceDomain: models.EducationDomainCommon,
			want:           true,
		},
		{
			name:           "大小写与空格会被规范化",
			actorDomain:    " Vocational ",
			resourceDomain: " COMMON ",
			want:           true,
		},
		{
			name:           "空Actor教育域严格拒绝",
			actorDomain:    "",
			resourceDomain: models.EducationDomainK12,
			want:           false,
		},
		{
			name:           "非法Actor教育域严格拒绝",
			actorDomain:    "unknown",
			resourceDomain: models.EducationDomainK12,
			want:           false,
		},
		{
			name:           "common不能作为Actor教学上下文",
			actorDomain:    models.EducationDomainCommon,
			resourceDomain: models.EducationDomainCommon,
			want:           false,
		},
		{
			name:           "空资源教育域严格拒绝",
			actorDomain:    models.EducationDomainK12,
			resourceDomain: "",
			want:           false,
		},
		{
			name:           "非法资源教育域严格拒绝",
			actorDomain:    models.EducationDomainK12,
			resourceDomain: "unknown",
			want:           false,
		},
		{
			name:           "mixed不能作为资源所属域",
			actorDomain:    models.EducationDomainMixed,
			resourceDomain: models.EducationDomainMixed,
			want:           false,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				actor := &AssistantActorContext{
					UserID:          "teacher-1",
					Role:            models.RoleOperator,
					EducationDomain: testCase.actorDomain,
				}
				assistant := &models.AIAssistant{
					ID:              "assistant-1",
					EducationDomain: testCase.resourceDomain,
				}

				got := assistantResourceEducationDomainMatches(
					actor,
					assistant,
				)
				if got != testCase.want {
					t.Fatalf(
						"actorDomain=%q resourceDomain=%q got=%v want=%v",
						testCase.actorDomain,
						testCase.resourceDomain,
						got,
						testCase.want,
					)
				}
			},
		)
	}
}

// TestAssistantResourceEducationDomainMatchesRejectsNil
// 锁定空Actor和空助手均严格拒绝，防止调用方异常时放大权限。
func TestAssistantResourceEducationDomainMatchesRejectsNil(
	t *testing.T,
) {
	validActor := &AssistantActorContext{
		UserID:          "teacher-1",
		Role:            models.RoleOperator,
		EducationDomain: models.EducationDomainK12,
	}
	validAssistant := &models.AIAssistant{
		ID:              "assistant-1",
		EducationDomain: models.EducationDomainK12,
	}

	if assistantResourceEducationDomainMatches(
		nil,
		validAssistant,
	) {
		t.Fatal("空Actor不应通过资源教育域校验")
	}

	if assistantResourceEducationDomainMatches(
		validActor,
		nil,
	) {
		t.Fatal("空助手不应通过资源教育域校验")
	}

	if assistantResourceEducationDomainMatches(
		nil,
		nil,
	) {
		t.Fatal("空Actor和空助手不应通过资源教育域校验")
	}
}
