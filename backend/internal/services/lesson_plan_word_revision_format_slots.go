package services

// lesson_plan_word_revision_format_slots.go — 原Word格式投影Harness的槽位构建与应用
//
// 这里把当前正式Word语义Markdown转换为不可增删的单行槽位：
//   - 表格行锚点、图片和公式保持只读；
//   - “第N列：”、标题、列表标记、活动标签和短冒号标签作为固定前缀；
//   - AI只能返回已有槽位id对应的单行替换文字；
//   - 后端按基线原行号重建全文，结构数量由程序而不是模型决定。

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	aiClient "tedna/internal/ai"
)

var (
	lessonPlanWordRevisionFormatColumnPrefixPattern     = regexp.MustCompile(`^(\s*-?\s*第\d+列[：:]\s*)(.*)$`)
	lessonPlanWordRevisionFormatHeadingPrefixPattern    = regexp.MustCompile(`^(\s*#{1,6}\s+)(.*)$`)
	lessonPlanWordRevisionFormatBulletPrefixPattern     = regexp.MustCompile(`^(\s*[-*+]\s+)(.*)$`)
	lessonPlanWordRevisionFormatNumberPrefixPattern     = regexp.MustCompile(`^(\s*\d+[.、．)]\s*)(.*)$`)
	lessonPlanWordRevisionFormatActionPrefixPattern     = regexp.MustCompile(`^(\s*【[^】]{1,30}】\s*)(.*)$`)
	lessonPlanWordRevisionFormatShortLabelPrefixPattern = regexp.MustCompile(`^(\s*[^：:\n]{1,40}[：:]\s*)(.*)$`)
	lessonPlanWordRevisionFormatReplacementLinePattern  = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:slot|s)\s*(\d+)\s*[|:：]\s*(.*)\s*$`)
)

type lessonPlanWordFormatSlot struct {
	ID        int
	LineIndex int
	Prefix    string
	Original  string
}

type lessonPlanWordFormatReplacement struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

type lessonPlanWordFormatRepairResponse struct {
	Replacements []lessonPlanWordFormatReplacement `json:"replacements"`
}

func buildLessonPlanWordFormatSlots(
	baseline string,
) []lessonPlanWordFormatSlot {
	lines :=
		strings.Split(
			baseline,
			"\n",
		)
	slots :=
		make(
			[]lessonPlanWordFormatSlot,
			0,
			len(lines),
		)

	for index, line := range lines {
		if strings.TrimSpace(line) == "" ||
			isLessonPlanWordImmutableFormatLine(
				line,
			) ||
			len(
				lessonPlanWordImageMarkdownPattern.
					FindAllString(
						line,
						-1,
					),
			) > 0 ||
			len(
				lessonPlanWordFormulaMarkdownPattern.
					FindAllString(
						line,
						-1,
					),
			) > 0 {
			continue
		}

		prefix, editable :=
			splitLessonPlanWordFormatSlot(
				line,
			)

		slots = append(
			slots,
			lessonPlanWordFormatSlot{
				ID:        len(slots) + 1,
				LineIndex: index,
				Prefix:    prefix,
				Original:  editable,
			},
		)
	}

	return slots
}

func isLessonPlanWordImmutableFormatLine(
	line string,
) bool {
	normalized :=
		normalizeLessonPlanWordAnchorLine(
			line,
		)

	if normalized == "" {
		return true
	}

	return isLessonPlanWordTableAnchorLine(
		normalized,
	) ||
		strings.EqualFold(
			normalized,
			"表格标签",
		) ||
		strings.HasPrefix(
			normalized,
			"<WORD_FIDELITY_",
		)
}

func splitLessonPlanWordFormatSlot(
	line string,
) (
	string,
	string,
) {
	for _, pattern := range []*regexp.Regexp{
		lessonPlanWordRevisionFormatColumnPrefixPattern,
		lessonPlanWordRevisionFormatHeadingPrefixPattern,
		lessonPlanWordRevisionFormatBulletPrefixPattern,
		lessonPlanWordRevisionFormatNumberPrefixPattern,
		lessonPlanWordRevisionFormatActionPrefixPattern,
		lessonPlanWordRevisionFormatShortLabelPrefixPattern,
	} {
		matches :=
			pattern.FindStringSubmatch(
				line,
			)
		if len(matches) == 3 {
			return matches[1],
				matches[2]
		}
	}

	leadingLength :=
		len(line) -
			len(
				strings.TrimLeft(
					line,
					" \t",
				),
			)
	return line[:leadingLength],
		line[leadingLength:]
}

func parseLessonPlanWordFormatRepairResponse(
	raw string,
) (
	[]lessonPlanWordFormatReplacement,
	error,
) {
	jsonText, ok :=
		aiClient.ExtractJSON(
			raw,
		)
	if ok {
		var response lessonPlanWordFormatRepairResponse
		if err :=
			json.Unmarshal(
				[]byte(jsonText),
				&response,
			); err == nil &&
			len(response.Replacements) > 0 {
			return response.Replacements,
				nil
		}
	}

	/*
	 * 少数模型会把JSON协议写成“S12 | 新文字”的逐行格式。
	 * 这里只接受带S/slot前缀的明确槽位行，避免把普通编号说明误当成修改。
	 */
	fallback :=
		make(
			[]lessonPlanWordFormatReplacement,
			0,
		)
	for _, line := range strings.Split(
		raw,
		"\n",
	) {
		matches :=
			lessonPlanWordRevisionFormatReplacementLinePattern.
				FindStringSubmatch(
					line,
				)
		if len(matches) != 3 {
			continue
		}

		var id int
		if _, err :=
			fmt.Sscanf(
				matches[1],
				"%d",
				&id,
			); err != nil {
			continue
		}

		fallback = append(
			fallback,
			lessonPlanWordFormatReplacement{
				ID:   id,
				Text: matches[2],
			},
		)
	}

	if len(fallback) == 0 {
		return nil,
			errors.New(
				"格式投影Harness没有返回可解析的槽位修改",
			)
	}

	return fallback,
		nil
}

func applyLessonPlanWordFormatReplacements(
	baseline string,
	slots []lessonPlanWordFormatSlot,
	replacements []lessonPlanWordFormatReplacement,
) (
	string,
	int,
	error,
) {
	lines :=
		strings.Split(
			baseline,
			"\n",
		)
	slotByID :=
		make(
			map[int]lessonPlanWordFormatSlot,
			len(slots),
		)
	for _, slot := range slots {
		slotByID[slot.ID] =
			slot
	}

	seen :=
		make(
			map[int]struct{},
			len(replacements),
		)
	changedCount := 0

	for _, replacement := range replacements {
		slot, ok :=
			slotByID[replacement.ID]
		if !ok {
			return "", 0,
				fmt.Errorf(
					"格式投影返回未知槽位id=%d",
					replacement.ID,
				)
		}
		if _, duplicated :=
			seen[replacement.ID]; duplicated {
			return "", 0,
				fmt.Errorf(
					"格式投影重复返回槽位id=%d",
					replacement.ID,
				)
		}
		seen[replacement.ID] =
			struct{}{}

		if strings.ContainsAny(
			replacement.Text,
			"\r\n\t",
		) {
			return "", 0,
				fmt.Errorf(
					"槽位id=%d包含禁止的换行或制表符",
					replacement.ID,
				)
		}
		if len(
			lessonPlanWordImageMarkdownPattern.
				FindAllString(
					replacement.Text,
					-1,
				),
		) > 0 ||
			len(
				lessonPlanWordFormulaMarkdownPattern.
					FindAllString(
						replacement.Text,
						-1,
					),
			) > 0 {
			return "", 0,
				fmt.Errorf(
					"槽位id=%d试图写入图片或公式标记",
					replacement.ID,
				)
		}

		nextText :=
			strings.TrimSpace(
				replacement.Text,
			)
		if nextText ==
			strings.TrimSpace(
				slot.Original,
			) {
			continue
		}

		lines[slot.LineIndex] =
			slot.Prefix +
				nextText
		changedCount++
	}

	if changedCount == 0 {
		return "", 0,
			errors.New(
				"格式投影没有形成实际修改",
			)
	}

	projected :=
		strings.Join(
			lines,
			"\n",
		)

	currentImages :=
		lessonPlanWordImageMarkdownPattern.
			FindAllString(
				baseline,
				-1,
			)
	projectedImages :=
		lessonPlanWordImageMarkdownPattern.
			FindAllString(
				projected,
				-1,
			)
	if !equalLessonPlanWordStrings(
		currentImages,
		projectedImages,
	) {
		return "", 0,
			ErrLessonPlanWordProtectedImageChanged
	}

	currentFormulas :=
		lessonPlanWordFormulaMarkdownPattern.
			FindAllString(
				baseline,
				-1,
			)
	projectedFormulas :=
		lessonPlanWordFormulaMarkdownPattern.
			FindAllString(
				projected,
				-1,
			)
	if !equalLessonPlanWordStrings(
		currentFormulas,
		projectedFormulas,
	) {
		return "", 0,
			ErrLessonPlanWordFormulaChangeUnsupported
	}

	return projected,
		changedCount,
		nil
}
