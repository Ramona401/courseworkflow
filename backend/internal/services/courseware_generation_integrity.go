package services

// courseware_generation_integrity.go — R-04 页面生成完整性运行包装
//
// 本文件把“冻结方案快照 → 版本化页面写回 → 终态逐页对账”抽成普通批量生成与全自动装配共用能力。
// 它刻意不修改两个超过900行的旧主编排文件：
//   - courseware_gen_service.go
//   - courseware_auto_assembly_service.go
//
// 两条旧生成链仍负责真正的AI生成；本文件只负责可信运行身份与完整性结果。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrCoursewareGenerationIntegrityIncomplete 表示生成主体结束，但冻结方案仍有失败、取消或缺失页。
	ErrCoursewareGenerationIntegrityIncomplete = errors.New("课件页数完整性对账未通过")

	// ErrCoursewareBatchRunKindMismatch 表示普通批量生成停止入口碰到了全自动装配运行。
	ErrCoursewareBatchRunKindMismatch = errors.New("当前活动运行不是普通批量生成")
)

func beginCoursewareGenerationRun(
	ctx context.Context,
	coursewareID string,
	startedBy string,
	runKind string,
	skipVideo bool,
) (*models.CoursewareAssemblyRun, error) {
	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件页面方案失败: %w",
			err,
		)
	}

	metadataJSON, err :=
		repository.BuildCoursewareGenerationRunMetadata(
			runKind,
			pages,
		)
	if err != nil {
		return nil, err
	}

	run, err :=
		repository.BeginCoursewareGenerationRun(
			ctx,
			coursewareID,
			startedBy,
			skipVideo,
			metadataJSON,
		)
	if err != nil {
		return nil, err
	}

	return run, nil
}

func appendCoursewareGenerationErrorMessage(
	current string,
	addition string,
) string {
	current = strings.TrimSpace(current)
	addition = strings.TrimSpace(addition)

	switch {
	case addition == "":
		return current
	case current == "":
		return addition
	default:
		return current + "；" + addition
	}
}

func finalizeCoursewareGenerationRun(
	run *models.CoursewareAssemblyRun,
	runKind string,
	skipVideo bool,
	proposedStatus string,
	errorMessage string,
	wrapper string,
) (
	string,
	string,
	*models.CoursewareGenerationIntegrity,
	error,
) {
	if run == nil ||
		strings.TrimSpace(run.ID) == "" ||
		strings.TrimSpace(run.CoursewareID) == "" ||
		run.Version <= 0 {
		return models.CoursewareAssemblyStatusFailed,
			"生成运行身份无效",
			nil,
			repository.ErrCoursewareAssemblyInvalid
	}

	finalStatus := proposedStatus
	if !models.IsCoursewareAssemblyFinalStatus(
		finalStatus,
	) {
		finalStatus =
			models.CoursewareAssemblyStatusFailed
		errorMessage =
			appendCoursewareGenerationErrorMessage(
				errorMessage,
				"生成运行终态无效",
			)
	}

	reconcileCtx, reconcileCancel :=
		context.WithTimeout(
			context.Background(),
			8*time.Second,
		)
	runKindFromDB, integrity, reconcileErr :=
		repository.ReconcileCoursewareGenerationIntegrity(
			reconcileCtx,
			run.CoursewareID,
			run.Version,
			run.ID,
			finalStatus,
		)
	reconcileCancel()

	if reconcileErr != nil {
		finalStatus =
			models.CoursewareAssemblyStatusFailed
		errorMessage =
			appendCoursewareGenerationErrorMessage(
				errorMessage,
				fmt.Sprintf(
					"页数完整性对账失败: %v",
					reconcileErr,
				),
			)
		integrity = nil
	} else {
		if strings.TrimSpace(runKindFromDB) != "" {
			runKind = runKindFromDB
		}

		if integrity == nil {
			finalStatus =
				models.CoursewareAssemblyStatusFailed
			errorMessage =
				appendCoursewareGenerationErrorMessage(
					errorMessage,
					"缺少本次运行的稳定页面方案快照",
				)
		} else if finalStatus ==
			models.CoursewareAssemblyStatusCompleted &&
			!integrity.Complete {
			finalStatus =
				models.CoursewareAssemblyStatusFailed
			errorMessage =
				appendCoursewareGenerationErrorMessage(
					errorMessage,
					fmt.Sprintf(
						"%s：期望%d页，数据库实际%d页，成功%d页，失败%d页，取消%d页，缺失%d页",
						ErrCoursewareGenerationIntegrityIncomplete.Error(),
						integrity.ExpectedCount,
						integrity.ActualPageCount,
						integrity.SuccessCount,
						integrity.FailedCount,
						integrity.CancelledCount,
						integrity.MissingCount,
					),
				)
		}
	}

	metadataPayload := map[string]interface{}{
		"wrapper":          wrapper,
		"run_kind":         runKind,
		"assembly_version": run.Version,
		"assembly_run_id":  run.ID,
		"skip_video":       skipVideo,
		"finished_at": time.Now().UTC().
			Format(
				time.RFC3339Nano,
			),
	}

	if integrity != nil {
		metadataPayload["reconciliation"] =
			integrity
	}
	if strings.TrimSpace(errorMessage) != "" {
		metadataPayload["error"] =
			strings.TrimSpace(errorMessage)
	}

	metadataJSON := "{}"
	encoded, encodeErr := json.Marshal(
		metadataPayload,
	)
	if encodeErr != nil {
		finalStatus =
			models.CoursewareAssemblyStatusFailed
		errorMessage =
			appendCoursewareGenerationErrorMessage(
				errorMessage,
				fmt.Sprintf(
					"序列化生成终态metadata失败: %v",
					encodeErr,
				),
			)
	} else {
		metadataJSON = string(encoded)
	}

	finishCtx, finishCancel :=
		context.WithTimeout(
			context.Background(),
			8*time.Second,
		)
	defer finishCancel()

	finishErr :=
		repository.FinishCoursewareAssembly(
			finishCtx,
			run.CoursewareID,
			run.Version,
			run.ID,
			finalStatus,
			errorMessage,
			metadataJSON,
		)
	if finishErr != nil {
		return finalStatus,
			errorMessage,
			integrity,
			finishErr
	}

	return finalStatus,
		errorMessage,
		integrity,
		nil
}

