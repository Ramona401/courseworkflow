package services

// lesson_plan_gen_service.go — 教案生成核心服务（主文件）
//
// v89-3拆分：评审相关逻辑移至lesson_plan_gen_review.go
// 子轮一·A拆分：阶段化对话的异步流式处理（processChatStageAsync）移至 lesson_plan_gen_chat_async.go。
// 助手轻量选择入口 Phase 1 拆分：一组纯辅助方法（checkPlanEditable/appendMessage/loadConversation/
//   resolveTemplateForReview/broadcastError/broadcastSoftRetryNotice/parseAIReply/
//   appendUnrecognizedTextbookNotice）搬至 lesson_plan_gen_helpers.go，使本主文件回到 600 行红线内。
//   本次纯位置搬移，逻辑零改动；同时删除无人调用的 GetStageService 死方法。
//
// 本文件职责（核心会话流程）：
//   1. StartConversation     — 创建教案+阶段初始化+配方上下文注入+发起AI开场白
//   2. genStageOpeningMessage — 生成首阶段开场白（吃老师×学科助手偏好）
//   3. Chat                  — 处理教师输入→（异步）流式AI回复（异步体在 lesson_plan_gen_chat_async.go）
//   4. resolveAssistantPrompt — 解析应注入第4层的助手 full_prompt（偏好→技能路由兜底）
//   5. GetConversation       — 获取教案对话历史
//   （纯辅助方法见 lesson_plan_gen_helpers.go）
//
// v110(TE-DNA 3.0 P0 STEP 3)改动:
//   - LessonPlanGenService 新增 assistantService 字段(可选,运行时通过 SetAssistantService 注入)
//   - Chat 接收 AssistantID,若非空则解析助手 full_prompt,透传到 processChatStageAsync
//
// v168改动(第二批治本·功能B·一键生成完整教案):
//   - processChatStageAsync 新增 fullGenerate 参数,write 阶段全委托一次性出稿
//
// v169改动(多阶段一键生成):
//   - 全委托从 write 单阶段扩展到 analyze/design/write/revise 四个"写内容"的阶段
//   - review 阶段不参与一键生成(推进过去会自动触发评审,无需额外按钮)
//
// v172改动(SSE 完成信号治本)：
//   - SSE 是「整个教案会话共享的一条长连接」，不每轮发 done 关连接；
//     「生成中」状态的复位改由前端 onMessageDone 处理。
//   - 详细历程见 lesson_plan_gen_chat_async.go 内 processChatStageAsync 的函数注释。
//
// v203改动(对话模式配方自动挂载·治本 yingjun 截图九大板块缺失问题)：
//   - StartConversation 在 req.RecipeID 为空时调 s.ResolveDefaultRecipe（recipe_resolver.go）
//     按「学校默认配方 → group/school 学科匹配配方 → 空」三级 fail-open 解析并回填 req.RecipeID。
//   - 回填后下方所有既有 `if req.RecipeID != ""` 分支自动命中，使对话模式与专家模式走完全相同的
//     下游（lp.RecipeID + recipeStagesConfig + InitStagesForPlan + RecordRecipeUsage），
//     配方的教案结构/流程/学情风格经 LoadStagePromptContextV2 全量注入，AI 不再是空骨架。
//   - 显式 recipe_id（专家模式）跳过解析，行为一字不变；解析失败退回纯骨架=改造前现状，零风险。
//
// 【技能路由 Phase 1 · 接线步骤2】默认助手解析（风格型技能·替换第4层）：
//   - resolveAssistantPrompt：当老师【没有】手动传 assistant_id 时，调用 skill_router.go 的
//     RouteDefaultAssistant，按「当前阶段场景+学科+学段+可见性」自动解析默认助手 ID
//     （优先级 个人>本校>系统），再走与手动选择【完全相同】的 LoadActiveAssistantForUse 加载路径。
//   - 静默降级铁律：任何一步失败 / 无可见默认助手 / 路由开关关闭 → 返回空串，沿用阶段原生第4层。
//
// 【对话式备课·助手轻量选择入口 Phase 1 · 读路径接入偏好】：
//   - resolveAssistantPrompt 在「老师没手动传 assistant_id」分支内，先查老师×学科偏好表
//     (repository.GetPref)，把「助手ID从哪来」从"立即走技能路由"改为"先看老师的显式选择"。
//   - 解析优先级落地为三态分流（见 resolveAssistantPrompt 函数体注释），RouteDefaultAssistant
//     由「第一顺位」降为「无偏好记录时的末位兜底」，函数与 flag 均保留(可回滚)。
//   - 偏好读取走后端(而非前端透传)，保证开场白等不经 chat 请求的路径也吃到偏好(PRD §6.2)。

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

	// ==================== v203：对话模式配方自动挂载 ====================
	// 对话模式 handleStart 不传 recipe_id，导致 lp.RecipeID 恒为 nil，配方的教案结构/流程/
	// 学情风格全部不注入，AI 拿到空骨架（yingjun 截图九大板块缺失的根因）。
	// 此处在【显式 recipe_id 为空】时，按「学校默认配方 → group/school 学科匹配配方 → 空」
	// 三级 fail-open 解析（详见 recipe_resolver.go 的 ResolveDefaultRecipe），解析到则回填
	// req.RecipeID。回填后，本函数下方所有既有 `if req.RecipeID != ""` 分支会自动命中，
	// 与专家模式走完全相同的下游，无需任何额外接线。
	// 显式 recipe_id（专家模式）不进此分支，行为一字不变；解析失败退回纯骨架=改造前现状。
	if strings.TrimSpace(req.RecipeID) == "" {
		if resolvedRecipeID := s.ResolveDefaultRecipe(ctx, authorID, req.Subject); resolvedRecipeID != "" {
			req.RecipeID = resolvedRecipeID
			lpGenLog.Info("对话模式自动挂载配方",
				"author", authorID, "subject", req.Subject, "topic", req.Topic,
				"recipe_id", resolvedRecipeID)
		}
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
	// 对话式备课·助手轻量选择入口 Phase 1：开场白也吃老师×学科偏好——
	// genStageOpeningMessage 内部会先解析偏好助手 prompt 并注入第4层（详见该函数）。
	var openingMsg *models.ConversationMessage
	openingMsg, err = s.genStageOpeningMessage(ctx, lp, snapshots, authorID)
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

// genStageOpeningMessage 阶段模式下生成第一阶段的AI开场白
//
// 对话式备课·助手轻量选择入口 Phase 1（本次改动）：
//   - 新增 authorID 形参，用于解析老师×学科的助手偏好。
//   - 开场白不再恒走「纯骨架」，而是先调 resolveAssistantPrompt 解析出应注入的助手 full_prompt，
//     再用 LoadStagePromptContextV2 把它注入第4层 —— 让首阶段开场白也体现老师选定/学科推荐的助手风格。
//   - 后续阶段开场白走 advanceStageWithComponents→genService.Chat 路径，已天然经过 resolveAssistantPrompt，
//     无需在此处理；本改动只补齐「首阶段开场白」这一条原本不注入助手的路径。
//   - 静默降级铁律不变：解析不到助手 / 偏好为空 → assistantPrompt 为空串 → 等价于原纯骨架行为。
func (s *LessonPlanGenService) genStageOpeningMessage(
	ctx context.Context,
	lp *models.LessonPlan,
	snapshots []models.StageConfigSnapshot,
	authorID string,
) (*models.ConversationMessage, error) {
	// 解析开场白应注入的助手 prompt（吃老师×学科偏好；解析不到则空串=纯骨架）。
	// 开场白无"老师当轮发言"，故 resolveAssistantPrompt 的 assistantID 传空，
	// 走「偏好表 → 学科推荐兜底」链，与对话路径同一套解析逻辑、同一份偏好。
	assistantPrompt, _ := s.resolveAssistantPrompt(ctx, lp, "", authorID)

	// 用 V2 注入助手 prompt（assistantPrompt 为空时 V2 内部等价于原 V1 纯骨架）。
	// recentUserText 传空：开场白阶段没有老师发言，第3层组件走保底全量匹配（与原行为一致）。
	stageSystemPrompt, err := s.stageService.LoadStagePromptContextV2(ctx, lp, snapshots[0].StageCode, assistantPrompt, "")
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
	openingAuthorID := lp.AuthorID
	// v197：解析作者所属学校ID，供分流判定境外授权（查不到空串→降级境内）
	openingSchoolID, _ := repository.GetSchoolIDByUserID(ctx, openingAuthorID)
	openingTraceCtx := &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &openingAuthorID,
		SchoolID:     schoolIDPtr(openingSchoolID),
	}
	result, err := aiClient.CallAI(aiCfg, stageSystemPrompt, userPrompt, openingTraceCtx)
	if err != nil {
		return nil, fmt.Errorf("AI开场白生成失败: %w", err)
	}

	content := strings.TrimSpace(StripSuggestedActionsBlock(result.Content))

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
// 异步处理体 processChatStageAsync 已搬至 lesson_plan_gen_chat_async.go（同包，调用方式不变）。
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

	// v110 新增 + 技能路由 Phase 1 + 助手轻量选择入口 Phase 1：解析 AI 助手。
	// 老师传了 assistant_id → 用指定助手；没传 → 由 resolveAssistantPrompt 内
	// 「偏好表 → RouteDefaultAssistant 兜底」链自动解析（解析不到则空串=不替换第4层）。
	// 传入 lp 供偏好查询(lp.Subject)与默认助手解析读取 当前阶段/学科/年级 维度。
	assistantPrompt, assistantLabel := s.resolveAssistantPrompt(ctx, lp, req.AssistantID, callerID)

	// v168/v169:全委托标志(按阶段在 processChatStageAsync 内判定)
	fullGenerate := req.FullGenerate

	// 子轮一·B：取本轮客户端轮次序号，透传给异步处理体，使本轮所有 SSE 事件都带上它。
	turnID := req.ClientTurnID
	go func() {
		bgCtx := context.Background()
		s.processChatStageAsync(bgCtx, lp, userMsg, currentStageMsgs, req, assistantPrompt, assistantLabel, fullGenerate, turnID)
	}()

	return nil
}

