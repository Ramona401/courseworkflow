package ai

// client_tts.go — 豆包(Volcengine) seed-tts-2.0 语音合成API客户端
//
// v0.42.9 新增：对接豆包 TTS API，为字幕条生成配音
//
// API 模式：同步调用（与图片生成类似），返回音频二进制流
//
// 请求格式（火山引擎 OpenAPI 兼容格式）：
//   POST {base_url}/audio/speech
//   Body: { model, input, voice, response_format, speed }
//   返回: 音频二进制流（mp3）
//
// 配置管理：
//   - 复用图片API的 image_api_base_url 和 image_api_key_enc（同一个豆包平台）
//   - tts_default_model: 默认TTS模型名（ai_configs表）
//   - ai_scene_configs.courseware_subtitle_tts 可覆盖模型
//
// 音色列表：
//   豆包 seed-tts-2.0 内置中文+英文音色，通过 voice 参数指定

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"os/exec"
	"strconv"

	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/utils"
)

// 模块日志
var ttsLog = logger.WithModule("ai_tts")

// ==================== 请求/响应结构体 ====================

// TTSRequest TTS语音合成请求体（豆包OpenAI兼容格式）
type TTSRequest struct {
	Model          string  `json:"model"`                     // 模型名
	Input          string  `json:"input"`                     // 要合成的文本
	Voice          string  `json:"voice"`                     // 音色代码
	ResponseFormat string  `json:"response_format,omitempty"` // 输出格式：mp3/wav/pcm，默认mp3
	Speed          float64 `json:"speed,omitempty"`           // 语速：0.25-4.0，默认1.0
}

// TTSResult TTS语音合成结果（业务层使用）
type TTSResult struct {
	AudioFilePath string  // 生成的音频文件本地路径
	AudioURL      string  // 音频文件的公网URL
	Duration      float64 // 音频时长（秒，由ffprobe获取）
	ModelUsed     string  // 使用的模型
	FileSize      int64   // 文件大小（字节）
}

// TTSConfig TTS语音合成API配置（从AI配置中心加载）
type TTSConfig struct {
	APIBaseURL string // API基地址（复用图片API）
	APIKey     string // 明文API Key（已解密，复用图片API）
	Model      string // TTS模型名
}

// ==================== 音色定义 ====================

// TTSVoice 单个音色定义
type TTSVoice struct {
	Code     string `json:"code"`     // 音色代码（传给API的voice参数）
	Name     string `json:"name"`     // 音色名称（中文展示）
	Language string `json:"language"` // 适用语言：zh-CN / en-US / multi
	Gender   string `json:"gender"`   // 性别：female / male
	Style    string `json:"style"`    // 风格描述
}

// AvailableTTSVoices 可用音色列表
// 豆包 seed-tts-2.0 内置音色（精选常用的）
var AvailableTTSVoices = []TTSVoice{
	// 中文女声
	{Code: "zh_female_shuangkuai", Name: "爽快女声", Language: "zh-CN", Gender: "female", Style: "活泼清晰"},
	{Code: "zh_female_wennuan", Name: "温暖女声", Language: "zh-CN", Gender: "female", Style: "温柔亲切"},
	{Code: "zh_female_tianmei", Name: "甜美女声", Language: "zh-CN", Gender: "female", Style: "甜美可爱"},
	{Code: "zh_female_qingche", Name: "清澈女声", Language: "zh-CN", Gender: "female", Style: "清澈干净"},
	// 中文男声
	{Code: "zh_male_chunhou", Name: "醇厚男声", Language: "zh-CN", Gender: "male", Style: "沉稳大气"},
	{Code: "zh_male_yangguang", Name: "阳光男声", Language: "zh-CN", Gender: "male", Style: "阳光积极"},
	{Code: "zh_male_qinqie", Name: "亲切男声", Language: "zh-CN", Gender: "male", Style: "亲切自然"},
	// 英文女声
	{Code: "en_female_sarah", Name: "Sarah", Language: "en-US", Gender: "female", Style: "Professional"},
	{Code: "en_female_emily", Name: "Emily", Language: "en-US", Gender: "female", Style: "Friendly"},
	// 英文男声
	{Code: "en_male_ryan", Name: "Ryan", Language: "en-US", Gender: "male", Style: "Clear"},
	{Code: "en_male_adam", Name: "Adam", Language: "en-US", Gender: "male", Style: "Deep"},
	// 多语言（中英双语）
	{Code: "multi_female_shuangyu", Name: "双语女声", Language: "multi", Gender: "female", Style: "中英流畅切换"},
	{Code: "multi_male_shuangyu", Name: "双语男声", Language: "multi", Gender: "male", Style: "中英流畅切换"},
}

// GetTTSVoicesByLanguage 按语言筛选可用音色
func GetTTSVoicesByLanguage(language string) []TTSVoice {
	var result []TTSVoice
	for _, v := range AvailableTTSVoices {
		// multi 音色对所有语言都可用
		if v.Language == language || v.Language == "multi" {
			result = append(result, v)
		}
	}
	// 如果没有匹配，返回全部
	if len(result) == 0 {
		return AvailableTTSVoices
	}
	return result
}

// ==================== TTS 合成核心函数 ====================

