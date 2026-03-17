package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/routes"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	database.Init(cfg)
	defer database.Close()

	// 注册路由
	mux := routes.Setup(cfg)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 600 * time.Second, // AI 调用可能很长
		IdleTimeout:  120 * time.Second,
	}

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("正在关闭服务器...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("TE-DNA 2.0 服务启动，端口: %s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败: %v", err)
	}
	log.Println("服务器已关闭")
}
