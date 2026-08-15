package services

// courseware_auto_assembly_version.go — 自动装配数据库业务版本包装
//
// 本文件负责在进入主编排前后建立数据库业务运行边界，并在新run创建前读取上一轮完整性事实，
// 把可信的定向补生成范围注入context；它不接受浏览器提交page_id，也不改IAOCI逐槽位协议。
//
//   1. 在领取数据库运行前执行作者权限和轻量前置检查；
//   2. 原子领取单调递增assembly_version和run_id；
//   3. 把运行身份写入context，沿现有调用链传播；
//   4. repository中的页面HTML、课件状态和媒体建议写入自动识别该身份；
//   5. 取消时先把数据库状态改为cancel_requested，再关闭进程内停止信号；
//   6. 正常、失败、取消和panic路径最终都收敛courseware_assembly_runs。
//
// 版本身份、定向补生成范围、取消语义与终态对账都在本层收敛，主编排只消费可信context，
// 从而保留IAOCI稳定索引、导航保护、背景处理和媒体计费幂等能力。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrCoursewareAutoAssemblyRunKindMismatch 表示自动装配停止入口碰到了普通批量生成运行。
	ErrCoursewareAutoAssemblyRunKindMismatch = errors.New("当前活动运行不是自动装配")
)

// preflightAutoAssemblyVersioned 在创建数据库运行记录前完成轻量预检。
//
// 原AutoAssemble仍会执行完整prepareAssembly；这里提前检查不依赖AI调用的
// 确定性条件，避免无权限、无导航、无锚点或无页面的请求产生虚假运行记录。
func (s *CoursewareAutoAssemblyService) preflightAutoAssemblyVersioned(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*CoursewareActorContext, error) {
	courseware, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	if courseware == nil {
		return nil,
			ErrCoursewareEducationDomainInvalid
	}

	if courseware.Status !=
		models.CoursewareStatusGenerating &&
		courseware.Status !=
			models.CoursewareStatusPreview {
		return nil, fmt.Errorf(
			"当前状态不允许全自动装配: %s",
			courseware.Status,
		)
	}

	if strings.TrimSpace(
		courseware.NavTemplateHTML,
	) == "" {
		return nil,
			fmt.Errorf(
				"请先确认导航栏样式再启用全自动装配",
			)
	}

	if courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(
			*courseware.StyleAnchorAssetID,
		) == "" {
		return nil,
			fmt.Errorf(
				"全自动装配需先设置风格锚点",
			)
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件页面失败: %w",
			err,
		)
	}
	if len(pages) == 0 {
		return nil,
			fmt.Errorf(
				"课件没有页面方案，请先生成索引",
			)
	}

	return scopedActor, nil
}

// AutoAssembleVersioned 保留无启动票据的内部调用兼容。
func (s *CoursewareAutoAssemblyService) AutoAssembleVersioned(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	skipVideo bool,
) error {
	return s.AutoAssembleVersionedWithLaunch(
		ctx,
		coursewareID,
		actor,
		skipVideo,
		"",
	)
}

