package services

// lesson_plan_shared_scope.go — 共享教案统一可见性底座
//
// 本文件是上下文17共享教案授权的唯一事实源。
// 列表、详情、互动、收藏和Fork不得自行复制教育域或组织范围判断。
//
// 共享市场授权必须同时满足：
//   1. 当前用户拥有唯一具体教学域：k12 / vocational / adult；
//   2. 目标教案状态为approved或published_shared；
//   3. visibility为group、school、region或public，personal永不进入市场；
//   4. 教案教育域等于用户具体域，或为受控common；
//   5. 教案作者位于当前用户组织可见作者白名单。
//
// 通用详情保持既有正式管理能力，但不能绕过共享市场教育域隔离：
//   - 作者可读取自己的任何未删除教案；
//   - 对于共享候选，非作者必须严格通过共享市场授权，异域统一404；
//   - 对于尚未共享的评审/管理资源，admin和DataScope范围内管理者可读取；
//   - 互动、收藏和Fork始终严格走共享市场授权。
//
// 所有不可见直接ID统一返回ErrLPNotFound，避免资源探测。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var lpSharedScopeLog = logger.WithModule("services.lp_shared_scope")

// errLPSharedAccessUnavailable 只在services包内部使用。
// 列表遇到该错误返回安全空集，直接ID入口转换为ErrLPNotFound。
var errLPSharedAccessUnavailable = errors.New(
	"当前用户没有可用于共享教案市场的唯一具体教学域",
)

// lessonPlanSharedAccessContext 是单次请求内稳定的共享访问快照。
type lessonPlanSharedAccessContext struct {
	UserID                 string
	CurrentEducationDomain string
	VisibleAuthorIDs       []string
	visibleAuthorSet       map[string]struct{}
}

// resolveLPSharedVisibleAuthorIDs 解析共享市场作者白名单。
//
// 白名单为正式DataScope成员、同校成员、所在区域成员、同教研组成员和本人并集。
// 任一组织查询失败只跳过该层并记日志，绝不放大访问范围。
func resolveLPSharedVisibleAuthorIDs(
	ctx context.Context,
	userID string,
	scope *DataScope,
) []string {
	idSet := make(map[string]struct{})
	userID = strings.TrimSpace(userID)
	if userID != "" {
		idSet[userID] = struct{}{}
	}

	if scope != nil && !scope.IsAdmin {
		for _, uid := range scope.UserIDs {
			uid = strings.TrimSpace(uid)
			if uid != "" {
				idSet[uid] = struct{}{}
			}
		}
	}

	schoolID, schoolErr := repository.GetSchoolIDByUserID(
		ctx,
		userID,
	)
	if schoolErr != nil {
		lpSharedScopeLog.Warn(
			"共享教案白名单：解析用户学校失败，跳过同校与同区域层",
			"user_id", userID,
			"error", schoolErr,
		)
	}

	if schoolID != "" {
		memberIDs, memberErr := repository.ListSchoolMemberIDs(
			ctx,
			schoolID,
		)
		if memberErr != nil {
			lpSharedScopeLog.Warn(
				"共享教案白名单：查询同校成员失败，跳过同校层",
				"school_id", schoolID,
				"error", memberErr,
			)
		} else {
			for _, uid := range memberIDs {
				uid = strings.TrimSpace(uid)
				if uid != "" {
					idSet[uid] = struct{}{}
				}
			}
		}

		schoolOrg, orgErr := repository.GetOrganizationByID(
			ctx,
			schoolID,
		)
		if orgErr != nil {
			lpSharedScopeLog.Warn(
				"共享教案白名单：查询学校组织失败，跳过同区域层",
				"school_id", schoolID,
				"error", orgErr,
			)
		} else if schoolOrg != nil &&
			schoolOrg.ParentID != nil &&
			strings.TrimSpace(*schoolOrg.ParentID) != "" {
			regionID := strings.TrimSpace(*schoolOrg.ParentID)
			regionSchoolIDs, regionErr :=
				repository.ListDescendantSchoolIDs(
					ctx,
					regionID,
				)
			if regionErr != nil {
				lpSharedScopeLog.Warn(
					"共享教案白名单：查询区域学校失败，跳过同区域层",
					"region_id", regionID,
					"error", regionErr,
				)
			} else {
				for _, regionSchoolID := range regionSchoolIDs {
					regionSchoolID = strings.TrimSpace(regionSchoolID)
					if regionSchoolID == "" ||
						regionSchoolID == schoolID {
						continue
					}

					regionMemberIDs, regionMemberErr :=
						repository.ListSchoolMemberIDs(
							ctx,
							regionSchoolID,
						)
					if regionMemberErr != nil {
						lpSharedScopeLog.Warn(
							"共享教案白名单：查询区域内学校成员失败，跳过该学校",
							"school_id", regionSchoolID,
							"error", regionMemberErr,
						)
						continue
					}
					for _, uid := range regionMemberIDs {
						uid = strings.TrimSpace(uid)
						if uid != "" {
							idSet[uid] = struct{}{}
						}
					}
				}
			}
		}
	}

	groups, groupsErr := repository.GetUserTeachingGroups(
		ctx,
		userID,
	)
	if groupsErr != nil {
		lpSharedScopeLog.Warn(
			"共享教案白名单：查询用户教研组失败，跳过同组层",
			"user_id", userID,
			"error", groupsErr,
		)
	} else {
		for _, group := range groups {
			memberIDs, memberErr :=
				repository.ListTeachingGroupMemberIDs(
					ctx,
					group.ID,
				)
			if memberErr != nil {
				lpSharedScopeLog.Warn(
					"共享教案白名单：查询教研组成员失败，跳过该组",
					"group_id", group.ID,
					"error", memberErr,
				)
				continue
			}
			for _, uid := range memberIDs {
				uid = strings.TrimSpace(uid)
				if uid != "" {
					idSet[uid] = struct{}{}
				}
			}
		}
	}

	result := make([]string, 0, len(idSet))
	for uid := range idSet {
		result = append(result, uid)
	}
	return result
}

