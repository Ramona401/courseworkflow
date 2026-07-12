package services

// courseware_lesson_normalize_service.go — 教案预处理规整层核心服务
//
// 【解决什么】课件生成是"教案原文 → 层2方案(8字段) → 逐页HTML"两段式。方案是对教案的
//   有损压缩：教案里"6个点子""剥洋葱各层"等具体案例被压成抽象描述，逐页生成时 AI 看不到
//   原文只能各页现编 → 还原度低、跨页案例对不上。
//   本服务在链路最前端加一道 AI 规整：把又长又乱、排版不规范的教案原文，规整成"统一格式、
//   去噪保核、预置清单一字不差"的干净教案，存库供后续逐页注入。
//
// 【核心设计原则（从踩坑中提炼，务必遵守）】
//   1. 规整极度克制、严防二次失真：预置清单类内容(点子/关卡/题目/选项/答案等)一字不改，
//      只允许重组结构、删噪音。此约束写在 prompt_courseware_lesson_normalize 里。
//   2. 只调一次 AI（教案级，非页级）：一次规整存库，后续逐页注入零 AI 成本。
//   3. 全程 best-effort 兜底：取原文失败/AI报错/输出异常，一律标记 failed 并返回，
//      绝不 panic、绝不阻断建索引与生成——注入层遇无规整结果自动退回原文。
//
// 【为什么取全文而非复用 loadLessonPlanContextForGen】
//   后者返回值被 cwLessonFullContextMaxRunes=8000 截断，而规整恰恰要吃完整原文
//   （真实教案 2.3 万字，8000 会截掉大半）。故本服务自己取全文、不做 8000 截断。
//
// 【为什么直连网关而非 ai.CallAI】
//   CallAI 内部 applyModelPolicy 未填 SchoolID 时会 fail-closed 降级境内 qwen
//   （长文本规整能力弱）。本服务用 GetEffectiveConfig 拿解密配置、直连网关、强制用
//   courseware_lesson_normalize 场景配置的 gemini，规整质量最纯净。
//
// 【触发时机】建索引时顺带触发（courseware_index_service 挂钩），best-effort 不阻断建索引。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwNormalizeLog 模块级结构化日志器
var cwNormalizeLog = logger.WithModule("courseware.lesson_normalize")

// ==================== 常量 ====================

const (
	// cwNormalizeScene 规整 AI 场景码（对应 ai_scene_configs 里已配的 gemini-3.1-pro-preview）
	cwNormalizeScene = "courseware_lesson_normalize"

	// cwNormalizeModel 规整模型（强制指定，避开 CallAI 分流降级到境内 qwen）
	cwNormalizeModel = "google/gemini-3.1-pro-preview"

	// cwNormalizeFallbackModel 规整降级模型（主模型不可用时直连尝试一次）
	cwNormalizeFallbackModel = "anthropic/claude-sonnet-4.6"

	// cwNormalizeMaxTokens 规整输出上限（规整后比原文短，留足空间）
	cwNormalizeMaxTokens = 32000

	// cwNormalizeTemperature 低温，减少改写倾向，贴合"整理不创作"
	cwNormalizeTemperature = 0.2

	// cwNormalizeHTTPTimeout 规整调用超时（长文本，给足时间）
	cwNormalizeHTTPTimeout = 600 * time.Second

	// cwNormalizeMinRawRunes 原文过短阈值：短于此长度无需规整（本就干净），跳过
	cwNormalizeMinRawRunes = 300

	// cwNormalizePromptKey 规整指令的 prompt_key（存 prompts 表，改库实时生效）
	cwNormalizePromptKey = "prompt_courseware_lesson_normalize"
)

// ==================== 服务定义 ====================

// CoursewareLessonNormalizeService 教案规整服务
//   持 cfg 以取 AES 密钥(解密网关Key)与 .env 兜底网关配置。
type CoursewareLessonNormalizeService struct {
	cfg *config.Config
}

