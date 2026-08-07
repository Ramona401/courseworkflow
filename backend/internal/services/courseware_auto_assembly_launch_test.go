package services

import (
	"errors"
	"testing"
	"time"
)

// uniqueCoursewareAssemblyLaunchTestID 避免不同测试共享进程级票据。
func uniqueCoursewareAssemblyLaunchTestID(
	t *testing.T,
) string {
	t.Helper()

	return "test-" +
		t.Name() +
		"-" +
		time.Now().UTC().Format(
			"20060102T150405.000000000",
		)
}

func TestCoursewareAutoAssemblyLaunchIdleCancelDoesNotAffectNextStart(
	t *testing.T,
) {
	coursewareID :=
		uniqueCoursewareAssemblyLaunchTestID(
			t,
		)

	lock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lock.Lock()
	marked :=
		markCoursewareAutoAssemblyPendingCancelLocked(
			coursewareID,
		)
	lock.Unlock()

	if marked {
		t.Fatal(
			"空闲课件不应创建启动前取消票据",
		)
	}

	token, err :=
		PrepareCoursewareAutoAssemblyLaunch(
			coursewareID,
			true,
		)
	if err != nil {
		t.Fatalf(
			"创建启动票据失败: %v",
			err,
		)
	}
	defer AbortCoursewareAutoAssemblyLaunch(
		coursewareID,
		token,
	)

	state :=
		GetCoursewareAutoAssemblyLaunchState(
			coursewareID,
		)
	if !state.Pending {
		t.Fatal(
			"新启动票据应处于pending状态",
		)
	}
	if !state.SkipVideo {
		t.Fatal(
			"启动状态必须保留skip_video交付模式",
		)
	}

	lock.Lock()
	cancelled, consumeErr :=
		consumeCoursewareAutoAssemblyLaunchLocked(
			coursewareID,
			token,
		)
	lock.Unlock()

	if consumeErr != nil {
		t.Fatalf(
			"消费启动票据失败: %v",
			consumeErr,
		)
	}
	if cancelled {
		t.Fatal(
			"空闲时的无效取消不得污染后续新启动",
		)
	}
}

func TestCoursewareAutoAssemblyLaunchCancellationFollowsExactTicket(
	t *testing.T,
) {
	coursewareID :=
		uniqueCoursewareAssemblyLaunchTestID(
			t,
		)

	token, err :=
		PrepareCoursewareAutoAssemblyLaunch(
			coursewareID,
			false,
		)
	if err != nil {
		t.Fatalf(
			"创建启动票据失败: %v",
			err,
		)
	}
	defer AbortCoursewareAutoAssemblyLaunch(
		coursewareID,
		token,
	)

	lock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lock.Lock()

	if !markCoursewareAutoAssemblyPendingCancelLocked(
		coursewareID,
	) {
		lock.Unlock()
		t.Fatal(
			"真实pending启动必须能够绑定取消",
		)
	}

	cancelled, consumeErr :=
		consumeCoursewareAutoAssemblyLaunchLocked(
			coursewareID,
			token,
		)
	lock.Unlock()

	if consumeErr != nil {
		t.Fatalf(
			"消费启动票据失败: %v",
			consumeErr,
		)
	}
	if !cancelled {
		t.Fatal(
			"与当前token绑定的取消必须阻止本次启动",
		)
	}
}

func TestCoursewareAutoAssemblyLaunchAbortIsTokenScoped(
	t *testing.T,
) {
	coursewareID :=
		uniqueCoursewareAssemblyLaunchTestID(
			t,
		)

	token, err :=
		PrepareCoursewareAutoAssemblyLaunch(
			coursewareID,
			false,
		)
	if err != nil {
		t.Fatalf(
			"创建启动票据失败: %v",
			err,
		)
	}

	AbortCoursewareAutoAssemblyLaunch(
		coursewareID,
		"not-the-current-token",
	)

	_, secondErr :=
		PrepareCoursewareAutoAssemblyLaunch(
			coursewareID,
			true,
		)
	if !errors.Is(
		secondErr,
		ErrCoursewareAutoAssemblyLaunchPending,
	) {
		t.Fatalf(
			"错误token不应清理当前票据，实际错误: %v",
			secondErr,
		)
	}

	AbortCoursewareAutoAssemblyLaunch(
		coursewareID,
		token,
	)

	nextToken, nextErr :=
		PrepareCoursewareAutoAssemblyLaunch(
			coursewareID,
			true,
		)
	if nextErr != nil {
		t.Fatalf(
			"正确token清理后应允许新启动: %v",
			nextErr,
		)
	}
	AbortCoursewareAutoAssemblyLaunch(
		coursewareID,
		nextToken,
	)
}
