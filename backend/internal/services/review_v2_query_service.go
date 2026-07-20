package services

// review_v2_query_service.go
//
// 教案多级审核的只读查询和审核流程配置：
//   - 审核历史；
//   - 待审核列表；
//   - 审核统计；
//   - 已审核记录；
//   - 审核流程配置。
//
// 上下文 6：
//   region_admin 的待审列表、统计和审核历史统一使用
//   ResolveRegionAdminEducationScope 解析出的同域学校白名单。
//
// 统一范围：
//   管辖区域树下 active 学校
//   AND 学校教育域等于管理员固定教育域
//   AND 教案教育域快照等于管理员固定教育域。
//
// 查询异常直接返回错误，不能退化成全局数据或不完整列表。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// GetReviewHistory 获取教案审核历史。
//
// region_admin 必须同时通过辖区学校和教育域验证。
// 其它角色保持既有访问行为，避免本上下文扩大修改边界。
func (s *ReviewV2Service) GetReviewHistory(
	ctx context.Context,
	planID string,
	callerID string,
	callerRole string,
) (*models.ReviewHistoryResponse, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		planID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil, ErrReviewPlanNotFound
		}
		return nil, err
	}

	if callerRole == models.RoleRegionAdmin {
		scope, scopeErr :=
			ResolveRegionAdminEducationScope(
				ctx,
				callerID,
			)
		if scopeErr != nil {
			reviewLog.Warn(
				"区域管理员审核历史范围解析失败",
				"user_id",
				callerID,
				"plan_id",
				planID,
				"error",
				scopeErr,
			)
			return nil, ErrReviewNoPermission
		}

		planDomain := strings.ToLower(
			strings.TrimSpace(
				lessonPlan.EducationDomain,
			),
		)
		if planDomain != scope.EducationDomain {
			return nil, ErrReviewNoPermission
		}

		planSchoolID := s.resolveSchoolID(
			ctx,
			lessonPlan,
		)
		if !reviewScopeContainsID(
			scope.SchoolIDs,
			planSchoolID,
		) {
			return nil, ErrReviewNoPermission
		}
	}

	reviews, err :=
		repository.ListReviewsV2ByPlan(
			ctx,
			planID,
		)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews =
			[]*models.ReviewV2ListItem{}
	}

	return &models.ReviewHistoryResponse{
		Reviews:      reviews,
		Total:        len(reviews),
		CurrentLevel: lessonPlan.ReviewLevel,
	}, nil
}