// AutoAssembleVersionedWithLaunch 在数据库业务版本保护下执行现有AutoAssemble。
//
// launchToken由Tracked Handler在后台任务登记前创建；
// 取消若发生在数据库运行创建前，只会取消与该token精确绑定的本次启动。
func (s *CoursewareAutoAssemblyService) AutoAssembleVersionedWithLaunch(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	skipVideo bool,
	launchToken string,
) (returnErr error) {
	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return ErrCoursewareActorRequired
	}

	repairMode := coursewareImageRepairModeFromContext(ctx)

	scopedActor, preflightErr :=
		s.preflightAutoAssemblyVersioned(
			ctx,
			coursewareID,
			actor,
		)
	if preflightErr != nil {
		s.pushError(
			coursewareID,
			preflightErr.Error(),
		)
		return preflightErr
	}

	// 消费启动票据、判断启动前取消和领取数据库运行必须在同一临界区。
	lifecycleLock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lifecycleLock.Lock()

	cancelBeforeStart, launchErr :=
		consumeCoursewareAutoAssemblyLaunchLocked(
			coursewareID,
			launchToken,
		)
	if launchErr != nil {
		lifecycleLock.Unlock()

		s.pushError(
			coursewareID,
			launchErr.Error(),
		)
		return launchErr
	}

	if cancelBeforeStart {
		lifecycleLock.Unlock()

		cwAssemblyLog.Info(
			"自动装配在正式启动前已被取消",
			"courseware_id",
			coursewareID,
		)
		return nil
	}

	// 普通装配必须在创建新run之前读取上一轮HTML完整性补页范围；
	// 图片智能补配与HTML完整性正交，repair模式严格不消费该范围。
	retryScope := coursewareAutoAssemblyRetryScope{}
	if !repairMode {
		var retryErr error
		retryScope, retryErr =
			resolveCoursewareAutoAssemblyRetryScope(
				ctx,
				coursewareID,
			)
		if retryErr != nil {
			lifecycleLock.Unlock()

			s.pushError(
				coursewareID,
				retryErr.Error(),
			)
			return retryErr
		}
	}

	run, err :=
		beginCoursewareGenerationRun(
			ctx,
			coursewareID,
			scopedActor.UserID,
			models.CoursewareGenerationRunKindAssembly,
			skipVideo,
		)

	lifecycleLock.Unlock()

	if err != nil {
		s.pushError(
			coursewareID,
			fmt.Sprintf(
				"启动装配运行失败: %v",
				err,
			),
		)
		return fmt.Errorf(
			"启动装配运行失败: %w",
			err,
		)
	}

	assemblyCtx :=
		repository.WithCoursewareAssemblyWriteContext(
			ctx,
			repository.CoursewareAssemblyWriteContext{
				CoursewareID: coursewareID,
				Version:      run.Version,
				RunID:        run.ID,
			},
		)

	if retryScope.Enabled {
		assemblyCtx =
			withCoursewareAutoAssemblyRetryScope(
				assemblyCtx,
				retryScope,
			)

		cwAssemblyLog.Info(
			"自动装配进入完整性定向补生成",
			"courseware_id", coursewareID,
			"assembly_version", run.Version,
			"retry_pages", len(retryScope.PageIDs),
			"force_pages", len(retryScope.ForcePageIDs),
		)
	}

	if repairMode {
		assemblyCtx = WithCoursewareImageRepairMode(assemblyCtx)
		cwAssemblyLog.Info(
			"自动装配进入失败配图定向智能补配",
			"courseware_id", coursewareID,
			"assembly_version", run.Version,
		)
	}

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
				"自动装配panic: %v",
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
			errorMessage =
				fmt.Sprintf(
					"读取装配终态失败: %v",
					stateErr,
				)
		}

		resolvedStatus,
			resolvedError,
			_,
			finishErr :=
			finalizeCoursewareGenerationRun(
				run,
				models.CoursewareGenerationRunKindAssembly,
				skipVideo,
				finalStatus,
				errorMessage,
				"database_versioned",
			)

		if finishErr != nil &&
			!errors.Is(
				finishErr,
				repository.ErrCoursewareAssemblyVersionConflict,
			) {
			cwAssemblyLog.Error(
				"数据库装配运行收敛失败",
				"courseware_id",
				coursewareID,
				"assembly_version",
				run.Version,
				"assembly_run_id",
				run.ID,
				"final_status",
				resolvedStatus,
				"error",
				finishErr,
			)

			if returnErr == nil &&
				recovered == nil {
				returnErr = fmt.Errorf(
					"装配主体已结束，但数据库运行收敛失败: %w",
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

	if repairMode {
		returnErr =
			s.RepairFailedCoursewareImages(
				assemblyCtx,
				coursewareID,
				scopedActor,
				skipVideo,
			)
		return returnErr
	}

	returnErr =
		s.AutoAssemble(
			assemblyCtx,
			coursewareID,
			scopedActor,
			skipVideo,
		)
	if returnErr != nil {
		return returnErr
	}

	// HTML完整不代表媒体完整。只把已经确认属于两类可智能修复错误的失败收敛为运行失败，
	// 这样前端不会在“配图仍失败”时自动跳进工作台；其它非修复型媒体错误保持原有best-effort语义。
	repairState, repairStateErr :=
		ReadCoursewareImageRepairState(
			assemblyCtx,
			coursewareID,
			scopedActor,
		)
	if repairStateErr != nil {
		returnErr = fmt.Errorf(
			"读取自动装配配图失败事实失败: %w",
			repairStateErr,
		)
		return returnErr
	}
	if repairState != nil && repairState.RetryableCount > 0 {
		returnErr = fmt.Errorf(
			"%w: 仍有%d处可智能补配的图片失败",
			ErrCoursewareImageRepairIncomplete,
			repairState.RetryableCount,
		)
	}

	return returnErr
}

// CancelAutoAssemblyVersioned 请求取消数据库当前运行，并停止进程继续派发。
//
// 数据库先进入cancel_requested，因此已经在远端执行的迟到请求即使返回，
// 后续HTML、课件状态和媒体建议写入也会被版本化SQL拒绝。
//
// 数据库运行尚未创建时，只允许给真实存在的启动票据绑定取消；
// 空闲课件调用取消不会遗留票据，也不会影响后续新装配。
func (s *CoursewareAutoAssemblyService) CancelAutoAssemblyVersioned(
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

	lifecycleLock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lifecycleLock.Lock()

	state, stateErr :=
		repository.GetCoursewareAssemblyState(
			operationCtx,
			coursewareID,
		)

	activeBatchRun :=
		stateErr == nil &&
			state != nil &&
			state.ActiveRunID != nil &&
			state.RunKind ==
				models.CoursewareGenerationRunKindBatch &&
			(state.Status ==
				models.CoursewareAssemblyStatusRunning ||
				state.Status ==
					models.CoursewareAssemblyStatusCancelRequested)

	if activeBatchRun {
		lifecycleLock.Unlock()
		return ErrCoursewareAutoAssemblyRunKindMismatch
	}

	activeDatabaseRun :=
		stateErr == nil &&
			state != nil &&
			state.ActiveRunID != nil &&
			(state.Status ==
				models.CoursewareAssemblyStatusRunning ||
				state.Status ==
					models.CoursewareAssemblyStatusCancelRequested)

	if activeDatabaseRun &&
		state.Status ==
			models.CoursewareAssemblyStatusRunning {
		databaseCancelErr =
			repository.RequestCoursewareAssemblyCancel(
				operationCtx,
				coursewareID,
				state.Version,
				*state.ActiveRunID,
			)

		if errors.Is(
			databaseCancelErr,
			repository.ErrCoursewareAssemblyVersionConflict,
		) {
			databaseCancelErr = nil
		}
	} else if stateErr != nil &&
		!errors.Is(
			stateErr,
			repository.ErrCoursewareAssemblyNotFound,
		) {
		databaseCancelErr =
			stateErr
	}

	// 仅当前进程确有待启动票据时，才记录启动前取消。
	if !activeDatabaseRun &&
		databaseCancelErr == nil {
		_ = markCoursewareAutoAssemblyPendingCancelLocked(
			coursewareID,
		)
	}

	lifecycleLock.Unlock()

	// 数据库冻结成功后，进程内停止信号不再依赖浏览器请求是否断开。
	signalCtx, signalCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
	signalErr :=
		s.CancelAutoAssembly(
			signalCtx,
			coursewareID,
			actor,
		)
	signalCancel()

	if databaseCancelErr != nil {
		return databaseCancelErr
	}

	return signalErr
}

// CancelCoursewareAutoAssemblyVersioned 供独立HTTP路由和其它外层调用使用。
//
// 取消链不依赖生成、资产或OSS实例字段，因此可以安全使用轻量服务实例。
func CancelCoursewareAutoAssemblyVersioned(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	service :=
		&CoursewareAutoAssemblyService{}

	return service.CancelAutoAssemblyVersioned(
		ctx,
		coursewareID,
		actor,
	)
}
