package services

// lesson_plan_service_courseoutline.go — 教案课程大纲中途挂载服务
//
// 正式精确ID链：
//   - nil或空字符串：解除精确挂载并清理publisher-only旧残留；
//   - 非空UUID：校验作者本人、可编辑状态、实时教育域、读取范围、学科和具体年级；
//   - Repository只写唯一course_outline_id；数据库触发器固化出版社、册次和学制快照。
//
// 旧publisher-only链暂时保留兼容：
//   - nil：解除；
//   - 空字符串：K12通用版或非K12普通大纲；
//   - 具名字符串：K12指定出版社。
// 新前端不得继续调用旧链。

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// loadEditableLessonPlanForCourseOutlineChange 统一校验课程大纲挂载写权限。
//
// 这里只允许作者本人修改可编辑教案；操作者实时教育域必须是具体教学域，
// 且必须与教案创建时固化的education_domain完全一致。
func (s *LessonPlanService) loadEditableLessonPlanForCourseOutlineChange(
	ctx context.Context,
	id string,
	callerID string,
) (*models.LessonPlan, *courseOutlineActor, error) {
	lessonPlan, err := repository.GetLessonPlanByID(ctx, id)
	if err != nil {
		return nil, nil, s.mapNotFoundErr(err)
	}

	if lessonPlan.AuthorID != callerID {
		return nil, nil, ErrLPNotAuthor
	}

	editableStatuses := map[string]bool{
		models.LPStatusDraft:             true,
		models.LPStatusPublishedPersonal: true,
		models.LPStatusRevision:          true,
		models.LPStatusApproved:          true,
		models.LPStatusPublishedShared:   true,
	}
	if !editableStatuses[lessonPlan.Status] {
		return nil, nil, ErrLPCannotEdit
	}

	actor, err := resolveCourseOutlineActor(ctx, callerID)
	if err != nil {
		return nil, nil, err
	}

	// mixed管理身份只有课程大纲基础数据管理能力，不能修改普通教学教案挂载。
	if actor.MixedManagement {
		return nil, nil, ErrOutlineEducationDomainRequired
	}

	snapshotDomain, err := resolveLessonPlanCourseOutlineSnapshotDomain(lessonPlan)
	if err != nil {
		return nil, nil, err
	}

	liveDomain := strings.ToLower(strings.TrimSpace(actor.EducationDomain))
	if !models.IsTeachingEducationDomain(liveDomain) {
		return nil, nil, ErrOutlineEducationDomainRequired
	}
	if liveDomain != snapshotDomain {
		return nil, nil, ErrOutlineEducationDomainMismatch
	}

	return lessonPlan, actor, nil
}

// validateUpdatedLessonPlanCourseOutlineSnapshot 复核数据库写入后的最终快照。
func validateUpdatedLessonPlanCourseOutlineSnapshot(
	requestedID *string,
	snapshot *models.LessonPlanCourseOutlineSnapshot,
) error {
	if snapshot == nil {
		return ErrOutlineExactSelectionInvalid
	}

	if requestedID == nil {
		if snapshot.CourseOutlineID != nil ||
			snapshot.CourseOutlinePublisher != nil ||
			snapshot.CourseOutlineVolume != nil ||
			snapshot.SchoolSystem != nil {
			return ErrOutlineExactSelectionInvalid
		}
		return nil
	}

	normalizedID := strings.TrimSpace(*requestedID)
	if normalizedID == "" ||
		snapshot.CourseOutlineID == nil ||
		strings.TrimSpace(*snapshot.CourseOutlineID) != normalizedID ||
		snapshot.CourseOutlinePublisher == nil ||
		snapshot.CourseOutlineVolume == nil ||
		strings.TrimSpace(*snapshot.CourseOutlineVolume) == "" ||
		snapshot.SchoolSystem == nil ||
		!models.IsValidCourseOutlineSchoolSystem(strings.TrimSpace(*snapshot.SchoolSystem)) {
		return ErrOutlineExactSelectionInvalid
	}

	return nil
}

// UpdateLessonPlanCourseOutline 设置、更换或解除唯一精确课程大纲。
func (s *LessonPlanService) UpdateLessonPlanCourseOutline(
	ctx context.Context,
	id string,
	callerID string,
	outlineID *string,
) (*models.LessonPlanCourseOutlineSnapshot, error) {
	lessonPlan, _, err := s.loadEditableLessonPlanForCourseOutlineChange(ctx, id, callerID)
	if err != nil {
		return nil, err
	}

	var normalizedID *string
	if outlineID != nil {
		value := strings.TrimSpace(*outlineID)
		if value != "" {
			if _, parseErr := uuid.Parse(value); parseErr != nil {
				return nil, ErrOutlineExactSelectionInvalid
			}

			req := &models.StartConversationRequest{
				Subject:         lessonPlan.Subject,
				Grade:           lessonPlan.Grade,
				CourseOutlineID: value,
			}
			if err := ValidateStartConversationCourseOutline(
				ctx,
				lessonPlan.EducationDomain,
				callerID,
				req,
			); err != nil {
				return nil, err
			}

			value = strings.TrimSpace(req.CourseOutlineID)
			normalizedID = &value
		}
	}

	snapshot, err := repository.UpdateLessonPlanCourseOutlineID(
		ctx,
		id,
		normalizedID,
	)
	if err != nil {
		lpLog.Error(
			"更新教案精确课程大纲关联失败",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
			"error", err,
		)
		return nil, s.mapNotFoundErr(err)
	}

	if err := validateUpdatedLessonPlanCourseOutlineSnapshot(normalizedID, snapshot); err != nil {
		lpLog.Error(
			"教案精确课程大纲写入后快照不完整",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
			"error", err,
		)
		return nil, err
	}

	if normalizedID == nil {
		lpLog.Info(
			"教案已解除精确课程大纲关联",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
		)
	} else {
		lpLog.Info(
			"教案已更新精确课程大纲关联",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
			"course_outline_id", *normalizedID,
		)
	}

	return snapshot, nil
}

// UpdateLessonPlanCourseOutlinePublisher 设置或解除publisher-only旧挂载。
//
// @deprecated 新前端必须调用UpdateLessonPlanCourseOutline并提交唯一ID。
func (s *LessonPlanService) UpdateLessonPlanCourseOutlinePublisher(
	ctx context.Context,
	id string,
	callerID string,
	publisher *string,
) error {
	lessonPlan, actor, err := s.loadEditableLessonPlanForCourseOutlineChange(ctx, id, callerID)
	if err != nil {
		return err
	}

	normalizedPublisher, err := normalizeLessonPlanCourseOutlineMount(
		ctx,
		lessonPlan,
		actor.EducationDomain,
		publisher,
	)
	if err != nil {
		return err
	}

	if err := repository.UpdateLessonPlanCourseOutlinePublisher(
		ctx,
		id,
		normalizedPublisher,
	); err != nil {
		lpLog.Error(
			"更新教案课程大纲旧挂载失败",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
			"error", err,
		)
		return err
	}

	if normalizedPublisher == nil {
		lpLog.Info(
			"教案已解除课程大纲旧关联",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
		)
	} else {
		lpLog.Info(
			"教案已更新课程大纲旧挂载",
			"plan_id", id,
			"caller", callerID,
			"education_domain", lessonPlan.EducationDomain,
			"has_named_publisher", *normalizedPublisher != "",
		)
	}

	return nil
}
