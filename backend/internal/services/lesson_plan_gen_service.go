package services

// lesson_plan_gen_service.go — 教案生成核心服务（主文件）
//
// 职责：
//   1. StartConversation  — 创建教案+阶段初始化+配方上下文注入/静默组件注入+发起AI开场白
//   2. Chat               — 处理教师输入→流式AI回复→SSE逐token推送
//   3. TriggerAIReview    — 触发AI质量评审（异步，SSE推送结果）
//   4. ApplyAISuggestions — 将AI建议应用到教案内容（优化+重新评审）
//   5. GetConversation    — 获取教案对话历史
//
// v74 重大优化：统一走阶段化流程，write/revise防重复生成
// v75 重大重构：去标签体系，AI输出直接推送
// v78 改动：独立场景配置 lessonPlanSceneCode
// v84 改动：分层记忆架构改造
//   - Chat() 改用 GetCurrentStageMessages 获取当前阶段消息（Working Memory）
//   - processChatStageAsync 使用 BuildStageChatPromptV2 构建分层上下文
//   - Episodic Memory 从 workshop_stage_outputs.narrative_output 获取
// v87 改动：AI教练集成
//   - processChatStageAsync 对话完成后检测停滞，超过3轮无进展插入教练建议
//   - 教练建议通过SSE推送给前端（使用现有LPSSEMessageDone事件类型）

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/utils"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/config"
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

// ==================== 常量定义 ====================

// lessonPlanSceneCode 教案生成场景代码，用于从ai_scene_configs获取独立模型配置
// v78新增：教案生成不再使用全局默认模型，改为独立场景配置
const lessonPlanSceneCode = "lesson_plan"

// ==================== 服务结构体 ====================

// LessonPlanGenService 教案生成服务
type LessonPlanGenService struct {
	cfg           interface{ GetAESKey() string }
	recipeService *RecipeService
	stageService  *WorkshopStageService
}

var lpGenLog = logger.WithModule("lp_gen")

// NewLessonPlanGenService 创建教案生成服务
func NewLessonPlanGenService(cfg interface{ GetAESKey() string }) *LessonPlanGenService {
	return &LessonPlanGenService{
		cfg:           cfg,
		recipeService: NewRecipeService(),
		stageService:  NewWorkshopStageService(),
	}
}

// ==================== 1. 开始备课会话 ====================

// StartConversation 创建教案+阶段初始化+配方上下文注入+发起AI开场白
//
// v74改动：统一走阶段化流程，无论有没有配方都初始化阶段
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
	if req.RecipeID != "" {
		lp.RecipeID = &req.RecipeID
	}

	if err := repository.CreateLessonPlan(ctx, lp); err != nil {
		return nil, nil, fmt.Errorf("创建教案失败: %w", err)
	}
	lpGenLog.Info("开始备课会话", "plan_id", lp.ID, "topic", req.Topic, "author", authorID, "recipe_id", req.RecipeID)

	// v74：统一走阶段化流程，不再区分有无配方
	recipeStagesConfig := ""
	if req.RecipeID != "" {
		recipe, err := repository.GetRecipeByID(ctx, req.RecipeID)
		if err == nil {
			recipeStagesConfig = recipe.StagesConfig
		}
	}

	snapshots, err := s.stageService.InitStagesForPlan(ctx, lp.ID, recipeStagesConfig, req.RecipeID)
	if err != nil {
		// 阶段初始化失败是严重错误，直接返回错误而非降级
		lpGenLog.Error("阶段初始化失败", "plan_id", lp.ID, "error", err)
		return nil, nil, fmt.Errorf("阶段初始化失败: %w", err)
	}

	lp.CurrentStage = snapshots[0].StageCode
	configJSON, _ := json.Marshal(snapshots)
	lp.StageConfig = string(configJSON)
	lpGenLog.Info("阶段初始化成功", "plan_id", lp.ID, "stages_count", len(snapshots), "first_stage", snapshots[0].StageCode)

	// 生成阶段化开场白
	var openingMsg *models.ConversationMessage
	openingMsg, err = s.genStageOpeningMessage(ctx, lp, snapshots)
	if err != nil {
		lpGenLog.Warn("阶段开场白生成失败，使用默认开场", "plan_id", lp.ID, "error", err)
		openingMsg = buildDefaultOpeningMessage(req)
	}

	// 推送阶段开始事件
	go func() {
		GlobalLPSSEHub.Broadcast(lp.ID, models.LPSSEEvent{
			EventType: models.LPSSEStageStarted,
			PlanID:    lp.ID,
			StageData: &models.StageEventData{
				StageCode:   snapshots[0].StageCode,
				StageName:   snapshots[0].StageName,
				StageOrder:  snapshots[0].StageOrder,
				TotalStages: len(snapshots),
			},
		})
	}()

	// 记录配方使用
	if req.RecipeID != "" {
		go func() {
			_ = repository.RecordRecipeUsage(context.Background(), req.RecipeID, lp.ID, authorID)
		}()
	}

	if err2 := s.appendMessage(ctx, lp.ID, openingMsg); err2 != nil {
		lpGenLog.Warn("写入开场消息失败", "plan_id", lp.ID, "error", err2)
	}

	return lp, openingMsg, nil
}

