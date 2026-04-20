package services

// workshop_stage_service.go — 阶段化备课工坊核心服务
//
// 包含：阶段查询、进度管理、阶段前进/跳过/回退/切换/重启、产出物保存、提示词上下文加载
//
// v76拆分：组件推荐+自定义阶段CRUD 移至 workshop_stage_components.go
// v77改动：ResetStage 改为按阶段分隔符截断对话
// v84改动：AdvanceStage 完成阶段时自动生成Episodic摘要
// v84拆分：合并算法+InitStagesForPlan+辅助函数 移至 workshop_stage_merge.go
// v87改动：AdvanceStage 阶段过渡前异步调用LLM质量评估
//   - WorkshopStageService新增aesKey字段（用于AI配置获取）
//   - SetAESKey方法注入密钥（由routes层调用）
//   - advanceStageWithComponents中新增LLM评估+SSE推送评估结果
// v110(TE-DNA 3.0 P0 STEP 3)改动:
//   - 新增 LoadStagePromptContextV2(ctx, lp, stageCode, assistantPrompt) 方法
//   - 原 LoadStagePromptContext 保持签名不变,内部转调 V2 传空 assistantPrompt
//   - V2 调用 BuildStageSystemPromptV2 透传助手 prompt,用于替换第4层阶段角色
//   - 其他层(配方/产出物/组件/教案结构/对话规范)完全保留,不受助手影响

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrStageNotInitialized = errors.New("教案尚未初始化阶段配置")
	ErrStageAlreadyFirst   = errors.New("已经是第一个阶段，无法回退")
	ErrStageAlreadyLast    = errors.New("已经是最后一个阶段")
	ErrStageNotSkippable   = errors.New("当前阶段不可跳过")
	ErrStageInvalidTarget  = errors.New("目标阶段不存在")
	ErrCustomStageLimit    = errors.New("自定义阶段数量已达上限（最多10个）")
)

// autoTriggerStages 进入后自动触发Chat的阶段及对应的触发消息
var autoTriggerStages = map[string]string{
	"design": "我们进入教学设计阶段了。请先简要介绍你是谁、你拿到了哪些前序阶段的分析成果，然后告诉我接下来你会带我做什么。用友好的口吻，不超过200字。",
	"write":  "我们进入教案撰写阶段了。请先简要介绍你是谁、你拿到了哪些前序阶段的设计方案，然后告诉我接下来你会怎么帮我写教案。用友好的口吻，不超过200字。",
	"review": "请对上一阶段完成的教案进行全面专业评审，直接输出评审报告，包含各维度评分和改进建议。",
	"revise": "我们进入修订定稿阶段了。请先简要介绍你是谁、你拿到了评审报告中的哪些改进建议，然后告诉我接下来你会怎么帮我修订教案。用友好的口吻，不超过200字。",
}

// ==================== 服务结构体 ====================

// WorkshopStageService 阶段化备课工坊服务
// v87新增：aesKey字段用于AI教练LLM评估获取配置
type WorkshopStageService struct {
	recipeService *RecipeService
	genService    interface {
		Chat(ctx context.Context, req *models.LessonPlanChatRequest, callerID string) error
	}
	aesKey string // v87新增：AES密钥（用于LLM教练评估获取AI配置）
}

var wsLog = logger.WithModule("workshop_stage")

// NewWorkshopStageService 创建阶段服务实例
func NewWorkshopStageService() *WorkshopStageService {
	return &WorkshopStageService{
		recipeService: NewRecipeService(),
	}
}

// SetGenService 注入生成服务（由routes层调用，避免循环依赖）
func (s *WorkshopStageService) SetGenService(gs interface {
	Chat(ctx context.Context, req *models.LessonPlanChatRequest, callerID string) error
}) {
	s.genService = gs
}

// SetAESKey 注入AES密钥（由routes层调用，用于LLM教练评估获取AI配置）
// v87新增
func (s *WorkshopStageService) SetAESKey(key string) {
	s.aesKey = key
}

// ==================== 1. 获取系统默认阶段 ====================