// resolveLessonPlanReadScope 解析未共享教案详情使用的正式管理范围。
// 此范围不能用于共享候选、互动、收藏或Fork授权。
func resolveLessonPlanReadScope(
	ctx context.Context,
	userID string,
	scope *DataScope,
) (*DataScope, error) {
	if scope != nil {
		return scope, nil
	}

	user, err := repository.FindUserByID(
		ctx,
		userID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrLPNotFound
		}
		return nil, fmt.Errorf(
			"读取教案访问用户失败: %w",
			err,
		)
	}
	if user == nil || strings.TrimSpace(user.Role) == "" {
		return nil, ErrLPNotFound
	}

	resolved := ResolveDataScope(
		ctx,
		user.Role,
		userID,
	)
	return &resolved, nil
}

// resolveLessonPlanSharedAccessContext 解析共享市场访问上下文。
// 用户角色和教育域均实时读取数据库，不接受前端教育域参数。
func resolveLessonPlanSharedAccessContext(
	ctx context.Context,
	userID string,
	scope *DataScope,
) (*lessonPlanSharedAccessContext, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errLPSharedAccessUnavailable
	}

	user, err := repository.FindUserByID(
		ctx,
		userID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errLPSharedAccessUnavailable
		}
		return nil, fmt.Errorf(
			"读取共享教案访问用户失败: %w",
			err,
		)
	}
	if user == nil || strings.TrimSpace(user.Role) == "" {
		return nil, errLPSharedAccessUnavailable
	}

	resolvedScope := scope
	if resolvedScope == nil {
		scopeValue := ResolveDataScope(
			ctx,
			user.Role,
			userID,
		)
		resolvedScope = &scopeValue
	}

	educationDomain, err :=
		repository.ResolveLessonPlanCreationEducationDomain(
			ctx,
			userID,
			user.Role,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.ErrLessonPlanCreationEducationDomainConflict,
			),
			errors.Is(
				err,
				repository.ErrRegionAdminEducationDomainNotReady,
			):
			return nil, errLPSharedAccessUnavailable
		default:
			return nil, fmt.Errorf(
				"解析共享教案访问教育域失败: %w",
				err,
			)
		}
	}

	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsTeachingEducationDomain(educationDomain) {
		return nil, errLPSharedAccessUnavailable
	}

	visibleAuthorIDs := resolveLPSharedVisibleAuthorIDs(
		ctx,
		userID,
		resolvedScope,
	)
	visibleAuthorSet := make(
		map[string]struct{},
		len(visibleAuthorIDs),
	)
	for _, authorID := range visibleAuthorIDs {
		authorID = strings.TrimSpace(authorID)
		if authorID != "" {
			visibleAuthorSet[authorID] = struct{}{}
		}
	}
	if len(visibleAuthorSet) == 0 {
		return nil, errLPSharedAccessUnavailable
	}

	return &lessonPlanSharedAccessContext{
		UserID:                 userID,
		CurrentEducationDomain: educationDomain,
		VisibleAuthorIDs:       visibleAuthorIDs,
		visibleAuthorSet:       visibleAuthorSet,
	}, nil
}

