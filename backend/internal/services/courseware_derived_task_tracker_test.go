package services

import (
	"testing"

	"tedna/internal/models"
)

func newAuthorizedDerivedAlignmentActor() *CoursewareActorContext {
	return &CoursewareActorContext{
		UserID:          "user-001",
		Role:            "teacher",
		EducationDomain: models.EducationDomainK12,
	}
}

func TestCoursewareDerivedTasksRejectDuringDraining(
	t *testing.T,
) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks =
		NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks =
			originalTracker
	}()

	GlobalBackgroundTasks.BeginDraining()

	alignmentService :=
		&CoursewareAlignmentService{}

	alignmentResult :=
		alignmentService.
			triggerAlignmentTrackedForAuthorizedActor(
				"cw-draining-alignment",
				newAuthorizedDerivedAlignmentActor(),
			)
	if alignmentResult !=
		BackgroundRejectedDraining {
		t.Fatalf(
			"排空期间对齐任务应被拒绝，实际=%s",
			alignmentResult,
		)
	}

	normalizeService :=
		&CoursewareLessonNormalizeService{}

	normalizeResult :=
		normalizeService.
			TriggerEnsureNormalizedTracked(
				"cw-draining-normalize",
			)
	if normalizeResult !=
		BackgroundRejectedDraining {
		t.Fatalf(
			"排空期间规整任务应被拒绝，实际=%s",
			normalizeResult,
		)
	}

	if summary :=
		GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf(
			"排空期间不应登记派生任务: %+v",
			summary,
		)
	}
}

func TestCoursewareDerivedTasksRejectInvalidResourceID(
	t *testing.T,
) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks =
		NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks =
			originalTracker
	}()

	alignmentService :=
		&CoursewareAlignmentService{}

	if result :=
		alignmentService.
			triggerAlignmentTrackedForAuthorizedActor(
				"",
				newAuthorizedDerivedAlignmentActor(),
			); result != BackgroundInvalid {
		t.Fatalf(
			"空课件ID应返回invalid，实际=%s",
			result,
		)
	}

	normalizeService :=
		&CoursewareLessonNormalizeService{}

	if result :=
		normalizeService.
			TriggerEnsureNormalizedTracked(""); result != BackgroundInvalid {
		t.Fatalf(
			"空课件ID应返回invalid，实际=%s",
			result,
		)
	}
}

func TestCoursewareDerivedTasksRejectMissingActor(
	t *testing.T,
) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks =
		NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks =
			originalTracker
	}()

	alignmentService :=
		&CoursewareAlignmentService{}

	if result :=
		alignmentService.
			triggerAlignmentTrackedForAuthorizedActor(
				"cw-missing-actor",
				nil,
			); result != BackgroundInvalid {
		t.Fatalf(
			"空Actor应返回invalid，实际=%s",
			result,
		)
	}

	if summary :=
		GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf(
			"空Actor不应登记后台任务: %+v",
			summary,
		)
	}
}

func newAuthorizedPageIndexBackfillActor() *CoursewareActorContext {
	return &CoursewareActorContext{
		UserID:          "backfill-owner-001",
		Role:            "teacher",
		EducationDomain: models.EducationDomainK12,
	}
}

func TestValidateCoursewarePageIndexBackfillSource(
	t *testing.T,
) {
	tests := []struct {
		name       string
		courseware *models.Courseware
		wantError  bool
	}{
		{
			name: "PPT来源允许",
			courseware: &models.Courseware{
				SourceType: models.CWSourcePPTUpload,
			},
		},
		{
			name: "Word来源允许",
			courseware: &models.Courseware{
				SourceType: models.CWSourceDocUpload,
			},
		},
		{
			name: "其它来源拒绝",
			courseware: &models.Courseware{
				SourceType: "topic",
			},
			wantError: true,
		},
		{
			name:       "空课件拒绝",
			courseware: nil,
			wantError:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				err :=
					validateCoursewarePageIndexBackfillSource(
						testCase.courseware,
					)

				if testCase.wantError &&
					err == nil {
					t.Fatal(
						"期望错误，实际为nil",
					)
				}

				if !testCase.wantError &&
					err != nil {
					t.Fatalf(
						"不期望错误，实际=%v",
						err,
					)
				}
			},
		)
	}
}

func TestCoursewarePageIndexBackfillRejectDuringDraining(
	t *testing.T,
) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks =
		NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks =
			originalTracker
	}()

	GlobalBackgroundTasks.BeginDraining()

	service := &CoursewareIndexService{}

	result :=
		service.
			triggerPageIndexBackfillForAuthorizedActor(
				"cw-backfill-draining",
				newAuthorizedPageIndexBackfillActor(),
				"测试原文",
			)

	if result !=
		BackgroundRejectedDraining {
		t.Fatalf(
			"排空期间回填任务应被拒绝，实际=%s",
			result,
		)
	}

	if summary :=
		GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf(
			"排空期间不应登记回填任务: %+v",
			summary,
		)
	}
}

func TestCoursewarePageIndexBackfillRejectInvalidInput(
	t *testing.T,
) {
	originalTracker := GlobalBackgroundTasks
	GlobalBackgroundTasks =
		NewBackgroundTaskTracker()

	defer func() {
		GlobalBackgroundTasks =
			originalTracker
	}()

	service := &CoursewareIndexService{}
	actor :=
		newAuthorizedPageIndexBackfillActor()

	tests := []struct {
		name         string
		coursewareID string
		actor        *CoursewareActorContext
		rawText      string
	}{
		{
			name:         "空课件ID",
			coursewareID: "",
			actor:        actor,
			rawText:      "测试原文",
		},
		{
			name:         "空Actor",
			coursewareID: "cw-invalid",
			actor:        nil,
			rawText:      "测试原文",
		},
		{
			name:         "空原文",
			coursewareID: "cw-invalid",
			actor:        actor,
			rawText:      "",
		},
	}

	for _, testCase := range tests {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				result :=
					service.
						triggerPageIndexBackfillForAuthorizedActor(
							testCase.coursewareID,
							testCase.actor,
							testCase.rawText,
						)

				if result !=
					BackgroundInvalid {
					t.Fatalf(
						"无效输入应返回invalid，实际=%s",
						result,
					)
				}
			},
		)
	}

	if summary :=
		GlobalBackgroundTasks.Summary(); summary.Active != 0 {
		t.Fatalf(
			"无效输入不应登记任务: %+v",
			summary,
		)
	}
}
