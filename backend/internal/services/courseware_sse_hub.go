package services

// courseware_sse_hub.go — 课件工坊SSE广播中心
//
// 复用教案SSE Hub模式（lp_sse_hub.go）
// 用于课件索引生成、课件AI生成等流式推送场景
// 独占模式：同一coursewareID同一时间只允许一个活跃SSE连接

import (
	"sync"
	"tedna/internal/logger"
)

// ==================== 课件SSE事件类型常量 ====================

const (
	CWSSEConnected     = "connected"      // SSE连接建立
	CWSSEIndexStart    = "index_start"    // 开始生成索引
	CWSSEIndexPage     = "index_page"     // 单页索引生成完成
	CWSSEIndexProgress = "index_progress" // 索引生成进度
	CWSSEIndexDone     = "index_done"     // 索引生成全部完成
	CWSSEChunk         = "chunk"          // AI流式输出片段
	CWSSEError         = "error"          // 错误
)

// ==================== 课件SSE事件结构 ====================

// CWSSEEvent 课件工坊SSE事件
type CWSSEEvent struct {
	EventType string      `json:"event_type"` // 事件类型
	Data      interface{} `json:"data"`       // 事件数据（根据类型不同而不同）
}

// ==================== 课件SSE广播中心 ====================

var cwSseLog = logger.WithModule("cw_sse")

// CWSSEHub 课件工坊SSE广播中心（全局单例）
type CWSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan CWSSEEvent]bool // coursewareID → channels
}

// GlobalCWSSEHub 全局课件SSE广播中心
var GlobalCWSSEHub = NewCWSSEHub()

// NewCWSSEHub 创建课件SSE广播中心
func NewCWSSEHub() *CWSSEHub {
	return &CWSSEHub{
		subscribers: make(map[string]map[chan CWSSEEvent]bool),
	}
}

// Subscribe 订阅指定课件的SSE事件（独占模式）
// 新连接建立前先关闭该coursewareID的所有旧channel
// 缓冲2000：索引生成每页约产生10-20个事件，30页课件约300-600个事件，留足余量
func (h *CWSSEHub) Subscribe(coursewareID string) chan CWSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 关闭该coursewareID的所有旧连接（独占模式核心）
	if oldSubs, exists := h.subscribers[coursewareID]; exists && len(oldSubs) > 0 {
		cwSseLog.Info("关闭旧SSE连接，建立新连接",
			"courseware_id", coursewareID,
			"old_count", len(oldSubs),
		)
		for ch, active := range oldSubs {
			if active {
				oldSubs[ch] = false
				close(ch)
			}
		}
		delete(h.subscribers, coursewareID)
	}

	// 建立新连接
	ch := make(chan CWSSEEvent, 2000)
	h.subscribers[coursewareID] = map[chan CWSSEEvent]bool{ch: true}

	cwSseLog.Debug("课件SSE新订阅（独占）",
		"courseware_id", coursewareID,
		"channel_buffer", 2000,
	)
	return ch
}

// Unsubscribe 取消订阅
func (h *CWSSEHub) Unsubscribe(coursewareID string, ch chan CWSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[coursewareID]
	if !ok {
		return
	}
	if active, exists := subs[ch]; exists && active {
		subs[ch] = false
		close(ch)
		delete(subs, ch)
	}
	if len(subs) == 0 {
		delete(h.subscribers, coursewareID)
	}
}

// Broadcast 向指定课件的所有订阅者广播事件
func (h *CWSSEHub) Broadcast(coursewareID string, event CWSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[coursewareID]
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
			cwSseLog.Warn("课件SSE channel已满，事件被丢弃",
				"courseware_id", coursewareID,
				"event_type", event.EventType,
				"channel_cap", cap(ch),
				"channel_len", len(ch),
			)
		}
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
