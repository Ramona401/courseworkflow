package ai

// AI多模态调用客户端
// 负责图片+文字的多模态Vision调用（CallAIMultimodal），用于课本OCR识别等场景
//
// 本文件包含：
//   - 多模态类型定义（MultimodalContent、MultimodalImageURL、MultimodalMessage、MultimodalChatRequest）
//   - 多模态AI调用（CallAIMultimodal + 重试 + Fallback + 埋点）
//
// 依赖 client.go 中的：
//   - 类型：EffectiveConfig、TraceContext、ChatMessage、CallResult
//   - 常量：MaxRetries、MaxFallbackRetries
//   - 函数：callAIOnce、isRetryableCallError、getRetryDelay、emitTrace、getSceneFromTrace
//
// v71新增：CallAIMultimodal多模态Vision调用
// v85新增：主模型失败后依次尝试fallback模型
// v92重构：从原client.go拆分为独立文件

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ==================== 多模态类型定义 ====================

// MultimodalContent OpenAI兼容的多模态消息内容项
type MultimodalContent struct {
	Type     string              `json:"type"`                // "text" 或 "image_url"
	Text     string              `json:"text,omitempty"`      // type=text时的文本
	ImageURL *MultimodalImageURL `json:"image_url,omitempty"` // type=image_url时的图片
}

// MultimodalImageURL 图片URL结构
type MultimodalImageURL struct {
	URL    string `json:"url"`              // data:image/jpeg;base64,xxx 或 https://xxx
	Detail string `json:"detail,omitempty"` // "auto"/"low"/"high"，默认auto
}

// MultimodalMessage OpenAI兼容的多模态消息（content为数组）
type MultimodalMessage struct {
	Role    string              `json:"role"`
	Content []MultimodalContent `json:"content"`
}

// MultimodalChatRequest 多模态请求体
type MultimodalChatRequest struct {
	Model       string        `json:"model"`
	Messages    []interface{} `json:"messages"` // 混合普通和多模态消息
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// ==================== 多模态AI调用（带重试 + Fallback + 埋点）====================

// CallAIMultimodal 调用AI API发送图片+文字（多模态Vision调用）
// 用于课本OCR识别等需要图像理解的场景
// imageDataURI 格式：data:image/jpeg;base64,xxxxx
//
// v85新增：主模型失败后依次尝试fallback模型
func CallAIMultimodal(cfg *EffectiveConfig, systemPrompt string, userText string, imageDataURI string, traceCtx *TraceContext) (*CallResult, error) {
	// 构造消息列表
	var messages []interface{}

	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}

	userContent := []MultimodalContent{
		{Type: "text", Text: userText},
		{Type: "image_url", ImageURL: &MultimodalImageURL{URL: imageDataURI, Detail: "high"}},
	}
	messages = append(messages, MultimodalMessage{
		Role:    "user",
		Content: userContent,
	})

	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + "/chat/completions"
	startTime := time.Now()

	// -------- 构建模型尝试列表：主模型 + fallback模型 --------
	type modelAttempt struct {
		model      string
		isFallback bool
	}
	modelsToTry := []modelAttempt{
		{model: cfg.Model, isFallback: false},
	}
	for _, fbModel := range cfg.FallbackModels {
		if fbModel != cfg.Model {
			modelsToTry = append(modelsToTry, modelAttempt{
				model: fbModel, isFallback: true,
			})
		}
	}

	primaryModel := cfg.Model
	var lastErr error

	for _, modelEntry := range modelsToTry {
		maxRetries := MaxRetries
		if modelEntry.isFallback {
			maxRetries = MaxFallbackRetries
		}

		reqBody := MultimodalChatRequest{
			Model:       modelEntry.model,
			Messages:    messages,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化多模态AI请求失败: %w", err)
		}

		// 带重试调用
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				delay := getRetryDelay(attempt - 1)
				log.Printf("[AI多模态重试] 模型 %s 第%d次重试，等待%s", modelEntry.model, attempt, delay)
				time.Sleep(delay)
			}

			result, err := callAIOnce(cfg, endpoint, jsonBody)
			if err == nil {
				if attempt > 0 {
					log.Printf("[AI多模态重试] 模型 %s 第%d次重试成功", modelEntry.model, attempt)
				}
				if modelEntry.isFallback {
					log.Printf("[AI Fallback] 多模态降级模型 %s 调用成功（原始: %s）", modelEntry.model, primaryModel)
				}
				emitTrace(traceCtx, result.ModelUsed, result.TokensUsed, 0, 0,
					time.Since(startTime).Milliseconds(), "success", "", len(result.Content), false,
					modelEntry.isFallback, primaryModel)
				return result, nil
			}
			lastErr = err
			if !isRetryableCallError(err) {
				// 不可重试错误（如401），不继续fallback
				emitTrace(traceCtx, modelEntry.model, 0, 0, 0,
					time.Since(startTime).Milliseconds(), "error", err.Error(), 0, false,
					modelEntry.isFallback, primaryModel)
				return nil, err
			}
			log.Printf("[AI多模态重试] 模型 %s 失败（第%d/%d次）: %s",
				modelEntry.model, attempt+1, maxRetries+1, err.Error())
		}

		// 当前模型所有重试失败
		if modelEntry.isFallback {
			log.Printf("[AI Fallback] 多模态降级模型 %s 所有重试失败", modelEntry.model)
		} else {
			log.Printf("[AI Fallback] 多模态主模型 %s 所有重试失败，开始尝试降级", modelEntry.model)
		}
	}

	// 所有模型都失败
	emitTrace(traceCtx, primaryModel, 0, 0, 0,
		time.Since(startTime).Milliseconds(), "error", lastErr.Error(), 0, false, false, "")
	return nil, fmt.Errorf("AI多模态调用失败（主模型 %s + %d个降级模型均失败）: %w",
		primaryModel, len(cfg.FallbackModels), lastErr)
}
