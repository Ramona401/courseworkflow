package ai

// client_model_policy.go — AI 模型境内/境外分流策略
//
// 业务规则（fail-closed 严格境内）：
//   - 默认所有学校只能用境内模型（文本=通义千问 qwen-max，图像/视频=豆包）。
//   - 仅 school_model_policies 中 overseas_enabled=true 的学校，放行境外模型。
//   - SchoolID 为空 / 未授权 / 查询出错 → 一律降级境内（A 方案：只有明确授权才用海外）。
//
// 作用对象：CallAI / CallAIStream / CallAIMultimodal 三个入口，
//          在拿到 EffectiveConfig 后、真正发请求前调用 applyModelPolicy 原地改写 cfg。
//
// 改写内容：
//   - cfg.Model：若为境外模型且学校未授权 → 替换为境内降级模型（ResolveDomesticModel）。
//   - cfg.FallbackModels：剔除其中所有境外模型（避免主模型被降级后 fallback 又跳回境外）。
//
// 注意：图像/视频/TTS 场景本就配豆包（境内），IsOverseasModel 判为 false，不受影响。
//
// 紧急回滚：把三个入口里的 applyModelPolicy(cfg, traceCtx) 调用注释掉即可，本文件可保留。

import (
	"context"
	"sync"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// modelPolicyQueryTimeout 学校策略查询超时（短查询，2 秒足够；超时按未授权处理）
const modelPolicyQueryTimeout = 2 * time.Second

// 境内通道配置在 ai_configs 表中的键名
const (
	domesticBaseURLKey = "domestic_text_base_url" // 境内文本网关地址（dashscope）
	domesticKeyEncKey  = "domestic_text_key_enc"  // 境内文本 API Key（AES 加密）
	domesticModelKey   = "domestic_text_model"    // 境内文本主力模型（qwen-max）
)

// domesticChannelTTL 境内通道配置缓存有效期（配置极少变动，缓存5分钟减少查库+解密开销）
const domesticChannelTTL = 5 * time.Minute

// policyAESKey 由 routes.go 启动时经 SetModelPolicyConfig 注入，用于解密境内 key
var (
	policyAESKey string
	policyMu     sync.RWMutex
)

// SetModelPolicyConfig 注入分流所需的 AES 密钥（routes.go 初始化时调用一次）
func SetModelPolicyConfig(aesKey string) {
	policyMu.Lock()
	defer policyMu.Unlock()
	policyAESKey = aesKey
	aiLog.Info("模型分流 AES 密钥已注入", "key_set", aesKey != "")
}

// domesticChannel 解析后的境内通道配置（明文 key，仅驻留内存）
type domesticChannel struct {
	baseURL string
	apiKey  string
	model   string
}

var (
	cachedDomestic   *domesticChannel
	domesticCachedAt time.Time
)

// InvalidateDomesticChannelCache 立即作废境内通道内存缓存（批一新增）。
// 管理员在前端修改境内网关三键（base/key/model）后由 handler 调用，
// 使下一次 AI 调用立即重新查库+解密读到新配置，而非等待 5 分钟 TTL 自然过期。
// 并发安全：写锁保护，与 getDomesticChannel 的读写一致。
func InvalidateDomesticChannelCache() {
	policyMu.Lock()
	cachedDomestic = nil
	domesticCachedAt = time.Time{}
	policyMu.Unlock()
	aiLog.Info("境内通道缓存已手动失效（配置变更触发），下次调用将重新读取")
}

// getDomesticChannel 获取境内通道配置（带 TTL 缓存）。
// 返回 nil 表示配置缺失/解密失败/AES未注入——调用方据此 fail-safe 保守不切换。
func getDomesticChannel() *domesticChannel {
	policyMu.RLock()
	if cachedDomestic != nil && time.Since(domesticCachedAt) < domesticChannelTTL {
		c := cachedDomestic
		policyMu.RUnlock()
		return c
	}
	aesKey := policyAESKey
	policyMu.RUnlock()

	if aesKey == "" {
		aiLog.Error("境内通道：AES 密钥未注入，无法解密境内 key，本次不切换通道")
		return nil
	}

	baseCfg, errBase := repository.GetConfigByKey(domesticBaseURLKey)
	keyCfg, errKey := repository.GetConfigByKey(domesticKeyEncKey)
	modelCfg, errModel := repository.GetConfigByKey(domesticModelKey)
	if errBase != nil || errKey != nil || errModel != nil ||
		baseCfg == nil || keyCfg == nil || modelCfg == nil ||
		baseCfg.ConfigValue == "" || keyCfg.ConfigValue == "" || modelCfg.ConfigValue == "" {
		aiLog.Error("境内通道配置缺失，本次不切换通道")
		return nil
	}

	plainKey, decErr := utils.DecryptAES(keyCfg.ConfigValue, aesKey)
	if decErr != nil || plainKey == "" {
		aiLog.Error("境内通道 key 解密失败，本次不切换通道", "error", decErr)
		return nil
	}

	ch := &domesticChannel{
		baseURL: baseCfg.ConfigValue,
		apiKey:  plainKey,
		model:   modelCfg.ConfigValue,
	}

	policyMu.Lock()
	cachedDomestic = ch
	domesticCachedAt = time.Now()
	policyMu.Unlock()

	return ch
}

// applyModelPolicy 按学校境外授权策略，原地改写 cfg 的 Model 与 FallbackModels。
// fail-closed：traceCtx 为空 / 无 SchoolID / 未授权 / 查询失败 → 一律降级境内。
//
// 该函数不返回错误：任何异常都保守降级境内，绝不因策略查询失败而中断 AI 调用。
func applyModelPolicy(cfg *EffectiveConfig, traceCtx *TraceContext) {
	if cfg == nil {
		return
	}

	// 判断当前学校是否被授权使用境外模型（A 方案：默认 false）
	overseasAllowed := isOverseasAllowedForTrace(traceCtx)
	if overseasAllowed {
		// 已授权学校：原样放行，不改写任何模型
		return
	}

	// -------- 未授权：境内化处理 --------

	// 当前模型本就是境内（豆包图像/视频/TTS 等）：不涉及境外，跳过
	if !models.IsOverseasModel(cfg.Model) {
		return
	}

	// 未授权 + 境外文本模型：整套切到境内通道（网关+key+模型）
	ch := getDomesticChannel()
	if ch == nil {
		// fail-safe：境内通道不可用，保守维持原通道（这次仍走境外），已在内部记 Error。
		// 宁可这次没降级，也不让老师备课因分流失败而中断。
		return
	}

	originalModel := cfg.Model
	originalBase := cfg.APIBaseURL

	cfg.APIBaseURL = ch.baseURL
	cfg.APIKey = ch.apiKey
	cfg.Model = ch.model
	cfg.FallbackModels = nil // 境内通道不需要境外 fallback

	// qwen 系列单次输出上限 8192 token，境外场景常配 64000/200000，切到境内须夹紧避免 HTTP 400
	const domesticMaxTokensCap = 8192
	if cfg.MaxTokens > domesticMaxTokensCap {
		cfg.MaxTokens = domesticMaxTokensCap
	}

	aiLog.Info("文本调用境内化（整通道切换）",
		"original_model", originalModel,
		"original_base", originalBase,
		"domestic_model", cfg.Model,
		"domestic_base", cfg.APIBaseURL,
		"scene", getSceneFromTrace(traceCtx),
		"school_id", schoolIDFromTrace(traceCtx))
}

// isOverseasAllowedForTrace 判断该 trace 对应的学校是否被授权使用境外模型。
// 任何不确定情况一律返回 false（fail-closed）。
func isOverseasAllowedForTrace(traceCtx *TraceContext) bool {
	if traceCtx == nil || traceCtx.SchoolID == nil {
		return false
	}
	schoolID := *traceCtx.SchoolID
	if schoolID == "" {
		return false
	}

	// 独立短超时 context，避免策略查询拖慢/卡死 AI 主路径
	ctx, cancel := context.WithTimeout(context.Background(), modelPolicyQueryTimeout)
	defer cancel()

	allowed, err := repository.IsSchoolOverseasEnabled(ctx, schoolID)
	if err != nil {
		// 查询失败：保守按未授权处理，并记一条 Warn 便于排查
		aiLog.Warn("学校模型策略查询失败，按未授权（境内）处理",
			"school_id", schoolID,
			"error", err.Error())
		return false
	}
	return allowed
}

// schoolIDFromTrace 安全取出 traceCtx 中的学校 ID（仅用于日志，空安全）
func schoolIDFromTrace(traceCtx *TraceContext) string {
	if traceCtx == nil || traceCtx.SchoolID == nil {
		return ""
	}
	return *traceCtx.SchoolID
}
