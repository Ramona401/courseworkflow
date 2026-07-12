package services

// lesson_plan_gen_chat_async.go — 教案生成服务：阶段化对话的异步流式处理
//
// 子轮一·A 拆分（从 lesson_plan_gen_service.go 搬出）：把"一轮 chat 请求的异步处理"整坨拆出。
// 子轮一·B 加 turnID（B2 轮次序号）：
//   processChatStageAsync 新增 turnID 形参，本函数内 7 处 SSE 广播全部带上 turnID
//   （thinking / chunk×2 / stage_output / message_done / suggested_actions / 重试可见性提示）；
//   调 broadcastSoftRetryNotice / broadcastError 时传入本轮 turnID；
//   派生的 checkAndInsertCoachAdvice 继承本轮 turnID（其 message_done 也带上）。
//   前端据 turnID 丢弃"过期轮次"的迟到回复，杜绝超时作废后旧回复污染新轮（B2 目标）。
//   turnID 为空（前端未传 client_turn_id）时，所有事件 ClientTurnID 回带空串，前端不过滤——行为零变化。
//
// 子轮一·B 重试可见性（依据业界最佳实践：等待期必须有可见反馈，10s 是注意力外限）：
//   首轮空流、决定自动重试时，广播一条 LPSSERetryNotice 事件（提示文案在 Content 字段），
//   前端据此把"思考中"替换为"刚才没接上、正在重试…"，让那几秒~十几秒的重试等待有解释，
//   避免老师误以为卡死而刷新。该事件带本轮 turnID，作废轮次的迟到重试提示同样被前端过滤。
//
// v189改动B补丁（流式气泡剥离 review 的 ```json 块）：
//   问题：review 阶段 AI 在对话里输出的评审报告末尾带 ```json 结构化块（供解析），
//   流式 message_done 下发 displayContent 时原仅剥了 suggested_actions 块，未剥 json 块，
//   导致对话气泡把 ```json{...}``` 原样显示给老师（改动B 只堵了落库 narrative 那条路，
//   流式气泡展示这条路是第二泄漏点）。
//   修法：构造 aiReply 前，仅当 currentStage=="review" 时，对 displayContent 追加调用
//   stripReviewJSONBlock（定义于同包 workshop_stage_extract.go，复用改动B已写好的剥离逻辑），
//   使 message_done 下发的文本干净。仅 review 阶段生效，其他阶段无 json 块不受影响。
//
// v204改动（Harness 输出采集 · 接入 harness_eval_samples）：
//   在本函数主流程全部完成后（message_done / suggested_actions 广播之后、函数收尾之前），
//   以【独立 goroutine + best-effort】把本轮的「输入画像 + 输出全文」采集落库，供后台
//   judge 分析与 admin 标注，驱动 harness 体检与进化。
//   铁律——采集是旁观者，绝不波及老师备课：
//     1) 放在主流程末尾，老师该收到的（message_done/芯片）已全部送达后才采集；
//     2) 独立 goroutine 异步执行，主流程不等待；内层 recover 兜 panic，写失败仅记 Warn；
//     3) 只读取主流程已算好的值（currentStage/lp/stageSystemPrompt/rawContent/assistantLabel/
//        result.ModelUsed/lpSchoolID），唯一新增计算是 is_downgraded（查学校境外授权），
//        且该查询 fail-safe（出错按未降级处理），绝不因采集让对话受任何影响；
//     4) 存的输出用 rawContent（AI 原始全文，含各围栏块）而非 displayContent——judge 需要
//        法医级原始证据（如"该出芯片却没出"只能从原始文本判断），不要美化过的展示版。
//
// 积分硬闸 batch (2026-07-04)：积分类错误短路。
//   背景：积分守卫在 CallAIStream 发请求前拦截（"积分余额不足…"），此前该错误
//   落进"空流自动重试 → 两轮空流软兜底"通道，被包装成"网络打了个盹，请重试"——
//   老师看不到真话、反复重试徒劳（实测截图三连撞同一句）。
//   修法：两处判断 isCreditGateError（定义于同包 lesson_plan_gen_helpers.go）：
//     1) 首轮失败即积分类 → 不重试（守卫是确定性拦截，重试必然再撞）、
//        不发"正在重试"提示，直接 broadcastCreditGateNotice 把守卫原话推给老师并返回；
//     2) 重试后仍失败且为积分类（如首轮网络错、重试时撞守卫）→ 同样走积分提示而非软兜底。
//   非积分类错误的重试/软兜底/硬错误路径逐字保留。
//
// 本文件职责：
//   1. processChatStageAsync     — 阶段模式异步AI流式回复
//   2. checkAndInsertCoachAdvice — 对话完成后停滞检测+教练建议插入（processChatStageAsync 派生，继承 turnID）

