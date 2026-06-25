package handlers

// domestic_gateway_handler.go — 境内文本网关（双网关分流的降级通道）连接配置处理器（批一新增，admin专属）
//
// 背景：
//   AI 文本调用走「双网关分流」——未授权学校的境外模型调用会被整通道切换到境内网关
//   （dashscope + qwen-max）。境内网关的三个配置键此前只能 SQL 直改或动态建立，
//   本处理器把它们搬到 AI 管理中心前端，让管理员可视化查看/修改/自测。
//
// 配置键（与 ai/client_model_policy.go 的 domesticBaseURLKey/domesticKeyEncKey/domesticModelKey 逐字一致）：
//   domestic_text_base_url  — 境内文本网关地址（dashscope /compatible-mode/v1）
//   domestic_text_key_enc   — 境内文本 API Key（AES 加密存储）
//   domestic_text_model     — 境内文本主力模型（qwen-max）
//
// 端点（路由层已套 authMW + adminOnly）：
//   GET  /api/v1/admin/domestic-gateway       — 查看当前境内通道配置（API Key 脱敏）
//   PUT  /api/v1/admin/domestic-gateway       — 更新三键（key 留空=不修改）
//   POST /api/v1/admin/domestic-gateway/test  — 用库内三键直连 dashscope 发一句测试请求验证链路
//
// 关键：PUT 成功后调用 ai.InvalidateDomesticChannelCache() 让 5 分钟内存缓存立即失效，
//       否则改完最长要等 5 分钟分流才读到新配置。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// 境内通道配置键名（必须与 ai/client_model_policy.go 中的常量逐字一致）
const (
	dgBaseURLKey = "domestic_text_base_url" // 境内文本网关地址
	dgKeyEncKey  = "domestic_text_key_enc"  // 境内文本 API Key（AES 加密）
	dgModelKey   = "domestic_text_model"    // 境内文本主力模型
)

// DomesticGatewayHandler 境内网关配置处理器
type DomesticGatewayHandler struct {
	cfg *config.Config
}

// NewDomesticGatewayHandler 创建处理器（构造函数内 config.Load() 取 AES 密钥，保持 routes 签名零侵入）
func NewDomesticGatewayHandler() *DomesticGatewayHandler {
	return &DomesticGatewayHandler{cfg: config.Load()}
}

// ==================== GET/PUT 分发 ====================

// HandleDomesticGateway GET=查看 / PUT=更新
func (h *DomesticGatewayHandler) HandleDomesticGateway(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.putConfig(w, r)
	default:
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET/PUT")
	}
}

// domesticGatewayView 境内网关配置响应结构（API Key 脱敏）
type domesticGatewayView struct {
	BaseURL    string `json:"base_url"`     // 境内网关地址（明文展示）
	Model      string `json:"model"`        // 境内主力模型（明文展示）
	APIKey     string `json:"api_key"`      // 脱敏展示（首尾4位）
	APIKeySet  bool   `json:"api_key_set"`  // 是否已配置 API Key
}

// dgReadConfigOrDefault 读单个配置键，缺失或为空返回默认值
func dgReadConfigOrDefault(key, def string) string {
	c, err := repository.GetConfigByKey(key)
	if err != nil || c == nil || strings.TrimSpace(c.ConfigValue) == "" {
		return def
	}
	return strings.TrimSpace(c.ConfigValue)
}

// buildDomesticGatewayView 组装当前配置视图（API Key 脱敏）
func (h *DomesticGatewayHandler) buildDomesticGatewayView() *domesticGatewayView {
	view := &domesticGatewayView{
		BaseURL: dgReadConfigOrDefault(dgBaseURLKey, ""),
		Model:   dgReadConfigOrDefault(dgModelKey, ""),
	}
	enc := dgReadConfigOrDefault(dgKeyEncKey, "")
	if enc == "" {
		view.APIKey = "未配置"
		view.APIKeySet = false
		return view
	}
	// 解密取原文做脱敏；解密失败按旧明文数据处理
	plain, err := utils.DecryptAES(enc, h.cfg.GetAESKey())
	if err != nil {
		plain = enc
	}
	if len(plain) <= 8 {
		view.APIKey = "***"
	} else {
		view.APIKey = plain[:4] + "***" + plain[len(plain)-4:]
	}
	view.APIKeySet = true
	return view
}

