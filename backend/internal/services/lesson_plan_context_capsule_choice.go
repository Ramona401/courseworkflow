package services

// lesson_plan_context_capsule_choice.go — 教师自然语言选定方案的确定性收拢
//
// 只处理一个窄语义：当前胶囊中存在多个“方案/选项”备选，教师本轮明确
// 选择其中一个。后端把同组其他备选移入superseded_items，并把被选方案
// 升级为teacher_explicit。比较、询问、犹豫或要求同时保留不会触发替代。

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"tedna/internal/models"
)

var lessonPlanCapsuleOptionLabelPattern = regexp.MustCompile(
	`(?i)(方案|选项)[[:space:]]*([一二三四五六七八九十0-9a-z])`,
)

type lessonPlanCapsuleOptionDescriptor struct {
	Label    string
	Category string
	Item     models.LessonPlanContextCapsuleItem
}

// applyLessonPlanCapsuleTeacherChoice 按教师明确选择收拢同组方案。
// 返回true表示正式胶囊语义发生变化，即使模型changes为空也必须继续保存。
func applyLessonPlanCapsuleTeacherChoice(
	document *models.LessonPlanContextCapsuleDocument,
	currentJSON, teacherMessage, turnID string,
) bool {
	if document == nil || strings.TrimSpace(currentJSON) == "" ||
		strings.TrimSpace(currentJSON) == "{}" {
		return false
	}

	selectedLabel, explicitChoice :=
		detectLessonPlanCapsuleTeacherChoice(teacherMessage)
	if !explicitChoice {
		return false
	}

	current := &models.LessonPlanContextCapsuleDocument{}
	if err := json.Unmarshal([]byte(currentJSON), current); err != nil {
		return false
	}

	selected, found :=
		findLessonPlanCapsuleSelectedOption(document, selectedLabel)
	if !found {
		selected, found =
			findLessonPlanCapsuleSelectedOption(current, selectedLabel)
	}
	if !found || strings.TrimSpace(selected.Item.Key) == "" {
		return false
	}

	competitors :=
		make(map[string]lessonPlanCapsuleOptionDescriptor)

	collectCompetitors :=
		func(options []lessonPlanCapsuleOptionDescriptor) {
			for _, option := range options {
				if option.Item.Key == selected.Item.Key ||
					option.Label == selected.Label ||
					!sameLessonPlanCapsuleOptionCategory(
						option.Category,
						selected.Category,
					) {
					continue
				}

				competitors[option.Item.Key] = option
			}
		}

	collectCompetitors(
		collectLessonPlanCapsulePositiveOptions(current),
	)
	collectCompetitors(
		collectLessonPlanCapsulePositiveOptions(document),
	)

	selectedWasSuperseded :=
		lessonPlanCapsuleContainsItem(
			current.SupersededItems,
			selected.Item.Key,
		) ||
			lessonPlanCapsuleContainsItem(
				document.SupersededItems,
				selected.Item.Key,
			)

	selectedNeedsPromotion :=
		selected.Item.Authority !=
			models.LessonPlanContextCapsuleAuthorityTeacherExplicit ||
			selected.Item.State !=
				models.LessonPlanContextCapsuleItemStateActive ||
			selected.Item.DoNotReconfirm ||
			strings.TrimSpace(selected.Item.ReplacedBy) != ""

	if len(competitors) == 0 &&
		!selectedWasSuperseded &&
		!selectedNeedsPromotion {
		return false
	}

	selectedItem := selected.Item
	selectedItem.State =
		models.LessonPlanContextCapsuleItemStateActive
	selectedItem.Authority =
		models.LessonPlanContextCapsuleAuthorityTeacherExplicit
	selectedItem.DoNotReconfirm = false
	selectedItem.ReplacedBy = ""

	if strings.TrimSpace(turnID) != "" {
		selectedItem.UpdatedByTurnID =
			strings.TrimSpace(turnID)
	}

	changed :=
		!lessonPlanCapsuleItemsEquivalent(
			selected.Item,
			selectedItem,
		) ||
			selectedWasSuperseded

	removeLessonPlanCapsulePositiveItemByKey(
		document,
		selectedItem.Key,
	)

	document.SupersededItems =
		removeLessonPlanCapsuleItem(
			document.SupersededItems,
			selectedItem.Key,
		)

	document.TeachingConsensus = append(
		document.TeachingConsensus,
		selectedItem,
	)

	document.StageFocus.AvoidRepeatingKeys =
		removeLessonPlanCapsuleString(
			document.StageFocus.AvoidRepeatingKeys,
			selectedItem.Key,
		)

	for _, competitor := range competitors {
		if strings.TrimSpace(competitor.Item.Key) == "" {
			continue
		}

		removeLessonPlanCapsulePositiveItemByKey(
			document,
			competitor.Item.Key,
		)

		superseded := competitor.Item
		superseded.State =
			models.LessonPlanContextCapsuleItemStateSuperseded
		superseded.DoNotReconfirm = true
		superseded.ReplacedBy = selectedItem.Key

		if strings.TrimSpace(turnID) != "" {
			superseded.UpdatedByTurnID =
				strings.TrimSpace(turnID)
		}

		document.SupersededItems =
			upsertLessonPlanCapsuleSupersededItem(
				document.SupersededItems,
				superseded,
			)

		document.StageFocus.AvoidRepeatingKeys =
			appendUniqueLessonPlanCapsuleString(
				document.StageFocus.AvoidRepeatingKeys,
				competitor.Item.Key,
				40,
			)

		changed = true
	}

	if !changed {
		return false
	}

	document.TeachingConsensus =
		normalizeLessonPlanCapsuleItems(
			document.TeachingConsensus,
			models.LessonPlanContextCapsuleItemStateActive,
			20,
			turnID,
			false,
		)

	document.SupersededItems =
		normalizeLessonPlanCapsuleItems(
			document.SupersededItems,
			models.LessonPlanContextCapsuleItemStateSuperseded,
			30,
			turnID,
			true,
		)

	return true
}

