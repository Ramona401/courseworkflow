package utils

// aoci_component_test.go — 组件AOCI索引解析函数单元测试
//
// v89-5新增：覆盖 aoci_component.go 中所有导出函数
//
// 测试范围：
//   - ParseIndexField: 编码行字段提取
//   - ParseCognitiveLevel: 认知层级解析
//   - ParseStageTiming: 课堂时机解析
//   - ParsePedagogyIntensity: 教法强度解析
//   - FormatIndexForPrompt: AI注入格式化
//   - BuildComponentFullText: 组件全文构建

import (
	"strings"
	"testing"
)

// ==================== 测试数据 ====================

// 标准索引文本样例（模拟真实AI压缩输出）
const sampleComponentIndex = `LT:PD|SJ:MA|GR:7-9|CG:4|TM:2|PQ:3
[F]用于新概念深化,通过问题链驱动探究
[T]方程求解,几何证明
[P]探究式,问题链
[D]需要学生有基本推理能力
[C]可与知识图谱库配合使用`

// 最简索引（只有编码行，无语义标签）
const minimalComponentIndex = `LT:CS|SJ:GN|GR:1-9|CG:2|TM:4|PQ:1`

// ==================== ParseIndexField 测试 ====================

func TestParseIndexField_Standard(t *testing.T) {
	// 测试从标准索引中提取各字段
	tests := []struct {
		field    string
		expected string
	}{
		{"LT", "PD"},
		{"SJ", "MA"},
		{"GR", "7-9"},
		{"CG", "4"},
		{"TM", "2"},
		{"PQ", "3"},
	}
	for _, tt := range tests {
		got := ParseIndexField(sampleComponentIndex, tt.field)
		if got != tt.expected {
			t.Errorf("ParseIndexField(%q) = %q, want %q", tt.field, got, tt.expected)
		}
	}
}

func TestParseIndexField_Empty(t *testing.T) {
	// 空输入应返回空字符串
	if got := ParseIndexField("", "CG"); got != "" {
		t.Errorf("ParseIndexField空输入 = %q, want empty", got)
	}
}

func TestParseIndexField_NotFound(t *testing.T) {
	// 不存在的字段应返回空字符串
	if got := ParseIndexField(sampleComponentIndex, "XX"); got != "" {
		t.Errorf("ParseIndexField不存在字段 = %q, want empty", got)
	}
}

func TestParseIndexField_SingleLine(t *testing.T) {
	// 只有一行编码（无语义标签行）
	got := ParseIndexField(minimalComponentIndex, "TM")
	if got != "4" {
		t.Errorf("ParseIndexField单行索引TM = %q, want '4'", got)
	}
}

// ==================== ParseCognitiveLevel 测试 ====================

func TestParseCognitiveLevel_Valid(t *testing.T) {
	// 有效值1-6
	tests := []struct {
		input    string
		expected int
	}{
		{sampleComponentIndex, 4},       // CG:4
		{minimalComponentIndex, 2},      // CG:2
		{"LT:PD|CG:1|TM:1|PQ:1", 1},   // 边界值：最小
		{"LT:PD|CG:6|TM:1|PQ:1", 6},   // 边界值：最大
	}
	for _, tt := range tests {
		got := ParseCognitiveLevel(tt.input)
		if got != tt.expected {
			t.Errorf("ParseCognitiveLevel(%q...) = %d, want %d", tt.input[:20], got, tt.expected)
		}
	}
}

func TestParseCognitiveLevel_Invalid(t *testing.T) {
	// 无效值应返回0
	tests := []struct {
		name  string
		input string
	}{
		{"空输入", ""},
		{"无CG字段", "LT:PD|SJ:MA|TM:2"},
		{"CG超范围", "LT:PD|CG:7|TM:1"},
		{"CG为0", "LT:PD|CG:0|TM:1"},
		{"CG非数字", "LT:PD|CG:abc|TM:1"},
		{"CG为负数", "LT:PD|CG:-1|TM:1"},
	}
	for _, tt := range tests {
		got := ParseCognitiveLevel(tt.input)
		if got != 0 {
			t.Errorf("ParseCognitiveLevel(%s) = %d, want 0", tt.name, got)
		}
	}
}

