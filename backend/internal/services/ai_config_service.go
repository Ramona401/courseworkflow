package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 错误常量 ====================

var (
	ErrAPIBaseURLRequired = errors.New("API基础地址不能为空")
	ErrModelRequired      = errors.New("默认模型不能为空")
	ErrInvalidTemperature = errors.New("温度必须在 0.0 ~ 2.0 之间")
	ErrInvalidMaxTokens   = errors.New("最大Token数必须在 100 ~ 200000 之间")
	ErrInvalidSceneCode   = errors.New("无效的场景代码")
)

// ==================== AIConfigService ====================

// AIConfigService AI配置业务逻辑层
type AIConfigService struct {
	aesKey string // AES加密密钥（从配置读取）
}

// NewAIConfigService 创建AI配置服务实例
func NewAIConfigService(cfg *config.Config) *AIConfigService {
	return &AIConfigService{
		aesKey: cfg.AESKey,
	}
}

// ==================== 全局配置方法 ====================

// GetGlobalConfig 获取全局配置（API Key 脱敏处理）
func (s *AIConfigService) GetGlobalConfig() (*models.GlobalConfigResponse, error) {
	// 从数据库获取所有配置项
	configs, err := repository.GetAllConfigs()
	if err != nil {
		return nil, fmt.Errorf("获取全局配置失败: %w", err)
	}

	// 构建配置映射
	configMap := make(map[string]*models.AIConfig)
	for _, c := range configs {
		configMap[c.ConfigKey] = c
	}

	// 组装响应
	resp := &models.GlobalConfigResponse{}

	// API 基础地址
	if c, ok := configMap[models.ConfigKeyAPIBaseURL]; ok {
		resp.APIBaseURL = c.ConfigValue
		resp.UpdatedAt = c.UpdatedAt
	}

	// API Key（脱敏处理）
	if c, ok := configMap[models.ConfigKeyAPIKeyEnc]; ok {
		resp.APIKey, resp.APIKeySet = s.maskAPIKey(c.ConfigValue)
	}

	// 默认模型
	if c, ok := configMap[models.ConfigKeyDefaultModel]; ok {
		resp.DefaultModel = c.ConfigValue
	}

	// 温度
	if c, ok := configMap[models.ConfigKeyTemperature]; ok {
		resp.Temperature = c.ConfigValue
	}

	// 最大Token数
	if c, ok := configMap[models.ConfigKeyMaxTokens]; ok {
		resp.MaxTokens = c.ConfigValue
	}

	return resp, nil
}

// UpdateGlobalConfig 更新全局配置
func (s *AIConfigService) UpdateGlobalConfig(req *models.UpdateGlobalConfigRequest, userID string) error {
	// 参数校验
	req.APIBaseURL = strings.TrimSpace(req.APIBaseURL)
	req.DefaultModel = strings.TrimSpace(req.DefaultModel)
	req.Temperature = strings.TrimSpace(req.Temperature)
	req.MaxTokens = strings.TrimSpace(req.MaxTokens)

	if req.APIBaseURL == "" {
		return ErrAPIBaseURLRequired
	}
	if req.DefaultModel == "" {
		return ErrModelRequired
	}

	// 校验温度范围
	if req.Temperature != "" {
		temp, err := strconv.ParseFloat(req.Temperature, 64)
		if err != nil || temp < 0 || temp > 2.0 {
			return ErrInvalidTemperature
		}
	}

	// 校验Token数范围
	if req.MaxTokens != "" {
		tokens, err := strconv.Atoi(req.MaxTokens)
		if err != nil || tokens < 100 || tokens > 200000 {
			return ErrInvalidMaxTokens
		}
	}

	// 逐项更新
	if err := repository.UpdateConfigValue(models.ConfigKeyAPIBaseURL, req.APIBaseURL, userID); err != nil {
		return err
	}
	if err := repository.UpdateConfigValue(models.ConfigKeyDefaultModel, req.DefaultModel, userID); err != nil {
		return err
	}
	if req.Temperature != "" {
		if err := repository.UpdateConfigValue(models.ConfigKeyTemperature, req.Temperature, userID); err != nil {
			return err
		}
	}
	if req.MaxTokens != "" {
		if err := repository.UpdateConfigValue(models.ConfigKeyMaxTokens, req.MaxTokens, userID); err != nil {
			return err
		}
	}

	// API Key：仅当非空时更新（空字符串表示不修改）
	if req.APIKey != "" {
		encrypted, err := utils.EncryptAES(req.APIKey, s.aesKey)
		if err != nil {
			return fmt.Errorf("加密API Key失败: %w", err)
		}
		if err := repository.UpdateConfigValue(models.ConfigKeyAPIKeyEnc, encrypted, userID); err != nil {
			return err
		}
	}

	return nil
}

