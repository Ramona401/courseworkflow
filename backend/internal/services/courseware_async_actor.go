package services

// courseware_async_actor.go — 课件异步任务Actor快照与方案链授权底座
//
// AssistantActorContext包含三个字符串切片。直接进行结构体浅复制会继续
// 共享这些切片的底层数组，因此异步任务统一使用CloneCoursewareActorContext
// 创建独立、只读、已经收敛到课件历史教育域的身份快照。
//
// 五条课件方案链实行两层授权：
//
//	Handler作者域预检
//	→ 独立Actor快照
//	→ 后台任务登记
//	→ Service重新加载正式课件
//	→ 作者、历史教育域和审核锁二次校验
//	→ AI与写库

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
)

// CloneCoursewareActorContext 深复制课件可信Actor。
func CloneCoursewareActorContext(
	actor *CoursewareActorContext,
) *CoursewareActorContext {
	if actor == nil {
		return nil
	}

	cloned := *actor

	cloned.MyGroupIDs = append(
		[]string(nil),
		actor.MyGroupIDs...,
	)
	cloned.MyLeadGroupIDs = append(
		[]string(nil),
		actor.MyLeadGroupIDs...,
	)
	cloned.MyLeadOrBackboneGroupIDs = append(
		[]string(nil),
		actor.MyLeadOrBackboneGroupIDs...,
	)

	return &cloned
}

// LoadCoursewareForOwnerControlMutation 暴露B17-B1统一作者控制面加载能力。
func (s *CoursewareService) LoadCoursewareForOwnerControlMutation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	return s.loadOwnedCoursewareForControlMutation(
		ctx,
		coursewareID,
		actor,
	)
}

// loadOwnedCoursewareForSchemeMutation 是五条方案Service统一二次授权入口。
func loadOwnedCoursewareForSchemeMutation(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	return (&CoursewareService{}).
		LoadCoursewareForOwnerControlMutation(
			ctx,
			coursewareID,
			actor,
		)
}

// validateCoursewareLinkedLessonPlanDomain 校验教案关联路径和教育域。
//
// 课件的lesson_plan_id必须与实际加载教案完全一致；
// 教案与课件必须属于相同的具体教学教育域。
func validateCoursewareLinkedLessonPlanDomain(
	courseware *models.Courseware,
	lessonPlan *models.LessonPlan,
) error {
	if courseware == nil ||
		lessonPlan == nil ||
		courseware.LessonPlanID == nil {
		return ErrCoursewareLessonPlanDomainInvalid
	}

	linkedLessonPlanID := strings.TrimSpace(
		*courseware.LessonPlanID,
	)
	if linkedLessonPlanID == "" ||
		strings.TrimSpace(lessonPlan.ID) == "" ||
		linkedLessonPlanID != strings.TrimSpace(
			lessonPlan.ID,
		) {
		return fmt.Errorf(
			"%w: 课件与教案关联路径不一致",
			ErrCoursewareLessonPlanDomainInvalid,
		)
	}

	coursewareDomain := strings.ToLower(
		strings.TrimSpace(
			courseware.EducationDomain,
		),
	)
	lessonPlanDomain := strings.ToLower(
		strings.TrimSpace(
			lessonPlan.EducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		coursewareDomain,
	) ||
		!models.IsTeachingEducationDomain(
			lessonPlanDomain,
		) ||
		coursewareDomain != lessonPlanDomain {
		return fmt.Errorf(
			"%w: 课件与关联教案教育域不一致",
			ErrCoursewareLessonPlanDomainInvalid,
		)
	}

	return nil
}
