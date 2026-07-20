package services

// lesson_plan_course_outline_guard.go — 教案课程大纲挂载与运行时统一硬闸
//
// 本文件是教案课程大纲正式运行链的唯一教育域入口。
//
// 统一规则：
//   1. 运行时只信任lesson_plans.education_domain正式快照；
//   2. 快照只允许k12、vocational、adult三个具体教学域；
//   3. 查询课程大纲时必须显式携带教案快照域；
//   4. K12允许空出版社或具名出版社；
//   5. vocational/adult只允许空字符串，表示“普通课程大纲挂载”；
//   6. 非K12空字符串不再解释或展示为“通用教材版本”；
//   7. 非K12具名出版社直接拒绝；
//   8. 挂载端点还必须验证操作者实时域与教案快照域完全一致；
//   9. K12挂载的版本必须在当前学科和年级下真实存在；
//  10. 数据库查询失败必须向上传递，不能静默伪装成没有大纲。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// courseOutlineListActiveByDomain 默认指向正式分域Repository。
//
// 测试可临时替换，覆盖挂载和运行时纯编排规则。
var courseOutlineListActiveByDomain =
	repository.ListActiveOutlinesBySubjectAndEducationDomain

// resolveLessonPlanCourseOutlineSnapshotDomain
// 严格读取教案正式教育域快照。
func resolveLessonPlanCourseOutlineSnapshotDomain(
	lessonPlan *models.LessonPlan,
) (string, error) {
	if lessonPlan == nil {
		return "", ErrOutlineEducationDomainRequired
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			lessonPlan.EducationDomain,
		),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrOutlineEducationDomainRequired
	}

	return domain, nil
}

// normalizeLessonPlanCourseOutlineMount
// 校验挂载请求并返回可安全落库的三态出版社指针。
//
// 返回值：
//   nil      = 解除课程大纲挂载；
//   &""      = K12通用版本，或非K12普通课程大纲挂载；
//   &"具名"  = K12具名教材版本。
func normalizeLessonPlanCourseOutlineMount(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	liveEducationDomain string,
	publisher *string,
) (*string, error) {
	snapshotDomain, err :=
		resolveLessonPlanCourseOutlineSnapshotDomain(
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	liveDomain := strings.ToLower(
		strings.TrimSpace(
			liveEducationDomain,
		),
	)
	if !models.IsTeachingEducationDomain(
		liveDomain,
	) {
		return nil, ErrOutlineEducationDomainRequired
	}

	if liveDomain != snapshotDomain {
		return nil, ErrOutlineEducationDomainMismatch
	}

	if publisher == nil {
		return nil, nil
	}

	normalized, err :=
		normalizeCourseOutlinePublisherForDomain(
			snapshotDomain,
			*publisher,
		)
	if err != nil {
		return nil, err
	}

	// 非K12空串是普通课程大纲挂载标记。
	//
	// 即使当前尚无匹配大纲，也允许先保存该标记；
	// 后续管理员录入同域普通大纲后，下一轮运行时即可自然命中。
	if snapshotDomain != models.EducationDomainK12 {
		return &normalized, nil
	}

	// K12具名或通用版本必须真实存在。
	//
	// 这道校验阻止直接调用API伪造一个不存在的版本字符串。
	candidates, err :=
		courseOutlineListActiveByDomain(
			ctx,
			lessonPlan.Subject,
			snapshotDomain,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 查询K12课程大纲候选失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	hits := MatchOutlinesByPublisher(
		lessonPlan.Grade,
		normalized,
		candidates,
	)
	if len(hits) == 0 {
		return nil, ErrOutlinePublisherUnavailable
	}

	return &normalized, nil
}

// ResolveLessonPlanCourseOutlines
// 按教案正式快照域、学科、学习层级和出版社选择解析运行时大纲。
//
// 没有挂载或没有匹配项返回空切片，不属于错误。
// 教育域非法、非K12具名出版社、数据库失败属于错误。
func ResolveLessonPlanCourseOutlines(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) ([]*models.CourseOutline, error) {
	snapshotDomain, err :=
		resolveLessonPlanCourseOutlineSnapshotDomain(
			lessonPlan,
		)
	if err != nil {
		return nil, err
	}

	if lessonPlan.CourseOutlinePublisher == nil {
		return []*models.CourseOutline{}, nil
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			snapshotDomain,
			*lessonPlan.CourseOutlinePublisher,
		)
	if err != nil {
		return nil, err
	}

	candidates, err :=
		courseOutlineListActiveByDomain(
			ctx,
			strings.TrimSpace(
				lessonPlan.Subject,
			),
			snapshotDomain,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 查询课程大纲候选失败: %v",
			ErrOutlineEducationDomainResolveFailed,
			err,
		)
	}

	hits := MatchOutlinesByPublisher(
		lessonPlan.Grade,
		publisher,
		candidates,
	)
	if hits == nil {
		hits = []*models.CourseOutline{}
	}

	return hits, nil
}

// BuildLessonPlanCourseOutlineContext
// 构建教案运行时课程大纲上下文，并同时返回命中大纲供日志和回执使用。
func BuildLessonPlanCourseOutlineContext(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (
	string,
	[]*models.CourseOutline,
	error,
) {
	hits, err := ResolveLessonPlanCourseOutlines(
		ctx,
		lessonPlan,
	)
	if err != nil {
		return "", nil, err
	}

	if len(hits) == 0 {
		return "", hits, nil
	}

	return BuildCourseOutlinesContext(hits),
		hits,
		nil
}
