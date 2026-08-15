package repository

// lesson_plan_legacy_stage_runtime_repo.go — 遗留完整教案阶段运行态原子恢复
//
// 历史上存在两类“正文完整但阶段运行态缺失”的教案：
//   - 早期Fork副本在旧创建链中没有持久化forked_from，也没有阶段快照；
//   - 其它Phase 7B之前形成的可编辑完整教案可能同样只有正文，没有current_stage。
//
// 本文件不根据标题或forked_from猜来源，只根据数据库可验证事实恢复：
//   - 当前调用者仍是作者；
//   - 教案仍处于Chat允许的可编辑状态；
//   - 正文非空；
//   - stage_config/current_stage均尚未初始化；
//   - workshop_stage_outputs仍为空。
//
// 恢复只写stage_config、current_stage和阶段output；正文、version、conversation_log、
// 教育域、挂载字段、审核结果和fork关系全部保持原值。
// 调用方必须提供按“已有完整教案”语义构造的阶段状态：review之前skipped，review in_progress。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected = errors.New(
	"遗留完整教案阶段运行态恢复被安全条件拒绝",
)

func RepairLegacyCompleteLessonPlanStageRuntime(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
	stageConfigJSON string,
	currentStage string,
	stageOutputs []models.WorkshopStageOutput,
) (bool, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	authorID = strings.TrimSpace(authorID)
	currentStage = strings.TrimSpace(currentStage)

	if lessonPlanID == "" ||
		authorID == "" ||
		currentStage == "" {
		return false, fmt.Errorf(
			"%w: 教案ID、作者ID或当前阶段为空",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
		)
	}

	var snapshots []models.StageConfigSnapshot
	if err := json.Unmarshal(
		[]byte(stageConfigJSON),
		&snapshots,
	); err != nil || len(snapshots) == 0 {
		return false, fmt.Errorf(
			"%w: stage_config为空或非法",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
		)
	}

	snapshotByCode :=
		make(
			map[string]models.StageConfigSnapshot,
			len(snapshots),
		)

	for index := range snapshots {
		snapshot := snapshots[index]
		stageCode :=
			strings.TrimSpace(
				snapshot.StageCode,
			)

		if stageCode == "" ||
			snapshot.StageOrder <= 0 {
			return false, fmt.Errorf(
				"%w: stage_config存在非法阶段",
				ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
			)
		}
		if _, exists := snapshotByCode[stageCode]; exists {
			return false, fmt.Errorf(
				"%w: stage_config存在重复阶段%s",
				ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
				stageCode,
			)
		}

		snapshotByCode[stageCode] =
			snapshot
	}

	if _, exists := snapshotByCode[currentStage]; !exists {
		return false, fmt.Errorf(
			"%w: 当前阶段%s不在stage_config中",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
			currentStage,
		)
	}

	if len(stageOutputs) == 0 {
		return false, fmt.Errorf(
			"%w: 阶段产出列表为空",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
		)
	}

	currentStageCount := 0
	outputSeen :=
		make(
			map[string]struct{},
			len(stageOutputs),
		)

	for index := range stageOutputs {
		output := stageOutputs[index]
		stageCode :=
			strings.TrimSpace(
				output.StageCode,
			)

		snapshot, exists :=
			snapshotByCode[stageCode]
		if !exists ||
			stageCode == "" ||
			output.StageOrder != snapshot.StageOrder {
			return false, fmt.Errorf(
				"%w: 阶段产出%s与stage_config不一致",
				ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
				stageCode,
			)
		}

		if _, duplicated :=
			outputSeen[stageCode]; duplicated {
			return false, fmt.Errorf(
				"%w: 阶段产出%s重复",
				ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
				stageCode,
			)
		}
		outputSeen[stageCode] = struct{}{}

		switch output.Status {
		case models.StageOutputSkipped:
			if stageCode == currentStage {
				return false, fmt.Errorf(
					"%w: 当前阶段不能标记为skipped",
					ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
				)
			}
		case models.StageOutputInProgress:
			if stageCode != currentStage {
				return false, fmt.Errorf(
					"%w: 非当前阶段%s不能标记为in_progress",
					ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
					stageCode,
				)
			}
			currentStageCount++
		default:
			return false, fmt.Errorf(
				"%w: 阶段%s状态非法",
				ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
				stageCode,
			)
		}
	}

	if currentStageCount != 1 {
		return false, fmt.Errorf(
			"%w: 当前阶段in_progress记录数量为%d",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
			currentStageCount,
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"开始遗留完整教案阶段恢复事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		storedAuthorID     string
		status             string
		contentMarkdown    string
		storedCurrentStage string
		storedStageConfig  string
	)

	err = tx.QueryRow(ctx, `
		SELECT
			author_id::text,
			status,
			COALESCE(content_markdown, ''),
			COALESCE(current_stage, ''),
			COALESCE(stage_config::text, '[]')
		FROM lesson_plans
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE
	`, lessonPlanID).Scan(
		&storedAuthorID,
		&status,
		&contentMarkdown,
		&storedCurrentStage,
		&storedStageConfig,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrLessonPlanNotFound
		}
		return false, fmt.Errorf(
			"锁定遗留完整教案失败: %w",
			err,
		)
	}

	if strings.TrimSpace(storedAuthorID) != authorID ||
		strings.TrimSpace(contentMarkdown) == "" ||
		!isLegacyCompleteLessonPlanEditableStatus(
			status,
		) {
		return false, fmt.Errorf(
			"%w: plan_id=%s status=%s",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
			lessonPlanID,
			status,
		)
	}

	storedCurrentStage =
		strings.TrimSpace(
			storedCurrentStage,
		)
	storedStageConfig =
		strings.TrimSpace(
			storedStageConfig,
		)

	currentInitialized :=
		storedCurrentStage != ""
	configInitialized :=
		!isEmptyLegacyLessonPlanStageConfig(
			storedStageConfig,
		)

	if currentInitialized && configInitialized {
		return false, tx.Commit(ctx)
	}
	if currentInitialized != configInitialized {
		return false, fmt.Errorf(
			"%w: 阶段运行态处于部分初始化状态",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
		)
	}

	var existingOutputCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM workshop_stage_outputs
		WHERE lesson_plan_id = $1
	`, lessonPlanID).Scan(
		&existingOutputCount,
	); err != nil {
		return false, fmt.Errorf(
			"检查遗留完整教案阶段产出失败: %w",
			err,
		)
	}
	if existingOutputCount != 0 {
		return false, fmt.Errorf(
			"%w: 阶段字段为空但已经存在%d条阶段产出",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
			existingOutputCount,
		)
	}

	result, err := tx.Exec(ctx, `
		UPDATE lesson_plans
		SET
			stage_config = $1::jsonb,
			current_stage = $2,
			updated_at = NOW()
		WHERE id = $3
		  AND author_id = $4
		  AND deleted_at IS NULL
		  AND COALESCE(current_stage, '') = ''
		  AND (
			stage_config IS NULL
			OR stage_config::jsonb = '[]'::jsonb
			OR stage_config::jsonb = 'null'::jsonb
		  )
	`,
		stageConfigJSON,
		currentStage,
		lessonPlanID,
		authorID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"写入遗留完整教案阶段运行态失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return false, fmt.Errorf(
			"%w: 更新时目标状态发生变化",
			ErrLegacyCompleteLessonPlanStageRuntimeRepairRejected,
		)
	}

	for index := range stageOutputs {
		output := stageOutputs[index]

		structuredOutput :=
			strings.TrimSpace(
				output.StructuredOutput,
			)
		if structuredOutput == "" {
			structuredOutput = "{}"
		}

		conversationSnapshot :=
			strings.TrimSpace(
				output.ConversationSnapshot,
			)
		if conversationSnapshot == "" {
			conversationSnapshot = "[]"
		}

		var completedAt *time.Time
		if output.Status ==
			models.StageOutputSkipped {
			now := time.Now()
			completedAt = &now
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO workshop_stage_outputs (
				lesson_plan_id,
				stage_code,
				stage_order,
				structured_output,
				narrative_output,
				conversation_snapshot,
				model_used,
				tokens_used,
				status,
				completed_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4::jsonb,
				$5,
				$6::jsonb,
				$7,
				$8,
				$9,
				$10
			)
		`,
			lessonPlanID,
			output.StageCode,
			output.StageOrder,
			structuredOutput,
			output.NarrativeOutput,
			conversationSnapshot,
			output.ModelUsed,
			output.TokensUsed,
			output.Status,
			completedAt,
		)
		if err != nil {
			return false, fmt.Errorf(
				"创建遗留完整教案阶段产出失败（%s）: %w",
				output.StageCode,
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf(
			"提交遗留完整教案阶段恢复事务失败: %w",
			err,
		)
	}

	return true, nil
}

func isLegacyCompleteLessonPlanEditableStatus(
	status string,
) bool {
	switch strings.TrimSpace(status) {
	case models.LPStatusDraft,
		models.LPStatusPublishedPersonal,
		models.LPStatusRevision,
		models.LPStatusDeveloping:
		return true
	default:
		return false
	}
}

func isEmptyLegacyLessonPlanStageConfig(
	raw string,
) bool {
	switch strings.TrimSpace(raw) {
	case "", "[]", "null":
		return true
	default:
		return false
	}
}
