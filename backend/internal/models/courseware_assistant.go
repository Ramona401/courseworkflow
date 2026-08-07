package models

// courseware_assistant.go
//
// 本文件定义课件页面教学智能体的编辑态数据协议。
//
// 安全边界：
//   1. 数据库原始JSON字段与浏览器响应模型分开；
//   2. 浏览器响应不包含AI助手完整提示词；
//   3. 上下文预览不包含页面完整HTML或完整教案正文；
//   4. 第一版只允许右下角悬浮助手；
//   5. 核心教学方案使用明确结构体，不使用map[string]any；
//   6. 教学方式由教师选择或服务端生成，不允许改变既有权限与答案保护边界。

import (
	"strings"
	"time"
)

// CoursewareAssistantProtocolVersion 是编辑态基础协议和上下文配置协议版本。
//
// 上下文结构本次未发生变化，因此继续使用v1，避免无必要迁移历史JSON。
// 教学方案自身使用独立的CoursewareAssistantGuidancePlanVersion版本。
const CoursewareAssistantProtocolVersion = "v1"

// ==================== 教学方案版本 ====================

const (
	// CoursewareAssistantGuidancePlanVersionV1 是历史纯苏格拉底式方案版本。
	CoursewareAssistantGuidancePlanVersionV1 = "v1"

	// CoursewareAssistantGuidancePlanVersionV2 增加明确教学方式，仍复用问题链和误区分支通用结构。
	CoursewareAssistantGuidancePlanVersionV2 = "v2"

	// CoursewareAssistantGuidancePlanCurrentVersion 是新建和重新保存方案使用的当前版本。
	CoursewareAssistantGuidancePlanCurrentVersion = CoursewareAssistantGuidancePlanVersionV2
)

// ==================== 教学方式 ====================

const (
	// CoursewareAssistantTeachingModeGuidedReasoning 通过连续小问题引导学生逐步推理。
	CoursewareAssistantTeachingModeGuidedReasoning = "guided_reasoning"

	// CoursewareAssistantTeachingModeExplainBack 让学生用自己的话解释，再针对理解缺口追问。
	CoursewareAssistantTeachingModeExplainBack = "explain_back"

	// CoursewareAssistantTeachingModePredictObserveExplain 引导学生预测、观察并解释现象。
	CoursewareAssistantTeachingModePredictObserveExplain = "predict_observe_explain"

	// CoursewareAssistantTeachingModeWorkedExample 先理解示例，再逐步撤除帮助并独立完成。
	CoursewareAssistantTeachingModeWorkedExample = "worked_example"

	// CoursewareAssistantTeachingModeCoachedPractice 先尝试，再根据错误提供最小必要提示。
	CoursewareAssistantTeachingModeCoachedPractice = "coached_practice"

	// CoursewareAssistantTeachingModeRetrievalCheck 通过短问题检索记忆并识别薄弱点。
	CoursewareAssistantTeachingModeRetrievalCheck = "retrieval_check"

	// CoursewareAssistantTeachingModeCompareContrast 通过比较对象发现共同点、差异与规律。
	CoursewareAssistantTeachingModeCompareContrast = "compare_contrast"

	// CoursewareAssistantTeachingModeEvidenceArgument 要求学生形成观点并用证据支持。
	CoursewareAssistantTeachingModeEvidenceArgument = "evidence_argument"
)

// IsValidCoursewareAssistantTeachingMode 校验教学方式代码。
func IsValidCoursewareAssistantTeachingMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case CoursewareAssistantTeachingModeGuidedReasoning,
		CoursewareAssistantTeachingModeExplainBack,
		CoursewareAssistantTeachingModePredictObserveExplain,
		CoursewareAssistantTeachingModeWorkedExample,
		CoursewareAssistantTeachingModeCoachedPractice,
		CoursewareAssistantTeachingModeRetrievalCheck,
		CoursewareAssistantTeachingModeCompareContrast,
		CoursewareAssistantTeachingModeEvidenceArgument:
		return true
	default:
		return false
	}
}

// NormalizeCoursewareAssistantTeachingMode 规范化教学方式并兼容历史方案。
//
// 历史v1方案没有teaching_mode字段，统一按guided_reasoning解释，保证既有部署行为不变。
// 非空非法值保持原值，交由校验器明确拒绝，禁止静默回退掩盖脏数据。
func NormalizeCoursewareAssistantTeachingMode(mode string) string {
	normalized := strings.TrimSpace(mode)
	if normalized == "" {
		return CoursewareAssistantTeachingModeGuidedReasoning
	}
	return normalized
}

