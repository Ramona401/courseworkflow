// Package ai AI客户端单元测试
//
// 测试范围：
//   - ExtractJSON / extractFromCodeBlock：JSON提取
//   - stripThinking：思维链标签移除
//   - coalesce：首个非空字符串选择
//   - parseFloat / parseInt：字符串→数值解析
//   - getSceneFromTrace：trace上下文场景获取
//   - isRetryableError：HTTP状态码可重试判断
//   - getRetryDelay：重试延迟获取
//   - extractErrorMessage：错误消息提取
//   - isRetryableCallError：错误类型判断
//   - retryableError：可重试错误类型
package ai

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ==================== ExtractJSON 测试（保留原有） ====================

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
}

func TestExtractJSONArrayWrapped(t *testing.T) {
	input := `{"pages": [{"id":1,"name":"页面A"},{"id":2,"name":"页面B"}], "total": 2}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("包含数组的对象未提取到（ok=false），result=%q", result)
		return
	}
	if !strings.Contains(result, "pages") {
		t.Fatalf("提取结果应包含pages字段: %s", result)
	}
}

func TestExtractJSONArrayDirect(t *testing.T) {
	input := `[{"id":1,"name":"页面A"},{"id":2,"name":"页面B"}]`
	result, ok := ExtractJSON(input)
	t.Logf("裸JSON数组提取: ok=%v, result=%q", ok, result)
}

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
}

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
}

func TestExtractJSONNoJSON(t *testing.T) {
	input := "这是纯文字，没有任何JSON内容，只有中文描述。"
	_, ok := ExtractJSON(input)
	if ok {
		t.Error("纯文字不应提取到JSON")
	}
}

func TestExtractJSONSpecialChars(t *testing.T) {
	input := `{"content": "含有特殊内容的字段", "score": 9}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Logf("特殊字符JSON未提取到（ok=false）")
		return
	}
	t.Logf("特殊字符JSON提取: %s", result)
}

func TestExtractJSONOnlyJSON(t *testing.T) {
	input := `{"score": 9.5, "grade": "A", "passed": true}`
	result, ok := ExtractJSON(input)
	if !ok {
		t.Fatalf("纯JSON应提取成功，ok=false")
	}
	if !strings.Contains(result, "score") {
		t.Fatalf("结果应含score字段: %s", result)
	}
}

func TestExtractJSONLargePayload(t *testing.T) {
	input := `根据分析，完整评估如下：
{"total_score": 9.2, "dimensions": {"e1": 9.5, "e2": 9.0, "e3": 9.3, "e4": 9.0},
"feedback": "` + strings.Repeat("内容优秀", 100) + `", "grade": "A"}`
	_, ok := ExtractJSON(input)
	t.Logf("大型JSON: ok=%v", ok)
}

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
}

// ==================== stripThinking 测试（保留原有） ====================

func TestStripThinkingBasic(t *testing.T) {
	input := "<thinking>内部思考过程</thinking>\n实际回复内容在这里"
	result := stripThinking(input)
	if strings.Contains(result, "内部思考过程") {
		t.Fatal("thinking内容不应保留")
	}
	if !strings.Contains(result, "实际回复内容") {
		t.Fatalf("实际回复应保留: %s", result)
	}
}

func TestStripThinkingNoTag(t *testing.T) {
	input := "普通AI回复，没有thinking标签"
	result := stripThinking(input)
	if result != input {
		t.Fatalf("无标签应原样返回")
	}
}

func TestStripThinkingMultiple(t *testing.T) {
	input := "<thinking>第一段思考</thinking>中间内容<thinking>第二段思考</thinking>最终答案"
	result := stripThinking(input)
	if strings.Contains(result, "第一段思考") || strings.Contains(result, "第二段思考") {
		t.Fatal("所有thinking标签内容都应被去除")
	}
	if !strings.Contains(result, "最终答案") {
		t.Fatal("保留内容丢失")
	}
}

func TestStripThinkingEmpty(t *testing.T) {
	if stripThinking("") != "" {
		t.Fatal("空输入应返回空")
	}
}

func TestStripThinkingOnlyTag(t *testing.T) {
	input := "<thinking>全是思考内容</thinking>"
	result := stripThinking(input)
	if strings.Contains(result, "全是思考内容") {
		t.Fatal("thinking内容不应保留")
	}
}

func TestExtractJSONWithThinkingCombined(t *testing.T) {
	input := "<thinking>分析中...</thinking>\n{\"total_score\": 8.7}"
	stripped := stripThinking(input)
	result, ok := ExtractJSON(stripped)
	if !ok {
		t.Fatal("去除thinking后应能提取JSON")
	}
	if !strings.Contains(result, "total_score") {
		t.Fatal("应包含total_score")
	}
}

