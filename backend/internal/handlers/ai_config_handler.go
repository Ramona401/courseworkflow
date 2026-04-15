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

// AIConfigHandler AI配置接口处理器
type AIConfigHandler struct {
	aiConfigService *services.AIConfigService
}

// NewAIConfigHandler 创建AI配置处理器实例
func NewAIConfigHandler(aiConfigService *services.AIConfigService) *AIConfigHandler {
	return &AIConfigHandler{aiConfigService: aiConfigService}
}

// ==================== 全局配置接口 ====================

// GetGlobalConfig GET /api/v1/ai-config/global
func (h *AIConfigHandler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	config, err := h.aiConfigService.GetGlobalConfig()
	if err != nil {
		utils.InternalError(w, "获取全局配置失败: "+err.Error())
		return
	}
	utils.Success(w, config)
}

// UpdateGlobalConfig PUT /api/v1/ai-config/global
func (h *AIConfigHandler) UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	var req models.UpdateGlobalConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	if err := h.aiConfigService.UpdateGlobalConfig(&req, claims.UserID); err != nil {
		handleAIConfigError(w, err)
		return
	}
	config, err := h.aiConfigService.GetGlobalConfig()
	if err != nil {
		utils.InternalError(w, "获取更新后配置失败")
		return
	}
	utils.Success(w, config)
}

// ==================== AI连通性测试接口 ====================

// TestConnection POST /api/v1/ai-config/test
func (h *AIConfigHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	result, err := h.aiConfigService.TestConnection()
	if err != nil {
		utils.InternalError(w, "连通性测试执行失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== 可用模型查询接口 ====================

// ListModels GET /api/v1/ai-config/models
func (h *AIConfigHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	modelList, err := h.aiConfigService.ListModels()
	if err != nil {
		handleAIConfigError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"models": modelList,
		"total":  len(modelList),
	})
}

// ==================== 场景配置接口 ====================

// GetSceneConfigs GET /api/v1/ai-config/scenes
func (h *AIConfigHandler) GetSceneConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	scenes, err := h.aiConfigService.GetAllSceneConfigs()
	if err != nil {
		utils.InternalError(w, "获取场景配置失败: "+err.Error())
		return
	}
	utils.Success(w, scenes)
}

// UpdateSceneConfig PUT /api/v1/ai-config/scenes/{code}
func (h *AIConfigHandler) UpdateSceneConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	prefix := "/api/v1/ai-config/scenes/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		utils.BadRequest(w, "无效的请求路径")
		return
	}
	sceneCode := strings.TrimPrefix(r.URL.Path, prefix)
	sceneCode = strings.TrimSuffix(sceneCode, "/")
	if sceneCode == "" {
		utils.BadRequest(w, "场景代码不能为空")
		return
	}
	var req models.UpdateSceneConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	if err := h.aiConfigService.UpdateSceneConfig(sceneCode, &req, claims.UserID); err != nil {
		handleAIConfigError(w, err)
		return
	}
	scenes, err := h.aiConfigService.GetAllSceneConfigs()
	if err != nil {
		utils.InternalError(w, "获取更新后场景配置失败")
		return
	}
	utils.Success(w, scenes)
}

// ==================== 错误处理 ====================

func handleAIConfigError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrAPIBaseURLRequired,
		services.ErrModelRequired,
		services.ErrInvalidTemperature,
		services.ErrInvalidMaxTokens,
		services.ErrInvalidSceneCode,
		services.ErrAPIKeyNotSet:
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}
