package services

import (
	"context"
	"errors"
	"testing"
)

func TestLessonPlanAIStageMutationOwnership(t *testing.T) {
	previousTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks = NewBackgroundTaskTracker()

	t.Cleanup(func() {
		GlobalBackgroundTasks = previousTracker
	})

	ownerTask, result := GlobalBackgroundTasks.TryStartExternal(
		lessonPlanAITaskType,
		"plan-owner",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted || ownerTask == nil {
		t.Fatalf(
			"登记owner任务失败: result=%s task_nil=%v",
			result,
			ownerTask == nil,
		)
	}
	t.Cleanup(ownerTask.Done)

	if err := ensureLessonPlanAIIdleForStageMutation(
		context.Background(),
		"plan-owner",
	); !errors.Is(err, ErrLPGenTaskRunning) {
		t.Fatalf(
			"外部阶段操作应被运行中的AI任务拦截，实际error=%v",
			err,
		)
	}

	ownerContext := withLessonPlanAIStageMutationOwner(
		context.Background(),
		ownerTask,
		"plan-owner",
	)

	if err := ensureLessonPlanAIIdleForStageMutation(
		ownerContext,
		"plan-owner",
	); err != nil {
		t.Fatalf(
			"当前Chat任务所有者应允许内部阶段准备，实际error=%v",
			err,
		)
	}

	mismatchedContext := withLessonPlanAIStageMutationOwner(
		context.Background(),
		ownerTask,
		"plan-other",
	)
	if err := ensureLessonPlanAIIdleForStageMutation(
		mismatchedContext,
		"plan-owner",
	); !errors.Is(err, ErrLPGenTaskRunning) {
		t.Fatalf(
			"错误planID不得借用owner任务能力，实际error=%v",
			err,
		)
	}

	otherTask, otherResult := GlobalBackgroundTasks.TryStartExternal(
		lessonPlanAITaskType,
		"plan-other",
		BackgroundTaskCritical,
		nil,
	)
	if otherResult != BackgroundStarted || otherTask == nil {
		t.Fatalf(
			"登记other任务失败: result=%s task_nil=%v",
			otherResult,
			otherTask == nil,
		)
	}
	t.Cleanup(otherTask.Done)

	if err := ensureLessonPlanAIIdleForStageMutation(
		ownerContext,
		"plan-other",
	); !errors.Is(err, ErrLPGenTaskRunning) {
		t.Fatalf(
			"一个教案的owner能力不得放行另一教案，实际error=%v",
			err,
		)
	}

	ownerTask.Done()

	if err := ensureLessonPlanAIIdleForStageMutation(
		context.Background(),
		"plan-owner",
	); err != nil {
		t.Fatalf(
			"AI任务结束后外部阶段操作应恢复可用，实际error=%v",
			err,
		)
	}
}