func (s *WorkshopStageService) GetDefaultStages(ctx context.Context) (*models.DefaultStagesResponse, error) {
	stages, err := repository.GetSystemDefaultStages(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取默认阶段失败: %w", err)
	}
	var items []*models.DefaultStageItem
	for _, st := range stages {
		items = append(items, &models.DefaultStageItem{
			StageCode:      st.StageCode,
			StageName:      st.StageName,
			StageOrder:     st.StageOrder,
			AIRole:         st.AIRole,
			GateMode:       st.GateMode,
			Skippable:      st.Skippable,
			ComponentTypes: st.ComponentTypes,
		})
	}
	if items == nil {
		items = []*models.DefaultStageItem{}
	}
	return &models.DefaultStagesResponse{Stages: items}, nil
}

// ==================== 2. 获取教案阶段进度 ====================

func (s *WorkshopStageService) GetStageStatus(ctx context.Context, lessonPlanID string, callerID string) (*models.StageStatusResponse, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}
	if len(snapshots) == 0 {
		return nil, ErrStageNotInitialized
	}
	outputs, _ := repository.ListStageOutputs(ctx, lessonPlanID)
	outputMap := make(map[string]*models.WorkshopStageOutput)
	for _, out := range outputs {
		outputMap[out.StageCode] = out
	}
	var items []*models.StageProgressItem
	for _, snap := range snapshots {
		item := &models.StageProgressItem{
			StageCode: snap.StageCode, StageName: snap.StageName, StageOrder: snap.StageOrder,
			AIRole: snap.AIRole, GateMode: snap.GateMode, Skippable: snap.Skippable,
			Status: "pending", IsCustom: snap.IsCustom,
		}
		if out, ok := outputMap[snap.StageCode]; ok {
			item.Status = out.Status
			item.HasOutput = out.StructuredOutput != "" && out.StructuredOutput != "{}"
			item.CompletedAt = out.CompletedAt
		}
		items = append(items, item)
	}
	return &models.StageStatusResponse{
		CurrentStage: lp.CurrentStage, TotalStages: len(snapshots), Stages: items,
	}, nil
}

// ==================== 3. 获取阶段产出物 ====================

func (s *WorkshopStageService) GetStageOutput(ctx context.Context, lessonPlanID string, stageCode string, callerID string) (*models.StageOutputResponse, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	out, err := repository.GetStageOutput(ctx, lessonPlanID, stageCode)
	if err != nil {
		return nil, err
	}
	return &models.StageOutputResponse{
		StageCode: out.StageCode, StageName: stageCodeToName(out.StageCode),
		StructuredOutput: out.StructuredOutput, NarrativeOutput: out.NarrativeOutput,
		Status: out.Status, ModelUsed: out.ModelUsed, TokensUsed: out.TokensUsed,
	}, nil
}

// ==================== 4. 进入下一阶段 ====================

func (s *WorkshopStageService) AdvanceStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(ctx, lessonPlanID, targetStageCode, callerID, nil)
}

// AdvanceStageWithComponents 进入下一阶段（带用户选中的组件ID）
func (s *WorkshopStageService) AdvanceStageWithComponents(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string, selectedComponentIDs []string) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(ctx, lessonPlanID, targetStageCode, callerID, selectedComponentIDs)
}

