package services

// lesson_plan_course_outline_guard_test.go
//
// 验证课程大纲挂载与运行时教育域硬闸：
//   - 操作者实时域必须与教案快照域一致；
//   - vocational/adult拒绝具名出版社；
//   - 非K12允许以空字符串挂载普通课程大纲；
//   - K12伪造不存在的出版社被拒绝；
//   - publisher-only旧兼容入口只负责挂载校验，不再参与运行时大纲解析；
//   - 非K12自由学习层级采用完整文本精确匹配。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
)

func preserveCourseOutlineGuardDependency(
	t *testing.T,
) {
	originalListReader :=
		courseOutlineListActiveByDomain
	originalSnapshotReader :=
		lessonPlanCourseOutlineSnapshotReader
	originalExactReader :=
		activeCourseOutlineExactReader
	originalVisibleReader :=
		visibleCourseOutlineReader

	t.Cleanup(func() {
		courseOutlineListActiveByDomain =
			originalListReader
		lessonPlanCourseOutlineSnapshotReader =
			originalSnapshotReader
		activeCourseOutlineExactReader =
			originalExactReader
		visibleCourseOutlineReader =
			originalVisibleReader
	})
}

func TestCourseOutlineMountRejectsLiveAndSnapshotDomainMismatch(
	t *testing.T,
) {
	lessonPlan := &models.LessonPlan{
		EducationDomain: models.EducationDomainVocational,
	}

	publisher := ""

	result, err :=
		normalizeLessonPlanCourseOutlineMount(
			context.Background(),
			lessonPlan,
			models.EducationDomainK12,
			&publisher,
		)

	if result != nil {
		t.Fatal(
			"跨教育域挂载不应返回可落库值",
		)
	}

	if !errors.Is(
		err,
		ErrOutlineEducationDomainMismatch,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}

func TestCourseOutlineMountRejectsNamedPublisherForVocational(
	t *testing.T,
) {
	preserveCourseOutlineGuardDependency(t)

	queryCalled := false

	courseOutlineListActiveByDomain = func(
		ctx context.Context,
		subject string,
		educationDomain string,
	) ([]*models.CourseOutline, error) {
		queryCalled = true
		return nil, nil
	}

	lessonPlan := &models.LessonPlan{
		Subject:         "会计实务",
		Grade:           "第一学期",
		EducationDomain: models.EducationDomainVocational,
	}

	publisher := "人教版"

	result, err :=
		normalizeLessonPlanCourseOutlineMount(
			context.Background(),
			lessonPlan,
			models.EducationDomainVocational,
			&publisher,
		)

	if result != nil {
		t.Fatal(
			"非K12具名出版社不应返回可落库值",
		)
	}

	if !errors.Is(
		err,
		ErrOutlinePublisherNotAllowed,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}

	if queryCalled {
		t.Fatal(
			"非法出版社应在查询数据库前被拒绝",
		)
	}
}

func TestCourseOutlineMountAllowsOrdinaryAdultOutline(
	t *testing.T,
) {
	lessonPlan := &models.LessonPlan{
		Subject:         "数字技能培训",
		Grade:           "零基础",
		EducationDomain: models.EducationDomainAdult,
	}

	publisher := " "

	result, err :=
		normalizeLessonPlanCourseOutlineMount(
			context.Background(),
			lessonPlan,
			models.EducationDomainAdult,
			&publisher,
		)
	if err != nil {
		t.Fatalf(
			"成人教育普通课程大纲挂载失败: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"普通课程大纲挂载应保留非nil三态",
		)
	}

	if *result != "" {
		t.Fatalf(
			"空出版社未正确规范化: %q",
			*result,
		)
	}
}

func TestCourseOutlineMountRejectsUnavailableK12Publisher(
	t *testing.T,
) {
	preserveCourseOutlineGuardDependency(t)

	courseOutlineListActiveByDomain = func(
		ctx context.Context,
		subject string,
		educationDomain string,
	) ([]*models.CourseOutline, error) {
		if subject != "数学" {
			t.Fatalf(
				"查询学科错误: %s",
				subject,
			)
		}

		if educationDomain !=
			models.EducationDomainK12 {
			t.Fatalf(
				"查询教育域错误: %s",
				educationDomain,
			)
		}

		return []*models.CourseOutline{},
			nil
	}

	lessonPlan := &models.LessonPlan{
		Subject:         "数学",
		Grade:           "七年级",
		EducationDomain: models.EducationDomainK12,
	}

	publisher := "不存在版"

	result, err :=
		normalizeLessonPlanCourseOutlineMount(
			context.Background(),
			lessonPlan,
			models.EducationDomainK12,
			&publisher,
		)

	if result != nil {
		t.Fatal(
			"不存在的K12版本不应返回可落库值",
		)
	}

	if !errors.Is(
		err,
		ErrOutlinePublisherUnavailable,
	) {
		t.Fatalf(
			"错误类型不正确: %v",
			err,
		)
	}
}
