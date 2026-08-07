package services

// lesson_plan_context_capsule_progress_summary.go — 确定性摘要与进度指纹
//
// 模型生成的summary可能出现长期陈旧的“下一步将……”。
// 本文件只依据结构化共识、已确认环节和待确认环节生成摘要。
// 内部教案确认进度条目不计入“有效教学共识”数量。
// 进入review阶段后，已确认和待确认状态仍持续保留。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tedna/internal/models"
)

// buildLessonPlanCapsuleWriteProgressTask 构造撰写阶段进度。
func buildLessonPlanCapsuleWriteProgressTask(
	confirmed map[int]struct{},
	pending []int,
	newlyConfirmed []int,
	stageCode string,
) string {
	if len(pending) > 0 {
		if len(newlyConfirmed) > 0 {
			return fmt.Sprintf(
				"教师已确认%s的教案内容；本轮已生成%s的详细教案内容，等待教师确认或修改。",
				formatLessonPlanCapsuleSections(
					newlyConfirmed,
				),
				formatLessonPlanCapsuleSections(
					pending,
				),
			)
		}

		return fmt.Sprintf(
			"本轮已生成%s的详细教案内容，等待教师确认或修改。",
			formatLessonPlanCapsuleSections(
				pending,
			),
		)
	}

	if len(newlyConfirmed) > 0 {
		return fmt.Sprintf(
			"教师已确认%s的教案内容，正在继续完善完整教案。",
			formatLessonPlanCapsuleSections(
				newlyConfirmed,
			),
		)
	}

	confirmedValues :=
		lessonPlanCapsuleSectionMapValues(
			confirmed,
		)

	if stageCode == "revise" {
		return "正在依据教师反馈修改完整教案，稳定教学共识继续有效。"
	}

	if len(confirmedValues) > 0 {
		return fmt.Sprintf(
			"正在依据已确认的%s继续撰写和完善完整教案。",
			formatLessonPlanCapsuleSections(
				confirmedValues,
			),
		)
	}

	return "正在依据已确认教学共识撰写完整教案。"
}

// buildLessonPlanCapsuleReviewProgressTask 构造教案核对阶段进度。
//
// 进入review只改变当前工作注意力，不抹掉write阶段已经生成、
// 已确认或尚待确认的环节状态。
func buildLessonPlanCapsuleReviewProgressTask(
	confirmed map[int]struct{},
	pending []int,
) string {
	confirmedValues :=
		lessonPlanCapsuleSectionMapValues(
			confirmed,
		)

	switch {
	case len(confirmedValues) > 0 &&
		len(pending) > 0:
		return fmt.Sprintf(
			"完整教案已生成，%s已确认；%s等待教师确认或修改，同时正在进行教案质量核对。",
			formatLessonPlanCapsuleSections(
				confirmedValues,
			),
			formatLessonPlanCapsuleSections(
				pending,
			),
		)

	case len(pending) > 0:
		return fmt.Sprintf(
			"完整教案已生成，%s等待教师确认或修改，同时正在进行教案质量核对。",
			formatLessonPlanCapsuleSections(
				pending,
			),
		)

	case len(confirmedValues) > 0:
		return fmt.Sprintf(
			"教案%s已确认，正在核对课程依据、结构完整性和课堂可执行性。",
			formatLessonPlanCapsuleSections(
				confirmedValues,
			),
		)

	default:
		return "完整教案已生成，正在核对课程依据、结构完整性和课堂可执行性。"
	}
}

// lessonPlanCapsuleStableStageTask 生成其他阶段的稳定焦点。
func lessonPlanCapsuleStableStageTask(
	stageCode string,
) string {
	switch strings.TrimSpace(
		stageCode,
	) {
	case "analyze":
		return "正在围绕本课学情、目标、重难点与课程依据收敛教学分析。"

	case "design":
		return "正在依据已确认共识完善教学活动链、任务支架与评价设计。"

	case "review":
		return "正在核对教案质量、课程依据和可执行性。"

	default:
		return ""
	}
}

