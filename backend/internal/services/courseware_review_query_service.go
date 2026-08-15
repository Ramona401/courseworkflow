package services

// courseware_review_query_service.go
//
// 课件多级审核只读查询：
//
//   - 审核历史；
//   - 待审核列表；
//   - 审核统计；
//   - 已审核记录；
//   - 审核详情；
//   - 当前级别、当前轮次需要复查的历史正式问题。
//
// 审核详情中的CarryoverItems只在以下条件全部满足时返回：
//
//   1. 课件当前处于submitted；
//   2. 当前用户具备该课件审核详情访问权限；
//   3. 问题已经随历史正式审核反馈交付；
//   4. 作者重新提交时将问题登记到本级、本轮；
//   5. 问题尚未被人工确认解决。
//
// R-01.1复审教师契约：
//
//   - CarryoverItems与普通整改项使用同一个BuildCWReviewItemTeacherView；
//   - 前端默认展示只消费教师字段；
//   - 旧Title/Description只作为教师化兼容映射；
//   - 页面与修改完成哈希固定不进入浏览器响应。
//
// Repository查询错误直接向上返回，不返回部分列表。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 审核历史 ====================

// GetReviewHistory 获取课件审核历史。
func (s *CoursewareReviewService) GetReviewHistory(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.CWReviewHistoryResponse,
	error,
) {
	if actor == nil ||
		actor.UserID == "" {
		return nil,
			ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil,
			ErrCWReviewCoursewareNotFound
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
		return nil,
			ErrCWReviewNoPermission
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
) (
	*models.CWPendingReviewListResponse,
	error,
) {
	if actor == nil ||
		actor.UserID == "" {
		return nil,
			ErrCoursewareActorRequired
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
					[]string{
						schoolID,
					},
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
					[]string{
						schoolID,
					},
					models.ReviewLevelL2,
					domain,
					100,
					0,
				)
			if l2Err != nil {
				return nil, l2Err
			}

			allItems :=
				append(
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

		allItems :=
			append(
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

		allItems :=
			append(
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
func (s *CoursewareReviewService) GetReviewStats(
	ctx context.Context,
	actor *CoursewareActorContext,
	level int,
) (
	*models.CWReviewStatsResponse,
	error,
) {
	if actor == nil ||
		actor.UserID == "" {
		return nil,
			ErrCoursewareActorRequired
	}

	switch actor.Role {
	case models.RoleAdmin,
		models.RoleRegionAdmin,
		models.RoleSeniorOperator,
		models.RoleOperator,
		models.RoleViewer:
	default:
		return &models.CWReviewStatsResponse{},
			nil
	}

	if actor.Role ==
		models.RoleRegionAdmin {
		scope, err :=
			ResolveRegionAdminEducationScope(
				ctx,
				actor.UserID,
			)
		if err != nil {
			return nil, err
		}

		return repository.
			GetCoursewareReviewStatsBySchoolsAndDomain(
				ctx,
				level,
				scope.SchoolIDs,
				scope.EducationDomain,
			)
	}

	isAdmin :=
		actor.Role ==
			models.RoleAdmin

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
					[]string{
						school.ID,
					}
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
func (s *CoursewareReviewService) GetReviewedRecords(
	ctx context.Context,
	actor *CoursewareActorContext,
	level int,
	decision string,
	limit int,
	offset int,
) (
	*models.CWReviewedListResponse,
	error,
) {
	if actor == nil ||
		actor.UserID == "" {
		return nil,
			ErrCoursewareActorRequired
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
			actor.Role ==
				models.RoleAdmin,
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

// isCWPendingReviewLevel 判断是否属于正式审核级别。
func isCWPendingReviewLevel(
	reviewLevel int,
) bool {
	return reviewLevel ==
		models.ReviewLevelL1 ||
		reviewLevel ==
			models.ReviewLevelL2
}

// GetReviewDetail 获取课件审核详情及当前轮次需要复查的问题。
func (s *CoursewareReviewService) GetReviewDetail(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	coursewareService *CoursewareService,
) (
	*models.CWReviewDetailResponse,
	error,
) {
	if actor == nil ||
		actor.UserID == "" {
		return nil,
			ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil,
			ErrCWReviewCoursewareNotFound
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
		return nil,
			ErrCWReviewNoPermission
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

	pendingReviewLevel :=
		courseware.ReviewLevel + 1

	pendingReviewRound := 0
	carryoverItems :=
		[]*models.CWReviewCarryoverItem{}

	if courseware.PublishState ==
		models.CWPublishSubmitted &&
		isCWPendingReviewLevel(
			pendingReviewLevel,
		) {
		reviewCount, countErr :=
			repository.CountCoursewareReviewsByLevel(
				ctx,
				coursewareID,
				pendingReviewLevel,
			)
		if countErr != nil {
			return nil, countErr
		}

		pendingReviewRound =
			reviewCount + 1

		items, itemErr :=
			repository.ListCoursewareReviewItemsForPendingRound(
				ctx,
				coursewareID,
				pendingReviewLevel,
				pendingReviewRound,
			)
		if itemErr != nil {
			return nil, itemErr
		}

		carryoverItems =
			buildCWReviewCarryoverItems(
				items,
			)
	}

	return &models.CWReviewDetailResponse{
		Courseware:         detail,
		Annotations:        annotations,
		Reviews:            reviews,
		PendingReviewRound: pendingReviewRound,
		CarryoverItems:     carryoverItems,
	}, nil
}

// buildCWReviewCarryoverItems 把数据库正式整改项转换为审核员复审安全视图。
//
// 教师内容统一复用BuildCWReviewItemTeacherView，不在复审链重新解析技术证据。
// 页面哈希和应用哈希只供后端状态校验，浏览器响应固定为空串。
func buildCWReviewCarryoverItems(
	items []*models.CoursewareReviewItem,
) []*models.CWReviewCarryoverItem {
	result :=
		make(
			[]*models.CWReviewCarryoverItem,
			0,
			len(items),
		)

	for _, item := range items {
		if item == nil {
			continue
		}

		teacherView :=
			BuildCWReviewItemTeacherView(
				item,
			)

		result = append(
			result,
			&models.CWReviewCarryoverItem{
				ID: item.ID,

				OriginalReviewLevel: item.ReviewLevel,
				OriginalReviewRound: item.ReviewRound,

				PendingReviewLevel: item.ResubmittedReviewLevel,
				PendingReviewRound: item.ResubmittedReviewRound,

				PageID:                item.PageID,
				PageNumberSnapshot:    item.PageNumberSnapshot,
				PageTitleSnapshot:     item.PageTitleSnapshot,
				PageHTMLHash:          "",
				PageUpdatedAtSnapshot: item.PageUpdatedAtSnapshot,

				Severity:  item.Severity,
				Dimension: item.Dimension,

				Title:       teacherView.TeacherTitle,
				Description: teacherView.WhatHappened,

				TeacherTitle:    teacherView.TeacherTitle,
				WhatHappened:    teacherView.WhatHappened,
				TeachingImpact:  teacherView.TeachingImpact,
				ImprovementGoal: teacherView.ImprovementGoal,
				AcceptanceChecks: append(
					[]string{},
					teacherView.AcceptanceChecks...,
				),
				TeacherContext:      teacherView.TeacherContext,
				ManualCheckRequired: teacherView.ManualCheckRequired,

				ConfirmedInstruction: item.ConfirmedInstruction,

				Status:          item.Status,
				AppliedPageHash: "",

				ConfirmedAt:   item.ConfirmedAt,
				AppliedAt:     item.AppliedAt,
				ResubmittedAt: item.ResubmittedAt,
			},
		)
	}

	return result
}
