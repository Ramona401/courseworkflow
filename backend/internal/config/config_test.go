// Package config 配置模块单元测试。
//
// 测试范围：
//   - AppVersion格式和当前版本；
//   - 字符串、整数和布尔环境变量读取；
//   - 教学智能体两个功能开关；
//   - 教师端开关关闭时公开开关强制收敛；
//   - AES密钥读取；
//   - Config结构体关键字段。
package config

import (
	"os"
	"strings"
	"testing"
)

// ==================== AppVersion ====================

func TestAppVersionFormat(t *testing.T) {
	if AppVersion == "" {
		t.Fatal("AppVersion不应为空字符串")
	}

	parts := strings.Split(AppVersion, ".")
	if len(parts) != 3 {
		t.Fatalf(
			"AppVersion应为X.Y.Z格式，实际为%s",
			AppVersion,
		)
	}

	for index, part := range parts {
		if part == "" {
			t.Fatalf(
				"AppVersion第%d段不能为空",
				index+1,
			)
		}

		for _, character := range part {
			if character < '0' ||
				character > '9' {
				t.Fatalf(
					"AppVersion段%q包含非数字字符%c",
					part,
					character,
				)
			}
		}
	}
}

func TestAppVersionIsCurrentVersion(t *testing.T) {
	const expected = "0.43.0"

	if AppVersion != expected {
		t.Fatalf(
			"AppVersion不一致：expected=%s actual=%s",
			expected,
			AppVersion,
		)
	}
}

// ==================== getEnv ====================

func TestGetEnvExistingValue(t *testing.T) {
	const (
		key      = "TEDNA_TEST_ENV_EXISTING"
		expected = "test-value"
	)

	t.Setenv(key, expected)

	if actual := getEnv(key, "fallback"); actual != expected {
		t.Fatalf(
			"环境变量读取错误：expected=%q actual=%q",
			expected,
			actual,
		)
	}
}

func TestGetEnvMissingOrEmptyUsesDefault(t *testing.T) {
	const key = "TEDNA_TEST_ENV_DEFAULT"

	_ = os.Unsetenv(key)

	if actual := getEnv(key, "fallback"); actual != "fallback" {
		t.Fatalf(
			"缺失变量没有使用默认值：actual=%q",
			actual,
		)
	}

	t.Setenv(key, "")

	if actual := getEnv(key, "fallback"); actual != "fallback" {
		t.Fatalf(
			"空变量没有使用默认值：actual=%q",
			actual,
		)
	}
}

// ==================== GetIntEnv ====================

