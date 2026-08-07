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
//   1. genStageOpeningMessage — 生成首阶段开场白（吃老师×学科助手偏好）
//   2. Chat                   — 处理教师输入→（异步）流式AI回复（异步体在 lesson_plan_gen_chat_async.go）
//   3. resolveAssistantPrompt — 解析应注入第4层的助手 full_prompt（偏好→技能路由兜底）
//   4. GetConversation        — 获取教案对话历史
//   （StartConversation见 lesson_plan_gen_start.go；纯辅助方法见 lesson_plan_gen_helpers.go）
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
//
// StartConversation及其“教育域硬闸 + 显式写域 + 失败补偿”编排，
// 已拆分到lesson_plan_gen_start.go，避免本核心文件继续超过600行。
// 本文件保留开场白生成、对话轮次、助手解析和对话历史读取能力。

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

	/*
	 * 明确的定稿或发布确认语不是AI生成任务。
	 *
	 * 必须在登记后台任务和写入用户消息之前fail-fast：
	 *   - 当前前端会直接调用发布接口；
	 *   - 旧缓存页面或手工API调用也不能启动正式产物Harness；
	 *   - 不在对话历史留下只有用户消息、没有终态回复的半轮记录。
	 */
	if isLessonPlanPublishIntent(
		req.Message,
	) {
		lpGenLog.Info(
			"Chat入口拦截明确发布意图",
			"plan_id", lp.ID,
			"caller_id", callerID,
		)
		return ErrLPGenPublishIntent
	}

	// 在写入老师消息之前登记任务。
	// draining或重复任务被拒绝时，不会留下只有用户消息、没有AI回复的半轮对话。
	task, taskErr := startLessonPlanAITask(lp.ID)
	if taskErr != nil {
		return taskErr
	}

	taskLaunched := false
	defer func() {
		if !taskLaunched {
			task.Done()
		}
	}()

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
	assistantResolution := s.resolveAssistantPromptForReceipt(
		ctx,
		lp,
		req.AssistantID,
		callerID,
	)
	assistantPrompt := assistantResolution.Prompt
	assistantLabel := ""
	if assistantResolution.Receipt != nil {
		assistantLabel = assistantResolution.Receipt.Name
	}

	// v168/v169:全委托标志(按阶段在 processChatStageAsync 内判定)
	fullGenerate := req.FullGenerate

	// 子轮一·B：取本轮客户端轮次序号，透传给异步处理体。
	turnID := req.ClientTurnID

	s.runLessonPlanAITask(
		task,
		lp.ID,
		turnID,
		"chat",
		func() {
			s.processChatStageAsync(
				context.Background(),
				lp,
				userMsg,
				currentStageMsgs,
				req,
				assistantPrompt,
				assistantLabel,
				assistantResolution.Receipt,
				fullGenerate,
				turnID,
			)
		},
	)
	taskLaunched = true

	return nil
}

