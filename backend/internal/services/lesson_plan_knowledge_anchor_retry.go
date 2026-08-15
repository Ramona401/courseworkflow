package services

// lesson_plan_knowledge_anchor_retry.go — 课程锚点AI提取的一次性受控重试
//
// 设计边界：
//   - 只重试AI输出协议失败，不重试权限、教师真实消息不足、对话过长、配置或数据库错误；
//   - 第二次调用使用与首轮完全相同的可信输入，只追加失败类别，不回喂首轮原始输出；
//   - 两轮都必须通过现有核心字段和原文证据校验，绝不因重试而放宽安全规则；
//   - 日志只记录plan_id和受控失败类别，不记录对话原文、证据片段或AI原始输出；
//   - Token统计累计两次真实调用，模型名按实际调用顺序去重汇总。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

type lessonPlanKnowledgeAnchorRetryReason string

const (
	lessonPlanKnowledgeAnchorRetryEmptyOutput      lessonPlanKnowledgeAnchorRetryReason = "empty_output"
	lessonPlanKnowledgeAnchorRetryInvalidJSON      lessonPlanKnowledgeAnchorRetryReason = "invalid_json"
	lessonPlanKnowledgeAnchorRetryIncompleteCore   lessonPlanKnowledgeAnchorRetryReason = "incomplete_core"
	lessonPlanKnowledgeAnchorRetryEvidenceMismatch lessonPlanKnowledgeAnchorRetryReason = "evidence_mismatch"
)

// lessonPlanKnowledgeAnchorRetryableError 仅标记“允许再独立提取一次”的AI协议失败。
// Error()保持原有教师可读错误文本，Unwrap()保持errors.Is对既有哨兵错误的兼容。
type lessonPlanKnowledgeAnchorRetryableError struct {
	reason lessonPlanKnowledgeAnchorRetryReason
	cause  error
}

func (e *lessonPlanKnowledgeAnchorRetryableError) Error() string {
	if e == nil || e.cause == nil {
		return "课程锚点AI输出未通过协议校验"
	}
	return e.cause.Error()
}

