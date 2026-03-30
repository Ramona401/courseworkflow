package services

// lesson_plan_gen_service.go — 教案生成核心服务（主文件）
//
// 职责：
//   1. StartConversation  — 创建教案+静默注入背景组件+发起AI开场白
//   2. Chat               — 处理教师输入→流式AI回复→SSE逐token推送
//   3. TriggerAIReview    — 触发AI质量评审（异步，SSE推送结果）
//   4. ApplyAISuggestions — 将AI建议应用到教案内容（优化+重新评审）
//   5. GetConversation    — 获取教案对话历史
//
// 提示词构建+解析辅助函数 → lesson_plan_gen_prompts.go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrLPGenPlanNotFound    = errors.New("教案不存在")
	ErrLPGenSubjectRequired = errors.New("学科不能为空")
	ErrLPGenGradeRequired   = errors.New("年级不能为空")
	ErrLPGenTopicRequired   = errors.New("课题不能为空")
	ErrLPGenUnauthorized    = errors.New("无权操作此教案")
	ErrLPGenNotEditable     = errors.New("教案当前状态不可编辑")
)

// ==================== 服务结构体 ====================

// LessonPlanGenService 教案生成服务
type LessonPlanGenService struct {
	cfg interface{ GetAESKey() string }
}

var lpGenLog = logger.WithModule("lp_gen")

// NewLessonPlanGenService 创建教案生成服务
func NewLessonPlanGenService(cfg interface{ GetAESKey() string }) *LessonPlanGenService {
	return &LessonPlanGenService{cfg: cfg}
}

// ==================== 1. 开始备课会话 ====================

// StartConversation 创建教案+静默注入背景组件+发起AI开场白
// 返回：新建教案对象 + AI开场白消息
func (s *LessonPlanGenService) StartConversation(
	ctx context.Context,
	req *models.StartConversationRequest,
	authorID string,
) (*models.LessonPlan, *models.ConversationMessage, error) {
	if strings.TrimSpace(req.Subject) == "" {
		return nil, nil, ErrLPGenSubjectRequired
	}
	if strings.TrimSpace(req.Grade) == "" {
		return nil, nil, ErrLPGenGradeRequired
	}
	if strings.TrimSpace(req.Topic) == "" {
		return nil, nil, ErrLPGenTopicRequired
	}
	dur := req.DurationMinutes
	if dur <= 0 {
		dur = 45
	}

	title := fmt.Sprintf("%s %s — %s", req.Grade, req.Subject, req.Topic)
	lp := &models.LessonPlan{
		Title:           title,
		Subject:         req.Subject,
		Grade:           req.Grade,
		Topic:           req.Topic,
		DurationMinutes: dur,
		Status:          models.LPStatusDraft,
		Visibility:      models.LPVisibilityPersonal,
		AuthorID:        authorID,
		ConversationLog: "[]",
	}
	if req.GroupID != "" {
		lp.GroupID = &req.GroupID
	}

	if err := repository.CreateLessonPlan(ctx, lp); err != nil {
		return nil, nil, fmt.Errorf("创建教案失败: %w", err)
	}
	lpGenLog.Info("开始备课会话", "plan_id", lp.ID, "topic", req.Topic, "author", authorID)

	// 静默匹配背景类组件，注入到系统上下文
	silentGroups, _ := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject:       req.Subject,
		GradeRange:    req.Grade,
		InjectionMode: "silent",
		Limit:         3,
	})
	silentContext := buildSilentContext(silentGroups)

	// 解析提示词模板继承链
	systemPrompt, genRules := s.resolveTemplateForGen(ctx, req.TemplateID, req.Subject)

	// AI生成开场白（失败时降级为默认开场白）
	openingMsg, err := s.genOpeningMessage(ctx, req, systemPrompt, genRules, silentContext)
	if err != nil {
		lpGenLog.Warn("AI开场白生成失败，使用默认开场", "plan_id", lp.ID, "error", err)
		openingMsg = buildDefaultOpeningMessage(req)
	}

	if err2 := s.appendMessage(ctx, lp.ID, openingMsg); err2 != nil {
		lpGenLog.Warn("写入开场消息失败", "plan_id", lp.ID, "error", err2)
	}

	return lp, openingMsg, nil
}

// ==================== 2. 对话轮次（流式SSE推送）====================

