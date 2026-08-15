package models

import "time"

// courseware_review_item_group.go
//
// R-06 正式问题组的数据协议。
//
// 设计边界：
//   1. 问题组只负责教师治理视图，不替代既有整改项关系；
//   2. 成员始终保留独立整改项身份、页面、严重度、修改要求和整改状态；
//   3. 组、成员和事件均使用稳定ID与version，支持明确的乐观并发冲突；
//   4. 所有治理动作由追加式事件记录，事件不向浏览器暴露actor_id。

const (
	CWReviewItemGroupStatusActive = "active"
	CWReviewItemGroupStatusMerged = "merged"

	CWReviewItemGroupMemberStatusActive  = "active"
	CWReviewItemGroupMemberStatusRemoved = "removed"

	CWReviewItemGroupEventCreated        = "created"
	CWReviewItemGroupEventRenamed        = "renamed"
	CWReviewItemGroupEventPrimaryChanged = "primary_changed"
	CWReviewItemGroupEventMemberAdded    = "member_added"
	CWReviewItemGroupEventMemberRemoved  = "member_removed"
	CWReviewItemGroupEventMemberMoved    = "member_moved"
	CWReviewItemGroupEventMerged         = "merged"
	CWReviewItemGroupEventSplit          = "split"
)

// CoursewareReviewItemGroup 对应 courseware_review_item_groups。
type CoursewareReviewItemGroup struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	Name          string  `json:"name"`
	PrimaryItemID *string `json:"primary_item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	MergedIntoGroupID *string `json:"merged_into_group_id"`

	CreatedBy string `json:"created_by"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareReviewItemGroupMember 对应 courseware_review_item_group_members。
//
// 同一个课件审核会话中的整改项只有一个稳定成员身份。移除后不删除记录，
// 后续重新加入时复用同一ID并递增version。
type CoursewareReviewItemGroupMember struct {
	ID              string `json:"id"`
	GroupID         string `json:"group_id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`
	ItemID          string `json:"item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	CreatedBy string  `json:"created_by"`
	RemovedBy *string `json:"removed_by"`

	RemovedAt *time.Time `json:"removed_at"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareReviewItemGroupEvent 对应 courseware_review_item_group_events。
//
// GroupVersion定义组时间线顺序；成员类事件同时固化MemberVersion。
type CoursewareReviewItemGroupEvent struct {
	ID              string `json:"id"`
	GroupID         string `json:"group_id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	GroupVersion int    `json:"group_version"`
	EventType    string `json:"event_type"`

	ActorID string `json:"actor_id"`

	MemberID       *string `json:"member_id"`
	MemberVersion  *int    `json:"member_version"`
	RelatedGroupID *string `json:"related_group_id"`

	Reason       string `json:"reason"`
	MetadataJSON string `json:"metadata_json"`

	CreatedAt *time.Time `json:"created_at"`
}
