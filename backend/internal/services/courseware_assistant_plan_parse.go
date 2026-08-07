package services

// courseware_assistant_plan_parse.go
//
// 本文件负责严格解析模型返回的教学智能体方案。
//
// 本功能生成的结果会直接进入教师编辑器，后续还可能被保存和发布，
// 因此不允许字段级猜测、教学方式漂移、超大方案或半成品降级。
//
// 解析策略：
//   - 允许标准JSON或包含合法JSON对象的Markdown代码块；
//   - 使用DisallowUnknownFields拒绝未定义字段；
//   - 所有业务字段使用明确结构体；
//   - 强制模型返回的teaching_mode与教师请求完全一致；
//   - AI结果采用比教师手工编辑更严格的规模限制；
//   - 复用完整方案校验；
//   - 重新验证步骤、分支引用和答案保护；
//   - 任意错误整轮失败，不返回部分结果。

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
)

const (
	// AI单次生成结果只应描述一页课件，512KiB已经远高于合理方案需要。
	// 超过该值通常意味着模型重复输出、扩写整课或生成异常。
	coursewareAssistantPlanAIResponseMaxBytes = 512 * 1024

	coursewareAssistantPlanMinimumQuestionSteps           = 4
	coursewareAssistantPlanMaximumQuestionSteps           = 8
	coursewareAssistantPlanMaximumMisconceptionBranches   = 6
	coursewareAssistantPlanMaximumGuidingPrinciples       = 6
	coursewareAssistantPlanMaximumForbiddenBehaviors      = 6
	coursewareAssistantPlanMaximumCompletionCriteria      = 6
	coursewareAssistantPlanMaximumSignalsPerStep          = 4
	coursewareAssistantPlanMaximumBranchesPerStep         = 3
	coursewareAssistantPlanMaximumSignalsPerMisconception = 4
)

// coursewareAssistantPlanAIResponse 是模型唯一允许返回的顶层结构。
type coursewareAssistantPlanAIResponse struct {
	TeachingMode      string `json:"teaching_mode"`
	Name              string `json:"name"`
	WelcomeMessage    string `json:"welcome_message"`
	TeachingRole      string `json:"teaching_role"`
	LearningObjective string `json:"learning_objective"`

	GuidingPrinciples     []string                                        `json:"guiding_principles"`
	QuestionChain         []models.CoursewareAssistantQuestionStep        `json:"question_chain"`
	MisconceptionBranches []models.CoursewareAssistantMisconceptionBranch `json:"misconception_branches"`
	ForbiddenBehaviors    []string                                        `json:"forbidden_behaviors"`
	CompletionCriteria    []string                                        `json:"completion_criteria"`

	AnswerLeakPolicy models.CoursewareAssistantAnswerLeakPolicy `json:"answer_leak_policy"`
	ContextScope     models.CoursewareAssistantContextConfig    `json:"context_scope"`
}

