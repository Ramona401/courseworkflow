package models

// ai_trace.go — AI调用追踪数据模型
//
// 职责：
//   1. 定义ai_call_traces表对应的数据结构
//   2. 定义追踪写入请求（内部使用，由ai/client.go埋点）
//   3. 定义仪表盘聚合查询的响应结构（含按用户/按组织维度）
//   4. 提供模型定价表和成本估算函数
//
// v85变更：
//   TraceRecord新增IsFallback+OriginalModel字段，记录模型降级信息
//   AICallTrace新增IsFallback+OriginalModel字段
//
// 被引用：
//   repository/ai_trace_repo.go — 数据访问层
//   handlers/ai_trace_handler.go — HTTP处理器
//   ai/client.go — 埋点写入

import (
	"time"
)

// ==================== 数据库实体 ====================

// AICallTrace 对应数据库 ai_call_traces 表的完整记录
type AICallTrace struct {
	ID               string    `json:"id"`
	SceneCode        string    `json:"scene_code"`
	ModelUsed        string    `json:"model_used"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	LatencyMs        int       `json:"latency_ms"`
	Status           string    `json:"status"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	PipelineID       *string   `json:"pipeline_id,omitempty"`
	LessonPlanID     *string   `json:"lesson_plan_id,omitempty"`
	UserID           *string   `json:"user_id,omitempty"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd"`
	OutputLength     int       `json:"output_length"`
	IsStream         bool      `json:"is_stream"`
	IsFallback       bool      `json:"is_fallback"`       // v85新增：是否为降级调用
	OriginalModel    string    `json:"original_model"`    // v85新增：降级前的原始模型
	CreatedAt        time.Time `json:"created_at"`
}

// ==================== 追踪写入请求（内部使用）====================

// TraceRecord AI调用埋点记录
// 由ai/client.go的CallAI/CallAIStream/CallAIMultimodal在调用完成后构造
// 通过repository.EnqueueTrace异步写入数据库，不影响主路径延迟
type TraceRecord struct {
	SceneCode        string  // 场景代码
	ModelUsed        string  // 实际使用的模型
	PromptTokens     int     // 输入token数（如果API返回了分开的usage）
	CompletionTokens int     // 输出token数
	TotalTokens      int     // 总token数
	LatencyMs        int64   // 调用耗时（毫秒）
	Status           string  // success / error / timeout / fallback
	ErrorMessage     string  // 失败时的错误信息
	PipelineID       *string // 关联Pipeline ID（可为空）
	LessonPlanID     *string // 关联教案 ID（可为空）
	UserID           *string // 关联用户 ID（可为空）
	OutputLength     int     // AI输出内容长度（字符数）
	IsStream         bool    // 是否流式调用
	IsFallback       bool    // v85新增：是否为降级调用
	OriginalModel    string  // v85新增：降级前的原始模型（仅IsFallback=true时有值）
}

// ==================== 仪表盘聚合响应 ====================

