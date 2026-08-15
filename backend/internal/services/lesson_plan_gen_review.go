package services

// lesson_plan_gen_review.go — 教案生成服务：评审+产出物处理+建议应用+自动索引
//
// v89-3拆分：从lesson_plan_gen_service.go中拆出评审相关逻辑
//
// 职责：
//   1. handleStageOutputSideEffects — 阶段产出物副作用分发
//   2. handleWriteStageOutput — 处理write/revise阶段教案正文提取+保存
//   3. handleReviewStageOutput — 处理review阶段评审结果保存+SSE推送+自动索引触发
//   4. TriggerAIReview — 手动触发AI评审
//   5. executeAIReviewAsync — 异步执行AI评审
//   6. ApplyAISuggestions — 应用AI改进建议
//   7. applyAndReviewAsync — 异步应用建议+重新评审
//   8. genOpeningMessage — 旧版开场白生成（保留兼容）
//   9. triggerAutoLessonIndex — review完成后自动生成教案AOCI索引
//
// v169改动（评审"有分无内容"治本）：
//   - handleReviewStageOutput 去掉「total_score<=0 静默return」的硬门槛：
//       旧逻辑下，只要 extractReviewStageFromNatural 没抠到分数（AI 格式飘了），
//       后端直接 return，既不保存也不广播 → 前端永远等不到 onReviewDone，
//       右侧停在"评审报告将自动显示在这里" = 用户"看不到内容"的直接成因。
//   - 新逻辑：解析得到的 reviewResult 若 total_score<=0，用对话原文兜底
//       （buildFallbackReview，复用 lesson_plan_gen_prompts.go 的现成函数），
//       保证前端永远能拿到一个可渲染的 review 对象。
//   - 仅当兜底也拿不到任何可用内容时才不保存分数（但仍广播 fallback 供展示）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 阶段产出物副作用处理 ====================

// handleStageOutputSideEffects 根据阶段类型处理产出物副作用
func (s *LessonPlanGenService) handleStageOutputSideEffects(
	ctx context.Context,
	planID string,
	lp *models.LessonPlan,
	stageCode string,
	structuredJSON string,
	rawContent string,
	turnID string,
) error {
	switch stageCode {
	case "write", "revise":
		return s.handleWriteStageOutput(
			ctx,
			planID,
			lp,
			stageCode,
			structuredJSON,
			rawContent,
			turnID,
		)

	case "review":
		s.handleReviewStageOutput(
			ctx,
			planID,
			structuredJSON,
			rawContent,
		)
	}

	return nil
}

