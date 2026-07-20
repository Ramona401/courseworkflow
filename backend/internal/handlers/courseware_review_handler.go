package handlers

// courseware_review_handler.go — 课件多级审核 HTTP 处理器（阶段3）
//
// 镜像 review_v2_handler.go。路由前缀采用 /api/v1/courseware-reviews/，
// 与教案的 /api/v1/reviews/ 物理隔离，避免两套审核中心路由冲突。
//
// 端点：
//   POST /api/v1/coursewares/{id}/submit-review            提交审核（作者，挂在课件子路由，见 routes_courseware.go）
//   POST /api/v1/courseware-reviews/{id}/l1                L1 教研组审核
//   POST /api/v1/courseware-reviews/{id}/l2                L2 学校审核
//   GET  /api/v1/courseware-reviews/{id}/history           审核历史
//   GET  /api/v1/courseware-reviews/{id}/detail            审核详情（课件+批注+历史，决策二联动批注）
//   GET  /api/v1/courseware-reviews/pending                待审核列表
//   GET  /api/v1/courseware-reviews/reviewed               已审核记录列表
//   GET  /api/v1/courseware-reviews/stats                  审核统计
//
// 审核流程配置【复用】教案 /api/v1/review-config 接口（同一张 review_flow_configs 表），
// 本处理器不重复实现配置读写。

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

// CoursewareReviewHandler 课件多级审核处理器
// 持有审核服务 + 课件服务引用（审核详情需要课件服务装配课件详情）。
type CoursewareReviewHandler struct {
	reviewService *services.CoursewareReviewService
	cwService     *services.CoursewareService
}

// NewCoursewareReviewHandler 创建课件多级审核处理器实例
func NewCoursewareReviewHandler(reviewService *services.CoursewareReviewService, cwService *services.CoursewareService) *CoursewareReviewHandler {
	return &CoursewareReviewHandler{reviewService: reviewService, cwService: cwService}
}

// ==================== 提交审核（作者发起，挂在课件子路由）====================

// SubmitForReview POST /api/v1/coursewares/{id}/submit-review
func (h *CoursewareReviewHandler) SubmitForReview(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCWReviewSubmitID(r.URL.Path)
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
	utils.Success(w, map[string]string{"message": "已提交审核"})
}

// ==================== L1 / L2 审核 ====================

// ReviewL1 POST /api/v1/courseware-reviews/{id}/l1
func (h *CoursewareReviewHandler) ReviewL1(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractCWReviewID(r.URL.Path, "/l1")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req models.CWReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
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
	utils.Success(w, map[string]string{"message": "L1审核完成"})
}

// ReviewL2 POST /api/v1/courseware-reviews/{id}/l2
func (h *CoursewareReviewHandler) ReviewL2(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractCWReviewID(r.URL.Path, "/l2")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req models.CWReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
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
	utils.Success(w, map[string]string{"message": "L2审核完成"})
}

// ==================== 审核历史 ====================

// GetReviewHistory GET /api/v1/courseware-reviews/{id}/history
func (h *CoursewareReviewHandler) GetReviewHistory(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCWReviewID(r.URL.Path, "/history")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	result, err := h.reviewService.GetReviewHistory(
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

// ==================== 审核详情（决策二：联动批注）====================

// GetReviewDetail GET /api/v1/courseware-reviews/{id}/detail
func (h *CoursewareReviewHandler) GetReviewDetail(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractCWReviewID(r.URL.Path, "/detail")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	result, err := h.reviewService.GetReviewDetail(
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

// ==================== 待审核列表 ====================

// GetPendingReviews GET /api/v1/courseware-reviews/pending
func (h *CoursewareReviewHandler) GetPendingReviews(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	result, err := h.reviewService.GetPendingReviews(
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

// ==================== 已审核记录列表 ====================

// GetReviewedRecords GET /api/v1/courseware-reviews/reviewed
// 参数：level(1/2), decision(approved/revision/空=全部), limit, offset
func (h *CoursewareReviewHandler) GetReviewedRecords(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	query := r.URL.Query()
	level, _ := strconv.Atoi(query.Get("level"))
	if level <= 0 {
		level = models.ReviewLevelL1
	}
	decision := query.Get("decision")
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	result, err := h.reviewService.GetReviewedRecords(
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

// ==================== 审核统计 ====================

// GetReviewStats GET /api/v1/courseware-reviews/stats
func (h *CoursewareReviewHandler) GetReviewStats(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	level, _ := strconv.Atoi(r.URL.Query().Get("level"))
	if level <= 0 {
		level = models.ReviewLevelL1
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	result, err := h.reviewService.GetReviewStats(
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

// ==================== 错误映射 ====================

func (h *CoursewareReviewHandler) handleReviewError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCWReviewCoursewareNotFound,
	):
		utils.Fail(w, http.StatusNotFound, err.Error())

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
			services.ErrCWSubmitNotReady,
		),
		errors.Is(
			err,
			services.ErrCWSubmitWrongState,
		),
		errors.Is(
			err,
			services.ErrCWSubmitNoSchool,
		):
		utils.BadRequest(w, err.Error())

	default:
		utils.InternalError(
			w,
			"审核操作失败，请稍后重试",
		)
	}
}

// ==================== 路径解析辅助 ====================

// extractCWReviewID 从 /api/v1/courseware-reviews/{id}/{suffix} 提取课件ID
func extractCWReviewID(path string, suffix string) string {
	prefix := "/api/v1/courseware-reviews/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(rest, "/"); idx > 0 {
		candidate := rest[:idx]
		tail := rest[idx:]
		if strings.HasPrefix(tail, suffix) {
			return candidate
		}
	}
	return ""
}

// extractCWReviewSubmitID 从 /api/v1/coursewares/{id}/submit-review 提取课件ID
func extractCWReviewSubmitID(path string) string {
	prefix := "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	const suffix = "/submit-review"
	if idx := strings.Index(rest, "/"); idx > 0 {
		candidate := rest[:idx]
		tail := rest[idx:]
		if strings.HasPrefix(tail, suffix) {
			return candidate
		}
	}
	return ""
}