// getConfig GET /api/v1/admin/domestic-gateway
func (h *DomesticGatewayHandler) getConfig(w http.ResponseWriter, _ *http.Request) {
	utils.Success(w, h.buildDomesticGatewayView())
}

// updateDomesticGatewayRequest 更新请求体（三字段均可选，留空=不修改对应项）
type updateDomesticGatewayRequest struct {
	BaseURL string `json:"base_url"` // 境内网关地址
	Model   string `json:"model"`    // 境内主力模型
	APIKey  string `json:"api_key"`  // API Key 明文，留空=不修改
}

// putConfig PUT /api/v1/admin/domestic-gateway
func (h *DomesticGatewayHandler) putConfig(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	var req updateDomesticGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.Model = strings.TrimSpace(req.Model)
	req.APIKey = strings.TrimSpace(req.APIKey)

	userID := claims.UserID

	// 逐项写入（UPSERT：键不存在自动新建）
	if req.BaseURL != "" {
		if err := repository.UpsertConfigValue(dgBaseURLKey, req.BaseURL,
			"境内文本网关地址（dashscope兼容模式）", userID); err != nil {
			utils.InternalError(w, "保存网关地址失败: "+err.Error())
			return
		}
	}
	if req.Model != "" {
		if err := repository.UpsertConfigValue(dgModelKey, req.Model,
			"境内文本主力模型（如qwen-max）", userID); err != nil {
			utils.InternalError(w, "保存模型失败: "+err.Error())
			return
		}
	}
	if req.APIKey != "" {
		encrypted, err := utils.EncryptAES(req.APIKey, h.cfg.GetAESKey())
		if err != nil {
			utils.InternalError(w, "加密API Key失败: "+err.Error())
			return
		}
		if err := repository.UpsertConfigValue(dgKeyEncKey, encrypted,
			"境内文本网关API Key（AES加密）", userID); err != nil {
			utils.InternalError(w, "保存API Key失败: "+err.Error())
			return
		}
	}

	// 关键：让分流模块的 5 分钟境内通道缓存立即失效，使本次修改即时生效
	ai.InvalidateDomesticChannelCache()

	utils.Success(w, h.buildDomesticGatewayView())
}

// ==================== 服务端直连自测 ====================

// dgTestResult 自测结果（与前端 DomesticGatewayTestResult 对齐）
type dgTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
	Model     string `json:"model"`
	BaseURL   string `json:"base_url"`
}

// TestDomesticGateway POST /api/v1/admin/domestic-gateway/test
// 用库内当前三键拼 {base}/chat/completions 向 dashscope 发一句最短测试请求，验证链路畅通。
// 该端点不写任何数据，仅做连通性验证。
func (h *DomesticGatewayHandler) TestDomesticGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST")
		return
	}

	// 1. 读三键
	baseURL := dgReadConfigOrDefault(dgBaseURLKey, "")
	model := dgReadConfigOrDefault(dgModelKey, "")
	enc := dgReadConfigOrDefault(dgKeyEncKey, "")

	if baseURL == "" {
		utils.Success(w, dgTestResult{Success: false, Message: "境内网关地址未配置"})
		return
	}
	if model == "" {
		utils.Success(w, dgTestResult{Success: false, Message: "境内主力模型未配置", BaseURL: baseURL})
		return
	}
	if enc == "" {
		utils.Success(w, dgTestResult{Success: false, Message: "境内网关API Key未配置", BaseURL: baseURL, Model: model})
		return
	}

	// 2. 解密 API Key（解密失败按旧明文兜底）
	apiKey, decErr := utils.DecryptAES(enc, h.cfg.GetAESKey())
	if decErr != nil || apiKey == "" {
		apiKey = enc
	}
	if strings.TrimSpace(apiKey) == "" {
		utils.Success(w, dgTestResult{Success: false, Message: "境内网关API Key解密为空", BaseURL: baseURL, Model: model})
		return
	}

	// 3. 构造 OpenAI 兼容格式最短测试请求
	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens":  10,
		"temperature": 0.0,
	}
	jsonBody, _ := json.Marshal(requestBody)

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		utils.Success(w, dgTestResult{Success: false, Message: "创建请求失败: " + err.Error(), BaseURL: baseURL, Model: model})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 4. 发送并计时（30秒超时）
	httpClient := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		utils.Success(w, dgTestResult{Success: false, Message: "网络连接失败: " + err.Error(), LatencyMs: latencyMs, BaseURL: baseURL, Model: model})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 5. 判断状态码
	if resp.StatusCode != http.StatusOK {
		msg := dgExtractErr(respBody)
		statusText := dgStatusText(resp.StatusCode)
		if msg != "" {
			statusText = statusText + "；详情: " + msg
		}
		utils.Success(w, dgTestResult{Success: false, Message: statusText, LatencyMs: latencyMs, BaseURL: baseURL, Model: model})
		return
	}

	// 6. 解析成功响应（须含 choices 字段）
	var chatResp map[string]interface{}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		utils.Success(w, dgTestResult{Success: false, Message: "响应格式异常，非标准JSON", LatencyMs: latencyMs, BaseURL: baseURL, Model: model})
		return
	}
	if _, ok := chatResp["choices"]; !ok {
		utils.Success(w, dgTestResult{Success: false, Message: "响应缺少choices字段，可能不是OpenAI兼容API", LatencyMs: latencyMs, BaseURL: baseURL, Model: model})
		return
	}

	utils.Success(w, dgTestResult{
		Success:   true,
		Message:   "境内通道畅通！响应延迟 " + itoaMs(latencyMs) + "ms",
		LatencyMs: latencyMs,
		Model:     model,
		BaseURL:   baseURL,
	})
}

