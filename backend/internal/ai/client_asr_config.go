package ai

// client_asr_config.go — 火山豆包流式语音识别运行配置
//
// 配置保存在现有ai_configs键值表中，不新增数据库表：
//   - asr_credential_source：tts / asr，默认tts；
//   - 默认复用tts_app_id和tts_access_token_enc；
//   - 如未来ASR与TTS使用不同应用，可切换为asr来源；
//   - asr_ws_url：缺省使用双向流式优化版；
//   - asr_resource_id：缺省使用ASR 2.0小时版；
//   - asr_max_duration_seconds：缺省120秒。
//
// 安全要求：
//   - Access Token只在内存中解密，不写日志；
//   - 数据库查询错误必须向上返回，不能伪装为“配置缺失”；
//   - 上游地址必须使用wss://；
//   - 单次录音时长必须有明确上下限。

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/utils"
)

// ==================== 默认配置 ====================

const (
	// ASRDefaultWebSocketURL 火山双向流式优化版接口。
	ASRDefaultWebSocketURL = "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"

	// ASRDefaultResourceID 豆包流式语音识别2.0小时版资源ID。
	ASRDefaultResourceID = "volc.seedasr.sauc.duration"

	// ASRDefaultMaxDurationSeconds 单次录音默认上限。
	ASRDefaultMaxDurationSeconds = 120

	// ASRCredentialSourceTTS 表示ASR临时复用TTS应用凭据。
	// 仅用于历史兼容；管理后台保存独立ASR配置后应切换为asr。
	ASRCredentialSourceTTS = "tts"

	// ASRCredentialSourceASR 表示使用独立的ASR APP ID和Access Token。
	ASRCredentialSourceASR = "asr"
)

// ASRConfig 豆包流式语音识别运行配置。
type ASRConfig struct {
	CredentialSource   string
	AppID              string
	AccessToken        string
	WebSocketURL       string
	ResourceID         string
	MaxDurationSeconds int
}

// Validate 在建立任何外部连接前校验配置。
func (cfg *ASRConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("ASR配置为空")
	}

	if strings.TrimSpace(cfg.AppID) == "" {
		return fmt.Errorf("ASR配置缺少APP ID")
	}

	if strings.TrimSpace(cfg.AccessToken) == "" {
		return fmt.Errorf("ASR配置缺少Access Token")
	}

	parsedURL, err := url.Parse(strings.TrimSpace(cfg.WebSocketURL))
	if err != nil {
		return fmt.Errorf("ASR上游地址解析失败: %w", err)
	}
	if strings.ToLower(parsedURL.Scheme) != "wss" || parsedURL.Host == "" {
		return fmt.Errorf("ASR上游地址必须是有效的wss://安全地址")
	}

	if strings.TrimSpace(cfg.ResourceID) == "" {
		return fmt.Errorf("ASR配置缺少Resource ID")
	}

	if cfg.MaxDurationSeconds < 5 || cfg.MaxDurationSeconds > 300 {
		return fmt.Errorf("ASR单次录音上限必须在5至300秒之间")
	}

	return nil
}

// GetASRConfig 从ai_configs加载ASR配置。
//
// 默认复用已存在的TTS应用凭据，避免同一火山应用重复保存密钥。
// ASR协议、Resource ID和运行代码仍与TTS完全隔离。
func GetASRConfig(aesKey string) (*ASRConfig, error) {
	ctx := context.Background()

	credentialSource, err := readASRConfigValue(
		ctx,
		"asr_credential_source",
	)
	if err != nil {
		return nil, err
	}
	credentialSource = strings.ToLower(credentialSource)
	if credentialSource == "" {
		credentialSource = ASRCredentialSourceTTS
	}

	appIDKey := ""
	tokenKey := ""

	switch credentialSource {
	case ASRCredentialSourceTTS:
		appIDKey = "tts_app_id"
		tokenKey = "tts_access_token_enc"

	case ASRCredentialSourceASR:
		appIDKey = "asr_app_id"
		tokenKey = "asr_access_token_enc"

	default:
		return nil, fmt.Errorf("ASR凭据来源无效: %s", credentialSource)
	}

	appID, err := readASRConfigValue(ctx, appIDKey)
	if err != nil {
		return nil, err
	}
	if appID == "" {
		return nil, fmt.Errorf("ASR未配置：缺少%s", appIDKey)
	}

	encryptedToken, err := readASRConfigValue(ctx, tokenKey)
	if err != nil {
		return nil, err
	}
	if encryptedToken == "" {
		return nil, fmt.Errorf("ASR未配置：缺少%s", tokenKey)
	}

	accessToken, err := utils.DecryptAES(encryptedToken, aesKey)
	if err != nil {
		return nil, fmt.Errorf("ASR Access Token解密失败: %w", err)
	}

	webSocketURL, err := readASRConfigValue(ctx, "asr_ws_url")
	if err != nil {
		return nil, err
	}
	if webSocketURL == "" {
		webSocketURL = ASRDefaultWebSocketURL
	}

	resourceID, err := readASRConfigValue(ctx, "asr_resource_id")
	if err != nil {
		return nil, err
	}
	if resourceID == "" {
		resourceID = ASRDefaultResourceID
	}

	maxDurationSeconds := ASRDefaultMaxDurationSeconds
	maxDurationRaw, err := readASRConfigValue(ctx, "asr_max_duration_seconds")
	if err != nil {
		return nil, err
	}
	if maxDurationRaw != "" {
		parsed, parseErr := strconv.Atoi(maxDurationRaw)
		if parseErr != nil {
			return nil, fmt.Errorf(
				"ASR录音时长配置不是有效整数: %w",
				parseErr,
			)
		}
		maxDurationSeconds = parsed
	}

	cfg := &ASRConfig{
		CredentialSource:   credentialSource,
		AppID:              strings.TrimSpace(appID),
		AccessToken:        strings.TrimSpace(accessToken),
		WebSocketURL:       strings.TrimSpace(webSocketURL),
		ResourceID:         strings.TrimSpace(resourceID),
		MaxDurationSeconds: maxDurationSeconds,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// readASRConfigValue 读取单个ai_configs值。
//
// 配置键不存在时返回空字符串；数据库基础设施错误则原样向上返回。
func readASRConfigValue(ctx context.Context, key string) (string, error) {
	var value string

	err := database.DB.QueryRow(
		ctx,
		`SELECT config_value
                 FROM ai_configs
                 WHERE config_key = $1`,
		key,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取ASR配置%s失败: %w", key, err)
	}

	return strings.TrimSpace(value), nil
}
