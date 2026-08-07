package ai

// client_tts.go — TTS语音合成客户端（多provider分流路由层）
//
// S-V1.5 重构（迭代3.5子专项S）：
//   - 引入 tts_provider 配置分流：volcano_v3（火山豆包语音v3，默认）/ volcano_openai（旧OpenAI兼容，已知不可用，保留回退）
//   - 真相修正：豆包语音合成2.0不走火山方舟，走 openspeech.bytedance.com 的 v3 接口，
//     鉴权为 APP ID + Access Token（X-Api-App-Id / X-Api-Access-Key 请求头），
//     旧实现调用的 {方舟base_url}/audio/speech 端点不存在，这是历史404的真正根因。
//   - 音色表替换为账号实际开通的豆包2.0真实音色（_uranus_ / saturn_ 系，10个）。
//   - v3具体实现在 client_tts_v3.go；本文件保留路由、音色表、配置加载、旧OpenAI实现。
//   - 将来接入阿里云百炼TTS：加 provider 常量 + 新实现文件 + GetTTSConfig 分支即可。
//
// 配置键（ai_configs表，经 /api/v1/admin/tts-config 维护）：
//   - tts_provider          : volcano_v3 / volcano_openai（缺省按 volcano_v3）
//   - tts_app_id            : 火山豆包语音应用的 APP ID（明文）
//   - tts_access_token_enc  : 火山 Access Token（AES加密存储）
//   - tts_v3_base_url       : 可选，v3接口基地址覆盖（缺省 https://openspeech.bytedance.com）
//   旧OpenAI路径仍读: image_api_base_url / image_api_key_enc / tts_default_model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// ==================== Provider 常量 ====================

const (
	// TTSProviderVolcanoV3 火山豆包语音 v3 接口（openspeech.bytedance.com，APP ID + Access Token 鉴权）
	TTSProviderVolcanoV3 = "volcano_v3"
	// TTSProviderOpenAI 旧 OpenAI 兼容接口路径（已知不可用，仅作历史回退保留）
	TTSProviderOpenAI = "volcano_openai"
	// ttsV3DefaultBaseURL v3 接口默认基地址
	ttsV3DefaultBaseURL = "https://openspeech.bytedance.com"
)

// ==================== 请求/响应结构体 ====================

// TTSRequest TTS语音合成请求体（旧OpenAI兼容格式，仅 volcano_openai 路径使用）
type TTSRequest struct {
	Model          string  `json:"model"`                     // 模型名
	Input          string  `json:"input"`                     // 要合成的文本
	Voice          string  `json:"voice"`                     // 音色代码
	ResponseFormat string  `json:"response_format,omitempty"` // 输出格式：mp3/wav/pcm，默认mp3
	Speed          float64 `json:"speed,omitempty"`           // 语速：0.25-4.0，默认1.0
}

// TTSResult TTS语音合成结果（业务层使用）
type TTSResult struct {
	AudioFilePath string
	AudioURL      string
	Duration      float64
	ModelUsed     string
	FileSize      int64
}

// TTSSynthesisError 描述供应商调用失败后的成本事实。
//
// Uncertain表示请求可能已经到达供应商，但无法确认是否成功；
// ProviderSucceeded表示供应商已经明确成功，只是本地文件处理失败。
type TTSSynthesisError struct {
	Cause             error
	Uncertain         bool
	ProviderSucceeded bool
}

func (err *TTSSynthesisError) Error() string {
	if err == nil ||
		err.Cause == nil {
		return "TTS合成失败"
	}

	return err.Cause.Error()
}

func (err *TTSSynthesisError) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Cause
}

// IsTTSSynthesisUncertain 判断是否禁止自动释放和重复调用供应商。
func IsTTSSynthesisUncertain(
	err error,
) bool {
	var synthesisErr *TTSSynthesisError

	return errors.As(
		err,
		&synthesisErr,
	) &&
		synthesisErr.Uncertain
}

