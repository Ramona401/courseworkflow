package ai

// client_image.go — 豆包(Volcengine) Seedream 图片生成API客户端
//
// v0.42 多媒体：对接豆包 doubao-seedream API（含图生图/参考图）
//
// 配置管理：通过 ai_configs 表独立管理（与文本AI分开）
//   - image_api_base_url: 图片API基地址
//   - image_api_key_enc: AES加密的API Key
//   - image_default_model: 默认模型名
//
// 图生图：在请求体中传 image 字段（公网可访问的图片URL），豆包会参考该图生成

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/utils"
)

// 模块日志
var imageLog = logger.WithModule("ai_image")

// ==================== 请求/响应结构体 ====================

// ImageGenerateRequest 图片生成请求（豆包OpenAI兼容格式）
type ImageGenerateRequest struct {
	Model  string `json:"model"`            // 模型名
	Prompt string `json:"prompt"`           // 生成提示词
	Size   string `json:"size,omitempty"`   // 图片尺寸，默认 1920x1920
	N      int    `json:"n,omitempty"`      // 生成数量，默认 1
	Image  string `json:"image,omitempty"`  // 参考图URL（图生图模式，公网可访问）
}

// ImageGenerateResponse 图片生成响应
type ImageGenerateResponse struct {
	Created int64               `json:"created"`
	Data    []ImageGenerateItem `json:"data"`
}

// ImageGenerateItem 单张生成图片
type ImageGenerateItem struct {
	URL           string `json:"url"`            // 图片URL（临时链接，需下载保存）
	RevisedPrompt string `json:"revised_prompt"` // 模型修改后的提示词（可选）
	B64JSON       string `json:"b64_json"`       // Base64格式（当response_format=b64_json时）
}

// ImageGenerateResult 图片生成结果（业务层使用）
type ImageGenerateResult struct {
	URLs          []string // 生成的图片URL列表
	ModelUsed     string   // 使用的模型
	RevisedPrompt string   // 修改后的提示词
}

// ImageConfig 图片生成API配置（从AI配置中心加载）
type ImageConfig struct {
	APIBaseURL string // API基地址
	APIKey     string // 明文API Key（已解密）
	Model      string // 模型名
}

// ==================== 图片生成核心函数 ====================

// GenerateImage 调用豆包API生成图片（支持图生图）
// 参数：
//   - cfg: 图片API配置
//   - prompt: 生成提示词
//   - size: 图片尺寸（空则默认1920x1920，总像素需≥3686400）
//   - n: 生成数量（0或1则生成1张，最多4张）
//   - refImageURL: 参考图URL（空则纯文生图，非空则图生图）
//   - traceCtx: 追踪上下文（可为nil）
func GenerateImage(ctx context.Context, cfg *ImageConfig, prompt string, size string, n int, refImageURL string, traceCtx *TraceContext) (*ImageGenerateResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("图片生成配置为空")
	}
	if cfg.APIBaseURL == "" || cfg.APIKey == "" {
		return nil, fmt.Errorf("图片生成API未配置（请在AI管理中心配置图片生成API地址和密钥）")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("图片生成模型未配置")
	}
	if prompt == "" {
		return nil, fmt.Errorf("图片生成提示词不能为空")
	}

	// 默认参数
	if size == "" {
		size = "1920x1920"
	}
	if n <= 0 {
		n = 1
	}
	if n > 4 {
		n = 4
	}

	// 构建API URL
	apiURL := strings.TrimRight(cfg.APIBaseURL, "/") + "/images/generations"

	// 构建请求体
	reqBody := ImageGenerateRequest{
		Model:  cfg.Model,
		Prompt: prompt,
		Size:   size,
		N:      n,
	}
	// 图生图：传入参考图URL
	if refImageURL != "" {
		reqBody.Image = refImageURL
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	imageLog.Info("调用图片生成API",
		"url", apiURL,
		"model", cfg.Model,
		"prompt_len", len(prompt),
		"size", size,
		"n", n,
		"has_ref_image", refImageURL != "",
	)

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// 发送请求（60秒超时）
	client := &http.Client{Timeout: 60 * time.Second}
	startTime := time.Now()
	httpResp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		imageLog.Error("图片生成HTTP请求失败", "error", err, "latency_ms", latencyMs)
		return nil, fmt.Errorf("图片生成请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		imageLog.Error("图片生成API返回错误",
			"status", httpResp.StatusCode,
			"body", truncateStr(string(respBody), 500),
			"latency_ms", latencyMs,
		)
		return nil, fmt.Errorf("图片生成API返回错误(HTTP %d): %s", httpResp.StatusCode, truncateStr(string(respBody), 200))
	}

	// 解析响应
	var apiResp ImageGenerateResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析图片生成响应失败: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("图片生成API未返回任何图片")
	}

	// 提取URL列表
	var urls []string
	var revisedPrompt string
	for _, item := range apiResp.Data {
		if item.URL != "" {
			urls = append(urls, item.URL)
		}
		if item.RevisedPrompt != "" && revisedPrompt == "" {
			revisedPrompt = item.RevisedPrompt
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("图片生成API返回的数据中没有有效URL")
	}

	imageLog.Info("图片生成成功",
		"model", cfg.Model,
		"count", len(urls),
		"latency_ms", latencyMs,
		"has_ref_image", refImageURL != "",
	)

	// 写入追踪记录
	if traceCtx != nil {
		go func() {
			emitTrace(
				traceCtx,  cfg.Model,
				0, 0, 0,   // tokens（图片生成不计token）
				latencyMs, "success", "",
				len(urls), // outputLength
				false, false, "",
			)
		}()
	}

	return &ImageGenerateResult{
		URLs:          urls,
		ModelUsed:     cfg.Model,
		RevisedPrompt: revisedPrompt,
	}, nil
}

// ==================== 配置加载 ====================

// GetImageConfig 从AI配置中心加载图片生成API的独立配置
func GetImageConfig(aesKey string) (*ImageConfig, error) {
	ctx := context.Background()
	cfg := &ImageConfig{}

	var baseURL string
	err := database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_base_url'`).Scan(&baseURL)
	if err != nil {
		return nil, fmt.Errorf("图片生成API地址未配置: %w", err)
	}
	cfg.APIBaseURL = baseURL

	var encryptedKey string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_key_enc'`).Scan(&encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("图片生成API密钥未配置: %w", err)
	}
	decrypted, err := utils.DecryptAES(encryptedKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("图片生成API密钥解密失败: %w", err)
	}
	cfg.APIKey = decrypted

	var model string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_default_model'`).Scan(&model)
	if err != nil {
		return nil, fmt.Errorf("图片生成模型未配置: %w", err)
	}
	cfg.Model = model

	var sceneModel *string
	_ = database.DB.QueryRow(ctx,
		`SELECT model FROM ai_scene_configs WHERE scene_code = 'courseware_image_gen' AND is_active = true`).Scan(&sceneModel)
	if sceneModel != nil && *sceneModel != "" {
		cfg.Model = *sceneModel
	}

	return cfg, nil
}

// truncateStr 截断字符串到指定长度
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
