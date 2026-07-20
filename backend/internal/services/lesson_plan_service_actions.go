package services

// lesson_plan_service_actions.go — 教案状态动作与课件开发
//
// 本文件从lesson_plan_service.go拆出，承载：
//   - 个人发布；
//   - 提交评审；
//   - 评审教案；
//   - 共享发布；
//   - 进入课件开发。
//
// 教案Fork已经进一步拆分到lesson_plan_fork_service.go，
// 避免本文件超过600行。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// StartDevelopmentResult 进入课件开发的返回结果。
type StartDevelopmentResult struct {
	PipelineID string `json:"pipeline_id"`
	Message    string `json:"message"`
}

// ==================== 教案状态流转 ====================

// PublishPersonal 个人发布。
func (s *LessonPlanService) PublishPersonal(
	ctx context.Context,
	id string,
	callerID string,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lessonPlan.AuthorID != callerID {
		return ErrLPNotAuthor
	}

	if lessonPlan.Status != models.LPStatusDraft &&
		lessonPlan.Status != models.LPStatusRevision &&
		lessonPlan.Status != models.LPStatusPublishedPersonal {
		return errors.New(
			"只有草稿、退回或已发布状态的教案可以个人发布",
		)
	}

	if strings.TrimSpace(
		lessonPlan.ContentMarkdown,
	) == "" {
		lpLog.Info(
			"个人发布被拦截：教案正文为空",
			"plan_id", id,
		)
		return ErrLPContentEmpty
	}

	return repository.UpdateLessonPlanStatus(
		ctx,
		id,
		models.LPStatusPublishedPersonal,
	)
}

// SubmitForReview 提交评审。
func (s *LessonPlanService) SubmitForReview(
	ctx context.Context,
	id string,
	callerID string,
	groupID string,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		return s.mapNotFoundErr(err)
	}
	if lessonPlan.AuthorID != callerID {
		return ErrLPNotAuthor
	}
	if groupID == "" {
		return ErrLPGroupRequired
	}
	if lessonPlan.Status != models.LPStatusDraft &&
		lessonPlan.Status != models.LPStatusPublishedPersonal &&
		lessonPlan.Status != models.LPStatusRevision {
		return ErrLPCannotSubmit
	}
	if strings.TrimSpace(
		lessonPlan.ContentMarkdown,
	) == "" {
		lpLog.Info(
			"提交评审被拦截：教案正文为空",
			"plan_id", id,
		)
		return ErrLPContentEmpty
	}

	if archiveErr :=
		repository.ArchiveAnnotationsByPlanID(
			ctx,
			id,
		); archiveErr != nil {
		lpLog.Error(
			"归档批注失败（不影响提交）",
			"plan_id", id,
			"error", archiveErr,
		)
	} else {
		lpLog.Info(
			"已归档本轮待处理批注",
			"plan_id", id,
		)
	}

	if err := repository.UpdateLessonPlanVisibility(
		ctx,
		id,
		models.LPVisibilityGroup,
		&groupID,
	); err != nil {
		return err
	}

	schoolID := ""
	if group, groupErr :=
		repository.GetTeachingGroupByID(
			ctx,
			groupID,
		); groupErr == nil {
		schoolID = group.SchoolID
	}
	if schoolID != "" {
		_ = repository.UpdateLessonPlanReviewLevel(
			ctx,
			id,
			0,
			&schoolID,
		)
	}

	return repository.UpdateLessonPlanStatus(
		ctx,
		id,
		models.LPStatusSubmitted,
	)
}

// ReviewLessonPlan 评审教案。
func (s *LessonPlanService) ReviewLessonPlan(
	ctx context.Context,
	planID string,
	reviewerID string,
	req *models.CreateLessonPlanReviewRequest,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		planID,
	)
	if err != nil {
		return s.mapNotFoundErr(err)
	}

	if lessonPlan.Status != models.LPStatusSubmitted {
		return errors.New(
			"只有已提交评审的教案可以评审",
		)
	}

	if req.Decision != "approved" &&
		req.Decision != "revision" &&
		req.Decision != "rejected" {
		return errors.New(
			"评审决策无效，可选值：approved/revision/rejected",
		)
	}

	existingReviews, _ :=
		repository.ListLessonPlanReviews(
			ctx,
			planID,
		)
	round := len(existingReviews) + 1

	review := &models.LessonPlanReview{
		LessonPlanID: planID,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Dimensions:   req.Dimensions,
		Comments:     req.Comments,
		Suggestions:  req.Suggestions,
		Round:        round,
	}
	if err := repository.CreateLessonPlanReview(
		ctx,
		review,
	); err != nil {
		lpLog.Error(
			"创建评审记录失败",
			"plan_id", planID,
			"error", err,
		)
		return err
	}

	switch req.Decision {
	case "approved":
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusApproved,
		)

	case "revision":
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusRevision,
		)

		if restoreErr :=
			repository.RestoreArchivedAnnotationsForLatestRound(
				ctx,
				planID,
			); restoreErr != nil {
			lpLog.Error(
				"恢复归档批注失败（不影响退回）",
				"plan_id", planID,
				"error", restoreErr,
			)
		} else {
			lpLog.Info(
				"已恢复最新一轮归档批注为待处理",
				"plan_id", planID,
			)
		}

	case "rejected":
		_ = repository.UpdateLessonPlanStatus(
			ctx,
			planID,
			models.LPStatusDraft,
		)
		_ = repository.UpdateLessonPlanVisibility(
			ctx,
			planID,
			models.LPVisibilityPersonal,
			nil,
		)
	}

	lpLog.Info(
		"教案评审完成",
		"plan_id", planID,
		"decision", req.Decision,
		"round", round,
	)

	if req.Decision == "approved" &&
		lessonPlan.AIReviewScore != nil &&
		*lessonPlan.AIReviewScore >= 8.5 {
		planContent := lessonPlan.ContentMarkdown
		subject := lessonPlan.Subject
		grade := lessonPlan.Grade
		aiScore := *lessonPlan.AIReviewScore

		go func() {
			backgroundContext := context.Background()

			lpLog.Info(
				"触发通道二自动萃取",
				"plan_id", planID,
				"ai_score", aiScore,
			)

			if err := s.compService.AutoExtractFromLessonPlan(
				backgroundContext,
				planID,
				planContent,
				subject,
				grade,
				reviewerID,
			); err != nil {
				lpLog.Error(
					"通道二自动萃取失败",
					"plan_id", planID,
					"error", err,
				)
			}
		}()
	}

	return nil
}