// buildLessonPlanContextCapsuleDeterministicSummary 构造稳定摘要。
func buildLessonPlanContextCapsuleDeterministicSummary(
	document *models.LessonPlanContextCapsuleDocument,
	confirmed map[int]struct{},
	pending []int,
) string {
	if document == nil {
		return ""
	}

	consensusCount :=
		lessonPlanCapsuleReliableConsensusCount(
			document.TeachingConsensus,
		)

	stageCode :=
		strings.TrimSpace(
			document.StageFocus.StageCode,
		)

	if stageCode == "write" ||
		stageCode == "revise" ||
		stageCode == "review" {
		confirmedValues :=
			lessonPlanCapsuleSectionMapValues(
				confirmed,
			)

		progressLabel :=
			"教案撰写进度"

		if stageCode == "review" {
			progressLabel =
				"教案核对进度"
		}

		switch {
		case len(confirmedValues) > 0 &&
			len(pending) > 0:
			return fmt.Sprintf(
				"本课已形成%d项有效教学共识。%s：%s已确认；%s已生成待确认。",
				consensusCount,
				progressLabel,
				formatLessonPlanCapsuleSections(
					confirmedValues,
				),
				formatLessonPlanCapsuleSections(
					pending,
				),
			)

		case len(confirmedValues) > 0:
			return fmt.Sprintf(
				"本课已形成%d项有效教学共识。%s：%s已确认。",
				consensusCount,
				progressLabel,
				formatLessonPlanCapsuleSections(
					confirmedValues,
				),
			)

		case len(pending) > 0:
			return fmt.Sprintf(
				"本课已形成%d项有效教学共识。%s：%s已生成待确认。",
				consensusCount,
				progressLabel,
				formatLessonPlanCapsuleSections(
					pending,
				),
			)
		}

		if stageCode == "review" {
			return fmt.Sprintf(
				"本课已形成%d项有效教学共识，完整教案正在进行质量核对。",
				consensusCount,
			)
		}

		return fmt.Sprintf(
			"本课已形成%d项有效教学共识，正在据此撰写和完善完整教案。",
			consensusCount,
		)
	}

	return fmt.Sprintf(
		"本课已形成%d项有效教学共识，展开后可查看全部内容。",
		consensusCount,
	)
}

// lessonPlanCapsuleReliableConsensusCount 统计可靠且不重复的教学共识。
func lessonPlanCapsuleReliableConsensusCount(
	items []models.LessonPlanContextCapsuleItem,
) int {
	seen :=
		make(map[string]struct{})

	for _, item := range items {
		if item.Authority ==
			models.LessonPlanContextCapsuleAuthorityAIInferred {
			continue
		}

		if lessonPlanCapsuleConfirmationProgressItem(
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

		key :=
			strings.TrimSpace(
				item.Key,
			)

		if key == "" {
			key = content
		}

		seen[key] =
			struct{}{}
	}

	return len(seen)
}

// lessonPlanCapsuleSectionMapValues 转换为升序切片。
func lessonPlanCapsuleSectionMapValues(
	values map[int]struct{},
) []int {
	output := make(
		[]int,
		0,
		len(values),
	)

	for value := range values {
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

// formatLessonPlanCapsuleSections 格式化环节编号。
func formatLessonPlanCapsuleSections(
	values []int,
) string {
	values =
		lessonPlanCapsuleUniqueSortedSections(
			values,
		)

	labels := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		labels = append(
			labels,
			"环节"+
				lessonPlanCapsuleChineseNumber(
					value,
				),
		)
	}

	return strings.Join(
		labels,
		"、",
	)
}

// lessonPlanCapsuleChineseNumber 返回常用中文编号。
func lessonPlanCapsuleChineseNumber(
	value int,
) string {
	mapping := map[int]string{
		1:  "一",
		2:  "二",
		3:  "三",
		4:  "四",
		5:  "五",
		6:  "六",
		7:  "七",
		8:  "八",
		9:  "九",
		10: "十",
	}

	if label, exists :=
		mapping[value]; exists {
		return label
	}

	return strconv.Itoa(
		value,
	)
}

// lessonPlanContextCapsuleProgressRecentChange 生成近期变化说明。
func lessonPlanContextCapsuleProgressRecentChange(
	document *models.LessonPlanContextCapsuleDocument,
) string {
	if document == nil {
		return "本课共识已同步更新。"
	}

	task :=
		strings.TrimSpace(
			document.StageFocus.CurrentTask,
		)

	if task != "" {
		return task
	}

	return "本课共识已同步更新。"
}

// hashLessonPlanContextCapsuleVersionWithProgress 叠加进度指纹。
func hashLessonPlanContextCapsuleVersionWithProgress(
	stableHash string,
	document *models.LessonPlanContextCapsuleDocument,
) string {
	payload := struct {
		StableHash  string `json:"stable_hash"`
		StageCode   string `json:"stage_code"`
		CurrentTask string `json:"current_task"`
		Summary     string `json:"summary"`
	}{
		StableHash: strings.TrimSpace(
			stableHash,
		),
	}

	if document != nil {
		payload.StageCode =
			strings.TrimSpace(
				document.StageFocus.StageCode,
			)

		payload.CurrentTask =
			strings.TrimSpace(
				document.StageFocus.CurrentTask,
			)

		payload.Summary =
			strings.TrimSpace(
				document.Summary,
			)
	}

	encoded, err :=
		json.Marshal(
			payload,
		)

	if err != nil {
		encoded = []byte(
			strings.Join(
				[]string{
					payload.StableHash,
					payload.StageCode,
					payload.CurrentTask,
					payload.Summary,
				},
				"\x1f",
			),
		)
	}

	sum :=
		sha256.Sum256(
			encoded,
		)

	return hex.EncodeToString(
		sum[:],
	)
}
