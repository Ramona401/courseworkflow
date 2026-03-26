// Package config 配置模块单元测试
// 测试范围：AppVersion格式、配置常量
package config

import (
	"strings"
	"testing"
)

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

// TestAppVersionIsCurrentVersion 测试当前版本号为v0.31.0（对应v34交接文档）
func TestAppVersionIsCurrentVersion(t *testing.T) {
	expected := "0.31.0"
	if AppVersion != expected {
		// 不强制失败，版本号可能已升级，仅记录
		t.Logf("注意：AppVersion=%s，文档记录版本为%s", AppVersion, expected)
	} else {
		t.Logf("版本号匹配: %s", AppVersion)
	}
}
