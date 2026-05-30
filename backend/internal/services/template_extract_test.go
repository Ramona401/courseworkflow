package services

// template_extract_test.go — v139/v139.1 模板提取相关纯函数单元测试
//
// 测试范围：
//   - sanitizeNonASCIIForJSON: 非 ASCII 拉丁字符转义
//   - isCJKRune: CJK 字符判定
//   - cwStripCodeFences: 代码围栏剥离
//   - cwCleanChinesePunctuation: 中文标点清理
//   - cwFixJSONQuotes: 内嵌引号修复
//   - extractStringField / extractObjectField / extractArrayField: 字段级宽容提取
//   - truncateForLog: 日志截断

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==================== sanitizeNonASCIIForJSON 测试 ====================

// TestSanitizeNonASCII_PureASCII 纯 ASCII 输入不变
func TestSanitizeNonASCII_PureASCII(t *testing.T) {
	input := `{"name":"hello","value":"world 123"}`
	result := sanitizeNonASCIIForJSON(input)
	if result != input {
		t.Errorf("纯ASCII不应改变\n  输入: %s\n  输出: %s", input, result)
	}
}

// TestSanitizeNonASCII_ChinesePreserved CJK 中文字符保留原样
func TestSanitizeNonASCII_ChinesePreserved(t *testing.T) {
	input := `{"name":"简约清新模板","desc":"适合小学"}`
	result := sanitizeNonASCIIForJSON(input)
	if result != input {
		t.Errorf("中文字符不应被转义\n  输入: %s\n  输出: %s", input, result)
	}
}

// TestSanitizeNonASCII_LatinSpecialEscaped 拉丁特殊字符被转义
func TestSanitizeNonASCII_LatinSpecialEscaped(t *testing.T) {
	// é = U+00E9, ′ = U+2032, ™ = U+2122
	input := `{"text":"café","symbol":"5′","tm":"AI™"}`
	result := sanitizeNonASCIIForJSON(input)

	// 转义后应能被 json.Unmarshal 正确解析
	var m map[string]string
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("转义后 JSON 仍无法解析: %v\n  结果: %s", err, result)
	}

	// 解析后值应还原为原始 Unicode 字符
	if m["text"] != "café" {
		t.Errorf("text 期望 'café', 实际 '%s'", m["text"])
	}
	if m["symbol"] != "5′" {
		t.Errorf("symbol 期望 '5′', 实际 '%s'", m["symbol"])
	}
	if m["tm"] != "AI™" {
		t.Errorf("tm 期望 'AI™', 实际 '%s'", m["tm"])
	}
}

// TestSanitizeNonASCII_MixedChineseAndLatin 中文+拉丁混合:中文保留,拉丁转义
func TestSanitizeNonASCII_MixedChineseAndLatin(t *testing.T) {
	input := `{"name":"简约café","desc":"résumé模板"}`
	result := sanitizeNonASCIIForJSON(input)

	var m map[string]string
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("混合内容转义后无法解析: %v\n  结果: %s", err, result)
	}
	if m["name"] != "简约café" {
		t.Errorf("name 期望 '简约café', 实际 '%s'", m["name"])
	}
	if m["desc"] != "résumé模板" {
		t.Errorf("desc 期望 'résumé模板', 实际 '%s'", m["desc"])
	}
}

// TestSanitizeNonASCII_ExistingEscape 已有的 \uXXXX 转义不被重复处理
func TestSanitizeNonASCII_ExistingEscape(t *testing.T) {
	input := `{"name":"\\u0048ello"}`
	result := sanitizeNonASCIIForJSON(input)
	if result != input {
		t.Errorf("已有转义序列不应被改变\n  输入: %s\n  输出: %s", input, result)
	}
}

// TestSanitizeNonASCII_EmptyString 空字符串不崩溃
func TestSanitizeNonASCII_EmptyString(t *testing.T) {
	result := sanitizeNonASCIIForJSON("")
	if result != "" {
		t.Errorf("空字符串应返回空, 实际 '%s'", result)
	}
}

// TestSanitizeNonASCII_OutsideStringNotChanged JSON 键名中的非 ASCII 也被转义(在引号内)
func TestSanitizeNonASCII_KeyAlsoSanitized(t *testing.T) {
	input := `{"café":"value"}`
	result := sanitizeNonASCIIForJSON(input)

	var m map[string]string
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("键名转义后无法解析: %v\n  结果: %s", err, result)
	}
	if _, ok := m["café"]; !ok {
		t.Errorf("键名 'café' 应存在于解析结果中, 实际 keys=%v", m)
	}
}

