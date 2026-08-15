package services

// workshop_stage_transition.go — 阶段自然衔接与自动触发语
//
// 阶段状态仍然在后台正常推进，但教师看到的是一段连续备课对话。
//
// 硬性体验规则：
//   - 不重新自我介绍；
//   - 不宣告“现在进入某阶段”；
//   - 不复述完整流程路线图；
//   - 不重新确认前面已经确认的目标、主线或活动方向；
//   - 第一自然句直接承接前序有效结论；
//   - 阶段变化只改变当前注意力，不切断全课共同记忆。
//
// 自动触发消息保留“我们进入……阶段了。”前缀，仅作为后端和前端识别的
// 隐藏系统标记。AI必须忽略该标记的表面措辞，不能将其复述给教师。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var autoTriggerStages = map[string]string{
	"design": "我们进入教学设计阶段了。【系统内部触发，绝不能复述这句话】把这看成上一段备课对话的自然延续。直接承接前面已经确认的课程核心、学情判断、教学目标、重难点和教学方向，推进课堂主线、环节与活动设计。不要重新自我介绍，不要说明阶段名称，不要复述流程，不要再次询问已经确认的问题。第一句应像自然接着老师上一句话继续讨论。",
	"write":  "我们进入教案撰写阶段了。【系统内部触发，绝不能复述这句话】直接沿用前面已经确定的教学主线、环节、活动、评价和约束。不要重新介绍自己，不要宣布进入撰写，不要把刚确定的设计完整复述一遍，也不要要求老师再次确认。自然询问老师希望逐段落笔还是直接形成完整正文；若对话上下文已经明确选择，则直接按该选择继续。",
	"review": "我们进入评审阶段了。【系统内部触发，绝不能复述这句话】先在内部核对是否已有完整教案正文。若没有，只自然说明目前还缺少可评审的完整正文，不得编造评分；若已有，直接开始专业评审。不得重新介绍自己、宣布阶段变化或复述此前流程。",
	"revise": "我们进入修订定稿阶段了。【系统内部触发，绝不能复述这句话】直接承接评审中最关键且仍未解决的改进意见，并结合教师此前已经确认的教学主线修订。不要重新介绍自己，不要宣布进入修订，不要复述整份评审报告，也不要再次确认已明确接受的修改。",
}

// stageContinuationTurnIDPrefix 标识“阶段自动自然承接”专用轮次。
//
// 普通教师主动对话使用页面生成的tN_xxx；后台评审继续保持空turnID。
// 前端只把本前缀视为“阶段承接中”，因此不会误把后台评审锁成聊天busy。
const stageContinuationTurnIDPrefix = "stage_continuation_"

// 对话模式和专家模式统一采用无感衔接，避免两套体验继续漂移。
var autoTriggerStagesLean = autoTriggerStages

// triggerStageContinuation 旁路触发自然的跨阶段连续对话。
func (s *WorkshopStageService) triggerStageContinuation(
	lessonPlanID string,
	stageCode string,
	callerID string,
	useLeanPrompt bool,
	delay time.Duration,
) {
	if s.genService == nil {
		return
	}

	triggerMessage, exists := autoTriggerStages[stageCode]
	if !exists {
		return
	}

	if useLeanPrompt {
		if leanMessage, ok := autoTriggerStagesLean[stageCode]; ok {
			triggerMessage = leanMessage
		}
	}

	wsLog.Info(
		"自动触发自然阶段承接",
		"plan_id", lessonPlanID,
		"stage", stageCode,
		"lean", useLeanPrompt,
	)

	// 先在当前阶段切换请求内完成短延迟和Chat任务登记，再返回HTTP。
	//
	// 旧实现把整段登记放进goroutine：current_stage已经更新、HTTP也可能已经返回，
	// 但lesson_plan_ai互斥任务还没登记，老师可在这几十到几百毫秒窗口再次点击芯片。
	// 这里最多只增加调用方传入的100/200ms自然停顿；Chat真实AI仍由其自身后台任务执行。
	time.Sleep(delay)

	continuationTurnID := fmt.Sprintf(
		"%s%s_%d",
		stageContinuationTurnIDPrefix,
		stageCode,
		time.Now().UnixMilli(),
	)

	request := &models.LessonPlanChatRequest{
		PlanID:       lessonPlanID,
		Message:      triggerMessage,
		ClientTurnID: continuationTurnID,
	}

	if err := s.genService.Chat(
		context.Background(),
		request,
		callerID,
	); err != nil {
		wsLog.Warn(
			"自动触发自然阶段承接失败",
			"plan_id", lessonPlanID,
			"stage", stageCode,
			"error", err,
		)
	}
}

// leavingStageHasSubstantiveContent 判断当前阶段是否有教师真实发言。
func (s *WorkshopStageService) leavingStageHasSubstantiveContent(
	ctx context.Context,
	lessonPlanID string,
) bool {
	messages, err := repository.GetCurrentStageMessages(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return true
	}

	for _, message := range messages {
		if message == nil {
			continue
		}
		if message.Role == models.ConvRoleUser &&
			!isStageAutoTriggerContent(message.Content) {
			return true
		}
	}

	return false
}

// isStageAutoTriggerContent 识别后台自动触发消息。
func isStageAutoTriggerContent(content string) bool {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "我们进入") &&
		strings.Contains(content, "阶段了。") {
		return true
	}

	if strings.HasPrefix(content, "请先检查上一阶段") {
		return true
	}

	return false
}
