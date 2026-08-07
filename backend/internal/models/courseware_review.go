package models

// courseware_review.go
//
// 课件多级审核数据模型。
//
// 审核状态由与课件制作状态正交的publish_state承载。
// 正式审核记录写入courseware_reviews。
//
// V1.3复审闭环：
//
//   - 作者在课件被退回后重新提交时，未解决的正式问题保留原问题ID；
//   - 每条问题记录准备进入的审核级别和预计轮次；
//   - 审核员打开当前待审课件时，审核详情直接返回本级、本轮需要复审的问题；
//   - 审核员提交正式决定时，明确指出哪些旧问题已经解决；
//   - 审核记录、旧问题解决、新问题交付和课件状态必须原子提交；
//   - 复审问题响应不暴露内部AI会话、问题创建者或课件作者ID。

import "time"

// ==================== 数据库实体 ====================

// CoursewareReview 课件多级审核记录。
type CoursewareReview struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	ReviewLevel  int        `json:"review_level"`
	ReviewerID   string     `json:"reviewer_id"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Comment      string     `json:"comment"`
	Dimensions   string     `json:"dimensions"`
	ReviewRound  int        `json:"review_round"`
	CreatedAt    *time.Time `json:"created_at"`
}

// ==================== 请求结构体 ====================

// SubmitCoursewareReviewRequest 提交课件审核请求。
type SubmitCoursewareReviewRequest struct {
	Note string `json:"note"`
}

// CWReviewDecisionRequest 课件审核决策请求。
//
// AIReviewSessionID和ReviewItemIDs用于本轮新发现问题：
//
//   - 不使用AI辅助时可以为空；
//   - ReviewItemIDs非空时必须同时提供AIReviewSessionID；
//   - 后端会重新读取最终报告和选中整改项；
//   - 前端不能提交AI报告正文。
//
// ResolvedReviewItemIDs用于上一轮问题复审：
//
//   - 只能包含审核详情返回的本级、本轮旧问题；
//   - 审核通过时，必须包含本级、本轮全部旧问题；
//   - 继续退回时，可以只确认其中一部分已经解决；
//   - 未选中的旧问题继续保留，供作者下一轮整改。
type CWReviewDecisionRequest struct {
	Decision   string   `json:"decision"`
	Score      *float64 `json:"score"`
	Comment    string   `json:"comment"`
	Dimensions string   `json:"dimensions"`

	AIReviewSessionID string   `json:"ai_review_session_id"`
	ReviewItemIDs     []string `json:"review_item_ids"`

	ResolvedReviewItemIDs []string `json:"resolved_review_item_ids"`
}

// ==================== 响应结构体 ====================

// CWReviewListItem 课件审核记录列表项。
type CWReviewListItem struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	ReviewLevel  int        `json:"review_level"`
	LevelName    string     `json:"level_name"`
	ReviewerID   string     `json:"reviewer_id"`
	ReviewerName string     `json:"reviewer_name"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Comment      string     `json:"comment"`
	ReviewRound  int        `json:"review_round"`
	CreatedAt    *time.Time `json:"created_at"`
}

// CWReviewHistoryResponse 课件审核历史响应。
type CWReviewHistoryResponse struct {
	Reviews      []*CWReviewListItem `json:"reviews"`
	Total        int                 `json:"total"`
	CurrentLevel int                 `json:"current_level"`
}

// CWPendingReviewItem 课件待审核列表项。
type CWPendingReviewItem struct {
	CoursewareID string     `json:"courseware_id"`
	Title        string     `json:"title"`
	Subject      string     `json:"subject"`
	Grade        string     `json:"grade"`
	PageCount    int        `json:"page_count"`
	SourceType   string     `json:"source_type"`
	SourceName   string     `json:"source_name"`
	AuthorID     string     `json:"author_id"`
	AuthorName   string     `json:"author_name"`
	SchoolName   string     `json:"school_name"`
	ReviewLevel  int        `json:"review_level"`
	LevelName    string     `json:"level_name"`
	SubmittedAt  *time.Time `json:"submitted_at"`
}

// CWPendingReviewListResponse 待审核课件列表响应。
type CWPendingReviewListResponse struct {
	Items []*CWPendingReviewItem `json:"items"`
	Total int                    `json:"total"`
}

// CWReviewStatsResponse 课件审核统计响应。
type CWReviewStatsResponse struct {
	TotalPending  int `json:"total_pending"`
	TotalReviewed int `json:"total_reviewed"`
	TotalApproved int `json:"total_approved"`
	TotalRevision int `json:"total_revision"`
}

// CWReviewedListItem 已审核记录列表项。
type CWReviewedListItem struct {
	ID              string     `json:"id"`
	CoursewareID    string     `json:"courseware_id"`
	CoursewareTitle string     `json:"courseware_title"`
	Subject         string     `json:"subject"`
	Grade           string     `json:"grade"`
	AuthorName      string     `json:"author_name"`
	ReviewLevel     int        `json:"review_level"`
	LevelName       string     `json:"level_name"`
	ReviewerName    string     `json:"reviewer_name"`
	Decision        string     `json:"decision"`
	Score           *float64   `json:"score"`
	Comment         string     `json:"comment"`
	CreatedAt       *time.Time `json:"created_at"`
}

// CWReviewedListResponse 已审核记录响应。
type CWReviewedListResponse struct {
	Items []*CWReviewedListItem `json:"items"`
	Total int                   `json:"total"`
}

// CWReviewCarryoverItem 是当前审核员需要复查的上一轮正式问题。
//
// 该结构只返回复审所需内容，不返回：
//
//   - source_session_id；
//   - source_finding_id；
//   - created_by；
//   - owner_id；
//   - 内部讨论消息身份。
type CWReviewCarryoverItem struct {
	ID string `json:"id"`

	// OriginalReviewLevel和OriginalReviewRound表示问题最初正式交付的轮次。
	OriginalReviewLevel int `json:"original_review_level"`
	OriginalReviewRound int `json:"original_review_round"`

	// PendingReviewLevel和PendingReviewRound表示当前准备接受复查的轮次。
	PendingReviewLevel int `json:"pending_review_level"`
	PendingReviewRound int `json:"pending_review_round"`

	PageID                *string    `json:"page_id"`
	PageNumberSnapshot    int        `json:"page_number_snapshot"`
	PageTitleSnapshot     string     `json:"page_title_snapshot"`
	PageHTMLHash          string     `json:"page_html_hash"`
	PageUpdatedAtSnapshot *time.Time `json:"page_updated_at_snapshot"`

	Severity  string `json:"severity"`
	Dimension string `json:"dimension"`

	Title                string `json:"title"`
	Description          string `json:"description"`
	ConfirmedInstruction string `json:"confirmed_instruction"`

	Status          string `json:"status"`
	AppliedPageHash string `json:"applied_page_hash"`

	ConfirmedAt   *time.Time `json:"confirmed_at"`
	AppliedAt     *time.Time `json:"applied_at"`
	ResubmittedAt *time.Time `json:"resubmitted_at"`
}

// CWReviewDetailResponse 课件审核详情响应。
//
// PendingReviewRound是当前级别提交审核决定后将形成的轮次。
// CarryoverItems是作者重新提交后，本级本轮需要复查的旧问题。
type CWReviewDetailResponse struct {
	Courseware  *CoursewareDetailResponse `json:"courseware"`
	Annotations []*CoursewareAnnotation   `json:"annotations"`
	Reviews     []*CWReviewListItem       `json:"reviews"`

	PendingReviewRound int                      `json:"pending_review_round"`
	CarryoverItems     []*CWReviewCarryoverItem `json:"carryover_items"`
}