// PublishShared 共享发布。
func (s *LessonPlanService) PublishShared(
	ctx context.Context,
	id string,
	callerID string,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		return s.mapNotFoundErr(err)
	}

	isAuthor := lessonPlan.AuthorID == callerID
	isAdmin := false
	if !isAuthor {
		if user, userErr := repository.FindUserByID(
			ctx,
			callerID,
		); userErr == nil &&
			user.Role == models.RoleAdmin {
			isAdmin = true
		}
	}

	if !isAuthor && !isAdmin {
		lpLog.Info(
			"共享发布被拦截：调用者非作者且非管理员",
			"plan_id", id,
			"caller", callerID,
		)
		return ErrLPNotPublisher
	}

	if lessonPlan.Status != models.LPStatusApproved {
		return errors.New(
			"只有评审通过的教案可以共享发布",
		)
	}

	if strings.TrimSpace(
		lessonPlan.ContentMarkdown,
	) == "" {
		lpLog.Info(
			"共享发布被拦截：教案正文为空",
			"plan_id", id,
		)
		return ErrLPContentEmpty
	}

	return repository.UpdateLessonPlanStatus(
		ctx,
		id,
		models.LPStatusPublishedShared,
	)
}

// ==================== 课件开发 ====================

// StartDevelopment 进入课件开发。
func (s *LessonPlanService) StartDevelopment(
	ctx context.Context,
	id string,
	callerID string,
) (*StartDevelopmentResult, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, s.mapNotFoundErr(err)
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}
	if lessonPlan.Status != models.LPStatusPublishedPersonal &&
		lessonPlan.Status != models.LPStatusApproved &&
		lessonPlan.Status != models.LPStatusPublishedShared {
		return nil, ErrLPCannotDevelop
	}
	if lessonPlan.Status == models.LPStatusDeveloping {
		return nil, ErrLPAlreadyDeveloping
	}

	existingPipeline, pipelineErr :=
		repository.GetPipelineByLessonPlanID(
			id,
		)
	if pipelineErr == nil &&
		existingPipeline != nil {
		lpLog.Info(
			"教案已有关联Pipeline，跳过创建",
			"plan_id", id,
			"pipeline_id", existingPipeline.ID,
		)

		_ = repository.UpdateLessonPlanStatus(
			ctx,
			id,
			models.LPStatusDeveloping,
		)

		return &StartDevelopmentResult{
			PipelineID: existingPipeline.ID,
			Message:    "教案已关联课件开发任务",
		}, nil
	}

	courseCode := fmt.Sprintf(
		"LP-%s-%s",
		lessonPlan.Subject,
		lessonPlan.Grade,
	)
	courseName := fmt.Sprintf(
		"%s %s — %s",
		lessonPlan.Grade,
		lessonPlan.Subject,
		lessonPlan.Topic,
	)

	lessonPlanID := id
	callerIDCopy := callerID

	pipeline := &models.Pipeline{
		CourseCode:   courseCode,
		CourseName:   courseName,
		StartedBy:    &callerIDCopy,
		CurrentStep:  models.StepDbCheck,
		Status:       models.PipelineStatusPending,
		AutoMode:     true,
		ReviewRound:  1,
		LessonPlanID: &lessonPlanID,
	}

	defaultConfig := models.DefaultPipelineConfig()
	configBytes, _ := json.Marshal(defaultConfig)
	pipeline.Config = string(configBytes)

	if err := repository.CreatePipeline(
		pipeline,
	); err != nil {
		lpLog.Error(
			"创建课件开发Pipeline失败",
			"plan_id", id,
			"error", err,
		)
		return nil, fmt.Errorf(
			"创建课件开发任务失败: %w",
			err,
		)
	}

	if err := repository.UpdateLessonPlanStatus(
		ctx,
		id,
		models.LPStatusDeveloping,
	); err != nil {
		lpLog.Error(
			"更新教案状态失败",
			"plan_id", id,
			"error", err,
		)
		return nil, err
	}

	lpLog.Info(
		"教案进入课件开发",
		"plan_id", id,
		"pipeline_id", pipeline.ID,
		"course_code", courseCode,
	)

	return &StartDevelopmentResult{
		PipelineID: pipeline.ID,
		Message:    "已创建课件开发任务，请在课件审核系统继续操作",
	}, nil
}
