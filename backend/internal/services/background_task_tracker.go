package services

// background_task_tracker.go — 进程级后台任务登记、排空与防重复控制
//
// 设计目标：
//   1. 解决HTTP已经返回、实际AI任务仍在goroutine中运行，部署时无法被发现的问题；
//   2. 部署前进入draining状态，拒绝新的外部长任务，但允许已有任务完成；
//   3. 为每个任务保存类型、资源ID、启动时间和任务级别，便于运维观察；
//   4. 支持唯一任务键，避免同一课件被连续点击后重复调用AI和重复写库；
//   5. 支持onDrain钩子，让批量生成在排空时停止继续派发新页面；
//   6. 统一后台goroutine的Done和panic恢复，避免单个任务panic终止整个Go进程。
//
// 使用约定：
//   - TryStartExternal：处理用户新请求。进入draining后拒绝启动；
//   - TryStartChild：已有任务产生的必要派生任务。draining期间仍可登记；
//   - task.Run：后台goroutine统一执行包装，自动Done并把panic转换为error；
//   - BeginDraining：部署前调用，设置draining并触发已有任务的onDrain钩子；
//   - Wait：等待全部已登记任务结束，超时由调用方context统一控制。
//
// 注意：
//   Tracker只管理明确登记的任务。新增异步业务时必须先登记，再启动goroutine，
//   不能在goroutine内部才登记，否则部署可能在goroutine真正执行前误判任务为零。

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"tedna/internal/logger"
)

// ==================== 类型与常量 ====================

// BackgroundTaskClass 表示后台任务对业务结果的重要程度。
type BackgroundTaskClass string

const (
	// BackgroundTaskCritical 表示老师主动发起、需要尽量完成并落库的关键任务。
	BackgroundTaskCritical BackgroundTaskClass = "critical"

	// BackgroundTaskBestEffort 表示可重新计算的派生任务，例如对齐、规整和索引回填。
	BackgroundTaskBestEffort BackgroundTaskClass = "best_effort"
)

// BackgroundStartResult 表示任务登记结果。
type BackgroundStartResult string

const (
	// BackgroundStarted 表示任务登记成功，可以启动。
	BackgroundStarted BackgroundStartResult = "started"

	// BackgroundAlreadyRunning 表示同一唯一任务键已在运行，不应重复启动。
	BackgroundAlreadyRunning BackgroundStartResult = "already_running"

	// BackgroundRejectedDraining 表示服务正在排空，新的外部任务被拒绝。
	BackgroundRejectedDraining BackgroundStartResult = "service_draining"

	// BackgroundInvalid 表示任务类型或资源标识无效。
	BackgroundInvalid BackgroundStartResult = "invalid"
)

// BackgroundTaskInfo 是供日志、状态端点和部署脚本读取的任务快照。
type BackgroundTaskInfo struct {
	Key         string              `json:"key"`
	TaskType    string              `json:"task_type"`
	ResourceID  string              `json:"resource_id"`
	Class       BackgroundTaskClass `json:"class"`
	StartedAt   time.Time           `json:"started_at"`
	ElapsedMS   int64               `json:"elapsed_ms"`
	AllowDuring bool                `json:"allow_during_draining"`
}

// BackgroundTaskSummary 是Tracker整体状态快照。
type BackgroundTaskSummary struct {
	Draining   bool                 `json:"draining"`
	Active     int                  `json:"active"`
	Critical   int                  `json:"critical"`
	BestEffort int                  `json:"best_effort"`
	Tasks      []BackgroundTaskInfo `json:"tasks"`
}

// backgroundTaskRecord 是Tracker内部保存的任务记录。
type backgroundTaskRecord struct {
	key                 string
	taskType            string
	resourceID          string
	class               BackgroundTaskClass
	startedAt           time.Time
	allowDuringDraining bool
	onDrain             func()
}

// BackgroundTask 是调用方持有的任务句柄。
//
// Done是幂等的，即使错误路径和defer重复调用也只会完成一次。
type BackgroundTask struct {
	tracker *BackgroundTaskTracker
	key     string
	once    sync.Once
}

// BackgroundTaskTracker 管理当前进程中的全部已登记后台任务。
type BackgroundTaskTracker struct {
	mu       sync.Mutex
	draining bool
	tasks    map[string]*backgroundTaskRecord

	// changed在任务增删或draining状态变化时关闭并重建，
	// Wait通过监听该channel实现无轮询等待。
	changed chan struct{}
}

// 模块日志。
var backgroundTaskLog = logger.WithModule("background_tasks")

// GlobalBackgroundTasks 是生产进程统一使用的后台任务Tracker。
var GlobalBackgroundTasks = NewBackgroundTaskTracker()

// ==================== 构造与任务键 ====================

// NewBackgroundTaskTracker 创建独立Tracker。
//
// 测试可以创建独立实例，避免污染生产全局状态。
func NewBackgroundTaskTracker() *BackgroundTaskTracker {
	return &BackgroundTaskTracker{
		tasks:   make(map[string]*backgroundTaskRecord),
		changed: make(chan struct{}),
	}
}

