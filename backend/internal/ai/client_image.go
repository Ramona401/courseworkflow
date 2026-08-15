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
//
// 【尺寸兜底闸门 v0.42.1】
//   豆包 Seedream 硬性要求 image size 总像素须 ≥ 3,686,400（约 1920×1920 或等效面积），
//   低于此值直接返回 HTTP 400 InvalidParameter。历史上全自动装配链曾硬编码 1024×1024、
//   1280×720 等偏小尺寸导致整批配图失败。为杜绝同类问题复发，本文件在发请求前统一经
//   normalizeImageSize 校验：凡总像素低于下限的尺寸，按原始宽高比自动放大到达标，任何
//   调用方（手动生图/装配配图/视频首帧）传偏小尺寸都会被自动纠正，不再整批 400。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/utils"
)

// 模块日志
var imageLog = logger.WithModule("ai_image")

// 豆包 Seedream 尺寸约束常量
const (
	// doubaoMinImagePixels 豆包图片生成的最小总像素约束（宽×高须 ≥ 此值，否则 HTTP 400）
	doubaoMinImagePixels = 3686400
	// doubaoDefaultSize 未指定尺寸时的默认尺寸（1920×1920 = 3,686,400，正好达标）
	doubaoDefaultSize = "1920x1920"
	// doubaoSizeAlign 放大后宽高对齐到该倍数（避免奇数/非对齐尺寸再被拒）
	doubaoSizeAlign = 8
)

// ==================== 请求/响应结构体 ====================

// ImageGenerateRequest 图片生成请求（豆包OpenAI兼容格式）
type ImageGenerateRequest struct {
	Model  string `json:"model"`           // 模型名
	Prompt string `json:"prompt"`          // 生成提示词
	Size   string `json:"size,omitempty"`  // 图片尺寸，默认 1920x1920
	N      int    `json:"n,omitempty"`     // 生成数量，默认 1
	Image  string `json:"image,omitempty"` // 参考图URL（图生图模式，公网可访问）
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

// ImageProviderError 是图片供应商明确返回的结构化HTTP错误。
//
// 业务层只能依赖StatusCode与Code做稳定分类，禁止解析Error()文本或供应商原始响应体。
// Message仅用于服务端诊断；Error()刻意不拼接Message，避免请求ID或供应商内部信息进入教师端。
type ImageProviderError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *ImageProviderError) Error() string {
	if e == nil {
		return "图片生成供应商返回错误"
	}
	if strings.TrimSpace(e.Code) != "" {
		return fmt.Sprintf(
			"图片生成API返回错误(HTTP %d, code=%s)",
			e.StatusCode,
			strings.TrimSpace(e.Code),
		)
	}
	return fmt.Sprintf(
		"图片生成API返回错误(HTTP %d)",
		e.StatusCode,
	)
}

// IsImageInputTextSensitiveError 判断是否为供应商明确的“输入文本内容审核未通过”。
//
// 该判断只接受结构化provider error，不允许对普通错误文本做模糊匹配，避免把网络、配置、
// 下载或计费错误误判为内容审核并自动重试。
func IsImageInputTextSensitiveError(err error) bool {
	var providerErr *ImageProviderError
	if !errors.As(err, &providerErr) || providerErr == nil {
		return false
	}
	return strings.TrimSpace(providerErr.Code) ==
		"InputTextSensitiveContentDetected"
}

// parseImageProviderError 把供应商错误响应解析为稳定类型；解析失败时仍保留HTTP状态码。
func parseImageProviderError(
	statusCode int,
	body []byte,
) error {
	envelope := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}

	if len(body) > 0 {
		_ = json.Unmarshal(body, &envelope)
	}

	return &ImageProviderError{
		StatusCode: statusCode,
		Code:       strings.TrimSpace(envelope.Error.Code),
		Message:    strings.TrimSpace(envelope.Error.Message),
	}
}

// ImageConfig 图片生成API配置（从AI配置中心加载）
type ImageConfig struct {
	APIBaseURL string // API基地址
	APIKey     string // 明文API Key（已解密）
	Model      string // 模型名
}

// ==================== 尺寸兜底闸门 ====================

