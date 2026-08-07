package services

// lesson_plan_context_capsule_presentation.go — 胶囊运行时短版与教师端安全展示
//
// 本文件负责两个严格分离的输出：
//
// 1. AI运行时短版：
//   - 只包含可靠课程核心、教师确认共识、边界和负向记忆；
//   - 不包含课本全文、附件清单、Token信息或当前待确认正文；
//   - 内部“教案环节确认进度”条目不作为普通教学方向直接注入；
//   - 可靠确认范围通过独立防重复确认区块转述给主对话。
//
// 2. 教师端安全展示：
//   - 展开后展示全部有效教学共识，不再统一截断为6条；
//   - 最近确认内容优先排列；
//   - 当前撰写进度单独展示；
//   - 待确认、等待教师确认、尚待确认等表述统一识别；
//   - 内部进度条目不会重复出现在“已经确定的”区域。

import (
	"sort"
	"strconv"
	"strings"

	"tedna/internal/models"
)

// buildLessonPlanContextCapsuleContextText 构建跨阶段稳定短版上下文。
func buildLessonPlanContextCapsuleContextText(
	document *models.LessonPlanContextCapsuleDocument,
) string {
	if !lessonPlanContextCapsuleHasUsableCore(
		document,
	) {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"\n\n【本课共同认识·跨阶段持续有效】\n",
	)

	builder.WriteString(
		"以下内容是教师与AI在自然对话中已经形成的当前有效共识。阶段切换只改变工作重点，不得清空或重新询问已经确认的内容。\n",
	)

	writeLessonPlanCapsuleRuntimeItems(
		&builder,
		"课程与课本核心",
		document.CourseCore,
	)

	writeLessonPlanCapsuleRuntimeItems(
		&builder,
		"已经确定的教学方向",
		document.TeachingConsensus,
	)

	// 教案环节确认不是普通教学方向，而是后续对话必须遵守的状态。
	// 独立模块只读取可靠的结构化确认条目，并生成防重复确认约束。
	writeLessonPlanCapsuleRuntimeConfirmedSections(
		&builder,
		document,
	)

	writeLessonPlanCapsuleRuntimeItems(
		&builder,
		"必须遵守的边界",
		document.Constraints,
	)

	if len(document.SupersededItems) > 0 {
		builder.WriteString(
			"\n【已经纠正或放弃的旧内容·禁止复发】\n",
		)

		for _, item := range document.SupersededItems {
			builder.WriteString(
				"- " + item.Content + "\n",
			)
		}

		builder.WriteString(
			"上述旧内容不得重新包装成问题，不得再次要求教师确认。只有教师明确要求恢复时才可重新讨论。\n",
		)
	}

	builder.WriteString(
		"\n交互要求：直接承接已有共识推进当前任务，不要用“现在进入某阶段”“接下来我们进入”等仪式性过渡语；不要重复介绍自己、重复复述流程或让教师再次确认已经确认的内容。\n",
	)

	builder.WriteString(
		"【本课共同认识·结束】\n",
	)

	return strings.TrimSpace(
		builder.String(),
	)
}

// writeLessonPlanCapsuleRuntimeItems 写入可作为正式约束的普通条目。
func writeLessonPlanCapsuleRuntimeItems(
	builder *strings.Builder,
	title string,
	items []models.LessonPlanContextCapsuleItem,
) {
	validItems := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(items),
	)

	for _, item := range items {
		if item.Authority ==
			models.LessonPlanContextCapsuleAuthorityAIInferred {
			continue
		}

		// 内部教案确认条目不进入普通教学方向。
		// 它由独立的防重复确认区块进行确定性转述。
		if lessonPlanCapsuleConfirmationProgressItem(
			item,
		) {
			continue
		}

		validItems = append(
			validItems,
			item,
		)
	}

	if len(validItems) == 0 {
		return
	}

	builder.WriteString(
		"\n【" + title + "】\n",
	)

	for _, item := range validItems {
		builder.WriteString(
			"- " + item.Content + "\n",
		)
	}
}

