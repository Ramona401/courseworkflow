package services

// courseware_derived_task_tracker.go — 课件方案落库后的派生后台任务管理
//
// 管理两类可重新计算的best-effort任务：
//   1. courseware_alignment：课件方案与原教案的对齐校验；
//   2. courseware_normalize：教案原文预处理规整。
//
// 对齐任务安全要求：
//   - 外部入口必须传可信CoursewareActorContext；
//   - 任务登记前重新加载正式课件并执行作者私有控制面授权；
//   - 教案来源课件同时校验lesson_plan_id和education_domain；
//   - Actor深复制后才进入goroutine；
//   - runAlignment开始后再次加载正式课件，形成执行前最终防线。
//
// 教案规整当前保持原有Tracker治理，本批不改变其运行语义。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	backgroundTaskTypeCoursewareAlignment         = "courseware_alignment"
	backgroundTaskTypeCoursewareNormalize         = "courseware_normalize"
	backgroundTaskTypeCoursewarePageIndexBackfill = "courseware_page_index_backfill"
)

var coursewareDerivedTaskLog = logger.WithModule("courseware.derived_tasks")

// loadOwnedAlignmentInputs 加载并校验一次对齐任务的正式输入。
//
// 对于教案来源课件，同时加载关联教案并校验：
//   - courseware.lesson_plan_id与教案ID完全一致；
//   - 课件与教案均为具体教学域；
//   - 两者education_domain完全一致。
//
// 非教案来源返回nil lessonPlan，由runAlignment保持静默跳过语义。
func loadOwnedAlignmentInputs(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	*models.LessonPlan,
	error,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	courseware, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, nil, nil, err
	}

	if courseware.SourceType !=
		models.CWSourceLessonPlan {
		return courseware,
			scopedActor,
			nil,
			nil
	}

	if courseware.LessonPlanID == nil ||
		strings.TrimSpace(
			*courseware.LessonPlanID,
		) == "" {
		return nil,
			nil,
			nil,
			ErrCoursewareLessonPlanDomainInvalid
	}

	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				*courseware.LessonPlanID,
			),
		)
	if err != nil {
		return nil,
			nil,
			nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareLessonPlanDomainInvalid,
				err,
			)
	}

	if err :=
		validateCoursewareLinkedLessonPlanDomain(
			courseware,
			lessonPlan,
		); err != nil {
		return nil, nil, nil, err
	}

	return courseware,
		scopedActor,
		lessonPlan,
		nil
}

// TriggerAlignmentTracked 异步触发受Tracker管理的课件对齐任务。
//
// 本方法在任务登记前执行Service层预检，避免内部代码绕过Handler
// 并抢占对齐任务键。
func (s *CoursewareAlignmentService) TriggerAlignmentTracked(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	BackgroundStartResult,
	error,
) {
	_, scopedActor, _, err :=
		loadOwnedAlignmentInputs(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return BackgroundInvalid, err
	}

	return s.triggerAlignmentTrackedForAuthorizedActor(
		coursewareID,
		scopedActor,
	), nil
}

// triggerAlignmentTrackedForAuthorizedActor 登记已完成预检的对齐任务。
//
// 本方法仅供包内正式入口和不访问数据库的Tracker测试使用。
func (s *CoursewareAlignmentService) triggerAlignmentTrackedForAuthorizedActor(
	coursewareID string,
	actor *CoursewareActorContext,
) BackgroundStartResult {
	if strings.TrimSpace(coursewareID) == "" ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return BackgroundInvalid
	}

	asyncActor :=
		CloneCoursewareActorContext(actor)
	if asyncActor == nil {
		return BackgroundInvalid
	}

	task, result :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewareAlignment,
			coursewareID,
			BackgroundTaskBestEffort,
			nil,
		)
	if result != BackgroundStarted {
		coursewareDerivedTaskLog.Info(
			"课件对齐任务未启动",
			"courseware_id", coursewareID,
			"result", string(result),
		)
		return result
	}

	go func(
		taskActor *CoursewareActorContext,
	) {
		err := task.Run(func() error {
			s.runAlignment(
				context.Background(),
				coursewareID,
				taskActor,
			)
			return nil
		})
		if err == nil {
			return
		}

		coursewareDerivedTaskLog.Error(
			"课件对齐后台任务异常",
			"courseware_id", coursewareID,
			"error", err,
		)

		s.markAlignmentBackgroundFailed(
			coursewareID,
			err,
		)
	}(asyncActor)

	return BackgroundStarted
}

