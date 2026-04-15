package services

// workshop_stage_extract_test.go — 阶段产出物自然语言提取单元测试
//
// v89-5新增：覆盖 workshop_stage_extract.go 中核心导出函数
//
// 测试范围：
//   - ExtractStructuredFromNaturalReply: 各阶段提取入口
//   - DetectLessonPlanContent: 教案Markdown检测
//   - extractScoreFromText: 分数提取
//   - extractDimensionsFromTable: 表格维度提取
//   - extractGoodPoints: 亮点提取
//   - extractImprovements: 改进建议提取

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==================== DetectLessonPlanContent 测试 ====================

func TestDetectLessonPlanContent_Valid(t *testing.T) {
	// 构造包含足够教案标记的完整教案内容
	lessonPlan := `好的，以下是完整教案：

# 一元一次方程教学设计

## 教学目标
1. 理解一元一次方程的定义
2. 掌握基本求解方法

## 教学重难点
重点：方程的基本解法
难点：移项变号的理解

## 教学准备
PPT课件、练习题

## 教学方法
讲授法、讨论法

## 教学过程

### 一、导入新课（5分钟）
通过生活情境引入方程概念...展开讨论...
详细说明导入环节的具体操作步骤和内容安排。

### 二、探究新知（20分钟）
引导学生发现方程的本质...
通过具体的数学问题，让学生体会方程的作用和意义。
详细展开教学过程中的每一个步骤和环节。

### 三、巩固练习（15分钟）
分层练习设计...
基础题、提高题、拓展题三个层次的练习安排。

### 四、课堂小结（5分钟）
总结本课重点...回顾方程的定义和解法。

## 作业布置
1. 课本第42页练习1-5
2. 思考题：生活中哪些问题可以用方程解决

## 板书设计
一元一次方程
定义 → 解法 → 应用
` + strings.Repeat("补充教案内容以确保长度足够。这是一份完整的数学教案，包含了所有必要的教学环节。\n", 50)

	result := DetectLessonPlanContent(lessonPlan)
	if result == "" {
		t.Error("应检测到教案内容，但返回空")
	}
	// 检测结果应包含关键教案元素
	if !strings.Contains(result, "教学目标") {
		t.Error("提取结果应包含教学目标")
	}
	if !strings.Contains(result, "教学过程") {
		t.Error("提取结果应包含教学过程")
	}
}

func TestDetectLessonPlanContent_TooFewMarkers(t *testing.T) {
	// 标记不足5个，应返回空
	content := "# 教学目标\n理解方程\n## 教学重点\n解方程"
	result := DetectLessonPlanContent(content)
	if result != "" {
		t.Errorf("标记不足应返回空，实际返回%d字符", len(result))
	}
}

func TestDetectLessonPlanContent_Empty(t *testing.T) {
	if DetectLessonPlanContent("") != "" {
		t.Error("空输入应返回空")
	}
}

func TestDetectLessonPlanContent_NoProcess(t *testing.T) {
	// 有足够标记但缺少"教学过程"
	content := `# 教学设计
## 教学目标
目标内容
## 教学重点
重点内容
## 教学难点
难点内容
## 教学准备
准备内容
## 教学方法
方法内容
## 作业布置
作业内容
## 板书设计
板书内容`
	result := DetectLessonPlanContent(content)
	if result != "" {
		t.Error("缺少教学过程应返回空")
	}
}

// ==================== extractScoreFromText 测试 ====================

func TestExtractScoreFromText_Various(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		expected float64
	}{
		{"标准格式", "总分：8.5/10", []string{"总分"}, 8.5},
		{"带冒号", "总评分: 7.0", []string{"总评分"}, 7.0},
		{"带括号", "综合评分（8.0）", []string{"综合评分"}, 8.0},
		{"中文冒号", "总分：9.0/10分", []string{"总分"}, 9.0},
		{"整数分", "总评分：8", []string{"总评分"}, 8.0},
		{"无匹配", "这里没有分数", []string{"总分"}, 0},
		{"分数超范围", "总分：15.0", []string{"总分"}, 0},
		{"多关键词命中第二个", "综合得分：7.5", []string{"总分", "综合得分"}, 7.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractScoreFromText(tt.text, tt.keywords)
			if got != tt.expected {
				t.Errorf("extractScoreFromText(%s) = %.1f, want %.1f", tt.name, got, tt.expected)
			}
		})
	}
}

// ==================== extractDimensionsFromTable 测试 ====================

func TestExtractDimensionsFromTable_Standard(t *testing.T) {
	content := `## 评审维度

| 维度 | 评分 | 点评 |
|------|------|------|
| T1 教学目标 | 8.5 | 目标明确 |
| T2 教学内容 | 7.0 | 内容合理 |
| T3 教学方法 | 8.0 | 方法得当 |
| T4 教学评价 | 7.5 | 评价到位 |
`
	dims := extractDimensionsFromTable(content)
	if len(dims) != 4 {
		t.Errorf("应提取4个维度，实际%d个", len(dims))
		return
	}
	// 验证第一个维度
	if dims[0]["name"] != "教学目标" {
		t.Errorf("第一个维度名称应为教学目标，实际%v", dims[0]["name"])
	}
	if dims[0]["score"] != 8.5 {
		t.Errorf("第一个维度分数应为8.5，实际%v", dims[0]["score"])
	}
	if dims[0]["code"] != "T1" {
		t.Errorf("第一个维度代码应为T1，实际%v", dims[0]["code"])
	}
}

