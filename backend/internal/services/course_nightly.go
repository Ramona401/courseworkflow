package services

// course_nightly.go — 夜间索引自动同步定时任务
//
// 每天凌晨3:30（北京时间）自动从OSS全量刷新所有已注册课程的索引
// 模式参考 verify_batch.go 的 StartNightlyVerifyScheduler
// v125新增

import (
	"fmt"
	"time"

	"tedna/internal/logger"
	"tedna/internal/repository"
)

// courseNightlyLog 夜间索引同步专用日志器
var courseNightlyLog = logger.WithModule("course_nightly")

// StartNightlyIndexSyncScheduler 启动夜间索引自动同步定时任务
//
// 功能说明：
//   - 每天凌晨3:30（Asia/Shanghai）自动执行一次全量索引刷新
//   - 遍历所有已注册且绑定了外部模块ID的课程
//   - 从OSS重新拉取最新索引文件并更新数据库
//   - 与夜间验收任务（2:00）错开，避免资源竞争
//   - 单次执行失败不影响后续调度，每天都会重新尝试
//
// 调用方式：
//   在 routes.go 的 Setup() 中调用 courseService.StartNightlyIndexSyncScheduler()
func (s *CourseService) StartNightlyIndexSyncScheduler() {
	go func() {
		// 加载北京时间时区
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			courseNightlyLog.Warn("加载Asia/Shanghai时区失败，降级为UTC+8", "error", err)
			loc = time.FixedZone("CST", 8*3600)
		}

		for {
			// 计算下次执行时间：每天凌晨3:30
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 30, 0, 0, loc)
			if now.After(next) {
				// 今天的3:30已过，等明天
				next = next.Add(24 * time.Hour)
			}
			waitDuration := next.Sub(now)

			courseNightlyLog.Info("夜间索引同步调度器等待中",
				"next_run", next.Format("2006-01-02 15:04:05"),
				"wait_duration", waitDuration.String())

			// 等待到指定时间
			timer := time.NewTimer(waitDuration)
			<-timer.C

			// 执行全量索引刷新
			startTime := time.Now()
			fmt.Printf("[夜间索引同步] %s 开始执行全量索引刷新...\n",
				startTime.In(loc).Format("2006-01-02 15:04:05"))

			result := s.doNightlyIndexSync()

			elapsed := time.Since(startTime)
			fmt.Printf("[夜间索引同步] %s 执行完成 耗时=%s 成功=%d 失败=%d 跳过=%d 总计=%d\n",
				time.Now().In(loc).Format("2006-01-02 15:04:05"),
				elapsed.Round(time.Millisecond).String(),
				result.Success, result.Failed, result.Skipped, result.Total)

			if result.Failed > 0 {
				courseNightlyLog.Warn("夜间索引同步部分失败",
					"success", result.Success,
					"failed", result.Failed,
					"skipped", result.Skipped,
					"total", result.Total,
					"errors", result.Errors,
					"elapsed", elapsed.String())
			} else {
				courseNightlyLog.Info("夜间索引同步全部成功",
					"success", result.Success,
					"skipped", result.Skipped,
					"total", result.Total,
					"elapsed", elapsed.String())
			}
		}
	}()
}

// NightlyIndexSyncResult 夜间索引同步结果
type NightlyIndexSyncResult struct {
	Total   int      `json:"total"`   // 已注册课程总数
	Success int      `json:"success"` // 成功刷新数
	Failed  int      `json:"failed"`  // 失败数
	Skipped int      `json:"skipped"` // 跳过数（无外部模块ID）
	Errors  []string `json:"errors"`  // 失败详情
}

// doNightlyIndexSync 执行夜间索引同步的具体逻辑
//
// 处理流程：
//   1. 查询所有已注册课程
//   2. 跳过未绑定外部模块ID的课程
//   3. 逐个调用 FetchIndex 从OSS拉取最新索引
//   4. 汇总成功/失败/跳过统计
//
// 错误处理：
//   - 单个课程刷新失败不影响其他课程
//   - 失败信息记录到 Errors 列表中，便于排查
func (s *CourseService) doNightlyIndexSync() *NightlyIndexSyncResult {
	result := &NightlyIndexSyncResult{}

	// 获取所有已注册课程列表
	items, err := repository.ListCourses()
	if err != nil {
		courseNightlyLog.Error("夜间索引同步：获取课程列表失败", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("获取课程列表失败: %s", err.Error()))
		return result
	}

	result.Total = len(items)

	for _, item := range items {
		// 跳过未绑定外部模块ID的课程
		if item.ExternalModuleID == nil || *item.ExternalModuleID <= 0 {
			result.Skipped++
			continue
		}

		// 调用 FetchIndex 从OSS拉取最新索引并更新数据库
		_, fetchErr := s.FetchIndex(item.CourseCode)
		if fetchErr != nil {
			result.Failed++
			errMsg := fmt.Sprintf("%s(module=%d): %s", item.CourseCode, *item.ExternalModuleID, fetchErr.Error())
			result.Errors = append(result.Errors, errMsg)
			courseNightlyLog.Warn("夜间索引同步：课程刷新失败",
				"course_code", item.CourseCode,
				"module_id", *item.ExternalModuleID,
				"error", fetchErr)
		} else {
			result.Success++
		}
	}

	return result
}
