package services

// lesson_plan_full_artifact_intent.go — 对话中“直接生成完整教案”的确定性意图与正式出稿路由
//
// 目标：
//   - 老师自己输入“直接给我完整教案”等明确整稿动作时，与系统full_generate入口归一到同一条正式出稿链；
//   - analyze/design中的整稿动作自动进入write，review中的整稿动作进入revise，不让阶段骨架与整稿任务互相冲突；
//   - “为什么没通过、哪里不一致”等诊断问题保持普通讨论，不因为出现“完整教案”几个字再次触发整稿；
//   - 阶段切换、课本优先级、Harness和正文提交仍复用现有正式链，不建立第二套生成系统。

import (
	"context"
	"strings"
	"time"

	"tedna/internal/models"
)

// isLessonPlanArtifactDiagnosticIntent 判断老师是在询问失败原因、冲突位置或处理建议，
// 而不是要求系统立即再次生成正式产物。
func isLessonPlanArtifactDiagnosticIntent(
	text string,
) bool {
	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"为什么", "为何", "怎么回事", "什么原因", "原因是什么",
			"没通过", "未通过", "没有通过", "失败", "报错", "错误",
			"一致性校验", "一致性检查", "校验结果", "哪里不一致",
			"有什么问题", "哪里有问题", "为什么被拦", "为什么阻止",
			"什么意思", "怎么理解", "如何处理", "怎么处理",
			"该怎么改", "应该怎么改", "建议怎么改",
		},
	)
}

func isLessonPlanWholeDraftObjectIntent(
	text string,
) bool {
	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"完整教案", "整份教案", "整套教案", "正式教案",
			"教案全文", "完整正文", "整份正文", "整套正文",
			"完整教学设计稿", "全案", "教案",
		},
	)
}

func isLessonPlanWholeDraftRetryCommandIntent(
	text string,
) bool {
	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"请重新生成完整教案", "重新生成完整教案", "重新生成教案",
			"再生成完整教案", "再生成教案", "重新输出完整教案",
			"再输出完整教案", "重新给我完整教案", "再给我一份教案",
			"再来一版教案", "重写一版教案", "重新写一份教案",
			"再写一版教案", "重新出完整教案", "再出一份教案",
			"重做一版教案", "重试生成完整教案",
		},
	)
}

func isLessonPlanWholeDraftCommandIntent(
	text string,
) bool {
	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"请生成", "帮我生成", "直接生成", "生成教案",
			"请输出", "直接输出", "输出教案", "输出完整",
			"直接给我", "给我完整", "直接给出",
			"请写", "帮我写", "直接写", "写一份", "写出完整", "写教案",
			"直接出", "出一份", "出教案",
			"一键生成", "一键写", "一键出",
		},
	) ||
		isLessonPlanWholeDraftRetryCommandIntent(
			text,
		)
}

// isLessonPlanStrongArtifactCommandIntent 判断一句话中是否包含明确执行动作。
func isLessonPlanStrongArtifactCommandIntent(
	text string,
) bool {
	if isLessonPlanWholeDraftCommandIntent(
		text,
	) {
		return true
	}

	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"请修改", "帮我修改", "直接修改", "重新修改",
			"帮我改", "直接改", "再改一版",
			"按评审意见修改", "按这个修改", "全部修改",
			"完整修改", "更新正文", "修订定稿", "最终定稿",
		},
	)
}

// isLessonPlanWholeDraftMutationIntent 识别“现在生成整份教案”的明确动作。
func isLessonPlanWholeDraftMutationIntent(
	text string,
) bool {
	text =
		normalizeLessonPlanTurnText(
			text,
		)

	if text == "" ||
		!isLessonPlanWholeDraftObjectIntent(
			text,
		) {
		return false
	}

	if isLessonPlanArtifactDiagnosticIntent(
		text,
	) {
		return isLessonPlanWholeDraftRetryCommandIntent(
			text,
		)
	}

	return isLessonPlanWholeDraftCommandIntent(
		text,
	)
}

// isLessonPlanFormalArtifactTextIntent 识别除局部正文修改外的正式产物任务。
func isLessonPlanFormalArtifactTextIntent(
	text string,
) bool {
	text =
		normalizeLessonPlanTurnText(
			text,
		)

	if text == "" {
		return false
	}

	if isLessonPlanWholeDraftMutationIntent(
		text,
	) {
		return true
	}

	if isLessonPlanArtifactDiagnosticIntent(
		text,
	) &&
		!isLessonPlanStrongArtifactCommandIntent(
			text,
		) {
		return false
	}

	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"一键生成", "完整分析", "完整设计",
			"全面评审", "正式评审", "完整评审",
			"修订定稿", "最终定稿", "发布前检查", "发布前校验",
			"完整方案", "完整替换", "改一版教案", "更新完整教案",
			"按评审意见修改", "按这个修改", "全部修改",
		},
	)
}

