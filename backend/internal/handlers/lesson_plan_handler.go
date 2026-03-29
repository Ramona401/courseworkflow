package handlers

// 教案管理HTTP处理器
// 负责教案CRUD、状态流转、评审、Fork、提示词模板管理的HTTP接口

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// LessonPlanHandler 教案管理接口处理器
type LessonPlanHandler struct {
	lpService *services.LessonPlanService
}

// NewLessonPlanHandler 创建教案管理处理器实例
func NewLessonPlanHandler(lpService *services.LessonPlanService) *LessonPlanHandler {
	return &LessonPlanHandler{lpService: lpService}
}

// ==================== 教案列表 ====================

// ListLessonPlans 获取教案列表
// GET /api/v1/lesson-plans/plans?author_id=xx&group_id=xx&status=xx&subject=xx&grade=xx&limit=20&offset=0
func (h *LessonPlanHandler) ListLessonPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	q := r.URL.Query()
	authorID := q.Get("author_id")
	groupID := q.Get("group_id")
	status := q.Get("status")
	subject := q.Get("subject")
	grade := q.Get("grade")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	result, err := h.lpService.ListLessonPlans(r.Context(), authorID, groupID, status, subject, grade, limit, offset)
	if err != nil {
		log.Printf("获取教案列表失败: %v", err)
		utils.InternalError(w, "获取教案列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 创建教案 ====================

// CreateLessonPlan 创建教案
// POST /api/v1/lesson-plans/plans
func (h *LessonPlanHandler) CreateLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CreateLessonPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	lp, err := h.lpService.CreateLessonPlan(r.Context(), &req, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, lp)
}

// ==================== 获取教案详情 ====================

// GetLessonPlan 获取教案详情
// GET /api/v1/lesson-plans/plans/{id}
func (h *LessonPlanHandler) GetLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	detail, err := h.lpService.GetLessonPlan(r.Context(), id)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, detail)
}

// ==================== 更新教案 ====================

// UpdateLessonPlan 更新教案内容
// PUT /api/v1/lesson-plans/plans/{id}
func (h *LessonPlanHandler) UpdateLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)

	var req models.UpdateLessonPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.lpService.UpdateLessonPlan(r.Context(), id, userID, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除教案 ====================

// DeleteLessonPlan 删除教案
// DELETE /api/v1/lesson-plans/plans/{id}
func (h *LessonPlanHandler) DeleteLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)

	if err := h.lpService.DeleteLessonPlan(r.Context(), id, userID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 教案状态操作 ====================

// PublishPersonal 个人发布
// POST /api/v1/lesson-plans/plans/{id}/publish-personal
func (h *LessonPlanHandler) PublishPersonal(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/publish-personal")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if err := h.lpService.PublishPersonal(r.Context(), id, userID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "个人发布成功"})
}

// SubmitForReview 提交评审
// POST /api/v1/lesson-plans/plans/{id}/submit-review
func (h *LessonPlanHandler) SubmitForReview(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/submit-review")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)

	var req models.SubmitLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.lpService.SubmitForReview(r.Context(), id, userID, req.GroupID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "已提交评审"})
}

// ReviewLessonPlan 评审教案
// POST /api/v1/lesson-plans/plans/{id}/review
func (h *LessonPlanHandler) ReviewLessonPlan(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/review")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)

	var req models.CreateLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.lpService.ReviewLessonPlan(r.Context(), id, userID, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "评审完成"})
}

// PublishShared 共享发布
// POST /api/v1/lesson-plans/plans/{id}/publish-shared
func (h *LessonPlanHandler) PublishShared(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/publish-shared")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	if err := h.lpService.PublishShared(r.Context(), id, userID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "共享发布成功"})
}

