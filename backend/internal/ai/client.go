package ai

// AI调用客户端 —— 配置获取 + 非流式调用 + 重试逻辑 + 工具函数 + 追踪埋点
//
// 本文件包含：
//   - 常量定义（超时、重试次数、退避时间表）
//   - 类型定义（EffectiveConfig、TraceContext、ChatMessage、CallResult等）
//   - 配置获取（GetEffectiveConfig 三级回退）
//   - 非流式AI调用（CallAI + callAIWithRetries + callAIOnce）
//   - 重试错误类型（retryableError）
//   - JSON提取工具（ExtractJSON）
//   - 内部工具函数（stripThinking、coalesce、parseFloat、parseInt等）
//   - 追踪埋点（emitTrace）
//
// 流式调用见 client_stream.go
// 多模态调用见 client_multimodal.go
//
// 超时策略说明：
//   Pipeline中Translator/Evaluator/Generator等步骤调用Opus模型，
//   通过中转API（oneapi类）时首字节延迟可能超过3-5分钟（排队+推理）。
//   使用900秒（15分钟）作为总超时，足以覆盖最慢的Opus长文本生成。
//   Engine信号量（8并发）防止过多goroutine同时等待，不会耗尽资源。
//
// 重试策略说明（v67.1新增）：
//   OneAPI等中转API的nginx可能在Opus长推理时返回502/504超时错误，
//   或返回500+不完整JSON。这些都是临时性错误，自动重试可有效恢复。
//   非流式CallAI最多重试3次（间隔30s/60s/120s指数退避）。
//   流式CallAIStream最多重试2次（间隔30s/60s）。
//
// v80新增：AI调用追踪埋点
//   每次CallAI/CallAIStream/CallAIMultimodal调用完成后（成功或失败），
//   通过repository.EnqueueTrace异步写入ai_call_traces表。
//   埋点不影响主路径延迟（非阻塞channel写入）。
//   场景代码通过TraceContext传入，调用方负责设置。
//
// v85新增：多模型Fallback降级
//   GetEffectiveConfig返回FallbackModels列表（从ai_scene_configs.fallback_models获取）。
//   CallAI/CallAIStream/CallAIMultimodal在主模型所有重试耗尽后，
//   依次尝试fallback模型（每个fallback模型有1次重试机会）。
//   降级调用的trace记录标记is_fallback=true + original_model=主模型。
//
// v92重构：从原1036行client.go拆分为3个文件（client.go + client_stream.go + client_multimodal.go）

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 常量 ====================

const (
	// AICallTimeout AI调用HTTP超时时间（900秒=15分钟）
	AICallTimeout = 900 * time.Second

	// MaxRetries 非流式调用最大重试次数（首次调用 + 最多3次重试 = 总共4次尝试）
	MaxRetries = 3

	// MaxStreamRetries 流式调用最大重试次数（首次调用 + 最多2次重试 = 总共3次尝试）
	MaxStreamRetries = 2

	// MaxFallbackRetries 每个fallback模型的最大重试次数（v85新增）
	MaxFallbackRetries = 1
)

// retryDelays 重试间隔时间表（指数退避：30秒 → 60秒 → 120秒）
var retryDelays = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

// ==================== 类型定义 ====================

// EffectiveConfig 合并后的有效AI配置
type EffectiveConfig struct {
	APIBaseURL     string   // API基础地址
	APIKey         string   // 解密后的API Key（明文）
	Model          string   // 使用的模型名称
	Temperature    float64  // 温度参数（0.0~2.0）
	MaxTokens      int      // 最大Token数
	FallbackModels []string // v85新增：降级模型列表（按优先级排序）
}

