package handlers

// courseware_ai_review_item_views.go
//
// 课件审核整改闭环和问题治理的浏览器安全响应视图。
//
// 正式反馈不向浏览器返回：
//   - ai_review_session_id；
//   - created_by。
//
// 页级整改项不向浏览器返回：
//   - created_by；
//   - owner_id；
//   - 页面HTML哈希和应用结果哈希；
//   - 原始页面、代码和连续性证据；
//   - AI内部执行计划和审核配置哈希。
//
// 页级整改项允许返回current、delivered和applied三个稳定指令版本ID，
// 供浏览器读取版本、提交乐观并发参数和追溯实际交付/执行版本。
//
// 关系不向浏览器返回：
//   - created_by；
//   - cancelled_by。
//
// 关系事件不向浏览器返回actor_id。
// 独立讨论消息不向浏览器返回user_id、内部session_id、模型和Token。

import (
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
)

// coursewareAIReviewFeedbackView 是作者可见的正式审核整体反馈。
type coursewareAIReviewFeedbackView struct {
	ID                 string `json:"id"`
	CoursewareReviewID string `json:"courseware_review_id"`
	CoursewareID       string `json:"courseware_id"`

	ReviewLevel int    `json:"review_level"`
	ReviewRound int    `json:"review_round"`
	Decision    string `json:"decision"`

	OverallRisk    string `json:"overall_risk"`
	OverallSummary string `json:"overall_summary"`

	StrengthsJSON       string `json:"strengths_json"`
	ObviousProblemsJSON string `json:"obvious_problems_json"`

	ReviewCommentSnapshot string     `json:"review_comment_snapshot"`
	CreatedAt             *time.Time `json:"created_at"`
}

// coursewareAIReviewItemView 是浏览器可见的页级整改项。
type coursewareAIReviewItemView struct {
	ID           string `json:"id"`
	CoursewareID string `json:"courseware_id"`

	SourceSessionID string `json:"source_session_id"`
	SourceFindingID string `json:"source_finding_id"`

	// OriginType用于明确区分AI发现和全局讨论人工新增。
	OriginType string `json:"origin_type"`

	// SourceGlobalMessageID只在人工新增问题中存在，
	// 用于在同一授权会话中定位候选来源。
	SourceGlobalMessageID *string `json:"source_global_message_id"`

	CoursewareReviewID *string `json:"courseware_review_id"`
	FeedbackID         *string `json:"feedback_id"`

	SourceType  string `json:"source_type"`
	ReviewLevel int    `json:"review_level"`
	ReviewRound int    `json:"review_round"`

	PageID                *string    `json:"page_id"`
	PageNumberSnapshot    int        `json:"page_number_snapshot"`
	PageTitleSnapshot     string     `json:"page_title_snapshot"`
	PageHTMLHash          string     `json:"page_html_hash"`
	PageUpdatedAtSnapshot *time.Time `json:"page_updated_at_snapshot"`

	Severity  string `json:"severity"`
	Dimension string `json:"dimension"`

	// 旧字段继续返回教师化兼容内容。
	Title       string `json:"title"`
	Description string `json:"description"`

	// 新教师视图字段供共享教师改进卡直接使用。
	TeacherTitle        string   `json:"teacher_title"`
	WhatHappened        string   `json:"what_happened"`
	TeachingImpact      string   `json:"teaching_impact"`
	ImprovementGoal     string   `json:"improvement_goal"`
	AcceptanceChecks    []string `json:"acceptance_checks"`
	TeacherContext      string   `json:"teacher_context"`
	ManualCheckRequired bool     `json:"manual_check_required"`

	// EvidenceJSON只包含教师视图快照，不包含内部事实。
	EvidenceJSON       string `json:"evidence_json"`
	OriginalSuggestion string `json:"original_suggestion"`

	ConfirmedInstruction string `json:"confirmed_instruction"`

	CurrentInstructionVersionID   *string `json:"current_instruction_version_id"`
	DeliveredInstructionVersionID *string `json:"delivered_instruction_version_id"`
	AppliedInstructionVersionID   *string `json:"applied_instruction_version_id"`

	Status          string `json:"status"`
	AppliedPageHash string `json:"applied_page_hash"`

	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	AppliedAt   *time.Time `json:"applied_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

// coursewareAIReviewItemMessageView 是浏览器可见的正式讨论消息。
type coursewareAIReviewItemMessageView struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`

	MetaJSON string `json:"meta_json"`

	// 为兼容旧前端保留字段，但不返回模型和Token。
	ModelUsed  string `json:"model_used"`
	TokensUsed int    `json:"tokens_used"`

	CreatedAt *time.Time `json:"created_at"`
}

// coursewareAIReviewItemDiscussionView 是单条整改项讨论响应。
type coursewareAIReviewItemDiscussionView struct {
	Item *coursewareAIReviewItemView `json:"item"`

	Messages []*coursewareAIReviewItemMessageView `json:"messages"`

	Summary              string `json:"summary"`
	ReadyForConfirmation bool   `json:"ready_for_confirmation"`
	SuggestedInstruction string `json:"suggested_instruction"`
}

