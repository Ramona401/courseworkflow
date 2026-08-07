package services

// courseware_export_assistant_unit24_test.go
//
// 历史导出助手痕迹清理回归测试。
//
// 测试不连接数据库、不查询部署、不访问网络、不创建会话。
// 验证完整旧注入块可清除，而异常孤立残留会被识别并阻断离线导出。

import (
	"strings"
	"testing"
)

func TestStripCoursewareExportAssistantArtifactsRemovesMarkedBlock(t *testing.T) {
	source := `<!DOCTYPE html><html><body>
<main>正常课件正文</main>
<!-- TEDNA-ASSISTANT-EXPORT:BEGIN -->
<style id="tedna-assistant-export-style">.x{display:block}</style>
<div id="tedna-assistant-export-root" data-public-id="public_ABC-123">
<button id="tedna-assistant-export-launcher">AI教学助手</button>
</div>
<script>frame.src="https://workflow.pkuailab.com/embed/assistant/public_ABC-123";</script>
<!-- TEDNA-ASSISTANT-EXPORT:END -->
<footer>课件页脚</footer>
</body></html>`

	cleaned := stripCoursewareExportAssistantArtifacts(source)

	if !strings.Contains(cleaned, "<main>正常课件正文</main>") {
		t.Fatal("清理旧助手片段时不得删除正常课件正文")
	}
	if !strings.Contains(cleaned, "<footer>课件页脚</footer>") {
		t.Fatal("清理旧助手片段时不得删除注入块之后的正文")
	}
	if containsCoursewareExportAssistantArtifacts(cleaned) {
		t.Fatalf("完整旧助手片段清理后仍有残留：%s", cleaned)
	}
}

func TestStripCoursewareExportAssistantArtifactsIsIdempotent(t *testing.T) {
	source := `<!DOCTYPE html><html><body><main>纯课件</main></body></html>`

	first := stripCoursewareExportAssistantArtifacts(source)
	second := stripCoursewareExportAssistantArtifacts(first)

	if first != source {
		t.Fatal("没有助手片段的页面必须保持完全不变")
	}
	if second != first {
		t.Fatal("重复清理必须保持幂等")
	}
}

func TestContainsCoursewareExportAssistantArtifactsDetectsResiduals(t *testing.T) {
	tests := []struct {
		name     string
		document string
	}{
		{
			name:     "孤立稳定标记",
			document: `<!-- TEDNA-ASSISTANT-EXPORT:BEGIN -->`,
		},
		{
			name:     "孤立旧DOM",
			document: `<div id="tedna-assistant-export-root"></div>`,
		},
		{
			name:     "公开编号数据属性",
			document: `<div data-public-id="public_123"></div>`,
		},
		{
			name:     "在线运行站点属性",
			document: `<div data-base-url="https://workflow.pkuailab.com"></div>`,
		},
		{
			name:     "在线embed地址",
			document: `<iframe src="/embed/assistant/public_123"></iframe>`,
		},
		{
			name:     "教学智能体运行API",
			document: `<script>fetch("/api/v1/assistant-runtime/sessions/session_123")</script>`,
		},
		{
			name:     "短时运行令牌字段",
			document: `<script>var runtime_token="secret";</script>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !containsCoursewareExportAssistantArtifacts(test.document) {
				t.Fatalf("应识别教学智能体残留：%s", test.document)
			}
		})
	}
}

func TestIncompleteMarkedBlockIsNotDestructivelyRemoved(t *testing.T) {
	source := `<main>正常正文</main><!-- TEDNA-ASSISTANT-EXPORT:BEGIN --><div>异常残片</div>`

	cleaned := stripCoursewareExportAssistantArtifacts(source)

	if cleaned != source {
		t.Fatal("缺少END边界时不得猜测性删除页面后半部分")
	}
	if !containsCoursewareExportAssistantArtifacts(cleaned) {
		t.Fatal("不完整标记块必须被残留检查识别，以便编排服务阻断导出")
	}
}
