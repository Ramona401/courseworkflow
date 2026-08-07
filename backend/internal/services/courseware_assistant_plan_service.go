package services

// courseware_assistant_plan_service.go
//
// 本文件负责根据一页课件的确定性教学上下文和教师选择的教学方式，
// 生成可编辑的教学智能体方案。
//
// 完整流程：
//   1. 校验可信教师Actor、教学方式和请求长度；
//   2. 重新读取作者自己的正式课件；
//   3. 校验submitted、in_pipeline等写入锁；
//   4. 使用确定性代码构建页面上下文；
//   5. 历史页面绑定可用助手时，可将其作为教学风格参考；
//   6. 无助手或历史助手失效时，安全回退为系统默认页面教学风格；
//   7. 服务端解析教师当前真实学校ID；
//   8. 使用真实UserID和SchoolID调用统一AI客户端；
//   9. 严格解析并重新执行结构化教学方案校验；
//   10. 首次结构校验失败时自动增加纠正要求并重试一次；
//   11. 强制AI返回的教学方式与教师选择一致；
//   12. 只返回教师可编辑草稿。
//
// 本单元明确不做：
//   - 不保存courseware_assistant_slots；
//   - 不创建assistant_deployments；
//   - 不修改courseware_pages.html_content；
//   - 不保存AI原始响应；
//   - 不把助手完整提示词返回浏览器。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	// coursewareAssistantPlanSceneCode 是AI配置、追踪和积分使用的独立场景。
	coursewareAssistantPlanSceneCode = "courseware_assistant_plan"

	// 教师补充要求最多8000个Unicode字符。
	coursewareAssistantPlanTeacherInstructionMaxRunes = 8000
)

var (
	// ErrCoursewareAssistantPlanServiceUnavailable 表示服务依赖没有完成初始化。
	ErrCoursewareAssistantPlanServiceUnavailable = errors.New(
		"课件教学智能体方案生成服务不可用",
	)

	// ErrCoursewareAssistantPlanInstructionTooLong 表示教师补充要求超过上限。
	ErrCoursewareAssistantPlanInstructionTooLong = errors.New(
		"课件教学智能体方案补充要求长度超过上限",
	)

	// ErrCoursewareAssistantPlanSchoolRequired 表示无法建立真实学校计费和模型分流上下文。
	ErrCoursewareAssistantPlanSchoolRequired = errors.New(
		"当前教师未绑定可用学校，不能生成教学智能体方案",
	)

	// ErrCoursewareAssistantPlanSchoolResolveFailed 表示学校查询发生数据库异常。
	ErrCoursewareAssistantPlanSchoolResolveFailed = errors.New(
		"解析教师学校失败",
	)

	// ErrCoursewareAssistantPlanAssistantUnavailable 保留用于兼容历史错误识别。
	ErrCoursewareAssistantPlanAssistantUnavailable = errors.New(
		"选择的AI助手没有可用的教学提示词",
	)

	// ErrCoursewareAssistantPlanAIConfigUnavailable 表示AI配置不可用。
	ErrCoursewareAssistantPlanAIConfigUnavailable = errors.New(
		"课件教学智能体方案AI配置不可用",
	)

	// ErrCoursewareAssistantPlanCreditsInsufficient 是积分不足业务错误。
	ErrCoursewareAssistantPlanCreditsInsufficient = errors.New(
		"积分余额不足，无法生成课件教学智能体方案",
	)

	// ErrCoursewareAssistantPlanAICallFailed 表示模型或网络调用失败。
	ErrCoursewareAssistantPlanAICallFailed = errors.New(
		"课件教学智能体方案AI调用失败",
	)

	// ErrCoursewareAssistantPlanInvalidOutput 表示AI没有返回完整安全协议。
	ErrCoursewareAssistantPlanInvalidOutput = errors.New(
		"课件教学智能体方案AI输出无效",
	)
)

// CoursewareAssistantPlanService 是教师端方案生成服务。
type CoursewareAssistantPlanService struct {
	aesKey       string
	apiBaseURL   string
	apiKey       string
	defaultModel string

	coursewareService *CoursewareService
	contextService    *CoursewareAssistantContextService
	assistantService  *AIAssistantService
}