func TestExtractDimensionsFromTable_Empty(t *testing.T) {
	dims := extractDimensionsFromTable("没有表格的内容")
	if len(dims) != 0 {
		t.Errorf("无表格应返回空，实际%d个", len(dims))
	}
}

// ==================== extractGoodPoints 测试 ====================

func TestExtractGoodPoints_Standard(t *testing.T) {
	content := `## 做得好的点

**1. 教学目标设定合理**
目标明确，层次分明，符合课标要求。

**2. 教学活动设计丰富**
活动形式多样，能有效调动学生积极性。

## 改进建议
`
	points := extractGoodPoints(content)
	if len(points) < 2 {
		t.Errorf("应提取至少2个亮点，实际%d个", len(points))
	}
}

func TestExtractGoodPoints_NoSection(t *testing.T) {
	points := extractGoodPoints("这里没有亮点章节")
	if len(points) != 0 {
		t.Errorf("无亮点章节应返回空")
	}
}

// ==================== extractImprovements 测试 ====================

func TestExtractImprovements_Standard(t *testing.T) {
	content := `## 改进建议

**1. 导入环节可以更紧凑**
建议将导入时间从10分钟缩短到5分钟，增加练习时间。

**2. 评价方式需要多元化**
当前仅有课堂练习，建议增加小组互评和自评环节。

## 总评
`
	imps := extractImprovements(content)
	if len(imps) < 2 {
		t.Errorf("应提取至少2条改进建议，实际%d条", len(imps))
		return
	}
	// 验证结构
	if _, ok := imps[0]["id"]; !ok {
		t.Error("改进建议应包含id字段")
	}
	if _, ok := imps[0]["issue"]; !ok {
		t.Error("改进建议应包含issue字段")
	}
	if _, ok := imps[0]["suggestion"]; !ok {
		t.Error("改进建议应包含suggestion字段")
	}
}

// ==================== ExtractStructuredFromNaturalReply 综合测试 ====================

func TestExtractStructuredFromNaturalReply_Generic(t *testing.T) {
	// 通用阶段(analyze/design)返回stage+summary
	structured, narrative, hasContent := ExtractStructuredFromNaturalReply("analyze", "这是一段分析内容，讨论了学情特征和教学需求。")
	if !hasContent {
		t.Error("非空内容应返回hasContent=true")
	}
	if narrative == "" {
		t.Error("narrative不应为空")
	}
	// structured应是合法JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(structured), &data); err != nil {
		t.Errorf("structured应为合法JSON: %v", err)
	}
	if data["stage"] != "analyze" {
		t.Errorf("stage应为analyze，实际%v", data["stage"])
	}
}

func TestExtractStructuredFromNaturalReply_EmptyContent(t *testing.T) {
	structured, _, hasContent := ExtractStructuredFromNaturalReply("analyze", "")
	if hasContent {
		t.Error("空内容应返回hasContent=false")
	}
	if structured != "{}" {
		t.Errorf("空内容structured应为{}，实际%s", structured)
	}
}

func TestExtractStructuredFromNaturalReply_ReviewWithScore(t *testing.T) {
	// review阶段有评审总分时应提取成功
	reviewContent := `# 教案评审报告

## 评审维度

| 维度 | 评分 | 点评 |
|------|------|------|
| T1 教学目标 | 8.5 | 目标明确 |
| T2 教学内容 | 7.0 | 内容合理 |
| T3 教学方法 | 8.0 | 方法得当 |
| T4 教学评价 | 7.5 | 评价到位 |

## 做得好的点

**1. 目标设定合理**
目标清晰，层次分明。

## 改进建议

**1. 时间分配需优化**
建议调整各环节时间。

## 总评

综合评分：7.8/10

整体来看，这是一份较为完整的教案。
`
	structured, _, hasContent := ExtractStructuredFromNaturalReply("review", reviewContent)
	if !hasContent {
		t.Error("有总分的评审应返回hasContent=true")
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(structured), &data); err != nil {
		t.Errorf("structured应为合法JSON: %v", err)
	}
	if score, ok := data["total_score"]; ok {
		if s, ok := score.(float64); !ok || s <= 0 {
			t.Errorf("total_score应>0，实际%v", score)
		}
	} else {
		t.Error("应包含total_score字段")
	}
}

// ==================== trimTrailingChatter 测试 ====================

func TestTrimTrailingChatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // 结果应包含
		notContains string // 结果不应包含
	}{
		{
			"去除客套话",
			"教案内容正文\n\n如果您有任何修改意见请告诉我",
			"教案内容正文",
			"如果您有任何",
		},
		{
			"去除以上是",
			"教案正文内容\n\n以上是完整教案",
			"教案正文内容",
			"以上是",
		},
		{
			"保留正常内容",
			"这是正常的教案内容\n没有客套话",
			"没有客套话",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimTrailingChatter(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("结果应包含 %q", tt.contains)
			}
			if tt.notContains != "" && strings.Contains(result, tt.notContains) {
				t.Errorf("结果不应包含 %q", tt.notContains)
			}
		})
	}
}