// SynthesizeSpeech 调用豆包TTS API合成语音
// 参数：
//   - cfg: TTS API配置
//   - text: 要合成的文本
//   - voice: 音色代码
//   - speed: 语速（0则默认1.0）
//   - outputDir: 输出文件目录
//   - outputName: 输出文件名（不含扩展名，自动加.mp3）
//   - traceCtx: 追踪上下文（可为nil）
func SynthesizeSpeech(ctx context.Context, cfg *TTSConfig, text string, voice string, speed float64, outputDir string, outputName string, traceCtx *TraceContext) (*TTSResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("TTS配置为空")
	}
	if cfg.APIBaseURL == "" || cfg.APIKey == "" {
		return nil, fmt.Errorf("TTS API未配置（请在AI管理中心配置图片/视频生成API地址和密钥）")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("TTS模型未配置")
	}
	if text == "" {
		return nil, fmt.Errorf("合成文本不能为空")
	}
	if voice == "" {
		voice = "zh_female_shuangkuai" // 默认中文爽快女声
	}
	if speed <= 0 {
		speed = 1.0
	}

	// 构建API URL
	apiURL := strings.TrimRight(cfg.APIBaseURL, "/") + "/audio/speech"

	// 构建请求体
	reqBody := TTSRequest{
		Model:          cfg.Model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: "mp3",
		Speed:          speed,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化TTS请求失败: %w", err)
	}

	ttsLog.Info("调用TTS语音合成API",
		"url", apiURL,
		"model", cfg.Model,
		"text_len", len(text),
		"voice", voice,
		"speed", speed,
	)

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("创建TTS HTTP请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// 发送请求（30秒超时，TTS通常很快）
	client := &http.Client{Timeout: 30 * time.Second}
	startTime := time.Now()
	httpResp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		ttsLog.Error("TTS HTTP请求失败", "error", err, "latency_ms", latencyMs)
		return nil, fmt.Errorf("TTS请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	// 检查HTTP状态码
	if httpResp.StatusCode != http.StatusOK {
		// 错误响应是JSON格式
		respBody, _ := io.ReadAll(httpResp.Body)
		ttsLog.Error("TTS API返回错误",
			"status", httpResp.StatusCode,
			"body", truncateStr(string(respBody), 500),
			"latency_ms", latencyMs,
		)
		return nil, fmt.Errorf("TTS API返回错误(HTTP %d): %s", httpResp.StatusCode, truncateStr(string(respBody), 200))
	}

	// 成功时返回的是音频二进制流，直接写入文件
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	outputPath := filepath.Join(outputDir, outputName+".mp3")
	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("创建音频文件失败: %w", err)
	}

	written, err := io.Copy(outFile, httpResp.Body)
	outFile.Close()
	if err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("写入音频文件失败: %w", err)
	}

	if written < 100 {
		// 文件太小，可能是空响应或错误
		content, _ := os.ReadFile(outputPath)
		os.Remove(outputPath)
		return nil, fmt.Errorf("TTS生成的音频文件异常小(%d字节): %s", written, truncateStr(string(content), 200))
	}

	// 获取音频时长（通过ffprobe）
	duration := getAudioDuration(outputPath)

	ttsLog.Info("TTS合成成功",
		"model", cfg.Model,
		"voice", voice,
		"text_len", len(text),
		"file_size", written,
		"duration", duration,
		"latency_ms", latencyMs,
	)

	// 写入追踪记录
	if traceCtx != nil {
		go func() {
			// TTS按字符数粗估tokens（每个字符约1 token）
			estimatedTokens := len([]rune(text))
			emitTrace(
				traceCtx, cfg.Model,
				estimatedTokens, 0, estimatedTokens,
				latencyMs, "success", "",
				int(written),
				false, false, "",
			)
		}()
	}

	return &TTSResult{
		AudioFilePath: outputPath,
		AudioURL:      "", // 由调用方拼接公网URL
		Duration:      duration,
		ModelUsed:     cfg.Model,
		FileSize:      written,
	}, nil
}

// ==================== 音频时长获取 ====================

// getAudioDuration 通过ffprobe获取音频文件时长（秒）
// 在 ai 包内独立实现，不依赖 services 包的 getVideoDuration
func getAudioDuration(filePath string) float64 {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		ttsLog.Warn("ffprobe获取音频时长失败", "file", filePath, "error", err)
		return 0
	}
	// 解析 JSON 中的 format.duration 字段
	var probe struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &probe); err != nil {
		ttsLog.Warn("ffprobe输出解析失败", "file", filePath, "error", err)
		return 0
	}
	dur, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return 0
	}
	return dur
}

// ==================== 配置加载 ====================

// GetTTSConfig 从AI配置中心加载TTS API配置
// 复用图片API的 base_url 和 api_key，单独读取 tts_default_model
func GetTTSConfig(aesKey string) (*TTSConfig, error) {
	ctx := context.Background()
	cfg := &TTSConfig{}

	// 复用图片API的基地址
	var baseURL string
	err := database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_base_url'`).Scan(&baseURL)
	if err != nil {
		return nil, fmt.Errorf("TTS API地址未配置（需要先配置图片API地址）: %w", err)
	}
	cfg.APIBaseURL = baseURL

	// 复用图片API的密钥
	var encryptedKey string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'image_api_key_enc'`).Scan(&encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("TTS API密钥未配置（需要先配置图片API密钥）: %w", err)
	}
	decrypted, err := utils.DecryptAES(encryptedKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("TTS API密钥解密失败: %w", err)
	}
	cfg.APIKey = decrypted

	// 读取TTS专用模型名
	var model string
	err = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'tts_default_model'`).Scan(&model)
	if err != nil {
		// 兜底默认模型
		model = "doubao-seed-tts-2.0"
		ttsLog.Info("TTS模型未配置，使用默认值", "model", model)
	}
	cfg.Model = model

	// 场景配置可覆盖模型（ai_scene_configs.courseware_subtitle_tts）
	var sceneModel *string
	_ = database.DB.QueryRow(ctx,
		`SELECT model FROM ai_scene_configs WHERE scene_code = 'courseware_subtitle_tts' AND is_active = true`).Scan(&sceneModel)
	if sceneModel != nil && *sceneModel != "" {
		cfg.Model = *sceneModel
	}

	return cfg, nil
}