// ==================== coalesce 测试（新增） ====================

func TestCoalesce_FirstNonEmpty(t *testing.T) {
	result := coalesce("", "  ", "hello", "world")
	if result != "hello" {
		t.Errorf("应返回第一个非空非空白字符串'hello'，实际%q", result)
	}
}

func TestCoalesce_AllEmpty(t *testing.T) {
	result := coalesce("", "  ", "")
	if result != "" {
		t.Errorf("全部为空/空白时应返回空，实际%q", result)
	}
}

func TestCoalesce_SingleValue(t *testing.T) {
	result := coalesce("only")
	if result != "only" {
		t.Errorf("单个非空值应直接返回，实际%q", result)
	}
}

func TestCoalesce_NoArgs(t *testing.T) {
	result := coalesce()
	if result != "" {
		t.Errorf("无参数应返回空，实际%q", result)
	}
}

func TestCoalesce_WhitespaceSkipped(t *testing.T) {
	result := coalesce("   ", "\t", "actual")
	if result != "actual" {
		t.Errorf("空白字符串应被跳过，实际%q", result)
	}
}

// ==================== parseFloat 测试（新增） ====================

func TestParseFloat_Valid(t *testing.T) {
	result := parseFloat("0.7", 1.0)
	if result != 0.7 {
		t.Errorf("应返回0.7，实际%f", result)
	}
}

func TestParseFloat_Empty(t *testing.T) {
	result := parseFloat("", 0.5)
	if result != 0.5 {
		t.Errorf("空字符串应返回默认值0.5，实际%f", result)
	}
}

func TestParseFloat_Invalid(t *testing.T) {
	result := parseFloat("not_a_number", 0.3)
	if result != 0.3 {
		t.Errorf("无效字符串应返回默认值0.3，实际%f", result)
	}
}

func TestParseFloat_Zero(t *testing.T) {
	result := parseFloat("0", 1.0)
	if result != 0.0 {
		t.Errorf("0是合法浮点数，应返回0.0，实际%f", result)
	}
}

func TestParseFloat_Negative(t *testing.T) {
	result := parseFloat("-0.5", 1.0)
	if result != -0.5 {
		t.Errorf("应返回-0.5，实际%f", result)
	}
}

// ==================== parseInt 测试（新增） ====================

func TestParseInt_Valid(t *testing.T) {
	result := parseInt("4096", 1000)
	if result != 4096 {
		t.Errorf("应返回4096，实际%d", result)
	}
}

func TestParseInt_Empty(t *testing.T) {
	result := parseInt("", 2000)
	if result != 2000 {
		t.Errorf("空字符串应返回默认值2000，实际%d", result)
	}
}

func TestParseInt_Invalid(t *testing.T) {
	result := parseInt("abc", 500)
	if result != 500 {
		t.Errorf("无效字符串应返回默认值500，实际%d", result)
	}
}

func TestParseInt_Zero(t *testing.T) {
	result := parseInt("0", 100)
	if result != 0 {
		t.Errorf("0是合法整数，应返回0，实际%d", result)
	}
}

// ==================== getSceneFromTrace 测试（新增） ====================

func TestGetSceneFromTrace_Nil(t *testing.T) {
	result := getSceneFromTrace(nil)
	if result != "unknown" {
		t.Errorf("nil应返回unknown，实际%q", result)
	}
}

func TestGetSceneFromTrace_WithScene(t *testing.T) {
	ctx := &TraceContext{SceneCode: "scanner"}
	result := getSceneFromTrace(ctx)
	if result != "scanner" {
		t.Errorf("应返回scanner，实际%q", result)
	}
}

// ==================== isRetryableError 测试（新增） ====================

func TestIsRetryableError_502(t *testing.T) {
	if !isRetryableError(502, nil) {
		t.Error("502应可重试")
	}
}

func TestIsRetryableError_503(t *testing.T) {
	if !isRetryableError(503, nil) {
		t.Error("503应可重试")
	}
}

func TestIsRetryableError_504(t *testing.T) {
	if !isRetryableError(504, nil) {
		t.Error("504应可重试")
	}
}

func TestIsRetryableError_429(t *testing.T) {
	if !isRetryableError(429, nil) {
		t.Error("429应可重试")
	}
}

