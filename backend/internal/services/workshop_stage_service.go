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
// 大单元备课·批次二改动（本次）:
//   - LoadStagePromptContextV2 在课本注入层之后、课程大纲注入层之前，
//     新增「单元方案注入层」：老师显式挂载的 active 单元方案，五阶段全程注入。
//   - 课程大纲注入层开头加「让位」判断：若本教案已成功注入 active 单元方案，
//     则 analyze/design 两阶段不再注入课程大纲（有大单元就不要大纲——Yuhan 决策）。
//   - 单元方案的「按ID取+拼块」逻辑：GetUnitPlanByID + 校验 Status==active + BuildUnitPlanContext，
//     拼块函数在 unit_plan_match.go。显式挂载无需匹配/打分，任何缺失都静默跳过不阻断。

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
	"review": "请先检查上一阶段是否已生成完整的教案正文。如果教案正文尚未生成（只有讨论或开场白、没有成型的教学目标/教学过程/作业等完整内容），请明确提醒老师：当前还没有可评审的教案正文，需要先回到「教案撰写」阶段生成完整教案，不要凭空编写评审报告或给出分数。只有在确认教案正文已完整生成的前提下，才进行全面专业评审并输出评审报告（含各维度评分和改进建议）。",
	"revise": "我们进入修订定稿阶段了。请先简要介绍你是谁、你拿到了评审报告中的哪些改进建议，然后告诉我接下来你会怎么帮我修订教案。用友好的口吻，不超过200字。",
}

// autoTriggerStagesLean 对话模式专用的精简触发消息（无缝衔接：不重新自我介绍、不复述路线图）。
// design/write/review/revise 均提供对话模式版；review 版在“查有无完整教案再评审”的功能检查之外，补上对话模式包装（必须以「我们进入…阶段了。」开头，否则触发消息会原文漏成气泡）。
// 必须与 autoTriggerStages 一样以「我们进入X阶段了。」开头，以便前端 renderMessages 与
// isStageAutoTriggerContent 统一按「我们进入…阶段了。」过滤隐藏（专家模式原触发同样命中）。
var autoTriggerStagesLean = map[string]string{
	"design": "我们进入教学设计阶段了。【系统提示·对话模式】你和老师是一段不间断的连续对话，老师看不到任何阶段切换，绝不能重新自我介绍、绝不能复述流程路线图。你的上文里已经带着教学分析阶段定下的结论——学情、这节课学什么、教学重难点、以及老师已经选定的教学方向。请先读懂这些已定结论，开场第一句只用半句话点出前面已经定下的方向以示承接，然后立刻推进到本阶段真正该做的事：和老师一起把这节课的课堂环节和活动一步步排出来。特别注意：分析阶段已经定过的方向，绝不能再当成没决定的问题重新问一遍（例如再问偏体验还是偏概念、侧重赏析还是读写结合这类——已经定了，再问就是失忆，老师会很反感）。可参考的开场：既然前面定了走某某方向，我先把这节课的环节骨架搭出来——你看是先定导入，还是先敲核心活动？不超过90字。",
	"write":  "我们进入教案撰写阶段了。【系统提示·对话模式】你和老师是一段不间断的连续对话，老师看不到任何阶段切换，绝不能重新自我介绍、绝不能复述流程路线图。上文教学设计阶段刚刚已经把课堂环节框架和活动安排列清楚了，所以：绝不能再把这些环节逐条罗列/复述一遍（老师刚看过、会嫌啰嗦），也不要再要求老师把框架确认一遍，更不要再宣告一次“进入撰写/去写完整教案”。开场只用半句话承接（如“好，框架都齐了，这就开始落笔”），然后用一句话把写法选择交给老师——是一个环节一个环节地写、每写一两个环节就停下等他确认，还是一次性出完整正文（下面会给到“逐环节写”和“一键写出完整正文”两枚芯片，你不必自己造芯片）。不要自作主张直接开始写正文。不超过80字。",
	"review": "我们进入评审阶段了。【系统提示·对话模式】你和老师是一段不间断的连续对话，老师看不到任何阶段切换，绝不能重新自我介绍、绝不能复述流程路线图，尤其绝不能把这条系统指令原文复述或念出来。请你先在心里核对上文里到底有没有一份完整的教案正文（要有成型的教学目标、教学过程或各环节、作业等，而不是只有讨论、开场白或环节大纲）。若还没有完整教案，就只用一句话提醒老师：现在还没有可评审的教案正文，需要先回到「教案撰写」阶段把完整教案生成出来，并且绝不能编造评分。若已有完整教案，则不必寒暄，直接开始全面专业评审，输出评审报告（含各维度评分与改进建议）。",
	"revise": "我们进入修订定稿阶段了。【系统提示·对话模式】你和老师是一段不间断的连续对话，老师看不到任何阶段切换，绝不能重新自我介绍、绝不能复述流程路线图。你的上文里已经带着AI评审阶段给出的评审意见和改进建议。请先读懂，开场第一句只用半句话点出评审里最关键的那条改进建议以示承接，然后推进到修订该做的事。可参考的开场：评审里最该改的是某某，我先按这条来调整，可以吗？还是你想先改别的？不超过90字。",
}