// DidTTSSynthesisProviderSucceed 判断供应商是否已经明确成功。
func DidTTSSynthesisProviderSucceed(
	err error,
) bool {
	var synthesisErr *TTSSynthesisError

	return errors.As(
		err,
		&synthesisErr,
	) &&
		synthesisErr.ProviderSucceeded
}

// TTSConfig TTS语音合成API配置（从AI配置中心加载）
type TTSConfig struct {
	Provider    string // provider标识：volcano_v3 / volcano_openai
	APIBaseURL  string // API基地址（v3为openspeech域名；openai路径复用图片API）
	APIKey      string // 明文API Key（仅openai路径使用）
	Model       string // 模型/资源标识（v3路径仅作展示与追踪标签）
	AppID       string // 火山豆包语音 APP ID（仅v3路径）
	AccessToken string // 火山 Access Token 明文（仅v3路径，已解密）
}

// ==================== 音色定义 ====================

// TTSVoice 单个音色定义
type TTSVoice struct {
	Code     string `json:"code"`     // 音色代码（传给API的speaker/voice参数）
	Name     string `json:"name"`     // 音色名称（中文展示）
	Language string `json:"language"` // 适用语言：zh-CN / en-US / multi
	Gender   string `json:"gender"`   // 性别：female / male
	Style    string `json:"style"`    // 风格描述
}

// AvailableTTSVoices 可用音色列表
// S-V1.5：替换为账号实际开通的豆包语音合成2.0真实音色（来自火山控制台音色详情页）。
// 命名规律：*_uranus_bigtts 为通用2.0音色，saturn_* 为角色扮演2.0音色，
// 两类对应 X-Api-Resource-Id 均为 seed-tts-2.0（由 client_tts_v3.go 自动推导）。
// 账号共99个音色，此处精选K12课件配音常用10个；新增音色只需在此加一行。
var AvailableTTSVoices = []TTSVoice{
	// 中文女声·通用
	{Code: "zh_female_vv_uranus_bigtts", Name: "vivi 2.0", Language: "zh-CN", Gender: "female", Style: "通用场景·自然清晰"},
	{Code: "zh_female_xiaohe_uranus_bigtts", Name: "小何", Language: "zh-CN", Gender: "female", Style: "通用场景·亲切讲解"},
	// 中文男声·通用
	{Code: "zh_male_m191_uranus_bigtts", Name: "云舟", Language: "zh-CN", Gender: "male", Style: "通用场景·沉稳叙述"},
	{Code: "zh_male_taocheng_uranus_bigtts", Name: "小天", Language: "zh-CN", Gender: "male", Style: "通用场景·阳光自然"},
	// 中文女声·角色扮演（适合低学段课件）
	{Code: "saturn_zh_female_cancan_tob", Name: "知性灿灿", Language: "zh-CN", Gender: "female", Style: "角色扮演·知性教师感"},
	{Code: "saturn_zh_female_keainvsheng_tob", Name: "可爱女生", Language: "zh-CN", Gender: "female", Style: "角色扮演·活泼可爱"},
	{Code: "saturn_zh_female_tiaopigongzhu_tob", Name: "调皮公主", Language: "zh-CN", Gender: "female", Style: "角色扮演·俏皮童趣"},
	// 中文男声·角色扮演
	{Code: "saturn_zh_male_shuanglangshaonian_tob", Name: "爽朗少年", Language: "zh-CN", Gender: "male", Style: "角色扮演·少年朝气"},
	{Code: "saturn_zh_male_tiancaitongzhuo_tob", Name: "天才同桌", Language: "zh-CN", Gender: "male", Style: "角色扮演·聪明同伴"},
	// 英文男声
	{Code: "en_male_tim_uranus_bigtts", Name: "Tim", Language: "en-US", Gender: "male", Style: "General · Clear"},
}