// resolveAssistantPrompt 将 assistant_id 解析为 full_prompt
// （v110 起；技能路由 Phase 1 扩展默认助手解析；助手轻量选择入口 Phase 1 接入老师×学科偏好）
//
// 解析优先级（本次改动后的最终形态）：
//
// 1）老师手动传了 assistant_id（assistantID 非空）→ 用该指定助手（最高优先，老师对当下最有发言权）。
// 2）老师没传（assistantID 为空）→ 查老师×学科偏好表 repository.GetPref(callerID, lp.Subject)：
//
//	2a）查到记录且 prefID 非空        → 用偏好里的助手（老师之前为该学科选定的）。
//	2b）查到记录且 prefID == ""       → 老师显式选了「系统默认(纯骨架)」→ 直接返回空串，
//	                                    【绝不再走 RouteDefaultAssistant 兜底】（尊重显式选择）。
//	2c）查询出真实 DB 错误            → 记 Warn，降级到步骤3兜底（不阻塞对话）。
//	2d）无记录（从没选过）            → 落到步骤3兜底。
//
// 3）RouteDefaultAssistant（降为末位兜底）按「场景+学科+学段+可见性」自动解析默认助手：
//
//	3a）命中 → 用它。
//	3b）空串 → 返回空串（= 不替换第4层，沿用阶段原生角色，老行为）。
//
// 无论走哪条路，拿到最终 assistantID 后都经【同一条】加载路径
// （LoadActiveAssistantForUse：可见性校验 + is_active 校验 + 使用量埋点）取 full_prompt，
// 保证手动选择 / 偏好命中 / 默认挂载的加载口径、埋点口径完全一致。
// 偏好指向的助手若已被删/停用，LoadActiveAssistantForUse 会失败 → 末尾统一降级返回空串（纯骨架），
// 即「偏好失效后静默退回最朴素状态」，符合 PRD 强默认+逃生口哲学。
//
// 静默降级：assistantService 未注入 / 用户反查失败 / 助手加载失败 / full_prompt 为空 /
// 无可见默认助手 → 一律返回空串，绝不报错给老师、绝不阻塞对话。
func (s *LessonPlanGenService) resolveAssistantPrompt(
	ctx context.Context,
	lp *models.LessonPlan,
	assistantID string,
	callerID string,
) (string, string) {
	assistantID = strings.TrimSpace(assistantID)

	// assistantService 未注入：手动与默认两条路都无从加载，直接走空串老行为。
	if s.assistantService == nil {
		if assistantID != "" {
			lpGenLog.Warn("Chat 收到 assistant_id 但 assistantService 未注入,降级到默认 prompt",
				"assistant_id", assistantID)
		}
		return "", ""
	}

	// 构造操作者上下文（一次构造，供默认助手解析与 LoadActiveAssistantForUse 复用）。
	// 反查用户角色失败则无法判定可见性，安全起见走空串老行为。
	user, err := repository.FindUserByID(ctx, callerID)
	if err != nil {
		lpGenLog.Warn("Chat 加载用户角色失败,降级到默认 prompt",
			"caller_id", callerID, "error", err)
		return "", ""
	}
	actor := BuildActorFromClaims(ctx, callerID, user.Role)

	// 老师没手动选助手 → 先查老师×学科偏好，再以技能路由兜底。
	if assistantID == "" {
		// lp 为空时无法查偏好/解析默认助手（理论上 Chat 与开场白路径 lp 必非空，此为防御）。
		if lp == nil {
			return "", ""
		}

		// —— 步骤2：查老师×学科偏好表（助手轻量选择入口 Phase 1 核心）——
		prefID, found, perr := repository.GetPref(ctx, callerID, lp.Subject)
		if perr != nil {
			// 2c：真实 DB 错误。只 Warn，不阻塞——降级到步骤3技能路由兜底。
			lpGenLog.Warn("查询老师×学科助手偏好失败,降级到默认助手兜底",
				"caller_id", callerID, "subject", lp.Subject, "error", perr)
		} else if found {
			if strings.TrimSpace(prefID) != "" {
				// 2a：老师为该学科显式选定了某助手 → 用它（走下方统一加载路径）。
				assistantID = strings.TrimSpace(prefID)
				lpGenLog.Info("命中老师×学科助手偏好",
					"plan_id", lp.ID, "subject", lp.Subject, "stage", lp.CurrentStage,
					"pref_assistant_id", assistantID)
			} else {
				// 2b：老师显式选了「系统默认(纯骨架)」→ 直接返回空串，绝不再走兜底。
				// 这正是三态语义的关键：显式选系统默认 ≠ 从没选过。
				lpGenLog.Info("老师×学科偏好为显式系统默认(纯骨架),不挂任何助手",
					"plan_id", lp.ID, "subject", lp.Subject, "stage", lp.CurrentStage)
				return "", ""
			}
		}

		// —— 步骤3：无偏好命中（无记录 / 偏好查询出错）→ RouteDefaultAssistant 末位兜底 ——
		// 仅当上面没把 assistantID 赋成偏好助手时才进入（2a 命中则跳过本段）。
		if assistantID == "" {
			defaultID := RouteDefaultAssistant(ctx, s.assistantService, actor, lp.CurrentStage, lp.Subject, lp.Grade)
			if defaultID == "" {
				// 3b：无可见默认助手 → 不替换第4层，沿用阶段原生角色（与改造前"未选助手"行为一致）。
				return "", ""
			}
			// 3a：技能路由解析到默认助手。
			assistantID = defaultID
			lpGenLog.Info("Chat 自动挂载默认助手（技能路由兜底）",
				"plan_id", lp.ID, "stage", lp.CurrentStage,
				"default_assistant_id", defaultID)
		}
	}

	// 统一加载路径：手动选择 / 偏好命中 / 默认挂载在此汇合。
	a, err := s.assistantService.LoadActiveAssistantForUse(ctx, actor, assistantID)
	if err != nil {
		lpGenLog.Warn("Chat 加载 AI 助手失败,降级到默认 prompt",
			"assistant_id", assistantID, "caller_id", callerID, "error", err)
		return "", ""
	}

	if strings.TrimSpace(a.FullPrompt) == "" {
		lpGenLog.Warn("Chat 助手 full_prompt 为空,降级到默认 prompt",
			"assistant_id", assistantID)
		return "", ""
	}

	lpGenLog.Info("Chat 使用 AI 助手",
		"assistant_id", assistantID, "assistant_name", a.Name,
		"source", a.Source, "prompt_len", len(a.FullPrompt))
	return a.FullPrompt, a.Name
}

// ==================== 3. 获取对话历史 ====================

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

// schoolIDPtr 把学校ID字符串转为 *string：空串→nil（fail-closed，分流按未授权处理），
// 非空→指向该值。供备课各路径构造 TraceContext.SchoolID 复用。
func schoolIDPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
