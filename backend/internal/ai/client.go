package ai

// AI调用客户端
// 负责从数据库读取AI配置、解密API Key、调用OpenAI兼容API
// 支持三级配置回退：场景配置 → 全局配置 → .env环境变量
//
// 超时策略说明：
//   Pipeline中Translator/Evaluator/Generator等步骤调用Opus模型，
//   通过中转API（oneapi类）时首字节延迟可能超过3-5分钟（排队+推理）。
//   使用900秒（15分钟）作为总超时，足以覆盖最慢的Opus长文本生成。
//   Engine信号量（8并发）防止过多goroutine同时等待，不会耗尽资源。

import (
	"bufio"
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

// ==================== 常量 ====================

const (
	// AICallTimeout AI调用HTTP超时时间（900秒=15分钟）
	AICallTimeout = 900 * time.Second
)

// ==================== 类型定义 ====================

// EffectiveConfig 合并后的有效AI配置
type EffectiveConfig struct {
	APIBaseURL  string  // API基础地址
	APIKey      string  // 解密后的API Key（明文）
	Model       string  // 使用的模型名称
	Temperature float64 // 温度参数（0.0~2.0）
	MaxTokens   int     // 最大Token数
}

// ChatMessage OpenAI兼容的消息格式
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest OpenAI兼容的请求体（非流式）
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// ChatRequestStream OpenAI兼容的请求体（流式）
type ChatRequestStream struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"` // 开启流式输出
}

// ChatResponse OpenAI兼容的响应体（非流式）
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

// StreamChunkResponse OpenAI流式响应中每个SSE数据块的结构
type StreamChunkResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"` // 增量内容（可能为空）
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"` // 非null时表示结束
	} `json:"choices"`
	Model string `json:"model"`
	Usage *struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"` // 部分API在最后一条返回usage
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
// 三级回退策略：场景配置 → 全局配置 → .env环境变量
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
				cfg.APIKey = encKey // 解密失败尝试当明文用
			} else {
				cfg.APIKey = plain
			}
		} else {
			cfg.APIKey = fallbackKey
		}

		cfg.Model = coalesce(global["default_model"], fallbackModel)
		cfg.Temperature = parseFloat(global["temperature"], 0.7)
		cfg.MaxTokens = parseInt(global["max_tokens"], 4096)
	}

	// -------- 第2步：场景配置覆盖（优先级最高）--------
	if sceneCode != "" {
		sceneCfg, sceneErr := repository.GetSceneConfigByCode(sceneCode)
		if sceneErr == nil && sceneCfg != nil && sceneCfg.IsActive {
			if sceneCfg.Model != nil && *sceneCfg.Model != "" {
				cfg.Model = *sceneCfg.Model
			}
			if sceneCfg.Temperature != nil {
				cfg.Temperature = *sceneCfg.Temperature
			}
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

// ==================== 非流式AI调用 ====================

// CallAI 调用AI API（OpenAI兼容格式，等待完整回复后返回）
// 用于Pipeline各步骤、AI评审等需要完整结果的场景
func CallAI(cfg *EffectiveConfig, systemPrompt string, userPrompt string) (*CallResult, error) {
	var messages []ChatMessage
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt})

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

	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	httpClient := &http.Client{Timeout: AICallTimeout}
	startTime := time.Now()

	resp, err := httpClient.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("AI API调用失败（网络错误，超时%s）: %w", AICallTimeout.String(), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取AI响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := extractErrorMessage(respBody)
		return nil, fmt.Errorf("AI API返回错误(HTTP %d): %s", resp.StatusCode, errMsg)
	}

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

	content = stripThinking(content)

	return &CallResult{
		Content:    content,
		ModelUsed:  coalesce(chatResp.Model, cfg.Model),
		TokensUsed: chatResp.Usage.TotalTokens,
		LatencyMs:  latencyMs,
	}, nil
}

// ==================== 流式AI调用 ====================

