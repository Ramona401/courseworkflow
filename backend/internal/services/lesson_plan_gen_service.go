package services

// lesson_plan_gen_service.go — 教案生成核心服务（主文件）
//
// v89-3拆分：评审相关逻辑移至lesson_plan_gen_review.go
//
// 本文件职责：
//   1. StartConversation  — 创建教案+阶段初始化+配方上下文注入+发起AI开场白
//   2. Chat               — 处理教师输入→流式AI回复→SSE逐token推送
//   3. processChatStageAsync — 阶段模式异步AI流式回复
//   4. checkAndInsertCoachAdvice — 停滞检测+教练建议插入
//   5. GetConversation    — 获取教案对话历史
//   6. 内部辅助方法（checkPlanEditable/appendMessage/broadcastError/parseAIReply等）
//
// v110(TE-DNA 3.0 P0 STEP 3)改动:
//   - LessonPlanGenService 新增 assistantService 字段(可选,运行时通过 SetAssistantService 注入)
//   - Chat 接收 AssistantID,若非空则解析助手 full_prompt,透传到 processChatStageAsync
//
// v168改动(第二批治本·功能B·一键生成完整教案):
//   - processChatStageAsync 新增 fullGenerate 参数,write 阶段全委托一次性出稿
//
// v169改动(多阶段一键生成):
//   - 全委托从 write 单阶段扩展到 analyze/design/write/revise 四个"写内容"的阶段:
//       * fullGenerateAnalyzePrompt — 教学分析一键生成(落库走 extractGenericStageFromNatural,宽松)
//       * fullGenerateDesignPrompt  — 教学设计一键生成(同上)
//       * fullGenerateWritePrompt   — 教案撰写一键生成(落库走 DetectLessonPlanContent,严格,v168已有)
//       * fullGenerateRevisePrompt  — 修订定稿一键生成(基于评审建议产出完整教案,落库同 write 严格)
//   - processChatStageAsync 用 resolveFullGeneratePrompt(stage) 按阶段返回对应指令:
//       * write 仍保留"已有正文→防重复"三态逻辑(正文为空且 fullGenerate 才注入)
//       * analyze/design/revise 在 fullGenerate=true 时直接注入各自指令
//   - review 阶段不参与一键生成(推进过去会自动触发评审,无需额外按钮)
//   - 重要:本批次产出"纯 AI 一次性生成"内容,前端已在二次确认弹窗与产出卡片明确警示
//     "纯 AI 产出可能有幻觉、与真实学情/教材不符,务必核对"。后端不重复警示,只负责出稿。

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

// ==================== 常量定义 ====================

// lessonPlanSceneCode 教案生成场景代码，用于从ai_scene_configs获取独立模型配置
const lessonPlanSceneCode = "lesson_plan"

// fullGenerateWritePrompt 教案撰写阶段·全委托一键生成指令（v168·功能B）
//
// 设计原则：本指令格式严格对齐 DetectLessonPlanContent（workshop_stage_extract.go）的判定条件，
// 确保 AI 一次性产出的教案能被识别并落库。DetectLessonPlanContent 要求：
//  1. ≥3 个教案标记词（教学目标/重点/难点/过程/准备/作业/板书...）
//  2. 有 # 开头的标题行，且标题含"教案/教学设计/教学目标/课题..."之一
//  3. 含"教学过程"或"教学环节"或"教学活动"
//  4. 含结尾标记（作业布置/板书设计/课后作业/课堂小结/教学反思...之一）
//  5. 去掉末尾客套话后正文 ≥800 字符
//
// 因此本指令强制：用 # 标题、含全部必备小节、教学过程分环节带时间、不分段、不寒暄。
const fullGenerateWritePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认要求）==
老师已明确选择"全委托 AI 一次性生成完整教案"，请你立即一次性输出一份**完整、可直接使用**的教案 Markdown，不要分段、不要等老师确认、不要说"接下来写下一部分"，本次回复就给出全部内容。

