package ai

// client_video.go — 豆包(Volcengine) Seedance 视频生成API客户端
//
// v0.42.1 多媒体：对接豆包 doubao-seedance 视频生成API
//
// 视频生成是异步模式（与图片生成的同步模式不同）：
//   1. 提交任务 POST /contents/generations/tasks → 返回 task_id
//   2. 轮询状态 GET  /contents/generations/tasks/{id} → 返回 status + video_url
//
// 支持两种模式：
//   - 文生视频：仅传 text prompt
//   - 图生视频：传 image_url + text prompt（参考图必须为公网可访问URL）
//
// 配置管理：
//   - video_default_model: 默认视频模型名（ai_configs表）
//   - 复用图片API的 image_api_base_url 和 image_api_key_enc（同一个豆包平台）
//   - ai_scene_configs.courseware_video_gen 可覆盖模型

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
var videoLog = logger.WithModule("ai_video")

// ==================== 请求/响应结构体 ====================

// VideoGenerateRequest 视频生成请求体（豆包 contents/generations/tasks 格式）
type VideoGenerateRequest struct {
	Model   string              `json:"model"`             // 模型名
	Content []VideoContentBlock `json:"content"`           // 内容块数组（文本+可选图片）
}

// VideoContentBlock 视频生成内容块（文本或图片）
type VideoContentBlock struct {
	Type     string         `json:"type"`                // "text" 或 "image_url"
	Text     string         `json:"text,omitempty"`      // 文本描述（type=text时使用）
	ImageURL *VideoImageURL `json:"image_url,omitempty"` // 图片URL（type=image_url时使用）
}

// VideoImageURL 视频生成参考图URL
type VideoImageURL struct {
	URL string `json:"url"` // 公网可访问的图片URL
}

// VideoTaskResponse 视频生成任务提交响应（仅含task_id）
type VideoTaskResponse struct {
	ID string `json:"id"` // 任务ID，如 "cgt-20260523173444-llfp9"
}

// VideoTaskStatusResponse 视频任务状态查询完整响应
type VideoTaskStatusResponse struct {
	ID        string           `json:"id"`         // 任务ID
	Model     string           `json:"model"`      // 使用的模型
	Status    string           `json:"status"`     // 任务状态：running/succeeded/failed
	Content   *VideoContent    `json:"content"`    // 视频内容（succeeded时有值）
	Usage     *VideoUsage      `json:"usage"`      // Token用量
	Error     *VideoError      `json:"error"`      // 错误信息（failed时有值）
	CreatedAt int64            `json:"created_at"` // 创建时间（Unix秒）
	UpdatedAt int64            `json:"updated_at"` // 更新时间（Unix秒）
	Seed      int64            `json:"seed"`       // 随机种子
	Resolution string          `json:"resolution"` // 分辨率，如 "720p"
	Ratio     string           `json:"ratio"`      // 画面比例，如 "16:9"
	Duration  int              `json:"duration"`   // 视频时长（秒）
	FPS       int              `json:"framespersecond"` // 帧率
}

// VideoContent 视频内容（任务成功后的结果）
type VideoContent struct {
	VideoURL string `json:"video_url"` // 视频下载URL（临时链接，24小时有效）
}

// VideoUsage 视频生成Token用量
type VideoUsage struct {
	CompletionTokens int `json:"completion_tokens"` // 生成消耗tokens
	TotalTokens      int `json:"total_tokens"`      // 总tokens
}

// VideoError 视频生成错误信息
type VideoError struct {
	Code    string `json:"code"`    // 错误码
	Message string `json:"message"` // 错误描述
}

// ==================== 业务层使用的结果结构 ====================

// VideoSubmitResult 视频任务提交结果
type VideoSubmitResult struct {
	TaskID    string // 豆包返回的任务ID
	ModelUsed string // 使用的模型
}

// VideoQueryResult 视频任务查询结果（业务层使用）
type VideoQueryResult struct {
	TaskID     string // 任务ID
	Status     string // 状态：running/succeeded/failed
	VideoURL   string // 视频下载URL（succeeded时有值）
	Duration   int    // 视频时长（秒）
	Resolution string // 分辨率
	Ratio      string // 画面比例
	FPS        int    // 帧率
	ErrorMsg   string // 错误信息（failed时有值）
	TotalTokens int   // 消耗的tokens
}

// VideoConfig 视频生成API配置（从AI配置中心加载）
type VideoConfig struct {
	APIBaseURL string // API基地址（复用图片API）
	APIKey     string // 明文API Key（已解密，复用图片API）
	Model      string // 视频模型名
}

// ==================== 视频任务提交 ====================

