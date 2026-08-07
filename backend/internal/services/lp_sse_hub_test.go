package services

// lp_sse_hub_test.go — 教案SSE多订阅与独立注销回归测试
//
// 覆盖本次故障的核心场景：
//   - 同一教案建立第二条连接时不得关闭第一条连接；
//   - 广播事件必须送达同一教案的全部活动订阅者；
//   - 注销其中一条连接不得影响其它连接；
//   - 不同教案之间的事件必须保持隔离；
//   - Subscribe必须维持历史单返回值协议，现有排空测试可以继续编译。

import (
	"testing"
	"time"

	"tedna/internal/models"
)

const lessonPlanSSETestTimeout = 500 * time.Millisecond

func receiveLessonPlanSSETestEvent(
	t *testing.T,
	ch <-chan models.LPSSEEvent,
) models.LPSSEEvent {
	t.Helper()

	select {
	case event, open := <-ch:
		if !open {
			t.Fatal("预期收到SSE事件，但channel已经关闭")
		}
		return event

	case <-time.After(lessonPlanSSETestTimeout):
		t.Fatal("等待SSE事件超时")
		return models.LPSSEEvent{}
	}
}

func assertLessonPlanSSETestChannelClosed(
	t *testing.T,
	ch <-chan models.LPSSEEvent,
) {
	t.Helper()

	select {
	case _, open := <-ch:
		if open {
			t.Fatal("预期channel已经关闭，但仍收到活动数据")
		}

	case <-time.After(lessonPlanSSETestTimeout):
		t.Fatal("等待SSE channel关闭超时")
	}
}

// TestLPSSEHubAllowsMultipleSubscribersForSamePlan
// 验证同一教案的第二条连接不会关闭第一条连接。
func TestLPSSEHubAllowsMultipleSubscribersForSamePlan(
	t *testing.T,
) {
	hub := NewLPSSEHub()
	planID := "plan-multiple-subscribers"

	first := hub.Subscribe(planID)

	if count := hub.SubscriberCount(planID); count != 1 {
		t.Fatalf(
			"第一条连接后的订阅数量错误：got=%d want=1",
			count,
		)
	}

	second := hub.Subscribe(planID)

	if count := hub.SubscriberCount(planID); count != 2 {
		t.Fatalf(
			"第二条连接后的订阅数量错误：got=%d want=2",
			count,
		)
	}

	event := models.LPSSEEvent{
		EventType: models.LPSSEThinking,
		PlanID:    planID,
	}

	hub.Broadcast(planID, event)

	firstEvent := receiveLessonPlanSSETestEvent(t, first)
	secondEvent := receiveLessonPlanSSETestEvent(t, second)

	if firstEvent.EventType != models.LPSSEThinking {
		t.Fatalf(
			"第一条连接收到错误事件：%s",
			firstEvent.EventType,
		)
	}

	if secondEvent.EventType != models.LPSSEThinking {
		t.Fatalf(
			"第二条连接收到错误事件：%s",
			secondEvent.EventType,
		)
	}

	hub.Unsubscribe(planID, first)
	assertLessonPlanSSETestChannelClosed(t, first)

	if count := hub.SubscriberCount(planID); count != 1 {
		t.Fatalf(
			"注销第一条连接后的订阅数量错误：got=%d want=1",
			count,
		)
	}

	followUp := models.LPSSEEvent{
		EventType: models.LPSSEContentUpdate,
		PlanID:    planID,
		Content:   "更新后的教案正文",
	}

	hub.Broadcast(planID, followUp)

	secondFollowUp := receiveLessonPlanSSETestEvent(
		t,
		second,
	)

	if secondFollowUp.EventType != models.LPSSEContentUpdate ||
		secondFollowUp.Content != "更新后的教案正文" {
		t.Fatalf(
			"注销第一条连接后，第二条连接收到的事件错误：%#v",
			secondFollowUp,
		)
	}

	hub.Unsubscribe(planID, second)
	assertLessonPlanSSETestChannelClosed(t, second)

	if count := hub.SubscriberCount(planID); count != 0 {
		t.Fatalf(
			"全部连接注销后的订阅数量错误：got=%d want=0",
			count,
		)
	}
}

// TestLPSSEHubKeepsPlansIsolated 验证不同教案之间不会串流。
func TestLPSSEHubKeepsPlansIsolated(
	t *testing.T,
) {
	hub := NewLPSSEHub()

	firstPlanChannel := hub.Subscribe("plan-first")
	secondPlanChannel := hub.Subscribe("plan-second")

	hub.Broadcast(
		"plan-first",
		models.LPSSEEvent{
			EventType: models.LPSSEMessageDone,
			PlanID:    "plan-first",
		},
	)

	firstEvent := receiveLessonPlanSSETestEvent(
		t,
		firstPlanChannel,
	)

	if firstEvent.PlanID != "plan-first" {
		t.Fatalf(
			"第一份教案收到错误plan_id：%s",
			firstEvent.PlanID,
		)
	}

	select {
	case event, open := <-secondPlanChannel:
		if !open {
			t.Fatal("第二份教案的连接被错误关闭")
		}

		t.Fatalf(
			"第二份教案错误收到第一份教案事件：%#v",
			event,
		)

	case <-time.After(50 * time.Millisecond):
		// 预期：第二份教案没有收到任何事件。
	}

	hub.Unsubscribe(
		"plan-first",
		firstPlanChannel,
	)
	hub.Unsubscribe(
		"plan-second",
		secondPlanChannel,
	)
}