// NewCoursewareLessonNormalizeService 构造规整服务
func NewCoursewareLessonNormalizeService(cfg *config.Config) *CoursewareLessonNormalizeService {
	return &CoursewareLessonNormalizeService{cfg: cfg}
}

// ==================== 对外入口 ====================

// EnsureNormalized 确保课件有可用的规整结果（若已 done 则跳过，否则触发规整）。
//   供建索引流程调用。同步执行（调用方决定是否放 goroutine）。
//   返回 error 仅用于调用方日志观测；即便返回 error 也绝不应阻断建索引/生成。
//
// 幂等：已有 done 记录且正文非空时直接返回，不重复调 AI（省成本）。
func (s *CoursewareLessonNormalizeService) EnsureNormalized(ctx context.Context, cw *models.Courseware) error {
	if cw == nil {
		return fmt.Errorf("课件为空，跳过规整")
	}

	// 幂等短路：已 done 且正文非空 → 复用，不重复规整
	existing, _ := repository.GetNormalizedByCoursewareID(ctx, cw.ID)
	if existing != nil && existing.HasUsableContent() {
		cwNormalizeLog.Info("规整结果已存在，跳过",
			"courseware_id", cw.ID, "norm_chars", existing.NormCharCount)
		return nil
	}

	return s.RunNormalize(ctx, cw)
}

// RunNormalize 强制执行一次规整（不做幂等短路，供"重新规整"或首次触发用）。
//   全程 best-effort：任何一步失败都写 failed 记录并返回 error（供观测），绝不 panic。
func (s *CoursewareLessonNormalizeService) RunNormalize(ctx context.Context, cw *models.Courseware) error {
	if cw == nil {
		return fmt.Errorf("课件为空，跳过规整")
	}

	// -------- 步骤1：取教案原文全文（不做 8000 截断，规整要吃完整原文）--------
	rawContent, sourceType, sourceRef, err := s.loadFullRawContent(ctx, cw)
	if err != nil {
		// 非教案/文档来源，或取数失败：不产生失败记录（本就不该规整），静默跳过
		cwNormalizeLog.Info("课件来源无可规整原文，跳过规整",
			"courseware_id", cw.ID, "source_type", cw.SourceType, "reason", err.Error())
		return nil
	}

	rawRunes := len([]rune(rawContent))

	// 原文过短（本就干净，无需规整）：直接把原文当规整结果落库，省一次 AI 调用
	if rawRunes < cwNormalizeMinRawRunes {
		cwNormalizeLog.Info("教案原文过短，直接沿用原文作规整结果",
			"courseware_id", cw.ID, "raw_runes", rawRunes)
		if e := repository.UpsertDoneNormalized(ctx, cw.ID, sourceType, sourceRef,
			rawContent, "raw_passthrough", 0, rawRunes, rawRunes); e != nil {
			cwNormalizeLog.Warn("短原文直存规整结果失败", "courseware_id", cw.ID, "error", e)
		}
		return nil
	}

	// -------- 步骤2：落 generating 占位（便于排查/看进度）--------
	if e := repository.UpsertGeneratingNormalized(ctx, cw.ID, sourceType, sourceRef, rawRunes); e != nil {
		cwNormalizeLog.Warn("写规整占位记录失败（不阻断）", "courseware_id", cw.ID, "error", e)
	}

	// -------- 步骤3：调 AI 规整（直连网关，强制 gemini，避开分流降级）--------
	normalized, modelUsed, tokensUsed, err := s.callNormalizeAI(rawContent)
	if err != nil {
		cwNormalizeLog.Warn("规整AI调用失败，标记failed（下游退回原文，不阻断生成）",
			"courseware_id", cw.ID, "error", err)
		_ = repository.MarkNormalizedFailed(ctx, cw.ID, sourceType, sourceRef, err.Error(), rawRunes)
		return fmt.Errorf("规整AI调用失败: %w", err)
	}

	normalized = strings.TrimSpace(normalized)
	normRunes := len([]rune(normalized))

	// -------- 步骤4：输出合理性校验（防AI返回空/异常短，那样反而不如用原文）--------
	//   规整结果不应短于原文的 5%（正常压缩到 15%~30%）；过短视为异常，标记 failed 退回原文。
	if normRunes < rawRunes/20 || normRunes < 200 {
		reason := fmt.Sprintf("规整输出异常短(原文%d字→规整%d字)，判为失败退回原文", rawRunes, normRunes)
		cwNormalizeLog.Warn(reason, "courseware_id", cw.ID)
		_ = repository.MarkNormalizedFailed(ctx, cw.ID, sourceType, sourceRef, reason, rawRunes)
		return fmt.Errorf(reason)
	}

	// -------- 步骤5：写成功结果 --------
	if e := repository.UpsertDoneNormalized(ctx, cw.ID, sourceType, sourceRef,
		normalized, modelUsed, tokensUsed, rawRunes, normRunes); e != nil {
		cwNormalizeLog.Warn("写规整成功结果失败", "courseware_id", cw.ID, "error", e)
		return fmt.Errorf("写规整成功结果失败: %w", e)
	}

	cwNormalizeLog.Info("教案规整成功",
		"courseware_id", cw.ID, "model", modelUsed,
		"raw_runes", rawRunes, "norm_runes", normRunes,
		"compress_pct", fmt.Sprintf("%.0f%%", float64(normRunes)/float64(rawRunes)*100),
		"tokens", tokensUsed)
	return nil
}

