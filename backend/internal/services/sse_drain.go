package services

// sse_drain.go — 四类SSE连接的进程级排空控制
//
// 管理范围：
//   - 教案生成SSE：GlobalLPSSEHub；
//   - 课件工坊SSE：GlobalCWSSEHub；
//   - Pipeline进度SSE：GlobalSSEHub；
//   - 知识库压缩SSE：GlobalKBSSEHub。
//
// 排空顺序：
//   1. 先把全局draining原子状态设置为true；
//   2. 再逐个锁住Hub并关闭全部已登记channel；
//   3. 新Subscribe在Hub锁内检查draining，不能越过排空重新登记；
//   4. Handler读取关闭channel后自然返回，释放长期HTTP请求。
//
// 本状态是单向的：生产进程进入draining后不会恢复running。
// 新版本由systemd启动一个全新进程，全局状态自然回到false。

import (
	"sync"
	"sync/atomic"

	"tedna/internal/logger"
)

// SSEDrainSummary 是四类SSE连接的排空快照。
type SSEDrainSummary struct {
	Draining          bool `json:"draining"`
	LessonPlan        int  `json:"lesson_plan"`
	Courseware        int  `json:"courseware"`
	Pipeline          int  `json:"pipeline"`
	KnowledgeBase     int  `json:"knowledge_base"`
	Total             int  `json:"total"`
	FirstTransition   bool `json:"first_transition"`
	ConnectionsClosed int  `json:"connections_closed"`
}

var (
	globalSSEHandshakeMu sync.RWMutex
	globalSSEDraining    atomic.Bool
	sseDrainLog          = logger.WithModule("sse_drain")
)

// IsGlobalSSEDraining 返回当前进程是否已经进入SSE排空状态。

func IsGlobalSSEDraining() bool {
	return globalSSEDraining.Load()
}

// BeginGlobalSSEHandshake 尝试进入一次短时SSE连接握手。
//
// 返回成功后，调用方应在完成以下步骤后尽快调用finish：
//   - 检查http.Flusher；
//   - 设置SSE响应头；
//   - 向对应Hub登记订阅。
//
// finish可以重复调用，内部通过sync.Once保证只释放一次读锁。
// BeginGlobalSSEDraining需要取得写锁，因此不会与尚未完成的订阅握手交叉。
func BeginGlobalSSEHandshake() (finish func(), accepted bool) {
	globalSSEHandshakeMu.RLock()

	if globalSSEDraining.Load() {
		globalSSEHandshakeMu.RUnlock()
		return nil, false
	}

	var once sync.Once

	return func() {
		once.Do(globalSSEHandshakeMu.RUnlock)
	}, true
}

// BeginGlobalSSEDraining 进入SSE排空状态并关闭全部现有连接。
//
// 重复调用安全：后续调用仍会检查并关闭极端竞态下残留的连接，
// 但FirstTransition只会在第一次由false切换到true时返回true。
func BeginGlobalSSEDraining() SSEDrainSummary {
	globalSSEHandshakeMu.Lock()
	defer globalSSEHandshakeMu.Unlock()
	firstTransition := globalSSEDraining.CompareAndSwap(false, true)

	lessonPlanClosed := GlobalLPSSEHub.CloseAll()
	coursewareClosed := GlobalCWSSEHub.CloseAll()
	pipelineClosed := GlobalSSEHub.CloseAll()
	knowledgeBaseClosed := GlobalKBSSEHub.CloseAll()

	totalClosed := lessonPlanClosed +
		coursewareClosed +
		pipelineClosed +
		knowledgeBaseClosed

	summary := SSEDrainSummary{
		Draining:          true,
		LessonPlan:        lessonPlanClosed,
		Courseware:        coursewareClosed,
		Pipeline:          pipelineClosed,
		KnowledgeBase:     knowledgeBaseClosed,
		Total:             totalClosed,
		FirstTransition:   firstTransition,
		ConnectionsClosed: totalClosed,
	}

	sseDrainLog.Info("SSE连接排空完成",
		"first_transition", firstTransition,
		"lesson_plan_closed", lessonPlanClosed,
		"courseware_closed", coursewareClosed,
		"pipeline_closed", pipelineClosed,
		"knowledge_base_closed", knowledgeBaseClosed,
		"total_closed", totalClosed,
	)

	return summary
}

// GetGlobalSSEDrainSummary 返回当前连接数量，不关闭连接。
func GetGlobalSSEDrainSummary() SSEDrainSummary {
	lessonPlan := GlobalLPSSEHub.GetTotalSubscribers()
	courseware := GlobalCWSSEHub.GetTotalSubscribers()
	pipeline := GlobalSSEHub.GetTotalSubscribers()
	knowledgeBase := GlobalKBSSEHub.GetTotalSubscribers()

	total := lessonPlan + courseware + pipeline + knowledgeBase

	return SSEDrainSummary{
		Draining:      IsGlobalSSEDraining(),
		LessonPlan:    lessonPlan,
		Courseware:    courseware,
		Pipeline:      pipeline,
		KnowledgeBase: knowledgeBase,
		Total:         total,
	}
}

// CloseAll 关闭全部教案SSE连接，返回实际关闭数量。
func (h *LPSSEHub) CloseAll() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	closed := 0

	for planID, subs := range h.subscribers {
		for ch, active := range subs {
			if !active {
				continue
			}

			subs[ch] = false
			safeCloseLPChan(ch)
			closed++
		}

		delete(h.subscribers, planID)
	}

	return closed
}

// GetTotalSubscribers 返回全部活动教案SSE连接数。
func (h *LPSSEHub) GetTotalSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0

	for _, subs := range h.subscribers {
		for _, active := range subs {
			if active {
				total++
			}
		}
	}

	return total
}

// CloseAll 关闭全部课件SSE连接，返回实际关闭数量。
func (h *CWSSEHub) CloseAll() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	closed := 0

	for coursewareID, subs := range h.subscribers {
		for ch, active := range subs {
			if !active {
				continue
			}

			subs[ch] = false
			safeCloseCWChan(ch)
			closed++
		}

		delete(h.subscribers, coursewareID)
	}

	return closed
}

// GetTotalSubscribers 返回全部活动课件SSE连接数。
func (h *CWSSEHub) GetTotalSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0

	for _, subs := range h.subscribers {
		for _, active := range subs {
			if active {
				total++
			}
		}
	}

	return total
}

// CloseAll 关闭全部Pipeline SSE连接，返回实际关闭数量。
func (h *SSEHub) CloseAll() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	closed := 0

	for pipelineID, subs := range h.subscribers {
		for ch, active := range subs {
			if !active {
				continue
			}

			subs[ch] = false
			close(ch)
			closed++
		}

		delete(h.subscribers, pipelineID)
	}

	return closed
}

// CloseAll 关闭全部知识库SSE连接，返回实际关闭数量。
func (h *KBSSEHub) CloseAll() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	closed := 0

	for jobID, subs := range h.subscribers {
		for ch, active := range subs {
			if !active {
				continue
			}

			subs[ch] = false
			safeCloseKBChan(ch)
			closed++
		}

		delete(h.subscribers, jobID)
	}

	return closed
}

// GetTotalSubscribers 返回全部活动知识库SSE连接数。
func (h *KBSSEHub) GetTotalSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0

	for _, subs := range h.subscribers {
		for _, active := range subs {
			if active {
				total++
			}
		}
	}

	return total
}
