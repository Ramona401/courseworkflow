package ai

// AI流式调用客户端
// 负责SSE流式AI调用（CallAIStream），用于备课对话、AI快修流式等场景
//
// 本文件包含：
//   - 流式请求/响应类型（ChatRequestStream、StreamChunkResponse）
//   - 流式AI调用（CallAIStream + 连接建立重试 + Fallback + SSE解析 + 埋点）
//
// 依赖 client.go 中的：
//   - 类型：EffectiveConfig、TraceContext、ChatMessage、CallResult
//   - 常量：AICallTimeout、MaxStreamRetries、MaxFallbackRetries
//   - 函数：isRetryableError、getRetryDelay、emitTrace、coalesce、stripThinking、
//           extractErrorMessage、getSceneFromTrace
//
// v85新增：主模型连接建立阶段所有重试失败后，依次尝试fallback模型。
//   注意：一旦某个模型的流式连接建立成功并开始推送数据，就不再fallback。
//
// v92重构：从原client.go拆分为独立文件

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ==================== 流式请求/响应类型 ====================

// ChatRequestStream OpenAI兼容的请求体（流式）
type ChatRequestStream struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"` // 开启流式输出
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
		TotalTokens      int `json:"total_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"` // 部分API在最后一条返回usage
}

// ==================== 流式AI调用（带重试 + Fallback + 埋点）====================

