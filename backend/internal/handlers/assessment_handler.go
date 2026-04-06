package handlers

// assessment_handler.go — 教学风格前测处理器
//
// 迭代3新增：
//   - StartAssessment：开始前测对话
//   - ChatAssessment：前测对话轮次
//   - SubmitAssessment：提交AI判定结果
//   - SkipAssessment：跳过前测（使用默认画像）
//   - GetAssessmentResult：获取前测结果
//   - AutoGenerateRecipe：手动触发自动生成配方
//
// 所有接口需认证（authMW），从JWT提取userID

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
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

// StartAssessment 开始前测对话
// POST /api/v1/lesson-plans/assessment/start
// 返回AI的开场白消息
func (h *AssessmentHandler) StartAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	resp, err := h.assessmentSvc.StartAssessment(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 前测对话 ====================

// ChatAssessment 前测对话轮次
// POST /api/v1/lesson-plans/assessment/chat
// Body: { "message": "用户回复", "conversation_history": [...] }
func (h *AssessmentHandler) ChatAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	// 解析请求体
	var req struct {
		Message             string                       `json:"message"`
		ConversationHistory []models.AssessmentMessage   `json:"conversation_history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"code": -1, "message": "请求参数格式错误",
		})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"code": -1, "message": "消息内容不能为空",
		})
		return
	}

	resp, err := h.assessmentSvc.ChatAssessment(r.Context(), claims.UserID, req.Message, req.ConversationHistory)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 提交前测结果 ====================

// SubmitAssessment 提交AI判定结果
// POST /api/v1/lesson-plans/assessment/submit
// Body: { "experience_years": 5, "subject_primary": "...", ... , "conversation_log": [...] }
func (h *AssessmentHandler) SubmitAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	// 解析请求体
	var req struct {
		models.AssessmentSubmitRequest
		ConversationLog []models.AssessmentMessage `json:"conversation_log"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"code": -1, "message": "请求参数格式错误",
		})
		return
	}

	resp, err := h.assessmentSvc.SubmitAssessment(r.Context(), claims.UserID, &req.AssessmentSubmitRequest, req.ConversationLog)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 跳过前测 ====================

// SkipAssessment 跳过前测，使用默认画像
// POST /api/v1/lesson-plans/assessment/skip
func (h *AssessmentHandler) SkipAssessment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	resp, err := h.assessmentSvc.SkipAssessment(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 获取前测结果 ====================

// GetAssessmentResult 获取当前用户的前测结果
// GET /api/v1/lesson-plans/assessment/result
func (h *AssessmentHandler) GetAssessmentResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持GET请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	resp, err := h.assessmentSvc.GetAssessmentResult(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 自动生成配方 ====================

// AutoGenerateRecipe 手动触发根据前测结果自动生成配方
// POST /api/v1/lesson-plans/assessment/auto-recipe
// 适用场景：用户完成前测后没有自动生成配方，或想重新生成
func (h *AssessmentHandler) AutoGenerateRecipe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"code": -1, "message": "未获取到用户信息",
		})
		return
	}

	resp, err := h.assessmentSvc.AutoGenerateRecipeFromProfile(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code": 0, "data": resp,
	})
}

// ==================== 工具函数 ====================

// writeJSON 统一JSON响应输出
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
