package ai

// client_stream.go
//
// 提供OpenAI兼容的流式文本调用。连接建立前允许重试和Fallback；一旦开始
// 读取响应流，任何上游读取错误、请求取消、协议中断或消费回调错误都会
// 直接返回，禁止把部分输出误判为成功。旧CallAIStream入口保持兼容，
// 新入口支持context和完整user/assistant历史消息。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
)

var streamLog = logger.WithModule("ai.stream")

// ChatRequestStream 是OpenAI兼容的流式请求体。
type ChatRequestStream struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

// StreamChunkResponse 是单个SSE数据块。
type StreamChunkResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage *struct {
		TotalTokens      int `json:"total_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// StreamCallResult 包含运行时原子结算需要的输入、输出Token拆分。
type StreamCallResult struct {
	CallResult
	PromptTokens     int
	CompletionTokens int
}

// CallAIStream 保留旧入口。
func CallAIStream(
	cfg *EffectiveConfig,
	systemPrompt string,
	userPrompt string,
	onChunk func(string) error,
	traceCtx *TraceContext,
) (*CallResult, error) {
	return CallAIStreamContext(
		context.Background(),
		cfg,
		systemPrompt,
		userPrompt,
		onChunk,
		traceCtx,
	)
}

// CallAIStreamContext 提供可取消的system+user流式入口。
func CallAIStreamContext(
	ctx context.Context,
	cfg *EffectiveConfig,
	systemPrompt string,
	userPrompt string,
	onChunk func(string) error,
	traceCtx *TraceContext,
) (*CallResult, error) {
	messages := make([]ChatMessage, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt})

	result, err := CallAIStreamMessagesContext(ctx, cfg, messages, onChunk, traceCtx)
	if err != nil {
		return nil, err
	}
	return &result.CallResult, nil
}

// CallAIStreamMessagesContext 使用完整正式消息历史执行可取消流式调用。
func CallAIStreamMessagesContext(
	ctx context.Context,
	cfg *EffectiveConfig,
	messages []ChatMessage,
	onChunk func(string) error,
	traceCtx *TraceContext,
) (*StreamCallResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, fmt.Errorf("AI流式配置不能为空")
	}
	if err := validateAIStreamMessages(messages); err != nil {
		return nil, err
	}

	// 在完整消息入口统一应用系统级数学输出策略，覆盖旧system+user入口和正式历史消息入口。
	runtimeMessages := applyMathOutputPolicy(cfg, messages, traceCtx)
	runtimeConfig := cloneAIStreamConfig(cfg)
	startTime := time.Now()

	if allowed, errMsg := invokeCreditCheck(traceCtx); !allowed {
		return nil, fmt.Errorf("%s", errMsg)
	}

	// 分流必须先于endpoint和模型尝试链计算。
	applyModelPolicy(runtimeConfig, traceCtx)
	endpoint := strings.TrimRight(runtimeConfig.APIBaseURL, "/") + "/chat/completions"

	response, actualModel, usedFallback, err := openAIStreamResponse(
		ctx,
		runtimeConfig,
		endpoint,
		runtimeMessages,
	)
	if err != nil {
		emitTrace(
			traceCtx,
			runtimeConfig.Model,
			0,
			0,
			0,
			time.Since(startTime).Milliseconds(),
			"error",
			err.Error(),
			0,
			true,
			false,
			"",
		)
		return nil, err
	}
	defer response.Body.Close()

	readResult, err := readAIStreamResponse(response, onChunk)
	latencyMs := time.Since(startTime).Milliseconds()
	modelUsed := coalesce(readResult.ModelUsed, actualModel)

	if err != nil {
		emitTrace(
			traceCtx,
			modelUsed,
			readResult.TotalTokens,
			readResult.PromptTokens,
			readResult.CompletionTokens,
			latencyMs,
			"error",
			err.Error(),
			len(readResult.Content),
			true,
			usedFallback,
			runtimeConfig.Model,
		)
		return nil, err
	}

	content := stripThinking(readResult.Content)
	if strings.TrimSpace(content) == "" {
		err = fmt.Errorf("AI流式返回内容为空")
		emitTrace(
			traceCtx,
			modelUsed,
			readResult.TotalTokens,
			readResult.PromptTokens,
			readResult.CompletionTokens,
			latencyMs,
			"error",
			err.Error(),
			0,
			true,
			usedFallback,
			runtimeConfig.Model,
		)
		return nil, err
	}

	promptTokens, completionTokens, totalTokens := normalizeAIStreamTokenUsage(
		runtimeMessages,
		content,
		readResult.PromptTokens,
		readResult.CompletionTokens,
		readResult.TotalTokens,
	)

	emitTrace(
		traceCtx,
		modelUsed,
		totalTokens,
		promptTokens,
		completionTokens,
		latencyMs,
		"success",
		"",
		len(content),
		true,
		usedFallback,
		runtimeConfig.Model,
	)

	invokeCreditConsume(
		traceCtx,
		modelUsed,
		promptTokens,
		completionTokens,
		totalTokens,
		latencyMs,
	)

	return &StreamCallResult{
		CallResult: CallResult{
			Content:    content,
			ModelUsed:  modelUsed,
			TokensUsed: totalTokens,
			LatencyMs:  latencyMs,
		},
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}, nil
}