// normalizeImageSize 校验并纠正图片尺寸，保证总像素满足豆包下限。
//
// 逻辑：
//  1. 空字符串 → 返回默认尺寸 doubaoDefaultSize；
//  2. 无法解析为 "宽x高"（如 "1024x1024"）→ 记 Warn，返回默认尺寸兜底；
//  3. 总像素已达标 → 原样返回；
//  4. 总像素不足 → 按原始宽高比等比放大到刚好达标，宽高对齐到 doubaoSizeAlign 的倍数，
//     并记一条 INFO 说明"自动放大"，返回纠正后的尺寸。
//
// 返回：纠正后的尺寸字符串（形如 "2560x1440"）。永不返回不合法尺寸。
func normalizeImageSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return doubaoDefaultSize
	}

	w, h, ok := parseSize(size)
	if !ok || w <= 0 || h <= 0 {
		imageLog.Warn("图片尺寸无法解析，回退默认尺寸", "raw_size", size, "default", doubaoDefaultSize)
		return doubaoDefaultSize
	}

	// 已达标，原样返回
	if w*h >= doubaoMinImagePixels {
		return size
	}

	// 需放大：保持宽高比 r=w/h，令 (w*s)*(h*s) = doubaoMinImagePixels，
	// 即 s = sqrt(doubaoMinImagePixels / (w*h))；用整数运算避免引入 math 依赖：
	// 逐步放大直到达标（每次乘以比例并对齐），最多放大一次即可覆盖大多数情况，
	// 循环仅为对齐后可能仍差一点点的极端情况兜底。
	origW, origH := w, h
	for w*h < doubaoMinImagePixels {
		// 目标缩放因子（放大 1.02 倍冗余，抵消对齐向下取整带来的损耗）
		// 用整数放大：newArea 需 ≥ 下限，按面积比开方近似
		scaledW := scaleUpDimension(w, h)
		scaledH := scaleUpDimension(h, w)
		if scaledW <= w && scaledH <= h {
			// 兜底防死循环：直接强制放大到默认尺寸
			imageLog.Warn("图片尺寸放大异常，强制回退默认尺寸", "raw_size", size, "default", doubaoDefaultSize)
			return doubaoDefaultSize
		}
		w, h = scaledW, scaledH
	}

	fixed := fmt.Sprintf("%dx%d", w, h)
	imageLog.Info("图片尺寸低于豆包下限，已自动放大达标",
		"orig_size", fmt.Sprintf("%dx%d", origW, origH),
		"orig_pixels", origW*origH,
		"fixed_size", fixed,
		"fixed_pixels", w*h,
		"min_pixels", doubaoMinImagePixels,
	)
	return fixed
}

// parseSize 解析 "宽x高" 字符串（分隔符兼容小写 x 与大写 X）为宽高整数。
func parseSize(size string) (int, int, bool) {
	lower := strings.ToLower(size)
	parts := strings.SplitN(lower, "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return w, h, true
}

// scaleUpDimension 把某一维度（dim）按需要放大一档：
// 目标是让面积达标，这里对单个维度乘以一个足以覆盖面积缺口的因子，并对齐到 doubaoSizeAlign 的倍数。
// other 为另一维度（用于估算当前面积缺口）。
func scaleUpDimension(dim, other int) int {
	if dim <= 0 || other <= 0 {
		return dim
	}
	// 当前面积
	area := dim * other
	if area >= doubaoMinImagePixels {
		return alignUp(dim)
	}
	// 面积缺口比例（放大 1.02 倍冗余）：ratio = sqrt(min/area) ≈ 用整数近似
	// 为避免 math 依赖，用"面积比 * 1.02"的平方根近似：先算 min*10000/area 再开方（整数牛顿法）
	scaledArea := doubaoMinImagePixels*102/100 + 1
	ratioX10000 := isqrt(int64(scaledArea) * 10000 / int64(area)) // = sqrt(scaledArea/area)*100
	if ratioX10000 < 100 {
		ratioX10000 = 100
	}
	newDim := dim * int(ratioX10000) / 100
	if newDim <= dim {
		newDim = dim + doubaoSizeAlign
	}
	return alignUp(newDim)
}

// alignUp 把整数向上对齐到 doubaoSizeAlign 的倍数。
func alignUp(v int) int {
	if v <= 0 {
		return doubaoSizeAlign
	}
	r := v % doubaoSizeAlign
	if r == 0 {
		return v
	}
	return v + (doubaoSizeAlign - r)
}

// isqrt 整数平方根（牛顿迭代），用于避免引入 math 包。返回 floor(sqrt(n))。
func isqrt(n int64) int64 {
	if n < 0 {
		return 0
	}
	if n < 2 {
		return n
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// ==================== 图片生成核心函数 ====================

// GenerateImage 调用豆包API生成图片（支持图生图）
// 参数：
//   - cfg: 图片API配置
//   - prompt: 生成提示词
//   - size: 图片尺寸（空则默认1920x1920，总像素需≥3686400；低于下限会被 normalizeImageSize 自动放大）
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

	// 尺寸兜底闸门：空则默认，低于豆包下限则自动放大达标（杜绝 HTTP 400 InvalidParameter）
	size = normalizeImageSize(size)

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
	client := &http.Client{Timeout: 120 * time.Second}
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
		return nil, parseImageProviderError(
			httpResp.StatusCode,
			respBody,
		)
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
				traceCtx, cfg.Model,
				0, 0, 0, // tokens（图片生成不计token）
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
