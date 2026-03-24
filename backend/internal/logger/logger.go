package logger

// ==================== 统一结构化日志模块 ====================
// 使用 Go 1.21 内置 log/slog，零额外依赖
//
// 输出格式（生产环境）：
//   {"time":"2026-03-24T15:04:05.000+08:00","level":"INFO","msg":"用户登录成功","module":"auth","username":"admin","user_id":"xxx"}
//
// 日志级别：
//   DEBUG  — 详细调试信息（生产建议关闭）
//   INFO   — 正常业务事件（登录/创建/启动等）
//   WARN   — 可恢复的异常（队列满/降级处理/更新失败但不影响主流程）
//   ERROR  — 需要关注的错误（数据库错误/AI调用失败/token生成失败）
//
// 使用示例：
//   logger.Info("用户登录成功", "module", "auth", "username", "admin", "user_id", "xxx")
//   logger.Warn("队列已满，任务被拒绝", "module", "engine", "pipeline_id", id)
//   logger.Error("数据库查询失败", "module", "user", "error", err)

import (
	"log/slog"
	"os"
	"time"
)

// Logger 全局 slog.Logger 实例（使用结构化 JSON Handler）
var Logger *slog.Logger

// 日志级别常量（供 SetLevel 使用）
var currentLevel = &slog.LevelVar{}

func init() {
	// 默认 INFO 级别
	currentLevel.Set(slog.LevelInfo)

	// 生产环境使用 JSON Handler，便于日志采集（ELK/Loki等）
	// ReplaceAttr 统一处理时间格式和字段名
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: currentLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 统一时间字段格式为 RFC3339Nano，并附加东八区时区
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				loc, err := time.LoadLocation("Asia/Shanghai")
				if err != nil {
					loc = time.FixedZone("CST", 8*3600)
				}
				a.Value = slog.StringValue(t.In(loc).Format("2006-01-02T15:04:05.000-07:00"))
			}
			// 统一级别字段：大写英文（INFO/WARN/ERROR/DEBUG）
			if a.Key == slog.LevelKey {
				level, ok := a.Value.Any().(slog.Level)
				if ok {
					a.Value = slog.StringValue(level.String())
				}
			}
			return a
		},
	})

	Logger = slog.New(handler)

	// 同时设为 slog 全局默认，兼容第三方包使用 slog.Info() 等调用
	slog.SetDefault(Logger)
}

// SetLevel 动态调整日志级别（DEBUG/INFO/WARN/ERROR）
// 可用于运行时切换调试模式，无需重启服务
func SetLevel(level slog.Level) {
	currentLevel.Set(level)
}

// ==================== 便捷函数（带 module 字段的快捷调用）====================
// 所有日志自动携带 module 字段，便于按模块过滤

// Info 记录正常业务事件
// 示例：logger.Info("用户登录成功", "module","auth", "username","admin")
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn 记录可恢复的警告（非致命，但需要关注）
// 示例：logger.Warn("AI队列已满", "module","engine", "pipeline_id",id)
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error 记录需要处理的错误
// 示例：logger.Error("数据库查询失败", "module","user", "error",err)
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// Debug 记录调试详情（生产环境默认不输出）
// 示例：logger.Debug("SSE广播", "module","sse", "pipeline_id",id, "count",n)
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Fatal 记录致命错误后退出进程（仅在启动阶段使用）
// 等效于 log.Fatalf，输出结构化日志后调用 os.Exit(1)
func Fatal(msg string, args ...any) {
	Logger.Error(msg, args...)
	os.Exit(1)
}

// WithModule 返回携带固定 module 字段的子 Logger
// 推荐在每个包内创建模块 Logger：
//   var log = logger.WithModule("pipeline")
//   log.Info("Pipeline启动", "id", pipelineID)
func WithModule(module string) *slog.Logger {
	return Logger.With("module", module)
}
