package services

// sse_hub.go — Pipeline SSE事件广播中心
//
// 功能：
//   - 按Pipeline ID管理SSE长连接；
//   - Pipeline步骤更新时向订阅者广播；
//   - 所有订阅、取消和广播操作使用同一把互斥锁，避免关闭竞态。
//
// 部署排空：
//   - 全局进入SSE draining后不再登记新Pipeline连接；
//   - Subscribe在持锁状态下检查draining；
//   - draining期间返回已关闭channel；
//   - CloseAll在同一把锁内关闭并清空全部连接，实现在sse_drain.go。

import (
	"sync"

	"tedna/internal/logger"
)

// SSEEvent Pipeline推送事件。
type SSEEvent struct {
	EventType   string `json:"type"`
	PipelineID  string `json:"pipeline_id"`
	CurrentStep string `json:"current_step"`
	StepStatus  string `json:"step_status"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// SSEHub Pipeline SSE广播中心。
type SSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan SSEEvent]bool
}

var sseLog = logger.WithModule("sse")

// GlobalSSEHub 全局Pipeline SSE广播中心。
var GlobalSSEHub = NewSSEHub()

// NewSSEHub 创建Pipeline SSE广播中心。
func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan SSEEvent]bool),
	}
}

// Subscribe 订阅指定Pipeline的SSE事件。
func (h *SSEHub) Subscribe(pipelineID string) chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	if IsGlobalSSEDraining() {
		ch := make(chan SSEEvent)
		close(ch)

		sseLog.Debug("服务正在排空，拒绝新的Pipeline SSE订阅",
			"pipeline_id", pipelineID,
		)
		return ch
	}

	ch := make(chan SSEEvent, 10)

	if h.subscribers[pipelineID] == nil {
		h.subscribers[pipelineID] = make(map[chan SSEEvent]bool)
	}

	h.subscribers[pipelineID][ch] = true

	sseLog.Debug("Pipeline SSE新订阅",
		"pipeline_id", pipelineID,
		"subscriber_count", len(h.subscribers[pipelineID]),
	)

	return ch
}

// Unsubscribe 取消Pipeline SSE订阅。
func (h *SSEHub) Unsubscribe(
	pipelineID string,
	ch chan SSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[pipelineID]
	if !exists {
		return
	}

	active, channelExists := subs[ch]
	if !channelExists {
		return
	}

	if active {
		subs[ch] = false
		close(ch)
		delete(subs, ch)
	}

	if len(subs) == 0 {
		delete(h.subscribers, pipelineID)
	}

	remaining := 0
	if current, stillExists := h.subscribers[pipelineID]; stillExists {
		remaining = len(current)
	}

	sseLog.Debug("Pipeline SSE取消订阅",
		"pipeline_id", pipelineID,
		"remaining_subscribers", remaining,
	)
}

// Broadcast 向指定Pipeline的全部订阅者广播事件。
func (h *SSEHub) Broadcast(
	pipelineID string,
	event SSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[pipelineID]
	if !exists || len(subs) == 0 {
		return
	}

	sent := 0

	for ch, active := range subs {
		if !active {
			continue
		}

		select {
		case ch <- event:
			sent++
		default:
			// 慢消费者channel已满时跳过，不阻塞Pipeline执行。
		}
	}

	sseLog.Debug("Pipeline SSE广播事件",
		"pipeline_id", pipelineID,
		"event_type", event.EventType,
		"step", event.CurrentStep,
		"subscriber_count", sent,
	)
}

// GetSubscriberCount 返回指定Pipeline订阅者数量。
func (h *SSEHub) GetSubscriberCount(pipelineID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.subscribers[pipelineID])
}

// GetTotalSubscribers 返回全部Pipeline订阅者数量。
func (h *SSEHub) GetTotalSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0
	for _, subs := range h.subscribers {
		for _, active := range subs {
			if active {
				total++
			}
		}
	}

	return total
}
