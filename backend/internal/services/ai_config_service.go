package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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
	ErrAPIKeyNotSet       = errors.New("API Key尚未配置，请先在全局配置中设置API Key")
)

// ==================== AI连通性测试结果 ====================

// TestConnectionResult AI连通性测试结果
type TestConnectionResult struct {
	Success    bool   `json:"success"`      // 测试是否成功
	Message    string `json:"message"`      // 结果描述
	LatencyMs  int64  `json:"latency_ms"`   // 延迟（毫秒）
	Model      string `json:"model"`        // 测试使用的模型
	APIBaseURL string `json:"api_base_url"` // 测试使用的API地址
}

// ==================== 可用模型信息 ====================

// ModelInfo 单个可用模型信息
type ModelInfo struct {
	ID string `json:"id"` // 模型ID（如 anthropic/claude-haiku-4.5）
}

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

// ==================== AI连通性测试方法（P2-2新增）====================

// TestConnection 测试AI API连通性
// 使用当前全局配置（API Base URL + Key + Model）向AI API发送简短测试请求
// 返回测试结果（成功/失败/延迟毫秒数）
func (s *AIConfigService) TestConnection() (*TestConnectionResult, error) {
	// 1. 获取全局配置中的API Base URL
	baseURLConfig, err := repository.GetConfigByKey(models.ConfigKeyAPIBaseURL)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "无法获取API基础地址配置: " + err.Error(),
		}, nil
	}
	apiBaseURL := strings.TrimSpace(baseURLConfig.ConfigValue)
	if apiBaseURL == "" {
		return &TestConnectionResult{
			Success: false,
			Message: "API基础地址未配置",
		}, nil
	}

	// 2. 获取解密后的API Key
	apiKey, err := s.GetDecryptedAPIKey()
	if err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "API Key未配置或解密失败: " + err.Error(),
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 3. 获取默认模型
	modelConfig, err := repository.GetConfigByKey(models.ConfigKeyDefaultModel)
	if err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "无法获取默认模型配置: " + err.Error(),
			APIBaseURL: apiBaseURL,
		}, nil
	}
	modelName := strings.TrimSpace(modelConfig.ConfigValue)
	if modelName == "" {
		return &TestConnectionResult{
			Success:    false,
			Message:    "默认模型未配置",
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 4. 构造OpenAI兼容格式的测试请求
	// 使用最简短的消息，降低Token消耗
	requestBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens":  10,  // 限制响应长度，节省Token
		"temperature": 0.0, // 确定性输出
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "构造请求体失败: " + err.Error(),
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 5. 构造HTTP请求
	// API地址格式：{base_url}/chat/completions（OpenAI兼容）
	endpoint := strings.TrimRight(apiBaseURL, "/") + "/chat/completions"

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "创建HTTP请求失败: " + err.Error(),
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 设置请求头（OpenAI兼容格式）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 6. 发送请求并计时
	// 设置30秒超时，避免长时间等待
	httpClient := &http.Client{Timeout: 30 * time.Second}

	startTime := time.Now()
	resp, err := httpClient.Do(req)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		// 网络错误（超时、DNS解析失败、连接拒绝等）
		return &TestConnectionResult{
			Success:    false,
			Message:    "网络连接失败: " + err.Error(),
			LatencyMs:  latencyMs,
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}
	defer resp.Body.Close()

	// 7. 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "读取响应失败: " + err.Error(),
			LatencyMs:  latencyMs,
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 8. 判断HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试从响应体提取错误信息
		errMsg := s.extractAPIErrorMessage(respBody)
		statusText := fmt.Sprintf("HTTP %d", resp.StatusCode)

		// 针对常见错误码给出友好提示
		switch resp.StatusCode {
		case 401:
			statusText = "认证失败(401) — API Key无效或已过期"
		case 403:
			statusText = "访问被拒绝(403) — 无权限访问该模型"
		case 404:
			statusText = "接口不存在(404) — 请检查API地址是否正确"
		case 429:
			statusText = "请求过于频繁(429) — API频率限制"
		case 500, 502, 503:
			statusText = fmt.Sprintf("服务端错误(%d) — AI服务暂时不可用", resp.StatusCode)
		}

		message := statusText
		if errMsg != "" {
			message = statusText + "；详情: " + errMsg
		}

		return &TestConnectionResult{
			Success:    false,
			Message:    message,
			LatencyMs:  latencyMs,
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 9. 解析成功响应，验证内容有效
	var chatResp map[string]interface{}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return &TestConnectionResult{
			Success:    false,
			Message:    "响应格式异常，非标准JSON",
			LatencyMs:  latencyMs,
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 检查响应中是否包含choices字段（OpenAI兼容格式标志）
	if _, ok := chatResp["choices"]; !ok {
		return &TestConnectionResult{
			Success:    false,
			Message:    "响应缺少choices字段，可能不是OpenAI兼容API",
			LatencyMs:  latencyMs,
			Model:      modelName,
			APIBaseURL: apiBaseURL,
		}, nil
	}

	// 10. 测试成功
	return &TestConnectionResult{
		Success:    true,
		Message:    fmt.Sprintf("连接成功！响应延迟 %dms", latencyMs),
		LatencyMs:  latencyMs,
		Model:      modelName,
		APIBaseURL: apiBaseURL,
	}, nil
}

// extractAPIErrorMessage 从AI API错误响应中提取错误信息
// 支持OpenAI标准错误格式：{"error":{"message":"...","type":"..."}}
func (s *AIConfigService) extractAPIErrorMessage(body []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return errResp.Error.Message
	}

	// 如果无法解析，返回截断的原始响应（最多200字符）
	raw := string(body)
	if len(raw) > 200 {
		raw = raw[:200] + "..."
	}
	if raw != "" {
		return raw
	}
	return ""
}

// ==================== 可用模型查询方法 ====================

// ListModels 查询当前 Key 下可用的模型列表
// 调用上游 {api_base_url}/models 接口（OpenAI兼容），返回模型ID列表（按字母排序）
func (s *AIConfigService) ListModels() ([]ModelInfo, error) {
	// 1. 获取API Base URL
	baseURLConfig, err := repository.GetConfigByKey(models.ConfigKeyAPIBaseURL)
	if err != nil || strings.TrimSpace(baseURLConfig.ConfigValue) == "" {
		return nil, errors.New("API基础地址未配置")
	}
	apiBaseURL := strings.TrimRight(strings.TrimSpace(baseURLConfig.ConfigValue), "/")

	// 2. 获取解密后的API Key
	apiKey, err := s.GetDecryptedAPIKey()
	if err != nil {
		return nil, ErrAPIKeyNotSet
	}

	// 3. 调用 GET {base_url}/models 接口（OpenAI兼容标准端点）
	endpoint := apiBaseURL + "/models"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 15秒超时，模型列表查询通常很快
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 4. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 5. 处理非200状态码
	if resp.StatusCode != http.StatusOK {
		errMsg := s.extractAPIErrorMessage(body)
		if errMsg != "" {
			return nil, fmt.Errorf("API返回错误(HTTP %d): %s", resp.StatusCode, errMsg)
		}
		return nil, fmt.Errorf("API返回错误(HTTP %d)，该Key可能无权查询模型列表", resp.StatusCode)
	}

	// 6. 解析 OpenAI 兼容格式：{"object":"list","data":[{"id":"..."},...]}
	var listResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("解析模型列表响应失败: %w", err)
	}

	// 7. 提取模型ID并按字母排序，过滤空ID
	result := make([]ModelInfo, 0, len(listResp.Data))
	for _, m := range listResp.Data {
		if strings.TrimSpace(m.ID) != "" {
			result = append(result, ModelInfo{ID: m.ID})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
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
				SceneGroup:     models.SceneGroupMap[sc.SceneCode],
			Model:          sc.Model,
			Temperature:    sc.Temperature,
			MaxTokens:      sc.MaxTokens,
			SystemPromptID: sc.SystemPromptID,
			IsActive:       sc.IsActive,
			UpdatedAt:      sc.UpdatedAt,
		}
		// 兜底：未注册的场景给默认值
			if resp.SceneName == "" {
				resp.SceneName = sc.SceneCode
			}
			if resp.SceneGroup == "" {
				resp.SceneGroup = "pipeline"
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
