package services

// ==================== P5-2 并发引擎 + AI限流 ====================
// Phase8修复E-02：新增优雅关闭机制
//   原版：SIGTERM时队列中待执行任务直接丢失，Pipeline状态停留在running
//   修复后：
//     1. Engine持有context，Stop()方法关闭taskChan触发所有worker退出
//     2. routes.Setup监听os.Signal(SIGTERM/SIGINT)，收到信号后调用engine.Stop()
//     3. engine.Wait()等待所有worker完成当前任务后再退出
//     4. 队列中尚未取出的任务无法被执行，但正在执行的任务可以正常完成

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ==================== 常量配置 ====================

const (
	DefaultMaxWorkers       = 3
	DefaultMaxAIConcurrency = 2
	DefaultQueueSize        = 50
	// GracefulShutdownTimeout 优雅关闭等待超时（E-02新增）
	// 超过此时间强制退出，防止任务卡死导致进程无法退出
	GracefulShutdownTimeout = 30 * time.Second
)

// ==================== 任务类型定义 ====================

type TaskType string

const (
	TaskTypePipeline TaskType = "pipeline"
	TaskTypeRetrial  TaskType = "retrial"
	TaskTypeVerify   TaskType = "verify"
)

// EngineTask 引擎任务
type EngineTask struct {
	Type       TaskType
	PipelineID string
	ExecFunc   func()
}

// ==================== Engine 并发引擎 ====================

// Engine 并发执行引擎
// Phase8修复E-02：新增 ctx/cancel/stopOnce 支持优雅关闭
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
	// E-02优雅关闭相关字段
	ctx      context.Context    // 引擎生命周期context
	cancel   context.CancelFunc // 触发关闭
	stopOnce sync.Once          // 防止多次调用Stop()
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

	fmt.Printf("[并发引擎] 已启动: %d个Worker, AI并发上限%d, 队列容量%d\n",
		e.maxWorkers, e.maxAI, e.queueSize)
}

// worker 单个Worker的执行循环
func (e *Engine) worker(workerID int) {
	defer e.wg.Done()

	for task := range e.taskChan {
		e.stats.mu.Lock()
		e.stats.CurrentRunning++
		e.stats.mu.Unlock()

		startTime := time.Now()
		fmt.Printf("[Worker-%d] 开始执行: type=%s, pipeline=%s\n",
			workerID, task.Type, task.PipelineID)

		func() {
			defer func() {
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

		e.stats.mu.Lock()
		e.stats.CurrentRunning--
		e.stats.TotalCompleted++
		e.stats.mu.Unlock()

		fmt.Printf("[Worker-%d] 执行完成: type=%s, pipeline=%s, 耗时=%s\n",
			workerID, task.Type, task.PipelineID, elapsed.Round(time.Second))
	}

	fmt.Printf("[Worker-%d] 已退出（引擎关闭）\n", workerID)
}

// ==================== 任务提交 ====================

// Submit 提交任务到执行队列（非阻塞）
// E-02：引擎关闭后拒绝新任务提交
func (e *Engine) Submit(task *EngineTask) bool {
	// 引擎关闭后不接受新任务
	select {
	case <-e.ctx.Done():
		fmt.Printf("[并发引擎] 引擎已关闭，拒绝新任务: type=%s, pipeline=%s\n",
			task.Type, task.PipelineID)
		return false
	default:
	}

	select {
	case e.taskChan <- task:
		e.stats.mu.Lock()
		e.stats.TotalSubmitted++
		e.stats.mu.Unlock()

		fmt.Printf("[并发引擎] 任务已提交: type=%s, pipeline=%s, 队列长度=%d/%d\n",
			task.Type, task.PipelineID, len(e.taskChan), e.queueSize)
		return true
	default:
		fmt.Printf("[并发引擎] 队列已满，任务被拒绝: type=%s, pipeline=%s\n",
			task.Type, task.PipelineID)
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

// ==================== E-02：优雅关闭 ====================

// Stop 触发引擎优雅关闭
// 调用后：
//   1. 不再接受新任务（Submit返回false）
//   2. 关闭taskChan，所有worker完成当前任务后退出range循环
//   3. 队列中尚未被worker取出的任务会被丢弃（无法避免，但已运行中的任务可完成）
//
// 注意：Stop()是幂等的，多次调用安全
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		fmt.Printf("[并发引擎] 开始优雅关闭，等待 %d 个正在执行的任务完成...\n",
			e.stats.CurrentRunning)
		e.cancel()         // 取消context，通知不再接受新任务
		close(e.taskChan)  // 关闭channel，worker完成当前任务后退出for range循环
	})
}

// Wait 等待所有Worker完成当前任务（带超时保护）
// 应在Stop()之后调用，用于优雅关闭流程
func (e *Engine) Wait() {
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Printf("[并发引擎] 所有Worker已退出，引擎关闭完成\n")
	case <-time.After(GracefulShutdownTimeout):
		fmt.Printf("[并发引擎] 等待超时（%s），强制退出\n", GracefulShutdownTimeout)
	}
}

// StartGracefulShutdown 在独立goroutine中监听系统信号并触发优雅关闭
// Phase8修复E-02：在routes.Setup中调用此方法，监听SIGTERM/SIGINT
// 收到信号后：Stop()→Wait()→进程退出
// 此方法会阻塞调用goroutine直到收到信号，应在独立goroutine中运行
func (e *Engine) StartGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		fmt.Printf("[并发引擎] 收到系统信号 %s，开始优雅关闭...\n", sig)
		e.Stop()
		e.Wait()
		fmt.Printf("[并发引擎] 优雅关闭完成，进程退出\n")
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
