package ai

// client_stream_helpers.go
//
// 放置流式客户端的纯函数辅助，避免主调用文件超过600行。

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// validateAIStreamMessages 校验允许提交给文本AI的正式消息。
func validateAIStreamMessages(
	messages []ChatMessage,
) error {
	if len(messages) == 0 {
		return fmt.Errorf(
			"AI流式消息不能为空",
		)
	}

	for _, message := range messages {
		switch strings.TrimSpace(
			message.Role,
		) {
		case "system",
			"user",
			"assistant":

		default:
			return fmt.Errorf(
				"AI流式消息角色无效",
			)
		}

		if strings.TrimSpace(
			message.Content,
		) == "" {
			return fmt.Errorf(
				"AI流式消息内容不能为空",
			)
		}
	}

	return nil
}

// cloneAIStreamConfig 深复制Fallback切片，避免并发调用互相修改模型策略。
func cloneAIStreamConfig(
	cfg *EffectiveConfig,
) *EffectiveConfig {
	cloned := *cfg

	cloned.FallbackModels = append(
		[]string(nil),
		cfg.FallbackModels...,
	)

	return &cloned
}

// waitAIStreamRetry 让重试等待能够响应请求取消。
func waitAIStreamRetry(
	ctx context.Context,
	delay time.Duration,
) error {
	timer := time.NewTimer(
		delay,
	)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}

// normalizeAIStreamTokenUsage 规范化供应商返回的Token计量。
//
// 若供应商只返回total_tokens，则按输入和输出的字符估算权重拆分。
// 若完全没有usage，则按现有平台字符估算法生成可追溯计量。
func normalizeAIStreamTokenUsage(
	messages []ChatMessage,
	content string,
	promptTokens int,
	completionTokens int,
	totalTokens int,
) (
	int,
	int,
	int,
) {
	if promptTokens > 0 ||
		completionTokens > 0 {
		if totalTokens <
			promptTokens+
				completionTokens {
			totalTokens =
				promptTokens +
					completionTokens
		}

		return promptTokens,
			completionTokens,
			totalTokens
	}

	estimatedPrompt :=
		estimateAIStreamMessagesTokens(
			messages,
		)

	estimatedCompletion :=
		estimateAIStreamTextTokens(
			content,
		)

	if totalTokens <= 0 {
		return estimatedPrompt,
			estimatedCompletion,
			estimatedPrompt +
				estimatedCompletion
	}

	estimatedTotal :=
		estimatedPrompt +
			estimatedCompletion

	if estimatedTotal <= 0 {
		return totalTokens,
			0,
			totalTokens
	}

	promptShare :=
		float64(
			estimatedPrompt,
		) /
			float64(
				estimatedTotal,
			)

	promptTokens =
		int(
			math.Round(
				float64(
					totalTokens,
				) *
					promptShare,
			),
		)

	if totalTokens >= 2 {
		if promptTokens < 1 {
			promptTokens = 1
		}

		if promptTokens >=
			totalTokens {
			promptTokens =
				totalTokens - 1
		}
	}

	completionTokens =
		totalTokens -
			promptTokens

	return promptTokens,
		completionTokens,
		totalTokens
}

// estimateAIStreamMessagesTokens 估算完整输入消息Token。
func estimateAIStreamMessagesTokens(
	messages []ChatMessage,
) int {
	total := 0

	for _, message := range messages {
		total +=
			estimateAIStreamTextTokens(
				message.Content,
			) +
				4
	}

	if total < 1 {
		return 1
	}

	return total
}

// estimateAIStreamTextTokens 使用现有中文为主场景的保守估算。
func estimateAIStreamTextTokens(
	content string,
) int {
	runeCount :=
		utf8.RuneCountInString(
			content,
		)

	if runeCount == 0 {
		return 0
	}

	estimated :=
		int(
			math.Ceil(
				float64(
					runeCount,
				) *
					0.7,
			),
		)

	if estimated < 1 {
		return 1
	}

	return estimated
}

// aiStreamThinkingFilter 跨chunk过滤<thinking>隐藏块。
//
// 标签可能被供应商拆到多个SSE块中，因此不能只检查当前chunk。
type aiStreamThinkingFilter struct {
	inThinking bool
	pending    string
}

// Push 输入一个原始增量并返回允许展示的增量。
func (f *aiStreamThinkingFilter) Push(
	input string,
) string {
	const (
		startTag = "<thinking>"
		endTag   = "</thinking>"
	)

	f.pending += input

	var visible strings.Builder

	for {
		if f.inThinking {
			endIndex :=
				strings.Index(
					f.pending,
					endTag,
				)

			if endIndex < 0 {
				keep :=
					longestAIStreamTagPrefixSuffix(
						f.pending,
						endTag,
					)

				if keep == 0 {
					f.pending = ""
				} else {
					f.pending = f.pending[len(f.pending)-keep:]
				}

				return visible.String()
			}

			f.pending = f.pending[endIndex+len(endTag):]
			f.inThinking = false

			continue
		}

		startIndex :=
			strings.Index(
				f.pending,
				startTag,
			)

		if startIndex >= 0 {
			visible.WriteString(
				f.pending[:startIndex],
			)

			f.pending = f.pending[startIndex+len(startTag):]
			f.inThinking = true

			continue
		}

		keep :=
			longestAIStreamTagPrefixSuffix(
				f.pending,
				startTag,
			)

		if keep == 0 {
			visible.WriteString(
				f.pending,
			)

			f.pending = ""
		} else {
			visible.WriteString(
				f.pending[:len(f.pending)-keep],
			)

			f.pending = f.pending[len(f.pending)-keep:]
		}

		return visible.String()
	}
}

// Flush 在流结束时输出剩余可见内容。
// 未闭合隐藏块被安全丢弃。
func (f *aiStreamThinkingFilter) Flush() string {
	if f.inThinking {
		f.pending = ""
		return ""
	}

	visible :=
		f.pending

	f.pending = ""

	return visible
}

// longestAIStreamTagPrefixSuffix 返回value尾部与tag前缀重合长度。
func longestAIStreamTagPrefixSuffix(
	value string,
	tag string,
) int {
	maximum :=
		len(tag) - 1

	if len(value) <
		maximum {
		maximum =
			len(value)
	}

	for size := maximum; size > 0; size-- {
		if strings.HasSuffix(
			value,
			tag[:size],
		) {
			return size
		}
	}

	return 0
}
