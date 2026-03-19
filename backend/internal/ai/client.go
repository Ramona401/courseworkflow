package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== AI调用客户端（P4-2新增）====================
// 负责从数据库读取AI配置、解密API Key、调用OpenAI兼容API
// 支持三级配置回退：场景配置 → 全局配置 → .env环境变量

// EffectiveConfig 合并后的有效AI配置
// 由GetEffectiveConfig()计算得出，供CallAI()使用
type EffectiveConfig struct {
	APIBaseURL  string  // API基础地址（如 https://oneapi.xingyunlink.com/v1）
	APIKey      string  // 解密后的API Key（明文）
	Model       string  // 使用的模型名称
	Temperature float64 // 温度参数（0.0~2.0）
	MaxTokens   int     // 最大Token数
}

// ChatMessage OpenAI兼容的消息格式
type ChatMessage struct {
	Role    string `json:"role"`    // 角色：system/user/assistant
	Content string `json:"content"` // 消息内容
}

// ChatRequest OpenAI兼容的请求体
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// ChatResponse OpenAI兼容的响应体
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

// CallResult AI调用结果
type CallResult struct {
	Content    string // AI输出的文本内容
	ModelUsed  string // 实际使用的模型
	TokensUsed int    // 消耗的Token数
	LatencyMs  int64  // 调用耗时（毫秒）
}

// ==================== 配置获取 ====================

// GetEffectiveConfig 获取指定场景的有效AI配置
// 三级回退策略：
//   1. 场景配置（ai_scene_configs表中scene_code对应记录）
//   2. 全局配置（ai_configs表）
//   3. .env环境变量（兜底）
//
// 参数 aesKey：用于解密数据库中存储的加密API Key
// 参数 sceneCode：场景代码（scanner/evaluator/meta/translator/reviewer/generator）
// 参数 fallbackBaseURL/fallbackKey/fallbackModel：来自.env的兜底值
func GetEffectiveConfig(
	aesKey string,
	sceneCode string,
	fallbackBaseURL string,
	fallbackKey string,
	fallbackModel string,
) (*EffectiveConfig, error) {
	cfg := &EffectiveConfig{}

	// -------- 第1步：读取全局配置作为基础 --------
	globalConfigs, err := repository.GetAllConfigs()
	if err != nil {
		// 全局配置读取失败，使用.env兜底
		cfg.APIBaseURL = fallbackBaseURL
		cfg.APIKey = fallbackKey
		cfg.Model = fallbackModel
		cfg.Temperature = 0.7
		cfg.MaxTokens = 4096
	} else {
		// 构建全局配置映射
		global := make(map[string]string)
		for _, c := range globalConfigs {
			global[c.ConfigKey] = c.ConfigValue
		}

		// API Base URL：全局 → .env兜底
		cfg.APIBaseURL = coalesce(global["api_base_url"], fallbackBaseURL)

		// API Key：从数据库解密
		if encKey, ok := global["api_key_enc"]; ok && encKey != "" && encKey != "PLACEHOLDER_SET_IN_ADMIN" {
			plain, decErr := utils.DecryptAES(encKey, aesKey)
			if decErr != nil {
				// 解密失败，尝试作为明文使用
				cfg.APIKey = encKey
			} else {
				cfg.APIKey = plain
			}
		} else {
			// 数据库中无有效Key，使用.env兜底
			cfg.APIKey = fallbackKey
		}

		// 默认模型
		cfg.Model = coalesce(global["default_model"], fallbackModel)

		// 温度（全局默认0.7）
		cfg.Temperature = parseFloat(global["temperature"], 0.7)

		// 最大Token数（全局默认4096）
		cfg.MaxTokens = parseInt(global["max_tokens"], 4096)
	}

	// -------- 第2步：场景配置覆盖（优先级最高）--------
	if sceneCode != "" {
		sceneCfg, sceneErr := repository.GetSceneConfigByCode(sceneCode)
		if sceneErr == nil && sceneCfg != nil && sceneCfg.IsActive {
			// 场景有独立模型配置时覆盖
			if sceneCfg.Model != nil && *sceneCfg.Model != "" {
				cfg.Model = *sceneCfg.Model
			}
			// 场景有独立温度时覆盖
			if sceneCfg.Temperature != nil {
				cfg.Temperature = *sceneCfg.Temperature
			}
			// 场景有独立Token数时覆盖
			if sceneCfg.MaxTokens != nil {
				cfg.MaxTokens = *sceneCfg.MaxTokens
			}
		}
	}

	// -------- 第3步：验证必要字段 --------
	if cfg.APIBaseURL == "" {
		return nil, fmt.Errorf("AI API地址未配置，请在全局配置中设置api_base_url")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("AI API Key未配置，请在全局配置中设置api_key_enc")
	}
	if cfg.Model == "" {
		cfg.Model = "anthropic/claude-sonnet-4-5" // 最终兜底模型
	}

	return cfg, nil
}

// ==================== AI调用 ====================

