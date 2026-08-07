package services

// workshop_stage_flow.go — 阶段状态流转
//
// 本文件负责：
//   - 前进到下一阶段；
//   - 跳过当前阶段；
//   - 回退和切换阶段；
//   - 重启指定阶段；
//   - 保存内部阶段分隔符；
//   - 调用自然衔接与质量记忆旁路能力。
//
// 体验边界：
//   - 阶段切换不代表重新开始一段对话；
//   - 前序有效共识继续保留；
//   - 自动触发消息只用于推动自然承接，不允许向教师宣告阶段变化；
//   - 已确认内容不得因阶段切换而重新询问。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// AdvanceStage 进入下一阶段。
func (s *WorkshopStageService) AdvanceStage(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		nil,
		false,
	)
}

// AdvanceStageSilent 是对话模式使用的无质量教练阶段推进。
func (s *WorkshopStageService) AdvanceStageSilent(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	selectedComponentIDs []string,
) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		selectedComponentIDs,
		true,
	)
}

// AdvanceStageWithComponents 推进阶段并保存教师选择的组件。
func (s *WorkshopStageService) AdvanceStageWithComponents(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	selectedComponentIDs []string,
) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		selectedComponentIDs,
		false,
	)
}

func (s *WorkshopStageService) advanceStageWithComponents(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	selectedComponentIDs []string,
	skipQualityEvaluation bool,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, err
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIndex, err := s.resolveStages(lessonPlan)
	if err != nil {
		return nil, err
	}

	targetIndex := currentIndex + 1
	if targetStageCode != "" {
		targetIndex = findStageIndex(
			snapshots,
			targetStageCode,
		)
		if targetIndex == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else if targetIndex >= len(snapshots) {
		return nil, ErrStageAlreadyLast
	}

	if s.aesKey != "" &&
		!skipQualityEvaluation &&
		s.leavingStageHasSubstantiveContent(
			ctx,
			lessonPlanID,
		) {
		go s.asyncLLMEvaluateAndBroadcast(
			ctx,
			lessonPlanID,
			lessonPlan.CurrentStage,
		)
	}

	s.generateAndSaveEpisodicSummary(
		ctx,
		lessonPlanID,
		lessonPlan.CurrentStage,
	)

	_ = repository.CompleteStageOutput(
		ctx,
		lessonPlanID,
		lessonPlan.CurrentStage,
		"[]",
	)

	targetStage := snapshots[targetIndex]
	initialStructuredOutput := "{}"

	if len(selectedComponentIDs) > 0 {
		data := map[string]interface{}{
			"selected_component_ids": selectedComponentIDs,
		}

		if encoded, marshalErr := json.Marshal(data); marshalErr == nil {
			initialStructuredOutput = string(encoded)
		}

		wsLog.Info(
			"用户为阶段选择了组件",
			"plan_id", lessonPlanID,
			"stage", targetStage.StageCode,
			"component_count", len(selectedComponentIDs),
		)
	}

	output := &models.WorkshopStageOutput{
		LessonPlanID:         lessonPlanID,
		StageCode:            targetStage.StageCode,
		StageOrder:           targetStage.StageOrder,
		StructuredOutput:     initialStructuredOutput,
		NarrativeOutput:      "",
		ConversationSnapshot: "[]",
		Status:               models.StageOutputInProgress,
	}

	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn(
			"创建阶段产出记录失败，可能已经存在",
			"plan_id", lessonPlanID,
			"stage", targetStage.StageCode,
			"error", err,
		)
	}

	if err := repository.UpdateLessonPlanCurrentStage(
		ctx,
		lessonPlanID,
		targetStage.StageCode,
	); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}

	// 教师主动从write推进到review，代表当前正式正文已完成撰写，
	// 并被提交到下一步专业评审。
	//
	// 对话模式“AI评审一遍”和专家模式“完成本阶段，进入AI评审”
	// 都统一经过本入口，因此不需要依赖前端额外发送确认文本。
	//
	// 胶囊同步是确定性旁路操作；失败只记录日志，不撤销阶段推进。
	if lessonPlan.CurrentStage == "write" &&
		targetStage.StageCode == "review" {
		capsule, changed, capsuleErr :=
			applyLessonPlanContextCapsuleStageDecision(
				ctx,
				lessonPlan,
				targetStage.StageCode,
			)

		if capsuleErr != nil {
			wsLog.Warn(
				"write进入review时同步胶囊确认失败",
				"plan_id", lessonPlanID,
				"error", capsuleErr,
			)
		} else if changed &&
			capsule != nil {
			wsLog.Info(
				"write进入review时已同步结构化胶囊确认",
				"plan_id", lessonPlanID,
				"capsule_version", capsule.Version,
			)
		}
	}

	wsLog.Info(
		"进入下一阶段",
		"plan_id", lessonPlanID,
		"from", lessonPlan.CurrentStage,
		"to", targetStage.StageCode,
	)

	s.appendStageSeparator(
		ctx,
		lessonPlanID,
		targetStage,
	)

	s.triggerStageContinuation(
		lessonPlanID,
		targetStage.StageCode,
		callerID,
		skipQualityEvaluation,
		100*time.Millisecond,
	)

	return &targetStage, nil
}

