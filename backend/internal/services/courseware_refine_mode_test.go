package services

import (
	"strings"
	"testing"
)

func TestNormalizeCWRefineMode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "preserve", want: cwRefineModePreserve},
		{input: "rebuild", want: cwRefineModeRebuild},
		{input: " REBUILD ", want: cwRefineModeRebuild},
		{input: "", want: cwRefineModePreserve},
		{input: "unknown", want: cwRefineModePreserve},
	}

	for _, tc := range tests {
		if got := normalizeCWRefineMode(tc.input); got != tc.want {
			t.Fatalf("normalizeCWRefineMode(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestExtractHTMLPrefersFullDocument(t *testing.T) {
	input := `模型说明
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<style>.title{color:red}</style>
</head>
<body>
<header><div id="brand">品牌栏</div></header>
<main><div id="content">正文</div></main>
<script>function runPage(){ return true }</script>
</body>
</html>
模型尾注`

	got := (&CoursewareGenService{}).extractHTMLFromAIOutput(input)
	lower := strings.ToLower(strings.TrimSpace(got))

	if !strings.HasPrefix(lower, "<!doctype html") {
		t.Fatalf("expected complete document start, got: %s", got)
	}
	if !strings.Contains(got, `<style>.title{color:red}</style>`) {
		t.Fatal("complete document extraction lost head/style")
	}
	if !strings.Contains(got, `function runPage()`) {
		t.Fatal("complete document extraction lost tail script")
	}
	if !strings.HasSuffix(lower, "</html>") {
		t.Fatal("complete document extraction did not retain </html>")
	}
}

func TestRebuildValidationAllowsReplacingOldAssets(t *testing.T) {
	oldHTML := `<div style="width:1920px;height:1080px">
<div id="old-a"></div>
<div id="old-b"></div>
<div id="old-c"></div>
<div id="old-d" onclick="oldOne()"></div>
<script>
function oldOne(){}
function oldTwo(){}
</script>
</div>`

	newHTML := `<div style="width:1920px;height:1080px">
<section id="new-layout">
<div id="new-a"></div>
<div id="new-b"></div>
<div id="new-c" onclick="newOne()"></div>
</section>
<script>
function newOne(){}
</script>
</div>`

	preserve := validateRefinedPageHTML(
		oldHTML,
		newHTML,
		"重新设计本页内容区",
		false,
	)
	if preserve.OK {
		t.Fatal("preserve mode should reject loss of old functions and IDs")
	}

	rebuild := validateRefinedPageHTML(
		oldHTML,
		newHTML,
		"重新设计本页内容区",
		true,
	)
	if !rebuild.OK {
		t.Fatalf("rebuild mode should allow replacing old assets: %s", rebuild.Reason)
	}
}

func TestAutoAssemblyStyleInsertionSkipsUnclosedScript(t *testing.T) {
	html := `<div class="cw-page"><script>
function resetGame() {
  document.body.innerHTML = '<div>broken</div>';
}`

	styleTag := `<style>/* TEDNA-AUTO-PRESENTATION */</style>`
	got := insertAutoAssemblyStyleAtDocumentEnd(html, styleTag)

	if got != html {
		t.Fatalf("unclosed script must remain untouched, got: %s", got)
	}
}

func TestAutoAssemblyStyleInsertionUsesRealRootClose(t *testing.T) {
	html := `<div class="cw-page">
<script>
const template = "<div>inside-script</div>";
</script>
</div>`

	styleTag := `<style>/* TEDNA-AUTO-PRESENTATION */</style>`
	got := insertAutoAssemblyStyleAtDocumentEnd(html, styleTag)

	markerIndex := strings.Index(got, "TEDNA-AUTO-PRESENTATION")
	scriptCloseIndex := strings.Index(got, "</script>")
	rootCloseIndex := strings.LastIndex(got, "</div>")

	if markerIndex < 0 {
		t.Fatal("presentation style was not inserted")
	}
	if markerIndex <= scriptCloseIndex {
		t.Fatal("presentation style was inserted inside script content")
	}
	if markerIndex >= rootCloseIndex {
		t.Fatal("presentation style must remain inside the page root")
	}
}

func TestRebuildValidationRemovesSingleTrailingExtraDiv(t *testing.T) {
	html := `<div class="cw-page">
<div id="content"></div>
<script>
const template = "<div>inside-script</div>";
</script>
</div>
</div>`

	result := validateRefinedPageHTML(
		"",
		html,
		"重构页面",
		true,
	)

	if !result.OK {
		t.Fatalf("single trailing extra div should be repaired: %s", result.Reason)
	}
	if result.FixedHTML == "" {
		t.Fatal("expected repaired HTML")
	}

	openCount, closeCount := cwCountDivTags(result.FixedHTML)
	if openCount != closeCount {
		t.Fatalf("repaired HTML is still unbalanced: %d/%d", openCount, closeCount)
	}
	if !strings.Contains(result.Detail, "auto_remove_extra_div") {
		t.Fatalf("unexpected repair detail: %s", result.Detail)
	}
}

func TestRebuildValidationRejectsInteriorExtraDiv(t *testing.T) {
	html := `<div class="cw-page"><div></div></div></div><span>tail</span>`

	result := validateRefinedPageHTML(
		"",
		html,
		"重构页面",
		true,
	)

	if result.OK {
		t.Fatal("interior extra div must not be guessed or removed")
	}
	if !strings.Contains(result.Reason, "多余闭合标签") {
		t.Fatalf("unexpected error reason: %s", result.Reason)
	}
}

func TestExtractHTMLFragmentRetainsSiblingScriptAndStyle(t *testing.T) {
	input := `模型说明
<div class="cw-page">
  <div id="content">正文</div>
</div>
<script>
const normal = '<div class="status">正在抽取</div>';
const template = ` + "`<div class=\"result\">${normal}</div>`" + `;
function runWheel(){ return template; }
</script>
<style>
.result::after{content:"</div>";}
</style>
模型尾注`

	got := (&CoursewareGenService{}).
		extractHTMLFromAIOutput(
			input,
		)

	if !strings.HasPrefix(
		strings.ToLower(
			strings.TrimSpace(
				got,
			),
		),
		"<div",
	) {
		t.Fatalf(
			"expected fragment root, got: %s",
			got,
		)
	}

	if !strings.Contains(
		got,
		"function runWheel()",
	) ||
		!strings.Contains(
			got,
			"</script>",
		) {
		t.Fatalf(
			"fragment extraction lost sibling script: %s",
			got,
		)
	}

	if !strings.Contains(
		got,
		".result::after",
	) ||
		!strings.Contains(
			got,
			"</style>",
		) {
		t.Fatalf(
			"fragment extraction lost sibling style: %s",
			got,
		)
	}

	if strings.Contains(
		got,
		"模型尾注",
	) {
		t.Fatal(
			"fragment extraction retained trailing model prose",
		)
	}

	result := validateRefinedPageHTML(
		"",
		got,
		"重构页面",
		true,
	)

	if !result.OK {
		t.Fatalf(
			"extracted fragment should pass validation: %s detail=%s",
			result.Reason,
			result.Detail,
		)
	}
}

func TestExtractHTMLFragmentRetainsUnclosedScriptForValidation(t *testing.T) {
	input := `<div class="cw-page">
<div id="content">正文</div>
</div>
<script>
const template = "<div>inside-script</div>";
function broken(){`

	got := (&CoursewareGenService{}).
		extractHTMLFromAIOutput(
			input,
		)

	if !strings.Contains(
		got,
		"<script>",
	) {
		t.Fatal(
			"extractor must retain an unclosed sibling script for validation",
		)
	}

	result := validateRefinedPageHTML(
		"",
		got,
		"重构页面",
		true,
	)

	if result.OK {
		t.Fatal(
			"unclosed sibling script must be rejected",
		)
	}

	if !strings.Contains(
		result.Reason,
		"脚本块未闭合",
	) {
		t.Fatalf(
			"unexpected validation reason: %s",
			result.Reason,
		)
	}
}

func TestHTMLStructureIgnoresPseudoDivsInsideRawText(t *testing.T) {
	html := `<div class="cw-page">
<div id="real">正文</div>
<script>
const one = "<div>inside-script</div>";
const two = ` + "`<div>${one}</div>`" + `;
</script>
<style>
.demo::after{content:"<div></div>";}
</style>
</div>`

	structure :=
		cwScanHTMLStructure(
			html,
		)

	if structure.DivOpen != 2 ||
		structure.DivClose != 2 {
		t.Fatalf(
			"pseudo divs inside script/style must be ignored: %s",
			cwDescribeHTMLStructure(
				html,
			),
		)
	}

	if structure.ScriptOpen != 1 ||
		structure.ScriptClose != 1 ||
		structure.StyleOpen != 1 ||
		structure.StyleClose != 1 {
		t.Fatalf(
			"unexpected raw-text structure: %s",
			cwDescribeHTMLStructure(
				html,
			),
		)
	}
}

func TestExtractHTMLFragmentRetainsDetachedSiblingCanvas(t *testing.T) {
	input := `<div class="tedna-nav-shell">导航栏</div>
<div class="cw-page">内容画布</div>
<script>function ready(){ return true }</script>
解释文字`

	got := (&CoursewareGenService{}).
		extractHTMLFromAIOutput(
			input,
		)

	if !strings.Contains(
		got,
		`class="tedna-nav-shell"`,
	) ||
		!strings.Contains(
			got,
			`class="cw-page"`,
		) ||
		!strings.Contains(
			got,
			"function ready()",
		) {
		t.Fatalf(
			"detached sibling canvases were not preserved: %s",
			got,
		)
	}

	if strings.Contains(
		got,
		"解释文字",
	) {
		t.Fatal(
			"detached fragment extraction retained trailing prose",
		)
	}
}
