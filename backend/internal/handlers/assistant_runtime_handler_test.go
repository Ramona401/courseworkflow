package handlers

// assistant_runtime_handler_test.go
//
// 只验证不连接数据库的公开HTTP安全辅助：Bearer令牌、严格JSON、客户端IP、
// 错误收敛、SSE头和embed最小信息。真实数据库和网络流并发留到后端验收单元。

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
)

// TestAssistantRuntimeHTTPBearerToken 验证运行令牌只接受Authorization头。
func TestAssistantRuntimeHTTPBearerToken(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)
	request.Header.Set(
		"Authorization",
		"Bearer runtime-token",
	)

	token, err := extractAssistantRuntimeBearerToken(
		request,
	)
	if err != nil ||
		token != "runtime-token" {
		t.Fatalf(
			"合法Bearer令牌解析失败: token=%s error=%v",
			token,
			err,
		)
	}

	for _, header := range []string{
		"",
		"runtime-token",
		"Basic runtime-token",
		"Bearer",
		"Bearer a b",
	} {
		request.Header.Set(
			"Authorization",
			header,
		)

		if _, err := extractAssistantRuntimeBearerToken(
			request,
		); err == nil {
			t.Fatalf(
				"非法Authorization应被拒绝: %q",
				header,
			)
		}
	}
}

// TestAssistantRuntimeHTTPStrictJSON 验证未知字段和多JSON值被拒绝。
func TestAssistantRuntimeHTTPStrictJSON(
	t *testing.T,
) {
	type body struct {
		Message string `json:"message"`
	}

	cases := []struct {
		name    string
		payload string
		valid   bool
	}{
		{
			name:    "合法对象",
			payload: `{"message":"你好"}`,
			valid:   true,
		},
		{
			name:    "未知字段",
			payload: `{"message":"你好","owner_id":"x"}`,
			valid:   false,
		},
		{
			name:    "多个JSON值",
			payload: `{"message":"你好"}{"message":"再次"}`,
			valid:   false,
		},
		{
			name:    "空正文",
			payload: ``,
			valid:   false,
		},
	}

	for _, item := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/",
			bytes.NewBufferString(
				item.payload,
			),
		)

		var decoded body
		err := decodeAssistantRuntimeJSON(
			recorder,
			request,
			&decoded,
			assistantRuntimeChatBodyMaxBytes,
		)

		if item.valid && err != nil {
			t.Fatalf(
				"%s应解析成功: %v",
				item.name,
				err,
			)
		}

		if !item.valid && err == nil {
			t.Fatalf(
				"%s应被拒绝",
				item.name,
			)
		}
	}
}

// TestAssistantRuntimeHTTPClientIP 验证只使用X-Real-IP或RemoteAddr。
func TestAssistantRuntimeHTTPClientIP(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		nil,
	)
	request.RemoteAddr = "10.0.0.8:43210"
	request.Header.Set(
		"X-Forwarded-For",
		"203.0.113.9",
	)

	ip, err := assistantRuntimeClientIP(
		request,
	)
	if err != nil ||
		ip != "10.0.0.8" {
		t.Fatalf(
			"RemoteAddr解析错误: ip=%s error=%v",
			ip,
			err,
		)
	}

	request.Header.Set(
		"X-Real-IP",
		"198.51.100.7",
	)

	ip, err = assistantRuntimeClientIP(
		request,
	)
	if err != nil ||
		ip != "198.51.100.7" {
		t.Fatalf(
			"X-Real-IP解析错误: ip=%s error=%v",
			ip,
			err,
		)
	}
}

// TestAssistantRuntimeHTTPErrorMapping 验证公开错误不泄露内部信息。
func TestAssistantRuntimeHTTPErrorMapping(
	t *testing.T,
) {
	cases := []struct {
		err    error
		status int
		text   string
	}{
		{
			err:    services.ErrAssistantRuntimeTokenInvalid,
			status: http.StatusUnauthorized,
			text:   "运行令牌无效或已过期",
		},
		{
			err:    repository.ErrAssistantRuntimeDailyQuotaExceeded,
			status: http.StatusTooManyRequests,
			text:   "今日调用额度已用尽",
		},
		{
			err:    repository.ErrAssistantRuntimeTurnInProgress,
			status: http.StatusConflict,
			text:   "已有正在处理的消息",
		},
		{
			err:    repository.ErrAssistantRuntimeBillingAccountUnavailable,
			status: http.StatusServiceUnavailable,
			text:   "暂时不可用",
		},
		{
			err:    errors.New("database password leaked"),
			status: http.StatusInternalServerError,
			text:   "服务异常",
		},
	}

	for _, item := range cases {
		status, message := assistantRuntimeHTTPStatusAndMessage(
			item.err,
		)

		if status != item.status ||
			!strings.Contains(
				message,
				item.text,
			) {
			t.Fatalf(
				"错误映射不正确: error=%v status=%d message=%s",
				item.err,
				status,
				message,
			)
		}

		if strings.Contains(
			message,
			"database password",
		) {
			t.Fatal(
				"公开错误泄露内部信息",
			)
		}
	}
}

// TestAssistantRuntimeHTTPSSEHeaders 验证不设置通配CORS。
func TestAssistantRuntimeHTTPSSEHeaders(
	t *testing.T,
) {
	recorder := httptest.NewRecorder()

	prepareAssistantRuntimeSSE(
		recorder,
	)

	if actual := recorder.Header().Get(
		"Content-Type",
	); !strings.Contains(
		actual,
		"text/event-stream",
	) {
		t.Fatalf(
			"SSE Content-Type错误: %s",
			actual,
		)
	}

	if actual := recorder.Header().Get(
		"Access-Control-Allow-Origin",
	); actual != "" {
		t.Fatalf(
			"不得设置通配或静态CORS: %s",
			actual,
		)
	}

	if actual := recorder.Header().Get(
		"X-Accel-Buffering",
	); actual != "no" {
		t.Fatalf(
			"Nginx缓冲头错误: %s",
			actual,
		)
	}
}

// TestAssistantRuntimeHTTPEmbedSafeFields 验证embed只包含公开展示信息。
func TestAssistantRuntimeHTTPEmbedSafeFields(
	t *testing.T,
) {
	recorder := httptest.NewRecorder()

	writeAssistantRuntimeEmbedHTML(
		recorder,
		&models.AssistantDeploymentPublicDescriptor{
			PublicID:            "public-safe-id",
			Title:               `<script>alert("x")</script>面积伙伴`,
			WelcomeMessage:      `欢迎 <b>同学</b>`,
			DisplayMode:         models.CoursewareAssistantDisplayModeFloating,
			DisplayPosition:     models.CoursewareAssistantPositionBottomRight,
			MaximumSessionTurns: 8,
		},
	)

	body := recorder.Body.String()

	if strings.Contains(
		body,
		"<script>alert",
	) ||
		strings.Contains(
			body,
			"<b>同学</b>",
		) {
		t.Fatalf(
			"embed未转义公开文字: %s",
			body,
		)
	}

	for _, sensitive := range []string{
		"owner_user_id",
		"school_id",
		"api_key",
		"assistant_prompt",
		"model_name",
	} {
		if strings.Contains(
			body,
			sensitive,
		) {
			t.Fatalf(
				"embed泄露敏感字段: %s",
				sensitive,
			)
		}
	}

	if !strings.Contains(
		body,
		"public-safe-id",
	) ||
		!strings.Contains(
			body,
			"当前会话最多 8 轮",
		) {
		t.Fatalf(
			"embed缺少公开信息: %s",
			body,
		)
	}
}
