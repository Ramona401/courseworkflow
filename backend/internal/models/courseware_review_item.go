package models

// courseware_review_item.go
//
// 课件AI审核整改闭环的数据协议。
//
// 设计边界：
//   1. CoursewareReviewFeedback保存正式人工审核提交时的整体反馈快照；
//   2. CoursewareReviewItem保存一条可讨论、确认、修改和复审的问题；
//   3. 页面定位以PageID为主，PageNumberSnapshot只用于展示审核时的页码；
//   4. PageHTMLHash用于判断旧意见是否仍适用于当前页面；
//   5. 自审和正式审核共用整改项，但权限、审核级别和归档方式不同；
//   6. 页面修改成功只表示applied，不会自动等于resolved；
//   7. AI只能形成建议，不能自动改变课件审核决定或页面内容；
//   8. 确认指令以不可变版本为事实源，ConfirmedInstruction仅保留兼容快照；
//   9. 正式交付和页面应用分别冻结实际使用的版本引用。

import "time"

// ==================== 整改项来源 ====================

const (
	CWReviewItemSourceSelf   = "self"
	CWReviewItemSourceFormal = "formal"
)

// ==================== 整改项产生方式 ====================

const (
	CWReviewItemOriginAIFinding              = "ai_finding"
	CWReviewItemOriginGlobalDiscussionManual = "global_discussion_manual"
)

// IsCWReviewItemOriginType 判断整改项产生方式是否合法。
func IsCWReviewItemOriginType(originType string) bool {
	switch originType {
	case CWReviewItemOriginAIFinding,
		CWReviewItemOriginGlobalDiscussionManual:
		return true
	default:
		return false
	}
}

// ==================== 整改项状态 ====================

const (
	CWReviewItemStatusDetected   = "detected"
	CWReviewItemStatusDiscussing = "discussing"
	CWReviewItemStatusConfirmed  = "confirmed"
	CWReviewItemStatusApplying   = "applying"
	CWReviewItemStatusApplied    = "applied"
	CWReviewItemStatusResolved   = "resolved"
	CWReviewItemStatusDismissed  = "dismissed"
	CWReviewItemStatusStale      = "stale"
	CWReviewItemStatusOrphaned   = "orphaned"
)

// ActiveCWReviewItemStatuses 返回仍可继续处理的整改项状态。
//
// 返回新切片，调用方可以安全修改，不会污染全局状态。
func ActiveCWReviewItemStatuses() []string {
	return []string{
		CWReviewItemStatusDetected,
		CWReviewItemStatusDiscussing,
		CWReviewItemStatusConfirmed,
		CWReviewItemStatusApplying,
		CWReviewItemStatusApplied,
	}
}

// IsCWReviewItemStatus 判断状态是否合法。
func IsCWReviewItemStatus(status string) bool {
	switch status {
	case CWReviewItemStatusDetected,
		CWReviewItemStatusDiscussing,
		CWReviewItemStatusConfirmed,
		CWReviewItemStatusApplying,
		CWReviewItemStatusApplied,
		CWReviewItemStatusResolved,
		CWReviewItemStatusDismissed,
		CWReviewItemStatusStale,
		CWReviewItemStatusOrphaned:
		return true
	default:
		return false
	}
}

// ==================== 严重程度 ====================

const (
	CWReviewSeverityCritical = "critical"
	CWReviewSeverityHigh     = "high"
	CWReviewSeverityMedium   = "medium"
	CWReviewSeverityLow      = "low"
	CWReviewSeverityInfo     = "info"
)

// IsCWReviewSeverity 判断课件审核严重程度是否合法。
func IsCWReviewSeverity(severity string) bool {
	switch severity {
	case CWReviewSeverityCritical,
		CWReviewSeverityHigh,
		CWReviewSeverityMedium,
		CWReviewSeverityLow,
		CWReviewSeverityInfo:
		return true
	default:
		return false
	}
}

// ==================== 正式审核整体反馈快照 ====================