// applyLessonPlanEducationDomainToAssistantActor 把具体教案的资源教育域快照写入助手Actor。
//
// 教案是本轮教学运行的事实主体。无论操作者是普通教师还是mixed管理员，
// 进入同一份具体教案后，都必须使用lesson_plans.education_domain创建时快照，
// 不能继续使用登录用户当前组织域，更不能让mixed绕过具体教学域隔离。
//
// 当前教案正常只会保存k12、vocational或adult。若读取到空值、非法值、
// common或mixed，函数会主动清空Actor.EducationDomain，使后续候选列表和
// 按ID加载统一fail-closed，绝不错误回退K12。
func applyLessonPlanEducationDomainToAssistantActor(
	actor *AssistantActorContext,
	lp *models.LessonPlan,
) {
	if actor == nil {
		return
	}

	// 先清空登录用户原教育域，确保异常教案快照不会沿用mixed或其它旧值。
	actor.EducationDomain = ""

	if lp == nil {
		return
	}

	lessonDomain := strings.ToLower(
		strings.TrimSpace(lp.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(lessonDomain) {
		return
	}

	actor.EducationDomain = lessonDomain
}

// resolveAssistantPrompt 将 assistant_id 解析为 full_prompt
// （v110 起；技能路由 Phase 1 扩展默认助手解析；助手轻量选择入口 Phase 1 接入老师×学科偏好）
//
// 解析优先级（本次改动后的最终形态）：
//
// 1）老师手动传了 assistant_id（assistantID 非空）→ 用该指定助手（最高优先，老师对当下最有发言权）。
// 2）老师没传（assistantID 为空）→ 查老师×学科偏好表 repository.GetPref(callerID, lp.Subject)：
//
//      2a）查到记录且 prefID 非空        → 用偏好里的助手（老师之前为该学科选定的）。
//      2b）查到记录且 prefID == ""       → 老师显式选了「系统默认(纯骨架)」→ 直接返回空串，
//                                          【绝不再走 RouteDefaultAssistant 兜底】（尊重显式选择）。
//      2c）查询出真实 DB 错误            → 记 Warn，降级到步骤3兜底（不阻塞对话）。
//      2d）无记录（从没选过）            → 落到步骤3兜底。
//
// 3）RouteDefaultAssistant（降为末位兜底）按「场景+学科+学段+可见性」自动解析默认助手：
//
//      3a）命中 → 用它。
//      3b）空串 → 返回空串（= 不替换第4层，沿用阶段原生角色，老行为）。
//
// 无论走哪条路，拿到最终 assistantID 后都经【同一条】加载路径
// （LoadActiveAssistantForUse：可见性校验 + is_active 校验 + 使用量埋点）取 full_prompt，
// 保证手动选择 / 偏好命中 / 默认挂载的加载口径、埋点口径完全一致。
// 偏好指向的助手若已被删/停用，LoadActiveAssistantForUse 会失败 → 末尾统一降级返回空串（纯骨架），
// 即「偏好失效后静默退回最朴素状态」，符合 PRD 强默认+逃生口哲学。
//
// 静默降级：assistantService 未注入 / 用户反查失败 / 助手加载失败 / full_prompt 为空 /
// 无可见默认助手 → 一律返回空串，绝不报错给老师、绝不阻塞对话。

// resolveAssistantPrompt 解析本轮应叠加的助手提示词。
//
// 最终优先级：
//  1. 专家模式当轮明确指定：手动通道；
//  2. 对话模式老师×学科偏好：手动通道；
//  3. 老师明确选择系统默认：不挂助手；
//  4. 无有效偏好时由RouteDefaultAssistant严格自动匹配。
//
// 手动通道：active + 可见性 + 同学科 + 当前场景，忽略具体年级。
// 自动通道：学科 + 具体年级 + 当前场景全部严格一致。
func (s *LessonPlanGenService) resolveAssistantPrompt(
	ctx context.Context,
	lp *models.LessonPlan,
	assistantID string,
	callerID string,
) (string, string) {
	if s.assistantService == nil || lp == nil {
		return "", ""
	}

	user, err := repository.FindUserByID(
		ctx,
		callerID,
	)
	if err != nil {
		lpGenLog.Warn(
			"加载用户角色失败，使用系统阶段骨架",
			"caller_id", callerID,
			"error", err,
		)
		return "", ""
	}

	actor := BuildActorFromClaims(
		ctx,
		callerID,
		user.Role,
	)
	// B2-B：使用教案教育域快照覆盖助手Actor。
	// 列表候选、手动助手、偏好助手和自动助手均消费同一个Actor，
	// 因此在任何助手解析发生之前统一覆盖一次即可完成运行链收口。
	applyLessonPlanEducationDomainToAssistantActor(
		actor,
		lp,
	)

	scene := stageCodeToAssistantScene(
		lp.CurrentStage,
	)

	loadManual := func(
		id string,
	) (*models.AIAssistant, error) {
		assistant, loadErr :=
			s.assistantService.
				LoadActiveAssistantForManualLessonUse(
					ctx,
					actor,
					id,
					lp.Subject,
					scene,
				)
		if loadErr != nil {
			return nil, loadErr
		}
		if strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
			return nil, errors.New("助手内容为空")
		}
		return assistant, nil
	}

	loadAuto := func(
		id string,
	) (*models.AIAssistant, error) {
		assistant, loadErr :=
			s.assistantService.
				LoadActiveAssistantForLessonUse(
					ctx,
					actor,
					id,
					lp.Subject,
					lp.Grade,
					scene,
				)
		if loadErr != nil {
			return nil, loadErr
		}
		if strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
			return nil, errors.New("助手内容为空")
		}
		return assistant, nil
	}

	assistantID = strings.TrimSpace(assistantID)

	if assistantID != "" {
		assistant, loadErr :=
			loadManual(assistantID)
		if loadErr != nil {
			lpGenLog.Warn(
				"老师指定的助手当前不可用，使用系统阶段骨架",
				"assistant_id", assistantID,
				"subject", lp.Subject,
				"grade", lp.Grade,
				"scene", scene,
				"error", loadErr,
			)
			return "", ""
		}
		return assistant.FullPrompt, assistant.Name
	}

	prefID, found, prefErr := repository.GetPref(
		ctx,
		callerID,
		lp.Subject,
	)
	if prefErr != nil {
		lpGenLog.Warn(
			"查询老师助手偏好失败，继续自动匹配",
			"caller_id", callerID,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"error", prefErr,
		)
	} else if found {
		prefID = strings.TrimSpace(prefID)

		if prefID == "" {
			return "", ""
		}

		assistant, loadErr := loadManual(prefID)
		if loadErr == nil {
			return assistant.FullPrompt, assistant.Name
		}

		lpGenLog.Info(
			"老师助手偏好不适用于当前学科或阶段，继续自动匹配",
			"plan_id", lp.ID,
			"pref_assistant_id", prefID,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"scene", scene,
			"error", loadErr,
		)
	}

	defaultID := RouteDefaultAssistant(
		ctx,
		s.assistantService,
		actor,
		lp.CurrentStage,
		lp.Subject,
		lp.Grade,
	)
	if defaultID == "" {
		return "", ""
	}

	assistant, loadErr := loadAuto(defaultID)
	if loadErr != nil {
		lpGenLog.Warn(
			"自动助手最终严格复核失败，使用系统阶段骨架",
			"assistant_id", defaultID,
			"subject", lp.Subject,
			"grade", lp.Grade,
			"scene", scene,
			"error", loadErr,
		)
		return "", ""
	}

	return assistant.FullPrompt, assistant.Name
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