// parseCoursewareAssistantPlanAIResult 严格解析并规范模型输出。
func parseCoursewareAssistantPlanAIResult(
	raw string,
	requestedTeachingMode string,
) (
	*models.CoursewareAssistantPlanResult,
	error,
) {
	requestedTeachingMode = models.NormalizeCoursewareAssistantTeachingMode(
		requestedTeachingMode,
	)
	if !models.IsValidCoursewareAssistantTeachingMode(requestedTeachingMode) {
		return nil, coursewareAssistantPlanOutputError("请求教学方式无效", nil)
	}

	jsonText, err := extractCoursewareAssistantPlanJSON(raw)
	if err != nil {
		return nil, err
	}

	if len([]byte(jsonText)) > coursewareAssistantPlanAIResponseMaxBytes {
		return nil, coursewareAssistantPlanOutputError(
			"AI返回的教学方案数据量异常，请重新生成",
			nil,
		)
	}

	var response coursewareAssistantPlanAIResponse

	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&response); err != nil {
		return nil, coursewareAssistantPlanOutputError("JSON结构解析失败", err)
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, coursewareAssistantPlanOutputError("JSON对象后存在额外数据", nil)
		}
		return nil, coursewareAssistantPlanOutputError("JSON尾部数据无效", err)
	}

	response.TeachingMode = models.NormalizeCoursewareAssistantTeachingMode(
		response.TeachingMode,
	)
	if !models.IsValidCoursewareAssistantTeachingMode(response.TeachingMode) {
		return nil, coursewareAssistantPlanOutputError("AI返回的教学方式无效", nil)
	}

	if response.TeachingMode != requestedTeachingMode {
		return nil, coursewareAssistantPlanOutputError(
			"AI返回的教学方式与教师选择不一致",
			nil,
		)
	}

	result := &models.CoursewareAssistantPlanResult{
		Title:             strings.TrimSpace(response.Name),
		WelcomeMessage:    strings.TrimSpace(response.WelcomeMessage),
		TeachingRole:      strings.TrimSpace(response.TeachingRole),
		LearningObjective: strings.TrimSpace(response.LearningObjective),
		GuidancePlan: models.CoursewareAssistantGuidancePlan{
			Version:               models.CoursewareAssistantGuidancePlanCurrentVersion,
			TeachingMode:          response.TeachingMode,
			GuidingPrinciples:     response.GuidingPrinciples,
			QuestionChain:         response.QuestionChain,
			MisconceptionBranches: response.MisconceptionBranches,
			ForbiddenBehaviors:    response.ForbiddenBehaviors,
			CompletionCriteria:    response.CompletionCriteria,
			AnswerLeakPolicy:      response.AnswerLeakPolicy,
		},
		ContextConfig: response.ContextScope,
	}

	// 复用插槽创建协议进行完整规范化与引用校验。
	draft := &models.CreateCoursewareAssistantSlotRequest{
		Title:             result.Title,
		WelcomeMessage:    result.WelcomeMessage,
		TeachingRole:      result.TeachingRole,
		LearningObjective: result.LearningObjective,
		GuidancePlan:      result.GuidancePlan,
		ContextConfig:     result.ContextConfig,
	}

	if err := prepareCoursewareAssistantCreateRequest(draft); err != nil {
		return nil, coursewareAssistantPlanOutputError(
			"结构化教学方案未通过业务校验",
			err,
		)
	}

	// AI生成方案比教师手工编辑采用更严格的默认安全标准。
	if !draft.GuidancePlan.AnswerLeakPolicy.RequireStudentTry {
		return nil, coursewareAssistantPlanOutputError("生成方案必须要求学生先参与", nil)
	}

	if draft.GuidancePlan.AnswerLeakPolicy.MaximumHintLevel > 3 {
		return nil, coursewareAssistantPlanOutputError("生成方案的最大提示层级不能超过3", nil)
	}

	if draft.GuidancePlan.TeachingMode != requestedTeachingMode {
		return nil, coursewareAssistantPlanOutputError("规范化后教学方式发生漂移", nil)
	}

	if err := validateGeneratedCoursewareAssistantPlanScale(
		&draft.GuidancePlan,
	); err != nil {
		return nil, err
	}

	// 使用校验器规范化后的最终值，避免返回空数组、空白或默认版本漂移。
	result.Title = draft.Title
	result.WelcomeMessage = draft.WelcomeMessage
	result.TeachingRole = draft.TeachingRole
	result.LearningObjective = draft.LearningObjective
	result.GuidancePlan = draft.GuidancePlan
	result.ContextConfig = draft.ContextConfig

	return result, nil
}

