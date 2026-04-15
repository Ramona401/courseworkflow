package services

// stage_coach_rules_test.go — 阶段教练规则引擎纯函数单元测试
//
// 测试范围（全部为纯函数，无数据库依赖）：
//   - checkAnalyzeStage / checkDesignStage / checkWriteStage / checkReviewStage / checkReviseStage / checkGenericStage
//   - getOutputNarrative / getOutputStructured
//   - checkUserEngagement / checkContainsKeywords / checkMinLength / boolHint

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

// ==================== getOutputNarrative 测试 ====================

func TestGetOutputNarrative_Nil(t *testing.T) {
	if getOutputNarrative(nil) != "" {
		t.Error("nil输入应返回空字符串")
	}
}

func TestGetOutputNarrative_WithContent(t *testing.T) {
	output := &models.WorkshopStageOutput{NarrativeOutput: "这是摘要内容"}
	if getOutputNarrative(output) != "这是摘要内容" {
		t.Error("应返回NarrativeOutput内容")
	}
}

// ==================== getOutputStructured 测试 ====================

func TestGetOutputStructured_Nil(t *testing.T) {
	if getOutputStructured(nil) != "" {
		t.Error("nil输入应返回空字符串")
	}
}

func TestGetOutputStructured_EmptyJSON(t *testing.T) {
	output := &models.WorkshopStageOutput{StructuredOutput: "{}"}
	if getOutputStructured(output) != "" {
		t.Error("空JSON对象应返回空字符串")
	}
}

func TestGetOutputStructured_ValidJSON(t *testing.T) {
	output := &models.WorkshopStageOutput{StructuredOutput: `{"total_score": 8.5, "dimensions": ["T1"]}`}
	result := getOutputStructured(output)
	if result == "" {
		t.Error("有效JSON不应返回空")
	}
	if !strings.Contains(result, "total_score") {
		t.Error("结果应包含total_score")
	}
}

func TestGetOutputStructured_InvalidJSON(t *testing.T) {
	output := &models.WorkshopStageOutput{StructuredOutput: "not json"}
	result := getOutputStructured(output)
	if result != "not json" {
		t.Errorf("无效JSON应返回原始内容，实际%q", result)
	}
}

// ==================== checkUserEngagement 测试 ====================

func TestCheckUserEngagement_BelowThreshold(t *testing.T) {
	item := checkUserEngagement(0, 2, "需要2轮对话")
	if item.Passed {
		t.Error("0条消息不应通过2轮门槛")
	}
	if item.Detail != "需要2轮对话" {
		t.Errorf("Detail应为提示文本，实际%q", item.Detail)
	}
}

func TestCheckUserEngagement_AtThreshold(t *testing.T) {
	item := checkUserEngagement(2, 2, "需要2轮对话")
	if !item.Passed {
		t.Error("恰好达到门槛应通过")
	}
}

func TestCheckUserEngagement_AboveThreshold(t *testing.T) {
	item := checkUserEngagement(5, 2, "需要2轮对话")
	if !item.Passed {
		t.Error("超过门槛应通过")
	}
	if !strings.Contains(item.Detail, "充分交流") {
		t.Errorf("通过时Detail应包含'充分交流'，实际%q", item.Detail)
	}
}

// ==================== checkContainsKeywords 测试 ====================

func TestCheckContainsKeywords_Found(t *testing.T) {
	item := checkContainsKeywords("教学目标", "本课的教学目标是培养学生思维能力", []string{"教学目标", "学习目标"}, 1)
	if !item.Passed {
		t.Error("文本包含关键词应通过")
	}
}

func TestCheckContainsKeywords_NotFound(t *testing.T) {
	item := checkContainsKeywords("教学目标", "本课没有相关内容", []string{"教学目标", "学习目标"}, 1)
	if item.Passed {
		t.Error("文本不包含任何关键词不应通过")
	}
	if !strings.Contains(item.Detail, "缺少") {
		t.Error("未通过时Detail应包含'缺少'")
	}
}

func TestCheckContainsKeywords_NoUserMessages(t *testing.T) {
	// 用户未发言时即使文本有关键词也不通过（避免AI开场白误判）
	item := checkContainsKeywords("教学目标", "教学目标已确定", []string{"教学目标"}, 0)
	if item.Passed {
		t.Error("用户未发言时不应通过关键词检测")
	}
}

func TestCheckContainsKeywords_MultipleKeywords(t *testing.T) {
	// 只需命中一个关键词即可
	item := checkContainsKeywords("学情", "学生基础较好", []string{"学情", "学生特点", "学生基础"}, 2)
	if !item.Passed {
		t.Error("命中任一关键词应通过")
	}
}

// ==================== checkMinLength 测试 ====================

func TestCheckMinLength_BelowThreshold(t *testing.T) {
	item := checkMinLength("教案篇幅", "短文本", 2000, "内容偏短")
	if item.Passed {
		t.Error("短文本不应通过2000字门槛")
	}
	if item.Detail != "内容偏短" {
		t.Errorf("Detail应为提示文本，实际%q", item.Detail)
	}
}

func TestCheckMinLength_AboveThreshold(t *testing.T) {
	longText := strings.Repeat("这是一段较长的教案内容，", 200)
	item := checkMinLength("教案篇幅", longText, 2000, "内容偏短")
	if !item.Passed {
		t.Error("长文本应通过门槛")
	}
}

func TestCheckMinLength_ChineseChars(t *testing.T) {
	// 确保按字符（rune）而非字节计算
	text := strings.Repeat("中", 2000)
	item := checkMinLength("教案篇幅", text, 2000, "内容偏短")
	if !item.Passed {
		t.Error("2000个中文字符应通过2000字门槛")
	}
}

