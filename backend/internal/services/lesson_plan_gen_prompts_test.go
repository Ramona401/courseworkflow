package services

// lesson_plan_gen_prompts_test.go — 教案生成提示词构建+解析函数单元测试
//
// 测试范围（全部为纯函数，无外部依赖）：
//   - buildDefaultReviewRules：评审规则构建（通用+AI学科特化）
//   - buildReviewSystemPrompt：评审系统提示词
//   - buildReviewPrompt：评审用户提示词
//   - buildOptimizePrompt：优化提示词
//   - buildDefaultOpeningMessage：默认开场消息
//   - parseAIReviewResult：AI评审JSON解析
//   - buildFallbackReview：降级评审结果
//   - appendReviewToHistory：评审历史追加
//   - extractSuggestionsByIDs：按ID提取建议
//   - convertGroupsToConvComponents：组件格式转换
//   - cleanComponentMarkers：清除组件标记
//   - formatSelectedOptions：选项格式化
//   - formatSelectedComponents：组件数量格式化
//   - generateMsgID：消息ID生成

import (
	"encoding/json"
	"strings"
	"testing"

	"tedna/internal/models"
)

// ==================== buildDefaultReviewRules 测试 ====================

// TestBuildDefaultReviewRules_Generic 通用学科包含5个T维度
func TestBuildDefaultReviewRules_Generic(t *testing.T) {
	rules := buildDefaultReviewRules("数学")
	if !strings.Contains(rules, "T1") {
		t.Error("通用规则应包含T1维度")
	}
	if !strings.Contains(rules, "T5") {
		t.Error("通用规则应包含T5维度")
	}
	// 非AI学科不应包含S维度
	if strings.Contains(rules, "S1") {
		t.Error("非AI学科不应包含S1维度")
	}
}

// TestBuildDefaultReviewRules_AISubject AI学科包含额外S维度
func TestBuildDefaultReviewRules_AISubject(t *testing.T) {
	for _, subject := range []string{"AI", "人工智能"} {
		rules := buildDefaultReviewRules(subject)
		if !strings.Contains(rules, "S1") {
			t.Errorf("学科%q应包含S1维度", subject)
		}
		if !strings.Contains(rules, "S5") {
			t.Errorf("学科%q应包含S5维度", subject)
		}
		if !strings.Contains(rules, "T1") {
			t.Errorf("学科%q也应包含通用T维度", subject)
		}
	}
}

// ==================== buildReviewSystemPrompt 测试 ====================

// TestBuildReviewSystemPrompt 评审系统提示词包含学科和JSON格式要求
func TestBuildReviewSystemPrompt(t *testing.T) {
	prompt := buildReviewSystemPrompt("数学")
	if !strings.Contains(prompt, "数学") {
		t.Error("评审提示词应包含学科名称")
	}
	if !strings.Contains(prompt, "total_score") {
		t.Error("评审提示词应包含total_score字段要求")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("评审提示词应提及JSON格式")
	}
}

// ==================== buildReviewPrompt 测试 ====================

// TestBuildReviewPrompt 评审用户提示词包含教案信息
func TestBuildReviewPrompt(t *testing.T) {
	lp := &models.LessonPlan{
		Subject:         "英语",
		Grade:           "七年级",
		Topic:           "过去式",
		DurationMinutes: 45,
		ContentMarkdown: "# 教案内容\n这是一节关于过去式的课。",
	}
	prompt := buildReviewPrompt(lp, "T1目标清晰度")
	if !strings.Contains(prompt, "英语") {
		t.Error("应包含学科")
	}
	if !strings.Contains(prompt, "七年级") {
		t.Error("应包含年级")
	}
	if !strings.Contains(prompt, "过去式") {
		t.Error("应包含课题")
	}
	if !strings.Contains(prompt, "45") {
		t.Error("应包含课时时长")
	}
	if !strings.Contains(prompt, "教案内容") {
		t.Error("应包含教案内容")
	}
	if !strings.Contains(prompt, "T1目标清晰度") {
		t.Error("应包含评审维度")
	}
}

// ==================== buildOptimizePrompt 测试 ====================