// appendStageSeparator 持久化后台内部阶段边界。
func (s *WorkshopStageService) appendStageSeparator(
	ctx context.Context,
	lessonPlanID string,
	targetStage models.StageConfigSnapshot,
) {
	content := "__STAGE_SEP__" +
		targetStage.StageName +
		"__" +
		targetStage.AIRole

	message := &models.ConversationMessage{
		ID: fmt.Sprintf(
			"stage_sep_%s_%d",
			targetStage.StageCode,
			time.Now().UnixMilli(),
		),
		Role:      "system",
		Type:      "text",
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := repository.AppendConversationMessage(
		ctx,
		lessonPlanID,
		message,
	); err != nil {
		wsLog.Warn(
			"持久化阶段分隔符失败",
			"plan_id", lessonPlanID,
			"stage", targetStage.StageCode,
			"error", err,
		)
	}
}

// SkipStage 跳过当前可跳过阶段。
func (s *WorkshopStageService) SkipStage(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, err
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIndex, err := s.resolveStages(lessonPlan)
	if err != nil {
		return nil, err
	}
	if !snapshots[currentIndex].Skippable {
		return nil, ErrStageNotSkippable
	}

	s.generateAndSaveEpisodicSummary(
		ctx,
		lessonPlanID,
		lessonPlan.CurrentStage,
	)

	_ = repository.SkipStageOutput(
		ctx,
		lessonPlanID,
		lessonPlan.CurrentStage,
	)

	targetIndex := currentIndex + 1
	if targetStageCode != "" {
		targetIndex = findStageIndex(
			snapshots,
			targetStageCode,
		)
		if targetIndex == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else if targetIndex >= len(snapshots) {
		return nil, ErrStageAlreadyLast
	}

	targetStage := snapshots[targetIndex]
	output := &models.WorkshopStageOutput{
		LessonPlanID:         lessonPlanID,
		StageCode:            targetStage.StageCode,
		StageOrder:           targetStage.StageOrder,
		StructuredOutput:     "{}",
		NarrativeOutput:      "",
		ConversationSnapshot: "[]",
		Status:               models.StageOutputInProgress,
	}

	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn(
			"创建跳过后的阶段产出失败，可能已经存在",
			"plan_id", lessonPlanID,
			"stage", targetStage.StageCode,
			"error", err,
		)
	}

	if err := repository.UpdateLessonPlanCurrentStage(
		ctx,
		lessonPlanID,
		targetStage.StageCode,
	); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}

	wsLog.Info(
		"跳过阶段",
		"plan_id", lessonPlanID,
		"skipped", lessonPlan.CurrentStage,
		"to", targetStage.StageCode,
	)

	s.appendStageSeparator(
		ctx,
		lessonPlanID,
		targetStage,
	)

	s.triggerStageContinuation(
		lessonPlanID,
		targetStage.StageCode,
		callerID,
		false,
		100*time.Millisecond,
	)

	return &targetStage, nil
}

// BackStage 回退到上一阶段。
func (s *WorkshopStageService) BackStage(
	ctx context.Context,
	lessonPlanID string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, err
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, currentIndex, err := s.resolveStages(lessonPlan)
	if err != nil {
		return nil, err
	}
	if currentIndex <= 0 {
		return nil, ErrStageAlreadyFirst
	}

	targetStage := snapshots[currentIndex-1]

	if err := repository.UpdateLessonPlanCurrentStage(
		ctx,
		lessonPlanID,
		targetStage.StageCode,
	); err != nil {
		return nil, fmt.Errorf("回退阶段失败: %w", err)
	}

	wsLog.Info(
		"回退阶段",
		"plan_id", lessonPlanID,
		"from", lessonPlan.CurrentStage,
		"to", targetStage.StageCode,
	)

	return &targetStage, nil
}

