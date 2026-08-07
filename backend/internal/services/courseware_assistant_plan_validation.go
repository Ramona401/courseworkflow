package services

// courseware_assistant_plan_validation.go
//
// 本文件负责课件教学智能体结构化教学方案的详细校验：
//   - 教学方式和方案版本；
//   - 引导原则；
//   - 通用教学互动步骤；
//   - 误区或学习困难分支；
//   - 步骤和分支引用关系；
//   - 答案保护策略。
//
// 本文件不读取数据库、不调用AI，也不保存任何内容。

import (
	"strings"

	"tedna/internal/models"
)

// validateCoursewareAssistantGuidancePlan 校验完整教学互动协议。
func validateCoursewareAssistantGuidancePlan(
	plan *models.CoursewareAssistantGuidancePlan,
) error {
	if plan == nil {
		return coursewareAssistantInvalid("缺少结构化教学方案")
	}

	normalizeCoursewareAssistantPlan(plan)

	if err := validateCoursewareAssistantPlanOverview(plan); err != nil {
		return err
	}

	stepIDs, err := validateCoursewareAssistantQuestionSteps(plan)
	if err != nil {
		return err
	}

	branchIDs, err := validateCoursewareAssistantBranches(plan)
	if err != nil {
		return err
	}

	return validateCoursewareAssistantPlanReferences(plan, stepIDs, branchIDs)
}

// normalizeCoursewareAssistantPlan 规范化顶层方案字段并升级历史v1方案。
func normalizeCoursewareAssistantPlan(
	plan *models.CoursewareAssistantGuidancePlan,
) {
	plan.Version = strings.TrimSpace(plan.Version)

	switch plan.Version {
	case "", models.CoursewareAssistantGuidancePlanVersionV1:
		// 历史v1方案在重新保存时确定性升级为v2。
		// 其缺失教学方式按guided_reasoning兼容，原有教学行为不变。
		plan.Version = models.CoursewareAssistantGuidancePlanCurrentVersion
	case models.CoursewareAssistantGuidancePlanVersionV2:
		// 当前版本保持不变。
	default:
		// 非法版本保持原值，交由校验器明确拒绝。
	}

	plan.TeachingMode = models.NormalizeCoursewareAssistantTeachingMode(plan.TeachingMode)
	plan.GuidingPrinciples = normalizeCoursewareAssistantStringSlice(plan.GuidingPrinciples)
	plan.ForbiddenBehaviors = normalizeCoursewareAssistantStringSlice(plan.ForbiddenBehaviors)
	plan.CompletionCriteria = normalizeCoursewareAssistantStringSlice(plan.CompletionCriteria)
	plan.AnswerLeakPolicy.ProhibitedBehaviors = normalizeCoursewareAssistantStringSlice(
		plan.AnswerLeakPolicy.ProhibitedBehaviors,
	)
	plan.AnswerLeakPolicy.SafeClosureGuidance = strings.TrimSpace(
		plan.AnswerLeakPolicy.SafeClosureGuidance,
	)
}

