package services

// lesson_plan_background_tasks.go — 教案AI任务与派生索引的统一后台治理
//
// 管理范围：
//   1. 对话式备课Chat；
//   2. 手动触发AI评审；
//   3. 应用评审建议并重新评审；
//   4. review阶段完成后的自动AOCI索引；
//   5. 对话完成后的Harness样本采集。
//
// 并发策略：
//   - Chat、评审和应用建议都可能读取或覆盖同一教案正文、阶段产出和评审结果；
//   - 三者统一使用lesson_plan_ai:<planID>任务键；
//   - 同一教案同一时间只能执行一条AI主链；
//   - 不同教案仍可并发运行。
//
// 排空策略：
//   - AI主链属于critical，部署时等待其自然完成；
//   - 自动教案索引属于best_effort，draining后不再启动；
//   - Harness采集在message_done之后转为独立best_effort子任务；
//   - lesson_plan_ai互斥锁不再被样本采集尾巴占用，老师收到回复后可立即推进阶段；
//   - Harness子任务仍由Tracker登记，部署draining期间允许已有主任务派生并等待其自然收尾。
//
// panic策略：
//   - 所有AI主链通过BackgroundTask.Run执行；
//   - panic被转换为error，不再导致整个Go生产进程退出；
//   - 主链异常时补发SSE error，使前端退出“生成中”状态。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tedna/internal/repository"
)

// ==================== 对外错误 ====================

var (
	// ErrLPGenServiceDraining 表示服务正在部署排空，新任务没有启动。
	ErrLPGenServiceDraining = errors.New("系统正在升级，本次AI任务尚未开始，请稍后重试")

	// ErrLPGenTaskRunning 表示同一教案已有一条AI主链正在运行。
	ErrLPGenTaskRunning = errors.New("该教案已有AI任务正在执行，请等待当前任务完成")
)

const (
	// lessonPlanAITaskType 是Chat、评审和应用建议共享的互斥任务类型。
	lessonPlanAITaskType = "lesson_plan_ai"

	// lessonPlanIndexTaskType 是自动教案AOCI索引任务类型。
	lessonPlanIndexTaskType = "lesson_plan_index"

	// lessonPlanHarnessTaskType 是主回复完成后的Harness样本采集任务类型。
	lessonPlanHarnessTaskType = "lesson_plan_harness_sample"
)

// ==================== AI主任务登记与执行 ====================

// startLessonPlanAITask 登记同一教案唯一AI主任务。
func startLessonPlanAITask(
	planID string,
) (*BackgroundTask, error) {
	task, result := GlobalBackgroundTasks.TryStartExternal(
		lessonPlanAITaskType,
		planID,
		BackgroundTaskCritical,
		nil,
	)

	switch result {
	case BackgroundStarted:
		return task, nil

	case BackgroundRejectedDraining:
		return nil, ErrLPGenServiceDraining

	case BackgroundAlreadyRunning:
		return nil, ErrLPGenTaskRunning

	case BackgroundInvalid:
		return nil, fmt.Errorf("教案后台任务参数无效")

	default:
		return nil, fmt.Errorf("教案后台任务登记失败: %s", result)
	}
}

// lessonPlanAIStageMutationOwnerContextKey 是Chat内部阶段准备能力的私有context键。
//
// 值不是planID，而是已经登记成功的BackgroundTask唯一key；
// 外部HTTP请求无法构造这个私有Go值，普通阶段操作仍严格受AI busy硬闸保护。
type lessonPlanAIStageMutationOwnerContextKey struct{}

// withLessonPlanAIStageMutationOwner 仅在调用方真实持有当前教案lesson_plan_ai任务句柄时
// 授予本条同步调用链“内部阶段准备”能力。
//
// 典型场景：Chat已经取得唯一AI主任务锁，随后整稿意图需要在AI真正启动前
// 从review切到revise。若没有这项所有权能力，阶段状态机会把当前Chat自己的任务
// 误判成“另一条AI仍在运行”，形成自锁。
func withLessonPlanAIStageMutationOwner(
	ctx context.Context,
	task *BackgroundTask,
	planID string,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	expectedKey := buildBackgroundTaskKey(
		lessonPlanAITaskType,
		planID,
	)
	if expectedKey == "" ||
		task == nil ||
		task.key != expectedKey {
		return ctx
	}

	return context.WithValue(
		ctx,
		lessonPlanAIStageMutationOwnerContextKey{},
		expectedKey,
	)
}