// ttsDefaultVoice 默认音色（未指定时使用）
const ttsDefaultVoice = "zh_female_vv_uranus_bigtts"

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

// ==================== TTS 合成入口（provider分流路由） ====================

// SynthesizeSpeech 语音合成统一入口——按配置中的provider分流到具体实现
// 参数与返回值对所有provider保持一致，调用方（字幕服务等）零改动：
//   - cfg: TTS API配置（GetTTSConfig加载）
//   - text: 要合成的文本
//   - voice: 音色代码（空则用默认音色）
//   - speed: 语速（0则默认1.0）
//   - outputDir: 输出文件目录
//   - outputName: 输出文件名（不含扩展名，自动加.mp3）
//   - traceCtx: 追踪上下文（可为nil）
func SynthesizeSpeech(ctx context.Context, cfg *TTSConfig, text string, voice string, speed float64, outputDir string, outputName string, traceCtx *TraceContext) (*TTSResult, error) {
	// 公共参数校验与默认值
	if cfg == nil {
		return nil, fmt.Errorf("TTS配置为空")
	}
	if text == "" {
		return nil, fmt.Errorf("合成文本不能为空")
	}
	if voice == "" {
		voice = ttsDefaultVoice
	}
	if speed <= 0 {
		speed = 1.0
	}

	// 按provider分流
	switch cfg.Provider {
	case TTSProviderVolcanoV3:
		return synthesizeSpeechV3(ctx, cfg, text, voice, speed, outputDir, outputName, traceCtx)
	case TTSProviderOpenAI:
		return synthesizeSpeechOpenAI(ctx, cfg, text, voice, speed, outputDir, outputName, traceCtx)
	default:
		// 未知provider按v3处理（v3为当前唯一可用通道）
		ttsLog.Warn("未知TTS provider，按volcano_v3处理", "provider", cfg.Provider)
		return synthesizeSpeechV3(ctx, cfg, text, voice, speed, outputDir, outputName, traceCtx)
	}
}

// ==================== 旧OpenAI兼容实现（volcano_openai路径，保留回退） ====================

