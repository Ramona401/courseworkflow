package services

// lesson_plan_context_capsule_choice_test.go — 教师方案选择确定性收拢测试
//
// 不连接数据库、不调用模型，只验证方案选择、替代、恢复和防误判规则。

import (
	"encoding/json"
	"testing"

	"tedna/internal/models"
)

func TestApplyLessonPlanCapsuleTeacherChoice(
	t *testing.T,
) {
	optionA := lessonPlanCapsuleChoiceTestItem(
		"teaching.activity.option_a",
		"方案A：手账拼图",
		"教师此前选择手账拼图作为核心活动",
		models.LessonPlanContextCapsuleItemStateActive,
		models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
	)

	optionOne := lessonPlanCapsuleChoiceTestItem(
		"teaching.activity.option_1",
		"方案一：探究型贴纸归位",
		"通过贴纸归位理清课文结构",
		models.LessonPlanContextCapsuleItemStateCandidate,
		models.LessonPlanContextCapsuleAuthorityAIInferred,
	)

	current :=
		models.LessonPlanContextCapsuleDocument{
			TeachingConsensus: []models.LessonPlanContextCapsuleItem{
				optionA,
			},
			OpenQuestions: []models.LessonPlanContextCapsuleItem{
				optionOne,
			},
		}

	currentJSON :=
		mustMarshalLessonPlanCapsuleChoiceTest(
			t,
			current,
		)

	next := current

	next.TeachingConsensus = append(
		[]models.LessonPlanContextCapsuleItem(nil),
		current.TeachingConsensus...,
	)

	next.OpenQuestions = append(
		[]models.LessonPlanContextCapsuleItem(nil),
		current.OpenQuestions...,
	)

	changed :=
		applyLessonPlanCapsuleTeacherChoice(
			&next,
			currentJSON,
			"方案一：探究型贴纸归位，作为本课核心活动",
			"turn_12",
		)

	if !changed {
		t.Fatal(
			"教师明确选择方案一时应产生确定性变化",
		)
	}

	selected, ok :=
		lessonPlanCapsuleChoiceTestFind(
			next.TeachingConsensus,
			optionOne.Key,
		)

	if !ok {
		t.Fatal(
			"被选方案应进入教学共识",
		)
	}

	if selected.Authority !=
		models.LessonPlanContextCapsuleAuthorityTeacherExplicit {
		t.Fatalf(
			"被选方案权威等级异常: %q",
			selected.Authority,
		)
	}

	if selected.State !=
		models.LessonPlanContextCapsuleItemStateActive {
		t.Fatalf(
			"被选方案状态异常: %q",
			selected.State,
		)
	}

	if selected.DoNotReconfirm {
		t.Fatal(
			"当前有效方案不应带do_not_reconfirm",
		)
	}

	if lessonPlanCapsuleContainsItem(
		next.TeachingConsensus,
		optionA.Key,
	) ||
		lessonPlanCapsuleContainsItem(
			next.OpenQuestions,
			optionA.Key,
		) ||
		lessonPlanCapsuleContainsItem(
			next.DeferredItems,
			optionA.Key,
		) {
		t.Fatal(
			"未选方案不应继续留在正向区域",
		)
	}

	superseded, ok :=
		lessonPlanCapsuleChoiceTestFind(
			next.SupersededItems,
			optionA.Key,
		)

	if !ok {
		t.Fatal(
			"未选方案应进入superseded_items",
		)
	}

	if !superseded.DoNotReconfirm ||
		superseded.ReplacedBy != optionOne.Key {
		t.Fatalf(
			"替代关系异常: do_not=%v replaced_by=%q",
			superseded.DoNotReconfirm,
			superseded.ReplacedBy,
		)
	}

	if superseded.UpdatedByTurnID !=
		"turn_12" {
		t.Fatalf(
			"替代轮次异常: %q",
			superseded.UpdatedByTurnID,
		)
	}
}