func coursewareGenerationRunCancelRequested(
	ctx context.Context,
	run *models.CoursewareAssemblyRun,
) (
	bool,
	error,
) {
	if run == nil {
		return false,
			repository.ErrCoursewareAssemblyInvalid
	}

	state, err :=
		repository.GetCoursewareAssemblyState(
			ctx,
			run.CoursewareID,
		)
	if err != nil {
		return false, err
	}

	return state != nil &&
			state.ActiveRunID != nil &&
			*state.ActiveRunID == run.ID &&
			state.Version == run.Version &&
			state.Status ==
				models.CoursewareAssemblyStatusCancelRequested,
		nil
}

// GenerateRemainingPagesVersioned 为普通批量HTML生成增加与自动装配一致的数据库版本与R-04完整性边界。
//
// 旧 GenerateRemainingPages 保持不动；它收到的context已经包含版本身份，因此既有
// repository.UpdateCWPageHTML / UpdateCoursewareStatus 会自动走版本化SQL。
// 旧实现仍只生成html_content为空的页面，所以“只补生成缺失页”不会覆盖已成功页面。
func (s *CoursewareGenService) GenerateRemainingPagesVersioned(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (returnErr error) {
	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return ErrCoursewareActorRequired
	}

	courseware, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return err
	}
	if courseware == nil {
		return ErrCoursewareEducationDomainInvalid
	}

	if courseware.Status !=
		models.CoursewareStatusGenerating &&
		courseware.Status !=
			models.CoursewareStatusPreview {
		return fmt.Errorf(
			"当前状态不允许批量生成课件页面: %s",
			courseware.Status,
		)
	}
	if strings.TrimSpace(
		courseware.NavTemplateHTML,
	) == "" {
		return fmt.Errorf(
			"请先确认导航栏样式再批量生成页面",
		)
	}

	run, err :=
		beginCoursewareGenerationRun(
			ctx,
			coursewareID,
			scopedActor.UserID,
			models.CoursewareGenerationRunKindBatch,
			true,
		)
	if err != nil {
		return fmt.Errorf(
			"启动批量生成运行失败: %w",
			err,
		)
	}

	generationCtx :=
		repository.WithCoursewareAssemblyWriteContext(
			ctx,
			repository.CoursewareAssemblyWriteContext{
				CoursewareID: coursewareID,
				Version:      run.Version,
				RunID:        run.ID,
			},
		)

	defer func() {
		recovered := recover()

		finalStatus :=
			models.CoursewareAssemblyStatusCompleted
		errorMessage := ""

		stateCtx, stateCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		cancelRequested, stateErr :=
			coursewareGenerationRunCancelRequested(
				stateCtx,
				run,
			)
		stateCancel()

		switch {
		case recovered != nil:
			finalStatus =
				models.CoursewareAssemblyStatusFailed
			errorMessage = fmt.Sprintf(
				"批量页面生成panic: %v",
				recovered,
			)

		case cancelRequested:
			finalStatus =
				models.CoursewareAssemblyStatusCancelled

		case returnErr != nil:
			finalStatus =
				models.CoursewareAssemblyStatusFailed
			errorMessage =
				returnErr.Error()

		case stateErr != nil:
			finalStatus =
				models.CoursewareAssemblyStatusFailed
			errorMessage = fmt.Sprintf(
				"读取批量生成终态失败: %v",
				stateErr,
			)
		}

		resolvedStatus,
			resolvedError,
			_,
			finishErr :=
			finalizeCoursewareGenerationRun(
				run,
				models.CoursewareGenerationRunKindBatch,
				true,
				finalStatus,
				errorMessage,
				"database_versioned_batch",
			)

		if finishErr != nil &&
			!errors.Is(
				finishErr,
				repository.ErrCoursewareAssemblyVersionConflict,
			) {
			if returnErr == nil &&
				recovered == nil {
				returnErr = fmt.Errorf(
					"批量生成主体已结束，但数据库运行收敛失败: %w",
					finishErr,
				)
			}
		}

		if returnErr == nil &&
			recovered == nil &&
			resolvedStatus ==
				models.CoursewareAssemblyStatusFailed {
			returnErr = fmt.Errorf(
				"%w: %s",
				ErrCoursewareGenerationIntegrityIncomplete,
				strings.TrimSpace(
					resolvedError,
				),
			)
		}

		if recovered != nil {
			panic(recovered)
		}
	}()

	returnErr =
		s.GenerateRemainingPages(
			generationCtx,
			coursewareID,
			scopedActor,
		)

	return returnErr
}

