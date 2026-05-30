package services

// courseware_sse_hub.go — 课件工坊SSE广播中心 (v142 P3-3 竞态修复)
//
// 复用教案SSE Hub模式（lp_sse_hub.go）
// 用于课件索引生成、课件AI生成、模板微调等流式推送场景
// 独占模式：同一coursewareID同一时间只允许一个活跃SSE连接
//
// v142 P3-3 修复:
//   - safeCloseCWChan: 防止double-close panic(recover保护)
//   - Broadcast: send操作加recover防御,防止send-on-closed panic
//   - Subscribe/Unsubscribe: close(ch)统一替换为safeCloseCWChan(ch)

import (
	"sync"
	"tedna/internal/logger"
)

// ==================== 课件SSE事件类型常量 ====================

const (
	// ---- 索引生成阶段事件（Phase 3） ----
	CWSSEConnected     = "connected"      // SSE连接建立
	CWSSEIndexStart    = "index_start"    // 开始生成索引
	CWSSEIndexPage     = "index_page"     // 单页索引生成完成
	CWSSEIndexProgress = "index_progress" // 索引生成进度
	CWSSEIndexDone     = "index_done"     // 索引生成全部完成

	// ---- 课件HTML生成阶段事件（Phase 4B） ----
	CWSSEGenStart    = "gen_start"    // 开始生成课件HTML
	CWSSEGenPage     = "gen_page"     // 单页HTML生成完成
	CWSSEGenProgress = "gen_progress" // 生成进度更新
	CWSSEGenDone     = "gen_done"     // 全部页面HTML生成完成

	// ---- 通用事件 ----
	CWSSEChunk = "chunk" // AI流式输出片段
	CWSSEError = "error" // 错误
)

// ==================== 课件SSE事件结构 ====================

// CWSSEEvent 课件工坊SSE事件
type CWSSEEvent struct {
	EventType string      `json:"event_type"` // 事件类型
	Data      interface{} `json:"data"`       // 事件数据（根据类型不同而不同）
}

// ==================== 防御性辅助函数 ====================

var cwSseLog = logger.WithModule("cw_sse")

// safeCloseCWChan 安全关闭课件SSE channel
// 使用 recover 防止 double-close panic（防御性编程，正常路径不应触发）
func safeCloseCWChan(ch chan CWSSEEvent) {
	defer func() {
		if r := recover(); r != nil {
			cwSseLog.Warn("课件SSE channel double-close被捕获(已安全忽略)", "recover", r)
		}
	}()
	close(ch)
}

// safeSendCWEvent 安全发送事件到课件SSE channel
// 非阻塞发送，满则丢弃；使用 recover 防止 send-on-closed panic
// 返回值: true=发送成功, false=丢弃或channel已关闭
func safeSendCWEvent(ch chan CWSSEEvent, event CWSSEEvent) bool {
	defer func() {
		if r := recover(); r != nil {
			cwSseLog.Warn("课件SSE send-on-closed被捕获(已安全忽略)",
				"event_type", event.EventType, "recover", r)
		}
	}()
	select {
	case ch <- event:
		return true
	default:
		return false
	}
}

// ==================== 课件SSE广播中心 ====================

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
				safeCloseCWChan(ch) // v142: 安全关闭防止double-close
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
		safeCloseCWChan(ch) // v142: 安全关闭防止double-close
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
		// v142: 使用safeSend防御send-on-closed
		if safeSendCWEvent(ch, event) {
			sent++
		} else {
			dropped++
			cwSseLog.Warn("课件SSE channel已满或已关闭，事件被丢弃",
				"courseware_id", coursewareID,
				"event_type", event.EventType,
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