// buildBackgroundTaskKey 构建稳定唯一任务键。
func buildBackgroundTaskKey(taskType string, resourceID string) string {
	taskType = strings.TrimSpace(taskType)
	resourceID = strings.TrimSpace(resourceID)

	if taskType == "" || resourceID == "" {
		return ""
	}
	return taskType + ":" + resourceID
}

// ==================== 任务登记 ====================

// TryStartExternal 登记用户或外部请求发起的后台任务。
//
// 服务进入draining后，新外部任务会被拒绝。
func (t *BackgroundTaskTracker) TryStartExternal(
	taskType string,
	resourceID string,
	class BackgroundTaskClass,
	onDrain func(),
) (*BackgroundTask, BackgroundStartResult) {
	return t.tryStart(taskType, resourceID, class, false, onDrain)
}

// TryStartChild 登记已运行任务派生出的必要子任务。
//
// 即使服务已经进入draining，已有父任务仍可能需要完成必要收尾，
// 因而派生任务允许登记。调用方应谨慎使用：可重算的非必要任务应直接跳过。
func (t *BackgroundTaskTracker) TryStartChild(
	taskType string,
	resourceID string,
	class BackgroundTaskClass,
	onDrain func(),
) (*BackgroundTask, BackgroundStartResult) {
	return t.tryStart(taskType, resourceID, class, true, onDrain)
}

// tryStart 执行统一任务登记。
func (t *BackgroundTaskTracker) tryStart(
	taskType string,
	resourceID string,
	class BackgroundTaskClass,
	allowDuringDraining bool,
	onDrain func(),
) (*BackgroundTask, BackgroundStartResult) {
	key := buildBackgroundTaskKey(taskType, resourceID)
	if key == "" {
		return nil, BackgroundInvalid
	}

	if class != BackgroundTaskCritical && class != BackgroundTaskBestEffort {
		return nil, BackgroundInvalid
	}

	now := time.Now()

	t.mu.Lock()

	if t.draining && !allowDuringDraining {
		t.mu.Unlock()

		backgroundTaskLog.Warn("服务正在排空，拒绝新的后台任务",
			"task_type", taskType,
			"resource_id", resourceID,
		)
		return nil, BackgroundRejectedDraining
	}

	if _, exists := t.tasks[key]; exists {
		t.mu.Unlock()

		backgroundTaskLog.Info("相同后台任务已经运行，拒绝重复启动",
			"task_key", key,
		)
		return nil, BackgroundAlreadyRunning
	}

	record := &backgroundTaskRecord{
		key:                 key,
		taskType:            strings.TrimSpace(taskType),
		resourceID:          strings.TrimSpace(resourceID),
		class:               class,
		startedAt:           now,
		allowDuringDraining: allowDuringDraining,
		onDrain:             onDrain,
	}
	t.tasks[key] = record
	alreadyDraining := t.draining
	t.signalChangedLocked()
	active := len(t.tasks)
	t.mu.Unlock()

	backgroundTaskLog.Info("后台任务已登记",
		"task_key", key,
		"class", string(class),
		"active_tasks", active,
	)

	// 派生任务在draining期间登记时，若配置了onDrain钩子，应立即触发。
	if alreadyDraining && onDrain != nil {
		safeInvokeBackgroundDrainHook(key, onDrain)
	}

	return &BackgroundTask{
		tracker: t,
		key:     key,
	}, BackgroundStarted
}

// ==================== 任务执行与完成 ====================

// Run 执行任务函数，自动处理Done与panic。
//
// 调用示例：
//
//	go func() {
//	    if err := task.Run(func() error { return service.DoWork() }); err != nil {
//	        log.Warn(...)
//	    }
//	}()
func (task *BackgroundTask) Run(fn func() error) (err error) {
	if task == nil {
		return fmt.Errorf("后台任务句柄为空")
	}

	defer task.Done()

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("后台任务panic: %v", recovered)

			backgroundTaskLog.Error("后台任务panic已恢复",
				"task_key", task.key,
				"panic_value", fmt.Sprintf("%v", recovered),
			)
		}
	}()

	if fn == nil {
		return fmt.Errorf("后台任务执行函数为空")
	}

	return fn()
}

// Done 标记任务完成，幂等。
func (task *BackgroundTask) Done() {
	if task == nil || task.tracker == nil {
		return
	}

	task.once.Do(func() {
		task.tracker.finish(task.key)
	})
}

// finish 从Tracker中删除任务记录。
func (t *BackgroundTaskTracker) finish(key string) {
	t.mu.Lock()

	record, exists := t.tasks[key]
	if !exists {
		t.mu.Unlock()
		return
	}

	delete(t.tasks, key)
	t.signalChangedLocked()
	active := len(t.tasks)
	elapsed := time.Since(record.startedAt)
	t.mu.Unlock()

	backgroundTaskLog.Info("后台任务已完成",
		"task_key", key,
		"elapsed_ms", elapsed.Milliseconds(),
		"active_tasks", active,
	)
}

