package services

// courseware_access_service.go — 课件教育域可信Actor与访问控制底座
//
// 本文件只负责课件教育域这一条正交安全边界，不替代课件原有的：
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
	ErrCoursewareCreationDomainRequired = errors.New(
		"无确定教学教育域不能创建课件",
	)
	ErrCoursewareLessonPlanNotOwned = errors.New(
		"只能从自己的教案创建课件，请先Fork到我的教案",
	)
	ErrCoursewareLessonPlanDomainInvalid = errors.New(
		"关联教案缺少有效的具体教学教育域",
	)
	ErrCoursewareForkSourceDomainUnsupported = errors.New(
		"来源课件没有可继承的具体教学教育域，暂不能复制",
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

// BuildCoursewareActorFromClaims 根据JWT中的用户ID和角色构造课件可信Actor。
//
// 实际解析工作复用BuildActorFromClaims：
//   - 普通教学角色解析为k12、vocational或adult；
//   - admin、region_admin、district_inspector保持mixed管理上下文；
//   - 普通用户解析失败时EducationDomain为空并在后续校验中fail-closed。
func BuildCoursewareActorFromClaims(
	ctx context.Context,
	userID string,
	role string,
) *CoursewareActorContext {
	return BuildActorFromClaims(ctx, userID, role)
}

// ResolveCoursewareCreationEducationDomain 解析无教案来源课件的创建教育域。
//
// 适用入口：
//   - 从主题创建；
//   - PPT上传创建；
//   - Word上传创建；
//   - 3D互动单页创建；
//   - 未来其它不依赖教案的课件创建入口。
//
// 返回值只可能是k12、vocational或adult。
// 不使用NormalizeEducationDomain，避免把空值、mixed或非法值静默回退为k12。
func ResolveCoursewareCreationEducationDomain(
	actor *CoursewareActorContext,
) (string, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return "", ErrCoursewareActorRequired
	}

	domain := strings.ToLower(
		strings.TrimSpace(actor.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrCoursewareCreationDomainRequired
	}

	return domain, nil
}

// ResolveCoursewareEducationDomainFromLessonPlan 解析从教案创建课件时应继承的教育域。
//
// 安全规则：
//   - 只能从当前Actor本人创建的教案直接创建个人课件；
//   - 他人共享或审核通过的教案必须先走教案Fork，不能直接猜ID复制；
//   - admin、region_admin、district_inspector属于mixed管理身份，
//     只能管理和查看资源，不能借教案入口创建个人教学课件；
//   - 教案快照必须是k12、vocational或adult，common/mixed/空值/非法值均拒绝；
//   - 普通作者当前所在教育域可以与教案快照不同：老师换校或跨域调动后，
//     仍以历史教案创建时的education_domain为准，不重新按当前学校重分类。
func ResolveCoursewareEducationDomainFromLessonPlan(
	actor *CoursewareActorContext,
	lessonPlan *models.LessonPlan,
) (string, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return "", ErrCoursewareActorRequired
	}
	if lessonPlan == nil {
		return "", ErrCoursewareLessonPlanDomainInvalid
	}

	// 管理角色保持mixed管理语义，不能直接创建归属于自己的教学课件。
	if isCoursewareMixedManagementRole(actor.Role) {
		return "", ErrCoursewareCreationDomainRequired
	}

	if strings.TrimSpace(lessonPlan.AuthorID) == "" ||
		lessonPlan.AuthorID != actor.UserID {
		return "", ErrCoursewareLessonPlanNotOwned
	}

	domain := strings.ToLower(
		strings.TrimSpace(lessonPlan.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrCoursewareLessonPlanDomainInvalid
	}

	return domain, nil
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
		!isCoursewareMixedManagementRole(actor.Role) {
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

// ResolveCoursewareForkEducationDomain 解析共享课件Fork副本应写入的教育域。
//
// 安全规则：
//   - Fork会创建归当前用户所有的个人教学课件，因此操作者必须具有
//     k12、vocational或adult具体教学域；mixed管理账号不能Fork。
//   - 操作者只能Fork当前教学域可访问的来源课件。
//   - 副本继承来源课件的具体教学域，不根据复制者学校重新推导。
//   - common共享课件可以跨域查看，但common不能作为一份具体课件的
//     运行域，因此本阶段不允许直接Fork common课件。
func ResolveCoursewareForkEducationDomain(
	actor *CoursewareActorContext,
	source *models.Courseware,
) (string, error) {
	if _, err := ResolveCoursewareCreationEducationDomain(
		actor,
	); err != nil {
		return "", err
	}

	if err := ValidateCoursewareEducationDomainForActor(
		actor,
		source,
	); err != nil {
		return "", err
	}

	domain := strings.ToLower(
		strings.TrimSpace(source.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrCoursewareForkSourceDomainUnsupported
	}

	return domain, nil
}

// ScopeCoursewareActorToSnapshot 将Actor收敛到具体课件的教育域快照。
//
// 使用场景：
//   - 课件方案生成；
//   - HTML批量生成；
//   - 单页微调和整页重构；
//   - 图片、视频、TTS和字幕派生任务；
//   - 模板保存与其它会加载教学资源的具体课件运行链。
//
// 该函数先执行教育域访问校验，再复制Actor并覆盖EducationDomain。
// 原Actor不会被修改，避免同一请求后续误把管理上下文永久改变。
//
// 具体课件运行必须拥有k12、vocational或adult快照。
// common可以作为跨域通用候选资源，但不能作为一份具体课件的运行域。
func ScopeCoursewareActorToSnapshot(
	actor *CoursewareActorContext,
	courseware *models.Courseware,
) (*CoursewareActorContext, error) {
	if err := ValidateCoursewareEducationDomainForActor(
		actor,
		courseware,
	); err != nil {
		return nil, err
	}

	snapshotDomain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	if !models.IsTeachingEducationDomain(snapshotDomain) {
		return nil, ErrCoursewareRuntimeDomainRequired
	}

	scoped := *actor
	scoped.EducationDomain = snapshotDomain

	// 深拷贝切片，避免调用方修改scoped中的组织集合时污染原Actor。
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

	return &scoped, nil
}

// isCoursewareMixedManagementRole 判断角色是否允许持有mixed课件管理上下文。
func isCoursewareMixedManagementRole(role string) bool {
	switch strings.TrimSpace(role) {
	case models.RoleAdmin,
		models.RoleRegionAdmin,
		models.RoleDistrictInspector:
		return true
	default:
		return false
	}
}
