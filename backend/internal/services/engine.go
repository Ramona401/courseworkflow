package services

// 并发执行引擎：Pipeline执行队列 + AI信号量限流 + 统一优雅关闭
//
// 运行职责：
//   - 接收Pipeline、重试和验收任务；
//   - 通过固定数量Worker执行任务；
//   - 通过AI信号量限制Pipeline内部AI调用并发；
//   - 维护提交、运行、成功、业务失败和panic统计。
//
// 关闭职责：
//   - Engine不再自行监听SIGTERM/SIGINT；
//   - Engine不再调用os.Exit，避免跳过main及其他模块的defer；
//   - main.go是全系统唯一的信号入口；
//   - main收到关闭信号后调用ShutdownDefaultEngine；
//   - Stop拒绝新任务并关闭队列入口；
//   - Worker继续排空已进入队列及正在执行的任务；
//   - WaitContext等待全部Worker退出或由调用方统一控制超时。
//
// 并发安全修复：
//   旧Submit先检查ctx、随后向taskChan发送，Stop可能恰好在两步之间关闭channel，
//   存在send on closed channel竞态。当前Submit与Stop共用Engine.mu串行保护
//   running状态和channel关闭，保证不会向已关闭队列发送。

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tedna/internal/logger"
)

// ==================== 常量配置 ====================

const (
	DefaultMaxWorkers       = 3
	DefaultMaxAIConcurrency = 2
	DefaultQueueSize        = 50

	// GracefulShutdownTimeout 供保留的Wait()兼容方法使用。
	//
	// 生产进程由main.go传入统一的12分钟上下文调用WaitContext；
	// 该常量保持相同口径，供其他旧调用方在没有外部context时安全等待。
	GracefulShutdownTimeout = 12 * time.Minute
)

// 模块日志
var engineLog = logger.WithModule("engine")

// ==================== 默认生产Engine登记 ====================

// defaultEngine 是routes.Setup创建的生产Engine。
//
// 当前生产进程只创建一个Pipeline Engine。通过包级登记，让main.go无需改变
// routes.Setup返回签名，也能在关闭阶段取得该Engine并等待任务排空。
var (
	defaultEngineMu sync.RWMutex
	defaultEngine   *Engine
)

// registerDefaultEngine 登记生产默认Engine。
// NewEngine每次创建后调用；生产环境只有一个实例，测试环境以后创建的实例覆盖之前实例。
func registerDefaultEngine(engine *Engine) {
	defaultEngineMu.Lock()
	defaultEngine = engine
	defaultEngineMu.Unlock()
}

// ShutdownDefaultEngine 停止并等待默认生产Engine。
//
// ctx由main.go统一创建，HTTP Shutdown与Engine排空共享同一个总期限。
// 未登记Engine时视为无需关闭，直接返回nil。
func ShutdownDefaultEngine(ctx context.Context) error {
	defaultEngineMu.RLock()
	engine := defaultEngine
	defaultEngineMu.RUnlock()

	if engine == nil {
		engineLog.Info("未登记默认Engine，跳过Engine排空")
		return nil
	}

	engine.Stop()
	return engine.WaitContext(ctx)
}

// ==================== 任务类型定义 ====================

// TaskType 任务类型枚举。
type TaskType string

const (
	TaskTypePipeline TaskType = "pipeline"
	TaskTypeRetrial  TaskType = "retrial"
	TaskTypeVerify   TaskType = "verify"
)

// EngineTask 引擎任务。
//
// ExecFunc返回值语义：
//   - nil：业务成功；
//   - error：业务失败；
//   - panic：系统故障，由Worker recover捕获。
type EngineTask struct {
	Type       TaskType
	PipelineID string
	ExecFunc   func() error
}

// ==================== Engine 并发引擎 ====================

// Engine 并发执行引擎。
type Engine struct {
	taskChan    chan *EngineTask
	aiSemaphore chan struct{}
	maxWorkers  int
	maxAI       int
	queueSize   int

	wg          sync.WaitGroup
	workersDone chan struct{}

	running  bool
	mu       sync.Mutex
	stats    *EngineStats
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once
}

