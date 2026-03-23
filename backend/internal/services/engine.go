package services

// ==================== P5-2 并发引擎 + AI限流 ====================
// 功能：
//   1. Pipeline执行队列：channel缓冲+N个worker goroutine并行消费
//   2. AI信号量：限制同时进行的AI调用数量，防止API限流
//   3. Submit方法：将Pipeline执行任务提交到队列，非阻塞
//   4. 统一管理所有异步执行（StartPipeline、2审流程、批量验收）
//
// 架构：
//   Engine{} <- 在routes.Setup中创建并注入PipelineService
//   +- taskChan (channel) <- Submit()投递任务
//   +- N个worker goroutine <- 从taskChan消费并执行
//   +- aiSemaphore (channel) <- AcquireAI/ReleaseAI 控制AI并发
//
// 使用方式：
//   engine := NewEngine(cfg) -> 启动worker
//   pipelineService.SetEngine(engine) -> 注入引擎
//   engine.Submit(task) -> 投递任务到队列
//   engine.AcquireAI() / engine.ReleaseAI() -> AI调用前后获取/释放信号量

import (
	"fmt"
	"sync"
	"time"
)

// ==================== 常量配置 ====================

const (
	// DefaultMaxWorkers 默认最大Worker数量（同时执行的Pipeline数）
	DefaultMaxWorkers = 3

	// DefaultMaxAIConcurrency 默认最大AI并发数（同时进行的AI调用数）
	// 设为2是因为Opus模型每次调用300-600s，2个并发已经接近API限流边界
	DefaultMaxAIConcurrency = 2

	// DefaultQueueSize 默认任务队列缓冲大小
	// 超过此数量的Submit会返回false（正常情况不会达到）
	DefaultQueueSize = 50
)

// ==================== 任务类型定义 ====================

// TaskType 任务类型枚举
type TaskType string

const (
	// TaskTypePipeline 普通Pipeline执行任务（StartPipeline->全链路）
	TaskTypePipeline TaskType = "pipeline"

	// TaskTypeRetrial 2审流程任务（验收失败->重跑evaluator+meta+translator+generator）
	TaskTypeRetrial TaskType = "retrial"

	// TaskTypeVerify 验收任务（finalized->verify步骤）
	TaskTypeVerify TaskType = "verify"
)

// EngineTask 引擎任务（投递到队列的工作单元）
type EngineTask struct {
	Type       TaskType // 任务类型
	PipelineID string   // Pipeline ID
	ExecFunc   func()   // 实际执行函数（闭包，由调用方构造）
}

// ==================== Engine 并发引擎 ====================

// Engine 并发执行引擎
// 管理Pipeline执行队列和AI调用限流
type Engine struct {
	taskChan    chan *EngineTask // 任务队列（带缓冲的channel）
	aiSemaphore chan struct{}    // AI并发信号量（带缓冲的channel，容量=maxAIConcurrency）
	maxWorkers  int             // 最大Worker数量
	maxAI       int             // 最大AI并发数
	queueSize   int             // 队列缓冲大小
	wg          sync.WaitGroup  // 用于优雅关闭时等待所有worker完成
	running     bool            // 引擎是否已启动
	mu          sync.Mutex      // 保护running状态
	stats       *EngineStats    // 运行统计
}

// EngineStats 引擎运行统计
type EngineStats struct {
	mu              sync.Mutex
	TotalSubmitted  int64 `json:"total_submitted"`   // 总提交任务数
	TotalCompleted  int64 `json:"total_completed"`   // 总完成任务数
	TotalFailed     int64 `json:"total_failed"`      // 总失败任务数
	CurrentRunning  int64 `json:"current_running"`   // 当前正在执行的任务数
	CurrentAIActive int64 `json:"current_ai_active"` // 当前正在进行的AI调用数
	QueueLength     int   `json:"queue_length"`      // 当前队列中等待的任务数
}

// NewEngine 创建并启动并发引擎
// maxWorkers: 最大Worker数量（同时执行的Pipeline数），0则使用默认值
// maxAIConcurrency: 最大AI并发数，0则使用默认值
// queueSize: 任务队列缓冲大小，0则使用默认值
func NewEngine(maxWorkers, maxAIConcurrency, queueSize int) *Engine {
	// 参数校验和默认值
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}
	if maxAIConcurrency <= 0 {
		maxAIConcurrency = DefaultMaxAIConcurrency
	}
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}

	e := &Engine{
		taskChan:    make(chan *EngineTask, queueSize),
		aiSemaphore: make(chan struct{}, maxAIConcurrency),
		maxWorkers:  maxWorkers,
		maxAI:       maxAIConcurrency,
		queueSize:   queueSize,
		stats:       &EngineStats{},
	}

	// 启动Worker goroutine
	e.start()

	return e
}