// ==================== ParseStageTiming 测试 ====================

func TestParseStageTiming_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{sampleComponentIndex, 2},       // TM:2
		{minimalComponentIndex, 4},      // TM:4
		{"LT:PD|CG:1|TM:1|PQ:1", 1},   // 边界值：开场
		{"LT:PD|CG:1|TM:3|PQ:1", 3},   // 收尾
		{"LT:PD|CG:1|TM:4|PQ:1", 4},   // 贯穿
	}
	for _, tt := range tests {
		got := ParseStageTiming(tt.input)
		if got != tt.expected {
			t.Errorf("ParseStageTiming = %d, want %d", got, tt.expected)
		}
	}
}

func TestParseStageTiming_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空输入", ""},
		{"无TM字段", "LT:PD|CG:4|PQ:2"},
		{"TM超范围", "LT:PD|TM:5|CG:1"},
		{"TM为0", "LT:PD|TM:0|CG:1"},
	}
	for _, tt := range tests {
		got := ParseStageTiming(tt.input)
		if got != 0 {
			t.Errorf("ParseStageTiming(%s) = %d, want 0", tt.name, got)
		}
	}
}

// ==================== ParsePedagogyIntensity 测试 ====================

func TestParsePedagogyIntensity_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{sampleComponentIndex, 3},       // PQ:3
		{minimalComponentIndex, 1},      // PQ:1
		{"LT:PD|CG:1|TM:1|PQ:2", 2},   // 特定
	}
	for _, tt := range tests {
		got := ParsePedagogyIntensity(tt.input)
		if got != tt.expected {
			t.Errorf("ParsePedagogyIntensity = %d, want %d", got, tt.expected)
		}
	}
}

func TestParsePedagogyIntensity_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空输入", ""},
		{"无PQ字段", "LT:PD|CG:4|TM:2"},
		{"PQ超范围", "LT:PD|PQ:4|CG:1"},
		{"PQ为0", "LT:PD|PQ:0|CG:1"},
	}
	for _, tt := range tests {
		got := ParsePedagogyIntensity(tt.input)
		if got != 0 {
			t.Errorf("ParsePedagogyIntensity(%s) = %d, want 0", tt.name, got)
		}
	}
}

// ==================== FormatIndexForPrompt 测试 ====================

func TestFormatIndexForPrompt_Standard(t *testing.T) {
	// 标准索引应包含编码短标识 + 展示名 + 语义标签
	result := FormatIndexForPrompt(sampleComponentIndex, "探究式教学——问题链驱动")

	// 检查编码短标识
	if !strings.Contains(result, "PD-MA-7-9") {
		t.Errorf("缺少编码短标识PD-MA-7-9，实际: %s", result)
	}
	// 检查展示名
	if !strings.Contains(result, "探究式教学——问题链驱动") {
		t.Errorf("缺少展示名")
	}
	// 检查语义标签
	for _, tag := range []string{"[F]", "[T]", "[P]", "[D]", "[C]"} {
		if !strings.Contains(result, tag) {
			t.Errorf("缺少语义标签 %s", tag)
		}
	}
	// 检查以▸开头
	if !strings.HasPrefix(result, "▸") {
		t.Errorf("应以▸开头")
	}
}

func TestFormatIndexForPrompt_Empty(t *testing.T) {
	// 空索引应返回"未索引"提示
	result := FormatIndexForPrompt("", "某组件名")
	if !strings.Contains(result, "未索引") {
		t.Errorf("空索引应返回未索引提示，实际: %s", result)
	}
	if !strings.Contains(result, "某组件名") {
		t.Errorf("应包含组件名")
	}
}

func TestFormatIndexForPrompt_MinimalIndex(t *testing.T) {
	// 只有编码行无语义标签的索引
	result := FormatIndexForPrompt(minimalComponentIndex, "通用组件")
	if !strings.Contains(result, "CS-GN-1-9") {
		t.Errorf("缺少编码短标识")
	}
	if !strings.Contains(result, "通用组件") {
		t.Errorf("缺少展示名")
	}
}