// isLessonPlanFormalContentMutationIntent 判断write/revise是否明确要求修改正式正文。
func isLessonPlanFormalContentMutationIntent(
	stageCode string,
	text string,
) bool {
	stageCode =
		strings.ToLower(
			strings.TrimSpace(
				stageCode,
			),
		)

	if stageCode != "write" &&
		stageCode != "revise" {
		return false
	}

	if isLessonPlanArtifactDiagnosticIntent(
		text,
	) &&
		!isLessonPlanStrongArtifactCommandIntent(
			text,
		) {
		return false
	}

	hasMutationVerb :=
		containsAnyLessonPlanTurnText(
			text,
			[]string{
				"修改", "只改", "直接改", "帮我改",
				"调整", "补充", "删除", "删掉",
				"替换", "改成", "改为", "加上",
				"加入", "写入", "写进", "完善",
				"优化", "修订", "同步", "保存",
			},
		)

	if !hasMutationVerb {
		return false
	}

	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"教案", "正文", "原文", "原有段落",
			"段落文字", "教学目标", "教学重点",
			"教学难点", "教学活动", "教学过程",
			"导入", "作业", "板书", "反思",
			"前面的问题", "上面的内容", "这个版本",
			"右侧", "画布", "正式正文", "正式教案",
		},
	)
}

// appendLessonPlanDiscussionOnlyPrompt 为write/revise普通讨论追加生成边界。
func appendLessonPlanDiscussionOnlyPrompt(
	stageSystemPrompt string,
	stageCode string,
	turnPlan *lessonPlanTurnContextPlan,
) string {
	if turnPlan == nil ||
		!turnPlan.DiscussionOnly ||
		(stageCode != "write" &&
			stageCode != "revise") {
		return stageSystemPrompt
	}

	return stageSystemPrompt + `

== 本轮普通讨论边界（系统级指令）==
本轮不是正式整稿生成或正式正文修改任务，只回答老师当前明确提出的问题。
如果老师正在询问“为什么校验失败、哪里不一致、是否已经生成、某条规则是什么意思”，只解释原因和下一步，不得重新输出整份教案。
如果老师只要求讨论一个局部环节，可以回答该局部内容，但不得自行扩展成完整教案正文。
不得声称已经更新、保存或覆盖右侧教案；只有正式产物任务通过Harness并成功提交后才能这样表述。`
}

// buildLessonPlanChatUserMessage 统一构造落入对话历史的教师消息。
func buildLessonPlanChatUserMessage(
	req *models.LessonPlanChatRequest,
) *models.ConversationMessage {
	content := ""

	if req != nil {
		content = req.Message

		if len(req.SelectedOptions) > 0 {
			content =
				formatSelectedOptions(
					req.SelectedOptions,
					req.Message,
				)
		}

		if len(req.SelectedComponents) > 0 {
			content +=
				formatSelectedComponents(
					req.SelectedComponents,
				)
		}
	}

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleUser,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// prepareLessonPlanWholeDraftChatIntent 在AI任务真正启动前归一“整份教案”动作。
// 系统full_generate的analyze/design阶段任务保持原阶段；只有整份教案动作才自动切到write/revise。
func (s *LessonPlanGenService) prepareLessonPlanWholeDraftChatIntent(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	req *models.LessonPlanChatRequest,
	callerID string,
) (*models.LessonPlan, error) {
	if lessonPlan == nil ||
		req == nil {
		return lessonPlan, nil
	}

	originalFullGenerate :=
		req.FullGenerate

	text :=
		normalizeLessonPlanTurnText(
			req.Message,
		)

	if !isLessonPlanWholeDraftMutationIntent(
		text,
	) {
		return lessonPlan, nil
	}

	currentStage :=
		strings.ToLower(
			strings.TrimSpace(
				lessonPlan.CurrentStage,
			),
		)

	if currentStage == "write" ||
		currentStage == "revise" {
		req.FullGenerate = true
		return lessonPlan, nil
	}

	targetStageCode := "write"

	if currentStage == "review" {
		targetStageCode = "revise"
	}

	targetStage, err :=
		s.stageService.SwitchToStagePrepared(
			ctx,
			lessonPlan.ID,
			targetStageCode,
			callerID,
		)
	if err != nil {
		return nil, err
	}

	switchedPlan := *lessonPlan
	switchedPlan.CurrentStage =
		targetStage.StageCode

	// 自动跨阶段后重新按目标阶段解析助手，不能把analyze/design/review助手带入整稿。
	req.AssistantID = ""
	req.FullGenerate = true

	intentSource :=
		"natural_language"

	if originalFullGenerate {
		intentSource =
			"system_full_generate"
	}

	lpGenLog.Info(
		"教师整稿请求已归一到正式全委托链",
		"plan_id", switchedPlan.ID,
		"from_stage", currentStage,
		"to_stage",
		switchedPlan.CurrentStage,
		"intent_source",
		intentSource,
	)

	return &switchedPlan, nil
}
