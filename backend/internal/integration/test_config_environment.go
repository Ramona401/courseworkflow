package integration

// test_config_environment.go
//
// 完整生产路由中有少量历史Handler构造器仍在内部调用config.Load。
// 集成测试必须在构造路由前，把已经验证过的tedna_test配置写入当前
// 测试进程环境，避免这些构造器：
//   - 因找不到.env而退出测试进程；
//   - 误读服务器生产环境配置；
//   - 使用生产数据库名、生产JWT密钥或生产AI通道。
//
// testing.T.Setenv会在当前测试结束时自动恢复原环境。
// 本辅助只能接收DBName严格为tedna_test的配置。

import (
	"strconv"
	"testing"

	"tedna/internal/config"
)

// bindTestConfigEnvironment 把隔离测试配置绑定到当前测试进程。
func bindTestConfigEnvironment(
	t *testing.T,
	cfg *config.Config,
) {
	t.Helper()

	if cfg == nil {
		t.Fatal(
			"测试环境绑定失败：配置为nil",
		)
	}

	if cfg.DBName != "tedna_test" {
		t.Fatalf(
			"测试环境绑定拒绝非测试数据库：%s",
			cfg.DBName,
		)
	}

	if cfg.JWTSecret == "" {
		t.Fatal(
			"测试环境绑定失败：JWT_SECRET为空",
		)
	}

	if cfg.AESKey == "" {
		t.Fatal(
			"测试环境绑定失败：AES_KEY为空",
		)
	}

	if cfg.AssistantRuntimePrivacySalt == "" {
		t.Fatal(
			"测试环境绑定失败：运行隐私盐为空",
		)
	}

	values := map[string]string{
		// PostgreSQL测试连接。
		"DB_HOST":     cfg.DBHost,
		"DB_PORT":     cfg.DBPort,
		"DB_USER":     cfg.DBUser,
		"DB_PASSWORD": cfg.DBPassword,
		"DB_NAME":     cfg.DBName,

		// 部分历史部署环境可能使用PostgreSQL标准别名。
		// 同时绑定不会改变Config正式读取的DB_*单一真相源。
		"PGHOST":     cfg.DBHost,
		"PGPORT":     cfg.DBPort,
		"PGUSER":     cfg.DBUser,
		"PGPASSWORD": cfg.DBPassword,
		"PGDATABASE": cfg.DBName,

		// HTTP和运行模式。
		"PORT":               cfg.Port,
		"SERVER_PORT":        cfg.Port,
		"GIN_MODE":           cfg.GinMode,
		"APP_ENV":            "test",
		"DISABLE_SCHEDULERS": "true",

		// 认证和加密。
		"JWT_SECRET": cfg.JWTSecret,
		"AES_KEY":    cfg.AESKey,

		// 兼容旧环境命名，但业务读取仍以AES_KEY为准。
		"AES_ENCRYPTION_KEY": cfg.AESKey,

		// AI仅提供不可访问的本地占位配置。
		// 本单元测试不会发起任何AI请求。
		"AI_API_BASE_URL": cfg.AIAPIBaseURL,
		"AI_API_KEY":      cfg.AIAPIKey,
		"AI_DEFAULT_MODEL": cfg.AIDefaultModel,

		// 教学智能体公开运行安全配置。
		"ASSISTANT_RUNTIME_PRIVACY_SALT":
			cfg.AssistantRuntimePrivacySalt,
		"ASSISTANT_RUNTIME_TOKEN_TTL_MINUTES":
			strconv.Itoa(
				cfg.AssistantRuntimeTokenTTLMinutes,
			),

		// 降低测试构造时的后台并发规模。
		"COURSEWARE_GEN_CONCURRENCY":
			strconv.Itoa(
				cfg.CoursewareGenConcurrency,
			),
		"COURSEWARE_ASSEMBLY_IMG_CONCURRENCY":
			strconv.Itoa(
				cfg.CoursewareAssemblyImgConcurrency,
			),
	}

	for key, value := range values {
		t.Setenv(
			key,
			value,
		)
	}
}
