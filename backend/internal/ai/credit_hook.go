package ai

// credit_hook.go — AI调用积分回调钩子
//
// 普通AI调用继续使用全局积分前置检查和成功后异步消费回调。
//
// 课件教学智能体公开运行时采用独立的原子计费桥：
//   - 调用前在部署行锁下占用每日额度和会话主轮次；
//   - 使用部署创建者的个人积分账户执行严格检查；
//   - AI成功后在同一数据库事务内写运行流水、扣积分和写消费流水；
//   - AI失败时只写失败流水并释放主轮次。
//
// 因此SceneCoursewareAssistantRuntime必须跳过本文件的全局钩子，
// 否则会同时触发异步ConsumeTokens而造成重复扣费。
// 跳过的只是积分钩子，不影响模型分流和AI调用追踪。

import (
	"strings"
	"sync"

	"tedna/internal/logger"
)

// SceneCoursewareAssistantRuntime 是公开运行时专用AI场景。
//
// 该场景只能由AssistantRuntimeBillingService完成额度占用和精确结算。
const SceneCoursewareAssistantRuntime = "courseware_assistant_runtime"

// hookLog 模块级结构化日志器。
var hookLog = logger.WithModule("ai.credit_hook")

// CreditConsumeFunc 是AI成功后的积分消费回调。
type CreditConsumeFunc func(
	traceCtx *TraceContext,
	modelUsed string,
	inputTokens int,
	outputTokens int,
	totalTokens int,
	latencyMs int64,
)

// CreditCheckFunc 是AI调用前的余额检查回调。
type CreditCheckFunc func(
	traceCtx *TraceContext,
) (
	bool,
	string,
)

var (
	creditConsumeHook CreditConsumeFunc
	creditCheckHook   CreditCheckFunc
	hookMu            sync.RWMutex
)

// SetCreditHook 设置普通AI调用使用的积分钩子。
func SetCreditHook(
	consumeHook CreditConsumeFunc,
	checkHook CreditCheckFunc,
) {
	hookMu.Lock()
	defer hookMu.Unlock()

	creditConsumeHook = consumeHook
	creditCheckHook = checkHook

	hookLog.Info(
		"积分回调钩子已设置",
		"consume_hook_set",
		consumeHook != nil,
		"check_hook_set",
		checkHook != nil,
	)
}

// IsExternallyBilledTrace 判断是否由独立运行时计费桥结算。
//
// 本函数导出仅供同一后端内部服务和定向测试使用。
func IsExternallyBilledTrace(
	traceCtx *TraceContext,
) bool {
	return traceCtx != nil &&
		strings.TrimSpace(
			traceCtx.SceneCode,
		) ==
			SceneCoursewareAssistantRuntime
}

// invokeCreditConsume 调用普通AI成功消费回调。
func invokeCreditConsume(
	traceCtx *TraceContext,
	modelUsed string,
	inputTokens int,
	outputTokens int,
	totalTokens int,
	latencyMs int64,
) {
	if traceCtx == nil ||
		IsExternallyBilledTrace(traceCtx) {
		return
	}

	hookMu.RLock()
	hook := creditConsumeHook
	hookMu.RUnlock()

	if hook == nil {
		return
	}

	// 流式接口可能不返回usage。全部为0时不生成无法追溯的估算消费。
	if inputTokens == 0 &&
		outputTokens == 0 &&
		totalTokens == 0 {
		hookLog.Info(
			"tokens全部为0，跳过积分消费",
			"scene",
			traceCtx.SceneCode,
			"model",
			modelUsed,
		)
		return
	}

	// 保持存量普通AI链路的异步消费行为。
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				hookLog.Error(
					"积分消费回调panic",
					"error",
					recovered,
				)
			}
		}()

		hook(
			traceCtx,
			modelUsed,
			inputTokens,
			outputTokens,
			totalTokens,
			latencyMs,
		)
	}()
}

// invokeCreditCheck 调用普通AI前置积分检查。
func invokeCreditCheck(
	traceCtx *TraceContext,
) (
	bool,
	string,
) {
	if traceCtx == nil ||
		IsExternallyBilledTrace(traceCtx) {
		return true, ""
	}

	hookMu.RLock()
	hook := creditCheckHook
	hookMu.RUnlock()

	if hook == nil {
		return true, ""
	}

	return hook(traceCtx)
}