func (s *WorkshopStageService) advanceStageWithComponents(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string, selectedComponentIDs []string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}
	var targetIdx int
	if targetStageCode != "" {
		targetIdx = findStageIndex(snapshots, targetStageCode)
		if targetIdx == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else {
		targetIdx = currentIdx + 1
		if targetIdx >= len(snapshots) {
			return nil, ErrStageAlreadyLast
		}
	}

	// v87新增：阶段过渡前异步调用LLM评估产出物质量
	// 评估结果通过SSE推送给前端（建议性质，不阻塞阶段过渡）
	if s.aesKey != "" {
		go s.asyncLLMEvaluateAndBroadcast(ctx, lessonPlanID, lp.CurrentStage)
	}

	// v84新增：完成当前阶段前，自动生成Episodic摘要保存到narrative_output
	s.generateAndSaveEpisodicSummary(ctx, lessonPlanID, lp.CurrentStage)

	_ = repository.CompleteStageOutput(ctx, lessonPlanID, lp.CurrentStage, "[]")
	targetStage := snapshots[targetIdx]
	// v76：写入用户选中的组件ID
	initialStructured := "{}"
	if len(selectedComponentIDs) > 0 {
		data := map[string]interface{}{"selected_component_ids": selectedComponentIDs}
		if b, err := json.Marshal(data); err == nil {
			initialStructured = string(b)
		}
		wsLog.Info("用户为阶段选择了组件", "plan_id", lessonPlanID, "stage", targetStage.StageCode, "component_count", len(selectedComponentIDs))
	}
	output := &models.WorkshopStageOutput{
		LessonPlanID: lessonPlanID, StageCode: targetStage.StageCode, StageOrder: targetStage.StageOrder,
		StructuredOutput: initialStructured, NarrativeOutput: "", ConversationSnapshot: "[]", Status: models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn("创建阶段产出记录失败（可能已存在）", "error", err)
	}
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}
	wsLog.Info("进入下一阶段", "plan_id", lessonPlanID, "from", lp.CurrentStage, "to", targetStage.StageCode)

	// v77b：持久化阶段分隔符到对话记录
	sepContent := "__STAGE_SEP__" + targetStage.StageName + "__" + targetStage.AIRole
	sepMsg := &models.ConversationMessage{
		ID:        fmt.Sprintf("stage_sep_%s_%d", targetStage.StageCode, time.Now().UnixMilli()),
		Role:      "system",
		Type:      "text",
		Content:   sepContent,
		CreatedAt: time.Now(),
	}
	if err := repository.AppendConversationMessage(ctx, lessonPlanID, sepMsg); err != nil {
		wsLog.Warn("持久化阶段分隔符失败", "plan_id", lessonPlanID, "stage", targetStage.StageCode, "error", err)
	}

	// 自动触发AI开场白
	if triggerMsg, needsTrigger := autoTriggerStages[targetStage.StageCode]; needsTrigger && s.genService != nil {
		wsLog.Info("自动触发阶段AI开场白", "plan_id", lessonPlanID, "stage", targetStage.StageCode)
		go func() {
			time.Sleep(100 * time.Millisecond)
			bgCtx := context.Background()
			req := &models.LessonPlanChatRequest{PlanID: lessonPlanID, Message: triggerMsg}
			if err := s.genService.Chat(bgCtx, req, callerID); err != nil {
				wsLog.Warn("自动触发阶段AI开场白失败", "plan_id", lessonPlanID, "stage", targetStage.StageCode, "error", err)
			}
		}()
	}
	return &targetStage, nil
}

// ==================== v87新增：异步LLM评估并推送结果 ====================

// asyncLLMEvaluateAndBroadcast 异步调用LLM评估阶段产出物质量，并通过SSE推送结果
//
// 设计决策：
//   - 异步执行，不阻塞AdvanceStage主流程（用户不会等待评估完成才进入下一阶段）
//   - 评估结果通过SSE的LPSSEStageOutput事件推送，前端可选择展示
//   - 如果评估失败（AI调用失败等），静默降级，不影响用户流程
//   - 使用stage_coach场景（Haiku模型），确保低成本
func (s *WorkshopStageService) asyncLLMEvaluateAndBroadcast(ctx context.Context, lessonPlanID string, stageCode string) {
	// 使用background context，因为原始请求可能已经返回
	bgCtx := context.Background()

	evalResult, err := LLMEvaluateStageQuality(bgCtx, s.aesKey, lessonPlanID, stageCode)
	if err != nil {
		wsLog.Warn("v87 LLM阶段评估失败",
			"plan_id", lessonPlanID, "stage", stageCode, "error", err)
		return
	}

	wsLog.Info("v87 LLM阶段评估完成",
		"plan_id", lessonPlanID, "stage", stageCode,
		"score", evalResult.OverallScore, "qualified", evalResult.IsQualified,
		"suggestion", coachTruncateStr(evalResult.Suggestion, 50),
	)

	// 如果评估结果不合格且有建议，通过SSE推送给前端
	if !evalResult.IsQualified && evalResult.Suggestion != "" {
		// 构建教练评估消息，插入到对话历史
		evalMsg := &models.ConversationMessage{
			ID:        fmt.Sprintf("coach_eval_%s_%d", stageCode, time.Now().UnixMilli()),
			Role:      models.ConvRoleAssistant,
			Type:      models.ConvMsgTypeText,
			Content:   fmt.Sprintf("📋 阶段评估（%s，%d分）：%s", stageCodeToName(stageCode), evalResult.OverallScore, evalResult.Suggestion),
			CreatedAt: time.Now(),
		}

		// 保存到对话历史
		if appendErr := repository.AppendConversationMessage(bgCtx, lessonPlanID, evalMsg); appendErr != nil {
			wsLog.Warn("v87 LLM评估消息写入失败", "plan_id", lessonPlanID, "error", appendErr)
		}

		// 通过SSE推送
		GlobalLPSSEHub.Broadcast(lessonPlanID, models.LPSSEEvent{
			EventType: models.LPSSEMessageDone,
			PlanID:    lessonPlanID,
			MessageID: evalMsg.ID,
			Message:   evalMsg,
		})
	}
}