// EngineStats 引擎运行统计。
type EngineStats struct {
	mu                  sync.Mutex
	TotalSubmitted      int64 `json:"total_submitted"`
	TotalCompleted      int64 `json:"total_completed"`
	TotalBusinessFailed int64 `json:"total_business_failed"`
	TotalFailed         int64 `json:"total_failed"`
	CurrentRunning      int64 `json:"current_running"`
	CurrentAIActive     int64 `json:"current_ai_active"`
	QueueLength         int   `json:"queue_length"`
}

// NewEngine 创建并启动并发引擎。
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

	engine := &Engine{
		taskChan:    make(chan *EngineTask, queueSize),
		aiSemaphore: make(chan struct{}, maxAIConcurrency),
		maxWorkers:  maxWorkers,
		maxAI:       maxAIConcurrency,
		queueSize:   queueSize,
		workersDone: make(chan struct{}),
		stats:       &EngineStats{},
		ctx:         ctx,
		cancel:      cancel,
	}

	engine.start()

	// start已经完成全部wg.Add，此时才启动等待协程，避免Wait与Add并发误用。
	go func() {
		engine.wg.Wait()
		close(engine.workersDone)
	}()

	registerDefaultEngine(engine)
	return engine
}

// start 启动所有Worker。
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

	engineLog.Info("并发引擎已启动",
		"max_workers", e.maxWorkers,
		"max_ai_concurrency", e.maxAI,
		"queue_capacity", e.queueSize,
	)
}

// worker 单个Worker执行循环。
//
// taskChan关闭后，range仍会继续读取并处理channel缓冲区中已经接收的全部任务；
// 只有队列真正排空后Worker才退出，因此Stop不会丢弃已提交任务。
func (e *Engine) worker(workerID int) {
	defer e.wg.Done()

	for task := range e.taskChan {
		e.stats.mu.Lock()
		e.stats.CurrentRunning++
		e.stats.mu.Unlock()

		startTime := time.Now()

		engineLog.Debug("Worker开始执行任务",
			"worker_id", workerID,
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
		)

		var taskErr error
		var panicked bool

		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicked = true

					engineLog.Error("Worker任务发生panic",
						"worker_id", workerID,
						"task_type", string(task.Type),
						"pipeline_id", task.PipelineID,
						"panic_value", fmt.Sprintf("%v", recovered),
					)

					e.stats.mu.Lock()
					e.stats.TotalFailed++
					e.stats.mu.Unlock()
				}
			}()

			taskErr = task.ExecFunc()
		}()

		elapsed := time.Since(startTime)

		e.stats.mu.Lock()
		e.stats.CurrentRunning--
		if !panicked {
			if taskErr != nil {
				e.stats.TotalBusinessFailed++
			} else {
				e.stats.TotalCompleted++
			}
		}
		e.stats.mu.Unlock()

		if panicked {
			// panic已在recover中记录。
		} else if taskErr != nil {
			engineLog.Warn("Worker任务业务失败",
				"worker_id", workerID,
				"task_type", string(task.Type),
				"pipeline_id", task.PipelineID,
				"elapsed_ms", elapsed.Milliseconds(),
				"error", taskErr.Error(),
			)
		} else {
			engineLog.Debug("Worker任务完成",
				"worker_id", workerID,
				"task_type", string(task.Type),
				"pipeline_id", task.PipelineID,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		}
	}

	engineLog.Info("Worker已退出",
		"worker_id", workerID,
		"reason", "任务队列已关闭并排空",
	)
}

// ==================== 任务提交 ====================

