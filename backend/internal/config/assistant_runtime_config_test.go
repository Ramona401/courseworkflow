package config

// assistant_runtime_config_test.go
//
// 验证公开运行短时令牌配置与AssistantRuntimeTokenService的5—15分钟
// 合法范围保持一致，防止配置层再次回退到令牌服务拒绝的30分钟。

import (
	"testing"
	"time"
)

// TestAssistantRuntimeConfigTTLValid 验证合法边界值原样返回。
func TestAssistantRuntimeConfigTTLValid(
	t *testing.T,
) {
	cases := []struct {
		name     string
		minutes  int
		expected time.Duration
	}{
		{
			name:     "最短五分钟",
			minutes:  5,
			expected: 5 * time.Minute,
		},
		{
			name:     "中间十分钟",
			minutes:  10,
			expected: 10 * time.Minute,
		},
		{
			name:     "最长十五分钟",
			minutes:  15,
			expected: 15 * time.Minute,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				cfg := &Config{
					AssistantRuntimeTokenTTLMinutes:
						item.minutes,
				}

				actual := cfg.GetAssistantRuntimeTokenTTL()
				if actual != item.expected {
					t.Fatalf(
						"运行令牌TTL错误: minutes=%d expected=%s actual=%s",
						item.minutes,
						item.expected,
						actual,
					)
				}
			},
		)
	}
}

// TestAssistantRuntimeConfigTTLFallback 验证非法值统一回退15分钟。
func TestAssistantRuntimeConfigTTLFallback(
	t *testing.T,
) {
	cases := []struct {
		name    string
		config  *Config
		minutes int
	}{
		{
			name:   "空配置",
			config: nil,
		},
		{
			name:    "零分钟",
			minutes: 0,
		},
		{
			name:    "低于最短值",
			minutes: 4,
		},
		{
			name:    "旧默认三十分钟",
			minutes: 30,
		},
		{
			name:    "超长一天",
			minutes: 1440,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				cfg := item.config
				if cfg == nil &&
					item.name != "空配置" {
					cfg = &Config{
						AssistantRuntimeTokenTTLMinutes:
							item.minutes,
					}
				}

				actual := cfg.GetAssistantRuntimeTokenTTL()
				if actual != 15*time.Minute {
					t.Fatalf(
						"非法运行令牌TTL未回退15分钟: minutes=%d actual=%s",
						item.minutes,
						actual,
					)
				}
			},
		)
	}
}