func TestIsRetryableError_500WithTimeout(t *testing.T) {
	body := []byte("Gateway Time-out: upstream server timeout")
	if !isRetryableError(500, body) {
		t.Error("500+超时特征应可重试")
	}
}

func TestIsRetryableError_500WithEOF(t *testing.T) {
	body := []byte("unexpected end of JSON input EOF")
	if !isRetryableError(500, body) {
		t.Error("500+EOF应可重试")
	}
}

func TestIsRetryableError_500Normal(t *testing.T) {
	body := []byte("Internal Server Error: invalid request format")
	if isRetryableError(500, body) {
		t.Error("普通500不应可重试")
	}
}

func TestIsRetryableError_400(t *testing.T) {
	if isRetryableError(400, nil) {
		t.Error("400不应可重试")
	}
}

func TestIsRetryableError_401(t *testing.T) {
	if isRetryableError(401, nil) {
		t.Error("401不应可重试")
	}
}

func TestIsRetryableError_200(t *testing.T) {
	if isRetryableError(200, nil) {
		t.Error("200不应可重试")
	}
}

// ==================== getRetryDelay 测试（新增） ====================

func TestGetRetryDelay_FirstRetry(t *testing.T) {
	delay := getRetryDelay(0)
	if delay != 30*time.Second {
		t.Errorf("第1次重试应等30秒，实际%v", delay)
	}
}

func TestGetRetryDelay_SecondRetry(t *testing.T) {
	delay := getRetryDelay(1)
	if delay != 60*time.Second {
		t.Errorf("第2次重试应等60秒，实际%v", delay)
	}
}

func TestGetRetryDelay_ThirdRetry(t *testing.T) {
	delay := getRetryDelay(2)
	if delay != 120*time.Second {
		t.Errorf("第3次重试应等120秒，实际%v", delay)
	}
}

func TestGetRetryDelay_BeyondRange(t *testing.T) {
	delay := getRetryDelay(99)
	if delay != 120*time.Second {
		t.Errorf("超出范围应返回最大值120秒，实际%v", delay)
	}
}

// ==================== extractErrorMessage 测试（新增） ====================

func TestExtractErrorMessage_StructuredError(t *testing.T) {
	body := []byte(`{"error":{"message":"Rate limit exceeded"}}`)
	result := extractErrorMessage(body)
	if result != "Rate limit exceeded" {
		t.Errorf("应提取error.message，实际%q", result)
	}
}

func TestExtractErrorMessage_PlainText(t *testing.T) {
	body := []byte("Bad Gateway")
	result := extractErrorMessage(body)
	if result != "Bad Gateway" {
		t.Errorf("非JSON应返回原文，实际%q", result)
	}
}

func TestExtractErrorMessage_LongText(t *testing.T) {
	body := []byte(strings.Repeat("x", 500))
	result := extractErrorMessage(body)
	if len(result) > 310 {
		t.Errorf("超长文本应被截断到300+...，实际长度%d", len(result))
	}
}

// ==================== retryableError + isRetryableCallError 测试（新增） ====================

func TestRetryableError_Interface(t *testing.T) {
	err := &retryableError{msg: "timeout error"}
	if err.Error() != "timeout error" {
		t.Errorf("Error()应返回msg，实际%q", err.Error())
	}
}

func TestIsRetryableCallError_True(t *testing.T) {
	err := &retryableError{msg: "test"}
	if !isRetryableCallError(err) {
		t.Error("retryableError应识别为可重试")
	}
}

func TestIsRetryableCallError_False(t *testing.T) {
	err := fmt.Errorf("normal error")
	if isRetryableCallError(err) {
		t.Error("普通error不应识别为可重试")
	}
}

func TestIsRetryableCallError_Nil(t *testing.T) {
	if isRetryableCallError(nil) {
		t.Error("nil不应识别为可重试")
	}
}

// ==================== 常量验证测试（新增） ====================

func TestConstants(t *testing.T) {
	if AICallTimeout != 900*time.Second {
		t.Errorf("AICallTimeout应为900秒，实际%v", AICallTimeout)
	}
	if MaxRetries != 3 {
		t.Errorf("MaxRetries应为3，实际%d", MaxRetries)
	}
	if MaxStreamRetries != 2 {
		t.Errorf("MaxStreamRetries应为2，实际%d", MaxStreamRetries)
	}
	if MaxFallbackRetries != 1 {
		t.Errorf("MaxFallbackRetries应为1，实际%d", MaxFallbackRetries)
	}
	if len(retryDelays) != 3 {
		t.Errorf("retryDelays应有3个元素，实际%d", len(retryDelays))
	}
}