// detectLessonPlanCapsuleTeacherChoice 仅识别唯一且明确的方案选择。
func detectLessonPlanCapsuleTeacherChoice(
	message string,
) (string, bool) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", false
	}

	matches :=
		lessonPlanCapsuleOptionLabelPattern.
			FindAllStringSubmatchIndex(
				message,
				-1,
			)

	labels := make(map[string]string)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		canonical :=
			canonicalLessonPlanCapsuleOptionLabel(
				message[match[4]:match[5]],
			)

		if canonical != "" {
			labels[canonical] =
				message[match[0]:match[1]]
		}
	}

	if len(labels) != 1 {
		return "", false
	}

	selectedLabel := ""
	rawLabel := ""

	for label, raw := range labels {
		selectedLabel = label
		rawLabel = raw
	}

	compact :=
		compactLessonPlanCapsuleChoiceText(
			message,
		)

	compactLabel :=
		compactLessonPlanCapsuleChoiceText(
			rawLabel,
		)

	if compact == "" ||
		compactLabel == "" ||
		containsLessonPlanCapsuleChoiceQuestion(
			compact,
		) {
		return "", false
	}

	if compact == compactLabel {
		return selectedLabel, true
	}

	for _, prefix := range []string{
		"选择",
		"选定",
		"采用",
		"采纳",
		"就用",
		"决定用",
		"确定用",
		"改用",
		"换成",
		"按",
		"使用",
		"保留",
		"我选",
		"就",
	} {
		if strings.Contains(
			compact,
			prefix+compactLabel,
		) {
			return selectedLabel, true
		}
	}

	for _, suffix := range []string{
		"就好",
		"即可",
		"吧",
		"为主",
		"作为主线",
		"作为核心",
		"更合适",
		"更好",
	} {
		if strings.Contains(
			compact,
			compactLabel+suffix,
		) {
			return selectedLabel, true
		}
	}

	if strings.Contains(
		compact,
		"以"+compactLabel+"为主",
	) {
		return selectedLabel, true
	}

	// 建议芯片通常直接以“方案一：……”形式回传。
	for _, separator := range []string{
		"：",
		":",
		"-",
		"—",
	} {
		if strings.HasPrefix(
			compact,
			compactLabel+separator,
		) {
			return selectedLabel, true
		}
	}

	return "", false
}

func containsLessonPlanCapsuleChoiceQuestion(
	compact string,
) bool {
	for _, signal := range []string{
		"比较",
		"对比",
		"区别",
		"不同",
		"哪个",
		"哪一个",
		"哪种",
		"哪个好",
		"优缺点",
		"怎么选",
		"如何选",
		"还是",
		"或者",
		"分别",
		"都保留",
		"都采用",
		"都可以",
		"为什么",
		"是否",
		"能否",
		"可不可以",
		"什么",
		"怎么",
		"如何",
		"吗",
		"呢",
		"?",
		"？",
	} {
		if strings.Contains(compact, signal) {
			return true
		}
	}

	return false
}

func collectLessonPlanCapsulePositiveOptions(
	document *models.LessonPlanContextCapsuleDocument,
) []lessonPlanCapsuleOptionDescriptor {
	if document == nil {
		return nil
	}

	groups := [][]models.LessonPlanContextCapsuleItem{
		document.TeachingConsensus,
		document.OpenQuestions,
		document.DeferredItems,
	}

	output :=
		make([]lessonPlanCapsuleOptionDescriptor, 0)

	seen := make(map[string]struct{})

	for _, group := range groups {
		for _, item := range group {
			descriptor, ok :=
				describeLessonPlanCapsuleOptionItem(
					item,
				)

			if !ok ||
				strings.TrimSpace(item.Key) == "" {
				continue
			}

			if _, exists := seen[item.Key]; exists {
				continue
			}

			seen[item.Key] = struct{}{}
			output = append(output, descriptor)
		}
	}

	return output
}