// validateGeneratedCoursewareAssistantPlanScale 限制AI自动生成方案的课堂规模。
//
// 教师在高级设置中仍可手工编辑符合通用协议的更复杂方案；
// 本限制只针对AI一次性生成结果，避免模型把单页任务扩写为整课脚本。
func validateGeneratedCoursewareAssistantPlanScale(
	plan *models.CoursewareAssistantGuidancePlan,
) error {
	if plan == nil {
		return coursewareAssistantPlanOutputError("生成方案为空", nil)
	}

	stepCount := len(plan.QuestionChain)
	if stepCount < coursewareAssistantPlanMinimumQuestionSteps ||
		stepCount > coursewareAssistantPlanMaximumQuestionSteps {
		return coursewareAssistantPlanOutputError(
			fmt.Sprintf(
				"AI生成的互动步骤必须为%d至%d个",
				coursewareAssistantPlanMinimumQuestionSteps,
				coursewareAssistantPlanMaximumQuestionSteps,
			),
			nil,
		)
	}

	if len(plan.MisconceptionBranches) >
		coursewareAssistantPlanMaximumMisconceptionBranches {
		return coursewareAssistantPlanOutputError(
			fmt.Sprintf(
				"AI生成的学习困难方案不能超过%d个",
				coursewareAssistantPlanMaximumMisconceptionBranches,
			),
			nil,
		)
	}

	if len(plan.GuidingPrinciples) >
		coursewareAssistantPlanMaximumGuidingPrinciples {
		return coursewareAssistantPlanOutputError("AI生成的引导原则过多", nil)
	}

	if len(plan.ForbiddenBehaviors) >
		coursewareAssistantPlanMaximumForbiddenBehaviors {
		return coursewareAssistantPlanOutputError("AI生成的禁止行为过多", nil)
	}

	if len(plan.CompletionCriteria) >
		coursewareAssistantPlanMaximumCompletionCriteria {
		return coursewareAssistantPlanOutputError("AI生成的完成标准过多", nil)
	}

	for index, step := range plan.QuestionChain {
		if len(step.ExpectedSignals) >
			coursewareAssistantPlanMaximumSignalsPerStep {
			return coursewareAssistantPlanOutputError(
				fmt.Sprintf(
					"第%d个互动步骤的理解信号过多",
					index+1,
				),
				nil,
			)
		}

		if len(step.HintLadder) == 0 ||
			len(step.HintLadder) > 3 {
			return coursewareAssistantPlanOutputError(
				fmt.Sprintf(
					"第%d个互动步骤的提示必须为1至3层",
					index+1,
				),
				nil,
			)
		}

		if len(step.MisconceptionBranchIDs) >
			coursewareAssistantPlanMaximumBranchesPerStep {
			return coursewareAssistantPlanOutputError(
				fmt.Sprintf(
					"第%d个互动步骤引用的学习困难方案过多",
					index+1,
				),
				nil,
			)
		}
	}

	for index, branch := range plan.MisconceptionBranches {
		if len(branch.MatchSignals) >
			coursewareAssistantPlanMaximumSignalsPerMisconception {
			return coursewareAssistantPlanOutputError(
				fmt.Sprintf(
					"第%d个学习困难方案的匹配信号过多",
					index+1,
				),
				nil,
			)
		}
	}

	return nil
}

// extractCoursewareAssistantPlanJSON 提取合法JSON对象。
func extractCoursewareAssistantPlanJSON(
	raw string,
) (
	string,
	error,
) {
	cleaned := strings.TrimSpace(raw)

	if cleaned == "" {
		return "", coursewareAssistantPlanOutputError("AI返回内容为空", nil)
	}

	if extracted, ok := ai.ExtractJSON(cleaned); ok && strings.TrimSpace(extracted) != "" {
		return strings.TrimSpace(extracted), nil
	}

	// 没有代码块时仍允许模型直接返回完整JSON对象。
	if strings.HasPrefix(cleaned, "{") && strings.HasSuffix(cleaned, "}") {
		return cleaned, nil
	}

	return "", coursewareAssistantPlanOutputError("未找到完整合法的JSON对象", nil)
}

// coursewareAssistantPlanOutputError 构造统一的可识别输出错误。
func coursewareAssistantPlanOutputError(
	detail string,
	cause error,
) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", ErrCoursewareAssistantPlanInvalidOutput, detail)
	}

	return fmt.Errorf("%w: %s: %v", ErrCoursewareAssistantPlanInvalidOutput, detail, cause)
}
