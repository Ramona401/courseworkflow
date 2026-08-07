package services

// courseware_creation_access.go — 课件创建、Fork与运行域收敛策略
//
// 本文件从原courseware_access_service.go拆出，专门负责：
//   - 构建课件可信Actor；
//   - 无教案来源课件的创建域解析；
//   - 从本人教案创建课件时继承教育域快照；
//   - 共享课件Fork域解析；
//   - 将Actor收敛到具体课件教育域快照。
//
// 安全原则：
//   1. 前端提交的education_domain永远不能决定课件资源域；
//   2. 从教案创建时只读取lesson_plans.education_domain快照；
//   3. 普通admin仍保持mixed管理语义，不能创建个人教学课件；
//   4. superadmin仅在“教案作者就是本人”且数据库实时身份仍为
//      admin+is_super=true时，允许从该教案创建课件；
//   5. superadmin例外不会放宽他人教案、无教案来源创建或共享课件Fork。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// 创建与Fork相关稳定错误，供Handler按业务类型映射状态码。
var (
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
)

// BuildCoursewareActorFromClaims 根据JWT中的用户ID和角色构造课件可信Actor。
//
// 实际组织、学校和教育域解析复用BuildActorFromClaims：
//   - 普通教学角色解析为k12、vocational或adult；
//   - admin、district_inspector保持mixed管理上下文；
//   - region_admin按其固定任命域解析；
//   - 解析异常时后续业务入口fail-closed。
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
// mixed、common、空值和非法值全部拒绝，superadmin也不例外。
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

// resolveCoursewareEducationDomainFromLessonPlanPolicy
// 是不访问数据库的确定性策略函数。
//
// allowSuperAdminOwner只表示调用方已经通过数据库实时确认：
// 当前Actor是admin且is_super=true。即使该值为true，也仍必须满足：
//   - Actor角色确实是admin；
//   - 教案作者就是Actor本人；
//   - 教案快照域是具体教学域。
func resolveCoursewareEducationDomainFromLessonPlanPolicy(
	actor *CoursewareActorContext,
	lessonPlan *models.LessonPlan,
	allowSuperAdminOwner bool,
) (string, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return "", ErrCoursewareActorRequired
	}
	if lessonPlan == nil {
		return "", ErrCoursewareLessonPlanDomainInvalid
	}

	actorDomain := strings.ToLower(
		strings.TrimSpace(
			actor.EducationDomain,
		),
	)

	// region_admin只有在正式任命解析出具体教学域时，
	// 才能进入个人教学资源创建通道。
	//
	// 显式mixed管理上下文继续拒绝创建个人课件。
	if actor.Role == models.RoleRegionAdmin &&
		!models.IsTeachingEducationDomain(
			actorDomain,
		) {
		return "",
			ErrCoursewareCreationDomainRequired
	}

	if isCoursewareMixedManagementRole(actor.Role) {
		if actor.Role != models.RoleAdmin ||
			!allowSuperAdminOwner {
			return "", ErrCoursewareCreationDomainRequired
		}
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

	// region_admin的任命域必须与教案快照域一致。
	//
	// 防止区域管理员使用历史或异常的跨域教案
	// 创建其它教育域的个人课件。
	if actor.Role == models.RoleRegionAdmin &&
		actorDomain != domain {
		return "",
			ErrCoursewareCreationDomainRequired
	}

	return domain, nil
}

// ResolveCoursewareEducationDomainFromLessonPlan
// 保留原有严格策略，供不需要superadmin例外的内部调用兼容使用。
//
// mixed管理身份在本入口仍全部拒绝。
func ResolveCoursewareEducationDomainFromLessonPlan(
	actor *CoursewareActorContext,
	lessonPlan *models.LessonPlan,
) (string, error) {
	return resolveCoursewareEducationDomainFromLessonPlanPolicy(
		actor,
		lessonPlan,
		false,
	)
}

// ResolveCoursewareEducationDomainFromLessonPlanForCreate
// 是“POST /coursewares 从教案创建课件”的正式解析入口。
//
// 普通教学用户：
//   - 仅允许从自己的教案创建；
//   - 直接继承教案教育域快照，不按当前学校重新分类。
//
// mixed管理用户：
//   - region_admin与district_inspector继续拒绝；
//   - 普通admin继续拒绝；
//   - 只有数据库实时身份仍为admin且is_super=true的本人教案作者可创建。
//
// 这里重新读取数据库身份，不单独信任JWT中的历史is_super值，防止账号降权后
// 旧令牌继续使用超级管理员例外。
func ResolveCoursewareEducationDomainFromLessonPlanForCreate(
	ctx context.Context,
	actor *CoursewareActorContext,
	lessonPlan *models.LessonPlan,
) (string, error) {
	// 先用纯策略完成Actor、角色、作者归属和教案快照域校验。
	// allow=true只让admin进入下一步实时身份核验，不能直接完成授权。
	domain, err :=
		resolveCoursewareEducationDomainFromLessonPlanPolicy(
			actor,
			lessonPlan,
			true,
		)
	if err != nil {
		return "", err
	}

	if !isCoursewareMixedManagementRole(actor.Role) {
		return domain, nil
	}

	// 纯策略已保证mixed例外只能是RoleAdmin；此处仍保留显式判断，
	// 防止未来角色集合调整时误把其它管理角色带入超级管理员通道。
	if actor.Role != models.RoleAdmin {
		return "", ErrCoursewareCreationDomainRequired
	}

	currentUser, err := repository.FindUserByID(
		ctx,
		actor.UserID,
	)
	if err != nil {
		return "", fmt.Errorf(
			"校验超级管理员实时身份失败: %w",
			err,
		)
	}
	if currentUser == nil ||
		currentUser.Role != models.RoleAdmin ||
		!currentUser.IsSuper {
		return "", ErrCoursewareCreationDomainRequired
	}

	return domain, nil
}

// ResolveCoursewareForkEducationDomain 解析共享课件Fork副本应写入的教育域。
//
// Fork会创建当前用户自己的教学课件，因此调用者必须已经拥有具体教学域。
// superadmin的mixed管理身份不通过本入口Fork，以免管理身份被转换为个人资源身份。
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
// 该函数先执行教育域访问校验，再复制Actor并覆盖EducationDomain。
// 原Actor不会被修改；组织和教研组切片执行深拷贝，避免异步任务共享底层数组。
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

		models.RoleDistrictInspector:
		return true
	default:
		return false
	}
}
