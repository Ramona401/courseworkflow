package services

// lesson_plan_context_capsule_resilience.go — 胶囊模型输入压缩与JSON容错
//
// 安全边界：
//   - 只修复明确可判定的Markdown围栏和JSON尾逗号；
//   - 不猜测缺失的引号、对象、数组或业务字段；
//   - 截断JSON不会在本地强行补齐，而是交由模型独立重试一次；
//   - 当前active胶囊只做模型输入副本压缩，不修改数据库原文；
//   - 压缩不会改变条目key、权威等级、状态、替代关系和负向记忆标志。

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
)

const (
	lessonPlanContextCapsuleModelSummaryRunes        = 260
	lessonPlanContextCapsuleModelItemTitleRunes      = 120
	lessonPlanContextCapsuleModelItemContentRunes    = 320
	lessonPlanContextCapsuleModelStageTaskRunes      = 220
	lessonPlanContextCapsuleModelCourseCoreLimit     = 10
	lessonPlanContextCapsuleModelConsensusLimit      = 14
	lessonPlanContextCapsuleModelConstraintLimit     = 10
	lessonPlanContextCapsuleModelQuestionLimit       = 8
	lessonPlanContextCapsuleModelDeferredLimit       = 8
	lessonPlanContextCapsuleModelSupersededLimit     = 24
	lessonPlanContextCapsuleModelSourceKeyLimit      = 10
	lessonPlanContextCapsuleModelApplicableLimit     = 8
	lessonPlanContextCapsuleModelCarryForwardLimit   = 24
	lessonPlanContextCapsuleModelAvoidRepeatingLimit = 32
)

var lessonPlanContextCapsuleTrailingCommaPattern = regexp.MustCompile(`,\s*([}\]])`)

// parseLessonPlanContextCapsuleAIResult 解析模型返回的完整胶囊结果。
//
// 解析顺序：
//  1. 使用平台通用ExtractJSON；
//  2. 去除Markdown代码围栏；
//  3. 提取第一个“{”至最后一个“}”；
//  4. 移除对象或数组结束符前的尾逗号。
//
// 不对缺失结束括号、缺失引号等截断内容做猜测性修复。
func parseLessonPlanContextCapsuleAIResult(
	raw string,
) (
	*models.LessonPlanContextCapsuleAIResult,
	error,
) {
	candidates :=
		lessonPlanContextCapsuleJSONCandidates(
			raw,
		)

	if len(candidates) == 0 {
		return nil, errors.New(
			"胶囊旁路AI结果没有可解析的JSON候选",
		)
	}

	var lastError error

	for _, candidate := range candidates {
		repaired :=
			repairLessonPlanContextCapsuleJSON(
				candidate,
			)

		if !json.Valid([]byte(repaired)) {
			lastError = errors.New(
				"胶囊JSON候选语法无效",
			)
			continue
		}

		result :=
			&models.LessonPlanContextCapsuleAIResult{}

		if err := json.Unmarshal(
			[]byte(repaired),
			result,
		); err != nil {
			lastError = err
			continue
		}

		return result, nil
	}

	if lastError == nil {
		lastError = errors.New(
			"胶囊旁路AI结果不是合法JSON",
		)
	}

	return nil, fmt.Errorf(
		"解析胶囊旁路AI结果失败: %w",
		lastError,
	)
}

// lessonPlanContextCapsuleJSONCandidates 形成去重后的JSON候选。
func lessonPlanContextCapsuleJSONCandidates(
	raw string,
) []string {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return nil
	}

	output := make([]string, 0, 4)
	seen := make(map[string]struct{})

	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)

		if value == "" {
			return
		}

		if _, exists := seen[value]; exists {
			return
		}

		seen[value] = struct{}{}
		output = append(output, value)
	}

	if extracted, ok :=
		aiClient.ExtractJSON(raw); ok {
		appendCandidate(extracted)
	}

	withoutFence :=
		stripLessonPlanContextCapsuleMarkdownFence(
			raw,
		)

	appendCandidate(withoutFence)

	firstBrace := strings.Index(
		withoutFence,
		"{",
	)
	lastBrace := strings.LastIndex(
		withoutFence,
		"}",
	)

	if firstBrace >= 0 &&
		lastBrace > firstBrace {
		appendCandidate(
			withoutFence[firstBrace : lastBrace+1],
		)
	}

	appendCandidate(raw)

	return output
}