// GetDecryptedAPIKey 获取解密后的API Key（仅内部使用，供AI调用时获取真实Key）
func (s *AIConfigService) GetDecryptedAPIKey() (string, error) {
	c, err := repository.GetConfigByKey(models.ConfigKeyAPIKeyEnc)
	if err != nil {
		return "", fmt.Errorf("获取API Key配置失败: %w", err)
	}

	// 如果是占位符，直接返回
	if c.ConfigValue == "PLACEHOLDER_SET_IN_ADMIN" {
		return "", errors.New("API Key尚未配置")
	}

	// 尝试解密
	plaintext, err := utils.DecryptAES(c.ConfigValue, s.aesKey)
	if err != nil {
		// 如果解密失败，可能是明文存储的旧数据，直接返回
		return c.ConfigValue, nil
	}
	return plaintext, nil
}

// maskAPIKey API Key 脱敏处理
// 返回值：(脱敏后的字符串, 是否已配置)
func (s *AIConfigService) maskAPIKey(encValue string) (string, bool) {
	// 占位符或空值
	if encValue == "" || encValue == "PLACEHOLDER_SET_IN_ADMIN" {
		return "未配置", false
	}

	// 尝试解密获取原文前几位用于脱敏显示
	plaintext, err := utils.DecryptAES(encValue, s.aesKey)
	if err != nil {
		// 解密失败，可能是明文旧数据
		plaintext = encValue
	}

	// 脱敏：显示前8位 + *** + 后4位
	if len(plaintext) <= 12 {
		return plaintext[:2] + "***" + plaintext[len(plaintext)-2:], true
	}
	return plaintext[:8] + "***" + plaintext[len(plaintext)-4:], true
}

// ==================== 场景配置方法 ====================

// GetAllSceneConfigs 获取所有场景配置（含中文名）
func (s *AIConfigService) GetAllSceneConfigs() ([]*models.SceneConfigResponse, error) {
	scenes, err := repository.GetAllSceneConfigs()
	if err != nil {
		return nil, fmt.Errorf("获取场景配置失败: %w", err)
	}

	var result []*models.SceneConfigResponse
	for _, sc := range scenes {
		resp := &models.SceneConfigResponse{
			ID:             sc.ID,
			SceneCode:      sc.SceneCode,
			SceneName:      models.SceneNameMap[sc.SceneCode],
			Model:          sc.Model,
			Temperature:    sc.Temperature,
			MaxTokens:      sc.MaxTokens,
			SystemPromptID: sc.SystemPromptID,
			IsActive:       sc.IsActive,
			UpdatedAt:      sc.UpdatedAt,
		}
		result = append(result, resp)
	}
	return result, nil
}

// UpdateSceneConfig 更新指定场景配置
func (s *AIConfigService) UpdateSceneConfig(code string, req *models.UpdateSceneConfigRequest, userID string) error {
	// 校验场景代码
	if !models.IsValidSceneCode(code) {
		return ErrInvalidSceneCode
	}

	// 校验温度范围（如果提供了值）
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2.0 {
			return ErrInvalidTemperature
		}
	}

	// 校验Token数范围（如果提供了值）
	if req.MaxTokens != nil {
		if *req.MaxTokens < 100 || *req.MaxTokens > 200000 {
			return ErrInvalidMaxTokens
		}
	}

	// 获取现有配置，合并更新
	existing, err := repository.GetSceneConfigByCode(code)
	if err != nil {
		return err
	}

	// 合并：只更新非nil字段
	merged := &models.UpdateSceneConfigRequest{
		Model:          existing.Model,
		Temperature:    existing.Temperature,
		MaxTokens:      existing.MaxTokens,
		SystemPromptID: existing.SystemPromptID,
		IsActive:       &existing.IsActive,
	}
	if req.Model != nil {
		merged.Model = req.Model
	}
	if req.Temperature != nil {
		merged.Temperature = req.Temperature
	}
	if req.MaxTokens != nil {
		merged.MaxTokens = req.MaxTokens
	}
	if req.SystemPromptID != nil {
		merged.SystemPromptID = req.SystemPromptID
	}
	if req.IsActive != nil {
		merged.IsActive = req.IsActive
	}

	return repository.UpdateSceneConfig(code, merged, userID)
}
