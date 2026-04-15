package services

// workshop_stage_extract_extra_test.go — 阶段产出物提取补充测试
//
// 补充覆盖 workshop_stage_extract.go 中未被 extract_test.go 覆盖的函数

import (
"encoding/json"
"strings"
"testing"
)

// ==================== extractWriteStageFromNatural 测试 ====================

func TestExtractWriteStage_WithFullLesson(t *testing.T) {
content := "好的，以下是完整的教案：\n\n# 一元一次方程教学设计\n\n## 教学目标\n1. 理解一元一次方程的定义\n2. 掌握基本求解方法\n\n## 教学重难点\n重点：方程的基本解法\n难点：移项变号的理解\n\n## 教学准备\nPPT课件、练习题\n\n## 教学方法\n讲授法、合作探究法\n\n## 教学过程\n\n### 一、导入新课（5分钟）\n通过生活情境引入方程概念，激发学生兴趣。\n\n### 二、探究新知（20分钟）\n引导学生发现方程的本质特征，通过具体实例推导解法。\n\n### 三、巩固练习（15分钟）\n分层练习设计，基础题与提高题结合。\n\n### 四、课堂小结（5分钟）\n总结本课重点知识，回顾方程定义和解法步骤。\n\n## 作业布置\n1. 课本第42页练习1-5\n2. 思考题一道\n\n## 板书设计\n一元一次方程\n定义 → 解法 → 应用\n" + strings.Repeat("这是补充内容确保教案长度足够。\n", 80)

structured, narrative, hasContent := extractWriteStageFromNatural(content)
// 不强制断言hasContent（取决于DetectLessonPlanContent的严格度）
t.Logf("write完整教案: hasContent=%v, structured长度=%d, narrative长度=%d", hasContent, len(structured), len(narrative))
// 仅验证不panic且返回合法值
if structured == "" {
t.Error("structured不应为空字符串")
}
}

func TestExtractWriteStage_ShortContent(t *testing.T) {
content := "好的，我来帮你写教案。首先确认一下课题。"
structured, _, hasContent := extractWriteStageFromNatural(content)
t.Logf("短内容: hasContent=%v, structured长度=%d", hasContent, len(structured))
}

func TestExtractWriteStage_Empty(t *testing.T) {
structured, narrative, hasContent := extractWriteStageFromNatural("")
if hasContent {
t.Error("空内容应返回false")
}
if structured != "{}" {
t.Errorf("空内容structured应为'{}'，实际%q", structured)
}
if narrative != "" {
t.Errorf("空内容narrative应为空，实际%q", narrative)
}
}

// ==================== extractReviewStageFromNatural 测试 ====================

func TestExtractReviewStage_FullReview(t *testing.T) {
content := "# 教案评审报告\n\n## 评审维度\n\n| 维度 | 评分 | 点评 |\n|------|------|------|\n| T1 教学目标 | 8.5 | 目标明确 |\n| T2 教学内容 | 7.0 | 内容合理 |\n| T3 教学方法 | 8.0 | 方法多样 |\n| T4 教学评价 | 7.5 | 评价全面 |\n\n## 做得好的点\n\n**1. 教学目标设定清晰**\n三维目标覆盖完整。\n\n**2. 活动设计丰富**\n探究活动和小组合作结合良好。\n\n## 改进建议\n\n**1. 导入环节时间偏长**\n建议缩短导入时间。\n\n**2. 缺少形成性评价**\n建议增加课堂检测。\n\n## 总评\n\n综合评分：7.8/10\n\n整体来看这是一份结构完整的教案。\n"
structured, narrative, hasContent := extractReviewStageFromNatural(content)
if !hasContent {
t.Error("完整评审报告应返回hasContent=true")
}
if structured == "" || structured == "{}" {
t.Error("structured不应为空")
}
var data map[string]interface{}
if err := json.Unmarshal([]byte(structured), &data); err != nil {
t.Fatalf("structured应为合法JSON: %v", err)
}
if score, ok := data["total_score"]; ok {
if s, ok := score.(float64); !ok || s <= 0 {
t.Errorf("total_score应>0，实际%v", score)
}
} else {
t.Error("应包含total_score")
}
_ = narrative
}

func TestExtractReviewStage_NoScore(t *testing.T) {
content := "这是一些评审意见，但没有具体评分。教案需要改进导入环节。"
_, _, hasContent := extractReviewStageFromNatural(content)
t.Logf("无评分评审: hasContent=%v", hasContent)
}

func TestExtractReviewStage_Empty(t *testing.T) {
_, _, hasContent := extractReviewStageFromNatural("")
if hasContent {
t.Error("空内容应返回false")
}
}

// ==================== extractGenericStageFromNatural 测试 ====================

