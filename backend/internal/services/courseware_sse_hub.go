package services

// courseware_sse_hub.go — 课件工坊SSE广播中心
//
// 用于课件索引生成、HTML生成、模板微调和全自动装配进度推送。
// 同一coursewareID采用独占连接，新连接建立时关闭旧连接。
//
// 部署排空：
//   - 全局进入SSE draining后不再登记新课件SSE连接；
//   - Subscribe在持有Hub锁时检查draining，避免与CloseAll发生竞态；
//   - draining期间返回已关闭channel，Handler可自然退出；
//   - CloseAll实现位于sse_drain.go。
//
// 防御：
//   - safeCloseCWChan防止double-close；
//   - safeSendCWEvent防止send-on-closed；
//   - Broadcast与关闭动作共用同一把互斥锁。

import (
	"sync"

	"tedna/internal/logger"
)

// ==================== 课件SSE事件类型常量 ====================

const (
	// 索引生成阶段。
	CWSSEConnected     = "connected"
	CWSSEIndexStart    = "index_start"
	CWSSEIndexPage     = "index_page"
	CWSSEIndexProgress = "index_progress"
	CWSSEIndexDone     = "index_done"

	// 课件HTML生成阶段。
	CWSSEGenStart    = "gen_start"
	CWSSEGenPage     = "gen_page"
	CWSSEGenProgress = "gen_progress"
	CWSSEGenDone     = "gen_done"

	// 通用事件。
	CWSSEChunk = "chunk"
	CWSSEError = "error"
)

// CWSSEEvent 课件工坊SSE事件。
type CWSSEEvent struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

// ==================== 防御性辅助函数 ====================

var cwSseLog = logger.WithModule("cw_sse")

// safeCloseCWChan 安全关闭课件SSE channel。
func safeCloseCWChan(ch chan CWSSEEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			cwSseLog.Warn("课件SSE channel double-close被捕获",
				"recover", recovered,
			)
		}
	}()

	close(ch)
}

// safeSendCWEvent 非阻塞发送课件SSE事件。
func safeSendCWEvent(ch chan CWSSEEvent, event CWSSEEvent) bool {
	sent := false

	defer func() {
		if recovered := recover(); recovered != nil {
			cwSseLog.Warn("课件SSE send-on-closed被捕获",
				"event_type", event.EventType,
				"recover", recovered,
			)
			sent = false
		}
	}()

	select {
	case ch <- event:
		sent = true
	default:
		sent = false
	}

	return sent
}

// ==================== 课件SSE广播中心 ====================

// CWSSEHub 课件工坊SSE广播中心。
type CWSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan CWSSEEvent]bool
}

// GlobalCWSSEHub 全局课件SSE广播中心。
var GlobalCWSSEHub = NewCWSSEHub()

// NewCWSSEHub 创建课件SSE广播中心。
func NewCWSSEHub() *CWSSEHub {
	return &CWSSEHub{
		subscribers: make(map[string]map[chan CWSSEEvent]bool),
	}
}

// Subscribe 订阅指定课件的SSE事件。
func (h *CWSSEHub) Subscribe(coursewareID string) chan CWSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	if IsGlobalSSEDraining() {
		ch := make(chan CWSSEEvent)
		close(ch)

		cwSseLog.Debug("服务正在排空，拒绝新的课件SSE订阅",
			"courseware_id", coursewareID,
		)
		return ch
	}

	// 独占模式：关闭同一coursewareID的旧连接。
	if oldSubs, exists := h.subscribers[coursewareID]; exists && len(oldSubs) > 0 {
		cwSseLog.Info("关闭旧课件SSE连接，建立新连接",
			"courseware_id", coursewareID,
			"old_count", len(oldSubs),
		)

		for ch, active := range oldSubs {
			if !active {
				continue
			}

			oldSubs[ch] = false
			safeCloseCWChan(ch)
		}

		delete(h.subscribers, coursewareID)
	}

	ch := make(chan CWSSEEvent, 2000)
	h.subscribers[coursewareID] = map[chan CWSSEEvent]bool{
		ch: true,
	}

	cwSseLog.Debug("课件SSE新订阅",
		"courseware_id", coursewareID,
		"channel_buffer", 2000,
	)

	return ch
}

// Unsubscribe 取消课件SSE订阅。
func (h *CWSSEHub) Unsubscribe(
	coursewareID string,
	ch chan CWSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[coursewareID]
	if !exists {
		return
	}

	if active, channelExists := subs[ch]; channelExists && active {
		subs[ch] = false
		safeCloseCWChan(ch)
		delete(subs, ch)
	}

	if len(subs) == 0 {
		delete(h.subscribers, coursewareID)
	}
}

// Broadcast 向指定课件的全部订阅者广播事件。
func (h *CWSSEHub) Broadcast(
	coursewareID string,
	event CWSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[coursewareID]
	if !exists || len(subs) == 0 {
		return
	}

	sent := 0
	dropped := 0

	for ch, active := range subs {
		if !active {
			continue
		}

		if safeSendCWEvent(ch, event) {
			sent++
			continue
		}

		dropped++
		cwSseLog.Warn("课件SSE channel已满或已关闭，事件被丢弃",
			"courseware_id", coursewareID,
			"event_type", event.EventType,
		)
	}

	if event.EventType != CWSSEChunk || dropped > 0 {
		cwSseLog.Debug("课件SSE广播",
			"courseware_id", coursewareID,
			"event_type", event.EventType,
			"sent", sent,
			"dropped", dropped,
		)
	}
}