// CallAIStream 流式调用AI API（OpenAI兼容 SSE 格式）
//
// v85新增：主模型连接建立阶段所有重试失败后，依次尝试fallback模型。
// 注意：一旦某个模型的流式连接建立成功并开始推送数据，就不再fallback。
//
// 参数：
//
//	cfg          — AI配置（模型/温度/maxTokens/fallback列表）
//	systemPrompt — 系统提示词
//	userPrompt   — 用户消息
//	onChunk      — 每收到一个token片段时的回调，返回error可中止流式读取
//	traceCtx     — 追踪上下文（可为nil）
//
// 返回：
//
//	完整的 CallResult（包含全文拼接内容和token统计）
func CallAIStream(
	cfg *EffectiveConfig,
	systemPrompt string,
	userPrompt string,
	onChunk func(chunk string) error,
	traceCtx *TraceContext,
) (*CallResult, error) {
	// -------- 构造消息列表 --------
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

	// -------- 构建模型尝试列表：主模型 + fallback模型 --------
	type modelAttempt struct {
		model      string
		maxRetries int
		isFallback bool
	}
	modelsToTry := []modelAttempt{
		{model: cfg.Model, maxRetries: MaxStreamRetries, isFallback: false},
	}
	for _, fbModel := range cfg.FallbackModels {
		if fbModel != cfg.Model {
			modelsToTry = append(modelsToTry, modelAttempt{
				model: fbModel, maxRetries: MaxFallbackRetries, isFallback: true,
			})
		}
	}

	// -------- 依次尝试每个模型建立流式连接 --------
	var resp *http.Response
	var lastErr error
	var actualModel string
	var isFallback bool

	for _, modelEntry := range modelsToTry {
		reqBody := ChatRequestStream{
			Model:       modelEntry.model,
			Messages:    messages,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
			Stream:      true,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化AI流式请求失败: %w", err)
		}

		// 带重试的连接建立
		connected := false
		for attempt := 0; attempt <= modelEntry.maxRetries; attempt++ {
			if attempt > 0 {
				delay := getRetryDelay(attempt - 1)
				log.Printf("[AI流式重试] 模型 %s 第%d次重试，等待%s", modelEntry.model, attempt, delay)
				time.Sleep(delay)
			}

			httpReq, reqErr := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
			if reqErr != nil {
				return nil, fmt.Errorf("创建HTTP流式请求失败: %w", reqErr)
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
			httpReq.Header.Set("Accept", "text/event-stream")

			httpClient := &http.Client{Timeout: AICallTimeout}
			resp, err = httpClient.Do(httpReq)
			if err != nil {
				lastErr = fmt.Errorf("AI流式API调用失败（模型 %s）: %w", modelEntry.model, err)
				log.Printf("[AI流式重试] 模型 %s 网络错误（第%d/%d次）: %s",
					modelEntry.model, attempt+1, modelEntry.maxRetries+1, err.Error())
				continue
			}

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				errMsg := extractErrorMessage(body)

				if isRetryableError(resp.StatusCode, body) {
					lastErr = fmt.Errorf("AI流式API返回错误（模型 %s, HTTP %d）: %s",
						modelEntry.model, resp.StatusCode, errMsg)
					log.Printf("[AI流式重试] 模型 %s HTTP %d（第%d/%d次）: %s",
						modelEntry.model, resp.StatusCode, attempt+1, modelEntry.maxRetries+1, errMsg)
					continue
				}
				// 不可重试的错误（如401认证失败），不继续fallback，直接返回
				emitTrace(traceCtx, modelEntry.model, 0, 0, 0,
					time.Since(startTime).Milliseconds(), "error",
					fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errMsg), 0, true,
					modelEntry.isFallback, cfg.Model)
				return nil, fmt.Errorf("AI流式API返回错误(HTTP %d): %s", resp.StatusCode, errMsg)
			}

			// 连接建立成功
			connected = true
			actualModel = modelEntry.model
			isFallback = modelEntry.isFallback
			if attempt > 0 {
				log.Printf("[AI流式重试] 模型 %s 第%d次重试成功", modelEntry.model, attempt)
			}
			break
		}

		if connected {
			if isFallback {
				log.Printf("[AI Fallback] 流式降级模型 %s 连接成功（原始模型: %s）", actualModel, cfg.Model)
			}
			break
		}

		// 当前模型所有重试失败，尝试下一个
		if modelEntry.isFallback {
			log.Printf("[AI Fallback] 流式降级模型 %s 所有重试失败", modelEntry.model)
		} else {
			log.Printf("[AI Fallback] 流式主模型 %s 所有重试失败，开始尝试降级", modelEntry.model)
		}
	}

	// 所有模型都失败
	if resp == nil || lastErr != nil {
		if lastErr == nil {
			lastErr = fmt.Errorf("AI流式调用所有模型均失败")
		}
		emitTrace(traceCtx, cfg.Model, 0, 0, 0,
			time.Since(startTime).Milliseconds(), "error", lastErr.Error(), 0, true, false, "")
		return nil, lastErr
	}
	defer resp.Body.Close()

	// -------- 逐行读取SSE流（此阶段不再重试/fallback）--------
	var fullContent strings.Builder
	var modelUsed string
	totalTokens := 0
	promptTokens := 0
	completionTokens := 0

	scanner := bufio.NewScanner(resp.Body)
	scanBuf := make([]byte, 64*1024)
	scanner.Buffer(scanBuf, 64*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || line == ": keep-alive" {
			continue
		}

		if line == "data: [DONE]" {
			break
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "" {
			continue
		}

		var chunk StreamChunkResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if modelUsed == "" && chunk.Model != "" {
			modelUsed = chunk.Model
		}

		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
			promptTokens = chunk.Usage.PromptTokens
			completionTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			if chunk.Choices[0].FinishReason != nil {
				break
			}
			continue
		}

		if strings.Contains(delta, "<thinking>") || strings.Contains(delta, "</thinking>") {
			continue
		}

		fullContent.WriteString(delta)

		if onChunk != nil {
			if callbackErr := onChunk(delta); callbackErr != nil {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		_ = err
	}

	latencyMs := time.Since(startTime).Milliseconds()

	content := stripThinking(fullContent.String())
	if strings.TrimSpace(content) == "" {
		emitTrace(traceCtx, coalesce(modelUsed, actualModel), totalTokens, promptTokens, completionTokens,
			latencyMs, "error", "AI流式返回内容为空", 0, true, isFallback, cfg.Model)
		return nil, fmt.Errorf("AI流式返回内容为空")
	}

	// 记录成功trace
	emitTrace(traceCtx, coalesce(modelUsed, actualModel), totalTokens, promptTokens, completionTokens,
		latencyMs, "success", "", len(content), true, isFallback, cfg.Model)

	// v129新增：积分消费回调（对齐AOCI的ConsumeCreditsForStream）
	// v129.1修复：部分模型（Gemini等）流式不返回usage统计
	// 当tokens全为0时，用content长度粗估（中文约0.7token/字符，英文约0.25token/字符）
	creditPromptTokens := promptTokens
	creditCompletionTokens := completionTokens
	creditTotalTokens := totalTokens
	if creditTotalTokens == 0 && creditPromptTokens == 0 && creditCompletionTokens == 0 && len(content) > 0 {
		// 粗估：输出约 content长度×0.7 tokens，输入约输出的2倍（系统提示词+对话历史）
		estimatedOutput := int(float64(len(content)) * 0.7)
		estimatedInput := estimatedOutput * 2
		creditPromptTokens = estimatedInput
		creditCompletionTokens = estimatedOutput
		creditTotalTokens = estimatedInput + estimatedOutput
		log.Printf("[AI积分] 流式tokens为0，用content长度(%d字符)估算: in=%d out=%d total=%d",
			len(content), estimatedInput, estimatedOutput, creditTotalTokens)
	}
	invokeCreditConsume(traceCtx, coalesce(modelUsed, actualModel), creditPromptTokens, creditCompletionTokens, creditTotalTokens, latencyMs)

	return &CallResult{
		Content:    content,
		ModelUsed:  coalesce(modelUsed, actualModel),
		TokensUsed: totalTokens,
		LatencyMs:  latencyMs,
	}, nil
}