// synthesizeSpeechOpenAI 旧OpenAI兼容格式实现（原SynthesizeSpeech主体，原样保留）
// 已知该路径对豆包2.0不可用（端点不存在），仅作provider回退选项保留。
func synthesizeSpeechOpenAI(ctx context.Context, cfg *TTSConfig, text string, voice string, speed float64, outputDir string, outputName string, traceCtx *TraceContext) (*TTSResult, error) {
	if cfg.APIBaseURL == "" || cfg.APIKey == "" {
		return nil, fmt.Errorf("TTS API未配置（请在AI管理中心配置图片/视频生成API地址和密钥）")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("TTS模型未配置")
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

	ttsLog.Info("调用TTS语音合成API(OpenAI兼容路径)",
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
	httpResp, err :=
		client.Do(
			httpReq,
		)

	latencyMs :=
		time.Since(
			startTime,
		).Milliseconds()

	if err != nil {
		ttsLog.Error(
			"TTS HTTP请求结果不确定",
			"error",
			err,
			"latency_ms",
			latencyMs,
		)

		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"TTS请求网络结果不确定: %w",
					err,
				),
				Uncertain: true,
			}
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

	// 成功HTTP响应表示供应商已经完成本次合成。
	// 后续本地目录、文件或响应流处理失败不能释放预留。
	if err :=
		os.MkdirAll(
			outputDir,
			0755,
		); err != nil {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"创建输出目录失败: %w",
					err,
				),
				ProviderSucceeded: true,
			}
	}

	outputPath :=
		filepath.Join(
			outputDir,
			outputName+".mp3",
		)

	outFile, err :=
		os.Create(
			outputPath,
		)

	if err != nil {
		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"创建音频文件失败: %w",
					err,
				),
				ProviderSucceeded: true,
			}
	}

	written, copyErr :=
		io.Copy(
			outFile,
			httpResp.Body,
		)

	closeErr :=
		outFile.Close()

	if copyErr != nil {
		_ = os.Remove(outputPath)

		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"写入音频文件失败: %w",
					copyErr,
				),
				ProviderSucceeded: true,
			}
	}

	if closeErr != nil {
		_ = os.Remove(outputPath)

		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"关闭音频文件失败: %w",
					closeErr,
				),
				ProviderSucceeded: true,
			}
	}

	if written < 100 {
		content, _ :=
			os.ReadFile(
				outputPath,
			)

		_ = os.Remove(outputPath)

		return nil,
			&TTSSynthesisError{
				Cause: fmt.Errorf(
					"TTS生成的音频文件异常小(%d字节): %s",
					written,
					truncateStr(
						string(content),
						200,
					),
				),
				ProviderSucceeded: true,
			}
	}
	// 获取音频时长（通过ffprobe）
	duration := getAudioDuration(outputPath)

	ttsLog.Info("TTS合成成功(OpenAI兼容路径)",
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

// GetTTSConfig 从AI配置中心加载TTS配置（S-V1.5：按provider分支加载）
//   - volcano_v3（默认）: 读 tts_app_id + tts_access_token_enc(AES解密) + 可选tts_v3_base_url
//   - volcano_openai    : 沿用旧逻辑（image_api_base_url/image_api_key_enc/tts_default_model）
func GetTTSConfig(aesKey string) (*TTSConfig, error) {
	ctx := context.Background()
	cfg := &TTSConfig{}

	// 读取provider（缺省/为空按 volcano_v3）
	var provider string
	_ = database.DB.QueryRow(ctx,
		`SELECT config_value FROM ai_configs WHERE config_key = 'tts_provider'`).Scan(&provider)
	if strings.TrimSpace(provider) == "" {
		provider = TTSProviderVolcanoV3
	}
	cfg.Provider = provider

	// ---------- volcano_v3 分支 ----------
	if provider == TTSProviderVolcanoV3 {
		// APP ID（必填）
		var appID string
		if err := database.DB.QueryRow(ctx,
			`SELECT config_value FROM ai_configs WHERE config_key = 'tts_app_id'`).Scan(&appID); err != nil || strings.TrimSpace(appID) == "" {
			return nil, fmt.Errorf("TTS未配置：缺少火山APP ID（请在AI管理中心或 /api/v1/admin/tts-config 配置）")
		}
		cfg.AppID = strings.TrimSpace(appID)

		// Access Token（必填，AES解密）
		var tokenEnc string
		if err := database.DB.QueryRow(ctx,
			`SELECT config_value FROM ai_configs WHERE config_key = 'tts_access_token_enc'`).Scan(&tokenEnc); err != nil || strings.TrimSpace(tokenEnc) == "" {
			return nil, fmt.Errorf("TTS未配置：缺少火山Access Token（请在AI管理中心或 /api/v1/admin/tts-config 配置）")
		}
		token, err := utils.DecryptAES(tokenEnc, aesKey)
		if err != nil {
			return nil, fmt.Errorf("TTS Access Token解密失败: %w", err)
		}
		cfg.AccessToken = token

		// 基地址（可选覆盖，缺省官方域名）
		var baseURL string
		_ = database.DB.QueryRow(ctx,
			`SELECT config_value FROM ai_configs WHERE config_key = 'tts_v3_base_url'`).Scan(&baseURL)
		if strings.TrimSpace(baseURL) == "" {
			baseURL = ttsV3DefaultBaseURL
		}
		cfg.APIBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

		// 当前开放的官方2.0音色使用真实Resource ID，
		// 该值同时作为统一积分价格表的严格模型身份。
		cfg.Model = "seed-tts-2.0"
		return cfg, nil
	}

	// ---------- volcano_openai 分支（旧逻辑原样保留） ----------
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