// ==================== v84新增：生成并保存Episodic摘要 ====================

// generateAndSaveEpisodicSummary 在阶段完成时自动生成摘要并保存到narrative_output
//
// v84新增：分层记忆架构的核心环节
// 在 AdvanceStage/SkipStage 完成当前阶段之前调用
// 从当前阶段对话中提取结构化摘要，保存到 workshop_stage_outputs.narrative_output
func (s *WorkshopStageService) generateAndSaveEpisodicSummary(ctx context.Context, lessonPlanID string, stageCode string) {
	// 获取当前阶段的对话消息
	currentMsgs, err := repository.GetCurrentStageMessages(ctx, lessonPlanID)
	if err != nil {
		wsLog.Warn("生成Episodic摘要-获取当前阶段消息失败", "plan_id", lessonPlanID, "stage", stageCode, "error", err)
		return
	}

	// 获取当前阶段的structured_output（摘要生成可能需要参考）
	stageOutput, err := repository.GetStageOutput(ctx, lessonPlanID, stageCode)
	if err != nil {
		wsLog.Warn("生成Episodic摘要-获取阶段产出失败", "plan_id", lessonPlanID, "stage", stageCode, "error", err)
		stageOutput = &models.WorkshopStageOutput{}
	}

	// 检查是否需要生成摘要
	existingNarrative := strings.TrimSpace(stageOutput.NarrativeOutput)
	if len([]rune(existingNarrative)) > 100 {
		wsLog.Info("Episodic摘要-已有较长narrative，跳过生成",
			"plan_id", lessonPlanID, "stage", stageCode, "existing_len", len(existingNarrative))
		return
	}

	// 生成摘要
	summary := GenerateStageSummary(stageCode, currentMsgs, stageOutput.StructuredOutput)
	if summary == "" {
		wsLog.Info("Episodic摘要-生成结果为空，跳过保存", "plan_id", lessonPlanID, "stage", stageCode)
		return
	}

	// 保存摘要到narrative_output
	if err := repository.UpdateStageNarrativeOutput(ctx, lessonPlanID, stageCode, summary); err != nil {
		wsLog.Warn("Episodic摘要-保存失败", "plan_id", lessonPlanID, "stage", stageCode, "error", err)
		return
	}

	wsLog.Info("Episodic摘要-生成并保存成功",
		"plan_id", lessonPlanID, "stage", stageCode, "summary_len", len(summary))
}

// ==================== 5. 跳过当前阶段 ====================