// Submit 非阻塞提交任务。
//
// running检查和channel发送由同一把mu保护，Stop关闭channel时也持有该锁，
// 因此不会出现检查时仍运行、发送前channel却被关闭的竞态。
func (e *Engine) Submit(task *EngineTask) bool {
	if task == nil || task.ExecFunc == nil {
		engineLog.Warn("拒绝无效任务")
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		engineLog.Warn("引擎正在关闭，拒绝新任务",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
		)
		return false
	}

	select {
	case e.taskChan <- task:
		e.stats.mu.Lock()
		e.stats.TotalSubmitted++
		e.stats.mu.Unlock()

		engineLog.Debug("任务已提交到队列",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
			"queue_length", len(e.taskChan),
			"queue_capacity", e.queueSize,
		)
		return true

	default:
		engineLog.Warn("任务队列已满，任务被拒绝",
			"task_type", string(task.Type),
			"pipeline_id", task.PipelineID,
			"queue_capacity", e.queueSize,
		)
		return false
	}
}

// ==================== AI信号量控制 ====================

// AcquireAI 获取AI调用信号量。
func (e *Engine) AcquireAI() {
	e.aiSemaphore <- struct{}{}

	e.stats.mu.Lock()
	e.stats.CurrentAIActive++
	e.stats.mu.Unlock()
}

// ReleaseAI 释放AI调用信号量。
func (e *Engine) ReleaseAI() {
	<-e.aiSemaphore

	e.stats.mu.Lock()
	e.stats.CurrentAIActive--
	e.stats.mu.Unlock()
}

// ==================== 优雅关闭 ====================

// Stop 触发Engine关闭，幂等。
//
// 关闭动作：
//   1. running=false，使后续Submit立即被拒绝；
//   2. cancel内部context，为未来需要感知关闭的功能保留信号；
//   3. close(taskChan)，不再接收新任务；
//   4. 已在队列中的任务仍由Worker继续处理直至排空。
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		e.stats.mu.Lock()
		currentRunning := e.stats.CurrentRunning
		currentAIActive := e.stats.CurrentAIActive
		queueLength := len(e.taskChan)
		e.stats.mu.Unlock()

		engineLog.Info("开始优雅关闭Engine",
			"current_running_tasks", currentRunning,
			"current_ai_active", currentAIActive,
			"queued_tasks", queueLength,
		)

		e.mu.Lock()
		e.running = false
		e.cancel()
		close(e.taskChan)
		e.mu.Unlock()
	})
}

// WaitContext 等待全部Worker完成当前和已排队任务。
//
// 不自行创建超时，不结束进程；超时策略由main统一控制。
func (e *Engine) WaitContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-e.workersDone:
		engineLog.Info("所有Worker已退出，Engine关闭完成")
		return nil

	case <-ctx.Done():
		engineLog.Warn("Engine排空等待超时",
			"error", ctx.Err(),
		)
		return ctx.Err()
	}
}

// Wait 保留旧调用兼容，内部使用默认12分钟超时。
func (e *Engine) Wait() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		GracefulShutdownTimeout,
	)
	defer cancel()

	_ = e.WaitContext(ctx)
}

// StartGracefulShutdown 保留旧接线兼容。
//
// routes.Setup目前仍调用本方法。为避免本批完整覆盖超大routes.go，本方法改为明确的
// 兼容空操作：不注册signal.Notify、不启动goroutine、不调用os.Exit。
// 真正的系统信号处理和Engine关闭由main.go统一负责。
func (e *Engine) StartGracefulShutdown() {
	engineLog.Info("Engine关闭信号已交由main统一管理")
}

// ==================== 状态查询 ====================

// GetStats 获取Engine运行统计。
func (e *Engine) GetStats() EngineStats {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	return EngineStats{
		TotalSubmitted:      e.stats.TotalSubmitted,
		TotalCompleted:      e.stats.TotalCompleted,
		TotalBusinessFailed: e.stats.TotalBusinessFailed,
		TotalFailed:         e.stats.TotalFailed,
		CurrentRunning:      e.stats.CurrentRunning,
		CurrentAIActive:     e.stats.CurrentAIActive,
		QueueLength:         len(e.taskChan),
	}
}

// IsQueueFull 检查任务队列是否已满。
func (e *Engine) IsQueueFull() bool {
	return len(e.taskChan) >= e.queueSize
}

// GetQueueLength 获取等待中的任务数。
func (e *Engine) GetQueueLength() int {
	return len(e.taskChan)
}
