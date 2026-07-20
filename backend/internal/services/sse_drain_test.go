package services

import (
	"testing"
	"time"

	"tedna/internal/models"
)

// resetSSEDrainTestGlobals 重置测试使用的全局Hub和单向draining状态。
//
// 仅测试调用；生产进程不会从draining恢复running。
func resetSSEDrainTestGlobals() {
	globalSSEDraining.Store(false)
	GlobalLPSSEHub = NewLPSSEHub()
	GlobalCWSSEHub = NewCWSSEHub()
	GlobalSSEHub = NewSSEHub()
	GlobalKBSSEHub = NewKBSSEHub()
}

func TestBeginGlobalSSEDrainingClosesAllConnections(t *testing.T) {
	resetSSEDrainTestGlobals()
	defer resetSSEDrainTestGlobals()

	lpCh := GlobalLPSSEHub.Subscribe("plan-001")
	cwCh := GlobalCWSSEHub.Subscribe("cw-001")
	pipelineCh := GlobalSSEHub.Subscribe("pipeline-001")
	kbCh := GlobalKBSSEHub.Subscribe("job-001")

	before := GetGlobalSSEDrainSummary()
	if before.Total != 4 {
		t.Fatalf("排空前连接数错误: %+v", before)
	}

	summary := BeginGlobalSSEDraining()

	if !summary.Draining {
		t.Fatalf("排空后draining应为true")
	}
	if !summary.FirstTransition {
		t.Fatalf("第一次进入排空应标记FirstTransition")
	}
	if summary.Total != 4 || summary.ConnectionsClosed != 4 {
		t.Fatalf("关闭数量错误: %+v", summary)
	}

	assertLPChannelClosed(t, lpCh)
	assertCWChannelClosed(t, cwCh)
	assertPipelineChannelClosed(t, pipelineCh)
	assertKBChannelClosed(t, kbCh)

	after := GetGlobalSSEDrainSummary()
	if after.Total != 0 {
		t.Fatalf("排空后连接应归零: %+v", after)
	}
}

func TestSSESubscribeRejectedDuringDraining(t *testing.T) {
	resetSSEDrainTestGlobals()
	defer resetSSEDrainTestGlobals()

	globalSSEDraining.Store(true)

	lpHub := NewLPSSEHub()
	cwHub := NewCWSSEHub()
	pipelineHub := NewSSEHub()
	kbHub := NewKBSSEHub()

	assertLPChannelClosed(t, lpHub.Subscribe("plan-002"))
	assertCWChannelClosed(t, cwHub.Subscribe("cw-002"))
	assertPipelineChannelClosed(t, pipelineHub.Subscribe("pipeline-002"))
	assertKBChannelClosed(t, kbHub.Subscribe("job-002"))

	if lpHub.GetTotalSubscribers() != 0 {
		t.Fatalf("draining期间不应登记教案SSE")
	}
	if cwHub.GetTotalSubscribers() != 0 {
		t.Fatalf("draining期间不应登记课件SSE")
	}
	if pipelineHub.GetTotalSubscribers() != 0 {
		t.Fatalf("draining期间不应登记Pipeline SSE")
	}
	if kbHub.GetTotalSubscribers() != 0 {
		t.Fatalf("draining期间不应登记知识库SSE")
	}
}

func TestSSECloseAllIsIdempotent(t *testing.T) {
	resetSSEDrainTestGlobals()
	defer resetSSEDrainTestGlobals()

	GlobalLPSSEHub.Subscribe("plan-003")
	GlobalCWSSEHub.Subscribe("cw-003")
	GlobalSSEHub.Subscribe("pipeline-003")
	GlobalKBSSEHub.Subscribe("job-003")

	first := BeginGlobalSSEDraining()
	second := BeginGlobalSSEDraining()

	if first.Total != 4 {
		t.Fatalf("第一次应关闭4条连接: %+v", first)
	}
	if second.Total != 0 {
		t.Fatalf("第二次排空不应重复关闭连接: %+v", second)
	}
	if second.FirstTransition {
		t.Fatalf("第二次排空不应标记FirstTransition")
	}
}

func TestSSEBroadcastBeforeDrainStillWorks(t *testing.T) {
	resetSSEDrainTestGlobals()
	defer resetSSEDrainTestGlobals()

	lpCh := GlobalLPSSEHub.Subscribe("plan-004")
	GlobalLPSSEHub.Broadcast("plan-004", models.LPSSEEvent{
		EventType: models.LPSSEConnected,
		PlanID:    "plan-004",
	})

	select {
	case event, open := <-lpCh:
		if !open {
			t.Fatalf("正常运行状态下channel不应关闭")
		}
		if event.PlanID != "plan-004" {
			t.Fatalf("收到错误事件: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatalf("未收到教案SSE广播")
	}
}

func assertLPChannelClosed(
	t *testing.T,
	ch chan models.LPSSEEvent,
) {
	t.Helper()

	select {
	case _, open := <-ch:
		if open {
			t.Fatalf("教案SSE channel应已关闭")
		}
	case <-time.After(time.Second):
		t.Fatalf("等待教案SSE channel关闭超时")
	}
}

func assertCWChannelClosed(
	t *testing.T,
	ch chan CWSSEEvent,
) {
	t.Helper()

	select {
	case _, open := <-ch:
		if open {
			t.Fatalf("课件SSE channel应已关闭")
		}
	case <-time.After(time.Second):
		t.Fatalf("等待课件SSE channel关闭超时")
	}
}

func assertPipelineChannelClosed(
	t *testing.T,
	ch chan SSEEvent,
) {
	t.Helper()

	select {
	case _, open := <-ch:
		if open {
			t.Fatalf("Pipeline SSE channel应已关闭")
		}
	case <-time.After(time.Second):
		t.Fatalf("等待Pipeline SSE channel关闭超时")
	}
}

func assertKBChannelClosed(
	t *testing.T,
	ch chan KBSSEEvent,
) {
	t.Helper()

	select {
	case _, open := <-ch:
		if open {
			t.Fatalf("知识库SSE channel应已关闭")
		}
	case <-time.After(time.Second):
		t.Fatalf("等待知识库SSE channel关闭超时")
	}
}

// TestGlobalSSEHandshakeLifecycle 验证：
//   - running状态可以取得握手；
//   - finish允许重复调用；
//   - 进入全局排空后拒绝新的握手。
func TestGlobalSSEHandshakeLifecycle(t *testing.T) {
	resetSSEDrainTestGlobals()
	defer resetSSEDrainTestGlobals()

	finish, accepted := BeginGlobalSSEHandshake()
	if !accepted || finish == nil {
		t.Fatalf("running状态应允许SSE握手")
	}

	finish()
	finish()

	summary := BeginGlobalSSEDraining()
	if !summary.Draining {
		t.Fatalf("应成功进入SSE排空状态: %+v", summary)
	}

	rejectedFinish, accepted := BeginGlobalSSEHandshake()
	if accepted {
		if rejectedFinish != nil {
			rejectedFinish()
		}
		t.Fatalf("draining状态不应接受新的SSE握手")
	}

	if rejectedFinish != nil {
		t.Fatalf("被拒绝的SSE握手不应返回finish函数")
	}
}
