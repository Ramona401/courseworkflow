package services

// 并发执行引擎：Pipeline执行队列 + AI信号量限流 + 优雅关闭
// Phase8日志升级：
//   - 引擎启动/关闭 → INFO
//   - Worker任务开始/完成 → DEBUG（高频，生产环境默认不输出，不干扰主日志）
//   - Worker panic → ERROR（需要立即关注）
//   - 任务提交成功 → DEBUG（高频）
//   - 队列已满/引擎已关闭 → WARN（需要关注的异常情况）
//   - 优雅关闭信号 → INFO

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tedna/internal/logger"
)

// ==================== 常量配置 ====================

const (
	DefaultMaxWorkers       = 3
	DefaultMaxAIConcurrency = 2
	DefaultQueueSize        = 50
	// GracefulShutdownTimeout 优雅关闭等待超时
	GracefulShutdownTimeout = 30 * time.Second
)

// 模块日志
var engineLog = logger.WithModule("engine")

// ==================== 任务类型定义 ====================

// TaskType 任务类型枚举
type TaskType string

const (
	TaskTypePipeline TaskType = "pipeline"
	TaskTypeRetrial  TaskType = "retrial"
	TaskTypeVerify   TaskType = "verify"
)

// EngineTask 引擎任务（投递到队列的工作单元）
type EngineTask struct {
	Type       TaskType // 任务类型
	PipelineID string   // Pipeline ID
	ExecFunc   func()   // 实际执行函数（闭包，由调用方构造）
}

// ==================== Engine 并发引擎 ====================

// Engine 并发执行引擎
type Engine struct {
	taskChan    chan *EngineTask
	aiSemaphore chan struct{}
	maxWorkers  int
	maxAI       int
	queueSize   int
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
	stats       *EngineStats
	ctx         context.Context
	cancel      context.CancelFunc
	stopOnce    sync.Once
}

// EngineStats 引擎运行统计
type EngineStats struct {
	mu              sync.Mutex
	TotalSubmitted  int64 `json:"total_submitted"`
	TotalCompleted  int64 `json:"total_completed"`
	TotalFailed     int64 `json:"total_failed"`
	CurrentRunning  int64 `json:"current_running"`
	CurrentAIActive int64 `json:"current_ai_active"`
	QueueLength     int   `json:"queue_length"`
}

// NewEngine 创建并启动并发引擎
func NewEngine(maxWorkers, maxAIConcurrency, queueSize int) *Engine {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}
	if maxAIConcurrency <= 0 {
		maxAIConcurrency = DefaultMaxAIConcurrency
	}
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	e := &Engine{
		taskChan:    make(chan *EngineTask, queueSize),
		aiSemaphore: make(chan struct{}, maxAIConcurrency),
		maxWorkers:  maxWorkers,
		maxAI:       maxAIConcurrency,
		queueSize:   queueSize,
		stats:       &EngineStats{},
		ctx:         ctx,
		cancel:      cancel,
	}

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

	// INFO：引擎启动是重要事件，记录配置参数
	engineLog.Info("并发引擎已启动",
		"max_workers", e.maxWorkers,
		"max_ai_concurrency", e.maxAI,
		"queue_capacity", e.queueSize,
	)
}

// worker 单个Worker的执行循环
func (e *Engine) worker(workerID int) {
	defer e.wg.Done()

	for task := range e.taskChan {
		e.stats.mu.Lock()
		e.stats.CurrentRunning++
		e.stats.mu.Unlock()

		startTime := time.Now()

		// DEBUG：任务开始（高频事件，生产环境通常不需要）
		engineLog.Debug("Worker开始执行任务",
			"worker_id", workerID,
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
		)

		func() {
			defer func() {
				if r := recover(); r != nil {
					// ERROR：panic需要立即关注，包含完整上下文
					engineLog.Error("Worker任务发生panic",
						"worker_id", workerID,
						"task_type", string(task.Type),
						"pipeline_id", task.PipelineID,
						"panic_value", fmt.Sprintf("%v", r),
					)
					e.stats.mu.Lock()
					e.stats.TotalFailed++
					e.stats.mu.Unlock()
				}
			}()
			task.ExecFunc()
		}()

		elapsed := time.Since(startTime)

		e.stats.mu.Lock()
		e.stats.CurrentRunning--
		e.stats.TotalCompleted++
		e.stats.mu.Unlock()

		// DEBUG：任务完成（高频事件）
		engineLog.Debug("Worker任务完成",
			"worker_id", workerID,
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
			"elapsed_ms", elapsed.Milliseconds(),
		)
	}

	// INFO：Worker退出（通常只在引擎关闭时发生）
	engineLog.Info("Worker已退出",
		"worker_id", workerID,
		"reason", "引擎关闭",
	)
}

// ==================== 任务提交 ====================

// Submit 提交任务到执行队列（非阻塞）
func (e *Engine) Submit(task *EngineTask) bool {
	// 引擎关闭后不接受新任务
	select {
	case <-e.ctx.Done():
		engineLog.Warn("引擎已关闭，拒绝新任务",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
		)
		return false
	default:
	}

	select {
	case e.taskChan <- task:
		e.stats.mu.Lock()
		e.stats.TotalSubmitted++
		e.stats.mu.Unlock()

		// DEBUG：任务提交成功（高频事件）
		engineLog.Debug("任务已提交到队列",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
			"queue_length", len(e.taskChan),
			"queue_capacity", e.queueSize,
		)
		return true
	default:
		// WARN：队列满是需要关注的异常情况
		engineLog.Warn("任务队列已满，任务被拒绝",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
			"queue_capacity", e.queueSize,
		)
		return false
	}
}

// ==================== AI信号量控制 ====================

// AcquireAI 获取AI调用信号量（阻塞直到获取成功）
func (e *Engine) AcquireAI() {
	e.aiSemaphore <- struct{}{}

	e.stats.mu.Lock()
	e.stats.CurrentAIActive++
	e.stats.mu.Unlock()
}

// ReleaseAI 释放AI调用信号量
func (e *Engine) ReleaseAI() {
	<-e.aiSemaphore

	e.stats.mu.Lock()
	e.stats.CurrentAIActive--
	e.stats.mu.Unlock()
}

// ==================== 优雅关闭 ====================

// Stop 触发引擎优雅关闭（幂等）
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		e.stats.mu.Lock()
		currentRunning := e.stats.CurrentRunning
		e.stats.mu.Unlock()

		// INFO：关闭是重要生命周期事件
		engineLog.Info("开始优雅关闭引擎",
			"current_running_tasks", currentRunning,
		)
		e.cancel()
		close(e.taskChan)
	})
}

// Wait 等待所有Worker完成当前任务（带超时保护）
func (e *Engine) Wait() {
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		engineLog.Info("所有Worker已退出，引擎关闭完成")
	case <-time.After(GracefulShutdownTimeout):
		engineLog.Warn("优雅关闭等待超时，强制退出",
			"timeout", GracefulShutdownTimeout.String(),
		)
	}
}

// StartGracefulShutdown 监听系统信号，收到后执行优雅关闭
func (e *Engine) StartGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		// INFO：收到系统信号是重要事件
		engineLog.Info("收到系统信号，开始优雅关闭",
			"signal", sig.String(),
		)
		e.Stop()
		e.Wait()
		engineLog.Info("优雅关闭完成，进程退出")
		os.Exit(0)
	}()
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
