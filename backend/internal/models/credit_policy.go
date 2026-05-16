package models

// credit_policy.go — 积分策略 + 模型单价数据模型
//
// v129 新增（积分机制融合 · 对齐AOCI精确积分计算）：
//   - CreditPolicy：积分策略实体（系统级/学校级，查询链回退）
//   - ModelPrice：模型单价实体（每1K token的美元成本）
//   - CreditCalculation：AI调用后的积分计算结果（完整过程可追溯）
//   - 相关请求/响应结构体
//
// 积分计算公式（对齐AOCI）：
//   cost_usd = (input_tokens/1000 × cost_per_1k_input) + (output_tokens/1000 × cost_per_1k_output)
//   credits  = cost_usd × exchange_rate × multiplier
//
// 对应数据库表：
//   token_credit_policies — 积分策略（系统级/学校级）
//   token_model_prices    — 模型单价

import "time"

// ==================== 积分策略常量 ====================

const (
	// PolicyScopeSystem 系统级策略
	PolicyScopeSystem = "system"
	// PolicyScopeSchool 学校级策略
	PolicyScopeSchool = "school"

	// DefaultExchangeRate 默认汇率（美元→积分），对齐AOCI默认值
	DefaultExchangeRate = 7.0
	// DefaultMultiplier 默认倍率
	DefaultMultiplier = 1.0
)

// ==================== 积分策略实体 ====================

// CreditPolicy 积分策略实体（对应 token_credit_policies 表）
// 对齐AOCI的credit_policies表设计
// 查询链: 学校策略(school, schoolID) → 系统策略(system, NULL) → 默认兜底(7.0 × 1.0)
type CreditPolicy struct {
	ID           string    `json:"id"`
	Scope        string    `json:"scope"`         // system / school
	ScopeID      *string   `json:"scope_id"`      // system: nil, school: organization_id
	ExchangeRate float64   `json:"exchange_rate"`  // 美元→积分汇率，默认7.0
	Multiplier   float64   `json:"multiplier"`     // 倍率，默认1.0
	Description  string    `json:"description"`    // 策略描述
	UpdatedBy    *string   `json:"updated_by"`     // 最后修改人
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EffectiveRate 有效汇率 = 汇率 × 倍率
func (p *CreditPolicy) EffectiveRate() float64 {
	return p.ExchangeRate * p.Multiplier
}

// CalculateCredits 计算积分消耗 = 美元成本 × 汇率 × 倍率
func (p *CreditPolicy) CalculateCredits(costUSD float64) float64 {
	return costUSD * p.ExchangeRate * p.Multiplier
}

// ==================== 模型单价实体 ====================

// ModelPrice 模型单价实体（对应 token_model_prices 表）
// 对齐AOCI的ai_models表中cost_per_1k_in/cost_per_1k_out字段
type ModelPrice struct {
	ID              string    `json:"id"`
	ModelName       string    `json:"model_name"`        // 如 claude-sonnet-4-20250514
	Provider        string    `json:"provider"`          // anthropic / google / openai
	CostPer1kInput  float64   `json:"cost_per_1k_input"` // 每1K输入token美元成本
	CostPer1kOutput float64   `json:"cost_per_1k_output"` // 每1K输出token美元成本
	DisplayName     string    `json:"display_name"`      // 显示名称
	IsActive        bool      `json:"is_active"`         // 是否启用
	UpdatedBy       *string   `json:"updated_by"`        // 最后修改人
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CalculateCostUSD 根据真实token数计算美元成本
// 公式: (input_tokens/1000 × cost_per_1k_input) + (output_tokens/1000 × cost_per_1k_output)
func (mp *ModelPrice) CalculateCostUSD(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) / 1000.0 * mp.CostPer1kInput
	outputCost := float64(outputTokens) / 1000.0 * mp.CostPer1kOutput
	return inputCost + outputCost
}

// ==================== 积分计算结果 ====================

// CreditCalculation AI调用后的积分计算结果
// 包含完整计算过程，写入consumption_logs供追溯
type CreditCalculation struct {
	InputTokens     int     `json:"input_tokens"`      // 输入token数
	OutputTokens    int     `json:"output_tokens"`     // 输出token数
	ModelName       string  `json:"model_name"`        // 模型名称
	Provider        string  `json:"provider"`          // 供应商
	CostUSD         float64 `json:"cost_usd"`          // 美元成本
	ExchangeRate    float64 `json:"exchange_rate"`     // 汇率
	Multiplier      float64 `json:"multiplier"`        // 倍率
	CreditsConsumed float64 `json:"credits_consumed"`  // 最终积分消耗
	LatencyMs       int64   `json:"latency_ms"`        // 调用耗时（毫秒）
}

// ==================== 请求结构体 ====================

// UpdateCreditPolicyRequest 更新策略请求
type UpdateCreditPolicyRequest struct {
	ExchangeRate *float64 `json:"exchange_rate"` // 汇率（nil表示不更新）
	Multiplier   *float64 `json:"multiplier"`    // 倍率（nil表示不更新）
	Description  *string  `json:"description"`   // 描述（nil表示不更新）
}

// CreateModelPriceRequest 创建模型单价请求
type CreateModelPriceRequest struct {
	ModelName       string  `json:"model_name"`        // 模型名称（唯一）
	Provider        string  `json:"provider"`          // 供应商
	CostPer1kInput  float64 `json:"cost_per_1k_input"` // 输入单价
	CostPer1kOutput float64 `json:"cost_per_1k_output"` // 输出单价
	DisplayName     string  `json:"display_name"`      // 显示名称
}

// UpdateModelPriceRequest 更新模型单价请求
type UpdateModelPriceRequest struct {
	CostPer1kInput  *float64 `json:"cost_per_1k_input"`  // nil表示不更新
	CostPer1kOutput *float64 `json:"cost_per_1k_output"` // nil表示不更新
	DisplayName     *string  `json:"display_name"`       // nil表示不更新
	IsActive        *bool    `json:"is_active"`          // nil表示不更新
}

// SimulateCreditRequest 模拟积分计算请求
type SimulateCreditRequest struct {
	ModelName    string  `json:"model_name"`    // 模型名称
	InputTokens  int     `json:"input_tokens"`  // 输入token数
	OutputTokens int     `json:"output_tokens"` // 输出token数
	SchoolID     *string `json:"school_id"`     // 学校ID（可选，用于查学校策略）
}

// ==================== 响应结构体 ====================

// ModelPricePreview 模型积分预览（列表展示用）
type ModelPricePreview struct {
	ModelName          string  `json:"model_name"`
	Provider           string  `json:"provider"`
	DisplayName        string  `json:"display_name"`
	CostPer1kInput     float64 `json:"cost_per_1k_input"`
	CostPer1kOutput    float64 `json:"cost_per_1k_output"`
	CreditsPer1kInput  float64 `json:"credits_per_1k_input"`  // 每1K输入token的积分
	CreditsPer1kOutput float64 `json:"credits_per_1k_output"` // 每1K输出token的积分
}

// CreditPolicyListItem 策略列表项
type CreditPolicyListItem struct {
	CreditPolicy                            // 嵌入策略实体
	EffectiveRateValue float64 `json:"effective_rate"` // 有效汇率（汇率×倍率）
	SchoolName         string  `json:"school_name"`    // 学校名称（仅学校级策略）
}
