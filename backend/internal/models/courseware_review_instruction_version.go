package models

// courseware_review_instruction_version.go
//
// 课件审核整改指令不可变版本协议。
//
// 设计边界：
//   1. 每次明确确认都创建新版本，既有版本正文不得覆盖；
//   2. current、delivered、applied三类引用由整改项保存，版本实体只保存事实；
//   3. 浏览器可读取版本内容、来源、时间、页面快照和状态，不能提交可信身份字段；
//   4. 页面变化或删除后版本历史仍保留，但状态变为invalid_for_page，不再允许执行；
//   5. confirmed_instruction仅作为旧代码兼容快照，不再作为唯一事实源。

import "time"

// ==================== 指令版本来源 ====================

const (
	CWReviewInstructionVersionSourceLegacyBackfill     = "legacy_backfill"
	CWReviewInstructionVersionSourceLegacyDirectUpdate = "legacy_direct_update"
	CWReviewInstructionVersionSourceManual             = "manual"
	CWReviewInstructionVersionSourceAICandidate        = "ai_candidate"
	CWReviewInstructionVersionSourceGlobalDiscussion   = "global_discussion"
)

// IsCWReviewInstructionVersionSourceType 判断版本来源是否合法。
func IsCWReviewInstructionVersionSourceType(sourceType string) bool {
	switch sourceType {
	case CWReviewInstructionVersionSourceLegacyBackfill,
		CWReviewInstructionVersionSourceLegacyDirectUpdate,
		CWReviewInstructionVersionSourceManual,
		CWReviewInstructionVersionSourceAICandidate,
		CWReviewInstructionVersionSourceGlobalDiscussion:
		return true
	default:
		return false
	}
}

// ==================== 指令版本状态 ====================

const (
	CWReviewInstructionVersionStatusDraft          = "draft"
	CWReviewInstructionVersionStatusConfirmed      = "confirmed"
	CWReviewInstructionVersionStatusSuperseded     = "superseded"
	CWReviewInstructionVersionStatusInvalidForPage = "invalid_for_page"
)

// IsCWReviewInstructionVersionStatus 判断版本状态是否合法。
func IsCWReviewInstructionVersionStatus(status string) bool {
	switch status {
	case CWReviewInstructionVersionStatusDraft,
		CWReviewInstructionVersionStatusConfirmed,
		CWReviewInstructionVersionStatusSuperseded,
		CWReviewInstructionVersionStatusInvalidForPage:
		return true
	default:
		return false
	}
}

// CoursewareReviewInstructionVersion 对应courseware_review_instruction_versions。
//
// CreatedBy和ConfirmedBy只供后端授权、审计和持久化使用。
// HTTP安全视图不得直接暴露这两个内部用户ID。
type CoursewareReviewInstructionVersion struct {
	ID        string `json:"id"`
	ItemID    string `json:"item_id"`
	VersionNo int    `json:"version_no"`

	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
	SourceType  string `json:"source_type"`

	CreatedBy string     `json:"created_by"`
	CreatedAt *time.Time `json:"created_at"`

	ConfirmedBy *string    `json:"confirmed_by"`
	ConfirmedAt *time.Time `json:"confirmed_at"`

	PageSnapshotHash string `json:"page_snapshot_hash"`
	Status           string `json:"status"`
}

// IsExecutable 判断版本当前是否仍具有直接执行资格。
func (version *CoursewareReviewInstructionVersion) IsExecutable() bool {
	return version != nil &&
		version.Status == CWReviewInstructionVersionStatusConfirmed
}