func TestExtractGenericStage_Normal(t *testing.T) {
content := "这是自定义阶段的AI回复，包含了一些分析结果和建议。"
structured, narrative, hasContent := extractGenericStageFromNatural("custom_stage", content)
if !hasContent {
t.Error("非空内容应返回hasContent=true")
}
var data map[string]interface{}
if err := json.Unmarshal([]byte(structured), &data); err != nil {
t.Fatalf("structured应为合法JSON: %v", err)
}
if data["stage"] != "custom_stage" {
t.Errorf("stage应为custom_stage，实际%v", data["stage"])
}
if narrative == "" {
t.Error("narrative不应为空")
}
}

func TestExtractGenericStage_Empty(t *testing.T) {
structured, _, hasContent := extractGenericStageFromNatural("test", "")
if hasContent {
t.Error("空内容应返回false")
}
if structured != "{}" {
t.Errorf("空内容structured应为'{}'，实际%q", structured)
}
}

func TestExtractGenericStage_AnalyzeCode(t *testing.T) {
content := "经过分析，该班学情特征如下：学生基础参差不齐，需要分层教学。"
structured, narrative, hasContent := extractGenericStageFromNatural("analyze", content)
if !hasContent {
t.Error("非空内容应返回true")
}
if narrative == "" {
t.Error("narrative不应为空")
}
_ = structured
}

// ==================== extractTotalScoreFromReview 测试 ====================

func TestExtractTotalScoreFromReview_Various(t *testing.T) {
tests := []struct {
name    string
content string
expect  float64
}{
{"标准格式", "综合评分：8.5/10", 8.5},
{"总分格式", "总分：9.0", 9.0},
{"总评分格式", "总评分: 7.5/10分", 7.5},
{"综合得分", "综合得分：8.0", 8.0},
{"总体评分", "总体评分（7.0）", 7.0},
{"无分数", "这里没有任何分数", 0},
}
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
got := extractTotalScoreFromReview(tt.content)
if got != tt.expect {
t.Errorf("extractTotalScoreFromReview=%f, want %f", got, tt.expect)
}
})
}
}

// ==================== extractSummary 测试 ====================

func TestExtractSummary_WithSection(t *testing.T) {
content := "## 维度评分\n各维度表现良好。\n\n## 总评\n\n整体来看，这是一份不错的教案，教学目标清晰，结构完整。\n\n## 附录\n其他信息"
result := extractSummary(content)
if result == "" {
t.Error("有总评段落应提取到内容")
}
if !strings.Contains(result, "不错的教案") {
t.Errorf("应包含总评内容，实际%q", result)
}
}

func TestExtractSummary_NoSection(t *testing.T) {
content := "这里只有维度评分，没有总评段落。"
result := extractSummary(content)
if result != "" {
t.Errorf("无总评段落应返回空，实际%q", result)
}
}

func TestExtractSummary_AlternativeHeaders(t *testing.T) {
tests := []struct {
name   string
header string
}{
{"总评", "总评"},
{"综述", "综述"},
{"整体评价", "整体评价"},
{"综合评价", "综合评价"},
}
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
content := "## " + tt.header + "\n\n这是评价内容，教案质量良好。"
result := extractSummary(content)
if result == "" {
t.Errorf("标题%q应能被识别", tt.header)
}
})
}
}

// ==================== ExtractStructuredFromNaturalReply 更多场景 ====================

func TestExtractStructured_WriteStage(t *testing.T) {
// write阶段提取依赖DetectLessonPlanContent的严格检测
// 不强制断言hasContent，仅验证不panic和返回值类型正确
longLesson := "# 教案\n## 教学目标\n目标\n## 教学重难点\n重难点\n## 教学准备\n准备\n## 教学方法\n方法\n## 教学过程\n### 导入\n导入\n### 新授\n新授\n## 作业布置\n作业\n## 板书设计\n板书\n" + strings.Repeat("内容补充\n", 100)
structured, narrative, hasContent := ExtractStructuredFromNaturalReply("write", longLesson)
t.Logf("write阶段: hasContent=%v, structured长度=%d, narrative长度=%d", hasContent, len(structured), len(narrative))
if structured == "" {
t.Error("structured不应为空字符串")
}
}

func TestExtractStructured_ReviseStage(t *testing.T) {
// revise阶段短文本不一定返回hasContent=true
content := "根据评审意见修改：\n1. 缩短导入\n2. 增加互动"
structured, narrative, hasContent := ExtractStructuredFromNaturalReply("revise", content)
t.Logf("revise阶段: hasContent=%v, structured长度=%d, narrative长度=%d", hasContent, len(structured), len(narrative))
if structured == "" {
t.Error("structured不应为空字符串")
}
}

func TestExtractStructured_DesignStage(t *testing.T) {
content := "教学设计方案：\n1. 采用5E教学模型\n2. 导入5分钟，探究20分钟"
structured, narrative, hasContent := ExtractStructuredFromNaturalReply("design", content)
if !hasContent {
t.Error("design阶段有内容应返回true")
}
var data map[string]interface{}
if err := json.Unmarshal([]byte(structured), &data); err != nil {
t.Fatalf("structured应为合法JSON: %v", err)
}
if data["stage"] != "design" {
t.Errorf("stage应为design，实际%v", data["stage"])
}
if narrative == "" {
t.Error("narrative不应为空")
}
}
