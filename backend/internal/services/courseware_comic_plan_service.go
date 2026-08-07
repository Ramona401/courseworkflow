package services

// courseware_comic_plan_service.go — 知识点漫画AI规划服务
//
// 完整流程：
//   1. 校验可信教师Actor和请求版本；
//   2. 重新读取作者自己的正式课件并收敛到课件教育域快照；
//   3. 校验课件审核锁、Pipeline锁和K12教育域；
//   4. 按课件、项目和创建者三重边界读取漫画项目；
//   5. 校验项目核心知识、学科和年级快照；
//   6. 解析本轮叙事方式，空值沿用项目当前值；
//   7. 服务端加载可选漫画助手完整提示词；
//   8. 服务端解析真实学校ID和AI配置；
//   9. 使用版本CAS原子保存叙事方式并领取planning状态；
//  10. 在planning锁定状态下读取受限参考资源上下文；
//  11. 调用统一AI客户端并执行积分检查、模型分流和追踪；
//  12. 严格解析人物和4至8格分镜；
//  13. 后端确定性生成IAOCI、稳定图片键和自动排版覆盖层；
//  14. 保存项目级人物设定，再原子替换全部分格。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const coursewareComicPlanTeacherInstructionMaxRunes =
	8000

var (
	ErrCoursewareComicPlanServiceUnavailable = errors.New(
		"知识点漫画规划服务不可用",
	)

	ErrCoursewareComicPlanActorRequired = errors.New(
		"知识点漫画规划需要登录教师身份",
	)

	ErrCoursewareComicPlanInvalidRequest = errors.New(
		"知识点漫画规划请求无效",
	)

	ErrCoursewareComicPlanInstructionTooLong = errors.New(
		"知识点漫画补充要求长度超过上限",
	)

	ErrCoursewareComicPlanContextInvalid = errors.New(
		"知识点漫画项目上下文无效",
	)

	ErrCoursewareComicPlanMutationLocked = errors.New(
		"当前课件正在审核或生产流程中，不能规划知识点漫画",
	)

	ErrCoursewareComicPlanK12Required = errors.New(
		"知识点漫画第一版仅支持K12课件",
	)

	ErrCoursewareComicPlanAssistantUnavailable = errors.New(
		"选择的漫画助手不可用或没有有效提示词",
	)

	ErrCoursewareComicPlanSchoolRequired = errors.New(
		"当前教师未绑定可用学校，不能生成知识点漫画规划",
	)

	ErrCoursewareComicPlanSchoolResolveFailed = errors.New(
		"解析教师学校失败",
	)

	ErrCoursewareComicPlanAIConfigUnavailable = errors.New(
		"知识点漫画规划AI配置不可用",
	)

	ErrCoursewareComicPlanCreditsInsufficient = errors.New(
		"积分余额不足，无法生成知识点漫画规划",
	)

	ErrCoursewareComicPlanAICallFailed = errors.New(
		"知识点漫画规划AI调用失败",
	)

	ErrCoursewareComicPlanInvalidOutput = errors.New(
		"知识点漫画规划AI输出无效",
	)
)

// CoursewareComicPlanService 是知识点漫画AI规划服务。
type CoursewareComicPlanService struct {
	aesKey       string
	apiBaseURL   string
	apiKey       string
	defaultModel string

	coursewareService *CoursewareService
	assistantService  *AIAssistantService
}

// NewCoursewareComicPlanService 创建默认服务。
func NewCoursewareComicPlanService(
	aesKey string,
	apiBaseURL string,
	apiKey string,
	defaultModel string,
) *CoursewareComicPlanService {
	return &CoursewareComicPlanService{
		aesKey:            aesKey,
		apiBaseURL:        apiBaseURL,
		apiKey:            apiKey,
		defaultModel:      defaultModel,
		coursewareService: NewCoursewareService(),
		assistantService:  NewAIAssistantService(),
	}
}

// NewCoursewareComicPlanServiceWithDependencies
// 创建可注入依赖的服务。
func NewCoursewareComicPlanServiceWithDependencies(
	aesKey string,
	apiBaseURL string,
	apiKey string,
	defaultModel string,
	coursewareService *CoursewareService,
	assistantService *AIAssistantService,
) *CoursewareComicPlanService {
	return &CoursewareComicPlanService{
		aesKey:            aesKey,
		apiBaseURL:        apiBaseURL,
		apiKey:            apiKey,
		defaultModel:      defaultModel,
		coursewareService: coursewareService,
		assistantService:  assistantService,
	}
}

