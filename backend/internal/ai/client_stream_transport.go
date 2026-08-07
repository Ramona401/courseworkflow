package ai

// client_stream_transport.go
//
// 负责流式连接建立与SSE响应读取。连接建立前允许重试和Fallback；
// 响应开始读取后只接受完整协议终态，不对部分输出执行重试。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openAIStreamResponse 只负责在尚未输出内容前建立连接。
func openAIStreamResponse(
	ctx context.Context,
	cfg *EffectiveConfig,
	endpoint string,
	messages []ChatMessage,
) (
	*http.Response,
	string,
	bool,
	error,
) {
	type modelAttempt struct {
		model      string
		maxRetries int
		fallback   bool
	}

	attempts :=
		[]modelAttempt{
			{
				model:      cfg.Model,
				maxRetries: MaxStreamRetries,
			},
		}

	for _, model := range cfg.FallbackModels {
		model =
			strings.TrimSpace(
				model,
			)

		if model != "" &&
			model != cfg.Model {
			attempts =
				append(
					attempts,
					modelAttempt{
						model:      model,
						maxRetries: MaxFallbackRetries,
						fallback:   true,
					},
				)
		}
	}

	var lastErr error

	for _, modelEntry := range attempts {
		requestBody, err :=
			json.Marshal(
				struct {
					Model          string        `json:"model"`
					Messages       []ChatMessage `json:"messages"`
					MaxTokens      int           `json:"max_tokens"`
					Temperature    float64       `json:"temperature"`
					Stream         bool          `json:"stream"`
					EnableThinking *bool         `json:"enable_thinking,omitempty"`
				}{
					Model:          modelEntry.model,
					Messages:       messages,
					MaxTokens:      cfg.MaxTokens,
					Temperature:    cfg.Temperature,
					Stream:         true,
					EnableThinking: disableThinkingForModel(modelEntry.model),
				},
			)
		if err != nil {
			return nil,
				"",
				false,
				fmt.Errorf(
					"序列化AI流式请求失败: %w",
					err,
				)
		}

		for retry := 0; retry <=
			modelEntry.maxRetries; retry++ {
			if retry > 0 {
				delay :=
					getRetryDelay(
						retry - 1,
					)

				streamLog.Info(
					"流式重试等待",
					"model",
					modelEntry.model,
					"delay",
					delay,
				)

				if err :=
					waitAIStreamRetry(
						ctx,
						delay,
					); err != nil {
					return nil,
						"",
						false,
						err
				}
			}

			request, err :=
				http.NewRequestWithContext(
					ctx,
					http.MethodPost,
					endpoint,
					bytes.NewReader(
						requestBody,
					),
				)
			if err != nil {
				return nil,
					"",
					false,
					fmt.Errorf(
						"创建HTTP流式请求失败: %w",
						err,
					)
			}

			request.Header.Set(
				"Content-Type",
				"application/json",
			)
			request.Header.Set(
				"Authorization",
				"Bearer "+cfg.APIKey,
			)
			request.Header.Set(
				"Accept",
				"text/event-stream",
			)

			response, err :=
				(&http.Client{
					Timeout: AICallTimeout,
				}).Do(
					request,
				)
			if err != nil {
				lastErr =
					fmt.Errorf(
						"AI流式API调用失败（模型 %s）: %w",
						modelEntry.model,
						err,
					)
				continue
			}

			if response.StatusCode ==
				http.StatusOK {
				return response,
					modelEntry.model,
					modelEntry.fallback,
					nil
			}

			statusCode :=
				response.StatusCode

			body, _ :=
				io.ReadAll(
					response.Body,
				)

			_ =
				response.Body.Close()

			errorMessage :=
				extractErrorMessage(
					body,
				)

			lastErr =
				fmt.Errorf(
					"AI流式API返回错误（模型 %s, HTTP %d）: %s",
					modelEntry.model,
					statusCode,
					errorMessage,
				)

			if !isRetryableError(
				statusCode,
				body,
			) {
				return nil,
					"",
					false,
					lastErr
			}
		}
	}

	if lastErr == nil {
		lastErr =
			fmt.Errorf(
				"AI流式调用所有模型均失败",
			)
	}

	return nil,
		"",
		false,
		lastErr
}

// aiStreamReadResult 是已建立连接后的读取结果。
type aiStreamReadResult struct {
	Content          string
	ModelUsed        string
	TotalTokens      int
	PromptTokens     int
	CompletionTokens int
}

// readAIStreamResponse 在已建立连接后读取流。
// 此阶段禁止重试和Fallback；只有收到[DONE]或finish_reason才算完整成功。
func readAIStreamResponse(
	response *http.Response,
	onChunk func(string) error,
) (
	*aiStreamReadResult,
	error,
) {
	result :=
		&aiStreamReadResult{}

	if response == nil ||
		response.Body == nil {
		return result,
			fmt.Errorf(
				"AI流式响应为空",
			)
	}

	filter :=
		&aiStreamThinkingFilter{}

	var content strings.Builder
	completed := false

	scanner :=
		bufio.NewScanner(
			response.Body,
		)

	scanner.Buffer(
		make(
			[]byte,
			64*1024,
		),
		1024*1024,
	)

	for scanner.Scan() {
		line :=
			scanner.Text()

		if line == "" ||
			strings.HasPrefix(
				line,
				":",
			) ||
			!strings.HasPrefix(
				line,
				"data:",
			) {
			continue
		}

		data :=
			strings.TrimSpace(
				strings.TrimPrefix(
					line,
					"data:",
				),
			)

		if data == "" {
			continue
		}

		if data == "[DONE]" {
			completed = true
			break
		}

		var chunk StreamChunkResponse

		if err :=
			json.Unmarshal(
				[]byte(data),
				&chunk,
			); err != nil {
			result.Content =
				content.String()

			return result,
				fmt.Errorf(
					"解析AI流式响应块失败: %w",
					err,
				)
		}

		if result.ModelUsed == "" {
			result.ModelUsed =
				strings.TrimSpace(
					chunk.Model,
				)
		}

		if chunk.Usage != nil {
			result.TotalTokens =
				chunk.Usage.TotalTokens

			result.PromptTokens =
				chunk.Usage.PromptTokens

			result.CompletionTokens =
				chunk.Usage.CompletionTokens
		}

		if len(
			chunk.Choices,
		) == 0 {
			continue
		}

		visible :=
			filter.Push(
				chunk.Choices[0].
					Delta.
					Content,
			)

		if visible != "" {
			content.WriteString(
				visible,
			)

			if onChunk != nil {
				if err :=
					onChunk(
						visible,
					); err != nil {
					result.Content =
						content.String()

					return result,
						fmt.Errorf(
							"AI流式输出回调中止: %w",
							err,
						)
				}
			}
		}

		if chunk.Choices[0].
			FinishReason != nil {
			completed = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Content =
			content.String()

		return result,
			fmt.Errorf(
				"读取AI流式响应失败: %w",
				err,
			)
	}

	if !completed {
		result.Content =
			content.String()

		return result,
			fmt.Errorf(
				"AI流式响应在完成标记前中断",
			)
	}

	visible :=
		filter.Flush()

	if visible != "" {
		content.WriteString(
			visible,
		)

		if onChunk != nil {
			if err :=
				onChunk(
					visible,
				); err != nil {
				result.Content =
					content.String()

				return result,
					fmt.Errorf(
						"AI流式输出回调中止: %w",
						err,
					)
			}
		}
	}

	result.Content =
		content.String()

	return result, nil
}
