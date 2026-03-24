package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// AppVersion 应用版本号
// 修复R-03：从硬编码迁移到配置，healthHandler和其他需要版本号的地方统一读取此常量
// 发版时只需修改此处一个位置，避免多处硬编码漏改
const AppVersion = "0.30.0"

// Config 全局配置结构体
type Config struct {
	// 数据库配置
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// 服务器配置
	Port    string
	GinMode string

	// JWT 配置
	JWTSecret string

	// AES 加密密钥
	AESKey string

	// AI API 配置
	AIAPIBaseURL   string
	AIAPIKey       string
	AIDefaultModel string
}

// Load 从环境变量加载配置
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用系统环境变量")
	}

	cfg := &Config{
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "tedna_user"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "tedna"),
		Port:           getEnv("PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "release"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		AESKey:         getEnv("AES_KEY", ""),
		AIAPIBaseURL:   getEnv("AI_API_BASE_URL", ""),
		AIAPIKey:       getEnv("AI_API_KEY", ""),
		AIDefaultModel: getEnv("AI_DEFAULT_MODEL", "anthropic/claude-sonnet-4-5"),
	}

	// 验证必要配置
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET 未配置")
	}
	if cfg.DBPassword == "" {
		log.Fatal("DB_PASSWORD 未配置")
	}

	return cfg
}

// getEnv 获取环境变量，不存在则返回默认值
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetIntEnv 获取整型环境变量
func GetIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

// GetDSN 返回 PostgreSQL 连接字符串
func (c *Config) GetDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=disable TimeZone=Asia/Shanghai"
}
