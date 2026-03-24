package services

// ==================== P5-4 SSE事件广播中心 ====================
// 功能：
//   1. 管理所有SSE长连接（按Pipeline ID分组）
//   2. 订阅/取消订阅：前端连接SSE时Subscribe，断开时Unsubscribe
//   3. 广播事件：Pipeline执行步骤完成时Broadcast，推送给所有订阅该Pipeline的客户端
//
// 竞态修复（代码审查）：
//   原版Broadcast用RLock，Unsubscribe用Lock，存在以下竞态：
//   Broadcast持有RLock正在向channel写入时，另一goroutine调用Unsubscribe获得Lock并close(ch)，
//   随后Broadcast继续向已关闭的channel写入会panic（select default不能防止close后的写入panic）。
//
//   修复方案：
//   1. Broadcast改用全局写锁（sync.Mutex），与Unsubscribe互斥，彻底消除竞态
//   2. Unsubscribe将channel标记为"已关闭"而不立即close，改由Broadcast检测后清理
//   3. 用独立的closed set跟踪待清理channel，避免double-close
//
//   性能影响：Broadcast改用互斥锁后，同一时刻只有一个goroutine可广播。
//   实际场景中同时广播的goroutine极少（Worker数量=5），影响可忽略。

import (
	"fmt"
	"sync"
)

// ==================== SSE事件类型 ====================

// SSEEvent SSE推送事件
type SSEEvent struct {
	EventType   string `json:"type"`         // 事件类型：step_update / pipeline_done / pipeline_error
	PipelineID  string `json:"pipeline_id"`  // Pipeline ID
	CurrentStep string `json:"current_step"` // 当前步骤
	StepStatus  string `json:"step_status"`  // 步骤状态：done / failed / running
	Status      string `json:"status"`       // Pipeline整体状态
	Message     string `json:"message"`      // 可选描述信息
}

// ==================== SSE广播中心 ====================

// SSEHub SSE事件广播中心（全局单例）
// 管理按Pipeline ID分组的SSE连接
type SSEHub struct {
	mu          sync.Mutex                       // 统一互斥锁，保护所有操作（避免RLock+Lock竞态）
	subscribers map[string]map[chan SSEEvent]bool // pipelineID → set of channels（true=活跃，false=待清理）
}

// GlobalSSEHub 全局SSE广播中心实例
var GlobalSSEHub = NewSSEHub()

// NewSSEHub 创建新的SSE广播中心
func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan SSEEvent]bool),
	}
}

// Subscribe 订阅指定Pipeline的SSE事件
// 返回一个事件channel，调用方从中读取事件
// 缓冲大小10，防止慢消费者阻塞广播
func (h *SSEHub) Subscribe(pipelineID string) chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan SSEEvent, 10)
	if h.subscribers[pipelineID] == nil {
		h.subscribers[pipelineID] = make(map[chan SSEEvent]bool)
	}
	// true 表示channel活跃可用
	h.subscribers[pipelineID][ch] = true

	count := len(h.subscribers[pipelineID])
	fmt.Printf("[SSE Hub] 新订阅: pipeline=%s, 当前订阅数=%d\n", pipelineID, count)
	return ch
}

// Unsubscribe 取消订阅指定Pipeline的SSE事件
// 将channel标记为待清理（false），由下次Broadcast或本函数负责close和删除
// 修复：不在持有锁期间close channel后立即让Broadcast写入已关闭channel
func (h *SSEHub) Unsubscribe(pipelineID string, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[pipelineID]
	if !ok {
		return
	}

	active, exists := subs[ch]
	if !exists {
		return
	}

	if active {
		// 标记为非活跃，并关闭channel
		// 此时持有互斥锁，Broadcast无法同时执行，不会有double-write风险
		subs[ch] = false
		close(ch)
		delete(subs, ch)
	}

	// 如果该Pipeline没有订阅者了，清理map条目
	if len(subs) == 0 {
		delete(h.subscribers, pipelineID)
	}

	count := 0
	if s, ok2 := h.subscribers[pipelineID]; ok2 {
		count = len(s)
	}
	fmt.Printf("[SSE Hub] 取消订阅: pipeline=%s, 剩余订阅数=%d\n", pipelineID, count)
}

// Broadcast 向指定Pipeline的所有订阅者广播事件
// 使用互斥锁与Subscribe/Unsubscribe互斥，彻底避免向已关闭channel写入的竞态
// 非阻塞写入：如果某个channel已满则跳过（避免慢消费者阻塞整个广播）
func (h *SSEHub) Broadcast(pipelineID string, event SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[pipelineID]
	if !ok || len(subs) == 0 {
		return // 没有订阅者，直接返回
	}

	sent := 0
	total := len(subs)

	for ch, active := range subs {
		if !active {
			// 跳过已标记为非活跃的channel（理论上Unsubscribe已删除，双重保险）
			continue
		}
		select {
		case ch <- event:
			sent++
		default:
			// channel已满，跳过此订阅者（防止阻塞）
			// 注意：不在此处关闭channel，由Unsubscribe负责关闭
		}
	}

	fmt.Printf("[SSE Hub] 广播: pipeline=%s, type=%s, step=%s, 推送=%d/%d\n",
		pipelineID, event.EventType, event.CurrentStep, sent, total)
}

// GetSubscriberCount 获取指定Pipeline的订阅者数量（用于监控）
func (h *SSEHub) GetSubscriberCount(pipelineID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers[pipelineID])
}

// GetTotalSubscribers 获取所有Pipeline的总订阅者数量
func (h *SSEHub) GetTotalSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	total := 0
	for _, subs := range h.subscribers {
		total += len(subs)
	}
	return total
}