// coursewareAIReviewItemRelationEventView 是浏览器可见的关系审计事件。
//
// 不返回actor_id，避免泄露内部用户标识；接口本身已经绑定当前会话创建者。
type coursewareAIReviewItemRelationEventView struct {
	ID              string `json:"id"`
	RelationVersion int    `json:"relation_version"`
	EventType       string `json:"event_type"`
	Reason          string `json:"reason"`

	SourceGlobalMessageID *string `json:"source_global_message_id"`

	CreatedAt *time.Time `json:"created_at"`
}

// coursewareAIReviewItemRelationView 是浏览器可见的结构化问题关系。
type coursewareAIReviewItemRelationView struct {
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

	ConfirmedAt *time.Time `json:"confirmed_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`

	Events []*coursewareAIReviewItemRelationEventView `json:"events"`
}

// coursewareAIReviewOwnerRemediationView 是作者整改中心响应。
type coursewareAIReviewOwnerRemediationView struct {
	Feedbacks []*coursewareAIReviewFeedbackView `json:"feedbacks"`
	Items     []*coursewareAIReviewItemView     `json:"items"`
}

func buildCoursewareAIReviewFeedbackView(
	feedback *models.CoursewareReviewFeedback,
) *coursewareAIReviewFeedbackView {
	if feedback == nil {
		return nil
	}

	return &coursewareAIReviewFeedbackView{
		ID:                    feedback.ID,
		CoursewareReviewID:    feedback.CoursewareReviewID,
		CoursewareID:          feedback.CoursewareID,
		ReviewLevel:           feedback.ReviewLevel,
		ReviewRound:           feedback.ReviewRound,
		Decision:              feedback.Decision,
		OverallRisk:           feedback.OverallRisk,
		OverallSummary:        feedback.OverallSummary,
		StrengthsJSON:         feedback.StrengthsJSON,
		ObviousProblemsJSON:   feedback.ObviousProblemsJSON,
		ReviewCommentSnapshot: feedback.ReviewCommentSnapshot,
		CreatedAt:             feedback.CreatedAt,
	}
}

func buildCoursewareAIReviewFeedbackViews(
	feedbacks []*models.CoursewareReviewFeedback,
) []*coursewareAIReviewFeedbackView {
	result := make(
		[]*coursewareAIReviewFeedbackView,
		0,
		len(feedbacks),
	)

	for _, feedback := range feedbacks {
		if feedback == nil {
			continue
		}

		result = append(
			result,
			buildCoursewareAIReviewFeedbackView(
				feedback,
			),
		)
	}

	return result
}

func buildCoursewareAIReviewItemView(
	item *models.CoursewareReviewItem,
) *coursewareAIReviewItemView {
	if item == nil {
		return nil
	}

	teacherView :=
		services.BuildCWReviewItemTeacherView(
			item,
		)

	return &coursewareAIReviewItemView{
		ID:                    item.ID,
		CoursewareID:          item.CoursewareID,
		SourceSessionID:       item.SourceSessionID,
		SourceFindingID:       item.SourceFindingID,
		OriginType:            item.OriginType,
		SourceGlobalMessageID: item.SourceGlobalMessageID,
		CoursewareReviewID:    item.CoursewareReviewID,
		FeedbackID:            item.FeedbackID,
		SourceType:            item.SourceType,
		ReviewLevel:           item.ReviewLevel,
		ReviewRound:           item.ReviewRound,
		PageID:                item.PageID,
		PageNumberSnapshot:    item.PageNumberSnapshot,
		PageTitleSnapshot:     item.PageTitleSnapshot,

		// 保留旧字段形状，但不返回可信页面哈希。
		PageHTMLHash:          "",
		PageUpdatedAtSnapshot: item.PageUpdatedAtSnapshot,

		Severity:  item.Severity,
		Dimension: item.Dimension,

		Title:       teacherView.TeacherTitle,
		Description: teacherView.WhatHappened,

		TeacherTitle:    teacherView.TeacherTitle,
		WhatHappened:    teacherView.WhatHappened,
		TeachingImpact:  teacherView.TeachingImpact,
		ImprovementGoal: teacherView.ImprovementGoal,
		AcceptanceChecks: append(
			[]string{},
			teacherView.AcceptanceChecks...,
		),
		TeacherContext:      teacherView.TeacherContext,
		ManualCheckRequired: teacherView.ManualCheckRequired,

		EvidenceJSON: services.BuildCWReviewItemTeacherEvidenceJSON(
			teacherView,
		),
		OriginalSuggestion: teacherView.ImprovementGoal,

		ConfirmedInstruction: item.ConfirmedInstruction,

		CurrentInstructionVersionID:   item.CurrentInstructionVersionID,
		DeliveredInstructionVersionID: item.DeliveredInstructionVersionID,
		AppliedInstructionVersionID:   item.AppliedInstructionVersionID,

		Status: item.Status,

		// 保留旧字段形状，但不返回可信应用哈希。
		AppliedPageHash: "",

		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		ConfirmedAt: item.ConfirmedAt,
		AppliedAt:   item.AppliedAt,
		ResolvedAt:  item.ResolvedAt,
	}
}

