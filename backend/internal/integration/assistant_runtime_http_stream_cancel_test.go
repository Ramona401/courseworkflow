package integration

// assistant_runtime_http_stream_cancel_test.go
//
// 使用完整生产路由和本机阻塞上游验证：
//   - 浏览器在收到connected和首个chunk后取消请求；
//   - 取消必须传播到AI上游HTTP Context；
//   - 后端使用脱离浏览器请求的短时Context完成失败结算；
//   - error_code必须为client_cancelled；
//   - 已经下发到浏览器的部分输出不得进入正式历史；
//   - 不扣积分、不留下active_turn。
//
// 本测试不调用真实AI。

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAssistantRuntimeHTTPStreamClientCancellation 验证浏览器取消。
func TestAssistantRuntimeHTTPStreamClientCancellation(
	t *testing.T,
) {
	upstreamStarted :=
		make(
			chan struct{},
			1,
		)

	upstreamCancelled :=
		make(
			chan struct{},
			1,
		)

	upstream :=
		httptest.NewServer(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					w.Header().Set(
						"Content-Type",
						"text/event-stream",
					)
					w.WriteHeader(
						http.StatusOK,
					)

					if err :=
						writeAssistantRuntimeOpenAISSEData(
							w,
							map[string]interface{}{
								"model":
									"qwen-max",
								"choices":
									[]map[string]interface{}{
										{
											"delta":
												map[string]string{
													"content":
														"取消前部分输出",
												},
										},
									},
							},
						); err != nil {
						return
					}

					select {
					case upstreamStarted <-
						struct{}{}:
					default:
					}

					<-r.Context().Done()

					select {
					case upstreamCancelled <-
						struct{}{}:
					default:
					}
				},
			),
		)
	t.Cleanup(
		upstream.Close,
	)

	fixture :=
		newAssistantRuntimeHTTPStreamFixture(
			t,
			upstream.URL,
		)

	requestContext,
		cancelRequest :=
		context.WithCancel(
			context.Background(),
		)

	response :=
		requestAssistantRuntimeChatStream(
			t,
			requestContext,
			fixture.Server.URL,
			fixture.Session.SessionID,
			fixture.Session.RuntimeToken,
			"我将在流式回复途中关闭页面。",
		)

	if response.StatusCode !=
		http.StatusOK ||
		!strings.HasPrefix(
			response.Header.Get(
				"Content-Type",
			),
			"text/event-stream",
		) {
		cancelRequest()
		_ = response.Body.Close()

		t.Fatalf(
			"取消测试没有建立SSE: HTTP=%d type=%s",
			response.StatusCode,
			response.Header.Get(
				"Content-Type",
			),
		)
	}

	reader :=
		bufio.NewReader(
			response.Body,
		)

	connected,
		err :=
		readAssistantRuntimeNextSSEEvent(
			reader,
		)
	if err != nil {
		cancelRequest()
		_ = response.Body.Close()

		t.Fatalf(
			"读取取消测试connected事件失败: %v",
			err,
		)
	}

	if connected.Event !=
		"connected" {
		cancelRequest()
		_ = response.Body.Close()

		t.Fatalf(
			"取消测试首事件不是connected: %+v",
			connected,
		)
	}

	chunk,
		err :=
		readAssistantRuntimeNextSSEEvent(
			reader,
		)
	if err != nil {
		cancelRequest()
		_ = response.Body.Close()

		t.Fatalf(
			"读取取消测试chunk事件失败: %v",
			err,
		)
	}

	if chunk.Event !=
		"chunk" ||
		assistantRuntimeSSEChunkText(
			t,
			chunk,
		) !=
			"取消前部分输出" {
		cancelRequest()
		_ = response.Body.Close()

		t.Fatalf(
			"取消测试首个chunk错误: %+v",
			chunk,
		)
	}

	select {
	case <-upstreamStarted:

	case <-time.After(
		2 * time.Second,
	):
		cancelRequest()
		_ = response.Body.Close()

		t.Fatal(
			"取消测试上游没有开始流式输出",
		)
	}

	cancelRequest()

	_ = response.Body.Close()

	select {
	case <-upstreamCancelled:

	case <-time.After(
		3 * time.Second,
	):
		t.Fatal(
			"浏览器取消没有传播到AI上游Context",
		)
	}

	waitAssistantRuntimeFailureSettlement(
		t,
		fixture.Session.SessionID,
		"client_cancelled",
	)

	viewResult :=
		requestAssistantRuntimeSessionView(
			t,
			fixture.Server.URL,
			fixture.Session.SessionID,
			fixture.Session.RuntimeToken,
		)

	if viewResult.Response.StatusCode !=
		http.StatusOK ||
		strings.Contains(
			string(viewResult.Body),
			"取消前部分输出",
		) {
		t.Fatalf(
			"浏览器取消后的部分输出进入正式历史: HTTP=%d body=%s",
			viewResult.Response.StatusCode,
			string(viewResult.Body),
		)
	}
}
