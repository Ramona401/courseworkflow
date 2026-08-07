package models

// courseware_ai_review_final.go
//
// 课件 AI 审核最终综合报告协议。
//
// 最终报告的职责：
//   - 汇总全部顺序批次的审核发现；
//   - 对风险页面执行一次综合回看；
//   - 合并重复问题，形成有优先级的修改清单；
//   - 给人工审核员提供审核意见草稿；
//   - 固化本次审核会话实际使用的R-02配置。
//
// 最终报告无权：
//   - 自动通过或退回课件；
//   - 修改 coursewares.publish_state；
//   - 替代审核员作出 L1 或 L2 决策；
//   - 自行声明或改写审核维度、教案模式和配置哈希。

// CWAIReviewDimensionReportItem 是最终报告中可展示的审核维度。
type CWAIReviewDimensionReportItem struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// CWAIReviewConfigReport 是最终报告中由后端覆盖写入的配置事实。
//
// 该结构不能以AI输出为事实源。
// 最终解析完成后，后端必须从不可变Session配置重新生成并覆盖。
type CWAIReviewConfigReport struct {
	SchemaVersion int `json:"schema_version"`

	ReviewDimensions     []string                        `json:"review_dimensions"`
	ReviewDimensionItems []CWAIReviewDimensionReportItem `json:"review_dimension_items"`

	CustomDimensionDescription string `json:"custom_dimension_description"`

	LessonReferenceMode  string `json:"lesson_reference_mode"`
	LessonReferenceLabel string `json:"lesson_reference_label"`
	UsesLessonMaterials  bool   `json:"uses_lesson_materials"`

	ReviewConfigHash string `json:"review_config_hash"`
}

// CWAIReviewPriorityAction 最终报告中的优先修改动作。
type CWAIReviewPriorityAction struct {
	Priority int `json:"priority"`

	Title       string `json:"title"`
	Description string `json:"description"`

	PageNumbers []int `json:"page_numbers"`

	Reason string `json:"reason"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// CWAIReviewFinalReport 全课件综合报告。
type CWAIReviewFinalReport struct {
	// ReviewConfig 由后端根据不可变会话配置覆盖写入。
	ReviewConfig CWAIReviewConfigReport `json:"review_config"`

	// OverallRisk 只表达风险程度，不表达通过或退回。
	// 建议值：critical / high / medium / low / info。
	OverallRisk string `json:"overall_risk"`

	Summary string `json:"summary"`

	Strengths []string `json:"strengths"`

	Findings []CWAIReviewFinding `json:"findings"`

	PriorityActions []CWAIReviewPriorityAction `json:"priority_actions"`

	ManualReviewPages []int `json:"manual_review_pages"`

	// ReviewCommentDraft 只是供审核员编辑的意见草稿。
	// 前端不得自动提交该文本。
	ReviewCommentDraft string `json:"review_comment_draft"`

	HumanDecisionReminder string `json:"human_decision_reminder"`
}

// CWAIReviewFinalizeResponse 最终综合接口响应。
type CWAIReviewFinalizeResponse struct {
	Session *CoursewareAIReviewSession `json:"session"`
	Report  *CWAIReviewFinalReport     `json:"report"`
}