// ==================== 插槽状态 ====================

const (
	CoursewareAssistantSlotStatusActive   = "active"
	CoursewareAssistantSlotStatusDisabled = "disabled"
)

// IsValidCoursewareAssistantSlotStatus 校验插槽状态。
func IsValidCoursewareAssistantSlotStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case CoursewareAssistantSlotStatusActive,
		CoursewareAssistantSlotStatusDisabled:
		return true
	default:
		return false
	}
}

// ==================== 展示方式 ====================

const (
	CoursewareAssistantDisplayModeFloating = "floating"
	CoursewareAssistantPositionBottomRight = "bottom_right"
)

// IsValidCoursewareAssistantDisplayMode 校验MVP展示方式。
func IsValidCoursewareAssistantDisplayMode(mode string) bool {
	return strings.TrimSpace(mode) == CoursewareAssistantDisplayModeFloating
}

// IsValidCoursewareAssistantPosition 校验MVP展示位置。
func IsValidCoursewareAssistantPosition(position string) bool {
	return strings.TrimSpace(position) == CoursewareAssistantPositionBottomRight
}

// ==================== 通用结构化教学方案 ====================

// CoursewareAssistantQuestionStep 是教学互动链中的一个步骤。
//
// 不同教学方式共用这一确定性结构：
//   - guided_reasoning表示连续追问；
//   - explain_back表示解释、检查缺口和重新解释；
//   - worked_example表示观察示例、补全步骤和独立练习；
//   - 其他方式按各自教学动作组织步骤。
//
// ID必须在同一方案内唯一。NextStepID为空表示由服务层按顺序进入下一步。
// HintLadder从弱提示到强提示排列，但不得直接泄露当前学生任务的最终答案。
type CoursewareAssistantQuestionStep struct {
	ID                     string   `json:"id"`
	Prompt                 string   `json:"prompt"`
	TeachingIntent         string   `json:"teaching_intent"`
	ExpectedSignals        []string `json:"expected_signals"`
	HintLadder             []string `json:"hint_ladder"`
	MisconceptionBranchIDs []string `json:"misconception_branch_ids"`
	NextStepID             string   `json:"next_step_id,omitempty"`
	CompletionSignal       string   `json:"completion_signal,omitempty"`
}

// CoursewareAssistantMisconceptionBranch 定义一个受控误区或学习困难分支。
//
// MatchSignals只保存教师可理解的错误特征，不保存模型隐藏推理。
// ReturnToStepID用于完成纠偏后回到主教学互动链。
type CoursewareAssistantMisconceptionBranch struct {
	ID               string   `json:"id"`
	MatchSignals     []string `json:"match_signals"`
	ResponseStrategy string   `json:"response_strategy"`
	FollowUpQuestion string   `json:"follow_up_question"`
	ReturnToStepID   string   `json:"return_to_step_id"`
}

// CoursewareAssistantAnswerLeakPolicy 定义答案保护协议。
//
// 服务层必须要求DirectAnswerAllowed=false。
// WorkedExample模式可以解释发布快照中已经呈现的示例，但仍不得代替学生完成当前任务。
type CoursewareAssistantAnswerLeakPolicy struct {
	DirectAnswerAllowed bool     `json:"direct_answer_allowed"`
	RequireStudentTry   bool     `json:"require_student_try"`
	MaximumHintLevel    int      `json:"maximum_hint_level"`
	ProhibitedBehaviors []string `json:"prohibited_behaviors"`
	SafeClosureGuidance string   `json:"safe_closure_guidance,omitempty"`
}

// CoursewareAssistantGuidancePlan 是可编辑的完整教学方案。
//
// TeachingMode决定运行时采用哪种教学互动方式。
// 历史方案缺少TeachingMode时，服务层按guided_reasoning兼容。
type CoursewareAssistantGuidancePlan struct {
	Version               string                                   `json:"version"`
	TeachingMode          string                                   `json:"teaching_mode"`
	GuidingPrinciples     []string                                 `json:"guiding_principles"`
	QuestionChain         []CoursewareAssistantQuestionStep        `json:"question_chain"`
	MisconceptionBranches []CoursewareAssistantMisconceptionBranch `json:"misconception_branches"`
	ForbiddenBehaviors    []string                                 `json:"forbidden_behaviors"`
	CompletionCriteria    []string                                 `json:"completion_criteria"`
	AnswerLeakPolicy      CoursewareAssistantAnswerLeakPolicy      `json:"answer_leak_policy"`
}