// ==================== 服务结构体 ====================

// WorkshopStageService 阶段化备课工坊服务
// v87新增：aesKey字段用于AI教练LLM评估获取配置
type WorkshopStageService struct {
	recipeService *RecipeService
	genService    interface {
		Chat(ctx context.Context, req *models.LessonPlanChatRequest, callerID string) error
	}
	aesKey          string           // v87新增：AES密钥（用于LLM教练评估获取AI配置）
	textbookService *TextbookService // 迭代7B：课本服务（用于在阶段提示词注入课本原文），由 routes 层注入，可为 nil
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

// SetTextbookService 注入课本服务（由routes层调用）
// 迭代7B：用于 LoadStagePromptContextV2 在各阶段提示词中注入老师勾选的课本原文。
// 注入前若为 nil，则课本注入逻辑静默跳过，不影响原有流程。
func (s *WorkshopStageService) SetTextbookService(ts *TextbookService) {
	s.textbookService = ts
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
	return s.advanceStageWithComponents(ctx, lessonPlanID, targetStageCode, callerID, nil, false)
}

// AdvanceStageSilent 进入下一阶段（对话模式专用：跳过阶段质量教练过程打分）
//
// 与 AdvanceStageWithComponents 唯一区别：skipQualityEval=true，本次推进不触发
// asyncLLMEvaluateAndBroadcast 过程评估。对话模式（ConversationModePage）推进走此方法；
// 专家模式（WorkshopPage）仍走 AdvanceStage / AdvanceStageWithComponents，过程评估不变。
// selectedComponentIDs 仍照常透传给下一阶段。
func (s *WorkshopStageService) AdvanceStageSilent(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string, selectedComponentIDs []string) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(ctx, lessonPlanID, targetStageCode, callerID, selectedComponentIDs, true)
}

// AdvanceStageWithComponents 进入下一阶段（带用户选中的组件ID）
func (s *WorkshopStageService) AdvanceStageWithComponents(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string, selectedComponentIDs []string) (*models.StageConfigSnapshot, error) {
	return s.advanceStageWithComponents(ctx, lessonPlanID, targetStageCode, callerID, selectedComponentIDs, false)
}