// genStageOpeningMessage 阶段模式下生成第一阶段的AI开场白
//
// v75改动：去掉 CleanStageMarkers 调用，AI不再输出标签
// v78改动：sceneCode改为lessonPlanSceneCode，使用独立场景配置
func (s *LessonPlanGenService) genStageOpeningMessage(
	ctx context.Context,
	lp *models.LessonPlan,
	snapshots []models.StageConfigSnapshot,
) (*models.ConversationMessage, error) {
	stageSystemPrompt, err := s.stageService.LoadStagePromptContext(ctx, lp, snapshots[0].StageCode)
	if err != nil {
		return nil, fmt.Errorf("加载阶段提示词失败: %w", err)
	}

	var stage *models.WorkshopStage
	stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, snapshots[0].StageCode)
	if err != nil {
		return nil, fmt.Errorf("加载阶段定义失败: %w", err)
	}

	userPrompt := BuildStageOpeningPrompt(lp, stage, snapshots[0].StageOrder, len(snapshots))

	// v78改动：传入lessonPlanSceneCode，使用教案生成独立场景配置
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("AI配置加载失败: %w", err)
	}

	result, err := aiClient.CallAI(aiCfg, stageSystemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI开场白生成失败: %w", err)
	}

	// v75/v82：AI不再输出标签，直接使用TrimSpace清理内容
	content := strings.TrimSpace(result.Content)

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

// buildSilentComponentContext 静默匹配背景类组件构建上下文（配方不存在时的兜底）
func (s *LessonPlanGenService) buildSilentComponentContext(ctx context.Context, subject string, grade string) string {
	silentGroups, _ := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject:       subject,
		GradeRange:    grade,
		InjectionMode: "silent",
		Limit:         3,
	})
	return buildSilentContext(silentGroups)
}

// ==================== 2. 对话轮次（流式SSE推送）====================

// Chat 处理教师输入，AI生成回复并通过SSE流式推送
//
// v74改动：统一走阶段化对话，移除旧模式判断
// v84改动：改用分层记忆加载当前阶段消息
func (s *LessonPlanGenService) Chat(
	ctx context.Context,
	req *models.LessonPlanChatRequest,
	callerID string,
) error {
	lp, err := s.checkPlanEditable(ctx, req.PlanID, callerID)
	if err != nil {
		return err
	}

	// v84改动：只加载当前阶段的对话消息（Working Memory）
	// 替代旧的 loadConversation（全量加载）
	currentStageMsgs, err := repository.GetCurrentStageMessages(ctx, lp.ID)
	if err != nil {
		lpGenLog.Warn("加载当前阶段消息失败，降级为空历史", "plan_id", lp.ID, "error", err)
		currentStageMsgs = []*models.ConversationMessage{}
	}

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

	// v84改动：传入当前阶段消息而非全量历史
	go func() {
		bgCtx := context.Background()
		s.processChatStageAsync(bgCtx, lp, userMsg, currentStageMsgs, req)
	}()

	return nil
}

// ==================== 2.1 阶段化对话（v84分层记忆改造 + v87教练集成）====================

