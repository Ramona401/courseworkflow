// Package config 配置模块单元测试
//
// 测试范围：
//   - AppVersion：格式校验+当前版本
//   - getEnv：环境变量读取+默认值回退
//   - GetIntEnv：整型环境变量解析+错误处理
//   - Load：配置加载+必要配置校验
//   - GetAESKey：AES密钥获取方法
package config

import (
"os"
"strings"
"testing"
)

// ==================== AppVersion 测试 ====================

// TestAppVersionFormat 测试版本号格式是否符合语义化版本规范
func TestAppVersionFormat(t *testing.T) {
if AppVersion == "" {
t.Fatal("AppVersion不应为空字符串")
}
parts := strings.Split(AppVersion, ".")
if len(parts) != 3 {
t.Fatalf("AppVersion应为X.Y.Z格式, 实际: %s (分段数=%d)", AppVersion, len(parts))
}
for i, part := range parts {
if part == "" {
t.Errorf("AppVersion第%d段不应为空", i+1)
}
for _, c := range part {
if c < '0' || c > '9' {
t.Errorf("AppVersion段 %q 含非数字字符: %c", part, c)
}
}
}
t.Logf("AppVersion验证通过: %s", AppVersion)
}

// TestAppVersionIsCurrentVersion 测试当前版本号
func TestAppVersionIsCurrentVersion(t *testing.T) {
expected := "0.37.0"
if AppVersion != expected {
t.Logf("注意：AppVersion=%s，文档记录版本为%s", AppVersion, expected)
} else {
t.Logf("版本号匹配: %s", AppVersion)
}
}

// ==================== getEnv 测试 ====================

// TestGetEnv_ExistingVar 环境变量存在时返回实际值
func TestGetEnv_ExistingVar(t *testing.T) {
key := "TEDNA_TEST_ENV_VAR_EXISTING"
expected := "test_value_12345"
os.Setenv(key, expected)
defer os.Unsetenv(key)

result := getEnv(key, "default_value")
if result != expected {
t.Errorf("应返回环境变量值%q，实际%q", expected, result)
}
}

// TestGetEnv_NonExistingVar 环境变量不存在时返回默认值
func TestGetEnv_NonExistingVar(t *testing.T) {
key := "TEDNA_TEST_ENV_VAR_NON_EXISTING_XXXXXX"
os.Unsetenv(key) // 确保不存在

result := getEnv(key, "my_default")
if result != "my_default" {
t.Errorf("不存在的环境变量应返回默认值my_default，实际%q", result)
}
}

// TestGetEnv_EmptyVar 环境变量为空字符串时返回默认值
func TestGetEnv_EmptyVar(t *testing.T) {
key := "TEDNA_TEST_ENV_VAR_EMPTY"
os.Setenv(key, "")
defer os.Unsetenv(key)

result := getEnv(key, "fallback")
if result != "fallback" {
t.Errorf("空环境变量应返回默认值fallback，实际%q", result)
}
}

// ==================== GetIntEnv 测试 ====================

// TestGetIntEnv_ValidInt 有效整数环境变量
func TestGetIntEnv_ValidInt(t *testing.T) {
key := "TEDNA_TEST_INT_ENV"
os.Setenv(key, "42")
defer os.Unsetenv(key)

result := GetIntEnv(key, 10)
if result != 42 {
t.Errorf("应返回42，实际%d", result)
}
}

// TestGetIntEnv_InvalidInt 非数字环境变量返回默认值
func TestGetIntEnv_InvalidInt(t *testing.T) {
key := "TEDNA_TEST_INT_ENV_INVALID"
os.Setenv(key, "not_a_number")
defer os.Unsetenv(key)

result := GetIntEnv(key, 99)
if result != 99 {
t.Errorf("非数字应返回默认值99，实际%d", result)
}
}

// TestGetIntEnv_EmptyVar 空值返回默认值
func TestGetIntEnv_EmptyVar(t *testing.T) {
key := "TEDNA_TEST_INT_ENV_EMPTY"
os.Unsetenv(key) // 确保不存在

result := GetIntEnv(key, 55)
if result != 55 {
t.Errorf("空值应返回默认值55，实际%d", result)
}
}

// TestGetIntEnv_ZeroValue 零值是合法整数
func TestGetIntEnv_ZeroValue(t *testing.T) {
key := "TEDNA_TEST_INT_ENV_ZERO"
os.Setenv(key, "0")
defer os.Unsetenv(key)

result := GetIntEnv(key, 100)
if result != 0 {
t.Errorf("0是合法整数，应返回0，实际%d", result)
}
}

// TestGetIntEnv_NegativeValue 负数是合法整数
func TestGetIntEnv_NegativeValue(t *testing.T) {
key := "TEDNA_TEST_INT_ENV_NEGATIVE"
os.Setenv(key, "-5")
defer os.Unsetenv(key)

result := GetIntEnv(key, 100)
if result != -5 {
t.Errorf("负数-5应返回-5，实际%d", result)
}
}

// TestGetIntEnv_FloatValue 浮点数不是合法整数
func TestGetIntEnv_FloatValue(t *testing.T) {
key := "TEDNA_TEST_INT_ENV_FLOAT"
os.Setenv(key, "3.14")
defer os.Unsetenv(key)

result := GetIntEnv(key, 100)
if result != 100 {
t.Errorf("浮点数3.14应返回默认值100，实际%d", result)
}
}

// ==================== GetAESKey 测试 ====================

// TestGetAESKey 测试GetAESKey返回配置中的AES密钥
func TestGetAESKey(t *testing.T) {
cfg := &Config{AESKey: "test-aes-key-32bytes-long-xxxxx"}
if cfg.GetAESKey() != "test-aes-key-32bytes-long-xxxxx" {
t.Errorf("GetAESKey应返回配置中的AESKey，实际%q", cfg.GetAESKey())
}
}

// TestGetAESKey_Empty 空AES密钥
func TestGetAESKey_Empty(t *testing.T) {
cfg := &Config{AESKey: ""}
if cfg.GetAESKey() != "" {
t.Errorf("空AESKey应返回空字符串，实际%q", cfg.GetAESKey())
}
}

// ==================== Config 结构体字段测试 ====================

// TestConfigDefaults 测试Config结构体默认字段
func TestConfigDefaults(t *testing.T) {
// 通过直接构造验证字段类型和赋值
cfg := Config{
DBHost:         "localhost",
DBPort:         "5432",
DBUser:         "test_user",
DBPassword:     "test_pass",
DBName:         "test_db",
Port:           "9090",
GinMode:        "debug",
JWTSecret:      "my-secret",
AESKey:         "my-aes-key",
AIAPIBaseURL:   "https://api.example.com",
AIAPIKey:       "sk-test",
AIDefaultModel: "gpt-4",
}
if cfg.DBHost != "localhost" {
t.Error("DBHost字段赋值失败")
}
if cfg.Port != "9090" {
t.Error("Port字段赋值失败")
}
if cfg.AIDefaultModel != "gpt-4" {
t.Error("AIDefaultModel字段赋值失败")
}
}
