package models

import (
	"encoding/json"
	"time"
)

// ==================== Pipeline 模型（对应 pipelines 表） ====================

// Pipeline Pipeline主记录（对应 pipelines 表）
// 一个Pipeline对应一门课程的完整7步评估流程
type Pipeline struct {
	ID               string     `json:"id"`                 // UUID主键
	CourseCode       string     `json:"course_code"`        // 课程编号（如 G1-01）
	CourseName       string     `json:"course_name"`        // 课程名称
	ExternalModuleID *int       `json:"external_module_id"` // 外部模块ID
	StartedBy        *string    `json:"started_by"`         // 发起者用户ID
	StartedAt        *time.Time `json:"started_at"`         // 启动时间
	CompletedAt      *time.Time `json:"completed_at"`       // 完成时间
	CurrentStep      string     `json:"current_step"`       // 当前步骤名称
	Status           string     `json:"status"`             // Pipeline状态
	AutoMode         bool       `json:"auto_mode"`          // 是否自动模式（自动推进到下一步）
	ErrorMessage     string     `json:"error_message"`      // 错误信息
	Config           string     `json:"config"`             // 配置JSON（JSONB存储）
	CreatedAt        *time.Time `json:"created_at"`         // 创建时间
	UpdatedAt        *time.Time `json:"updated_at"`         // 更新时间
}

// PipelineStep Pipeline步骤记录（对应 pipeline_steps 表）
// 每个Pipeline有7个步骤，每步独立记录状态和数据
type PipelineStep struct {
	ID           string     `json:"id"`            // UUID主键
	PipelineID   string     `json:"pipeline_id"`   // 关联Pipeline ID
	StepName     string     `json:"step_name"`     // 步骤名称（如 dbCheck/scanner/evaluator等）
	StepOrder    int        `json:"step_order"`    // 步骤序号（1-7）
	Status       string     `json:"status"`        // 步骤状态
	StartedAt    *time.Time `json:"started_at"`    // 开始时间
	CompletedAt  *time.Time `json:"completed_at"`  // 完成时间
	DurationMs   int64      `json:"duration_ms"`   // 耗时（毫秒）
	Attempts     int        `json:"attempts"`      // 尝试次数
	StepData     string     `json:"step_data"`     // 步骤输出数据（JSONB存储）
	ErrorMessage string     `json:"error_message"` // 错误信息
	ModelUsed    string     `json:"model_used"`    // 使用的AI模型（AI步骤有值）
	TokensUsed   int        `json:"tokens_used"`   // 消耗的Token数（AI步骤有值）
	CreatedAt    *time.Time `json:"created_at"`    // 创建时间
	UpdatedAt    *time.Time `json:"updated_at"`    // 更新时间
}

// ==================== Pipeline 配置结构体 ====================

// PipelineConfig Pipeline运行时配置（存入pipelines.config JSONB字段）
// 控制Pipeline执行行为的参数集合
type PipelineConfig struct {
	EvalRounds   int     `json:"eval_rounds"`    // Evaluator评估轮数（默认3）
	Threshold    float64 `json:"threshold"`      // 达标分数线（默认9.0）
	VarianceWarn float64 `json:"variance_warn"`  // 方差警告阈值（默认1.5）
	MaxMetaRetry int     `json:"max_meta_retry"` // Meta步骤最大重试次数（默认3）
	MaxTRLoop    int     `json:"max_tr_loop"`    // Translator-Reviewer最大循环次数（默认3）
}

// DefaultPipelineConfig 返回默认Pipeline配置
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		EvalRounds:   3,
		Threshold:    9.0,
		VarianceWarn: 1.5,
		MaxMetaRetry: 3,
		MaxTRLoop:    3,
	}
}

// ToJSON 将配置序列化为JSON字符串（存入数据库JSONB字段）
func (c *PipelineConfig) ToJSON() string {
	data, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ParsePipelineConfig 从JSON字符串解析Pipeline配置
// 解析失败时返回默认配置，保证不会因配置异常导致Pipeline无法运行
func ParsePipelineConfig(jsonStr string) *PipelineConfig {
	if jsonStr == "" {
		return DefaultPipelineConfig()
	}
	cfg := &PipelineConfig{}
	if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
		return DefaultPipelineConfig()
	}
	// 校验并修正异常值：确保关键参数不为零
	if cfg.EvalRounds <= 0 {
		cfg.EvalRounds = 3
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 9.0
	}
	if cfg.VarianceWarn <= 0 {
		cfg.VarianceWarn = 1.5
	}
	if cfg.MaxMetaRetry <= 0 {
		cfg.MaxMetaRetry = 3
	}
	if cfg.MaxTRLoop <= 0 {
		cfg.MaxTRLoop = 3
	}
	return cfg
}

// ==================== dbCheck 步骤数据结构 ====================

