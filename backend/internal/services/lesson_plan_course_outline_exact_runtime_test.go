package services

// lesson_plan_course_outline_exact_runtime_test.go
//
// 验证唯一课程大纲运行时解析：
//   - 只读取lesson_plans固化的唯一ID、出版社、册次和学制快照；
//   - 可见性复核、active精确读取均显式使用教案education_domain快照；
//   - publisher-only旧兼容状态不再触发模糊候选查询或正文注入。

import (
	"context"
	"testing"

	"tedna/internal/models"
)

func TestResolveCourseOutlinesUsesLessonPlanSnapshotDomain(
	t *testing.T,
) {
	preserveCourseOutlineGuardDependency(t)

	planID := "lesson-plan-adult-1"
	authorID := "teacher-adult-1"
	outlineID := "adult-outline-1"
	publisher := ""
	volume := "第一册"
	schoolSystem :=
		models.CourseOutlineSchoolSystemStandard

	snapshotCalled := false
	visibleCalled := false
	exactCalled := false

	lessonPlanCourseOutlineSnapshotReader = func(
		ctx context.Context,
		lessonPlanID string,
	) (
		*models.LessonPlanCourseOutlineSnapshot,
		error,
	) {
		snapshotCalled = true

		if lessonPlanID != planID {
			t.Fatalf(
				"读取了错误教案快照: %s",
				lessonPlanID,
			)
		}

		return &models.LessonPlanCourseOutlineSnapshot{
			CourseOutlineID:        &outlineID,
			CourseOutlinePublisher: &publisher,
			CourseOutlineVolume:    &volume,
			SchoolSystem:           &schoolSystem,
		}, nil
	}

	visibleCourseOutlineReader = func(
		ctx context.Context,
		userID string,
		requestedOutlineID string,
	) (
		*models.CourseOutline,
		string,
		error,
	) {
		visibleCalled = true

		if userID != authorID {
			t.Fatalf(
				"可见性复核使用了错误作者: %s",
				userID,
			)
		}

		if requestedOutlineID != outlineID {
			t.Fatalf(
				"可见性复核使用了错误大纲ID: %s",
				requestedOutlineID,
			)
		}

		return &models.CourseOutline{
				ID:           outlineID,
				Subject:      "数字技能培训",
				Grade:        "零基础",
				Volume:       volume,
				Publisher:    publisher,
				SchoolSystem: schoolSystem,
				Content:      "成人教育课程大纲",
				Status:       models.CourseOutlineStatusActive,
			},
			models.EducationDomainAdult,
			nil
	}

	activeCourseOutlineExactReader = func(
		ctx context.Context,
		requestedOutlineID string,
		educationDomain string,
	) (
		*models.CourseOutline,
		error,
	) {
		exactCalled = true

		if requestedOutlineID != outlineID {
			t.Fatalf(
				"精确读取使用了错误大纲ID: %s",
				requestedOutlineID,
			)
		}

		if educationDomain !=
			models.EducationDomainAdult {
			t.Fatalf(
				"未使用教案教育域快照: %s",
				educationDomain,
			)
		}

		return &models.CourseOutline{
			ID:           outlineID,
			Subject:      "数字技能培训",
			Grade:        "零基础",
			Volume:       volume,
			Publisher:    publisher,
			SchoolSystem: schoolSystem,
			Content:      "成人教育课程大纲",
			Status:       models.CourseOutlineStatusActive,
		}, nil
	}

	lessonPlan := &models.LessonPlan{
		ID:                     planID,
		AuthorID:               authorID,
		Subject:                "数字技能培训",
		Grade:                  "零基础",
		EducationDomain:        models.EducationDomainAdult,
		CourseOutlineID:        &outlineID,
		CourseOutlinePublisher: &publisher,
		CourseOutlineVolume:    &volume,
		SchoolSystem:           &schoolSystem,
	}

	hits, err :=
		ResolveLessonPlanCourseOutlines(
			context.Background(),
			lessonPlan,
		)
	if err != nil {
		t.Fatalf(
			"运行时课程大纲解析失败: %v",
			err,
		)
	}

	if !snapshotCalled ||
		!visibleCalled ||
		!exactCalled {
		t.Fatalf(
			"运行时精确读取链未完整执行: snapshot=%t visible=%t exact=%t",
			snapshotCalled,
			visibleCalled,
			exactCalled,
		)
	}

	if len(hits) != 1 {
		t.Fatalf(
			"命中数量错误: %d",
			len(hits),
		)
	}

	if hits[0].ID != outlineID {
		t.Fatalf(
			"命中了错误大纲: %s",
			hits[0].ID,
		)
	}
}

func TestResolveCourseOutlinesDoesNotFallbackToPublisherOnly(
	t *testing.T,
) {
	preserveCourseOutlineGuardDependency(t)

	legacyQueryCalled := false
	snapshotCalled := false

	courseOutlineListActiveByDomain = func(
		ctx context.Context,
		subject string,
		educationDomain string,
	) (
		[]*models.CourseOutline,
		error,
	) {
		legacyQueryCalled = true
		return []*models.CourseOutline{
			{
				ID:      "legacy-outline",
				Subject: subject,
				Grade:   "零基础",
			},
		}, nil
	}

	lessonPlanCourseOutlineSnapshotReader = func(
		ctx context.Context,
		lessonPlanID string,
	) (
		*models.LessonPlanCourseOutlineSnapshot,
		error,
	) {
		snapshotCalled = true
		return &models.LessonPlanCourseOutlineSnapshot{},
			nil
	}

	publisher := ""
	lessonPlan := &models.LessonPlan{
		ID:                     "publisher-only-plan",
		AuthorID:               "teacher-adult-1",
		Subject:                "数字技能培训",
		Grade:                  "零基础",
		EducationDomain:        models.EducationDomainAdult,
		CourseOutlinePublisher: &publisher,
	}

	hits, err :=
		ResolveLessonPlanCourseOutlines(
			context.Background(),
			lessonPlan,
		)
	if err != nil {
		t.Fatalf(
			"publisher-only兼容状态不应报错: %v",
			err,
		)
	}

	if !snapshotCalled {
		t.Fatal(
			"运行时应先读取唯一课程大纲快照",
		)
	}

	if legacyQueryCalled {
		t.Fatal(
			"publisher-only兼容状态不得重新启用模糊候选查询",
		)
	}

	if len(hits) != 0 {
		t.Fatalf(
			"publisher-only兼容状态不应返回大纲: %d",
			len(hits),
		)
	}
}
