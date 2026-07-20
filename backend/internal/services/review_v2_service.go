package services

// review_v2_service.go
//
// 教案多级审核核心写业务：
//   - L1 教研组审核；
//   - L2 学校审核；
//   - 审核状态流转；
//   - 审核通知；
//   - 符合条件时旁路触发组件萃取。
//
// 查询、统计和配置接口已经拆至 review_v2_query_service.go。
// 内部辅助方法已经拆至 review_v2_helpers.go。
//
// 上下文 6 的区域管理员同域查询规则由查询文件负责；
// 本文件不扩大区域管理员审核决策权，region_admin 仍然只是只读审核视图。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrReviewNotSubmitted = errors.New(
		"只有已提交评审的教案可以审核",
	)
	ErrReviewNotL2Status = errors.New(
		"该教案不在L2待审核状态",
	)
	ErrReviewNoPermission = errors.New(
		"您没有审核此教案的权限",
	)
	ErrReviewInvalidDecision = errors.New(
		"审核决策无效，可选值：approved/revision",
	)
	ErrReviewPlanNotFound = errors.New(
		"教案不存在",
	)
	ErrReviewContentEmpty = errors.New(
		"教案正文为空，无法审核通过；请退回修改，由作者补全正文后再提交",
	)
)

var reviewLog = logger.WithModule("review_v2")

// ReviewV2Service 多级审核服务。
type ReviewV2Service struct {
	compService *ComponentService
}

// NewReviewV2Service 创建多级审核服务。
func NewReviewV2Service(
	compService *ComponentService,
) *ReviewV2Service {
	return &ReviewV2Service{
		compService: compService,
	}
}

// ==================== L1 教研组审核 ====================

// ReviewL1 执行教案 L1 审核。
//
// 权限阶梯：
//   - admin；
//   - 教案审核学校的 senior_operator；
//   - 教案绑定教研组的 lead/backbone。
//
// region_admin 不具备审核决策权，只能通过同域辖区视图读取。
func (s *ReviewV2Service) ReviewL1(
	ctx context.Context,
	planID string,
	reviewerID string,
	req *models.ReviewDecisionV2Request,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		planID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return ErrReviewPlanNotFound
		}
		return err
	}

	if lessonPlan.Status != models.LPStatusSubmitted ||
		lessonPlan.ReviewLevel != 0 {
		return ErrReviewNotSubmitted
	}

	reviewerRole := ""
	if user, findErr := repository.FindUserByID(
		ctx,
		reviewerID,
	); findErr == nil {
		reviewerRole = user.Role
	}

	allowed := reviewerRole == models.RoleAdmin

	if !allowed &&
		reviewerRole == models.RoleSeniorOperator {
		if school, schoolErr :=
			repository.GetSchoolByAdminUserID(
				ctx,
				reviewerID,
			); schoolErr == nil &&
			school != nil {
			planSchoolID := s.resolveSchoolID(
				ctx,
				lessonPlan,
			)
			allowed = planSchoolID != "" &&
				planSchoolID == school.ID
		}
	}

	if !allowed {
		if lessonPlan.GroupID == nil ||
			*lessonPlan.GroupID == "" {
			return ErrReviewNoPermission
		}

		hasPermission, permissionErr :=
			repository.IsGroupLeadOrBackbone(
				ctx,
				*lessonPlan.GroupID,
				reviewerID,
			)
		if permissionErr != nil {
			return fmt.Errorf(
				"校验审核权限失败: %w",
				permissionErr,
			)
		}

		allowed = hasPermission
	}

	if !allowed {
		return ErrReviewNoPermission
	}

	if req == nil ||
		(req.Decision != models.ReviewDecisionApproved &&
			req.Decision != models.ReviewDecisionRevision) {
		return ErrReviewInvalidDecision
	}

	// 只有通过决策要求正文非空。
	// 退回空正文是合法且必要的纠错行为。
	if req.Decision == models.ReviewDecisionApproved &&
		strings.TrimSpace(
			lessonPlan.ContentMarkdown,
		) == "" {
		reviewLog.Info(
			"L1审核通过被拦截：教案正文为空",
			"plan_id",
			planID,
			"reviewer",
			reviewerID,
		)
		return ErrReviewContentEmpty
	}

	existingCount, _ :=
		repository.CountReviewsV2ByPlanAndLevel(
			ctx,
			planID,
			models.ReviewLevelL1,
		)
	round := existingCount + 1

	review := &models.ReviewV2{
		LessonPlanID: planID,
		ReviewLevel:  models.ReviewLevelL1,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}

	if err := repository.CreateReviewV2(
		ctx,
		review,
	); err != nil {
		reviewLog.Error(
			"创建L1审核记录失败",
			"plan_id",
			planID,
			"error",
			err,
		)
		return err
	}

	s.syncLegacyReview(
		ctx,
		planID,
		reviewerID,
		req,
		round,
	)

	switch req.Decision {
	case models.ReviewDecisionApproved:
		schoolID := s.resolveSchoolID(
			ctx,
			lessonPlan,
		)

		var schoolIDPtr *string
		if schoolID != "" {
			schoolIDPtr = &schoolID
		}

		needL2 := false
		if schoolID != "" {
			config, configErr :=
				repository.GetReviewFlowConfig(
					ctx,
					schoolID,
				)
			if configErr == nil &&
				config.L2Enabled {
				needL2 = true
			}
		}

		if needL2 {
			_ = repository.UpdateLessonPlanReviewLevel(
				ctx,
				planID,
				models.ReviewLevelL1,
				schoolIDPtr,
			)

			reviewLog.Info(
				"L1审核通过，进入L2待审核",
				"plan_id",
				planID,
				"school_id",
				schoolID,
				"round",
				round,
			)
		} else {
			_ = repository.UpdateLessonPlanStatus(
				ctx,
				planID,
				models.LPStatusApproved,
			)
			_ = repository.UpdateLessonPlanReviewLevel(
				ctx,
				planID,
				models.ReviewLevelL1,
				schoolIDPtr,
			)

			reviewLog.Info(
				"L1审核通过，直接终审",
				"plan_id",
				planID,
				"round",
				round,
			)

			s.triggerAutoExtractIfEligible(
				lessonPlan,
				reviewerID,
			)

			notifyLPAuthorReviewResult(
				ctx,
				lessonPlan,
				reviewerID,
				models.ReviewDecisionApproved,
				req.Comment,
			)
		}

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusRevision,
		)
		_ = repository.UpdateLessonPlanReviewLevel(
			ctx,
			planID,
			0,
			nil,
		)

		if restoreErr :=
			repository.RestoreArchivedAnnotationsForLatestRound(
				ctx,
				planID,
			); restoreErr != nil {
			reviewLog.Error(
				"恢复归档批注失败（不影响退回）",
				"plan_id",
				planID,
				"error",
				restoreErr,
			)
		}

		reviewLog.Info(
			"L1审核退回",
			"plan_id",
			planID,
			"round",
			round,
		)

		notifyLPAuthorReviewResult(
			ctx,
			lessonPlan,
			reviewerID,
			models.ReviewDecisionRevision,
			req.Comment,
		)
	}

	return nil
}

