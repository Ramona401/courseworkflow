package services

// courseware_comic_plan_character_subject_type.go
// 漫画AI角色主体类型归一化。
//
// 对外输出协议只允许三种稳定类型：
//   - person：真人、教师、学生和其他人物；
//   - animal：真实或拟人化动物；
//   - object：工具、器材、图形、数字、符号、公式、概念载体以及
//     其他拟人知识对象。
//
// AI模型有时会使用tool、instrument、device、shape、concept、
// anthropomorphic_object等近义值。它们不改变业务语义，但若直接进入
// 严格解析器会导致整份规划失败。因此在JSON解码阶段统一收敛到正式枚举。
//
// 本文件为coursewareComicAICharacter实现自定义JSON解码：
//   - 仍然拒绝未知字段；
//   - 仍然拒绝一个对象后的尾随内容；
//   - 只归一化subject_type，不放宽其他人物字段校验。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tedna/internal/models"
)

// UnmarshalJSON 严格解码AI角色并归一化subject_type。
func (
	target *coursewareComicAICharacter,
) UnmarshalJSON(
	data []byte,
) error {
	if target == nil {
		return fmt.Errorf(
			"漫画AI角色接收对象为空",
		)
	}

	type characterWire struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		Role             string   `json:"role"`
		SubjectType      string   `json:"subject_type"`
		Appearance       string   `json:"appearance"`
		DefaultPosition  string   `json:"default_position"`
		FixedFeatures    []string `json:"fixed_features"`
		ForbiddenChanges []string `json:"forbidden_changes"`
	}

	var decoded characterWire

	decoder :=
		json.NewDecoder(
			bytes.NewReader(
				data,
			),
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			&decoded,
		); err != nil {
		return err
	}

	var trailing interface{}

	if err :=
		decoder.Decode(
			&trailing,
		); err != io.EOF {
		if err == nil {
			return fmt.Errorf(
				"漫画AI角色JSON只能包含一个对象",
			)
		}

		return err
	}

	*target =
		coursewareComicAICharacter{
			ID:
				decoded.ID,
			Name:
				decoded.Name,
			Role:
				decoded.Role,
			SubjectType:
				normalizeCoursewareComicCharacterSubjectType(
					decoded.SubjectType,
				),
			Appearance:
				decoded.Appearance,
			DefaultPosition:
				decoded.DefaultPosition,
			FixedFeatures:
				decoded.FixedFeatures,
			ForbiddenChanges:
				decoded.ForbiddenChanges,
		}

	return nil
}

// normalizeCoursewareComicCharacterSubjectType
// 把模型常见近义值收敛到person、animal或object。
//
// 无法识别的值只执行小写和分隔符规范化，随后仍由原有
// IsValidCWComicCharacterSubjectType严格拒绝，避免静默吞掉真正异常。
func normalizeCoursewareComicCharacterSubjectType(
	value string,
) string {
	normalized :=
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	normalized =
		strings.NewReplacer(
			"-", "_",
			" ", "_",
			"/", "_",
			"\\", "_",
		).Replace(
			normalized,
		)

	for strings.Contains(
		normalized,
		"__",
	) {
		normalized =
			strings.ReplaceAll(
				normalized,
				"__",
				"_",
			)
	}

	normalized =
		strings.Trim(
			normalized,
			"_",
		)

	switch normalized {
	case models.CWComicCharacterSubjectPerson,
		"human",
		"human_being",
		"people",
		"student",
		"teacher",
		"child",
		"kid",
		"boy",
		"girl",
		"adult",
		"人物",
		"人",
		"人类",
		"真人",
		"学生",
		"教师",
		"老师",
		"儿童",
		"男孩",
		"女孩":
		return models.
			CWComicCharacterSubjectPerson

	case models.CWComicCharacterSubjectAnimal,
		"pet",
		"mammal",
		"bird",
		"fish",
		"insect",
		"reptile",
		"creature",
		"动物",
		"宠物",
		"鸟",
		"鱼",
		"昆虫":
		return models.
			CWComicCharacterSubjectAnimal

	case models.CWComicCharacterSubjectObject,
		"tool",
		"instrument",
		"device",
		"equipment",
		"apparatus",
		"utensil",
		"machine",
		"geometry_tool",
		"geometric_object",
		"geometry",
		"shape",
		"symbol",
		"number",
		"formula",
		"concept",
		"knowledge_object",
		"anthropomorphic_object",
		"anthropomorphized_object",
		"mascot_object",
		"物体",
		"物品",
		"工具",
		"器具",
		"器材",
		"仪器",
		"设备",
		"量角器",
		"尺子",
		"圆规",
		"图形",
		"几何图形",
		"数字",
		"符号",
		"公式",
		"概念",
		"拟人物体",
		"拟人知识对象":
		return models.
			CWComicCharacterSubjectObject
	}

	compact :=
		strings.ReplaceAll(
			normalized,
			"_",
			"",
		)

	switch {
	case containsCoursewareComicSubjectTypeHint(
		compact,
		"person",
		"human",
		"student",
		"teacher",
		"child",
		"人物",
		"人类",
		"学生",
		"教师",
		"老师",
	):
		return models.
			CWComicCharacterSubjectPerson

	case containsCoursewareComicSubjectTypeHint(
		compact,
		"animal",
		"mammal",
		"bird",
		"fish",
		"insect",
		"reptile",
		"动物",
		"宠物",
		"昆虫",
	):
		return models.
			CWComicCharacterSubjectAnimal

	case containsCoursewareComicSubjectTypeHint(
		compact,
		"object",
		"tool",
		"instrument",
		"device",
		"equipment",
		"apparatus",
		"geometry",
		"shape",
		"symbol",
		"number",
		"formula",
		"concept",
		"anthropomorphic",
		"物体",
		"工具",
		"器材",
		"仪器",
		"设备",
		"量角器",
		"图形",
		"数字",
		"符号",
		"公式",
		"概念",
		"拟人",
	):
		return models.
			CWComicCharacterSubjectObject
	}

	return normalized
}

func containsCoursewareComicSubjectTypeHint(
	value string,
	hints ...string,
) bool {
	for _, hint :=
		range hints {
		hint =
			strings.ToLower(
				strings.TrimSpace(
					hint,
				),
			)

		if hint != "" &&
			strings.Contains(
				value,
				hint,
			) {
			return true
		}
	}

	return false
}
