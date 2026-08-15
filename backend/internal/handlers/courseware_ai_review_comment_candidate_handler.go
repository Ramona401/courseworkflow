package handlers

// courseware_ai_review_comment_candidate_handler.go
//
// R-08 正式审核意见重新汇总HTTP入口。
//
// 路由：
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/comment-candidates
//       根据教师当前本次修改清单和当前审核意见生成不可变新候选。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/comment-candidates/{candidate_id}/apply
//       教师明确选择replace或append。
//       Service重新读取当前全部可信事实，stale时409并拒绝应用。
//
// 安全边界：
//
//   1. session_id和candidate_id只取URL；
//   2. actor只从JWT重新构建；
//   3. Generate正文只允许selected_item_ids和教师original_comment；
//   4. Apply正文只允许action、selected_item_ids和教师current_comment；
//   5. 浏览器永远不能提交candidate_text、diff、input_snapshot或input_hash；
//   6. Apply成功只返回新的输入框文本，不直接提交正式审核决定。

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// 教师意见允许到20000字符，因此HTTP上限留足UTF-8和JSON结构空间。
const coursewareAIReviewCommentCandidateBodyMaxBytes = 128 * 1024

type generateCWReviewCommentCandidateRequest struct {
	SelectedItemIDs []string `json:"selected_item_ids"`
	OriginalComment string   `json:"original_comment"`
}

type applyCWReviewCommentCandidateRequest struct {
	Action          string   `json:"action"`
	SelectedItemIDs []string `json:"selected_item_ids"`
	CurrentComment  string   `json:"current_comment"`
}

// isCoursewareAIReviewCommentCandidateRoute 判断是否属于R-08审核意见候选路径。
func isCoursewareAIReviewCommentCandidateRoute(
	parts []string,
) bool {
	if len(parts) < 2 ||
		len(parts) > 4 ||
		parts[0] == "items" ||
		parts[1] != "comment-candidates" {
		return false
	}

	if len(parts) == 2 {
		return true
	}

	return len(parts) == 4 &&
		strings.TrimSpace(parts[2]) != "" &&
		parts[3] == "apply"
}

// HandleReviewCommentCandidateRoute 处理Generate和教师明确Apply。
func (h *CoursewareAIReviewHandler) HandleReviewCommentCandidateRoute(
	w http.ResponseWriter,
	r *http.Request,
	parts []string,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	sessionID := ""
	if len(parts) > 0 {
		sessionID = strings.TrimSpace(parts[0])
	}

	switch {
	case len(parts) == 2 &&
		r.Method == http.MethodPost:
		h.generateReviewCommentCandidate(
			w,
			r,
			sessionID,
			actor,
		)

	case len(parts) == 4 &&
		parts[3] == "apply" &&
		r.Method == http.MethodPost:
		h.applyReviewCommentCandidate(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"审核意见候选路由或请求方法无效",
		)
	}
}

func (h *CoursewareAIReviewHandler) generateReviewCommentCandidate(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req generateCWReviewCommentCandidateRequest

	if !decodeCWReviewCommentCandidateRequest(
		w,
		r,
		&req,
	) {
		return
	}

	result, err :=
		h.runner.GenerateCWReviewCommentCandidate(
			r.Context(),
			sessionID,
			&services.CWReviewCommentCandidateGenerateInput{
				SelectedItemIDs: req.SelectedItemIDs,
				OriginalComment: req.OriginalComment,
			},
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewCommentCandidateView(
			result,
		),
	)
}

func (h *CoursewareAIReviewHandler) applyReviewCommentCandidate(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	candidateID string,
	actor *services.CoursewareActorContext,
) {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		utils.BadRequest(w, "缺少审核意见候选ID")
		return
	}

	var req applyCWReviewCommentCandidateRequest

	if !decodeCWReviewCommentCandidateRequest(
		w,
		r,
		&req,
	) {
		return
	}

	req.Action = strings.TrimSpace(req.Action)

	result, err :=
		h.runner.ApplyCWReviewCommentCandidate(
			r.Context(),
			sessionID,
			&services.CWReviewCommentCandidateApplyInput{
				CandidateID:     candidateID,
				Action:          req.Action,
				SelectedItemIDs: req.SelectedItemIDs,
				CurrentComment:  req.CurrentComment,
			},
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewCommentCandidateApplyView(
			result,
		),
	)
}

func decodeCWReviewCommentCandidateRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAIReviewCommentCandidateBodyMaxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误或内容过大",
		)
		return false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}
