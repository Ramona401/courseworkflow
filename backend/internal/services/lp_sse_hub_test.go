package services

// lp_sse_hub_test.go — 教案系统SSE广播中心单元测试
//
// 测试范围：
//   - NewLPSSEHub：创建实例
//   - Subscribe：独占模式（新连接关闭旧连接）
//   - Unsubscribe：取消订阅
//   - Broadcast：事件广播（含channel满处理）
//   - 并发安全性

import (
"sync"
"testing"
"time"

"tedna/internal/models"
)

// ==================== 基础功能测试 ====================

// TestNewLPSSEHub 测试创建新的教案SSE Hub
func TestNewLPSSEHub(t *testing.T) {
hub := NewLPSSEHub()
if hub == nil {
t.Fatal("NewLPSSEHub不应返回nil")
}
if hub.subscribers == nil {
t.Fatal("subscribers map不应为nil")
}
}

// TestLPSSEHub_Subscribe 测试订阅
func TestLPSSEHub_Subscribe(t *testing.T) {
hub := NewLPSSEHub()
ch := hub.Subscribe("plan-1")
if ch == nil {
t.Fatal("Subscribe不应返回nil channel")
}
// channel缓冲应为2000
if cap(ch) != 2000 {
t.Errorf("channel缓冲应为2000，实际%d", cap(ch))
}
}

// TestLPSSEHub_SubscribeExclusiveMode 独占模式：新订阅关闭旧连接
func TestLPSSEHub_SubscribeExclusiveMode(t *testing.T) {
hub := NewLPSSEHub()

// 第一次订阅
ch1 := hub.Subscribe("plan-1")

// 第二次订阅同一planID——ch1应被关闭
ch2 := hub.Subscribe("plan-1")

// ch1应已关闭
select {
case _, ok := <-ch1:
if ok {
t.Error("旧channel应已关闭（ok应为false）")
}
default:
t.Error("旧channel应已关闭且可读取到关闭信号")
}

// ch2应正常可用
if ch2 == nil {
t.Error("新channel不应为nil")
}

// 只有1个活跃订阅
hub.mu.Lock()
subs, exists := hub.subscribers["plan-1"]
subCount := 0
if exists {
subCount = len(subs)
}
hub.mu.Unlock()
if subCount != 1 {
t.Errorf("独占模式下应只有1个订阅者，实际%d", subCount)
}
}

// TestLPSSEHub_Unsubscribe 测试取消订阅
func TestLPSSEHub_Unsubscribe(t *testing.T) {
hub := NewLPSSEHub()
ch := hub.Subscribe("plan-1")

hub.Unsubscribe("plan-1", ch)

// channel应已关闭
select {
case _, ok := <-ch:
if ok {
t.Error("取消订阅后channel应已关闭")
}
default:
t.Error("取消订阅后channel应已关闭且可读")
}

// map条目应已清理
hub.mu.Lock()
_, exists := hub.subscribers["plan-1"]
hub.mu.Unlock()
if exists {
t.Error("取消后map条目应已清理")
}
}

// TestLPSSEHub_UnsubscribeNonExistent 取消不存在的订阅不panic
func TestLPSSEHub_UnsubscribeNonExistent(t *testing.T) {
hub := NewLPSSEHub()
fakeCh := make(chan models.LPSSEEvent, 1)
// 不应panic
hub.Unsubscribe("non-existent", fakeCh)
}

// ==================== 广播测试 ====================

// TestLPSSEHub_Broadcast 测试正常广播
func TestLPSSEHub_Broadcast(t *testing.T) {
hub := NewLPSSEHub()
ch := hub.Subscribe("plan-1")

event := models.LPSSEEvent{
EventType: models.LPSSEChunk,
PlanID:    "plan-1",
Chunk:     "Hello",
}
hub.Broadcast("plan-1", event)

select {
case received := <-ch:
if received.EventType != models.LPSSEChunk {
t.Errorf("事件类型应为chunk，实际%s", string(received.EventType))
}
if received.Chunk != "Hello" {
t.Errorf("Chunk应为Hello，实际%s", received.Chunk)
}
case <-time.After(100 * time.Millisecond):
t.Error("广播超时未收到事件")
}
}

// TestLPSSEHub_BroadcastNoSubscribers 无订阅者时广播不panic
func TestLPSSEHub_BroadcastNoSubscribers(t *testing.T) {
hub := NewLPSSEHub()
event := models.LPSSEEvent{EventType: models.LPSSEChunk, PlanID: "plan-1"}
// 不应panic
hub.Broadcast("plan-1", event)
hub.Broadcast("non-existent", event)
}

// TestLPSSEHub_BroadcastIsolation 不同Plan的广播互不干扰
func TestLPSSEHub_BroadcastIsolation(t *testing.T) {
hub := NewLPSSEHub()
ch1 := hub.Subscribe("plan-1")
ch2 := hub.Subscribe("plan-2")

hub.Broadcast("plan-1", models.LPSSEEvent{EventType: models.LPSSEChunk, PlanID: "plan-1"})

select {
case <-ch1:
// 正常
case <-time.After(100 * time.Millisecond):
t.Error("plan-1订阅者应收到事件")
}

select {
case <-ch2:
t.Error("plan-2订阅者不应收到plan-1的事件")
case <-time.After(50 * time.Millisecond):
// 正常
}
}

// ==================== 并发安全性测试 ====================

// TestLPSSEHub_ConcurrentOperations 并发操作不panic
func TestLPSSEHub_ConcurrentOperations(t *testing.T) {
hub := NewLPSSEHub()
var wg sync.WaitGroup

for i := 0; i < 30; i++ {
wg.Add(1)
go func() {
defer wg.Done()
ch := hub.Subscribe("plan-concurrent")
hub.Broadcast("plan-concurrent", models.LPSSEEvent{EventType: models.LPSSEChunk})
hub.Unsubscribe("plan-concurrent", ch)
}()
}

wg.Wait()
}

// TestLPSSEHub_SubscribeMultipleThenBroadcast 多次独占订阅后广播
func TestLPSSEHub_SubscribeMultipleThenBroadcast(t *testing.T) {
hub := NewLPSSEHub()

// 连续订阅3次，只有最后一个有效
_ = hub.Subscribe("plan-1")
_ = hub.Subscribe("plan-1")
ch3 := hub.Subscribe("plan-1")

hub.Broadcast("plan-1", models.LPSSEEvent{EventType: models.LPSSEMessageDone, PlanID: "plan-1"})

select {
case received := <-ch3:
if received.EventType != models.LPSSEMessageDone {
t.Errorf("最后一个订阅者应收到message_done，实际%s", string(received.EventType))
}
case <-time.After(100 * time.Millisecond):
t.Error("最后一个订阅者应收到广播")
}
}