// CallAIStream 流式调用AI API（OpenAI兼容 SSE 格式）
//
// 原理：
//   OpenAI stream=true 时，服务端以 SSE 格式逐行推送 data: {...} 行，
//   每行包含一个增量 token（delta.content）。
//   本函数逐行读取，每收到一个非空 token 立即调用 onChunk 回调，
//   调用方可在回调中将 token 实时推送给前端。
//
// 参数：
//   cfg      — AI配置（模型/温度/maxTokens）
//   systemPrompt — 系统提示词
//   userPrompt   — 用户消息
//   onChunk      — 每收到一个token片段时的回调，返回error可中止流式读取
//
// 返回：
//   完整的 CallResult（包含全文拼接内容和token统计）
func CallAIStream(
	cfg *EffectiveConfig,
	systemPrompt string,
	userPrompt string,
	onChunk func(chunk string) error,
) (*CallResult, error) {
	// -------- 构造消息列表 --------
	var messages []ChatMessage
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt})

	// -------- 构造流式请求体 --------
	reqBody := ChatRequestStream{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Stream:      true, // 开启流式
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化AI流式请求失败: %w", err)
	}

	// -------- 构造HTTP请求 --------
	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP流式请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream") // 明确声明接受SSE格式

	// 流式请求使用相同的900秒超时（覆盖完整流式传输时长）
	httpClient := &http.Client{Timeout: AICallTimeout}
	startTime := time.Now()

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI流式API调用失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI流式API返回错误(HTTP %d): %s", resp.StatusCode, extractErrorMessage(body))
	}

	// -------- 逐行读取SSE流 --------
	var fullContent strings.Builder // 拼接完整回复
	var modelUsed string
	totalTokens := 0

	scanner := bufio.NewScanner(resp.Body)
	// 设置较大的缓冲区，防止超长行截断
	scanBuf := make([]byte, 64*1024)
	scanner.Buffer(scanBuf, 64*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式：每行以 "data: " 开头
		// 空行是消息分隔符，跳过
		if line == "" || line == ": keep-alive" {
			continue
		}

		// 流式结束标志
		if line == "data: [DONE]" {
			break
		}

		// 提取data内容
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "" {
			continue
		}

		// 解析JSON块
		var chunk StreamChunkResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 解析失败时跳过该行（部分API会发送非标准行）
			continue
		}

		// 记录模型名（首次出现时）
		if modelUsed == "" && chunk.Model != "" {
			modelUsed = chunk.Model
		}

		// 记录token统计（部分API在最后一条携带usage）
		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
		}

		// 提取增量内容
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			// delta为空可能是finish_reason行，检查是否结束
			if chunk.Choices[0].FinishReason != nil {
				break
			}
			continue
		}

		// 清理思维链标签（如果模型逐token输出thinking标签）
		// 注意：这里只做简单过滤，完整的thinking清理在最后对全文处理
		if strings.Contains(delta, "<thinking>") || strings.Contains(delta, "</thinking>") {
			continue
		}

		// 追加到全文
		fullContent.WriteString(delta)

		// 回调：将token推送给调用方
		if onChunk != nil {
			if callbackErr := onChunk(delta); callbackErr != nil {
				// 调用方返回error时中止流式读取（例如客户端断开）
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// scanner错误（如连接中断）不作为fatal，已积累的内容仍然有效
		// 记录日志但继续返回已有内容
		_ = err
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// 对完整内容做一次thinking清理
	content := stripThinking(fullContent.String())
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI流式返回内容为空")
	}

	return &CallResult{
		Content:    content,
		ModelUsed:  coalesce(modelUsed, cfg.Model),
		TokensUsed: totalTokens,
		LatencyMs:  latencyMs,
	}, nil
}

// ==================== JSON提取工具 ====================

// ExtractJSON 从AI输出文本中提取第一个完整的JSON对象
func ExtractJSON(text string) (string, bool) {
	if jsonStr, ok := extractFromCodeBlock(text); ok {
		return jsonStr, true
	}
	start := strings.Index(text, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				candidate := text[start : i+1]
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(candidate), &obj); err == nil {
					return candidate, true
				}
			}
		}
	}
	return "", false
}

// extractFromCodeBlock 从Markdown代码块中提取JSON
func extractFromCodeBlock(text string) (string, bool) {
	for _, marker := range []string{"```json\n", "```\n"} {
		startIdx := strings.Index(text, marker)
		if startIdx < 0 {
			continue
		}
		afterMarker := text[startIdx+len(marker):]
		endIdx := strings.Index(afterMarker, "```")
		if endIdx < 0 {
			continue
		}
		candidate := strings.TrimSpace(afterMarker[:endIdx])
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(candidate), &obj); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// ==================== 内部工具函数 ====================

// stripThinking 移除AI输出中的<thinking>...</thinking>思维链标签
func stripThinking(content string) string {
	for {
		start := strings.Index(content, "<thinking>")
		if start < 0 {
			break
		}
		end := strings.Index(content, "</thinking>")
		if end < 0 {
			content = content[:start] + content[start+len("<thinking>"):]
			break
		}
		content = content[:start] + content[end+len("</thinking>"):]
	}
	return strings.TrimSpace(content)
}

// extractErrorMessage 从错误响应中提取错误信息
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

// coalesce 返回第一个非空字符串
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