func buildCoursewareAIReviewItemViews(
	items []*models.CoursewareReviewItem,
) []*coursewareAIReviewItemView {
	result := make(
		[]*coursewareAIReviewItemView,
		0,
		len(items),
	)

	for _, item := range items {
		if item == nil {
			continue
		}

		result = append(
			result,
			buildCoursewareAIReviewItemView(
				item,
			),
		)
	}

	return result
}

func buildCoursewareAIReviewItemRelationEventView(
	event *models.CoursewareReviewItemRelationEvent,
) *coursewareAIReviewItemRelationEventView {
	if event == nil {
		return nil
	}

	return &coursewareAIReviewItemRelationEventView{
		ID:                    event.ID,
		RelationVersion:       event.RelationVersion,
		EventType:             event.EventType,
		Reason:                event.Reason,
		SourceGlobalMessageID: event.SourceGlobalMessageID,
		CreatedAt:             event.CreatedAt,
	}
}

func buildCoursewareAIReviewItemRelationEventViews(
	events []*models.CoursewareReviewItemRelationEvent,
) []*coursewareAIReviewItemRelationEventView {
	result := make(
		[]*coursewareAIReviewItemRelationEventView,
		0,
		len(events),
	)

	for _, event := range events {
		if event == nil {
			continue
		}

		result = append(
			result,
			buildCoursewareAIReviewItemRelationEventView(
				event,
			),
		)
	}

	return result
}

func buildCoursewareAIReviewGlobalRelationRecordView(
	record *services.CWAIReviewGlobalRelationRecord,
) *coursewareAIReviewItemRelationView {
	if record == nil ||
		record.Relation == nil {
		return nil
	}

	relation := record.Relation

	return &coursewareAIReviewItemRelationView{
		ID:                    relation.ID,
		CoursewareID:          relation.CoursewareID,
		SourceSessionID:       relation.SourceSessionID,
		SourceItemID:          relation.SourceItemID,
		TargetItemID:          relation.TargetItemID,
		RelationType:          relation.RelationType,
		Status:                relation.Status,
		Version:               relation.Version,
		Explanation:           relation.Explanation,
		SourceGlobalMessageID: relation.SourceGlobalMessageID,
		ConfirmedAt:           relation.ConfirmedAt,
		CancelledAt:           relation.CancelledAt,
		CreatedAt:             relation.CreatedAt,
		UpdatedAt:             relation.UpdatedAt,
		Events: buildCoursewareAIReviewItemRelationEventViews(
			record.Events,
		),
	}
}

func buildCoursewareAIReviewGlobalRelationRecordViews(
	records []*services.CWAIReviewGlobalRelationRecord,
) []*coursewareAIReviewItemRelationView {
	result := make(
		[]*coursewareAIReviewItemRelationView,
		0,
		len(records),
	)

	for _, record := range records {
		view :=
			buildCoursewareAIReviewGlobalRelationRecordView(
				record,
			)
		if view == nil {
			continue
		}

		result = append(result, view)
	}

	return result
}

func buildCoursewareAIReviewOwnerRemediationView(
	bundle *services.CWOwnerReviewRemediationBundle,
) *coursewareAIReviewOwnerRemediationView {
	if bundle == nil {
		return &coursewareAIReviewOwnerRemediationView{
			Feedbacks: []*coursewareAIReviewFeedbackView{},
			Items:     []*coursewareAIReviewItemView{},
		}
	}

	return &coursewareAIReviewOwnerRemediationView{
		Feedbacks: buildCoursewareAIReviewFeedbackViews(
			bundle.Feedbacks,
		),
		Items: buildCoursewareAIReviewItemViews(
			bundle.Items,
		),
	}
}

func buildCoursewareAIReviewItemDiscussionView(
	result *services.CWReviewItemDiscussionResult,
) *coursewareAIReviewItemDiscussionView {
	if result == nil {
		return nil
	}

	messages := make(
		[]*coursewareAIReviewItemMessageView,
		0,
		len(result.Messages),
	)

	for _, message := range result.Messages {
		if message == nil {
			continue
		}

		messages = append(
			messages,
			&coursewareAIReviewItemMessageView{
				ID:         message.ID,
				Role:       message.Role,
				Content:    message.Content,
				MetaJSON:   message.CitationsJSON,
				ModelUsed:  "",
				TokensUsed: 0,
				CreatedAt:  message.CreatedAt,
			},
		)
	}

	return &coursewareAIReviewItemDiscussionView{
		Item: buildCoursewareAIReviewItemView(
			result.Item,
		),
		Messages:             messages,
		Summary:              result.Summary,
		ReadyForConfirmation: result.ReadyForConfirmation,
		SuggestedInstruction: result.SuggestedInstruction,
	}
}