// ==================== BuildComponentFullText 测试 ====================

func TestBuildComponentFullText_Complete(t *testing.T) {
	// 完整字段
	result := BuildComponentFullText(
		"pedagogy", "数学", "7-9",
		"探究式教学", "通过问题链驱动", "完整指引内容", "案例片段",
	)
	checks := []string{"库类型=pedagogy", "学科=数学", "学段=7-9", "名称=探究式教学",
		"设计逻辑=通过问题链驱动", "完整指引=完整指引内容", "参考案例=案例片段"}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("缺少: %s", check)
		}
	}
}

func TestBuildComponentFullText_EmptyOptional(t *testing.T) {
	// 可选字段为空时不应输出对应行
	result := BuildComponentFullText("pedagogy", "数学", "7-9", "探究式教学", "", "", "")
	if strings.Contains(result, "设计逻辑=") {
		t.Errorf("空设计逻辑不应输出")
	}
	if strings.Contains(result, "完整指引=") {
		t.Errorf("空完整指引不应输出")
	}
	if strings.Contains(result, "参考案例=") {
		t.Errorf("空参考案例不应输出")
	}
	// 必填字段仍应存在
	if !strings.Contains(result, "名称=探究式教学") {
		t.Errorf("缺少必填字段名称")
	}
}

func TestBuildComponentFullText_LongGuide(t *testing.T) {
	// 超长全指引应被截断到1500字
	longGuide := strings.Repeat("这是测试内容", 500) // 3000字
	result := BuildComponentFullText("pedagogy", "数学", "7-9", "测试", "", longGuide, "")
	if !strings.Contains(result, "已截断") {
		t.Errorf("超长指引应被截断")
	}
}

// ==================== 编码映射表完整性测试 ====================

func TestLibraryTypeCodeMap_Completeness(t *testing.T) {
	// 验证13种库类型都有编码
	expectedTypes := []string{
		"curriculum_standard", "knowledge_graph", "student_profile",
		"pedagogy", "assessment_strategy", "activity_design",
		"questioning_strategy", "cross_subject", "teaching_tool",
		"scenario_material", "quality_rubric", "design_defect", "review_rubric",
	}
	for _, lt := range expectedTypes {
		if _, ok := LibraryTypeCodeMap[lt]; !ok {
			t.Errorf("LibraryTypeCodeMap缺少: %s", lt)
		}
	}
	if len(LibraryTypeCodeMap) != 13 {
		t.Errorf("LibraryTypeCodeMap应有13项，实际%d项", len(LibraryTypeCodeMap))
	}
}

func TestSubjectCodeMap_Completeness(t *testing.T) {
	// 验证14种学科都有编码
	if len(SubjectCodeMap) != 14 {
		t.Errorf("SubjectCodeMap应有14项，实际%d项", len(SubjectCodeMap))
	}
	// 检查关键学科
	for _, subject := range []string{"general", "数学", "语文", "英语", "科学", "AI"} {
		if _, ok := SubjectCodeMap[subject]; !ok {
			t.Errorf("SubjectCodeMap缺少: %s", subject)
		}
	}
}

func TestCognitiveLevelNameMap_Completeness(t *testing.T) {
	if len(CognitiveLevelNameMap) != 6 {
		t.Errorf("CognitiveLevelNameMap应有6项，实际%d项", len(CognitiveLevelNameMap))
	}
}

func TestStageTimingNameMap_Completeness(t *testing.T) {
	if len(StageTimingNameMap) != 4 {
		t.Errorf("StageTimingNameMap应有4项，实际%d项", len(StageTimingNameMap))
	}
}

func TestPedagogyIntensityNameMap_Completeness(t *testing.T) {
	if len(PedagogyIntensityNameMap) != 3 {
		t.Errorf("PedagogyIntensityNameMap应有3项，实际%d项", len(PedagogyIntensityNameMap))
	}
}
