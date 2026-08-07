package models

// courseware_ai_review_run.go
//
// 课件 AI 审核分批执行的数据协议。
//
// 设计要求：
//   - 每个问题必须能回到具体页码和页面证据；
//   - 内容问题与互动操作问题使用同一套结构化协议；
//   - 静态脚本分析不能确认的行为必须标记人工操作复核；
//   - 每批返回更新后的连续性账本，供下一批顺序继承；
//   - AI 结果只辅助人工审核，不直接改变课件审核决定；
//   - 教师默认展示必须使用固化的教师视图快照，不直接依赖技术证据。

// ==================== 教师展示快照 ====================

// CWAIReviewTeacherViewSnapshot 是一条审核发现面向教师的固化表达。
//
// 该快照与原始技术证据分离：
//   - 教师界面默认只使用本结构中的字段；
//   - 原始页面、代码、连续性证据仍由后端保留；
//   - TeacherContext 只表示教师补充意图，初次检测时应为空；
//   - InternalExecutionPlan 不属于本结构，不能进入教师默认响应。
//
// 新记录在AI审核结果生成和后端归一化阶段形成快照；
// 历史记录由后端使用确定性规则安全降级，不在页面打开时临时调用AI改写。
type CWAIReviewTeacherViewSnapshot struct {
	// TeacherTitle 用一句教师语言说明页面或教学环节的问题。
	TeacherTitle string `json:"teacher_title"`

	// WhatHappened 描述教师能够直接观察到的现象。
	WhatHappened string `json:"what_happened"`

	// TeachingImpact 说明对讲解、互动、理解、评价、操作或可读性的影响。
	TeachingImpact string `json:"teaching_impact"`

	// ImprovementGoal 描述希望达到的教学效果，不描述平台实现细节。
	ImprovementGoal string `json:"improvement_goal"`

	// AcceptanceChecks 必须包含2至5条教师可以直接完成的检查。
	AcceptanceChecks []string `json:"acceptance_checks"`

	// TeacherContext 保存教师补充的教学目标、保留要求和注意事项。
	//
	// 初次AI检测阶段必须为空，后续只能来自教师明确输入。
	TeacherContext string `json:"teacher_context"`

	// ManualCheckRequired 表示当前结论需要教师实际打开页面操作或观察。
	ManualCheckRequired bool `json:"manual_check_required"`
}

// ==================== 审核问题 ====================

// CWAIReviewFinding 单项审核发现。
type CWAIReviewFinding struct {
	ID string `json:"id"`

	// Severity 建议值：critical / high / medium / low / info。
	Severity string `json:"severity"`

	// Dimension 必须属于会话创建时固化的R-02审核维度。
	Dimension string `json:"dimension"`

	PageNumbers []int `json:"page_numbers"`

	// Title、Description保留为原始审核事实兼容字段。
	//
	// 教师默认展示不得直接依赖这两个字段，而应使用TeacherViewSnapshot。
	Title       string `json:"title"`
	Description string `json:"description"`

	// TeacherViewSnapshot 是创建或综合时固化的教师展示事实。
	TeacherViewSnapshot CWAIReviewTeacherViewSnapshot `json:"teacher_view_snapshot"`

	LessonOrOutlineBasis string `json:"lesson_or_outline_basis"`
	PageEvidence         string `json:"page_evidence"`
	CodeEvidence         string `json:"code_evidence"`
	ContinuityEvidence   string `json:"continuity_evidence"`

	// Suggestion 是旧协议兼容字段。
	//
	// 后端归一化后该字段保持教师可读，并与ImprovementGoal语义一致。
	Suggestion string `json:"suggestion"`

	// InternalExecutionPlan 是仅供后端和AI后续执行链使用的内部计划。
	//
	// HTTP安全响应必须剥离本字段，教师默认页面不得接收或展示。
	InternalExecutionPlan string `json:"internal_execution_plan,omitempty"`

	// Confidence 范围建议为 0-100。
	Confidence int `json:"confidence"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// ==================== 风险页 ====================

// CWAIReviewRiskPage 需要风险回看的页面。
type CWAIReviewRiskPage struct {
	PageNumber int    `json:"page_number"`
	Severity   string `json:"severity"`
	Reason     string `json:"reason"`

	// EvidenceType 示例：
	// content / script / css / continuity / outline / runtime。
	EvidenceType string `json:"evidence_type"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// ==================== 批次AI结果 ====================

// CWAIReviewBatchAIResult 模型必须返回的单批结构化结果。
type CWAIReviewBatchAIResult struct {
	BatchNo     int   `json:"batch_no"`
	PageNumbers []int `json:"page_numbers"`

	BatchSummary string `json:"batch_summary"`

	Findings []CWAIReviewFinding `json:"findings"`

	// ContinuityLedger 是本批完成后的完整连续性账本。
	//
	// 后端不会直接覆盖旧账本，而会执行字段级递归合并：
	//   - 对象递归合并；
	//   - 数组合并去重；
	//   - 新值不得让旧案例和旧结论无依据消失。
	ContinuityLedger map[string]interface{} `json:"continuity_ledger"`

	RiskPages []CWAIReviewRiskPage `json:"risk_pages"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// ==================== 执行响应 ====================

// CWAIReviewRunNextResponse 执行下一批后的后端响应。
type CWAIReviewRunNextResponse struct {
	Session *CoursewareAIReviewSession `json:"session"`
	Batch   *CoursewareAIReviewBatch   `json:"batch"`

	Result *CWAIReviewBatchAIResult `json:"result"`

	HasMore          bool `json:"has_more"`
	RequiresFinalize bool `json:"requires_finalize"`
}