func (s *WorkshopStageService) SkipStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}
	if !snapshots[currentIdx].Skippable {
		return nil, ErrStageNotSkippable
	}

	// v84新增：跳过前也生成摘要
	s.generateAndSaveEpisodicSummary(ctx, lessonPlanID, lp.CurrentStage)

	_ = repository.SkipStageOutput(ctx, lessonPlanID, lp.CurrentStage)
	var targetIdx int
	if targetStageCode != "" {
		targetIdx = findStageIndex(snapshots, targetStageCode)
		if targetIdx == -1 {
			return nil, ErrStageInvalidTarget
		}
	} else {
		targetIdx = currentIdx + 1
		if targetIdx >= len(snapshots) {
			return nil, ErrStageAlreadyLast
		}
	}
	targetStage := snapshots[targetIdx]
	output := &models.WorkshopStageOutput{
		LessonPlanID: lessonPlanID, StageCode: targetStage.StageCode, StageOrder: targetStage.StageOrder,
		StructuredOutput: "{}", NarrativeOutput: "", ConversationSnapshot: "[]", Status: models.StageOutputInProgress,
	}
	if err := repository.CreateStageOutput(ctx, output); err != nil {
		wsLog.Warn("创建阶段产出记录失败（可能已存在）", "error", err)
	}
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("更新当前阶段失败: %w", err)
	}
	wsLog.Info("跳过阶段", "plan_id", lessonPlanID, "skipped", lp.CurrentStage, "to", targetStage.StageCode)

	// v77b：持久化阶段分隔符
	sepContent := "__STAGE_SEP__" + targetStage.StageName + "__" + targetStage.AIRole
	skipSepMsg := &models.ConversationMessage{
		ID:        fmt.Sprintf("stage_sep_%s_%d", targetStage.StageCode, time.Now().UnixMilli()),
		Role:      "system",
		Type:      "text",
		Content:   sepContent,
		CreatedAt: time.Now(),
	}
	_ = repository.AppendConversationMessage(ctx, lessonPlanID, skipSepMsg)

	if triggerMsg, needsTrigger := autoTriggerStages[targetStage.StageCode]; needsTrigger && s.genService != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			bgCtx := context.Background()
			req := &models.LessonPlanChatRequest{PlanID: lessonPlanID, Message: triggerMsg}
			if err := s.genService.Chat(bgCtx, req, callerID); err != nil {
				wsLog.Warn("跳过后自动触发AI开场白失败", "plan_id", lessonPlanID, "error", err)
			}
		}()
	}
	return &targetStage, nil
}

// ==================== 6. 回退到上一阶段 ====================

func (s *WorkshopStageService) BackStage(ctx context.Context, lessonPlanID string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	snapshots, currentIdx, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}
	if currentIdx <= 0 {
		return nil, ErrStageAlreadyFirst
	}
	targetStage := snapshots[currentIdx-1]
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStage.StageCode); err != nil {
		return nil, fmt.Errorf("回退阶段失败: %w", err)
	}
	wsLog.Info("回退阶段", "plan_id", lessonPlanID, "from", lp.CurrentStage, "to", targetStage.StageCode)
	return &targetStage, nil
}

// ==================== 7. 切换到指定阶段（不动产出物、不清对话）====================

// SwitchToStage 切换到指定阶段继续对话
//
// 与ResetStage不同：不清产出物、不清对话、不触发AI开场白
// 只更新current_stage，让用户可以继续在该阶段聊天
func (s *WorkshopStageService) SwitchToStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, ErrLPGenPlanNotFound
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	snapshots, _, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}
	targetIdx := findStageIndex(snapshots, targetStageCode)
	if targetIdx == -1 {
		return nil, ErrStageInvalidTarget
	}
	targetStage := snapshots[targetIdx]
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStageCode); err != nil {
		return nil, fmt.Errorf("切换阶段失败: %w", err)
	}
	wsLog.Info("切换到指定阶段", "plan_id", lessonPlanID, "from", lp.CurrentStage, "to", targetStageCode)
	return &targetStage, nil
}

// ==================== 8. 保存阶段产出物 ====================

func (s *WorkshopStageService) SaveStageOutput(ctx context.Context, lessonPlanID string, stageCode string, structuredJSON string, narrative string, modelUsed string, tokensUsed int) error {
	return repository.UpdateStageOutputContent(ctx, lessonPlanID, stageCode, structuredJSON, narrative, modelUsed, tokensUsed)
}

// ==================== 9. 加载阶段提示词上下文 ====================

// LoadStagePromptContext 加载阶段提示词上下文(向后兼容版,v110 前原签名不变)
// 对应场景:未选择 AI 助手的原有调用路径(StartConversation 开场白等)
//
// v110 改造:内部转调 V2,传空 assistantPrompt 走原行为
func (s *WorkshopStageService) LoadStagePromptContext(ctx context.Context, lp *models.LessonPlan, stageCode string) (string, error) {
	return s.LoadStagePromptContextV2(ctx, lp, stageCode, "")
}

