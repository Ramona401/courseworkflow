package services

// workshop_stage_quality_memory.go — 阶段质量旁路与Episodic记忆
//
// 本文件承载两个非关键路径能力：
//   - 阶段离开前的异步质量评估；
//   - 阶段结束时的轻量Episodic摘要。
//
// 两者都不能阻塞阶段切换和教师主对话。
//
// 注意：Episodic摘要目前仍是历史兼容层。新的备课核心共识胶囊接入后，
// 课程核心、教师确认决定、禁止偏离项和已纠正错误将以胶囊为正式事实源；
// 阶段摘要只承担阶段产出的轻量回顾，不能覆盖胶囊中的有效共识。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// asyncLLMEvaluateAndBroadcast 异步评估阶段质量并按需推送建议。
func (s *WorkshopStageService) asyncLLMEvaluateAndBroadcast(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
) {
	_ = ctx
	backgroundContext := context.Background()

	result, err := LLMEvaluateStageQuality(
		backgroundContext,
		s.aesKey,
		lessonPlanID,
		stageCode,
	)
	if err != nil {
		wsLog.Warn(
			"LLM阶段评估失败",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
		return
	}

	wsLog.Info(
		"LLM阶段评估完成",
		"plan_id", lessonPlanID,
		"stage", stageCode,
		"score", result.OverallScore,
		"qualified", result.IsQualified,
		"suggestion", coachTruncateStr(result.Suggestion, 50),
	)

	if result.IsQualified ||
		strings.TrimSpace(result.Suggestion) == "" {
		return
	}

	message := &models.ConversationMessage{
		ID: fmt.Sprintf(
			"coach_eval_%s_%d",
			stageCode,
			time.Now().UnixMilli(),
		),
		Role: models.ConvRoleAssistant,
		Type: models.ConvMsgTypeText,
		Content: fmt.Sprintf(
			"📋 阶段评估（%s，%d分）：%s",
			stageCodeToName(stageCode),
			result.OverallScore,
			result.Suggestion,
		),
		CreatedAt: time.Now(),
	}

	if err := repository.AppendConversationMessage(
		backgroundContext,
		lessonPlanID,
		message,
	); err != nil {
		wsLog.Warn(
			"LLM阶段评估消息写入失败",
			"plan_id", lessonPlanID,
			"error", err,
		)
	}

	GlobalLPSSEHub.Broadcast(
		lessonPlanID,
		models.LPSSEEvent{
			EventType: models.LPSSEMessageDone,
			PlanID:    lessonPlanID,
			MessageID: message.ID,
			Message:   message,
		},
	)
}

// generateAndSaveEpisodicSummary 保存当前阶段轻量摘要。
func (s *WorkshopStageService) generateAndSaveEpisodicSummary(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
) {
	messages, err := repository.GetCurrentStageMessages(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		wsLog.Warn(
			"生成Episodic摘要时读取阶段消息失败",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
		return
	}

	output, err := repository.GetStageOutput(
		ctx,
		lessonPlanID,
		stageCode,
	)
	if err != nil {
		wsLog.Warn(
			"生成Episodic摘要时读取阶段产出失败",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
		output = &models.WorkshopStageOutput{}
	}

	existingNarrative := strings.TrimSpace(
		output.NarrativeOutput,
	)
	if len([]rune(existingNarrative)) > 100 {
		wsLog.Info(
			"Episodic摘要已有正式内容，跳过覆盖",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"existing_length", len([]rune(existingNarrative)),
		)
		return
	}

	summary := GenerateStageSummary(
		stageCode,
		messages,
		output.StructuredOutput,
	)
	if strings.TrimSpace(summary) == "" {
		return
	}

	if err := repository.UpdateStageNarrativeOutput(
		ctx,
		lessonPlanID,
		stageCode,
		summary,
	); err != nil {
		wsLog.Warn(
			"保存Episodic摘要失败",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
		return
	}

	wsLog.Info(
		"Episodic摘要生成并保存成功",
		"plan_id", lessonPlanID,
		"stage", stageCode,
		"summary_length", len([]rune(summary)),
	)
}
