package services

// lesson_plan_context_capsule_document.go — 胶囊文档规则与负向记忆保护
//
// 本文件集中负责：
//   - 判断哪些消息不应触发胶囊AI更新；
//   - 归一化AI返回的完整胶囊；
//   - 规范原子条目、稳定键、可信等级和适用阶段；
//   - 保护已经纠正、否定或替代的负向记忆；
//   - 判断胶囊是否具备可进入正式上下文的可靠核心。
//
// 已拆分职责：
//   - 运行时短版和教师端安全展示：lesson_plan_context_capsule_presentation.go；
//   - 稳定语义哈希和原文证据路由：lesson_plan_context_capsule_evidence.go。
//
// 关键产品原则：
//   - “好、可以、继续”等短表达可能是自然确认，不能被当作无意义消息跳过；
//   - 教师已经订正的旧错误不得因模型重新总结而复活；
//   - AI推断不能进入正式课程核心或强约束；
//   - 阶段变化只调整注意力，不清空跨阶段共识。

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"tedna/internal/models"
)

// shouldSkipLessonPlanContextCapsuleUpdate 判断本轮是否完全不需要AI更新。
//
// “好、好的、可以、继续”可能是对上一轮方案的自然确认，不能跳过。
// 这里只跳过系统自动触发语、空文本和没有教学语义的纯礼貌表达。
func shouldSkipLessonPlanContextCapsuleUpdate(
	message string,
) bool {
	if isStageAutoTriggerContent(message) {
		return true
	}

	normalized := normalizeLessonPlanTurnText(message)

	switch normalized {
	case "",
		"谢谢",
		"谢谢你",
		"多谢",
		"辛苦了",
		"再见",
		"拜拜":
		return true
	default:
		return false
	}
}

// normalizeLessonPlanContextCapsuleDocument 归一化完整胶囊。
func normalizeLessonPlanContextCapsuleDocument(
	document *models.LessonPlanContextCapsuleDocument,
	stageCode string,
	turnID string,
) {
	if document == nil {
		return
	}

	document.SchemaVersion =
		models.LessonPlanContextCapsuleSchemaVersion

	document.Summary = normalizeLessonPlanCapsuleText(
		document.Summary,
		600,
	)

	document.CourseCore = normalizeLessonPlanCapsuleItems(
		document.CourseCore,
		models.LessonPlanContextCapsuleItemStateActive,
		16,
		turnID,
		false,
	)

	document.TeachingConsensus = normalizeLessonPlanCapsuleItems(
		document.TeachingConsensus,
		models.LessonPlanContextCapsuleItemStateActive,
		20,
		turnID,
		false,
	)

	document.Constraints = normalizeLessonPlanCapsuleItems(
		document.Constraints,
		models.LessonPlanContextCapsuleItemStateActive,
		20,
		turnID,
		false,
	)

	document.OpenQuestions = normalizeLessonPlanCapsuleItems(
		document.OpenQuestions,
		models.LessonPlanContextCapsuleItemStateCandidate,
		12,
		turnID,
		false,
	)

	document.DeferredItems = normalizeLessonPlanCapsuleItems(
		document.DeferredItems,
		models.LessonPlanContextCapsuleItemStateDeferred,
		12,
		turnID,
		false,
	)

	document.SupersededItems = normalizeLessonPlanCapsuleItems(
		document.SupersededItems,
		models.LessonPlanContextCapsuleItemStateSuperseded,
		30,
		turnID,
		true,
	)

	document.StageFocus.StageCode =
		strings.TrimSpace(stageCode)

	document.StageFocus.CurrentTask =
		normalizeLessonPlanCapsuleText(
			document.StageFocus.CurrentTask,
			500,
		)

	document.StageFocus.CarryForwardKeys =
		normalizeLessonPlanCapsuleStringList(
			document.StageFocus.CarryForwardKeys,
			30,
			160,
		)

	document.StageFocus.AvoidRepeatingKeys =
		normalizeLessonPlanCapsuleStringList(
			document.StageFocus.AvoidRepeatingKeys,
			40,
			160,
		)

	for _, item := range document.SupersededItems {
		document.StageFocus.AvoidRepeatingKeys =
			appendUniqueLessonPlanCapsuleString(
				document.StageFocus.AvoidRepeatingKeys,
				item.Key,
				40,
			)
	}
}