// LoadStagePromptContextV2 v110(TE-DNA 3.0 P0 STEP 3)新增:支持 AI 助手 full_prompt 注入的版本
//
// assistantPrompt 语义:
//   - 空字符串 → 完全走原行为(使用 stage.SystemPrompt + 变体段)
//   - 非空字符串 → 整段替换第 4 层(阶段角色+变体段),其他层(配方/产出物/组件/教案结构/对话规范)保持不变
//
// 调用场景:
//   - StartConversation 开场白路径:恒定传空(老师还没机会选助手)
//   - processChatStageAsync 对话路径:老师选中助手时传 full_prompt,否则传空
func (s *WorkshopStageService) LoadStagePromptContextV2(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
	assistantPrompt string,
) (string, error) {
	// 判断是否为自定义阶段(来自配方 recipe_stages 表)
	isCustomStage := false
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal([]byte(lp.StageConfig), &snapshots) == nil {
			for _, snap := range snapshots {
				if snap.StageCode == stageCode && snap.IsCustom {
					isCustomStage = true
					break
				}
			}
		}
	}

	// 加载阶段定义(优先级:自定义阶段 > 配方覆盖 > 系统默认)
	var stage *models.WorkshopStage
	var err error
	if isCustomStage && lp.RecipeID != nil && *lp.RecipeID != "" {
		stage, err = repository.GetRecipeStageByCode(ctx, *lp.RecipeID, stageCode)
		if err != nil {
			wsLog.Warn("自定义阶段加载失败，尝试系统默认", "stage_code", stageCode, "error", err)
			stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
		}
	} else if lp.RecipeID != nil && *lp.RecipeID != "" {
		stage, err = repository.GetStageByCode(ctx, models.StageSourceRecipe, stageCode)
		if err != nil {
			stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
		}
	} else {
		stage, err = repository.GetStageByCode(ctx, models.StageSourceSystem, stageCode)
	}
	if err != nil {
		return "", fmt.Errorf("加载阶段定义失败: %w", err)
	}

	// 加载配方(用于全局上下文+promptMode+lessonStructure)
	var recipe *models.TeachingRecipe
	promptMode := models.PromptModeGuided
	lessonStructure := ""
	if lp.RecipeID != nil && *lp.RecipeID != "" {
		recipe, _ = repository.GetRecipeByID(ctx, *lp.RecipeID)
		if recipe != nil {
			if recipe.PromptMode != "" {
				promptMode = recipe.PromptMode
			}
			if recipe.LessonStructure != "" && recipe.LessonStructure != "[]" {
				lessonStructure = recipe.LessonStructure
			}
		}
	}

	// 阶段级别的 promptMode 覆盖(StageConfig 快照中的配置优先)
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal([]byte(lp.StageConfig), &snapshots) == nil {
			for _, snap := range snapshots {
				if snap.StageCode == stageCode && snap.PromptModeOverride != "" {
					promptMode = snap.PromptModeOverride
					break
				}
			}
		}
	}

	// 加载前序阶段产出物
	allOutputs, _ := repository.ListStageOutputs(ctx, lp.ID)
	var priorOutputs []*models.WorkshopStageOutput
	for _, out := range allOutputs {
		if out.StageCode == stageCode {
			break
		}
		priorOutputs = append(priorOutputs, out)
	}

	// 读取用户为当前阶段选中的组件 ID
	selectedCompIDs := s.getSelectedComponentIDsFromOutput(ctx, lp.ID, stageCode)
	if len(selectedCompIDs) > 0 {
		wsLog.Info("检测到用户选中的阶段组件", "plan_id", lp.ID, "stage", stageCode, "selected_count", len(selectedCompIDs))
	}

	// v110 核心改动:调用 V2 版本透传 assistantPrompt
	// 为空时 V2 内部走原逻辑(完全等价于 V1);非空时替换第4层阶段角色
	return BuildStageSystemPromptV2(
		ctx, stage, recipe, priorOutputs,
		lp.Subject, lp.Grade, promptMode, lessonStructure, selectedCompIDs,
		assistantPrompt,
	), nil
}

// ==================== 10. 重启指定阶段 ====================

