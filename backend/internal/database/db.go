package database

// 数据库连接池初始化与管理
// Phase8日志升级：使用 logger 包替换原 log.Println/log.Fatalf，输出结构化JSON日志
// 审查修复D-01：增加连接池参数调优（MaxConns/MinConns/MaxConnIdleTime/HealthCheckPeriod）
// 审查修复D-02：对密码做URL转义，避免特殊字符导致连接失败
// 审查修复C-03：删除config.GetDSN死代码，统一使用URI格式

import (
	"context"
	"fmt"
	"net/url"
	"time"

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
	// 审查修复D-02：对密码做URL编码，防止密码含@:/等特殊字符时URI解析出错
	escapedPassword := url.QueryEscape(cfg.DBPassword)

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, escapedPassword,
		cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	// 审查修复D-01：解析DSN后设置连接池参数，避免高并发时连接不足
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("数据库DSN解析失败",
			"module", "database",
			"error", err,
		)
	}

	// 连接池调优参数（适配并发引擎+SSE+多AI场景调用的负载特征）
	poolConfig.MaxConns = 30                          // 最大连接数（默认4太少，系统有并发Pipeline引擎）
	poolConfig.MinConns = 5                           // 最小空闲连接（避免冷启动延迟）
	poolConfig.MaxConnLifetime = 60 * time.Minute     // 连接最大存活时间（防止长连接累积问题）
	poolConfig.MaxConnIdleTime = 30 * time.Minute     // 空闲连接最大存活时间
	poolConfig.HealthCheckPeriod = 60 * time.Second   // 健康检查间隔

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
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
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
	)
}

// Close 关闭数据库连接池（在 main 中 defer 调用）
func Close() {
	if DB != nil {
		DB.Close()
		log.Info("数据库连接池已关闭")
	}
}

// Ping 检查数据库连接是否正常（供健康检查接口使用）
// 返回 error 表示连接异常，nil 表示正常
func Ping(ctx context.Context) error {
	if DB == nil {
		return fmt.Errorf("数据库连接池未初始化")
	}
	return DB.Ping(ctx)
}
