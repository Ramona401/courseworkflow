package services

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundTaskTrackerStartDuplicateAndDone(t *testing.T) {
	tracker := NewBackgroundTaskTracker()

	task, result := tracker.TryStartExternal(
		"courseware_index",
		"cw-001",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted || task == nil {
		t.Fatalf("首次登记失败: result=%s task=%v", result, task)
	}

	duplicate, duplicateResult := tracker.TryStartExternal(
		"courseware_index",
		"cw-001",
		BackgroundTaskCritical,
		nil,
	)
	if duplicate != nil {
		t.Fatalf("重复任务不应返回任务句柄")
	}
	if duplicateResult != BackgroundAlreadyRunning {
		t.Fatalf("重复任务结果错误: %s", duplicateResult)
	}

	summary := tracker.Summary()
	if summary.Active != 1 || summary.Critical != 1 {
		t.Fatalf("任务统计错误: %+v", summary)
	}

	task.Done()
	task.Done()

	summary = tracker.Summary()
	if summary.Active != 0 {
		t.Fatalf("Done后任务应归零: %+v", summary)
	}
}

func TestBackgroundTaskTrackerDrainingRejectsExternalAllowsChild(t *testing.T) {
	tracker := NewBackgroundTaskTracker()
	tracker.BeginDraining()

	external, externalResult := tracker.TryStartExternal(
		"courseware_generate",
		"cw-002",
		BackgroundTaskCritical,
		nil,
	)
	if external != nil {
		t.Fatalf("排空期间外部任务不应启动")
	}
	if externalResult != BackgroundRejectedDraining {
		t.Fatalf("排空期间外部任务结果错误: %s", externalResult)
	}

	child, childResult := tracker.TryStartChild(
		"courseware_alignment",
		"cw-002",
		BackgroundTaskBestEffort,
		nil,
	)
	if childResult != BackgroundStarted || child == nil {
		t.Fatalf("排空期间必要子任务应允许登记: %s", childResult)
	}

	child.Done()
}

func TestBackgroundTaskTrackerWait(t *testing.T) {
	tracker := NewBackgroundTaskTracker()

	task, result := tracker.TryStartExternal(
		"kb_compress",
		"job-001",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted {
		t.Fatalf("任务登记失败: %s", result)
	}

	waitDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		waitDone <- tracker.Wait(ctx)
	}()

	select {
	case err := <-waitDone:
		t.Fatalf("任务未完成前Wait不应返回: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	task.Done()

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("任务完成后Wait返回错误: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("任务完成后Wait未及时返回")
	}
}

func TestBackgroundTaskTrackerWaitTimeout(t *testing.T) {
	tracker := NewBackgroundTaskTracker()

	task, result := tracker.TryStartExternal(
		"courseware_auto_assembly",
		"cw-003",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted {
		t.Fatalf("任务登记失败: %s", result)
	}
	defer task.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := tracker.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("预期DeadlineExceeded，实际: %v", err)
	}
}

func TestBackgroundTaskTrackerDrainHook(t *testing.T) {
	tracker := NewBackgroundTaskTracker()
	var called atomic.Int32

	task, result := tracker.TryStartExternal(
		"courseware_generate_pages",
		"cw-004",
		BackgroundTaskCritical,
		func() {
			called.Add(1)
		},
	)
	if result != BackgroundStarted {
		t.Fatalf("任务登记失败: %s", result)
	}
	defer task.Done()

	tracker.BeginDraining()
	tracker.BeginDraining()

	if called.Load() != 1 {
		t.Fatalf("排空钩子应只调用一次，实际=%d", called.Load())
	}
}

func TestBackgroundTaskRunRecoversPanicAndCompletes(t *testing.T) {
	tracker := NewBackgroundTaskTracker()

	task, result := tracker.TryStartExternal(
		"courseware_preview",
		"cw-005",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted {
		t.Fatalf("任务登记失败: %s", result)
	}

	err := task.Run(func() error {
		panic("test panic")
	})
	if err == nil || !strings.Contains(err.Error(), "test panic") {
		t.Fatalf("panic应转换为error，实际: %v", err)
	}

	if tracker.Summary().Active != 0 {
		t.Fatalf("panic恢复后任务应自动完成")
	}
}

func TestBackgroundTaskRunReturnsBusinessError(t *testing.T) {
	tracker := NewBackgroundTaskTracker()

	task, result := tracker.TryStartExternal(
		"courseware_3d",
		"cw-006",
		BackgroundTaskCritical,
		nil,
	)
	if result != BackgroundStarted {
		t.Fatalf("任务登记失败: %s", result)
	}

	expected := errors.New("business failure")
	err := task.Run(func() error {
		return expected
	})

	if !errors.Is(err, expected) {
		t.Fatalf("业务错误应原样返回: %v", err)
	}

	if tracker.Summary().Active != 0 {
		t.Fatalf("业务错误返回后任务应自动完成")
	}
}
