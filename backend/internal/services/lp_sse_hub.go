package services

// lp_sse_hub.go — 教案系统SSE广播中心
//
// v73 BugFix1：channel缓冲从20扩大到2000，避免write阶段chunk丢失
//
// v73 BugFix2：Subscribe时关闭旧连接（独占模式）
//   问题：用户切换页面再回来时，旧SSE连接goroutine尚未退出，
//         新连接建立后两路chunk同时写入前端streaming状态，
//         导致两段AI输出内容字符级交织，显示为乱码。
//   修复：同一planID同一时间只允许一个活跃SSE连接。
//         Subscribe时先关闭该planID的所有旧channel，再建新channel。
//   注意：关闭旧channel会让旧goroutine的 for-select 收到channel关闭信号，
//         自然退出，不会有资源泄漏。

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

// Subscribe 订阅指定教案的SSE事件（独占模式）
//
// v73 BugFix2：同一planID只允许一个活跃连接
//   新连接建立前，先关闭该planID的所有旧channel。
//   旧channel关闭后，持有它的SSE handler goroutine会从for-select退出，
//   彻底断开旧连接，避免两路chunk同时写入前端导致内容乱码。
//
// 缓冲设计：
//   write阶段单次AI回复可产生700-1000个chunk事件，
//   加上控制事件共约1100个，设置2000确保不丢包。
func (h *LPSSEHub) Subscribe(planID string) chan models.LPSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 关闭该planID的所有旧连接（独占模式核心）
	if oldSubs, exists := h.subscribers[planID]; exists && len(oldSubs) > 0 {
		lpSseLog.Info("关闭旧SSE连接，建立新连接",
			"plan_id", planID,
			"old_count", len(oldSubs),
		)
		for ch, active := range oldSubs {
			if active {
				oldSubs[ch] = false
				close(ch) // 关闭旧channel，让旧goroutine自然退出
			}
		}
		delete(h.subscribers, planID)
	}

	// 建立新连接
	ch := make(chan models.LPSSEEvent, 2000)
	h.subscribers[planID] = map[chan models.LPSSEEvent]bool{ch: true}

	lpSseLog.Debug("教案SSE新订阅（独占）",
		"plan_id", planID,
		"channel_buffer", 2000,
	)
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
//
// 独占模式下subscribers最多只有1个，Broadcast逻辑不变
func (h *LPSSEHub) Broadcast(planID string, event models.LPSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[planID]
	if !ok || len(subs) == 0 {
		return
	}

	sent := 0
	dropped := 0
	for ch, active := range subs {
		if !active {
			continue
		}
		select {
		case ch <- event:
			sent++
		default:
			dropped++
			lpSseLog.Warn("教案SSE channel已满，事件被丢弃",
				"plan_id", planID,
				"event_type", event.EventType,
				"channel_cap", cap(ch),
				"channel_len", len(ch),
			)
		}
	}

	if event.EventType != models.LPSSEChunk || dropped > 0 {
		lpSseLog.Debug("教案SSE广播",
			"plan_id", planID,
			"event_type", event.EventType,
			"sent", sent,
			"dropped", dropped,
		)
	}
}
