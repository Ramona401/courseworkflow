package services

// courseware_assistant_validation.go
//
// 本文件负责课件教学智能体编辑请求、主字段和上下文范围的确定性校验。
//
// 详细问题链、误区分支及引用关系校验拆分到：
// courseware_assistant_plan_validation.go。
//
// 校验边界：
//   - 不读取数据库；
//   - 不调用AI；
//   - 不生成部署；
//   - 不接受任意URL、工具或自由扩展字段；
//   - 中文长度统一按Unicode字符数量计算。

import (
	"fmt"
	"strings"

	"tedna/internal/models"
)

const (
	coursewareAssistantTitleMaxRunes              = 120
	coursewareAssistantWelcomeMaxRunes            = 4000
	coursewareAssistantRoleMaxRunes               = 4000
	coursewareAssistantObjectiveMaxRunes          = 8000
	coursewareAssistantPrincipleMaxRunes          = 1000
	coursewareAssistantQuestionPromptMaxRunes     = 4000
	coursewareAssistantTeachingIntentMaxRunes     = 2000
	coursewareAssistantSignalMaxRunes             = 500
	coursewareAssistantHintMaxRunes               = 2000
	coursewareAssistantBranchStrategyMaxRunes     = 4000
	coursewareAssistantFollowUpMaxRunes           = 4000
	coursewareAssistantCompletionMaxRunes         = 1000
	coursewareAssistantForbiddenMaxRunes          = 1000
	coursewareAssistantSafeClosureMaxRunes        = 2000
	coursewareAssistantMaxQuestionSteps           = 64
	coursewareAssistantMaxMisconceptionBranches   = 64
	coursewareAssistantMaxHintsPerStep            = 8
	coursewareAssistantMaxSignalsPerStep          = 32
	coursewareAssistantMaxBranchRefsPerStep       = 16
	coursewareAssistantMaxLessonExcerptCharacters = 12000
)

