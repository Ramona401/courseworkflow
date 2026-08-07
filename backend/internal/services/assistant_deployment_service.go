package services

// assistant_deployment_service.go
//
// 本文件把教师编辑态插槽发布为可运行的不可变部署版本，并提供教师端部署管理服务。
//
// 首次发布：
//   1. 重新加载作者自己的正式课件并收敛到课件历史教育域；
//   2. 校验课件生产状态和审核写锁；
//   3. 重新读取当前页插槽并验证active状态；
//   4. 历史页面仍绑定可用AI助手时，将其作为可选教学风格；
//   5. 未绑定助手或历史助手失效时，使用系统内置页面教学风格；
//   6. 服务端解析教师当前学校ID；
//   7. 重新构建页面上下文和HTML哈希；
//   8. 生成完整版本快照；
//   9. 由Repository在单一事务中创建部署和版本1。
//
// 新版本发布重复执行同一套重读和校验，不信任旧插槽响应或前端快照。
// 暂停、恢复、撤销和策略更新只操作部署主记录，不修改历史版本。
//
// 本文件不调用AI、不修改课件HTML、不注册HTTP路由。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// AssistantDeploymentService 是教师端部署发布与管理服务。
type AssistantDeploymentService struct {
	coursewareService *CoursewareService
	slotService       *CoursewareAssistantSlotService
	contextService    *CoursewareAssistantContextService
	assistantService  *AIAssistantService
}

// NewAssistantDeploymentService 创建默认部署服务。
func NewAssistantDeploymentService() *AssistantDeploymentService {
	return &AssistantDeploymentService{
		coursewareService: NewCoursewareService(),
		slotService:       NewCoursewareAssistantSlotService(),
		contextService:    NewCoursewareAssistantContextService(),
		assistantService:  NewAIAssistantService(),
	}
}

// NewAssistantDeploymentServiceWithDependencies 创建可注入依赖的部署服务。
func NewAssistantDeploymentServiceWithDependencies(
	coursewareService *CoursewareService,
	slotService *CoursewareAssistantSlotService,
	contextService *CoursewareAssistantContextService,
	assistantService *AIAssistantService,
) *AssistantDeploymentService {
	return &AssistantDeploymentService{
		coursewareService: coursewareService,
		slotService:       slotService,
		contextService:    contextService,
		assistantService:  assistantService,
	}
}