// normalizeLessonPlanCapsuleItems 清理并限制同类原子条目。
func normalizeLessonPlanCapsuleItems(
	items []models.LessonPlanContextCapsuleItem,
	defaultState string,
	limit int,
	turnID string,
	forceDoNotReconfirm bool,
) []models.LessonPlanContextCapsuleItem {
	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(items),
	)

	seen := make(map[string]struct{})

	for _, item := range items {
		item.Key = normalizeLessonPlanCapsuleKey(
			item.Key,
		)

		item.Title = normalizeLessonPlanCapsuleText(
			item.Title,
			180,
		)

		item.Content = normalizeLessonPlanCapsuleText(
			item.Content,
			1000,
		)

		if item.Key == "" ||
			item.Title == "" ||
			item.Content == "" {
			continue
		}

		if _, exists := seen[item.Key]; exists {
			continue
		}

		if !models.IsValidLessonPlanContextCapsuleAuthority(
			item.Authority,
		) {
			item.Authority =
				models.LessonPlanContextCapsuleAuthorityAIInferred
		}

		if item.Importance < 1 {
			item.Importance = 1
		}
		if item.Importance > 5 {
			item.Importance = 5
		}

		item.State = defaultState

		item.ApplicableStages =
			normalizeLessonPlanCapsuleStringList(
				item.ApplicableStages,
				10,
				100,
			)

		item.SourceKeys =
			normalizeLessonPlanCapsuleStringList(
				item.SourceKeys,
				12,
				200,
			)

		item.ReplacedBy = normalizeLessonPlanCapsuleKey(
			item.ReplacedBy,
		)

		if strings.TrimSpace(item.UpdatedByTurnID) == "" {
			item.UpdatedByTurnID = strings.TrimSpace(
				turnID,
			)
		} else {
			item.UpdatedByTurnID =
				normalizeLessonPlanCapsuleText(
					item.UpdatedByTurnID,
					255,
				)
		}

		if forceDoNotReconfirm {
			item.DoNotReconfirm = true
		}

		seen[item.Key] = struct{}{}
		output = append(output, item)

		if len(output) >= limit {
			break
		}
	}

	sort.SliceStable(
		output,
		func(left int, right int) bool {
			return output[left].Importance >
				output[right].Importance
		},
	)

	return output
}

// normalizeLessonPlanCapsuleKey 生成数据库约束允许的稳定条目键。
func normalizeLessonPlanCapsuleKey(
	value string,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastSeparator := false

	for _, character := range value {
		allowed := unicode.IsDigit(character) ||
			(character >= 'a' && character <= 'z') ||
			character == '.' ||
			character == '_' ||
			character == ':' ||
			character == '-'

		if allowed {
			builder.WriteRune(character)

			lastSeparator = character == '.' ||
				character == '_' ||
				character == ':' ||
				character == '-'

			continue
		}

		if !lastSeparator {
			builder.WriteRune('_')
			lastSeparator = true
		}
	}

	result := strings.Trim(
		builder.String(),
		"._:-",
	)

	if len(result) > 160 {
		result = result[:160]
	}

	return result
}

// normalizeLessonPlanCapsuleText 按Unicode字符限制文本长度。
func normalizeLessonPlanCapsuleText(
	value string,
	limit int,
) string {
	value = strings.TrimSpace(value)

	if value == "" || limit <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return string(runes[:limit])
}

// normalizeLessonPlanCapsuleStringList 清理、去重并限制字符串数组。
func normalizeLessonPlanCapsuleStringList(
	values []string,
	limit int,
	itemLimit int,
) []string {
	output := make([]string, 0, len(values))
	seen := make(map[string]struct{})

	for _, value := range values {
		value = normalizeLessonPlanCapsuleText(
			value,
			itemLimit,
		)

		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		output = append(output, value)

		if len(output) >= limit {
			break
		}
	}

	return output
}

// appendUniqueLessonPlanCapsuleString 安全追加一个唯一字符串。
func appendUniqueLessonPlanCapsuleString(
	values []string,
	value string,
	limit int,
) []string {
	value = strings.TrimSpace(value)

	if value == "" {
		return values
	}

	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	if len(values) >= limit {
		return values
	}

	return append(values, value)
}