// handleWriteStageOutput 处理write/revise阶段产出物。
//
// 正文保存统一经过UpdateLessonPlanContentPreservingWord：
//   - 普通教案走事务级CAS；
//   - Word保真教案同时生成DOCX不可变版本；
//   - 并发冲突或结构不匹配时，原正文和原Word都不改变。
func (s *LessonPlanGenService) handleWriteStageOutput(
	ctx context.Context,
	planID string,
	lp *models.LessonPlan,
	stageCode string,
	structuredJSON string,
	rawContent string,
	turnID string,
) error {
	if lp == nil {
		return errors.New("教案上下文为空")
	}

	content := ""

	if structuredJSON != "" &&
		structuredJSON != "{}" {
		var structured map[string]interface{}

		if err :=
			json.Unmarshal(
				[]byte(structuredJSON),
				&structured,
			); err == nil {
			if contentRaw, ok :=
				structured["content_markdown"]; ok {
				if contentString, ok :=
					contentRaw.(string); ok {
					content =
						strings.TrimSpace(
							contentString,
						)
				}
			}
		}
	}

	if content == "" &&
		rawContent != "" {
		content =
			DetectLessonPlanContent(
				rawContent,
			)

		if content != "" {
			lpGenLog.Info(
				"write/revise阶段从rawContent提取教案正文",
				"plan_id", planID,
				"stage", stageCode,
				"client_turn_id", turnID,
				"content_len", len(content),
			)
		}
	}

	if content == "" {
		lpGenLog.Warn(
			"write/revise阶段未提取到可保存正文",
			"plan_id", planID,
			"stage", stageCode,
			"client_turn_id", turnID,
		)

		existingForBroadcast :=
			strings.TrimSpace(
				lp.ContentMarkdown,
			)
		if existingForBroadcast != "" {
			GlobalLPSSEHub.Broadcast(
				planID,
				models.LPSSEEvent{
					EventType: models.LPSSEContentUpdate,
					PlanID:    planID,
					Content:   existingForBroadcast,
				},
			)
		}

		return errors.New(
			"AI没有输出可保存的完整教案正文",
		)
	}

	existingMarkdown :=
		strings.TrimSpace(
			lp.ContentMarkdown,
		)
	existingRunes :=
		[]rune(existingMarkdown)
	newRunes :=
		[]rune(content)

	if len(existingRunes) > 800 &&
		len(newRunes) <
			len(existingRunes)*7/10 &&
		!hasLessonPlanOpeningStructure(
			content,
		) {
		lpGenLog.Warn(
			"write/revise阶段缩写保护拒绝覆盖",
			"plan_id", planID,
			"stage", stageCode,
			"client_turn_id", turnID,
			"existing_runes", len(existingRunes),
			"new_runes", len(newRunes),
		)

		GlobalLPSSEHub.Broadcast(
			planID,
			models.LPSSEEvent{
				EventType: models.LPSSEContentUpdate,
				PlanID:    planID,
				Content:   existingMarkdown,
			},
		)

		return errors.New(
			"AI生成的新正文明显不完整，系统已保留原教案",
		)
	}

	changeSummary :=
		"AI在教案撰写阶段更新完整正文"
	if stageCode == "revise" {
		changeSummary =
			"AI在修订定稿阶段更新完整正文"
	}

	mutation, err :=
		UpdateLessonPlanContentPreservingWord(
			ctx,
			LessonPlanContentMutationInput{
				PlanID:            planID,
				CallerID:          lp.AuthorID,
				Title:             lp.Title,
				ContentMarkdown:   content,
				ContentStructured: lp.ContentStructured,
				DurationMinutes:   lp.DurationMinutes,
				ExpectedVersion:   lp.Version,
				ExpectedContent:   lp.ContentMarkdown,
				ChangeSource: models.
					LessonPlanWordChangeSourceAI,
				ChangeSummary: changeSummary,
			},
		)
	if err != nil {
		lpGenLog.Warn(
			"write/revise阶段正文与Word同步失败",
			"plan_id", planID,
			"stage", stageCode,
			"client_turn_id", turnID,
			"error", err,
		)
		return err
	}

	GlobalLPSSEHub.Broadcast(
		planID,
		models.LPSSEEvent{
			EventType: models.LPSSEContentUpdate,
			PlanID:    planID,
			Content:   mutation.ContentMarkdown,
		},
	)

	lp.ContentMarkdown =
		mutation.ContentMarkdown
	lp.Version =
		mutation.CurrentVersion

	// 阶段产出内容与 completed 状态由提交层在正文成功后通过
	// SaveCompletedStageOutput 一次性原子固化；这里不再提前单独 Complete，
	// 避免“正文已更新、完成状态更新失败、随后产出保存又失败”的双事实源。

	lpGenLog.Info(
		"write/revise阶段教案正文已安全更新",
		"plan_id", planID,
		"stage", stageCode,
		"content_len",
		len(mutation.ContentMarkdown),
		"current_version",
		mutation.CurrentVersion,
	)

	return nil
}

