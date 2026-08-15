package services

// courseware_review_history_service.go
//
// R-03“已审核记录只读详情”的主业务编排。
//
// 本文件只负责：
//   - review_id读取真实人工审核记录；
//   - review -> courseware真实关系复核；
//   - 当前可信Actor、教育域和组织范围授权；
//   - 审核教师展示身份；
//   - feedback与review关系校验；
//   - 调用配置、整改项和页面三个只读历史子模块完成响应。
//
// 具体职责拆分：
//   - R-02不可变配置：courseware_review_history_config.go；
//   - 正式交付问题：courseware_review_history_issue.go；
//   - 审核时/当前页面：courseware_review_history_pages.go。
//
// 不存在与无权访问统一收敛为404语义，避免review_id跨校枚举。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ErrCWReviewHistoryDetailNotFound 同时承载不存在和无权访问的公开语义。
var ErrCWReviewHistoryDetailNotFound = errors.New("审核记录不存在")

// GetReviewHistoryDetail 获取一条已审核记录的完整只读历史详情。
func (s *CoursewareReviewService) GetReviewHistoryDetail(
	ctx context.Context,
	reviewID string,
	actor *CoursewareActorContext,
) (*models.CoursewareReviewHistoryDetail, error) {
	reviewID = strings.TrimSpace(reviewID)

	if !isCWReviewHistoryUUID(reviewID) ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWReviewHistoryDetailNotFound
	}

	review, err :=
		repository.GetCoursewareReviewHistoryRecordByID(
			ctx,
			reviewID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrCoursewareReviewHistoryNotFound,
		) {
			return nil, ErrCWReviewHistoryDetailNotFound
		}

		return nil, err
	}

	if review == nil ||
		(review.ReviewLevel != models.ReviewLevelL1 &&
			review.ReviewLevel != models.ReviewLevelL2) {
		return nil, errors.New(
			"课件审核历史记录层级异常",
		)
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			review.CoursewareID,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCWReviewHistoryDetailNotFound
		}

		return nil, fmt.Errorf(
			"读取课件审核历史关联课件失败: %w",
			err,
		)
	}

	if courseware == nil ||
		strings.TrimSpace(courseware.ID) !=
			strings.TrimSpace(review.CoursewareID) {
		return nil, ErrCWReviewHistoryDetailNotFound
	}

	allowed, accessErr :=
		s.canViewCWReviewHistoryRecord(
			ctx,
			courseware,
			review,
			actor,
		)
	if accessErr != nil {
		switch {
		case errors.Is(
			accessErr,
			ErrCoursewareEducationDomainMismatch,
		),
			errors.Is(
				accessErr,
				ErrCoursewareActorRequired,
			),
			errors.Is(
				accessErr,
				ErrCoursewareEducationDomainInvalid,
			),
			errors.Is(
				accessErr,
				ErrCoursewareRuntimeDomainRequired,
			):
			// 资源存在但当前Actor不满足历史读取域边界时，
			// 与不存在统一返回，防止review_id探测。
			return nil, ErrCWReviewHistoryDetailNotFound

		default:
			return nil, accessErr
		}
	}

	if !allowed {
		return nil, ErrCWReviewHistoryDetailNotFound
	}

	reviewer, err :=
		buildCWReviewHistoryReviewer(
			ctx,
			review,
		)
	if err != nil {
		return nil, err
	}

	result :=
		&models.CoursewareReviewHistoryDetail{
			ReviewID: review.ID,

			RecordTitle: fmt.Sprintf(
				"L%d审核记录 · 第%d轮",
				review.ReviewLevel,
				review.ReviewRound,
			),

			Courseware: models.CoursewareReviewHistoryCourseware{
				ID:      courseware.ID,
				Title:   courseware.Title,
				Subject: courseware.Subject,
				Grade:   courseware.Grade,
			},

			Reviewer: reviewer,

			Review: models.CoursewareReviewHistoryDecision{
				ReviewLevel: review.ReviewLevel,
				ReviewRound: review.ReviewRound,
				Decision:    review.Decision,
				Score:       review.Score,
				Comment:     review.Comment,
				ReviewedAt:  review.CreatedAt,
			},

			ReviewConfig: emptyCWReviewHistoryConfig(
				models.CWReviewHistoryConfigUnavailableNoAI,
			),

			IssuesAvailable:         false,
			IssuesUnavailableReason: models.CWReviewHistoryIssuesUnavailableLegacy,
			Issues:                  []models.CoursewareReviewHistoryIssue{},

			HistoricalPages: []models.CoursewareReviewHistoryPage{},
			CurrentPages:    []models.CoursewareReviewHistoryCurrentPage{},
		}

	feedback, feedbackErr :=
		repository.GetCoursewareReviewFeedbackByReviewID(
			ctx,
			review.ID,
		)

	switch {
	case feedbackErr == nil:
		if err :=
			validateCWReviewHistoryFeedback(
				review,
				feedback,
			); err != nil {
			return nil, err
		}

		result.IssuesAvailable = true
		result.IssuesUnavailableReason = ""

		config, err :=
			buildCWReviewHistoryConfig(
				ctx,
				review,
				feedback,
			)
		if err != nil {
			return nil, err
		}
		result.ReviewConfig = config

		issues, err :=
			buildCWReviewHistoryIssues(
				ctx,
				courseware,
				review,
				feedback,
			)
		if err != nil {
			return nil, err
		}
		result.Issues = issues

	case errors.Is(
		feedbackErr,
		repository.ErrCoursewareReviewFeedbackNotFound,
	):
		// 旧审核没有feedback时，问题历史和完整R-02配置均不可证明。
		// 不能使用空数组或当前配置冒充历史事实。
		result.ReviewConfig =
			emptyCWReviewHistoryConfig(
				models.CWReviewHistoryConfigUnavailableLegacy,
			)

	default:
		return nil, feedbackErr
	}

	historicalPages,
		currentPages,
		historicalAvailable,
		historicalUnavailableReason,
		err :=
		buildCWReviewHistoryPages(
			ctx,
			review,
		)
	if err != nil {
		return nil, err
	}

	result.HistoricalPages = historicalPages
	result.CurrentPages = currentPages
	result.HistoricalPagesAvailable = historicalAvailable
	result.HistoricalPagesUnavailableReason =
		historicalUnavailableReason

	return result, nil
}

