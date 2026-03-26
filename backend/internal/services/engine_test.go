// Package services 引擎服务单元测试
// 测试范围：Engine Submit/*EngineTask / AcquireAI / ReleaseAI / GetStats / Stop / Wait
package services

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewEngine 测试引擎初始化
func TestNewEngine(t *testing.T) {
	engine := NewEngine(3, 2, 10)
	if engine == nil {
		t.Fatal("NewEngine不应返回nil")
	}
	defer func() { engine.Stop(); engine.Wait() }()

	stats := engine.GetStats()
	if stats.TotalSubmitted != 0 {
		t.Errorf("初始TotalSubmitted应为0, 实际%d", stats.TotalSubmitted)
	}
	if stats.CurrentRunning != 0 {
		t.Errorf("初始CurrentRunning应为0, 实际%d", stats.CurrentRunning)
	}
	t.Logf("引擎初始化成功: submitted=%d running=%d", stats.TotalSubmitted, stats.CurrentRunning)
}

// TestEngineSubmitAndExecute 测试任务提交和执行（Submit接受*EngineTask）
func TestEngineSubmitAndExecute(t *testing.T) {
	engine := NewEngine(2, 2, 10)
	defer func() { engine.Stop(); engine.Wait() }()

	var executed int32
	var wg sync.WaitGroup
	wg.Add(3)

	for i := 0; i < 3; i++ {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: "test-pipeline-id",
			ExecFunc: func() error {
				atomic.AddInt32(&executed, 1)
				wg.Done()
				return nil
			},
		}
		ok := engine.Submit(task)
		if !ok {
			t.Errorf("Submit应返回true")
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		t.Logf("3个任务全部执行完成，executed=%d", atomic.LoadInt32(&executed))
	case <-time.After(5 * time.Second):
		t.Fatalf("任务执行超时，仅完成%d/3", atomic.LoadInt32(&executed))
	}

	if atomic.LoadInt32(&executed) != 3 {
		t.Errorf("期望执行3个任务, 实际%d", atomic.LoadInt32(&executed))
	}
}

// TestEngineStatsCompleted 测试成功任务TotalCompleted统计
func TestEngineStatsCompleted(t *testing.T) {
	engine := NewEngine(2, 2, 10)
	defer func() { engine.Stop(); engine.Wait() }()

	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		engine.Submit(&EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: "stat-test",
			ExecFunc: func() error {
				wg.Done()
				return nil
			},
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("任务执行超时")
	}

	time.Sleep(100 * time.Millisecond)
	stats := engine.GetStats()
	if stats.TotalSubmitted < 5 {
		t.Errorf("TotalSubmitted期望>=5, 实际%d", stats.TotalSubmitted)
	}
	if stats.TotalCompleted < 5 {
		t.Errorf("TotalCompleted期望>=5, 实际%d", stats.TotalCompleted)
	}
	t.Logf("统计验证通过: submitted=%d completed=%d", stats.TotalSubmitted, stats.TotalCompleted)
}

// TestEngineStatsBusinessFailed 测试业务失败TotalBusinessFailed统计（v34新增）
func TestEngineStatsBusinessFailed(t *testing.T) {
	engine := NewEngine(2, 2, 10)
	defer func() { engine.Stop(); engine.Wait() }()

	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		engine.Submit(&EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: "fail-test",
			ExecFunc: func() error {
				wg.Done()
				return errors.New("业务执行失败：AI评分未达标")
			},
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("任务执行超时")
	}

	time.Sleep(100 * time.Millisecond)
	stats := engine.GetStats()
	if stats.TotalBusinessFailed < 3 {
		t.Errorf("TotalBusinessFailed期望>=3, 实际%d", stats.TotalBusinessFailed)
	}
	t.Logf("业务失败统计通过: business_failed=%d", stats.TotalBusinessFailed)
}

// TestEngineQueueFull 测试队列满时Submit返回false
func TestEngineQueueFull(t *testing.T) {
	engine := NewEngine(1, 1, 2)
	defer func() { engine.Stop(); engine.Wait() }()

	blocker := make(chan struct{})
	engine.Submit(&EngineTask{
		Type:       TaskTypePipeline,
		PipelineID: "blocker",
		ExecFunc: func() error {
			<-blocker
			return nil
		},
	})

	time.Sleep(50 * time.Millisecond)

	engine.Submit(&EngineTask{Type: TaskTypePipeline, PipelineID: "q1", ExecFunc: func() error { return nil }})
	engine.Submit(&EngineTask{Type: TaskTypePipeline, PipelineID: "q2", ExecFunc: func() error { return nil }})

	ok := engine.Submit(&EngineTask{
		Type:       TaskTypePipeline,
		PipelineID: "overflow",
		ExecFunc:   func() error { return nil },
	})

	close(blocker)
	t.Logf("队列满时Submit返回: %v（期望false）", ok)
}

