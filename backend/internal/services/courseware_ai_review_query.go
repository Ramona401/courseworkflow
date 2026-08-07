package services

// courseware_ai_review_query.go
//
// 课件 AI 审核会话的只读查询服务。
//
// 安全边界：
//   - 会话创建者只能查看自己的会话；
//   - admin 可查看全部会话；
//   - 即使会话属于当前用户，也必须重新校验其对课件的审核查看权限；
//   - 不直接相信会话中旧的角色或教育域信息；
//   - 所有页面、批次和报告只读返回，不修改人工审核决定。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// GetSessionDetail 查询指定会话及其全部批次。
func (s *CoursewareAIReviewService) GetSessionDetail(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAIReviewSession,
	[]*models.CoursewareAIReviewBatch,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, nil, ErrCWAIReviewActorRequired
	}
	if s == nil ||
		s.reviewService == nil ||
		s.coursewareService == nil {
		return nil, nil, errors.New(
			"课件AI审核查询服务未初始化",
		)
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			strings.TrimSpace(sessionID),
		)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrCWAIReviewSessionNotFound
	}

	if session.ReviewerID != actor.UserID &&
		actor.Role != models.RoleAdmin {
		return nil, nil,
			ErrCWAIReviewSessionOwnerMismatch
	}

	if err := s.validateSessionCoursewareAccess(
		ctx,
		session,
		actor,
	); err != nil {
		return nil, nil, err
	}

	batches, err :=
		repository.ListCoursewareAIReviewBatches(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, nil, err
	}

	return session, batches, nil
}

// GetLatestSessionDetail 查询当前审核员对某课件、某审核级别的最新会话。
//
// 没有历史会话时返回 session=nil、空批次数组，不视为错误。
func (s *CoursewareAIReviewService) GetLatestSessionDetail(
	ctx context.Context,
	coursewareID string,
	reviewLevel int,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAIReviewSession,
	[]*models.CoursewareAIReviewBatch,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, nil, ErrCWAIReviewActorRequired
	}
	if s == nil ||
		s.reviewService == nil ||
		s.coursewareService == nil {
		return nil, nil, errors.New(
			"课件AI审核查询服务未初始化",
		)
	}

	session, err :=
		repository.GetLatestCoursewareAIReviewSession(
			ctx,
			strings.TrimSpace(coursewareID),
			actor.UserID,
			reviewLevel,
		)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil,
			[]*models.CoursewareAIReviewBatch{},
			nil
	}

	if err := s.validateSessionCoursewareAccess(
		ctx,
		session,
		actor,
	); err != nil {
		return nil, nil, err
	}

	batches, err :=
		repository.ListCoursewareAIReviewBatches(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, nil, err
	}

	return session, batches, nil
}

// validateSessionCoursewareAccess 重新读取正式课件并执行人工审核权限校验。
func (s *CoursewareAIReviewService) validateSessionCoursewareAccess(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	actor *CoursewareActorContext,
) error {
	if session == nil {
		return ErrCWAIReviewSessionNotFound
	}

	// 自审会话始终重新执行作者专属权限。
	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		_, _, err :=
			s.coursewareService.
				LoadCoursewareForOwnerRuntime(
					ctx,
					session.CoursewareID,
					actor,
				)
		return err
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			session.CoursewareID,
		)
	if err != nil ||
		courseware == nil {
		return ErrCWAIReviewCoursewareNotFound
	}

	allowed, err :=
		s.reviewService.
			CanReviewLoadedCourseware(
				ctx,
				courseware,
				actor,
			)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrCWAIReviewNoPermission
	}

	return nil
}
