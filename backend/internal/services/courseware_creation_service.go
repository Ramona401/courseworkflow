package services

// courseware_creation_service.go — 从教案创建课件
//
// 本文件从courseware_service.go拆出，确保核心服务文件保持600行以内。
//
// 创建顺序：
//   1. 校验可信Actor与教案ID；
//   2. 读取教案数据库真值；
//   3. 按教案作者、管理身份和教育域快照执行创建授权；
//   4. 使用教案标题、学科、年级和教育域构造课件；
//   5. 写入coursewares。
//
// superadmin例外只发生在第3步：数据库实时身份仍为admin+is_super=true，
// 且教案作者就是本人时，可以从自己的历史具体域教案创建课件。
// 客户端不能提交education_domain，普通admin和其它mixed角色仍被拒绝。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// 从教案创建课件的稳定错误，供Handler准确映射HTTP状态码。
var (
	ErrCoursewareLessonPlanRequired = errors.New(
		"教案ID不能为空",
	)
	ErrCoursewareLessonPlanNotFound = errors.New(
		"关联教案不存在",
	)
)

// CreateCourseware 创建课件（从教案出发）。
//
// 教案是标题、学科、年级与教育域快照的唯一事实源。
// 请求中的title只允许覆盖展示标题，不能改变教学属性或资源域。
func (s *CoursewareService) CreateCourseware(
	ctx context.Context,
	actor *CoursewareActorContext,
	req *models.CreateCoursewareRequest,
) (*models.Courseware, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCoursewareActorRequired
	}
	if req == nil ||
		strings.TrimSpace(req.LessonPlanID) == "" {
		return nil, ErrCoursewareLessonPlanRequired
	}

	lessonPlanID := strings.TrimSpace(
		req.LessonPlanID,
	)

	// 教案是课件教育域快照的唯一来源。
	// 不得按创建者当前学校重新推导，否则换校会使历史教案静默重分类。
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil,
				ErrCoursewareLessonPlanNotFound
		}
		return nil, fmt.Errorf(
			"查询关联教案失败: %w",
			err,
		)
	}

	domain, err :=
		ResolveCoursewareEducationDomainFromLessonPlanForCreate(
			ctx,
			actor,
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = lessonPlan.Title
	}

	courseware := &models.Courseware{
		LessonPlanID:    &lessonPlanID,
		UserID:          actor.UserID,
		Title:           title,
		Subject:         lessonPlan.Subject,
		Grade:           lessonPlan.Grade,
		EducationDomain: domain,
		Status:          models.CoursewareStatusDraft,
		SourceType:      models.CWSourceLessonPlan,
		PageCount:       0,
	}

	if err := repository.CreateCourseware(
		ctx,
		courseware,
	); err != nil {
		return nil, fmt.Errorf(
			"创建课件失败: %w",
			err,
		)
	}

	return courseware, nil
}