// resolveCoursewareComicPlanNarrativeMode
// 解析本轮规划使用的叙事方式。
func resolveCoursewareComicPlanNarrativeMode(
	requested string,
	current string,
) (string, error) {
	requested =
		strings.TrimSpace(
			requested,
		)

	if requested != "" {
		if !models.IsValidCWComicNarrativeMode(
			requested,
		) {
			return "",
				ErrCoursewareComicPlanInvalidRequest
		}

		return requested, nil
	}

	current =
		strings.TrimSpace(
			current,
		)

	if !models.IsValidCWComicNarrativeMode(
		current,
	) {
		return "",
			ErrCoursewareComicPlanContextInvalid
	}

	return current, nil
}

// PlanProject
// 为已有漫画项目生成完整人物、分镜、IAOCI和自动排版。
func (s *CoursewareComicPlanService) PlanProject(
	ctx context.Context,
	coursewareID string,
	projectID string,
	actor *CoursewareActorContext,
	request *models.PlanCoursewareComicRequest,
) (*models.CoursewareComicPlanResult, error) {
	if s == nil {
		return nil,
			ErrCoursewareComicPlanServiceUnavailable
	}

	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		request == nil ||
		request.ExpectedVersion < 1 {
		return nil,
			ErrCoursewareComicPlanInvalidRequest
	}

	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCoursewareComicPlanActorRequired
	}

	request.TeacherInstruction =
		strings.TrimSpace(
			request.TeacherInstruction,
		)

	request.NarrativeMode =
		strings.TrimSpace(
			request.NarrativeMode,
		)

	if utf8.RuneCountInString(
		request.TeacherInstruction,
	) >
		coursewareComicPlanTeacherInstructionMaxRunes {
		return nil,
			ErrCoursewareComicPlanInstructionTooLong
	}

	courseware, scopedActor, err :=
		s.resolveCoursewareService().
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if err :=
		validateCoursewareControlMutationState(
			courseware,
		); err != nil {
		return nil,
			ErrCoursewareComicPlanMutationLocked
	}

	if strings.ToLower(
		strings.TrimSpace(
			courseware.EducationDomain,
		),
	) !=
		models.EducationDomainK12 {
		return nil,
			ErrCoursewareComicPlanK12Required
	}

	project, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	if project.Version !=
		request.ExpectedVersion {
		return nil,
			repository.
				ErrCoursewareComicProjectConflict
	}

	if !models.IsEditableCWComicProjectStatus(
		project.Status,
	) {
		return nil,
			repository.
				ErrCoursewareComicProjectNotEditable
	}

	if strings.ToLower(
		strings.TrimSpace(
			project.EducationDomain,
		),
	) !=
		models.EducationDomainK12 ||
		strings.TrimSpace(
			project.Subject,
		) !=
			strings.TrimSpace(
				courseware.Subject,
			) ||
		strings.TrimSpace(
			project.Grade,
		) !=
			strings.TrimSpace(
				courseware.Grade,
			) {
		return nil,
			ErrCoursewareComicPlanContextInvalid
	}

	if strings.TrimSpace(
		project.PublisherSnapshot,
	) == "" ||
		project.TextbookUnitID == nil ||
		strings.TrimSpace(
			*project.TextbookUnitID,
		) == "" ||
		strings.TrimSpace(
			project.KnowledgeContentSnapshot,
		) == "" {
		return nil,
			ErrCoursewareComicPlanContextInvalid
	}

	narrativeMode, err :=
		resolveCoursewareComicPlanNarrativeMode(
			request.NarrativeMode,
			project.NarrativeMode,
		)
	if err != nil {
		return nil, err
	}

	selectedAssistant, err :=
		s.loadCoursewareComicAssistant(
			ctx,
			courseware,
			scopedActor,
			project.AssistantID,
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
				ErrCoursewareComicPlanSchoolResolveFailed,
				err,
			)
	}

	schoolID =
		strings.TrimSpace(
			schoolID,
		)

	if schoolID == "" {
		return nil,
			ErrCoursewareComicPlanSchoolRequired
	}

	systemPrompt :=
		loadCoursewareComicPlanSystemPrompt()

	aiConfig, err :=
		ai.GetEffectiveConfig(
			s.aesKey,
			models.SceneCoursewareComicPlan,
			s.apiBaseURL,
			s.apiKey,
			s.defaultModel,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareComicPlanAIConfigUnavailable,
				err,
			)
	}

	// 先使用版本CAS领取planning状态。
	//
	// 参考资源新增和删除都必须锁定同一项目行，并且只允许可编辑状态。
	// 项目进入planning后再读取参考资源，可以保证本轮AI看到的是稳定集合。
	planningProject, err :=
		repository.
			BeginCoursewareComicProjectPlanningWithNarrative(
				ctx,
				coursewareID,
				projectID,
				scopedActor.UserID,
				request.ExpectedVersion,
				narrativeMode,
			)
	if err != nil {
		return nil, err
	}

	references, err :=
		loadCoursewareComicReferencePromptContext(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil, err
	}

	userPrompt, err :=
		buildCoursewareComicPlanUserPrompt(
			courseware,
			planningProject,
			selectedAssistant,
			request.TeacherInstruction,
			references,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil, err
	}

	userIDSnapshot :=
		scopedActor.UserID

	schoolIDSnapshot :=
		schoolID

	traceContext :=
		&ai.TraceContext{
			SceneCode:
				models.SceneCoursewareComicPlan,
			UserID:
				&userIDSnapshot,
			SchoolID:
				&schoolIDSnapshot,
		}

	callResult, err :=
		ai.CallAI(
			aiConfig,
			systemPrompt,
			userPrompt,
			traceContext,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil,
			mapCoursewareComicPlanAIError(
				err,
			)
	}

	if callResult == nil ||
		strings.TrimSpace(
			callResult.Content,
		) == "" {
		emptyErr :=
			errors.New(
				"AI未返回漫画规划内容",
			)

		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			emptyErr,
		)

		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareComicPlanAICallFailed,
				emptyErr,
			)
	}

	parsedPlan, err :=
		parseCoursewareComicPlanAIResult(
			callResult.Content,
			planningProject,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil, err
	}

	projectWithPlan, err :=
		repository.SaveCoursewareComicProjectPlanningResult(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			planningProject.Version,
			parsedPlan.StyleAOCIText,
			parsedPlan.CharacterBibleJSON,
			parsedPlan.ContinuityLedgerJSON,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil, err
	}

	panels, err :=
		repository.ReplaceCoursewareComicPanels(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			projectWithPlan.Version,
			parsedPlan.Panels,
		)
	if err != nil {
		s.markPlanningFailed(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
			err,
		)

		return nil, err
	}

	finalProject, err :=
		repository.GetCoursewareComicProjectByIDForUser(
			ctx,
			coursewareID,
			projectID,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return &models.CoursewareComicPlanResult{
		Project:
			finalProject,
		Panels:
			panels,
	}, nil
}

