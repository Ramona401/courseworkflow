package services

// price_sync_scheduler.go — 模型及媒体价格定时同步器。
//
// 调度器只处理价格同步，不参与图片、视频或TTS业务结算。
//
// 三层安全开关：
//   1. 应用级DisableSchedulers=true时不启动；
//   2. price_sync_enabled=false时只低频轮询配置；
//   3. 每个价格目标必须auto_sync_enabled=true才进入调度器预览。
//
// 自动应用还必须同时满足price_sync_auto_apply=true。
// 自动应用关闭时仍会保存预览批次，供超级管理员人工检查和应用。

import (
	"context"
	"sync"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	priceSyncDisabledPollInterval = 10 * time.Minute
	priceSyncSchedulerTimeout     = 5 * time.Minute
)

var (
	priceSyncSchedulerOnce sync.Once
	priceSyncSchedulerLog  = logger.WithModule(
		"services.price_sync_scheduler",
	)
)

// StartScheduler 启动全局唯一的价格同步调度器。
//
// 调用方必须先判断应用级DisableSchedulers。
// 多次调用只会启动一个后台协程。
func (service *PriceSyncService) StartScheduler() {
	if service == nil {
		return
	}

	priceSyncSchedulerOnce.Do(
		func() {
			go service.runPriceSyncScheduler()
		},
	)
}

// runPriceSyncScheduler 按动态配置循环调度。
//
// 关闭状态下每10分钟重新读取一次配置；
// 启用后按interval_hours等待，并在真正执行前再次读取最新配置。
func (service *PriceSyncService) runPriceSyncScheduler() {
	for {
		settings, err := service.GetSettings(
			context.Background(),
		)
		if err != nil {
			priceSyncSchedulerLog.Error(
				"读取价格同步配置失败",
				"error",
				err,
			)

			waitPriceSyncScheduler(
				priceSyncDisabledPollInterval,
			)
			continue
		}

		if !settings.Enabled {
			priceSyncSchedulerLog.Info(
				"价格自动同步当前关闭",
				"next_check",
				priceSyncDisabledPollInterval,
			)

			waitPriceSyncScheduler(
				priceSyncDisabledPollInterval,
			)
			continue
		}

		waitDuration := time.Duration(
			settings.IntervalHours,
		) * time.Hour

		priceSyncSchedulerLog.Info(
			"价格自动同步下次执行",
			"interval_hours",
			settings.IntervalHours,
			"next_run",
			time.Now().
				Add(waitDuration).
				Format("2006-01-02 15:04:05"),
		)

		waitPriceSyncScheduler(waitDuration)
		service.executeScheduledPriceSync()
	}
}

// executeScheduledPriceSync 执行一次定时同步。
func (service *PriceSyncService) executeScheduledPriceSync() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		priceSyncSchedulerTimeout,
	)
	defer cancel()

	settings, err := service.GetSettings(ctx)
	if err != nil {
		priceSyncSchedulerLog.Error(
			"定时同步前重新读取配置失败",
			"error",
			err,
		)
		return
	}

	if !settings.Enabled {
		priceSyncSchedulerLog.Info(
			"价格自动同步已在等待期间关闭，本次跳过",
		)
		return
	}

	hasTargets, err := hasScheduledPriceSyncTargets(ctx)
	if err != nil {
		priceSyncSchedulerLog.Error(
			"检查自动同步目标失败",
			"error",
			err,
		)
		return
	}

	if !hasTargets {
		priceSyncSchedulerLog.Info(
			"没有开启自动同步的价格目标，本次不创建空批次",
		)
		return
	}

	preview, err := service.Preview(
		ctx,
		&models.PriceSyncPreviewRequest{
			TriggerType: models.PriceSyncTriggerScheduler,
			Group:       settings.Group,

			IncludeText:  true,
			IncludeMedia: true,

			MaxChangePercent: settings.MaxChangePercent,
		},
		"",
	)
	if err != nil {
		priceSyncSchedulerLog.Error(
			"价格定时同步预览失败",
			"error",
			err,
		)
		return
	}

	if preview == nil || preview.Run == nil {
		priceSyncSchedulerLog.Error(
			"价格定时同步返回空预览结果",
		)
		return
	}

	priceSyncSchedulerLog.Info(
		"价格定时同步预览完成",
		"run_id",
		preview.Run.ID,
		"total",
		preview.Summary.TotalCount,
		"updates",
		preview.Summary.UpdateCount,
		"unchanged",
		preview.Summary.UnchangedCount,
		"skipped",
		preview.Summary.SkippedCount,
		"auto_apply",
		settings.AutoApply,
	)

	if !settings.AutoApply ||
		preview.Summary.UpdateCount <= 0 {
		return
	}

	applied, err := service.ApplySelected(
		ctx,
		preview.Run.ID,
		nil,
		"",
	)
	if err != nil {
		priceSyncSchedulerLog.Error(
			"价格定时同步自动应用失败",
			"run_id",
			preview.Run.ID,
			"error",
			err,
		)
		return
	}

	priceSyncSchedulerLog.Info(
		"价格定时同步自动应用完成",
		"run_id",
		preview.Run.ID,
		"applied",
		applied.Summary.AppliedCount,
		"stale",
		applied.Summary.StaleCount,
		"skipped",
		applied.Summary.SkippedCount,
	)
}

// hasScheduledPriceSyncTargets 防止没有启用目标时反复创建空批次。
func hasScheduledPriceSyncTargets(
	ctx context.Context,
) (bool, error) {
	textTargets, err :=
		repository.ListTextPriceSyncTargets(
			ctx,
			true,
		)
	if err != nil {
		return false, err
	}

	if len(textTargets) > 0 {
		return true, nil
	}

	mediaTargets, err :=
		repository.ListMediaPriceSyncTargets(
			ctx,
			true,
		)
	if err != nil {
		return false, err
	}

	return len(mediaTargets) > 0, nil
}

// waitPriceSyncScheduler 使用Timer而不是永久占用忙循环。
func waitPriceSyncScheduler(
	duration time.Duration,
) {
	if duration <= 0 {
		duration = priceSyncDisabledPollInterval
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	<-timer.C
}