// processChatStageAsync 阶段模式：异步处理AI流式回复
//
// v75重大重构：去标签，AI输出直接推送
// v78改动：独立场景配置
// v84改动：分层记忆架构
//   - history参数现在只包含当前阶段的消息（Working Memory）
//   - 新增Episodic Memory从workshop_stage_outputs获取
//   - 使用BuildStageChatPromptV2构建分层上下文
// v87改动：AI教练集成
//   - 对话完成后异步检测停滞
//   - 停滞时插入教练建议消息并通过SSE推送
func (s *LessonPlanGenService) processChatStageAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	currentStageMsgs []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
) {
	planID := lp.ID
	currentStage := lp.CurrentStage

	// 推送thinking状态
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
		MessageID: generateMsgID(),
	})

	// v78改动：传入lessonPlanSceneCode，使用教案生成独立场景配置
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		s.broadcastError(planID, "AI配置加载失败: "+err.Error())
		return
	}

	// 加载阶段系统提示词
	stageSystemPrompt, err := s.stageService.LoadStagePromptContext(ctx, lp, currentStage)
	if err != nil {
		lpGenLog.Warn("加载阶段提示词失败", "plan_id", planID, "stage", currentStage, "error", err)
		s.broadcastError(planID, "加载阶段配置失败，请刷新重试")
		return
	}

	// ===== v74保留：write阶段防重复生成 =====
	if currentStage == "write" {
		latestLP, freshErr := repository.GetLessonPlanByID(ctx, planID)
		if freshErr == nil && len(strings.TrimSpace(latestLP.ContentMarkdown)) > 2000 {
			contentLen := len(latestLP.ContentMarkdown)
			stageSystemPrompt += fmt.Sprintf(`

== 重要提示（系统级指令，最高优先级）==
教案正文已经成功生成并保存（共%d字符），右侧面板已经展示给了老师。
请注意以下规则：
1. 不要再重新输出完整教案。教案已经保存好了。
2. 如果老师说"输出""生成""写出来"等话，请告诉老师教案已经生成完毕并显示在右侧面板，问老师是否需要修改某个部分。
3. 如果老师要求修改教案的某个具体部分，可以针对性地讨论修改方案，但不要输出完整教案。
4. 你现在的角色是帮助老师确认教案是否满意、讨论是否需要局部调整。
5. 如果老师确认教案没问题，建议老师点击"完成本阶段"按钮进入下一阶段（AI评审）。`, contentLen)

			lpGenLog.Info("write阶段已有教案内容，注入防重复生成指令",
				"plan_id", planID,
				"stage", currentStage,
				"content_len", contentLen,
			)
		}
	}

	// ===== v84新增：构建Episodic Memory =====
	// 从之前阶段的workshop_stage_outputs中获取narrative_output摘要
	allOutputs, _ := repository.ListStageOutputs(ctx, planID)
	var priorOutputs []*models.WorkshopStageOutput
	for _, out := range allOutputs {
		if out.StageCode == currentStage {
			break
		}
		priorOutputs = append(priorOutputs, out)
	}
	episodicSummary := repository.BuildEpisodicSummaryFromOutputs(priorOutputs)

	// v84改动：使用BuildStageChatPromptV2构建分层上下文
	userPrompt := BuildStageChatPromptV2(lp, currentStageMsgs, episodicSummary, userMsg)

	lpGenLog.Info("v84分层记忆上下文构建完成",
		"plan_id", planID,
		"stage", currentStage,
		"working_msgs", len(currentStageMsgs),
		"episodic_len", len(episodicSummary),
		"prior_stages", len(priorOutputs),
	)

	// ===== v75：流式推送——直接推送所有内容，不过滤标签 =====
	chunkCount := 0
	var fullContent strings.Builder

	result, err := aiClient.CallAIStream(aiCfg, stageSystemPrompt, userPrompt, func(chunk string) error {
		if strings.TrimSpace(chunk) == "" {
			return nil
		}
		chunkCount++
		fullContent.WriteString(chunk)

		// v75：直接推送chunk给用户，不经过filter
		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType: models.LPSSEChunk,
			PlanID:    planID,
			Chunk:     chunk,
		})
		return nil
	}, nil)
	if err != nil {
		s.broadcastError(planID, "AI回复失败: "+err.Error())
		return
	}

	rawContent := result.Content
	if rawContent == "" {
		rawContent = fullContent.String()
	}

	// ===== v75：从自然语言中提取结构化数据 =====
	structuredJSON, narrative, hasContent := ExtractStructuredFromNaturalReply(currentStage, rawContent)
	if hasContent {
		// 保存阶段产出物
		if err := s.stageService.SaveStageOutput(ctx, planID, currentStage, structuredJSON, narrative, result.ModelUsed, result.TokensUsed); err != nil {
			lpGenLog.Warn("保存阶段产出物失败", "plan_id", planID, "stage", currentStage, "error", err)
		} else {
			lpGenLog.Info("阶段产出物已保存", "plan_id", planID, "stage", currentStage)
		}

		// 处理阶段副作用
		s.handleStageOutputSideEffects(ctx, planID, lp, currentStage, structuredJSON, rawContent)

		// 推送产出物事件
		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType: models.LPSSEStageOutput,
			PlanID:    planID,
			StageData: &models.StageEventData{
				StageCode: currentStage,
				StageName: stageCodeToName(currentStage),
			},
		})
	}

	// 构造AI回复消息并保存
	aiReply := s.parseAIReply(ctx, rawContent, lp)

	if err := s.appendMessage(ctx, planID, aiReply); err != nil {
		lpGenLog.Warn("写入AI消息失败", "plan_id", planID, "error", err)
	}

	// 推送消息完成事件
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEMessageDone,
		PlanID:    planID,
		MessageID: aiReply.ID,
		Message:   aiReply,
	})

	lpGenLog.Info("AI对话流式回复完成（v84分层记忆）",
		"plan_id", planID,
		"stage", currentStage,
		"tokens", result.TokensUsed,
		"latency_ms", result.LatencyMs,
		"chunks", chunkCount,
		"has_content", hasContent,
		"working_msgs", len(currentStageMsgs),
	)

	// ===== v87新增：对话完成后异步检测停滞，插入教练建议 =====
	go s.checkAndInsertCoachAdvice(ctx, planID, currentStage)
}

