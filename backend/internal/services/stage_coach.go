package services

// stage_coach.go — 阶段教练：LLM质量评估 + 停滞检测 + 教练建议
//
// v87新增（规则+LLM混合版本的LLM部分）
// v89-2变更：LLMEvaluateStageQuality传入真实TraceContext
// v89-3拆分：规则引擎部分移至stage_coach_rules.go
//
// 职责：
//   1. LLMEvaluateStageQuality — 调用Haiku评估产出物质量
//   2. DetectStagnation — 检测对话停滞（连续3轮无实质进展）
//   3. GenerateCoachSuggestion — 生成教练引导建议
//   4. LLM评估相关辅助函数（提示词加载/上下文构建/结果解析/降级兜底）
//
// 被引用：
//   services/workshop_stage_service.go — AdvanceStage阶段过渡质量评估
//   services/lesson_plan_gen_service.go — Chat对话停滞检测+建议插入

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var coachLog = logger.WithModule("stage_coach")

// ==================== 常量定义 ====================

// stagnationThreshold 停滞检测阈值：连续多少轮用户消息无实质进展时触发教练建议
const stagnationThreshold = 3

// stageCoachSceneCode 教练评估使用的AI场景代码（对应ai_scene_configs中的stage_coach）
const stageCoachSceneCode = "stage_coach"

// ==================== 结构体定义 ====================

// StageCompletenessResponse 阶段完成度检测结果
type StageCompletenessResponse struct {
	StageCode    string             `json:"stage_code"`    // 阶段代码
	StageName    string             `json:"stage_name"`    // 阶段名称
	Percentage   int                `json:"percentage"`    // 完成度百分比（0-100）
	IsComplete   bool               `json:"is_complete"`   // 是否视为完整（≥80%）
	UserMessages int                `json:"user_messages"` // 当前阶段用户消息数
	CheckedItems []CompletenessItem `json:"checked_items"` // 各检查项详情
	MissingHints []string           `json:"missing_hints"` // 缺失要素的友好提示
}

// CompletenessItem 单个检查项
type CompletenessItem struct {
	Label  string `json:"label"`  // 检查项名称
	Passed bool   `json:"passed"` // 是否通过
	Detail string `json:"detail"` // 详情说明
}

// CoachLLMEvalResult LLM教练评估结果
type CoachLLMEvalResult struct {
	OverallScore int             `json:"overall_score"` // 总体评分（0-100）
	IsQualified  bool            `json:"is_qualified"`  // 是否达到过渡门槛（>=70分）
	Items        []CoachEvalItem `json:"items"`         // 各检查项详情
	Suggestion   string          `json:"suggestion"`    // 一句话改进建议
	Summary      string          `json:"summary"`       // 一句话总结
}

// CoachEvalItem LLM评估的单个检查项
type CoachEvalItem struct {
	Label  string `json:"label"`  // 检查项名称
	Passed bool   `json:"passed"` // 是否通过
	Score  int    `json:"score"`  // 单项评分（0-100）
	Detail string `json:"detail"` // 详情说明
}

// StagnationCheckResult 停滞检测结果
type StagnationCheckResult struct {
	IsStagnant        bool   `json:"is_stagnant"`         // 是否处于停滞状态
	ConsecutiveRounds int    `json:"consecutive_rounds"`   // 连续无进展轮数
	Suggestion        string `json:"suggestion"`           // 教练建议（停滞时非空）
	StageCode         string `json:"stage_code"`           // 当前阶段代码
}

// ==================== 1. LLM质量评估 ====================

