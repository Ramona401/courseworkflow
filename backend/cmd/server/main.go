package main

// TE-DNA 2.0 服务入口
//
// 进程生命周期：
//   1. 初始化配置、数据库和路由；
//   2. 启动HTTP服务；
//   3. 监听SIGTERM与SIGINT；
//   4. 关停时拒绝新的后台长任务；
//   5. 通知批量生成和全自动装配停止继续派发新页面；
//   6. 关闭SSE、语音WebSocket和HTTP监听；
//   7. 最多等待短时间内能够完成的HTTP与Engine任务；
//   8. 不等待远端AI长请求，快速退出并由systemd启动新版本。
//
// 断点恢复原则：
//   - 每个成功页面都会立即写入数据库；
//   - 重启后批量生成只处理html_content为空的页面；
//   - 已完成页面不会重新生成；
//   - 图片使用稳定计费幂等键恢复；
//   - 异步视频使用已保存task_id恢复查询；
//   - 正在执行但尚未落库的同步文本请求可能需要重新生成当前页。
//
// 部署可用性优先：
//   不允许后台AI任务把单实例HTTP服务停止时间拖长到数分钟。
//   应用内部最多等待8秒，systemd在15秒时执行最终兜底。

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/repository"
	"tedna/internal/routes"
	"tedna/internal/services"
)

// 模块日志。
var log = logger.WithModule("main")

const (
	// fastShutdownTimeout 是日常部署时应用内部的最长关停等待时间。
	//
	// 该时间只用于：
	//   - 停止接受新HTTP连接；
	//   - 给极短的活动HTTP请求完成机会；
	//   - 停止Pipeline Engine；
	//   - 触发后台任务的停止派发钩子。
	//
	// 不等待远端AI长调用完整返回。
	fastShutdownTimeout = 8 * time.Second
)