// CoursewareReviewFeedback 对应courseware_review_feedback。
//
// 本记录在正式审核员提交决定时生成，之后不跟随AI报告、整改项讨论
// 或课件页面变化而覆盖，确保审核历史可追溯。
type CoursewareReviewFeedback struct {
	ID                 string  `json:"id"`
	CoursewareReviewID string  `json:"courseware_review_id"`
	CoursewareID       string  `json:"courseware_id"`
	AIReviewSessionID  *string `json:"ai_review_session_id"`

	ReviewLevel int    `json:"review_level"`
	ReviewRound int    `json:"review_round"`
	Decision    string `json:"decision"`

	OverallRisk           string `json:"overall_risk"`
	OverallSummary        string `json:"overall_summary"`
	StrengthsJSON         string `json:"strengths_json"`
	ObviousProblemsJSON   string `json:"obvious_problems_json"`
	ReviewCommentSnapshot string `json:"review_comment_snapshot"`

	CreatedBy string     `json:"created_by"`
	CreatedAt *time.Time `json:"created_at"`
}

// ==================== 页级整改项 ====================

// CoursewareReviewItem 对应courseware_review_items。
//
// 一条跨多页AI发现应由服务层拆分成多条整改项，使每条记录只对应
// 一个稳定页面。整课全局问题允许PageID为空且PageNumberSnapshot为0。
type CoursewareReviewItem struct {
	ID           string `json:"id"`
	CoursewareID string `json:"courseware_id"`

	SourceSessionID string `json:"source_session_id"`
	SourceFindingID string `json:"source_finding_id"`

	OriginType            string  `json:"origin_type"`
	SourceGlobalMessageID *string `json:"source_global_message_id"`

	CoursewareReviewID *string `json:"courseware_review_id"`
	FeedbackID         *string `json:"feedback_id"`

	SourceType  string `json:"source_type"`
	ReviewLevel int    `json:"review_level"`
	ReviewRound int    `json:"review_round"`

	CreatedBy string `json:"created_by"`
	OwnerID   string `json:"owner_id"`

	PageID                *string    `json:"page_id"`
	PageNumberSnapshot    int        `json:"page_number_snapshot"`
	PageTitleSnapshot     string     `json:"page_title_snapshot"`
	PageHTMLHash          string     `json:"page_html_hash"`
	PageUpdatedAtSnapshot *time.Time `json:"page_updated_at_snapshot"`

	Severity  string `json:"severity"`
	Dimension string `json:"dimension"`

	Title       string `json:"title"`
	Description string `json:"description"`

	EvidenceJSON       string `json:"evidence_json"`
	OriginalSuggestion string `json:"original_suggestion"`

	// ConfirmedInstruction是当前确认版本正文的兼容快照。
	//
	// 新业务判断、正式交付和页面应用必须使用下面三个稳定版本ID，
	// 不能继续仅凭本字段推断实际确认或执行版本。
	ConfirmedInstruction string `json:"confirmed_instruction"`

	CurrentInstructionVersionID   *string `json:"current_instruction_version_id"`
	DeliveredInstructionVersionID *string `json:"delivered_instruction_version_id"`
	AppliedInstructionVersionID   *string `json:"applied_instruction_version_id"`

	Status          string `json:"status"`
	AppliedPageHash string `json:"applied_page_hash"`

	// ResubmittedAt表示作者完成正式整改后最近一次重新提交审核的时间。
	//
	// ResubmittedReviewLevel和ResubmittedReviewRound描述该次提交准备进入
	// 的审核级别与预计轮次。作者自审问题始终保持空时间和零值。
	ResubmittedAt          *time.Time `json:"resubmitted_at"`
	ResubmittedReviewLevel int        `json:"resubmitted_review_level"`
	ResubmittedReviewRound int        `json:"resubmitted_review_round"`

	// ResolvedBy表示最终明确确认问题已经解决的人。
	//
	// 正式问题同时记录对应审核记录、审核级别和审核轮次；
	// 作者自审问题由作者本人确认，因此ResolvedReviewID为空且级别、轮次为0。
	ResolvedBy          *string `json:"resolved_by"`
	ResolvedReviewID    *string `json:"resolved_review_id"`
	ResolvedReviewLevel int     `json:"resolved_review_level"`
	ResolvedReviewRound int     `json:"resolved_review_round"`
	ResolutionNote      string  `json:"resolution_note"`

	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	AppliedAt   *time.Time `json:"applied_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

// IsGlobalIssue 判断该整改项是否为没有单独页面的整课问题。
func (item *CoursewareReviewItem) IsGlobalIssue() bool {
	if item == nil {
		return false
	}

	return item.PageID == nil &&
		item.PageNumberSnapshot == 0
}

// IsTerminal 判断整改项是否已经进入不可继续处理的终态。
func (item *CoursewareReviewItem) IsTerminal() bool {
	if item == nil {
		return false
	}

	switch item.Status {
	case CWReviewItemStatusResolved,
		CWReviewItemStatusDismissed,
		CWReviewItemStatusStale,
		CWReviewItemStatusOrphaned:
		return true
	default:
		return false
	}
}

// ==================== 整改项结构化关系 ====================

const (
	CWReviewItemRelationDuplicate        = "duplicate"
	CWReviewItemRelationConflict         = "conflict"
	CWReviewItemRelationMerge            = "merge"
	CWReviewItemRelationDependency       = "dependency"
	CWReviewItemRelationPossiblyResolved = "possibly_resolved"

	CWReviewItemRelationStatusActive    = "active"
	CWReviewItemRelationStatusCancelled = "cancelled"

	CWReviewItemRelationEventConfirmed   = "confirmed"
	CWReviewItemRelationEventCancelled   = "cancelled"
	CWReviewItemRelationEventReactivated = "reactivated"
)

// IsCWReviewItemRelationType 判断整改项关系类型是否合法。
func IsCWReviewItemRelationType(relationType string) bool {
	switch relationType {
	case CWReviewItemRelationDuplicate,
		CWReviewItemRelationConflict,
		CWReviewItemRelationMerge,
		CWReviewItemRelationDependency,
		CWReviewItemRelationPossiblyResolved:
		return true
	default:
		return false
	}
}

// CoursewareReviewItemRelation 对应courseware_review_item_relations。
//
// duplicate、merge、dependency和possibly_resolved具有方向：
// source_item_id是被归并、依赖或可能被连带解决的问题，
// target_item_id是主问题、前置问题或实际执行问题。
// conflict是无方向关系，保存时按UUID文本升序规范化。
type CoursewareReviewItemRelation struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
	RelationType string `json:"relation_type"`

	Status      string `json:"status"`
	Version     int    `json:"version"`
	Explanation string `json:"explanation"`

	SourceGlobalMessageID *string `json:"source_global_message_id"`

	CreatedBy   string     `json:"created_by"`
	ConfirmedAt *time.Time `json:"confirmed_at"`

	CancelledBy *string    `json:"cancelled_by"`
	CancelledAt *time.Time `json:"cancelled_at"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareReviewItemRelationEvent 对应关系的追加式治理历史。
type CoursewareReviewItemRelationEvent struct {
	ID              string `json:"id"`
	RelationID      string `json:"relation_id"`
	SourceSessionID string `json:"source_session_id"`

	RelationVersion int    `json:"relation_version"`
	EventType       string `json:"event_type"`

	ActorID string `json:"actor_id"`
	Reason  string `json:"reason"`

	SourceGlobalMessageID *string `json:"source_global_message_id"`
	MetadataJSON          string  `json:"metadata_json"`

	CreatedAt *time.Time `json:"created_at"`
}

// ==================== 整改项独立讨论消息 ====================

// CoursewareReviewItemMessage 是courseware_ai_review_messages中
// review_item_id非空记录的浏览器安全业务模型。
type CoursewareReviewItemMessage struct {
	ID           string  `json:"id"`
	SessionID    string  `json:"session_id"`
	ReviewItemID string  `json:"review_item_id"`
	UserID       *string `json:"user_id"`

	Role    string `json:"role"`
	Content string `json:"content"`

	CitationsJSON string `json:"citations_json"`
	TokensUsed    int    `json:"tokens_used"`
	ModelUsed     string `json:"model_used"`

	CreatedAt *time.Time `json:"created_at"`
}