// stripLessonPlanContextCapsuleMarkdownFence 移除外围Markdown代码围栏。
func stripLessonPlanContextCapsuleMarkdownFence(
	value string,
) string {
	value = strings.TrimSpace(value)

	if !strings.HasPrefix(value, "```") {
		return value
	}

	lines := strings.Split(
		value,
		"\n",
	)

	if len(lines) < 2 {
		return value
	}

	lines = lines[1:]

	if len(lines) > 0 &&
		strings.HasPrefix(
			strings.TrimSpace(
				lines[len(lines)-1],
			),
			"```",
		) {
		lines = lines[:len(lines)-1]
	}

	return strings.TrimSpace(
		strings.Join(lines, "\n"),
	)
}

// repairLessonPlanContextCapsuleJSON 修复明确安全的尾逗号。
func repairLessonPlanContextCapsuleJSON(
	value string,
) string {
	value = strings.TrimSpace(value)

	for attempt := 0; attempt < 4; attempt++ {
		repaired :=
			lessonPlanContextCapsuleTrailingCommaPattern.
				ReplaceAllString(
					value,
					"$1",
				)

		if repaired == value {
			break
		}

		value = repaired
	}

	return strings.TrimSpace(value)
}

// compactLessonPlanContextCapsuleForModel 构造仅供模型读取的当前胶囊副本。
//
// 数据库中的原始CapsuleJSON不会被修改。
// 压缩目标是减少长对话后模型重新输出完整JSON时发生截断的概率。
func compactLessonPlanContextCapsuleForModel(
	currentJSON string,
) map[string]interface{} {
	output := map[string]interface{}{}

	currentJSON = strings.TrimSpace(
		currentJSON,
	)

	if currentJSON == "" ||
		currentJSON == "{}" {
		return output
	}

	document :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		[]byte(currentJSON),
		document,
	); err != nil {
		return output
	}

	document.Summary =
		compactLessonPlanContextCapsuleTextForModel(
			document.Summary,
			lessonPlanContextCapsuleModelSummaryRunes,
		)

	document.CourseCore =
		compactLessonPlanContextCapsuleItemsForModel(
			document.CourseCore,
			lessonPlanContextCapsuleModelCourseCoreLimit,
		)

	document.TeachingConsensus =
		compactLessonPlanContextCapsuleItemsForModel(
			document.TeachingConsensus,
			lessonPlanContextCapsuleModelConsensusLimit,
		)

	document.Constraints =
		compactLessonPlanContextCapsuleItemsForModel(
			document.Constraints,
			lessonPlanContextCapsuleModelConstraintLimit,
		)

	document.OpenQuestions =
		compactLessonPlanContextCapsuleItemsForModel(
			document.OpenQuestions,
			lessonPlanContextCapsuleModelQuestionLimit,
		)

	document.DeferredItems =
		compactLessonPlanContextCapsuleItemsForModel(
			document.DeferredItems,
			lessonPlanContextCapsuleModelDeferredLimit,
		)

	document.SupersededItems =
		compactLessonPlanContextCapsuleItemsForModel(
			document.SupersededItems,
			lessonPlanContextCapsuleModelSupersededLimit,
		)

	document.StageFocus.CurrentTask =
		compactLessonPlanContextCapsuleTextForModel(
			document.StageFocus.CurrentTask,
			lessonPlanContextCapsuleModelStageTaskRunes,
		)

	document.StageFocus.CarryForwardKeys =
		compactLessonPlanContextCapsuleStringListForModel(
			document.StageFocus.CarryForwardKeys,
			lessonPlanContextCapsuleModelCarryForwardLimit,
		)

	document.StageFocus.AvoidRepeatingKeys =
		compactLessonPlanContextCapsuleStringListForModel(
			document.StageFocus.AvoidRepeatingKeys,
			lessonPlanContextCapsuleModelAvoidRepeatingLimit,
		)

	encoded, err := json.Marshal(document)
	if err != nil {
		return output
	}

	if err := json.Unmarshal(
		encoded,
		&output,
	); err != nil {
		return map[string]interface{}{}
	}

	return output
}

