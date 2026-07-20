package handlers

// review_v2_handler.go
//
// 教案多级审核 HTTP 处理器。
//
// 上下文 6：审核历史端点现在把调用者身份传给 Service，
// 由 Service 对 region_admin 执行辖区学校和教育域双重验证。

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

// ReviewV2Handler 多级审核处理器。
type ReviewV2Handler struct {
	reviewService *services.ReviewV2Service
}

// NewReviewV2Handler 创建处理器。
func NewReviewV2Handler(
	reviewService *services.ReviewV2Service,
) *ReviewV2Handler {
	return &ReviewV2Handler{
		reviewService: reviewService,
	}
}

// ReviewL1 POST /api/v1/reviews/{plan_id}/l1。
func (h *ReviewV2Handler) ReviewL1(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	planID := extractReviewPlanID(
		r.URL.Path,
		"/l1",
	)
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var req models.ReviewDecisionV2Request
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if err := h.reviewService.ReviewL1(
		r.Context(),
		planID,
		userID,
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

// ReviewL2 POST /api/v1/reviews/{plan_id}/l2。
func (h *ReviewV2Handler) ReviewL2(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	planID := extractReviewPlanID(
		r.URL.Path,
		"/l2",
	)
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var req models.ReviewDecisionV2Request
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if err := h.reviewService.ReviewL2(
		r.Context(),
		planID,
		claims.UserID,
		claims.Role,
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

// GetReviewHistory GET /api/v1/reviews/{plan_id}/history。
func (h *ReviewV2Handler) GetReviewHistory(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	planID := extractReviewPlanID(
		r.URL.Path,
		"/history",
	)
	if planID == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	result, err :=
		h.reviewService.GetReviewHistory(
			r.Context(),
			planID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		h.handleReviewError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetPendingReviews GET /api/v1/reviews/pending。
func (h *ReviewV2Handler) GetPendingReviews(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(
		query.Get("limit"),
	)
	offset, _ := strconv.Atoi(
		query.Get("offset"),
	)

	result, err :=
		h.reviewService.GetPendingReviews(
			r.Context(),
			claims.UserID,
			claims.Role,
			limit,
			offset,
		)
	if err != nil {
		utils.InternalError(
			w,
			"获取待审核列表失败",
		)
		return
	}

	utils.Success(w, result)
}

// GetReviewedRecords GET /api/v1/reviews/reviewed。
func (h *ReviewV2Handler) GetReviewedRecords(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
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

	result, err :=
		h.reviewService.GetReviewedRecords(
			r.Context(),
			claims.UserID,
			claims.Role,
			level,
			decision,
			limit,
			offset,
		)
	if err != nil {
		utils.InternalError(
			w,
			"获取已审核记录失败",
		)
		return
	}

	utils.Success(w, result)
}

// GetReviewStats GET /api/v1/reviews/stats。
func (h *ReviewV2Handler) GetReviewStats(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	level, _ := strconv.Atoi(
		r.URL.Query().Get("level"),
	)
	if level <= 0 {
		level = models.ReviewLevelL1
	}

	result, err :=
		h.reviewService.GetReviewStats(
			r.Context(),
			claims.UserID,
			claims.Role,
			level,
		)
	if err != nil {
		utils.InternalError(
			w,
			"获取审核统计失败",
		)
		return
	}

	utils.Success(w, result)
}

// GetReviewConfig GET /api/v1/review-config。
func (h *ReviewV2Handler) GetReviewConfig(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	schoolID := r.URL.Query().Get(
		"school_id",
	)
	if schoolID == "" {
		utils.BadRequest(
			w,
			"缺少 school_id 参数",
		)
		return
	}

	result, err :=
		h.reviewService.GetReviewFlowConfig(
			r.Context(),
			schoolID,
		)
	if err != nil {
		utils.InternalError(
			w,
			"获取审核配置失败",
		)
		return
	}

	utils.Success(w, result)
}

// UpdateReviewConfig PUT /api/v1/review-config。
func (h *ReviewV2Handler) UpdateReviewConfig(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var body struct {
		SchoolID string                               `json:"school_id"`
		Config   models.UpdateReviewFlowConfigRequest `json:"config"`
	}
	if err := json.NewDecoder(
		r.Body,
	).Decode(&body); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}
	if body.SchoolID == "" {
		utils.BadRequest(
			w,
			"缺少 school_id",
		)
		return
	}

	if err := h.reviewService.UpdateReviewFlowConfig(
		r.Context(),
		body.SchoolID,
		&body.Config,
		claims.UserID,
	); err != nil {
		utils.BadRequest(
			w,
			err.Error(),
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "审核配置更新成功",
		},
	)
}

// handleReviewError 统一映射审核业务错误。
func (h *ReviewV2Handler) handleReviewError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrReviewNotSubmitted,
	),
		errors.Is(
			err,
			services.ErrReviewNotL2Status,
		),
		errors.Is(
			err,
			services.ErrReviewContentEmpty,
		),
		errors.Is(
			err,
			services.ErrReviewInvalidDecision,
		):
		utils.BadRequest(w, err.Error())

	case errors.Is(
		err,
		services.ErrReviewNoPermission,
	):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrReviewPlanNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	default:
		utils.InternalError(
			w,
			"审核操作失败，请稍后重试",
		)
	}
}

// extractReviewPlanID 提取审核路径中的教案 ID。
func extractReviewPlanID(
	path string,
	suffix string,
) string {
	const prefix = "/api/v1/reviews/"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
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
