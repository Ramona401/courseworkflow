package handlers

// lesson_plan_interaction_handler.go — 教案互动（点赞/收藏）HTTP处理器
//
// 接口清单：
//   POST   /api/v1/lesson-plans/plans/{id}/interact    切换点赞/收藏
//   GET    /api/v1/lesson-plans/plans/{id}/interactions 查询教案互动统计
//   GET    /api/v1/lesson-plans/my-favorites            查询我的收藏列表

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

// LessonPlanInteractionHandler 教案互动HTTP处理器
type LessonPlanInteractionHandler struct {
	interactionService *services.LessonPlanInteractionService
}

// NewLessonPlanInteractionHandler 创建教案互动处理器实例
func NewLessonPlanInteractionHandler(svc *services.LessonPlanInteractionService) *LessonPlanInteractionHandler {
	return &LessonPlanInteractionHandler{interactionService: svc}
}

// ==================== POST /plans/{id}/interact — 切换点赞/收藏 ====================

func (h *LessonPlanInteractionHandler) ToggleInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	// 从路径提取教案ID: /api/v1/lesson-plans/plans/{id}/interact
	planID := extractLPMiddleID(r.URL.Path, "/interact")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	var req models.ToggleInteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	resp, err := h.interactionService.ToggleInteraction(r.Context(), userID, planID, req.InteractionType)
	if err != nil {
		h.handleInteractionError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== GET /plans/{id}/interactions — 查询教案互动统计 ====================

func (h *LessonPlanInteractionHandler) GetInteractions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	userID := getCurrentUserID(r)

	// 从路径提取教案ID: /api/v1/lesson-plans/plans/{id}/interactions
	planID := extractLPMiddleID(r.URL.Path, "/interactions")
	if planID == "" {
		utils.BadRequest(w, utils.MsgMissingLessonPlanID)
		return
	}

	counts, err := h.interactionService.GetInteractionCounts(r.Context(), planID, userID)
	if err != nil {
		h.handleInteractionError(w, err)
		return
	}
	utils.Success(w, counts)
}

// ==================== GET /my-favorites — 我的收藏列表 ====================

func (h *LessonPlanInteractionHandler) ListMyFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	userID := getCurrentUserID(r)
	if userID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	resp, err := h.interactionService.ListMyFavorites(r.Context(), userID, limit, offset)
	if err != nil {
		log.Printf("查询收藏列表失败: %v", err)
		utils.InternalError(w, "查询收藏列表失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 错误处理 ====================

func (h *LessonPlanInteractionHandler) handleInteractionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidInteractionType):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrLPNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("互动操作失败: %v", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}

// ==================== 辅助函数 ====================

// extractMiddleSegment 已在 lesson_plan_handler.go 中定义，此处复用
// extractLPMiddleID 已在 lesson_plan_handler.go 中定义，此处复用
// getCurrentUserID 已在 pipeline_handler.go 中定义，此处复用

// 确保引用不报未使用错误（strings 包用于路径解析的引用保留）
var _ = strings.HasPrefix
