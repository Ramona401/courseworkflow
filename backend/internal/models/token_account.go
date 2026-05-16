package models

// token_account.go — Token/积分系统数据模型
//
// v128 新增（阶段C · Token/积分系统）
// v129 改造（积分机制融合 · 对齐AOCI精确积分计算）：
//   - 删除 TokensPerCredit/TokensToCredits/CreditsToTokens 固定汇率逻辑
//   - 所有余额字段从 int64 改为 float64（对齐数据库 DECIMAL(16,4)）
//   - TokenConsumeRequest 新增 CreditCalculation 字段（精确计算结果）
//   - 积分计算由 CreditPolicyService.CalculateCredits 完成（见 credit_policy.go）
//
// 积分计算公式（v129 新机制）：
//   cost_usd = (input_tokens/1000 × cost_per_1k_input) + (output_tokens/1000 × cost_per_1k_output)
//   credits  = cost_usd × exchange_rate × multiplier
//
// 对应数据库表：
//   token_accounts          — 三级账户
//   token_allocations       — 分配记录
//   token_consumption_logs  — 消费流水
//   token_purchases         — 采购/充值记录
//   token_alert_configs     — 预警配置

import "time"

// ==================== 账户类型常量 ====================

const (
	AccountTypeRegion   = "region"   // 区域账户
	AccountTypeSchool   = "school"   // 学校账户
	AccountTypePersonal = "personal" // 个人账户
)

// AccountTypeNameMap 账户类型中文名
var AccountTypeNameMap = map[string]string{
	AccountTypeRegion:   "区域账户",
	AccountTypeSchool:   "学校账户",
	AccountTypePersonal: "个人账户",
}

// ==================== 账户状态常量 ====================

const (
	AccountStatusActive    = "active"    // 正常
	AccountStatusSuspended = "suspended" // 已冻结
	AccountStatusExpired   = "expired"   // 已过期
)

// AccountStatusNameMap 账户状态中文名
var AccountStatusNameMap = map[string]string{
	AccountStatusActive:    "正常",
	AccountStatusSuspended: "已冻结",
	AccountStatusExpired:   "已过期",
}

// ==================== 分配类型常量 ====================

const (
	AllocationTypeManual  = "manual"  // 手动分配
	AllocationTypeMonthly = "monthly" // 月度自动
	AllocationTypeInitial = "initial" // 初始分配
)

// ==================== 采购类型常量 ====================

const (
	PurchaseTypePurchase = "purchase" // 采购
	PurchaseTypeRecharge = "recharge" // 充值
	PurchaseTypeGift     = "gift"     // 赠送
	PurchaseTypeSystem   = "system"   // 系统补贴
)

// ==================== 数据库实体 ====================