// LLMEvaluateStageQuality 使用LLM评估阶段产出物质量
//
// 在阶段过渡时调用（AdvanceStage），用Haiku模型对产出物做快速质量评估。
// 返回评估结果；如果AI调用失败，降级为规则引擎结果（不阻塞流程）。
//
// v89-2变更：传入真实TraceContext，记录教案ID用于成本追踪
func LLMEvaluateStageQuality(ctx context.Context, aesKey string, lessonPlanID string, stageCode string) (*CoachLLMEvalResult, error) {
	// 获取教案信息
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}

	// 获取阶段产出物
	output, _ := repository.GetStageOutput(ctx, lessonPlanID, stageCode)

	// 获取当前阶段的对话消息
	currentMsgs, _ := repository.GetCurrentStageMessages(ctx, lessonPlanID)

	// 构建评估上下文文本
	evalContext := buildEvalContext(lp, stageCode, output, currentMsgs)

	// 如果上下文太短（不足100字），说明几乎没有产出，直接返回低分
	if len([]rune(evalContext)) < 100 {
		coachLog.Info("LLM评估-上下文过短，跳过AI调用",
			"plan_id", lessonPlanID, "stage", stageCode, "context_len", len([]rune(evalContext)))
		return &CoachLLMEvalResult{
			OverallScore: 10,
			IsQualified:  false,
			Items:        []CoachEvalItem{},
			Suggestion:   "当前阶段内容太少，建议继续与AI深入讨论",
			Summary:      "产出不足",
		}, nil
	}

	// 获取教练评估提示词
	promptContent := loadCoachPrompt()

	// 构建用户消息
	userPrompt := fmt.Sprintf("请评估以下【%s】阶段的产出物质量：\n\n%s", stageCodeToName(stageCode), evalContext)

	// 获取AI配置（使用stage_coach场景，Haiku低成本）
	aiCfg, err := aiClient.GetEffectiveConfig(aesKey, stageCoachSceneCode, "", "", "")
	if err != nil {
		coachLog.Warn("LLM评估-获取AI配置失败，降级为规则结果",
			"plan_id", lessonPlanID, "stage", stageCode, "error", err)
		return fallbackToRuleResult(ctx, lessonPlanID, stageCode)
	}

	// v89-2：构建真实TraceContext，记录教案ID和场景
	pid := lessonPlanID
	traceCtx := &aiClient.TraceContext{
		SceneCode:    stageCoachSceneCode,
		LessonPlanID: &pid,
	}

	// 调用AI评估
	result, err := aiClient.CallAI(aiCfg, promptContent, userPrompt, traceCtx)
	if err != nil {
		coachLog.Warn("LLM评估-AI调用失败，降级为规则结果",
			"plan_id", lessonPlanID, "stage", stageCode, "error", err)
		return fallbackToRuleResult(ctx, lessonPlanID, stageCode)
	}

	// 解析AI输出的JSON
	evalResult, parseErr := parseCoachEvalResult(result.Content)
	if parseErr != nil {
		coachLog.Warn("LLM评估-解析AI输出失败，降级为规则结果",
			"plan_id", lessonPlanID, "stage", stageCode, "error", parseErr,
			"raw_content", coachTruncateStr(result.Content, 200))
		return fallbackToRuleResult(ctx, lessonPlanID, stageCode)
	}

	coachLog.Info("LLM评估完成",
		"plan_id", lessonPlanID, "stage", stageCode,
		"score", evalResult.OverallScore, "qualified", evalResult.IsQualified,
		"model", result.ModelUsed, "tokens", result.TokensUsed,
	)

	return evalResult, nil
}

// ==================== 2. 停滞检测 ====================

