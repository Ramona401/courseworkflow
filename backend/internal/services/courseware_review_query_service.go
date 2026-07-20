package services

// courseware_review_query_service.go
//
// 课件多级审核的只读查询：
//   - 审核历史；
//   - 待审核列表；
//   - 审核统计；
//   - 已审核记录；
//   - 审核详情。
//
// 上下文 6：
//   region_admin 的待审列表和统计统一使用
//   ResolveRegionAdminEducationScope 解析出的同域学校白名单。
//
// 统一范围：
//   管辖区域树下 active 学校
//   AND 学校教育域等于管理员固定教育域
//   AND 课件教育域快照等于管理员固定教育域。
//
// Repository 查询错误直接向上返回，不返回部分列表，不退化成全局数据。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 审核历史 ====================

// GetReviewHistory 获取课件审核历史。
//
// 权限通过 CanViewLoadedCoursewareReviewHistory 统一裁决。
func (s *CoursewareReviewService) GetReviewHistory(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.CWReviewHistoryResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, ErrCWReviewCoursewareNotFound
	}

	allowed, err :=
		s.CanViewLoadedCoursewareReviewHistory(
			ctx,
			courseware,
			actor,
		)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrCWReviewNoPermission
	}

	reviews, err :=
		repository.ListCoursewareReviewsByCourseware(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews =
			[]*models.CWReviewListItem{}
	}

	return &models.CWReviewHistoryResponse{
		Reviews:      reviews,
		Total:        len(reviews),
		CurrentLevel: courseware.ReviewLevel,
	}, nil
}

// ==================== 待审核列表 ====================

// GetPendingReviews 获取课件待审核列表。
func (s *CoursewareReviewService) GetPendingReviews(
	ctx context.Context,
	actor *CoursewareActorContext,
	limit int,
	offset int,
) (*models.CWPendingReviewListResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrCoursewareActorRequired
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	userID := actor.UserID
	domain := actor.EducationDomain

	switch actor.Role {
	case models.RoleOperator,
		models.RoleViewer:
		items, total, err :=
			repository.ListCWPendingReviewsL1(
				ctx,
				userID,
				domain,
				limit,
				offset,
			)
		if err != nil {
			return nil, err
		}

		return &models.CWPendingReviewListResponse{
			Items: items,
			Total: total,
		}, nil

	case models.RoleSeniorOperator:
		schoolID := ""

		if school, err :=
			repository.GetSchoolByAdminUserID(
				ctx,
				userID,
			); err == nil &&
			school != nil {
			schoolID = school.ID
		}

		if schoolID != "" {
			l1Items, _, l1Err :=
				repository.ListCWPendingReviewsBySchools(
					ctx,
					[]string{schoolID},
					models.ReviewLevelL1,
					domain,
					100,
					0,
				)
			if l1Err != nil {
				return nil, l1Err
			}

			l2Items, _, l2Err :=
				repository.ListCWPendingReviewsBySchools(
					ctx,
					[]string{schoolID},
					models.ReviewLevelL2,
					domain,
					100,
					0,
				)
			if l2Err != nil {
				return nil, l2Err
			}

			allItems := append(
				l1Items,
				l2Items...,
			)

			return &models.CWPendingReviewListResponse{
				Items: allItems,
				Total: len(allItems),
			}, nil
		}

		items, _, err :=
			repository.ListCWPendingReviewsL1(
				ctx,
				userID,
				domain,
				100,
				0,
			)
		if err != nil {
			return nil, err
		}
		if items == nil {
			items =
				[]*models.CWPendingReviewItem{}
		}

		return &models.CWPendingReviewListResponse{
			Items: items,
			Total: len(items),
		}, nil

	case models.RoleRegionAdmin:
		scope, err :=
			ResolveRegionAdminEducationScope(
				ctx,
				userID,
			)
		if err != nil {
			return nil, err
		}

		l1Items, _, l1Err :=
			repository.ListCWPendingReviewsBySchools(
				ctx,
				scope.SchoolIDs,
				models.ReviewLevelL1,
				scope.EducationDomain,
				limit,
				offset,
			)
		if l1Err != nil {
			return nil, l1Err
		}

		l2Items, _, l2Err :=
			repository.ListCWPendingReviewsBySchools(
				ctx,
				scope.SchoolIDs,
				models.ReviewLevelL2,
				scope.EducationDomain,
				limit,
				offset,
			)
		if l2Err != nil {
			return nil, l2Err
		}

		allItems := append(
			l1Items,
			l2Items...,
		)

		return &models.CWPendingReviewListResponse{
			Items: allItems,
			Total: len(allItems),
		}, nil

	case models.RoleAdmin:
		l1Items, _, l1Err :=
			repository.ListCWPendingReviewsL1All(
				ctx,
				domain,
				100,
				0,
			)
		if l1Err != nil {
			return nil, l1Err
		}

		l2Items, _, l2Err :=
			repository.ListCWPendingReviewsL2(
				ctx,
				"",
				domain,
				100,
				0,
			)
		if l2Err != nil {
			return nil, l2Err
		}

		allItems := append(
			l1Items,
			l2Items...,
		)

		return &models.CWPendingReviewListResponse{
			Items: allItems,
			Total: len(allItems),
		}, nil

	default:
		return &models.CWPendingReviewListResponse{
			Items: []*models.CWPendingReviewItem{},
			Total: 0,
		}, nil
	}
}