import (
	"context"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 阶段化对话（v84分层记忆 + v87教练集成 + v110助手注入 + v168/v169全委托 + 子轮一·B turnID）====================

// processChatStageAsync 阶段模式：异步处理AI流式回复
//
// 子轮一·B：新增 turnID 形参（本轮客户端轮次序号），本函数所有 SSE 广播与派生调用都带上它。
//
// v172说明（已撤销 done 补发）：SSE 是教案会话级共享长连接，发 done 会让前端关闭整条连接，
//
//	故不每轮发 done；「生成中」状态复位由前端 onMessageDone 处理。
// refMaterialInjectMaxRunes 参考资料注入 system prompt 的防御性上限(rune 计)。
// 前端长文档已压缩,正常远小于此;此处为兜底,防止异常超长文本撑爆上下文。
const refMaterialInjectMaxRunes = 8000

func (s *LessonPlanGenService) processChatStageAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	currentStageMsgs []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
	assistantPrompt string,
	assistantLabel string, // 助手轻量选择入口·可见性补丁:本轮匹配的助手名,随 message_done 回传前端显示
	fullGenerate bool,
	turnID string,
) {
	planID := lp.ID
	currentStage := lp.CurrentStage
	// v197：解析作者所属学校ID，供模型分流判定是否授权境外（查不到则空串→fail-closed降级境内）
	lpSchoolID, _ := repository.GetSchoolIDByUserID(ctx, lp.AuthorID)

	// 推送thinking状态（带本轮 turnID）
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType:    models.LPSSEThinking,
		PlanID:       planID,
		ClientTurnID: turnID,
		MessageID:    generateMsgID(),
	})

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), lessonPlanSceneCode, "", "", "")
	if err != nil {
		s.broadcastError(planID, turnID, "AI配置加载失败: "+err.Error())
		return
	}

	// 加载阶段系统提示词(v110:使用 V2 版本支持 assistantPrompt 注入)
	stageSystemPrompt, err := s.stageService.LoadStagePromptContextV2(ctx, lp, currentStage, assistantPrompt, userMsg.Content)
	if err != nil {
		lpGenLog.Warn("加载阶段提示词失败", "plan_id", planID, "stage", currentStage, "error", err)
		s.broadcastError(planID, turnID, "加载阶段配置失败，请刷新重试")
		return
	}

	// ---------- 参考资料附件(PDF/Word)注入 ----------
	// 老师上传的会话级参考资料，作为背景资料先落位（在写作/全委托块之前）。
	// 前端已在浏览器端提取(短文档=原文/长文档=压缩要点)，此处仅拼接 + 防御性 rune 截断兜底。
	// req.RefMaterial 为空(未挂附件)时行为 100% 不变。不落库、不复用。
	if rm := strings.TrimSpace(req.RefMaterial); rm != "" {
		if rr := []rune(rm); len(rr) > refMaterialInjectMaxRunes {
			rm = string(rr[:refMaterialInjectMaxRunes])
			lpGenLog.Info("参考资料超上限已截断注入", "plan_id", planID, "max_runes", refMaterialInjectMaxRunes)
		}
		stageSystemPrompt += "\n\n【备课参考资料（老师上传的附件）】\n" +
			"以下是老师为本次备课上传的参考资料，请在备课时充分参考其中的知识点、教学要求与重点，" +
			"但不要照搬，需结合本课实际情况取舍。\n" + rm + "\n"
		lpGenLog.Info("已注入参考资料附件", "plan_id", planID, "stage", currentStage, "ref_runes", len([]rune(rm)))
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
			EventType:    models.LPSSEChunk,
			PlanID:       planID,
			ClientTurnID: turnID,
			Chunk:        chunk,
		})
		return nil
	}, &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &lp.AuthorID,
		SchoolID:     schoolIDPtr(lpSchoolID),
	})
	if err != nil && chunkCount == 0 {
		// 积分硬闸 batch：积分类错误短路——守卫是确定性拦截（发请求前拒绝），
		// 重试必然再撞，且"正在重试/网络打盹"的包装会让老师徒劳重试。
		// 直接把守卫原话以友好消息推给老师并结束本轮。
		if isCreditGateError(err) {
			s.broadcastCreditGateNotice(ctx, planID, turnID, err.Error())
			lpGenLog.Warn("积分守卫拦截本轮对话，已推送积分提示（不重试）",
				"plan_id", planID, "stage", currentStage, "error", err)
			return
		}

		// 空流自动重试一次：仅当"一个字都没流出来"(chunkCount==0)时——干净的空，无半截内容、无重复风险。
		// 空流多为偶发（撞内容过滤/API打盹），二次请求绝大多数即恢复。
		lpGenLog.Warn("AI首轮空流，自动重试一次", "plan_id", planID, "stage", currentStage, "error", err)

		// 子轮一·B 重试可见性：广播重试提示，让前端把"思考中"换成"正在重试…"，
		// 使重试期间的等待有解释（避免老师误以为卡死而刷新）。带本轮 turnID。
		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType:    models.LPSSERetryNotice,
			PlanID:       planID,
			ClientTurnID: turnID,
			Content:      "刚才没接上话，正在帮你重试，请稍候…",
		})

		fullContent.Reset()
		chunkCount = 0
		time.Sleep(400 * time.Millisecond)
		result, err = aiClient.CallAIStream(aiCfg, stageSystemPrompt, userPrompt, func(chunk string) error {
			if strings.TrimSpace(chunk) == "" {
				return nil
			}
			chunkCount++
			fullContent.WriteString(chunk)
			GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{EventType: models.LPSSEChunk, PlanID: planID, ClientTurnID: turnID, Chunk: chunk})
			return nil
		}, &aiClient.TraceContext{SceneCode: lessonPlanSceneCode, LessonPlanID: &planID, UserID: &lp.AuthorID, SchoolID: schoolIDPtr(lpSchoolID)})
	}
	if err != nil {
		if chunkCount == 0 {
			if isCreditGateError(err) {
				// 积分硬闸 batch：重试路径上撞守卫（如首轮网络错、重试时被拦）→ 同样直达真话
				s.broadcastCreditGateNotice(ctx, planID, turnID, err.Error())
				lpGenLog.Warn("积分守卫拦截（重试路径），已推送积分提示",
					"plan_id", planID, "stage", currentStage, "error", err)
			} else {
				// 软兜底：两轮都没接上话 → 给老师一句人话（带本轮 turnID）
				s.broadcastSoftRetryNotice(ctx, planID, turnID)
				lpGenLog.Warn("AI两轮空流，已软兜底", "plan_id", planID, "stage", currentStage, "error", err)
			}
		} else {
			// 已有半截内容上屏后才失败：保留原硬错误提示（带本轮 turnID）
			s.broadcastError(planID, turnID, "AI回复失败: "+err.Error())
		}
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
			EventType:    models.LPSSEStageOutput,
			PlanID:       planID,
			ClientTurnID: turnID,
			StageData: &models.StageEventData{
				StageCode: currentStage,
				StageName: stageCodeToName(currentStage),
			},
		})
	}

	// 构造AI回复消息并保存
	// 迭代3.5 Phase B：先解析建议芯片，再把芯片块从展示文本剥离（老师不该看到 JSON）
	suggestedChips, hasChips := ParseSuggestedActions(rawContent)
	displayContent := StripSuggestedActionsBlock(rawContent)
	// v189改动B补丁：review阶段AI在对话里输出的评审报告末尾带```json块，
	// 流式 message_done 下发的气泡会原样显示给老师。此处复用 stripReviewJSONBlock
	// （定义于同包 workshop_stage_extract.go）把 json 块也剥掉，使下发文本干净。
	// 仅 review 阶段生效——其他阶段 AI 不输出 json 块，不受影响。
	if currentStage == "review" {
		displayContent = stripReviewJSONBlock(displayContent)
	}
	// v193补丁：write/revise 阶段 AI 可能在回复里输出 ```teacher_suggestion 创新建议块。
	// 落库路径（ExtractStructuredFromNaturalReply→narrative）已切该块，但气泡展示用的
	// displayContent 是独立构造的，此前只剥了 suggested_actions 块和 review 的 json 块，
	// 漏剥 teacher_suggestion 块，导致老师在对话气泡里看到原始 ```teacher_suggestion 围栏。
	// 修法：与落库路径对齐——切走围栏(pureContent)，再把建议以"💡 我的补充建议"格式拼回，
	// 使气泡显示「干净正文 + 💡 建议」而非原始围栏，且建议内容不丢失。
	if currentStage == "write" || currentStage == "revise" {
		pureDisplay, suggestionText := splitSuggestionBlock(displayContent)
		displayContent = appendSuggestionToNarrative(pureDisplay, suggestionText)
	}
	aiReply := s.parseAIReply(ctx, displayContent, lp)

	if err := s.appendMessage(ctx, planID, aiReply); err != nil {
		lpGenLog.Warn("写入AI消息失败", "plan_id", planID, "error", err)
	}

	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType:      models.LPSSEMessageDone,
		PlanID:         planID,
		ClientTurnID:   turnID,
		MessageID:      aiReply.ID,
		Message:        aiReply,
		AssistantLabel: assistantLabel, // 本轮实际注入(匹配)的助手名;空=纯骨架,前端据此显示或回退"自动匹配"
	})

	// 迭代3.5 Phase B：message_done 之后广播动态建议芯片（若有）。
	if hasChips {
		GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
			EventType:        models.LPSSESuggestedActions,
			PlanID:           planID,
			ClientTurnID:     turnID,
			MessageID:        aiReply.ID,
			SuggestedActions: suggestedChips,
		})
		lpGenLog.Info("已广播建议芯片", "plan_id", planID, "stage", currentStage, "chip_count", len(suggestedChips))
	}

	lpGenLog.Info("AI对话流式回复完成（v84分层记忆+v110助手注入+v168/v169全委托+子轮一·B turnID）",
		"plan_id", planID, "stage", currentStage,
		"tokens", result.TokensUsed, "latency_ms", result.LatencyMs,
		"chunks", chunkCount, "has_content", hasContent,
		"working_msgs", len(currentStageMsgs),
		"assistant_injected", assistantPrompt != "",
		"full_generate", fullGenerate)

	// ==================== v204：Harness 输出采集（best-effort，绝不阻塞主流程）====================
	// 主流程已全部完成（message_done / 芯片均已送达老师），此处把本轮的「输入画像 + 输出全文」
	// 落库到 harness_eval_samples，供后台 judge 分析与 admin 标注。
	// 独立 goroutine + recover 兜底 + 写失败仅 Warn——采集的任何问题都不波及老师备课。
	// 取值全部来自本函数已算好的变量；唯一新增计算是 is_downgraded（查学校境外授权，fail-safe）。
	// 存输出用 rawContent（AI 原始全文，含各围栏块），judge 需要法医级原始证据。
	go func(
		planID, stageCode, authorID, schoolID, assistantID, assistantLabel,
		modelUsed, sysPrompt, aiRaw string,
	) {
		defer func() {
			if r := recover(); r != nil {
				lpGenLog.Warn("harness采集 goroutine panic，已兜底忽略",
					"plan_id", planID, "stage", stageCode, "panic", r)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 计算 is_downgraded：学校未授权境外模型 → 本轮 generator 走境内降级。
		// fail-safe：schoolID 空或查询出错 → 按"未授权=降级"无法确定时，沿用 IsSchoolOverseasEnabled
		// 的 fail-closed 语义（出错返 false=未授权），即 is_downgraded=true 偏保守；
		// 但查询出错本身只记 Warn，绝不让采集失败。
		isDowngraded := false
		if enabled, qerr := repository.IsSchoolOverseasEnabled(bgCtx, schoolID); qerr != nil {
			lpGenLog.Warn("harness采集-查学校境外授权失败，is_downgraded 按未授权(降级)保守处理",
				"plan_id", planID, "school_id", schoolID, "error", qerr)
			isDowngraded = true // 查不到授权状态时保守标记为降级（与系统 fail-closed 一致）
		} else {
			isDowngraded = !enabled // 已授权=不降级；未授权=降级
		}

		in := repository.HarnessEvalSampleInput{
			LessonPlanID:  planID,
			StageCode:     stageCode,
			UserID:        authorID,
			SchoolID:      schoolID,
			ModelUsed:     modelUsed,
			IsDowngraded:  isDowngraded,
			AssistantID:   assistantID,
			AssistantName: assistantLabel,
			SystemPrompt:  sysPrompt,
			AIOutput:      aiRaw,
		}
		if err := repository.InsertHarnessEvalSample(bgCtx, in); err != nil {
			lpGenLog.Warn("harness采集-写入样本失败（不影响主流程）",
				"plan_id", planID, "stage", stageCode, "error", err)
			return
		}
		lpGenLog.Info("harness采集-样本已落库",
			"plan_id", planID, "stage", stageCode, "model", modelUsed,
			"is_downgraded", isDowngraded, "assistant", assistantLabel,
			"sys_prompt_len", len(sysPrompt), "ai_output_len", len(aiRaw))
	}(
		planID, currentStage, lp.AuthorID, lpSchoolID, req.AssistantID, assistantLabel,
		result.ModelUsed, stageSystemPrompt, rawContent,
	)

	// v87：对话完成后异步检测停滞，插入教练建议
	// v168/v169：全委托一次性出稿不需要停滞检测（本就是一次性完成），跳过避免误插建议
	// 子轮一·B：教练建议是本轮派生的尾巴，继承本轮 turnID（作废轮次的教练建议会被前端过滤掉）
	// 【已停用 B-3】对话流改由建议芯片做确定性引导，教练自动插话会与芯片语义冲突，故停用。
	// checkAndInsertCoachAdvice 函数定义保留便于回滚。
	// if !fullGenInjected {
	//         go s.checkAndInsertCoachAdvice(ctx, planID, currentStage, turnID)
	// }
	_ = fullGenInjected
}

// ==================== v87：停滞检测+教练建议插入 ====================

// checkAndInsertCoachAdvice 对话完成后检测停滞，插入教练建议
//
// 子轮一·B：新增 turnID 形参，继承自派生它的那一轮 processChatStageAsync。
// 它的 message_done 广播带上本轮 turnID——这样若本轮已被前端超时作废（turnID 推进），
// 这条迟到的教练建议会因 turnID 不匹配被前端丢弃，不污染新轮次的对话流（B2 核心目标）。
func (s *LessonPlanGenService) checkAndInsertCoachAdvice(ctx context.Context, planID string, stageCode string, turnID string) {
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
		EventType:    models.LPSSEMessageDone,
		PlanID:       planID,
		ClientTurnID: turnID,
		MessageID:    coachMsg.ID,
		Message:      coachMsg,
	})

	lpGenLog.Info("v87教练建议已插入",
		"plan_id", planID, "stage", stageCode,
		"user_rounds", stagnation.ConsecutiveRounds)
}
