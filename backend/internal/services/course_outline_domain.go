package services

// course_outline_domain.go — 课程大纲出版社语义教育域硬闸
//
// 本文件集中承载课程大纲相关接口需要复用的实时身份与教育域规则，
// 避免列表、详情、创建、更新、教案挂载和运行时注入各自实现一套不一致逻辑。
//
// 安全原则：
//   1. JWT只提供当前用户ID，不把JWT中的历史角色当作权限真相；
//   2. 每次请求实时读取users.role；
//   3. 普通教学角色调用严格教育域解析器；
//   4. 只接受k12、vocational、adult三个具体教学域；
//   5. mixed、common、空值、非法值、无教学组织和跨域冲突均不获得教学资源权限；
//   6. K12允许空出版社和具名出版社；
//   7. 职教、成教仅允许空出版社；
//   8. group和school资源的教育域由正式组织关系决定，调用者不能自行声明；
//   9. system维持既有K12全局资源语义，不能成为非K12绕过通道。
//
// 管理员兼容：
//   admin是mixed管理身份，不能作为普通教师请求出版社选择列表，
//   但课程大纲管理页历史上由admin维护K12全局与K12基础大纲。
//   因此admin在“课程大纲管理”链中使用受限的K12管理上下文：
//   - 可以查看和维护K12资源；
//   - 出版社选择接口仍返回空数组；
//   - 不能借此上下文管理职教或成教资源；
//   - 不能将该管理上下文复用于普通教案创建或运行时授权。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrOutlineEducationDomainRequired 表示当前用户没有唯一、确定的教学教育域。
	ErrOutlineEducationDomainRequired = errors.New(
		"无确定教学教育域，不能使用课程大纲管理能力，请联系管理员检查学校归属或教育域配置",
	)

	// ErrOutlineEducationDomainConflict 表示当前用户同时关联多个具体教学域。
	ErrOutlineEducationDomainConflict = errors.New(
		"当前账号同时关联多个教学教育域，暂不能使用课程大纲管理能力，请联系管理员处理归属冲突",
	)

	// ErrOutlineEducationDomainResolveFailed 表示数据库或基础设施解析失败。
	ErrOutlineEducationDomainResolveFailed = errors.New(
		"课程大纲教育域解析失败，请稍后重试",
	)

	// ErrOutlineEducationDomainMismatch 表示操作者实时域与资源正式归属域不一致。
	ErrOutlineEducationDomainMismatch = errors.New(
		"课程大纲归属教育域与当前账号教学教育域不一致",
	)

	// ErrOutlinePublisherNotAllowed 表示非K12请求伪造了具名教材出版社。
	ErrOutlinePublisherNotAllowed = errors.New(
		"当前教育域不使用教材出版社字段",
	)

	// ErrOutlinePublisherUnavailable 表示请求选择的出版社没有真实可用大纲。
	//
	// 本错误会在后续教案挂载和单元方案挂载链中复用。
	ErrOutlinePublisherUnavailable = errors.New(
		"所选课程大纲版本当前不可用",
	)
)

// 以下依赖默认指向正式Repository。
//
// 使用函数变量是为了让纯编排测试脱离数据库，验证：
//   - 实时角色读取；
//   - 管理员兼容分支；
//   - 教育域错误映射；
//   - 资源归属域校验。
var (
	courseOutlineFindUser =
		repository.FindUserByID

	courseOutlineResolveActorDomain =
		repository.ResolveLessonPlanCreationEducationDomain

	courseOutlineResolveScopeDomain =
		repository.ResolveCourseOutlineScopeEducationDomain
)

// courseOutlineActor 是一次请求中经数据库实时确认的课程大纲操作者。
type courseOutlineActor struct {
	UserID string
	Role   string

	// EducationDomain 是本次课程大纲资源查询和管理使用的具体域。
	//
	// 普通教学角色取严格解析结果；
	// admin仅在课程大纲管理链中使用k12兼容管理域。
	EducationDomain string

	// MixedManagement表示当前操作者本质上是mixed管理身份。
	//
	// 该标记用于出版社选择接口继续返回安全空列表，
	// 防止管理员管理兼容域被误当成普通K12教学身份。
	MixedManagement bool
}

