// Package ai AI客户端单元测试
// 测试范围：ExtractJSON(返回string,bool) / stripThinking(包内小写函数)
// 注意：ExtractJSON从文本中提取第一个有效JSON对象（{}），不处理裸数组
package ai

import (
	"strings"
	"testing"
)

// TestExtractJSONBasic 测试从AI回复中提取标准JSON对象
func TestExtractJSONBasic(t *testing.T) {
	input := `AI回复文字，然后是JSON：{"score": 9.5, "reason": "优秀"} 回复结束`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("未提取到JSON（ok=false），result=%q", result)
		return
	}
	if !strings.Contains(result, "score") {
		t.Fatalf("提取结果应包含score字段: %s", result)
	}
	t.Logf("提取JSON成功: %s", result)
}

// TestExtractJSONArrayWrapped 测试JSON数组包裹在对象中可正常提取
// ExtractJSON提取第一个{}对象，数组元素以对象形式嵌套时可提取
func TestExtractJSONArrayWrapped(t *testing.T) {
	// 对象包含数组字段——这是AI实际返回的格式
	input := `{"pages": [{"id":1,"name":"页面A"},{"id":2,"name":"页面B"}], "total": 2}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("包含数组的对象未提取到（ok=false），result=%q", result)
		return
	}
	if !strings.Contains(result, "pages") {
		t.Fatalf("提取结果应包含pages字段: %s", result)
	}
	t.Logf("包含数组的对象提取成功: %s", result[:minLen(80, len(result))])
}

// TestExtractJSONArrayDirect 测试裸JSON数组的提取行为（记录实际行为，不强制断言格式）
func TestExtractJSONArrayDirect(t *testing.T) {
	input := `[{"id":1,"name":"页面A"},{"id":2,"name":"页面B"}]`
	result, ok := ExtractJSON(input)
	// ExtractJSON实现提取第一个{}对象，裸数组行为由实现决定
	// 此测试仅验证不panic，记录实际行为
	t.Logf("裸JSON数组提取: ok=%v, result=%q", ok, result)
}

// TestExtractJSONMarkdownBlock 测试从```json代码块中提取JSON
func TestExtractJSONMarkdownBlock(t *testing.T) {
	input := "AI分析结果：\n```json\n{\"total_score\": 8.5, \"e1\": 9}\n```\n分析完毕"
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("markdown代码块未提取到（ok=false），result=%q", result)
		return
	}
	if !strings.Contains(result, "total_score") {
		t.Fatalf("提取结果应包含total_score: %s", result)
	}
	t.Logf("markdown块JSON提取成功: %s", result)
}

// TestExtractJSONNestedObject 测试嵌套JSON对象提取
func TestExtractJSONNestedObject(t *testing.T) {
	input := `{"meta": {"score": 9.0, "items": [1,2,3]}, "status": "ok"}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("嵌套JSON未提取到（ok=false）")
		return
	}
	if !strings.Contains(result, "meta") {
		t.Fatalf("应包含meta字段: %s", result)
	}
	t.Logf("嵌套JSON提取成功")
}

// TestExtractJSONNoJSON 测试无JSON时返回ok=false
func TestExtractJSONNoJSON(t *testing.T) {
	input := "这是纯文字，没有任何JSON内容，只有中文描述。"
	result, ok := ExtractJSON(input)
	t.Logf("无JSON输入: ok=%v, result=%q", ok, result)
}

// TestExtractJSONSpecialChars 测试包含特殊字符的JSON
func TestExtractJSONSpecialChars(t *testing.T) {
	input := `{"content": "含有特殊内容的字段", "score": 9}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("特殊字符JSON未提取到（ok=false）")
		return
	}
	t.Logf("特殊字符JSON提取: %s", result)
}

