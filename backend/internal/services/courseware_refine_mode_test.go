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
