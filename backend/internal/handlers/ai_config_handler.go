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

// GetGlobalConfig 获取全局AI配置
// GET /api/v1/ai-config/global
func (h *AIConfigHandler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	config, err := h.aiConfigService.GetGlobalConfig()
	if err != nil {
		utils.InternalError(w, "获取全局配置失败: "+err.Error())
		return
	}

	utils.Success(w, config)
}

// UpdateGlobalConfig 更新全局AI配置
// PUT /api/v1/ai-config/global
func (h *AIConfigHandler) UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 解析请求体
	var req models.UpdateGlobalConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 获取当前操作者ID（GetClaims返回两个值）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "无法获取用户信息")
		return
	}

	// 执行更新
	if err := h.aiConfigService.UpdateGlobalConfig(&req, claims.UserID); err != nil {
		handleAIConfigError(w, err)
		return
	}

	// 返回更新后的配置
	config, err := h.aiConfigService.GetGlobalConfig()
	if err != nil {
		utils.InternalError(w, "获取更新后配置失败")
		return
	}

	utils.Success(w, config)
}

// ==================== AI连通性测试接口（P2-2新增）====================

// TestConnection 测试AI API连通性
// POST /api/v1/ai-config/test
// 使用当前全局配置向AI API发送测试请求，验证连通性并返回延迟
func (h *AIConfigHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 执行连通性测试
	result, err := h.aiConfigService.TestConnection()
	if err != nil {
		utils.InternalError(w, "连通性测试执行失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 可用模型查询接口 ====================

// ListModels 查询当前Key下可用的模型列表
// GET /api/v1/ai-config/models
// 调用上游 {api_base_url}/models 接口，返回模型ID列表（按字母排序）
func (h *AIConfigHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 查询可用模型列表
	modelList, err := h.aiConfigService.ListModels()
	if err != nil {
		handleAIConfigError(w, err)
		return
	}

	// 返回模型列表及总数
	utils.Success(w, map[string]interface{}{
		"models": modelList,
		"total":  len(modelList),
	})
}

// ==================== 场景配置接口 ====================

// GetSceneConfigs 获取所有场景配置
// GET /api/v1/ai-config/scenes
func (h *AIConfigHandler) GetSceneConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	scenes, err := h.aiConfigService.GetAllSceneConfigs()
	if err != nil {
		utils.InternalError(w, "获取场景配置失败: "+err.Error())
		return
	}

	utils.Success(w, scenes)
}

// UpdateSceneConfig 更新指定场景配置
// PUT /api/v1/ai-config/scenes/{code}
func (h *AIConfigHandler) UpdateSceneConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 从路径中提取场景代码
	// 路径格式：/api/v1/ai-config/scenes/{code}
	prefix := "/api/v1/ai-config/scenes/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		utils.BadRequest(w, "无效的请求路径")
		return
	}
	sceneCode := strings.TrimPrefix(r.URL.Path, prefix)
	sceneCode = strings.TrimSuffix(sceneCode, "/") // 去掉可能的尾部斜杠

	if sceneCode == "" {
		utils.BadRequest(w, "场景代码不能为空")
		return
	}

	// 解析请求体
	var req models.UpdateSceneConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 获取当前操作者ID（GetClaims返回两个值）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "无法获取用户信息")
		return
	}

	// 执行更新
	if err := h.aiConfigService.UpdateSceneConfig(sceneCode, &req, claims.UserID); err != nil {
		handleAIConfigError(w, err)
		return
	}

	// 返回更新后的所有场景配置
	scenes, err := h.aiConfigService.GetAllSceneConfigs()
	if err != nil {
		utils.InternalError(w, "获取更新后场景配置失败")
		return
	}

	utils.Success(w, scenes)
}

// ==================== 错误处理 ====================

// handleAIConfigError 统一AI配置错误分发
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