// PublishAssistantDeployment 首次发布当前页插槽。
func (s *AssistantDeploymentService) PublishAssistantDeployment(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
	request *models.CreateAssistantDeploymentRequest,
) (
	*models.AssistantDeploymentView,
	error,
) {
	if err := validateAssistantDeploymentActor(
		actor,
	); err != nil {
		return nil, err
	}

	if request == nil {
		return nil,
			ErrAssistantDeploymentPolicyInvalid
	}

	policy, err :=
		normalizeAssistantDeploymentPolicy(
			request.DailyCallLimit,
			request.PerSessionTurnLimit,
			request.AllowedOrigins,
			request.ValidUntil,
			time.Now().UTC(),
		)
	if err != nil {
		return nil, err
	}

	courseware,
		scopedActor,
		slot,
		assistant,
		contextResult,
		err := s.loadAssistantDeploymentPublishSources(
		ctx,
		coursewareID,
		pageID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	liveDeployment, liveErr :=
		repository.GetLiveAssistantDeploymentByPageForOwner(
			ctx,
			courseware.ID,
			slot.PageID,
			scopedActor.UserID,
		)

	if liveErr == nil &&
		liveDeployment != nil {
		return nil,
			repository.ErrAssistantDeploymentPageAlreadyLive
	}

	if liveErr != nil &&
		!errors.Is(
			liveErr,
			repository.ErrAssistantDeploymentNotFound,
		) {
		return nil, liveErr
	}

	schoolID, err :=
		resolveAssistantDeploymentSchoolID(
			ctx,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	version, err :=
		buildAssistantDeploymentVersionRecord(
			courseware,
			slot,
			assistant,
			contextResult,
			policy,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	slotID := strings.TrimSpace(
		slot.ID,
	)

	deployment :=
		&models.AssistantDeployment{
			SlotID: &slotID,
			CoursewareID: strings.TrimSpace(
				courseware.ID,
			),
			PageID: strings.TrimSpace(
				slot.PageID,
			),
			OwnerUserID: strings.TrimSpace(
				scopedActor.UserID,
			),
			SchoolID: schoolID,
			EducationDomain: strings.ToLower(
				strings.TrimSpace(
					courseware.EducationDomain,
				),
			),
			AccessMode:          models.AssistantDeploymentAccessOriginAllowlist,
			Status:              models.AssistantDeploymentStatusActive,
			DailyCallLimit:      policy.DailyCallLimit,
			PerSessionTurnLimit: policy.PerSessionTurnLimit,
			AllowedOriginsJSON:  policy.AllowedOriginsJSON,
			ValidUntil:          policy.ValidUntil,
		}

	if err :=
		repository.CreateAssistantDeploymentWithFirstVersion(
			ctx,
			deployment,
			version,
		); err != nil {
		return nil, err
	}

	return assistantDeploymentViewFromRecord(
		deployment,
	)
}

// PublishAssistantDeploymentVersion 从当前插槽追加一个不可变版本。
func (s *AssistantDeploymentService) PublishAssistantDeploymentVersion(
	ctx context.Context,
	deploymentID string,
	actor *CoursewareActorContext,
) (
	*models.AssistantDeploymentVersionView,
	error,
) {
	deployment, err :=
		s.loadOwnedAssistantDeployment(
			ctx,
			deploymentID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if deployment.Status ==
		models.AssistantDeploymentStatusRevoked {
		return nil,
			repository.ErrAssistantDeploymentRevoked
	}

	courseware,
		scopedActor,
		slot,
		assistant,
		contextResult,
		err := s.loadAssistantDeploymentPublishSources(
		ctx,
		deployment.CoursewareID,
		deployment.PageID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	if deployment.SlotID == nil ||
		strings.TrimSpace(
			*deployment.SlotID,
		) == "" ||
		strings.TrimSpace(
			*deployment.SlotID,
		) !=
			strings.TrimSpace(
				slot.ID,
			) {
		return nil,
			ErrAssistantDeploymentSlotChanged
	}

	storedOrigins, err :=
		assistantDeploymentAllowedOriginsFromJSON(
			deployment.AllowedOriginsJSON,
		)
	if err != nil {
		return nil, err
	}

	policy, err :=
		normalizeAssistantDeploymentPolicy(
			deployment.DailyCallLimit,
			deployment.PerSessionTurnLimit,
			storedOrigins,
			deployment.ValidUntil,
			time.Now().UTC(),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrAssistantDeploymentStoredPolicyInvalid,
				err,
			)
	}

	version, err :=
		buildAssistantDeploymentVersionRecord(
			courseware,
			slot,
			assistant,
			contextResult,
			policy,
			scopedActor.UserID,
		)
	if err != nil {
		return nil, err
	}

	created, err :=
		repository.AppendAssistantDeploymentVersion(
			ctx,
			deployment.ID,
			deployment.CoursewareID,
			deployment.PageID,
			deployment.OwnerUserID,
			version,
		)
	if err != nil {
		return nil, err
	}

	return assistantDeploymentVersionViewFromRecord(
		created,
	), nil
}

// loadAssistantDeploymentPublishSources 重读并校验发布所需全部来源。
func (s *AssistantDeploymentService) loadAssistantDeploymentPublishSources(
	ctx context.Context,
	coursewareID string,
	pageID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	*models.CoursewareAssistantSlotView,
	*models.AIAssistant,
	*CoursewareAssistantContextBuildResult,
	error,
) {
	if err := validateAssistantDeploymentActor(
		actor,
	); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	courseware,
		scopedActor,
		err := s.resolveCoursewareService().
		LoadCoursewareForOwnerRuntime(
			ctx,
			strings.TrimSpace(
				coursewareID,
			),
			actor,
		)
	if err != nil {
		return nil, nil, nil, nil, nil,
			mapCoursewareAssistantWriteAccessError(
				err,
			)
	}

	if err :=
		validateAssistantDeploymentPublishableCourseware(
			courseware,
		); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	slot, err :=
		s.resolveSlotService().
			GetCoursewareAssistantSlotByPage(
				ctx,
				courseware.ID,
				strings.TrimSpace(
					pageID,
				),
				scopedActor,
			)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if err :=
		validateAssistantDeploymentSlot(
			slot,
			courseware.ID,
			pageID,
		); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	assistant, err :=
		s.loadAssistantDeploymentOptionalAssistant(
			ctx,
			courseware,
			scopedActor,
			slot.AssistantID,
		)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	contextResult, err :=
		s.resolveContextService().
			BuildCoursewareAssistantContext(
				ctx,
				courseware.ID,
				slot.PageID,
				scopedActor,
				slot.ContextConfig,
			)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return courseware,
		scopedActor,
		slot,
		assistant,
		contextResult,
		nil
}

// loadAssistantDeploymentOptionalAssistant 加载历史页面仍然有效的风格助手。
//
// 已有助手不再是发布前置条件。
// 已删除、停用、无权使用、教育域或学科不匹配时，安全回退到默认教学风格。
// 数据库等未知服务错误仍然向上返回，避免掩盖基础设施故障。
func (s *AssistantDeploymentService) loadAssistantDeploymentOptionalAssistant(
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

	if courseware == nil ||
		scopedActor == nil {
		return nil,
			ErrAssistantDeploymentSnapshotInvalid
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
			assistant.ID,
		) == "" ||
		strings.TrimSpace(
			assistant.FullPrompt,
		) == "" {
		return nil, nil
	}

	return assistant, nil
}

// resolveAssistantDeploymentSchoolID 从服务端组织关系解析发布学校。
func resolveAssistantDeploymentSchoolID(
	ctx context.Context,
	userID string,
) (
	string,
	error,
) {
	schoolID, err :=
		repository.GetSchoolIDByUserID(
			ctx,
			strings.TrimSpace(
				userID,
			),
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"解析教学智能体部署学校失败: %w",
				err,
			)
	}

	schoolID =
		strings.TrimSpace(
			schoolID,
		)

	if schoolID == "" {
		return "",
			ErrAssistantDeploymentSchoolRequired
	}

	return schoolID, nil
}

// resolveCoursewareService 返回可用课件服务。
func (s *AssistantDeploymentService) resolveCoursewareService() *CoursewareService {
	if s != nil &&
		s.coursewareService != nil {
		return s.coursewareService
	}

	return NewCoursewareService()
}

// resolveSlotService 返回可用插槽服务。
func (s *AssistantDeploymentService) resolveSlotService() *CoursewareAssistantSlotService {
	if s != nil &&
		s.slotService != nil {
		return s.slotService
	}

	return NewCoursewareAssistantSlotService()
}

// resolveContextService 返回可用上下文服务。
func (s *AssistantDeploymentService) resolveContextService() *CoursewareAssistantContextService {
	if s != nil &&
		s.contextService != nil {
		return s.contextService
	}

	return NewCoursewareAssistantContextService()
}

// resolveAssistantService 返回可用AI助手校验服务。
func (s *AssistantDeploymentService) resolveAssistantService() *AIAssistantService {
	if s != nil &&
		s.assistantService != nil {
		return s.assistantService
	}

	return NewAIAssistantService()
}
