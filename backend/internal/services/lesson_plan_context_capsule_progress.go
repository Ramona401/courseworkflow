package services

// lesson_plan_context_capsule_progress.go — 胶囊撰写进度协调入口
//
// 本文件负责合并数据库当前状态、教师确认动作和当前AI生成范围。
// 当前AI刚生成的正文只能进入“已生成待确认”，不能自动升级为教师确认。
//
// 进度继承规则：
//   - 未点名的“确认、按这个来”等表达，只确认上一版待确认范围；
//   - “确认环节三和环节四”可直接确认教师明确点名的范围；
//   - “确认，请开始写环节三和环节四”不会把三、四误判为已确认；
//   - write、revise、review阶段均持续携带已确认和待确认状态。

import (
	"encoding/json"
	"reflect"
	"strings"

	"tedna/internal/models"
)

const lessonPlanCapsuleConfirmedSectionsKey = "consensus.lesson_plan_confirmed_sections"

// lessonPlanContextCapsuleCurrentJSON 取得当前active胶囊正文。
func lessonPlanContextCapsuleCurrentJSON(
	current *models.LessonPlanContextCapsule,
) string {
	if current == nil {
		return ""
	}

	return strings.TrimSpace(
		current.CapsuleJSON,
	)
}

// lessonPlanContextCapsuleAssistantProgressText 返回进度扫描使用的AI正文。
func lessonPlanContextCapsuleAssistantProgressText(
	message *models.ConversationMessage,
) string {
	if message == nil {
		return ""
	}

	value :=
		strings.TrimSpace(
			message.Content,
		)

	runes := []rune(value)

	if len(runes) > 30000 {
		return string(
			runes[:30000],
		)
	}

	return value
}

// reconcileLessonPlanContextCapsuleProgress 生成确定性摘要和撰写进度。
func reconcileLessonPlanContextCapsuleProgress(
	document *models.LessonPlanContextCapsuleDocument,
	currentJSON string,
	teacherMessage string,
	assistantMessage string,
	stageCode string,
	turnID string,
) bool {
	if document == nil {
		return false
	}

	current :=
		parseLessonPlanCapsuleCurrentDocument(
			currentJSON,
		)

	oldSummary := ""
	oldStageCode := ""
	oldCurrentTask := ""

	if current != nil {
		oldSummary =
			strings.TrimSpace(
				current.Summary,
			)

		oldStageCode =
			strings.TrimSpace(
				current.StageFocus.StageCode,
			)

		oldCurrentTask =
			strings.TrimSpace(
				current.StageFocus.CurrentTask,
			)
	}

	confirmed :=
		make(map[int]struct{})

	mergeLessonPlanCapsuleSectionSet(
		confirmed,
		lessonPlanCapsuleConfirmedSectionsFromDocument(
			current,
			true,
		),
	)

	previousPending :=
		lessonPlanCapsulePendingSectionsFromDocument(
			current,
		)

	newlyConfirmed := []int(nil)

	// 教师明确点名确认时直接使用点名范围，
	// 不要求数据库必须事先存在previousPending。
	explicitlyConfirmed :=
		lessonPlanCapsuleExplicitlyConfirmedSections(
			teacherMessage,
		)

	switch {
	case len(explicitlyConfirmed) > 0:
		newlyConfirmed = append(
			newlyConfirmed,
			explicitlyConfirmed...,
		)

		mergeLessonPlanCapsuleSectionSet(
			confirmed,
			explicitlyConfirmed,
		)

	case teacherExplicitlyConfirmsLessonPlanProgress(
		teacherMessage,
	) &&
		len(previousPending) > 0:
		// 未点名确认只确认上一版待确认范围。
		newlyConfirmed = append(
			newlyConfirmed,
			previousPending...,
		)

		mergeLessonPlanCapsuleSectionSet(
			confirmed,
			previousPending,
		)

		previousPending = nil
	}

	newlyConfirmed =
		lessonPlanCapsuleUniqueSortedSections(
			newlyConfirmed,
		)

	generatedSections :=
		lessonPlanCapsuleDetailedGeneratedSections(
			assistantMessage,
		)

	pending := append(
		[]int(nil),
		previousPending...,
	)

	for _, sectionNumber := range generatedSections {
		if _, alreadyConfirmed :=
			confirmed[sectionNumber]; alreadyConfirmed {
			continue
		}

		pending = append(
			pending,
			sectionNumber,
		)
	}

	pending =
		lessonPlanCapsuleUniqueSortedSections(
			pending,
		)

	pending =
		lessonPlanCapsuleRemoveConfirmedSections(
			pending,
			confirmed,
		)

	documentChanged :=
		sanitizeLessonPlanCapsuleCurrentTurnConfirmations(
			document,
			confirmed,
			generatedSections,
			turnID,
		)

	stageCode =
		strings.TrimSpace(
			stageCode,
		)

	// 已确认范围是跨阶段稳定记忆。
	// 进入review后仍需继续携带并允许教师自然确认。
	if len(confirmed) > 0 {
		if upsertLessonPlanCapsuleConfirmedSections(
			document,
			current,
			confirmed,
			newlyConfirmed,
			turnID,
		) {
			documentChanged = true
		}
	}

	document.StageFocus.StageCode =
		stageCode

	switch stageCode {
	case "write", "revise":
		document.StageFocus.CurrentTask =
			buildLessonPlanCapsuleWriteProgressTask(
				confirmed,
				pending,
				newlyConfirmed,
				stageCode,
			)

	case "review":
		document.StageFocus.CurrentTask =
			buildLessonPlanCapsuleReviewProgressTask(
				confirmed,
				pending,
			)

	default:
		document.StageFocus.CurrentTask =
			lessonPlanCapsuleStableStageTask(
				stageCode,
			)
	}

	document.StageFocus.CurrentTask =
		normalizeLessonPlanCapsuleText(
			document.StageFocus.CurrentTask,
			500,
		)

	document.Summary =
		buildLessonPlanContextCapsuleDeterministicSummary(
			document,
			confirmed,
			pending,
		)

	document.Summary =
		normalizeLessonPlanCapsuleText(
			document.Summary,
			600,
		)

	return documentChanged ||
		oldSummary !=
			strings.TrimSpace(
				document.Summary,
			) ||
		oldStageCode !=
			strings.TrimSpace(
				document.StageFocus.StageCode,
			) ||
		oldCurrentTask !=
			strings.TrimSpace(
				document.StageFocus.CurrentTask,
			)
}