// ==================== 内部：取原文全文（分流，不截断）====================

// loadFullRawContent 取课件对应的教案原文完整全文（不做 8000 截断，供规整吃完整原文）。
//   分流逻辑与 loadLessonPlanContextForGen 一致，但去掉了全文截断：
//     - lesson_plan：查教案表 → ExtractLessonPlanContentForCW 提取正文
//     - doc_upload ：读上传的 docx 全文（readDocxFullText）
//     - 其余来源   ：返回 error（无可靠原文，调用方据此静默跳过）
//
// 返回：(原文全文, 来源类型, 原文出处, error)。error 非nil 表示"该来源不该规整"或取数失败。
func (s *CoursewareLessonNormalizeService) loadFullRawContent(ctx context.Context, cw *models.Courseware) (string, string, string, error) {
	switch cw.SourceType {
	case models.CWSourceLessonPlan:
		if cw.LessonPlanID == nil || strings.TrimSpace(*cw.LessonPlanID) == "" {
			return "", "", "", fmt.Errorf("教案来源但无 lesson_plan_id")
		}
		lp, err := repository.GetLessonPlanByID(ctx, *cw.LessonPlanID)
		if err != nil || lp == nil {
			return "", "", "", fmt.Errorf("查教案失败: %v", err)
		}
		content := strings.TrimSpace(ExtractLessonPlanContentForCW(lp))
		if content == "" {
			return "", "", "", fmt.Errorf("教案正文为空")
		}
		return content, models.CWSourceLessonPlan, *cw.LessonPlanID, nil

	case models.CWSourceDocUpload:
		if strings.TrimSpace(cw.SourceFilePath) == "" {
			return "", "", "", fmt.Errorf("文档来源但无 source_file_path")
		}
		docFullPath := filepath.Join(DocUploadDir, cw.SourceFilePath)
		text, err := readDocxFullText(docFullPath)
		if err != nil || strings.TrimSpace(text) == "" {
			return "", "", "", fmt.Errorf("读docx全文失败: %v", err)
		}
		return strings.TrimSpace(text), models.CWSourceDocUpload, cw.SourceFilePath, nil

	default:
		// ppt_upload / topic_direct / 3d_single / html_import：无可靠完整教案原文
		return "", "", "", fmt.Errorf("来源 %s 无可规整原文", cw.SourceType)
	}
}

// ==================== 内部：调 AI 规整（直连网关，强制 gemini）====================