// canViewCWReviewHistoryRecord 按历史审核层级裁决当前Actor是否仍具有查看范围。
//
// 不能使用courseware当前review_level判断历史L1/L2权限，
// 否则课件后来退回、通过或再次提交会改变旧记录的访问语义。
func (s *CoursewareReviewService) canViewCWReviewHistoryRecord(
	ctx context.Context,
	courseware *models.Courseware,
	review *models.CoursewareReview,
	actor *CoursewareActorContext,
) (bool, error) {
	if courseware == nil ||
		review == nil ||
		actor == nil {
		return false, ErrCoursewareActorRequired
	}

	// 作者继续复用既有历史策略：
	// 作者换校后仍可查看自己的合法审核历史。
	if courseware.UserID == actor.UserID {
		return s.CanViewLoadedCoursewareReviewHistory(
			ctx,
			courseware,
			actor,
		)
	}

	if err :=
		ValidateCoursewareReviewEducationDomain(
			actor,
			courseware,
		); err != nil {
		return false, err
	}

	if actor.Role == models.RoleAdmin {
		return true, nil
	}

	if actor.Role == models.RoleRegionAdmin {
		reviewSchoolID :=
			s.resolveReviewSchoolID(
				ctx,
				courseware,
			)
		if strings.TrimSpace(reviewSchoolID) == "" {
			return false, nil
		}

		scope :=
			ResolveDataScope(
				ctx,
				actor.Role,
				actor.UserID,
			)
		if scope.Blocked {
			return false, nil
		}

		for _, schoolID := range scope.SchoolIDs {
			if schoolID == reviewSchoolID {
				return true, nil
			}
		}

		return false, nil
	}

	if actor.Role == models.RoleSeniorOperator &&
		s.isSeniorOfReviewSchool(
			ctx,
			courseware,
			actor.UserID,
		) {
		return true, nil
	}

	// lead/backbone是L1审核范围，不因当前课件后来进入L2或其它状态
	// 获得历史L2记录的额外访问权。
	if review.ReviewLevel == models.ReviewLevelL1 {
		return s.isReviewerInAuthorGroupAsLeadOrBackbone(
			ctx,
			courseware.UserID,
			actor.UserID,
		)
	}

	return false, nil
}

func buildCWReviewHistoryReviewer(
	ctx context.Context,
	review *models.CoursewareReview,
) (
	models.CoursewareReviewHistoryReviewer,
	error,
) {
	result :=
		models.CoursewareReviewHistoryReviewer{
			ID: strings.TrimSpace(
				review.ReviewerID,
			),
			DisplayName: "原审核教师账号不可用",
		}

	if result.ID == "" {
		return result, errors.New(
			"课件审核历史缺少审核教师身份",
		)
	}

	user, err :=
		repository.FindUserByID(
			ctx,
			result.ID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserNotFound,
		) {
			return result, nil
		}

		return result, fmt.Errorf(
			"读取历史审核教师身份失败: %w",
			err,
		)
	}

	if user == nil {
		return result, nil
	}

	displayName :=
		strings.TrimSpace(
			user.DisplayName,
		)
	if displayName == "" {
		displayName =
			strings.TrimSpace(
				user.Username,
			)
	}

	if displayName != "" {
		result.DisplayName = displayName
	}

	return result, nil
}

func validateCWReviewHistoryFeedback(
	review *models.CoursewareReview,
	feedback *models.CoursewareReviewFeedback,
) error {
	if review == nil ||
		feedback == nil {
		return errors.New(
			"课件审核历史反馈关系不完整",
		)
	}

	if strings.TrimSpace(
		feedback.CoursewareReviewID,
	) != strings.TrimSpace(review.ID) ||
		strings.TrimSpace(
			feedback.CoursewareID,
		) != strings.TrimSpace(review.CoursewareID) ||
		feedback.ReviewLevel != review.ReviewLevel ||
		feedback.ReviewRound != review.ReviewRound ||
		strings.TrimSpace(
			feedback.Decision,
		) != strings.TrimSpace(review.Decision) {
		return errors.New(
			"课件审核历史反馈关系异常",
		)
	}

	return nil
}

// isCWReviewHistoryUUID 在进入UUID数据库比较前做轻量格式校验。
func isCWReviewHistoryUUID(
	value string,
) bool {
	if len(value) != 36 {
		return false
	}

	for index, r := range value {
		switch index {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}

		default:
			if !((r >= '0' && r <= '9') ||
				(r >= 'a' && r <= 'f') ||
				(r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}

	return true
}