// parseLessonPlanCapsuleCurrentDocument 解析数据库当前胶囊。
func parseLessonPlanCapsuleCurrentDocument(
	currentJSON string,
) *models.LessonPlanContextCapsuleDocument {
	currentJSON =
		strings.TrimSpace(
			currentJSON,
		)

	if currentJSON == "" ||
		currentJSON == "{}" {
		return nil
	}

	document :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		[]byte(currentJSON),
		document,
	); err != nil {
		return nil
	}

	return document
}

// teacherExplicitlyConfirmsLessonPlanProgress 判断教师是否确认上一版正文。
//
// 否定、疑问和“确认后再……”不视为确认。
func teacherExplicitlyConfirmsLessonPlanProgress(
	message string,
) bool {
	normalized :=
		normalizeLessonPlanTurnText(
			message,
		)

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
			return false
		}
	}

	for _, signal := range []string{
		"确认",
		"可以确认",
		"我可以确认",
		"按这个来",
		"按此执行",
		"就这样",
		"没问题",
		"同意",
		"这个可以",
		"以上可以",
	} {
		if strings.Contains(
			normalized,
			signal,
		) {
			return true
		}
	}

	return false
}

// sanitizeLessonPlanCapsuleCurrentTurnConfirmations 清理当前轮错误确认。
func sanitizeLessonPlanCapsuleCurrentTurnConfirmations(
	document *models.LessonPlanContextCapsuleDocument,
	confirmed map[int]struct{},
	generatedSections []int,
	turnID string,
) bool {
	if document == nil ||
		len(generatedSections) == 0 {
		return false
	}

	generated :=
		make(map[int]struct{})

	for _, sectionNumber := range generatedSections {
		generated[sectionNumber] =
			struct{}{}
	}

	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(document.TeachingConsensus),
	)

	changed := false

	for _, item := range document.TeachingConsensus {
		if strings.TrimSpace(
			item.UpdatedByTurnID,
		) != strings.TrimSpace(
			turnID,
		) ||
			!lessonPlanCapsuleConfirmationProgressItem(
				item,
			) {
			output = append(
				output,
				item,
			)
			continue
		}

		sections :=
			lessonPlanCapsuleSectionNumbersFromText(
				strings.Join(
					[]string{
						item.Key,
						item.Title,
						item.Content,
					},
					" ",
				),
			)

		remove := false

		for _, sectionNumber := range sections {
			if _, generatedThisTurn :=
				generated[sectionNumber]; !generatedThisTurn {
				continue
			}

			if _, wasAlreadyConfirmed :=
				confirmed[sectionNumber]; !wasAlreadyConfirmed {
				remove = true
				break
			}
		}

		if remove {
			changed = true
			continue
		}

		output = append(
			output,
			item,
		)
	}

	document.TeachingConsensus =
		output

	return changed
}