// start 启动所有Worker goroutine
func (e *Engine) start() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return
	}
	e.running = true

	for i := 0; i < e.maxWorkers; i++ {
		workerID := i + 1
		e.wg.Add(1)
		go e.worker(workerID)
	}

	fmt.Printf("[并发引擎] 已启动: %d个Worker, AI并发上限%d, 队列容量%d\n",
		e.maxWorkers, e.maxAI, e.queueSize)
}

// worker 单个Worker的执行循环
// 从taskChan中持续消费任务并执行
func (e *Engine) worker(workerID int) {
	defer e.wg.Done()

	for task := range e.taskChan {
		// 更新统计：开始执行
		e.stats.mu.Lock()
		e.stats.CurrentRunning++
		e.stats.mu.Unlock()

		startTime := time.Now()
		fmt.Printf("[Worker-%d] 开始执行: type=%s, pipeline=%s\n",
			workerID, task.Type, task.PipelineID)

		// 执行任务（闭包中包含完整的执行逻辑和错误处理）
		func() {
			defer func() {
				// 捕获panic防止worker退出
				if r := recover(); r != nil {
					fmt.Printf("[Worker-%d] 任务panic: type=%s, pipeline=%s, error=%v\n",
						workerID, task.Type, task.PipelineID, r)
					e.stats.mu.Lock()
					e.stats.TotalFailed++
					e.stats.mu.Unlock()
				}
			}()
			task.ExecFunc()
		}()

		elapsed := time.Since(startTime)

		// 更新统计：执行完成
		e.stats.mu.Lock()
		e.stats.CurrentRunning--
		e.stats.TotalCompleted++
		e.stats.mu.Unlock()

		fmt.Printf("[Worker-%d] 执行完成: type=%s, pipeline=%s, 耗时=%s\n",
			workerID, task.Type, task.PipelineID, elapsed.Round(time.Second))
	}
}

// ==================== 任务提交 ====================

// Submit 提交任务到执行队列
// 非阻塞：如果队列未满则立即返回true；队列已满则返回false
func (e *Engine) Submit(task *EngineTask) bool {
	select {
	case e.taskChan <- task:
		// 更新统计
		e.stats.mu.Lock()
		e.stats.TotalSubmitted++
		e.stats.mu.Unlock()

		fmt.Printf("[并发引擎] 任务已提交: type=%s, pipeline=%s, 队列长度=%d/%d\n",
			task.Type, task.PipelineID, len(e.taskChan), e.queueSize)
		return true
	default:
		// 队列已满
		fmt.Printf("[并发引擎] 队列已满，任务被拒绝: type=%s, pipeline=%s\n",
			task.Type, task.PipelineID)
		return false
	}
}

// ==================== AI信号量控制 ====================

// AcquireAI 获取AI调用信号量（阻塞直到获取成功）
// 在每次调用ai.CallAI之前调用，限制同时进行的AI调用数量
func (e *Engine) AcquireAI() {
	e.aiSemaphore <- struct{}{}

	e.stats.mu.Lock()
	e.stats.CurrentAIActive++
	e.stats.mu.Unlock()
}

// ReleaseAI 释放AI调用信号量
// 在每次ai.CallAI调用完成后调用（无论成功失败都必须释放）
func (e *Engine) ReleaseAI() {
	<-e.aiSemaphore

	e.stats.mu.Lock()
	e.stats.CurrentAIActive--
	e.stats.mu.Unlock()
}

// ==================== 状态查询 ====================

// GetStats 获取引擎运行统计（线程安全）
func (e *Engine) GetStats() EngineStats {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	return EngineStats{
		TotalSubmitted:  e.stats.TotalSubmitted,
		TotalCompleted:  e.stats.TotalCompleted,
		TotalFailed:     e.stats.TotalFailed,
		CurrentRunning:  e.stats.CurrentRunning,
		CurrentAIActive: e.stats.CurrentAIActive,
		QueueLength:     len(e.taskChan),
	}
}

// IsQueueFull 检查任务队列是否已满
func (e *Engine) IsQueueFull() bool {
	return len(e.taskChan) >= e.queueSize
}

// GetQueueLength 获取当前队列中等待的任务数
func (e *Engine) GetQueueLength() int {
	return len(e.taskChan)
}

