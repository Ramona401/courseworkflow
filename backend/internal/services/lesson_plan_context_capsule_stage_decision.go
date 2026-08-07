package services

// lesson_plan_context_capsule_stage_decision.go
//
// 教师主动把完整教案从write推进到review时，属于结构化确认动作：
//   - 当前正式教案正文已经形成；
//   - 教师主动结束撰写阶段并提交AI评审；
//   - 正文中实际存在的详细教学环节可并入已确认范围。
//
// 安全规则：
//   - 只处理write → review，不影响普通阶段切换；
//   - 只识别达到详细教案粒度的环节，简短框架不算确认；
//   - 不调用模型，不伪造教师聊天消息；
//   - 不改变课程核心、稳定教学共识、约束和负向记忆；
//   - 保存失败只记录日志，不撤销已经成功的阶段推进。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// reconcileLessonPlanContextCapsuleWriteReviewDecision
// 将write阶段正式正文中的详细环节合并到确认状态。
//
// 返回值依次为：
//   - 正文中识别到的详细环节；
//   - 本次新增确认环节；
//   - 合并后的全部确认环节；
//   - 胶囊是否发生真实变化。
func reconcileLessonPlanContextCapsuleWriteReviewDecision(
	document *models.LessonPlanContextCapsuleDocument,
	current *models.LessonPlanContextCapsuleDocument,
	content string,
	turnID string,
) (
	[]int,
	[]int,
	[]int,
	bool,
) {
	if document == nil {
		return nil, nil, nil, false
	}

	generated :=
		lessonPlanCapsuleDetailedGeneratedSections(
			content,
		)

	if len(generated) == 0 {
		return nil, nil, nil, false
	}

	if current == nil {
		current = document
	}

	oldSummary :=
		strings.TrimSpace(
			document.Summary,
		)

	oldStageCode :=
		strings.TrimSpace(
			document.StageFocus.StageCode,
		)

	oldCurrentTask :=
		strings.TrimSpace(
			document.StageFocus.CurrentTask,
		)

	confirmedSet :=
		make(map[int]struct{})

	mergeLessonPlanCapsuleSectionSet(
		confirmedSet,
		lessonPlanCapsuleConfirmedSectionsFromDocument(
			current,
			true,
		),
	)

	newlyConfirmed :=
		make([]int, 0)

	for _, sectionNumber := range generated {
		if _, exists :=
			confirmedSet[sectionNumber]; exists {
			continue
		}

		newlyConfirmed = append(
			newlyConfirmed,
			sectionNumber,
		)
	}

	newlyConfirmed =
		lessonPlanCapsuleUniqueSortedSections(
			newlyConfirmed,
		)

	mergeLessonPlanCapsuleSectionSet(
		confirmedSet,
		generated,
	)

	confirmed :=
		lessonPlanCapsuleSectionMapValues(
			confirmedSet,
		)

	itemChanged :=
		upsertLessonPlanCapsuleConfirmedSections(
			document,
			current,
			confirmedSet,
			newlyConfirmed,
			turnID,
		)

	pending :=
		lessonPlanCapsulePendingSectionsFromDocument(
			current,
		)

	pending =
		lessonPlanCapsuleRemoveConfirmedSections(
			pending,
			confirmedSet,
		)

	document.StageFocus.StageCode =
		"review"

	document.StageFocus.CurrentTask =
		normalizeLessonPlanCapsuleText(
			buildLessonPlanCapsuleReviewProgressTask(
				confirmedSet,
				pending,
			),
			500,
		)

	document.Summary =
		normalizeLessonPlanCapsuleText(
			buildLessonPlanContextCapsuleDeterministicSummary(
				document,
				confirmedSet,
				pending,
			),
			600,
		)

	if document.SchemaVersion < 1 {
		document.SchemaVersion =
			models.LessonPlanContextCapsuleSchemaVersion
	}

	changed :=
		itemChanged ||
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

	return generated,
		newlyConfirmed,
		confirmed,
		changed
}

