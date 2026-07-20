package services

// lesson_plan_service_courseoutline.go — 教案课程大纲挂载服务
//
// PUT挂载字段三态：
//   nil       = 解除关联；
//   ""        = K12通用版，或非K12普通课程大纲；
//   "人教版" = K12具名教材版本。
//
// 上下文16硬闸：
//   1. 先读取教案并校验作者和可编辑状态；
//   2. 再从数据库实时读取操作者角色与具体教学域；
//   3. mixed管理身份不能作为普通教学作者执行挂载；
//   4. 操作者实时域必须与lesson_plans.education_domain快照一致；
//   5. 非K12只能写nil或空字符串；
//   6. K12版本必须在当前学科和年级下真实存在；
//   7. 最终由数据库CHECK约束再做纵深防御。

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UpdateLessonPlanCourseOutlinePublisher
// 设置或解除教案课程大纲挂载。
func (s *LessonPlanService) UpdateLessonPlanCourseOutlinePublisher(
	ctx context.Context,
	id string,
	callerID string,
	publisher *string,
) error {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			id,
		)
	if err != nil {
		return s.mapNotFoundErr(err)
	}

	if lessonPlan.AuthorID != callerID {
		return ErrLPNotAuthor
	}

	editableStatuses := map[string]bool{
		models.LPStatusDraft:             true,
		models.LPStatusPublishedPersonal: true,
		models.LPStatusRevision:          true,
		models.LPStatusApproved:          true,
		models.LPStatusPublishedShared:   true,
	}
	if !editableStatuses[lessonPlan.Status] {
		return ErrLPCannotEdit
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		callerID,
	)
	if err != nil {
		return err
	}

	// admin等mixed管理身份只有课程大纲基础数据管理兼容域，
	// 不能借该兼容域给普通教案挂载K12出版社。
	if actor.MixedManagement {
		return ErrOutlineEducationDomainRequired
	}

	normalizedPublisher, err :=
		normalizeLessonPlanCourseOutlineMount(
			ctx,
			lessonPlan,
			actor.EducationDomain,
			publisher,
		)
	if err != nil {
		return err
	}

	if err := repository.
		UpdateLessonPlanCourseOutlinePublisher(
			ctx,
			id,
			normalizedPublisher,
		); err != nil {
		lpLog.Error(
			"更新教案课程大纲挂载失败",
			"plan_id", id,
			"caller", callerID,
			"education_domain",
			lessonPlan.EducationDomain,
			"error", err,
		)
		return err
	}

	if normalizedPublisher == nil {
		lpLog.Info(
			"教案已解除课程大纲关联",
			"plan_id", id,
			"caller", callerID,
			"education_domain",
			lessonPlan.EducationDomain,
		)
	} else {
		lpLog.Info(
			"教案已更新课程大纲挂载",
			"plan_id", id,
			"caller", callerID,
			"education_domain",
			lessonPlan.EducationDomain,
			"has_named_publisher",
			*normalizedPublisher != "",
		)
	}

	return nil
}
