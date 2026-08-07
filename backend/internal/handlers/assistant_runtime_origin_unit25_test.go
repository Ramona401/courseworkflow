package handlers

// assistant_runtime_origin_unit25_test.go
//
// 开发单元25会话创建来源三方绑定回归测试。
//
// 测试不连接数据库、不创建会话、不访问网络，也不产生AI费用。
// 重点验证：
//   - HTTPS公网Origin、本机HTTP和回环HTTP可以被规范化；
//   - 公网HTTP、路径、查询、片段和用户凭据被拒绝；
//   - 会话创建必须同时匹配当前站点Origin、官方embed Referer和public_id；
//   - Referer查询参数、片段、跨源地址、尾部斜线和错误public_id全部被拒绝；
//   - 反向代理协议字段必须为HTTP或HTTPS。

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func unit25AssistantRuntimeStartRequest(origin string, referer string) *http.Request {
	request := httptest.NewRequest(
		http.MethodPost,
		"http://127.0.0.1/api/v1/assistant-runtime/deployments/public_ABC-123/session",
		nil,
	)

	request.Host = "workflow.pkuailab.com"
	request.Header.Set("X-Forwarded-Proto", "https")

	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if referer != "" {
		request.Header.Set("Referer", referer)
	}

	return request
}

func unit25AssertCanonicalOrigin(
	t *testing.T,
	name string,
	input string,
	expected string,
) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		actual, err := assistantRuntimeCanonicalHTTPOrigin(input)
		if err != nil {
			t.Fatalf("有效Origin不应失败：%v", err)
		}
		if actual != expected {
			t.Fatalf("Origin规范化期望为%q，实际为%q", expected, actual)
		}
	})
}

func unit25AssertInvalidOrigin(
	t *testing.T,
	name string,
	input string,
) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		if actual, err := assistantRuntimeCanonicalHTTPOrigin(input); err == nil {
			t.Fatalf("无效Origin应被拒绝，实际返回%q", actual)
		}
	})
}

func TestAssistantRuntimeCanonicalHTTPOriginUnit25(t *testing.T) {
	unit25AssertCanonicalOrigin(
		t,
		"HTTPS公网默认端口",
		"https://Classroom.Example.EDU:443/",
		"https://classroom.example.edu",
	)
	unit25AssertCanonicalOrigin(
		t,
		"localhost默认HTTP端口",
		"http://localhost:80",
		"http://localhost",
	)
	unit25AssertCanonicalOrigin(
		t,
		"IPv4回环自定义端口",
		"http://127.0.0.1:8080",
		"http://127.0.0.1:8080",
	)
	unit25AssertCanonicalOrigin(
		t,
		"IPv6回环自定义端口",
		"http://[::1]:8080",
		"http://[::1]:8080",
	)

	unit25AssertInvalidOrigin(t, "空Origin", "")
	unit25AssertInvalidOrigin(t, "公网HTTP", "http://classroom.example.edu")
	unit25AssertInvalidOrigin(t, "带路径", "https://classroom.example.edu/course")
	unit25AssertInvalidOrigin(t, "带查询", "https://classroom.example.edu/?course=1")
	unit25AssertInvalidOrigin(t, "带片段", "https://classroom.example.edu/#chapter")
	unit25AssertInvalidOrigin(t, "带用户凭据", "https://teacher@classroom.example.edu")
	unit25AssertInvalidOrigin(t, "不支持协议", "ftp://classroom.example.edu")
}

func unit25AssertStartRequestContext(
	t *testing.T,
	name string,
	origin string,
	referer string,
	publicID string,
	shouldPass bool,
) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		request := unit25AssistantRuntimeStartRequest(origin, referer)
		err := validateAssistantRuntimeStartRequestContext(request, publicID)

		if shouldPass && err != nil {
			t.Fatalf("有效三方绑定不应失败：%v", err)
		}
		if !shouldPass && err == nil {
			t.Fatal("无效三方绑定应被拒绝")
		}
	})
}

func TestValidateAssistantRuntimeStartRequestContextUnit25(t *testing.T) {
	const publicID = "public_ABC-123"
	const validOrigin = "https://workflow.pkuailab.com"
	const validReferer = "https://workflow.pkuailab.com/embed/assistant/public_ABC-123"

	unit25AssertStartRequestContext(
		t,
		"精确三方绑定",
		validOrigin,
		validReferer,
		publicID,
		true,
	)
	unit25AssertStartRequestContext(
		t,
		"Origin默认端口等价",
		"https://workflow.pkuailab.com:443",
		"https://workflow.pkuailab.com:443/embed/assistant/public_ABC-123",
		publicID,
		true,
	)
	unit25AssertStartRequestContext(
		t,
		"缺少Origin",
		"",
		validReferer,
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"跨源Origin",
		"https://attacker.example",
		validReferer,
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"缺少Referer",
		validOrigin,
		"",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"跨源Referer",
		validOrigin,
		"https://attacker.example/embed/assistant/public_ABC-123",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"错误公开编号Referer",
		validOrigin,
		"https://workflow.pkuailab.com/embed/assistant/public_OTHER",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"Referer带查询参数",
		validOrigin,
		validReferer+"?source=test",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"Referer带片段",
		validOrigin,
		validReferer+"#fragment",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"Referer带尾部斜线",
		validOrigin,
		validReferer+"/",
		publicID,
		false,
	)
	unit25AssertStartRequestContext(
		t,
		"公开编号包含路径",
		validOrigin,
		validReferer,
		"public/invalid",
		false,
	)
}

func TestAssistantRuntimeExpectedRequestOriginRejectsInvalidForwardedProtocolUnit25(t *testing.T) {
	request := unit25AssistantRuntimeStartRequest(
		"https://workflow.pkuailab.com",
		"https://workflow.pkuailab.com/embed/assistant/public_ABC-123",
	)
	request.Header.Set("X-Forwarded-Proto", "ftp")

	if origin, err := assistantRuntimeExpectedRequestOrigin(request); err == nil {
		t.Fatalf("无效反向代理协议应被拒绝，实际返回%q", origin)
	}
}
