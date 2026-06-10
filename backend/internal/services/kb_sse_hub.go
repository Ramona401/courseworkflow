package services

// kb_sse_hub.go — 知识库压缩进度 SSE 广播中枢（按 job_id 维度）
//
// 照搬 courseware_sse_hub.go 的成熟范式：
//   - 全局单例 GlobalKBSSEHub
//   - subscribers map[jobID]map[chan]bool，独占订阅（新连接关旧）
//   - safeClose/safeSend 双 recover 防护（防 double-close 与 send-on-closed panic）
//   - Broadcast 非阻塞发送，满则丢弃
//   - 缓冲 500：一个任务抽取出几十个知识点，每个 item 压缩推几条进度，留足余量

import (
	"sync"

	"tedna/internal/logger"
)

// ==================== KB 压缩 SSE 事件类型常量 ====================

const (
	KBSSEConnected    = "connected"     // SSE 连接建立
	KBSSEExtractStart = "extract_start" // 开始抽取知识点
	KBSSEExtractDone  = "extract_done"  // 抽取完成（报告共 N 个待压缩单元）
	KBSSEItemStart    = "item_start"    // 单个单元开始压缩
	KBSSEItemDone     = "item_done"     // 单个单元压缩+仲裁完成
	KBSSEProgress     = "progress"      // 通用进度文字
	KBSSEJobDone      = "job_done"      // 整个任务全部完成
	KBSSEError        = "error"         // 错误
)

// ==================== KB SSE 事件结构 ====================

// KBSSEEvent 知识库压缩 SSE 事件
type KBSSEEvent struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

// ==================== 防御性辅助函数 ====================

var kbSseLog = logger.WithModule("kb_sse")

// safeCloseKBChan 安全关闭 KB SSE channel（recover 防 double-close panic）
func safeCloseKBChan(ch chan KBSSEEvent) {
	defer func() {
		if r := recover(); r != nil {
			kbSseLog.Warn("KB SSE channel double-close被捕获(已安全忽略)", "recover", r)
		}
	}()
	close(ch)
}

// safeSendKBEvent 安全发送事件到 KB SSE channel（非阻塞，满则丢弃，recover 防 send-on-closed）
func safeSendKBEvent(ch chan KBSSEEvent, event KBSSEEvent) bool {
	defer func() {
		if r := recover(); r != nil {
			kbSseLog.Warn("KB SSE send-on-closed被捕获(已安全忽略)",
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

// ==================== KB SSE 广播中心 ====================

// KBSSEHub 知识库压缩 SSE 广播中心（全局单例）
type KBSSEHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan KBSSEEvent]bool // jobID → channels
}

// GlobalKBSSEHub 全局 KB 压缩 SSE 广播中心
var GlobalKBSSEHub = NewKBSSEHub()

// NewKBSSEHub 创建 KB SSE 广播中心
func NewKBSSEHub() *KBSSEHub {
	return &KBSSEHub{
		subscribers: make(map[string]map[chan KBSSEEvent]bool),
	}
}

// Subscribe 订阅指定任务的 SSE 事件（独占模式：新连接前关闭该 jobID 所有旧 channel）
func (h *KBSSEHub) Subscribe(jobID string) chan KBSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	if oldSubs, exists := h.subscribers[jobID]; exists && len(oldSubs) > 0 {
		for ch, active := range oldSubs {
			if active {
				oldSubs[ch] = false
				safeCloseKBChan(ch)
			}
		}
		delete(h.subscribers, jobID)
	}

	ch := make(chan KBSSEEvent, 500)
	h.subscribers[jobID] = map[chan KBSSEEvent]bool{ch: true}
	return ch
}

// Unsubscribe 取消订阅
func (h *KBSSEHub) Unsubscribe(jobID string, ch chan KBSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[jobID]
	if !ok {
		return
	}
	if active, exists := subs[ch]; exists && active {
		subs[ch] = false
		safeCloseKBChan(ch)
		delete(subs, ch)
	}
	if len(subs) == 0 {
		delete(h.subscribers, jobID)
	}
}

// Broadcast 向指定任务的所有订阅者广播事件（非阻塞，满则丢弃）
func (h *KBSSEHub) Broadcast(jobID string, event KBSSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[jobID]
	if !ok || len(subs) == 0 {
		return
	}
	for ch, active := range subs {
		if !active {
			continue
		}
		if !safeSendKBEvent(ch, event) {
			kbSseLog.Warn("KB SSE channel已满或已关闭，事件被丢弃",
				"job_id", jobID, "event_type", event.EventType)
		}
	}
}
