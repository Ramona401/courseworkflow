package handlers

// asr_config_handler.go — 流式语音识别独立配置处理器
//
// 管理端点（路由层套authMW + adminOnly）：
//   GET  /api/v1/admin/asr-config
//   PUT  /api/v1/admin/asr-config
//   POST /api/v1/admin/asr-config/test
//
// 配置键：
//   - asr_credential_source：保存成功后固定为asr；
//   - asr_app_id：流式语音识别应用APP ID；
//   - asr_access_token_enc：AES加密后的Access Token；
//   - asr_ws_url：双向流式优化版WebSocket地址；
//   - asr_resource_id：ASR 2.0小时版资源ID；
//   - asr_max_duration_seconds：单次录音上限。
//
// 安全边界：
//   - GET只返回脱敏Token；
//   - PUT中Token留空表示保留原值；
//   - 首次保存必须提交Token；
//   - 六个配置键在一个数据库事务中原子写入；
//   - 测试连接不采集音频，只完成豆包握手与首包确认；
//   - TTS配置不被读取、覆盖或修改。
//
// 配置读取、脱敏和严格JSON辅助位于asr_config_handler_support.go。

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ASRConfigHandler 管理独立ASR配置。
type ASRConfigHandler struct {
	cfg *config.Config
}

// NewASRConfigHandler 创建ASR配置处理器。
//
// 与现有TTSConfigHandler保持相同构造范式，避免扩大课件路由注册函数签名。
func NewASRConfigHandler() *ASRConfigHandler {
	return &ASRConfigHandler{
		cfg: config.Load(),
	}
}

// ==================== API模型 ====================

// asrConfigView 是管理后台可安全展示的ASR配置。
type asrConfigView struct {
	CredentialSource         string `json:"credential_source"`
	UsingSeparateCredentials bool   `json:"using_separate_credentials"`
	Configured               bool   `json:"configured"`
	AppID                    string `json:"app_id"`
	AccessToken              string `json:"access_token"`
	AccessTokenSet           bool   `json:"access_token_set"`
	WebSocketURL             string `json:"ws_url"`
	ResourceID               string `json:"resource_id"`
	MaxDurationSeconds       int    `json:"max_duration_seconds"`
	ServiceName              string `json:"service_name"`
	BillingMode              string `json:"billing_mode"`
}

// updateASRConfigRequest 是管理后台保存请求。
//
// MaxDurationSeconds使用指针区分“未提交”和“提交0”。
type updateASRConfigRequest struct {
	AppID              string `json:"app_id"`
	AccessToken        string `json:"access_token"`
	WebSocketURL       string `json:"ws_url"`
	ResourceID         string `json:"resource_id"`
	MaxDurationSeconds *int   `json:"max_duration_seconds"`
}

// asrStoredConfig 是数据库内独立ASR键的原始状态。
type asrStoredConfig struct {
	CredentialSource   string
	AppID              string
	AccessTokenEnc     string
	WebSocketURL       string
	ResourceID         string
	MaxDurationSeconds int
}

// ==================== GET/PUT分发 ====================

// HandleASRConfig 处理GET查看和PUT保存。
func (handler *ASRConfigHandler) HandleASRConfig(
	writer http.ResponseWriter,
	request *http.Request,
) {
	switch request.Method {
	case http.MethodGet:
		handler.getConfig(writer)

	case http.MethodPut:
		handler.putConfig(writer, request)

	default:
		utils.Fail(
			writer,
			http.StatusMethodNotAllowed,
			"仅支持GET/PUT",
		)
	}
}

// getConfig GET /api/v1/admin/asr-config。
func (handler *ASRConfigHandler) getConfig(
	writer http.ResponseWriter,
) {
	view, err := handler.buildASRConfigView()
	if err != nil {
		utils.InternalError(
			writer,
			err.Error(),
		)
		return
	}

	utils.Success(writer, view)
}

