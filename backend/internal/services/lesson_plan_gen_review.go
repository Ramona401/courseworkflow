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
) {
	switch stageCode {
	case "write", "revise":
		s.handleWriteStageOutput(ctx, planID, lp, structuredJSON, rawContent)
	case "review":
		s.handleReviewStageOutput(ctx, planID, structuredJSON)
	}
}

// handleWriteStageOutput 处理write/revise阶段产出物
func (s *LessonPlanGenService) handleWriteStageOutput(
	ctx context.Context,
	planID string,
	lp *models.LessonPlan,
	structuredJSON string,
	rawContent string,
) {
	content := ""

	// 正常路径：structuredJSON有效
	if structuredJSON != "" && structuredJSON != "{}" {
		var structured map[string]interface{}
		if err := json.Unmarshal([]byte(structuredJSON), &structured); err == nil {
			if contentRaw, ok := structured["content_markdown"]; ok {
				if cs, ok := contentRaw.(string); ok {
					content = strings.TrimSpace(cs)
				}
			}
		}
	}

	// 降级路径：从rawContent中重新检测教案内容
	if content == "" && rawContent != "" {
		content = DetectLessonPlanContent(rawContent)
		if content != "" {
			lpGenLog.Info("write阶段从rawContent fallback提取教案内容",
				"plan_id", planID, "content_len", len(content))

			updatedStructured := map[string]interface{}{
				"content_markdown": content,
			}
			if b, err := json.Marshal(updatedStructured); err == nil {
				_ = s.stageService.SaveStageOutput(ctx, planID, lp.CurrentStage, string(b), "", "", 0)
			}
		}
	}

	if content == "" {
		lpGenLog.Warn("write阶段未能提取到教案内容", "plan_id", planID)
		return
	}

	// 更新教案正文到lesson_plans表
	if err := repository.UpdateLessonPlanContent(ctx, planID, lp.Title, content, "{}", lp.DurationMinutes); err != nil {
		lpGenLog.Warn("write阶段更新教案正文失败", "plan_id", planID, "error", err)
		return
	}

	// 推送内容更新事件给前端
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEContentUpdate,
		PlanID:    planID,
		Content:   content,
	})

	// v74保留：自动将write/revise阶段标记为completed
	if err := repository.CompleteStageOutput(ctx, planID, lp.CurrentStage, "[]"); err != nil {
		lpGenLog.Warn("自动完成write阶段产出失败", "plan_id", planID, "error", err)
	} else {
		lpGenLog.Info("write/revise阶段产出自动标记completed", "plan_id", planID, "stage", lp.CurrentStage)
	}

	lpGenLog.Info("write/revise阶段教案正文已更新", "plan_id", planID, "content_len", len(content))
}

// handleReviewStageOutput 处理review阶段产出物
func (s *LessonPlanGenService) handleReviewStageOutput(
	ctx context.Context,
	planID string,
	structuredJSON string,
) {
	if structuredJSON == "" || structuredJSON == "{}" {
		return
	}

	var reviewResult *models.AIReviewResult
	if err := json.Unmarshal([]byte(structuredJSON), &reviewResult); err != nil || reviewResult == nil {
		lpGenLog.Warn("解析review阶段structured为AIReviewResult失败", "plan_id", planID, "error", err)
		return
	}

	if reviewResult.TotalScore <= 0 {
		lpGenLog.Warn("review阶段structured的total_score无效", "plan_id", planID, "score", reviewResult.TotalScore)
		return
	}

	reviewResult.ReviewedAt = time.Now()

	resultJSON, _ := json.Marshal(reviewResult)
	if err := repository.UpdateLessonPlanAIReview(ctx, planID,
		reviewResult.TotalScore,
		string(resultJSON),
		"[]",
	); err != nil {
		lpGenLog.Warn("保存review阶段评审结果失败", "plan_id", planID, "error", err)
		return
	}

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEReviewDone,
		PlanID:    planID,
		Review:    reviewResult,
	})

	lpGenLog.Info("review阶段评审结果已保存并推送", "plan_id", planID, "score", reviewResult.TotalScore)

	// v89新增：review阶段完成后自动触发教案索引生成
	go s.triggerAutoLessonIndex(ctx, planID, &reviewResult.TotalScore)
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
	lpGenLog.Info("触发AI评审", "plan_id", planID)
	go func() {
		bgCtx := context.Background()
		s.executeAIReviewAsync(bgCtx, lp)
	}()
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
		s.broadcastError(planID, "AI评审配置失败: "+err.Error())
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
		s.broadcastError(planID, "AI评审失败: "+err.Error())
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
	lpGenLog.Info("应用AI建议", "plan_id", req.PlanID, "suggestions_count", len(req.Suggestions))
	go func() {
		bgCtx := context.Background()
		s.applyAndReviewAsync(bgCtx, lp, req.Suggestions)
	}()
	return nil
}