// Chat 处理教师输入，AI生成回复并通过SSE流式推送
// 立即返回ACK，实际AI回复通过goroutine+SSE异步推送
func (s *LessonPlanGenService) Chat(
	ctx context.Context,
	req *models.LessonPlanChatRequest,
	callerID string,
) error {
	lp, err := s.checkPlanEditable(ctx, req.PlanID, callerID)
	if err != nil {
		return err
	}

	history, err := s.loadConversation(ctx, lp.ID)
	if err != nil {
		history = []*models.ConversationMessage{}
	}

	// 构造用户消息
	userMsg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleUser,
		Type:      models.ConvMsgTypeText,
		Content:   req.Message,
		CreatedAt: time.Now(),
	}
	if len(req.SelectedOptions) > 0 {
		userMsg.Content = formatSelectedOptions(req.SelectedOptions, req.Message)
	}
	if len(req.SelectedComponents) > 0 {
		userMsg.Content += formatSelectedComponents(req.SelectedComponents)
	}

	if err := s.appendMessage(ctx, lp.ID, userMsg); err != nil {
		lpGenLog.Warn("写入用户消息失败", "plan_id", lp.ID, "error", err)
	}

	// 异步执行流式AI回复
	go func() {
		bgCtx := context.Background()
		s.processChatAsync(bgCtx, lp, userMsg, history, req)
	}()

	return nil
}

// processChatAsync 异步处理AI流式回复
//
// 流程：
//  1. 推送 thinking 事件（前端显示呼吸灯）
//  2. CallAIStream：每个token回调推送 chunk 事件
//  3. 全部token收完后解析消息类型
//  4. 推送 message_done 事件（完整消息结构）
func (s *LessonPlanGenService) processChatAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	history []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
) {
	planID := lp.ID

	// 步骤1：推送thinking事件
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
		MessageID: generateMsgID(),
	})

	// 步骤2：获取AI配置
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
	if err != nil {
		s.broadcastError(planID, "AI配置加载失败: "+err.Error())
		return
	}

	// 步骤3：构造提示词
	systemPrompt, _ := s.resolveTemplateForGen(ctx, "", lp.Subject)
	userPrompt := buildChatPrompt(history, userMsg, lp)

	// 步骤4：流式调用AI，每个token立即推送chunk事件
	chunkCount := 0
	result, err := aiClient.CallAIStream(aiCfg, systemPrompt, userPrompt, func(chunk string) error {
		if strings.TrimSpace(chunk) == "" {
			return nil
		}
		chunkCount++
		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType: models.LPSSEChunk,
			PlanID:    planID,
			Chunk:     chunk,
		})
		return nil
	})
	if err != nil {
		s.broadcastError(planID, "AI回复失败: "+err.Error())
		return
	}
	_ = chunkCount

	// 步骤5：解析AI回复类型
	aiReply := s.parseAIReply(ctx, result.Content, lp)

	// 步骤6：保存消息到数据库
	if err := s.appendMessage(ctx, planID, aiReply); err != nil {
		lpGenLog.Warn("写入AI消息失败", "plan_id", planID, "error", err)
	}

	// 步骤7：内容块时更新教案正文并推送content_update事件
	if aiReply.Type == models.ConvMsgTypeContent {
		newContent := extractContentFromReply(result.Content)
		if newContent != "" {
			_ = repository.UpdateLessonPlanContent(ctx, planID,
				lp.Title, newContent, "{}", lp.DurationMinutes)
			GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
				EventType: models.LPSSEContentUpdate,
				PlanID:    planID,
				Content:   newContent,
			})
		}
	}

	// 步骤8：推送message_done事件
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEMessageDone,
		PlanID:    planID,
		MessageID: aiReply.ID,
		Message:   aiReply,
	})

	lpGenLog.Info("AI对话流式回复完成",
		"plan_id", planID,
		"tokens", result.TokensUsed,
		"latency_ms", result.LatencyMs,
		"chunks", chunkCount,
	)
}

// ==================== 3. 触发AI评审 ====================

// TriggerAIReview 异步触发AI质量评审，完成后SSE推送结果
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

// executeAIReviewAsync 异步执行AI评审（非流式，等待完整JSON）
func (s *LessonPlanGenService) executeAIReviewAsync(ctx context.Context, lp *models.LessonPlan) {
	planID := lp.ID

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
	})

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
	if err != nil {
		s.broadcastError(planID, "AI评审配置失败: "+err.Error())
		return
	}

	_, reviewRules := s.resolveTemplateForReview(ctx, lp.Subject)
	reviewPrompt := buildReviewPrompt(lp, reviewRules)
	systemPrompt := buildReviewSystemPrompt(lp.Subject)

	result, err := aiClient.CallAI(aiCfg, systemPrompt, reviewPrompt)
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

// ==================== 4. 应用AI建议 ====================