// ==================== v87新增：停滞检测+教练建议插入 ====================

// checkAndInsertCoachAdvice 对话完成后检测停滞，插入教练建议
//
// 在每轮AI回复完成后异步调用。如果检测到对话停滞（连续3轮无实质进展），
// 自动插入一条教练建议消息并通过SSE推送给前端。
//
// 设计决策：
//   - 异步执行，不影响AI回复的主路径延迟
//   - 教练建议使用预设模板（不调AI），确保零额外成本
//   - 建议消息作为assistant角色插入对话历史，前端正常展示
//   - 短暂延迟（500ms）确保AI回复先到达前端
func (s *LessonPlanGenService) checkAndInsertCoachAdvice(ctx context.Context, planID string, stageCode string) {
	// 短暂延迟，确保AI回复已经完整推送到前端
	time.Sleep(500 * time.Millisecond)

	// 检测停滞
	stagnation := DetectStagnation(ctx, planID, stageCode)
	if stagnation == nil || !stagnation.IsStagnant {
		return
	}

	// 生成教练建议
	suggestion := GenerateCoachSuggestion(stagnation)
	if suggestion == "" {
		return
	}

	// 构建教练建议消息
	coachMsg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   suggestion,
		CreatedAt: time.Now(),
	}

	// 保存到对话历史
	if err := s.appendMessage(ctx, planID, coachMsg); err != nil {
		lpGenLog.Warn("v87教练建议-写入消息失败", "plan_id", planID, "error", err)
		return
	}

	// 通过SSE推送给前端
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEMessageDone,
		PlanID:    planID,
		MessageID: coachMsg.ID,
		Message:   coachMsg,
	})

	lpGenLog.Info("v87教练建议已插入",
		"plan_id", planID, "stage", stageCode,
		"user_rounds", stagnation.ConsecutiveRounds,
	)
}

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

// ==================== 3. 触发AI评审 ====================

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

	result, err := aiClient.CallAI(aiCfg, systemPrompt, reviewPrompt, nil)
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

	result, err := aiClient.CallAI(aiCfg, systemPrompt, optimizePrompt, nil)
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

// GetConversation 获取教案对话历史
// 注意：此API仍返回全量对话（前端展示需要），分层记忆只影响AI上下文构建
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

// checkPlanEditable 检查教案是否存在、归属正确、且处于可编辑状态
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
		lp.Status != models.LPStatusRevision &&
		lp.Status != models.LPStatusDeveloping {
		return nil, ErrLPGenNotEditable
	}
	return lp, nil
}

// appendMessage 追加消息到教案对话历史
func (s *LessonPlanGenService) appendMessage(ctx context.Context, planID string, msg *models.ConversationMessage) error {
	return repository.AppendConversationMessage(ctx, planID, msg)
}

// loadConversation 加载教案全量对话历史（前端展示用，不用于AI上下文）
func (s *LessonPlanGenService) loadConversation(ctx context.Context, planID string) ([]*models.ConversationMessage, error) {
	return repository.GetConversationLog(ctx, planID)
}

// resolveTemplateForGen 解析教案生成模板（暂保留，未来可能恢复使用）
func (s *LessonPlanGenService) resolveTemplateForGen(ctx context.Context, templateID string, subject string) (systemPrompt string, genRules string) {
	if templateID != "" {
		resolved, err := repository.ResolvePromptTemplateChain(ctx, templateID)
		if err == nil {
			return resolved.SystemPrompt, resolved.GenerationRules
		}
	}
	return buildDefaultSystemPrompt(subject), buildDefaultGenRules()
}

// resolveTemplateForReview 解析评审模板
func (s *LessonPlanGenService) resolveTemplateForReview(ctx context.Context, subject string) (systemPrompt string, reviewRules string) {
	return buildReviewSystemPrompt(subject), buildDefaultReviewRules(subject)
}

// broadcastError 通过SSE推送错误消息给前端
func (s *LessonPlanGenService) broadcastError(planID string, msg string) {
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEError,
		PlanID:    planID,
		Error:     msg,
	})
}

// parseAIReply 解析AI回复，判断消息类型（普通文本/教案内容/组件推荐）
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

	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, nil)
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

// ==================== v89新增：自动教案索引触发 ====================

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
	indexText, err := liService.CompressLessonIndex(fullText)
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
