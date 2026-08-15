package models

// courseware_review_history.go
//
// R-03“已审核记录只读详情”的浏览器安全历史协议。
//
// 核心约束：
//   - 历史审核事实按courseware_review_id读取，不从当前课件状态反推；
//   - 历史整改要求只展示正式交付版本，不展示current版本；
//   - 历史页面与当前页面是两个明确的数据集合，不能互相冒充；
//   - R-03上线前没有完整页面快照的旧审核必须明确标记不可还原；
//   - 缺少旧feedback快照时不能用空数组冒充“当时没有交付问题”；
//   - 不向浏览器暴露页面哈希、版本哈希、created_by、confirmed_by等内部字段。

import "time"

const (
	// CWReviewHistoryPagesUnavailableLegacy 表示审核发生在R-03页面快照上线前。
	CWReviewHistoryPagesUnavailableLegacy = "legacy_review_without_page_snapshot"

	// CWReviewHistoryIssuesUnavailableLegacy 表示旧审核没有可证明的正式反馈快照。
	CWReviewHistoryIssuesUnavailableLegacy = "legacy_review_without_feedback_snapshot"

	// CWReviewHistoryConfigUnavailableNoAI 表示本次人工审核没有关联AI审核会话。
	CWReviewHistoryConfigUnavailableNoAI = "review_without_ai_session"

	// CWReviewHistoryConfigUnavailableLegacy 表示存在历史AI会话，
	// 但没有可按现行R-02不可变协议可靠恢复的配置事实。
	CWReviewHistoryConfigUnavailableLegacy = "legacy_review_without_immutable_config"

	// CWReviewHistoryConfigUnavailableSession 表示反馈中记录了会话ID，
	// 但对应不可变AI会话已经无法读取。
	CWReviewHistoryConfigUnavailableSession = "review_ai_session_unavailable"

	// CWReviewHistoryInstructionUnavailableLegacy 表示旧整改项缺少
	// 可证明的正式交付版本引用。
	CWReviewHistoryInstructionUnavailableLegacy = "legacy_item_without_delivered_instruction_version"
)

// CoursewareReviewHistoryCourseware 是历史详情页需要的课件基本信息。
type CoursewareReviewHistoryCourseware struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
	Grade   string `json:"grade"`
}

// CoursewareReviewHistoryReviewer 是审核教师的安全展示身份。
//
// ID来自真实courseware_reviews.reviewer_id；DisplayName只是展示值，
// 不能取代ReviewerID作为审核身份事实。
type CoursewareReviewHistoryReviewer struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// CoursewareReviewHistoryDecision 是一次真实正式审核决定。
type CoursewareReviewHistoryDecision struct {
	ReviewLevel int        `json:"review_level"`
	ReviewRound int        `json:"review_round"`
	Decision    string     `json:"decision"`
	Score       *float64   `json:"score"`
	Comment     string     `json:"comment"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
}

// CoursewareReviewHistoryConfig 是本次审核实际冻结的R-02配置。
//
// LessonMaterialsUsed使用指针：
//   - true：不可变上下文证明本次确实带入了至少一种教案类材料；
//   - false：不可变上下文证明本次没有使用教案类材料；
//   - nil：旧记录没有足够事实证明实际使用情况。
type CoursewareReviewHistoryConfig struct {
	Available bool `json:"available"`

	SchemaVersion int      `json:"schema_version"`
	Dimensions    []string `json:"dimensions"`

	CustomFocus string `json:"custom_focus"`

	LessonReferenceMode string `json:"lesson_reference_mode"`
	LessonMaterialsUsed *bool  `json:"lesson_materials_used"`

	UnavailableReason string `json:"unavailable_reason"`
}

// CoursewareReviewHistoryDeliveredInstruction 是本次正式审核真正交付给作者的版本。
//
// 不暴露current标志、哈希、内部用户ID或当前版本状态。
type CoursewareReviewHistoryDeliveredInstruction struct {
	VersionID string `json:"version_id"`
	VersionNo int    `json:"version_no"`

	Content    string `json:"content"`
	SourceType string `json:"source_type"`

	ConfirmedAt *time.Time `json:"confirmed_at"`
}

// CoursewareReviewHistoryModificationRecord 是作者后来追加的正式执行补充。
//
// 它作为整改过程历史单独展示，不参与重算本次审核决定或原始问题事实。
type CoursewareReviewHistoryModificationRecord struct {
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"created_at"`
}

// CoursewareReviewHistoryIssue 是本次审核实际交付的问题。
//
// 不包含整改项当前status，避免后续applied/resolved/stale/dismissed等状态
// 反向污染旧审核证据。
type CoursewareReviewHistoryIssue struct {
	ID string `json:"id"`

	PageID     *string `json:"page_id"`
	PageNumber int     `json:"page_number"`
	PageTitle  string  `json:"page_title"`

	Severity  string `json:"severity"`
	Dimension string `json:"dimension"`

	TeacherView CWAIReviewTeacherViewSnapshot `json:"teacher_view"`

	DeliveredInstructionAvailable bool `json:"delivered_instruction_available"`

	DeliveredInstruction *CoursewareReviewHistoryDeliveredInstruction `json:"delivered_instruction"`

	DeliveredInstructionUnavailableReason string `json:"delivered_instruction_unavailable_reason"`

	PreviousModificationRecords []CoursewareReviewHistoryModificationRecord `json:"previous_modification_records"`
}

// CoursewareReviewHistoryPage 是“审核时页面”。
// HTMLContent只能来自R-03专用不可变快照表。
type CoursewareReviewHistoryPage struct {
	PageID     string `json:"page_id"`
	PageNumber int    `json:"page_number"`
	PageTitle  string `json:"page_title"`

	HTMLContent string `json:"html_content"`

	PageUpdatedAt *time.Time `json:"page_updated_at"`

	// false表示稳定page_id已经不在当前课件页面集合中。
	CurrentExists bool `json:"current_exists"`
}

// CoursewareReviewHistoryCurrentPage 是显式“当前页面”Tab的数据。
type CoursewareReviewHistoryCurrentPage struct {
	PageID     string `json:"page_id"`
	PageNumber int    `json:"page_number"`
	PageTitle  string `json:"page_title"`

	HTMLContent string `json:"html_content"`

	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareReviewHistoryDetail 是R-03专用只读详情响应。
type CoursewareReviewHistoryDetail struct {
	ReviewID string `json:"review_id"`

	RecordTitle string `json:"record_title"`

	Courseware CoursewareReviewHistoryCourseware `json:"courseware"`
	Reviewer   CoursewareReviewHistoryReviewer   `json:"reviewer"`
	Review     CoursewareReviewHistoryDecision   `json:"review"`

	ReviewConfig CoursewareReviewHistoryConfig `json:"review_config"`

	IssuesAvailable         bool                           `json:"issues_available"`
	IssuesUnavailableReason string                         `json:"issues_unavailable_reason"`
	Issues                  []CoursewareReviewHistoryIssue `json:"issues"`

	HistoricalPagesAvailable bool `json:"historical_pages_available"`

	HistoricalPagesUnavailableReason string `json:"historical_pages_unavailable_reason"`

	HistoricalPages []CoursewareReviewHistoryPage `json:"historical_pages"`

	CurrentPages []CoursewareReviewHistoryCurrentPage `json:"current_pages"`
}