// NewCoursewareAssistantPlanService 创建默认方案生成服务。
func NewCoursewareAssistantPlanService(
	aesKey string,
	apiBaseURL string,
	apiKey string,
	defaultModel string,
) *CoursewareAssistantPlanService {
	return &CoursewareAssistantPlanService{
		aesKey:       aesKey,
		apiBaseURL:   apiBaseURL,
		apiKey:       apiKey,
		defaultModel: defaultModel,

		coursewareService: NewCoursewareService(),
		contextService:    NewCoursewareAssistantContextService(),
		assistantService:  NewAIAssistantService(),
	}
}

// NewCoursewareAssistantPlanServiceWithDependencies 创建可注入依赖的服务。
func NewCoursewareAssistantPlanServiceWithDependencies(
	aesKey string,
	apiBaseURL string,
	apiKey string,
	defaultModel string,
	coursewareService *CoursewareService,
	contextService *CoursewareAssistantContextService,
	assistantService *AIAssistantService,
) *CoursewareAssistantPlanService {
	return &CoursewareAssistantPlanService{
		aesKey:       aesKey,
		apiBaseURL:   apiBaseURL,
		apiKey:       apiKey,
		defaultModel: defaultModel,

		coursewareService: coursewareService,
		contextService:    contextService,
		assistantService:  assistantService,
	}
}

// GenerateCoursewareAssistantPlan 根据当前页和教学方式生成可编辑方案草稿。
//
// 返回成功不代表方案已经保存。
// 教师必须在后续插槽保存接口中明确确认并提交。
func (s *CoursewareAssistantPlanService) GenerateCoursewareAssistantPlan(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
	request *models.GenerateCoursewareAssistantPlanRequest,
) (
	*models.CoursewareAssistantPlanResult,
	error,
) {
	if s == nil {
		return nil,
			ErrCoursewareAssistantPlanServiceUnavailable
	}

	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	pageID = strings.TrimSpace(
		pageID,
	)

	if coursewareID == "" ||
		pageID == "" ||
		request == nil {
		return nil,
			ErrCoursewareAssistantInvalidRequest
	}

	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCoursewareAssistantActorRequired
	}

	normalizeCoursewareAssistantOptionalID(
		&request.AssistantID,
	)

	request.TeachingMode =
		models.NormalizeCoursewareAssistantTeachingMode(
			request.TeachingMode,
		)

	if !models.IsValidCoursewareAssistantTeachingMode(
		request.TeachingMode,
	) {
		return nil,
			coursewareAssistantInvalid(
				"教学方式无效",
			)
	}

	request.TeacherInstruction =
		strings.TrimSpace(
			request.TeacherInstruction,
		)

	if runeLength(
		request.TeacherInstruction,
	) >
		coursewareAssistantPlanTeacherInstructionMaxRunes {
		return nil,
			ErrCoursewareAssistantPlanInstructionTooLong
	}

	// 方案生成会消耗积分，并且生成结果预期随后可保存。
	// 因此先执行作者专属权限和审核写锁，避免无效AI消费。
	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil,
			mapCoursewareAssistantWriteAccessError(
				err,
			)
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil,
			ErrCoursewareAssistantMutationLocked
	}

	contextResult, err :=
		s.resolveContextService().
			BuildCoursewareAssistantContext(
				ctx,
				courseware.ID,
				pageID,
				scopedActor,
				models.DefaultCoursewareAssistantContextConfig(),
			)
	if err != nil {
		return nil, err
	}

	selectedAssistant, err :=
		s.loadCoursewareAssistantPlanAssistant(
			ctx,
			courseware,
			scopedActor,
			request.AssistantID,
		)
	if err != nil {
		return nil, err
	}

	schoolID, err :=
		repository.GetSchoolIDByUserID(
			ctx,
			scopedActor.UserID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareAssistantPlanSchoolResolveFailed,
				err,
			)
	}

	schoolID = strings.TrimSpace(
		schoolID,
	)

	if schoolID == "" {
		return nil,
			ErrCoursewareAssistantPlanSchoolRequired
	}

	systemPrompt :=
		loadCoursewareAssistantPlanSystemPrompt()

	userPrompt, err :=
		buildCoursewareAssistantPlanUserPrompt(
			courseware,
			contextResult,
			selectedAssistant,
			request.TeachingMode,
			request.TeacherInstruction,
		)
	if err != nil {
		return nil, err
	}

	aiConfig, err :=
		ai.GetEffectiveConfig(
			s.aesKey,
			coursewareAssistantPlanSceneCode,
			s.apiBaseURL,
			s.apiKey,
			s.defaultModel,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareAssistantPlanAIConfigUnavailable,
				err,
			)
	}

	userIDSnapshot :=
		scopedActor.UserID

	schoolIDSnapshot :=
		schoolID

	traceContext :=
		&ai.TraceContext{
			SceneCode: coursewareAssistantPlanSceneCode,
			UserID:    &userIDSnapshot,
			SchoolID:  &schoolIDSnapshot,
		}

	callResult, err :=
		ai.CallAI(
			aiConfig,
			systemPrompt,
			userPrompt,
			traceContext,
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantPlanAIError(
				err,
			)
	}

	plan, parseErr :=
		parseCoursewareAssistantPlanAIResult(
			callResult.Content,
			request.TeachingMode,
		)
	if parseErr == nil {
		return plan, nil
	}

	if !errors.Is(
		parseErr,
		ErrCoursewareAssistantPlanInvalidOutput,
	) {
		return nil, parseErr
	}

	// 模型偶发会返回多余字段、错误步骤数量或不完整JSON。
	// 不保存第一次原始响应，也不把原始响应再次拼回提示词；
	// 仅增加确定性的协议纠正要求，再自动生成一次。
	correctionPrompt :=
		buildCoursewareAssistantPlanCorrectionPrompt(
			userPrompt,
			parseErr,
		)

	correctionResult, err :=
		ai.CallAI(
			aiConfig,
			systemPrompt,
			correctionPrompt,
			traceContext,
		)
	if err != nil {
		return nil,
			mapCoursewareAssistantPlanAIError(
				err,
			)
	}

	correctedPlan, correctedErr :=
		parseCoursewareAssistantPlanAIResult(
			correctionResult.Content,
			request.TeachingMode,
		)
	if correctedErr != nil {
		return nil,
			fmt.Errorf(
				"%w: 自动纠正后仍未通过校验: %v",
				ErrCoursewareAssistantPlanInvalidOutput,
				correctedErr,
			)
	}

	return correctedPlan, nil
}

