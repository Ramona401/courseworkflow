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

func (h *LessonPlanHandler) ListLessonPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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
	qualityLevel, _ := strconv.Atoi(q.Get("quality_level"))
	structureType, _ := strconv.Atoi(q.Get("structure_type"))
	cognitiveLevel, _ := strconv.Atoi(q.Get("cognitive_level"))
	pedagogyIntensity, _ := strconv.Atoi(q.Get("pedagogy_intensity"))

	result, err := h.lpService.ListLessonPlans(r.Context(), authorID, groupID, status, subject, grade, limit, offset, qualityLevel, structureType, cognitiveLevel, pedagogyIntensity)
	if err != nil {
		log.Printf("获取教案列表失败: %v", err)
		utils.InternalError(w, "获取教案列表失败")
		return
	}
	utils.Success(w, result)
}

// ==================== 创建教案 ====================

func (h *LessonPlanHandler) CreateLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.CreateLessonPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *LessonPlanHandler) GetLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
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

func (h *LessonPlanHandler) UpdateLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	var req models.UpdateLessonPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.UpdateLessonPlan(r.Context(), id, userID, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除教案 ====================

func (h *LessonPlanHandler) DeleteLessonPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
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

func (h *LessonPlanHandler) PublishPersonal(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/publish-personal")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if err := h.lpService.PublishPersonal(r.Context(), id, userID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "个人发布成功"})
}

func (h *LessonPlanHandler) SubmitForReview(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/submit-review")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	var req models.SubmitLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.SubmitForReview(r.Context(), id, userID, req.GroupID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "已提交评审"})
}

func (h *LessonPlanHandler) ReviewLessonPlan(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/review")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	var req models.CreateLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.ReviewLessonPlan(r.Context(), id, userID, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "评审完成"})
}

func (h *LessonPlanHandler) PublishShared(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/publish-shared")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	if err := h.lpService.PublishShared(r.Context(), id, userID); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "共享发布成功"})
}

func (h *LessonPlanHandler) StartDevelopment(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/start-development")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}
	userID := getCurrentUserID(r)
	result, err := h.lpService.StartDevelopment(r.Context(), id, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *LessonPlanHandler) ForkLessonPlan(w http.ResponseWriter, r *http.Request) {
	id := extractLPMiddleID(r.URL.Path, "/fork")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
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

func (h *LessonPlanHandler) ListPromptTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *LessonPlanHandler) CreatePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	userID := getCurrentUserID(r)
	var req models.CreatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	pt, err := h.lpService.CreatePromptTemplate(r.Context(), &req, userID)
	if err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, pt)
}

func (h *LessonPlanHandler) GetPromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *LessonPlanHandler) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	id := extractTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}
	var req models.UpdatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.lpService.UpdatePromptTemplate(r.Context(), id, &req); err != nil {
		h.handleLPError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

func (h *LessonPlanHandler) ResolvePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func extractLPMiddleID(path string, suffix string) string {
	prefix := "/api/v1/lesson-plans/plans/"
	return extractMiddleSegment(path, prefix, suffix)
}

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