func findLessonPlanCapsuleSelectedOption(
	document *models.LessonPlanContextCapsuleDocument,
	selectedLabel string,
) (
	lessonPlanCapsuleOptionDescriptor,
	bool,
) {
	if document == nil ||
		selectedLabel == "" {
		return lessonPlanCapsuleOptionDescriptor{},
			false
	}

	groups := [][]models.LessonPlanContextCapsuleItem{
		document.TeachingConsensus,
		document.OpenQuestions,
		document.DeferredItems,
		document.SupersededItems,
	}

	for _, group := range groups {
		for _, item := range group {
			descriptor, ok :=
				describeLessonPlanCapsuleOptionItem(
					item,
				)

			if ok &&
				descriptor.Label ==
					selectedLabel {
				return descriptor, true
			}
		}
	}

	return lessonPlanCapsuleOptionDescriptor{},
		false
}

func describeLessonPlanCapsuleOptionItem(
	item models.LessonPlanContextCapsuleItem,
) (
	lessonPlanCapsuleOptionDescriptor,
	bool,
) {
	for _, text := range []string{
		item.Title,
		item.Content,
	} {
		descriptor, ok :=
			describeLessonPlanCapsuleOptionText(
				text,
			)

		if ok {
			descriptor.Item = item
			return descriptor, true
		}
	}

	return lessonPlanCapsuleOptionDescriptor{},
		false
}

func describeLessonPlanCapsuleOptionText(
	text string,
) (
	lessonPlanCapsuleOptionDescriptor,
	bool,
) {
	text = strings.TrimSpace(text)

	matches :=
		lessonPlanCapsuleOptionLabelPattern.
			FindAllStringSubmatchIndex(
				text,
				-1,
			)

	if len(matches) != 1 ||
		len(matches[0]) < 6 {
		return lessonPlanCapsuleOptionDescriptor{},
			false
	}

	match := matches[0]

	label :=
		canonicalLessonPlanCapsuleOptionLabel(
			text[match[4]:match[5]],
		)

	if label == "" {
		return lessonPlanCapsuleOptionDescriptor{},
			false
	}

	category :=
		normalizeLessonPlanCapsuleOptionCategory(
			text[:match[0]],
		)

	if len([]rune(category)) > 10 {
		return lessonPlanCapsuleOptionDescriptor{},
			false
	}

	return lessonPlanCapsuleOptionDescriptor{
			Label:    label,
			Category: category,
		},
		true
}

func normalizeLessonPlanCapsuleOptionCategory(
	value string,
) string {
	value =
		compactLessonPlanCapsuleChoiceText(
			value,
		)

	value = strings.NewReplacer(
		"教师明确选择", "",
		"教师确认选择", "",
		"教师选择", "",
		"明确选择", "",
		"当前选择", "",
		"推荐", "",
		"备选", "",
		"候选", "",
		"选择", "",
		"采用", "",
		"确定", "",
	).Replace(value)

	return strings.TrimFunc(
		value,
		func(character rune) bool {
			return unicode.IsSpace(character) ||
				strings.ContainsRune(
					"：:-—·|/（）()【】[]，,。.",
					character,
				)
		},
	)
}

func canonicalLessonPlanCapsuleOptionLabel(
	value string,
) string {
	value = strings.ToUpper(
		strings.TrimSpace(value),
	)

	mapping := map[string]string{
		"一": "1",
		"二": "2",
		"三": "3",
		"四": "4",
		"五": "5",
		"六": "6",
		"七": "7",
		"八": "8",
		"九": "9",
		"十": "10",
	}

	if mapped, exists := mapping[value]; exists {
		return mapped
	}

	if len([]rune(value)) == 1 {
		return value
	}

	return ""
}

func compactLessonPlanCapsuleChoiceText(
	value string,
) string {
	return strings.Map(
		func(character rune) rune {
			if unicode.IsSpace(character) {
				return -1
			}

			return character
		},
		strings.TrimSpace(value),
	)
}

func sameLessonPlanCapsuleOptionCategory(
	left string,
	right string,
) bool {
	return strings.TrimSpace(left) ==
		strings.TrimSpace(right)
}

func removeLessonPlanCapsuleString(
	values []string,
	value string,
) []string {
	output := make(
		[]string,
		0,
		len(values),
	)

	for _, existing := range values {
		if existing != value {
			output = append(
				output,
				existing,
			)
		}
	}

	return output
}

func upsertLessonPlanCapsuleSupersededItem(
	items []models.LessonPlanContextCapsuleItem,
	item models.LessonPlanContextCapsuleItem,
) []models.LessonPlanContextCapsuleItem {
	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(items)+1,
	)

	updated := false

	for _, existing := range items {
		if existing.Key == item.Key {
			output = append(output, item)
			updated = true
			continue
		}

		output = append(output, existing)
	}

	if !updated {
		output = append(output, item)
	}

	return output
}

func lessonPlanCapsuleItemsEquivalent(
	left models.LessonPlanContextCapsuleItem,
	right models.LessonPlanContextCapsuleItem,
) bool {
	return left.Key == right.Key &&
		left.Title == right.Title &&
		left.Content == right.Content &&
		left.State == right.State &&
		left.Authority == right.Authority &&
		left.Importance == right.Importance &&
		left.DoNotReconfirm ==
			right.DoNotReconfirm &&
		left.ReplacedBy == right.ReplacedBy &&
		left.UpdatedByTurnID ==
			right.UpdatedByTurnID
}
