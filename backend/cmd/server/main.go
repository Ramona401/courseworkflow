package main

// TE-DNA 2.0 服务入口
// Phase8日志升级：使用 logger 包统一结构化日志，替换原 log.Printf/log.Println
// 启动日志输出示例：
//   {"time":"2026-03-24T15:04:05.000+08:00","level":"INFO","msg":"TE-DNA 2.0 服务启动","module":"main","port":"8080","version":"0.30.0"}

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
)

// 模块日志
var log = logger.WithModule("main")

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化数据库（失败则Fatal退出）
	database.Init(cfg)
	defer database.Close()

	// 3. 注册路由（含Engine创建+夜间任务启动+优雅关闭监听）
	mux := routes.Setup(cfg)

	// 4. 创建 HTTP 服务器
	//
	// v0.43.1 修复（大文件上传超时）:
	//   原 ReadTimeout=30s 限制的是"读完整个请求体"的总时长,
	//   导致 22MB+ 的 PPT/Word 上传在网络稍慢时 30 秒内读不完被服务端直接掐断,
	//   表现为前端"上传解析中…"卡死、Nginx access log 记 499(客户端超时断开)。
	//   改用 ReadHeaderTimeout=30s —— 只限制"读完 HTTP 请求头"的时长(防慢连接攻击),
	//   请求体(大文件)的读取不再受 30 秒上限约束,大上传可正常传完。
	//   这是 Go 官方推荐的大上传处理方式,既解除上传超时又保留对慢攻击的防护。
	//   WriteTimeout/IdleTimeout 维持不变。
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,  // 仅限读请求头,大文件请求体上传不受限
		WriteTimeout:      600 * time.Second, // AI 调用可能很长，与Nginx proxy_read_timeout对齐
		IdleTimeout:       120 * time.Second,
	}

	// 5. 优雅关闭：监听 SIGTERM/SIGINT
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		log.Info("收到系统信号，开始关闭HTTP服务器",
			"signal", sig.String(),
		)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("HTTP服务器关闭异常", "error", err)
		}
		log.Info("HTTP服务器已关闭")
	}()

	// 6. 启动服务
	log.Info("TE-DNA 2.0 服务启动",
		"port", cfg.Port,
		"version", config.AppVersion,
		"read_header_timeout", "30s",
		"write_timeout", "600s",
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("服务启动失败",
			"module", "main",
			"port", cfg.Port,
			"error", err,
		)
	}

	log.Info("服务器已完全关闭")
}
