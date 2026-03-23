package services

// ==================== P5-4 SSE事件广播中心 ====================
// 功能：
//   1. 管理所有SSE长连接（按Pipeline ID分组）
//   2. 订阅/取消订阅：前端连接SSE时Subscribe，断开时Unsubscribe
//   3. 广播事件：Pipeline执行步骤完成时Broadcast，推送给所有订阅该Pipeline的客户端
//
// 使用方式：
//   GlobalSSEHub.Subscribe(pipelineID) → chan SSEEvent
//   GlobalSSEHub.Unsubscribe(pipelineID, ch)
//   GlobalSSEHub.Broadcast(pipelineID, event)

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
	mu          sync.RWMutex
	subscribers map[string]map[chan SSEEvent]bool // pipelineID → set of channels
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
	h.subscribers[pipelineID][ch] = true

	count := len(h.subscribers[pipelineID])
	fmt.Printf("[SSE Hub] 新订阅: pipeline=%s, 当前订阅数=%d\n", pipelineID, count)
	return ch
}

// Unsubscribe 取消订阅指定Pipeline的SSE事件
// 关闭channel并从订阅列表中移除
func (h *SSEHub) Unsubscribe(pipelineID string, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.subscribers[pipelineID]; ok {
		if _, exists := subs[ch]; exists {
			delete(subs, ch)
			close(ch)
		}
		// 如果该Pipeline没有订阅者了，清理map条目
		if len(subs) == 0 {
			delete(h.subscribers, pipelineID)
		}
	}

	count := 0
	if subs, ok := h.subscribers[pipelineID]; ok {
		count = len(subs)
	}
	fmt.Printf("[SSE Hub] 取消订阅: pipeline=%s, 剩余订阅数=%d\n", pipelineID, count)
}

// Broadcast 向指定Pipeline的所有订阅者广播事件
// 非阻塞：如果某个channel已满则跳过（避免慢消费者阻塞整个广播）
func (h *SSEHub) Broadcast(pipelineID string, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subs, ok := h.subscribers[pipelineID]
	if !ok || len(subs) == 0 {
		return // 没有订阅者，直接返回
	}

	sent := 0
	for ch := range subs {
		select {
		case ch <- event:
			sent++
		default:
			// channel已满，跳过此订阅者（防止阻塞）
		}
	}

	fmt.Printf("[SSE Hub] 广播: pipeline=%s, type=%s, step=%s, 推送=%d/%d\n",
		pipelineID, event.EventType, event.CurrentStep, sent, len(subs))
}

// GetSubscriberCount 获取指定Pipeline的订阅者数量（用于监控）
func (h *SSEHub) GetSubscriberCount(pipelineID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[pipelineID])
}

// GetTotalSubscribers 获取所有Pipeline的总订阅者数量
func (h *SSEHub) GetTotalSubscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, subs := range h.subscribers {
		total += len(subs)
	}
	return total
}