// callNormalizeAI 用规整 prompt + gemini 对原文规整一次。
//   直连网关（不走 ai.CallAI）以避开 applyModelPolicy 的境内降级；主模型失败则尝试 fallback。
//   返回：(规整正文, 实际模型, 消耗token, error)。
func (s *CoursewareLessonNormalizeService) callNormalizeAI(rawContent string) (string, string, int, error) {
	// 取解密后的网关配置（复用 GetEffectiveConfig 的自动解密；模型下面强制覆盖）
	eff, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		cwNormalizeScene,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", "", 0, fmt.Errorf("取规整网关配置失败: %w", err)
	}
	if strings.TrimSpace(eff.APIKey) == "" || strings.TrimSpace(eff.APIBaseURL) == "" {
		return "", "", 0, fmt.Errorf("规整网关地址或Key为空")
	}

	// 取规整指令（存 prompts 表，改库实时生效；查不到则用内置兜底）
	systemPrompt := s.loadNormalizePrompt()
	userContent := "以下是教案原文，请按上述规则规整：\n\n" + rawContent

	// 主模型先试，失败再试 fallback
	models := []string{cwNormalizeModel, cwNormalizeFallbackModel}
	var lastErr error
	for _, model := range models {
		content, tokens, callErr := s.doNormalizeHTTP(eff.APIBaseURL, eff.APIKey, model, systemPrompt, userContent)
		if callErr == nil {
			return content, model, tokens, nil
		}
		lastErr = callErr
		cwNormalizeLog.Warn("规整模型调用失败，尝试下一个", "model", model, "error", callErr)
	}
	return "", "", 0, fmt.Errorf("所有规整模型均失败: %w", lastErr)
}

// doNormalizeHTTP 执行单次直连网关调用（OpenAI 兼容格式）。
func (s *CoursewareLessonNormalizeService) doNormalizeHTTP(baseURL, apiKey, model, systemPrompt, userContent string) (string, int, error) {
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type chatRequest struct {
		Model       string        `json:"model"`
		Messages    []chatMessage `json:"messages"`
		MaxTokens   int           `json:"max_tokens"`
		Temperature float64       `json:"temperature"`
	}
	type chatResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		MaxTokens:   cwNormalizeMaxTokens,
		Temperature: cwNormalizeTemperature,
	}
	jsonBody, _ := json.Marshal(reqBody)

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", 0, fmt.Errorf("创建规整请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: cwNormalizeHTTPTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("规整网关调用失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len([]rune(snippet)) > 300 {
			snippet = string([]rune(snippet)[:300])
		}
		return "", 0, fmt.Errorf("规整网关返回HTTP %d: %s", resp.StatusCode, snippet)
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", 0, fmt.Errorf("解析规整响应失败: %w", err)
	}
	if len(cr.Choices) == 0 || strings.TrimSpace(cr.Choices[0].Message.Content) == "" {
		return "", 0, fmt.Errorf("规整响应内容为空")
	}
	return cr.Choices[0].Message.Content, cr.Usage.TotalTokens, nil
}

// loadNormalizePrompt 取规整指令：优先从 prompts 表读当前版本，查不到则用内置兜底。
func (s *CoursewareLessonNormalizeService) loadNormalizePrompt() string {
	p, err := repository.GetCurrentPromptByKey(cwNormalizePromptKey)
	if err == nil && p != nil && strings.TrimSpace(p.Content) != "" {
		return p.Content
	}
	cwNormalizeLog.Warn("规整prompt库中未取到，用内置兜底", "key", cwNormalizePromptKey, "error", err)
	// 内置兜底（与入库版核心一致的极简版；正常不会走到这里）
	return "你是教案规整助手。把收到的教案原文规整成结构清晰、去噪保核的干净教案。" +
		"其中所有预置清单类内容（点子/例子/题目/选项/答案/角色台词/剥洋葱各层等具体条目）" +
		"必须一字不差、原样照抄，包括括号里的真伪标注（如（真）（伪）（迷惑）（答案B）），" +
		"绝不改写、精简、翻译、增删、合并。去除设计理念/教学法阐述/交互动画技术细节等噪音。" +
		"每个教学环节之间用空行分隔。直接输出规整后的教案，不要任何解释或开场白。"
}
