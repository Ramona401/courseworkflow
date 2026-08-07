package services

// courseware_access_service.go — 课件教育域可信Actor与访问控制底座
//
// 本文件负责课件查看、编辑与作者运行通道的教育域安全边界。
// 创建、Fork和Actor快照公共函数已拆分到courseware_creation_access.go。
// 本文件不替代课件原有的：
//   - 作者归属权限；
//   - 共享课件可见范围；
//   - 审核员审核权限；
//   - 集体备课参与者微调权限；
//   - 状态机编辑限制。
//
// 设计原则：
//   1. 复用已有AssistantActorContext及BuildActorFromClaims，避免重新实现学校、
//      教研组和教育域解析，防止不同资源使用不同身份口径。
//   2. 新建无教案来源课件时，只接受k12、vocational、adult具体教学域。
//      mixed、common、空值和非法值一律在应用层拒绝，不能依赖数据库回退K12。
//   3. 读取已有课件时，以coursewares.education_domain创建快照为准。
//   4. mixed只属于跨域管理上下文，且仅允许指定管理角色持有。
//   5. 进入具体课件生成、微调、素材等运行链时，将Actor教育域收敛为课件快照域，
//      防止mixed管理员在单份课件内部继续跨域加载其它教学资源。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"tedna/internal/repository"

	"tedna/internal/models"
)

// 课件教育域访问控制错误。
//
// Handler接线后应按以下口径映射：
//   - ErrCoursewareActorRequired：401或403；
//   - ErrCoursewareCreationDomainRequired：403；
//   - ErrCoursewareEducationDomainInvalid：500，表示存量数据或数据库异常；
//   - ErrCoursewareEducationDomainMismatch：403；
//   - ErrCoursewareRuntimeDomainRequired：500，表示具体课件缺少可运行的教学域快照。
var (
	ErrCoursewareActorRequired = errors.New(
		"缺少可信课件操作上下文",
	)
	ErrCoursewareAccessNotFound = errors.New(
		"课件不存在",
	)
	ErrCoursewareViewDenied = errors.New(
		"无权查看此课件",
	)
	ErrCoursewareEditDenied = errors.New(
		"无权编辑此课件",
	)
	ErrCoursewareOwnerRuntimeDenied = errors.New(
		"只有课件作者本人可以执行此操作",
	)
	ErrCoursewareEducationDomainInvalid = errors.New(
		"课件教育域无效",
	)
	ErrCoursewareEducationDomainMismatch = errors.New(
		"课件教育域与当前教学域不匹配",
	)
	ErrCoursewareRuntimeDomainRequired = errors.New(
		"课件缺少可运行的具体教学教育域",
	)
)

// CoursewareActorContext 课件可信操作者上下文。
//
// 直接复用AssistantActorContext，统一使用以下已经稳定运行的字段：
//   - UserID、Role；
//   - SchoolID；
//   - EducationDomain；
//   - MyGroupIDs；
//   - MyLeadGroupIDs；
//   - MyLeadOrBackboneGroupIDs。
//
// 使用类型别名而不是重新定义结构体，可以保证助手、配方和课件三条资源链
// 永远共享同一套组织身份与教育域解析规则。
type CoursewareActorContext = AssistantActorContext

// isCoursewareMixedReadManagementRole
// 判断显式mixed Actor能否进入跨教育域管理读取通道。
//
// 该判断只用于课件和课件组件的管理读取：
//   - admin、region_admin、district_inspector允许；
//   - 不授予个人教学资源创建权限；
//   - 普通角色伪造mixed仍然拒绝。
func isCoursewareMixedReadManagementRole(
	role string,
) bool {
	switch strings.TrimSpace(role) {
	case models.RoleAdmin,
		models.RoleRegionAdmin,
		models.RoleDistrictInspector:
		return true
	default:
		return false
	}
}

