package services

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestCloneCoursewareActorContext(
	t *testing.T,
) {
	t.Run("nil保持nil", func(t *testing.T) {
		if CloneCoursewareActorContext(nil) != nil {
			t.Fatal("nil Actor克隆结果应为nil")
		}
	})

	t.Run("三个组织切片独立复制", func(t *testing.T) {
		original := &CoursewareActorContext{
			UserID:          "owner-1",
			Role:            "teacher",
			SchoolID:        "school-current",
			EducationDomain: models.EducationDomainK12,
			MyGroupIDs: []string{
				"group-member",
			},
			MyLeadGroupIDs: []string{
				"group-lead",
			},
			MyLeadOrBackboneGroupIDs: []string{
				"group-backbone",
			},
		}

		cloned :=
			CloneCoursewareActorContext(
				original,
			)

		if cloned == nil {
			t.Fatal("克隆结果不应为空")
		}
		if cloned == original {
			t.Fatal("克隆结果不能复用原Actor指针")
		}

		cloned.MyGroupIDs[0] = "changed-member"
		cloned.MyLeadGroupIDs[0] = "changed-lead"
		cloned.MyLeadOrBackboneGroupIDs[0] =
			"changed-backbone"

		if original.MyGroupIDs[0] != "group-member" {
			t.Fatal("MyGroupIDs仍共享底层数组")
		}
		if original.MyLeadGroupIDs[0] != "group-lead" {
			t.Fatal("MyLeadGroupIDs仍共享底层数组")
		}
		if original.MyLeadOrBackboneGroupIDs[0] !=
			"group-backbone" {
			t.Fatal(
				"MyLeadOrBackboneGroupIDs仍共享底层数组",
			)
		}
	})
}

func TestValidateCoursewareLinkedLessonPlanDomain(
	t *testing.T,
) {
	lessonPlanID := "lesson-plan-1"

	courseware := &models.Courseware{
		ID:              "courseware-1",
		LessonPlanID:    &lessonPlanID,
		EducationDomain: models.EducationDomainK12,
	}
	lessonPlan := &models.LessonPlan{
		ID:              lessonPlanID,
		EducationDomain: models.EducationDomainK12,
	}

	t.Run("同路径同教育域通过", func(t *testing.T) {
		if err :=
			validateCoursewareLinkedLessonPlanDomain(
				courseware,
				lessonPlan,
			); err != nil {
			t.Fatalf("不期望错误：%v", err)
		}
	})

	t.Run("教育域不同拒绝", func(t *testing.T) {
		changed := *lessonPlan
		changed.EducationDomain =
			models.EducationDomainVocational

		err :=
			validateCoursewareLinkedLessonPlanDomain(
				courseware,
				&changed,
			)

		if !errors.Is(
			err,
			ErrCoursewareLessonPlanDomainInvalid,
		) {
			t.Fatalf(
				"期望教案教育域错误，实际：%v",
				err,
			)
		}
	})

	t.Run("关联ID不同拒绝", func(t *testing.T) {
		changed := *lessonPlan
		changed.ID = "lesson-plan-other"

		err :=
			validateCoursewareLinkedLessonPlanDomain(
				courseware,
				&changed,
			)

		if !errors.Is(
			err,
			ErrCoursewareLessonPlanDomainInvalid,
		) {
			t.Fatalf(
				"期望教案关联错误，实际：%v",
				err,
			)
		}
	})
}
