package services

// workshop_stage_summary_extra_test.go — 阶段摘要生成补充测试

import (
"strings"
"testing"

"tedna/internal/models"
)

// ==================== extractSummaryFromStructuredOutput 测试 ====================

func TestExtractSummaryFromStructured_WithSummary(t *testing.T) {
output := `{"summary": "教案整体质量良好", "total_score": 8.0}`
result := extractSummaryFromStructuredOutput("review", output)
if result != "教案整体质量良好" {
t.Errorf("应优先返回summary字段，实际%q", result)
}
}

func TestExtractSummaryFromStructured_WithContentMarkdown(t *testing.T) {
output := `{"content_markdown": "# 教案内容\n这是一份完整教案"}`
result := extractSummaryFromStructuredOutput("write", output)
if !strings.Contains(result, "教案撰写") {
t.Errorf("应包含阶段名称，实际%q", result)
}
if !strings.Contains(result, "字符") {
t.Errorf("应包含字符数信息，实际%q", result)
}
}

func TestExtractSummaryFromStructured_WithTotalScore(t *testing.T) {
output := `{"total_score": 8.5}`
result := extractSummaryFromStructuredOutput("review", output)
if !strings.Contains(result, "8.5") {
t.Errorf("应包含评审分数，实际%q", result)
}
}

func TestExtractSummaryFromStructured_Empty(t *testing.T) {
if extractSummaryFromStructuredOutput("review", "{}") != "" {
t.Error("空JSON应返回空")
}
if extractSummaryFromStructuredOutput("review", "") != "" {
t.Error("空字符串应返回空")
}
}

func TestExtractSummaryFromStructured_InvalidJSON(t *testing.T) {
if extractSummaryFromStructuredOutput("review", "not json") != "" {
t.Error("无效JSON应返回空")
}
}

// ==================== generateAnalyzeStageSummary 更多场景 ====================

func TestGenerateAnalyzeSummary_OnlyAIReply(t *testing.T) {
msgs := []*models.ConversationMessage{
makeAIMsg("根据课标分析，本课应聚焦概念理解。"),
}
summary := generateAnalyzeStageSummary(msgs)
t.Logf("仅AI回复的analyze摘要: %q", summary)
}

func TestGenerateAnalyzeSummary_MultiRound(t *testing.T) {
msgs := []*models.ConversationMessage{
makeUserMsg("班上有35个学生"),
makeAIMsg("了解，35人的班级"),
makeUserMsg("数学基础参差不齐"),
makeAIMsg("需要分层教学"),
makeUserMsg("对的"),
makeAIMsg("建议增加辅导环节"),
}
summary := generateAnalyzeStageSummary(msgs)
if summary == "" {
t.Error("多轮对话应生成非空摘要")
}
}

// ==================== generateDesignStageSummary 更多场景 ====================

func TestGenerateDesignSummary_Normal(t *testing.T) {
msgs := []*models.ConversationMessage{
makeUserMsg("我想用探究式教学"),
makeAIMsg("好的，探究式教学适合本课题。设计方案如下：导入5分钟。"),
}
summary := generateDesignStageSummary(msgs)
if summary == "" {
t.Error("设计阶段摘要不应为空")
}
}

// ==================== generateWriteStageSummary 更多场景 ====================

func TestGenerateWriteSummary_WithStructured(t *testing.T) {
msgs := []*models.ConversationMessage{
makeUserMsg("开始写教案"),
makeAIMsg("好的，这是教案框架..."),
}
structuredOutput := `{"content_markdown": "# 完整教案\n内容..."}`
summary := generateWriteStageSummary("write", msgs, structuredOutput)
if summary == "" {
t.Error("write摘要不应为空")
}
}

func TestGenerateWriteSummary_Empty(t *testing.T) {
summary := generateWriteStageSummary("write", nil, "{}")
t.Logf("空write摘要: %q", summary)
}

// ==================== generateReviewStageSummary 更多场景 ====================

func TestGenerateReviewSummary_WithStructured(t *testing.T) {
msgs := []*models.ConversationMessage{
makeAIMsg("评审报告：总分8.0"),
}
structuredOutput := `{"total_score":8.0,"improvements":[{"id":"1"},{"id":"2"},{"id":"3"}],"summary":"整体良好"}`
summary := generateReviewStageSummary(msgs, structuredOutput)
if summary == "" {
t.Error("review摘要不应为空")
}
if !strings.Contains(summary, "8.0") {
t.Error("应包含评审分数")
}
}

func TestGenerateReviewSummary_NoStructured(t *testing.T) {
msgs := []*models.ConversationMessage{
makeAIMsg("评审结果：总体良好。"),
}
summary := generateReviewStageSummary(msgs, "{}")
t.Logf("无structured的review摘要: %q", summary)
}

// ==================== generateGenericStageSummary 更多场景 ====================

func TestGenerateGenericSummary_Normal(t *testing.T) {
msgs := []*models.ConversationMessage{
makeUserMsg("自定义问题"),
makeAIMsg("这是详细回复。"),
}
summary := generateGenericStageSummary(msgs)
if summary == "" {
t.Error("通用阶段摘要不应为空")
}
}

func TestGenerateGenericSummary_Empty(t *testing.T) {
summary := generateGenericStageSummary(nil)
// 实际行为：即使空消息也会生成"阶段对话0轮"格式的基本摘要
t.Logf("空消息通用摘要: %q", summary)
}

// ==================== GenerateStageSummary 更多边界场景 ====================

func TestGenerateStageSummary_ReviseStageExtra(t *testing.T) {
msgs := []*models.ConversationMessage{
makeUserMsg("按评审建议修改"),
makeAIMsg("好的，已修订。"),
}
summary := GenerateStageSummary("revise", msgs, "{}")
if summary == "" {
t.Error("revise摘要不应为空")
}
}

func TestGenerateStageSummary_LongMessagesExtra(t *testing.T) {
longMsg := strings.Repeat("这是很长的消息", 100)
msgs := []*models.ConversationMessage{
makeUserMsg(longMsg),
makeAIMsg(longMsg),
}
summary := GenerateStageSummary("analyze", msgs, "{}")
if len([]rune(summary)) > 510 {
t.Errorf("摘要应不超过500字，实际%d字", len([]rune(summary)))
}
}

func TestGenerateStageSummary_OnlyStructuredExtra(t *testing.T) {
summary := GenerateStageSummary("write", nil, `{"content_markdown":"# 教案\n内容..."}`)
if summary == "" {
t.Error("有structured时摘要不应为空")
}
}