// ApplyAISuggestions 将AI评审建议应用到教案并重新评审
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

// applyAndReviewAsync 异步应用AI建议并重新评审
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

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
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

	result, err := aiClient.CallAI(aiCfg, systemPrompt, optimizePrompt)
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

// ==================== 5. 获取对话历史 ====================

// GetConversation 获取教案的完整对话记录
func (s *LessonPlanGenService) GetConversation(
	ctx context.Context,
	planID string,
	callerID string,
) ([]*models.ConversationMessage, error) {
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPGenPlanNotFound
		}
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	return s.loadConversation(ctx, planID)
}

// ==================== 内部辅助方法 ====================

// checkPlanEditable 校验教案存在、调用者有权限、状态可编辑
func (s *LessonPlanGenService) checkPlanEditable(ctx context.Context, planID string, callerID string) (*models.LessonPlan, error) {
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPGenPlanNotFound
		}
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	if lp.Status != models.LPStatusDraft &&
		lp.Status != models.LPStatusPublishedPersonal &&
		lp.Status != models.LPStatusRevision {
		return nil, ErrLPGenNotEditable
	}
	return lp, nil
}

// appendMessage 将消息追加到教案对话记录
func (s *LessonPlanGenService) appendMessage(ctx context.Context, planID string, msg *models.ConversationMessage) error {
	return repository.AppendConversationMessage(ctx, planID, msg)
}

// loadConversation 从数据库加载对话记录
func (s *LessonPlanGenService) loadConversation(ctx context.Context, planID string) ([]*models.ConversationMessage, error) {
	return repository.GetConversationLog(ctx, planID)
}

// resolveTemplateForGen 解析提示词模板（生成用）
// 优先使用指定模板，无模板时使用默认提示词
func (s *LessonPlanGenService) resolveTemplateForGen(ctx context.Context, templateID string, subject string) (systemPrompt string, genRules string) {
	if templateID != "" {
		resolved, err := repository.ResolvePromptTemplateChain(ctx, templateID)
		if err == nil {
			return resolved.SystemPrompt, resolved.GenerationRules
		}
	}
	return buildDefaultSystemPrompt(subject), buildDefaultGenRules()
}

// resolveTemplateForReview 解析提示词模板（评审用）
func (s *LessonPlanGenService) resolveTemplateForReview(ctx context.Context, subject string) (systemPrompt string, reviewRules string) {
	return buildReviewSystemPrompt(subject), buildDefaultReviewRules(subject)
}

// broadcastError 推送错误事件到SSE
func (s *LessonPlanGenService) broadcastError(planID string, msg string) {
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEError,
		PlanID:    planID,
		Error:     msg,
	})
}

// parseAIReply 解析AI回复，判断消息类型（文本/组件推荐/教案内容）
func (s *LessonPlanGenService) parseAIReply(ctx context.Context, content string, lp *models.LessonPlan) *models.ConversationMessage {
	msg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		CreatedAt: time.Now(),
	}

	// 教案内容块：包含教案标识关键词
	if strings.Contains(content, "## 教学目标") || strings.Contains(content, "# 教案") {
		msg.Type = models.ConvMsgTypeContent
		msg.Content = content
		return msg
	}

	// 组件推荐块：包含组件触发标记
	if strings.Contains(content, "【推荐组件】") || strings.Contains(content, "推荐以下教学方案") {
		msg.Type = models.ConvMsgTypeComponents
		msg.Content = cleanComponentMarkers(content)
		groups, _ := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
			Subject:       lp.Subject,
			GradeRange:    lp.Grade,
			InjectionMode: "recommend",
			Limit:         3,
		})
		msg.Components = convertGroupsToConvComponents(groups)
		return msg
	}

	// 普通文本
	msg.Type = models.ConvMsgTypeText
	msg.Content = content
	return msg
}

// genOpeningMessage AI生成备课开场白
func (s *LessonPlanGenService) genOpeningMessage(
	ctx context.Context,
	req *models.StartConversationRequest,
	systemPrompt string,
	genRules string,
	silentContext string,
) (*models.ConversationMessage, error) {
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
	if err != nil {
		return nil, err
	}

	userPrompt := fmt.Sprintf(`教师想开始备课：
学科：%s
年级：%s
课题：%s
课时：%d分钟
%s

请用友好的对话方式开场，采集2-3个关于学情的关键问题。
不要超过150字，用自然的口吻，可以用emoji增加亲和力。`,
		req.Subject, req.Grade, req.Topic, req.DurationMinutes, silentContext)

	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt)
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
