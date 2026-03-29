package services

// ==================== 教案生成核心服务 ====================
// Phase3：教案对话式生成全流程
//
// 职责：
//   1. StartConversation  — 创建教案+静默注入背景组件+发起AI开场白
//   2. Chat               — 处理教师输入→组装上下文→AI流式回复→SSE逐token推送
//   3. TriggerAIReview    — 触发AI质量评审（异步，完成后SSE推送）
//   4. ApplyAISuggestions — 将AI建议应用到教案内容（更新+重新评审）
//
// 流式输出说明（v2改动）：
//   processChatAsync 改用 aiClient.CallAIStream()，
//   每收到一个token立即广播 LPSSEChunk 事件给前端，
//   全部token收完后广播 LPSSEMessageDone 事件（含完整消息结构）。
//   前端在 chunk 事件中逐字追加到临时气泡，message_done 时替换为正式消息。

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

// StartConversation 创建教案+静默注入背景+发起AI开场白
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

	// 静默匹配背景类组件
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

// ==================== 2. 对话轮次（AI回复流式SSE推送）====================

// Chat 处理教师输入，AI生成回复并通过SSE流式推送
// 立即返回ACK，实际AI回复通过goroutine+SSE流式推送
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

// processChatAsync 异步处理AI回复（流式版本）
//
// 流程：
//   1. 推送 thinking 事件（显示呼吸灯）
//   2. 调用 CallAIStream，每个token回调推送 chunk 事件
//   3. 全部token收完后，解析消息类型（文本/组件/内容）
//   4. 推送 message_done 事件（含完整消息结构）
func (s *LessonPlanGenService) processChatAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	history []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
) {
	planID := lp.ID

	// 步骤1：推送thinking事件（前端显示呼吸灯）
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
		MessageID: generateMsgID(),
	})

	// 步骤2：获取AI配置
	aiCfg, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(), "", "", "", "",
	)
	if err != nil {
		s.broadcastError(planID, "AI配置加载失败: "+err.Error())
		return
	}

	// 步骤3：构造提示词
	systemPrompt, _ := s.resolveTemplateForGen(ctx, "", lp.Subject)
	userPrompt := buildChatPrompt(history, userMsg, lp)

	// 步骤4：流式调用AI，每个token立即推送chunk事件
	// chunkCount 用于过滤掉极短的首个空chunk
	chunkCount := 0
	result, err := aiClient.CallAIStream(aiCfg, systemPrompt, userPrompt, func(chunk string) error {
		// 跳过空chunk
		if strings.TrimSpace(chunk) == "" {
			return nil
		}
		chunkCount++
		// 广播chunk事件给前端（前端逐字追加到临时气泡）
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

	_ = chunkCount // 已使用

	// 步骤5：解析AI回复类型（文本/组件推荐/内容块）
	aiReply := s.parseAIReply(ctx, result.Content, lp)

	// 步骤6：保存完整消息到数据库
	if err := s.appendMessage(ctx, planID, aiReply); err != nil {
		lpGenLog.Warn("写入AI消息失败", "plan_id", planID, "error", err)
	}

	// 步骤7：如果是教案内容块，更新教案正文并推送content_update事件
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

	// 步骤8：推送message_done事件（前端用完整消息替换临时流式气泡）
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

// executeAIReviewAsync 异步执行AI评审（非流式，等待完整JSON结果）
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

	// 评审使用非流式调用（需要完整JSON才能解析）
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

// ApplyAISuggestions 将AI评审建议应用到教案，并重新评审
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
	systemPrompt := fmt.Sprintf("你是一位专业的%s课教案优化专家。请根据评审建议改进教案内容，保持原有结构，重点改进被指出的问题。输出完整的改进后教案Markdown。", lp.Subject)

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

// checkPlanEditable 校验教案存在且调用者有权限且状态可编辑
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

// parseAIReply 解析AI回复，判断消息类型
func (s *LessonPlanGenService) parseAIReply(ctx context.Context, content string, lp *models.LessonPlan) *models.ConversationMessage {
	msg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		CreatedAt: time.Now(),
	}

	if strings.Contains(content, "## 教学目标") || strings.Contains(content, "# 教案") {
		msg.Type = models.ConvMsgTypeContent
		msg.Content = content
		return msg
	}

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

	msg.Type = models.ConvMsgTypeText
	msg.Content = content
	return msg
}

// ==================== 提示词构建函数 ====================

// buildSilentContext 将静默注入组件转为上下文文本
func buildSilentContext(groups []*models.MatchedComponentGroup) string {
	if len(groups) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n【背景参考资料（请纳入教学设计考量）】\n")
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("\n### %s\n", g.LibraryName))
		for _, c := range g.Components {
			sb.WriteString(fmt.Sprintf("- %s\n", c.DisplayLabel))
			if c.DesignLogic != "" {
				sb.WriteString(fmt.Sprintf("  参考逻辑：%s\n", c.DesignLogic))
			}
		}
	}
	return sb.String()
}

