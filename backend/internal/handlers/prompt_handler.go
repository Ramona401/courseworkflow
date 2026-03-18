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

// ==================== 提示词管理接口处理器（仅admin） ====================

// PromptHandler 提示词管理处理器
type PromptHandler struct {
	promptService *services.PromptService
}

// NewPromptHandler 创建处理器实例
func NewPromptHandler(promptService *services.PromptService) *PromptHandler {
	return &PromptHandler{promptService: promptService}
}

// ListPrompts 获取所有提示词（当前生效版本）
// GET /api/v1/prompts
func (h *PromptHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	// 仅允许 GET 请求
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "仅支持GET请求",
		})
		return
	}

	// 调用服务层获取提示词列表
	result, err := h.promptService.ListCurrentPrompts()
	if err != nil {
		utils.InternalError(w, "获取提示词列表失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// GetPrompt 获取指定槽位的当前生效提示词
// GET /api/v1/prompts/{key}
func (h *PromptHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	// 仅允许 GET 请求
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "仅支持GET请求",
		})
		return
	}

	// 从路径提取 prompt_key
	key := extractPromptKey(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, "缺少提示词标识")
		return
	}

	// 调用服务层
	result, err := h.promptService.GetPromptByKey(key)
	if err != nil {
		handlePromptError(w, err)
		return
	}

	utils.Success(w, result)
}

// UpdatePrompt 更新提示词内容（创建新版本）
// PUT /api/v1/prompts/{key}
func (h *PromptHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	// 仅允许 PUT 请求
	if r.Method != http.MethodPut {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "仅支持PUT请求",
		})
		return
	}

	// 从路径提取 prompt_key
	key := extractPromptKey(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, "缺少提示词标识")
		return
	}

	// 解析请求体
	var req models.UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	// 获取当前用户ID
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "无法获取用户信息")
		return
	}

	// 调用服务层更新（创建新版本）
	result, err := h.promptService.UpdatePrompt(key, req.Content, claims.UserID)
	if err != nil {
		handlePromptError(w, err)
		return
	}

	utils.Success(w, result)
}

// GetVersionHistory 获取指定槽位的版本历史
// GET /api/v1/prompts/{key}/versions
func (h *PromptHandler) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	// 仅允许 GET 请求
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "仅支持GET请求",
		})
		return
	}

	// 从路径提取 prompt_key（路径格式：/api/v1/prompts/{key}/versions）
	key := extractPromptKeyBeforeVersions(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, "缺少提示词标识")
		return
	}

	// 调用服务层
	result, err := h.promptService.GetVersionHistory(key)
	if err != nil {
		handlePromptError(w, err)
		return
	}

	utils.Success(w, result)
}

// RollbackVersion 回滚到指定版本
// POST /api/v1/prompts/{key}/rollback
func (h *PromptHandler) RollbackVersion(w http.ResponseWriter, r *http.Request) {
	// 仅允许 POST 请求
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "仅支持POST请求",
		})
		return
	}

	// 从路径提取 prompt_key（路径格式：/api/v1/prompts/{key}/rollback）
	key := extractPromptKeyBeforeRollback(r.URL.Path)
	if key == "" {
		utils.BadRequest(w, "缺少提示词标识")
		return
	}

	// 解析请求体（包含目标版本ID）
	var req struct {
		VersionID string `json:"version_id"` // 要回滚到的版本ID
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	if strings.TrimSpace(req.VersionID) == "" {
		utils.BadRequest(w, "目标版本ID不能为空")
		return
	}

	// 调用服务层执行回滚
	result, err := h.promptService.RollbackToVersion(key, req.VersionID)
	if err != nil {
		handlePromptError(w, err)
		return
	}

	utils.Success(w, result)
}

// ==================== 内部辅助方法 ====================

// extractPromptKey 从路径中提取提示词标识
// 路径格式：/api/v1/prompts/{key}
func extractPromptKey(path string) string {
	prefix := "/api/v1/prompts/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	remaining := strings.TrimPrefix(path, prefix)
	// 去掉末尾的斜杠
	remaining = strings.TrimSuffix(remaining, "/")
	// 如果还包含子路径（如 /versions 或 /rollback），不在此处理
	if strings.Contains(remaining, "/") {
		return ""
	}
	return remaining
}

// extractPromptKeyBeforeVersions 从路径中提取 {key}（/versions 前）
// 路径格式：/api/v1/prompts/{key}/versions
func extractPromptKeyBeforeVersions(path string) string {
	prefix := "/api/v1/prompts/"
	suffix := "/versions"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	middle := path[len(prefix) : len(path)-len(suffix)]
	return middle
}

// extractPromptKeyBeforeRollback 从路径中提取 {key}（/rollback 前）
// 路径格式：/api/v1/prompts/{key}/rollback
func extractPromptKeyBeforeRollback(path string) string {
	prefix := "/api/v1/prompts/"
	suffix := "/rollback"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	middle := path[len(prefix) : len(path)-len(suffix)]
	return middle
}

// handlePromptError 统一处理提示词相关错误
func handlePromptError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrPromptKeyRequired, services.ErrInvalidPromptKey,
		services.ErrPromptContentEmpty, services.ErrAlreadyCurrent:
		utils.BadRequest(w, err.Error())
	case services.ErrPromptNotFound, services.ErrVersionNotFound:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": err.Error(),
		})
	default:
		utils.InternalError(w, err.Error())
	}
}
