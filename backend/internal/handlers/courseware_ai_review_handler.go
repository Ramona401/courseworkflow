package handlers

// courseware_ai_review_handler.go
//
// 课件 AI 审核助手 HTTP 处理器。
//
// 会话API：
//   POST /api/v1/courseware-ai-reviews
//        创建并准备新的AI审核会话。
//
//   GET /api/v1/courseware-ai-reviews
//        按courseware_id和review_level查询当前审核员最新会话。
//
//   GET /api/v1/courseware-ai-reviews/{session_id}
//        查询指定会话、批次和最终报告。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/run-next
//        顺序运行下一批页面审核。
//
//   POST /api/v1/courseware-ai-reviews/{session_id}/finalize
//        全部批次完成后生成最终综合报告。
//
// 全局讨论和AI建议治理API由
// courseware_ai_review_global_discussion_handler.go处理。
//
// 整改项API由courseware_ai_review_item_handler.go处理。
//
// 权限原则：
//   - 所有操作者都从JWT重新构建CoursewareActorContext；
//   - 不接受前端提供教育域、学科、年级、教案ID或材料正文；
//   - 后端重新读取课件和配置允许使用的来源材料；
//   - AI结果不能自动提交人工审核决定；
//   - 单条或全局讨论都不能直接修改页面；
//   - 正式交付的历史整改项不能被治理接口改写。
//
// R-02 启动配置：
//   - 至少选择一个审核维度；
//   - 自定义维度必须填写说明；
//   - 旧客户端未提交配置时使用现行兼容预设；
//   - 请求正文严格限长、拒绝未知字段和尾随JSON；
//   - 浏览器不能提交配置哈希或可信材料状态。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareAIReviewPrepareBodyMaxBytes = 64 * 1024

// CoursewareAIReviewHandler 课件AI审核HTTP入口。
type CoursewareAIReviewHandler struct {
	service *services.CoursewareAIReviewService
	runner  *services.CoursewareAIReviewRunner
}

// NewCoursewareAIReviewHandler 创建处理器。
func NewCoursewareAIReviewHandler(
	service *services.CoursewareAIReviewService,
	runner *services.CoursewareAIReviewRunner,
) *CoursewareAIReviewHandler {
	return &CoursewareAIReviewHandler{
		service: service,
		runner:  runner,
	}
}

// PrepareRequest 创建课件AI分析会话请求。
//
// ReviewLevel用途：
//   - 0：课件作者自审；
//   - 1：L1正式审核辅助；
//   - 2：L2正式审核辅助。
//
// 配置字段使用指针区分旧客户端字段缺失和明确提交空数组。
type PrepareRequest struct {
	CoursewareID string `json:"courseware_id"`
	ReviewLevel  int    `json:"review_level"`
	AssistantID  string `json:"assistant_id"`

	ReviewDimensions           *[]string `json:"review_dimensions"`
	CustomDimensionDescription *string   `json:"custom_dimension_description"`
	LessonReferenceMode        *string   `json:"lesson_reference_mode"`
}

// HandleCollection 处理集合级POST创建和GET最新会话查询。
func (h *CoursewareAIReviewHandler) HandleCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodPost:
		h.Prepare(w, r)

	case http.MethodGet:
		h.GetLatest(w, r)

	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET或POST请求")
	}
}

// HandleItem 处理会话、全局讨论、关系治理和整改项通配子路由。
func (h *CoursewareAIReviewHandler) HandleItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	parts := parseCoursewareAIReviewPathParts(r.URL.Path)

	if isCoursewareAIReviewGlobalDiscussionRoute(parts) {
		h.HandleGlobalDiscussionRoute(w, r, parts)
		return
	}

	if isCoursewareAIReviewRelationRoute(parts) {
		h.HandleReviewItemRelationRoute(w, r, parts)
		return
	}

	if isCoursewareAIReviewItemRoute(parts) {
		h.HandleReviewItemRoute(w, r, parts)
		return
	}

	sessionID, action := parseCoursewareAIReviewPath(r.URL.Path)
	if sessionID == "" {
		utils.BadRequest(w, "缺少AI审核会话ID")
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		h.GetSession(w, r, sessionID)

	case action == "run-next" && r.Method == http.MethodPost:
		h.RunNext(w, r, sessionID)

	case action == "finalize" && r.Method == http.MethodPost:
		h.Finalize(w, r, sessionID)

	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "AI审核会话路由或请求方法无效")
	}
}

