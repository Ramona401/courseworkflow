package services

// lesson_plan_context_capsule_progress_sections.go — 教案环节与确认范围识别
//
// 本文件负责：
//   - 提取“环节一、环节二”“环节三和四”等编号；
//   - 从历史共识与进度文字恢复已确认范围；
//   - 从进度文字恢复待确认范围；
//   - 识别教师明确点名确认的教案环节；
//   - 识别AI是否真正生成了详细教案正文；
//   - 排除只有标题和时间分配的简短教学框架。

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tedna/internal/models"
)

var (
	lessonPlanCapsuleSectionPattern = regexp.MustCompile(
		`环节[[:space:]]*([一二三四五六七八九十0-9]+)`,
	)

	// 支持：
	//   - 环节三和环节四；
	//   - 环节三和四；
	//   - 环节一、环节二、环节三；
	//   - 环节1、2、3。
	lessonPlanCapsuleSectionGroupPattern = regexp.MustCompile(
		`环节[[:space:]]*([一二三四五六七八九十0-9]+(?:[[:space:]]*(?:、|，|,|和|及|与)[[:space:]]*(?:环节[[:space:]]*)?[一二三四五六七八九十0-9]+)*)`,
	)

	lessonPlanCapsuleSectionSeparatorPattern = regexp.MustCompile(
		`[[:space:]]*(?:、|，|,|和|及|与)[[:space:]]*`,
	)

	lessonPlanCapsuleDetailedSectionHeadingPattern = regexp.MustCompile(
		`(?m)^[[:space:]]*(?:#{1,6}[[:space:]]*)?(?:\*\*)?环节[[:space:]]*([一二三四五六七八九十0-9]+)[[:space:]]*[：:]`,
	)

	lessonPlanCapsuleConfirmedPrefixPattern = regexp.MustCompile(
		`(?:教师已确认|教师确认|已经确认|已确认)([^。；;\n]*?)(?:的(?:具体|详细)?教案内容|(?:具体|详细)?教案内容)`,
	)

	lessonPlanCapsuleConfirmedSuffixPattern = regexp.MustCompile(
		`([^。；;\n]*?)(?:已经确认|已确认)`,
	)

	// “确认环节三……”中，“确认”和“环节”必须紧邻。
	// “确认，请开始写环节三……”不会命中。
	lessonPlanCapsuleExplicitConfirmPrefixPattern = regexp.MustCompile(
		`^(?:我)?(?:可以)?确认(?:一下)?[[:space:]]*环节`,
	)

	// 同时兼容“环节三和环节四可以确认”。
	lessonPlanCapsuleExplicitConfirmSuffixPattern = regexp.MustCompile(
		`环节[^。；;\r\n]{0,80}(?:可以|已经|已)?确认(?:了)?(?:[，,][^。；;\r\n]*)?$`,
	)

	lessonPlanCapsuleProgressClausePattern = regexp.MustCompile(
		`[。；;\r\n]+`,
	)
)

