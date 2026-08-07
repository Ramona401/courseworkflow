package services

// course_outline_service_scope.go — 课程大纲读取范围与写权限
//
// 普通教师可读取：
//   - 自己所属的group级大纲；
//   - 所属教研组所在学校的school级大纲；
//   - K12 system级大纲。
//
// school级写权限仍仅属于该校senior_operator，
// group级写权限仍仅属于组长或骨干，system级仅admin可写。

import (
	"context"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// canViewScope 判断用户是否拥有资源读取范围。
func (s *CourseOutlineService) canViewScope(
	ctx context.Context,
	actor *courseOutlineActor,
	scope string,
	targetID string,
) bool {
	if actor == nil {
		return false
	}
	if actor.Role == models.RoleAdmin {
		return true
	}

	switch scope {
	case models.CourseOutlineScopeSystem:
		return actor.EducationDomain ==
			models.EducationDomainK12

	case models.CourseOutlineScopeGroup,
		models.CourseOutlineScopeSchool:
		groupIDs, schoolIDs :=
			s.resolveUserVisibleScopeIDs(
				ctx,
				actor.Role,
				actor.UserID,
			)

		if scope ==
			models.CourseOutlineScopeGroup {
			return containsCourseOutlineScopeID(
				groupIDs,
				targetID,
			)
		}
		return containsCourseOutlineScopeID(
			schoolIDs,
			targetID,
		)
	}

	return false
}

// canManageScope 判断用户是否拥有资源写权限。
func (s *CourseOutlineService) canManageScope(
	ctx context.Context,
	actor *courseOutlineActor,
	scope string,
	targetID string,
) bool {
	if actor == nil {
		return false
	}
	if actor.Role == models.RoleAdmin {
		return true
	}

	switch scope {
	case models.CourseOutlineScopeSystem:
		return false

	case models.CourseOutlineScopeSchool:
		if actor.Role !=
			models.RoleSeniorOperator {
			return false
		}
		school, err :=
			repository.GetSchoolByAdminUserID(
				ctx,
				actor.UserID,
			)
		return err == nil &&
			school != nil &&
			school.ID == targetID

	case models.CourseOutlineScopeGroup:
		allowed, err :=
			repository.IsGroupLeadOrBackbone(
				ctx,
				targetID,
				actor.UserID,
			)
		if err != nil {
			courseOutlineLog.Warn(
				"校验课程大纲教研组管理权限失败",
				"group", targetID,
				"user", actor.UserID,
				"error", err,
			)
			return false
		}
		return allowed
	}

	return false
}

// resolveUserVisibleScopeIDs 返回用户可见的教研组和学校范围。
//
// 普通教师所属教研组的school_id也属于其学校级资源读取范围；
// senior_operator额外加入其直接管理学校。
// 读取失败时返回已确认的最小范围，不扩大权限。
func (s *CourseOutlineService) resolveUserVisibleScopeIDs(
	ctx context.Context,
	role string,
	userID string,
) (
	[]string,
	[]string,
) {
	groupSeen := make(map[string]struct{})
	schoolSeen := make(map[string]struct{})

	groups, err :=
		repository.GetUserTeachingGroups(
			ctx,
			userID,
		)
	if err != nil {
		courseOutlineLog.Warn(
			"查询用户课程大纲可见教研组失败",
			"user", userID,
			"error", err,
		)
	} else {
		for _, group := range groups {
			if group == nil {
				continue
			}

			groupID := strings.TrimSpace(
				group.ID,
			)
			schoolID := strings.TrimSpace(
				group.SchoolID,
			)

			if groupID != "" {
				groupSeen[groupID] =
					struct{}{}
			}
			if schoolID != "" {
				schoolSeen[schoolID] =
					struct{}{}
			}
		}
	}

	if role == models.RoleSeniorOperator {
		school, schoolErr :=
			repository.GetSchoolByAdminUserID(
				ctx,
				userID,
			)
		if schoolErr == nil &&
			school != nil {
			schoolID := strings.TrimSpace(
				school.ID,
			)
			if schoolID != "" {
				schoolSeen[schoolID] =
					struct{}{}
			}
		}
	}

	return sortedCourseOutlineScopeIDs(
			groupSeen,
		),
		sortedCourseOutlineScopeIDs(
			schoolSeen,
		)
}

func containsCourseOutlineScopeID(
	values []string,
	target string,
) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// sortedCourseOutlineScopeIDs 返回稳定有序的ID列表，便于日志、测试和SQL参数复现。
func sortedCourseOutlineScopeIDs(
	values map[string]struct{},
) []string {
	result := make(
		[]string,
		0,
		len(values),
	)
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
