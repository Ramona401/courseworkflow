package handlers

// courseware_ai_review_item_group_views.go
//
// R-06 问题组浏览器安全响应视图。
//
// 不向浏览器返回：
//   - group.created_by；
//   - member.created_by、removed_by；
//   - event.actor_id；
//   - 任何页面HTML哈希或内部数据库锁信息。
//
// 成员只返回稳定member_id、item_id、状态和version，具体整改项内容继续复用
// 现有整改项列表接口，避免在问题组接口重复暴露问题内部字段。

import (
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
)

type coursewareAIReviewItemGroupMemberView struct {
	ID      string `json:"id"`
	GroupID string `json:"group_id"`
	ItemID  string `json:"item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	RemovedAt *time.Time `json:"removed_at"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type coursewareAIReviewItemGroupEventView struct {
	ID           string `json:"id"`
	GroupVersion int    `json:"group_version"`
	EventType    string `json:"event_type"`

	MemberID       *string `json:"member_id"`
	MemberVersion  *int    `json:"member_version"`
	RelatedGroupID *string `json:"related_group_id"`

	Reason       string `json:"reason"`
	MetadataJSON string `json:"metadata_json"`

	CreatedAt *time.Time `json:"created_at"`
}

type coursewareAIReviewItemGroupView struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	Name          string  `json:"name"`
	PrimaryItemID *string `json:"primary_item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	MergedIntoGroupID *string `json:"merged_into_group_id"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`

	Members []*coursewareAIReviewItemGroupMemberView `json:"members"`
	Events  []*coursewareAIReviewItemGroupEventView  `json:"events"`
}

type coursewareAIReviewItemGroupPairView struct {
	Source *coursewareAIReviewItemGroupView `json:"source"`
	Target *coursewareAIReviewItemGroupView `json:"target"`
}

func buildCoursewareAIReviewItemGroupMemberView(
	member *models.CoursewareReviewItemGroupMember,
) *coursewareAIReviewItemGroupMemberView {
	if member == nil {
		return nil
	}

	return &coursewareAIReviewItemGroupMemberView{
		ID:        member.ID,
		GroupID:   member.GroupID,
		ItemID:    member.ItemID,
		Status:    member.Status,
		Version:   member.Version,
		RemovedAt: member.RemovedAt,
		CreatedAt: member.CreatedAt,
		UpdatedAt: member.UpdatedAt,
	}
}

func buildCoursewareAIReviewItemGroupMemberViews(
	members []*models.CoursewareReviewItemGroupMember,
) []*coursewareAIReviewItemGroupMemberView {
	result := make(
		[]*coursewareAIReviewItemGroupMemberView,
		0,
		len(members),
	)

	for _, member := range members {
		view := buildCoursewareAIReviewItemGroupMemberView(member)
		if view == nil {
			continue
		}

		result = append(result, view)
	}

	return result
}

func buildCoursewareAIReviewItemGroupEventView(
	event *models.CoursewareReviewItemGroupEvent,
) *coursewareAIReviewItemGroupEventView {
	if event == nil {
		return nil
	}

	return &coursewareAIReviewItemGroupEventView{
		ID:             event.ID,
		GroupVersion:   event.GroupVersion,
		EventType:      event.EventType,
		MemberID:       event.MemberID,
		MemberVersion:  event.MemberVersion,
		RelatedGroupID: event.RelatedGroupID,
		Reason:         event.Reason,
		MetadataJSON:   event.MetadataJSON,
		CreatedAt:      event.CreatedAt,
	}
}

func buildCoursewareAIReviewItemGroupEventViews(
	events []*models.CoursewareReviewItemGroupEvent,
) []*coursewareAIReviewItemGroupEventView {
	result := make(
		[]*coursewareAIReviewItemGroupEventView,
		0,
		len(events),
	)

	for _, event := range events {
		view := buildCoursewareAIReviewItemGroupEventView(event)
		if view == nil {
			continue
		}

		result = append(result, view)
	}

	return result
}

func buildCoursewareAIReviewItemGroupRecordView(
	record *services.CWAIReviewItemGroupRecord,
) *coursewareAIReviewItemGroupView {
	if record == nil || record.Group == nil {
		return nil
	}

	group := record.Group

	return &coursewareAIReviewItemGroupView{
		ID:                group.ID,
		CoursewareID:      group.CoursewareID,
		SourceSessionID:   group.SourceSessionID,
		Name:              group.Name,
		PrimaryItemID:     group.PrimaryItemID,
		Status:            group.Status,
		Version:           group.Version,
		MergedIntoGroupID: group.MergedIntoGroupID,
		CreatedAt:         group.CreatedAt,
		UpdatedAt:         group.UpdatedAt,
		Members: buildCoursewareAIReviewItemGroupMemberViews(
			record.Members,
		),
		Events: buildCoursewareAIReviewItemGroupEventViews(
			record.Events,
		),
	}
}

func buildCoursewareAIReviewItemGroupRecordViews(
	records []*services.CWAIReviewItemGroupRecord,
) []*coursewareAIReviewItemGroupView {
	result := make(
		[]*coursewareAIReviewItemGroupView,
		0,
		len(records),
	)

	for _, record := range records {
		view := buildCoursewareAIReviewItemGroupRecordView(record)
		if view == nil {
			continue
		}

		result = append(result, view)
	}

	return result
}

func buildCoursewareAIReviewItemGroupPairView(
	result *services.CWAIReviewItemGroupPairResult,
) *coursewareAIReviewItemGroupPairView {
	if result == nil {
		return nil
	}

	return &coursewareAIReviewItemGroupPairView{
		Source: buildCoursewareAIReviewItemGroupRecordView(
			result.Source,
		),
		Target: buildCoursewareAIReviewItemGroupRecordView(
			result.Target,
		),
	}
}
