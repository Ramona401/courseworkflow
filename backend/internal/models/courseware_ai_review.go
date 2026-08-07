package models

// courseware_ai_review.go
//
// 课件 AI 审核助手的数据模型。
//
// 与人工 L1/L2 审核的关系：
//   - 本文件中的 AI 审核只是辅助分析，不改变 coursewares.publish_state；
//   - AI 不能自动执行 approved 或 revision；
//   - 人工审核员仍需在课件审核工作台明确提交最终决定。
//
// 长上下文治理：
//   - Session 保存课件、教案、大纲和提示词快照；
//   - Batch 保存每一批页面的输入、结果和连续性账本；
//   - 后批必须继承前批账本，避免前后案例、数字、符号和结论漂移。
//
// 互动逻辑治理：
//   - 页面摘要不能只保存可见文字；
//   - InteractionEvidence 同时保存事件入口、可达函数、DOM 目标、状态变量、
//     CSS 显隐规则、答案暴露风险和人工操作复核标记。
//
// R-02 审核配置治理：
//   - 审核维度和教案参考模式在会话创建时固化为不可变快照；
//   - 旧客户端未提交配置时使用现行兼容预设；
//   - no_lesson 模式必须在服务端真实切断教案、大纲和对齐报告输入；
//   - 配置哈希由数据库根据规范化字段生成，浏览器不能提交可信哈希。

import "time"

// ==================== 会话状态 ====================

const (
	CWAIReviewStatusPending     = "pending"
	CWAIReviewStatusPreparing   = "preparing"
	CWAIReviewStatusReviewing   = "reviewing"
	CWAIReviewStatusAggregating = "aggregating"
	CWAIReviewStatusDone        = "done"
	CWAIReviewStatusFailed      = "failed"
	CWAIReviewStatusCancelled   = "cancelled"
)

// ==================== 审核阶段 ====================

const (
	CWAIReviewStageBaseline    = "baseline"
	CWAIReviewStageIndexing    = "indexing"
	CWAIReviewStageBatch       = "batch_review"
	CWAIReviewStageRiskRecheck = "risk_recheck"
	CWAIReviewStageFinalize    = "finalize"
	CWAIReviewStageDone        = "done"
)

// ==================== 分批状态 ====================

const (
	CWAIReviewBatchPending = "pending"
	CWAIReviewBatchRunning = "running"
	CWAIReviewBatchDone    = "done"
	CWAIReviewBatchFailed  = "failed"
)

// ==================== R-02 审核配置 ====================

const (
	CWAIReviewConfigSchemaVersion = 1

	CWAIReviewDimensionTeachingLogic           = "teaching_logic"
	CWAIReviewDimensionTechnicalImplementation = "technical_implementation"
	CWAIReviewDimensionInteractionExperience   = "interaction_experience"
	CWAIReviewDimensionLessonAlignment         = "lesson_alignment"
	CWAIReviewDimensionAuthenticity            = "authenticity"
	CWAIReviewDimensionKnowledgeAccuracy       = "knowledge_accuracy"
	CWAIReviewDimensionPageReadability         = "page_readability"
	CWAIReviewDimensionOperationalUsability    = "operational_usability"
	CWAIReviewDimensionCustom                  = "custom"
	CWAIReviewLessonReferenceCurrentCompatible = "current_compatible"
	CWAIReviewLessonReferenceStrictAlignment   = "strict_alignment"
	CWAIReviewLessonReferenceLessonIntent      = "lesson_intent"
	CWAIReviewLessonReferenceNoLesson          = "no_lesson"
)

