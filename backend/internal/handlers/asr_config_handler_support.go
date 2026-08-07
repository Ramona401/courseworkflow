package handlers

// asr_config_handler_support.go — ASR配置读取、脱敏与校验辅助
//
// 本文件不注册HTTP端点，也不连接豆包。
// 它负责：
//   - 读取独立ASR配置键且不回退到TTS；
//   - 组装管理后台脱敏视图；
//   - 严格解析保存请求JSON；
//   - 校验APP ID和处理Token脱敏。

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"tedna/internal/ai"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// loadASRStoredConfig 读取独立ASR配置键，不回退到TTS。
func loadASRStoredConfig() (
	*asrStoredConfig,
	error,
) {
	source, err := repository.GetConfigValue(
		"asr_credential_source",
	)
	if err != nil {
		return nil, err
	}
	if source == "" {
		source =
			ai.ASRCredentialSourceTTS
	}

	appID, err := repository.GetConfigValue(
		"asr_app_id",
	)
	if err != nil {
		return nil, err
	}

	tokenEncrypted, err :=
		repository.GetConfigValue(
			"asr_access_token_enc",
		)
	if err != nil {
		return nil, err
	}

	webSocketURL, err :=
		repository.GetConfigValue(
			"asr_ws_url",
		)
	if err != nil {
		return nil, err
	}
	if webSocketURL == "" {
		webSocketURL =
			ai.ASRDefaultWebSocketURL
	}

	resourceID, err :=
		repository.GetConfigValue(
			"asr_resource_id",
		)
	if err != nil {
		return nil, err
	}
	if resourceID == "" {
		resourceID =
			ai.ASRDefaultResourceID
	}

	maxDurationSeconds :=
		ai.ASRDefaultMaxDurationSeconds

	maxDurationRaw, err :=
		repository.GetConfigValue(
			"asr_max_duration_seconds",
		)
	if err != nil {
		return nil, err
	}
	if maxDurationRaw != "" {
		parsed, parseErr := strconv.Atoi(
			maxDurationRaw,
		)
		if parseErr != nil {
			return nil, errors.New(
				"当前ASR录音时长配置不是有效整数",
			)
		}
		maxDurationSeconds = parsed
	}

	return &asrStoredConfig{
		CredentialSource: strings.ToLower(
			strings.TrimSpace(source),
		),
		AppID: strings.TrimSpace(appID),
		AccessTokenEnc: strings.TrimSpace(
			tokenEncrypted,
		),
		WebSocketURL: strings.TrimSpace(
			webSocketURL,
		),
		ResourceID: strings.TrimSpace(
			resourceID,
		),
		MaxDurationSeconds: maxDurationSeconds,
	}, nil
}

// buildASRConfigView 组装脱敏管理视图。
func (handler *ASRConfigHandler) buildASRConfigView() (
	*asrConfigView,
	error,
) {
	stored, err :=
		loadASRStoredConfig()
	if err != nil {
		return nil, err
	}

	maskedToken := "未配置"
	tokenSet :=
		stored.AccessTokenEnc != ""

	if tokenSet {
		tokenPlain, decryptErr :=
			utils.DecryptAES(
				stored.AccessTokenEnc,
				handler.cfg.GetAESKey(),
			)
		if decryptErr != nil {
			return nil, errors.New(
				"ASR Access Token解密失败，请重新填写并保存",
			)
		}

		maskedToken =
			maskASRSecret(
				tokenPlain,
			)
	}

	separate :=
		stored.CredentialSource ==
			ai.ASRCredentialSourceASR

	return &asrConfigView{
		CredentialSource:         stored.CredentialSource,
		UsingSeparateCredentials: separate,
		Configured: separate &&
			stored.AppID != "" &&
			tokenSet,
		AppID:              stored.AppID,
		AccessToken:        maskedToken,
		AccessTokenSet:     tokenSet,
		WebSocketURL:       stored.WebSocketURL,
		ResourceID:         stored.ResourceID,
		MaxDurationSeconds: stored.MaxDurationSeconds,
		ServiceName:        "豆包流式语音识别模型2.0",
		BillingMode:        "小时版",
	}, nil
}

// decodeStrictASRConfigJSON 严格解析一个JSON对象。
func decodeStrictASRConfigJSON(
	body io.Reader,
	target interface{},
) error {
	raw, err := io.ReadAll(
		io.LimitReader(
			body,
			64*1024,
		),
	)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(
		bytes.NewReader(raw),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		target,
	); err != nil {
		return err
	}

	var trailing interface{}
	if err := decoder.Decode(
		&trailing,
	); !errors.Is(err, io.EOF) {
		return errors.New(
			"请求体包含多余内容",
		)
	}

	return nil
}

// firstASRConfigValue 返回第一个非空配置值。
func firstASRConfigValue(
	values ...string,
) string {
	for _, value := range values {
		normalized :=
			strings.TrimSpace(value)
		if normalized != "" {
			return normalized
		}
	}

	return ""
}

// isDecimalASRAppID 校验APP ID只包含ASCII十进制数字。
func isDecimalASRAppID(
	value string,
) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	for _, char := range value {
		if !unicode.IsDigit(char) ||
			char > unicode.MaxASCII {
			return false
		}
	}

	return true
}

// maskASRSecret 返回首尾4位脱敏结果。
func maskASRSecret(
	value string,
) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)

	if len(runes) <= 8 {
		return "***"
	}

	return string(runes[:4]) +
		"***" +
		string(
			runes[len(runes)-4:],
		)
}