// ValidateCoursewareEducationDomainForActor 校验Actor是否可以进入某份课件的教育域。
//
// 本函数只判断教育域，不判断作者、共享、审核或集体备课权限。
// 调用方必须同时执行原有业务权限校验，不能把本函数当成完整课件授权。
//
// 规则：
//   - k12、vocational、adult Actor只可访问同域或common资源；
//   - mixed管理Actor可访问全部合法资源域；
//   - mixed只有admin、region_admin、district_inspector可以持有；
//   - Actor为空、current=common、当前域非法、资源域非法均拒绝。
func ValidateCoursewareEducationDomainForActor(
	actor *CoursewareActorContext,
	courseware *models.Courseware,
) error {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return ErrCoursewareActorRequired
	}
	if courseware == nil {
		return ErrCoursewareEducationDomainInvalid
	}

	resourceDomain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	if !models.IsResourceEducationDomain(resourceDomain) {
		return ErrCoursewareEducationDomainInvalid
	}

	currentDomain := strings.ToLower(
		strings.TrimSpace(actor.EducationDomain),
	)

	// ResourceEducationDomainMatches本身允许mixed跨域管理。
	// 此处额外校验角色，防止异常调用方手工构造普通角色+mixed的伪Actor。
	if currentDomain == models.EducationDomainMixed &&
		!isCoursewareMixedReadManagementRole(actor.Role) {
		return ErrCoursewareEducationDomainMismatch
	}

	if !models.ResourceEducationDomainMatches(
		resourceDomain,
		currentDomain,
	) {
		return ErrCoursewareEducationDomainMismatch
	}

	return nil
}

// validateCoursewareDomainForAuthorizedActor 校验已通过业务身份候选判断的Actor与课件域。
//
// 作者例外：
//
//	课件创建后教育域快照不随作者换校变化。作者本人仍可访问自己的合法历史课件，
//	后续进入编辑/生成链时再把Actor收敛到该历史快照域。
//
// 非作者：
//
//	必须严格通过ValidateCoursewareEducationDomainForActor，不能跨域共享、协作或审核。
func validateCoursewareDomainForAuthorizedActor(
	actor *CoursewareActorContext,
	courseware *models.Courseware,
	allowOwnerSnapshot bool,
) error {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return ErrCoursewareActorRequired
	}
	if courseware == nil {
		return ErrCoursewareEducationDomainInvalid
	}

	resourceDomain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	if !models.IsResourceEducationDomain(resourceDomain) {
		return ErrCoursewareEducationDomainInvalid
	}

	if allowOwnerSnapshot &&
		courseware.UserID == actor.UserID {
		return nil
	}

	return ValidateCoursewareEducationDomainForActor(
		actor,
		courseware,
	)
}

// coursewareViewPolicyAllows 是不访问数据库的纯查看策略。
//
// 调用方须先完成教育域校验，并提前解析：
//   - isCollabMember：当前用户是否为进行中的集体备课参与者；
//   - sharesOrganization：共享课件作者是否处于当前用户同校或同组作者范围。
func coursewareViewPolicyAllows(
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	isCollabMember bool,
	sharesOrganization bool,
) bool {
	if courseware == nil || actor == nil {
		return false
	}
	if courseware.UserID == actor.UserID {
		return true
	}
	if actor.Role == models.RoleAdmin {
		return true
	}
	if courseware.CollabState == models.CWCollabInSession &&
		isCollabMember {
		return true
	}
	return courseware.PublishState ==
		models.CWPublishPublishedShared &&
		sharesOrganization
}