// buildLessonPlanContextCapsuleDisplayView 构建教师端安全视图。
func buildLessonPlanContextCapsuleDisplayView(
	document *models.LessonPlanContextCapsuleDocument,
	recentChange string,
) models.LessonPlanContextCapsuleDisplayView {
	sections := make(
		[]models.LessonPlanContextCapsuleDisplaySection,
		0,
		5,
	)

	appendSection := func(
		key string,
		title string,
		items []models.LessonPlanContextCapsuleItem,
		emphasis string,
		limit int,
		newestFirst bool,
		includeAIInferred bool,
		excludeProgressItems bool,
	) {
		values :=
			lessonPlanCapsuleDisplayValues(
				items,
				limit,
				newestFirst,
				includeAIInferred,
				excludeProgressItems,
			)

		if len(values) == 0 {
			return
		}

		sections = append(
			sections,
			models.LessonPlanContextCapsuleDisplaySection{
				Key:      key,
				Title:    title,
				Items:    values,
				Emphasis: emphasis,
			},
		)
	}

	appendSection(
		"course_core",
		"课程核心",
		document.CourseCore,
		"stable",
		8,
		false,
		false,
		false,
	)

	// 教学共识归一化上限为20条。
	// 展开态展示全部有效条目，不再在第6条处截断。
	appendSection(
		"teaching_consensus",
		"已经确定的",
		document.TeachingConsensus,
		"active",
		20,
		true,
		false,
		true,
	)

	appendSection(
		"constraints",
		"不能偏离的",
		document.Constraints,
		"guard",
		12,
		true,
		false,
		false,
	)

	currentTask :=
		strings.TrimSpace(
			document.StageFocus.CurrentTask,
		)

	if currentTask != "" {
		sections = append(
			sections,
			models.LessonPlanContextCapsuleDisplaySection{
				Key:   "stage_focus",
				Title: "当前进度",
				Items: []string{
					currentTask,
				},
				Emphasis: "soft",
			},
		)
	}

	appendSection(
		"open_questions",
		"仍在推敲的",
		document.OpenQuestions,
		"soft",
		12,
		true,
		true,
		false,
	)

	stateLabel := "理解同步"

	switch {
	case lessonPlanCapsuleProgressIsPending(
		currentTask,
	):
		stateLabel =
			"教案撰写中，部分内容待确认"

	case document.StageFocus.StageCode == "write" ||
		document.StageFocus.StageCode == "revise":
		stateLabel =
			"教案撰写持续同步"

	case len(document.OpenQuestions) > 0:
		stateLabel =
			"主线清晰，局部仍在收敛"
	}

	return models.LessonPlanContextCapsuleDisplayView{
		StateLabel: stateLabel,
		Headline:   "我们已经确定的",
		Summary: normalizeLessonPlanCapsuleText(
			document.Summary,
			600,
		),
		RecentChange: normalizeLessonPlanCapsuleText(
			recentChange,
			300,
		),
		Sections: sections,
	}
}

// lessonPlanCapsuleDisplayValues 清理、排序并限制展示条目。
func lessonPlanCapsuleDisplayValues(
	items []models.LessonPlanContextCapsuleItem,
	limit int,
	newestFirst bool,
	includeAIInferred bool,
	excludeProgressItems bool,
) []string {
	if limit <= 0 ||
		len(items) == 0 {
		return nil
	}

	type indexedItem struct {
		Item     models.LessonPlanContextCapsuleItem
		Index    int
		Sequence int64
	}

	ordered := make(
		[]indexedItem,
		0,
		len(items),
	)

	for index, item := range items {
		ordered = append(
			ordered,
			indexedItem{
				Item:  item,
				Index: index,
				Sequence: lessonPlanCapsuleTurnSequence(
					item.UpdatedByTurnID,
				),
			},
		)
	}

	if newestFirst {
		sort.SliceStable(
			ordered,
			func(left int, right int) bool {
				if ordered[left].Sequence ==
					ordered[right].Sequence {
					return ordered[left].Index >
						ordered[right].Index
				}

				return ordered[left].Sequence >
					ordered[right].Sequence
			},
		)
	}

	values := make(
		[]string,
		0,
		len(ordered),
	)

	seen := make(
		map[string]struct{},
	)

	for _, entry := range ordered {
		item := entry.Item

		if !includeAIInferred &&
			item.Authority ==
				models.LessonPlanContextCapsuleAuthorityAIInferred {
			continue
		}

		if excludeProgressItems &&
			lessonPlanCapsuleConfirmationProgressItem(
				item,
			) {
			continue
		}

		content :=
			strings.TrimSpace(
				item.Content,
			)

		if content == "" {
			continue
		}

		if _, exists :=
			seen[content]; exists {
			continue
		}

		seen[content] =
			struct{}{}

		values = append(
			values,
			content,
		)

		if len(values) >= limit {
			break
		}
	}

	return values
}

// lessonPlanCapsuleTurnSequence 从turnID末尾提取排序序号。
func lessonPlanCapsuleTurnSequence(
	turnID string,
) int64 {
	turnID =
		strings.TrimSpace(
			turnID,
		)

	if turnID == "" {
		return 0
	}

	separator :=
		strings.LastIndex(
			turnID,
			"_",
		)

	if separator < 0 ||
		separator+1 >= len(turnID) {
		return 0
	}

	value, err :=
		strconv.ParseInt(
			turnID[separator+1:],
			10,
			64,
		)

	if err != nil {
		return 0
	}

	return value
}
