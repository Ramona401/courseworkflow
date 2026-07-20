package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestValidateGeneratedPageInteractionStatic(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"static",
		`<div class="cw-page"><h1>静态知识页</h1></div>`,
	)
	if !result.OK {
		t.Fatalf("静态页面不应被强制要求交互：%s", result.Reason)
	}
}

func TestValidateGeneratedPageInteractionClickRejectsFakePrompt(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"click",
		`<div class="card" style="cursor:pointer">点击卡片查看详情</div>`,
	)
	if result.OK {
		t.Fatal("只有点击提示文字、没有事件的页面不应通过")
	}
}

func TestValidateGeneratedPageInteractionClickPassesRealEvent(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"click",
		`<button onclick="document.getElementById('detail').hidden=false">查看</button><div id="detail" hidden>详情</div>`,
	)
	if !result.OK {
		t.Fatalf("真实点击事件应通过：%s", result.Reason)
	}
}

func TestValidateGeneratedPageInteractionInputRejectsClickOnly(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"input",
		`<button onclick="nextStep()">下一步</button><div id="result"></div>`,
	)
	if result.OK {
		t.Fatal("输入填写不能被下一步点击按钮代替")
	}
}

func TestValidateGeneratedPageInteractionInputPasses(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"input",
		`<input id="answer"><button onclick="checkAnswer()">检查</button><div id="result">结果提示</div>`,
	)
	if !result.OK {
		t.Fatalf("输入框、检查事件和结果区域齐全时应通过：%s", result.Reason)
	}
}

func TestValidateGeneratedPageInteractionDragPasses(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"drag",
		`<div id="ball" draggable="true" ondragstart="startDrag(event)"></div><div ondragover="event.preventDefault()" ondrop="dropBall(event)"></div>`,
	)
	if !result.OK {
		t.Fatalf("真实拖拽事件链应通过：%s", result.Reason)
	}
}

func TestValidateGeneratedPageInteractionAnimationPasses(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"animation",
		`<style>@keyframes move{from{left:0}to{left:100px}}.ball{animation:move 2s linear}</style><div class="ball"></div>`,
	)
	if !result.OK {
		t.Fatalf("真实CSS动画应通过：%s", result.Reason)
	}
}

func TestValidateGeneratedPageInteractionQuizPasses(t *testing.T) {
	result := validateGeneratedPageInteraction(
		"quiz",
		`<label><input type="radio" name="answer">选项A</label><button onclick="checkAnswer()">提交</button><div id="feedback">正确错误反馈</div>`,
	)
	if !result.OK {
		t.Fatalf("作答控件、事件和反馈齐全时应通过：%s", result.Reason)
	}
}

func TestCWInteractionLevelForPlanUsesTeacherSelection(t *testing.T) {
	if got := cwInteractionLevelForPlan("input", 1, 5); got != 3 {
		t.Fatalf("input应优先映射为3，实际=%d", got)
	}
	if got := cwInteractionLevelForPlan("drag", 1, 2); got != 4 {
		t.Fatalf("drag应优先映射为4，实际=%d", got)
	}
	if got := cwInteractionLevelForPlan("", 4, 2); got != 4 {
		t.Fatalf("互动类型为空时应回退索引等级4，实际=%d", got)
	}
}

func TestCWVisualFormatForMatchUsesPlanField(t *testing.T) {
	page := &models.CoursewarePage{
		VisualFormat:    "text_heavy",
		IdxVisualFormat: "diagram",
	}
	if got := cwVisualFormatForMatch(page); got != "text_heavy" {
		t.Fatalf("老师确认的视觉形式应优先，实际=%s", got)
	}
}

func TestBuildCWInteractionRepairPromptIncludesFailureAndContract(t *testing.T) {
	page := &models.CoursewarePage{
		InteractionType: "input",
	}
	prompt := buildCWInteractionRepairPrompt(
		"原始提示词",
		page,
		cwInteractionValidationResult{
			OK:     false,
			Reason: "没有输入框",
			Detail: "missing input control",
		},
	)

	for _, required := range []string{
		"原始提示词",
		"没有输入框",
		"输入填写",
		"input、textarea",
		"完整重新生成",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("纠偏提示词缺少内容：%s", required)
		}
	}
}
