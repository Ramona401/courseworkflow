package services

import (
	"strings"
	"testing"
)

// TestTemplateRefinePreserveAppliesWithoutRewriting 验证样式变化生效，DOM和脚本仍保留。
func TestTemplateRefinePreserveAppliesWithoutRewriting(t *testing.T) {
	pages := []string{
		`<!DOCTYPE html>
<html>
<head>
<style>
.card{background:#112233;color:#445566;box-shadow:0 4px 12px rgba(0,0,0,0.20)}
</style>
</head>
<body>
<div id="lesson-card" class="card" onclick="showAnswer()">原始正文</div>
<script>
function showAnswer(){return "#112233"}
</script>
</body>
</html>`,
	}

	oldColors := map[string]string{
		"primary":    "#112233",
		"secondary":  "#223344",
		"background": "#FFFFFF",
		"accent":     "#334455",
		"text":       "#445566",
	}
	newColors := map[string]string{
		"primary":    "#2563EB",
		"secondary":  "#60A5FA",
		"background": "#F8FAFC",
		"accent":     "#F59E0B",
		"text":       "#1E293B",
	}
	oldVariables := map[string]string{
		"--cw-primary":      "#112233",
		"--cw-secondary":    "#223344",
		"--cw-bg":           "#FFFFFF",
		"--cw-accent":       "#334455",
		"--cw-text":         "#445566",
		"--cw-font-heading": "'Old Heading',sans-serif",
		"--cw-font-body":    "'Old Body',sans-serif",
		"--cw-radius":       "8px",
		"--cw-shadow":       "0 4px 12px rgba(0,0,0,0.20)",
	}
	newVariables := map[string]string{
		"--cw-primary":      "#2563EB",
		"--cw-secondary":    "#60A5FA",
		"--cw-bg":           "#F8FAFC",
		"--cw-accent":       "#F59E0B",
		"--cw-text":         "#1E293B",
		"--cw-font-heading": "'Noto Serif SC',serif",
		"--cw-font-body":    "'Noto Sans SC',sans-serif",
		"--cw-radius":       "16px",
		"--cw-shadow":       "0 8px 24px rgba(0,0,0,0.12)",
	}

	result, err := applyTemplateStyleRefinement(
		pages,
		oldColors,
		newColors,
		oldVariables,
		newVariables,
		`.card{border-radius:var(--cw-radius)!important;}`,
	)
	if err != nil {
		t.Fatalf("应用样式失败: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("页面数量发生变化: got=%d want=1", len(result))
	}

	html := result[0]

	requiredOriginalFragments := []string{
		`id="lesson-card"`,
		`onclick="showAnswer()"`,
		`function showAnswer()`,
		`原始正文`,
	}
	for _, fragment := range requiredOriginalFragments {
		if !strings.Contains(html, fragment) {
			t.Fatalf("原始结构或脚本丢失: %s", fragment)
		}
	}

	requiredNewFragments := []string{
		"#2563EB",
		"#1E293B",
		"0 8px 24px rgba(0,0,0,0.12)",
		templateRefineStyleStartMarker,
		`border-radius:var(--cw-radius)!important`,
	}
	for _, fragment := range requiredNewFragments {
		if !strings.Contains(html, fragment) {
			t.Fatalf("新样式未应用: %s", fragment)
		}
	}
}

// TestTemplateRefinePreserveReplacesOverrideBlock 验证多轮微调不会重复叠加CSS块。
func TestTemplateRefinePreserveReplacesOverrideBlock(t *testing.T) {
	pages := []string{
		`<div class="cw-page"><button class="action">开始</button></div>`,
	}

	colors := map[string]string{
		"primary":    "#2563EB",
		"secondary":  "#60A5FA",
		"background": "#F8FAFC",
		"accent":     "#F59E0B",
		"text":       "#1E293B",
	}
	variables := map[string]string{
		"--cw-primary":      "#2563EB",
		"--cw-secondary":    "#60A5FA",
		"--cw-bg":           "#F8FAFC",
		"--cw-accent":       "#F59E0B",
		"--cw-text":         "#1E293B",
		"--cw-font-heading": "serif",
		"--cw-font-body":    "sans-serif",
		"--cw-radius":       "12px",
		"--cw-shadow":       "0 4px 12px rgba(0,0,0,0.12)",
	}

	first, err := applyTemplateStyleRefinement(
		pages,
		colors,
		colors,
		variables,
		variables,
		`.action{border-radius:12px!important;}`,
	)
	if err != nil {
		t.Fatalf("第一次微调失败: %v", err)
	}

	second, err := applyTemplateStyleRefinement(
		first,
		colors,
		colors,
		variables,
		variables,
		`.action{border-radius:20px!important;}`,
	)
	if err != nil {
		t.Fatalf("第二次微调失败: %v", err)
	}

	html := second[0]

	if strings.Count(html, templateRefineStyleStartMarker) != 1 {
		t.Fatalf("CSS覆盖层发生重复叠加")
	}
	if strings.Contains(html, "border-radius:12px!important") {
		t.Fatalf("旧CSS覆盖规则未被替换")
	}
	if !strings.Contains(html, "border-radius:20px!important") {
		t.Fatalf("新CSS覆盖规则未写入")
	}
	if !strings.Contains(html, `<button class="action">开始</button>`) {
		t.Fatalf("原始按钮DOM发生变化")
	}
}

// TestTemplateRefinePreserveRejectsUnsafeCSS 验证外部资源和危险CSS被拒绝。
func TestTemplateRefinePreserveRejectsUnsafeCSS(t *testing.T) {
	unsafeCases := []string{
		`@import url("https://example.com/a.css");`,
		`.x{background:url("https://example.com/a.png")}`,
		`.x{width:expression(alert(1))}`,
		`.x{background:javascript:alert(1)}`,
		`</style><script>alert(1)</script>`,
	}

	for _, css := range unsafeCases {
		if _, err := sanitizeTemplateCSSOverrides(css); err == nil {
			t.Fatalf("危险CSS未被拒绝: %s", css)
		}
	}
}