// ==================== L2 学校审核 ====================

// ReviewL2 执行教案 L2 审核。
//
// 仅 admin 或审核学校的 senior_operator 可以执行。
func (s *ReviewV2Service) ReviewL2(
	ctx context.Context,
	planID string,
	reviewerID string,
	reviewerRole string,
	req *models.ReviewDecisionV2Request,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		planID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return ErrReviewPlanNotFound
		}
		return err
	}

	if lessonPlan.Status != models.LPStatusSubmitted ||
		lessonPlan.ReviewLevel != models.ReviewLevelL1 {
		return ErrReviewNotL2Status
	}

	if reviewerRole != models.RoleSeniorOperator &&
		reviewerRole != models.RoleAdmin {
		return ErrReviewNoPermission
	}

	if reviewerRole == models.RoleSeniorOperator {
		school, schoolErr :=
			repository.GetSchoolByAdminUserID(
				ctx,
				reviewerID,
			)
		if schoolErr != nil ||
			lessonPlan.ReviewSchoolID == nil ||
			*lessonPlan.ReviewSchoolID != school.ID {
			return ErrReviewNoPermission
		}
	}

	if req == nil ||
		(req.Decision != models.ReviewDecisionApproved &&
			req.Decision != models.ReviewDecisionRevision) {
		return ErrReviewInvalidDecision
	}

	if req.Decision == models.ReviewDecisionApproved &&
		strings.TrimSpace(
			lessonPlan.ContentMarkdown,
		) == "" {
		reviewLog.Info(
			"L2审核通过被拦截：教案正文为空",
			"plan_id",
			planID,
			"reviewer",
			reviewerID,
		)
		return ErrReviewContentEmpty
	}

	existingCount, _ :=
		repository.CountReviewsV2ByPlanAndLevel(
			ctx,
			planID,
			models.ReviewLevelL2,
		)
	round := existingCount + 1

	review := &models.ReviewV2{
		LessonPlanID: planID,
		ReviewLevel:  models.ReviewLevelL2,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}

	if err := repository.CreateReviewV2(
		ctx,
		review,
	); err != nil {
		reviewLog.Error(
			"创建L2审核记录失败",
			"plan_id",
			planID,
			"error",
			err,
		)
		return err
	}

	s.syncLegacyReview(
		ctx,
		planID,
		reviewerID,
		req,
		round,
	)

	switch req.Decision {
	case models.ReviewDecisionApproved:
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusApproved,
		)
		_ = repository.UpdateLessonPlanReviewLevel(
			ctx,
			planID,
			models.ReviewLevelL2,
			nil,
		)

		reviewLog.Info(
			"L2审核通过",
			"plan_id",
			planID,
			"round",
			round,
		)

		s.triggerAutoExtractIfEligible(
			lessonPlan,
			reviewerID,
		)

		notifyLPAuthorReviewResult(
			ctx,
			lessonPlan,
			reviewerID,
			models.ReviewDecisionApproved,
			req.Comment,
		)

	case models.ReviewDecisionRevision:
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusRevision,
		)
		_ = repository.UpdateLessonPlanReviewLevel(
			ctx,
			planID,
			0,
			nil,
		)

		if restoreErr :=
			repository.RestoreArchivedAnnotationsForLatestRound(
				ctx,
				planID,
			); restoreErr != nil {
			reviewLog.Error(
				"恢复归档批注失败（不影响退回）",
				"plan_id",
				planID,
				"error",
				restoreErr,
			)
		}

		reviewLog.Info(
			"L2审核退回",
			"plan_id",
			planID,
			"round",
			round,
		)

		notifyLPAuthorReviewResult(
			ctx,
			lessonPlan,
			reviewerID,
			models.ReviewDecisionRevision,
			req.Comment,
		)
	}

	return nil
}