// teacherExplicitlyRestoresCapsuleMemory 判断教师是否明确恢复旧方向。
func teacherExplicitlyRestoresCapsuleMemory(
	message string,
) bool {
	normalized := normalizeLessonPlanTurnText(message)

	return containsAnyLessonPlanTurnText(
		normalized,
		[]string{
			"恢复之前",
			"改回原来",
			"重新采用",
			"重新启用",
			"把之前的加回来",
			"撤销刚才的否定",
			"之前那条可以继续",
		},
	)
}

// preserveLessonPlanCapsuleNegativeMemory 防止历史错误或旧方案复活。
func preserveLessonPlanCapsuleNegativeMemory(
	document *models.LessonPlanContextCapsuleDocument,
	currentJSON string,
	teacherMessage string,
) {
	if document == nil ||
		strings.TrimSpace(currentJSON) == "" ||
		strings.TrimSpace(currentJSON) == "{}" {
		return
	}

	current := &models.LessonPlanContextCapsuleDocument{}
	if err := json.Unmarshal(
		[]byte(currentJSON),
		current,
	); err != nil {
		return
	}

	explicitRestore :=
		teacherExplicitlyRestoresCapsuleMemory(
			teacherMessage,
		)

	for _, oldItem := range current.SupersededItems {
		if explicitRestore {
			continue
		}

		removeLessonPlanCapsulePositiveItemByKey(
			document,
			oldItem.Key,
		)

		if !lessonPlanCapsuleContainsItem(
			document.SupersededItems,
			oldItem.Key,
		) {
			oldItem.State =
				models.LessonPlanContextCapsuleItemStateSuperseded

			oldItem.DoNotReconfirm = true

			document.SupersededItems = append(
				document.SupersededItems,
				oldItem,
			)
		}

		document.StageFocus.AvoidRepeatingKeys =
			appendUniqueLessonPlanCapsuleString(
				document.StageFocus.AvoidRepeatingKeys,
				oldItem.Key,
				40,
			)
	}

	for _, oldConstraint := range current.Constraints {
		if !oldConstraint.DoNotReconfirm ||
			explicitRestore {
			continue
		}

		if !lessonPlanCapsuleContainsItem(
			document.Constraints,
			oldConstraint.Key,
		) {
			document.Constraints = append(
				document.Constraints,
				oldConstraint,
			)
		}
	}
}

// removeLessonPlanCapsulePositiveItemByKey 从所有正向区域删除旧条目。
func removeLessonPlanCapsulePositiveItemByKey(
	document *models.LessonPlanContextCapsuleDocument,
	key string,
) {
	document.CourseCore =
		removeLessonPlanCapsuleItem(
			document.CourseCore,
			key,
		)

	document.TeachingConsensus =
		removeLessonPlanCapsuleItem(
			document.TeachingConsensus,
			key,
		)

	document.Constraints =
		removeLessonPlanCapsuleItem(
			document.Constraints,
			key,
		)

	document.OpenQuestions =
		removeLessonPlanCapsuleItem(
			document.OpenQuestions,
			key,
		)

	document.DeferredItems =
		removeLessonPlanCapsuleItem(
			document.DeferredItems,
			key,
		)
}

func removeLessonPlanCapsuleItem(
	items []models.LessonPlanContextCapsuleItem,
	key string,
) []models.LessonPlanContextCapsuleItem {
	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(items),
	)

	for _, item := range items {
		if item.Key != key {
			output = append(output, item)
		}
	}

	return output
}

func lessonPlanCapsuleContainsItem(
	items []models.LessonPlanContextCapsuleItem,
	key string,
) bool {
	for _, item := range items {
		if item.Key == key {
			return true
		}
	}

	return false
}

// lessonPlanContextCapsuleHasUsableCore 要求至少存在一条非AI推断的有效核心。
func lessonPlanContextCapsuleHasUsableCore(
	document *models.LessonPlanContextCapsuleDocument,
) bool {
	if document == nil {
		return false
	}

	groups := [][]models.LessonPlanContextCapsuleItem{
		document.CourseCore,
		document.TeachingConsensus,
		document.Constraints,
	}

	for _, group := range groups {
		for _, item := range group {
			if item.Authority !=
				models.LessonPlanContextCapsuleAuthorityAIInferred {
				return true
			}
		}
	}

	return false
}
