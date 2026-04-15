package handlers

// assessment_handler.go — 教学风格前测处理器
//
// 所有接口需认证（authMW），从JWT提取userID

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AssessmentHandler 前测处理器
type AssessmentHandler struct {
	assessmentSvc *services.AssessmentService
}

// NewAssessmentHandler 创建前测处理器实例
func NewAssessmentHandler(assessSvc *services.AssessmentService) *AssessmentHandler {
	return &AssessmentHandler{assessmentSvc: assessSvc}
}

// ==================== 开始前测 ====================

// StartAssessment POST /api/v1/lesson-plans/assessment/start
func (h *AssessmentHandler) StartAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	resp, err := h.assessmentSvc.StartAssessment(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 前测对话 ====================

// ChatAssessment POST /api/v1/lesson-plans/assessment/chat
func (h *AssessmentHandler) ChatAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	var req struct {
		Message             string                     `json:"message"`
		ConversationHistory []models.AssessmentMessage `json:"conversation_history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.Message == "" {
		utils.BadRequest(w, "消息内容不能为空")
		return
	}
	resp, err := h.assessmentSvc.ChatAssessment(r.Context(), claims.UserID, req.Message, req.ConversationHistory)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 提交前测结果 ====================

// SubmitAssessment POST /api/v1/lesson-plans/assessment/submit
func (h *AssessmentHandler) SubmitAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	var req struct {
		models.AssessmentSubmitRequest
		ConversationLog []models.AssessmentMessage `json:"conversation_log"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	resp, err := h.assessmentSvc.SubmitAssessment(r.Context(), claims.UserID, &req.AssessmentSubmitRequest, req.ConversationLog)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 跳过前测 ====================

// SkipAssessment POST /api/v1/lesson-plans/assessment/skip
func (h *AssessmentHandler) SkipAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	resp, err := h.assessmentSvc.SkipAssessment(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取前测结果 ====================

// GetAssessmentResult GET /api/v1/lesson-plans/assessment/result
func (h *AssessmentHandler) GetAssessmentResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	resp, err := h.assessmentSvc.GetAssessmentResult(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ==================== 自动生成配方 ====================

// AutoGenerateRecipe POST /api/v1/lesson-plans/assessment/auto-recipe
func (h *AssessmentHandler) AutoGenerateRecipe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	resp, err := h.assessmentSvc.AutoGenerateRecipeFromProfile(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}