// handleReviewStageOutput 处理review阶段产出物
//
// v169改动：新增 rawContent 参数 + 去掉 total_score<=0 静默return 硬门槛。
//   - 优先用 extractReviewStageFromNatural 解析出的 structuredJSON
//   - 若解析为空或 total_score<=0，用 rawContent（AI对话原文）构造 fallback review 广播，
//     保证前端永远能渲染（评分用兜底7.0，summary 用原文截断）
//   - 只有在能拿到有效 total_score 时才落库到 lesson_plans.ai_review_result（避免脏分入库）；
//     fallback 仅广播不落库（不污染数据，但解决"看不到"）
func (s *LessonPlanGenService) handleReviewStageOutput(
	ctx context.Context,
	planID string,
	structuredJSON string,
	rawContent string,
) {
	var reviewResult *models.AIReviewResult

	// 尝试解析结构化 JSON
	if structuredJSON != "" && structuredJSON != "{}" {
		if err := json.Unmarshal([]byte(structuredJSON), &reviewResult); err != nil {
			lpGenLog.Warn("解析review阶段structured为AIReviewResult失败", "plan_id", planID, "error", err)
			reviewResult = nil
		}
	}

	// 判定是否拿到有效评审（含有效总分）
	hasValidScore := reviewResult != nil && reviewResult.TotalScore > 0

	if !hasValidScore {
		// v169兜底：解析失败或无有效分数时，用对话原文构造 fallback review 广播
		// 保证前端右侧面板永远能渲染，不再卡在"评审报告将自动显示在这里"
		fallback := buildFallbackReview(safeReviewRawFallback(rawContent))
		fallback.ReviewedAt = time.Now()

		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType: models.LPSSEReviewDone,
			PlanID:    planID,
			Review:    fallback,
		})
		lpGenLog.Warn("review阶段未解析到有效结构化评审，已广播fallback供前端展示（不落库）",
			"plan_id", planID, "raw_len", len(rawContent))
		return
	}

	// 正常路径：有有效分数，落库 + 广播
	reviewResult.ReviewedAt = time.Now()

	resultJSON, _ := json.Marshal(reviewResult)
	if err := repository.UpdateLessonPlanAIReview(ctx, planID,
		reviewResult.TotalScore,
		string(resultJSON),
		"[]",
	); err != nil {
		lpGenLog.Warn("保存review阶段评审结果失败", "plan_id", planID, "error", err)
		// 落库失败也仍然广播，保证前端能看到
	}

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEReviewDone,
		PlanID:    planID,
		Review:    reviewResult,
	})

	lpGenLog.Info("review阶段评审结果已保存并推送",
		"plan_id", planID, "score", reviewResult.TotalScore,
		"good_points", len(reviewResult.GoodPoints),
		"improvements", len(reviewResult.Improvements))

	// v89新增：review阶段完成后自动触发教案索引生成
	s.triggerAutoLessonIndexTracked(
		planID,
		&reviewResult.TotalScore,
	)
}

// safeReviewRawFallback 为 fallback review 的 suggestion 字段准备原文（v169新增）
// 截断到 1500 字符，避免把超长对话整段塞进 review 对象
func safeReviewRawFallback(rawContent string) string {
	raw := strings.TrimSpace(rawContent)
	if raw == "" {
		return "AI已输出评审内容，请在左侧对话中查看完整评审报告。"
	}
	return safeUTF8Truncate(raw, 1500)
}

// ==================== 触发AI评审 ====================

// TriggerAIReview 触发AI质量评审（异步执行，结果通过SSE推送）
func (s *LessonPlanGenService) TriggerAIReview(
	ctx context.Context,
	planID string,
	callerID string,
) error {
	lp, err := s.checkPlanEditable(ctx, planID, callerID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(lp.ContentMarkdown) == "" {
		return errors.New("教案内容为空，无法评审")
	}
	task, taskErr := startLessonPlanAITask(planID)
	if taskErr != nil {
		return taskErr
	}

	lpGenLog.Info("触发AI评审", "plan_id", planID)

	s.runLessonPlanAITask(
		task,
		planID,
		"",
		"review",
		func() {
			s.executeAIReviewAsync(
				context.Background(),
				lp,
			)
		},
	)

	return nil
}

// executeAIReviewAsync 异步执行AI评审
func (s *LessonPlanGenService) executeAIReviewAsync(ctx context.Context, lp *models.LessonPlan) {
	planID := lp.ID

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
	})

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		s.broadcastError(planID, "", "AI评审配置失败: "+err.Error())
		return
	}

	_, reviewRules := s.resolveTemplateForReview(ctx, lp.Subject)
	reviewPrompt := buildReviewPrompt(lp, reviewRules)
	systemPrompt := buildReviewSystemPrompt(lp.Subject)

	// v89-2：构建TraceContext，关联教案ID和作者
	reviewTraceCtx := &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &lp.AuthorID,
	}
	result, err := aiClient.CallAI(aiCfg, systemPrompt, reviewPrompt, reviewTraceCtx)
	if err != nil {
		s.broadcastError(planID, "", "AI评审失败: "+err.Error())
		return
	}

	reviewResult, err := parseAIReviewResult(result.Content)
	if err != nil {
		lpGenLog.Warn("解析AI评审结果失败，使用原始文本", "plan_id", planID, "error", err)
		reviewResult = buildFallbackReview(result.Content)
	}

	oldHistory := "[]"
	if lp.AIReviewHistory != "" {
		oldHistory = lp.AIReviewHistory
	}
	newHistory := appendReviewToHistory(oldHistory, reviewResult)
	resultJSON, _ := json.Marshal(reviewResult)

	if err := repository.UpdateLessonPlanAIReview(ctx, planID,
		reviewResult.TotalScore,
		string(resultJSON),
		newHistory,
	); err != nil {
		lpGenLog.Error("保存AI评审结果失败", "plan_id", planID, "error", err)
	}

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEReviewDone,
		PlanID:    planID,
		Review:    reviewResult,
	})

	lpGenLog.Info("AI评审完成",
		"plan_id", planID,
		"score", reviewResult.TotalScore,
		"tokens", result.TokensUsed,
	)
}