func TestApplyLessonPlanCapsuleTeacherChoiceRestoresOption(
	t *testing.T,
) {
	optionA := lessonPlanCapsuleChoiceTestItem(
		"teaching.activity.option_a",
		"方案A：手账拼图",
		"使用手账拼图组织活动",
		models.LessonPlanContextCapsuleItemStateActive,
		models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
	)

	optionOne := lessonPlanCapsuleChoiceTestItem(
		"teaching.activity.option_1",
		"方案一：贴纸归位",
		"使用贴纸归位理清结构",
		models.LessonPlanContextCapsuleItemStateSuperseded,
		models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
	)

	optionOne.DoNotReconfirm = true
	optionOne.ReplacedBy = optionA.Key

	current :=
		models.LessonPlanContextCapsuleDocument{
			TeachingConsensus: []models.LessonPlanContextCapsuleItem{
				optionA,
			},
			SupersededItems: []models.LessonPlanContextCapsuleItem{
				optionOne,
			},
			StageFocus: models.LessonPlanContextCapsuleStageFocus{
				AvoidRepeatingKeys: []string{
					optionOne.Key,
				},
			},
		}

	next := current

	changed :=
		applyLessonPlanCapsuleTeacherChoice(
			&next,
			mustMarshalLessonPlanCapsuleChoiceTest(
				t,
				current,
			),
			"改用方案一",
			"turn_20",
		)

	if !changed {
		t.Fatal(
			"教师改用旧方案时应恢复该方案",
		)
	}

	if lessonPlanCapsuleContainsItem(
		next.SupersededItems,
		optionOne.Key,
	) {
		t.Fatal(
			"恢复后的方案不应留在superseded_items",
		)
	}

	if !lessonPlanCapsuleContainsItem(
		next.TeachingConsensus,
		optionOne.Key,
	) {
		t.Fatal(
			"恢复后的方案应进入教学共识",
		)
	}
}

func TestLessonPlanCapsuleChoiceQuestionsDoNotSelect(
	t *testing.T,
) {
	for _, message := range []string{
		"请比较方案一和方案A的区别",
		"方案一有哪些优点？",
		"方案一还是方案A更合适？",
		"两个方案都保留可以吗？",
	} {
		message := message

		t.Run(
			message,
			func(t *testing.T) {
				if label, selected :=
					detectLessonPlanCapsuleTeacherChoice(
						message,
					); selected {
					t.Fatalf(
						"比较或询问不应被识别为选择: %q",
						label,
					)
				}
			},
		)
	}
}

func TestDetectLessonPlanCapsuleTeacherChoice(
	t *testing.T,
) {
	cases := map[string]string{
		"我选择方案A": "A",
		"按方案二继续": "2",
		"方案一：探究型贴纸归位，作为核心活动": "1",
		"方案3就好": "3",
	}

	for message, expected := range cases {
		message := message
		expected := expected

		t.Run(
			message,
			func(t *testing.T) {
				label, selected :=
					detectLessonPlanCapsuleTeacherChoice(
						message,
					)

				if !selected ||
					label != expected {
					t.Fatalf(
						"选择识别异常: selected=%v got=%q want=%q",
						selected,
						label,
						expected,
					)
				}
			},
		)
	}
}

func lessonPlanCapsuleChoiceTestItem(
	key string,
	title string,
	content string,
	state string,
	authority string,
) models.LessonPlanContextCapsuleItem {
	return models.LessonPlanContextCapsuleItem{
		Key:        key,
		Title:      title,
		Content:    content,
		State:      state,
		Authority:  authority,
		Importance: 5,
		ApplicableStages: []string{
			"design",
			"write",
		},
	}
}

func mustMarshalLessonPlanCapsuleChoiceTest(
	t *testing.T,
	document models.LessonPlanContextCapsuleDocument,
) string {
	t.Helper()

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf(
			"构造测试胶囊失败: %v",
			err,
		)
	}

	return string(encoded)
}

func lessonPlanCapsuleChoiceTestFind(
	items []models.LessonPlanContextCapsuleItem,
	key string,
) (
	models.LessonPlanContextCapsuleItem,
	bool,
) {
	for _, item := range items {
		if item.Key == key {
			return item, true
		}
	}

	return models.LessonPlanContextCapsuleItem{},
		false
}
