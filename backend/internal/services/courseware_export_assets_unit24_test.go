package services

// courseware_export_assets_unit24_test.go
//
// 纯离线HTML包装与README回归测试。
//
// 测试不访问网络、不连接数据库、不创建会话，也不产生AI费用。
// 验证页面包装、导航、播放器说明和离线智能体边界。

import (
	"strings"
	"testing"
)

func TestBuildOfflinePageDocWrapsFragmentWithoutAssistant(t *testing.T) {
	document := buildOfflinePageDoc(
		`<section id="lesson-content">页面正文</section>`,
		1,
		2,
		"第一页",
		"测试课件",
	)

	required := []string{
		`<!DOCTYPE html>`,
		`id="cw-stage"`,
		`id="lesson-content"`,
		`class="cw-ofl-nav"`,
		`p2.html`,
	}

	for _, value := range required {
		if !strings.Contains(document, value) {
			t.Fatalf("离线页面缺少必要结构：%s", value)
		}
	}

	if containsCoursewareExportAssistantArtifacts(document) {
		t.Fatal("纯离线页面不得包含教学智能体痕迹")
	}
}

func TestBuildOfflinePageDocPreservesFullDocumentAndAddsNavigation(t *testing.T) {
	source := `<!doctype html><html><head><title>完整页</title></head><body><main>3D正文</main></body></html>`

	document := buildOfflinePageDoc(
		source,
		2,
		3,
		"完整页",
		"测试课件",
	)

	if !strings.Contains(document, "<main>3D正文</main>") {
		t.Fatal("完整HTML正文必须保留")
	}
	if !strings.Contains(document, `id="cw-ofl-nav"`) {
		t.Fatal("完整HTML页面必须注入框架感知离线导航")
	}
	if !strings.Contains(document, "p1.html") ||
		!strings.Contains(document, "p3.html") {
		t.Fatal("完整HTML页面必须包含正确的前后页地址")
	}
	if containsCoursewareExportAssistantArtifacts(document) {
		t.Fatal("完整HTML离线页面不得包含教学智能体痕迹")
	}
}

func TestBuildOfflineReadmeDeclaresPureOfflineBoundary(t *testing.T) {
	readme := buildOfflineReadme(
		"测试课件",
		5,
	)

	required := []string{
		"教学智能体：不包含",
		"本ZIP是纯离线课件包",
		"不包含教学智能体悬浮入口",
		"只包含课件页面、离线播放器和已打包媒体",
		"不会尝试连接教学智能体服务",
		"TE-DNA平台使用登录态预览",
		"在线教学智能体与本离线ZIP是两种独立交付方式",
	}

	for _, value := range required {
		if !strings.Contains(readme, value) {
			t.Fatalf("纯离线使用说明缺少内容：%s", value)
		}
	}
}

func TestBuildOfflineReadmeDoesNotExposeAssistantRuntimeData(t *testing.T) {
	readme := buildOfflineReadme(
		"测试课件",
		5,
	)

	forbidden := []string{
		"TEDNA-ASSISTANT-EXPORT",
		"tedna-assistant-export",
		"public_id",
		"deployment_id",
		"runtime_token",
		"data-public-id",
		"data-base-url",
		"/embed/assistant/",
		"/api/v1/assistant-runtime/",
		"workflow.pkuailab.com",
	}

	for _, value := range forbidden {
		if strings.Contains(readme, value) {
			t.Fatalf("纯离线使用说明包含运行数据或注入代码：%s", value)
		}
	}
}
