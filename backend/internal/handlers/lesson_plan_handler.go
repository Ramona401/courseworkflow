package handlers

// lesson_plan_handler.go — 教案基础CRUD HTTP处理器
//
// 本文件只承载：
//   - Handler结构体和构造函数；
//   - 教案列表；
//   - 普通教案创建；
//   - 教案详情、更新和删除。
//
// 状态流转与提示词模板端点拆至lesson_plan_handler_actions.go。
// 错误映射和路径解析拆至lesson_plan_handler_support.go。
//
// 上下文10：普通教案创建成功后写入lesson_plan.create审计，
// 审计记录数据库最终返回的education_domain快照，不记录教案正文。

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// LessonPlanHandler 教案管理接口处理器。
type LessonPlanHandler struct {
	lpService *services.LessonPlanService
}

// NewLessonPlanHandler 创建教案管理处理器实例。
func NewLessonPlanHandler(
	lpService *services.LessonPlanService,
) *LessonPlanHandler {
	return &LessonPlanHandler{
		lpService: lpService,
	}
}

// ==================== 教案列表 ====================

// ListLessonPlans 查询当前用户可见的教案列表。
func (h *LessonPlanHandler) ListLessonPlans(
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

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	scope := services.ResolveDataScope(
		r.Context(),
		claims.Role,
		claims.UserID,
	)

	query := r.URL.Query()
	authorID := query.Get("author_id")
	groupID := query.Get("group_id")
	status := query.Get("status")
	subject := query.Get("subject")
	grade := query.Get("grade")
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	qualityLevel, _ := strconv.Atoi(
		query.Get("quality_level"),
	)
	structureType, _ := strconv.Atoi(
		query.Get("structure_type"),
	)
	cognitiveLevel, _ := strconv.Atoi(
		query.Get("cognitive_level"),
	)
	pedagogyIntensity, _ := strconv.Atoi(
		query.Get("pedagogy_intensity"),
	)

	result, err := h.lpService.ListLessonPlans(
		r.Context(),
		claims.UserID,
		authorID,
		groupID,
		status,
		subject,
		grade,
		limit,
		offset,
		qualityLevel,
		structureType,
		cognitiveLevel,
		pedagogyIntensity,
		&scope,
	)
	if err != nil {
		log.Printf("获取教案列表失败: %v", err)
		utils.InternalError(w, "获取教案列表失败")
		return
	}

	utils.Success(w, result)
}

// ==================== 创建教案 ====================

// CreateLessonPlan 创建普通教案。
//
// Service先解析唯一具体教学域，Repository再显式写入该域。
// 创建成功后，审计数据库RETURNING返回的最终教育域快照。
func (h *LessonPlanHandler) CreateLessonPlan(
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

	lessonPlan, err := h.lpService.CreateLessonPlan(
		r.Context(),
		&req,
		userID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	// WriteAuditLog采用异步best-effort语义，不阻塞正常创建响应。
	// 只记录创建元数据，不写入教案正文、对话或生成配置。
	repository.WriteAuditLog(
		userID,
		repository.ActionLessonPlanCreate,
		map[string]interface{}{
			"plan_id":           lessonPlan.ID,
			"title":             lessonPlan.Title,
			"subject":           lessonPlan.Subject,
			"grade":             lessonPlan.Grade,
			"topic":             lessonPlan.Topic,
			"education_domain":  lessonPlan.EducationDomain,
			"duration_minutes":  lessonPlan.DurationMinutes,
			"creation_entry":    "ordinary",
			"explicit_snapshot": true,
		},
		repository.GetClientIP(r.RemoteAddr),
	)

	utils.Success(w, lessonPlan)
}

// ==================== 获取教案详情 ====================

// GetLessonPlan 获取教案详情。
func (h *LessonPlanHandler) GetLessonPlan(
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

	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	detail, err := h.lpService.GetLessonPlan(
		r.Context(),
		id,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, detail)
}

// ==================== 更新教案 ====================

// UpdateLessonPlan 更新教案内容。
func (h *LessonPlanHandler) UpdateLessonPlan(
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

	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)

	var req models.UpdateLessonPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.lpService.UpdateLessonPlan(
		r.Context(),
		id,
		userID,
		&req,
	); err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// ==================== 删除教案 ====================

// DeleteLessonPlan 软删除教案。
func (h *LessonPlanHandler) DeleteLessonPlan(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodDeleteOnly,
		)
		return
	}

	id := extractLPID(r.URL.Path)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)
	if err := h.lpService.DeleteLessonPlan(
		r.Context(),
		id,
		userID,
	); err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}
