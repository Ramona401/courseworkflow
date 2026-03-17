package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"tedna/internal/config"
)

// DB 全局数据库连接池
var DB *pgxpool.Pool

// Init 初始化数据库连接池
func Init(cfg *config.Config) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword,
		cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 测试连接
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("数据库 Ping 失败: %v", err)
	}

	DB = pool
	log.Println("数据库连接成功")
}

// Close 关闭数据库连接池
func Close() {
	if DB != nil {
		DB.Close()
	}
}