// buildDefaultSystemPrompt 默认系统提示词
func buildDefaultSystemPrompt(subject string) string {
	return fmt.Sprintf(`你是一位专业的%s课AI备课助手，帮助教师设计高质量教案。

工作原则：
1. 用友好的对话方式引导教师，每次只问2-3个问题
2. 提供具体可操作的建议，避免空泛描述
3. 遵循"学生为主体，教师为引导"的教学理念
4. 考虑AI课程的特殊性：技术体验真实性、批判性思维、工具可用性
5. 生成教案时使用Markdown格式，结构清晰

教案标准结构：
## 教学目标（三维：知识与技能/过程与方法/情感态度价值观）
## 教学重难点
## 课前准备
## 教学过程（含时间分配）
### 导入（5-8分钟）
### 主体活动（25-30分钟）
### 总结延伸（5-8分钟）
## 作业设计
## 板书设计`, subject)
}

// buildDefaultGenRules 默认生成规则
func buildDefaultGenRules() string {
	return `教学流程设计规则：
1. 导入环节：创设情境，激发兴趣，与学生生活经验关联
2. 主体活动：学生实操为主，教师讲解不超过总时间的30%
3. 总结延伸：引发深度思考，布置有价值的课后任务
4. 每个环节标注预计时间（分钟）
5. 活动描述要具体到"教师说什么/学生做什么"`
}

// buildDefaultReviewRules 默认评审规则
func buildDefaultReviewRules(subject string) string {
	base := `通用评审维度（各10分）：
T1 目标清晰度：三维目标是否具体、可观察、可评估
T2 结构完整性：环节是否齐全、时间分配是否合理
T3 学生参与度：学生主动参与vs被动接收，讲授占比
T4 评估对齐度：评估方式能否检验目标达成
T5 可操作性：活动步骤清晰、材料可获得`

	if subject == "AI" || subject == "人工智能" {
		base += `

学科维度（各10分）：
S1 技术体验真实性：学生是否真正操作了AI工具
S2 概念准确性：AI相关概念是否准确、适龄
S3 批判性思维：是否引导学生思考AI的局限
S4 跨学科连接：是否与已有学科知识关联
S5 工具可用性：所用AI工具是否免费、无需翻墙`
	}
	return base
}

