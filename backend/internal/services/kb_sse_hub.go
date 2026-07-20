package services

// kb_sse_hub.go — 知识库压缩进度SSE广播中心
//
// 按jobID管理知识库压缩进度连接，同一个jobID采用独占订阅。
//
// 部署排空：
//   - 全局进入SSE draining后不再登记新知识库SSE连接；
//   - Subscribe在持有Hub锁时检查draining；
//   - draining期间返回已关闭channel；
//   - CloseAll实现位于sse_drain.go。
//
// 防御：
//   - safeCloseKBChan防止double-close；
//   - safeSendKBEvent防止send-on-closed；
//   - 广播和关闭共用互斥锁。

import (
	"sync"

	"tedna/internal/logger"
)

// ==================== 知识库SSE事件类型 ====================

const (
	KBSSEConnected    = "connected"
	KBSSEExtractStart = "extract_start"
	KBSSEExtractDone  = "extract_done"
	KBSSEItemStart    = "item_start"
	KBSSEItemDone     = "item_done"
	KBSSEProgress     = "progress"
	KBSSEJobDone      = "job_done"
	KBSSEError        = "error"
)

// KBSSEEvent 知识库压缩SSE事件。
type KBSSEEvent struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

var kbSseLog = logger.WithModule("kb_sse")

// safeCloseKBChan 安全关闭知识库SSE channel。
func safeCloseKBChan(ch chan KBSSEEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			kbSseLog.Warn("知识库SSE channel double-close被捕获",
				"recover", recovered,
			)
		}
	}()

	close(ch)
}

// safeSendKBEvent 非阻塞发送知识库SSE事件。
func safeSendKBEvent(ch chan KBSSEEvent, event KBSSEEvent) bool {
	sent := false

	defer func() {
		if recovered := recover(); recovered != nil {
			kbSseLog.Warn("知识库SSE send-on-closed被捕获",
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

// KBSSEHub 知识库压缩SSE广播中心。
type KBSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan KBSSEEvent]bool
}

// GlobalKBSSEHub 全局知识库SSE广播中心。
var GlobalKBSSEHub = NewKBSSEHub()

// NewKBSSEHub 创建知识库SSE广播中心。
func NewKBSSEHub() *KBSSEHub {
	return &KBSSEHub{
		subscribers: make(map[string]map[chan KBSSEEvent]bool),
	}
}

// Subscribe 订阅指定知识库任务的SSE事件。
func (h *KBSSEHub) Subscribe(jobID string) chan KBSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	if IsGlobalSSEDraining() {
		ch := make(chan KBSSEEvent)
		close(ch)

		kbSseLog.Debug("服务正在排空，拒绝新的知识库SSE订阅",
			"job_id", jobID,
		)
		return ch
	}

	// 独占模式：关闭同一jobID旧连接。
	if oldSubs, exists := h.subscribers[jobID]; exists && len(oldSubs) > 0 {
		for ch, active := range oldSubs {
			if !active {
				continue
			}

			oldSubs[ch] = false
			safeCloseKBChan(ch)
		}

		delete(h.subscribers, jobID)
	}

	ch := make(chan KBSSEEvent, 500)
	h.subscribers[jobID] = map[chan KBSSEEvent]bool{
		ch: true,
	}

	return ch
}

// Unsubscribe 取消知识库SSE订阅。
func (h *KBSSEHub) Unsubscribe(
	jobID string,
	ch chan KBSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[jobID]
	if !exists {
		return
	}

	if active, channelExists := subs[ch]; channelExists && active {
		subs[ch] = false
		safeCloseKBChan(ch)
		delete(subs, ch)
	}

	if len(subs) == 0 {
		delete(h.subscribers, jobID)
	}
}

// Broadcast 向指定知识库任务的全部订阅者广播事件。
func (h *KBSSEHub) Broadcast(
	jobID string,
	event KBSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[jobID]
	if !exists || len(subs) == 0 {
		return
	}

	for ch, active := range subs {
		if !active {
			continue
		}

		if safeSendKBEvent(ch, event) {
			continue
		}

		kbSseLog.Warn("知识库SSE channel已满或已关闭，事件被丢弃",
			"job_id", jobID,
			"event_type", event.EventType,
		)
	}
}