func TestGetIntEnv(t *testing.T) {
	const key = "TEDNA_TEST_INT_ENV"

	cases := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{
			name:         "正整数",
			value:        "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "零",
			value:        "0",
			defaultValue: 10,
			expected:     0,
		},
		{
			name:         "负整数",
			value:        "-5",
			defaultValue: 10,
			expected:     -5,
		},
		{
			name:         "非数字",
			value:        "invalid",
			defaultValue: 99,
			expected:     99,
		},
		{
			name:         "浮点数",
			value:        "3.14",
			defaultValue: 88,
			expected:     88,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				t.Setenv(key, item.value)

				actual :=
					GetIntEnv(
						key,
						item.defaultValue,
					)

				if actual != item.expected {
					t.Fatalf(
						"整数环境变量错误：expected=%d actual=%d",
						item.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestGetIntEnvMissingUsesDefault(t *testing.T) {
	const key = "TEDNA_TEST_INT_ENV_MISSING"

	_ = os.Unsetenv(key)

	if actual := GetIntEnv(key, 55); actual != 55 {
		t.Fatalf(
			"缺失整数变量没有使用默认值：actual=%d",
			actual,
		)
	}
}

// ==================== GetBoolEnv ====================

func TestGetBoolEnvAcceptedValues(t *testing.T) {
	const key = "TEDNA_TEST_BOOL_ENV"

	cases := []struct {
		value    string
		expected bool
	}{
		{value: "true", expected: true},
		{value: "TRUE", expected: true},
		{value: "1", expected: true},
		{value: "yes", expected: true},
		{value: "on", expected: true},
		{value: "false", expected: false},
		{value: "FALSE", expected: false},
		{value: "0", expected: false},
		{value: "no", expected: false},
		{value: "off", expected: false},
	}

	for _, item := range cases {
		t.Run(
			item.value,
			func(t *testing.T) {
				t.Setenv(key, item.value)

				actual :=
					GetBoolEnv(
						key,
						!item.expected,
					)

				if actual != item.expected {
					t.Fatalf(
						"布尔环境变量解析错误：value=%q expected=%t actual=%t",
						item.value,
						item.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestGetBoolEnvMissingUsesDefault(t *testing.T) {
	const key = "TEDNA_TEST_BOOL_ENV_MISSING"

	_ = os.Unsetenv(key)

	if !GetBoolEnv(key, true) {
		t.Fatal("缺失布尔变量没有返回true默认值")
	}

	if GetBoolEnv(key, false) {
		t.Fatal("缺失布尔变量没有返回false默认值")
	}
}

func TestGetBoolEnvInvalidUsesDefault(t *testing.T) {
	const key = "TEDNA_TEST_BOOL_ENV_INVALID"

	t.Setenv(key, "not-a-boolean")

	if !GetBoolEnv(key, true) {
		t.Fatal("非法布尔变量没有返回true默认值")
	}

	if GetBoolEnv(key, false) {
		t.Fatal("非法布尔变量没有返回false默认值")
	}
}

// ==================== 教学智能体开关 ====================

func setRequiredLoadEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("DB_PASSWORD", "test-database-password")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv(
		"AES_KEY",
		"test-aes-key-32bytes-long-000000",
	)
	t.Setenv(
		"ASSISTANT_RUNTIME_PRIVACY_SALT",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	t.Setenv(
		"ASSISTANT_RUNTIME_TOKEN_TTL_MINUTES",
		"15",
	)
}

func TestLoadCoursewareAssistantFlags(t *testing.T) {
	setRequiredLoadEnvironment(t)

	t.Setenv(
		"COURSEWARE_ASSISTANT_ENABLED",
		"true",
	)
	t.Setenv(
		"COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED",
		"false",
	)

	cfg := Load()

	if !cfg.CoursewareAssistantEnabled {
		t.Fatal("教师端教学智能体开关应为true")
	}

	if cfg.CoursewareAssistantPublicRuntimeEnabled {
		t.Fatal("公开运行开关应为false")
	}
}

func TestLoadForcesPublicRuntimeOffWhenTeacherFeatureDisabled(
	t *testing.T,
) {
	setRequiredLoadEnvironment(t)

	t.Setenv(
		"COURSEWARE_ASSISTANT_ENABLED",
		"false",
	)
	t.Setenv(
		"COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED",
		"true",
	)

	cfg := Load()

	if cfg.CoursewareAssistantEnabled {
		t.Fatal("教师端教学智能体开关应为false")
	}

	if cfg.CoursewareAssistantPublicRuntimeEnabled {
		t.Fatal(
			"教师端开关关闭时公开运行开关必须强制收敛为false",
		)
	}
}

// ==================== 其他配置 ====================

func TestGetAESKey(t *testing.T) {
	cfg := &Config{
		AESKey: "test-aes-key-32bytes-long-xxxxx",
	}

	if actual := cfg.GetAESKey(); actual != cfg.AESKey {
		t.Fatalf(
			"GetAESKey返回错误：expected=%q actual=%q",
			cfg.AESKey,
			actual,
		)
	}
}

func TestConfigKeyFields(t *testing.T) {
	cfg := Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "test-user",
		DBPassword: "test-password",
		DBName:     "test-database",
		Port:       "9090",
		GinMode:    "debug",

		CoursewareAssistantEnabled:              true,
		CoursewareAssistantPublicRuntimeEnabled: false,

		JWTSecret:      "test-jwt-secret",
		AESKey:         "test-aes-key",
		AIAPIBaseURL:   "https://api.example.com",
		AIAPIKey:       "test-api-key",
		AIDefaultModel: "test-model",
	}

	if cfg.DBHost != "localhost" ||
		cfg.Port != "9090" ||
		cfg.AIDefaultModel != "test-model" ||
		!cfg.CoursewareAssistantEnabled ||
		cfg.CoursewareAssistantPublicRuntimeEnabled {
		t.Fatalf(
			"Config关键字段赋值错误：cfg=%+v",
			cfg,
		)
	}
}
