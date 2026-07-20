package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteSSEServiceUnavailable(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeSSEServiceUnavailable(recorder)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"SSE排空响应状态应为503，实际=%d",
			recorder.Code,
		)
	}

	if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "30" {
		t.Fatalf(
			"SSE排空响应Retry-After错误，实际=%q",
			retryAfter,
		)
	}

	if !strings.Contains(recorder.Body.String(), sseServiceDrainingMessage) {
		t.Fatalf(
			"SSE排空响应未包含标准提示：%s",
			recorder.Body.String(),
		)
	}
}
