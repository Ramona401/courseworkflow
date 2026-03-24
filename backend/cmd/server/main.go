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
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 600 * time.Second, // AI 调用可能很长，与Nginx proxy_read_timeout对齐
		IdleTimeout:  120 * time.Second,
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
		"read_timeout", "30s",
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