// ==================== 应用AI建议 ====================

// ApplyAISuggestions 将AI评审建议应用到教案内容（异步优化+重新评审）
func (s *LessonPlanGenService) ApplyAISuggestions(
	ctx context.Context,
	req *models.ApplyAISuggestionsRequest,
	callerID string,
) error {
	lp, err := s.checkPlanEditable(ctx, req.PlanID, callerID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(lp.ContentMarkdown) == "" {
		return errors.New("教案内容为空")
	}
	if strings.TrimSpace(lp.AIReviewResult) == "" {
		return errors.New("尚未生成AI评审，请先触发评审")
	}
	task, taskErr := startLessonPlanAITask(req.PlanID)
	if taskErr != nil {
		return taskErr
	}

	lpGenLog.Info(
		"应用AI建议",
		"plan_id", req.PlanID,
		"suggestions_count", len(req.Suggestions),
	)

	suggestionIDs := append(
		[]string(nil),
		req.Suggestions...,
	)

	s.runLessonPlanAITask(
		task,
		req.PlanID,
		"",
		"apply_suggestions",
		func() {
			s.applyAndReviewAsync(
				context.Background(),
				lp,
				suggestionIDs,
			)
		},
	)

	return nil
}

// applyAndReviewAsync 异步应用建议并重新评审
func (s *LessonPlanGenService) applyAndReviewAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	suggestionIDs []string,
) {
	if lp == nil {
		return
	}

	planID := lp.ID

	GlobalLPSSEHub.Broadcast(
		planID,
		models.LPSSEEvent{
			EventType: models.LPSSEThinking,
			PlanID:    planID,
		},
	)

	aiConfig, err :=
		aiClient.GetEffectiveConfig(
			s.cfg.GetAESKey(),
			lessonPlanSceneCode,
			"",
			"",
			"",
		)
	if err != nil {
		s.broadcastError(
			planID,
			"",
			"AI配置失败: "+err.Error(),
		)
		return
	}

	suggestions :=
		extractSuggestionsByIDs(
			lp.AIReviewResult,
			suggestionIDs,
		)
	if len(suggestions) == 0 {
		s.broadcastError(
			planID,
			"",
			"未找到有效的改进建议",
		)
		return
	}

	optimizePrompt :=
		buildOptimizePrompt(
			lp.ContentMarkdown,
			suggestions,
		)
	systemPrompt :=
		fmt.Sprintf(
			"你是一位专业的%s课教案优化专家。请根据评审建议改进教案内容，保持原有结构，重点改进被指出的问题。输出完整的改进后教案Markdown。",
			lp.Subject,
		)

	if wordPrompt, _ :=
		buildLessonPlanWordFidelityMutationPrompt(
			ctx,
			lp,
			"应用AI评审建议",
		); wordPrompt != "" {
		systemPrompt += wordPrompt
	}

	optimizeTraceContext :=
		&aiClient.TraceContext{
			SceneCode:    lessonPlanSceneCode,
			LessonPlanID: &planID,
			UserID:       &lp.AuthorID,
		}

	result, err :=
		aiClient.CallAI(
			aiConfig,
			systemPrompt,
			optimizePrompt,
			optimizeTraceContext,
		)
	if err != nil {
		s.broadcastError(
			planID,
			"",
			"AI优化失败: "+err.Error(),
		)
		return
	}

	newContent :=
		strings.TrimSpace(
			result.Content,
		)
	if newContent == "" {
		s.broadcastError(
			planID,
			"",
			"AI优化返回内容为空",
		)
		return
	}

	mutation, err :=
		UpdateLessonPlanContentPreservingWord(
			ctx,
			LessonPlanContentMutationInput{
				PlanID:            planID,
				CallerID:          lp.AuthorID,
				Title:             lp.Title,
				ContentMarkdown:   newContent,
				ContentStructured: lp.ContentStructured,
				DurationMinutes:   lp.DurationMinutes,
				ExpectedVersion:   lp.Version,
				ExpectedContent:   lp.ContentMarkdown,
				ChangeSource: models.
					LessonPlanWordChangeSourceAI,
				ChangeSummary: "应用AI评审建议并更新教案正文",
			},
		)
	if err != nil {
		lpGenLog.Warn(
			"应用AI评审建议保存失败",
			"plan_id", planID,
			"error", err,
		)

		s.broadcastError(
			planID,
			"",
			lessonPlanContentMutationPublicMessage(
				err,
			),
		)
		return
	}

	GlobalLPSSEHub.Broadcast(
		planID,
		models.LPSSEEvent{
			EventType: models.LPSSEContentUpdate,
			PlanID:    planID,
			Content:   mutation.ContentMarkdown,
		},
	)

	lp.ContentMarkdown =
		mutation.ContentMarkdown
	lp.Version =
		mutation.CurrentVersion

	s.executeAIReviewAsync(
		ctx,
		lp,
	)
}