// applyLessonPlanContextCapsuleStageDecision 保存write → review结构化确认。
func applyLessonPlanContextCapsuleStageDecision(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	targetStageCode string,
) (
	*models.LessonPlanContextCapsule,
	bool,
	error,
) {
	if lessonPlan == nil ||
		strings.TrimSpace(
			lessonPlan.CurrentStage,
		) != "write" ||
		strings.TrimSpace(
			targetStageCode,
		) != "review" {
		return nil, false, nil
	}

	current, err :=
		repository.GetLessonPlanContextCapsule(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return nil, false, err
	}

	if current == nil {
		return nil, false, nil
	}

	currentDocument :=
		parseLessonPlanCapsuleCurrentDocument(
			current.CapsuleJSON,
		)

	if currentDocument == nil {
		return nil, false, errors.New(
			"当前active胶囊正文无法解析",
		)
	}

	document :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		[]byte(current.CapsuleJSON),
		document,
	); err != nil {
		return nil, false, fmt.Errorf(
			"复制当前active胶囊失败: %w",
			err,
		)
	}

	decisionTurnID :=
		fmt.Sprintf(
			"stage_write_confirm_%d",
			time.Now().UnixMilli(),
		)

	generated,
		newlyConfirmed,
		confirmed,
		changed :=
		reconcileLessonPlanContextCapsuleWriteReviewDecision(
			document,
			currentDocument,
			lessonPlan.ContentMarkdown,
			decisionTurnID,
		)

	if len(generated) == 0 ||
		!changed {
		return current, false, nil
	}

	capsuleJSON, err :=
		json.Marshal(document)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化阶段确认胶囊失败: %w",
			err,
		)
	}

	contextText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	if strings.TrimSpace(
		contextText,
	) == "" {
		return nil, false, errors.New(
			"阶段确认后无法构建胶囊运行时上下文",
		)
	}

	updateReason :=
		fmt.Sprintf(
			"教师确认完整教案并进入评审，%s已确认。",
			formatLessonPlanCapsuleSections(
				confirmed,
			),
		)

	if len(newlyConfirmed) > 0 {
		updateReason =
			fmt.Sprintf(
				"教师确认完整教案并进入评审，本次确认%s。",
				formatLessonPlanCapsuleSections(
					newlyConfirmed,
				),
			)
	}

	displayView :=
		buildLessonPlanContextCapsuleDisplayView(
			document,
			updateReason,
		)

	displayJSON, err :=
		json.Marshal(
			displayView,
		)
	if err != nil {
		return nil, false, fmt.Errorf(
			"序列化阶段确认教师端视图失败: %w",
			err,
		)
	}

	stableHash, err :=
		hashLessonPlanContextCapsuleVersion(
			document,
		)
	if err != nil {
		return nil, false, err
	}

	sourceHash :=
		hashLessonPlanContextCapsuleVersionWithProgress(
			stableHash,
			document,
		)

	sourceManifest :=
		strings.TrimSpace(
			current.SourceManifest,
		)

	if sourceManifest == "" ||
		sourceManifest == "{}" {
		return nil, false, errors.New(
			"当前胶囊来源清单为空，不能保存阶段确认版本",
		)
	}

	saved, savedChanged, err :=
		repository.UpsertActiveLessonPlanContextCapsule(
			ctx,
			&repository.UpsertLessonPlanContextCapsuleInput{
				LessonPlanID:     lessonPlan.ID,
				SchemaVersion:    models.LessonPlanContextCapsuleSchemaVersion,
				CurrentStageCode: "review",
				CapsuleJSON:      string(capsuleJSON),
				DisplayJSON:      string(displayJSON),
				ContextText:      contextText,
				SourceManifest:   sourceManifest,
				SourceHash:       sourceHash,
				LastTurnID:       decisionTurnID,
				UpdateReason:     updateReason,
				// 本次没有新增课程事实，只更新确认与阶段进度。
				Evidence: nil,
			},
		)
	if err != nil {
		return nil, false, err
	}

	if saved != nil &&
		savedChanged {
		broadcastLessonPlanContextCapsuleUpdate(
			lessonPlan.ID,
			decisionTurnID,
			saved,
		)
	}

	return saved,
		savedChanged,
		nil
}