// ResetStage 重启指定阶段（v77改进：按阶段分隔符截断对话记录，保留之前历史）
func (s *WorkshopStageService) ResetStage(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string) (*models.StageConfigSnapshot, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, ErrLPGenPlanNotFound
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	snapshots, _, err := s.resolveStages(lp)
	if err != nil {
		return nil, err
	}

	targetIdx := findStageIndex(snapshots, targetStageCode)
	if targetIdx == -1 {
		return nil, ErrStageInvalidTarget
	}

	targetStage := snapshots[targetIdx]

	// 重置目标阶段产出物
	if err := repository.ResetStageOutput(ctx, lessonPlanID, targetStageCode); err != nil {
		wsLog.Warn("重置阶段产出失败", "plan_id", lessonPlanID, "stage", targetStageCode, "error", err)
	}

	// 删除目标阶段之后的所有产出物
	if err := repository.DeleteStageOutputsAfter(ctx, lessonPlanID, targetStage.StageOrder); err != nil {
		wsLog.Warn("删除后续阶段产出失败", "plan_id", lessonPlanID, "stage", targetStageCode, "error", err)
	}

	// 如果是 write/revise 阶段，清空教案正文
	if targetStageCode == "write" || targetStageCode == "revise" {
		_ = repository.UpdateLessonPlanContent(ctx, lessonPlanID, lp.Title, "", "{}", lp.DurationMinutes)
		wsLog.Info("重启write/revise阶段，清空教案正文", "plan_id", lessonPlanID)
	}

	// 更新 current_stage
	if err := repository.UpdateLessonPlanCurrentStage(ctx, lessonPlanID, targetStageCode); err != nil {
		return nil, fmt.Errorf("重置当前阶段失败: %w", err)
	}

	// 按阶段分隔符截断对话记录
	stageCodeToNameMap := make(map[string]string)
	for _, snap := range snapshots {
		stageCodeToNameMap[snap.StageCode] = snap.StageName
	}
	if err := repository.TruncateConversationFromStage(ctx, lessonPlanID, targetStageCode, stageCodeToNameMap); err != nil {
		wsLog.Warn("截断对话记录失败，尝试清空", "plan_id", lessonPlanID, "error", err)
		_ = repository.ClearConversationLog(ctx, lessonPlanID)
	}

	wsLog.Info("重启阶段成功", "plan_id", lessonPlanID, "target_stage", targetStageCode)

	// 重启后持久化阶段分隔符
	resetSepContent := "__STAGE_SEP__" + targetStage.StageName + "__"
	resetSepMsg := &models.ConversationMessage{
		ID:        fmt.Sprintf("stage_sep_%s_%d", targetStageCode, time.Now().UnixMilli()),
		Role:      "system",
		Type:      "text",
		Content:   resetSepContent,
		CreatedAt: time.Now(),
	}
	_ = repository.AppendConversationMessage(ctx, lessonPlanID, resetSepMsg)

	// 触发AI开场白
	if triggerMsg, needsTrigger := autoTriggerStages[targetStageCode]; needsTrigger && s.genService != nil {
		go func() {
			time.Sleep(200 * time.Millisecond)
			bgCtx := context.Background()
			req := &models.LessonPlanChatRequest{PlanID: lessonPlanID, Message: triggerMsg}
			if err := s.genService.Chat(bgCtx, req, callerID); err != nil {
				wsLog.Warn("重启阶段后自动触发AI开场白失败", "plan_id", lessonPlanID, "stage", targetStageCode, "error", err)
			}
		}()
	}

	return &targetStage, nil
}

// ==================== 内部辅助函数 ====================

// resolveStages 解析教案的阶段配置快照并定位当前阶段索引
func (s *WorkshopStageService) resolveStages(lp *models.LessonPlan) ([]models.StageConfigSnapshot, int, error) {
	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}
	if len(snapshots) == 0 {
		return nil, -1, ErrStageNotInitialized
	}
	currentIdx := findStageIndex(snapshots, lp.CurrentStage)
	if currentIdx == -1 {
		return nil, -1, fmt.Errorf("当前阶段 %s 不在配置中", lp.CurrentStage)
	}
	return snapshots, currentIdx, nil
}