// TraceSceneStats 按场景聚合的统计数据
type TraceSceneStats struct {
	SceneCode             string  `json:"scene_code"`
	SceneName             string  `json:"scene_name"`
	CallCount             int     `json:"call_count"`
	SuccessCount          int     `json:"success_count"`
	ErrorCount            int     `json:"error_count"`
	AvgLatencyMs          int     `json:"avg_latency_ms"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
}

// TraceModelStats 按模型聚合的统计数据
type TraceModelStats struct {
	ModelUsed        string  `json:"model_used"`
	CallCount        int     `json:"call_count"`
	SuccessCount     int     `json:"success_count"`
	ErrorCount       int     `json:"error_count"`
	AvgLatencyMs     int     `json:"avg_latency_ms"`
	TotalTokens      int64   `json:"total_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

// TraceUserStats 按用户聚合的统计数据（v81新增）
// 通过ai_call_traces.user_id关联users表获取用户信息
type TraceUserStats struct {
	UserID                string  `json:"user_id"`
	Username              string  `json:"username"`
	DisplayName           string  `json:"display_name"`
	Role                  string  `json:"role"`
	CallCount             int     `json:"call_count"`
	SuccessCount          int     `json:"success_count"`
	ErrorCount            int     `json:"error_count"`
	AvgLatencyMs          int     `json:"avg_latency_ms"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
}

// TraceOrgStats 按组织聚合的统计数据（v81新增）
// 通过user_id → teaching_group_members → teaching_groups → organizations关联
// 一个用户可能属于多个组织，此处按学校维度聚合
type TraceOrgStats struct {
	OrgID                 string  `json:"org_id"`
	OrgName               string  `json:"org_name"`
	OrgType               string  `json:"org_type"`
	MemberCount           int     `json:"member_count"`
	CallCount             int     `json:"call_count"`
	SuccessCount          int     `json:"success_count"`
	ErrorCount            int     `json:"error_count"`
	AvgLatencyMs          int     `json:"avg_latency_ms"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
}

// TraceDailyTrend 按日期聚合的趋势数据
type TraceDailyTrend struct {
	Date             string  `json:"date"` // 格式: YYYY-MM-DD
	CallCount        int     `json:"call_count"`
	ErrorCount       int     `json:"error_count"`
	TotalTokens      int64   `json:"total_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	AvgLatencyMs     int     `json:"avg_latency_ms"`
}

// TraceDashboard 仪表盘总览数据（一次请求返回全部聚合）
type TraceDashboard struct {
	// 概览数字
	TotalCalls   int     `json:"total_calls"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	AvgLatencyMs int     `json:"avg_latency_ms"`
	ErrorRate    float64 `json:"error_rate"` // 百分比，如 2.35 表示 2.35%
	// v85新增：降级统计
	FallbackCount int     `json:"fallback_count"` // 降级调用总次数
	FallbackRate  float64 `json:"fallback_rate"`  // 降级率（百分比）
	// 分维度聚合
	ByScene    []TraceSceneStats `json:"by_scene"`
	ByModel    []TraceModelStats `json:"by_model"`
	ByUser     []TraceUserStats  `json:"by_user"`  // v81新增：按用户聚合
	ByOrg      []TraceOrgStats   `json:"by_org"`   // v81新增：按组织聚合
	DailyTrend []TraceDailyTrend `json:"daily_trend"`
	// 最近的错误记录
	RecentErrors []AICallTrace `json:"recent_errors"`
}

// TraceQueryParams 仪表盘查询参数
type TraceQueryParams struct {
	SceneCode string // 按场景筛选（可为空）
	ModelUsed string // 按模型筛选（可为空）
	Status    string // 按状态筛选（可为空）
	DateFrom  string // 起始日期 YYYY-MM-DD（可为空）
	DateTo    string // 结束日期 YYYY-MM-DD（可为空）
	Page      int    // 分页页码（错误列表用）
	PageSize  int    // 分页大小
}

// ==================== 模型定价表 ====================

// ModelPricing 单个模型的定价（美元/百万token）
type ModelPricing struct {
	PromptPer1M     float64 // 输入价格
	CompletionPer1M float64 // 输出价格
}

// ModelPricingMap 已知模型的定价表
// 根据Anthropic官方定价维护，新增模型时在此添加
var ModelPricingMap = map[string]ModelPricing{
	"anthropic/claude-opus-4.6":   {PromptPer1M: 15.0, CompletionPer1M: 75.0},
	"anthropic/claude-sonnet-4.6": {PromptPer1M: 3.0, CompletionPer1M: 15.0},
	"anthropic/claude-haiku-4.5":  {PromptPer1M: 0.80, CompletionPer1M: 4.0},
	// 旧版模型兼容
	"anthropic/claude-sonnet-4-5": {PromptPer1M: 3.0, CompletionPer1M: 15.0},
}

// EstimateCost 根据模型名和token数估算调用成本（美元）
// 未知模型按Sonnet价格估算（中间档）
func EstimateCost(model string, promptTokens, completionTokens int) float64 {
	pricing, ok := ModelPricingMap[model]
	if !ok {
		// 未知模型按Sonnet价格估算
		pricing = ModelPricingMap["anthropic/claude-sonnet-4.6"]
	}
	cost := float64(promptTokens)/1_000_000*pricing.PromptPer1M +
		float64(completionTokens)/1_000_000*pricing.CompletionPer1M
	return cost
}
