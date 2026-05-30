package ai

// credit_hook.go — AI调用积分回调钩子
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）
//
// 设计思路：
//   CallAI/CallAIStream/CallAIMultimodal 都是包级函数，无法持有 TokenService 引用。
//   为避免修改所有调用方（十几个service文件），采用回调钩子方案：
//   - 在 routes.go 初始化时通过 SetCreditHook 注入回调函数
//   - CallAI 等函数在成功完成后自动调用钩子
//   - 钩子为nil时静默跳过（向后兼容）
//
// 回调数据通过 TraceContext 传递（已有 UserID/SceneCode/PipelineID/LessonPlanID）
// 新增 SchoolID 字段到 TraceContext 中

import (
	"sync"

	"tedna/internal/logger"
)

// hookLog 模块级结构化日志器
var hookLog = logger.WithModule("ai.credit_hook")

// CreditConsumeFunc 积分消费回调函数类型
// 参数说明：
//
//	traceCtx     — 追踪上下文（含UserID/SceneCode等，来自调用方）
//	modelUsed    — AI实际使用的模型名称
//	inputTokens  — 输入token数（prompt_tokens）
//	outputTokens — 输出token数（completion_tokens）
//	totalTokens  — 总token数
//	latencyMs    — 调用耗时（毫秒）
type CreditConsumeFunc func(
	traceCtx *TraceContext,
	modelUsed string,
	inputTokens int,
	outputTokens int,
	totalTokens int,
	latencyMs int64,
)

// CreditCheckFunc 积分前置检查回调函数类型
// 返回值：(是否放行, 错误信息)
// 返回 (true, "") 表示放行；(false, "积分余额不足") 表示拦截
type CreditCheckFunc func(traceCtx *TraceContext) (bool, string)

var (
	// creditConsumeHook 积分消费回调（AI调用成功后触发）
	creditConsumeHook CreditConsumeFunc
	// creditCheckHook 积分前置检查回调（AI调用前触发）
	creditCheckHook CreditCheckFunc
	// hookMu 保护回调设置的互斥锁
	hookMu sync.RWMutex
)

// SetCreditHook 设置积分回调钩子（在routes.go初始化时调用一次）
// consumeHook: AI调用成功后的积分消费回调
// checkHook: AI调用前的余额检查回调（可为nil表示不检查）
func SetCreditHook(consumeHook CreditConsumeFunc, checkHook CreditCheckFunc) {
	hookMu.Lock()
	defer hookMu.Unlock()
	creditConsumeHook = consumeHook
	creditCheckHook = checkHook
	hookLog.Info("积分回调钩子已设置",
		"consume_hook_set", consumeHook != nil,
		"check_hook_set", checkHook != nil)
}

// invokeCreditConsume 调用积分消费回调（内部函数，AI调用成功后调用）
// 如果钩子未设置或traceCtx为nil，静默跳过
func invokeCreditConsume(traceCtx *TraceContext, modelUsed string, inputTokens int, outputTokens int, totalTokens int, latencyMs int64) {
	if traceCtx == nil {
		return
	}
	hookMu.RLock()
	hook := creditConsumeHook
	hookMu.RUnlock()

	if hook == nil {
		return
	}

	// v129.1修复：流式API（特别是Gemini）可能不返回usage统计
	// 当所有token数都为0时，不触发积分消费（由emitTrace中的output_length间接估算不可靠）
	// 当totalTokens>0但input/output为0时，用总数按6:4估算
	if inputTokens == 0 && outputTokens == 0 && totalTokens == 0 {
		hookLog.Info("tokens全部为0，跳过积分消费",
			"scene", traceCtx.SceneCode,
			"model", modelUsed)
		return
	}

	// 异步执行积分消费，不阻塞AI调用返回
	go func() {
		defer func() {
			if r := recover(); r != nil {
				hookLog.Error("积分消费回调panic",
					"error", r)
			}
		}()
		hook(traceCtx, modelUsed, inputTokens, outputTokens, totalTokens, latencyMs)
	}()
}

// invokeCreditCheck 调用积分前置检查（内部函数，AI调用前调用）
// 如果钩子未设置或traceCtx为nil，返回放行
// 返回值：(是否放行, 错误信息)
func invokeCreditCheck(traceCtx *TraceContext) (bool, string) {
	if traceCtx == nil {
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