// TestBuildOptimizePrompt 优化提示词包含建议和原教案
func TestBuildOptimizePrompt(t *testing.T) {
	content := "# 原教案\n内容在这里"
	suggestions := []string{"增加互动环节", "调整时间分配"}
	prompt := buildOptimizePrompt(content, suggestions)
	if !strings.Contains(prompt, "增加互动环节") {
		t.Error("应包含第一条建议")
	}
	if !strings.Contains(prompt, "调整时间分配") {
		t.Error("应包含第二条建议")
	}
	if !strings.Contains(prompt, "原教案") {
		t.Error("应包含原教案内容")
	}
	if !strings.Contains(prompt, "1.") && !strings.Contains(prompt, "2.") {
		t.Error("建议应有编号")
	}
}

// ==================== buildDefaultOpeningMessage 测试 ====================

// TestBuildDefaultOpeningMessage 默认开场消息包含课程信息
func TestBuildDefaultOpeningMessage(t *testing.T) {
	req := &models.StartConversationRequest{
		Subject:         "数学",
		Grade:           "三年级",
		Topic:           "分数加法",
		DurationMinutes: 40,
	}
	msg := buildDefaultOpeningMessage(req)
	if msg == nil {
		t.Fatal("不应返回nil")
	}
	if msg.Role != models.ConvRoleAssistant {
		t.Errorf("角色应为assistant，实际%s", string(msg.Role))
	}
	if msg.Type != models.ConvMsgTypeText {
		t.Errorf("类型应为text，实际%s", string(msg.Type))
	}
	if !strings.Contains(msg.Content, "三年级") {
		t.Error("开场消息应包含年级")
	}
	if !strings.Contains(msg.Content, "数学") {
		t.Error("开场消息应包含学科")
	}
	if !strings.Contains(msg.Content, "分数加法") {
		t.Error("开场消息应包含课题")
	}
	if msg.ID == "" {
		t.Error("消息ID不应为空")
	}
}

// ==================== parseAIReviewResult 测试 ====================

// TestParseAIReviewResult_ValidJSON 解析正常的AI评审结果
func TestParseAIReviewResult_ValidJSON(t *testing.T) {
	input := `一些前导文字
` + "```json" + `
{
  "total_score": 8.5,
  "summary": "教案整体不错",
  "good_points": ["目标清晰", "结构完整"],
  "improvements": [
    {"id": "imp_1", "issue": "互动不足", "suggestion": "增加小组讨论"}
  ],
  "dimensions": [
    {"code": "T1", "name": "目标清晰度", "score": 9, "comment": "很好", "good": true}
  ]
}
` + "```"

	result, err := parseAIReviewResult(input)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.TotalScore != 8.5 {
		t.Errorf("总分应为8.5，实际%f", result.TotalScore)
	}
	if len(result.GoodPoints) != 2 {
		t.Errorf("GoodPoints应有2条，实际%d", len(result.GoodPoints))
	}
	if len(result.Improvements) != 1 {
		t.Errorf("Improvements应有1条，实际%d", len(result.Improvements))
	}
	if result.ReviewedAt.IsZero() {
		t.Error("ReviewedAt不应为零值")
	}
}

// TestParseAIReviewResult_NoJSON 无JSON内容应报错
func TestParseAIReviewResult_NoJSON(t *testing.T) {
	_, err := parseAIReviewResult("这段话里没有任何JSON内容")
	if err == nil {
		t.Error("无JSON内容应返回error")
	}
}

// ==================== buildFallbackReview 测试 ====================

// TestBuildFallbackReview 降级评审结果
func TestBuildFallbackReview(t *testing.T) {
	result := buildFallbackReview("原始AI回复内容")
	if result.TotalScore != 7.0 {
		t.Errorf("降级评分应为7.0，实际%f", result.TotalScore)
	}
	if len(result.Improvements) != 1 {
		t.Fatalf("降级应有1条改进建议，实际%d", len(result.Improvements))
	}
	if result.Improvements[0].Suggestion != "原始AI回复内容" {
		t.Error("降级建议应包含原始AI回复")
	}
	if result.ReviewedAt.IsZero() {
		t.Error("ReviewedAt不应为零值")
	}
}

// ==================== appendReviewToHistory 测试 ====================