func (s *WorkshopStageService) advanceStageWithComponents(ctx context.Context, lessonPlanID string, targetStageCode string, callerID string, selectedComponentIDs []string, skipQualityEval bool) (*models.StageConfigSnapshot, error) {
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
	if s.aesKey != "" && !skipQualityEval && s.leavingStageHasSubstantiveContent(ctx, lessonPlanID) {
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
		if skipQualityEval {
			if leanMsg, ok := autoTriggerStagesLean[targetStage.StageCode]; ok {
				triggerMsg = leanMsg
			}
		}
		wsLog.Info("自动触发阶段AI开场白", "plan_id", lessonPlanID, "stage", targetStage.StageCode, "lean", skipQualityEval)
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
	return s.LoadStagePromptContextV2(ctx, lp, stageCode, "", "")
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
//
// 大单元备课·批次二（本次新增两层外挂注入，均在 basePrompt 拼好之后、return 之前）：
//   - 单元方案注入层（五阶段全程）：老师显式挂载的 active 单元方案，整段注入并标记 unitPlanInjected。
//   - 课程大纲注入层（仅 analyze/design）：在原有基础上加「让位」判断——
//     若已注入 active 单元方案（unitPlanInjected=true），则跳过课程大纲（有大单元就不要大纲）。
func (s *WorkshopStageService) LoadStagePromptContextV2(
	ctx context.Context,
	lp *models.LessonPlan,
	stageCode string,
	assistantPrompt string,
	recentUserText string, // 技能路由 Phase1:透传给 BuildStageSystemPromptV2 第3层精排;空串→保底
) (string, error) {
	// 运行时重新校验lesson_plans.recipe_id。
	//
	// 存量教案可能已经关联了跨学科、跨年级或学段级配方。
	// 这里不修改数据库，只在本轮提示词装配中忽略不适用配方，
	// 防止错误配方继续提供阶段定义、教案结构、流程模式和上下文。
	var recipe *models.TeachingRecipe
	validRecipeID := ""

	if lp.RecipeID != nil &&
		strings.TrimSpace(*lp.RecipeID) != "" {
		candidate, recipeErr := loadRecipeForLesson(
			ctx,
			*lp.RecipeID,
			lp.Subject,
			lp.Grade,
		)
		if recipeErr != nil {
			wsLog.Warn(
				"教案关联配方不适用于当前学科或具体年级，本轮忽略",
				"plan_id", lp.ID,
				"stage", stageCode,
				"recipe_id", *lp.RecipeID,
				"subject", lp.Subject,
				"grade", lp.Grade,
				"error", recipeErr,
			)
		} else {
			recipe = candidate
			validRecipeID = candidate.ID
		}
	}

	// 只有严格有效的配方才能启用配方自定义阶段。
	isCustomStage := false
	if recipe != nil &&
		lp.StageConfig != "" &&
		lp.StageConfig != "[]" {
		var snapshots []models.StageConfigSnapshot
		if json.Unmarshal(
			[]byte(lp.StageConfig),
			&snapshots,
		) == nil {
			for _, snapshot := range snapshots {
				if snapshot.StageCode == stageCode &&
					snapshot.IsCustom {
					isCustomStage = true
					break
				}
			}
		}
	}

	// 加载系统阶段。若存量错误配方留下了自定义阶段代码，
	// 系统不存在同名阶段时安全退回系统“write”阶段，
	// 保证旧教案仍可继续运行，而不是因错误关联直接中断。
	loadSystemStage := func(
		code string,
	) (*models.WorkshopStage, error) {
		stage, stageErr := repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			code,
		)
		if stageErr == nil && stage != nil {
			return stage, nil
		}

		if code == "write" {
			return nil, stageErr
		}

		wsLog.Warn(
			"系统不存在当前阶段定义，退回系统教案撰写阶段",
			"plan_id", lp.ID,
			"requested_stage", code,
			"fallback_stage", "write",
			"error", stageErr,
		)

		return repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			"write",
		)
	}

	// 阶段定义优先级：
	// 有效配方自定义阶段 > 有效配方覆盖阶段 > 系统同名阶段 > 系统write兜底。
	var stage *models.WorkshopStage
	var err error

	switch {
	case isCustomStage:
		stage, err = repository.GetRecipeStageByCode(
			ctx,
			validRecipeID,
			stageCode,
		)
		if err != nil || stage == nil {
			wsLog.Warn(
				"有效配方的自定义阶段加载失败，退回系统阶段",
				"plan_id", lp.ID,
				"recipe_id", validRecipeID,
				"stage", stageCode,
				"error", err,
			)
			stage, err = loadSystemStage(stageCode)
		}

	case recipe != nil:
		stage, err = repository.GetStageByCode(
			ctx,
			models.StageSourceRecipe,
			stageCode,
		)
		if err != nil || stage == nil {
			stage, err = loadSystemStage(stageCode)
		}

	default:
		stage, err = loadSystemStage(stageCode)
	}

	if err != nil {
		return "", fmt.Errorf(
			"加载阶段定义失败: %w",
			err,
		)
	}
	if stage == nil {
		return "", errors.New(
			"加载阶段定义失败：阶段定义为空",
		)
	}

	// 只有严格有效的配方才能提供全局上下文、教案结构和备课模式。
	promptMode := models.PromptModeGuided
	lessonStructure := ""

	if recipe != nil {
		if recipe.PromptMode != "" {
			promptMode = recipe.PromptMode
		}
		if recipe.LessonStructure != "" &&
			recipe.LessonStructure != "[]" {
			lessonStructure = recipe.LessonStructure
		}
	}

	// 阶段级别的 promptMode 覆盖(StageConfig 快照中的配置优先)
	if recipe != nil &&
		lp.StageConfig != "" &&
		lp.StageConfig != "[]" {
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
	basePrompt := BuildStageSystemPromptV2(
		ctx, stage, recipe, priorOutputs,
		lp.Subject, lp.Grade, promptMode, lessonStructure, selectedCompIDs,
		assistantPrompt,
		recentUserText,
	)

	// 迭代7B：注入老师勾选的课本原文（新增一层，置于系统提示词末尾）
	// 数据来源：lesson_plans.textbook_page_ids（备课工坊新建时勾选并落库）。
	// 仅在 textbookService 已注入、且本教案确有关联课本时拼接；任何缺失都静默跳过，
	// 不影响原有六层提示词。课本原文用 OCR 文字（未识别的页会带"未识别"占位提示）。
	if s.textbookService != nil && strings.TrimSpace(lp.TextbookPageIDs) != "" && lp.TextbookPageIDs != "[]" {
		var tbIDs []string
		if uErr := json.Unmarshal([]byte(lp.TextbookPageIDs), &tbIDs); uErr == nil && len(tbIDs) > 0 {
			tbContext := s.textbookService.BuildTextbookContext(ctx, tbIDs)
			if strings.TrimSpace(tbContext) != "" {
				basePrompt += "\n" + tbContext
				wsLog.Info("已注入课本原文上下文",
					"plan_id", lp.ID, "stage", stageCode, "textbook_count", len(tbIDs))
			}
		}
	} else if strings.TrimSpace(lp.TextbookPageIDs) != "" && lp.TextbookPageIDs != "[]" {
		// 防御性告警：教案确实关联了课本图，却因 textbookService 未注入而无法注入课本原文。
		// 这正是迭代7B曾踩过的"双实例导致 textbookService==nil"坑——一旦再发生，
		// 这条 Warn 会在日志里立刻暴露，不必再靠老师反馈"AI说看不到课本"才发现。
		wsLog.Warn("应注入课本但课本服务未注入（textbookService==nil），本次跳过课本原文注入",
			"plan_id", lp.ID, "stage", stageCode, "textbook_page_ids", lp.TextbookPageIDs)
	}

	// 单元方案注入层（大单元备课·批次二·注入）——置于课本层之后、课程大纲层之前。
	//
	// 与课程大纲的区别：单元方案是老师「显式挂载」的（lesson_plans.unit_plan_id），
	// 是「这堂课所属大单元的纲」，五个阶段（analyze/design/write/review/revise）全程注入，
	// 不做任何匹配/打分（显式挂载无需匹配，直接按挂载的 ID 取那一份）。
	//
	// 注入成功后置 unitPlanInjected=true，供下方课程大纲层做「让位」判断
	// （有大单元就不要大纲——Yuhan 决策）。
	//
	// 任何缺失（未挂载/取不到/非 active/拼块为空）都静默跳过，记 Info/Warn，绝不阻断备课。
	unitPlanInjected := false
	if lp.UnitPlanID != nil && strings.TrimSpace(*lp.UnitPlanID) != "" {
		up, upErr := repository.GetUnitPlanByID(ctx, *lp.UnitPlanID)
		if upErr != nil {
			// 取不到（已被物理删除等）：无外键约束设计的预期情形，静默跳过。
			wsLog.Warn("已挂载单元方案但查询失败，跳过单元方案注入",
				"plan_id", lp.ID, "stage", stageCode, "unit_plan_id", *lp.UnitPlanID, "error", upErr)
		} else if up.Status != models.UnitPlanStatusActive {
			// 挂载的单元方案不是 active（草稿未定稿 / 已归档软删）：不注入半成品或已删方案。
			wsLog.Info("已挂载单元方案但非 active 状态，跳过单元方案注入",
				"plan_id", lp.ID, "stage", stageCode, "unit_plan_id", *lp.UnitPlanID, "status", up.Status)
		} else if upContext := BuildUnitPlanContext(up); strings.TrimSpace(upContext) != "" {
			// active 且拼块非空：五阶段全程注入，并标记已注入（供大纲层让位）。
			basePrompt += upContext
			unitPlanInjected = true
			wsLog.Info("已注入单元方案上下文（大单元，五阶段全程）",
				"plan_id", lp.ID, "stage", stageCode,
				"unit_plan_id", up.ID, "unit", up.Unit, "unit_theme", up.UnitTheme)
		}
	}

	// 课程大纲注入（大单元备课能力·批次一·注入）
	// 仅在「教学分析 analyze」「教学设计 design」两阶段注入——撰写/评审/修订不需要全册大纲，省 token 又不干扰。
	// 匹配：按学科粗筛同学科 active 大纲，再用「学段范围覆盖」打分取最贴合一份；一份都没命中 → 静默跳过
	//（正常单课备课，不报错不提示，符合"没匹配上就不注入"的设计）。
	//
	// 大单元备课·批次二「让位」：若本教案已成功注入 active 单元方案（unitPlanInjected=true），
	// 则 analyze/design 两阶段不再注入课程大纲——有大单元就不要大纲（Yuhan 决策）。
	// 单元方案是更具体的「本课所属大单元的纲」，与册级大纲并存会冗余且抢 token，故让位。
	if (stageCode == "analyze" || stageCode == "design") && !unitPlanInjected {
		if strings.TrimSpace(lp.Subject) != "" && strings.TrimSpace(lp.Grade) != "" {
			candidates, coErr := repository.ListActiveOutlinesBySubject(ctx, lp.Subject)
			if coErr != nil {
				wsLog.Warn("查询课程大纲候选失败，跳过大纲注入",
					"plan_id", lp.ID, "stage", stageCode, "subject", lp.Subject, "error", coErr)
			} else if lp.CourseOutlinePublisher == nil {
				// 教材版本增强（Yuhan 决策）：大纲注入改为「老师在备课首屏显式选定教材版本」。
				// CourseOutlinePublisher == nil 表示老师没选版本 / 没关联大纲 → 静默不注入，
				// 不再像旧逻辑那样自动按学段匹配。没选版本就是不注入，正常单课备课。
				wsLog.Info("教案未选定课程大纲教材版本，跳过大纲注入",
					"plan_id", lp.ID, "stage", stageCode, "subject", lp.Subject, "plan_grade", lp.Grade)
			} else if hits := MatchOutlinesByPublisher(lp.Grade, *lp.CourseOutlinePublisher, candidates); len(hits) > 0 {
				// 版本精确注入（Yuhan 决策）：只注入 publisher == 老师选定版本 的大纲，零跨版本兜底。
				// 选定版本下、学段相交命中的大纲多份全注入（同年级上下册等）。
				// *lp.CourseOutlinePublisher 为空串=老师选了"通用/不限版本"，则精确匹配 publisher 为空串的大纲。
				if outlineCtx := BuildCourseOutlinesContext(hits); strings.TrimSpace(outlineCtx) != "" {
					basePrompt += outlineCtx
					// 收集命中大纲的标题，便于日志核对"到底注入了哪几份"
					hitTitles := make([]string, 0, len(hits))
					for _, h := range hits {
						hitTitles = append(hitTitles, h.Title)
					}
					wsLog.Info("已注入课程大纲上下文（多份）",
						"plan_id", lp.ID, "stage", stageCode,
						"subject", lp.Subject, "plan_grade", lp.Grade,
						"outline_count", len(hits),
						"outline_titles", strings.Join(hitTitles, " | "))
				}
			}
		}
	} else if (stageCode == "analyze" || stageCode == "design") && unitPlanInjected {
		// 让位日志：明确记录"因已注入单元方案而跳过课程大纲"，便于核对优先级生效。
		wsLog.Info("已注入单元方案，课程大纲让位（本阶段不注入大纲）",
			"plan_id", lp.ID, "stage", stageCode)
	}

	// 班级学情注入层（差异化教学·批次3·注入）——置于课程大纲层之后、return 之前。
	//
	// 与单元方案的区别：班级学情是老师「显式挂载」的（lesson_plans.class_profile_id），
	// 描述的是这节课要面对的真实学生群体，仅在 analyze / design / write 三阶段注入
	// （review/revise 是评分与定稿，不需要学情；省 token 又不干扰）。
	//
	// 独立不让位（与单元方案/课程大纲维度正交——Yuhan 决策）：本层不读、不改 unitPlanInjected，
	// 无论单元方案/课程大纲是否注入都照常注入。一节课可以同时挂单元方案 + 班级学情，两者叠加。
	//
	// 取数：repository.GetClassProfileByID（静态包级调用，与单元方案层同形态，故两个 wsStage 实例
	// 天然都注入，无需 Setter）。校验 status==active 且 created_by==教案作者（归属兜底，防越权注入他人卡）。
	// 拼块函数 BuildClassProfileContext 只用四大段群体结论（合规：个体明细永不进注入链路）。
	// 任何缺失（未挂载/取不到/非 active/非本人/拼块为空）都静默跳过，记 Info/Warn，绝不阻断备课。
	if (stageCode == "analyze" || stageCode == "design" || stageCode == "write") &&
		lp.ClassProfileID != nil && strings.TrimSpace(*lp.ClassProfileID) != "" {
		cp, cpErr := repository.GetClassProfileByID(ctx, *lp.ClassProfileID)
		if cpErr != nil {
			// 取不到（已被软删/物理删除等）：无外键约束设计的预期情形，静默跳过。
			wsLog.Warn("已挂载班级学情卡但查询失败，跳过班级学情注入",
				"plan_id", lp.ID, "stage", stageCode, "class_profile_id", *lp.ClassProfileID, "error", cpErr)
		} else if cp.Status != models.ClassProfileStatusActive {
			// 已归档（软删）：不注入已删卡。
			wsLog.Info("已挂载班级学情卡但非 active 状态，跳过班级学情注入",
				"plan_id", lp.ID, "stage", stageCode, "class_profile_id", *lp.ClassProfileID, "status", cp.Status)
		} else if cp.CreatedBy != lp.AuthorID {
			// 归属兜底：只注入教案作者本人的班级卡，绝不注入他人卡（纵深防御，防越权挂载/串台）。
			wsLog.Warn("挂载的班级学情卡归属与教案作者不一致，跳过班级学情注入",
				"plan_id", lp.ID, "stage", stageCode, "class_profile_id", *lp.ClassProfileID,
				"card_owner", cp.CreatedBy, "plan_author", lp.AuthorID)
		} else if cpContext := BuildClassProfileContext(cp); strings.TrimSpace(cpContext) != "" {
			// active + 本人 + 拼块非空：analyze/design/write 三阶段注入，独立不让位。
			basePrompt += cpContext
			wsLog.Info("已注入班级学情上下文（差异化教学，三阶段 analyze/design/write）",
				"plan_id", lp.ID, "stage", stageCode,
				"class_profile_id", cp.ID, "class_name", cp.ClassName)
		}
	}
	return basePrompt, nil
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

// ==================== 第一刀新增：空阶段守门小助手 ====================

// leavingStageHasSubstantiveContent 判断“正在离开的阶段”是否有老师的真实发言。
// 排除系统自动注入的阶段开场白触发消息（以 user 角色落库、但非老师真实输入）。
// 没有真实发言 = 空阶段 = 不评，避免给只有开场白的阶段甩 0 分。必须在 current_stage
// 翻页之前调用——此刻 GetCurrentStageMessages 返回的正是“离开阶段”的消息。
// 读取出错时保守返回 true（宁可评、不漏评）。
func (s *WorkshopStageService) leavingStageHasSubstantiveContent(ctx context.Context, lessonPlanID string) bool {
	msgs, err := repository.GetCurrentStageMessages(ctx, lessonPlanID)
	if err != nil {
		return true
	}
	for _, m := range msgs {
		if m.Role == "user" && !isStageAutoTriggerContent(m.Content) {
			return true
		}
	}
	return false
}

// isStageAutoTriggerContent 是否为系统自动注入的阶段开场白触发消息（与前端
// renderMessages 过滤口径一致）。这些消息驱动 AI 开场白，但不是老师真实输入。
func isStageAutoTriggerContent(content string) bool {
	c := strings.TrimSpace(content)
	if strings.HasPrefix(c, "我们进入") && strings.Contains(c, "阶段了。") {
		return true
	}
	if strings.HasPrefix(c, "请先检查上一阶段") {
		return true
	}
	return false
}