// resolveCourseOutlineActor 实时解析课程大纲操作者。
//
// 普通教学角色：
//   - FindUserByID实时读取users.role；
//   - ResolveLessonPlanCreationEducationDomain严格解析唯一具体教学域。
//
// admin：
//   - 仅获得课程大纲K12管理兼容域；
//   - MixedManagement保持true；
//   - 不调用普通教案创建教育域解析器。
//
// 不调用NormalizeEducationDomain，防止异常值静默回退K12。
func resolveCourseOutlineActor(
	ctx context.Context,
	userID string,
) (*courseOutlineActor, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrOutlineEducationDomainRequired
	}

	user, err := courseOutlineFindUser(
		ctx,
		userID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserNotFound,
		) {
			return nil, ErrOutlineEducationDomainRequired
		}

		return nil, fmt.Errorf(
			"%w: 读取用户失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}
	if user == nil {
		return nil, ErrOutlineEducationDomainRequired
	}

	role := strings.TrimSpace(user.Role)
	if role == "" {
		return nil, ErrOutlineEducationDomainRequired
	}

	// admin历史上负责K12基础数据与全局课程大纲。
	//
	// 这里赋予的是“课程大纲管理兼容域”，不是普通教学身份，
	// 因此必须保留MixedManagement=true，由出版社选择接口继续返回空数组。
	if role == models.RoleAdmin {
		return &courseOutlineActor{
			UserID:          userID,
			Role:            role,
			EducationDomain: models.EducationDomainK12,
			MixedManagement: true,
		}, nil
	}

	domain, err := courseOutlineResolveActorDomain(
		ctx,
		userID,
		role,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.
				ErrLessonPlanCreationEducationDomainConflict,
		):
			return nil, ErrOutlineEducationDomainConflict

		case errors.Is(
			err,
			repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.
					ErrRegionAdminEducationDomainNotReady,
			):
			return nil, ErrOutlineEducationDomainRequired

		default:
			return nil, fmt.Errorf(
				"%w: %v",
				ErrOutlineEducationDomainResolveFailed,
				err,
			)
		}
	}

	domain = strings.ToLower(
		strings.TrimSpace(domain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return nil, ErrOutlineEducationDomainRequired
	}

	return &courseOutlineActor{
		UserID:          userID,
		Role:            role,
		EducationDomain: domain,
		MixedManagement: false,
	}, nil
}

// isCourseOutlineSafeEmptyDomainError 判断某类教育域错误是否应返回成功空列表。
//
// 列表和出版社查询采用安全空结果：
//   - mixed管理身份但不属于admin管理兼容分支；
//   - 无教学组织；
//   - 教育域为空或非法；
//   - 跨具体教学域冲突。
//
// 数据库和其它基础设施故障不属于安全空结果，必须返回5xx。
func isCourseOutlineSafeEmptyDomainError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrOutlineEducationDomainRequired,
	) || errors.Is(
		err,
		ErrOutlineEducationDomainConflict,
	)
}

// resolveCourseOutlineResourceDomain 解析课程大纲正式归属教育域。
func resolveCourseOutlineResourceDomain(
	ctx context.Context,
	scope string,
	targetID string,
) (string, error) {
	domain, err := courseOutlineResolveScopeDomain(
		ctx,
		scope,
		targetID,
	)
	if err != nil {
		return "", fmt.Errorf(
			"%w: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	domain = strings.ToLower(
		strings.TrimSpace(domain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: 资源归属域非法",
			ErrOutlineEducationDomainResolveFailed,
		)
	}

	return domain, nil
}

// normalizeCourseOutlinePublisherForDomain 规范化并校验出版社字段。
func normalizeCourseOutlinePublisherForDomain(
	educationDomain string,
	publisher string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	publisher = strings.TrimSpace(publisher)

	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrOutlineEducationDomainRequired
	}

	if domain != models.EducationDomainK12 &&
		publisher != "" {
		return "", ErrOutlinePublisherNotAllowed
	}

	return publisher, nil
}