// applyAndReviewAsync 异步应用建议并重新评审
func (s *LessonPlanGenService) applyAndReviewAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	suggestionIDs []string,
) {
	planID := lp.ID

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
	})

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		s.broadcastError(planID, "AI配置失败: "+err.Error())
		return
	}

	suggestions := extractSuggestionsByIDs(lp.AIReviewResult, suggestionIDs)
	if len(suggestions) == 0 {
		s.broadcastError(planID, "未找到有效的改进建议")
		return
	}

	optimizePrompt := buildOptimizePrompt(lp.ContentMarkdown, suggestions)
	systemPrompt := fmt.Sprintf(
		"你是一位专业的%s课教案优化专家。请根据评审建议改进教案内容，保持原有结构，重点改进被指出的问题。输出完整的改进后教案Markdown。",
		lp.Subject,
	)

	// v89-2：构建TraceContext，关联教案ID和作者
	optimizeTraceCtx := &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &lp.AuthorID,
	}
	result, err := aiClient.CallAI(aiCfg, systemPrompt, optimizePrompt, optimizeTraceCtx)
	if err != nil {
		s.broadcastError(planID, "AI优化失败: "+err.Error())
		return
	}

	newContent := strings.TrimSpace(result.Content)
	if newContent == "" {
		s.broadcastError(planID, "AI优化返回内容为空")
		return
	}

	_ = repository.UpdateLessonPlanContent(ctx, planID, lp.Title, newContent, "{}", lp.DurationMinutes)

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEContentUpdate,
		PlanID:    planID,
		Content:   newContent,
	})

	lp.ContentMarkdown = newContent
	s.executeAIReviewAsync(ctx, lp)
}

// ==================== 旧版开场白（保留兼容）====================

// genOpeningMessage 旧版开场白生成（保留兼容，已不常用）
func (s *LessonPlanGenService) genOpeningMessage(
	ctx context.Context,
	req *models.StartConversationRequest,
	systemPrompt string,
	genRules string,
	backgroundContext string,
) (*models.ConversationMessage, error) {
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		return nil, err
	}

	recipeHint := ""
	if req.RecipeID != "" && backgroundContext != "" {
		recipeHint = "\n注意：老师已选择了备课配方，你已经了解了学情、教学风格等背景信息。开场时可以直接体现你对学生情况的了解，不需要从零开始问学情。可以直接进入教学方案探讨。"
	}

	userPrompt := fmt.Sprintf(`教师想开始备课：
学科：%s
年级：%s
课题：%s
课时：%d分钟
%s
%s
请用友好的对话方式开场，采集2-3个关于学情的关键问题。
不要超过150字，用自然的口吻，可以用emoji增加亲和力。`,
		req.Subject, req.Grade, req.Topic, req.DurationMinutes, backgroundContext, recipeHint)

	// v89-2：构建TraceContext（旧版开场白保留兼容，仅标记场景）
	oldOpeningTraceCtx := &aiClient.TraceContext{
		SceneCode: lessonPlanSceneCode,
	}
	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, oldOpeningTraceCtx)
	if err != nil {
		return nil, err
	}

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   result.Content,
		CreatedAt: time.Now(),
	}, nil
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
