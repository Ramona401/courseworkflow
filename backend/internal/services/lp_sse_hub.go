package services

// lp_sse_hub.go — 教案系统SSE广播中心
//
// 连接模型：
//   - 按planID保存订阅channel；
//   - 同一planID采用独占连接，新连接建立时关闭旧连接；
//   - channel缓冲2000，覆盖长教案生成产生的大量chunk；
//   - Broadcast非阻塞发送，满时丢弃并记录日志。
//
// 部署排空：
//   - 全局进入SSE draining后，不再建立新的教案SSE连接；
//   - Subscribe在持有Hub互斥锁时二次确认draining状态，避免与CloseAll竞态；
//   - draining期间返回一个已经关闭的channel，现有Handler读取到open=false后自然退出；
//   - CloseAll实现位于sse_drain.go，与Subscribe/Broadcast/Unsubscribe共用本Hub的锁。
//
// 防御：
//   - safeCloseLPChan防止double-close panic；
//   - safeSendLPEvent防止send-on-closed panic；
//   - 正常路径仍通过互斥锁保证关闭和发送不会并发执行。

import (
	"sync"

	"tedna/internal/logger"
	"tedna/internal/models"
)

// ==================== 防御性辅助函数 ====================

var lpSseLog = logger.WithModule("lp_sse")

// safeCloseLPChan 安全关闭教案SSE channel。
func safeCloseLPChan(ch chan models.LPSSEEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			lpSseLog.Warn("教案SSE channel double-close被捕获",
				"recover", recovered,
			)
		}
	}()

	close(ch)
}

// safeSendLPEvent 非阻塞发送教案SSE事件。
//
// 返回true表示发送成功；返回false表示channel已满或已经关闭。
func safeSendLPEvent(ch chan models.LPSSEEvent, event models.LPSSEEvent) bool {
	sent := false

	defer func() {
		if recovered := recover(); recovered != nil {
			lpSseLog.Warn("教案SSE send-on-closed被捕获",
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

// ==================== 教案SSE广播中心 ====================

// LPSSEHub 教案生成SSE广播中心。
type LPSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan models.LPSSEEvent]bool
}

// GlobalLPSSEHub 全局教案SSE广播中心。
var GlobalLPSSEHub = NewLPSSEHub()

// NewLPSSEHub 创建教案SSE广播中心。
func NewLPSSEHub() *LPSSEHub {
	return &LPSSEHub{
		subscribers: make(map[string]map[chan models.LPSSEEvent]bool),
	}
}

// Subscribe 订阅指定教案的SSE事件。
//
// 同一个planID只保留一条活动连接。
// 服务进入draining后返回已关闭channel，不再登记新连接。
func (h *LPSSEHub) Subscribe(planID string) chan models.LPSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 必须在持有Hub锁后检查。
	//
	// BeginGlobalSSEDraining会先设置全局draining，再逐个锁住Hub执行CloseAll。
	// 因此：
	//   - 本方法先拿到锁并创建连接时，随后CloseAll一定会将其关闭；
	//   - CloseAll先执行或draining已设置时，本方法直接返回关闭channel；
	// 不会出现排空后又遗留新连接的窗口。
	if IsGlobalSSEDraining() {
		ch := make(chan models.LPSSEEvent)
		close(ch)

		lpSseLog.Debug("服务正在排空，拒绝新的教案SSE订阅",
			"plan_id", planID,
		)
		return ch
	}

	// 独占模式：关闭同一planID的旧连接。
	if oldSubs, exists := h.subscribers[planID]; exists && len(oldSubs) > 0 {
		lpSseLog.Info("关闭旧教案SSE连接，建立新连接",
			"plan_id", planID,
			"old_count", len(oldSubs),
		)

		for ch, active := range oldSubs {
			if !active {
				continue
			}

			oldSubs[ch] = false
			safeCloseLPChan(ch)
		}

		delete(h.subscribers, planID)
	}

	ch := make(chan models.LPSSEEvent, 2000)
	h.subscribers[planID] = map[chan models.LPSSEEvent]bool{
		ch: true,
	}

	lpSseLog.Debug("教案SSE新订阅",
		"plan_id", planID,
		"channel_buffer", 2000,
	)

	return ch
}

// Unsubscribe 取消指定教案的SSE订阅。
func (h *LPSSEHub) Unsubscribe(
	planID string,
	ch chan models.LPSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[planID]
	if !exists {
		return
	}

	if active, channelExists := subs[ch]; channelExists && active {
		subs[ch] = false
		safeCloseLPChan(ch)
		delete(subs, ch)
	}

	if len(subs) == 0 {
		delete(h.subscribers, planID)
	}
}

// Broadcast 向指定教案的全部订阅者广播事件。
func (h *LPSSEHub) Broadcast(
	planID string,
	event models.LPSSEEvent,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[planID]
	if !exists || len(subs) == 0 {
		return
	}

	sent := 0
	dropped := 0

	for ch, active := range subs {
		if !active {
			continue
		}

		if safeSendLPEvent(ch, event) {
			sent++
			continue
		}

		dropped++
		lpSseLog.Warn("教案SSE channel已满或已关闭，事件被丢弃",
			"plan_id", planID,
			"event_type", event.EventType,
		)
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
