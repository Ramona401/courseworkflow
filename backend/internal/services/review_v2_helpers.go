package services

// review_v2_helpers.go
//
// 教案多级审核 Service 的内部辅助方法：
//   - 审核学校解析；
//   - 学校白名单判定；
//   - 旧版审核记录同步；
//   - 高质量教案组件萃取旁路。
//
// 本文件不定义任何 HTTP 或数据库范围规则。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// resolveSchoolID 解析教案审核学校。
//
// 优先级：
//  1. review_school_id；
//  2. lesson_plans.school_id；
//  3. 教案绑定教研组所属学校。
func (s *ReviewV2Service) resolveSchoolID(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) string {
	if lessonPlan == nil {
		return ""
	}

	if lessonPlan.ReviewSchoolID != nil &&
		*lessonPlan.ReviewSchoolID != "" {
		return *lessonPlan.ReviewSchoolID
	}

	if lessonPlan.SchoolID != nil &&
		*lessonPlan.SchoolID != "" {
		return *lessonPlan.SchoolID
	}

	if lessonPlan.GroupID != nil &&
		*lessonPlan.GroupID != "" {
		group, err :=
			repository.GetTeachingGroupByID(
				ctx,
				*lessonPlan.GroupID,
			)
		if err == nil &&
			group != nil {
			return group.SchoolID
		}
	}

	return ""
}

// reviewScopeContainsID 判断目标 ID 是否位于白名单。
func reviewScopeContainsID(
	values []string,
	target string,
) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}

	return false
}

// syncLegacyReview 同步旧版审核记录。
//
// 旧表写入失败只记录日志，不回滚新版审核主流程。
func (s *ReviewV2Service) syncLegacyReview(
	ctx context.Context,
	planID string,
	reviewerID string,
	req *models.ReviewDecisionV2Request,
	round int,
) {
	legacyReview := &models.LessonPlanReview{
		LessonPlanID: planID,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comments:     req.Comment,
		Dimensions:   req.Dimensions,
		Round:        round,
	}

	if err := repository.CreateLessonPlanReview(
		ctx,
		legacyReview,
	); err != nil {
		reviewLog.Error(
			"同步旧版审核记录失败（不影响主流程）",
			"plan_id",
			planID,
			"error",
			err,
		)
	}
}

// triggerAutoExtractIfEligible 在满足条件时旁路触发组件萃取。
//
// 使用后台上下文，不阻断审核响应。
func (s *ReviewV2Service) triggerAutoExtractIfEligible(
	lessonPlan *models.LessonPlan,
	reviewerID string,
) {
	if lessonPlan == nil ||
		lessonPlan.AIReviewScore == nil ||
		*lessonPlan.AIReviewScore < 8.5 ||
		s.compService == nil {
		return
	}

	planID := lessonPlan.ID
	planContent := lessonPlan.ContentMarkdown
	subject := lessonPlan.Subject
	grade := lessonPlan.Grade
	score := *lessonPlan.AIReviewScore

	go func() {
		backgroundContext := context.Background()

		reviewLog.Info(
			"触发通道二自动萃取",
			"plan_id",
			planID,
			"ai_score",
			score,
		)

		if err := s.compService.AutoExtractFromLessonPlan(
			backgroundContext,
			planID,
			planContent,
			subject,
			grade,
			reviewerID,
		); err != nil {
			reviewLog.Error(
				"通道二自动萃取失败",
				"plan_id",
				planID,
				"error",
				err,
			)
		}
	}()
}
