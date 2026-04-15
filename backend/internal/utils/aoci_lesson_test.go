package utils

// aoci_lesson_test.go — 教案AOCI索引解析+质量计算单元测试
//
// v89-5新增：覆盖 aoci_lesson.go 中所有导出函数
//
// 测试范围：
//   - ParseLessonIndexField: 教案索引字段提取
//   - ParseLessonCognitiveLevel / ParseLessonPedagogyIntensity: 冗余列解析
//   - ParseStructureType / ParseQualityLevel: 结构类型和质量等级解析
//   - CalculateQualityLevel: 质量等级计算（核心业务逻辑）
//   - FormatLessonIndexForPrompt: 教案索引格式化
//   - BuildLessonFullText: 教案全文构建
//   - ValidateLessonIndex: 索引格式验证

import (
	"strings"
	"testing"
)

// ==================== 测试数据 ====================

// 标准教案索引样例
const sampleLessonIndex = `SJ:MA|GR:7|CG:4|PQ:2|ST:2|QL:4
[O]掌握一元一次方程的基本解法
[S]导入(5min)-探究(20min)-练习(15min)-总结(5min)
[M]探究式教学,问题链驱动,小组合作
[H]数学建模思想贯穿全课,紧密联系生活实际
[R]适合中等及以上水平学生,需预习方程概念`

// 最简教案索引
const minimalLessonIndex = `SJ:CN|GR:3|CG:2|PQ:1|ST:1|QL:3
[O]理解课文主旨`

// ==================== ParseLessonIndexField 测试 ====================

func TestParseLessonIndexField_Standard(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"SJ", "MA"},
		{"GR", "7"},
		{"CG", "4"},
		{"PQ", "2"},
		{"ST", "2"},
		{"QL", "4"},
	}
	for _, tt := range tests {
		got := ParseLessonIndexField(sampleLessonIndex, tt.field)
		if got != tt.expected {
			t.Errorf("ParseLessonIndexField(%q) = %q, want %q", tt.field, got, tt.expected)
		}
	}
}

func TestParseLessonIndexField_Edge(t *testing.T) {
	// 空输入
	if got := ParseLessonIndexField("", "SJ"); got != "" {
		t.Errorf("空输入应返回空")
	}
	// 不存在的字段
	if got := ParseLessonIndexField(sampleLessonIndex, "XX"); got != "" {
		t.Errorf("不存在字段应返回空")
	}
}

// ==================== 冗余列解析测试 ====================

func TestParseLessonCognitiveLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"标准", sampleLessonIndex, 4},
		{"最小", minimalLessonIndex, 2},
		{"边界最大", "SJ:MA|CG:6|PQ:1|ST:1|QL:1", 6},
		{"边界最小", "SJ:MA|CG:1|PQ:1|ST:1|QL:1", 1},
		{"空", "", 0},
		{"超范围", "SJ:MA|CG:7|PQ:1", 0},
		{"无CG", "SJ:MA|PQ:1|ST:1", 0},
	}
	for _, tt := range tests {
		got := ParseLessonCognitiveLevel(tt.input)
		if got != tt.expected {
			t.Errorf("ParseLessonCognitiveLevel(%s) = %d, want %d", tt.name, got, tt.expected)
		}
	}
}

func TestParseLessonPedagogyIntensity(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{sampleLessonIndex, 2},
		{minimalLessonIndex, 1},
		{"SJ:MA|PQ:3", 3},
		{"SJ:MA|PQ:4", 0}, // 超范围
		{"", 0},
	}
	for _, tt := range tests {
		got := ParseLessonPedagogyIntensity(tt.input)
		if got != tt.expected {
			t.Errorf("ParseLessonPedagogyIntensity = %d, want %d", got, tt.expected)
		}
	}
}

func TestParseStructureType(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{sampleLessonIndex, 2},          // ST:2 探究型
		{minimalLessonIndex, 1},         // ST:1 讲授型
		{"SJ:MA|ST:5|QL:1", 5},         // 混合型
		{"SJ:MA|ST:6|QL:1", 0},         // 超范围
		{"SJ:MA|ST:0|QL:1", 0},         // 0无效
		{"", 0},
	}
	for _, tt := range tests {
		got := ParseStructureType(tt.input)
		if got != tt.expected {
			t.Errorf("ParseStructureType = %d, want %d", got, tt.expected)
		}
	}
}

func TestParseQualityLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{sampleLessonIndex, 4},          // QL:4
		{minimalLessonIndex, 3},         // QL:3
		{"SJ:MA|QL:5", 5},              // 精品
		{"SJ:MA|QL:1", 1},              // 草稿
		{"SJ:MA|QL:6", 0},              // 超范围
		{"", 0},
	}
	for _, tt := range tests {
		got := ParseQualityLevel(tt.input)
		if got != tt.expected {
			t.Errorf("ParseQualityLevel = %d, want %d", got, tt.expected)
		}
	}
}

// ==================== CalculateQualityLevel 测试（核心业务逻辑）====================

func TestCalculateQualityLevel_Precision(t *testing.T) {
	// 精品级: AI≥9.0 + approved
	score90 := 9.0
	if got := CalculateQualityLevel(&score90, "approved"); got != 5 {
		t.Errorf("9.0+approved应为5(精品), got %d", got)
	}
	if got := CalculateQualityLevel(&score90, "published_shared"); got != 5 {
		t.Errorf("9.0+published_shared应为5(精品), got %d", got)
	}
	// 9.0但不是approved → 不是精品（降为优秀）
	if got := CalculateQualityLevel(&score90, "draft"); got != 4 {
		t.Errorf("9.0+draft应为4(优秀), got %d", got)
	}
}

func TestCalculateQualityLevel_Excellent(t *testing.T) {
	// 优秀级: AI≥8.0 或 (AI≥7.5+approved)
	score80 := 8.0
	if got := CalculateQualityLevel(&score80, "draft"); got != 4 {
		t.Errorf("8.0+draft应为4(优秀), got %d", got)
	}
	score75 := 7.5
	if got := CalculateQualityLevel(&score75, "approved"); got != 4 {
		t.Errorf("7.5+approved应为4(优秀), got %d", got)
	}
	// 7.5但不approved → 良好
	if got := CalculateQualityLevel(&score75, "draft"); got != 3 {
		t.Errorf("7.5+draft应为3(良好), got %d", got)
	}
}

func TestCalculateQualityLevel_Good(t *testing.T) {
	// 良好级: AI≥7.0 或 approved（无评分）
	score70 := 7.0
	if got := CalculateQualityLevel(&score70, "draft"); got != 3 {
		t.Errorf("7.0+draft应为3(良好), got %d", got)
	}
	// approved无评分也是良好
	score0 := 0.0
	if got := CalculateQualityLevel(&score0, "approved"); got != 3 {
		t.Errorf("0+approved应为3(良好), got %d", got)
	}
}

func TestCalculateQualityLevel_Usable(t *testing.T) {
	// 可用级: AI≥5.0 或 published_personal
	score50 := 5.0
	if got := CalculateQualityLevel(&score50, "draft"); got != 2 {
		t.Errorf("5.0+draft应为2(可用), got %d", got)
	}
	score0 := 0.0
	if got := CalculateQualityLevel(&score0, "published_personal"); got != 2 {
		t.Errorf("0+published_personal应为2(可用), got %d", got)
	}
}

func TestCalculateQualityLevel_Draft(t *testing.T) {
	// 草稿级: 无评分+普通状态
	score0 := 0.0
	if got := CalculateQualityLevel(&score0, "draft"); got != 1 {
		t.Errorf("0+draft应为1(草稿), got %d", got)
	}
	// nil评分
	if got := CalculateQualityLevel(nil, "draft"); got != 1 {
		t.Errorf("nil+draft应为1(草稿), got %d", got)
	}
	score49 := 4.9
	if got := CalculateQualityLevel(&score49, "draft"); got != 1 {
		t.Errorf("4.9+draft应为1(草稿), got %d", got)
	}
}

// ==================== FormatLessonIndexForPrompt 测试 ====================

func TestFormatLessonIndexForPrompt_Standard(t *testing.T) {
	result := FormatLessonIndexForPrompt(sampleLessonIndex, "一元一次方程教学设计")
	// 检查编码短标识
	if !strings.Contains(result, "MA-7-ST2-QL4") {
		t.Errorf("缺少编码短标识，实际: %s", result)
	}
	// 检查标题
	if !strings.Contains(result, "一元一次方程教学设计") {
		t.Errorf("缺少标题")
	}
	// 检查语义标签
	for _, tag := range []string{"[O]", "[S]", "[M]", "[H]", "[R]"} {
		if !strings.Contains(result, tag) {
			t.Errorf("缺少语义标签 %s", tag)
		}
	}
}