// dgStatusText 把常见 HTTP 状态码翻译为人话
func dgStatusText(code int) string {
	switch code {
	case 401:
		return "认证失败(401) — API Key无效或已过期"
	case 403:
		return "访问被拒绝(403) — 无权限访问该模型"
	case 404:
		return "接口不存在(404) — 请检查网关地址是否正确"
	case 429:
		return "请求过于频繁(429) — API频率限制"
	case 400:
		return "请求被拒绝(400) — 可能是模型名错误或参数超限"
	case 500, 502, 503:
		return "服务端错误 — 境内网关暂时不可用"
	default:
		return "HTTP " + itoaInt(code)
	}
}

// dgExtractErr 从错误响应提取 message
func dgExtractErr(body []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return errResp.Error.Message
	}
	raw := string(body)
	if len(raw) > 200 {
		raw = raw[:200] + "..."
	}
	return raw
}

// itoaMs / itoaInt 极简整数转字符串（避免再引入 strconv 仅为拼一条消息）
func itoaMs(v int64) string {
	return itoaInt(int(v))
}

func itoaInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ListDomesticModels GET /api/v1/admin/domestic-gateway/models
// 用境内网关库内三键（base_url + key）调 dashscope 的 OpenAI 兼容 GET {base}/models，
// 返回境内网关实际可用的模型名列表，便于 admin 查到真实模型名（如 qwen-max）填到单价表。
// 纯查询，不写任何数据。
func (h *DomesticGatewayHandler) ListDomesticModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}

	// 1. 读三键中的 base_url + key（model 不需要）
	baseURL := dgReadConfigOrDefault(dgBaseURLKey, "")
	enc := dgReadConfigOrDefault(dgKeyEncKey, "")
	if baseURL == "" {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "境内网关地址未配置"})
		return
	}
	if enc == "" {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "境内网关API Key未配置"})
		return
	}

	// 2. 解密 key（解密失败按旧明文兜底）
	apiKey, decErr := utils.DecryptAES(enc, h.cfg.GetAESKey())
	if decErr != nil || apiKey == "" {
		apiKey = enc
	}
	if strings.TrimSpace(apiKey) == "" {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "境内网关API Key解密为空"})
		return
	}

	// 3. 调 GET {base}/models
	endpoint := strings.TrimRight(baseURL, "/") + "/models"
	httpReq, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "创建请求失败: " + err.Error()})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "网络连接失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		msg := dgExtractErr(respBody)
		statusText := dgStatusText(resp.StatusCode)
		if msg != "" {
			statusText = statusText + "；详情: " + msg
		}
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": statusText})
		return
	}

	// 4. 解析 OpenAI 兼容返回：{ "data": [ {"id": "qwen-max", ...}, ... ] }
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		utils.Success(w, map[string]interface{}{"models": []string{}, "total": 0, "message": "响应格式异常，非标准 /models 返回"})
		return
	}

	modelIDs := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if strings.TrimSpace(m.ID) != "" {
			modelIDs = append(modelIDs, m.ID)
		}
	}

	utils.Success(w, map[string]interface{}{
		"models": modelIDs,
		"total":  len(modelIDs),
	})
}