// CanViewLoadedCourseware 判断Actor是否可以查看一份已经加载的课件。
//
// 查看权：
//   - 作者本人；
//   - admin管理通道；
//   - 同教育域且在进行中的集体备课参与者；
//   - 同教育域且与作者同校或同组的共享课件查看者。
//
// region_admin与district_inspector不会因mixed身份自动获得普通课件详情查看权；
// 它们只能通过已有的审核、辖区或其它明确管理业务入口取得对应只读权限。
func (s *CoursewareService) CanViewLoadedCourseware(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) (bool, error) {
	if err := validateCoursewareDomainForAuthorizedActor(
		actor,
		courseware,
		true,
	); err != nil {
		return false, err
	}

	if courseware.UserID == actor.UserID ||
		actor.Role == models.RoleAdmin {
		return true, nil
	}

	isCollabMember := false
	if courseware.CollabState ==
		models.CWCollabInSession {
		member, err := repository.IsCollabMember(
			ctx,
			courseware.ID,
			actor.UserID,
		)
		if err != nil {
			return false, err
		}
		isCollabMember = member
	}

	sharesOrganization := false
	if courseware.PublishState ==
		models.CWPublishPublishedShared {
		visibleAuthorIDs := s.resolveSameOrgUserIDs(
			ctx,
			actor.UserID,
		)
		for _, authorID := range visibleAuthorIDs {
			if authorID == courseware.UserID {
				sharesOrganization = true
				break
			}
		}
	}

	return coursewareViewPolicyAllows(
		courseware,
		actor,
		isCollabMember,
		sharesOrganization,
	), nil
}

// LoadCoursewareForView 按ID加载并验证课件查看权。
func (s *CoursewareService) LoadCoursewareForView(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.Courseware, error) {
	courseware, err := repository.GetCoursewareByID(
		ctx,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrCoursewareAccessNotFound,
			err,
		)
	}

	allowed, err := s.CanViewLoadedCourseware(
		ctx,
		courseware,
		actor,
	)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrCoursewareViewDenied
	}

	return courseware, nil
}

// coursewareEditPolicyAllows 是不访问数据库的纯编辑策略。
//
// allowAdmin用于区分：
//   - 完整编辑管理通道：作者/admin/集体备课参与者；
//   - 教研微调语义：仅作者/集体备课参与者，不自动放行admin。
func coursewareEditPolicyAllows(
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	isCollabMember bool,
	allowAdmin bool,
) bool {
	if courseware == nil || actor == nil {
		return false
	}

	if courseware.Status ==
		models.CoursewareStatusInPipeline {
		return false
	}
	if courseware.PublishState ==
		models.CWPublishSubmitted {
		return false
	}

	if allowAdmin &&
		actor.Role == models.RoleAdmin {
		return true
	}
	if courseware.UserID == actor.UserID {
		return true
	}
	return courseware.CollabState ==
		models.CWCollabInSession &&
		isCollabMember
}

// canEditLoadedCoursewareWithMode 执行统一编辑教育域与业务权限校验。
func (s *CoursewareService) canEditLoadedCoursewareWithMode(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	allowAdmin bool,
) (bool, error) {
	if err := validateCoursewareDomainForAuthorizedActor(
		actor,
		courseware,
		true,
	); err != nil {
		return false, err
	}

	// 具体课件编辑和生成必须具有具体教学域。
	// common只能作为跨域查看资源，不能进入课件运行链。
	domain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return false,
			ErrCoursewareRuntimeDomainRequired
	}

	isCollabMember := false
	if courseware.UserID != actor.UserID &&
		courseware.CollabState ==
			models.CWCollabInSession {
		member, err := repository.IsCollabMember(
			ctx,
			courseware.ID,
			actor.UserID,
		)
		if err != nil {
			return false, err
		}
		isCollabMember = member
	}

	return coursewareEditPolicyAllows(
		courseware,
		actor,
		isCollabMember,
		allowAdmin,
	), nil
}

// CanEditLoadedCourseware 判断作者、admin或集体备课参与者是否可编辑。
func (s *CoursewareService) CanEditLoadedCourseware(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) (bool, error) {
	return s.canEditLoadedCoursewareWithMode(
		ctx,
		courseware,
		actor,
		true,
	)
}

// CanRefineLoadedCourseware 判断教研微调语义下是否可编辑。
//
// 与原canRefineCourseware一致：只放行作者和集体备课参与者，
// admin不因平台角色自动进入教研活动微调通道。
func (s *CoursewareService) CanRefineLoadedCourseware(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) (bool, error) {
	return s.canEditLoadedCoursewareWithMode(
		ctx,
		courseware,
		actor,
		false,
	)
}

