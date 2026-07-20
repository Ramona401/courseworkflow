package services

import (
	"strings"
	"testing"
)

func TestTemplateMasterDetectsHeaderNav(t *testing.T) {
	source := `<!DOCTYPE html><html><head><style>.main-header{height:70px}</style></head><body><header class="main-header"><span>学校</span><span>01/18</span></header><main><h1>主标题文字</h1></main></body></html>`

	start, end, ok := findTemplateNavRegion(source)
	if !ok {
		t.Fatal("expected header nav to be detected")
	}
	if start < 0 || end <= start {
		t.Fatalf("invalid nav range: %d %d", start, end)
	}

	wrapped := wrapTemplateNavRegion(source, start, end)
	if !strings.Contains(wrapped, cwNavStartMarker) || !strings.Contains(wrapped, cwNavEndMarker) {
		t.Fatal("expected NAV markers")
	}

	nav := ExtractNavByMarkers(wrapped)
	if !strings.Contains(nav, `class="main-header"`) {
		t.Fatalf("unexpected nav: %s", nav)
	}
}

func TestTemplateMasterVisibleTextReplacementPreservesStructure(t *testing.T) {
	source := `<section class="cw-page"><style>.x{color:red}</style><h1>主标题文字</h1><p>副标题文字说明</p><script>const x = "不要修改";</script></section>`

	segments, slots := scanTemplateVisibleText(source)
	if len(slots) != 2 {
		t.Fatalf("expected 2 visible slots, got %d", len(slots))
	}

	out := applyTemplateMasterReplacements(segments, []cwMasterReplacement{
		{Slot: 1, Text: "新的标题"},
		{Slot: 2, Text: "新的副标题"},
	})

	if !strings.Contains(out, `<style>.x{color:red}</style>`) {
		t.Fatal("style block changed")
	}
	if !strings.Contains(out, `const x = "不要修改"`) {
		t.Fatal("script block changed")
	}
	if !strings.Contains(out, `<h1>新的标题</h1>`) {
		t.Fatal("title was not replaced")
	}
	if !strings.Contains(out, `<p>新的副标题</p>`) {
		t.Fatal("subtitle was not replaced")
	}
}

func TestTemplateMasterInsertsGeneratedNavIntoBody(t *testing.T) {
	source := `<html><body><main>正文</main></body></html>`
	nav := `<div class="generated-nav">导航</div>`

	out := insertGeneratedNavIntoTemplate(source, nav)

	if !strings.Contains(out, cwNavStartMarker) || !strings.Contains(out, cwNavEndMarker) {
		t.Fatal("expected generated NAV markers")
	}
	if !strings.Contains(out, nav) {
		t.Fatal("generated nav missing")
	}
	if !strings.Contains(out, `<main>正文</main>`) {
		t.Fatal("body content changed")
	}
}

func TestTemplateNavPageNumberPreservesOriginalFormat(t *testing.T) {
	nav := `<header class="main-header"><div class="page-number">01/18</div></header>`

	template := StripNavPageNumbers(nav)
	if !strings.Contains(template, `{{PAGE_NUM_2}}/{{TOTAL_PAGES}}`) {
		t.Fatalf("page format placeholder not preserved: %s", template)
	}

	out := injectPageNumIntoNav(template, 3, 20)
	if !strings.Contains(out, `>03/20<`) {
		t.Fatalf("expected 03/20, got: %s", out)
	}
}

func TestTemplateHeaderWithoutPageNumberAppendsSibling(t *testing.T) {
	nav := `<header class="ct-top"><div class="ct-brand">品牌</div><div class="ct-chip">任务</div></header>`

	out := injectPageNumIntoNav(nav, 1, 8)

	taskEnd := strings.Index(out, `任务</div>`)
	pageIndex := strings.Index(out, `1 / 8`)
	headerEnd := strings.LastIndex(out, `</header>`)

	if taskEnd < 0 || pageIndex < 0 || headerEnd < 0 {
		t.Fatalf("unexpected output: %s", out)
	}
	if !(taskEnd < pageIndex && pageIndex < headerEnd) {
		t.Fatalf(
			"fallback page number was inserted into an inner div: %s",
			out,
		)
	}
}