// lessonPlanAIStageMutationOwnedByContext 判断当前调用链是否真实持有同一教案AI主任务。
func lessonPlanAIStageMutationOwnedByContext(
	ctx context.Context,
	planID string,
) bool {
	if ctx == nil {
		return false
	}

	expectedKey := buildBackgroundTaskKey(
		lessonPlanAITaskType,
		planID,
	)
	if expectedKey == "" {
		return false
	}

	ownerKey, ok := ctx.Value(
		lessonPlanAIStageMutationOwnerContextKey{},
	).(string)

	return ok &&
		ownerKey == expectedKey
}

// ensureLessonPlanAIIdleForStageMutation 确保阶段状态变更不会与另一条同教案AI主任务并发。
//
// 外部阶段操作：只要lesson_plan_ai正在运行就拒绝。
// 当前Chat内部阶段准备：若context携带“当前已登记任务句柄”产生的精确所有权能力，
// 允许在真正启动AI前完成确定性的write/revise阶段准备。
//
// 这样同时保持两条安全边界：
//   - 老师按钮/另一条Chat不能在AI运行中提前改变current_stage；
//   - 当前Chat不会因为自己已经先登记AI任务而把整稿意图的内部切阶段锁死。
func ensureLessonPlanAIIdleForStageMutation(
	ctx context.Context,
	planID string,
) error {
	if !GlobalBackgroundTasks.IsRunning(
		lessonPlanAITaskType,
		planID,
	) {
		return nil
	}

	if lessonPlanAIStageMutationOwnedByContext(
		ctx,
		planID,
	) {
		lpGenLog.Info(
			"当前Chat任务获准执行内部阶段准备",
			"plan_id", planID,
		)
		return nil
	}

	return ErrLPGenTaskRunning
}

// runLessonPlanAITask 启动已经登记的教案AI主任务。
//
// work内部业务错误仍沿用原有SSE提示逻辑；
// 本包装只处理未预期panic等异常。
func (s *LessonPlanGenService) runLessonPlanAITask(
	task *BackgroundTask,
	planID string,
	turnID string,
	operation string,
	work func(),
) {
	go func() {
		err := task.Run(func() error {
			if work == nil {
				return fmt.Errorf("教案后台任务执行函数为空")
			}

			work()
			return nil
		})
		if err == nil {
			return
		}

		lpGenLog.Error(
			"教案后台AI任务异常",
			"plan_id", planID,
			"operation", operation,
			"error", err,
		)

		s.broadcastError(
			planID,
			turnID,
			"后台AI任务发生异常，本轮内容未能完整生成，请稍后重试",
		)
	}()
}

// ==================== 自动教案索引 ====================

// triggerAutoLessonIndexTracked 启动受Tracker管理的自动教案索引。
//
// 自动索引可重新计算，属于best_effort：
//   - draining后不再启动；
//   - 同一教案已有索引任务时不重复启动；
//   - 索引失败不影响已完成的教案和评审结果。
func (s *LessonPlanGenService) triggerAutoLessonIndexTracked(
	planID string,
	aiScore *float64,
) BackgroundStartResult {
	task, result := GlobalBackgroundTasks.TryStartExternal(
		lessonPlanIndexTaskType,
		planID,
		BackgroundTaskBestEffort,
		nil,
	)
	if result != BackgroundStarted {
		lpGenLog.Info(
			"自动教案索引任务未启动",
			"plan_id", planID,
			"result", string(result),
		)
		return result
	}

	// 复制分数，避免异步任务持有调用栈中临时字段的原始指针。
	var scoreCopy *float64
	if aiScore != nil {
		value := *aiScore
		scoreCopy = &value
	}

	go func() {
		err := task.Run(func() error {
			s.triggerAutoLessonIndex(
				context.Background(),
				planID,
				scoreCopy,
			)
			return nil
		})
		if err == nil {
			return
		}

		lpGenLog.Warn(
			"自动教案索引后台任务异常",
			"plan_id", planID,
			"error", err,
		)
	}()

	return BackgroundStarted
}

