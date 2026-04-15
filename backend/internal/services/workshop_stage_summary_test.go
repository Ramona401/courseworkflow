package services

// workshop_stage_summary_test.go — 阶段摘要生成单元测试
//
// v89-5新增：覆盖 workshop_stage_summary.go 中核心函数

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

// ==================== 测试辅助：构造对话消息 ====================

func makeMsg(role models.ConversationRole, content string) *models.ConversationMessage {
	return &models.ConversationMessage{
		Role:    role,
		Content: content,
	}
}

func makeUserMsg(content string) *models.ConversationMessage {
	return makeMsg(models.ConvRoleUser, content)
}

func makeAIMsg(content string) *models.ConversationMessage {
	return makeMsg(models.ConvRoleAssistant, content)
}

// ==================== getLastAIReply 测试 ====================

func TestGetLastAIReply_Normal(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("你好"),
		makeAIMsg("你好！我是AI助手"),
		makeUserMsg("帮我分析学情"),
		makeAIMsg("好的，我来分析一下这个班级的学情特征..."),
	}
	got := getLastAIReply(msgs)
	if !strings.Contains(got, "学情特征") {
		t.Errorf("应返回最后一条AI回复，实际: %s", got)
	}
}

func TestGetLastAIReply_NoAI(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("你好"),
		makeUserMsg("再说一次"),
	}
	if got := getLastAIReply(msgs); got != "" {
		t.Errorf("无AI回复应返回空，实际: %s", got)
	}
}

func TestGetLastAIReply_Empty(t *testing.T) {
	if got := getLastAIReply(nil); got != "" {
		t.Errorf("nil消息应返回空")
	}
}

// ==================== countDialogueRounds 测试 ====================

func TestCountDialogueRounds(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("你好"),
		makeAIMsg("你好！"),
		makeUserMsg("分析学情"),
		makeAIMsg("好的..."),
		makeUserMsg("继续"),
	}
	if got := countDialogueRounds(msgs); got != 3 {
		t.Errorf("应有3轮对话，实际%d轮", got)
	}
}

func TestCountDialogueRounds_Empty(t *testing.T) {
	if got := countDialogueRounds(nil); got != 0 {
		t.Errorf("空消息应为0轮")
	}
}

// ==================== GenerateStageSummary 测试 ====================

func TestGenerateStageSummary_Analyze(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("这个班级有35个学生，数学基础参差不齐，有5个学生跟不上进度"),
		makeAIMsg("根据您的描述，这个班级的学情特征如下：1.班级规模适中 2.学习差异较大 3.需要分层教学策略"),
		makeUserMsg("对，而且他们对应用题特别头疼"),
		makeAIMsg("了解了，应用题薄弱是这个年龄段常见问题。建议在教学设计中增加情境化的应用题练习环节。"),
	}
	summary := GenerateStageSummary("analyze", msgs, "{}")
	if summary == "" {
		t.Error("analyze摘要不应为空")
	}
	if !strings.Contains(summary, "教师提供的信息") {
		t.Error("应包含教师信息标记")
	}
	if len([]rune(summary)) > 510 {
		t.Errorf("摘要超过500字限制，实际%d字", len([]rune(summary)))
	}
}

func TestGenerateStageSummary_Design(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("我想用探究式教学"),
		makeAIMsg("探究式教学非常适合这个课题。我建议采用5E教学模型。"),
		makeUserMsg("好的，就用5E模型"),
		makeAIMsg("确认采用5E教学模型。设计方案如下：导入5分钟，探究20分钟，讲解10分钟，练习10分钟。"),
	}
	summary := GenerateStageSummary("design", msgs, "{}")
	if summary == "" {
		t.Error("design摘要不应为空")
	}
	if !strings.Contains(summary, "设计方案") {
		t.Error("应包含设计方案标记")
	}
}

func TestGenerateStageSummary_Write(t *testing.T) {
	structuredOutput := `{"content_markdown":"# 教案内容\n很长的教案..."}`
	msgs := []*models.ConversationMessage{
		makeUserMsg("开始写教案"),
		makeAIMsg("好的，我先展示框架..."),
	}
	summary := GenerateStageSummary("write", msgs, structuredOutput)
	if summary == "" {
		t.Error("write摘要不应为空")
	}
	if !strings.Contains(summary, "完成") && !strings.Contains(summary, "对话") {
		t.Errorf("write摘要格式异常: %s", summary)
	}
}

func TestGenerateStageSummary_Review(t *testing.T) {
	structuredOutput := `{"total_score":8.5,"improvements":[{"id":"imp_1","issue":"导入过长"},{"id":"imp_2","issue":"评价单一"}],"summary":"整体良好"}`
	msgs := []*models.ConversationMessage{
		makeAIMsg("评审报告...总分8.5"),
	}
	summary := GenerateStageSummary("review", msgs, structuredOutput)
	if summary == "" {
		t.Error("review摘要不应为空")
	}
	if !strings.Contains(summary, "8.5") {
		t.Error("review摘要应包含评审分数")
	}
	if !strings.Contains(summary, "2条") {
		t.Error("review摘要应包含改进建议数量")
	}
}

func TestGenerateStageSummary_EmptyMessages(t *testing.T) {
	summary := GenerateStageSummary("review", nil, `{"total_score":7.0}`)
	if summary == "" {
		t.Error("有structured_output时摘要不应为空")
	}
}

func TestGenerateStageSummary_AllEmpty(t *testing.T) {
	summary := GenerateStageSummary("analyze", nil, "{}")
	if summary != "" {
		t.Errorf("全空应返回空摘要，实际: %s", summary)
	}
}

func TestGenerateStageSummary_Generic(t *testing.T) {
	msgs := []*models.ConversationMessage{
		makeUserMsg("自定义问题"),
		makeAIMsg("这是自定义阶段的AI回复内容，包含了一些有价值的分析。"),
	}
	summary := GenerateStageSummary("custom_stage", msgs, "{}")
	if summary == "" {
		t.Error("通用阶段摘要不应为空")
	}
}

// ==================== stageCodeToNameForSummary 测试 ====================

func TestStageCodeToNameForSummary(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"analyze", "教学分析"},
		{"design", "教学设计"},
		{"write", "教案撰写"},
		{"review", "AI评审"},
		{"revise", "修订定稿"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := stageCodeToNameForSummary(tt.code)
		if got != tt.expected {
			t.Errorf("stageCodeToNameForSummary(%q) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}