// GetPendingReviews 获取当前用户的待审核教案。
func (s *ReviewV2Service) GetPendingReviews(
	ctx context.Context,
	userID string,
	userRole string,
	limit int,
	offset int,
) (*models.PendingReviewListResponse, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	switch userRole {
	case models.RoleOperator,
		models.RoleViewer:
		items, total, err :=
			repository.ListPendingReviewsL1(
				ctx,
				userID,
				limit,
				offset,
			)
		if err != nil {
			return nil, err
		}

		return &models.PendingReviewListResponse{
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
				repository.ListPendingReviewsL1BySchool(
					ctx,
					schoolID,
					100,
					0,
				)
			if l1Err != nil {
				return nil, l1Err
			}

			l2Items, _, l2Err :=
				repository.ListPendingReviewsL2(
					ctx,
					schoolID,
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

			return &models.PendingReviewListResponse{
				Items: allItems,
				Total: len(allItems),
			}, nil
		}

		l1Items, _, err :=
			repository.ListPendingReviewsL1(
				ctx,
				userID,
				100,
				0,
			)
		if err != nil {
			return nil, err
		}
		if l1Items == nil {
			l1Items =
				[]*models.PendingReviewItem{}
		}

		return &models.PendingReviewListResponse{
			Items: l1Items,
			Total: len(l1Items),
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

		l1Items, _, err :=
			repository.ListPendingLessonReviewsBySchoolsAndDomain(
				ctx,
				scope.SchoolIDs,
				models.ReviewLevelL1,
				scope.EducationDomain,
				limit,
				offset,
			)
		if err != nil {
			return nil, err
		}

		l2Items, _, err :=
			repository.ListPendingLessonReviewsBySchoolsAndDomain(
				ctx,
				scope.SchoolIDs,
				models.ReviewLevelL2,
				scope.EducationDomain,
				limit,
				offset,
			)
		if err != nil {
			return nil, err
		}

		allItems := append(
			l1Items,
			l2Items...,
		)

		return &models.PendingReviewListResponse{
			Items: allItems,
			Total: len(allItems),
		}, nil

	case models.RoleAdmin:
		l1Items, _, l1Err :=
			repository.ListPendingReviewsL1All(
				ctx,
				100,
				0,
			)
		if l1Err != nil {
			return nil, l1Err
		}

		l2Items, _, l2Err :=
			repository.ListPendingReviewsL2(
				ctx,
				"",
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

		return &models.PendingReviewListResponse{
			Items: allItems,
			Total: len(allItems),
		}, nil

	default:
		return &models.PendingReviewListResponse{
			Items: []*models.PendingReviewItem{},
			Total: 0,
		}, nil
	}
}

// GetReviewStats 获取审核统计。
//
// region_admin 返回辖区同域聚合结果，不使用个人 reviewer_id 口径。
func (s *ReviewV2Service) GetReviewStats(
	ctx context.Context,
	reviewerID string,
	userRole string,
	level int,
) (*models.ReviewStatsResponse, error) {
	if userRole == models.RoleRegionAdmin {
		scope, err :=
			ResolveRegionAdminEducationScope(
				ctx,
				reviewerID,
			)
		if err != nil {
			return nil, err
		}

		return repository.GetLessonReviewStatsBySchoolsAndDomain(
			ctx,
			level,
			scope.SchoolIDs,
			scope.EducationDomain,
		)
	}

	isAdmin := userRole == models.RoleAdmin

	var groupIDs []string
	schoolID := ""

	if !isAdmin {
		switch userRole {
		case models.RoleSeniorOperator:
			if school, err :=
				repository.GetSchoolByAdminUserID(
					ctx,
					reviewerID,
				); err == nil &&
				school != nil {
				schoolID = school.ID
			} else if level ==
				models.ReviewLevelL1 {
				groupIDs, _ =
					repository.GetUserLeadOrBackboneGroupIDs(
						ctx,
						reviewerID,
					)
			}

		default:
			if level == models.ReviewLevelL1 {
				groupIDs, _ =
					repository.GetUserLeadOrBackboneGroupIDs(
						ctx,
						reviewerID,
					)
			}
		}
	}

	return repository.GetReviewStats(
		ctx,
		reviewerID,
		level,
		isAdmin,
		groupIDs,
		schoolID,
	)
}

// GetReviewedRecords 获取已审核记录。
//
// 非 admin 仍按本人审核产出查询。
// region_admin 的辖区聚合要求只作用于统计，不扩大已审核明细的可见范围。
func (s *ReviewV2Service) GetReviewedRecords(
	ctx context.Context,
	reviewerID string,
	userRole string,
	level int,
	decision string,
	limit int,
	offset int,
) (*models.ReviewedListResponse, error) {
	isAdmin := userRole == models.RoleAdmin

	items, total, err :=
		repository.ListReviewedRecords(
			ctx,
			reviewerID,
			level,
			decision,
			isAdmin,
			limit,
			offset,
		)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items =
			[]*models.ReviewedListItem{}
	}

	return &models.ReviewedListResponse{
		Items: items,
		Total: total,
	}, nil
}

// GetReviewFlowConfig 获取学校审核流程配置。
func (s *ReviewV2Service) GetReviewFlowConfig(
	ctx context.Context,
	schoolID string,
) (*models.ReviewFlowConfigResponse, error) {
	config, err := repository.GetReviewFlowConfig(
		ctx,
		schoolID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrReviewConfigNotFound,
		) {
			school, _ :=
				repository.GetOrganizationByID(
					ctx,
					schoolID,
				)

			schoolName := ""
			if school != nil {
				schoolName = school.Name
			}

			return &models.ReviewFlowConfigResponse{
				SchoolID:              schoolID,
				SchoolName:            schoolName,
				L2Enabled:             false,
				L3SampleRate:          0.20,
				AutoPublishOnApproved: false,
			}, nil
		}

		return nil, err
	}

	school, _ := repository.GetOrganizationByID(
		ctx,
		config.SchoolID,
	)

	schoolName := ""
	if school != nil {
		schoolName = school.Name
	}

	return &models.ReviewFlowConfigResponse{
		SchoolID:              config.SchoolID,
		SchoolName:            schoolName,
		L2Enabled:             config.L2Enabled,
		L3SampleRate:          config.L3SampleRate,
		AutoPublishOnApproved: config.AutoPublishOnApproved,
	}, nil
}

// UpdateReviewFlowConfig 更新学校审核流程配置。
func (s *ReviewV2Service) UpdateReviewFlowConfig(
	ctx context.Context,
	schoolID string,
	req *models.UpdateReviewFlowConfigRequest,
	updatedBy string,
) error {
	if req.L3SampleRate < 0 ||
		req.L3SampleRate > 1.0 {
		return errors.New(
			"抽查比例必须在 0.00 - 1.00 之间",
		)
	}

	return repository.UpsertReviewFlowConfig(
		ctx,
		schoolID,
		req,
		updatedBy,
	)
}