// CancelGenerateVersioned 先冻结当前batch数据库运行，再发送旧生成链停止信号。
//
// 如果当前活动运行是assembly，本入口不会错误取消自动装配；自动装配仍只能走专用取消端点。
func (s *CoursewareGenService) CancelGenerateVersioned(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	operationCtx, operationCancel :=
		context.WithTimeout(
			ctx,
			5*time.Second,
		)
	defer operationCancel()

	if _, _, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				operationCtx,
				coursewareID,
				actor,
			); err != nil {
		return err
	}

	var databaseCancelErr error

	state, stateErr :=
		repository.GetCoursewareAssemblyState(
			operationCtx,
			coursewareID,
		)
	if stateErr != nil &&
		!errors.Is(
			stateErr,
			repository.ErrCoursewareAssemblyNotFound,
		) {
		databaseCancelErr = stateErr
	}

	if stateErr == nil &&
		state != nil &&
		state.ActiveRunID != nil {
		active :=
			state.Status ==
				models.CoursewareAssemblyStatusRunning ||
				state.Status ==
					models.CoursewareAssemblyStatusCancelRequested

		if active &&
			state.RunKind !=
				models.CoursewareGenerationRunKindBatch {
			return ErrCoursewareBatchRunKindMismatch
		}

		if active {
			switch state.Status {
			case models.CoursewareAssemblyStatusRunning:
				databaseCancelErr =
					repository.RequestCoursewareAssemblyCancel(
						operationCtx,
						coursewareID,
						state.Version,
						*state.ActiveRunID,
					)

			case models.CoursewareAssemblyStatusCancelRequested:
				// 已进入取消请求态，保持幂等。
			}
		}
	}

	signalCtx, signalCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
	signalErr :=
		s.CancelGenerate(
			signalCtx,
			coursewareID,
			actor,
		)
	signalCancel()

	if databaseCancelErr != nil &&
		!errors.Is(
			databaseCancelErr,
			repository.ErrCoursewareAssemblyVersionConflict,
		) {
		return databaseCancelErr
	}

	return signalErr
}