func main() {
	// 1. 加载配置。
	cfg := config.Load()

	// 2. 初始化数据库。
	database.Init(cfg)
	defer database.Close()

	// 3. 收敛旧进程遗留的数据库装配运行。
	//
	// systemd单实例重启时，新进程启动意味着旧进程已经退出。
	// 数据库里仍为running/cancel_requested的装配均属于旧进程遗留，
	// 必须在接受新HTTP请求前标记为interrupted并清空活动运行指针。
	recoveryCtx, recoveryCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	recoveredAssemblies, recoveryErr :=
		repository.RecoverInterruptedCoursewareAssemblies(
			recoveryCtx,
		)

	recoveryCancel()

	if recoveryErr != nil {
		logger.Fatal(
			"恢复旧课件装配运行失败",
			"module", "main",
			"error", recoveryErr,
		)
	}

	if recoveredAssemblies > 0 {
		log.Warn(
			"已收敛旧进程遗留的课件装配运行",
			"interrupted_coursewares",
			recoveredAssemblies,
		)
	}

	// 4. 注册完整业务路由。
	mux :=
		routes.SetupWithCoursewareAssemblyRuntime(
			cfg,
		)

	// 5. 创建HTTP服务器。
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      600 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 6. 启动HTTP服务。
	serverErrCh := make(chan error, 1)

	log.Info(
		"TE-DNA 2.0 服务启动",
		"port", cfg.Port,
		"version", config.AppVersion,
		"read_header_timeout", "30s",
		"write_timeout", "600s",
		"shutdown_timeout",
		fastShutdownTimeout.String(),
		"shutdown_mode",
		"fast_restart_with_database_resume",
	)

	go func() {
		serverErrCh <- srv.ListenAndServe()
	}()

	// 7. main是全系统唯一系统信号监听入口。
	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		if err != nil &&
			err != http.ErrServerClosed {
			logger.Fatal(
				"服务运行异常",
				"module", "main",
				"port", cfg.Port,
				"error", err,
			)
		}

		log.Info("HTTP服务器已停止监听")
		return

	case sig := <-signalCh:
		log.Info(
			"收到系统信号，开始快速重启收敛",
			"signal", sig.String(),
			"timeout",
			fastShutdownTimeout.String(),
		)
	}

	// 8. 创建短关停期限。
	shutdownCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			fastShutdownTimeout,
		)
	defer cancel()

	// 9. 后台任务进入draining。
	//
	// 本动作会：
	//   - 拒绝新的后台长任务；
	//   - 执行已登记任务的onDrain钩子；
	//   - 批量生成停止继续派发尚未开始的页面；
	//   - 全自动装配停止继续派发新的HTML和图片任务。
	backgroundSnapshot :=
		services.BeginGlobalBackgroundDraining()

	log.Info(
		"后台任务进入快速收敛状态",
		"active_tasks",
		len(backgroundSnapshot),
	)

	for _, task := range backgroundSnapshot {
		log.Info(
			"快速重启时存在后台任务",
			"task_key", task.Key,
			"task_type", task.TaskType,
			"resource_id", task.ResourceID,
			"class", string(task.Class),
			"elapsed_ms", task.ElapsedMS,
		)
	}

	// 10. 主动关闭SSE，使浏览器刷新后连接到新进程并重新拉取数据库状态。
	sseSummary :=
		services.BeginGlobalSSEDraining()

	log.Info(
		"SSE连接已关闭，客户端可刷新恢复",
		"lesson_plan_closed",
		sseSummary.LessonPlan,
		"courseware_closed",
		sseSummary.Courseware,
		"pipeline_closed",
		sseSummary.Pipeline,
		"knowledge_base_closed",
		sseSummary.KnowledgeBase,
		"total_closed",
		sseSummary.Total,
	)

	// 11. 关闭语音WebSocket。
	speechClosed :=
		services.BeginGlobalSpeechDraining()

	log.Info(
		"语音WebSocket已关闭",
		"total_closed",
		speechClosed,
	)

	// 12. 并行停止Pipeline Engine。
	engineErrCh := make(chan error, 1)

	go func() {
		engineErrCh <- services.ShutdownDefaultEngine(
			shutdownCtx,
		)
	}()

	// 13. 停止接受新HTTP连接并短暂等待活动请求。
	httpShutdownErr :=
		srv.Shutdown(shutdownCtx)

	if httpShutdownErr != nil {
		log.Warn(
			"短关停期限内HTTP请求未全部结束，执行强制关闭",
			"error",
			httpShutdownErr,
		)

		if closeErr :=
			srv.Close(); closeErr != nil {
			log.Warn(
				"强制关闭HTTP服务器返回错误",
				"error",
				closeErr,
			)
		}
	} else {
		log.Info("HTTP请求已在快速期限内收敛")
	}

	// 14. 等待Engine，最多使用同一个8秒期限。
	select {
	case engineShutdownErr := <-engineErrCh:
		if engineShutdownErr != nil {
			log.Warn(
				"Engine未在快速期限内完全排空",
				"error",
				engineShutdownErr,
			)
		} else {
			log.Info("Engine已停止")
		}

	case <-shutdownCtx.Done():
		log.Warn(
			"Engine等待达到快速关停期限",
			"error",
			shutdownCtx.Err(),
		)
	}

	// 15. 不再等待后台AI任务。
	//
	// 已经落库的页面和媒体事实保留；
	// 尚未落库的同步AI当前页由新进程重新生成。
	remaining :=
		services.GetGlobalBackgroundTaskSummary()

	if remaining.Active > 0 {
		log.Warn(
			"快速重启不等待后台AI任务，未落库工作将在新进程断点续生",
			"active_tasks",
			remaining.Active,
			"critical_tasks",
			remaining.Critical,
			"best_effort_tasks",
			remaining.BestEffort,
		)

		for _, task := range remaining.Tasks {
			log.Warn(
				"退出时仍存在后台任务",
				"task_key", task.Key,
				"task_type", task.TaskType,
				"resource_id", task.ResourceID,
				"class", string(task.Class),
				"elapsed_ms", task.ElapsedMS,
			)
		}
	}

	log.Info(
		"快速关停完成，systemd可启动新版本",
		"resume_source",
		"database",
	)
}
