package services

// kb_compress_engine.go — 知识库压缩-仲裁引擎（专利独权1 在压缩场景的实施）
//
// 职责（纯计算单元，不碰存储/DB/SSE）：
//   - CompressOneItem    对一段原文跑 N 轮独立压缩，产出多轮草稿（draft_rounds）
//   - ArbitrateConsistency 对多轮草稿做语义一致性仲裁，产出 confidence + arbitration
//
// 设计照搬 pipeline_eval_meta.go 的多轮盲审+仲裁范式：
//   - 多轮独立：for 循环逐轮独立 CallAI，各轮不共享中间态，失败 continue 累计；
//   - 仲裁：把多轮结果拼成文本喂仲裁 AI，单次调用解析结论；
//   - token/错误：每轮累计 tokens、记录 last_error，供成本追溯与失败重试。
//
// 与 Pipeline 评估的差异：
//   - 评估是"给索引打分"，本引擎是"把原文压缩成索引"；
//   - 仲裁判据从"评分阈值"改为"语义一致性"——语义/逻辑一致即高置信(取最优轮)，
//     关键事实矛盾(符号码不同/关键数量关系不同/知识点归属矛盾)即低置信(进人工)。
//
// 模型档：压缩用场景码 kb_compress、仲裁用 kb_arbitrate（默认都配 opus，
//   未在 ai_scene_configs 实配则回退全局默认，与课件 courseware_media_prompt 同机制）。
//
// fail-safe：仲裁解析失败一律保守判低置信(进人工)，绝不静默当高置信放过。