// upsertLessonPlanCapsuleConfirmedSections 维护累积确认条目。
func upsertLessonPlanCapsuleConfirmedSections(
	document *models.LessonPlanContextCapsuleDocument,
	current *models.LessonPlanContextCapsuleDocument,
	confirmed map[int]struct{},
	newlyConfirmed []int,
	turnID string,
) bool {
	if document == nil ||
		len(confirmed) == 0 {
		return false
	}

	confirmedValues :=
		lessonPlanCapsuleSectionMapValues(
			confirmed,
		)

	desired :=
		models.LessonPlanContextCapsuleItem{
			Key:   lessonPlanCapsuleConfirmedSectionsKey,
			Title: "已确认教案环节",
			Content: "教师已确认" +
				formatLessonPlanCapsuleSections(
					confirmedValues,
				) +
				"的教案内容。",
			State:      models.LessonPlanContextCapsuleItemStateActive,
			Authority:  models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
			Importance: 5,
			ApplicableStages: []string{
				"write",
				"revise",
				"review",
			},
			DoNotReconfirm: true,
		}

	currentItem, currentExists :=
		findLessonPlanCapsuleItemByKey(
			current,
			lessonPlanCapsuleConfirmedSectionsKey,
		)

	if currentExists {
		desired.SourceKeys = append(
			[]string(nil),
			currentItem.SourceKeys...,
		)

		desired.UpdatedByTurnID =
			currentItem.UpdatedByTurnID
	}

	if len(newlyConfirmed) > 0 ||
		strings.TrimSpace(
			desired.UpdatedByTurnID,
		) == "" {
		desired.UpdatedByTurnID =
			strings.TrimSpace(
				turnID,
			)
	}

	var documentItem *models.LessonPlanContextCapsuleItem

	filtered := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
		len(document.TeachingConsensus)+1,
	)

	for _, item := range document.TeachingConsensus {
		if item.Key ==
			lessonPlanCapsuleConfirmedSectionsKey {
			copyItem := item
			documentItem = &copyItem
			continue
		}

		if lessonPlanCapsuleConfirmationProgressItem(
			item,
		) {
			continue
		}

		filtered = append(
			filtered,
			item,
		)
	}

	filtered = append(
		filtered,
		desired,
	)

	document.TeachingConsensus =
		filtered

	document.StageFocus.CarryForwardKeys =
		appendUniqueLessonPlanCapsuleString(
			document.StageFocus.CarryForwardKeys,
			lessonPlanCapsuleConfirmedSectionsKey,
			30,
		)

	if currentExists {
		return !reflect.DeepEqual(
			currentItem,
			desired,
		)
	}

	if documentItem != nil {
		return !reflect.DeepEqual(
			*documentItem,
			desired,
		)
	}

	return true
}

// findLessonPlanCapsuleItemByKey 按key查找教学共识。
func findLessonPlanCapsuleItemByKey(
	document *models.LessonPlanContextCapsuleDocument,
	key string,
) (
	models.LessonPlanContextCapsuleItem,
	bool,
) {
	if document == nil {
		return models.LessonPlanContextCapsuleItem{},
			false
	}

	for _, item := range document.TeachingConsensus {
		if item.Key == key {
			return item, true
		}
	}

	return models.LessonPlanContextCapsuleItem{},
		false
}
