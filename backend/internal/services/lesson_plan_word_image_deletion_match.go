package services

// lesson_plan_word_image_deletion_match.go — Word图片删除差异匹配
//
// 本文件只判断当前Markdown是否由原Word语义正文“删除若干图片”得到。
//
// 安全原则：
//   - 删除图片前后的空行允许按正式Markdown规则归一化；
//   - 除图片Token外的全部文字和Markdown标记必须完全一致；
//   - 当前保留的图片必须能按Token、前文锚点和后文锚点唯一映射到原图片；
//   - 相同图片在同一文字锚点重复出现且无法唯一判断时拒绝自动处理；
//   - 不猜测图片移动、图片替换、alt修改、URL修改或文字修改。

type lessonPlanWordMarkdownImageOccurrence struct {
	Token         string
	TextBefore    string
	TextAfter     string
	OriginalIndex int
}

// matchLessonPlanWordImageDeletionOnly 判断current是否只比old少若干Markdown图片。
//
// 算法分为两层：
//  1. 移除双方全部图片后，剩余文字与Markdown结构必须完全一致；
//  2. 当前仍保留的每张图片，必须通过图片Token、前文锚点和后文锚点
//     唯一映射回原正文中的某一张图片。
//
// 这样既允许图片删除后空行自然合并，又不会把文字修改、图片移动或
// 无法唯一判断的重复图片误判成安全的纯图片删除。
func matchLessonPlanWordImageDeletionOnly(
	oldValue string,
	currentValue string,
) (map[int]bool, bool) {
	oldValue = canonicalizeLessonPlanWordSemanticMarkdown(oldValue)
	currentValue = canonicalizeLessonPlanWordSemanticMarkdown(currentValue)

	if oldValue == "" || currentValue == "" || oldValue == currentValue {
		return nil, false
	}

	oldOccurrences := collectLessonPlanWordMarkdownImageOccurrences(oldValue)
	currentOccurrences := collectLessonPlanWordMarkdownImageOccurrences(currentValue)

	if len(oldOccurrences) == 0 || len(currentOccurrences) >= len(oldOccurrences) {
		return nil, false
	}

	oldTextOnly := removeLessonPlanWordMarkdownImages(oldValue)
	currentTextOnly := removeLessonPlanWordMarkdownImages(currentValue)

	// 删除双方全部图片后，剩余文字和Markdown结构必须完全一致。
	// canonicalize会处理图片删除后两侧空行合并，但不会改变普通空格、
	// 文字、标题标记、列表标记或其它正文内容。
	if oldTextOnly == "" || oldTextOnly != currentTextOnly {
		return nil, false
	}

	matchedOldIndexes := make(map[int]bool, len(currentOccurrences))

	for _, currentOccurrence := range currentOccurrences {
		candidates := make([]int, 0, 1)

		for _, oldOccurrence := range oldOccurrences {
			if matchedOldIndexes[oldOccurrence.OriginalIndex] {
				continue
			}

			if oldOccurrence.Token != currentOccurrence.Token ||
				oldOccurrence.TextBefore != currentOccurrence.TextBefore ||
				oldOccurrence.TextAfter != currentOccurrence.TextAfter {
				continue
			}

			candidates = append(candidates, oldOccurrence.OriginalIndex)
		}

		// 找不到候选说明图片被修改、移动或替换。
		// 多于一个候选说明重复图片无法唯一定位，必须fail-closed。
		if len(candidates) != 1 {
			return nil, false
		}

		matchedOldIndexes[candidates[0]] = true
	}

	deletedIndexes := make(map[int]bool)

	for _, occurrence := range oldOccurrences {
		if matchedOldIndexes[occurrence.OriginalIndex] {
			continue
		}

		deletedIndexes[occurrence.OriginalIndex] = true
	}

	if len(deletedIndexes) == 0 {
		return nil, false
	}

	return deletedIndexes, true
}

// collectLessonPlanWordMarkdownImageOccurrences 收集图片Token及其无图片文字锚点。
//
// TextBefore和TextAfter都会移除其中的所有图片，因此图片删除不会改变
// 文字锚点。相同图片连续重复且前后文字完全相同时会形成多个候选，
// 上层匹配会安全拒绝，而不是猜测具体删除了哪一个图片节点。
func collectLessonPlanWordMarkdownImageOccurrences(
	value string,
) []lessonPlanWordMarkdownImageOccurrence {
	ranges := lessonPlanWordMarkdownImagePattern.FindAllStringIndex(value, -1)

	result := make(
		[]lessonPlanWordMarkdownImageOccurrence,
		0,
		len(ranges),
	)

	for index, tokenRange := range ranges {
		start := tokenRange[0]
		end := tokenRange[1]

		result = append(
			result,
			lessonPlanWordMarkdownImageOccurrence{
				Token:         value[start:end],
				TextBefore:    removeLessonPlanWordMarkdownImages(value[:start]),
				TextAfter:     removeLessonPlanWordMarkdownImages(value[end:]),
				OriginalIndex: index,
			},
		)
	}

	return result
}

// removeLessonPlanWordMarkdownImages 移除全部Markdown图片并归一换行和空行。
func removeLessonPlanWordMarkdownImages(
	value string,
) string {
	withoutImages := lessonPlanWordMarkdownImagePattern.ReplaceAllString(
		value,
		"",
	)

	return canonicalizeLessonPlanWordSemanticMarkdown(withoutImages)
}
