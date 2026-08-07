package handlers

// courseware_review_handler.go — 课件多级审核HTTP处理器。
//
// 正式审核决定请求可以额外携带：
//   - ai_review_session_id：当前审核员已完成的AI审核会话；
//   - review_item_ids：审核员明确选中并交付作者的整改项。
//
// 后端会重新校验全部关联，不接收前端提供的AI报告正文。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareReviewHandler 课件多级审核处理器。
type CoursewareReviewHandler struct {
	reviewService *services.CoursewareReviewService
	cwService     *services.CoursewareService
}

// NewCoursewareReviewHandler 创建课件审核处理器。
func NewCoursewareReviewHandler(
	reviewService *services.CoursewareReviewService,
	cwService *services.CoursewareService,
) *CoursewareReviewHandler {
	return &CoursewareReviewHandler{
		reviewService: reviewService,
		cwService:     cwService,
	}
}

// SubmitForReview POST /api/v1/coursewares/{id}/submit-review。
func (h *CoursewareReviewHandler) SubmitForReview(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCWReviewSubmitID(
		r.URL.Path,
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	if err := h.reviewService.SubmitForReview(
		r.Context(),
		id,
		actor,
	); err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "已提交审核",
		},
	)
}

// ReviewL1 POST /api/v1/courseware-reviews/{id}/l1。
func (h *CoursewareReviewHandler) ReviewL1(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	id := extractCWReviewID(
		r.URL.Path,
		"/l1",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CWReviewDecisionRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	if err := h.reviewService.ReviewL1(
		r.Context(),
		id,
		actor,
		&req,
	); err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "L1审核完成",
		},
	)
}

// ReviewL2 POST /api/v1/courseware-reviews/{id}/l2。
func (h *CoursewareReviewHandler) ReviewL2(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	id := extractCWReviewID(
		r.URL.Path,
		"/l2",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CWReviewDecisionRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误",
		)
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	if err := h.reviewService.ReviewL2(
		r.Context(),
		id,
		actor,
		&req,
	); err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "L2审核完成",
		},
	)
}

// GetReviewHistory GET /api/v1/courseware-reviews/{id}/history。
func (h *CoursewareReviewHandler) GetReviewHistory(
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

	id := extractCWReviewID(
		r.URL.Path,
		"/history",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	result, err :=
		h.reviewService.GetReviewHistory(
			r.Context(),
			id,
			actor,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetReviewDetail GET /api/v1/courseware-reviews/{id}/detail。
func (h *CoursewareReviewHandler) GetReviewDetail(
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

	id := extractCWReviewID(
		r.URL.Path,
		"/detail",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	result, err :=
		h.reviewService.GetReviewDetail(
			r.Context(),
			id,
			actor,
			h.cwService,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetPendingReviews GET /api/v1/courseware-reviews/pending。
func (h *CoursewareReviewHandler) GetPendingReviews(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)
	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	result, err :=
		h.reviewService.GetPendingReviews(
			r.Context(),
			actor,
			limit,
			offset,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetReviewedRecords GET /api/v1/courseware-reviews/reviewed。
func (h *CoursewareReviewHandler) GetReviewedRecords(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	query := r.URL.Query()
	level, _ := strconv.Atoi(
		query.Get("level"),
	)
	if level <= 0 {
		level = models.ReviewLevelL1
	}

	decision := query.Get("decision")
	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)
	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	result, err :=
		h.reviewService.GetReviewedRecords(
			r.Context(),
			actor,
			level,
			decision,
			limit,
			offset,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetReviewStats GET /api/v1/courseware-reviews/stats。
func (h *CoursewareReviewHandler) GetReviewStats(
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

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	level, _ := strconv.Atoi(
		r.URL.Query().Get("level"),
	)
	if level <= 0 {
		level = models.ReviewLevelL1
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	result, err :=
		h.reviewService.GetReviewStats(
			r.Context(),
			actor,
			level,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

func (h *CoursewareReviewHandler) handleReviewError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCWReviewCoursewareNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCWReviewNoPermission,
	),
		errors.Is(
			err,
			services.ErrCWSubmitNotOwner,
		),
		errors.Is(
			err,
			services.ErrCoursewareActorRequired,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			"您没有访问或审核此课件的权限",
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

	case errors.Is(
		err,
		services.ErrCWReviewNotSubmitted,
	),
		errors.Is(
			err,
			services.ErrCWReviewNotL2Status,
		),
		errors.Is(
			err,
			services.ErrCWReviewInvalidDecision,
		),
		errors.Is(
			err,
			services.ErrCWReviewFeedbackInvalid,
		),
		errors.Is(
			err,
			services.ErrCWSubmitNotReady,
		),
		errors.Is(
			err,
			services.ErrCWSubmitWrongState,
		),
		errors.Is(
			err,
			services.ErrCWSubmitNoSchool,
		),
		errors.Is(
			err,
			services.ErrCWSubmitRemediationIncomplete,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	default:
		utils.InternalError(
			w,
			"审核操作失败，请稍后重试",
		)
	}
}

func extractCWReviewID(
	path string,
	suffix string,
) string {
	const prefix = "/api/v1/courseware-reviews/"

	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := strings.TrimPrefix(
		path,
		prefix,
	)
	if index := strings.Index(
		rest,
		"/",
	); index > 0 {
		candidate := rest[:index]
		tail := rest[index:]

		if strings.HasPrefix(
			tail,
			suffix,
		) {
			return candidate
		}
	}

	return ""
}

func extractCWReviewSubmitID(
	path string,
) string {
	const prefix = "/api/v1/coursewares/"
	const suffix = "/submit-review"

	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := strings.TrimPrefix(
		path,
		prefix,
	)
	if index := strings.Index(
		rest,
		"/",
	); index > 0 {
		candidate := rest[:index]
		tail := rest[index:]

		if strings.HasPrefix(
			tail,
			suffix,
		) {
			return candidate
		}
	}

	return ""
}