// prepareCoursewareAssistantCreateRequest 规范化并校验创建请求。
func prepareCoursewareAssistantCreateRequest(
	request *models.CreateCoursewareAssistantSlotRequest,
) error {
	if request == nil {
		return ErrCoursewareAssistantInvalidRequest
	}

	normalizeCoursewareAssistantOptionalID(
		&request.AssistantID,
	)

	request.Title =
		strings.TrimSpace(request.Title)
	request.WelcomeMessage =
		strings.TrimSpace(request.WelcomeMessage)
	request.TeachingRole =
		strings.TrimSpace(request.TeachingRole)
	request.LearningObjective =
		strings.TrimSpace(request.LearningObjective)

	if err := validateCoursewareAssistantCoreFields(
		request.Title,
		request.WelcomeMessage,
		request.TeachingRole,
		request.LearningObjective,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantGuidancePlan(
		&request.GuidancePlan,
	); err != nil {
		return err
	}

	return validateCoursewareAssistantContextConfig(
		&request.ContextConfig,
	)
}

// prepareCoursewareAssistantUpdateRequest 规范化并校验更新请求。
func prepareCoursewareAssistantUpdateRequest(
	request *models.UpdateCoursewareAssistantSlotRequest,
) error {
	if request == nil {
		return ErrCoursewareAssistantInvalidRequest
	}

	normalizeCoursewareAssistantOptionalID(
		&request.AssistantID,
	)

	request.Title =
		strings.TrimSpace(request.Title)
	request.WelcomeMessage =
		strings.TrimSpace(request.WelcomeMessage)
	request.TeachingRole =
		strings.TrimSpace(request.TeachingRole)
	request.LearningObjective =
		strings.TrimSpace(request.LearningObjective)
	request.Status =
		strings.TrimSpace(request.Status)

	if request.Status == "" {
		request.Status =
			models.CoursewareAssistantSlotStatusActive
	}

	if !models.IsValidCoursewareAssistantSlotStatus(
		request.Status,
	) {
		return coursewareAssistantInvalid(
			"插槽状态无效",
		)
	}

	if err := validateCoursewareAssistantCoreFields(
		request.Title,
		request.WelcomeMessage,
		request.TeachingRole,
		request.LearningObjective,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantGuidancePlan(
		&request.GuidancePlan,
	); err != nil {
		return err
	}

	return validateCoursewareAssistantContextConfig(
		&request.ContextConfig,
	)
}

// validateCoursewareAssistantCoreFields 校验教师可编辑主字段。
func validateCoursewareAssistantCoreFields(
	title string,
	welcomeMessage string,
	teachingRole string,
	learningObjective string,
) error {
	if err := validateCoursewareAssistantRequiredText(
		"智能体名称",
		title,
		coursewareAssistantTitleMaxRunes,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantRequiredText(
		"欢迎语",
		welcomeMessage,
		coursewareAssistantWelcomeMaxRunes,
	); err != nil {
		return err
	}

	if err := validateCoursewareAssistantRequiredText(
		"教学角色",
		teachingRole,
		coursewareAssistantRoleMaxRunes,
	); err != nil {
		return err
	}

	return validateCoursewareAssistantRequiredText(
		"教学目标",
		learningObjective,
		coursewareAssistantObjectiveMaxRunes,
	)
}

// validateCoursewareAssistantContextConfig 校验上下文范围。
func validateCoursewareAssistantContextConfig(
	config *models.CoursewareAssistantContextConfig,
) error {
	if config == nil {
		return coursewareAssistantInvalid(
			"缺少上下文范围配置",
		)
	}

	config.Version =
		strings.TrimSpace(config.Version)

	if config.Version == "" {
		config.Version =
			models.CoursewareAssistantProtocolVersion
	}

	if config.Version !=
		models.CoursewareAssistantProtocolVersion {
		return coursewareAssistantInvalid(
			"上下文配置协议版本无效",
		)
	}

	if !config.IncludeVisibleText &&
		!config.IncludePagePlan {
		return coursewareAssistantInvalid(
			"上下文必须包含当前页可见文字或当前页教学方案",
		)
	}

	if config.IncludeLessonPlanExcerpt {
		if config.MaxLessonPlanExcerptChars == 0 {
			config.MaxLessonPlanExcerptChars = 4000
		}

		if config.MaxLessonPlanExcerptChars < 500 ||
			config.MaxLessonPlanExcerptChars >
				coursewareAssistantMaxLessonExcerptCharacters {
			return coursewareAssistantInvalid(
				"教案相关片段长度范围无效",
			)
		}
	} else {
		config.MaxLessonPlanExcerptChars = 0
	}

	return nil
}

// validateCoursewareAssistantRequiredText 校验必填文本。
func validateCoursewareAssistantRequiredText(
	label string,
	value string,
	maxRunes int,
) error {
	if strings.TrimSpace(value) == "" {
		return coursewareAssistantInvalid(
			label + "不能为空",
		)
	}

	if runeLength(value) > maxRunes {
		return coursewareAssistantInvalid(
			label + "长度超过上限",
		)
	}

	return nil
}

// validateCoursewareAssistantStringList 校验字符串数组的单项长度。
func validateCoursewareAssistantStringList(
	label string,
	values []string,
	maxRunes int,
) error {
	for _, value := range values {
		if runeLength(value) > maxRunes {
			return coursewareAssistantInvalid(
				label + "单项长度超过上限",
			)
		}
	}

	return nil
}

// normalizeCoursewareAssistantOptionalID 规范化可空UUID字符串。
func normalizeCoursewareAssistantOptionalID(
	target **string,
) {
	if target == nil ||
		*target == nil {
		return
	}

	normalized :=
		strings.TrimSpace(**target)

	if normalized == "" {
		*target = nil
		return
	}

	*target = &normalized
}

// normalizeCoursewareAssistantStringSlice 去除空项并稳定返回非nil切片。
func normalizeCoursewareAssistantStringSlice(
	values []string,
) []string {
	result := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		normalized :=
			strings.TrimSpace(value)

		if normalized != "" {
			result = append(
				result,
				normalized,
			)
		}
	}

	return result
}

// coursewareAssistantInvalid 构造可被errors.Is识别的协议错误。
func coursewareAssistantInvalid(
	detail string,
) error {
	return fmt.Errorf(
		"%w: %s",
		ErrCoursewareAssistantInvalidRequest,
		detail,
	)
}

// runeLength 按Unicode字符数计算长度。
func runeLength(
	value string,
) int {
	return len(
		[]rune(value),
	)
}