// ==================== 上下文范围配置 ====================

// CoursewareAssistantContextConfig 是教师可编辑的上下文范围。
//
// 第一版只支持固定来源开关，不允许填写任意URL、任意数据库查询或任意工具。
type CoursewareAssistantContextConfig struct {
	Version                    string `json:"version"`
	IncludeVisibleText         bool   `json:"include_visible_text"`
	IncludePagePlan            bool   `json:"include_page_plan"`
	IncludeInteractionEvidence bool   `json:"include_interaction_evidence"`
	IncludeLessonPlanExcerpt   bool   `json:"include_lesson_plan_excerpt"`
	IncludePreviousPageSummary bool   `json:"include_previous_page_summary"`
	IncludeNextPageSummary     bool   `json:"include_next_page_summary"`
	MaxLessonPlanExcerptChars  int    `json:"max_lesson_plan_excerpt_chars"`
}

// DefaultCoursewareAssistantContextConfig 返回MVP默认上下文范围。
func DefaultCoursewareAssistantContextConfig() CoursewareAssistantContextConfig {
	return CoursewareAssistantContextConfig{
		Version:                    CoursewareAssistantProtocolVersion,
		IncludeVisibleText:         true,
		IncludePagePlan:            true,
		IncludeInteractionEvidence: true,
		IncludeLessonPlanExcerpt:   true,
		IncludePreviousPageSummary: true,
		IncludeNextPageSummary:     true,
		MaxLessonPlanExcerptChars:  4000,
	}
}

// ==================== 数据库记录 ====================

