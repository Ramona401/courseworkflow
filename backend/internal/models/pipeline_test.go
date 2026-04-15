// Package models 数据模型单元测试
// 测试范围：DefaultPipelineConfig / ParsePipelineConfig(string) / 常量完整性
// v92更新：Threshold期望值从9.0改为8.5（v90已变更默认阈值）
package models

import (
	"encoding/json"
	"testing"
)

// TestDefaultPipelineConfig 测试默认配置值符合业务规范
// v90变更：Threshold从9.0改为8.5
func TestDefaultPipelineConfig(t *testing.T) {
	cfg := DefaultPipelineConfig()
	if cfg == nil {
		t.Fatal("DefaultPipelineConfig不应返回nil")
	}
	if cfg.EvalRounds != 3 {
		t.Errorf("EvalRounds期望3, 实际%d", cfg.EvalRounds)
	}
	if cfg.Threshold != 8.5 {
		t.Errorf("Threshold期望8.5, 实际%.1f", cfg.Threshold)
	}
	if cfg.VarianceWarn != 1.5 {
		t.Errorf("VarianceWarn期望1.5, 实际%.1f", cfg.VarianceWarn)
	}
	if cfg.MaxMetaRetry != 3 {
		t.Errorf("MaxMetaRetry期望3, 实际%d", cfg.MaxMetaRetry)
	}
	if cfg.MaxTRLoop != 3 {
		t.Errorf("MaxTRLoop期望3, 实际%d", cfg.MaxTRLoop)
	}
	t.Logf("DefaultPipelineConfig验证通过: %+v", *cfg)
}

// TestParsePipelineConfigValid 测试有效JSON字符串解析
func TestParsePipelineConfigValid(t *testing.T) {
	jsonStr := `{"eval_rounds":5,"threshold":8.5,"variance_warn":2.0,"max_meta_retry":2,"max_tr_loop":4}`
	cfg := ParsePipelineConfig(jsonStr)
	if cfg == nil {
		t.Fatal("ParsePipelineConfig不应返回nil")
	}
	if cfg.EvalRounds != 5 {
		t.Errorf("EvalRounds期望5, 实际%d", cfg.EvalRounds)
	}
	if cfg.Threshold != 8.5 {
		t.Errorf("Threshold期望8.5, 实际%.1f", cfg.Threshold)
	}
	if cfg.MaxTRLoop != 4 {
		t.Errorf("MaxTRLoop期望4, 实际%d", cfg.MaxTRLoop)
	}
	t.Logf("有效配置解析通过: %+v", *cfg)
}

// TestParsePipelineConfigEmpty 测试空字符串返回默认配置
func TestParsePipelineConfigEmpty(t *testing.T) {
	cfg := ParsePipelineConfig("")
	if cfg == nil {
		t.Fatal("空字符串不应返回nil")
	}
	defaultCfg := DefaultPipelineConfig()
	if cfg.EvalRounds != defaultCfg.EvalRounds {
		t.Errorf("空输入应返回默认EvalRounds=%d, 实际%d", defaultCfg.EvalRounds, cfg.EvalRounds)
	}
	t.Logf("空字符串返回默认配置: %+v", *cfg)
}

// TestParsePipelineConfigInvalidJSON 测试无效JSON返回默认配置（容错）
func TestParsePipelineConfigInvalidJSON(t *testing.T) {
	cfg := ParsePipelineConfig("{invalid json}")
	if cfg == nil {
		t.Fatal("无效JSON不应返回nil")
	}
	// 解析失败应回退到默认配置，不panic
	t.Logf("无效JSON容错通过: %+v", *cfg)
}

// TestParsePipelineConfigPartial 测试部分字段JSON
func TestParsePipelineConfigPartial(t *testing.T) {
	cfg := ParsePipelineConfig(`{"eval_rounds":2}`)
	if cfg == nil {
		t.Fatal("不应返回nil")
	}
	if cfg.EvalRounds != 2 {
		t.Errorf("EvalRounds期望2, 实际%d", cfg.EvalRounds)
	}
	t.Logf("部分字段解析: %+v", *cfg)
}