// TestAppendReviewToHistory_EmptyHistory 空历史追加第一条
func TestAppendReviewToHistory_EmptyHistory(t *testing.T) {
	review := &models.AIReviewResult{TotalScore: 8.0, Summary: "不错"}
	result := appendReviewToHistory("[]", review)
	var history []models.AIReviewResult
	if err := json.Unmarshal([]byte(result), &history); err != nil {
		t.Fatalf("解析结果失败: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("历史应有1条，实际%d", len(history))
	}
	if history[0].TotalScore != 8.0 {
		t.Errorf("第一条评分应为8.0，实际%f", history[0].TotalScore)
	}
}

// TestAppendReviewToHistory_InvalidJSON 无效JSON历史应重置为只含新记录
func TestAppendReviewToHistory_InvalidJSON(t *testing.T) {
	review := &models.AIReviewResult{TotalScore: 7.5}
	result := appendReviewToHistory("invalid json", review)
	var history []models.AIReviewResult
	if err := json.Unmarshal([]byte(result), &history); err != nil {
		t.Fatalf("解析结果失败: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("无效历史应重置，只有1条，实际%d", len(history))
	}
}

// TestAppendReviewToHistory_AppendToExisting 已有历史追加
func TestAppendReviewToHistory_AppendToExisting(t *testing.T) {
	existing := `[{"total_score":7.0,"summary":"第一次"}]`
	review := &models.AIReviewResult{TotalScore: 8.5, Summary: "第二次"}
	result := appendReviewToHistory(existing, review)
	var history []models.AIReviewResult
	if err := json.Unmarshal([]byte(result), &history); err != nil {
		t.Fatalf("解析结果失败: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("历史应有2条，实际%d", len(history))
	}
}

// ==================== extractSuggestionsByIDs 测试 ====================

// TestExtractSuggestionsByIDs_SpecificIDs 提取指定ID的建议
func TestExtractSuggestionsByIDs_SpecificIDs(t *testing.T) {
	reviewJSON := `{
		"total_score": 8.0,
		"improvements": [
			{"id": "imp_1", "suggestion": "建议A"},
			{"id": "imp_2", "suggestion": "建议B"},
			{"id": "imp_3", "suggestion": "建议C"}
		]
	}`
	suggestions := extractSuggestionsByIDs(reviewJSON, []string{"imp_1", "imp_3"})
	if len(suggestions) != 2 {
		t.Fatalf("应提取2条建议，实际%d", len(suggestions))
	}
	if suggestions[0] != "建议A" {
		t.Errorf("第一条应为建议A，实际%s", suggestions[0])
	}
	if suggestions[1] != "建议C" {
		t.Errorf("第二条应为建议C，实际%s", suggestions[1])
	}
}

// TestExtractSuggestionsByIDs_EmptyIDs 空ID列表返回全部建议
func TestExtractSuggestionsByIDs_EmptyIDs(t *testing.T) {
	reviewJSON := `{"improvements": [{"id":"a","suggestion":"S1"},{"id":"b","suggestion":"S2"}]}`
	suggestions := extractSuggestionsByIDs(reviewJSON, []string{})
	if len(suggestions) != 2 {
		t.Errorf("空ID列表应返回全部建议，实际%d条", len(suggestions))
	}
}

// TestExtractSuggestionsByIDs_InvalidJSON 无效JSON返回nil
func TestExtractSuggestionsByIDs_InvalidJSON(t *testing.T) {
	suggestions := extractSuggestionsByIDs("not json", []string{"imp_1"})
	if suggestions != nil {
		t.Error("无效JSON应返回nil")
	}
}

// ==================== convertGroupsToConvComponents 测试 ====================

// TestConvertGroupsToConvComponents 组件组转换为对话组件格式
func TestConvertGroupsToConvComponents(t *testing.T) {
	groups := []*models.MatchedComponentGroup{
		{
			LibraryType: "pedagogy",
			Components: []*models.MatchedComponent{
				{ID: "c1", DisplayLabel: "组件1", DesignLogic: "逻辑1", QualityScore: 8.0, UsageCount: 5},
				{ID: "c2", DisplayLabel: "组件2", DesignLogic: "逻辑2", QualityScore: 7.5, UsageCount: 3},
			},
		},
		{
			LibraryType: "activity_design",
			Components: []*models.MatchedComponent{
				{ID: "c3", DisplayLabel: "组件3", QualityScore: 9.0},
			},
		},
	}
	result := convertGroupsToConvComponents(groups)
	if len(result) != 3 {
		t.Fatalf("应有3个组件，实际%d", len(result))
	}
	if result[0].LibraryType != "pedagogy" {
		t.Errorf("第一个组件类型应为pedagogy，实际%s", result[0].LibraryType)
	}
	if result[2].LibraryType != "activity_design" {
		t.Errorf("第三个组件类型应为activity_design，实际%s", result[2].LibraryType)
	}
}

// TestConvertGroupsToConvComponents_EmptyGroups 空组返回nil
func TestConvertGroupsToConvComponents_EmptyGroups(t *testing.T) {
	result := convertGroupsToConvComponents(nil)
	if result != nil {
		t.Error("nil输入应返回nil")
	}
	result2 := convertGroupsToConvComponents([]*models.MatchedComponentGroup{})
	if result2 != nil {
		t.Error("空切片应返回nil")
	}
}

// ==================== cleanComponentMarkers 测试 ====================

// TestCleanComponentMarkers 清除组件触发标记
func TestCleanComponentMarkers(t *testing.T) {
	// 测试【推荐组件】标记被移除
	input1 := "这是正文【推荐组件】继续正文"
	result1 := cleanComponentMarkers(input1)
	if strings.Contains(result1, "【推荐组件】") {
		t.Error("应移除【推荐组件】标记")
	}

	// 测试"推荐以下教学方案"被替换为带前缀的版本
	input2 := "推荐以下教学方案供参考"
	result2 := cleanComponentMarkers(input2)
	if !strings.Contains(result2, "根据你的情况") {
		t.Error("替换后应包含'根据你的情况'前缀")
	}

	// 测试无标记的文本不变
	input3 := "这是普通正文"
	result3 := cleanComponentMarkers(input3)
	if result3 != "这是普通正文" {
		t.Errorf("无标记文本不应改变，实际%q", result3)
	}
}

// ==================== formatSelectedOptions 测试 ====================

// TestFormatSelectedOptions_WithMessage 有原始消息时直接返回
func TestFormatSelectedOptions_WithMessage(t *testing.T) {
	result := formatSelectedOptions([]string{"a", "b"}, "我的原始消息")
	if result != "我的原始消息" {
		t.Errorf("有原始消息应直接返回，实际%s", result)
	}
}

// TestFormatSelectedOptions_WithoutMessage 无原始消息时格式化选项
func TestFormatSelectedOptions_WithoutMessage(t *testing.T) {
	result := formatSelectedOptions([]string{"选项A", "选项B"}, "")
	if !strings.Contains(result, "选项A") || !strings.Contains(result, "选项B") {
		t.Errorf("应包含所有选项，实际%s", result)
	}
	if !strings.HasPrefix(result, "我选择：") {
		t.Errorf("应以'我选择：'开头，实际%s", result)
	}
}

// ==================== formatSelectedComponents 测试 ====================

// TestFormatSelectedComponents_Empty 空列表返回空字符串
func TestFormatSelectedComponents_Empty(t *testing.T) {
	result := formatSelectedComponents([]string{})
	if result != "" {
		t.Errorf("空列表应返回空字符串，实际%q", result)
	}
}

// TestFormatSelectedComponents_WithIDs 有ID时显示数量
func TestFormatSelectedComponents_WithIDs(t *testing.T) {
	result := formatSelectedComponents([]string{"id1", "id2", "id3"})
	if !strings.Contains(result, "3") {
		t.Errorf("应包含数量3，实际%s", result)
	}
}

// ==================== generateMsgID 测试 ====================

// TestGenerateMsgID_Format 消息ID格式正确
func TestGenerateMsgID_Format(t *testing.T) {
	id := generateMsgID()
	if !strings.HasPrefix(id, "msg_") {
		t.Errorf("消息ID应以msg_开头，实际%s", id)
	}
}

// TestGenerateMsgID_Unique 多次生成的ID不重复
func TestGenerateMsgID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateMsgID()
		if ids[id] {
			t.Errorf("消息ID重复: %s", id)
		}
		ids[id] = true
	}
}