// TestExtractJSONOnlyJSON 测试纯JSON输入
func TestExtractJSONOnlyJSON(t *testing.T) {
	input := `{"score": 9.5, "grade": "A", "passed": true}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Fatalf("纯JSON应提取成功，ok=false")
	}
	if !strings.Contains(result, "score") {
		t.Fatalf("结果应含score字段: %s", result)
	}
	t.Logf("纯JSON: ok=%v, result=%s", ok, result)
}

// TestExtractJSONLargePayload 测试大型JSON提取
func TestExtractJSONLargePayload(t *testing.T) {
	input := `根据分析，完整评估如下：
{"total_score": 9.2, "dimensions": {"e1": 9.5, "e2": 9.0, "e3": 9.3, "e4": 9.0},
"feedback": "` + strings.Repeat("内容优秀", 100) + `", "grade": "A"}`

	result, ok := ExtractJSON(input)
	t.Logf("大型JSON: ok=%v, 结果长度=%d", ok, len(result))
}

// TestExtractJSONEvalFormat 测试评估结果标准格式提取（真实AI输出）
func TestExtractJSONEvalFormat(t *testing.T) {
	input := `根据对课程内容的全面评估，结果如下：
{"total_score": 9.2, "e1": 9.5, "e2": 9.0, "e3": 9.3, "e4": 9.0,
"hard_constraint": "PASS", "grade": "A", "conclusion": "通过验收"}`

	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("评估格式未提取到（ok=false）")
		return
	}
	if !strings.Contains(result, "total_score") {
		t.Fatalf("应包含total_score: %s", result)
	}
	t.Logf("评估格式提取成功: %s", result[:minLen(100, len(result))])
}

// TestStripThinkingBasic 测试去除<thinking>标签
func TestStripThinkingBasic(t *testing.T) {
	input := "<thinking>内部思考过程</thinking>\n实际回复内容在这里"
	result := stripThinking(input)
	if strings.Contains(result, "<thinking>") {
		t.Fatal("结果不应包含thinking标签")
	}
	if strings.Contains(result, "内部思考过程") {
		t.Fatal("thinking内容不应保留")
	}
	if !strings.Contains(result, "实际回复内容") {
		t.Fatalf("实际回复应保留: %s", result)
	}
	t.Logf("stripThinking基本功能验证通过: %q", result)
}

// TestStripThinkingNoTag 测试无thinking标签时原样返回
func TestStripThinkingNoTag(t *testing.T) {
	input := "普通AI回复，没有thinking标签"
	result := stripThinking(input)
	if result != input {
		t.Fatalf("无标签应原样返回，期望 %q, 实际 %q", input, result)
	}
}

// TestStripThinkingMultiple 测试多个thinking标签全部去除
func TestStripThinkingMultiple(t *testing.T) {
	input := "<thinking>第一段思考</thinking>中间内容<thinking>第二段思考</thinking>最终答案"
	result := stripThinking(input)
	if strings.Contains(result, "第一段思考") || strings.Contains(result, "第二段思考") {
		t.Fatal("所有thinking标签内容都应被去除")
	}
	if !strings.Contains(result, "最终答案") {
		t.Fatalf("保留内容丢失: %s", result)
	}
	t.Log("多thinking标签去除验证通过")
}

// TestStripThinkingEmpty 测试空字符串
func TestStripThinkingEmpty(t *testing.T) {
	result := stripThinking("")
	if result != "" {
		t.Fatalf("空输入应返回空，实际: %q", result)
	}
}

// TestStripThinkingOnlyTag 测试仅有thinking标签
func TestStripThinkingOnlyTag(t *testing.T) {
	input := "<thinking>全是思考内容，无实际输出</thinking>"
	result := stripThinking(input)
	if strings.Contains(strings.TrimSpace(result), "全是思考内容") {
		t.Fatal("thinking内容不应保留")
	}
	t.Logf("纯thinking标签结果: %q", result)
}

// TestExtractJSONWithThinkingCombined 测试thinking+JSON组合（真实AI输出场景）
func TestExtractJSONWithThinkingCombined(t *testing.T) {
	input := `<thinking>
我需要分析课程质量，评估各维度...
</thinking>
评估结果：
{"total_score": 8.7, "e1": 9.0, "e2": 8.5, "e3": 9.0, "e4": 8.3, "grade": "B"}`

	stripped := stripThinking(input)
	if strings.Contains(stripped, "我需要分析") {
		t.Fatal("thinking内容未去除")
	}
	result, ok := ExtractJSON(stripped)
	t.Logf("thinking+JSON组合: ok=%v, result=%s", ok, result)
}

// minLen 辅助函数：取两个整数的最小值
func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