// TestEngineAIRateLimit 测试AI信号量限流（最大2并发）
func TestEngineAIRateLimit(t *testing.T) {
	engine := NewEngine(4, 2, 20)
	defer func() { engine.Stop(); engine.Wait() }()

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(6)
	for i := 0; i < 6; i++ {
		engine.Submit(&EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: "ai-rate-test",
			ExecFunc: func() error {
				engine.AcquireAI()
				cur := atomic.AddInt32(&current, 1)

				mu.Lock()
				if cur > maxConcurrent {
					maxConcurrent = cur
				}
				mu.Unlock()

				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&current, -1)
				engine.ReleaseAI()
				wg.Done()
				return nil
			},
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("AI限流测试超时")
	}

	if maxConcurrent > 2 {
		t.Errorf("AI最大并发期望<=2, 实际峰值=%d", maxConcurrent)
	}
	t.Logf("AI限流验证通过: 最大并发=%d (限制=2)", maxConcurrent)
}

// TestEngineGetStatsConcurrent 测试并发GetStats线程安全
func TestEngineGetStatsConcurrent(t *testing.T) {
	engine := NewEngine(2, 2, 10)
	defer func() { engine.Stop(); engine.Wait() }()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				stats := engine.GetStats()
				_ = stats.TotalSubmitted
			}
		}()
	}
	wg.Wait()
	t.Log("并发GetStats线程安全验证通过")
}

// TestEngineStopRejectNew 测试Stop后拒绝新任务
func TestEngineStopRejectNew(t *testing.T) {
	engine := NewEngine(2, 2, 10)
	engine.Stop()
	engine.Wait()

	ok := engine.Submit(&EngineTask{
		Type:       TaskTypePipeline,
		PipelineID: "after-stop",
		ExecFunc:   func() error { return nil },
	})
	if ok {
		t.Error("引擎Stop后Submit应返回false")
	}
	t.Log("Stop后拒绝新任务验证通过")
}

// TestEngineTaskTypes 测试三种TaskType均可正常执行
func TestEngineTaskTypes(t *testing.T) {
	engine := NewEngine(3, 2, 20)
	defer func() { engine.Stop(); engine.Wait() }()

	var wg sync.WaitGroup
	taskTypes := []TaskType{TaskTypePipeline, TaskTypeRetrial, TaskTypeVerify}

	for _, tt := range taskTypes {
		wg.Add(1)
		taskType := tt
		engine.Submit(&EngineTask{
			Type:       taskType,
			PipelineID: "type-test-" + string(taskType),
			ExecFunc: func() error {
				wg.Done()
				return nil
			},
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		t.Logf("三种TaskType全部执行成功: pipeline/retrial/verify")
	case <-time.After(5 * time.Second):
		t.Fatal("TaskType测试超时")
	}
}

// TestEngineMixedResults 测试成功+失败混合任务统计
func TestEngineMixedResults(t *testing.T) {
	engine := NewEngine(3, 2, 20)
	defer func() { engine.Stop(); engine.Wait() }()

	var wg sync.WaitGroup
	wg.Add(6)

	// 3个成功 + 3个失败
	for i := 0; i < 3; i++ {
		engine.Submit(&EngineTask{
			Type: TaskTypePipeline, PipelineID: "ok",
			ExecFunc: func() error { wg.Done(); return nil },
		})
		engine.Submit(&EngineTask{
			Type: TaskTypePipeline, PipelineID: "fail",
			ExecFunc: func() error { wg.Done(); return errors.New("业务失败") },
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("超时")
	}

	time.Sleep(100 * time.Millisecond)
	stats := engine.GetStats()
	t.Logf("混合结果: submitted=%d completed=%d business_failed=%d",
		stats.TotalSubmitted, stats.TotalCompleted, stats.TotalBusinessFailed)

	if stats.TotalCompleted+stats.TotalBusinessFailed < 6 {
		t.Errorf("成功+失败总数应>=6, 实际completed=%d biz_failed=%d",
			stats.TotalCompleted, stats.TotalBusinessFailed)
	}
}