// compactLessonPlanContextCapsuleItemsForModel 压缩条目文字和列表长度。
func compactLessonPlanContextCapsuleItemsForModel(
	items []models.LessonPlanContextCapsuleItem,
	limit int,
) []models.LessonPlanContextCapsuleItem {
	if limit <= 0 ||
		len(items) == 0 {
		return nil
	}

	if len(items) > limit {
		items = items[:limit]
	}

	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(items),
	)

	for _, item := range items {
		item.Title =
			compactLessonPlanContextCapsuleTextForModel(
				item.Title,
				lessonPlanContextCapsuleModelItemTitleRunes,
			)

		item.Content =
			compactLessonPlanContextCapsuleTextForModel(
				item.Content,
				lessonPlanContextCapsuleModelItemContentRunes,
			)

		item.SourceKeys =
			compactLessonPlanContextCapsuleStringListForModel(
				item.SourceKeys,
				lessonPlanContextCapsuleModelSourceKeyLimit,
			)

		item.ApplicableStages =
			compactLessonPlanContextCapsuleStringListForModel(
				item.ApplicableStages,
				lessonPlanContextCapsuleModelApplicableLimit,
			)

		output = append(output, item)
	}

	return output
}

// compactLessonPlanContextCapsuleStringListForModel 清理模型输入中的字符串列表。
func compactLessonPlanContextCapsuleStringListForModel(
	values []string,
	limit int,
) []string {
	if limit <= 0 ||
		len(values) == 0 {
		return nil
	}

	output := make([]string, 0, len(values))
	seen := make(map[string]struct{})

	for _, value := range values {
		value = compactLessonPlanContextCapsuleTextForModel(
			value,
			200,
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

// compactLessonPlanContextCapsuleTextForModel 按Unicode字符限制模型输入。
func compactLessonPlanContextCapsuleTextForModel(
	value string,
	limit int,
) string {
	value = strings.TrimSpace(value)

	if value == "" ||
		limit <= 0 {
		return ""
	}

	runes := []rune(value)

	if len(runes) <= limit {
		return value
	}

	return string(runes[:limit])
}

// lessonPlanContextCapsuleRetrySystemPrompt 构造一次性JSON重试提示。
//
// 重试仍使用原始结构化输入，不把损坏的第一次输出再次喂给模型，
// 避免模型围绕截断内容继续补写并放大错误。
func lessonPlanContextCapsuleRetrySystemPrompt() string {
	return lessonPlanContextCapsuleSystemPrompt + `

【本次为JSON受控重试】
上一次输出没有通过严格JSON解析。请重新独立完成整个任务，不要续写、引用或解释上一次输出。

额外硬规则：
1. 第一个字符必须是“{”，最后一个字符必须是“}”。
2. 不得输出Markdown代码围栏、说明文字、注释或省略号。
3. 所有字符串中的换行必须正确转义。
4. 对象和数组最后一项后不得出现逗号。
5. 不得截断任何对象、数组或字符串。
6. 内容较多时必须先合并重复条目并压缩措辞，不得突破条目数量上限。
7. changes和evidence_bindings即使为空，也必须输出合法空数组。`
}