// ==================== Harness采集 ====================

// captureHarnessSampleBestEffort 在主AI回复完成后登记独立Harness样本采集任务。
//
// 关键并发边界：
//   - message_done已经发送给老师后，lesson_plan_ai主任务应尽快释放互斥锁；
//   - Harness样本是best_effort，不得继续占用lesson_plan_ai，否则老师立即点击阶段推进会被误判为并发AI；
//   - 样本采集仍通过TryStartChild登记，部署draining期间可以完成必要收尾。
func (s *LessonPlanGenService) captureHarnessSampleBestEffort(
	planID string,
	stageCode string,
	authorID string,
	schoolID string,
	assistantID string,
	assistantLabel string,
	modelUsed string,
	systemPrompt string,
	aiRaw string,
) {
	resourceID := fmt.Sprintf(
		"%s:%s:%d",
		planID,
		stageCode,
		time.Now().UnixNano(),
	)

	task, result := GlobalBackgroundTasks.TryStartChild(
		lessonPlanHarnessTaskType,
		resourceID,
		BackgroundTaskBestEffort,
		nil,
	)
	if result != BackgroundStarted {
		lpGenLog.Warn(
			"harness采集任务未登记，不影响主流程",
			"plan_id", planID,
			"stage", stageCode,
			"result", string(result),
		)
		return
	}

	go func() {
		err := task.Run(func() error {
			s.captureHarnessSample(
				planID,
				stageCode,
				authorID,
				schoolID,
				assistantID,
				assistantLabel,
				modelUsed,
				systemPrompt,
				aiRaw,
			)
			return nil
		})
		if err != nil {
			lpGenLog.Warn(
				"harness采集后台任务异常，不影响主流程",
				"plan_id", planID,
				"stage", stageCode,
				"error", err,
			)
		}
	}()
}

// captureHarnessSample 执行单次Harness评测样本采集。
//
// 最长5秒后由context结束；数据库、学校授权或panic异常均只影响本次样本，
// 不影响已经完成的教师主回复和阶段状态。
func (s *LessonPlanGenService) captureHarnessSample(
	planID string,
	stageCode string,
	authorID string,
	schoolID string,
	assistantID string,
	assistantLabel string,
	modelUsed string,
	systemPrompt string,
	aiRaw string,
) {
	bgCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	isDowngraded := false

	enabled, err := repository.IsSchoolOverseasEnabled(
		bgCtx,
		schoolID,
	)
	if err != nil {
		lpGenLog.Warn(
			"harness采集查询学校境外授权失败，按降级保守处理",
			"plan_id", planID,
			"school_id", schoolID,
			"error", err,
		)
		isDowngraded = true
	} else {
		isDowngraded = !enabled
	}

	input := repository.HarnessEvalSampleInput{
		LessonPlanID:  planID,
		StageCode:     stageCode,
		UserID:        authorID,
		SchoolID:      schoolID,
		ModelUsed:     modelUsed,
		IsDowngraded:  isDowngraded,
		AssistantID:   assistantID,
		AssistantName: assistantLabel,
		SystemPrompt:  systemPrompt,
		AIOutput:      aiRaw,
	}

	if err := repository.InsertHarnessEvalSample(
		bgCtx,
		input,
	); err != nil {
		lpGenLog.Warn(
			"harness采集写入样本失败，不影响主流程",
			"plan_id", planID,
			"stage", stageCode,
			"error", err,
		)
		return
	}

	lpGenLog.Info(
		"harness采集样本已落库",
		"plan_id", planID,
		"stage", stageCode,
		"model", modelUsed,
		"is_downgraded", isDowngraded,
		"assistant", assistantLabel,
		"system_prompt_len", len(systemPrompt),
		"ai_output_len", len(aiRaw),
	)
}