// scopeAuthorizedCoursewareActor 把已经通过编辑授权的Actor收敛到课件快照域。
func scopeAuthorizedCoursewareActor(
	actor *CoursewareActorContext,
	courseware *models.Courseware,
) *CoursewareActorContext {
	scoped := *actor
	scoped.EducationDomain = strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	scoped.MyGroupIDs = append(
		[]string(nil),
		actor.MyGroupIDs...,
	)
	scoped.MyLeadGroupIDs = append(
		[]string(nil),
		actor.MyLeadGroupIDs...,
	)
	scoped.MyLeadOrBackboneGroupIDs = append(
		[]string(nil),
		actor.MyLeadOrBackboneGroupIDs...,
	)
	return &scoped
}

// LoadCoursewareForEdit 按ID加载课件、校验编辑权并返回收敛后的Actor。
func (s *CoursewareService) LoadCoursewareForEdit(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	courseware, err := repository.GetCoursewareByID(
		ctx,
		coursewareID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w: %v",
			ErrCoursewareAccessNotFound,
			err,
		)
	}

	allowed, err := s.CanEditLoadedCourseware(
		ctx,
		courseware,
		actor,
	)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, ErrCoursewareEditDenied
	}

	return courseware,
		scopeAuthorizedCoursewareActor(
			actor,
			courseware,
		),
		nil
}

// coursewareOwnerRuntimePolicyAllows 是作者专属运行通道的纯策略。
//
// 本策略专用于以下不能扩权给admin或集体备课参与者的操作：
//   - 全量课件生成、3D生成与全自动装配；
//   - 素材生成、上传、删除、上云和风格锚点管理；
//   - 页面版本回退、整页HTML保存与外部HTML导入；
//   - 其它会修改作者私有课件或产生外部成本的运行任务。
//
// 作者换校后仍可操作自己的历史课件，因此不要求Actor当前教育域与
// 课件快照域相同；但课件快照本身必须是具体教学域。
func coursewareOwnerRuntimePolicyAllows(
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) bool {
	if courseware == nil ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return false
	}

	if courseware.UserID != actor.UserID {
		return false
	}

	domain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	return models.IsTeachingEducationDomain(domain)
}

// CanOperateOwnedCourseware 判断Actor是否可以进入作者专属课件运行通道。
//
// 与CanEditLoadedCourseware不同：
//   - 不放行admin；
//   - 不放行集体备课参与者；
//   - 只放行课件作者本人；
//   - 作者当前教育域可以不同于课件历史快照域；
//   - 课件快照必须是k12、vocational或adult。
func (s *CoursewareService) CanOperateOwnedCourseware(
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) (bool, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return false, ErrCoursewareActorRequired
	}
	if courseware == nil {
		return false, ErrCoursewareEducationDomainInvalid
	}

	domain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	if !models.IsResourceEducationDomain(domain) {
		return false, ErrCoursewareEducationDomainInvalid
	}
	if !models.IsTeachingEducationDomain(domain) {
		return false, ErrCoursewareRuntimeDomainRequired
	}

	return coursewareOwnerRuntimePolicyAllows(
		courseware,
		actor,
	), nil
}

// LoadCoursewareForOwnerRuntime 按ID加载课件并进入作者专属运行通道。
//
// 授权成功后返回收敛到课件历史教育域快照的Actor，供生成、素材、
// 版本和导入等下游任务继续使用，避免作者当前学校域污染历史课件。
func (s *CoursewareService) LoadCoursewareForOwnerRuntime(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	courseware, err := repository.GetCoursewareByID(
		ctx,
		coursewareID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w: %v",
			ErrCoursewareAccessNotFound,
			err,
		)
	}

	allowed, err := s.CanOperateOwnedCourseware(
		courseware,
		actor,
	)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, ErrCoursewareOwnerRuntimeDenied
	}

	return courseware,
		scopeAuthorizedCoursewareActor(
			actor,
			courseware,
		),
		nil
}