// buildCoursewareAssistantPlanCorrectionPrompt 为第二次生成增加严格纠正约束。
func buildCoursewareAssistantPlanCorrectionPrompt(
	originalPrompt string,
	firstError error,
) string {
	hint :=
		coursewareAssistantPlanCorrectionHint(
			firstError,
		)

	return strings.TrimSpace(
		originalPrompt,
	) + `

【自动纠正要求】
上一次生成结果没有通过系统的结构化安全校验。请重新从头生成，不要解释错误，不要引用或复述上一次内容。

必须做到：
1. 只输出一个完整JSON对象，不使用Markdown代码块，不输出任何JSON之外的文字。
2. 只能使用协议中定义的字段，不增加字段，不省略字段。
3. teaching_mode必须与教师选择完全一致。
4. question_chain必须为4至8步，每步必须有1至3层提示。
5. 所有步骤编号和学习困难分支引用必须真实存在且相互一致。
6. require_student_try必须为true，direct_answer_allowed必须为false，maximum_hint_level不能超过3。
7. 数组数量必须保持单页课堂合理规模。

本次重点纠正：` + hint
}

// coursewareAssistantPlanCorrectionHint 把内部校验错误收敛为模型可执行提示。
func coursewareAssistantPlanCorrectionHint(
	err error,
) string {
	message := ""

	if err != nil {
		message = err.Error()
	}

	switch {
	case strings.Contains(
		message,
		"JSON",
	),
		strings.Contains(
			message,
			"合法的JSON",
		):
		return "返回严格、完整且可直接解析的JSON对象。"

	case strings.Contains(
		message,
		"教学方式",
	):
		return "保持教师选择的教学方式，不要改成其他教学方式。"

	case strings.Contains(
		message,
		"互动步骤",
	):
		return "将互动步骤控制在4至8个，并保证每个步骤字段完整。"

	case strings.Contains(
		message,
		"提示",
	):
		return "每个互动步骤只设置1至3层由弱到强的提示。"

	case strings.Contains(
		message,
		"学习困难",
	),
		strings.Contains(
			message,
			"分支",
		):
		return "减少学习困难分支，并确保步骤引用的分支ID真实存在。"

	default:
		return "严格遵守全部字段、数量、引用和答案保护协议。"
	}
}

