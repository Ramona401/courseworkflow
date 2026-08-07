package integration

// test_config_environment_test.go
//
// 验证完整路由中的历史config.Load调用能够读取隔离测试配置，
// 并且绝不把数据库名切换到生产库。

import (
	"testing"

	"tedna/internal/config"
)

// TestBindTestConfigEnvironmentSupportsConfigLoad 验证二次配置加载。
func TestBindTestConfigEnvironmentSupportsConfigLoad(
	t *testing.T,
) {
	cfg := testConfig()

	bindTestConfigEnvironment(
		t,
		cfg,
	)

	loaded := config.Load()

	if loaded == nil {
		t.Fatal(
			"config.Load返回nil",
		)
	}

	if loaded.DBName != "tedna_test" {
		t.Fatalf(
			"二次配置加载使用了非测试数据库：%s",
			loaded.DBName,
		)
	}

	if loaded.DBHost != cfg.DBHost ||
		loaded.DBPort != cfg.DBPort ||
		loaded.DBUser != cfg.DBUser ||
		loaded.DBPassword != cfg.DBPassword {
		t.Fatalf(
			"二次配置加载的测试数据库连接不一致: loaded=%s:%s/%s user=%s",
			loaded.DBHost,
			loaded.DBPort,
			loaded.DBName,
			loaded.DBUser,
		)
	}

	if loaded.JWTSecret != cfg.JWTSecret {
		t.Fatal(
			"二次配置加载没有使用测试JWT_SECRET",
		)
	}

	if loaded.GetAESKey() != cfg.GetAESKey() {
		t.Fatal(
			"二次配置加载没有使用测试AES密钥",
		)
	}

	if loaded.AssistantRuntimePrivacySalt !=
		cfg.AssistantRuntimePrivacySalt {
		t.Fatal(
			"二次配置加载没有使用测试运行隐私盐",
		)
	}

	if loaded.AssistantRuntimeTokenTTLMinutes != 15 {
		t.Fatalf(
			"二次配置加载的运行令牌有效期错误：%d",
			loaded.AssistantRuntimeTokenTTLMinutes,
		)
	}

	if loaded.AIAPIBaseURL !=
		cfg.AIAPIBaseURL ||
		loaded.AIAPIKey !=
			cfg.AIAPIKey ||
		loaded.AIDefaultModel !=
			cfg.AIDefaultModel {
		t.Fatal(
			"二次配置加载没有保持测试AI占位配置",
		)
	}
}
