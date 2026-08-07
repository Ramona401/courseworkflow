package handlers

// lesson_plan_interaction_handler.go — 共享教案互动HTTP处理器
//
// 接口：
//   POST /api/v1/lesson-plans/plans/{id}/interact
//   GET  /api/v1/lesson-plans/plans/{id}/interactions
//   GET  /api/v1/lesson-plans/my-favorites
//
// 上下文17要求所有入口必须使用登录用户身份，并由Service在统计或写入前
// 执行统一共享可见性校验。不可见资源统一返回404。

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// LessonPlanInteractionHandler 教案互动HTTP处理器。
type LessonPlanInteractionHandler struct {
	interactionService *services.LessonPlanInteractionService
}

// NewLessonPlanInteractionHandler 创建教案互动处理器。
func NewLessonPlanInteractionHandler(
	service *services.LessonPlanInteractionService,
) *LessonPlanInteractionHandler {
	return &LessonPlanInteractionHandler{
		interactionService: service,
	}
}

// ToggleInteraction 切换点赞或收藏。
func (
	h *LessonPlanInteractionHandler,
) ToggleInteraction(
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

	planID := extractLPMiddleID(
		r.URL.Path,
		"/interact",
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	var request models.ToggleInteractionRequest
	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	response, err :=
		h.interactionService.ToggleInteraction(
			r.Context(),
			userID,
			planID,
			request.InteractionType,
		)
	if err != nil {
		h.handleInteractionError(w, err)
		return
	}

	utils.Success(w, response)
}

// GetInteractions 查询当前用户可见共享教案的互动统计。
func (
	h *LessonPlanInteractionHandler,
) GetInteractions(
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

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	planID := extractLPMiddleID(
		r.URL.Path,
		"/interactions",
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	counts, err :=
		h.interactionService.GetInteractionCounts(
			r.Context(),
			planID,
			userID,
		)
	if err != nil {
		h.handleInteractionError(w, err)
		return
	}

	utils.Success(w, counts)
}

// ListMyFavorites 查询当前用户仍有权限访问的收藏列表。
func (
	h *LessonPlanInteractionHandler,
) ListMyFavorites(
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

	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	response, err :=
		h.interactionService.ListMyFavorites(
			r.Context(),
			userID,
			limit,
			offset,
		)
	if err != nil {
		log.Printf(
			"查询收藏列表失败: %v",
			err,
		)
		utils.InternalError(w, "查询收藏列表失败")
		return
	}

	utils.Success(w, response)
}

// handleInteractionError 统一映射互动错误。
func (
	h *LessonPlanInteractionHandler,
) handleInteractionError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrInvalidInteractionType,
	):
		utils.BadRequest(w, err.Error())

	case errors.Is(
		err,
		services.ErrLPNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	default:
		log.Printf(
			"互动操作失败: %v",
			err,
		)
		utils.InternalError(
			w,
			"操作失败，请稍后重试",
		)
	}
}
