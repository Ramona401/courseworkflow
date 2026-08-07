package services

// courseware_comic_plan_character_subject_type_test.go
// 验证漫画AI角色主体类型兼容层不会降低JSON严格性，
// 并确保工具类主体可以通过完整漫画规划解析链路。

import (
	"encoding/json"
	"testing"

	"tedna/internal/models"
)

func TestNormalizeCoursewareComicCharacterSubjectType(
	t *testing.T,
) {
	t.Parallel()

	tests :=
		[]struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "正式人物枚举",
				input:    "person",
				expected: models.CWComicCharacterSubjectPerson,
			},
			{
				name:     "英文学生归人物",
				input:    "student",
				expected: models.CWComicCharacterSubjectPerson,
			},
			{
				name:     "中文教师归人物",
				input:    "教师",
				expected: models.CWComicCharacterSubjectPerson,
			},
			{
				name:     "正式动物枚举",
				input:    "animal",
				expected: models.CWComicCharacterSubjectAnimal,
			},
			{
				name:     "鸟类归动物",
				input:    "bird",
				expected: models.CWComicCharacterSubjectAnimal,
			},
			{
				name:     "工具归物体",
				input:    "tool",
				expected: models.CWComicCharacterSubjectObject,
			},
			{
				name:     "仪器归物体",
				input:    "instrument",
				expected: models.CWComicCharacterSubjectObject,
			},
			{
				name:     "量角器归物体",
				input:    "量角器",
				expected: models.CWComicCharacterSubjectObject,
			},
			{
				name:     "拟人知识对象归物体",
				input:    "anthropomorphic_object",
				expected: models.CWComicCharacterSubjectObject,
			},
			{
				name:     "几何工具复合值归物体",
				input:    "geometric-tool-character",
				expected: models.CWComicCharacterSubjectObject,
			},
			{
				name:     "未知值仍保留给严格校验拒绝",
				input:    "unknown_entity",
				expected: "unknown_entity",
			},
		}

	for _, test :=
		range tests {
		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				actual :=
					normalizeCoursewareComicCharacterSubjectType(
						test.input,
					)

				if actual != test.expected {
					t.Fatalf(
						"归一化结果不符合预期: input=%q actual=%q expected=%q",
						test.input,
						actual,
						test.expected,
					)
				}
			},
		)
	}
}

func TestCoursewareComicAICharacterUnmarshalNormalizesSubjectType(
	t *testing.T,
) {
	t.Parallel()

	raw :=
		[]byte(
			`{
				"id":"CHAR-01",
				"name":"量角器小助手",
				"role":"演示角度测量",
				"subject_type":"instrument",
				"appearance":"半圆形透明量角器，带有拟人眼睛和手臂",
				"default_position":"left_bottom",
				"fixed_features":["透明半圆外形","紫色刻度装饰"],
				"forbidden_changes":["不得改变为直尺","不得改变主体颜色"]
			}`,
		)

	var character coursewareComicAICharacter

	if err :=
		json.Unmarshal(
			raw,
			&character,
		); err != nil {
		t.Fatalf(
			"角色JSON解码失败: %v",
			err,
		)
	}

	if character.SubjectType !=
		models.CWComicCharacterSubjectObject {
		t.Fatalf(
			"instrument应归一化为object，实际为%q",
			character.SubjectType,
		)
	}

	if character.ID != "CHAR-01" ||
		character.Name != "量角器小助手" {
		t.Fatalf(
			"归一化过程中不应改变其他角色字段: %+v",
			character,
		)
	}
}

