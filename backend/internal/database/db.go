package database

// 数据库连接池初始化
// Phase8日志升级：使用 logger 包替换原 log.Println/log.Fatalf，输出结构化JSON日志

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"tedna/internal/config"
	"tedna/internal/logger"
)

// DB 全局数据库连接池（所有 repository 层共用）
var DB *pgxpool.Pool

// 模块日志：所有数据库相关日志自动携带 module=database 字段
var log = logger.WithModule("database")

// Init 初始化数据库连接池
// 连接失败或 Ping 失败时直接 Fatal（属于启动阶段关键错误，不可恢复）
func Init(cfg *config.Config) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword,
		cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		// Fatal：数据库连接失败无法继续启动
		logger.Fatal("数据库连接失败",
			"module", "database",
			"host", cfg.DBHost,
			"port", cfg.DBPort,
			"dbname", cfg.DBName,
			"error", err,
		)
	}

	// 测试连接是否可用
	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("数据库 Ping 失败",
			"module", "database",
			"host", cfg.DBHost,
			"error", err,
		)
	}

	DB = pool
	log.Info("数据库连接成功",
		"host", cfg.DBHost,
		"port", cfg.DBPort,
		"dbname", cfg.DBName,
	)
}

// Close 关闭数据库连接池（在 main 中 defer 调用）
func Close() {
	if DB != nil {
		DB.Close()
		log.Info("数据库连接池已关闭")
	}
}