输出格式硬性要求（务必全部满足，否则系统无法识别保存）：
1. 以一级标题开头，格式为：# 《课题》教学设计（或 # 课题 教案）。
2. 必须包含以下小节，每个小节用 ## 二级标题：
   ## 教学目标（分知识与技能、过程与方法、情感态度价值观三维，或按核心素养列出，要具体可观察）
   ## 教学重难点（明确教学重点与教学难点）
   ## 教学准备（教具、学具、课件、场地等）
   ## 教学过程（这是核心，必须按环节展开，每个环节标注时间分配，如"一、导入（5分钟）"，环节要包含教师活动与学生活动）
   ## 作业布置（具体的课后作业或练习）
   ## 板书设计（本节课的板书结构）
3. 教学过程要详实，环节完整（导入→新授→巩固→小结等），内容充实，确保整份教案不少于 800 字。
4. 结尾直接以"板书设计"小节自然结束，**不要**添加"如需修改请告诉我""希望这份教案对您有帮助"等客套话。
5. 全程使用规范的 Markdown 标题层次（# ## ###），正文用自然中文，不要输出 JSON 或代码块包裹整篇教案。

请现在就开始输出完整教案。`

// fullGenerateAnalyzePrompt 教学分析阶段·全委托一键生成指令（v169）
//
// 落库路径：analyze 走 extractGenericStageFromNatural（判定宽松，内容非空即存为 narrative），
// 故格式要求不必像 write 那样严格，重点是引导 AI 一次性输出完整的、结构清晰的学情/教材分析。
const fullGenerateAnalyzePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认/逐步追问要求）==
老师已明确选择"全委托 AI 一次性完成本阶段（教学分析）"，请你立即一次性输出一份**完整的教学分析**，不要分段、不要反过来追问老师、不要说"接下来分析下一部分"，本次回复就给出全部内容。

请用 Markdown 输出，至少包含以下方面（用 ## 二级标题分节）：
## 教材分析（本课内容在教材中的地位、知识结构、与前后内容的联系）
## 课程标准对接（本课对应的课程标准/核心素养要求）
## 学情分析（该年级学生的认知特点、已有知识基础、可能的学习难点与误区）
## 核心概念与重难点预判（本课的核心概念，以及预计的教学重点和难点）

要求：内容具体、贴合学科与年级，避免空话套话。结尾不要加"如需调整请告诉我"之类的客套话。

请现在就开始输出完整的教学分析。`

// fullGenerateDesignPrompt 教学设计阶段·全委托一键生成指令（v169）
//
// 落库路径：design 走 extractGenericStageFromNatural（宽松）。
const fullGenerateDesignPrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认/逐步追问要求）==
老师已明确选择"全委托 AI 一次性完成本阶段（教学设计）"，请你立即一次性输出一份**完整的教学设计方案**，不要分段、不要反过来追问老师，本次回复就给出全部内容。

请用 Markdown 输出，至少包含以下方面（用 ## 二级标题分节）：
## 教学目标（三维目标或核心素养目标，要具体可观察、可评估）
## 教学重难点（明确重点与难点，并说明突破难点的策略）
## 教学策略（采用的教学方法、学习方式，如探究式/任务驱动/小组合作等及理由）
## 教学活动设计（按环节展开，每个环节给出名称、预计时长、教师活动、学生活动、设计意图）
## 评价设计（如何检验目标达成，形成性评价与总结性评价的安排）

要求：活动设计要可操作、环节衔接合理、贴合学科与年级。结尾不要加客套话。

请现在就开始输出完整的教学设计方案。`

// fullGenerateRevisePrompt 修订定稿阶段·全委托一键生成指令（v169）
//
// 落库路径：revise 与 write 共用 handleWriteStageOutput，需命中 DetectLessonPlanContent（严格）。
// 故格式要求与 write 完全一致（# 标题 + 全部必备小节 + ≥800字），
// 区别在于：要求 AI 基于"前面阶段已完成的教案正文 + AI 评审建议"做修订后输出完整教案。
// 已完成的正文与评审结论已由 LoadStagePromptContextV2 + Episodic 摘要注入上下文，AI 可直接参考。
const fullGenerateRevisePrompt = `

== 全委托一键生成模式（系统级指令，最高优先级，覆盖上文所有分段确认要求）==
老师已明确选择"全委托 AI 一次性完成修订定稿"。请你参考前面阶段已经完成的教案正文，以及 AI 评审阶段提出的改进建议，**对教案进行修订并一次性输出修订后的完整教案 Markdown**。不要只输出修改点、不要分段、不要等老师确认，本次回复直接给出修订后的整份教案。

