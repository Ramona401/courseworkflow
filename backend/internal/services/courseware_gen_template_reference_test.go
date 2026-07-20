package services

import (
	"strings"
	"testing"
)

// TestCWTemplatePageReferenceExtract 验证内部引用标记会被解析并从老师指令中删除。
func TestCWTemplatePageReferenceExtract(t *testing.T) {
	instruction := `请参考所选模板页的交互逻辑，把本页改成可点击分步展示。

<!-- TEDNA_TEMPLATE_PAGE_REF {"template_id":"tpl-123","sample_page_index":3} -->`

	cleaned, ref, err := extractCWTemplatePageReference(instruction)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if ref == nil {
		t.Fatal("未解析到模板页引用")
	}
	if ref.TemplateID != "tpl-123" {
		t.Fatalf("模板ID错误: %s", ref.TemplateID)
	}
	if ref.SamplePageIndex != 3 {
		t.Fatalf("页面下标错误: %d", ref.SamplePageIndex)
	}
	if strings.Contains(cleaned, "TEDNA_TEMPLATE_PAGE_REF") {
		t.Fatal("内部引用标记未从老师指令中删除")
	}
	if !strings.Contains(cleaned, "交互逻辑") {
		t.Fatal("老师原始指令丢失")
	}
}

// TestCWTemplatePageReferenceRejectsMultiple 验证一次请求不能注入多个模板页。
func TestCWTemplatePageReferenceRejectsMultiple(t *testing.T) {
	instruction := `
<!-- TEDNA_TEMPLATE_PAGE_REF {"template_id":"a","sample_page_index":0} -->
请修改页面。
<!-- TEDNA_TEMPLATE_PAGE_REF {"template_id":"b","sample_page_index":1} -->
`

	if _, _, err := extractCWTemplatePageReference(instruction); err == nil {
		t.Fatal("重复模板页引用未被拒绝")
	}
}

// TestCWTemplatePageReferenceTruncateKeepsHeadAndTail 验证超长参考页保留头尾。
func TestCWTemplatePageReferenceTruncateKeepsHeadAndTail(t *testing.T) {
	head := "<style>.card{color:red}</style>"
	middle := strings.Repeat("中间内容", 16000)
	tail := "<script>function importantInteraction(){return true}</script>"

	source := head + middle + tail
	result, truncated := truncateCWTemplatePageReference(source)

	if !truncated {
		t.Fatal("超长模板页未执行截断")
	}
	if !strings.Contains(result, head) {
		t.Fatal("模板页头部HTML/CSS丢失")
	}
	if !strings.Contains(result, "importantInteraction") {
		t.Fatal("模板页尾部JavaScript丢失")
	}
	if !strings.Contains(result, "中间部分因体量过大已省略") {
		t.Fatal("截断说明标记缺失")
	}
}

// TestCWTemplatePageReferencePromptRules 验证提示词明确由老师指令决定参考用途。
func TestCWTemplatePageReferencePromptRules(t *testing.T) {
	ref := &cwResolvedTemplatePageReference{
		TemplateID:      "tpl-1",
		TemplateName:    "测试模板",
		TemplateScope:   "personal",
		SamplePageIndex: 2,
		SamplePageHTML:  `<div onclick="nextStep()">参考页</div>`,
	}

	prompt := buildCWTemplatePageReferencePrompt(ref)

	required := []string{
		"模板第3页",
		"老师本次修改指令决定",
		"样式、布局、交互逻辑或两者结合",
		"不得照抄参考页中的教学文字",
		"不得复制参考页的导航栏",
		"nextStep",
	}

	for _, fragment := range required {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("提示词缺少关键规则: %s", fragment)
		}
	}
}
