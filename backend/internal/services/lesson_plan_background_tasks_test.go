package services

import (
	"errors"
	"testing"
)

func TestLessonPlanAITaskDuplicateAndDone(t *testing.T) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks = NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks = originalTracker
	}()

	task, err := startLessonPlanAITask("plan-001")
	if err != nil || task == nil {
		t.Fatalf("首次教案AI任务登记失败: task=%v err=%v", task, err)
	}

	duplicate, duplicateErr := startLessonPlanAITask("plan-001")
	if duplicate != nil {
		t.Fatalf("重复任务不应返回任务句柄")
	}
	if !errors.Is(duplicateErr, ErrLPGenTaskRunning) {
		t.Fatalf("重复任务错误不正确: %v", duplicateErr)
	}

	summary := GlobalBackgroundTasks.Summary()
	if summary.Active != 1 || summary.Critical != 1 {
		t.Fatalf("教案AI任务统计错误: %+v", summary)
	}

	task.Done()

	if summary := GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf("任务完成后应归零: %+v", summary)
	}
}

func TestLessonPlanAITaskRejectedDuringDraining(t *testing.T) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks = NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks = originalTracker
	}()

	GlobalBackgroundTasks.BeginDraining()

	task, err := startLessonPlanAITask("plan-002")
	if task != nil {
		t.Fatalf("排空期间不应返回任务句柄")
	}
	if !errors.Is(err, ErrLPGenServiceDraining) {
		t.Fatalf("排空期间错误不正确: %v", err)
	}

	if summary := GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf("排空期间不应登记任务: %+v", summary)
	}
}

func TestLessonPlanAutoIndexRejectedDuringDraining(t *testing.T) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks = NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks = originalTracker
	}()

	GlobalBackgroundTasks.BeginDraining()

	service := &LessonPlanGenService{}
	result := service.triggerAutoLessonIndexTracked(
		"plan-003",
		nil,
	)

	if result != BackgroundRejectedDraining {
		t.Fatalf("排空期间自动索引应被拒绝，实际=%s", result)
	}

	if summary := GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf("排空期间不应登记自动索引: %+v", summary)
	}
}

func TestLessonPlanAITaskInvalidPlanID(t *testing.T) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks = NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks = originalTracker
	}()

	task, err := startLessonPlanAITask("")
	if task != nil {
		t.Fatalf("空教案ID不应返回任务句柄")
	}
	if err == nil {
		t.Fatalf("空教案ID应返回错误")
	}
}
