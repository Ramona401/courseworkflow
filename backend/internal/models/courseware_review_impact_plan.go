package models

import "time"

// courseware_review_impact_plan.go
//
// R-07 全局讨论结构化影响方案的数据协议。
//
// 安全边界：
//   1. AI只产生候选结构化操作，不自动执行；
//   2. operations_json在草稿创建后不可变；
//   3. 浏览器最终只回传plan_id、version和选中的operation_id；
//   4. 后端应用时重新读取可信消息和全部目标业务状态；
//   5. 任一目标过期或冲突时整个应用事务回滚。

const (
	CWReviewImpactPlanStatusDraft   = "draft"
	CWReviewImpactPlanStatusApplied = "applied"

	CWReviewImpactPlanEventDraftCreated = "draft_created"
	CWReviewImpactPlanEventApplied      = "applied"

	CWReviewImpactOperationCreateGroup               = "create_group"
	CWReviewImpactOperationMoveGroupMember           = "move_group_member"
	CWReviewImpactOperationMergeGroups               = "merge_groups"
	CWReviewImpactOperationSplitGroup                = "split_group"
	CWReviewImpactOperationCreateRelation            = "create_relation"
	CWReviewImpactOperationCancelRelation            = "cancel_relation"
	CWReviewImpactOperationCreateItem                = "create_item"
	CWReviewImpactOperationDismissItem               = "dismiss_item"
	CWReviewImpactOperationUpdateCandidateSuggestion = "update_candidate_suggestion"
)

// IsCWReviewImpactOperationType 判断结构化影响操作类型是否受当前协议支持。
func IsCWReviewImpactOperationType(value string) bool {
	switch value {
	case CWReviewImpactOperationCreateGroup,
		CWReviewImpactOperationMoveGroupMember,
		CWReviewImpactOperationMergeGroups,
		CWReviewImpactOperationSplitGroup,
		CWReviewImpactOperationCreateRelation,
		CWReviewImpactOperationCancelRelation,
		CWReviewImpactOperationCreateItem,
		CWReviewImpactOperationDismissItem,
		CWReviewImpactOperationUpdateCandidateSuggestion:
		return true

	default:
		return false
	}
}

// CoursewareReviewImpactOperation 是operations_json中的单条不可变候选操作。
//
// Payload保存操作目标和教师将看到的业务参数。
// Preconditions保存生成计划时由服务端读取的业务前置事实。
// 两者在最终Apply时都必须由后端重新解释和复核，不能由浏览器覆盖。
type CoursewareReviewImpactOperation struct {
	OperationID   string                 `json:"operation_id"`
	OperationType string                 `json:"operation_type"`
	Summary       string                 `json:"summary"`
	Payload       map[string]interface{} `json:"payload"`
	Preconditions map[string]interface{} `json:"preconditions"`
}

// CoursewareReviewImpactPlan 对应courseware_review_impact_plans。
type CoursewareReviewImpactPlan struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`
	SourceMessageID string `json:"source_message_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	OperationsSchemaVersion int    `json:"operations_schema_version"`
	OperationsJSON          string `json:"operations_json"`
	OperationsHash          string `json:"operations_hash"`
	SourceMessageHash       string `json:"source_message_hash"`

	CreatedBy string `json:"created_by"`

	CreatedAt *time.Time `json:"created_at"`

	AppliedOperationIDsJSON string     `json:"applied_operation_ids_json"`
	AppliedBy               *string    `json:"applied_by"`
	AppliedAt               *time.Time `json:"applied_at"`

	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareReviewImpactPlanEvent 对应courseware_review_impact_plan_events。
type CoursewareReviewImpactPlanEvent struct {
	ID              string `json:"id"`
	PlanID          string `json:"plan_id"`
	SourceSessionID string `json:"source_session_id"`

	PlanVersion int    `json:"plan_version"`
	EventType   string `json:"event_type"`
	ActorID     string `json:"actor_id"`

	SelectedOperationIDsJSON string `json:"selected_operation_ids_json"`
	MetadataJSON             string `json:"metadata_json"`

	CreatedAt *time.Time `json:"created_at"`
}
