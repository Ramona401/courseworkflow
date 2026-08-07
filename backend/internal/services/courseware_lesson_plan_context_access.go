package services

// courseware_lesson_plan_context_access.go
//
// 课件来源教案/上传文档原文的只读对照授权。
//
// 该授权只服务于 GET /api/v1/coursewares/{id}/lesson-plan-content：
//   - 保留既有普通课件查看权；
//   - 额外允许已经通过课件审核详情访问边界的审核侧用户读取来源材料；
//   - 不修改 LoadCoursewareForView，避免把审核权限扩散到普通课件详情、背景、字体、资产等接口；
//   - 不授予任何编辑、作者控制、微调或产生外部成本的权限。
//
// 审核路径完全复用 CoursewareReviewService.CanReviewLoadedCourseware，因此教育域、学校、
// 教研组、区域只读范围等规则仍以正式审核服务为唯一事实源，不在此复制角色判断。

import (
	"context"
	"fmt"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// LoadCoursewareForLessonPlanContext 按ID加载课件并裁决来源材料只读权限。
//
// 两条权限路径是“或”关系，但每条路径本身都继续执行各自完整的教育域与组织权限校验。
// 任一路径明确授权即可读取；两条路径都未授权时 fail-closed。
func (s *CoursewareService) LoadCoursewareForLessonPlanContext(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.Courseware, error) {
	courseware, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCoursewareAccessNotFound, err)
	}

	normalViewAllowed, normalViewErr := s.CanViewLoadedCourseware(
		ctx,
		courseware,
		actor,
	)
	if normalViewAllowed {
		return courseware, nil
	}

	reviewViewAllowed, reviewViewErr := NewCoursewareReviewService().
		CanReviewLoadedCourseware(
			ctx,
			courseware,
			actor,
		)
	if reviewViewAllowed {
		return courseware, nil
	}

	if err := resolveCoursewareLessonPlanContextAccessError(
		normalViewErr,
		reviewViewErr,
	); err != nil {
		return nil, err
	}

	return nil, ErrCoursewareViewDenied
}

// resolveCoursewareLessonPlanContextAccessError 统一处理两条授权路径都未放行时的错误。
//
// 审核路径的教育域错误优先，因为它代表本次新增审核读取通道自己的明确拒绝；
// 若审核路径只是“无权限”而普通查看路径发生真实内部错误，则保留普通路径错误，
// 避免把数据库故障错误伪装成403。
func resolveCoursewareLessonPlanContextAccessError(
	normalViewErr error,
	reviewViewErr error,
) error {
	if reviewViewErr != nil {
		return reviewViewErr
	}
	if normalViewErr != nil {
		return normalViewErr
	}
	return nil
}