// TestSanitizeNonASCII_HTMLInJSON 含 HTML 内容的 JSON 字符串值
func TestSanitizeNonASCII_HTMLInJSON(t *testing.T) {
	input := `{"html":"<div style=\"color:red\">héllo</div>"}`
	result := sanitizeNonASCIIForJSON(input)

	// 应能被 json.Unmarshal 解析
	var m map[string]string
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("HTML 内容转义后无法解析: %v\n  结果: %s", err, result)
	}
	if !strings.Contains(m["html"], "div") {
		t.Errorf("HTML 标签应保留, 实际 '%s'", m["html"])
	}
}

// ==================== isCJKRune 测试 ====================

func TestIsCJKRune(t *testing.T) {
	tests := []struct {
		name   string
		r      rune
		expect bool
	}{
		{"中文汉字-中", '中', true},
		{"中文汉字-国", '国', true},
		{"日文片假名-ア", 'ア', true},       // U+30A2, 在 0x3000-0x9FFF 范围
		{"全角逗号", '，', true},             // U+FF0C, 在 0xFF00-0xFFEF 范围
		{"中文句号", '。', true},             // U+3002, 在 0x3000-0x9FFF 范围
		{"ASCII字母", 'A', false},
		{"ASCII数字", '0', false},
		{"拉丁é", 'é', false},              // U+00E9
		{"拉丁ñ", 'ñ', false},              // U+00F1
		{"特殊符号™", '™', false},           // U+2122
		{"特殊符号′", '′', false},           // U+2032
		{"特殊符号¨", '¨', false},           // U+00A8
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCJKRune(tt.r)
			if got != tt.expect {
				t.Errorf("isCJKRune(%c/U+%04X) = %v, 期望 %v", tt.r, tt.r, got, tt.expect)
			}
		})
	}
}

// ==================== cwStripCodeFences 测试 ====================

func TestCWStripCodeFences_WithFences(t *testing.T) {
	input := "```json\n{\"name\":\"test\"}\n```"
	result := cwStripCodeFences(input)
	if result != `{"name":"test"}` {
		t.Errorf("期望去除围栏, 实际 '%s'", result)
	}
}

func TestCWStripCodeFences_WithoutFences(t *testing.T) {
	input := `{"name":"test"}`
	result := cwStripCodeFences(input)
	if result != input {
		t.Errorf("无围栏不应改变, 实际 '%s'", result)
	}
}

func TestCWStripCodeFences_OnlyBackticks(t *testing.T) {
	input := "```\nhello\n```"
	result := cwStripCodeFences(input)
	if result != "hello" {
		t.Errorf("期望 'hello', 实际 '%s'", result)
	}
}

func TestCWStripCodeFences_EmptyString(t *testing.T) {
	result := cwStripCodeFences("")
	if result != "" {
		t.Errorf("空字符串应返回空, 实际 '%s'", result)
	}
}

// ==================== cwCleanChinesePunctuation 测试 ====================

func TestCWCleanChinesePunctuation_ChineseQuotes(t *testing.T) {
	// 中文双引号 " " 应被删除而非替换为英文引号
	input := `{"name": \u201c测试\u201d}`
	result := cwCleanChinesePunctuation(input)
	if strings.Contains(result, "\u201c") || strings.Contains(result, "\u201d") {
		t.Errorf("中文引号应被删除, 实际 '%s'", result)
	}
}

func TestCWCleanChinesePunctuation_FullWidthComma(t *testing.T) {
	input := "项目一\uff0c项目二"
	result := cwCleanChinesePunctuation(input)
	if result != "项目一,项目二" {
		t.Errorf("全角逗号应替换为半角, 实际 '%s'", result)
	}
}

func TestCWCleanChinesePunctuation_FullWidthColon(t *testing.T) {
	input := "标题\uff1a内容"
	result := cwCleanChinesePunctuation(input)
	if result != "标题:内容" {
		t.Errorf("全角冒号应替换为半角, 实际 '%s'", result)
	}
}