// ==================== 排空控制 ====================

// BeginDraining 进入排空状态。
//
// 返回进入排空瞬间的任务快照，并在锁外触发各任务的onDrain钩子。
// 重复调用是幂等的，但仍会返回当前任务快照。
func (t *BackgroundTaskTracker) BeginDraining() []BackgroundTaskInfo {
	t.mu.Lock()

	firstTransition := !t.draining
	t.draining = true

	snapshot := t.snapshotLocked(time.Now())
	hooks := make([]struct {
		key string
		fn  func()
	}, 0)

	if firstTransition {
		for key, record := range t.tasks {
			if record.onDrain != nil {
				hooks = append(hooks, struct {
					key string
					fn  func()
				}{
					key: key,
					fn:  record.onDrain,
				})
			}
		}
		t.signalChangedLocked()
	}

	t.mu.Unlock()

	if firstTransition {
		backgroundTaskLog.Info("服务进入后台任务排空状态",
			"active_tasks", len(snapshot),
		)

		for _, hook := range hooks {
			safeInvokeBackgroundDrainHook(hook.key, hook.fn)
		}
	}

	return snapshot
}

// safeInvokeBackgroundDrainHook 安全执行任务排空钩子。
func safeInvokeBackgroundDrainHook(taskKey string, hook func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			backgroundTaskLog.Error("后台任务排空钩子panic已恢复",
				"task_key", taskKey,
				"panic_value", fmt.Sprintf("%v", recovered),
			)
		}
	}()

	hook()
}

// IsDraining 返回当前是否处于排空状态。
func (t *BackgroundTaskTracker) IsDraining() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.draining
}

// Wait 等待全部已登记任务结束。
//
// ctx超时或取消时返回ctx.Err；任务归零时返回nil。
func (t *BackgroundTaskTracker) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		t.mu.Lock()

		if len(t.tasks) == 0 {
			t.mu.Unlock()
			backgroundTaskLog.Info("全部后台任务已排空")
			return nil
		}

		changed := t.changed
		t.mu.Unlock()

		select {
		case <-changed:
			continue
		case <-ctx.Done():
			summary := t.Summary()
			backgroundTaskLog.Warn("等待后台任务排空超时",
				"active_tasks", summary.Active,
				"error", ctx.Err(),
			)
			return ctx.Err()
		}
	}
}

// ==================== 状态查询 ====================

// Summary 返回Tracker完整状态快照。
func (t *BackgroundTaskTracker) Summary() BackgroundTaskSummary {
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	tasks := t.snapshotLocked(now)

	summary := BackgroundTaskSummary{
		Draining: t.draining,
		Active:   len(tasks),
		Tasks:    tasks,
	}

	for _, task := range tasks {
		switch task.Class {
		case BackgroundTaskCritical:
			summary.Critical++
		case BackgroundTaskBestEffort:
			summary.BestEffort++
		}
	}

	return summary
}

// snapshotLocked 在调用方已持锁时创建任务快照。
func (t *BackgroundTaskTracker) snapshotLocked(now time.Time) []BackgroundTaskInfo {
	tasks := make([]BackgroundTaskInfo, 0, len(t.tasks))

	for _, record := range t.tasks {
		tasks = append(tasks, BackgroundTaskInfo{
			Key:         record.key,
			TaskType:    record.taskType,
			ResourceID:  record.resourceID,
			Class:       record.class,
			StartedAt:   record.startedAt,
			ElapsedMS:   now.Sub(record.startedAt).Milliseconds(),
			AllowDuring: record.allowDuringDraining,
		})
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].StartedAt.Equal(tasks[j].StartedAt) {
			return tasks[i].Key < tasks[j].Key
		}
		return tasks[i].StartedAt.Before(tasks[j].StartedAt)
	})

	return tasks
}

// signalChangedLocked 广播Tracker状态发生变化。
// 调用方必须已经持有t.mu。
func (t *BackgroundTaskTracker) signalChangedLocked() {
	close(t.changed)
	t.changed = make(chan struct{})
}

// ==================== 生产全局便捷入口 ====================

// BeginGlobalBackgroundDraining 让生产全局Tracker进入排空状态。
func BeginGlobalBackgroundDraining() []BackgroundTaskInfo {
	return GlobalBackgroundTasks.BeginDraining()
}

// IsGlobalBackgroundDraining 查询生产服务是否正在排空。
func IsGlobalBackgroundDraining() bool {
	return GlobalBackgroundTasks.IsDraining()
}

// GetGlobalBackgroundTaskSummary 获取生产全局任务快照。
func GetGlobalBackgroundTaskSummary() BackgroundTaskSummary {
	return GlobalBackgroundTasks.Summary()
}

// WaitGlobalBackgroundTasks 等待生产全局后台任务全部结束。
func WaitGlobalBackgroundTasks(ctx context.Context) error {
	return GlobalBackgroundTasks.Wait(ctx)
}