// loadCoursewareComicAssistant
// 加载可选漫画助手完整提示词。
func (s *CoursewareComicPlanService) loadCoursewareComicAssistant(
	ctx context.Context,
	courseware *models.Courseware,
	scopedActor *CoursewareActorContext,
	assistantID *string,
) (*models.AIAssistant, error) {
	if assistantID == nil ||
		strings.TrimSpace(
			*assistantID,
		) == "" {
		return nil, nil
	}

	assistant, err :=
		s.resolveAssistantService().
			ValidateAssistantForManualLesson(
				ctx,
				scopedActor,
				strings.TrimSpace(
					*assistantID,
				),
				strings.TrimSpace(
					courseware.Subject,
				),
				models.SceneCoursewareComicPlan,
			)
	if err != nil {
		return nil, err
	}

	if assistant == nil ||
		strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
		return nil,
			ErrCoursewareComicPlanAssistantUnavailable
	}

	return assistant, nil
}

// markPlanningFailed
// 在AI调用、参考上下文装配或解析失败后把planning收敛为failed。
func (s *CoursewareComicPlanService) markPlanningFailed(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	failure error,
) {
	message :=
		"知识点漫画规划失败"

	if failure != nil &&
		strings.TrimSpace(
			failure.Error(),
		) != "" {
		message =
			strings.TrimSpace(
				failure.Error(),
			)
	}

	_, _ =
		repository.TransitionCoursewareComicProjectStatus(
			ctx,
			coursewareID,
			projectID,
			userID,
			[]string{
				models.CWComicProjectStatusPlanning,
			},
			models.CWComicProjectStatusFailed,
			message,
		)
}

func (s *CoursewareComicPlanService) resolveCoursewareService() *CoursewareService {
	if s != nil &&
		s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}

func (s *CoursewareComicPlanService) resolveAssistantService() *AIAssistantService {
	if s != nil &&
		s.assistantService != nil {
		return s.assistantService
	}

	return NewAIAssistantService()
}

// mapCoursewareComicPlanAIError
// 区分积分不足与普通模型调用失败。
func mapCoursewareComicPlanAIError(
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
			ErrCoursewareComicPlanCreditsInsufficient,
			err,
		)
	}

	return fmt.Errorf(
		"%w: %v",
		ErrCoursewareComicPlanAICallFailed,
		err,
	)
}