// isSharedLessonPlanCandidate 判断目标是否为正式共享候选。
func isSharedLessonPlanCandidate(
	lessonPlan *models.LessonPlan,
) bool {
	if lessonPlan == nil {
		return false
	}

	switch lessonPlan.Status {
	case models.LPStatusApproved,
		models.LPStatusPublishedShared:
	default:
		return false
	}

	switch strings.TrimSpace(lessonPlan.Visibility) {
	case "group", "school", "region", "public":
		return true
	default:
		return false
	}
}

// canAccessSharedLessonPlan 判断共享候选是否与访问上下文兼容。
func (
	access *lessonPlanSharedAccessContext,
) canAccessSharedLessonPlan(
	lessonPlan *models.LessonPlan,
) bool {
	if access == nil ||
		lessonPlan == nil ||
		!isSharedLessonPlanCandidate(lessonPlan) {
		return false
	}

	if !models.ResourceEducationDomainMatches(
		strings.ToLower(
			strings.TrimSpace(lessonPlan.EducationDomain),
		),
		access.CurrentEducationDomain,
	) {
		return false
	}

	_, allowed := access.visibleAuthorSet[strings.TrimSpace(lessonPlan.AuthorID)]
	return allowed
}

// dataScopeAllowsLessonPlanAuthor 判断未共享详情是否在正式管理范围内。
func dataScopeAllowsLessonPlanAuthor(
	scope *DataScope,
	authorID string,
) bool {
	if scope == nil {
		return false
	}
	if scope.IsAdmin {
		return true
	}

	authorID = strings.TrimSpace(authorID)
	for _, visibleUserID := range scope.UserIDs {
		if strings.TrimSpace(visibleUserID) == authorID {
			return true
		}
	}
	return false
}

// loadSharedLessonPlanForRead 是互动和Fork共用的严格共享入口。
// 即使调用者是作者，私有或未共享教案也不能在共享市场执行互动或Fork。
func (
	s *LessonPlanService,
) loadSharedLessonPlanForRead(
	ctx context.Context,
	lessonPlanID string,
	callerID string,
	scope *DataScope,
) (*models.LessonPlan, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	callerID = strings.TrimSpace(callerID)
	if lessonPlanID == "" || callerID == "" {
		return nil, ErrLPNotFound
	}

	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}

	access, err := resolveLessonPlanSharedAccessContext(
		ctx,
		callerID,
		scope,
	)
	if err != nil {
		if errors.Is(
			err,
			errLPSharedAccessUnavailable,
		) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}
	if !access.canAccessSharedLessonPlan(lessonPlan) {
		return nil, ErrLPNotFound
	}

	return lessonPlan, nil
}

// loadLessonPlanForRead 是通用详情读取入口。
//
// 非作者遇到共享候选时必须先走共享市场授权，不能被admin或DataScope旁路；
// 只有尚未共享的评审和管理资源才允许正式管理范围读取。
func (
	s *LessonPlanService,
) loadLessonPlanForRead(
	ctx context.Context,
	lessonPlanID string,
	callerID string,
	scope *DataScope,
) (*models.LessonPlan, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	callerID = strings.TrimSpace(callerID)
	if lessonPlanID == "" || callerID == "" {
		return nil, ErrLPNotFound
	}

	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}
	if lessonPlan.AuthorID == callerID {
		return lessonPlan, nil
	}

	resolvedScope, err := resolveLessonPlanReadScope(
		ctx,
		callerID,
		scope,
	)
	if err != nil {
		return nil, err
	}

	// 共享候选必须严格执行同域和共享作者白名单判断，
	// 不能因为调用者具有更大的管理DataScope而绕过市场隔离。
	if isSharedLessonPlanCandidate(lessonPlan) {
		access, accessErr :=
			resolveLessonPlanSharedAccessContext(
				ctx,
				callerID,
				resolvedScope,
			)
		if accessErr != nil {
			if errors.Is(
				accessErr,
				errLPSharedAccessUnavailable,
			) {
				return nil, ErrLPNotFound
			}
			return nil, accessErr
		}
		if !access.canAccessSharedLessonPlan(lessonPlan) {
			return nil, ErrLPNotFound
		}
		return lessonPlan, nil
	}

	// 未共享资源仅供正式管理和审核链路读取。
	if dataScopeAllowsLessonPlanAuthor(
		resolvedScope,
		lessonPlan.AuthorID,
	) {
		return lessonPlan, nil
	}

	return nil, ErrLPNotFound
}