// markAlignmentBackgroundFailed 尽力把异常中断的对齐报告改为failed。
func (s *CoursewareAlignmentService) markAlignmentBackgroundFailed(
	coursewareID string,
	cause error,
) {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
	defer cancel()

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			coursewareID,
		)
	if err != nil || courseware == nil {
		coursewareDerivedTaskLog.Warn(
			"对齐任务异常后无法读取课件，跳过failed收尾",
			"courseware_id", coursewareID,
			"error", err,
		)
		return
	}

	if courseware.LessonPlanID == nil ||
		strings.TrimSpace(
			*courseware.LessonPlanID,
		) == "" {
		return
	}

	message := "后台对齐校验异常"
	if cause != nil {
		message = fmt.Sprintf(
			"后台对齐校验异常: %v",
			cause,
		)
	}

	s.failReport(
		ctx,
		coursewareID,
		courseware.LessonPlanID,
		message,
	)
}

// TriggerPageIndexBackfillTracked 启动受Tracker管理的页索引回填。
func (s *CoursewareIndexService) TriggerPageIndexBackfillTracked(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	rawText string,
) (
	BackgroundStartResult,
	error,
) {
	_, scopedActor, err :=
		loadOwnedPageIndexBackfillInputs(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return BackgroundInvalid, err
	}

	if strings.TrimSpace(rawText) == "" {
		return BackgroundInvalid,
			fmt.Errorf(
				"AOCI索引回填原文为空",
			)
	}

	return s.triggerPageIndexBackfillForAuthorizedActor(
		coursewareID,
		scopedActor,
		rawText,
	), nil
}

// triggerPageIndexBackfillForAuthorizedActor 登记已完成预检的回填任务。
func (s *CoursewareIndexService) triggerPageIndexBackfillForAuthorizedActor(
	coursewareID string,
	actor *CoursewareActorContext,
	rawText string,
) BackgroundStartResult {
	if strings.TrimSpace(coursewareID) == "" ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" ||
		strings.TrimSpace(rawText) == "" {
		return BackgroundInvalid
	}

	asyncActor :=
		CloneCoursewareActorContext(
			actor,
		)
	if asyncActor == nil {
		return BackgroundInvalid
	}

	task, result :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewarePageIndexBackfill,
			coursewareID,
			BackgroundTaskBestEffort,
			nil,
		)
	if result != BackgroundStarted {
		coursewareDerivedTaskLog.Info(
			"课件页索引回填任务未启动",
			"courseware_id", coursewareID,
			"result", string(result),
		)
		return result
	}

	go func(
		taskActor *CoursewareActorContext,
		sourceText string,
	) {
		err := task.Run(func() error {
			s.runPageIndexBackfill(
				coursewareID,
				taskActor,
				sourceText,
			)
			return nil
		})
		if err == nil {
			return
		}

		coursewareDerivedTaskLog.Warn(
			"课件页索引回填后台任务失败",
			"courseware_id", coursewareID,
			"error", err,
		)
	}(
		asyncActor,
		rawText,
	)

	return BackgroundStarted
}

// TriggerEnsureNormalizedTracked 异步触发受Tracker管理的教案规整任务。
func (s *CoursewareLessonNormalizeService) TriggerEnsureNormalizedTracked(
	coursewareID string,
) BackgroundStartResult {
	task, result :=
		GlobalBackgroundTasks.TryStartExternal(
			backgroundTaskTypeCoursewareNormalize,
			coursewareID,
			BackgroundTaskBestEffort,
			nil,
		)
	if result != BackgroundStarted {
		coursewareDerivedTaskLog.Info(
			"教案规整任务未启动",
			"courseware_id", coursewareID,
			"result", string(result),
		)
		return result
	}

	go func() {
		err := task.Run(func() error {
			ctx := context.Background()

			courseware, getErr :=
				repository.GetCoursewareByID(
					ctx,
					coursewareID,
				)
			if getErr != nil {
				return fmt.Errorf(
					"读取待规整课件失败: %w",
					getErr,
				)
			}
			if courseware == nil {
				return fmt.Errorf(
					"待规整课件为空",
				)
			}

			return s.EnsureNormalized(
				ctx,
				courseware,
			)
		})
		if err == nil {
			return
		}

		coursewareDerivedTaskLog.Warn(
			"教案规整后台任务失败",
			"courseware_id", coursewareID,
			"error", err,
		)
	}()

	return BackgroundStarted
}
