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
//
// v172改动(SSE 完成信号治本·修复"一键生成要刷新才出来 / 出来了还显示生成中"):
//   根因：
//     1) processChatStageAsync 全程只广播 thinking→chunk→(stage_output)→message_done，
//        从不广播 done 事件。前端 createLessonPlanSSE 只有收到 done 才会主动 es.close()。
//        后端不发 done → 这条 SSE HTTP 长连接生成完毕后仍挂着不关，直到被 Nginx/浏览器
//        超时掐断 → 前端 es.onerror 触发自动重连 → 重连调后端 Subscribe(独占模式) →
//        关闭正在推送的旧 channel → 后续关键事件(message_done/content_update)被丢弃 →
//        前端永远收不到收尾事件，但内容其实已落库 → "要刷新才出来"。
//     2) 前端 fullGenerating 复位只挂在 content_update(只有 write/revise 落库才发)，
//        analyze/design 阶段不发 content_update → "生成中"永不消失。
//   修复历程更正（v172 当日回滚）：
//     起初尝试在 processChatStageAsync 结尾 defer 补发 done 让前端 es.close() 收尾，
//     但本系统的 SSE 是「整个教案会话共享的一条长连接」，并非「一轮对话一条连接」。
//     每轮结束就发 done → 前端收到后 es.close() 关掉整条会话连接 →
//     进入阶段时开场白那一轮发的 done 会把连接关掉，导致随后「一键生成」时
//     已无活动 SSE 连接接收 chunk（前端 Network 中 stream 连接消失），结果收不到、要刷新才出来。
//     因此已撤销后端补发 done 的做法：SSE 长连接保持开启、等待后续轮次复用，不再每轮关闭。
//   最终方案：仅在前端 onMessageDone 里复位「生成中」类状态（fullGenerating 等），
//     既治「出来了还显示生成中」，又不关闭会话长连接（不影响后续轮次接收推送）。

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

// SetTextbookServiceForStage 注入课本服务到内部 stageService(由 routes 层调用)
// 迭代7B修复：LessonPlanGenService 内部 stageService 是 NewLessonPlanGenService 时
// 独立 new 的实例，与 routes 中给 handler 用的 wsStageService 不是同一个对象。
// 对话流程(Chat→processChatStageAsync)走的是内部这个 stageService，必须单独给它注入
// textbookService，否则 LoadStagePromptContextV2 里 s.textbookService==nil，课本注入被静默跳过。
func (s *LessonPlanGenService) SetTextbookServiceForStage(ts *TextbookService) {
	s.stageService.SetTextbookService(ts)
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

        // 迭代7B：备课工坊勾选的课本图片ID列表落库（写入 lesson_plans.textbook_page_ids，
        // 供 LoadStagePromptContextV2 在各阶段提示词中注入课本原文，让 AI 参考真实教材内容）
        if len(req.TextbookPageIDs) > 0 {
                if tbIDsJSON, mErr := json.Marshal(req.TextbookPageIDs); mErr == nil {
                        lp.TextbookPageIDs = string(tbIDsJSON)
                } else {
                        lpGenLog.Warn("课本图片ID序列化失败，忽略关联", "error", mErr)
                }
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

	// 迭代7B：若关联了课本图但其中有未成功识别（无 OCR 文字）的，
	// 由后端确定性地在开场白末尾拼一句点名提醒（不依赖 AI 自由发挥，必然出现、措辞精确）。
	// 覆盖 AI 开场白成功与降级两条路径（两者最终都汇合到 openingMsg）。
	s.appendUnrecognizedTextbookNotice(ctx, req, openingMsg)

	if err2 := s.appendMessage(ctx, lp.ID, openingMsg); err2 != nil {
		lpGenLog.Warn("写入开场消息失败", "plan_id", lp.ID, "error", err2)
	}

	return lp, openingMsg, nil
}

// appendUnrecognizedTextbookNotice 检查勾选的课本图中有几张未识别文字（OCR 为空），
// 若有，则在开场白消息末尾拼接一句确定性的点名提醒，让老师明确知道哪些课本无法被参考。
// 设计要点：
//   - 纯代码判断 + 字符串拼接，不经过 AI，保证"必然出现、措辞精确"；
//   - 用图在勾选列表中的序号（第X张）定位，老师在课本区按勾选顺序即可对应；
//   - 任何异常（无关联/查询失败/全部已识别）都静默返回，不影响开场白。
func (s *LessonPlanGenService) appendUnrecognizedTextbookNotice(
	ctx context.Context,
	req *models.StartConversationRequest,
	openingMsg *models.ConversationMessage,
) {
	if openingMsg == nil || len(req.TextbookPageIDs) == 0 {
		return
	}

	pages, err := repository.GetTextbookPagesByIDs(ctx, req.TextbookPageIDs)
	if err != nil || len(pages) == 0 {
		return
	}

	// GetTextbookPagesByIDs 返回顺序按 textbook_name+page_number 排，
	// 与前端勾选顺序未必一致，这里按返回顺序编号"第X张"，并尽量带上章节/教材名帮助老师定位。
	var unrecognized []string
	for i, p := range pages {
		if strings.TrimSpace(p.OCRText) == "" {
			label := strings.TrimSpace(p.Chapter)
			if label == "" {
				label = strings.TrimSpace(p.TextbookName)
			}
			if label == "" {
				unrecognized = append(unrecognized, fmt.Sprintf("第%d张", i+1))
			} else {
				unrecognized = append(unrecognized, fmt.Sprintf("第%d张（%s）", i+1, label))
			}
		}
	}

	if len(unrecognized) == 0 {
		return
	}

	notice := fmt.Sprintf(
		"\n\n---\n📷 **课本识别提醒**：你关联的 %d 张课本图中，%s 尚未成功识别文字，本次备课**无法参考这些页面的内容**。建议返回「课本管理」对这些图重新点击「AI识别」，识别成功后再关联进来。",
		len(pages), strings.Join(unrecognized, "、"),
	)
	openingMsg.Content += notice

	lpGenLog.Info("开场白已拼接未识别课本提醒",
		"plan_id", req.Topic, "total", len(pages), "unrecognized", len(unrecognized))
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
//
// v172说明（已撤销 done 补发）：
//   本函数不再补发 done 事件。原因：SSE 是教案会话级共享长连接，发 done 会让前端关闭整条连接，
//   导致进入阶段的开场白那轮 done 关连接后、随后「一键生成」无连接可收。
//   「生成中」状态的复位改由前端 onMessageDone 处理，不依赖关闭连接。
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
	//
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
