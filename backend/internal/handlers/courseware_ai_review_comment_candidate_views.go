package handlers

// courseware_ai_review_comment_candidate_views.go
//
// R-08审核意见候选浏览器安全视图。
//
// 浏览器允许获得：
//
//   - candidate ID；
//   - 原审核意见快照；
//   - 新审核意见候选；
//   - 确定性added / removed / adjusted差异；
//   - 创建时间。
//
// 明确不返回：
//
//   - created_by；
//   - original_comment_hash；
//   - selected_item_ids_json内部冻结副本；
//   - input_snapshot_json；
//   - input_hash；
//   - model_used / tokens_used；
//   - R-06内部版本快照和指令内容hash。
//
// 浏览器可以展示candidate_text，但Apply时绝不能把candidate_text回传作为事实源。

import (
	"encoding/json"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
)

type coursewareAIReviewCommentDiffAdjustmentView struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type coursewareAIReviewCommentDiffView struct {
	Added []string `json:"added"`

	Removed []string `json:"removed"`

	Adjusted []coursewareAIReviewCommentDiffAdjustmentView `json:"adjusted"`
}

type coursewareAIReviewCommentCandidateView struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`
	ReviewLevel     int    `json:"review_level"`

	CandidateSchemaVersion int `json:"candidate_schema_version"`

	OriginalComment string `json:"original_comment"`
	CandidateText   string `json:"candidate_text"`

	DiffSchemaVersion int                               `json:"diff_schema_version"`
	Diff              coursewareAIReviewCommentDiffView `json:"diff"`

	CreatedAt time.Time `json:"created_at"`
}

type coursewareAIReviewCommentCandidateApplyView struct {
	CandidateID string `json:"candidate_id"`
	Action      string `json:"action"`
	NextComment string `json:"next_comment"`
}

func buildCoursewareAIReviewCommentCandidateView(
	candidate *models.CoursewareReviewCommentCandidate,
) *coursewareAIReviewCommentCandidateView {
	if candidate == nil {
		return nil
	}

	return &coursewareAIReviewCommentCandidateView{
		ID:              candidate.ID,
		CoursewareID:    candidate.CoursewareID,
		SourceSessionID: candidate.SourceSessionID,
		ReviewLevel:     candidate.ReviewLevel,

		CandidateSchemaVersion: candidate.CandidateSchemaVersion,

		OriginalComment: candidate.OriginalCommentSnapshot,
		CandidateText:   candidate.CandidateText,

		DiffSchemaVersion: candidate.DiffSchemaVersion,
		Diff: parseCoursewareAIReviewCommentDiffView(
			candidate.DiffJSON,
		),

		CreatedAt: candidate.CreatedAt,
	}
}

func buildCoursewareAIReviewCommentCandidateApplyView(
	result *services.CWReviewCommentCandidateApplyResult,
) *coursewareAIReviewCommentCandidateApplyView {
	if result == nil {
		return nil
	}

	return &coursewareAIReviewCommentCandidateApplyView{
		CandidateID: result.CandidateID,
		Action:      result.Action,
		NextComment: result.NextComment,
	}
}

func parseCoursewareAIReviewCommentDiffView(
	raw string,
) coursewareAIReviewCommentDiffView {
	var diff models.CWReviewCommentDiff

	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(raw)),
		&diff,
	); err != nil {
		return emptyCoursewareAIReviewCommentDiffView()
	}

	result := emptyCoursewareAIReviewCommentDiffView()

	result.Added = append(
		result.Added,
		diff.Added...,
	)

	result.Removed = append(
		result.Removed,
		diff.Removed...,
	)

	for _, adjustment := range diff.Adjusted {
		result.Adjusted = append(
			result.Adjusted,
			coursewareAIReviewCommentDiffAdjustmentView{
				Before: adjustment.Before,
				After:  adjustment.After,
			},
		)
	}

	return result
}

func emptyCoursewareAIReviewCommentDiffView() coursewareAIReviewCommentDiffView {
	return coursewareAIReviewCommentDiffView{
		Added:    []string{},
		Removed:  []string{},
		Adjusted: []coursewareAIReviewCommentDiffAdjustmentView{},
	}
}
