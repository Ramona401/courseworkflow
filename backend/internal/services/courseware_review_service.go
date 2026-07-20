package services

// courseware_review_service.go
//
// 课件多级审核核心写业务：
//   - 作者提交课件审核；
//   - L1 教研组审核；
//   - L2 学校审核；
//   - 审核状态流转；
//   - 审核结果通知。
//
// 待审列表、统计、审核历史、已审核记录和审核详情已经拆至：
//   courseware_review_query_service.go
//
// 权限裁决辅助方法位于：
//   courseware_review_access.go
//
// 通知旁路位于：
//   courseware_review_notify.go
//
// 上下文 6 不扩大区域管理员审核决策权：
//   region_admin 只拥有辖区同域审核查看权限，不能执行 L1 或 L2 决策。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrCWReviewCoursewareNotFound = errors.New(
		"课件不存在",
	)
	ErrCWReviewNotSubmitted = errors.New(
		"只有已提交审核的课件可以审核",
	)
	ErrCWReviewNotL2Status = errors.New(
		"该课件不在L2待审核状态",
	)
	ErrCWReviewNoPermission = errors.New(
		"您没有审核此课件的权限",
	)
	ErrCWReviewInvalidDecision = errors.New(
		"审核决策无效，可选值：approved/revision",
	)
	ErrCWSubmitNotOwner = errors.New(
		"只有课件作者本人可以提交审核",
	)
	ErrCWSubmitNotReady = errors.New(
		"课件尚未生成完成，请先完成课件（至少进入预览阶段）再提交审核",
	)
	ErrCWSubmitWrongState = errors.New(
		"当前状态不可提交审核（仅私有/个人发布/已退回的课件可提交）",
	)
	ErrCWSubmitNoSchool = errors.New(
		"无法确定您所属的学校，请联系管理员配置组织归属后再提交审核",
	)
)

var cwReviewLog = logger.WithModule(
	"courseware_review",
)

// CoursewareReviewService 课件审核服务。
type CoursewareReviewService struct{}

// NewCoursewareReviewService 创建课件审核服务。
func NewCoursewareReviewService() *CoursewareReviewService {
	return &CoursewareReviewService{}
}

// ==================== 提交审核 ====================

// SubmitForReview 作者提交课件进入审核流程。
//
// 前置条件：
//   - Actor 已认证且具有确定教育域；
//   - 只有课件作者本人可以提交；
//   - 课件至少进入 preview 阶段；
//   - publish_state 为 private、published_personal 或 revision；
//   - 能够确定作者所属学校。
//
// 成功后：
//
//	publish_state=submitted
//	review_level=0
//	review_school_id=作者学校
func (s *CoursewareReviewService) SubmitForReview(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if actor == nil ||
		actor.UserID == "" {
		return ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}

	if err := ValidateCoursewareReviewEducationDomain(
		actor,
		courseware,
	); err != nil {
		return err
	}

	if courseware.UserID != actor.UserID {
		return ErrCWSubmitNotOwner
	}

	if models.CoursewareStatusOrder[courseware.Status] <
		models.CoursewareStatusOrder[models.CoursewareStatusPreview] {
		return ErrCWSubmitNotReady
	}

	switch courseware.PublishState {
	case models.CWPublishPrivate,
		models.CWPublishPublishedPersonal,
		models.CWPublishRevision:
	default:
		return ErrCWSubmitWrongState
	}

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			actor.UserID,
		)
	if schoolID == "" {
		return ErrCWSubmitNoSchool
	}

	if err := repository.UpdateCoursewarePublishState(
		ctx,
		coursewareID,
		models.CWPublishSubmitted,
		0,
		&schoolID,
	); err != nil {
		return err
	}

	cwReviewLog.Info(
		"课件提交审核",
		"courseware_id",
		coursewareID,
		"author",
		actor.UserID,
		"school_id",
		schoolID,
		"education_domain",
		courseware.EducationDomain,
	)

	s.notifyL1ReviewersOnSubmit(
		ctx,
		courseware,
	)

	return nil
}

// ==================== L1 教研组审核 ====================

