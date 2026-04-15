package services

// sse_hub_test.go — Pipeline SSE广播中心单元测试
//
// 测试范围：
//   - NewSSEHub：创建实例
//   - Subscribe/Unsubscribe：订阅/取消订阅
//   - Broadcast：事件广播（正常/无订阅者/channel满）
//   - GetSubscriberCount/GetTotalSubscribers：计数查询
//   - 并发安全性：多goroutine同时订阅/广播/取消

import (
"sync"
"testing"
"time"
)

// ==================== 基础功能测试 ====================

// TestNewSSEHub 测试创建新的SSE Hub
func TestNewSSEHub(t *testing.T) {
hub := NewSSEHub()
if hub == nil {
t.Fatal("NewSSEHub不应返回nil")
}
if hub.subscribers == nil {
t.Fatal("subscribers map不应为nil")
}
if hub.GetTotalSubscribers() != 0 {
t.Error("新Hub应有0个订阅者")
}
}

// TestSSEHub_Subscribe 测试订阅
func TestSSEHub_Subscribe(t *testing.T) {
hub := NewSSEHub()
ch := hub.Subscribe("pipeline-1")
if ch == nil {
t.Fatal("Subscribe不应返回nil channel")
}
if hub.GetSubscriberCount("pipeline-1") != 1 {
t.Errorf("订阅后应有1个订阅者，实际%d", hub.GetSubscriberCount("pipeline-1"))
}
if hub.GetTotalSubscribers() != 1 {
t.Errorf("总订阅者应为1，实际%d", hub.GetTotalSubscribers())
}
}

// TestSSEHub_MultipleSubscribers 测试同一Pipeline多个订阅者
func TestSSEHub_MultipleSubscribers(t *testing.T) {
hub := NewSSEHub()
ch1 := hub.Subscribe("pipeline-1")
ch2 := hub.Subscribe("pipeline-1")
ch3 := hub.Subscribe("pipeline-2")

if hub.GetSubscriberCount("pipeline-1") != 2 {
t.Errorf("pipeline-1应有2个订阅者，实际%d", hub.GetSubscriberCount("pipeline-1"))
}
if hub.GetSubscriberCount("pipeline-2") != 1 {
t.Errorf("pipeline-2应有1个订阅者，实际%d", hub.GetSubscriberCount("pipeline-2"))
}
if hub.GetTotalSubscribers() != 3 {
t.Errorf("总订阅者应为3，实际%d", hub.GetTotalSubscribers())
}

// 确保channel可用（避免unused变量警告）
_ = ch1
_ = ch2
_ = ch3
}

// TestSSEHub_Unsubscribe 测试取消订阅
func TestSSEHub_Unsubscribe(t *testing.T) {
hub := NewSSEHub()
ch := hub.Subscribe("pipeline-1")

hub.Unsubscribe("pipeline-1", ch)
if hub.GetSubscriberCount("pipeline-1") != 0 {
t.Errorf("取消订阅后应为0，实际%d", hub.GetSubscriberCount("pipeline-1"))
}

// channel应已关闭（读取应返回零值+false）
select {
case _, ok := <-ch:
if ok {
t.Error("取消订阅后channel应已关闭")
}
default:
t.Error("取消订阅后channel应已关闭且可读")
}
}

// TestSSEHub_UnsubscribeNonExistent 取消不存在的订阅不应panic
func TestSSEHub_UnsubscribeNonExistent(t *testing.T) {
hub := NewSSEHub()
fakeCh := make(chan SSEEvent, 1)

// 不应panic
hub.Unsubscribe("non-existent", fakeCh)
hub.Unsubscribe("pipeline-1", fakeCh)
}

// TestSSEHub_UnsubscribeCleansPipelineEntry 最后一个订阅者取消后清理map条目
func TestSSEHub_UnsubscribeCleansPipelineEntry(t *testing.T) {
hub := NewSSEHub()
ch := hub.Subscribe("pipeline-1")
hub.Unsubscribe("pipeline-1", ch)

// 内部map应已清理
hub.mu.Lock()
_, exists := hub.subscribers["pipeline-1"]
hub.mu.Unlock()
if exists {
t.Error("最后一个订阅者取消后，pipeline条目应从map中删除")
}
}

// ==================== 广播测试 ====================