func TestCWCleanChinesePunctuation_NoChange(t *testing.T) {
	input := `{"key":"value","num":123}`
	result := cwCleanChinesePunctuation(input)
	if result != input {
		t.Errorf("纯 ASCII JSON 不应改变\n  输入: %s\n  输出: %s", input, result)
	}
}

// ==================== extractStringField 测试 ====================

func TestExtractStringField_Normal(t *testing.T) {
	input := `{"name":"hello","age":20}`
	result := extractStringField(input, "name")
	if result != "hello" {
		t.Errorf("期望 'hello', 实际 '%s'", result)
	}
}

func TestExtractStringField_WithEscape(t *testing.T) {
	input := `{"path":"C:\\Users\\test"}`
	result := extractStringField(input, "path")
	if result != `C:\\Users\\test` {
		t.Errorf("期望保留转义, 实际 '%s'", result)
	}
}

func TestExtractStringField_Missing(t *testing.T) {
	input := `{"name":"hello"}`
	result := extractStringField(input, "missing")
	if result != "" {
		t.Errorf("不存在的字段应返回空, 实际 '%s'", result)
	}
}

func TestExtractStringField_EmptyValue(t *testing.T) {
	input := `{"name":""}`
	result := extractStringField(input, "name")
	if result != "" {
		t.Errorf("空值应返回空字符串, 实际 '%s'", result)
	}
}

// ==================== extractObjectField 测试 ====================

func TestExtractObjectField_Normal(t *testing.T) {
	input := `{"outer":"val","inner":{"a":"1","b":"2"},"after":"x"}`
	result := extractObjectField(input, "inner")
	if result != `{"a":"1","b":"2"}` {
		t.Errorf("期望嵌套对象, 实际 '%s'", result)
	}
}

func TestExtractObjectField_Nested(t *testing.T) {
	input := `{"data":{"level1":{"level2":"deep"}}}`
	result := extractObjectField(input, "data")
	expected := `{"level1":{"level2":"deep"}}`
	if result != expected {
		t.Errorf("期望 '%s', 实际 '%s'", expected, result)
	}
}

func TestExtractObjectField_Missing(t *testing.T) {
	input := `{"name":"test"}`
	result := extractObjectField(input, "missing")
	if result != "" {
		t.Errorf("不存在的字段应返回空, 实际 '%s'", result)
	}
}

// ==================== extractArrayField 测试 ====================

func TestExtractArrayField_Normal(t *testing.T) {
	input := `{"items":["a","b","c"],"count":3}`
	result := extractArrayField(input, "items")
	if result != `["a","b","c"]` {
		t.Errorf("期望数组, 实际 '%s'", result)
	}
}

func TestExtractArrayField_NestedQuotes(t *testing.T) {
	// 数组中的字符串含转义引号
	input := `{"pages":["<div class=\"test\">"]}`
	result := extractArrayField(input, "pages")
	if result != `["<div class=\"test\">"]` {
		t.Errorf("期望含转义引号的数组, 实际 '%s'", result)
	}
}

func TestExtractArrayField_Empty(t *testing.T) {
	input := `{"items":[]}`
	result := extractArrayField(input, "items")
	if result != "[]" {
		t.Errorf("空数组应返回 '[]', 实际 '%s'", result)
	}
}

func TestExtractArrayField_Missing(t *testing.T) {
	input := `{"name":"test"}`
	result := extractArrayField(input, "missing")
	if result != "" {
		t.Errorf("不存在的字段应返回空, 实际 '%s'", result)
	}
}

// ==================== truncateForLog 测试 ====================

func TestTruncateForLog_Short(t *testing.T) {
	result := truncateForLog("hello", 10)
	if result != "hello" {
		t.Errorf("短字符串不应截断, 实际 '%s'", result)
	}
}

func TestTruncateForLog_Exact(t *testing.T) {
	result := truncateForLog("12345", 5)
	if result != "12345" {
		t.Errorf("等长字符串不应截断, 实际 '%s'", result)
	}
}

func TestTruncateForLog_Long(t *testing.T) {
	result := truncateForLog("1234567890", 5)
	if result != "12345...(truncated)" {
		t.Errorf("期望截断, 实际 '%s'", result)
	}
}

func TestTruncateForLog_Empty(t *testing.T) {
	result := truncateForLog("", 10)
	if result != "" {
		t.Errorf("空字符串应返回空, 实际 '%s'", result)
	}
}