// TraceContext AI调用追踪上下文（v80新增）
// 调用方在调用CallAI等方法前设置，用于关联trace记录到业务实体
type TraceContext struct {
	SceneCode    string  // 场景代码（必填，如scanner/evaluator/lesson_plan）
	PipelineID   *string // 关联Pipeline ID（Pipeline步骤时设置）
	LessonPlanID *string // 关联教案 ID（备课对话时设置）
	UserID       *string // 关联用户 ID
	SchoolID     *string // v129新增：用户所属学校ID，用于积分策略查询
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

// ChatResponse OpenAI兼容的响应体（非流式）
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens      int `json:"total_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
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

// ==================== 重试错误类型 ====================

// retryableError 标记可重试的错误
type retryableError struct {
	msg string
}

func (e *retryableError) Error() string {
	return e.msg
}

// isRetryableCallError 判断错误是否为可重试类型
func isRetryableCallError(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}

// ==================== 配置获取 ====================

// GetEffectiveConfig 获取指定场景的有效AI配置
// 三级回退策略：场景配置 → 全局配置 → .env环境变量
// v85变更：新增FallbackModels字段从场景配置读取
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
			// v85: 读取fallback模型列表
			if len(sceneCfg.FallbackModels) > 0 {
				cfg.FallbackModels = sceneCfg.FallbackModels
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

// ==================== 重试辅助函数 ====================

// isRetryableError 判断HTTP状态码是否可以重试
// 502 Bad Gateway / 503 Service Unavailable / 504 Gateway Timeout 是典型的临时性错误
// 500 Internal Server Error 当响应内容包含典型超时特征时也可重试
func isRetryableError(statusCode int, body []byte) bool {
	switch statusCode {
	case 502, 503, 504:
		return true
	case 500:
		bodyStr := string(body)
		retryablePatterns := []string{
			"unexpected end of JSON input",
			"connection reset",
			"broken pipe",
			"EOF",
			"timeout",
			"Gateway Time-out",
			"Bad Gateway",
			"upstream",
		}
		for _, pattern := range retryablePatterns {
			if strings.Contains(bodyStr, pattern) {
				return true
			}
		}
		return false
	case 429:
		return true
	default:
		return false
	}
}

// getRetryDelay 获取第n次重试的等待时间（从0开始计数）
func getRetryDelay(attempt int) time.Duration {
	if attempt < len(retryDelays) {
		return retryDelays[attempt]
	}
	return retryDelays[len(retryDelays)-1]
}

// ==================== 非流式AI调用（带重试 + Fallback + 埋点）====================

// CallAI 调用AI API（OpenAI兼容格式，等待完整回复后返回）
// 用于Pipeline各步骤、AI评审等需要完整结果的场景
// 遇到502/503/504/429等临时错误时自动重试（最多3次，指数退避）
// v85新增：主模型所有重试耗尽后，依次尝试fallback模型
//
// traceCtx参数用于关联trace记录到业务实体，如果为nil不记录trace
func CallAI(cfg *EffectiveConfig, systemPrompt string, userPrompt string, traceCtx *TraceContext) (*CallResult, error) {
	var messages []ChatMessage
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt})

	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + "/chat/completions"
	startTime := time.Now()

	// v129新增：积分前置检查（对齐AOCI的checkCreditsGate）
	if allowed, errMsg := invokeCreditCheck(traceCtx); !allowed {
		return nil, fmt.Errorf(errMsg)
	}

	// -------- 第1阶段：主模型调用（带完整重试）--------
	primaryModel := cfg.Model
	result, err := callAIWithRetries(cfg, primaryModel, messages, endpoint, MaxRetries)
	if err == nil {
		// 主模型调用成功
		latMs := time.Since(startTime).Milliseconds()
		emitTrace(traceCtx, result.ModelUsed, result.TokensUsed, 0, 0,
			latMs, "success", "", len(result.Content), false, false, "")
		// v129新增：积分消费回调（对齐AOCI的ConsumeCredits）
		invokeCreditConsume(traceCtx, result.ModelUsed, 0, 0, result.TokensUsed, latMs)
		return result, nil
	}

	primaryErr := err
	log.Printf("[AI Fallback] 主模型 %s 所有重试失败: %s", primaryModel, err.Error())

	// -------- 第2阶段：依次尝试fallback模型（v85新增）--------
	for i, fbModel := range cfg.FallbackModels {
		// 跳过与主模型相同的fallback
		if fbModel == primaryModel {
			continue
		}

		log.Printf("[AI Fallback] 尝试降级模型 %d/%d: %s（场景: %s）",
			i+1, len(cfg.FallbackModels), fbModel, getSceneFromTrace(traceCtx))

		result, err = callAIWithRetries(cfg, fbModel, messages, endpoint, MaxFallbackRetries)
		if err == nil {
			// fallback模型调用成功
			fbLatMs := time.Since(startTime).Milliseconds()
			log.Printf("[AI Fallback] 降级模型 %s 调用成功（原始模型: %s）", fbModel, primaryModel)
			emitTrace(traceCtx, result.ModelUsed, result.TokensUsed, 0, 0,
				fbLatMs, "success", "", len(result.Content), false,
				true, primaryModel)
			// v129新增：降级模型成功也扣减积分
			invokeCreditConsume(traceCtx, result.ModelUsed, 0, 0, result.TokensUsed, fbLatMs)
			return result, nil
		}

		log.Printf("[AI Fallback] 降级模型 %s 也失败: %s", fbModel, err.Error())
	}

	// 所有模型（主+fallback）都失败
	totalLatency := time.Since(startTime).Milliseconds()
	emitTrace(traceCtx, primaryModel, 0, 0, 0,
		totalLatency, "error", primaryErr.Error(), 0, false, false, "")
	return nil, fmt.Errorf("AI调用失败（主模型 %s + %d个降级模型均失败）: %w",
		primaryModel, len(cfg.FallbackModels), primaryErr)
}

// callAIWithRetries 使用指定模型执行非流式AI调用（带重试）
// 这是从CallAI中提取的核心重试循环，供主模型和fallback模型复用
func callAIWithRetries(cfg *EffectiveConfig, model string, messages []ChatMessage, endpoint string, maxRetries int) (*CallResult, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化AI请求失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 非首次调用时等待退避时间
		if attempt > 0 {
			delay := getRetryDelay(attempt - 1)
			log.Printf("[AI重试] 模型 %s 第%d次重试，等待%s", model, attempt, delay)
			time.Sleep(delay)
		}

		result, err := callAIOnce(cfg, endpoint, jsonBody)
		if err == nil {
			if attempt > 0 {
				log.Printf("[AI重试] 模型 %s 第%d次重试成功", model, attempt)
			}
			return result, nil
		}

		lastErr = err

		// 判断是否可重试
		if !isRetryableCallError(err) {
			return nil, err
		}

		log.Printf("[AI重试] 模型 %s 调用失败（第%d/%d次）: %s",
			model, attempt+1, maxRetries+1, err.Error())
	}

	return nil, fmt.Errorf("模型 %s 在%d次尝试后失败: %w", model, maxRetries+1, lastErr)
}

// callAIOnce 执行单次非流式AI调用（不含重试逻辑）
func callAIOnce(cfg *EffectiveConfig, endpoint string, jsonBody []byte) (*CallResult, error) {
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
		return nil, &retryableError{
			msg: fmt.Sprintf("AI API调用失败（网络错误，超时%s）: %s", AICallTimeout.String(), err.Error()),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryableError{
			msg: fmt.Sprintf("读取AI响应失败: %s", err.Error()),
		}
	}

	// 非200状态码：判断是否可重试
	if resp.StatusCode != http.StatusOK {
		errMsg := extractErrorMessage(respBody)
		if isRetryableError(resp.StatusCode, respBody) {
			return nil, &retryableError{
				msg: fmt.Sprintf("AI API返回错误(HTTP %d): %s", resp.StatusCode, errMsg),
			}
		}
		return nil, fmt.Errorf("AI API返回错误(HTTP %d): %s", resp.StatusCode, errMsg)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, &retryableError{
			msg: fmt.Sprintf("解析AI响应JSON失败: %s", err.Error()),
		}
	}

	if len(chatResp.Choices) == 0 {
		return nil, &retryableError{
			msg: "AI响应中choices为空，未获得有效输出",
		}
	}

	content := chatResp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, &retryableError{
			msg: "AI返回内容为空",
		}
	}

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

// getSceneFromTrace 从TraceContext中安全获取场景代码（辅助日志）
func getSceneFromTrace(traceCtx *TraceContext) string {
	if traceCtx == nil {
		return "unknown"
	}
	return traceCtx.SceneCode
}

// ==================== v80新增 + v85改造：追踪埋点辅助函数 ====================

// emitTrace 异步记录AI调用trace（不阻塞主路径）
// 如果traceCtx为nil，静默跳过（兼容未传traceCtx的旧调用）
// v85新增：isFallback和originalModel参数记录降级信息
func emitTrace(
	traceCtx *TraceContext,
	modelUsed string,
	totalTokens int,
	promptTokens int,
	completionTokens int,
	latencyMs int64,
	status string,
	errorMsg string,
	outputLength int,
	isStream bool,
	isFallback bool,
	originalModel string,
) {
	if traceCtx == nil {
		return
	}

	// 仅在实际降级时记录originalModel
	actualOriginalModel := ""
	if isFallback {
		actualOriginalModel = originalModel
	}

	rec := models.TraceRecord{
		SceneCode:        traceCtx.SceneCode,
		ModelUsed:        modelUsed,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		LatencyMs:        latencyMs,
		Status:           status,
		ErrorMessage:     errorMsg,
		PipelineID:       traceCtx.PipelineID,
		LessonPlanID:     traceCtx.LessonPlanID,
		UserID:           traceCtx.UserID,
		OutputLength:     outputLength,
		IsStream:         isStream,
		IsFallback:       isFallback,
		OriginalModel:    actualOriginalModel,
	}

	repository.EnqueueTrace(rec)
}