// DetectStagnation 检测当前阶段对话是否处于停滞状态
//
// 在每轮Chat对话后调用。判断"用户连续多轮对话但阶段产出物无实质进展"的情况。
func DetectStagnation(ctx context.Context, lessonPlanID string, stageCode string) *StagnationCheckResult {
	result := &StagnationCheckResult{
		IsStagnant:        false,
		ConsecutiveRounds: 0,
		StageCode:         stageCode,
	}

	// review阶段是AI自动执行的，不检测停滞
	if stageCode == "review" {
		return result
	}

	// 获取教案信息
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return result
	}

	// 统计当前阶段用户消息数
	userMsgCount := countUserMessagesInStage(ctx, lessonPlanID, stageCode, lp)
	if userMsgCount < stagnationThreshold {
		return result
	}

	// 获取阶段产出物
	output, _ := repository.GetStageOutput(ctx, lessonPlanID, stageCode)

	// 检查产出物是否有实质内容
	hasSubstantialOutput := false
	if output != nil {
		narrative := strings.TrimSpace(output.NarrativeOutput)
		structured := strings.TrimSpace(output.StructuredOutput)

		if len([]rune(narrative)) > 200 {
			hasSubstantialOutput = true
		}
		if structured != "" && structured != "{}" && structured != "[]" {
			hasSubstantialOutput = true
		}
	}

	// 对于write/revise阶段，还要检查教案正文
	if stageCode == "write" || stageCode == "revise" {
		if lp.ContentMarkdown != "" && len([]rune(lp.ContentMarkdown)) > 500 {
			hasSubstantialOutput = true
		}
	}

	// 如果用户消息>=3轮但没有实质产出，判定为停滞
	if !hasSubstantialOutput {
		result.IsStagnant = true
		result.ConsecutiveRounds = userMsgCount
		result.Suggestion = generateStagnationSuggestion(stageCode, userMsgCount)

		coachLog.Info("检测到对话停滞",
			"plan_id", lessonPlanID, "stage", stageCode,
			"user_msgs", userMsgCount, "has_output", hasSubstantialOutput,
		)
	}

	return result
}

// ==================== 3. 教练建议生成 ====================

// GenerateCoachSuggestion 根据停滞检测结果生成教练引导建议消息
func GenerateCoachSuggestion(stagnation *StagnationCheckResult) string {
	if stagnation == nil || !stagnation.IsStagnant {
		return ""
	}
	return stagnation.Suggestion
}

// generateStagnationSuggestion 根据阶段生成停滞引导建议
func generateStagnationSuggestion(stageCode string, userMsgs int) string {
	prefix := fmt.Sprintf("💡 教练提示（已对话%d轮）：", userMsgs)

	switch stageCode {
	case "analyze":
		return prefix + "我注意到我们聊了好几轮，但教学分析还不够完整。要不我们先从明确教学目标开始？试试告诉AI：「我这节课希望学生能够……」"
	case "design":
		return prefix + "看起来教学设计还需要更多细节。建议你试着告诉AI具体的教学环节安排，比如：「导入环节我想用一个5分钟的小游戏……」"
	case "write":
		return prefix + "教案撰写似乎还没有实质进展。你可以试着让AI帮你生成教案初稿，比如说：「请根据之前的设计方案，帮我写出完整教案」"
	case "revise":
		return prefix + "修订阶段需要你针对评审建议做出回应。试试告诉AI你想修改哪些部分，比如：「我想重点改进教学活动的设计」"
	default:
		return prefix + "建议你更具体地描述你的想法和需求，这样AI能更好地帮助你完成这个阶段的任务。"
	}
}

// ==================== LLM评估内部辅助函数 ====================

// loadCoachPrompt 从prompts表加载教练评估提示词
func loadCoachPrompt() string {
	prompt, err := repository.GetCurrentPromptByKey("prompt_stage_coach")
	if err != nil || prompt == nil || strings.TrimSpace(prompt.Content) == "" {
		coachLog.Warn("加载教练提示词失败，使用内置默认", "error", err)
		return defaultCoachSystemPrompt()
	}
	return prompt.Content
}

// defaultCoachSystemPrompt 内置默认教练系统提示词（prompts表加载失败时的兜底）
func defaultCoachSystemPrompt() string {
	return `你是一个教案备课质量教练。评估教师在当前备课阶段的产出物质量。
输出严格JSON格式：{"overall_score":75,"is_qualified":false,"items":[{"label":"检查项","passed":true,"score":80,"detail":"说明"}],"suggestion":"改进建议","summary":"总结"}`
}