// ==================== 自动教案索引触发 ====================

// triggerAutoLessonIndex review阶段完成后自动生成教案AOCI索引
//
// 异步执行，不阻塞用户操作。失败时只记录日志不影响主流程。
// 使用scanner场景（Haiku模型）低成本压缩。
//
// 触发条件：review阶段评审结果保存成功后
// 跳过条件：教案已有索引（lesson_index非空）
func (s *LessonPlanGenService) triggerAutoLessonIndex(ctx context.Context, planID string, aiScore *float64) {
	// 延迟1秒，确保评审数据完全写入
	time.Sleep(1 * time.Second)

	// 查询教案完整信息
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		lpGenLog.Warn("v89自动索引-查询教案失败", "plan_id", planID, "error", err)
		return
	}

	// 跳过已有索引的教案
	if lp.LessonIndex != "" {
		lpGenLog.Debug("v89自动索引-教案已有索引，跳过", "plan_id", planID)
		return
	}

	// 跳过无内容的教案
	if strings.TrimSpace(lp.ContentMarkdown) == "" {
		lpGenLog.Debug("v89自动索引-教案无内容，跳过", "plan_id", planID)
		return
	}

	lpGenLog.Info("v89自动索引-开始生成", "plan_id", planID, "title", lp.Title)

	// 构建全文
	fullText := utils.BuildLessonFullText(
		lp.Subject, lp.Grade, lp.Topic, lp.Title, lp.DurationMinutes,
		lp.ContentMarkdown, lp.ContentStructured, lp.AIReviewResult, lp.MatchedComponents,
	)

	// 创建索引服务实例并压缩
	liService := NewLessonIndexService(s.cfg.(*config.Config))
	// v89-2：CompressLessonIndex新增planID参数
	indexText, err := liService.CompressLessonIndex(fullText, planID)
	if err != nil {
		lpGenLog.Warn("v89自动索引-AI压缩失败", "plan_id", planID, "error", err)
		return
	}

	// 保存索引
	if err := liService.SaveLessonIndex(ctx, planID, indexText, aiScore, string(lp.Status)); err != nil {
		lpGenLog.Warn("v89自动索引-保存失败", "plan_id", planID, "error", err)
		return
	}

	lpGenLog.Info("v89自动索引-生成完成", "plan_id", planID, "index_len", len(indexText))
}
