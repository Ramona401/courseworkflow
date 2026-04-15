package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// PromptHandler 提示词管理处理器
type PromptHandler struct {
	promptService *services.PromptService
}

func NewPromptHandler(promptService *services.PromptService) *PromptHandler {
	return &PromptHandler{promptService: promptService}
}

func (h *PromptHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	result, err := h.promptService.ListCurrentPrompts()
	if err != nil {
		utils.InternalError(w, "获取提示词列表失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *PromptHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	key := extractPromptKey(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, utils.MsgMissingPromptKey)
		return
	}
	result, err := h.promptService.GetPromptByKey(key)
	if err != nil {
		handlePromptError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *PromptHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	key := extractPromptKey(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, utils.MsgMissingPromptKey)
		return
	}
	var req models.UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	result, err := h.promptService.UpdatePrompt(key, req.Content, claims.UserID)
	if err != nil {
		handlePromptError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *PromptHandler) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	key := extractPromptKeyBeforeVersions(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, utils.MsgMissingPromptKey)
		return
	}
	result, err := h.promptService.GetVersionHistory(key)
	if err != nil {
		handlePromptError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *PromptHandler) RollbackVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	key := extractPromptKeyBeforeRollback(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, utils.MsgMissingPromptKey)
		return
	}
	var req struct {
		VersionID string `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if strings.TrimSpace(req.VersionID) == "" {
		utils.BadRequest(w, "目标版本ID不能为空")
		return
	}
	result, err := h.promptService.RollbackToVersion(key, req.VersionID)
	if err != nil {
		handlePromptError(w, err)
		return
	}
	utils.Success(w, result)
}

// ==================== 辅助方法 ====================

func extractPromptKey(path string) string {
	if !strings.HasPrefix(path, utils.PathPromptPrefix) {
		return ""
	}
	remaining := strings.TrimPrefix(path, utils.PathPromptPrefix)
	remaining = strings.TrimSuffix(remaining, "/")
	if strings.Contains(remaining, "/") {
		return ""
	}
	return remaining
}

func extractPromptKeyBeforeVersions(path string) string {
	suffix := "/versions"
	if !strings.HasPrefix(path, utils.PathPromptPrefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	return path[len(utils.PathPromptPrefix) : len(path)-len(suffix)]
}

func extractPromptKeyBeforeRollback(path string) string {
	suffix := "/rollback"
	if !strings.HasPrefix(path, utils.PathPromptPrefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	return path[len(utils.PathPromptPrefix) : len(path)-len(suffix)]
}

func handlePromptError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrPromptKeyRequired, services.ErrInvalidPromptKey,
		services.ErrPromptContentEmpty, services.ErrAlreadyCurrent:
		utils.BadRequest(w, err.Error())
	case services.ErrPromptNotFound, services.ErrVersionNotFound:
		utils.Fail(w, http.StatusNotFound, err.Error())
	default:
		utils.InternalError(w, err.Error())
	}
}