// TestSSEHub_Broadcast 测试正常广播
func TestSSEHub_Broadcast(t *testing.T) {
hub := NewSSEHub()
ch := hub.Subscribe("pipeline-1")

event := SSEEvent{
EventType:   "step_update",
PipelineID:  "pipeline-1",
CurrentStep: "scanner",
StepStatus:  "done",
}
hub.Broadcast("pipeline-1", event)

select {
case received := <-ch:
if received.EventType != "step_update" {
t.Errorf("事件类型应为step_update，实际%s", received.EventType)
}
if received.CurrentStep != "scanner" {
t.Errorf("CurrentStep应为scanner，实际%s", received.CurrentStep)
}
case <-time.After(100 * time.Millisecond):
t.Error("广播超时未收到事件")
}
}

// TestSSEHub_BroadcastToMultipleSubscribers 广播到多个订阅者
func TestSSEHub_BroadcastToMultipleSubscribers(t *testing.T) {
hub := NewSSEHub()
ch1 := hub.Subscribe("pipeline-1")
ch2 := hub.Subscribe("pipeline-1")

event := SSEEvent{EventType: "pipeline_done", PipelineID: "pipeline-1"}
hub.Broadcast("pipeline-1", event)

// 两个订阅者都应收到
for i, ch := range []chan SSEEvent{ch1, ch2} {
select {
case received := <-ch:
if received.EventType != "pipeline_done" {
t.Errorf("订阅者%d: 事件类型不匹配", i+1)
}
case <-time.After(100 * time.Millisecond):
t.Errorf("订阅者%d: 广播超时", i+1)
}
}
}

// TestSSEHub_BroadcastNoSubscribers 无订阅者时广播不panic
func TestSSEHub_BroadcastNoSubscribers(t *testing.T) {
hub := NewSSEHub()
event := SSEEvent{EventType: "step_update", PipelineID: "pipeline-1"}
// 不应panic
hub.Broadcast("pipeline-1", event)
hub.Broadcast("non-existent", event)
}

// TestSSEHub_BroadcastIsolation 不同Pipeline的广播互不干扰
func TestSSEHub_BroadcastIsolation(t *testing.T) {
hub := NewSSEHub()
ch1 := hub.Subscribe("pipeline-1")
ch2 := hub.Subscribe("pipeline-2")

hub.Broadcast("pipeline-1", SSEEvent{EventType: "step_update", PipelineID: "pipeline-1"})

// ch1应收到
select {
case <-ch1:
// 正常
case <-time.After(100 * time.Millisecond):
t.Error("pipeline-1订阅者应收到事件")
}

// ch2不应收到
select {
case <-ch2:
t.Error("pipeline-2订阅者不应收到pipeline-1的事件")
case <-time.After(50 * time.Millisecond):
// 正常，超时表示未收到
}
}

// TestSSEHub_BroadcastFullChannel 测试channel满时的非阻塞行为
func TestSSEHub_BroadcastFullChannel(t *testing.T) {
hub := NewSSEHub()
ch := hub.Subscribe("pipeline-1")

// 填满channel（缓冲大小为10）
for i := 0; i < 10; i++ {
hub.Broadcast("pipeline-1", SSEEvent{EventType: "step_update"})
}

// 第11条应被跳过，不阻塞
done := make(chan bool, 1)
go func() {
hub.Broadcast("pipeline-1", SSEEvent{EventType: "overflow"})
done <- true
}()

select {
case <-done:
// 正常，非阻塞返回
case <-time.After(1 * time.Second):
t.Error("channel满时Broadcast不应阻塞")
}

// 确认channel中有10条（缓冲满）
if len(ch) != 10 {
t.Errorf("channel应有10条消息，实际%d", len(ch))
}
}

// ==================== 并发安全性测试 ====================

// TestSSEHub_ConcurrentSubscribeUnsubscribe 并发订阅和取消不panic
func TestSSEHub_ConcurrentSubscribeUnsubscribe(t *testing.T) {
hub := NewSSEHub()
var wg sync.WaitGroup

// 50个goroutine同时订阅、广播、取消
for i := 0; i < 50; i++ {
wg.Add(1)
go func(id int) {
defer wg.Done()
pipelineID := "pipeline-concurrent"
ch := hub.Subscribe(pipelineID)
hub.Broadcast(pipelineID, SSEEvent{EventType: "test"})
hub.Unsubscribe(pipelineID, ch)
}(i)
}

wg.Wait()
// 所有goroutine完成后不应有残留订阅
if hub.GetTotalSubscribers() != 0 {
t.Errorf("并发完成后应无残留订阅者，实际%d", hub.GetTotalSubscribers())
}
}

// TestSSEHub_GetSubscriberCountNonExistent 查询不存在的Pipeline返回0
func TestSSEHub_GetSubscriberCountNonExistent(t *testing.T) {
hub := NewSSEHub()
if hub.GetSubscriberCount("non-existent") != 0 {
t.Error("不存在的Pipeline应返回0")
}
}