// buildEvalContext 构建LLM评估的上下文文本
func buildEvalContext(lp *models.LessonPlan, stageCode string, output *models.WorkshopStageOutput, msgs []*models.ConversationMessage) string {
	var sb strings.Builder

	// 基本信息
	sb.WriteString(fmt.Sprintf("学科：%s\n年级：%s\n课题：%s\n课时：%d分钟\n阶段：%s\n\n",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes, stageCodeToName(stageCode)))

	// 阶段产出物
	if output != nil {
		if narrative := strings.TrimSpace(output.NarrativeOutput); narrative != "" {
			sb.WriteString("== 阶段产出摘要 ==\n")
			sb.WriteString(coachTruncateStr(narrative, 1500))
			sb.WriteString("\n\n")
		}
		if structured := strings.TrimSpace(output.StructuredOutput); structured != "" && structured != "{}" {
			sb.WriteString("== 结构化产出 ==\n")
			sb.WriteString(coachTruncateStr(structured, 1000))
			sb.WriteString("\n\n")
		}
	}

	// write/revise阶段额外加入教案正文摘要
	if (stageCode == "write" || stageCode == "revise") && lp.ContentMarkdown != "" {
		sb.WriteString("== 教案正文（截取）==\n")
		sb.WriteString(coachTruncateStr(lp.ContentMarkdown, 2000))
		sb.WriteString("\n\n")
	}

	// 最近对话摘要（最多取最后5轮用户+AI对话）
	if len(msgs) > 0 {
		sb.WriteString("== 最近对话 ==\n")
		start := 0
		if len(msgs) > 10 {
			start = len(msgs) - 10
		}
		for i := start; i < len(msgs); i++ {
			msg := msgs[i]
			if string(msg.Role) == "system" {
				continue
			}
			roleLabel := "教师"
			if string(msg.Role) == "assistant" {
				roleLabel = "AI"
			}
			content := coachTruncateStr(msg.Content, 300)
			sb.WriteString(fmt.Sprintf("[%s] %s\n", roleLabel, content))
		}
	}

	return sb.String()
}

// parseCoachEvalResult 解析AI输出的教练评估JSON
func parseCoachEvalResult(content string) (*CoachLLMEvalResult, error) {
	jsonStr, ok := aiClient.ExtractJSON(content)
	if !ok {
		return nil, fmt.Errorf("AI输出中未找到有效JSON")
	}

	var result CoachLLMEvalResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if result.OverallScore < 0 || result.OverallScore > 100 {
		result.OverallScore = 50
	}

	return &result, nil
}

// fallbackToRuleResult LLM评估失败时，降级为规则引擎结果
func fallbackToRuleResult(ctx context.Context, lessonPlanID string, stageCode string) (*CoachLLMEvalResult, error) {
	ruleResult, err := CheckStageCompleteness(ctx, lessonPlanID, stageCode)
	if err != nil {
		return nil, err
	}

	var items []CoachEvalItem
	for _, item := range ruleResult.CheckedItems {
		score := 0
		if item.Passed {
			score = 80
		}
		items = append(items, CoachEvalItem{
			Label:  item.Label,
			Passed: item.Passed,
			Score:  score,
			Detail: item.Detail,
		})
	}

	suggestion := "继续与AI深入讨论，完善当前阶段的内容"
	if len(ruleResult.MissingHints) > 0 {
		suggestion = ruleResult.MissingHints[0]
	}

	return &CoachLLMEvalResult{
		OverallScore: ruleResult.Percentage,
		IsQualified:  ruleResult.IsComplete,
		Items:        items,
		Suggestion:   suggestion,
		Summary:      fmt.Sprintf("规则检测完成度%d%%", ruleResult.Percentage),
	}, nil
}

// coachTruncateStr 截断字符串到指定rune长度，超出部分加省略号
func coachTruncateStr(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