// Prepare 创建审核会话、生成页面索引并规划顺序批次。
func (h *CoursewareAIReviewHandler) Prepare(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	var req PrepareRequest
	if !decodeCoursewareAIReviewPrepareRequest(w, r, &req) {
		return
	}

	req.CoursewareID = strings.TrimSpace(req.CoursewareID)
	req.AssistantID = strings.TrimSpace(req.AssistantID)

	if req.CoursewareID == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if req.ReviewLevel != models.CWAIReviewLevelSelf &&
		req.ReviewLevel != models.ReviewLevelL1 &&
		req.ReviewLevel != models.ReviewLevelL2 {
		utils.BadRequest(w, "AI分析用途必须为0、1或2")
		return
	}

	session, err := h.service.PrepareSession(
		r.Context(),
		req.CoursewareID,
		req.ReviewLevel,
		actor,
		req.AssistantID,
		&services.CWAIReviewConfigInput{
			ReviewDimensions:           req.ReviewDimensions,
			CustomDimensionDescription: req.CustomDimensionDescription,
			LessonReferenceMode:        req.LessonReferenceMode,
		},
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	fullSession, batches, err := h.service.GetSessionDetail(
		r.Context(),
		session.ID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewBundleView(fullSession, batches),
	)
}

// GetLatest 查询当前审核员对课件的最新一次AI审核。
func (h *CoursewareAIReviewHandler) GetLatest(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	coursewareID := strings.TrimSpace(r.URL.Query().Get("courseware_id"))
	if coursewareID == "" {
		utils.BadRequest(w, "缺少courseware_id")
		return
	}

	reviewLevel, _ := strconv.Atoi(r.URL.Query().Get("review_level"))
	if reviewLevel != models.CWAIReviewLevelSelf &&
		reviewLevel != models.ReviewLevelL1 &&
		reviewLevel != models.ReviewLevelL2 {
		utils.BadRequest(w, "review_level必须为0、1或2")
		return
	}

	session, batches, err := h.service.GetLatestSessionDetail(
		r.Context(),
		coursewareID,
		reviewLevel,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewBundleView(session, batches),
	)
}

// GetSession 查询指定会话详情。
func (h *CoursewareAIReviewHandler) GetSession(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	if h == nil || h.service == nil {
		utils.InternalError(w, "课件AI审核服务未初始化")
		return
	}

	session, batches, err := h.service.GetSessionDetail(
		r.Context(),
		sessionID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewBundleView(session, batches),
	)
}

// RunNext 顺序执行下一批页面审核。
func (h *CoursewareAIReviewHandler) RunNext(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
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

	result, err := h.runner.RunNextBatch(
		r.Context(),
		sessionID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(w, buildCoursewareAIReviewRunNextView(result))
}

// Finalize 生成最终综合与风险回看报告。
func (h *CoursewareAIReviewHandler) Finalize(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
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

	result, err := h.runner.Finalize(
		r.Context(),
		sessionID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(w, buildCoursewareAIReviewFinalizeView(result))
}

// handleError 将服务错误映射为稳定HTTP状态。
func (h *CoursewareAIReviewHandler) handleError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, services.ErrCWAIReviewActorRequired):
		utils.Unauthorized(w, "未登录")

	case errors.Is(err, services.ErrCWAIReviewNoPermission),
		errors.Is(err, services.ErrCWAIReviewSessionOwnerMismatch),
		errors.Is(err, services.ErrCWReviewItemNotDelivered),
		errors.Is(err, services.ErrCoursewareEducationDomainMismatch),
		errors.Is(err, services.ErrAssistantPermDenied),
		errors.Is(err, services.ErrAssistantEducationDomainMismatch),
		errors.Is(err, services.ErrCoursewareOwnerRuntimeDenied),
		errors.Is(err, services.ErrCoursewareEditDenied):
		utils.Fail(
			w,
			http.StatusForbidden,
			"您没有访问或操作此课件AI审核整改项的权限",
		)

	case errors.Is(err, services.ErrCWAIReviewCoursewareNotFound),
		errors.Is(err, services.ErrCWAIReviewSessionNotFound),
		errors.Is(err, repository.ErrCoursewareReviewItemNotFound),
		errors.Is(err, repository.ErrCoursewareReviewItemRelationNotFound),
		errors.Is(err, repository.ErrCoursewareAIReviewMessageNotFound),
		errors.Is(err, repository.ErrAIAssistantNotFound),
		errors.Is(err, services.ErrCoursewareAccessNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())

	case errors.Is(err, services.ErrCWAIReviewSnapshotExpired),
		errors.Is(err, services.ErrCWAIReviewBatchBusy),
		errors.Is(err, services.ErrCWAIReviewSessionNotRunnable),
		errors.Is(err, services.ErrCWReviewItemSessionNotDone),
		errors.Is(err, services.ErrCWReviewItemStale),
		errors.Is(err, services.ErrCWReviewItemOrphaned),
		errors.Is(err, services.ErrCWReviewItemNotActionable),
		errors.Is(err, services.ErrCWAIReviewGlobalNotActionable),
		errors.Is(err, services.ErrCWAIReviewGlobalMessageLimit),
		errors.Is(err, services.ErrCWAIReviewGlobalProposalNotFound),
		errors.Is(err, services.ErrCWAIReviewGlobalRelationNotSuggested),
		errors.Is(err, services.ErrCWAIReviewGlobalDismissNotSuggested),
		errors.Is(err, repository.ErrCoursewareReviewItemConflict):
		utils.Fail(w, http.StatusConflict, err.Error())

	case errors.Is(err, services.ErrCWAIReviewConfigInvalid),
		errors.Is(err, services.ErrCWAIReviewInvalidLevel),
		errors.Is(err, services.ErrCWAIReviewNoPages),
		errors.Is(err, services.ErrCWAIReviewLessonPlanMissing),
		errors.Is(err, services.ErrCWAIReviewLessonDomainMismatch),
		errors.Is(err, services.ErrCWAIReviewPromptUnavailable),
		errors.Is(err, services.ErrCWReviewItemFindingRequired),
		errors.Is(err, services.ErrCWReviewItemFindingNotFound),
		errors.Is(err, services.ErrCWReviewItemContentTooLong),
		errors.Is(err, services.ErrCWReviewItemInstructionInvalid),
		errors.Is(err, services.ErrCWReviewItemDismissReasonInvalid),
		errors.Is(err, services.ErrCWAIReviewGlobalContentInvalid),
		errors.Is(err, services.ErrCWAIReviewGlobalSelectionInvalid),
		errors.Is(err, services.ErrCWAIReviewGlobalManualItemInvalid),
		errors.Is(err, services.ErrCWAIReviewGlobalRelationInvalid),
		errors.Is(err, services.ErrCWAIReviewGlobalRelationReasonInvalid),
		errors.Is(err, services.ErrCWAIReviewManualRelationInvalid),
		errors.Is(err, services.ErrCWAIReviewManualRelationExplanationInvalid),
		errors.Is(err, repository.ErrAIAssistantInactive),
		errors.Is(err, services.ErrAssistantManualMismatch),
		errors.Is(err, services.ErrCoursewareRuntimeDomainRequired):
		utils.BadRequest(w, err.Error())

	default:
		utils.InternalError(w, "课件AI审核操作失败，请稍后重试")
	}
}