// ==================== 审核统计 ====================

// GetReviewStats 获取课件审核统计。
//
// region_admin 返回辖区同域聚合统计，不使用其个人 reviewer_id 口径。
func (s *CoursewareReviewService) GetReviewStats(
	ctx context.Context,
	actor *CoursewareActorContext,
	level int,
) (*models.CWReviewStatsResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrCoursewareActorRequired
	}

	switch actor.Role {
	case models.RoleAdmin,
		models.RoleRegionAdmin,
		models.RoleSeniorOperator,
		models.RoleOperator,
		models.RoleViewer:
	default:
		return &models.CWReviewStatsResponse{}, nil
	}

	if actor.Role == models.RoleRegionAdmin {
		scope, err :=
			ResolveRegionAdminEducationScope(
				ctx,
				actor.UserID,
			)
		if err != nil {
			return nil, err
		}

		return repository.GetCoursewareReviewStatsBySchoolsAndDomain(
			ctx,
			level,
			scope.SchoolIDs,
			scope.EducationDomain,
		)
	}

	isAdmin := actor.Role == models.RoleAdmin

	var memberIDs []string
	var schoolIDs []string

	if !isAdmin {
		switch actor.Role {
		case models.RoleSeniorOperator:
			if school, err :=
				repository.GetSchoolByAdminUserID(
					ctx,
					actor.UserID,
				); err == nil &&
				school != nil {
				schoolIDs =
					[]string{school.ID}
			} else if level ==
				models.ReviewLevelL1 {
				memberIDs, _ =
					repository.GetCWReviewableMemberIDs(
						ctx,
						actor.UserID,
					)
			}

		case models.RoleOperator,
			models.RoleViewer:
			if level ==
				models.ReviewLevelL1 {
				memberIDs, _ =
					repository.GetCWReviewableMemberIDs(
						ctx,
						actor.UserID,
					)
			}
		}
	}

	return repository.GetCWReviewStats(
		ctx,
		actor.UserID,
		level,
		isAdmin,
		memberIDs,
		schoolIDs,
		actor.EducationDomain,
	)
}

// ==================== 已审核记录 ====================

// GetReviewedRecords 获取课件已审核记录。
//
// region_admin 的辖区聚合只作用于统计，不扩大已审核明细范围；
// 非 admin 的已审核记录仍只显示本人产生的审核记录。
func (s *CoursewareReviewService) GetReviewedRecords(
	ctx context.Context,
	actor *CoursewareActorContext,
	level int,
	decision string,
	limit int,
	offset int,
) (*models.CWReviewedListResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrCoursewareActorRequired
	}

	switch actor.Role {
	case models.RoleAdmin,
		models.RoleRegionAdmin,
		models.RoleSeniorOperator,
		models.RoleOperator,
		models.RoleViewer:
	default:
		return &models.CWReviewedListResponse{
			Items: []*models.CWReviewedListItem{},
			Total: 0,
		}, nil
	}

	items, total, err :=
		repository.ListCWReviewedRecords(
			ctx,
			actor.UserID,
			level,
			decision,
			actor.Role == models.RoleAdmin,
			actor.EducationDomain,
			limit,
			offset,
		)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items =
			[]*models.CWReviewedListItem{}
	}

	return &models.CWReviewedListResponse{
		Items: items,
		Total: total,
	}, nil
}

// ==================== 审核详情 ====================

// GetReviewDetail 获取课件审核详情。
//
// 权限通过 CanReviewLoadedCourseware 统一裁决。
// region_admin 只能读取辖区同域课件，不能执行审核决策。
func (s *CoursewareReviewService) GetReviewDetail(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	coursewareService *CoursewareService,
) (*models.CWReviewDetailResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, ErrCWReviewCoursewareNotFound
	}

	allowed, err :=
		s.CanReviewLoadedCourseware(
			ctx,
			courseware,
			actor,
		)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrCWReviewNoPermission
	}

	detail, err :=
		coursewareService.GetCourseware(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"装配课件详情失败: %w",
			err,
		)
	}

	annotations, annotationErr :=
		repository.ListCWAnnotationsByCoursewareID(
			ctx,
			coursewareID,
		)
	if annotationErr != nil ||
		annotations == nil {
		annotations =
			[]*models.CoursewareAnnotation{}
	}

	reviews, reviewErr :=
		repository.ListCoursewareReviewsByCourseware(
			ctx,
			coursewareID,
		)
	if reviewErr != nil ||
		reviews == nil {
		reviews =
			[]*models.CWReviewListItem{}
	}

	return &models.CWReviewDetailResponse{
		Courseware:  detail,
		Annotations: annotations,
		Reviews:     reviews,
	}, nil
}