// SubmitVideoTask 提交视频生成任务（异步，返回task_id）
// 参数：
//   - cfg: 视频API配置
//   - prompt: 视频描述提示词
//   - refImageURL: 参考图URL（空则纯文生视频，非空则图生视频）
//   - traceCtx: 追踪上下文（可为nil）
func SubmitVideoTask(ctx context.Context, cfg *VideoConfig, prompt string, refImageURL string, traceCtx *TraceContext) (*VideoSubmitResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("视频生成配置为空")
	}
	if cfg.APIBaseURL == "" || cfg.APIKey == "" {
		return nil, fmt.Errorf("视频生成API未配置（请在AI管理中心配置图片/视频生成API地址和密钥）")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("视频生成模型未配置")
	}
	if prompt == "" {
		return nil, fmt.Errorf("视频生成提示词不能为空")
	}

	// 构建API URL
	apiURL := strings.TrimRight(cfg.APIBaseURL, "/") + "/contents/generations/tasks"

	// 构建请求体：content数组
	var contentBlocks []VideoContentBlock

	// 图生视频：先放图片块，再放文本块
	if refImageURL != "" {
		contentBlocks = append(contentBlocks, VideoContentBlock{
			Type:     "image_url",
			ImageURL: &VideoImageURL{URL: refImageURL},
		})
	}

	// 文本描述块（必须有）
	contentBlocks = append(contentBlocks, VideoContentBlock{
		Type: "text",
		Text: prompt,
	})

	reqBody := VideoGenerateRequest{
		Model:   cfg.Model,
		Content: contentBlocks,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化视频请求失败: %w", err)
	}

	videoLog.Info("提交视频生成任务",
		"url", apiURL,
		"model", cfg.Model,
		"prompt_len", len(prompt),
		"has_ref_image", refImageURL != "",
	)

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// 发送请求（30秒超时，仅提交任务不需要很长）
	client := &http.Client{Timeout: 30 * time.Second}
	startTime := time.Now()
	httpResp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		videoLog.Error("视频任务提交HTTP请求失败", "error", err, "latency_ms", latencyMs)
		return nil, fmt.Errorf("视频任务提交失败: %w", err)
	}
	defer httpResp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取视频任务响应失败: %w", err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		videoLog.Error("视频任务提交API返回错误",
			"status", httpResp.StatusCode,
			"body", truncateStr(string(respBody), 500),
			"latency_ms", latencyMs,
		)
		return nil, fmt.Errorf("视频任务提交失败(HTTP %d): %s", httpResp.StatusCode, truncateStr(string(respBody), 200))
	}

	// 解析响应：成功时返回 {"id": "cgt-xxx"}
	var taskResp VideoTaskResponse
	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return nil, fmt.Errorf("解析视频任务响应失败: %w", err)
	}

	if taskResp.ID == "" {
		return nil, fmt.Errorf("视频任务提交成功但未返回任务ID")
	}

	videoLog.Info("视频任务提交成功",
		"task_id", taskResp.ID,
		"model", cfg.Model,
		"latency_ms", latencyMs,
	)

	// 写入追踪记录（提交阶段，tokens后续查询时补充）
	if traceCtx != nil {
		go func() {
			emitTrace(
				traceCtx, cfg.Model,
				0, 0, 0,
				latencyMs, "success", "",
				0, false, false, "",
			)
		}()
	}

	return &VideoSubmitResult{
		TaskID:    taskResp.ID,
		ModelUsed: cfg.Model,
	}, nil
}

// ==================== 视频任务状态查询 ====================

// QueryVideoTask 查询视频生成任务状态
// 返回状态：running（生成中）/ succeeded（成功）/ failed（失败）
func QueryVideoTask(ctx context.Context, cfg *VideoConfig, taskID string) (*VideoQueryResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("视频生成配置为空")
	}
	if taskID == "" {
		return nil, fmt.Errorf("任务ID不能为空")
	}

	// 构建查询URL
	apiURL := strings.TrimRight(cfg.APIBaseURL, "/") + "/contents/generations/tasks/" + taskID

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建查询请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// 发送请求
	client := &http.Client{Timeout: 15 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("查询视频任务失败: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取查询响应失败: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询视频任务失败(HTTP %d): %s", httpResp.StatusCode, truncateStr(string(respBody), 200))
	}

	// 解析完整状态响应
	var statusResp VideoTaskStatusResponse
	if err := json.Unmarshal(respBody, &statusResp); err != nil {
		return nil, fmt.Errorf("解析视频任务状态失败: %w", err)
	}

	result := &VideoQueryResult{
		TaskID:     statusResp.ID,
		Status:     statusResp.Status,
		Duration:   statusResp.Duration,
		Resolution: statusResp.Resolution,
		Ratio:      statusResp.Ratio,
		FPS:        statusResp.FPS,
	}

	// 成功：提取视频URL
	if statusResp.Status == "succeeded" && statusResp.Content != nil {
		result.VideoURL = statusResp.Content.VideoURL
	}

	// 失败：提取错误信息
	if statusResp.Status == "failed" && statusResp.Error != nil {
		result.ErrorMsg = statusResp.Error.Message
	}

	// Token用量
	if statusResp.Usage != nil {
		result.TotalTokens = statusResp.Usage.TotalTokens
	}

	return result, nil
}

// ==================== 配置加载 ====================

// GetVideoConfig 从AI配置中心加载视频生成API配置
// 复用图片API的 base_url 和 api_key，单独读取 video_default_model
func GetVideoConfig(aesKey string) (*VideoConfig, error) {
	ctx := context.Background()
	cfg := &VideoConfig{}

	// 复用图片API的基地址
	var baseURL string
	err := database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_base_url'`).Scan(&baseURL)
	if err != nil {
		return nil, fmt.Errorf("视频生成API地址未配置（需要先配置图片API地址）: %w", err)
	}
	cfg.APIBaseURL = baseURL

	// 复用图片API的密钥
	var encryptedKey string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_key_enc'`).Scan(&encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("视频生成API密钥未配置（需要先配置图片API密钥）: %w", err)
	}
	decrypted, err := utils.DecryptAES(encryptedKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("视频生成API密钥解密失败: %w", err)
	}
	cfg.APIKey = decrypted

	// 读取视频专用模型名
	var model string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'video_default_model'`).Scan(&model)
	if err != nil {
		return nil, fmt.Errorf("视频生成模型未配置: %w", err)
	}
	cfg.Model = model

	// 场景配置可覆盖模型（ai_scene_configs.courseware_video_gen）
	var sceneModel *string
	_ = database.DB.QueryRow(ctx,
		`SELECT model FROM ai_scene_configs WHERE scene_code = 'courseware_video_gen' AND is_active = true`).Scan(&sceneModel)
	if sceneModel != nil && *sceneModel != "" {
		cfg.Model = *sceneModel
	}

	return cfg, nil
}