// buildChatPrompt 组装对话上下文提示词
func buildChatPrompt(history []*models.ConversationMessage, userMsg *models.ConversationMessage, lp *models.LessonPlan) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【当前备课信息】\n学科：%s\n年级：%s\n课题：%s\n课时：%d分钟\n\n",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes))

	if lp.ContentMarkdown != "" {
		sb.WriteString("【已生成教案内容】\n")
		content := lp.ContentMarkdown
		if len(content) > 2000 {
			content = content[:2000] + "\n...(教案内容已截断)"
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	recentHistory := history
	if len(recentHistory) > 10 {
		recentHistory = recentHistory[len(recentHistory)-10:]
	}
	if len(recentHistory) > 0 {
		sb.WriteString("【对话记录】\n")
		for _, h := range recentHistory {
			role := "教师"
			if h.Role == models.ConvRoleAssistant {
				role = "AI助手"
			}
			sb.WriteString(fmt.Sprintf("%s：%s\n", role, h.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("教师：%s\n\nAI助手：", userMsg.Content))
	return sb.String()
}

// buildReviewSystemPrompt 评审专用系统提示词
func buildReviewSystemPrompt(subject string) string {
	return fmt.Sprintf(`你是一位经验丰富的%s课教案评审专家。
请对教案进行专业评审，输出格式严格按照以下JSON结构：

{
  "total_score": 8.5,
  "summary": "整体来说这份教案...(对话口吻，100-150字)",
  "good_points": ["做得好的1", "做得好的2"],
  "improvements": [
    {
      "id": "imp_1",
      "issue": "问题描述",
      "suggestion": "具体改进方案（对话口吻，如：试试把讲解时间从10分钟压缩到5分钟？）",
      "section": "涉及环节（可选）"
    }
  ],
  "dimensions": [
    {"code": "T1", "name": "目标清晰度", "score": 9, "comment": "...", "good": true}
  ]
}

评分原则：
- 总分为各维度平均分（0-10分制）
- 6分以下：明显问题  7-8分：可以改进  9-10分：优秀
- "做得好的"和"可以更好"各至少2-3条
- 所有描述使用对话口吻，如"这里可以试试..."而非"应该..."`, subject)
}

// buildReviewPrompt 组装评审用户提示词
func buildReviewPrompt(lp *models.LessonPlan, reviewRules string) string {
	return fmt.Sprintf("请评审以下%s课教案：\n\n**基本信息**\n年级：%s\n课题：%s\n课时：%d分钟\n\n**教案内容**\n%s\n\n**评审维度参考**\n%s",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes,
		lp.ContentMarkdown, reviewRules)
}

// buildOptimizePrompt 组装优化提示词
func buildOptimizePrompt(content string, suggestions []string) string {
	var sb strings.Builder
	sb.WriteString("请根据以下评审建议优化教案，保持Markdown格式，重点改进被指出的问题：\n\n")
	sb.WriteString("**改进建议：**\n")
	for i, s := range suggestions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	sb.WriteString("\n**原教案：**\n")
	sb.WriteString(content)
	sb.WriteString("\n\n**输出优化后的完整教案（Markdown格式）：**")
	return sb.String()
}

// buildDefaultOpeningMessage 构建默认开场消息（AI调用失败时降级）
func buildDefaultOpeningMessage(req *models.StartConversationRequest) *models.ConversationMessage {
	content := fmt.Sprintf("你好！我是你的AI备课助手 ✨\n\n我看到你要备一节**%s年级 %s课**，课题是「%s」，%d分钟课时。\n\n让我先了解一下你的学生情况，这样我能给你更精准的建议：\n\n1. 学生之前有没有接触过相关内容？\n2. 班级同学的整体接受能力怎么样？",
		req.Grade, req.Subject, req.Topic, req.DurationMinutes)
	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// genOpeningMessage AI生成开场白
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

// ==================== 解析辅助函数 ====================

// parseAIReviewResult 解析AI评审JSON结果
func parseAIReviewResult(content string) (*models.AIReviewResult, error) {
	jsonStr, ok := aiClient.ExtractJSON(content)
	if !ok {
		return nil, fmt.Errorf("AI回复中未找到JSON")
	}
	var result models.AIReviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析评审JSON失败: %w", err)
	}
	result.ReviewedAt = time.Now()
	return &result, nil
}

// buildFallbackReview 解析失败时的降级评审结果
func buildFallbackReview(rawContent string) *models.AIReviewResult {
	return &models.AIReviewResult{
		TotalScore: 7.0,
		Summary:    "AI评审已完成，请查看详细内容。",
		GoodPoints: []string{"教案结构基本完整"},
		Improvements: []models.AIReviewImprovement{{
			ID:         "imp_fallback",
			Issue:      "评审解析异常",
			Suggestion: rawContent,
		}},
		ReviewedAt: time.Now(),
	}
}

// appendReviewToHistory 将评审结果追加到历史
func appendReviewToHistory(oldHistory string, review *models.AIReviewResult) string {
	var history []models.AIReviewResult
	if err := json.Unmarshal([]byte(oldHistory), &history); err != nil {
		history = []models.AIReviewResult{}
	}
	history = append(history, *review)
	b, _ := json.Marshal(history)
	return string(b)
}

// extractSuggestionsByIDs 从评审结果中提取指定ID的建议文本
func extractSuggestionsByIDs(reviewResultJSON string, ids []string) []string {
	var result models.AIReviewResult
	if err := json.Unmarshal([]byte(reviewResultJSON), &result); err != nil {
		return nil
	}
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	var suggestions []string
	for _, imp := range result.Improvements {
		if len(ids) == 0 || idSet[imp.ID] {
			suggestions = append(suggestions, imp.Suggestion)
		}
	}
	return suggestions
}

// extractContentFromReply 从AI回复中提取教案内容
func extractContentFromReply(content string) string {
	if strings.Contains(content, "## 教学目标") || strings.Contains(content, "# 教案") {
		return content
	}
	return ""
}

// convertGroupsToConvComponents 将组件组转为对话消息中的组件格式
func convertGroupsToConvComponents(groups []*models.MatchedComponentGroup) []models.ConvComponent {
	var result []models.ConvComponent
	for _, g := range groups {
		for _, c := range g.Components {
			result = append(result, models.ConvComponent{
				ID:             c.ID,
				LibraryType:    g.LibraryType,
				DisplayLabel:   c.DisplayLabel,
				DesignLogic:    c.DesignLogic,
				ExampleSnippet: c.ExampleSnippet,
				QualityScore:   c.QualityScore,
				UsageCount:     c.UsageCount,
			})
		}
	}
	return result
}

// cleanComponentMarkers 清除AI回复中的组件标记
func cleanComponentMarkers(content string) string {
	content = strings.ReplaceAll(content, "【推荐组件】", "")
	content = strings.ReplaceAll(content, "推荐以下教学方案", "根据你的情况，我推荐以下教学方案")
	return strings.TrimSpace(content)
}

// formatSelectedOptions 将选项key转为可读文本
func formatSelectedOptions(keys []string, originalMsg string) string {
	if originalMsg != "" {
		return originalMsg
	}
	return "我选择：" + strings.Join(keys, "、")
}

// formatSelectedComponents 将选择的组件ID转为文本
func formatSelectedComponents(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("\n（已选择%d个教学组件）", len(ids))
}

// generateMsgID 生成消息ID
func generateMsgID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
