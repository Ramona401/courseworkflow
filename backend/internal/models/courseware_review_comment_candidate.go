package models

// courseware_review_comment_candidate.go
//
// R-08 正式课件审核意见重新汇总候选协议。
//
// 设计边界：
//
//   1. 候选只是正式审核提交前的教师辅助文本，不是courseware_reviews事实；
//   2. 候选创建后不可修改或删除，数据库trigger承担最终不可变防线；
//   3. 候选正文只能从服务端持久化记录重新读取，后续应用时浏览器不得回传正文；
//   4. stale由服务端重新读取当前修改清单、指令版本、R-06问题组和教师原意见后计算；
//   5. 替换或追加只返回新的教师输入框文本，不直接写正式审核记录；
//   6. 正式审核意见仍由既有CoursewareReviewDecision事务最终提交。

import "time"

const (
	CWReviewCommentCandidateSchemaVersion     = 1
	CWReviewCommentInputSnapshotSchemaVersion = 1
	CWReviewCommentDiffSchemaVersion          = 1
	CWReviewCommentCandidateApplyReplace      = "replace"
	CWReviewCommentCandidateApplyAppend       = "append"
)

// IsCWReviewCommentCandidateApplyAction 校验教师明确选择的候选应用方式。
func IsCWReviewCommentCandidateApplyAction(action string) bool {
	switch action {
	case CWReviewCommentCandidateApplyReplace,
		CWReviewCommentCandidateApplyAppend:
		return true
	default:
		return false
	}
}

// CWReviewCommentDiffAdjustment 表示一段原文字被候选文本调整后的前后内容。
//
// 这里只表达确定性的文本差异，不让AI解释调整原因，避免把AI推断误当事实。
type CWReviewCommentDiffAdjustment struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// CWReviewCommentDiff 是教师确认前看到的新增、删除和调整摘要。
//
// Added、Removed、Adjusted均由后端根据原意见和候选正文确定性计算，
// 不接受浏览器提交，也不直接信任AI返回的差异声明。
type CWReviewCommentDiff struct {
	Added    []string                        `json:"added"`
	Removed  []string                        `json:"removed"`
	Adjusted []CWReviewCommentDiffAdjustment `json:"adjusted"`
}

// CoursewareReviewCommentCandidate 对应courseware_review_comment_candidates。
//
// CreatedBy、事实快照、hash及AI调用审计字段属于服务端内部事实，
// 后续HTTP响应必须转换为专用安全View，不能直接把整个结构体序列化给浏览器。
type CoursewareReviewCommentCandidate struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	CreatedBy   string `json:"-"`
	ReviewLevel int    `json:"review_level"`

	CandidateSchemaVersion int    `json:"candidate_schema_version"`
	CandidateText          string `json:"candidate_text"`

	OriginalCommentSnapshot string `json:"original_comment_snapshot"`
	OriginalCommentHash     string `json:"-"`

	SelectedItemIDsJSON string `json:"-"`

	InputSnapshotSchemaVersion int    `json:"-"`
	InputSnapshotJSON          string `json:"-"`
	InputHash                  string `json:"-"`

	DiffSchemaVersion int    `json:"diff_schema_version"`
	DiffJSON          string `json:"-"`

	ModelUsed  string `json:"-"`
	TokensUsed int    `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}