// CoursewareAIReviewDefaultDimensions 返回与 R-02 上线前行为一致的兼容维度集合。
//
// 返回新切片，调用方可以安全修改，不会污染全局状态。
func CoursewareAIReviewDefaultDimensions() []string {
	return []string{
		CWAIReviewDimensionTeachingLogic,
		CWAIReviewDimensionTechnicalImplementation,
		CWAIReviewDimensionInteractionExperience,
		CWAIReviewDimensionLessonAlignment,
		CWAIReviewDimensionAuthenticity,
		CWAIReviewDimensionKnowledgeAccuracy,
		CWAIReviewDimensionPageReadability,
		CWAIReviewDimensionOperationalUsability,
	}
}

// CoursewareAIReviewAllDimensions 返回固定规范顺序下的全部可选维度。
func CoursewareAIReviewAllDimensions() []string {
	result := CoursewareAIReviewDefaultDimensions()
	return append(result, CWAIReviewDimensionCustom)
}

// IsCWAIReviewDimension 判断审核维度代码是否合法。
func IsCWAIReviewDimension(dimension string) bool {
	switch dimension {
	case CWAIReviewDimensionTeachingLogic,
		CWAIReviewDimensionTechnicalImplementation,
		CWAIReviewDimensionInteractionExperience,
		CWAIReviewDimensionLessonAlignment,
		CWAIReviewDimensionAuthenticity,
		CWAIReviewDimensionKnowledgeAccuracy,
		CWAIReviewDimensionPageReadability,
		CWAIReviewDimensionOperationalUsability,
		CWAIReviewDimensionCustom:
		return true
	default:
		return false
	}
}

// IsCWAIReviewLessonReferenceMode 判断教案参考模式是否合法。
func IsCWAIReviewLessonReferenceMode(mode string) bool {
	switch mode {
	case CWAIReviewLessonReferenceCurrentCompatible,
		CWAIReviewLessonReferenceStrictAlignment,
		CWAIReviewLessonReferenceLessonIntent,
		CWAIReviewLessonReferenceNoLesson:
		return true
	default:
		return false
	}
}

// ==================== 数据库实体 ====================

