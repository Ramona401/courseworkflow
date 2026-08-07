package handlers

// assistant_runtime_embed_security_unit25_test.go
//
// 开发单元25公开iframe安全响应回归测试。
//
// 测试不连接数据库、不启动服务、不访问网络，也不创建AI会话。
// 重点验证：
//   - frame-ancestors只包含部署明确允许的Origin，不隐式加入self；
//   - 重复Origin被去重，公网HTTP和非精确Origin被拒绝；
//   - embed HTML同时返回same-origin响应头和meta策略；
//   - 允许来源只进入CSP响应头，不进入HTML正文；
//   - 无效公开描述或安全策略返回收敛的500响应；
//   - 已保存部署来源策略损坏稳定映射为503。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tedna/internal/models"
	"tedna/internal/services"
)

func TestAssistantRuntimeFrameAncestorsCSPUnit25(t *testing.T) {
	actual, err := assistantRuntimeFrameAncestorsCSP(
		[]string{
			"https://classroom.example.edu",
			"https://classroom.example.edu/",
			"http://localhost:5173",
		},
	)
	if err != nil {
		t.Fatalf("有效frame-ancestors不应失败：%v", err)
	}

	const expected = "https://classroom.example.edu http://localhost:5173"
	if actual != expected {
		t.Fatalf("frame-ancestors期望为%q，实际为%q", expected, actual)
	}
	if strings.Contains(actual, "'self'") {
		t.Fatalf("frame-ancestors不得隐式加入self：%q", actual)
	}

	invalidCases := []struct {
		name    string
		origins []string
	}{
		{name: "空来源", origins: nil},
		{name: "公网HTTP", origins: []string{"http://classroom.example.edu"}},
		{name: "带路径", origins: []string{"https://classroom.example.edu/course"}},
		{name: "带查询", origins: []string{"https://classroom.example.edu/?course=1"}},
		{name: "带引号", origins: []string{`https://classroom.example.edu"` + `"`}},
	}

	for _, test := range invalidCases {
		t.Run(test.name, func(t *testing.T) {
			if value, err := assistantRuntimeFrameAncestorsCSP(test.origins); err == nil {
				t.Fatalf("无效frame-ancestors应被拒绝，实际返回%q", value)
			}
		})
	}
}

func unit25PublicDescriptor(
	frameAncestors []string,
) *models.AssistantDeploymentPublicDescriptor {
	return &models.AssistantDeploymentPublicDescriptor{
		PublicID:            "public_ABC-123",
		Title:               "<数学引导助手>",
		WelcomeMessage:      `欢迎<script>alert("x")</script>`,
		DisplayMode:         "floating",
		DisplayPosition:     "bottom_right",
		MaximumSessionTurns: 8,
		FrameAncestors:      frameAncestors,
	}
}

func TestWriteAssistantRuntimeEmbedHTMLSecurityHeadersUnit25(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeAssistantRuntimeEmbedHTML(
		recorder,
		unit25PublicDescriptor(
			[]string{
				"https://classroom.example.edu",
			},
		),
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("安全HTML期望HTTP 200，实际为%d", result.StatusCode)
	}

	if actual := result.Header.Get("Referrer-Policy"); actual != "same-origin" {
		t.Fatalf("Referrer-Policy期望same-origin，实际为%q", actual)
	}

	csp := result.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors https://classroom.example.edu") {
		t.Fatalf("CSP缺少精确允许来源：%q", csp)
	}
	if strings.Contains(csp, "frame-ancestors 'self'") {
		t.Fatalf("CSP不得隐式允许self嵌入：%q", csp)
	}
	if !strings.Contains(csp, "script-src 'self'") ||
		!strings.Contains(csp, "connect-src 'self'") {
		t.Fatalf("CSP必须保持脚本和运行API同源限制：%q", csp)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, `<meta name="referrer" content="same-origin">`) {
		t.Fatal("HTML缺少same-origin referrer meta")
	}
	if !strings.Contains(body, `<script type="module" src="/assets/assistant-embed.js"></script>`) {
		t.Fatal("HTML缺少固定学生端模块")
	}
	if strings.Contains(body, "https://classroom.example.edu") {
		t.Fatal("部署允许来源不得进入HTML正文")
	}
	if strings.Contains(body, `<script>alert("x")</script>`) {
		t.Fatal("欢迎语中的HTML不得作为活动脚本输出")
	}
	if !strings.Contains(body, "&lt;数学引导助手&gt;") {
		t.Fatal("标题应进行HTML转义")
	}
}

func TestWriteAssistantRuntimeEmbedHTMLRejectsInvalidDescriptorUnit25(t *testing.T) {
	t.Run("空公开描述", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		writeAssistantRuntimeEmbedHTML(recorder, nil)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("空公开描述期望HTTP 500，实际为%d", recorder.Code)
		}
		if recorder.Header().Get("Content-Security-Policy") != "" {
			t.Fatal("空公开描述不得输出不完整CSP")
		}
	})

	t.Run("空允许来源", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		writeAssistantRuntimeEmbedHTML(
			recorder,
			unit25PublicDescriptor(nil),
		)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("空允许来源期望HTTP 500，实际为%d", recorder.Code)
		}
		if recorder.Header().Get("Content-Security-Policy") != "" {
			t.Fatal("无效允许来源不得输出不完整CSP")
		}
		if strings.Contains(recorder.Body.String(), "public_ABC-123") {
			t.Fatal("安全策略无效时不得输出公开部署HTML")
		}
	})
}

func TestAssistantRuntimeStoredPolicyInvalidMapsToServiceUnavailableUnit25(t *testing.T) {
	status, message := assistantRuntimeHTTPStatusAndMessage(
		services.ErrAssistantDeploymentStoredPolicyInvalid,
	)

	if status != http.StatusServiceUnavailable {
		t.Fatalf("已保存策略损坏期望HTTP 503，实际为%d", status)
	}
	if message != "教学智能体暂时不可用，请稍后重试" {
		t.Fatalf("公开错误文案不符合收敛要求：%q", message)
	}
}
