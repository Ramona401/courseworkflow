package handlers

// tts_config_handler.go — TTS语音合成配置处理器（S-V1.5新增，admin专属）
//
// 端点（路由层已套 authMW + adminOnly）：
//   GET  /api/v1/admin/tts-config       — 查看当前TTS配置（Access Token脱敏）
//   PUT  /api/v1/admin/tts-config       — 更新provider/APP ID/Access Token（token留空=不修改）
//   POST /api/v1/admin/tts-config/test  — 服务端直连合成一句测试音频并立即删除，验证链路通畅
//
// 配置写入走 repository.UpsertConfigValue（键不存在自动INSERT），
// Access Token经 utils.EncryptAES 加密后存 tts_access_token_enc。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// TTSConfigHandler TTS配置处理器
type TTSConfigHandler struct {
	cfg *config.Config
}

// NewTTSConfigHandler 创建TTS配置处理器（构造函数内config.Load()，保持routes签名零侵入）
func NewTTSConfigHandler() *TTSConfigHandler {
	return &TTSConfigHandler{cfg: config.Load()}
}

// ==================== GET/PUT 分发 ====================

// HandleTTSConfig GET=查看 / PUT=更新
func (h *TTSConfigHandler) HandleTTSConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.putConfig(w, r)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT")
	}
}

// ttsConfigView TTS配置响应结构
type ttsConfigView struct {
	Provider       string `json:"provider"`         // volcano_v3 / volcano_openai
	AppID          string `json:"app_id"`           // 火山APP ID（明文展示）
	AccessToken    string `json:"access_token"`     // 脱敏展示
	AccessTokenSet bool   `json:"access_token_set"` // 是否已配置
	VoicesTotal    int    `json:"voices_total"`     // 当前内置音色数
}

// readConfigOrDefault 读单个配置键，缺失或为空返回默认值
func readConfigOrDefault(key, def string) string {
	c, err := repository.GetConfigByKey(key)
	if err != nil || strings.TrimSpace(c.ConfigValue) == "" {
		return def
	}
	return strings.TrimSpace(c.ConfigValue)
}

// buildTTSConfigView 组装当前配置视图（Access Token脱敏）
func (h *TTSConfigHandler) buildTTSConfigView() *ttsConfigView {
	view := &ttsConfigView{
		Provider:    readConfigOrDefault("tts_provider", ai.TTSProviderVolcanoV3),
		AppID:       readConfigOrDefault("tts_app_id", ""),
		VoicesTotal: len(ai.AvailableTTSVoices),
	}
	enc := readConfigOrDefault("tts_access_token_enc", "")
	if enc == "" {
		view.AccessToken = "未配置"
		view.AccessTokenSet = false
		return view
	}
	// 解密取原文做脱敏展示；解密失败按旧明文数据处理
	plain, err := utils.DecryptAES(enc, h.cfg.GetAESKey())
	if err != nil {
		plain = enc
	}
	if len(plain) <= 8 {
		view.AccessToken = "***"
	} else {
		view.AccessToken = plain[:4] + "***" + plain[len(plain)-4:]
	}
	view.AccessTokenSet = true
	return view
}

// getConfig GET /api/v1/admin/tts-config
func (h *TTSConfigHandler) getConfig(w http.ResponseWriter, _ *http.Request) {
	utils.Success(w, h.buildTTSConfigView())
}

// updateTTSConfigRequest 更新请求体
type updateTTSConfigRequest struct {
	Provider    string `json:"provider"`     // volcano_v3 / volcano_openai
	AppID       string `json:"app_id"`       // 火山APP ID
	AccessToken string `json:"access_token"` // 留空表示不修改
}

// putConfig PUT /api/v1/admin/tts-config
func (h *TTSConfigHandler) putConfig(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req updateTTSConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	req.Provider = strings.TrimSpace(req.Provider)
	req.AppID = strings.TrimSpace(req.AppID)
	req.AccessToken = strings.TrimSpace(req.AccessToken)

	// provider 校验（留空=不修改）
	if req.Provider != "" &&
		req.Provider != ai.TTSProviderVolcanoV3 &&
		req.Provider != ai.TTSProviderOpenAI {
		utils.BadRequest(w, "provider 无效，应为 volcano_v3 或 volcano_openai")
		return
	}

	userID := claims.UserID
	// 逐项写入（UPSERT：键不存在自动新建）
	if req.Provider != "" {
		if err := repository.UpsertConfigValue("tts_provider", req.Provider,
			"TTS语音合成provider（volcano_v3/volcano_openai）", userID); err != nil {
			utils.InternalError(w, "保存provider失败: "+err.Error())
			return
		}
	}
	if req.AppID != "" {
		if err := repository.UpsertConfigValue("tts_app_id", req.AppID,
			"火山豆包语音应用APP ID", userID); err != nil {
			utils.InternalError(w, "保存APP ID失败: "+err.Error())
			return
		}
	}
	if req.AccessToken != "" {
		encrypted, err := utils.EncryptAES(req.AccessToken, h.cfg.GetAESKey())
		if err != nil {
			utils.InternalError(w, "加密Access Token失败: "+err.Error())
			return
		}
		if err := repository.UpsertConfigValue("tts_access_token_enc", encrypted,
			"火山豆包语音Access Token（AES加密）", userID); err != nil {
			utils.InternalError(w, "保存Access Token失败: "+err.Error())
			return
		}
	}

	utils.Success(w, h.buildTTSConfigView())
}

// ==================== 服务端直连自测 ====================

// testTTSRequest 自测请求体（全部可选）
type testTTSRequest struct {
	Voice string `json:"voice"` // 缺省用默认音色
	Text  string `json:"text"`  // 缺省用固定测试句
}

// TestTTS POST /api/v1/admin/tts-config/test
// 用当前库内配置直连合成一句测试音频到临时目录，成功后立即删除，仅返回链路结论。
func (h *TTSConfigHandler) TestTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	// 解析可选参数（空请求体也允许）
	var req testTTSRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Text) == "" {
		req.Text = "你好，这是TE-DNA平台的语音合成链路测试。"
	}

	// 加载当前TTS配置
	ttsCfg, err := ai.GetTTSConfig(h.cfg.GetAESKey())
	if err != nil {
		utils.Success(w, map[string]interface{}{
			"success": false,
			"message": "配置加载失败: " + err.Error(),
		})
		return
	}

	// 直连合成到临时目录
	outputName := fmt.Sprintf("tts_selftest_%d", time.Now().UnixNano())
	start := time.Now()
	result, err := ai.SynthesizeSpeech(r.Context(), ttsCfg, req.Text, req.Voice, 1.0,
		os.TempDir(), outputName, nil)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		utils.Success(w, map[string]interface{}{
			"success":    false,
			"provider":   ttsCfg.Provider,
			"latency_ms": latencyMs,
			"message":    "合成失败: " + err.Error(),
		})
		return
	}

	// 测试文件用完即删
	_ = os.Remove(result.AudioFilePath)

	utils.Success(w, map[string]interface{}{
		"success":    true,
		"provider":   ttsCfg.Provider,
		"model":      result.ModelUsed,
		"duration":   result.Duration,
		"file_size":  result.FileSize,
		"latency_ms": latencyMs,
		"message":    fmt.Sprintf("链路畅通：合成%.1f秒音频(%d字节)，耗时%dms", result.Duration, result.FileSize, latencyMs),
	})
}