// SwitchToStage 切换到指定阶段，不清理产出或对话。
func (s *WorkshopStageService) SwitchToStage(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, ErrLPGenPlanNotFound
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, _, err := s.resolveStages(lessonPlan)
	if err != nil {
		return nil, err
	}

	targetIndex := findStageIndex(
		snapshots,
		targetStageCode,
	)
	if targetIndex == -1 {
		return nil, ErrStageInvalidTarget
	}

	targetStage := snapshots[targetIndex]

	if err := repository.UpdateLessonPlanCurrentStage(
		ctx,
		lessonPlanID,
		targetStageCode,
	); err != nil {
		return nil, fmt.Errorf("切换阶段失败: %w", err)
	}

	wsLog.Info(
		"切换到指定阶段",
		"plan_id", lessonPlanID,
		"from", lessonPlan.CurrentStage,
		"to", targetStageCode,
	)

	return &targetStage, nil
}

// ResetStage 重启指定阶段并清理其后产出。
func (s *WorkshopStageService) ResetStage(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, ErrLPGenPlanNotFound
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, _, err := s.resolveStages(lessonPlan)
	if err != nil {
		return nil, err
	}

	targetIndex := findStageIndex(
		snapshots,
		targetStageCode,
	)
	if targetIndex == -1 {
		return nil, ErrStageInvalidTarget
	}

	targetStage := snapshots[targetIndex]

	if err := repository.ResetStageOutput(
		ctx,
		lessonPlanID,
		targetStageCode,
	); err != nil {
		wsLog.Warn(
			"重置阶段产出失败",
			"plan_id", lessonPlanID,
			"stage", targetStageCode,
			"error", err,
		)
	}

	if err := repository.DeleteStageOutputsAfter(
		ctx,
		lessonPlanID,
		targetStage.StageOrder,
	); err != nil {
		wsLog.Warn(
			"删除后续阶段产出失败",
			"plan_id", lessonPlanID,
			"stage", targetStageCode,
			"error", err,
		)
	}

	if targetStageCode == "write" ||
		targetStageCode == "revise" {
		_ = repository.UpdateLessonPlanContent(
			ctx,
			lessonPlanID,
			lessonPlan.Title,
			"",
			"{}",
			lessonPlan.DurationMinutes,
		)
	}

	if err := repository.UpdateLessonPlanCurrentStage(
		ctx,
		lessonPlanID,
		targetStageCode,
	); err != nil {
		return nil, fmt.Errorf("重置当前阶段失败: %w", err)
	}

	stageNames := make(
		map[string]string,
		len(snapshots),
	)
	for _, snapshot := range snapshots {
		stageNames[snapshot.StageCode] = snapshot.StageName
	}

	if err := repository.TruncateConversationFromStage(
		ctx,
		lessonPlanID,
		targetStageCode,
		stageNames,
	); err != nil {
		wsLog.Warn(
			"截断对话记录失败，尝试清空",
			"plan_id", lessonPlanID,
			"stage", targetStageCode,
			"error", err,
		)
		_ = repository.ClearConversationLog(
			ctx,
			lessonPlanID,
		)
	}

	resetSeparator := &models.ConversationMessage{
		ID: fmt.Sprintf(
			"stage_sep_%s_%d",
			targetStageCode,
			time.Now().UnixMilli(),
		),
		Role: "system",
		Type: "text",
		Content: "__STAGE_SEP__" +
			targetStage.StageName +
			"__",
		CreatedAt: time.Now(),
	}

	_ = repository.AppendConversationMessage(
		ctx,
		lessonPlanID,
		resetSeparator,
	)

	s.triggerStageContinuation(
		lessonPlanID,
		targetStageCode,
		callerID,
		false,
		200*time.Millisecond,
	)

	wsLog.Info(
		"重启阶段成功",
		"plan_id", lessonPlanID,
		"target_stage", targetStageCode,
	)

	return &targetStage, nil
}
