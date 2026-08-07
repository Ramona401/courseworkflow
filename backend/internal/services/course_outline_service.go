package services

// course_outline_service.go — 课程大纲核心读取业务
//
// 本文件只承载：
//   - 公共错误、服务结构和构造函数；
//   - 当前用户可见大纲列表；
//   - 单条大纲详情与同域、可见范围校验。
//
// 创建、更新、删除位于course_outline_service_mutation.go；
// 精确候选与学制规范化位于course_outline_service_exact.go；
// 可见范围和管理权限位于course_outline_service_scope.go。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var courseOutlineLog = logger.WithModule(
	"services.course_outline",
)

var (
	ErrOutlineFieldRequired = errors.New(
		"学科、年级、册次、标题、正文均为必填",
	)
	ErrOutlineScopeInvalid = errors.New(
		"归属类型非法",
	)
	ErrOutlineNoPermission = errors.New(
		"您没有权限管理该归属的课程大纲",
	)
	ErrOutlineSchoolSystemInvalid = errors.New(
		"课程大纲学制必须是普通学制或五四制",
	)
)

// CourseOutlineService 课程大纲服务。
type CourseOutlineService struct{}

// NewCourseOutlineService 创建服务。
func NewCourseOutlineService() *CourseOutlineService {
	return &CourseOutlineService{}
}

// ListOutlines 列出当前用户同教育域且可见的大纲。
func (s *CourseOutlineService) ListOutlines(
	ctx context.Context,
	userID string,
) (
	[]*models.CourseOutlineListItem,
	string,
	error,
) {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		if isCourseOutlineSafeEmptyDomainError(err) {
			return []*models.CourseOutlineListItem{},
				"",
				nil
		}
		return nil, "", err
	}

	groupIDs, schoolIDs :=
		s.resolveUserVisibleScopeIDs(
			ctx,
			actor.Role,
			actor.UserID,
		)

	items, err :=
		repository.ListCourseOutlinesWithSchoolSystem(
			ctx,
			actor.Role == models.RoleAdmin,
			groupIDs,
			schoolIDs,
			actor.EducationDomain,
		)
	if err != nil {
		return nil, "", err
	}
	if items == nil {
		items = []*models.CourseOutlineListItem{}
	}

	return items, actor.EducationDomain, nil
}

// GetOutline 获取单条大纲并执行教育域与可见范围校验。
//
// 跨域、归属异常或当前用户不可见统一返回“课程大纲不存在”，
// 防止通过ID枚举探测其它教育域或组织范围的资源。
func (s *CourseOutlineService) GetOutline(
	ctx context.Context,
	userID string,
	id string,
) (
	*models.CourseOutline,
	string,
	error,
) {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return nil, "", err
	}

	outline, err :=
		repository.GetCourseOutlineByIDWithSchoolSystem(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return nil, "", err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			outline.Scope,
			outline.ScopeTargetID,
		)
	if err != nil {
		courseOutlineLog.Warn(
			"课程大纲详情归属教育域解析失败",
			"outline_id", outline.ID,
			"scope", outline.Scope,
			"scope_target_id", outline.ScopeTargetID,
			"error", err,
		)
		return nil, "",
			repository.ErrCourseOutlineNotFound
	}

	if resourceDomain != actor.EducationDomain ||
		!s.canViewScope(
			ctx,
			actor,
			outline.Scope,
			outline.ScopeTargetID,
		) {
		return nil, "",
			repository.ErrCourseOutlineNotFound
	}

	return outline, actor.EducationDomain, nil
}