func (e *lessonPlanKnowledgeAnchorRetryableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newLessonPlanKnowledgeAnchorRetryableError(
	reason lessonPlanKnowledgeAnchorRetryReason,
	cause error,
) error {
	return &lessonPlanKnowledgeAnchorRetryableError{
		reason: reason,
		cause:  cause,
	}
}

func lessonPlanKnowledgeAnchorRetryReasonOf(
	err error,
) (lessonPlanKnowledgeAnchorRetryReason, bool) {
	var retryable *lessonPlanKnowledgeAnchorRetryableError
	if !errors.As(err, &retryable) || retryable == nil {
		return "", false
	}
	return retryable.reason, true
}

type lessonPlanKnowledgeAnchorAttemptFunc func(
	systemPrompt string,
) (
	parsed *lessonPlanKnowledgeAnchorAIResult,
	modelUsed string,
	tokensUsed int,
	err error,
)

// extractLessonPlanKnowledgeAnchorAIWithRetry 构建与原逻辑完全一致的可信输入，
// 并把一次性受控重试交给独立编排函数执行。
func (s *WorkshopStageService) extractLessonPlanKnowledgeAnchorAIWithRetry(
	ctx context.Context,
	source *repository.LessonPlanKnowledgeLineageSource,
	transcript string,
) (*lessonPlanKnowledgeAnchorAIResult, string, int, error) {
	if source == nil {
		return nil, "", 0, fmt.Errorf(
			"%w: 教学分析来源为空",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
		)
	}
	if strings.TrimSpace(s.aesKey) == "" {
		return nil, "", 0, fmt.Errorf(
			"%w: 阶段服务AI密钥未初始化",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
		)
	}

	effectiveConfig, err := aiClient.GetEffectiveConfig(
		s.aesKey,
		models.SceneLessonPlanHarness,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, "", 0, fmt.Errorf(
			"%w: 加载课程锚点提取模型失败: %v",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			err,
		)
	}

	effectiveConfig.Temperature = 0
	if effectiveConfig.MaxTokens <= 0 ||
		effectiveConfig.MaxTokens > lessonPlanKnowledgeAnchorMaxTokens {
		effectiveConfig.MaxTokens = lessonPlanKnowledgeAnchorMaxTokens
	}

	inputData := map[string]interface{}{
		"lesson_plan": map[string]string{
			"subject": source.Subject,
			"grade":   source.Grade,
			"topic":   source.Topic,
		},
		"confirmation_event":              "教师刚刚主动完成教学分析阶段；只能结构化对话中已敲定的结论，不能补猜",
		"stage_structured_output":         source.StageStructuredOutput,
		"stage_narrative_output":          source.StageNarrativeOutput,
		"confirmed_analysis_conversation": transcript,
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, "", 0, fmt.Errorf(
			"%w: 序列化课程锚点输入失败: %v",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			err,
		)
	}

	attempt := func(systemPrompt string) (
		*lessonPlanKnowledgeAnchorAIResult,
		string,
		int,
		error,
	) {
		result, callErr := aiClient.CallAI(
			effectiveConfig,
			systemPrompt,
			string(inputJSON),
			buildLessonPlanKnowledgeTraceContext(
				ctx,
				source.LessonPlanID,
				source.AuthorID,
			),
		)
		if callErr != nil {
			return nil, "", 0, fmt.Errorf(
				"%w: 课程锚点提取调用失败: %v",
				ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
				callErr,
			)
		}
		if result == nil {
			return nil, "", 0, newLessonPlanKnowledgeAnchorRetryableError(
				lessonPlanKnowledgeAnchorRetryEmptyOutput,
				fmt.Errorf(
					"%w: 课程锚点提取结果为空",
					ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
				),
			)
		}

		parsed, parseErr := parseAndValidateLessonPlanKnowledgeAnchorAIContent(
			result.Content,
			transcript,
		)
		return parsed, result.ModelUsed, result.TokensUsed, parseErr
	}

	return runLessonPlanKnowledgeAnchorControlledRetry(
		source.LessonPlanID,
		attempt,
	)
}

// runLessonPlanKnowledgeAnchorControlledRetry 最多执行两次attempt。
// 第一次只有被明确标记为retryable的AI协议错误才允许进入第二次；其它错误立即原样返回。
func runLessonPlanKnowledgeAnchorControlledRetry(
	lessonPlanID string,
	attempt lessonPlanKnowledgeAnchorAttemptFunc,
) (*lessonPlanKnowledgeAnchorAIResult, string, int, error) {
	if attempt == nil {
		return nil, "", 0, fmt.Errorf(
			"%w: 课程锚点提取执行器为空",
			ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
		)
	}

	var (
		modelsUsed  []string
		totalTokens int
		firstReason lessonPlanKnowledgeAnchorRetryReason
	)

	for attemptIndex := 0; attemptIndex < 2; attemptIndex++ {
		systemPrompt := lessonPlanKnowledgeAnchorSystemPrompt
		if attemptIndex == 1 {
			systemPrompt = buildLessonPlanKnowledgeAnchorRetrySystemPrompt(
				firstReason,
			)
		}

		parsed, modelUsed, tokensUsed, err := attempt(systemPrompt)
		modelsUsed = appendLessonPlanKnowledgeAnchorAttemptModel(
			modelsUsed,
			modelUsed,
		)
		if tokensUsed > 0 {
			totalTokens += tokensUsed
		}

		if err == nil {
			if attemptIndex == 1 {
				wsLog.Info(
					"课程锚点受控重试成功",
					"plan_id", strings.TrimSpace(lessonPlanID),
					"first_reason", string(firstReason),
				)
			}
			return parsed, strings.Join(modelsUsed, ";"), totalTokens, nil
		}

		reason, retryable := lessonPlanKnowledgeAnchorRetryReasonOf(err)
		if attemptIndex == 0 && retryable {
			firstReason = reason
			wsLog.Warn(
				"课程锚点首轮提取未通过协议校验，准备受控重试",
				"plan_id", strings.TrimSpace(lessonPlanID),
				"reason", string(reason),
			)
			continue
		}

		if attemptIndex == 1 {
			secondReason := "non_retryable"
			if retryable {
				secondReason = string(reason)
			}
			wsLog.Warn(
				"课程锚点受控重试后仍未通过",
				"plan_id", strings.TrimSpace(lessonPlanID),
				"first_reason", string(firstReason),
				"second_reason", secondReason,
			)
		}

		return nil, strings.Join(modelsUsed, ";"), totalTokens, err
	}

	return nil, strings.Join(modelsUsed, ";"), totalTokens, fmt.Errorf(
		"%w: 课程锚点受控重试未得到结果",
		ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
	)
}

// parseAndValidateLessonPlanKnowledgeAnchorAIContent 只负责AI输出协议与既有安全校验。
// 所有可重试错误都保留原错误文本和errors.Is语义，外层只依据受控类别决定是否再调用一次AI。
func parseAndValidateLessonPlanKnowledgeAnchorAIContent(
	content string,
	transcript string,
) (*lessonPlanKnowledgeAnchorAIResult, error) {
	if strings.TrimSpace(content) == "" {
		return nil, newLessonPlanKnowledgeAnchorRetryableError(
			lessonPlanKnowledgeAnchorRetryEmptyOutput,
			fmt.Errorf(
				"%w: 课程锚点提取结果为空",
				ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			),
		)
	}

	jsonText, ok := aiClient.ExtractJSON(content)
	if !ok {
		return nil, newLessonPlanKnowledgeAnchorRetryableError(
			lessonPlanKnowledgeAnchorRetryInvalidJSON,
			fmt.Errorf(
				"%w: 课程锚点结果不是合法JSON",
				ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
			),
		)
	}

	parsed := &lessonPlanKnowledgeAnchorAIResult{}
	if err := json.Unmarshal([]byte(jsonText), parsed); err != nil {
		return nil, newLessonPlanKnowledgeAnchorRetryableError(
			lessonPlanKnowledgeAnchorRetryInvalidJSON,
			fmt.Errorf(
				"%w: 解析课程锚点失败: %v",
				ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
				err,
			),
		)
	}

	normalizeLessonPlanKnowledgeAnchors(&parsed.Anchors)

	// AI不能决定教师确认状态。调用入口仍必须是教师完成教学分析阶段。
	parsed.Anchors.TeacherConfirmed = true

	if !parsed.Ready || !parsed.Anchors.HasConfirmedCore() {
		return nil, newLessonPlanKnowledgeAnchorRetryableError(
			lessonPlanKnowledgeAnchorRetryIncompleteCore,
			buildKnowledgeAnchorIncompleteError(
				parsed.MissingFields,
				parsed.AmbiguityNotes,
			),
		)
	}

	if err := validateKnowledgeAnchorEvidenceAgainstTranscript(
		&parsed.Anchors,
		transcript,
	); err != nil {
		return nil, newLessonPlanKnowledgeAnchorRetryableError(
			lessonPlanKnowledgeAnchorRetryEvidenceMismatch,
			err,
		)
	}

	return parsed, nil
}

// buildLessonPlanKnowledgeAnchorRetrySystemPrompt 只接收枚举类别，永远不接收首轮AI原文或证据原文。
func buildLessonPlanKnowledgeAnchorRetrySystemPrompt(
	reason lessonPlanKnowledgeAnchorRetryReason,
) string {
	reasonInstruction := "上一轮输出未通过课程锚点协议校验。"

	switch reason {
	case lessonPlanKnowledgeAnchorRetryEmptyOutput:
		reasonInstruction = "上一轮没有返回可校验的JSON对象。"
	case lessonPlanKnowledgeAnchorRetryInvalidJSON:
		reasonInstruction = "上一轮返回内容不是可解析的唯一JSON对象。"
	case lessonPlanKnowledgeAnchorRetryIncompleteCore:
		reasonInstruction = "上一轮没有同时满足ready与全部课程锚点核心字段要求。"
	case lessonPlanKnowledgeAnchorRetryEvidenceMismatch:
		reasonInstruction = "上一轮证据字段没有全部使用教学分析对话中的真实原文短句。"
	}

	return lessonPlanKnowledgeAnchorSystemPrompt + `

【受控重试要求】
` + reasonInstruction + `
请重新独立阅读本次提供的同一份输入并重新生成完整JSON，不得引用、修补、续写或复述上一轮输出。
不得为了通过校验而补猜缺失事实；如果同一输入确实缺少必需信息，仍必须返回ready=false。
source_evidence与knowledge_points[].evidence仍必须逐字摘录confirmed_analysis_conversation中真实存在的短句。
除JSON对象外不得输出任何其它文字。`
}

func appendLessonPlanKnowledgeAnchorAttemptModel(
	modelsUsed []string,
	modelUsed string,
) []string {
	modelUsed = strings.TrimSpace(modelUsed)
	if modelUsed == "" {
		return modelsUsed
	}

	for _, current := range modelsUsed {
		if current == modelUsed {
			return modelsUsed
		}
	}

	return append(modelsUsed, modelUsed)
}
