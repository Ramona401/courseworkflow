package services

// courseware_review_service.go
//
// 课件多级审核核心写业务：
//
//   - 作者首次提交或退回后重新提交课件；
//   - L1教研组审核；
//   - L2学校审核；
//   - 审核状态流转；
//   - 审核结果通知。
//
// L1/L2权限判断、AI反馈装配和原子审核决定位于：
// courseware_review_decision_service.go。
//
// 通知旁路位于courseware_review_notify.go。
//
// V1.3重新提交边界：
//
//   - 课件状态更新和旧问题复审轮次登记必须位于同一事务；
//   - 未解决的正式问题保留原问题身份；
//   - 问题重新进入它原本所属的审核级别；
//   - 页面修改成功不自动表示问题已经解决。

import (
	"context"
	"errors"

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

	ErrCWReviewFeedbackInvalid = errors.New(
		"课件AI审核反馈参数无效或已发生变化",
	)

	ErrCWSubmitNotOwner = errors.New(
		"只有课件作者本人可以提交审核",
	)

	ErrCWSubmitNotReady = errors.New(
		"课件尚未生成完成，请先完成课件（至少进入预览阶段）再提交审核",
	)

	ErrCWSubmitWrongState = errors.New(
		"当前状态不可提交审核（仅私有、个人发布或已退回的课件可提交）",
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

// SubmitForReview 作者首次提交或退回后重新提交课件。
func (s *CoursewareReviewService) SubmitForReview(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if actor == nil ||
		actor.UserID == "" {
		return ErrCoursewareActorRequired
	}

	courseware, err := repository.GetCoursewareByID(
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

	currentStatusOrder :=
		models.CoursewareStatusOrder[courseware.Status]

	previewStatusOrder :=
		models.CoursewareStatusOrder[models.CoursewareStatusPreview]

	if currentStatusOrder < previewStatusOrder {
		return ErrCWSubmitNotReady
	}

	switch courseware.PublishState {
	case models.CWPublishPrivate,
		models.CWPublishPublishedPersonal,
		models.CWPublishRevision:
	default:
		return ErrCWSubmitWrongState
	}

	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		actor.UserID,
	)
	if schoolID == "" {
		return ErrCWSubmitNoSchool
	}

	result, err :=
		repository.CommitCoursewareReviewSubmission(
			ctx,
			coursewareID,
			actor.UserID,
			schoolID,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCWReviewSubmissionCoursewareNotFound,
		):
			return ErrCWReviewCoursewareNotFound

		case errors.Is(
			err,
			repository.ErrCWReviewSubmissionOwnerMismatch,
		):
			return ErrCWSubmitNotOwner

		case errors.Is(
			err,
			repository.ErrCWReviewSubmissionStateConflict,
		):
			return ErrCWSubmitWrongState

		default:
			return err
		}
	}

	var carryoverCount int64
	if result != nil {
		carryoverCount =
			result.CarryoverItemCount
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
		"carryover_review_item_count",
		carryoverCount,
	)

	s.notifyL1ReviewersOnSubmit(
		ctx,
		courseware,
	)

	return nil
}

// ==================== L1 / L2 审核 ====================

// ReviewL1 执行课件L1审核。
func (s *CoursewareReviewService) ReviewL1(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) error {
	return s.reviewCoursewareAtLevel(
		ctx,
		coursewareID,
		actor,
		req,
		models.ReviewLevelL1,
	)
}

// ReviewL2 执行课件L2审核。
func (s *CoursewareReviewService) ReviewL2(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) error {
	return s.reviewCoursewareAtLevel(
		ctx,
		coursewareID,
		actor,
		req,
		models.ReviewLevelL2,
	)
}