// validateCoursewareAssistantPlanOverview 校验方案顶层协议。
func validateCoursewareAssistantPlanOverview(
	plan *models.CoursewareAssistantGuidancePlan,
) error {
	if plan.Version != models.CoursewareAssistantGuidancePlanCurrentVersion {
		return coursewareAssistantInvalid("教学方案协议版本无效")
	}

	if !models.IsValidCoursewareAssistantTeachingMode(plan.TeachingMode) {
		return coursewareAssistantInvalid("教学方式无效")
	}

	if len(plan.GuidingPrinciples) == 0 {
		return coursewareAssistantInvalid("教学方案至少需要一条引导原则")
	}

	if len(plan.QuestionChain) == 0 {
		return coursewareAssistantInvalid("教学方案至少需要一个教学互动步骤")
	}

	if len(plan.QuestionChain) > coursewareAssistantMaxQuestionSteps {
		return coursewareAssistantInvalid("教学互动步骤数量超过上限")
	}

	if len(plan.MisconceptionBranches) > coursewareAssistantMaxMisconceptionBranches {
		return coursewareAssistantInvalid("误区分支数量超过上限")
	}

	if len(plan.CompletionCriteria) == 0 {
		return coursewareAssistantInvalid("教学方案至少需要一条完成标准")
	}

	if plan.AnswerLeakPolicy.DirectAnswerAllowed {
		return coursewareAssistantInvalid("教学智能体禁止直接泄露当前学生任务的答案")
	}

	if plan.AnswerLeakPolicy.MaximumHintLevel == 0 {
		plan.AnswerLeakPolicy.MaximumHintLevel = 3
	}

	if plan.AnswerLeakPolicy.MaximumHintLevel < 1 ||
		plan.AnswerLeakPolicy.MaximumHintLevel > coursewareAssistantMaxHintsPerStep {
		return coursewareAssistantInvalid("最大提示层级无效")
	}

	if err := validateCoursewareAssistantStringList(
		"引导原则",
		plan.GuidingPrinciples,
		coursewareAssistantPrincipleMaxRunes,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantStringList(
		"禁止行为",
		plan.ForbiddenBehaviors,
		coursewareAssistantForbiddenMaxRunes,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantStringList(
		"完成标准",
		plan.CompletionCriteria,
		coursewareAssistantCompletionMaxRunes,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantStringList(
		"答案保护禁止行为",
		plan.AnswerLeakPolicy.ProhibitedBehaviors,
		coursewareAssistantForbiddenMaxRunes,
	); err != nil {
		return err
	}

	if runeLength(plan.AnswerLeakPolicy.SafeClosureGuidance) >
		coursewareAssistantSafeClosureMaxRunes {
		return coursewareAssistantInvalid("安全收束说明长度超过上限")
	}

	return nil
}

// validateCoursewareAssistantQuestionSteps 校验教学互动步骤并返回ID集合。
func validateCoursewareAssistantQuestionSteps(
	plan *models.CoursewareAssistantGuidancePlan,
) (
	map[string]struct{},
	error,
) {
	stepIDs := make(map[string]struct{}, len(plan.QuestionChain))

	for index := range plan.QuestionChain {
		step := &plan.QuestionChain[index]

		step.ID = strings.TrimSpace(step.ID)
		step.Prompt = strings.TrimSpace(step.Prompt)
		step.TeachingIntent = strings.TrimSpace(step.TeachingIntent)
		step.NextStepID = strings.TrimSpace(step.NextStepID)
		step.CompletionSignal = strings.TrimSpace(step.CompletionSignal)
		step.ExpectedSignals = normalizeCoursewareAssistantStringSlice(step.ExpectedSignals)
		step.HintLadder = normalizeCoursewareAssistantStringSlice(step.HintLadder)
		step.MisconceptionBranchIDs = normalizeCoursewareAssistantStringSlice(
			step.MisconceptionBranchIDs,
		)

		if step.ID == "" {
			return nil, coursewareAssistantInvalid("教学互动步骤ID不能为空")
		}

		if _, exists := stepIDs[step.ID]; exists {
			return nil, coursewareAssistantInvalid("教学互动步骤ID不能重复")
		}

		stepIDs[step.ID] = struct{}{}

		if err := validateCoursewareAssistantRequiredText(
			"教学互动内容",
			step.Prompt,
			coursewareAssistantQuestionPromptMaxRunes,
		); err != nil {
			return nil, err
		}

		if err := validateCoursewareAssistantRequiredText(
			"教学互动意图",
			step.TeachingIntent,
			coursewareAssistantTeachingIntentMaxRunes,
		); err != nil {
			return nil, err
		}

		if len(step.ExpectedSignals) > coursewareAssistantMaxSignalsPerStep {
			return nil, coursewareAssistantInvalid("单个步骤的预期信号数量超过上限")
		}

		if len(step.HintLadder) > plan.AnswerLeakPolicy.MaximumHintLevel {
			return nil, coursewareAssistantInvalid("步骤提示数量超过最大提示层级")
		}

		if len(step.MisconceptionBranchIDs) > coursewareAssistantMaxBranchRefsPerStep {
			return nil, coursewareAssistantInvalid("单个步骤引用的误区分支数量超过上限")
		}

		if err := validateCoursewareAssistantStringList(
			"预期信号",
			step.ExpectedSignals,
			coursewareAssistantSignalMaxRunes,
		); err != nil {
			return nil, err
		}

		if err := validateCoursewareAssistantStringList(
			"提示内容",
			step.HintLadder,
			coursewareAssistantHintMaxRunes,
		); err != nil {
			return nil, err
		}
	}

	return stepIDs, nil
}

// validateCoursewareAssistantBranches 校验误区或学习困难分支并返回ID集合。
func validateCoursewareAssistantBranches(
	plan *models.CoursewareAssistantGuidancePlan,
) (
	map[string]struct{},
	error,
) {
	branchIDs := make(map[string]struct{}, len(plan.MisconceptionBranches))

	for index := range plan.MisconceptionBranches {
		branch := &plan.MisconceptionBranches[index]

		branch.ID = strings.TrimSpace(branch.ID)
		branch.ResponseStrategy = strings.TrimSpace(branch.ResponseStrategy)
		branch.FollowUpQuestion = strings.TrimSpace(branch.FollowUpQuestion)
		branch.ReturnToStepID = strings.TrimSpace(branch.ReturnToStepID)
		branch.MatchSignals = normalizeCoursewareAssistantStringSlice(branch.MatchSignals)

		if branch.ID == "" {
			return nil, coursewareAssistantInvalid("误区分支ID不能为空")
		}

		if _, exists := branchIDs[branch.ID]; exists {
			return nil, coursewareAssistantInvalid("误区分支ID不能重复")
		}

		branchIDs[branch.ID] = struct{}{}

		if len(branch.MatchSignals) == 0 {
			return nil, coursewareAssistantInvalid("误区分支至少需要一个匹配信号")
		}

		if err := validateCoursewareAssistantStringList(
			"误区匹配信号",
			branch.MatchSignals,
			coursewareAssistantSignalMaxRunes,
		); err != nil {
			return nil, err
		}

		if err := validateCoursewareAssistantRequiredText(
			"误区响应策略",
			branch.ResponseStrategy,
			coursewareAssistantBranchStrategyMaxRunes,
		); err != nil {
			return nil, err
		}

		if err := validateCoursewareAssistantRequiredText(
			"误区追问",
			branch.FollowUpQuestion,
			coursewareAssistantFollowUpMaxRunes,
		); err != nil {
			return nil, err
		}

		if branch.ReturnToStepID == "" {
			return nil, coursewareAssistantInvalid("误区分支必须指定返回步骤")
		}
	}

	return branchIDs, nil
}

// validateCoursewareAssistantPlanReferences 校验步骤和分支引用关系。
func validateCoursewareAssistantPlanReferences(
	plan *models.CoursewareAssistantGuidancePlan,
	stepIDs map[string]struct{},
	branchIDs map[string]struct{},
) error {
	for index := range plan.QuestionChain {
		step := &plan.QuestionChain[index]

		if step.NextStepID != "" {
			if _, exists := stepIDs[step.NextStepID]; !exists {
				return coursewareAssistantInvalid("下一步骤引用不存在")
			}
		}

		for _, branchID := range step.MisconceptionBranchIDs {
			if _, exists := branchIDs[branchID]; !exists {
				return coursewareAssistantInvalid("教学互动步骤引用的误区分支不存在")
			}
		}
	}

	for index := range plan.MisconceptionBranches {
		branch := &plan.MisconceptionBranches[index]

		if _, exists := stepIDs[branch.ReturnToStepID]; !exists {
			return coursewareAssistantInvalid("误区分支返回步骤不存在")
		}
	}

	return nil
}
