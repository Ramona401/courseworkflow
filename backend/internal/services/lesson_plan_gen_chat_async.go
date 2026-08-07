package services

// lesson_plan_gen_chat_async.go — 教案阶段对话异步执行主链。
//
// 构建上下文、流式生成、正式产物提交与SSE终态；所有事件携带turnID。
// 原格式Word修订专用提取与校验位于 lesson_plan_word_revision_stream.go。

import (
	"context"
	"errors"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// refMaterialInjectMaxRunes限制附件进入system prompt的最大字符数。
const refMaterialInjectMaxRunes = 8000

// processChatStageAsync 阶段模式：异步处理AI回复。
func (s *LessonPlanGenService) processChatStageAsync(
	ctx context.Context,
	lp *models.LessonPlan,
	userMsg *models.ConversationMessage,
	currentStageMsgs []*models.ConversationMessage,
	req *models.LessonPlanChatRequest,
	assistantPrompt string,
	assistantLabel string,
	assistantReceipt *models.AssistantContextReceipt,
	fullGenerate bool,
	turnID string,
) {
	planID := lp.ID
	currentStage := lp.CurrentStage

	// 解析作者所属学校ID，供模型分流判定是否授权境外。
	// 查不到为空串，模型策略按fail-closed收敛境内模型。
	lpSchoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			lp.AuthorID,
		)

	GlobalLPSSEHub.Broadcast(
		planID,
		models.LPSSEEvent{
			EventType:    models.LPSSEThinking,
			PlanID:       planID,
			ClientTurnID: turnID,
			MessageID:    generateMsgID(),
		},
	)

	aiCfg, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		lessonPlanSceneCode,
		"",
		"",
		"",
	)
	if err != nil {
		s.broadcastError(
			planID,
			turnID,
			"AI配置加载失败: "+err.Error(),
		)
		return
	}

	turnPlan := buildLessonPlanTurnContextPlan(
		lp,
		req,
	)

	if err := validateLessonPlanTurnContextPlan(
		ctx,
		lp,
		turnPlan,
	); err != nil {
		lpGenLog.Warn(
			"单轮上下文前置校验未通过",
			"plan_id", planID,
			"stage", currentStage,
			"error", err,
		)
		s.broadcastError(
			planID,
			turnID,
			err.Error(),
		)
		return
	}

	ctx = withLessonPlanTurnContextPlan(
		ctx,
		turnPlan,
	)

	var (
		stageSystemPrompt string
		contextReceipt    *models.ContextReceipt
	)

	if turnPlan.FormalArtifact {
		stageSystemPrompt,
			contextReceipt,
			err =
			s.stageService.
				LoadStagePromptContextWithReceipt(
					ctx,
					lp,
					currentStage,
					assistantPrompt,
					userMsg.Content,
					assistantReceipt,
				)
	} else {
		stageSystemPrompt,
			err =
			s.stageService.
				LoadStagePromptContextV2(
					ctx,
					lp,
					currentStage,
					assistantPrompt,
					userMsg.Content,
				)

		contextReceipt =
			buildLightweightLessonPlanContextReceipt(
				lp,
				req,
				assistantReceipt,
				turnPlan,
			)
	}

	if err != nil {
		lpGenLog.Warn(
			"加载阶段提示词失败",
			"plan_id", planID,
			"stage", currentStage,
			"error", err,
		)

		errorMessage :=
			"加载本轮备课资料失败，请检查已关联资源后重试"
		if errors.Is(
			err,
			ErrLessonPlanKnowledgeLineageAnalyzeRequired,
		) {
			errorMessage =
				"当前教案关联了课程大纲，但还没有可用知识脉络。请先切回教学分析阶段，确认课文范围、教学目标和知识点，再正式进入后续阶段。"
		}

		s.broadcastError(
			planID,
			turnID,
			errorMessage,
		)
		return
	}

	applyLessonPlanTurnPlanToReceipt(
		contextReceipt,
		lp,
		req,
		turnPlan,
	)

	stageSystemPrompt +=
		buildLessonPlanSourceAuthorityPrompt(
			turnPlan,
		)

	if turnPlan.UseRefMaterial {
		rm := strings.TrimSpace(
			req.RefMaterial,
		)
		if rm != "" {
			originalRunes := len([]rune(rm))

			if runes := []rune(rm); len(runes) >
				refMaterialInjectMaxRunes {
				rm = string(
					runes[:refMaterialInjectMaxRunes],
				)

				lpGenLog.Info(
					"参考资料超上限已截断注入",
					"plan_id", planID,
					"max_runes",
					refMaterialInjectMaxRunes,
				)
			}

			stageSystemPrompt +=
				"\n\n【备课参考资料（老师上传的附件）】\n" +
					"以下内容是老师提供的本轮事实与教学证据。篇名、正文、题目、页码、实体和数据必须忠实引用；不得用模型常识替换。\n" +
					rm +
					"\n"

			if contextReceipt != nil {
				contextReceipt.RefMaterial =
					&models.MaterialContextReceipt{
						Status: models.ContextReceiptLoaded,
						CharacterCount: len(
							[]rune(rm),
						),
					}

				if originalRunes >
					refMaterialInjectMaxRunes {
					contextReceipt.
						RefMaterial.
						Reason =
						"参考资料超过运行预算，本轮已安全截断后读取"
				}
			}

			lpGenLog.Info(
				"已按单轮规划注入参考资料附件",
				"plan_id", planID,
				"stage", currentStage,
				"ref_runes", len([]rune(rm)),
			)
		}
	}

	// 生成阶段提示词已按业务职责拆到lesson_plan_gen_word_prompt.go。
	stageSystemPrompt, fullGenInjected :=
		s.appendLessonPlanStageGenerationPrompt(
			ctx,
			lp,
			currentStage,
			stageSystemPrompt,
			fullGenerate,
		)

	streamWordRevision :=
		s.shouldStreamLessonPlanWordRevision(
			ctx,
			lp,
			currentStage,
			turnPlan,
		)

	if contextReceipt != nil {
		contextReceipt.SystemPromptRunes =
			len(
				[]rune(
					stageSystemPrompt,
				),
			)
	}

	// 普通讨论使用短工作记忆，正式产物按需加载前序阶段摘要。
	workingMessages := currentStageMsgs
	episodicSummary := ""
	priorOutputCount := 0

	if turnPlan.UsePriorOutputs {
		allOutputs, _ :=
			repository.ListStageOutputs(
				ctx,
				planID,
			)

		var priorOutputs []*models.WorkshopStageOutput
		for _, output := range allOutputs {
			if output.StageCode ==
				currentStage {
				break
			}
			priorOutputs = append(
				priorOutputs,
				output,
			)
		}

		priorOutputCount = len(priorOutputs)
		episodicSummary =
			repository.
				BuildEpisodicSummaryFromOutputs(
					priorOutputs,
				)
	} else {
		workingMessages =
			limitLessonPlanWorkingMessages(
				currentStageMsgs,
			)
	}

	userPrompt :=
		BuildStageChatPromptV2(
			lp,
			workingMessages,
			episodicSummary,
			userMsg,
		)

	lpGenLog.Info(
		"分层记忆上下文构建完成",
		"plan_id", planID,
		"stage", currentStage,
		"working_msgs", len(workingMessages),
		"episodic_len", len(episodicSummary),
		"prior_stages", priorOutputCount,
		"formal_artifact", turnPlan.FormalArtifact,
		"use_recipe", turnPlan.UseRecipe,
		"use_textbook", turnPlan.UseTextbook,
		"use_unit_plan", turnPlan.UseUnitPlan,
		"use_raw_course_outline", turnPlan.UseRawCourseOutline,
		"use_knowledge_lineage", turnPlan.UseKnowledgeLineage,
		"use_context_capsule", turnPlan.UseContextCapsule,
		"use_class_profile", turnPlan.UseClassProfile,
		"use_ref_material", turnPlan.UseRefMaterial,
		"context_plan_reason", turnPlan.Reason,
		"assistant_injected",
		assistantPrompt != "",
		"full_generate", fullGenerate,
		"full_gen_injected", fullGenInjected,
		"assistant_prompt_runes",
		len([]rune(assistantPrompt)),
		"system_prompt_runes",
		len([]rune(stageSystemPrompt)),
		"user_prompt_runes",
		len([]rune(userPrompt)),
	)

	traceContext := &aiClient.TraceContext{
		SceneCode:    lessonPlanSceneCode,
		LessonPlanID: &planID,
		UserID:       &lp.AuthorID,
		SchoolID: schoolIDPtr(
			lpSchoolID,
		),
	}

	var (
		result             *aiClient.CallResult
		rawContent         string
		chunkCount         int
		evidenceHarnessRun *lessonPlanEvidenceHarnessRun
	)

	deferArtifactDisplay :=
		turnPlan.FormalArtifact &&
			(currentStage == "write" || currentStage == "revise") &&
			!streamWordRevision

	if turnPlan.BlockingEvidenceHarness &&
		!streamWordRevision {
		stopEvidenceProgress :=
			startLessonPlanEvidenceHarnessProgress(
				ctx,
				planID,
				turnID,
			)

		evidenceHarnessRun, err =
			s.generateLessonPlanEvidenceGuardedReply(
				ctx,
				aiCfg,
				stageSystemPrompt,
				userPrompt,
				traceContext,
				turnPlan,
			)

		stopEvidenceProgress()
		if err != nil {
			if isCreditGateError(err) {
				s.broadcastCreditGateNotice(
					ctx,
					planID,
					turnID,
					err.Error(),
				)
				return
			}

			switch {
			case errors.Is(err, ErrLessonPlanEvidenceHarnessRejected):
				lpGenLog.Warn(
					"多证据正式Harness二次仍未通过，内容未展示未保存",
					"plan_id", planID,
					"stage", currentStage,
					"error", err,
				)
				s.broadcastError(
					planID,
					turnID,
					"本轮正式内容没有通过课本、附件与教学依据一致性校验，系统已阻止展示和保存。请核对资料后重试。",
				)
			case errors.Is(err, ErrLessonPlanEvidenceHarnessUnavailable):
				lpGenLog.Warn(
					"多证据正式Harness暂时不可用，本轮已fail-closed",
					"plan_id", planID,
					"stage", currentStage,
					"error", err,
				)
				s.broadcastError(
					planID,
					turnID,
					"正式资料一致性校验暂时不可用。为避免错误内容，本轮没有展示或保存，请稍后重试。",
				)
			default:
				s.broadcastError(
					planID,
					turnID,
					"AI回复生成失败: "+err.Error(),
				)
			}
			return
		}

		if evidenceHarnessRun == nil ||
			evidenceHarnessRun.Result == nil ||
			strings.TrimSpace(evidenceHarnessRun.Result.Content) == "" {
			s.broadcastError(
				planID,
				turnID,
				"正式资料一致性校验没有产生可展示内容，请稍后重试",
			)
			return
		}

		result = evidenceHarnessRun.Result
		rawContent = result.Content
		if !deferArtifactDisplay {
			chunkCount = broadcastBufferedCourseOutlineReply(
				planID,
				turnID,
				rawContent,
			)
		}

		lpGenLog.Info(
			"多证据正式Harness已通过",
			"plan_id", planID,
			"stage", currentStage,
			"repaired", evidenceHarnessRun.Repaired,
			"chunks", chunkCount,
			"display_deferred", deferArtifactDisplay,
		)
	} else {
		// 非正式Harness路径：保持真实流式调用和一次空流重试。
		var fullContent strings.Builder

		result, err = aiClient.CallAIStream(
			aiCfg,
			stageSystemPrompt,
			userPrompt,
			func(chunk string) error {
				if strings.TrimSpace(
					chunk,
				) == "" {
					return nil
				}

				chunkCount++
				fullContent.WriteString(
					chunk,
				)

				GlobalLPSSEHub.Broadcast(
					planID,
					models.LPSSEEvent{
						EventType:    models.LPSSEChunk,
						PlanID:       planID,
						ClientTurnID: turnID,
						Chunk:        chunk,
					},
				)

				return nil
			},
			traceContext,
		)

		if err != nil &&
			chunkCount == 0 {
			if isCreditGateError(err) {
				s.broadcastCreditGateNotice(
					ctx,
					planID,
					turnID,
					err.Error(),
				)

				lpGenLog.Warn(
					"积分守卫拦截本轮对话，已推送积分提示",
					"plan_id", planID,
					"stage", currentStage,
					"error", err,
				)
				return
			}

			lpGenLog.Warn(
				"AI首轮空流，自动重试一次",
				"plan_id", planID,
				"stage", currentStage,
				"error", err,
			)

			GlobalLPSSEHub.Broadcast(
				planID,
				models.LPSSEEvent{
					EventType:    models.LPSSERetryNotice,
					PlanID:       planID,
					ClientTurnID: turnID,
					Content:      "刚才没接上话，正在帮你重试，请稍候…",
				},
			)

			fullContent.Reset()
			chunkCount = 0
			time.Sleep(
				400 *
					time.Millisecond,
			)

			result, err =
				aiClient.CallAIStream(
					aiCfg,
					stageSystemPrompt,
					userPrompt,
					func(
						chunk string,
					) error {
						if strings.TrimSpace(
							chunk,
						) == "" {
							return nil
						}

						chunkCount++
						fullContent.
							WriteString(
								chunk,
							)

						GlobalLPSSEHub.
							Broadcast(
								planID,
								models.LPSSEEvent{
									EventType:    models.LPSSEChunk,
									PlanID:       planID,
									ClientTurnID: turnID,
									Chunk:        chunk,
								},
							)

						return nil
					},
					traceContext,
				)
		}

		if err != nil {
			if chunkCount == 0 {
				if isCreditGateError(
					err,
				) {
					s.broadcastCreditGateNotice(
						ctx,
						planID,
						turnID,
						err.Error(),
					)

					lpGenLog.Warn(
						"积分守卫拦截重试路径",
						"plan_id", planID,
						"stage", currentStage,
						"error", err,
					)
				} else {
					s.broadcastSoftRetryNotice(
						ctx,
						planID,
						turnID,
					)

					lpGenLog.Warn(
						"AI两轮空流，已软兜底",
						"plan_id", planID,
						"stage", currentStage,
						"error", err,
					)
				}
			} else {
				s.broadcastError(
					planID,
					turnID,
					"AI回复失败: "+
						err.Error(),
				)
			}
			return
		}

		if result == nil {
			s.broadcastError(
				planID,
				turnID,
				"AI回复结果为空，请稍后重试",
			)
			return
		}

		rawContent = result.Content
		if rawContent == "" {
			rawContent =
				fullContent.String()
		}
	}

	structuredJSON,
		narrative,
		hasContent,
		artifactExtractErr :=
		s.extractLessonPlanStageArtifact(
			ctx,
			lp,
			currentStage,
			rawContent,
			streamWordRevision,
		)

	if artifactExtractErr != nil &&
		streamWordRevision {
		rawContent,
			structuredJSON,
			narrative,
			result,
			artifactExtractErr =
			s.repairLessonPlanWordRevisionArtifact(
				ctx,
				lp,
				userMsg.Content,
				rawContent,
				aiCfg,
				traceContext,
				result,
				turnID,
			)
		hasContent =
			artifactExtractErr == nil
	}

	if artifactExtractErr != nil {
		lpGenLog.Warn(
			"原格式Word流式修订未形成可提交候选",
			"plan_id", planID,
			"stage", currentStage,
			"error", artifactExtractErr,
		)

		if streamWordRevision {
			s.finishRejectedLessonPlanWordRevision(
				ctx,
				lp,
				rawContent,
				artifactExtractErr,
				result,
				assistantLabel,
				turnID,
			)
		} else {
			s.broadcastError(
				planID,
				turnID,
				lessonPlanContentMutationPublicMessage(
					artifactExtractErr,
				),
			)
		}
		return
	}

	commitOutcome :=
		s.commitLessonPlanStageArtifactWithWordHarness(
			ctx,
			lp,
			currentStage,
			structuredJSON,
			narrative,
			rawContent,
			userMsg.Content,
			turnID,
			assistantLabel,
			hasContent,
			streamWordRevision,
			aiCfg,
			traceContext,
			result,
		)

	rawContent =
		commitOutcome.RawContent
	structuredJSON =
		commitOutcome.StructuredJSON
	narrative =
		commitOutcome.Narrative
	result =
		commitOutcome.Result
	artifactSaved :=
		commitOutcome.ArtifactSaved
	artifactCommitted :=
		commitOutcome.ArtifactCommitted

	if commitOutcome.Stop {
		return
	}

	if deferArtifactDisplay {
		if !artifactCommitted {
			s.broadcastError(
				planID,
				turnID,
				"AI没有产生可安全保存的完整教案，本轮内容未展示也未发布。",
			)
			return
		}
		chunkCount = broadcastBufferedCourseOutlineReply(
			planID,
			turnID,
			rawContent,
		)
	}

	suggestedChips, hasChips :=
		ParseSuggestedActions(
			rawContent,
		)

	displayContent :=
		StripSuggestedActionsBlock(
			rawContent,
		)

	if currentStage == "review" {
		displayContent =
			stripReviewJSONBlock(
				displayContent,
			)
	}

	if currentStage == "write" ||
		currentStage == "revise" {
		pureDisplay,
			suggestionText :=
			splitSuggestionBlock(
				displayContent,
			)

		displayContent =
			appendSuggestionToNarrative(
				pureDisplay,
				suggestionText,
			)
	}

	aiReply :=
		s.parseAIReply(
			ctx,
			displayContent,
			lp,
		)

	if contextReceipt != nil ||
		evidenceHarnessRun != nil ||
		artifactCommitted {
		if aiReply.Metadata == nil {
			aiReply.Metadata =
				make(
					map[string]interface{},
				)
		}
	}

	if contextReceipt != nil {
		aiReply.Metadata["context_receipt"] = contextReceipt
	}

	if evidenceHarnessRun != nil {
		aiReply.Metadata["evidence_harness"] = map[string]interface{}{
			"status":      "passed",
			"pre_display": !deferArtifactDisplay,
			"repaired":    evidenceHarnessRun.Repaired,
		}
	}

	if artifactCommitted {
		aiReply.Metadata["content_committed"] = true
		aiReply.Metadata["content_version"] = lp.Version
	}

	if err :=
		s.appendMessage(
			ctx,
			planID,
			aiReply,
		); err != nil {
		lpGenLog.Warn(
			"写入AI消息失败",
			"plan_id", planID,
			"error", err,
		)
	}

	GlobalLPSSEHub.Broadcast(
		planID,
		models.LPSSEEvent{
			EventType:      models.LPSSEMessageDone,
			PlanID:         planID,
			ClientTurnID:   turnID,
			MessageID:      aiReply.ID,
			Message:        aiReply,
			AssistantLabel: assistantLabel,
		},
	)

	if hasChips {
		GlobalLPSSEHub.Broadcast(
			planID,
			models.LPSSEEvent{
				EventType:        models.LPSSESuggestedActions,
				PlanID:           planID,
				ClientTurnID:     turnID,
				MessageID:        aiReply.ID,
				SuggestedActions: suggestedChips,
			},
		)

		lpGenLog.Info(
			"已广播建议芯片",
			"plan_id", planID,
			"stage", currentStage,
			"chip_count",
			len(suggestedChips),
		)
	}

	s.scheduleLessonPlanContextCapsuleUpdate(lp, userMsg, aiReply, turnID)

	lpGenLog.Info(
		"AI对话回复完成",
		"plan_id", planID,
		"stage", currentStage,
		"tokens", result.TokensUsed,
		"latency_ms", result.LatencyMs,
		"chunks", chunkCount,
		"has_content", hasContent,
		"artifact_saved", artifactSaved,
		"working_msgs",
		len(workingMessages),
		"assistant_injected",
		assistantPrompt != "",
		"full_generate", fullGenerate,
		"evidence_harness",
		evidenceHarnessRun != nil,
		"evidence_repaired",
		evidenceHarnessRun != nil &&
			evidenceHarnessRun.Repaired,
	)

	// 事后Harness样本采集继续保留。
	// 正式产物写入的是通过多证据Harness后的最终版本。
	s.captureHarnessSampleBestEffort(
		planID,
		currentStage,
		lp.AuthorID,
		lpSchoolID,
		req.AssistantID,
		assistantLabel,
		result.ModelUsed,
		stageSystemPrompt,
		rawContent,
	)

	_ = fullGenInjected
}
