package main

// TE-DNA 2.0 服务入口
//
// 本文件统一负责整个Go进程的生命周期：
//   1. 初始化配置、数据库和业务路由；
//   2. 启动HTTP服务；
//   3. 作为全系统唯一的SIGTERM与SIGINT监听入口；
//   4. 部署或人工停止时进入统一draining状态；
//   5. 关闭SSE长连接；
//   6. 同时等待HTTP、Pipeline Engine和已登记后台任务排空；
//   7. 全部关闭链路结束后才释放数据库连接池并退出。
//
// 当前统一关闭顺序：
//
//	收到SIGTERM
//	  → BackgroundTaskTracker进入draining，拒绝新的外部长任务
//	  → 触发已有后台任务的onDrain钩子
//	  → 关闭教案、课件、Pipeline、知识库四类SSE连接
//	  → http.Server.Shutdown停止接受新连接并等待活动请求
//	  → Engine停止接收新Pipeline任务并排空队列
//	  → BackgroundTaskTracker等待异步AI任务完成
//	  → main返回，最后执行database.Close
//
// 超时口径：
//   - 应用内部统一排空上限为12分钟；
//   - systemd TimeoutStopSec为13分钟；
//   - 应用获得完整12分钟处理业务，再预留1分钟供进程和systemd收尾。

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
	"tedna/internal/routes"
	"tedna/internal/services"
)

// 模块日志。
var log = logger.WithModule("main")

const (
	// gracefulShutdownTimeout 是应用内部统一排空的最长等待时间。
	//
	// 当前HTTP WriteTimeout为600秒，单次同步AI请求理论上在10分钟内结束；
	// 额外预留2分钟用于模型返回后的HTML校验、版本保存、数据库写入和任务收尾。
	gracefulShutdownTimeout = 12 * time.Minute
)

func main() {
	// 1. 加载配置。
	cfg := config.Load()

	// 2. 初始化数据库。
	//
	// 数据库连接池必须晚于HTTP、Engine和后台任务关闭。
	// main真正返回时才执行本defer。
	database.Init(cfg)
	defer database.Close()

	// 3. 注册业务路由并创建生产Engine。
	mux := routes.Setup(cfg)

	// 4. 创建HTTP服务器。
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      600 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 5. 在受控goroutine中启动HTTP服务。
	serverErrCh := make(chan error, 1)

	log.Info("TE-DNA 2.0 服务启动",
		"port", cfg.Port,
		"version", config.AppVersion,
		"read_header_timeout", "30s",
		"write_timeout", "600s",
		"graceful_shutdown_timeout", gracefulShutdownTimeout.String(),
	)

	go func() {
		serverErrCh <- srv.ListenAndServe()
	}()

	// 6. main是全系统唯一系统信号监听入口。
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		// 非主动关闭导致的ListenAndServe退出属于运行故障。
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务运行异常",
				"module", "main",
				"port", cfg.Port,
				"error", err,
			)
		}

		log.Info("HTTP服务器已停止监听")
		return

	case sig := <-signalCh:
		log.Info("收到系统信号，开始统一排空服务",
			"signal", sig.String(),
			"timeout", gracefulShutdownTimeout.String(),
		)
	}

	// 7. 创建HTTP、Engine和后台任务共用的统一关闭期限。
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		gracefulShutdownTimeout,
	)
	defer cancel()

	// 8. 后台任务进入draining。
	//
	// 新的外部长任务从此应被业务Handler拒绝；
	// 已登记任务的onDrain钩子会在此处执行，例如停止继续派发新的课件页面。
	backgroundSnapshot := services.BeginGlobalBackgroundDraining()

	log.Info("后台任务已进入排空状态",
		"active_tasks", len(backgroundSnapshot),
	)

	for _, task := range backgroundSnapshot {
		log.Info("排空开始时存在后台任务",
			"task_key", task.Key,
			"task_type", task.TaskType,
			"resource_id", task.ResourceID,
			"class", string(task.Class),
			"elapsed_ms", task.ElapsedMS,
		)
	}

	// 9. 主动关闭全部SSE长连接。
	//
	// 各SSE Handler在读取到channel关闭后会自然返回，
	// 避免空闲浏览器页面长期占住http.Server.Shutdown。
	sseSummary := services.BeginGlobalSSEDraining()

	log.Info("SSE长连接已进入排空状态",
		"lesson_plan_closed", sseSummary.LessonPlan,
		"courseware_closed", sseSummary.Courseware,
		"pipeline_closed", sseSummary.Pipeline,
		"knowledge_base_closed", sseSummary.KnowledgeBase,
		"total_closed", sseSummary.Total,
	)

	// 10. 并行关闭Pipeline Engine。
	engineErrCh := make(chan error, 1)

	go func() {
		engineErrCh <- services.ShutdownDefaultEngine(shutdownCtx)
	}()

	// 11. 并行等待全部已登记后台任务。
	backgroundErrCh := make(chan error, 1)

	go func() {
		backgroundErrCh <- services.WaitGlobalBackgroundTasks(shutdownCtx)
	}()

	// 12. 停止接受新HTTP连接并等待活动HTTP请求。
	//
	// 同步单页微调、导航栏微调、单页重新生成、文件上传等请求
	// 会继续执行，直到自然返回或统一12分钟期限到达。
	httpShutdownErr := srv.Shutdown(shutdownCtx)
	if httpShutdownErr != nil {
		log.Error("HTTP服务器优雅关闭未完成",
			"error", httpShutdownErr,
		)

		// 统一期限已到时强制关闭残余HTTP连接，
		// 避免旧进程永久卡住导致新版本无法启动。
		if closeErr := srv.Close(); closeErr != nil {
			log.Error("强制关闭HTTP服务器失败",
				"error", closeErr,
			)
		}
	} else {
		log.Info("HTTP活动请求已排空")
	}

	// 13. 等待Pipeline Engine排空结果。
	engineShutdownErr := <-engineErrCh
	if engineShutdownErr != nil {
		log.Error("Engine任务排空未完成",
			"error", engineShutdownErr,
		)
	} else {
		log.Info("Engine任务已排空")
	}

	// 14. 等待后台任务排空结果。
	backgroundShutdownErr := <-backgroundErrCh
	if backgroundShutdownErr != nil {
		summary := services.GetGlobalBackgroundTaskSummary()

		log.Error("后台任务排空未完成",
			"error", backgroundShutdownErr,
			"active_tasks", summary.Active,
			"critical_tasks", summary.Critical,
			"best_effort_tasks", summary.BestEffort,
		)

		for _, task := range summary.Tasks {
			log.Error("关闭期限到达时仍在运行的后台任务",
				"task_key", task.Key,
				"task_type", task.TaskType,
				"resource_id", task.ResourceID,
				"class", string(task.Class),
				"elapsed_ms", task.ElapsedMS,
			)
		}
	} else {
		log.Info("后台任务已排空")
	}

	// 15. main到这里才允许返回。
	//
	// 返回后才会执行database.Close，保证数据库连接池晚于所有受控关闭链路。
	log.Info("服务器已完全关闭")
}