输出格式硬性要求（务必全部满足，否则系统无法识别保存）：
1. 以一级标题开头，格式为：# 《课题》教学设计（或 # 课题 教案）。
2. 必须包含以下小节，每个小节用 ## 二级标题：教学目标、教学重难点、教学准备、教学过程（按环节展开并标注时间分配，含教师活动与学生活动）、作业布置、板书设计。
3. 在原教案基础上落实评审建议的改进点（如时间分配、评价工具、活动设计等），但仍输出**完整**教案而非仅改动部分。
4. 整份教案不少于 800 字，结尾以"板书设计"小节自然结束，不要加客套话。
5. 使用规范 Markdown 标题层次，不要用 JSON 或代码块包裹整篇教案。

请现在就开始输出修订后的完整教案。`

// resolveFullGeneratePrompt 按阶段返回对应的全委托一键生成指令（v169）
//
// 返回空字符串表示该阶段不支持一键生成（如 review）。
// write 阶段的"已有正文→防重复"判定不在此处，仍由 processChatStageAsync 单独处理。
func resolveFullGeneratePrompt(stageCode string) string {
	switch stageCode {
	case "analyze":
		return fullGenerateAnalyzePrompt
	case "design":
		return fullGenerateDesignPrompt
	case "write":
		return fullGenerateWritePrompt
	case "revise":
		return fullGenerateRevisePrompt
	default:
		return ""
	}
}

// ==================== 服务结构体 ====================

// LessonPlanGenService 教案生成服务
type LessonPlanGenService struct {
	cfg              interface{ GetAESKey() string }
	recipeService    *RecipeService
	stageService     *WorkshopStageService
	assistantService *AIAssistantService // v110:运行时注入,用于加载 AI 助手(可选)
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

// SetAssistantService 注入 AI 助手服务(由 routes 层调用)
func (s *LessonPlanGenService) SetAssistantService(as *AIAssistantService) {
	s.assistantService = as
}

// ==================== 1. 开始备课会话 ====================

// StartConversation 创建教案+阶段初始化+配方上下文注入+发起AI开场白
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

	// 统一走阶段化流程
	recipeStagesConfig := ""
	if req.RecipeID != "" {
		recipe, err := repository.GetRecipeByID(ctx, req.RecipeID)
		if err == nil {
			recipeStagesConfig = recipe.StagesConfig
		}
	}

	snapshots, err := s.stageService.InitStagesForPlan(ctx, lp.ID, recipeStagesConfig, req.RecipeID)
	if err != nil {
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

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("AI配置加载失败: %w", err)
	}

	planID := lp.ID
	authorID := lp.AuthorID
	openingTraceCtx := &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &authorID,
	}
	result, err := aiClient.CallAI(aiCfg, stageSystemPrompt, userPrompt, openingTraceCtx)
	if err != nil {
		return nil, fmt.Errorf("AI开场白生成失败: %w", err)
	}

	content := strings.TrimSpace(result.Content)

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

// ==================== 2. 对话轮次（流式SSE推送）====================

// Chat 处理教师输入，AI生成回复并通过SSE流式推送
//
// v169改动：fullGenerate 不再仅 write 生效，透传给 processChatStageAsync 按阶段判定
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

	// v110 新增:解析 AI 助手(若前端传了 assistant_id)
	assistantPrompt := s.resolveAssistantPrompt(ctx, req.AssistantID, callerID)

	// v168/v169:全委托标志(按阶段在 processChatStageAsync 内判定)
	fullGenerate := req.FullGenerate

	go func() {
		bgCtx := context.Background()
		s.processChatStageAsync(bgCtx, lp, userMsg, currentStageMsgs, req, assistantPrompt, fullGenerate)
	}()

	return nil
}

// resolveAssistantPrompt v110:将 assistant_id 解析为 full_prompt
func (s *LessonPlanGenService) resolveAssistantPrompt(
	ctx context.Context,
	assistantID string,
	callerID string,
) string {
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return ""
	}
	if s.assistantService == nil {
		lpGenLog.Warn("Chat 收到 assistant_id 但 assistantService 未注入,降级到默认 prompt",
			"assistant_id", assistantID)
		return ""
	}

	user, err := repository.FindUserByID(ctx, callerID)
	if err != nil {
		lpGenLog.Warn("Chat 加载用户角色失败,降级到默认 prompt",
			"caller_id", callerID, "error", err)
		return ""
	}

	actor := BuildActorFromClaims(ctx, callerID, user.Role)
	a, err := s.assistantService.LoadActiveAssistantForUse(ctx, actor, assistantID)
	if err != nil {
		lpGenLog.Warn("Chat 加载 AI 助手失败,降级到默认 prompt",
			"assistant_id", assistantID, "caller_id", callerID, "error", err)
		return ""
	}

	if strings.TrimSpace(a.FullPrompt) == "" {
		lpGenLog.Warn("Chat 助手 full_prompt 为空,降级到默认 prompt",
			"assistant_id", assistantID)
		return ""
	}

	lpGenLog.Info("Chat 使用 AI 助手",
		"assistant_id", assistantID, "assistant_name", a.Name,
		"source", a.Source, "prompt_len", len(a.FullPrompt))
	return a.FullPrompt
}

// ==================== 2.1 阶段化对话（v84分层记忆 + v87教练集成 + v110助手注入 + v168/v169全委托）====================

// processChatStageAsync 阶段模式：异步处理AI流式回复
//
// v169改动（多阶段一键生成）：
//   - write 阶段保留原三态（已有正文→防重复 / 正文空+fullGenerate→全委托 / 其余逐轮）
//   - analyze/design/revise 阶段：fullGenerate=true 时注入 resolveFullGeneratePrompt 返回的对应指令
//   - review 阶段：resolveFullGeneratePrompt 返回空，不参与一键生成
func (s *LessonPlanGenService) processChatStageAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	currentStageMsgs []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
	assistantPrompt string,
	fullGenerate bool,
) {
	planID := lp.ID
	currentStage := lp.CurrentStage

	// 推送thinking状态
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
		MessageID: generateMsgID(),
	})

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		s.broadcastError(planID, "AI配置加载失败: "+err.Error())
		return
	}

	// 加载阶段系统提示词(v110:使用 V2 版本支持 assistantPrompt 注入)
	stageSystemPrompt, err := s.stageService.LoadStagePromptContextV2(ctx, lp, currentStage, assistantPrompt)
	if err != nil {
		lpGenLog.Warn("加载阶段提示词失败", "plan_id", planID, "stage", currentStage, "error", err)
		s.broadcastError(planID, "加载阶段配置失败，请刷新重试")
		return
	}

	// 全委托标志：本轮是否实际注入了全委托指令（用于后续跳过停滞检测）
	fullGenInjected := false

	if currentStage == "write" {
		// ---------- write 阶段三态处理（v168，保持原逻辑）----------
		latestLP, freshErr := repository.GetLessonPlanByID(ctx, planID)
		hasExistingContent := freshErr == nil && len(strings.TrimSpace(latestLP.ContentMarkdown)) > 2000

		switch {
		case hasExistingContent:
			// 态a：已有教案内容 → 注入防重复生成指令
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
				"plan_id", planID, "stage", currentStage, "content_len", contentLen)

		case fullGenerate:
			// 态b：正文为空 + 老师选择全委托 → 注入全委托一次性出稿指令
			stageSystemPrompt += fullGenerateWritePrompt
			fullGenInjected = true
			lpGenLog.Info("write阶段全委托一键生成，注入全委托出稿指令",
				"plan_id", planID, "stage", currentStage)

		default:
			// 态c：原逐轮分段确认逻辑，stageSystemPrompt 不追加
		}
	} else if fullGenerate {
		// ---------- analyze/design/revise 阶段一键生成（v169）----------
		if fgPrompt := resolveFullGeneratePrompt(currentStage); fgPrompt != "" {
			stageSystemPrompt += fgPrompt
			fullGenInjected = true
			lpGenLog.Info("阶段全委托一键生成，注入全委托指令",
				"plan_id", planID, "stage", currentStage)
		} else {
			lpGenLog.Warn("收到 fullGenerate 但该阶段不支持一键生成，忽略",
				"plan_id", planID, "stage", currentStage)
		}
	}

	// 构建Episodic Memory
	allOutputs, _ := repository.ListStageOutputs(ctx, planID)
	var priorOutputs []*models.WorkshopStageOutput
	for _, out := range allOutputs {
		if out.StageCode == currentStage {
			break
		}
		priorOutputs = append(priorOutputs, out)
	}
	episodicSummary := repository.BuildEpisodicSummaryFromOutputs(priorOutputs)

	// 使用BuildStageChatPromptV2构建分层上下文
	userPrompt := BuildStageChatPromptV2(lp, currentStageMsgs, episodicSummary, userMsg)

	lpGenLog.Info("v84分层记忆上下文构建完成",
		"plan_id", planID, "stage", currentStage,
		"working_msgs", len(currentStageMsgs), "episodic_len", len(episodicSummary),
		"prior_stages", len(priorOutputs), "assistant_injected", assistantPrompt != "",
		"full_generate", fullGenerate, "full_gen_injected", fullGenInjected)

	// 流式推送
	chunkCount := 0
	var fullContent strings.Builder

	result, err := aiClient.CallAIStream(aiCfg, stageSystemPrompt, userPrompt, func(chunk string) error {
		if strings.TrimSpace(chunk) == "" {
			return nil
		}
		chunkCount++
		fullContent.WriteString(chunk)

		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType: models.LPSSEChunk,
			PlanID:    planID,
			Chunk:     chunk,
		})
		return nil
	}, &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &lp.AuthorID,
	})
	if err != nil {
		s.broadcastError(planID, "AI回复失败: "+err.Error())
		return
	}

	rawContent := result.Content
	if rawContent == "" {
		rawContent = fullContent.String()
	}

	// 从自然语言中提取结构化数据
	structuredJSON, narrative, hasContent := ExtractStructuredFromNaturalReply(currentStage, rawContent)
	if hasContent {
		if err := s.stageService.SaveStageOutput(ctx, planID, currentStage, structuredJSON, narrative, result.ModelUsed, result.TokensUsed); err != nil {
			lpGenLog.Warn("保存阶段产出物失败", "plan_id", planID, "stage", currentStage, "error", err)
		} else {
			lpGenLog.Info("阶段产出物已保存", "plan_id", planID, "stage", currentStage)
		}

		// 处理阶段副作用（在lesson_plan_gen_review.go中定义）
		s.handleStageOutputSideEffects(ctx, planID, lp, currentStage, structuredJSON, rawContent)

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

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEMessageDone,
		PlanID:    planID,
		MessageID: aiReply.ID,
		Message:   aiReply,
	})

	lpGenLog.Info("AI对话流式回复完成（v84分层记忆+v110助手注入+v168/v169全委托）",
		"plan_id", planID, "stage", currentStage,
		"tokens", result.TokensUsed, "latency_ms", result.LatencyMs,
		"chunks", chunkCount, "has_content", hasContent,
		"working_msgs", len(currentStageMsgs),
		"assistant_injected", assistantPrompt != "",
		"full_generate", fullGenerate)

	// v87：对话完成后异步检测停滞，插入教练建议
	// v168/v169：全委托一次性出稿不需要停滞检测（本就是一次性完成），跳过避免误插建议
	if !fullGenInjected {
		go s.checkAndInsertCoachAdvice(ctx, planID, currentStage)
	}
}

// ==================== v87：停滞检测+教练建议插入 ====================

// checkAndInsertCoachAdvice 对话完成后检测停滞，插入教练建议
func (s *LessonPlanGenService) checkAndInsertCoachAdvice(ctx context.Context, planID string, stageCode string) {
	time.Sleep(500 * time.Millisecond)

	stagnation := DetectStagnation(ctx, planID, stageCode)
	if stagnation == nil || !stagnation.IsStagnant {
		return
	}

	suggestion := GenerateCoachSuggestion(stagnation)
	if suggestion == "" {
		return
	}

	coachMsg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   suggestion,
		CreatedAt: time.Now(),
	}

	if err := s.appendMessage(ctx, planID, coachMsg); err != nil {
		lpGenLog.Warn("v87教练建议-写入消息失败", "plan_id", planID, "error", err)
		return
	}

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType: models.LPSSEMessageDone,
		PlanID:    planID,
		MessageID: coachMsg.ID,
		Message:   coachMsg,
	})

	lpGenLog.Info("v87教练建议已插入",
		"plan_id", planID, "stage", stageCode,
		"user_rounds", stagnation.ConsecutiveRounds)
}

// ==================== 5. 获取对话历史 ====================

// GetConversation 获取教案对话历史
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