// resolveCoursewareService 返回可用课件服务。
func (s *CoursewareAssistantPlanService) resolveCoursewareService() *CoursewareService {
	if s != nil &&
		s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}

// resolveContextService 返回可用确定性上下文服务。
func (s *CoursewareAssistantPlanService) resolveContextService() *CoursewareAssistantContextService {
	if s != nil &&
		s.contextService != nil {
		return s.contextService
	}

	return NewCoursewareAssistantContextService()
}

// resolvePlanAssistantService 返回可用AI助手服务。
func (s *CoursewareAssistantPlanService) resolvePlanAssistantService() *AIAssistantService {
	if s != nil &&
		s.assistantService != nil {
		return s.assistantService
	}

	return NewAIAssistantService()
}

// loadCoursewareAssistantPlanAssistant 加载历史页面仍然有效的可选风格助手。
//
// 普通教师流程不再要求选择助手。
// 历史助手不存在、停用、无权使用或范围不匹配时，安全回退到系统默认风格。
// 只有数据库等未知服务错误继续向上返回。
func (s *CoursewareAssistantPlanService) loadCoursewareAssistantPlanAssistant(
	ctx context.Context,
	courseware *models.Courseware,
	scopedActor *CoursewareActorContext,
	assistantID *string,
) (
	*models.AIAssistant,
	error,
) {
	if assistantID == nil ||
		strings.TrimSpace(
			*assistantID,
		) == "" {
		return nil, nil
	}

	assistant, err :=
		s.resolvePlanAssistantService().
			ValidateAssistantForManualLesson(
				ctx,
				scopedActor,
				strings.TrimSpace(
					*assistantID,
				),
				strings.TrimSpace(
					courseware.Subject,
				),
				CoursewareAssistantSelectionScene,
			)
	if err != nil {
		mapped :=
			mapCoursewareAssistantSelectionError(
				err,
			)

		if coursewareAssistantOptionalStyleUnavailable(
			mapped,
		) {
			return nil, nil
		}

		return nil, mapped
	}

	if assistant == nil ||
		strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
		return nil, nil
	}

	return assistant, nil
}

// coursewareAssistantOptionalStyleUnavailable 判断历史风格助手是否可安全忽略。
func coursewareAssistantOptionalStyleUnavailable(
	err error,
) bool {
	return errors.Is(
		err,
		ErrCoursewareAssistantAssistantNotFound,
	) ||
		errors.Is(
			err,
			ErrCoursewareAssistantAssistantInactive,
		) ||
		errors.Is(
			err,
			ErrCoursewareAssistantAssistantDomainMismatch,
		) ||
		errors.Is(
			err,
			ErrCoursewareAssistantAssistantScopeMismatch,
		) ||
		errors.Is(
			err,
			ErrCoursewareAssistantAssistantUseDenied,
		)
}

// mapCoursewareAssistantPlanAIError 把积分错误与普通AI故障分开。
func mapCoursewareAssistantPlanAIError(
	err error,
) error {
	if err == nil {
		return nil
	}

	message :=
		strings.TrimSpace(
			err.Error(),
		)

	if strings.Contains(
		message,
		"积分余额不足",
	) ||
		strings.Contains(
			message,
			"余额不足",
		) {
		return fmt.Errorf(
			"%w: %v",
			ErrCoursewareAssistantPlanCreditsInsufficient,
			err,
		)
	}

	return fmt.Errorf(
		"%w: %v",
		ErrCoursewareAssistantPlanAICallFailed,
		err,
	)
}
