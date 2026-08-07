package handlers

// lesson_plan_handler_actions.go — 教案动作与模板HTTP处理器
//
// 本文件承载：
//   - 个人发布、提交评审、评审、共享发布；
//   - 进入课件开发和Fork；
//   - 提示词模板列表、创建、详情、更新和继承解析。
//
// 基础CRUD位于lesson_plan_handler.go。
// 错误映射和路径解析位于lesson_plan_handler_support.go。

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// ==================== 教案状态操作 ====================

type publishLessonPlanPersonalRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

func (h *LessonPlanHandler) PublishPersonal(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	id := extractLPMiddleID(r.URL.Path, "/publish-personal")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req publishLessonPlanPersonalRequest
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.lpService.PublishPersonal(
		r.Context(),
		id,
		userID,
		req.ExpectedVersion,
	); err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "个人发布成功"})
}

func (h *LessonPlanHandler) SubmitForReview(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := extractLPMiddleID(
		r.URL.Path,
		"/submit-review",
	)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)

	var req models.SubmitLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.lpService.SubmitForReview(
		r.Context(),
		id,
		userID,
		req.GroupID,
	); err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "已提交评审",
		},
	)
}

func (h *LessonPlanHandler) ReviewLessonPlan(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := extractLPMiddleID(
		r.URL.Path,
		"/review",
	)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)

	var req models.CreateLessonPlanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	if err := h.lpService.ReviewLessonPlan(
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
			"message": "评审完成",
		},
	)
}

func (h *LessonPlanHandler) PublishShared(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := extractLPMiddleID(
		r.URL.Path,
		"/publish-shared",
	)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)
	if err := h.lpService.PublishShared(
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
			"message": "共享发布成功",
		},
	)
}

func (h *LessonPlanHandler) StartDevelopment(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := extractLPMiddleID(
		r.URL.Path,
		"/start-development",
	)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)
	result, err := h.lpService.StartDevelopment(
		r.Context(),
		id,
		userID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, result)
}

// ForkLessonPlan 复制当前用户可见的共享教案。
//
// 路由层使用后缀分发，因此Handler必须自行限制POST，
// 防止GET或其它方法误触发写操作。
func (h *LessonPlanHandler) ForkLessonPlan(
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

	id := extractLPMiddleID(
		r.URL.Path,
		"/fork",
	)
	if id == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	newLessonPlan, err := h.lpService.ForkLessonPlan(
		r.Context(),
		id,
		userID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, newLessonPlan)
}

// ==================== 提示词模板管理 ====================

func (h *LessonPlanHandler) ListPromptTemplates(
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

	query := r.URL.Query()
	result, err := h.lpService.ListPromptTemplates(
		r.Context(),
		query.Get("level"),
		query.Get("owner_id"),
	)
	if err != nil {
		log.Printf("获取模板列表失败: %v", err)
		utils.InternalError(w, "获取模板列表失败")
		return
	}

	utils.Success(w, result)
}

func (h *LessonPlanHandler) CreatePromptTemplate(
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

	var req models.CreatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	promptTemplate, err := h.lpService.CreatePromptTemplate(
		r.Context(),
		&req,
		userID,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, promptTemplate)
}

func (h *LessonPlanHandler) GetPromptTemplate(
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

	id := extractTemplateID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}

	promptTemplate, err := h.lpService.GetPromptTemplate(
		r.Context(),
		id,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, promptTemplate)
}

func (h *LessonPlanHandler) UpdatePromptTemplate(
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

	if err := h.lpService.UpdatePromptTemplate(
		r.Context(),
		id,
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

func (h *LessonPlanHandler) ResolvePromptTemplate(
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

	id := extractMiddleSegment(
		r.URL.Path,
		"/api/v1/lesson-plans/templates/",
		"/resolved",
	)
	if id == "" {
		utils.BadRequest(w, "缺少模板ID")
		return
	}

	resolved, err := h.lpService.ResolvePromptTemplate(
		r.Context(),
		id,
	)
	if err != nil {
		h.handleLPError(w, err)
		return
	}

	utils.Success(w, resolved)
}