import (
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// ==================== 场景码常量 ====================

const (
	// SceneKBCompress 多轮压缩场景码（默认配 opus，未实配回退全局默认）
	SceneKBCompress = "kb_compress"
	// SceneKBArbitrate 语义一致性仲裁场景码（默认配 opus）
	SceneKBArbitrate = "kb_arbitrate"
)

// ==================== 压缩输入/输出结构 ====================

// KBCompressInput 单个单元的压缩输入（引擎入参，不含任何存储概念）
type KBCompressInput struct {
	Kind           string  // curriculum / textbook（决定用哪套压缩提示词）
	SourceText     string  // 待压缩的原文片段（课标条目原文）
	ImageDataURI   string  // 可选：该单元配图 dataURI/公网URL（精准模式多模态读图，课标先行通常为空）
	CompressPrompt string  // 压缩系统提示词（由调用方从 prompt_curriculum_index 取出传入）
	Rounds         int     // 压缩轮数（<=0 兜底为 KBDefaultRounds）
	UserID         *string // 调用人（trace 埋点用，可空）
	SchoolID       *string // 学校（积分策略，可空）
}

// KBCompressOutput 单个单元的压缩输出（引擎产物）
type KBCompressOutput struct {
	Rounds      []models.KBDraftRound // 多轮草稿（含成功与失败轮，失败轮 Error 非空）
	DoneCount   int                   // 成功轮数
	FailCount   int                   // 失败轮数
	TokensTotal int64                 // 累计 token 消耗
	LastError   string                // 最后一次失败原因（全成功为空）
}

// ==================== 多轮独立压缩 ====================

// CompressOneItem 对一个单元跑 N 轮独立压缩。
// 各轮独立 CallAI、不共享中间态，全部留痕（成功记 Line，失败记 Error）。
// 即便部分轮失败也返回已成功的轮；全失败时 DoneCount=0、LastError 非空。
// 本函数不报 error（除非配置都取不到），引擎层错误通过 Output.LastError 上报。
func (e *KBCompressEngine) CompressOneItem(input KBCompressInput) (*KBCompressOutput, error) {
	rounds := input.Rounds
	if rounds <= 0 {
		rounds = models.KBDefaultRounds
	}

	// 取压缩场景 AI 配置（kb_compress 未实配→回退全局默认）
	aiCfg, err := ai.GetEffectiveConfig(
		e.aesKey, SceneKBCompress,
		e.fallbackBaseURL, e.fallbackKey, e.fallbackModel,
	)
	if err != nil {
		return nil, fmt.Errorf("压缩引擎: 获取AI配置失败: %w", err)
	}

	// 构建用户提示词：原文 + （可选）已有上下文。压缩提示词作 systemPrompt。
	userPrompt := buildKBCompressUserPrompt(input.SourceText)

	out := &KBCompressOutput{Rounds: make([]models.KBDraftRound, 0, rounds)}

	for i := 1; i <= rounds; i++ {
		traceCtx := &ai.TraceContext{
			SceneCode: SceneKBCompress,
			UserID:    input.UserID,
			SchoolID:  input.SchoolID,
		}

		var callResult *ai.CallResult
		var callErr error
		if strings.TrimSpace(input.ImageDataURI) != "" {
			// 精准模式：多模态读图压缩（课标先行通常走文本分支，此处为教材子系统预留）
			callResult, callErr = ai.CallAIMultimodal(
				aiCfg, input.CompressPrompt, userPrompt, input.ImageDataURI, traceCtx,
			)
		} else {
			// 急速/文本模式：纯文本压缩
			callResult, callErr = ai.CallAI(
				aiCfg, input.CompressPrompt, userPrompt, traceCtx,
			)
		}

		if callErr != nil {
			// 本轮失败：留痕，累计失败数，记 LastError，继续下一轮
			out.Rounds = append(out.Rounds, models.KBDraftRound{
				Round: i,
				Model: aiCfg.Model,
				At:    time.Now().Format(time.RFC3339),
				Error: callErr.Error(),
			})
			out.FailCount++
			out.LastError = callErr.Error()
			continue
		}

		// 本轮成功：清洗输出为单行索引（去围栏/多余空行，取首个 KP 行起的有效内容）
		line := cleanKBIndexLine(callResult.Content)
		out.Rounds = append(out.Rounds, models.KBDraftRound{
			Round:  i,
			Line:   line,
			Model:  callResult.ModelUsed,
			Tokens: callResult.TokensUsed,
			At:     time.Now().Format(time.RFC3339),
		})
		out.DoneCount++
		out.TokensTotal += int64(callResult.TokensUsed)
	}

	return out, nil
}

// ==================== 语义一致性仲裁 ====================

// ArbitrateConsistency 对多轮草稿做语义一致性仲裁，产出置信分级与仲裁结论。
//
// 判据（PRD §3.3）：
//   - 高置信(consistent=true)：多轮表达同一含义即视为一致，允许措辞丰富不同，
//     取表达最完整的一轮作 chosen_round；
//   - 低置信(consistent=false)：关键事实矛盾才算冲突——符号码不同(DP:2 vs DP:3)、
//     关键数量关系不同、知识点归属/课标映射矛盾、逻辑链方向相反等，conflicts 列出冲突点。
//
// 仅成功轮参与仲裁。成功轮 0 → 直接低置信(无可仲裁)。成功轮 1 → 直接高置信(无冲突可言)。
// 成功轮 >=2 → 调仲裁 AI 判定。
//
// fail-safe：仲裁 AI 失败或解析失败 → 保守判低置信(进人工)，绝不静默判高置信。
func (e *KBCompressEngine) ArbitrateConsistency(
	out *KBCompressOutput, userID, schoolID *string,
) (confidence string, arb *models.KBArbitration) {
	// 收集成功轮（Error 为空且 Line 非空）
	var doneRounds []models.KBDraftRound
	for _, r := range out.Rounds {
		if r.Error == "" && strings.TrimSpace(r.Line) != "" {
			doneRounds = append(doneRounds, r)
		}
	}

	// 成功轮 0：无可仲裁，低置信进人工
	if len(doneRounds) == 0 {
		return models.KBConfidenceLow, &models.KBArbitration{
			Consistent:  false,
			Conflicts:   []string{"所有压缩轮均失败，无可用草稿，需人工处理"},
			ChosenRound: 0,
			Reason:      "无成功压缩轮",
		}
	}

	// 成功轮 1：唯一一版，无冲突可言，高置信取该轮
	if len(doneRounds) == 1 {
		return models.KBConfidenceHigh, &models.KBArbitration{
			Consistent:  true,
			Conflicts:   []string{},
			ChosenRound: doneRounds[0].Round,
			Reason:      "仅一轮成功，直接采纳",
		}
	}

	// 成功轮 >=2：调仲裁 AI 判定语义一致性
	aiCfg, err := ai.GetEffectiveConfig(
		e.aesKey, SceneKBArbitrate,
		e.fallbackBaseURL, e.fallbackKey, e.fallbackModel,
	)
	if err != nil {
		// 取配置失败 → 保守低置信
		return models.KBConfidenceLow, &models.KBArbitration{
			Consistent:  false,
			Conflicts:   []string{"仲裁AI配置获取失败，转人工: " + err.Error()},
			ChosenRound: doneRounds[0].Round,
			Reason:      "仲裁配置失败保守判低置信",
		}
	}

	systemPrompt := kbArbitrateSystemPrompt
	userPrompt := buildKBArbitrateUserPrompt(doneRounds)

	traceCtx := &ai.TraceContext{
		SceneCode: SceneKBArbitrate,
		UserID:    userID,
		SchoolID:  schoolID,
	}
	callResult, callErr := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if callErr != nil {
		// 仲裁调用失败 → 保守低置信
		return models.KBConfidenceLow, &models.KBArbitration{
			Consistent:  false,
			Conflicts:   []string{"仲裁AI调用失败，转人工: " + callErr.Error()},
			ChosenRound: doneRounds[0].Round,
			Reason:      "仲裁调用失败保守判低置信",
		}
	}

	// 解析仲裁 JSON（ExtractJSON 先提对象，失败保守低置信）
	parsed := parseKBArbitration(callResult.Content)
	if parsed == nil {
		return models.KBConfidenceLow, &models.KBArbitration{
			Consistent:  false,
			Conflicts:   []string{"仲裁输出解析失败，转人工"},
			ChosenRound: doneRounds[0].Round,
			Reason:      "仲裁解析失败保守判低置信",
			Model:       callResult.ModelUsed,
			Tokens:      callResult.TokensUsed,
		}
	}

	parsed.Model = callResult.ModelUsed
	parsed.Tokens = callResult.TokensUsed

	// 校正 chosen_round：必须落在成功轮里，否则取第一成功轮兜底
	if !isRoundInDone(parsed.ChosenRound, doneRounds) {
		parsed.ChosenRound = doneRounds[0].Round
	}

	if parsed.Consistent {
		return models.KBConfidenceHigh, parsed
	}
	return models.KBConfidenceLow, parsed
}

// ==================== 引擎实例 ====================

// KBCompressEngine 压缩-仲裁引擎实例（持有 AI 配置回退参数）
// 不持有 DB/repo，纯计算单元；存储编排在 kb_compress_service.go 中完成。
type KBCompressEngine struct {
	aesKey          string
	fallbackBaseURL string
	fallbackKey     string
	fallbackModel   string
}

// NewKBCompressEngine 构造引擎（回退参数来自 config，与 Pipeline 取配置同源）
func NewKBCompressEngine(aesKey, fallbackBaseURL, fallbackKey, fallbackModel string) *KBCompressEngine {
	return &KBCompressEngine{
		aesKey:          aesKey,
		fallbackBaseURL: fallbackBaseURL,
		fallbackKey:     fallbackKey,
		fallbackModel:   fallbackModel,
	}
}

// ==================== 提示词与解析辅助 ====================

// kbArbitrateSystemPrompt 语义一致性仲裁系统提示词（内联，因仲裁逻辑通用、无需入库版本管理）
const kbArbitrateSystemPrompt = `你是TE-DNA知识库压缩的语义一致性仲裁专家。任务：判定同一条原文经多轮独立压缩产出的多个一行制索引是否语义一致、逻辑一致。

判定标准：
1. 判一致：多版表达同一含义即视为一致，允许措辞丰富不同。例如"能用毫米测量"与"会用毫米测量较短物体并准确读数"视为一致。
2. 判冲突：仅当存在关键事实矛盾才算冲突——
   - 符号码不同（如 DP:2 vs DP:3 深度档不同、SJ 学科码不同、GR 年级不同）
   - 关键数量关系不同（如换算关系、误差标准、正确率标准矛盾）
   - 知识点归属/领域/课标映射矛盾（kp_code 或 DM 领域指向不同知识点）
   - 逻辑链方向相反或核心结论矛盾

输出要求：严格输出一个JSON对象，不要任何解释文字或Markdown围栏：
{
  "consistent": true 或 false,
  "conflicts": ["冲突点1的中文描述", "冲突点2"],
  "chosen_round": 取表达最完整准确的那一轮的轮次序号(整数),
  "reason": "仲裁理由的简短中文说明"
}

注意：
- consistent=true 时 conflicts 为空数组 []，chosen_round 指向最完整的一轮。
- consistent=false 时 conflicts 必须具体列出矛盾点（指明哪个符号码或哪个事实矛盾），chosen_round 仍给出你认为相对最优的一轮供人工参考。
- 只判定语义与关键事实，不挑剔措辞风格差异。`

// buildKBCompressUserPrompt 构建压缩用户提示词（原文喂入，压缩提示词作 system）
func buildKBCompressUserPrompt(sourceText string) string {
	return strings.Join([]string{
		"【待压缩课标原文】",
		strings.TrimSpace(sourceText),
		"",
		"请严格按系统提示词的格式，将以上原文压缩为一行制KP索引。只输出索引行本身，不要解释。",
	}, "\n")
}

// buildKBArbitrateUserPrompt 把多轮成功草稿拼成仲裁用户提示词
func buildKBArbitrateUserPrompt(doneRounds []models.KBDraftRound) string {
	parts := []string{"【多轮独立压缩产出的索引（需判定是否语义一致）】", ""}
	for _, r := range doneRounds {
		parts = append(parts, fmt.Sprintf("=== 第%d轮 ===", r.Round))
		parts = append(parts, r.Line, "")
	}
	parts = append(parts, "请判定以上各版是否语义/逻辑一致，按系统提示词要求输出JSON。")
	return strings.Join(parts, "\n")
}

// parseKBArbitration 从仲裁 AI 输出解析 KBArbitration（多级兜底，失败返 nil）
func parseKBArbitration(aiOutput string) *models.KBArbitration {
	// 第一级：ExtractJSON 提取首个完整 JSON 对象
	if jsonStr, ok := ai.ExtractJSON(aiOutput); ok {
		if arb := models.ParseArbitration(jsonStr); arb != nil {
			return arb
		}
	}
	// 第二级：直接对整段 TrimSpace 后尝试解析（AI 可能裸输出 JSON 无围栏）
	if arb := models.ParseArbitration(strings.TrimSpace(aiOutput)); arb != nil {
		return arb
	}
	return nil
}

// cleanKBIndexLine 清洗压缩 AI 输出为单行索引
// 去 Markdown 围栏、取首个含 KP 标识的有效行起的内容、折叠多余空白
func cleanKBIndexLine(raw string) string {
	s := strings.TrimSpace(raw)
	// 去 ```...``` 围栏
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	// 逐行扫描，取首个以 "KP " 开头的行（容错：AI 可能加前导说明）
	lines := strings.Split(s, "\n")
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "KP ") {
			return t
		}
	}
	// 没找到 KP 行，返回整段去空白（让仲裁/审核去发现异常）
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}

// isRoundInDone 判断轮次序号是否在成功轮集合内
func isRoundInDone(round int, doneRounds []models.KBDraftRound) bool {
	for _, r := range doneRounds {
		if r.Round == round {
			return true
		}
	}
	return false
}