func decodeCoursewareAIReviewPrepareRequest(
	w http.ResponseWriter,
	r *http.Request,
	target *PrepareRequest,
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAIReviewPrepareBodyMaxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		utils.BadRequest(w, "请求参数格式错误或内容过大")
		return false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		utils.BadRequest(w, "请求正文只能包含一个JSON对象")
		return false
	}

	return true
}

// buildCoursewareAIReviewActor 从JWT构建可信课件操作者。
func buildCoursewareAIReviewActor(
	r *http.Request,
) (*services.CoursewareActorContext, bool) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		return nil, false
	}

	actor := services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, false
	}

	return actor, true
}

// parseCoursewareAIReviewPathParts 返回AI审核通配路径的非空片段。
func parseCoursewareAIReviewPathParts(path string) []string {
	const prefix = "/api/v1/courseware-ai-reviews/"

	if !strings.HasPrefix(path, prefix) {
		return []string{}
	}

	rest := strings.Trim(
		strings.TrimPrefix(path, prefix),
		"/",
	)
	if rest == "" {
		return []string{}
	}

	rawParts := strings.Split(rest, "/")
	parts := make([]string, 0, len(rawParts))

	for _, raw := range rawParts {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}

		parts = append(parts, value)
	}

	return parts
}

// isCoursewareAIReviewItemRoute 判断是否属于整改项子路由。
func isCoursewareAIReviewItemRoute(parts []string) bool {
	if len(parts) == 0 {
		return false
	}

	if parts[0] == "items" {
		return true
	}

	return len(parts) == 2 &&
		parts[1] == "items"
}

// parseCoursewareAIReviewPath 解析普通会话ID和动作。
func parseCoursewareAIReviewPath(path string) (string, string) {
	parts := parseCoursewareAIReviewPathParts(path)

	if len(parts) == 1 {
		return parts[0], ""
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", ""
}