// ReviewL1 执行课件 L1 审核。
//
// 权限阶梯：
//   - admin；
//   - 审核学校的 senior_operator；
//   - 作者所在教研组的 lead/backbone。
//
// region_admin 不在决策权限阶梯中。
func (s *CoursewareReviewService) ReviewL1(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) error {
	if actor == nil ||
		actor.UserID == "" {
		return ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}

	if err := ValidateCoursewareReviewEducationDomain(
		actor,
		courseware,
	); err != nil {
		return err
	}

	if courseware.PublishState !=
		models.CWPublishSubmitted ||
		courseware.ReviewLevel != 0 {
		return ErrCWReviewNotSubmitted
	}

	reviewerID := actor.UserID
	reviewerRole := actor.Role

	allowed := reviewerRole == models.RoleAdmin

	if !allowed &&
		reviewerRole == models.RoleSeniorOperator {
		allowed = s.isSeniorOfReviewSchool(
			ctx,
			courseware,
			reviewerID,
		)
	}

	if !allowed {
		hasPermission, permissionErr :=
			s.isReviewerInAuthorGroupAsLeadOrBackbone(
				ctx,
				courseware.UserID,
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
		return ErrCWReviewNoPermission
	}

	if req == nil ||
		(req.Decision !=
			models.ReviewDecisionApproved &&
			req.Decision !=
				models.ReviewDecisionRevision) {
		return ErrCWReviewInvalidDecision
	}

	existingCount, _ :=
		repository.CountCoursewareReviewsByLevel(
			ctx,
			coursewareID,
			models.ReviewLevelL1,
		)
	round := existingCount + 1

	review := &models.CoursewareReview{
		CoursewareID: coursewareID,
		ReviewLevel:  models.ReviewLevelL1,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}

	if err := repository.CreateCoursewareReview(
		ctx,
		review,
	); err != nil {
		return err
	}

	switch req.Decision {
	case models.ReviewDecisionApproved:
		schoolID := s.resolveReviewSchoolID(
			ctx,
			courseware,
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
			_ = repository.UpdateCoursewarePublishState(
				ctx,
				coursewareID,
				models.CWPublishSubmitted,
				models.ReviewLevelL1,
				schoolIDPtr,
			)

			cwReviewLog.Info(
				"课件L1审核通过，进入L2",
				"courseware_id",
				coursewareID,
				"school_id",
				schoolID,
				"round",
				round,
			)
		} else {
			_ = repository.UpdateCoursewarePublishState(
				ctx,
				coursewareID,
				models.CWPublishApproved,
				models.ReviewLevelL1,
				schoolIDPtr,
			)

			cwReviewLog.Info(
				"课件L1审核通过，直接终审",
				"courseware_id",
				coursewareID,
				"round",
				round,
			)

			s.notifyAuthorReviewResult(
				ctx,
				courseware,
				reviewerID,
				models.ReviewDecisionApproved,
				"",
			)
		}

	case models.ReviewDecisionRevision:
		_ = repository.UpdateCoursewarePublishState(
			ctx,
			coursewareID,
			models.CWPublishRevision,
			0,
			nil,
		)

		cwReviewLog.Info(
			"课件L1审核退回",
			"courseware_id",
			coursewareID,
			"round",
			round,
		)

		s.notifyAuthorReviewResult(
			ctx,
			courseware,
			reviewerID,
			models.ReviewDecisionRevision,
			req.Comment,
		)
	}

	return nil
}

// ==================== L2 学校审核 ====================

// ReviewL2 执行课件 L2 审核。
//
// 只有 admin 或审核学校的 senior_operator 可以执行。
func (s *CoursewareReviewService) ReviewL2(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) error {
	if actor == nil ||
		actor.UserID == "" {
		return ErrCoursewareActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}

	if err := ValidateCoursewareReviewEducationDomain(
		actor,
		courseware,
	); err != nil {
		return err
	}

	if courseware.PublishState !=
		models.CWPublishSubmitted ||
		courseware.ReviewLevel !=
			models.ReviewLevelL1 {
		return ErrCWReviewNotL2Status
	}

	reviewerID := actor.UserID
	reviewerRole := actor.Role

	if reviewerRole !=
		models.RoleSeniorOperator &&
		reviewerRole !=
			models.RoleAdmin {
		return ErrCWReviewNoPermission
	}

	if reviewerRole == models.RoleSeniorOperator {
		school, schoolErr :=
			repository.GetSchoolByAdminUserID(
				ctx,
				reviewerID,
			)
		if schoolErr != nil ||
			courseware.ReviewSchoolID == nil ||
			*courseware.ReviewSchoolID !=
				school.ID {
			return ErrCWReviewNoPermission
		}
	}

	if req == nil ||
		(req.Decision !=
			models.ReviewDecisionApproved &&
			req.Decision !=
				models.ReviewDecisionRevision) {
		return ErrCWReviewInvalidDecision
	}

	existingCount, _ :=
		repository.CountCoursewareReviewsByLevel(
			ctx,
			coursewareID,
			models.ReviewLevelL2,
		)
	round := existingCount + 1

	review := &models.CoursewareReview{
		CoursewareID: coursewareID,
		ReviewLevel:  models.ReviewLevelL2,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}

	if err := repository.CreateCoursewareReview(
		ctx,
		review,
	); err != nil {
		return err
	}

	switch req.Decision {
	case models.ReviewDecisionApproved:
		_ = repository.UpdateCoursewarePublishState(
			ctx,
			coursewareID,
			models.CWPublishApproved,
			models.ReviewLevelL2,
			courseware.ReviewSchoolID,
		)

		cwReviewLog.Info(
			"课件L2审核通过",
			"courseware_id",
			coursewareID,
			"round",
			round,
		)

		s.notifyAuthorReviewResult(
			ctx,
			courseware,
			reviewerID,
			models.ReviewDecisionApproved,
			"",
		)

	case models.ReviewDecisionRevision:
		_ = repository.UpdateCoursewarePublishState(
			ctx,
			coursewareID,
			models.CWPublishRevision,
			0,
			nil,
		)

		cwReviewLog.Info(
			"课件L2审核退回",
			"courseware_id",
			coursewareID,
			"round",
			round,
		)

		s.notifyAuthorReviewResult(
			ctx,
			courseware,
			reviewerID,
			models.ReviewDecisionRevision,
			req.Comment,
		)
	}

	return nil
}
