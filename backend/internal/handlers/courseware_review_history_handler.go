package handlers

// courseware_review_history_handler.go
//
// R-03“已审核记录只读详情”HTTP入口。
//
// 本文件只提供GET，不挂载任何业务写操作。
// 不存在与越权统一返回404，避免通过review_id探测其他学校或教育域审核记录。

import (
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GetReviewHistoryDetail
// GET /api/v1/courseware-reviews/records/{review_id}/detail。
func (h *CoursewareReviewHandler) GetReviewHistoryDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	reviewID :=
		extractCWReviewedRecordDetailID(
			r.URL.Path,
		)
	if reviewID == "" {
		utils.BadRequest(
			w,
			"审核记录路径无效",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	result, err :=
		h.reviewService.GetReviewHistoryDetail(
			r.Context(),
			reviewID,
			actor,
		)
	if err != nil {
		h.handleReviewHistoryDetailError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}

func (h *CoursewareReviewHandler) handleReviewHistoryDetailError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCWReviewHistoryDetailNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"审核记录不存在",
		)

	case errors.Is(
		err,
		services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		utils.InternalError(
			w,
			"课件教育域异常，请联系管理员处理",
		)

	default:
		utils.InternalError(
			w,
			"读取审核历史失败，请稍后重试",
		)
	}
}

// extractCWReviewedRecordDetailID 只接受精确的records/{review_id}/detail结构。
//
// 旧接口/courseware-reviews/{courseware_id}/detail继续由原extractCWReviewID处理，
// 两种ID语义不得混用。
func extractCWReviewedRecordDetailID(
	path string,
) string {
	const prefix = "/api/v1/courseware-reviews/records/"
	const suffix = "/detail"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return ""
	}

	rest :=
		strings.TrimPrefix(
			path,
			prefix,
		)

	if !strings.HasSuffix(
		rest,
		suffix,
	) {
		return ""
	}

	candidate :=
		strings.TrimSpace(
			strings.TrimSuffix(
				rest,
				suffix,
			),
		)

	if candidate == "" ||
		strings.Contains(
			candidate,
			"/",
		) {
		return ""
	}

	return candidate
}