// TokenAccount 积分账户实体（对应 token_accounts 表）
// v129变更：所有金额字段从 int64 → float64（对齐数据库 DECIMAL(16,4)）
type TokenAccount struct {
	ID              string     `json:"id"`
	AccountType     string     `json:"account_type"`      // region/school/personal
	OwnerID         string     `json:"owner_id"`          // 关联实体ID
	ParentAccountID *string    `json:"parent_account_id"` // 上级账户ID
	DisplayName     string     `json:"display_name"`      // 账户名称
	Balance         float64    `json:"balance"`           // 可用余额（积分，DECIMAL(16,4)）
	FrozenAmount    float64    `json:"frozen_amount"`     // 冻结额度
	TotalConsumed   float64    `json:"total_consumed"`    // 历史累计消费
	TotalQuota      float64    `json:"total_quota"`       // 历史累计配额
	MonthlyQuota    float64    `json:"monthly_quota"`     // 月度自动充值额度
	ExpiresAt       *time.Time `json:"expires_at"`        // 配额过期时间
	Status          string     `json:"status"`            // active/suspended/expired
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// TokenAllocation 积分分配记录（对应 token_allocations 表）
// v129变更：Amount 从 int64 → float64
type TokenAllocation struct {
	ID             string     `json:"id"`
	FromAccountID  string     `json:"from_account_id"`  // 来源账户
	ToAccountID    string     `json:"to_account_id"`    // 目标账户
	Amount         float64    `json:"amount"`           // 分配积分数量
	AllocationType string     `json:"allocation_type"`  // manual/monthly/initial
	Memo           string     `json:"memo"`             // 备注
	OperatorID     string     `json:"operator_id"`      // 操作人
	CreatedAt      *time.Time `json:"created_at"`
}

// TokenConsumptionLog 积分消费流水（对应 token_consumption_logs 表）
// v129变更：金额字段从 int64 → float64，新增精确计算字段
type TokenConsumptionLog struct {
	ID            string     `json:"id"`
	AccountID     string     `json:"account_id"`      // 消费账户
	UserID        string     `json:"user_id"`         // 用户ID
	Amount        float64    `json:"amount"`          // 消费积分数量
	BalanceBefore float64    `json:"balance_before"`  // 消费前余额
	BalanceAfter  float64    `json:"balance_after"`   // 消费后余额
	SceneCode     string     `json:"scene_code"`      // AI场景代码
	ModelUsed     string     `json:"model_used"`      // 旧字段，保留兼容
	TokensUsed    int        `json:"tokens_used"`     // 旧字段，保留兼容（总token数）
	LessonPlanID  *string    `json:"lesson_plan_id"`  // 关联教案ID
	PipelineID    *string    `json:"pipeline_id"`     // 关联Pipeline ID
	Memo          string     `json:"memo"`            // 备注
	CreatedAt     *time.Time `json:"created_at"`
	// v129新增：精确积分计算字段（对齐AOCI的ai_model_calls表）
	InputTokens     int     `json:"input_tokens"`      // 输入token数
	OutputTokens    int     `json:"output_tokens"`     // 输出token数
	ModelName       string  `json:"model_name"`        // 模型名称
	Provider        string  `json:"provider"`          // 供应商
	CostUSD         float64 `json:"cost_usd"`          // 美元成本
	ExchangeRate    float64 `json:"exchange_rate"`     // 汇率
	Multiplier      float64 `json:"multiplier"`        // 倍率
	CreditsConsumed float64 `json:"credits_consumed"`  // 精确积分消耗
	LatencyMs       int     `json:"latency_ms"`        // 调用耗时（毫秒）
}

// TokenPurchase 积分采购/充值记录（对应 token_purchases 表）
// v129变更：Amount 从 int64 → float64
type TokenPurchase struct {
	ID           string     `json:"id"`
	AccountID    string     `json:"account_id"`    // 目标账户
	Amount       float64    `json:"amount"`        // 采购积分数量
	PurchaseType string     `json:"purchase_type"` // purchase/recharge/gift/system
	OrderNo      string     `json:"order_no"`      // 订单号
	Memo         string     `json:"memo"`          // 备注
	OperatorID   string     `json:"operator_id"`   // 操作人
	ValidUntil   *time.Time `json:"valid_until"`   // 有效期
	CreatedAt    *time.Time `json:"created_at"`
}

// TokenAlertConfig 积分预警配置（对应 token_alert_configs 表）
type TokenAlertConfig struct {
	ID               string     `json:"id"`
	AccountID        string     `json:"account_id"`        // 关联账户
	WarnThreshold    int        `json:"warn_threshold"`    // 预警阈值（百分比）
	UrgentThreshold  int        `json:"urgent_threshold"`  // 紧急阈值（百分比）
	IsEnabled        bool       `json:"is_enabled"`        // 是否启用
	LastWarnAt       *time.Time `json:"last_warn_at"`      // 上次预警时间
	LastUrgentAt     *time.Time `json:"last_urgent_at"`    // 上次紧急预警时间
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// ==================== 请求结构体 ====================

// CreateTokenAccountRequest 创建积分账户请求
type CreateTokenAccountRequest struct {
	AccountType     string  `json:"account_type"`      // region/school/personal
	OwnerID         string  `json:"owner_id"`          // 关联实体ID
	ParentAccountID *string `json:"parent_account_id"` // 上级账户ID（可选）
	DisplayName     string  `json:"display_name"`      // 账户名称
	MonthlyQuota    float64 `json:"monthly_quota"`     // 月度配额
}

// AllocateTokensRequest 分配积分请求
type AllocateTokensRequest struct {
	ToAccountID string  `json:"to_account_id"` // 目标账户ID
	Amount      float64 `json:"amount"`        // 分配积分数
	Memo        string  `json:"memo"`          // 备注
}

// PurchaseTokensRequest 采购/充值积分请求
type PurchaseTokensRequest struct {
	AccountID    string  `json:"account_id"`    // 目标账户ID
	Amount       float64 `json:"amount"`        // 积分数量
	PurchaseType string  `json:"purchase_type"` // purchase/recharge/gift/system
	OrderNo      string  `json:"order_no"`      // 订单号（可选）
	Memo         string  `json:"memo"`          // 备注
	ValidUntil   *string `json:"valid_until"`   // 有效期（可选，ISO8601格式）
}

// UpdateAlertConfigRequest 更新预警配置请求
type UpdateAlertConfigRequest struct {
	WarnThreshold   int  `json:"warn_threshold"`   // 预警阈值
	UrgentThreshold int  `json:"urgent_threshold"` // 紧急阈值
	IsEnabled       bool `json:"is_enabled"`       // 是否启用
}

// TokenConsumeRequest Token消费内部请求（service层使用，非HTTP）
// v129变更：新增 Calculation 字段承载精确计算结果
type TokenConsumeRequest struct {
	UserID       string             // 用户ID
	SceneCode    string             // AI场景代码
	TokensUsed   int                // 总token数（兼容旧逻辑，当Calculation为nil时使用）
	ModelUsed    string             // 模型名称（兼容旧逻辑）
	LessonPlanID *string            // 关联教案ID
	PipelineID   *string            // 关联Pipeline ID
	Calculation  *CreditCalculation // v129新增：精确积分计算结果（由CreditPolicyService计算）
}

// ==================== 响应结构体 ====================

// TokenAccountListItem 账户列表项（含层级信息）
// v129变更：金额字段从 int64 → float64
type TokenAccountListItem struct {
	ID               string     `json:"id"`
	AccountType      string     `json:"account_type"`
	AccountTypeName  string     `json:"account_type_name"` // 中文名
	OwnerID          string     `json:"owner_id"`
	DisplayName      string     `json:"display_name"`
	Balance          float64    `json:"balance"`
	FrozenAmount     float64    `json:"frozen_amount"`
	AvailableBalance float64    `json:"available_balance"` // balance - frozen_amount
	TotalConsumed    float64    `json:"total_consumed"`
	TotalQuota       float64    `json:"total_quota"`
	MonthlyQuota     float64    `json:"monthly_quota"`
	UsagePercent     float64    `json:"usage_percent"`     // 已用占比
	Status           string     `json:"status"`
	StatusName       string     `json:"status_name"`       // 中文名
	ChildCount       int        `json:"child_count"`       // 子账户数
	ExpiresAt        *time.Time `json:"expires_at"`
	CreatedAt        *time.Time `json:"created_at"`
}

// TokenAccountListResponse 账户列表响应
type TokenAccountListResponse struct {
	Items []*TokenAccountListItem `json:"items"`
	Total int                     `json:"total"`
}

// TokenAccountDetail 账户详情（含预警配置+子账户摘要）
type TokenAccountDetail struct {
	TokenAccount                                  // 嵌入账户实体
	AccountTypeName  string                       `json:"account_type_name"`
	StatusName       string                       `json:"status_name"`
	AvailableBalance float64                      `json:"available_balance"`
	UsagePercent     float64                      `json:"usage_percent"`
	AlertConfig      *TokenAlertConfig            `json:"alert_config"`   // 预警配置（可能为nil）
	ChildAccounts    []*TokenAccountListItem      `json:"child_accounts"` // 子账户列表
}

// AllocationListItem 分配记录列表项
// v129变更：Amount 从 int64 → float64
type AllocationListItem struct {
	ID               string     `json:"id"`
	FromAccountName  string     `json:"from_account_name"` // 来源账户名
	ToAccountName    string     `json:"to_account_name"`   // 目标账户名
	Amount           float64    `json:"amount"`
	AllocationType   string     `json:"allocation_type"`
	Memo             string     `json:"memo"`
	OperatorName     string     `json:"operator_name"`     // 操作人名称
	CreatedAt        *time.Time `json:"created_at"`
}

// AllocationListResponse 分配记录列表响应
type AllocationListResponse struct {
	Items []*AllocationListItem `json:"items"`
	Total int                   `json:"total"`
}

// ConsumptionListItem 消费流水列表项
// v129变更：金额字段从 int64 → float64，新增精确计算字段
type ConsumptionListItem struct {
	ID            string     `json:"id"`
	AccountName   string     `json:"account_name"`    // 账户名称
	UserName      string     `json:"user_name"`       // 用户名
	Amount        float64    `json:"amount"`
	BalanceBefore float64    `json:"balance_before"`
	BalanceAfter  float64    `json:"balance_after"`
	SceneCode     string     `json:"scene_code"`
	ModelUsed     string     `json:"model_used"`
	TokensUsed    int        `json:"tokens_used"`
	Memo          string     `json:"memo"`
	CreatedAt     *time.Time `json:"created_at"`
	// v129新增
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ModelName       string  `json:"model_name"`
	Provider        string  `json:"provider"`
	CostUSD         float64 `json:"cost_usd"`
	ExchangeRate    float64 `json:"exchange_rate"`
	Multiplier      float64 `json:"multiplier"`
	CreditsConsumed float64 `json:"credits_consumed"`
	LatencyMs       int     `json:"latency_ms"`
}

// ConsumptionListResponse 消费流水列表响应
type ConsumptionListResponse struct {
	Items []*ConsumptionListItem `json:"items"`
	Total int                    `json:"total"`
}

// PurchaseListItem 采购记录列表项
// v129变更：Amount 从 int64 → float64
type PurchaseListItem struct {
	ID           string     `json:"id"`
	AccountName  string     `json:"account_name"`  // 账户名称
	Amount       float64    `json:"amount"`
	PurchaseType string     `json:"purchase_type"`
	OrderNo      string     `json:"order_no"`
	Memo         string     `json:"memo"`
	OperatorName string     `json:"operator_name"` // 操作人名称
	ValidUntil   *time.Time `json:"valid_until"`
	CreatedAt    *time.Time `json:"created_at"`
}

// PurchaseListResponse 采购记录列表响应
type PurchaseListResponse struct {
	Items []*PurchaseListItem `json:"items"`
	Total int                 `json:"total"`
}

// TokenOverviewStats Token概览统计（Dashboard用）
// v129变更：金额字段从 int64 → float64
type TokenOverviewStats struct {
	TotalAccounts     int     `json:"total_accounts"`     // 总账户数
	TotalBalance      float64 `json:"total_balance"`      // 全系统总余额
	TotalConsumed     float64 `json:"total_consumed"`     // 全系统总消费
	TotalQuota        float64 `json:"total_quota"`        // 全系统总配额
	TodayConsumed     float64 `json:"today_consumed"`     // 今日消费
	MonthConsumed     float64 `json:"month_consumed"`     // 本月消费
	LowBalanceCount   int     `json:"low_balance_count"`  // 余额预警账户数
	ExpiringSoonCount int     `json:"expiring_soon_count"` // 即将过期账户数
}

// TokenBalanceCheckResult Token余额检查结果（Guard用）
// v129变更：对齐AOCI的HasAvailableCredits三元组设计
type TokenBalanceCheckResult struct {
	HasAccount  bool    `json:"has_account"`   // 是否有账户
	HasBalance  bool    `json:"has_balance"`   // 余额是否 > 0（允许最后一次透支）
	Available   float64 `json:"available"`     // 可用余额
	AccountID   string  `json:"account_id"`    // 账户ID
	Message     string  `json:"message"`       // 提示信息
}