// CoursewareAIReviewSession 对应 courseware_ai_review_sessions。
type CoursewareAIReviewSession struct {
	ID           string  `json:"id"`
	CoursewareID string  `json:"courseware_id"`
	ReviewerID   string  `json:"reviewer_id"`
	AssistantID  *string `json:"assistant_id"`
	LessonPlanID *string `json:"lesson_plan_id"`

	ReviewLevel     int    `json:"review_level"`
	EducationDomain string `json:"education_domain"`
	Subject         string `json:"subject"`
	Grade           string `json:"grade"`

	ReviewConfigSchemaVersion  int    `json:"review_config_schema_version"`
	ReviewDimensionsJSON       string `json:"review_dimensions_json"`
	CustomDimensionDescription string `json:"custom_dimension_description"`
	LessonReferenceMode        string `json:"lesson_reference_mode"`
	ReviewConfigHash           string `json:"review_config_hash"`

	Status         string `json:"status"`
	CurrentStage   string `json:"current_stage"`
	CurrentBatchNo int    `json:"current_batch_no"`
	TotalBatches   int    `json:"total_batches"`

	CoursewareSnapshotHash    string `json:"courseware_snapshot_hash"`
	PagesSnapshotHash         string `json:"pages_snapshot_hash"`
	LessonPlanSnapshotHash    string `json:"lesson_plan_snapshot_hash"`
	CourseOutlineSnapshotHash string `json:"course_outline_snapshot_hash"`

	SystemPromptKey         string `json:"system_prompt_key"`
	SystemPromptVersion     int    `json:"system_prompt_version"`
	SystemPromptSnapshot    string `json:"system_prompt_snapshot"`
	AssistantPromptSnapshot string `json:"assistant_prompt_snapshot"`

	ContextManifestJSON  string `json:"context_manifest_json"`
	BaselineJSON         string `json:"baseline_json"`
	PageIndexJSON        string `json:"page_index_json"`
	ContinuityLedgerJSON string `json:"continuity_ledger_json"`
	FinalReportJSON      string `json:"final_report_json"`

	ModelUsed    string `json:"model_used"`
	TokensUsed   int    `json:"tokens_used"`
	ErrorMessage string `json:"error_message"`

	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// CoursewareAIReviewBatch 对应 courseware_ai_review_batches。
type CoursewareAIReviewBatch struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	BatchNo   int    `json:"batch_no"`

	PageScopeJSON        string `json:"page_scope_json"`
	Status               string `json:"status"`
	InputHash            string `json:"input_hash"`
	ContinuityBeforeJSON string `json:"continuity_before_json"`
	InputManifestJSON    string `json:"input_manifest_json"`
	ResultJSON           string `json:"result_json"`
	ContinuityAfterJSON  string `json:"continuity_after_json"`
	RiskPagesJSON        string `json:"risk_pages_json"`

	ModelUsed    string `json:"model_used"`
	TokensUsed   int    `json:"tokens_used"`
	ErrorMessage string `json:"error_message"`

	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// CoursewareAIReviewMessage 对应 courseware_ai_review_messages。
type CoursewareAIReviewMessage struct {
	ID        string  `json:"id"`
	SessionID string  `json:"session_id"`
	UserID    *string `json:"user_id"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`

	CitationsJSON string `json:"citations_json"`
	TokensUsed    int    `json:"tokens_used"`
	ModelUsed     string `json:"model_used"`

	CreatedAt *time.Time `json:"created_at"`
}

// ==================== 页面互动证据 ====================

// CWAIReviewInteractionEvent 页面中的一个真实操作入口。
type CWAIReviewInteractionEvent struct {
	EventType string `json:"event_type"`
	Trigger   string `json:"trigger"`
	Handler   string `json:"handler"`
	Evidence  string `json:"evidence"`
}

// CWAIReviewReachableFunction 从事件入口可达的脚本函数。
type CWAIReviewReachableFunction struct {
	Name     string `json:"name"`
	Depth    int    `json:"depth"`
	Evidence string `json:"evidence"`
}

// CWAIReviewInteractionEvidence 页面互动静态分析结果。
//
// 该结果描述“代码声明了什么行为”，不是浏览器真实执行结论。
// 解析不充分时必须把 ManualReviewRequired 设为 true。
type CWAIReviewInteractionEvidence struct {
	DeclaredType string `json:"declared_type"`

	ContractOK     bool   `json:"contract_ok"`
	ContractReason string `json:"contract_reason"`
	ContractDetail string `json:"contract_detail"`

	Events             []CWAIReviewInteractionEvent  `json:"events"`
	ReachableFunctions []CWAIReviewReachableFunction `json:"reachable_functions"`
	StateVariables     []string                      `json:"state_variables"`
	DOMTargets         []string                      `json:"dom_targets"`
	CSSStateRules      []string                      `json:"css_state_rules"`

	InitialExposureSignals []string `json:"initial_exposure_signals"`
	RiskFlags              []string `json:"risk_flags"`

	ScriptRuneCount      int  `json:"script_rune_count"`
	ManualReviewRequired bool `json:"manual_review_required"`
}

// CWAIReviewPageDigest 全课件轻量索引中的一页。
//
// VisibleText 负责内容审核；Interaction 负责操作逻辑审核。
// HTMLHash 用于判断页面修改后旧审核结果是否失效。
type CWAIReviewPageDigest struct {
	PageID     string `json:"page_id"`
	PageNumber int    `json:"page_number"`
	Title      string `json:"title"`
	Purpose    string `json:"purpose"`

	ContentSummary    string `json:"content_summary"`
	InteractionType   string `json:"interaction_type"`
	VisualFormat      string `json:"visual_format"`
	MediaRequirements string `json:"media_requirements"`

	PageIndex   string `json:"page_index"`
	VisibleText string `json:"visible_text"`
	HTMLHash    string `json:"html_hash"`

	Interaction CWAIReviewInteractionEvidence `json:"interaction"`
}