// DbCheckResult dbCheck步骤的输出数据（存入step_data JSONB字段）
// 验证课程索引的存在性和有效性
type DbCheckResult struct {
	CourseCode   string `json:"course_code"`   // 课程编号
	CourseID     string `json:"course_id"`     // 课程UUID
	ModuleID     int    `json:"module_id"`     // 外部模块ID
	HasIndex     bool   `json:"has_index"`     // 是否存在索引
	IndexHash    string `json:"index_hash"`    // 索引SHA-256校验码
	PageCount    int    `json:"page_count"`    // 索引页面数
	TotalLength  int    `json:"total_length"`  // 索引总字符数
	IsValid      bool   `json:"is_valid"`      // 索引是否有效（存在+长度>50+hash一致）
	ErrorDetail  string `json:"error_detail"`  // 验证失败时的详细原因
}

// ToJSON 将dbCheck结果序列化为JSON字符串
func (r *DbCheckResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Scanner 步骤数据结构（P4-2使用，此处预定义） ====================

// ScannerResult Scanner步骤的输出数据（存入step_data JSONB字段）
// 从AI返回的JSON中解析课程定位信息
type ScannerResult struct {
	RawOutput string          `json:"raw_output"` // AI原始输出文本
	Parsed    json.RawMessage `json:"parsed"`     // 解析后的JSON对象（target/ability_targets/grade_standard/course_standard）
	IsValid   bool            `json:"is_valid"`   // 是否成功解析出有效JSON
}

// ==================== 请求结构体 ====================

// CreatePipelineRequest 创建Pipeline请求
type CreatePipelineRequest struct {
	CourseCode string          `json:"course_code"` // 课程编号（必填）
	AutoMode   *bool           `json:"auto_mode"`   // 是否自动模式（可选，默认true）
	Config     *PipelineConfig `json:"config"`      // 自定义配置（可选，不填用默认值）
}

// ==================== 响应结构体 ====================

// PipelineListResponse Pipeline列表响应
type PipelineListResponse struct {
	Pipelines []*PipelineListItem `json:"pipelines"` // Pipeline列表
	Total     int                 `json:"total"`     // 总数
}

// PipelineListItem Pipeline列表单条（含步骤进度摘要）
type PipelineListItem struct {
	ID               string     `json:"id"`                 // UUID
	CourseCode       string     `json:"course_code"`        // 课程编号
	CourseName       string     `json:"course_name"`        // 课程名称
	ExternalModuleID *int       `json:"external_module_id"` // 外部模块ID
	CurrentStep      string     `json:"current_step"`       // 当前步骤
	CurrentStepName  string     `json:"current_step_name"`  // 当前步骤中文名
	Status           string     `json:"status"`             // 状态
	StatusName       string     `json:"status_name"`        // 状态中文名
	AutoMode         bool       `json:"auto_mode"`          // 自动模式
	StepsCompleted   int        `json:"steps_completed"`    // 已完成步骤数
	StepsTotal       int        `json:"steps_total"`        // 总步骤数（7）
	ErrorMessage     string     `json:"error_message"`      // 错误信息
	StartedBy        *string    `json:"started_by"`         // 发起者ID
	StartedAt        *time.Time `json:"started_at"`         // 启动时间
	CompletedAt      *time.Time `json:"completed_at"`       // 完成时间
	CreatedAt        *time.Time `json:"created_at"`         // 创建时间
}

// PipelineDetailResponse Pipeline详情响应（含完整步骤列表）
type PipelineDetailResponse struct {
	ID               string              `json:"id"`                 // UUID
	CourseCode       string              `json:"course_code"`        // 课程编号
	CourseName       string              `json:"course_name"`        // 课程名称
	ExternalModuleID *int                `json:"external_module_id"` // 外部模块ID
	CurrentStep      string              `json:"current_step"`       // 当前步骤
	CurrentStepName  string              `json:"current_step_name"`  // 当前步骤中文名
	Status           string              `json:"status"`             // 状态
	StatusName       string              `json:"status_name"`        // 状态中文名
	AutoMode         bool                `json:"auto_mode"`          // 自动模式
	Config           *PipelineConfig     `json:"config"`             // 运行配置
	ErrorMessage     string              `json:"error_message"`      // 错误信息
	StartedBy        *string             `json:"started_by"`         // 发起者ID
	StartedAt        *time.Time          `json:"started_at"`         // 启动时间
	CompletedAt      *time.Time          `json:"completed_at"`       // 完成时间
	CreatedAt        *time.Time          `json:"created_at"`         // 创建时间
	UpdatedAt        *time.Time          `json:"updated_at"`         // 更新时间
	Steps            []*StepListItem     `json:"steps"`              // 步骤列表
}

// StepListItem 步骤列表单条
type StepListItem struct {
	ID           string     `json:"id"`            // UUID
	StepName     string     `json:"step_name"`     // 步骤标识
	StepNameCN   string     `json:"step_name_cn"`  // 步骤中文名
	StepOrder    int        `json:"step_order"`    // 步骤序号
	Status       string     `json:"status"`        // 状态
	StatusName   string     `json:"status_name"`   // 状态中文名
	StartedAt    *time.Time `json:"started_at"`    // 开始时间
	CompletedAt  *time.Time `json:"completed_at"`  // 完成时间
	DurationMs   int64      `json:"duration_ms"`   // 耗时毫秒
	Attempts     int        `json:"attempts"`      // 尝试次数
	ModelUsed    string     `json:"model_used"`    // 使用的模型
	TokensUsed   int        `json:"tokens_used"`   // Token消耗
	ErrorMessage string     `json:"error_message"` // 错误信息
	HasData      bool       `json:"has_data"`      // 是否有输出数据
}

// StepDetailResponse 步骤详情响应（含完整step_data）
type StepDetailResponse struct {
	ID           string          `json:"id"`            // UUID
	PipelineID   string          `json:"pipeline_id"`   // Pipeline ID
	StepName     string          `json:"step_name"`     // 步骤标识
	StepNameCN   string          `json:"step_name_cn"`  // 步骤中文名
	StepOrder    int             `json:"step_order"`    // 步骤序号
	Status       string          `json:"status"`        // 状态
	StatusName   string          `json:"status_name"`   // 状态中文名
	StartedAt    *time.Time      `json:"started_at"`    // 开始时间
	CompletedAt  *time.Time      `json:"completed_at"`  // 完成时间
	DurationMs   int64           `json:"duration_ms"`   // 耗时毫秒
	Attempts     int             `json:"attempts"`      // 尝试次数
	StepData     json.RawMessage `json:"step_data"`     // 步骤输出数据（完整JSON）
	ErrorMessage string          `json:"error_message"` // 错误信息
	ModelUsed    string          `json:"model_used"`    // 使用的模型
	TokensUsed   int             `json:"tokens_used"`   // Token消耗
	CreatedAt    *time.Time      `json:"created_at"`    // 创建时间
	UpdatedAt    *time.Time      `json:"updated_at"`    // 更新时间
}

// ==================== Pipeline 常量 ====================

// Pipeline 状态常量
const (
	PipelineStatusPending    = "pending"      // 待启动
	PipelineStatusRunning    = "running"      // 运行中
	PipelineStatusReviewQueue = "review_queue" // 等待人工审核
	PipelineStatusFinalized  = "finalized"    // 已定稿
	PipelineStatusNeedsHuman = "needs_human"  // 需要人工干预
	PipelineStatusFailed     = "failed"       // 失败
	PipelineStatusCancelled  = "cancelled"    // 已取消
)

// Pipeline 步骤状态常量
const (
	StepStatusPending = "pending" // 待执行
	StepStatusRunning = "running" // 执行中
	StepStatusDone    = "done"    // 已完成
	StepStatusFailed  = "failed"  // 失败
	StepStatusSkipped = "skipped" // 已跳过
)

// Pipeline 7步名称常量（严格按执行顺序）
const (
	StepDbCheck    = "dbCheck"    // 步骤1：数据库检查（验证索引）
	StepScanner    = "scanner"    // 步骤2：扫描定位（Prompt A）
	StepEvaluator  = "evaluator"  // 步骤3：评估打分（Prompt B × N轮）
	StepMeta       = "meta"       // 步骤4：元评估仲裁（Prompt E）
	StepTranslator = "translator" // 步骤5：翻译转换+审核（Prompt C + D）
	StepGenerator  = "generator"  // 步骤6：页面生成（Prompt F × 每页）
	StepReview     = "review"     // 步骤7：人工终审
)

// StepDefinitions 7步定义列表（有序，step_order从1开始）
// 用于创建Pipeline时批量插入pipeline_steps记录
var StepDefinitions = []struct {
	Name  string // 步骤标识
	Order int    // 步骤序号
}{
	{StepDbCheck, 1},
	{StepScanner, 2},
	{StepEvaluator, 3},
	{StepMeta, 4},
	{StepTranslator, 5},
	{StepGenerator, 6},
	{StepReview, 7},
}

// StepNameMap 步骤标识→中文名映射
var StepNameMap = map[string]string{
	StepDbCheck:    "数据库检查",
	StepScanner:    "扫描定位",
	StepEvaluator:  "评估打分",
	StepMeta:       "元评估仲裁",
	StepTranslator: "翻译转换+审核",
	StepGenerator:  "页面生成",
	StepReview:     "人工终审",
}

// PipelineStatusNameMap Pipeline状态→中文名映射
var PipelineStatusNameMap = map[string]string{
	PipelineStatusPending:    "待启动",
	PipelineStatusRunning:    "运行中",
	PipelineStatusReviewQueue: "等待审核",
	PipelineStatusFinalized:  "已定稿",
	PipelineStatusNeedsHuman: "需人工干预",
	PipelineStatusFailed:     "失败",
	PipelineStatusCancelled:  "已取消",
}

// StepStatusNameMap 步骤状态→中文名映射
var StepStatusNameMap = map[string]string{
	StepStatusPending: "待执行",
	StepStatusRunning: "执行中",
	StepStatusDone:    "已完成",
	StepStatusFailed:  "失败",
	StepStatusSkipped: "已跳过",
}

// TotalSteps Pipeline总步骤数
const TotalSteps = 7

// MinIndexLength 索引最小有效长度（字符数，低于此值认为无效）
const MinIndexLength = 50
