package services

// ==================== 教案系统SSE广播中心 ====================
// 与Pipeline SSEHub独立，避免ID冲突
// 结构完全复用Pipeline SSEHub的设计（互斥锁+安全close）

import (
	"sync"
	"tedna/internal/logger"
	"tedna/internal/models"
)

// LPSSEHub 教案生成SSE广播中心（全局单例）
type LPSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan models.LPSSEEvent]bool
}

var lpSseLog = logger.WithModule("lp_sse")

// GlobalLPSSEHub 全局教案SSE广播中心
var GlobalLPSSEHub = NewLPSSEHub()

// NewLPSSEHub 创建教案SSE广播中心
func NewLPSSEHub() *LPSSEHub {
	return &LPSSEHub{
		subscribers: make(map[string]map[chan models.LPSSEEvent]bool),
	}
}

// Subscribe 订阅指定教案的SSE事件
func (h *LPSSEHub) Subscribe(planID string) chan models.LPSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan models.LPSSEEvent, 20) // 缓冲20，教案生成事件比Pipeline更密集
	if h.subscribers[planID] == nil {
		h.subscribers[planID] = make(map[chan models.LPSSEEvent]bool)
	}
	h.subscribers[planID][ch] = true

	lpSseLog.Debug("教案SSE新订阅", "plan_id", planID,
		"subscriber_count", len(h.subscribers[planID]))
	return ch
}

// Unsubscribe 取消订阅
func (h *LPSSEHub) Unsubscribe(planID string, ch chan models.LPSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[planID]
	if !ok {
		return
	}
	if active, exists := subs[ch]; exists && active {
		subs[ch] = false
		close(ch)
		delete(subs, ch)
	}
	if len(subs) == 0 {
		delete(h.subscribers, planID)
	}
}

// Broadcast 向指定教案的所有订阅者广播事件
func (h *LPSSEHub) Broadcast(planID string, event models.LPSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[planID]
	if !ok || len(subs) == 0 {
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
			// channel已满，跳过
		}
	}
	lpSseLog.Debug("教案SSE广播", "plan_id", planID,
		"event_type", event.EventType, "sent", sent)
}