// ==================== boolHint 测试 ====================

func TestBoolHint_True(t *testing.T) {
	result := boolHint(true, "已完成", "未完成")
	if result != "已完成" {
		t.Errorf("true应返回passMsg，实际%q", result)
	}
}

func TestBoolHint_False(t *testing.T) {
	result := boolHint(false, "已完成", "未完成")
	if result != "未完成" {
		t.Errorf("false应返回failMsg，实际%q", result)
	}
}

// ==================== checkAnalyzeStage 测试 ====================

func TestCheckAnalyzeStage_FullyComplete(t *testing.T) {
	output := &models.WorkshopStageOutput{
		NarrativeOutput:  "教学目标已明确，学情分析显示学生基础较好，重点是掌握核心概念，难点在于应用",
		StructuredOutput: "{}",
	}
	items := checkAnalyzeStage(output, 3)
	passedCount := 0
	for _, item := range items {
		if item.Passed {
			passedCount++
		}
	}
	if passedCount < 3 {
		t.Errorf("完整分析阶段应至少通过3项，实际通过%d项", passedCount)
		for _, item := range items {
			t.Logf("  %s: passed=%v detail=%s", item.Label, item.Passed, item.Detail)
		}
	}
}

func TestCheckAnalyzeStage_Empty(t *testing.T) {
	items := checkAnalyzeStage(nil, 0)
	// 4个检查项应全部不通过
	for _, item := range items {
		if item.Passed {
			t.Errorf("空产出+0消息时，%q不应通过", item.Label)
		}
	}
}

// ==================== checkDesignStage 测试 ====================

func TestCheckDesignStage_FullyComplete(t *testing.T) {
	output := &models.WorkshopStageOutput{
		NarrativeOutput: "设计了导入环节5分钟，采用合作探究方法，包含练习活动",
	}
	items := checkDesignStage(output, 3)
	passedCount := 0
	for _, item := range items {
		if item.Passed {
			passedCount++
		}
	}
	if passedCount < 3 {
		t.Errorf("完整设计阶段应至少通过3项，实际%d", passedCount)
	}
}

// ==================== checkWriteStage 测试 ====================

func TestCheckWriteStage_WithLessonPlan(t *testing.T) {
	longContent := "# 教学过程\n" + strings.Repeat("这是教案内容，包含教学环节详细描述。", 200) + "\n## 作业布置\n完成练习册"
	lp := &models.LessonPlan{ContentMarkdown: longContent}
	items := checkWriteStage(nil, lp, 2)
	passedCount := 0
	for _, item := range items {
		if item.Passed {
			passedCount++
		}
	}
	if passedCount < 3 {
		t.Errorf("有完整教案的write阶段应至少通过3项，实际%d", passedCount)
	}
}

func TestCheckWriteStage_ShortContent(t *testing.T) {
	lp := &models.LessonPlan{ContentMarkdown: "很短的内容"}
	items := checkWriteStage(nil, lp, 1)
	// 教案篇幅检查应不通过
	for _, item := range items {
		if item.Label == "教案篇幅" && item.Passed {
			t.Error("短内容的教案篇幅不应通过")
		}
	}
}

// ==================== checkReviewStage 测试 ====================

func TestCheckReviewStage_Complete(t *testing.T) {
	output := &models.WorkshopStageOutput{
		StructuredOutput: `{"total_score": 8.5, "dimensions": [{"code":"T1"}], "improvements": [{"id":"1"}]}`,
	}
	items := checkReviewStage(output)
	passedCount := 0
	for _, item := range items {
		if item.Passed {
			passedCount++
		}
	}
	if passedCount != 3 {
		t.Errorf("完整评审应通过3项，实际%d", passedCount)
		for _, item := range items {
			t.Logf("  %s: passed=%v", item.Label, item.Passed)
		}
	}
}

func TestCheckReviewStage_Empty(t *testing.T) {
	items := checkReviewStage(nil)
	for _, item := range items {
		if item.Passed {
			t.Errorf("空评审产出时，%q不应通过", item.Label)
		}
	}
}

// ==================== checkReviseStage 测试 ====================

func TestCheckReviseStage_Complete(t *testing.T) {
	longContent := strings.Repeat("修订后的教案内容", 200)
	lp := &models.LessonPlan{ContentMarkdown: longContent}
	output := &models.WorkshopStageOutput{
		NarrativeOutput: strings.Repeat("修订讨论记录", 20),
	}
	items := checkReviseStage(output, lp, 2)
	passedCount := 0
	for _, item := range items {
		if item.Passed {
			passedCount++
		}
	}
	if passedCount < 3 {
		t.Errorf("完整修订阶段应通过3项，实际%d", passedCount)
	}
}

// ==================== checkGenericStage 测试 ====================

func TestCheckGenericStage_WithOutput(t *testing.T) {
	output := &models.WorkshopStageOutput{Status: "in_progress"}
	items := checkGenericStage(output, 2)
	if len(items) != 2 {
		t.Fatalf("通用阶段应有2个检查项，实际%d", len(items))
	}
	if !items[0].Passed {
		t.Error("2条消息应通过参与度检查")
	}
	if !items[1].Passed {
		t.Error("有产出物应通过")
	}
}

func TestCheckGenericStage_NoOutput(t *testing.T) {
	items := checkGenericStage(nil, 0)
	for _, item := range items {
		if item.Passed {
			t.Errorf("无产出+0消息时，%q不应通过", item.Label)
		}
	}
}