// StartDevelopment 进入课件开发
// POST /api/v1/lesson-plans/plans/{id}/start-development
func (h *LessonPlanHandler) StartDevelopment(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/start-development")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)
	result, err := h.lpService.StartDevelopment(r.Context(), id, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	// Phase6：返回 pipeline_id，前端可跳转到 /workflow/pipelines/{pipeline_id}
	utils.Success(w, result)
}

// ForkLessonPlan Fork教案
// POST /api/v1/lesson-plans/plans/{id}/fork
func (h *LessonPlanHandler) ForkLessonPlan(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/fork")
	if id == "" {
		utils.BadRequest(w, "缺少教案ID")
		return
	}
	userID := getCurrentUserID(r)

	newLP, err := h.lpService.ForkLessonPlan(r.Context(), id, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, newLP)
}

// ==================== 提示词模板管理 ====================

// ListPromptTemplates 获取模板列表
// GET /api/v1/lesson-plans/templates?level=xxx&owner_id=xxx
func (h *LessonPlanHandler) ListPromptTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	q := r.URL.Query()
	level := q.Get("level")
	ownerID := q.Get("owner_id")

	result, err := h.lpService.ListPromptTemplates(r.Context(), level, ownerID)
	if err != nil {
		log.Printf("获取模板列表失败: %v", err)
		utils.InternalError(w, "获取模板列表失败")
		return
	}
	utils.Success(w, result)
}

// CreatePromptTemplate 创建模板
// POST /api/v1/lesson-plans/templates
func (h *LessonPlanHandler) CreatePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	userID := getCurrentUserID(r)

	var req models.CreatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	pt, err := h.lpService.CreatePromptTemplate(r.Context(), &req, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, pt)
}

// GetPromptTemplate 获取模板详情
// GET /api/v1/lesson-plans/templates/{id}
func (h *LessonPlanHandler) GetPromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}
	pt, err := h.lpService.GetPromptTemplate(r.Context(), id)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, pt)
}

// UpdatePromptTemplate 更新模板
// PUT /api/v1/lesson-plans/templates/{id}
func (h *LessonPlanHandler) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	id := extractTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}

	var req models.UpdatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.lpService.UpdatePromptTemplate(r.Context(), id, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ResolvePromptTemplate 解析继承链
// GET /api/v1/lesson-plans/templates/{id}/resolved
func (h *LessonPlanHandler) ResolvePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	prefix := "/api/v1/lesson-plans/templates/"
	suffix := "/resolved"
	id := extractMiddleSegment(r.URL.Path, prefix, suffix)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}
	resolved, err := h.lpService.ResolvePromptTemplate(r.Context(), id)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, resolved)
}

// ==================== 错误处理 ====================

func (h *LessonPlanHandler) handleLPError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrLPTitleRequired),
		errors.Is(err, services.ErrLPSubjectRequired),
		errors.Is(err, services.ErrLPGradeRequired),
		errors.Is(err, services.ErrLPTopicRequired),
		errors.Is(err, services.ErrLPGroupRequired),
		errors.Is(err, services.ErrTemplateNameRequired),
		errors.Is(err, services.ErrTemplateLevelInvalid):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrLPNotAuthor),
		errors.Is(err, services.ErrLPCannotEdit),
		errors.Is(err, services.ErrLPCannotSubmit),
		errors.Is(err, services.ErrLPCannotDevelop),
		errors.Is(err, services.ErrLPAlreadyDeveloping):
		utils.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrLPNotFound),
		errors.Is(err, services.ErrTemplateNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("教案操作失败: %v", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}

// ==================== 辅助函数 ====================

// extractLPID 从教案URL路径中提取ID
func extractLPID(path string) string {
	prefix := "/api/v1/lesson-plans/plans/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if idx := strings.Index(id, "/"); idx > 0 {
		id = id[:idx]
	}
	return id
}

// extractLPMiddleID 从教案子路径URL中提取ID
// 示例：/api/v1/lesson-plans/plans/{id}/publish-personal → id
func extractLPMiddleID(path string, suffix string) string {
	prefix := "/api/v1/lesson-plans/plans/"
	return extractMiddleSegment(path, prefix, suffix)
}

// extractTemplateID 从模板URL路径中提取ID
func extractTemplateID(path string) string {
	prefix := "/api/v1/lesson-plans/templates/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if idx := strings.Index(id, "/"); idx > 0 {
		id = id[:idx]
	}
	return id
}