// TestPipelineConfigToJSON 测试配置序列化
func TestPipelineConfigToJSON(t *testing.T) {
	cfg := DefaultPipelineConfig()
	jsonStr := cfg.ToJSON()
	if jsonStr == "" {
		t.Fatal("ToJSON不应返回空字符串")
	}
	// 验证JSON有效性
	var cfg2 PipelineConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg2); err != nil {
		t.Fatalf("ToJSON结果无法反序列化: %v，JSON: %s", err, jsonStr)
	}
	if cfg2.EvalRounds != cfg.EvalRounds {
		t.Errorf("序列化/反序列化后EvalRounds不一致")
	}
	t.Logf("ToJSON验证通过: %s", jsonStr)
}

// TestPipelineConfigToJSONRoundTrip 测试ToJSON往返
func TestPipelineConfigToJSONRoundTrip(t *testing.T) {
	original := &PipelineConfig{
		EvalRounds:   5,
		Threshold:    8.8,
		VarianceWarn: 2.0,
		MaxMetaRetry: 4,
		MaxTRLoop:    5,
	}
	jsonStr := original.ToJSON()
	recovered := ParsePipelineConfig(jsonStr)
	if recovered.EvalRounds != original.EvalRounds {
		t.Errorf("往返后EvalRounds不一致: %d vs %d", recovered.EvalRounds, original.EvalRounds)
	}
	if recovered.Threshold != original.Threshold {
		t.Errorf("往返后Threshold不一致: %.1f vs %.1f", recovered.Threshold, original.Threshold)
	}
	t.Log("配置ToJSON往返验证通过")
}

// TestStepDefinitionsCount 测试步骤定义数量=8
func TestStepDefinitionsCount(t *testing.T) {
	if len(StepDefinitions) != TotalSteps {
		t.Errorf("StepDefinitions数量(%d)应等于TotalSteps(%d)", len(StepDefinitions), TotalSteps)
	}
	if TotalSteps != 8 {
		t.Errorf("TotalSteps期望8步, 实际%d", TotalSteps)
	}
	t.Logf("步骤定义共%d步", TotalSteps)
}

// TestStepDefinitionsOrder 测试步骤Order连续从1开始
func TestStepDefinitionsOrder(t *testing.T) {
	for i, step := range StepDefinitions {
		if step.Order != i+1 {
			t.Errorf("步骤%s Order期望%d, 实际%d", step.Name, i+1, step.Order)
		}
		if step.Name == "" {
			t.Errorf("步骤%d Name不应为空", i+1)
		}
	}
	t.Logf("步骤顺序验证通过")
}

// TestPipelineStatusConstants 测试状态常量（前缀PipelineStatus*）
func TestPipelineStatusConstants(t *testing.T) {
	statuses := []string{
		PipelineStatusPending,
		PipelineStatusRunning,
		PipelineStatusReviewQueue,
		PipelineStatusPendingFinalize,
		PipelineStatusFinalized,
		PipelineStatusNeedsHuman,
		PipelineStatusFailed,
		PipelineStatusCancelled,
		PipelineStatusVerified,
		PipelineStatusVerifyFailed,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("状态常量不应为空字符串")
		}
	}
	t.Logf("共%d个Pipeline状态常量验证通过", len(statuses))
}

// TestPipelineStatusNameMap 测试状态名称映射完整性
func TestPipelineStatusNameMap(t *testing.T) {
	statuses := []string{
		PipelineStatusPending, PipelineStatusRunning, PipelineStatusReviewQueue,
		PipelineStatusPendingFinalize, PipelineStatusFinalized, PipelineStatusNeedsHuman,
		PipelineStatusFailed, PipelineStatusCancelled, PipelineStatusVerified, PipelineStatusVerifyFailed,
	}
	for _, s := range statuses {
		name, ok := PipelineStatusNameMap[s]
		if !ok || name == "" {
			t.Errorf("状态 %q 在PipelineStatusNameMap中无中文名称", s)
		}
	}
	t.Logf("PipelineStatusNameMap完整性验证通过，共%d项", len(PipelineStatusNameMap))
}