// putConfig PUT /api/v1/admin/asr-config。
func (handler *ASRConfigHandler) putConfig(
	writer http.ResponseWriter,
	request *http.Request,
) {
	claims, ok := middleware.GetClaims(
		request.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			writer,
			"未认证",
		)
		return
	}

	var input updateASRConfigRequest
	if err := decodeStrictASRConfigJSON(
		request.Body,
		&input,
	); err != nil {
		utils.BadRequest(
			writer,
			"请求体格式错误",
		)
		return
	}

	stored, err := loadASRStoredConfig()
	if err != nil {
		utils.InternalError(
			writer,
			err.Error(),
		)
		return
	}

	appID := firstASRConfigValue(
		input.AppID,
		stored.AppID,
	)
	if appID == "" {
		utils.BadRequest(
			writer,
			"APP ID不能为空",
		)
		return
	}
	if !isDecimalASRAppID(appID) {
		utils.BadRequest(
			writer,
			"APP ID只能包含数字",
		)
		return
	}

	tokenPlain := strings.TrimSpace(
		input.AccessToken,
	)
	tokenEncrypted := stored.AccessTokenEnc

	if tokenPlain == "" {
		if tokenEncrypted == "" {
			utils.BadRequest(
				writer,
				"首次保存必须填写Access Token",
			)
			return
		}

		tokenPlain, err = utils.DecryptAES(
			tokenEncrypted,
			handler.cfg.GetAESKey(),
		)
		if err != nil {
			utils.InternalError(
				writer,
				"当前ASR Access Token解密失败，请重新填写后保存",
			)
			return
		}
	} else {
		tokenEncrypted, err = utils.EncryptAES(
			tokenPlain,
			handler.cfg.GetAESKey(),
		)
		if err != nil {
			utils.InternalError(
				writer,
				"加密Access Token失败: "+err.Error(),
			)
			return
		}
	}

	webSocketURL := firstASRConfigValue(
		input.WebSocketURL,
		stored.WebSocketURL,
		ai.ASRDefaultWebSocketURL,
	)

	resourceID := firstASRConfigValue(
		input.ResourceID,
		stored.ResourceID,
		ai.ASRDefaultResourceID,
	)

	maxDurationSeconds :=
		stored.MaxDurationSeconds
	if input.MaxDurationSeconds != nil {
		maxDurationSeconds =
			*input.MaxDurationSeconds
	}

	candidate := &ai.ASRConfig{
		CredentialSource:   ai.ASRCredentialSourceASR,
		AppID:              appID,
		AccessToken:        strings.TrimSpace(tokenPlain),
		WebSocketURL:       webSocketURL,
		ResourceID:         resourceID,
		MaxDurationSeconds: maxDurationSeconds,
	}

	if err := candidate.Validate(); err != nil {
		utils.BadRequest(
			writer,
			err.Error(),
		)
		return
	}

	updates := []repository.ConfigValueUpdate{
		{
			Key:         "asr_credential_source",
			Value:       ai.ASRCredentialSourceASR,
			Description: "ASR凭据来源（asr=独立凭据，tts=历史兼容复用）",
		},
		{
			Key:         "asr_app_id",
			Value:       candidate.AppID,
			Description: "火山豆包流式语音识别应用APP ID",
		},
		{
			Key:         "asr_access_token_enc",
			Value:       tokenEncrypted,
			Description: "火山豆包流式语音识别Access Token（AES加密）",
		},
		{
			Key:         "asr_ws_url",
			Value:       candidate.WebSocketURL,
			Description: "豆包流式语音识别WebSocket地址",
		},
		{
			Key:         "asr_resource_id",
			Value:       candidate.ResourceID,
			Description: "豆包流式语音识别资源ID",
		},
		{
			Key: "asr_max_duration_seconds",
			Value: strconv.Itoa(
				candidate.MaxDurationSeconds,
			),
			Description: "单次语音输入最长秒数",
		},
	}

	if err := repository.UpsertConfigValues(
		updates,
		claims.UserID,
	); err != nil {
		utils.InternalError(
			writer,
			"保存ASR配置失败: "+err.Error(),
		)
		return
	}

	view, err := handler.buildASRConfigView()
	if err != nil {
		utils.InternalError(
			writer,
			err.Error(),
		)
		return
	}

	utils.Success(writer, view)
}

// ==================== 服务端直连测试 ====================

// TestASR POST /api/v1/admin/asr-config/test。
//
// 测试只建立豆包WebSocket、发送首包并等待确认，然后立即关闭。
// 不上传音频，不产生识别正文，也不修改任何配置。
func (handler *ASRConfigHandler) TestASR(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		utils.Fail(
			writer,
			http.StatusMethodNotAllowed,
			"仅支持POST",
		)
		return
	}

	startedAt := time.Now()

	asrConfig, err := ai.GetASRConfig(
		handler.cfg.GetAESKey(),
	)
	if err != nil {
		writeASRTestResult(
			writer,
			false,
			time.Since(startedAt),
			"",
			"",
			"",
			"",
			"配置加载失败: "+err.Error(),
		)
		return
	}

	if asrConfig.CredentialSource !=
		ai.ASRCredentialSourceASR {
		writeASRTestResult(
			writer,
			false,
			time.Since(startedAt),
			asrConfig.ResourceID,
			asrConfig.WebSocketURL,
			"",
			"",
			"请先保存独立ASR配置，当前仍在兼容复用TTS凭据",
		)
		return
	}

	testContext, cancel := context.WithTimeout(
		request.Context(),
		20*time.Second,
	)
	defer cancel()

	session, err := ai.OpenASRSession(
		testContext,
		asrConfig,
		ai.DefaultASRRequestOptions(""),
	)
	latency := time.Since(startedAt)

	if err != nil {
		writeASRTestResult(
			writer,
			false,
			latency,
			asrConfig.ResourceID,
			asrConfig.WebSocketURL,
			"",
			"",
			"连接失败: "+
				utils.SafeTruncate(
					err.Error(),
					500,
				),
		)
		return
	}
	defer session.Close()

	writeASRTestResult(
		writer,
		true,
		latency,
		asrConfig.ResourceID,
		asrConfig.WebSocketURL,
		session.RequestID(),
		session.LogID(),
		"链路畅通：已完成WebSocket握手和ASR首包确认",
	)
}

// writeASRTestResult 返回统一测试结果。
func writeASRTestResult(
	writer http.ResponseWriter,
	success bool,
	latency time.Duration,
	resourceID string,
	webSocketURL string,
	requestID string,
	logID string,
	message string,
) {
	utils.Success(
		writer,
		map[string]interface{}{
			"success":     success,
			"latency_ms":  latency.Milliseconds(),
			"resource_id": resourceID,
			"ws_url":      webSocketURL,
			"request_id":  requestID,
			"log_id":      logID,
			"message":     message,
		},
	)
}