// lessonPlanCapsuleConfirmedSectionsFromDocument 提取历史已确认环节。
func lessonPlanCapsuleConfirmedSectionsFromDocument(
	document *models.LessonPlanContextCapsuleDocument,
	includeProgressText bool,
) []int {
	if document == nil {
		return nil
	}

	output := make(
		[]int,
		0,
	)

	for _, item := range document.TeachingConsensus {
		if item.State != "" &&
			item.State !=
				models.LessonPlanContextCapsuleItemStateActive {
			continue
		}

		if item.Authority !=
			models.LessonPlanContextCapsuleAuthorityTeacherExplicit &&
			item.Authority !=
				models.LessonPlanContextCapsuleAuthorityTeacherSourceConfirmed {
			continue
		}

		if !lessonPlanCapsuleConfirmationProgressItem(
			item,
		) {
			continue
		}

		output = append(
			output,
			lessonPlanCapsuleSectionNumbersFromText(
				strings.Join(
					[]string{
						item.Key,
						item.Title,
						item.Content,
					},
					" ",
				),
			)...,
		)
	}

	if includeProgressText {
		output = append(
			output,
			lessonPlanCapsuleConfirmedSectionsFromText(
				document.Summary,
			)...,
		)

		output = append(
			output,
			lessonPlanCapsuleConfirmedSectionsFromText(
				document.StageFocus.CurrentTask,
			)...,
		)
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleConfirmedSectionsFromText 只解析明确确认语义中的编号。
//
// 对于“已确认环节一、环节二，后续撰写环节三、环节四”，
// 只提取确认语句中的环节一和环节二。
func lessonPlanCapsuleConfirmedSectionsFromText(
	text string,
) []int {
	text =
		strings.TrimSpace(
			text,
		)

	if text == "" {
		return nil
	}

	output := make(
		[]int,
		0,
	)

	for _, match := range lessonPlanCapsuleConfirmedPrefixPattern.
		FindAllStringSubmatch(
			text,
			-1,
		) {
		if len(match) < 2 {
			continue
		}

		output = append(
			output,
			lessonPlanCapsuleSectionNumbersFromText(
				match[1],
			)...,
		)
	}

	for _, clause := range lessonPlanCapsuleProgressClauses(
		text,
	) {
		if lessonPlanCapsuleProgressIsPending(
			clause,
		) {
			continue
		}

		for _, match := range lessonPlanCapsuleConfirmedSuffixPattern.
			FindAllStringSubmatch(
				clause,
				-1,
			) {
			if len(match) < 2 {
				continue
			}

			output = append(
				output,
				lessonPlanCapsuleSectionNumbersFromText(
					match[1],
				)...,
			)
		}
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsulePendingSectionsFromDocument 读取上一版待确认范围。
func lessonPlanCapsulePendingSectionsFromDocument(
	document *models.LessonPlanContextCapsuleDocument,
) []int {
	if document == nil {
		return nil
	}

	output := make(
		[]int,
		0,
	)

	for _, text := range []string{
		document.Summary,
		document.StageFocus.CurrentTask,
	} {
		for _, clause := range lessonPlanCapsuleProgressClauses(
			text,
		) {
			if !lessonPlanCapsuleProgressIsPending(
				clause,
			) {
				continue
			}

			output = append(
				output,
				lessonPlanCapsuleSectionNumbersFromText(
					clause,
				)...,
			)
		}
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleProgressClauses 将文本拆成独立语义分句。
func lessonPlanCapsuleProgressClauses(
	text string,
) []string {
	raw :=
		lessonPlanCapsuleProgressClausePattern.
			Split(
				strings.TrimSpace(text),
				-1,
			)

	output := make(
		[]string,
		0,
		len(raw),
	)

	for _, clause := range raw {
		clause =
			strings.TrimSpace(
				clause,
			)

		if clause != "" {
			output = append(
				output,
				clause,
			)
		}
	}

	return output
}

// lessonPlanCapsuleHasConfirmationSignal 识别明确确认语义。
func lessonPlanCapsuleHasConfirmationSignal(
	text string,
) bool {
	for _, signal := range []string{
		"教师已确认",
		"教师确认",
		"已确认",
		"已经确认",
		"确认了",
		"确认并",
		"教案确认",
		"已撰写并确认",
	} {
		if strings.Contains(
			text,
			signal,
		) {
			return true
		}
	}

	return false
}

// lessonPlanCapsuleExplicitlyConfirmedSections 提取教师明确点名确认的环节。
//
// 可以识别：
//   - “确认环节三和环节四”；
//   - “确认环节三、四，请继续”；
//   - “环节三和环节四可以确认”。
//
// 不会识别：
//   - “确认，请开始写环节三和环节四”；
//   - “暂不确认环节三和环节四”；
//   - “请确认环节三和环节四”；
//   - “确认后再继续写环节三”。
func lessonPlanCapsuleExplicitlyConfirmedSections(
	message string,
) []int {
	output := make(
		[]int,
		0,
	)

	for _, clause := range lessonPlanCapsuleProgressClauses(
		message,
	) {
		raw :=
			strings.TrimSpace(
				clause,
			)

		normalized :=
			normalizeLessonPlanTurnText(
				raw,
			)

		if normalized == "" {
			continue
		}

		negative := false

		for _, signal := range []string{
			"暂不确认",
			"先不确认",
			"不确认",
			"不能确认",
			"不要确认",
			"不算确认",
			"不视为确认",
			"尚未确认",
			"还未确认",
			"未确认",
			"确认后",
			"等待确认",
			"待确认",
			"请确认",
			"是否确认",
			"能否确认",
			"可否确认",
		} {
			if strings.Contains(
				normalized,
				signal,
			) {
				negative = true
				break
			}
		}

		if negative {
			continue
		}

		sections :=
			lessonPlanCapsuleSectionNumbersFromText(
				raw,
			)

		if len(sections) == 0 {
			continue
		}

		explicit :=
			lessonPlanCapsuleExplicitConfirmPrefixPattern.
				MatchString(raw) ||
				lessonPlanCapsuleExplicitConfirmSuffixPattern.
					MatchString(raw)

		if !explicit {
			continue
		}

		output = append(
			output,
			sections...,
		)
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleProgressIsPending 统一识别待确认表述。
func lessonPlanCapsuleProgressIsPending(
	text string,
) bool {
	for _, signal := range []string{
		"待确认",
		"等待教师确认",
		"等待确认",
		"尚待确认",
		"仍待确认",
		"需要教师确认",
		"需教师确认",
	} {
		if strings.Contains(
			text,
			signal,
		) {
			return true
		}
	}

	return false
}

// lessonPlanCapsuleConfirmationProgressItem 判断条目是否是教案确认进度。
func lessonPlanCapsuleConfirmationProgressItem(
	item models.LessonPlanContextCapsuleItem,
) bool {
	if item.Key ==
		lessonPlanCapsuleConfirmedSectionsKey {
		return true
	}

	combined :=
		strings.Join(
			[]string{
				item.Key,
				item.Title,
				item.Content,
			},
			" ",
		)

	if len(
		lessonPlanCapsuleSectionNumbersFromText(
			combined,
		),
	) == 0 {
		return false
	}

	return lessonPlanCapsuleHasConfirmationSignal(
		combined,
	) ||
		strings.Contains(
			item.Key,
			"lesson_plan_part",
		) ||
		strings.Contains(
			item.Key,
			"lesson_plan_section",
		)
}

// lessonPlanCapsuleDetailedGeneratedSections 只识别详细教案正文。
func lessonPlanCapsuleDetailedGeneratedSections(
	text string,
) []int {
	text =
		strings.TrimSpace(
			text,
		)

	if text == "" {
		return nil
	}

	matches :=
		lessonPlanCapsuleDetailedSectionHeadingPattern.
			FindAllStringSubmatchIndex(
				text,
				-1,
			)

	output := make(
		[]int,
		0,
		len(matches),
	)

	for index, match := range matches {
		if len(match) < 4 {
			continue
		}

		segmentEnd := len(text)

		if index+1 < len(matches) {
			segmentEnd =
				matches[index+1][0]
		}

		segment :=
			strings.TrimSpace(
				text[match[0]:segmentEnd],
			)

		if !lessonPlanCapsuleSectionSegmentIsDetailed(
			segment,
		) {
			continue
		}

		number, ok :=
			lessonPlanCapsuleSectionNumber(
				text[match[2]:match[3]],
			)

		if ok {
			output = append(
				output,
				number,
			)
		}
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleSectionSegmentIsDetailed 判断是否达到详细教案粒度。
func lessonPlanCapsuleSectionSegmentIsDetailed(
	segment string,
) bool {
	runeCount :=
		len([]rune(segment))

	if runeCount >= 120 {
		return true
	}

	if runeCount < 45 {
		return false
	}

	for _, marker := range []string{
		"教师话术",
		"学生活动",
		"学生预期反应",
		"教师活动",
		"评价标准",
		"设计意图",
		"过渡语",
		"任务支架",
		"操作方式",
		"教师引导",
	} {
		if strings.Contains(
			segment,
			marker,
		) {
			return true
		}
	}

	return false
}

// lessonPlanCapsuleSectionNumbersFromText 提取教案环节编号。
func lessonPlanCapsuleSectionNumbersFromText(
	text string,
) []int {
	text =
		strings.TrimSpace(
			text,
		)

	groups :=
		lessonPlanCapsuleSectionGroupPattern.
			FindAllStringSubmatch(
				text,
				-1,
			)

	output := make(
		[]int,
		0,
	)

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		parts :=
			lessonPlanCapsuleSectionSeparatorPattern.
				Split(
					group[1],
					-1,
				)

		for _, part := range parts {
			part =
				strings.TrimSpace(
					part,
				)

			part =
				strings.TrimSpace(
					strings.TrimPrefix(
						part,
						"环节",
					),
				)

			number, ok :=
				lessonPlanCapsuleSectionNumber(
					part,
				)

			if ok {
				output = append(
					output,
					number,
				)
			}
		}
	}

	// 保留对极端异常文本中单个“环节X”的兼容。
	if len(groups) == 0 {
		matches :=
			lessonPlanCapsuleSectionPattern.
				FindAllStringSubmatch(
					text,
					-1,
				)

		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			number, ok :=
				lessonPlanCapsuleSectionNumber(
					match[1],
				)

			if ok {
				output = append(
					output,
					number,
				)
			}
		}
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleSectionNumber 转换中文或阿拉伯编号。
func lessonPlanCapsuleSectionNumber(
	value string,
) (int, bool) {
	value =
		strings.TrimSpace(
			value,
		)

	if number, err :=
		strconv.Atoi(
			value,
		); err == nil {
		if number >= 1 &&
			number <= 20 {
			return number, true
		}

		return 0, false
	}

	mapping := map[string]int{
		"一": 1,
		"二": 2,
		"三": 3,
		"四": 4,
		"五": 5,
		"六": 6,
		"七": 7,
		"八": 8,
		"九": 9,
		"十": 10,
	}

	number, exists :=
		mapping[value]

	return number, exists
}

// mergeLessonPlanCapsuleSectionSet 合并编号集合。
func mergeLessonPlanCapsuleSectionSet(
	target map[int]struct{},
	values []int,
) {
	for _, value := range values {
		if value >= 1 &&
			value <= 20 {
			target[value] =
				struct{}{}
		}
	}
}

// lessonPlanCapsuleRemoveConfirmedSections 删除已确认的待确认编号。
func lessonPlanCapsuleRemoveConfirmedSections(
	values []int,
	confirmed map[int]struct{},
) []int {
	output := make(
		[]int,
		0,
		len(values),
	)

	for _, value := range values {
		if _, exists :=
			confirmed[value]; exists {
			continue
		}

		output = append(
			output,
			value,
		)
	}

	return lessonPlanCapsuleUniqueSortedSections(
		output,
	)
}

// lessonPlanCapsuleUniqueSortedSections 去重并升序排列。
func lessonPlanCapsuleUniqueSortedSections(
	values []int,
) []int {
	seen :=
		make(map[int]struct{})

	output := make(
		[]int,
		0,
		len(values),
	)

	for _, value := range values {
		if value < 1 ||
			value > 20 {
			continue
		}

		if _, exists :=
			seen[value]; exists {
			continue
		}

		seen[value] =
			struct{}{}

		output = append(
			output,
			value,
		)
	}

	sort.Ints(
		output,
	)

	return output
}
