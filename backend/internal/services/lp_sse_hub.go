package services

// lp_sse_hub.go — 教案系统SSE广播中心
//
// 连接模型：
//   - 按planID保存全部活动订阅channel；
//   - 同一planID允许多个浏览器标签页、窗口或设备同时订阅；
//   - 新连接不会关闭旧连接，避免客户端自动重连形成互相踢下线循环；
//   - 每条连接拥有独立channel，关闭某个连接只注销该连接；
//   - channel缓冲2000，覆盖长教案生成产生的大量chunk；
//   - Broadcast向同一教案的全部活动订阅者非阻塞发送；
//   - 单个channel已满时只丢弃该连接的当前事件，不影响其它订阅者。
//
// 接口兼容：
//   - Subscribe继续保持原有单返回值协议，避免影响现有处理器、排空逻辑和测试；
//   - SubscriberCount作为独立只读方法提供诊断数量，不改变已有调用合同。
//
// 部署排空：
//   - 全局进入SSE draining后，不再建立新的教案SSE连接；
//   - Subscribe在持有Hub互斥锁时二次确认draining状态，避免与CloseAll竞态；
//   - draining期间返回一个已经关闭的channel，现有Handler读取到open=false后自然退出；
//   - CloseAll实现位于sse_drain.go，与Subscribe、Broadcast和Unsubscribe共用本Hub的锁。
//
// 防御：
//   - safeCloseLPChan防止double-close panic；
//   - safeSendLPEvent防止send-on-closed panic；
//   - 正常路径通过互斥锁保证关闭和发送不会并发执行。

import (
	"sync"

	"tedna/internal/logger"
	"tedna/internal/models"
)

const lessonPlanSSEChannelBuffer = 2000

// ==================== 防御性辅助函数 ====================

var lpSseLog = logger.WithModule("lp_sse")

// safeCloseLPChan 安全关闭教案SSE channel。
func safeCloseLPChan(ch chan models.LPSSEEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			lpSseLog.Warn(
				"教案SSE channel double-close被捕获",
				"recover",
				recovered,
			)
		}
	}()

	close(ch)
}

// safeSendLPEvent 非阻塞发送教案SSE事件。
//
// 返回true表示发送成功；返回false表示channel已满或已经关闭。
func safeSendLPEvent(
	ch chan models.LPSSEEvent,
	event models.LPSSEEvent,
) bool {
	sent := false

	defer func() {
		if recovered := recover(); recovered != nil {
			lpSseLog.Warn(
				"教案SSE send-on-closed被捕获",
				"event_type",
				event.EventType,
				"recover",
				recovered,
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
		subscribers: make(
			map[string]map[chan models.LPSSEEvent]bool,
		),
	}
}

// Subscribe 订阅指定教案的SSE事件。
//
// 同一个planID允许存在多条活动连接。这样可以支持：
//   - 同一老师打开多个浏览器标签页；
//   - 同一账号在不同设备上查看同一教案；
//   - React组件短时间内发生连接生命周期交叠。
//
// 新连接绝不能关闭旧连接，否则两个自动重连客户端会形成无限互踢。
//
// 本函数保持历史单返回值协议。服务进入draining后返回已经关闭的channel，
// 不登记新连接，由现有Handler读取到open=false后自然退出。
func (h *LPSSEHub) Subscribe(
	planID string,
) chan models.LPSSEEvent {
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

		lpSseLog.Debug(
			"服务正在排空，拒绝新的教案SSE订阅",
			"plan_id",
			planID,
		)

		return ch
	}

	subs, exists := h.subscribers[planID]
	if !exists {
		subs = make(
			map[chan models.LPSSEEvent]bool,
		)
		h.subscribers[planID] = subs
	}

	ch := make(
		chan models.LPSSEEvent,
		lessonPlanSSEChannelBuffer,
	)
	subs[ch] = true

	subscriberCount := len(subs)

	if subscriberCount > 1 {
		lpSseLog.Info(
			"同一教案新增并行SSE连接",
			"plan_id",
			planID,
			"subscriber_count",
			subscriberCount,
		)
	} else {
		lpSseLog.Debug(
			"教案SSE新订阅",
			"plan_id",
			planID,
			"subscriber_count",
			subscriberCount,
			"channel_buffer",
			lessonPlanSSEChannelBuffer,
		)
	}

	return ch
}

// SubscriberCount 返回指定教案当前活动SSE订阅数量。
//
// 本方法只用于诊断日志和测试，不参与业务判断。调用完成后数量可能因其它
// 连接建立或退出而变化，因此调用方不得依赖该值实施权限或一致性控制。
func (h *LPSSEHub) SubscriberCount(
	planID string,
) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, exists := h.subscribers[planID]
	if !exists {
		return 0
	}

	count := 0

	for _, active := range subs {
		if active {
			count++
		}
	}

	return count
}

// Unsubscribe 取消指定教案中的一条SSE订阅。
//
// 只关闭和删除调用方传入的channel，不影响同一教案的其它连接。
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

	remainingCount := len(subs)

	if remainingCount == 0 {
		delete(h.subscribers, planID)
	}

	lpSseLog.Debug(
		"教案SSE订阅已注销",
		"plan_id",
		planID,
		"remaining_count",
		remainingCount,
	)
}

// Broadcast 向指定教案的全部活动订阅者广播事件。
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

		lpSseLog.Warn(
			"教案SSE channel已满或已关闭，事件被丢弃",
			"plan_id",
			planID,
			"event_type",
			event.EventType,
		)
	}

	if event.EventType != models.LPSSEChunk || dropped > 0 {
		lpSseLog.Debug(
			"教案SSE广播",
			"plan_id",
			planID,
			"event_type",
			event.EventType,
			"subscriber_count",
			len(subs),
			"sent",
			sent,
			"dropped",
			dropped,
		)
	}
}
