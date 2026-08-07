package models

// context_receipt.go — 备课上下文回执模型
//
// 设计目标：
//   1. 区分“已关联”与“本轮实际读取”；
//   2. 回执由确定性代码生成，不让AI自行声明读了什么；
//   3. 回执写入ConversationMessage.Metadata，实时SSE、断线补齐和历史恢复共用；
//   4. 只返回名称、数量和状态，不返回提示词正文、教材原文或组件完整指引；
//   5. 区分“课程大纲来源是否已关联”和“知识脉络快照是否实际注入”。

const ContextReceiptVersion = "v1"

// ContextReceiptStatus 单项上下文的实际处理状态。
const (
	ContextReceiptLoaded        = "loaded"
	ContextReceiptNotLinked     = "not_linked"
	ContextReceiptNotApplicable = "not_applicable"
	ContextReceiptDeferred      = "deferred"
	ContextReceiptSuperseded    = "superseded"
	ContextReceiptUnavailable   = "unavailable"
	ContextReceiptForbidden     = "forbidden"
	ContextReceiptExplicitNone  = "explicit_none"
	ContextReceiptNotFound      = "not_found"
)

// ContextReceipt 一轮AI回复实际使用的上下文摘要。
type ContextReceipt struct {
	Version       string `json:"version"`
	StageCode     string `json:"stage_code"`

	Assistant  *AssistantContextReceipt  `json:"assistant,omitempty"`
	Recipe     *MaterialContextReceipt   `json:"recipe,omitempty"`
	Components *ComponentsContextReceipt `json:"components,omitempty"`

	Textbook     *MaterialContextReceipt `json:"textbook,omitempty"`
	UnitPlan     *MaterialContextReceipt `json:"unit_plan,omitempty"`
	CourseOutline *MaterialContextReceipt `json:"course_outline,omitempty"`

	// KnowledgeLineage记录本轮是否实际注入教师确认后生成的
	// active知识脉络短版上下文。
	//
	// CourseOutline只表示权威来源是否精确关联和当前是否可读取；
	// 后续阶段不再把课程大纲全文作为普通提示词内容重复注入。
	KnowledgeLineage *MaterialContextReceipt `json:"knowledge_lineage,omitempty"`

	ClassProfile *MaterialContextReceipt `json:"class_profile,omitempty"`
	RefMaterial  *MaterialContextReceipt `json:"ref_material,omitempty"`

	SystemPromptRunes int `json:"system_prompt_runes,omitempty"`
}

// AssistantContextReceipt 记录本轮助手如何被解析，以及是否真正加载。
type AssistantContextReceipt struct {
	Status        string `json:"status"`
	SelectionMode string `json:"selection_mode,omitempty"`
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	Source        string `json:"source,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// MaterialContextReceipt 适用于配方、教材和各类挂载材料。
type MaterialContextReceipt struct {
	Status          string   `json:"status"`
	SelectionMode   string   `json:"selection_mode,omitempty"`
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Count           int      `json:"count,omitempty"`
	ReadableCount   int      `json:"readable_count,omitempty"`
	UnreadableCount int      `json:"unreadable_count,omitempty"`
	CharacterCount  int      `json:"character_count,omitempty"`
	Titles          []string `json:"titles,omitempty"`
}

// ComponentsContextReceipt 记录第3层专业组件的真实来源和选中结果。
type ComponentsContextReceipt struct {
	Status         string                        `json:"status"`
	SelectionMode  string                        `json:"selection_mode,omitempty"`
	CandidateCount int                           `json:"candidate_count,omitempty"`
	Reranked       bool                          `json:"reranked,omitempty"`
	Items          []ComponentContextReceiptItem `json:"items,omitempty"`
	Reason         string                        `json:"reason,omitempty"`
}

// ComponentContextReceiptItem 仅返回老师可理解的组件摘要。
type ComponentContextReceiptItem struct {
	ID           string  `json:"id"`
	LibraryType  string  `json:"library_type"`
	LibraryName  string  `json:"library_name"`
	DisplayLabel string  `json:"display_label"`
	QualityScore float64 `json:"quality_score,omitempty"`
}

// AssistantPromptResolution 是服务层内部的助手解析结果。
// Prompt只用于后端注入，不会通过JSON返回前端。
type AssistantPromptResolution struct {
	Prompt  string
	Receipt *AssistantContextReceipt
}
