package models

import (
	"encoding/json"
	"time"
)

// ==================== Pipeline 模型（对应 pipelines 表） ====================

// Pipeline Pipeline主记录（对应 pipelines 表）
// Phase8修复P-02：新增 RejectReason 字段，存储超级审核员退回重审时填写的原因
type Pipeline struct {
	ID               string     `json:"id"`
	CourseCode       string     `json:"course_code"`
	CourseName       string     `json:"course_name"`
	ExternalModuleID *int       `json:"external_module_id"`
	StartedBy        *string    `json:"started_by"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	CurrentStep      string     `json:"current_step"`
	Status           string     `json:"status"`
	AutoMode         bool       `json:"auto_mode"`
	ErrorMessage     string     `json:"error_message"`
	Config           string     `json:"config"`
	ReviewRound      int        `json:"review_round"`
	AssignedTo       *string    `json:"assigned_to"`
	// RejectReason 退回重审原因（超级审核员填写，Phase8新增）
	// 对应数据库 pipelines.reject_reason TEXT 字段
	// 审核员提交定稿被退回时，可在审核页面看到此原因
	RejectReason string `json:"reject_reason"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// PipelineStep Pipeline步骤记录（对应 pipeline_steps 表）
type PipelineStep struct {
	ID           string     `json:"id"`
	PipelineID   string     `json:"pipeline_id"`
	StepName     string     `json:"step_name"`
	StepOrder    int        `json:"step_order"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	DurationMs   int64      `json:"duration_ms"`
	Attempts     int        `json:"attempts"`
	StepData     string     `json:"step_data"`
	ErrorMessage string     `json:"error_message"`
	ModelUsed    string     `json:"model_used"`
	TokensUsed   int        `json:"tokens_used"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// ==================== Pipeline 配置结构体 ====================

// PipelineConfig Pipeline运行时配置
type PipelineConfig struct {
	EvalRounds   int     `json:"eval_rounds"`
	Threshold    float64 `json:"threshold"`
	VarianceWarn float64 `json:"variance_warn"`
	MaxMetaRetry int     `json:"max_meta_retry"`
	MaxTRLoop    int     `json:"max_tr_loop"`
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

// ToJSON 将配置序列化为JSON字符串
func (c *PipelineConfig) ToJSON() string {
	data, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ParsePipelineConfig 从JSON字符串解析Pipeline配置
func ParsePipelineConfig(jsonStr string) *PipelineConfig {
	if jsonStr == "" {
		return DefaultPipelineConfig()
	}
	cfg := &PipelineConfig{}
	if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
		return DefaultPipelineConfig()
	}
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

type DbCheckResult struct {
	CourseCode  string `json:"course_code"`
	CourseID    string `json:"course_id"`
	ModuleID    int    `json:"module_id"`
	HasIndex    bool   `json:"has_index"`
	IndexHash   string `json:"index_hash"`
	PageCount   int    `json:"page_count"`
	TotalLength int    `json:"total_length"`
	IsValid     bool   `json:"is_valid"`
	ErrorDetail string `json:"error_detail"`
}

func (r *DbCheckResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Scanner 步骤数据结构 ====================

type ScannerResult struct {
	RawOutput  string          `json:"raw_output"`
	Parsed     json.RawMessage `json:"parsed"`
	IsValid    bool            `json:"is_valid"`
	ModelUsed  string          `json:"model_used"`
	TokensUsed int             `json:"tokens_used"`
}

func (r *ScannerResult) ToJSON() string {
	if r.Parsed == nil {
		r.Parsed = json.RawMessage("null")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

type ScannerParsed struct {
	Target         string   `json:"target"`
	AbilityTargets []string `json:"ability_targets"`
	GradeStandard  string   `json:"grade_standard"`
	CourseStandard string   `json:"course_standard"`
}

// ==================== Evaluator 步骤数据结构 ====================

type EvalRoundRecord struct {
	ID          string     `json:"id"`
	PipelineID  string     `json:"pipeline_id"`
	RoundNumber int        `json:"round_number"`
	Status      string     `json:"status"`
	Output      string     `json:"output"`
	ScoreTotal  *float64   `json:"score_total"`
	ScoreE1     *float64   `json:"score_e1"`
	ScoreE2     *float64   `json:"score_e2"`
	ScoreE3     *float64   `json:"score_e3"`
	ScoreE4     *float64   `json:"score_e4"`
	Dimensions  string     `json:"dimensions"`
	ModelUsed   string     `json:"model_used"`
	TokensUsed  int        `json:"tokens_used"`
	CreatedAt   *time.Time `json:"created_at"`
}

type EvaluatorResult struct {
	TotalRounds    int       `json:"total_rounds"`
	DoneRounds     int       `json:"done_rounds"`
	FailedRounds   int       `json:"failed_rounds"`
	AvgTotal       float64   `json:"avg_total"`
	AvgE1          float64   `json:"avg_e1"`
	AvgE2          float64   `json:"avg_e2"`
	AvgE3          float64   `json:"avg_e3"`
	AvgE4          float64   `json:"avg_e4"`
	Variance       float64   `json:"variance"`
	VarianceWarn   bool      `json:"variance_warn"`
	RoundScores    []float64 `json:"round_scores"`
	TotalTokens    int       `json:"total_tokens"`
	TotalLatencyMs int64     `json:"total_latency_ms"`
	ModelUsed      string    `json:"model_used"`
}

func (r *EvaluatorResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Meta 步骤数据结构 ====================

type MetaResult struct {
	TotalFinal     float64   `json:"total_final"`
	E1Final        float64   `json:"e1_final"`
	E2Final        float64   `json:"e2_final"`
	E3Final        float64   `json:"e3_final"`
	E4Final        float64   `json:"e4_final"`
	HardConstraint string    `json:"hard_constraint"`
	Grade          string    `json:"grade"`
	Passed         bool      `json:"passed"`
	E1Rounds       []float64 `json:"e1_rounds"`
	E2Rounds       []float64 `json:"e2_rounds"`
	E3Rounds       []float64 `json:"e3_rounds"`
	E4Rounds       []float64 `json:"e4_rounds"`
	Attempt        int       `json:"attempt"`
	TotalRetries   int       `json:"total_retries"`
	RawOutput      string    `json:"raw_output"`
	ModelUsed      string    `json:"model_used"`
	TokensUsed     int       `json:"tokens_used"`
	LatencyMs      int64     `json:"latency_ms"`
}

func (r *MetaResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== 请求结构体 ====================

type CreatePipelineRequest struct {
	CourseCode string          `json:"course_code"`
	AutoMode   *bool           `json:"auto_mode"`
	Config     *PipelineConfig `json:"config"`
}

// ==================== 响应结构体 ====================

type PipelineListResponse struct {
	Pipelines []*PipelineListItem `json:"pipelines"`
	Total     int                 `json:"total"`
}

type PipelineListItem struct {
	ID               string     `json:"id"`
	CourseCode       string     `json:"course_code"`
	CourseName       string     `json:"course_name"`
	ExternalModuleID *int       `json:"external_module_id"`
	CurrentStep      string     `json:"current_step"`
	CurrentStepName  string     `json:"current_step_name"`
	Status           string     `json:"status"`
	StatusName       string     `json:"status_name"`
	AutoMode         bool       `json:"auto_mode"`
	StepsCompleted   int        `json:"steps_completed"`
	StepsTotal       int        `json:"steps_total"`
	ErrorMessage     string     `json:"error_message"`
	StartedBy        *string    `json:"started_by"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	CreatedAt        *time.Time `json:"created_at"`
	ReviewRound      int        `json:"review_round"`
	EvalAvgScore     *float64   `json:"eval_avg_score"`
	MetaScore        *float64   `json:"meta_score"`
	TranslatorScore  *float64   `json:"translator_score"`
	AssignedTo       *string    `json:"assigned_to"`
	AssignedName     string     `json:"assigned_name"`
}

// PipelineDetailResponse Pipeline详情响应
// Phase8修复P-02：新增 RejectReason 字段，前端审核页面可展示退回原因
type PipelineDetailResponse struct {
	ID               string          `json:"id"`
	CourseCode       string          `json:"course_code"`
	CourseName       string          `json:"course_name"`
	ExternalModuleID *int            `json:"external_module_id"`
	CurrentStep      string          `json:"current_step"`
	CurrentStepName  string          `json:"current_step_name"`
	Status           string          `json:"status"`
	StatusName       string          `json:"status_name"`
	AutoMode         bool            `json:"auto_mode"`
	Config           *PipelineConfig `json:"config"`
	ErrorMessage     string          `json:"error_message"`
	StartedBy        *string         `json:"started_by"`
	StartedAt        *time.Time      `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
	CreatedAt        *time.Time      `json:"created_at"`
	UpdatedAt        *time.Time      `json:"updated_at"`
	ReviewRound      int             `json:"review_round"`
	AssignedTo       *string         `json:"assigned_to"`
	AssignedName     string          `json:"assigned_name"`
	// RejectReason 最近一次退回重审的原因（空字符串表示未被退回或未填写原因）
	RejectReason string          `json:"reject_reason"`
	Steps        []*StepListItem `json:"steps"`
}

type StepListItem struct {
	ID           string     `json:"id"`
	StepName     string     `json:"step_name"`
	StepNameCN   string     `json:"step_name_cn"`
	StepOrder    int        `json:"step_order"`
	Status       string     `json:"status"`
	StatusName   string     `json:"status_name"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	DurationMs   int64      `json:"duration_ms"`
	Attempts     int        `json:"attempts"`
	ModelUsed    string     `json:"model_used"`
	TokensUsed   int        `json:"tokens_used"`
	ErrorMessage string     `json:"error_message"`
	HasData      bool       `json:"has_data"`
}

type StepDetailResponse struct {
	ID           string          `json:"id"`
	PipelineID   string          `json:"pipeline_id"`
	StepName     string          `json:"step_name"`
	StepNameCN   string          `json:"step_name_cn"`
	StepOrder    int             `json:"step_order"`
	Status       string          `json:"status"`
	StatusName   string          `json:"status_name"`
	StartedAt    *time.Time      `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at"`
	DurationMs   int64           `json:"duration_ms"`
	Attempts     int             `json:"attempts"`
	StepData     json.RawMessage `json:"step_data"`
	ErrorMessage string          `json:"error_message"`
	ModelUsed    string          `json:"model_used"`
	TokensUsed   int             `json:"tokens_used"`
	CreatedAt    *time.Time      `json:"created_at"`
	UpdatedAt    *time.Time      `json:"updated_at"`
}

// ==================== Pipeline 常量 ====================

const (
	PipelineStatusPending         = "pending"
	PipelineStatusRunning         = "running"
	PipelineStatusReviewQueue     = "review_queue"
	PipelineStatusPendingFinalize = "pending_finalize" // P7新增：提交定稿待超级审核员确认
	PipelineStatusFinalized       = "finalized"
	PipelineStatusNeedsHuman      = "needs_human"
	PipelineStatusFailed          = "failed"
	PipelineStatusCancelled       = "cancelled"
	PipelineStatusVerified        = "verified"
	PipelineStatusVerifyFailed    = "verify_failed"
)

const (
	StepStatusPending = "pending"
	StepStatusRunning = "running"
	StepStatusDone    = "done"
	StepStatusFailed  = "failed"
	StepStatusSkipped = "skipped"
)

const (
	StepDbCheck    = "dbCheck"
	StepScanner    = "scanner"
	StepEvaluator  = "evaluator"
	StepMeta       = "meta"
	StepTranslator = "translator"
	StepGenerator  = "generator"
	StepReview     = "review"
	StepVerify     = "verify"
)

var StepDefinitions = []struct {
	Name  string
	Order int
}{
	{StepDbCheck, 1},
	{StepScanner, 2},
	{StepEvaluator, 3},
	{StepMeta, 4},
	{StepTranslator, 5},
	{StepGenerator, 6},
	{StepReview, 7},
	{StepVerify, 8},
}

var StepNameMap = map[string]string{
	StepDbCheck:    "数据库检查",
	StepScanner:    "扫描定位",
	StepEvaluator:  "评估打分",
	StepMeta:       "元评估仲裁",
	StepTranslator: "翻译转换+审核",
	StepGenerator:  "页面生成",
	StepReview:     "人工终审",
	StepVerify:     "验收评估",
}

var PipelineStatusNameMap = map[string]string{
	PipelineStatusPending:         "待启动",
	PipelineStatusRunning:         "运行中",
	PipelineStatusReviewQueue:     "等待审核",
	PipelineStatusPendingFinalize: "待确认定稿", // P7新增
	PipelineStatusFinalized:       "已定稿",
	PipelineStatusNeedsHuman:      "需人工干预",
	PipelineStatusFailed:          "失败",
	PipelineStatusCancelled:       "已取消",
	PipelineStatusVerified:        "验收通过",
	PipelineStatusVerifyFailed:    "验收未通过",
}

var StepStatusNameMap = map[string]string{
	StepStatusPending: "待执行",
	StepStatusRunning: "执行中",
	StepStatusDone:    "已完成",
	StepStatusFailed:  "失败",
	StepStatusSkipped: "已跳过",
}

const TotalSteps = 8
const MinIndexLength = 50

// ==================== Translator+Reviewer 步骤数据结构 ====================

type TranslatorRoundRecord struct {
	Round           int     `json:"round"`
	TransOutput     string  `json:"trans_output,omitempty"`
	TransTokens     int     `json:"trans_tokens"`
	TransLatencyMs  int64   `json:"trans_latency_ms"`
	TransError      string  `json:"trans_error,omitempty"`
	ReviewOutput    string  `json:"review_output,omitempty"`
	ReviewTokens    int     `json:"review_tokens"`
	ReviewLatencyMs int64   `json:"review_latency_ms"`
	ReviewError     string  `json:"review_error,omitempty"`
	Score           float64 `json:"score"`
	QualityGate     string  `json:"quality_gate,omitempty"`
	HardCheck       string  `json:"hard_check,omitempty"`
	Grade           string  `json:"grade,omitempty"`
	E1              float64 `json:"e1"`
	E2              float64 `json:"e2"`
	E3              float64 `json:"e3"`
	E4              float64 `json:"e4"`
	Passed          bool    `json:"passed"`
}

type TranslatorResult struct {
	MaxLoops          int                      `json:"max_loops"`
	Threshold         float64                  `json:"threshold"`
	Passed            bool                     `json:"passed"`
	FinalScore        float64                  `json:"final_score"`
	FinalQualityGate  string                   `json:"final_quality_gate,omitempty"`
	FinalGrade        string                   `json:"final_grade,omitempty"`
	FinalRound        int                      `json:"final_round"`
	FinalTransOutput  string                   `json:"final_trans_output,omitempty"`
	FinalReviewOutput string                   `json:"final_review_output,omitempty"`
	Rounds            []*TranslatorRoundRecord `json:"rounds"`
	TotalTokens       int                      `json:"total_tokens"`
	TotalLatencyMs    int64                    `json:"total_latency_ms"`
	ModelUsed         string                   `json:"model_used"`
}

func (r *TranslatorResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Generator 步骤数据结构 ====================

type GeneratorPageRecord struct {
	PageNumber   int    `json:"page_number"`
	PageTitle    string `json:"page_title"`
	Operation    string `json:"operation"`
	LessonID     int    `json:"lesson_id,omitempty"`
	HasOrigHTML  bool   `json:"has_orig_html"`
	OrigHTMLLen  int    `json:"orig_html_len"`
	GenHTMLLen   int    `json:"gen_html_len"`
	MergeSources []int  `json:"merge_sources,omitempty"`
	TokensUsed   int    `json:"tokens_used"`
	LatencyMs    int64  `json:"latency_ms"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type GeneratorResult struct {
	TotalPages     int                    `json:"total_pages"`
	KeptPages      int                    `json:"kept_pages"`
	ModifiedPages  int                    `json:"modified_pages"`
	CreatedPages   int                    `json:"created_pages"`
	MergedPages    int                    `json:"merged_pages"`
	DeletedPages   int                    `json:"deleted_pages"`
	FailedPages    int                    `json:"failed_pages"`
	Pages          []*GeneratorPageRecord `json:"pages"`
	TotalTokens    int                    `json:"total_tokens"`
	TotalLatencyMs int64                  `json:"total_latency_ms"`
	ModelUsed      string                 `json:"model_used"`
}

func (r *GeneratorResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Verify 步骤数据结构 ====================

type VerifyResult struct {
	GeneratedIndex string  `json:"generated_index"`
	EvalScore      float64 `json:"eval_score"`
	EvalOutput     string  `json:"eval_output"`
	EvalE1         float64 `json:"eval_e1"`
	EvalE2         float64 `json:"eval_e2"`
	EvalE3         float64 `json:"eval_e3"`
	EvalE4         float64 `json:"eval_e4"`
	Passed         bool    `json:"passed"`
	ReviewRound    int     `json:"review_round"`
	ModelUsed      string  `json:"model_used"`
	TokensUsed     int     `json:"tokens_used"`
	LatencyMs      int64   `json:"latency_ms"`
}

func (r *VerifyResult) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== Generator 页面操作常量 ====================

const (
	PageOpKeep   = "keep"
	PageOpModify = "modify"
	PageOpCreate = "create"
	PageOpMerge  = "merge"
	PageOpDelete = "delete"
)

// PageOperation Translator输出解析后的页面操作
// v35修复：新增 OriginalPageNumber 字段
// 当Translator重排页码时（如"P05=原P04，页码顺延"），PageNumber=5 但 OriginalPageNumber=4
// Generator用 OriginalPageNumber 去 pageLessonMap 查找正确的 lesson_id 和原版HTML
type PageOperation struct {
	PageNumber         int    `json:"page_number"`
	OriginalPageNumber int    `json:"original_page_number"` // v35新增：原始页码（从标题"原Pxx"解析，0表示与PageNumber相同）
	Operation          string `json:"operation"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	MergeSources       []int  `json:"merge_sources,omitempty"`
}

// GetOrigPageNum 获取实际的原始页码（v35新增）
// 如果 OriginalPageNumber > 0，返回它；否则返回 PageNumber
func (op *PageOperation) GetOrigPageNum() int {
	if op.OriginalPageNumber > 0 {
		return op.OriginalPageNumber
	}
	return op.PageNumber
}
