package services

// assistant_runtime_chat_context_test.go
//
// 验证运行聊天失败结算上下文与浏览器请求取消解耦。
//
// 浏览器断开或HTTP请求超时后：
//   - 父Context可以已经Canceled或DeadlineExceeded；
//   - 结算Context仍应保持可用；
//   - 结算Context拥有独立的20秒上限；
//   - 调用自身cancel后必须立即结束。
//
// 本测试不连接数据库、不调用AI。

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestAssistantRuntimeDetachedContextSurvivesParentCancellation 验证取消解耦。
func TestAssistantRuntimeDetachedContextSurvivesParentCancellation(
	t *testing.T,
) {
	parent,
		cancelParent :=
		context.WithCancel(
			context.Background(),
		)

	cancelParent()

	if !errors.Is(
		parent.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"父Context没有进入Canceled状态: %v",
			parent.Err(),
		)
	}

	assertAssistantRuntimeDetachedSettlementContext(
		t,
		parent,
	)
}

// TestAssistantRuntimeDetachedContextSurvivesParentDeadline 验证超时解耦。
func TestAssistantRuntimeDetachedContextSurvivesParentDeadline(
	t *testing.T,
) {
	parent,
		cancelParent :=
		context.WithDeadline(
			context.Background(),
			time.Now().Add(
				-time.Second,
			),
		)
	defer cancelParent()

	if !errors.Is(
		parent.Err(),
		context.DeadlineExceeded,
	) {
		t.Fatalf(
			"父Context没有进入DeadlineExceeded状态: %v",
			parent.Err(),
		)
	}

	assertAssistantRuntimeDetachedSettlementContext(
		t,
		parent,
	)
}

// assertAssistantRuntimeDetachedSettlementContext 校验独立结算上下文。
func assertAssistantRuntimeDetachedSettlementContext(
	t *testing.T,
	parent context.Context,
) {
	t.Helper()

	detached,
		cancel :=
		assistantRuntimeDetachedContext(
			parent,
		)

	if detached == nil {
		cancel()

		t.Fatal(
			"独立结算Context为nil",
		)
	}

	if detached.Err() != nil {
		cancel()

		t.Fatalf(
			"独立结算Context错误继承了父请求状态: %v",
			detached.Err(),
		)
	}

	deadline,
		ok :=
		detached.Deadline()

	if !ok {
		cancel()

		t.Fatal(
			"独立结算Context缺少超时上限",
		)
	}

	remaining :=
		time.Until(
			deadline,
		)

	if remaining <= 19*time.Second ||
		remaining >
			assistantRuntimeSettlementTimeout+
				time.Second {
		cancel()

		t.Fatalf(
			"独立结算Context超时范围错误: remaining=%s configured=%s",
			remaining,
			assistantRuntimeSettlementTimeout,
		)
	}

	cancel()

	if !errors.Is(
		detached.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"独立结算Context自身cancel没有生效: %v",
			detached.Err(),
		)
	}
}