func TestCoursewareComicAICharacterUnmarshalRejectsUnknownField(
	t *testing.T,
) {
	t.Parallel()

	raw :=
		[]byte(
			`{
				"id":"CHAR-01",
				"name":"量角器小助手",
				"role":"演示角度测量",
				"subject_type":"object",
				"appearance":"透明半圆量角器",
				"default_position":"left_bottom",
				"fixed_features":["半圆外形"],
				"forbidden_changes":["不得改变外形"],
				"hidden_model_field":"不允许"
			}`,
		)

	var character coursewareComicAICharacter

	if err :=
		json.Unmarshal(
			raw,
			&character,
		); err == nil {
		t.Fatal(
			"包含未知字段的角色JSON必须被拒绝",
		)
	}
}

func TestParseCoursewareComicPlanAIResultAcceptsInstrumentSubjectType(
	t *testing.T,
) {
	t.Parallel()

	project :=
		&models.CoursewareComicProject{
			ID:
				"11111111-1111-4111-8111-111111111111",
			PanelCount:
				4,
		}

	raw :=
		`{
			"art_style_text":"明亮、清晰的教学漫画插画，紫蓝色主色，柔和光影",
			"continuity_notes":[
				"量角器小助手始终保持透明半圆外形和紫色刻度装饰"
			],
			"characters":[
				{
					"id":"CHAR-01",
					"name":"量角器小助手",
					"role":"演示角度测量方法",
					"subject_type":"instrument",
					"appearance":"透明半圆量角器，具有拟人眼睛、手臂和紫色刻度装饰",
					"default_position":"left_bottom",
					"fixed_features":[
						"透明半圆外形",
						"紫色刻度装饰"
					],
					"forbidden_changes":[
						"不得变成直尺",
						"不得改变紫色主色"
					]
				}
			],
			"panels":[
				{
					"panel_no":1,
					"story_purpose":"认识量角器的中心和零刻度线",
					"knowledge_claim":"测量角时，量角器中心必须与角的顶点重合",
					"scene_text":"明亮的数学教室桌面",
					"characters":["CHAR-01"],
					"action_text":"量角器小助手指向自己的中心点",
					"camera_text":"横向中景，主体位于左下方",
					"narration_text":"先找准角的顶点。",
					"knowledge_presentation":"拟人演示",
					"focus_text":"量角器中心与角顶点重合",
					"layout_text":"右上方预留旁白区域",
					"visual_prompt":"透明半圆量角器拟人角色在数学教室中演示中心点，无文字画面",
					"negative_prompt":"禁止文字、错误刻度和主体变形",
					"relations":[],
					"overlay_elements":[
						{
							"id":"EL-001",
							"type":"narration",
							"content":"量角器的中心要和角的顶点重合。",
							"speaker_id":"",
							"target_character_id":"",
							"target_anchor":"center",
							"style_id":"narration_clean",
							"preferred_region":"top_right",
							"priority":1,
							"question":null
						}
					]
				},
				{
					"panel_no":2,
					"story_purpose":"对齐角的一条边和零刻度线",
					"knowledge_claim":"角的一条边要与量角器的零刻度线重合",
					"scene_text":"同一张数学教室桌面",
					"characters":["CHAR-01"],
					"action_text":"量角器小助手将零刻度线贴合角的一条边",
					"camera_text":"横向近景，突出零刻度线",
					"narration_text":"再把一条边对准零刻度线。",
					"knowledge_presentation":"步骤演示",
					"focus_text":"角边与零刻度线重合",
					"layout_text":"左上方预留旁白区域",
					"visual_prompt":"同一透明半圆量角器拟人角色演示零刻度线对齐，无文字画面",
					"negative_prompt":"禁止文字、错误刻度和角色漂移",
					"relations":[
						{
							"target_panel_no":1,
							"relation_code":">",
							"inherit_mask":"AC",
							"semantic_note":"延续量角器角色外观"
						}
					],
					"overlay_elements":[
						{
							"id":"EL-001",
							"type":"narration",
							"content":"角的一条边要与零刻度线重合。",
							"speaker_id":"",
							"target_character_id":"",
							"target_anchor":"center",
							"style_id":"narration_clean",
							"preferred_region":"top_left",
							"priority":1,
							"question":null
						}
					]
				},
				{
					"panel_no":3,
					"story_purpose":"判断读取内圈还是外圈刻度",
					"knowledge_claim":"从与角边重合的零刻度一侧开始读数",
					"scene_text":"同一张数学教室桌面",
					"characters":["CHAR-01"],
					"action_text":"量角器小助手沿正确刻度方向指向另一条角边",
					"camera_text":"横向特写，突出读数方向",
					"narration_text":"从零刻度所在的一侧开始读。",
					"knowledge_presentation":"错误对比",
					"focus_text":"正确选择内外圈刻度",
					"layout_text":"右上方预留知识卡区域",
					"visual_prompt":"同一透明半圆量角器拟人角色演示正确读数方向，无文字画面",
					"negative_prompt":"禁止文字、刻度方向错误和角色漂移",
					"relations":[
						{
							"target_panel_no":2,
							"relation_code":">",
							"inherit_mask":"ACS",
							"semantic_note":"延续角色与桌面场景"
						}
					],
					"overlay_elements":[
						{
							"id":"EL-001",
							"type":"knowledge_card",
							"content":"从零刻度一侧开始读数。",
							"speaker_id":"",
							"target_character_id":"",
							"target_anchor":"center",
							"style_id":"knowledge_blue",
							"preferred_region":"top_right",
							"priority":1,
							"question":null
						}
					]
				},
				{
					"panel_no":4,
					"story_purpose":"完成测量并检查结果",
					"knowledge_claim":"另一条角边对应的刻度就是这个角的度数",
					"scene_text":"同一张数学教室桌面",
					"characters":["CHAR-01"],
					"action_text":"量角器小助手展示完成测量后的角",
					"camera_text":"横向全景，展示完整测量步骤",
					"narration_text":"最后读取另一条边对应的刻度。",
					"knowledge_presentation":"总结与练习",
					"focus_text":"读取角的最终度数",
					"layout_text":"底部预留问题卡区域",
					"visual_prompt":"同一透明半圆量角器拟人角色完成角度测量，无文字画面",
					"negative_prompt":"禁止文字、错误角度和角色漂移",
					"relations":[
						{
							"target_panel_no":3,
							"relation_code":">",
							"inherit_mask":"ACS",
							"semantic_note":"延续角色与教学场景"
						}
					],
					"overlay_elements":[
						{
							"id":"EL-001",
							"type":"question_card",
							"content":"想一想",
							"speaker_id":"",
							"target_character_id":"",
							"target_anchor":"center",
							"style_id":"question_purple",
							"preferred_region":"bottom_center",
							"priority":1,
							"question":{
								"question":"量角器中心应该和角的什么位置重合？",
								"options":[
									"角的顶点",
									"角的一条边"
								],
								"answer_index":0,
								"explanation":"测量角时，量角器中心必须与角的顶点重合。",
								"answer_mode":"click_reveal"
							}
						}
					]
				}
			]
		}`

	result, err :=
		parseCoursewareComicPlanAIResult(
			raw,
			project,
		)
	if err != nil {
		t.Fatalf(
			"完整规划不应因instrument类型失败: %v",
			err,
		)
	}

	if result == nil ||
		len(result.Panels) != 4 {
		t.Fatalf(
			"完整规划解析结果无效: %+v",
			result,
		)
	}

	var bible models.CoursewareComicCharacterBible

	if err :=
		json.Unmarshal(
			[]byte(
				result.CharacterBibleJSON,
			),
			&bible,
		); err != nil {
		t.Fatalf(
			"人物设定JSON解析失败: %v",
			err,
		)
	}

	if len(bible.Characters) != 1 ||
		bible.Characters[0].SubjectType !=
			models.CWComicCharacterSubjectObject {
		t.Fatalf(
			"工具角色应在完整解析后保存为object: %+v",
			bible.Characters,
		)
	}
}