// CallAI 调用AI API（OpenAI兼容格式）
// 参数说明：
//   cfg        — 由GetEffectiveConfig()得到的有效配置
//   systemPrompt — 系统提示词（可为空字符串）
//   userPrompt   — 用户消息（必填）
//
// 返回：*CallResult（含输出内容、模型名、Token数、耗时）
func CallAI(cfg *EffectiveConfig, systemPrompt string, userPrompt string) (*CallResult, error) {
	// -------- 构造消息列表 --------
	var messages []ChatMessage

	// 系统提示词（若非空则添加）
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// 用户消息（必须有）
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userPrompt,
	})

	// -------- 构造请求体 --------
	reqBody := ChatRequest{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化AI请求失败: %w", err)
	}

	// -------- 构造HTTP请求 --------
	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + "/chat/completions"

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// -------- 发送请求 --------
	// 超时时间120秒（AI生成可能较慢，特别是长文本）
	httpClient := &http.Client{Timeout: 120 * time.Second}
	startTime := time.Now()

	resp, err := httpClient.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return nil, fmt.Errorf("AI API调用失败（网络错误）: %w", err)
	}
	defer resp.Body.Close()

	// -------- 读取响应 --------
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取AI响应失败: %w", err)
	}

	// -------- 检查HTTP状态码 --------
	if resp.StatusCode != http.StatusOK {
		errMsg := extractErrorMessage(respBody)
		return nil, fmt.Errorf("AI API返回错误(HTTP %d): %s", resp.StatusCode, errMsg)
	}

	// -------- 解析响应 --------
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("解析AI响应JSON失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("AI响应中choices为空，未获得有效输出")
	}

	content := chatResp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI返回内容为空")
	}

	// -------- 清理思维链标签（部分模型会输出<thinking>...）--------
	content = stripThinking(content)

	return &CallResult{
		Content:    content,
		ModelUsed:  coalesce(chatResp.Model, cfg.Model),
		TokensUsed: chatResp.Usage.TotalTokens,
		LatencyMs:  latencyMs,
	}, nil
}

// ==================== JSON提取工具 ====================

// ExtractJSON 从AI输出文本中提取第一个完整的JSON对象
// 处理AI常见输出格式：
//   1. 纯JSON: {"key":"value"}
//   2. Markdown代码块: ```json
{...}
```
//   3. 混合文本: 前后有说明文字，JSON在中间
func ExtractJSON(text string) (string, bool) {
	// 优先尝试从Markdown代码块提取
	if jsonStr, ok := extractFromCodeBlock(text); ok {
		return jsonStr, true
	}

	// 尝试直接提取花括号范围内的JSON
	start := strings.Index(text, "{")
	if start < 0 {
		return "", false
	}

	// 从第一个{开始，找到匹配的}
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				candidate := text[start : i+1]
				// 验证是否是有效JSON
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(candidate), &obj); err == nil {
					return candidate, true
				}
			}
		}
	}
	return "", false
}

// extractFromCodeBlock 从Markdown代码块中提取JSON内容
// 支持格式：```json ... ``` 或 ``` ... ```
func extractFromCodeBlock(text string) (string, bool) {
	// 查找代码块开始标记
	markers := []string{"' + ``` + 'json
", "' + ``` + '
"}
	for _, marker := range markers {
		startIdx := strings.Index(text, marker)
		if startIdx < 0 {
			continue
		}
		content := text[startIdx+len(marker):]
		// 查找代码块结束标记
		endIdx := strings.Index(content, "' + ``` + '")
		if endIdx < 0 {
			continue
		}
		candidate := strings.TrimSpace(content[:endIdx])
		// 验证JSON有效性
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(candidate), &obj); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// ==================== 内部工具函数 ====================

// stripThinking 移除AI输出中的<thinking>...</thinking>思维链标签
// 部分推理模型（如claude-3.7-sonnet-thinking）会在输出前加入内部思考过程
func stripThinking(content string) string {
	// 处理<thinking>...</thinking>标签
	for {
		start := strings.Index(content, "<thinking>")
		if start < 0 {
			break
		}
		end := strings.Index(content, "</thinking>")
		if end < 0 {
			// 有开始标签但无结束标签，截断开始标签前的内容
			content = content[:start] + content[start+len("<thinking>"):]
			break
		}
		// 移除整个thinking块（含标签）
		content = content[:start] + content[end+len("</thinking>"):]
	}
	return strings.TrimSpace(content)
}

// extractErrorMessage 从AI API错误响应中提取错误信息
func extractErrorMessage(body []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return errResp.Error.Message
	}
	raw := string(body)
	if len(raw) > 300 {
		return raw[:300] + "..."
	}
	return raw
}

// coalesce 返回第一个非空字符串（类似SQL COALESCE）
func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// parseFloat 解析浮点数，失败返回默认值
func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return defaultVal
	}
	return v
}

// parseInt 解析整数，失败返回默认值
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return defaultVal
	}
	return v
}