// CoursewareAssistantSlot 对应courseware_assistant_slots。
//
// GuidancePlanJSON和ContextConfigJSON是仓储层原始JSON文本，禁止直接返回浏览器。
// 仓储读取后应解码为CoursewareAssistantSlotView中的明确结构体。
type CoursewareAssistantSlot struct {
	ID                string  `json:"-"`
	CoursewareID      string  `json:"-"`
	PageID            string  `json:"-"`
	AssistantID       *string `json:"-"`
	CreatedBy         string  `json:"-"`
	DisplayMode       string  `json:"-"`
	DisplayPosition   string  `json:"-"`
	Title             string  `json:"-"`
	WelcomeMessage    string  `json:"-"`
	TeachingRole      string  `json:"-"`
	LearningObjective string  `json:"-"`

	GuidancePlanJSON  string `json:"-"`
	ContextConfigJSON string `json:"-"`

	Status    string     `json:"-"`
	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// ==================== 浏览器安全响应 ====================

// CoursewareAssistantSlotView 是教师端可安全读取的插槽响应。
//
// 本模型不包含助手完整提示词，也不包含页面完整HTML和教案全文。
type CoursewareAssistantSlotView struct {
	ID                string  `json:"id"`
	CoursewareID      string  `json:"courseware_id"`
	PageID            string  `json:"page_id"`
	AssistantID       *string `json:"assistant_id"`
	AssistantName     string  `json:"assistant_name"`
	AssistantActive   bool    `json:"assistant_active"`
	DisplayMode       string  `json:"display_mode"`
	DisplayPosition   string  `json:"display_position"`
	Title             string  `json:"title"`
	WelcomeMessage    string  `json:"welcome_message"`
	TeachingRole      string  `json:"teaching_role"`
	LearningObjective string  `json:"learning_objective"`

	GuidancePlan  CoursewareAssistantGuidancePlan  `json:"guidance_plan"`
	ContextConfig CoursewareAssistantContextConfig `json:"context_config"`

	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareAssistantSlotListResponse 是课件全部插槽列表响应。
type CoursewareAssistantSlotListResponse struct {
	Slots []*CoursewareAssistantSlotView `json:"slots"`
	Total int                            `json:"total"`
}

// ==================== 写请求 ====================

// CreateCoursewareAssistantSlotRequest 创建页面插槽。
//
// courseware_id、page_id和created_by只能从路由及登录上下文取得，不得由请求正文指定。
type CreateCoursewareAssistantSlotRequest struct {
	AssistantID       *string `json:"assistant_id"`
	Title             string  `json:"title"`
	WelcomeMessage    string  `json:"welcome_message"`
	TeachingRole      string  `json:"teaching_role"`
	LearningObjective string  `json:"learning_objective"`

	GuidancePlan  CoursewareAssistantGuidancePlan  `json:"guidance_plan"`
	ContextConfig CoursewareAssistantContextConfig `json:"context_config"`
}

// UpdateCoursewareAssistantSlotRequest 更新现有插槽。
//
// 不提供courseware_id、page_id和created_by字段，避免越权迁移资源归属。
type UpdateCoursewareAssistantSlotRequest struct {
	AssistantID       *string `json:"assistant_id"`
	Title             string  `json:"title"`
	WelcomeMessage    string  `json:"welcome_message"`
	TeachingRole      string  `json:"teaching_role"`
	LearningObjective string  `json:"learning_objective"`

	GuidancePlan  CoursewareAssistantGuidancePlan  `json:"guidance_plan"`
	ContextConfig CoursewareAssistantContextConfig `json:"context_config"`

	Status string `json:"status"`
}

// GenerateCoursewareAssistantPlanRequest 请求根据当前页面和教师选择的教学方式生成方案。
//
// TeachingMode为空时按历史兼容模式guided_reasoning处理。
// assistant_id和teacher_instruction仍只用于本次方案生成，不代表保存或发布。
type GenerateCoursewareAssistantPlanRequest struct {
	AssistantID        *string `json:"assistant_id"`
	TeachingMode       string  `json:"teaching_mode"`
	TeacherInstruction string  `json:"teacher_instruction"`
}

// CoursewareAssistantPlanResult 是方案生成后的可编辑结果。
type CoursewareAssistantPlanResult struct {
	Title             string `json:"title"`
	WelcomeMessage    string `json:"welcome_message"`
	TeachingRole      string `json:"teaching_role"`
	LearningObjective string `json:"learning_objective"`

	GuidancePlan  CoursewareAssistantGuidancePlan  `json:"guidance_plan"`
	ContextConfig CoursewareAssistantContextConfig `json:"context_config"`
}

// ==================== 教师端上下文预览 ====================

// CoursewareAssistantPagePreview 是教师端可见的页面摘要。
type CoursewareAssistantPagePreview struct {
	PageID         string `json:"page_id"`
	PageNumber     int    `json:"page_number"`
	Title          string `json:"title"`
	Purpose        string `json:"purpose"`
	ContentSummary string `json:"content_summary"`
	VisibleText    string `json:"visible_text,omitempty"`
}

// CoursewareAssistantLessonPlanPreview 是来源教案相关片段的预览。
//
// ExcerptPreview必须经过长度限制，不能返回整份教案。
type CoursewareAssistantLessonPlanPreview struct {
	LessonPlanID   *string `json:"lesson_plan_id"`
	Title          string  `json:"title"`
	ExcerptPreview string  `json:"excerpt_preview"`
	CharacterCount int     `json:"character_count"`
}

// CoursewareAssistantInteractionPreview 是页面互动静态证据摘要。
type CoursewareAssistantInteractionPreview struct {
	DeclaredType         string   `json:"declared_type"`
	ContractOK           bool     `json:"contract_ok"`
	EventCount           int      `json:"event_count"`
	DOMTargetCount       int      `json:"dom_target_count"`
	RiskFlags            []string `json:"risk_flags"`
	ManualReviewRequired bool     `json:"manual_review_required"`
}

// CoursewareAssistantContextPreview 是教师端安全上下文预览。
type CoursewareAssistantContextPreview struct {
	CurrentPage  CoursewareAssistantPagePreview  `json:"current_page"`
	PreviousPage *CoursewareAssistantPagePreview `json:"previous_page,omitempty"`
	NextPage     *CoursewareAssistantPagePreview `json:"next_page,omitempty"`

	LessonPlan  *CoursewareAssistantLessonPlanPreview `json:"lesson_plan,omitempty"`
	Interaction CoursewareAssistantInteractionPreview `json:"interaction"`

	ContextConfig CoursewareAssistantContextConfig `json:"context_config"`
}
