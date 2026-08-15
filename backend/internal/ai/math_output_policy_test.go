package ai

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestApplyMathOutputPolicyLessonPlanAppendsToSystem(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneLessonPlan}
	original := []ChatMessage{
		{Role: "system", Content: "基础教案系统提示词"},
		{Role: "user", Content: "请生成一道力学题"},
	}

	got := applyMathOutputPolicy(cfg, original, nil)

	if len(got) != len(original) {
		t.Fatalf("消息数量异常：got=%d want=%d", len(got), len(original))
	}
	if !strings.Contains(got[0].Content, "【数学符号输出规范 · 强制】") {
		t.Fatalf("教案场景未注入数学符号规范")
	}
	if !strings.Contains(got[0].Content, "F₁ = G·tanθ") {
		t.Fatalf("教案场景缺少Unicode数学示例")
	}
	if original[0].Content != "基础教案系统提示词" {
		t.Fatalf("策略函数修改了调用方原始消息")
	}
}

func TestApplyMathOutputPolicyCoursewareUsesTraceScene(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneScanner}
	traceCtx := &TraceContext{SceneCode: models.SceneCW3DSingle}
	original := []ChatMessage{{Role: "user", Content: "生成3D力学课件"}}

	got := applyMathOutputPolicy(cfg, original, traceCtx)

	if len(got) != 2 {
		t.Fatalf("无system消息时应新增一条system消息：got=%d", len(got))
	}
	if got[0].Role != "system" {
		t.Fatalf("新增数学规范必须是system角色：got=%s", got[0].Role)
	}
	if !strings.Contains(got[0].Content, "HTML、CSS、JavaScript") {
		t.Fatalf("课件场景未使用HTML安全版本数学规范")
	}
}

func TestApplyMathOutputPolicyUnrelatedSceneDoesNotInject(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneScanner}
	original := []ChatMessage{
		{Role: "system", Content: "扫描任务"},
		{Role: "user", Content: "压缩索引"},
	}

	got := applyMathOutputPolicy(cfg, original, nil)

	if len(got) != len(original) {
		t.Fatalf("非目标场景不应改变消息数量")
	}
	if got[0].Content != original[0].Content {
		t.Fatalf("非目标场景不应修改system提示词")
	}
}

func TestApplyMathOutputPolicyIsIdempotent(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneCWGenerate}
	original := []ChatMessage{
		{Role: "system", Content: "课件生成基础规范"},
		{Role: "user", Content: "生成页面"},
	}

	first := applyMathOutputPolicy(cfg, original, nil)
	second := applyMathOutputPolicy(cfg, first, nil)

	if count := strings.Count(
		second[0].Content,
		"【数学符号输出规范 · 强制】",
	); count != 1 {
		t.Fatalf("数学规范重复注入：count=%d", count)
	}
}

func TestApplyMathOutputPolicyTraceSceneTakesPriority(t *testing.T) {
	cfg := &EffectiveConfig{SceneCode: models.SceneScanner}
	traceCtx := &TraceContext{SceneCode: models.SceneLessonPlanHarness}
	original := []ChatMessage{
		{Role: "system", Content: "Harness基础规则"},
		{Role: "user", Content: "修复候选教案"},
	}

	got := applyMathOutputPolicy(cfg, original, traceCtx)

	if !strings.Contains(got[0].Content, "F₁ = G·tanθ") {
		t.Fatalf("TraceContext中的Harness场景未优先应用教案数学规范")
	}
}