func TestFormatLessonIndexForPrompt_Empty(t *testing.T) {
	result := FormatLessonIndexForPrompt("", "测试教案")
	if !strings.Contains(result, "未索引") {
		t.Errorf("空索引应返回未索引提示")
	}
}

// ==================== BuildLessonFullText 测试 ====================

func TestBuildLessonFullText_Complete(t *testing.T) {
	result := BuildLessonFullText(
		"数学", "七年级", "一元一次方程", "第3课时",
		45, "# 教案正文\n内容...", `{"objectives":["目标1"]}`, `{"score":8.5}`, `["comp1"]`,
	)
	checks := []string{"学科=数学", "年级=七年级", "课题=一元一次方程", "时长=45分钟", "教案正文="}
	for _, c := range checks {
		if !strings.Contains(result, c) {
			t.Errorf("缺少: %s", c)
		}
	}
}

func TestBuildLessonFullText_EmptyOptional(t *testing.T) {
	result := BuildLessonFullText("数学", "7", "测试", "标题", 45, "", "", "", "")
	if strings.Contains(result, "教案正文=") {
		t.Errorf("空正文不应输出教案正文行")
	}
	if strings.Contains(result, "结构化内容=") {
		t.Errorf("空结构化不应输出")
	}
	// 必填信息仍应存在
	if !strings.Contains(result, "学科=数学") {
		t.Errorf("缺少必填字段")
	}
}

func TestBuildLessonFullText_LongContent(t *testing.T) {
	longMD := strings.Repeat("教案正文内容", 1000) // ~6000字
	result := BuildLessonFullText("数学", "7", "测试", "标题", 45, longMD, "", "", "")
	if !strings.Contains(result, "已截断") {
		t.Errorf("超长正文应被截断")
	}
}

func TestBuildLessonFullText_EmptyJsonSkip(t *testing.T) {
	// "{}" 和 "[]" 应被跳过
	result := BuildLessonFullText("数学", "7", "测试", "标题", 45, "", "{}", "{}", "[]")
	if strings.Contains(result, "结构化内容=") {
		t.Errorf("{} 不应输出结构化内容")
	}
	if strings.Contains(result, "AI评审结果=") {
		t.Errorf("{} 不应输出评审结果")
	}
	if strings.Contains(result, "使用的教学组件=") {
		t.Errorf("[] 不应输出组件")
	}
}

// ==================== ValidateLessonIndex 测试 ====================

func TestValidateLessonIndex_Valid(t *testing.T) {
	if !ValidateLessonIndex(sampleLessonIndex) {
		t.Error("标准索引应通过验证")
	}
	if !ValidateLessonIndex(minimalLessonIndex) {
		t.Error("最简索引应通过验证")
	}
}

func TestValidateLessonIndex_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空字符串", ""},
		{"无SJ", "GR:7|CG:4\n[O]目标"},
		{"无[O]", "SJ:MA|GR:7|CG:4\n[S]结构"},
		{"两个都没有", "随便写的内容"},
	}
	for _, tt := range tests {
		if ValidateLessonIndex(tt.input) {
			t.Errorf("ValidateLessonIndex(%s)应返回false", tt.name)
		}
	}
}

// ==================== 编码映射表完整性测试 ====================

func TestStructureTypeNameMap_Completeness(t *testing.T) {
	if len(StructureTypeNameMap) != 5 {
		t.Errorf("StructureTypeNameMap应有5项，实际%d项", len(StructureTypeNameMap))
	}
	// 验证1-5全部有值
	for i := 1; i <= 5; i++ {
		if _, ok := StructureTypeNameMap[i]; !ok {
			t.Errorf("StructureTypeNameMap缺少key=%d", i)
		}
	}
}

func TestQualityLevelNameMap_Completeness(t *testing.T) {
	if len(QualityLevelNameMap) != 5 {
		t.Errorf("QualityLevelNameMap应有5项，实际%d项", len(QualityLevelNameMap))
	}
}

func TestStructureTypeCodeMap_Completeness(t *testing.T) {
	if len(StructureTypeCodeMap) != 5 {
		t.Errorf("StructureTypeCodeMap应有5项，实际%d项", len(StructureTypeCodeMap))
	}
}
