package services

// lesson_plan_context_capsule_resilience_test.go — 胶囊JSON容错专项回归测试
//
// 本测试不连接数据库、不调用模型、不启动HTTP服务。
// 只验证本地确定性解析和模型输入压缩规则。

import (
	"encoding/json"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestParseLessonPlanContextCapsuleAIResult(
	t *testing.T,
) {
	t.Run(
		"接受Markdown围栏中的合法JSON",
		func(t *testing.T) {
			raw := "```json\n" +
				validLessonPlanContextCapsuleAIResultJSON() +
				"\n```"

			result, err :=
				parseLessonPlanContextCapsuleAIResult(
					raw,
				)
			if err != nil {
				t.Fatalf(
					"解析合法围栏JSON失败: %v",
					err,
				)
			}

			if result.UpdateReason !=
				"已同步教师本轮决定" {
				t.Fatalf(
					"update_reason异常: %q",
					result.UpdateReason,
				)
			}
		},
	)

	t.Run(
		"移除对象和数组结束前的尾逗号",
		func(t *testing.T) {
			raw := `{
				"update_reason":"已同步教师本轮决定",
				"capsule":{
					"schema_version":1,
					"summary":"确定课堂主线",
					"course_core":[],
					"teaching_consensus":[],
					"constraints":[],
					"open_questions":[],
					"deferred_items":[],
					"superseded_items":[],
					"stage_focus":{
						"stage_code":"design",
						"current_task":"完善教学设计",
						"carry_forward_keys":[],
						"avoid_repeating_keys":[],
					},
				},
				"changes":[],
				"evidence_bindings":[],
			}`

			result, err :=
				parseLessonPlanContextCapsuleAIResult(
					raw,
				)
			if err != nil {
				t.Fatalf(
					"尾逗号修复失败: %v",
					err,
				)
			}

			if result.Capsule.Summary !=
				"确定课堂主线" {
				t.Fatalf(
					"summary异常: %q",
					result.Capsule.Summary,
				)
			}
		},
	)

	t.Run(
		"拒绝截断JSON而不是猜测补齐",
		func(t *testing.T) {
			raw := `{
				"update_reason":"未完成",
				"capsule":{
					"summary":"字符串没有结束`

			_, err :=
				parseLessonPlanContextCapsuleAIResult(
					raw,
				)
			if err == nil {
				t.Fatal(
					"截断JSON不应被本地猜测性修复",
				)
			}
		},
	)
}

func TestCompactLessonPlanContextCapsuleForModel(
	t *testing.T,
) {
	items := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		20,
	)

	for index := 0; index < 20; index++ {
		items = append(
			items,
			models.LessonPlanContextCapsuleItem{
				Key:        "strategy.option",
				Title:      strings.Repeat("题", 200),
				Content:    strings.Repeat("内容", 300),
				State:      "active",
				Authority:  "teacher_explicit",
				Importance: 5,
				SourceKeys: []string{
					"teacher_turn:1",
					"teacher_turn:2",
				},
				ApplicableStages: []string{
					"analyze",
					"design",
					"write",
				},
			},
		)
	}

	document :=
		models.LessonPlanContextCapsuleDocument{
			SchemaVersion: 1,
			Summary: strings.Repeat(
				"总结",
				300,
			),
			TeachingConsensus: items,
		}

	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf(
			"构造测试胶囊失败: %v",
			err,
		)
	}

	compacted :=
		compactLessonPlanContextCapsuleForModel(
			string(raw),
		)

	encoded, err := json.Marshal(compacted)
	if err != nil {
		t.Fatalf(
			"序列化压缩结果失败: %v",
			err,
		)
	}

	decoded :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		encoded,
		decoded,
	); err != nil {
		t.Fatalf(
			"解析压缩结果失败: %v",
			err,
		)
	}

	if len([]rune(decoded.Summary)) >
		lessonPlanContextCapsuleModelSummaryRunes {
		t.Fatalf(
			"summary未按上限压缩: %d",
			len([]rune(decoded.Summary)),
		)
	}

	if len(decoded.TeachingConsensus) !=
		lessonPlanContextCapsuleModelConsensusLimit {
		t.Fatalf(
			"教学共识数量异常: got=%d want=%d",
			len(decoded.TeachingConsensus),
			lessonPlanContextCapsuleModelConsensusLimit,
		)
	}

	for _, item := range decoded.TeachingConsensus {
		if len([]rune(item.Title)) >
			lessonPlanContextCapsuleModelItemTitleRunes {
			t.Fatalf(
				"条目标题未压缩: %d",
				len([]rune(item.Title)),
			)
		}

		if len([]rune(item.Content)) >
			lessonPlanContextCapsuleModelItemContentRunes {
			t.Fatalf(
				"条目正文未压缩: %d",
				len([]rune(item.Content)),
			)
		}

		if item.Authority !=
			"teacher_explicit" {
			t.Fatalf(
				"压缩不应改变权威等级: %q",
				item.Authority,
			)
		}
	}
}

func validLessonPlanContextCapsuleAIResultJSON() string {
	return `{
		"update_reason":"已同步教师本轮决定",
		"capsule":{
			"schema_version":1,
			"summary":"确定课堂主线",
			"course_core":[],
			"teaching_consensus":[],
			"constraints":[],
			"open_questions":[],
			"deferred_items":[],
			"superseded_items":[],
			"stage_focus":{
				"stage_code":"design",
				"current_task":"完善教学设计",
				"carry_forward_keys":[],
				"avoid_repeating_keys":[]
			}
		},
		"changes":[],
		"evidence_bindings":[]
	}`
}
