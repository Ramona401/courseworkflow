package integration

// assistant_runtime_http_stream_failure_test.go
//
// 使用完整生产路由和本机上游验证：
//   - 令牌错误发生在ready前，必须返回普通HTTP JSON；
//   - malformed上游SSE发生在ready后，必须返回下游SSE error；
//   - 上游发送部分chunk后EOF但没有终态，必须按失败结算；
//   - 任何已经下发的部分输出都不能进入正式会话历史；
//   - 失败不扣积分、不增加成功轮数。
//
// 本测试不调用真实AI。

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tedna/internal/database"
)

// TestAssistantRuntimeHTTPStreamPreReadyJSON 验证ready前仍是JSON。
func TestAssistantRuntimeHTTPStreamPreReadyJSON(
	t *testing.T,
) {
	var upstreamCalls atomic.Int32

	upstream :=
		httptest.NewServer(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					_ *http.Request,
				) {
					upstreamCalls.Add(1)

					w.WriteHeader(
						http.StatusInternalServerError,
					)
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

	result :=
		doAssistantRuntimeHTTPRequest(
			t,
			http.MethodPost,
			fixture.Server.URL+
				"/api/v1/assistant-runtime/sessions/"+
				fixture.Session.SessionID+
				"/chat",
			assistantRuntimeChatBody(
				t,
				"这是不会进入AI上游的消息。",
			),
			map[string]string{
				"Authorization":
					"Bearer " +
						tamperAssistantRuntimeToken(
							fixture.Session.RuntimeToken,
						),
			},
		)

	if result.Response.StatusCode !=
		http.StatusUnauthorized ||
		!strings.HasPrefix(
			result.Response.Header.Get(
				"Content-Type",
			),
			"application/json",
		) ||
		strings.Contains(
			string(result.Body),
			"event:",
		) {
		t.Fatalf(
			"ready前错误没有返回标准JSON: HTTP=%d type=%s body=%s",
			result.Response.StatusCode,
			result.Response.Header.Get(
				"Content-Type",
			),
			string(result.Body),
		)
	}

	if upstreamCalls.Load() != 0 {
		t.Fatalf(
			"无效运行令牌仍调用了AI上游: %d",
			upstreamCalls.Load(),
		)
	}

	assertAssistantRuntimeFailedPreReadyHasNoUsage(
		t,
		fixture.Session.SessionID,
	)
}

// TestAssistantRuntimeHTTPStreamProtocolFailures 验证ready后错误。
func TestAssistantRuntimeHTTPStreamProtocolFailures(
	t *testing.T,
) {
	cases := []struct {
		name      string
		malformed bool
	}{
		{
			name:      "malformed upstream SSE",
			malformed: true,
		},
		{
			name:      "upstream EOF before completion",
			malformed: false,
		},
	}

	for _,
		testCase :=
		range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				upstream :=
					httptest.NewServer(
						http.HandlerFunc(
							func(
								w http.ResponseWriter,
								_ *http.Request,
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
																	"部分输出",
															},
													},
												},
										},
									); err != nil {
									return
								}

								if !testCase.malformed {
									// 直接返回形成EOF，
									// 且没有[DONE]或finish_reason。
									return
								}

								_,
									_ =
									fmt.Fprint(
										w,
										"data: {not-valid-json}\n\n",
									)

								_ =
									http.NewResponseController(
										w,
									).Flush()
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

				ctx,
					cancel :=
					context.WithTimeout(
						context.Background(),
						15*time.Second,
					)
				defer cancel()

				response :=
					requestAssistantRuntimeChatStream(
						t,
						ctx,
						fixture.Server.URL,
						fixture.Session.SessionID,
						fixture.Session.RuntimeToken,
						"请根据我的观察继续提问。",
					)

				if response.StatusCode !=
					http.StatusOK ||
					!strings.HasPrefix(
						response.Header.Get(
							"Content-Type",
						),
						"text/event-stream",
					) {
					body,
						_ :=
						io.ReadAll(
							response.Body,
						)

					_ = response.Body.Close()

					t.Fatalf(
						"ready后协议错误未保持SSE响应: HTTP=%d type=%s body=%s",
						response.StatusCode,
						response.Header.Get(
							"Content-Type",
						),
						string(body),
					)
				}

				events :=
					readAssistantRuntimeAllSSEEvents(
						t,
						response,
					)

				if len(events) < 3 ||
					events[0].Event !=
						"connected" {
					t.Fatalf(
						"协议失败SSE事件不完整: %+v",
						events,
					)
				}

				chunkText :=
					strings.Builder{}

				errorCount := 0
				doneCount := 0
				publicError := ""

				for _,
					event :=
					range events {
					switch event.Event {
					case "chunk":
						chunkText.WriteString(
							assistantRuntimeSSEChunkText(
								t,
								event,
							),
						)

					case "error":
						errorCount++

						publicError =
							assistantRuntimeSSEErrorText(
								t,
								event,
							)

					case "done":
						doneCount++
					}
				}

				if chunkText.String() !=
					"部分输出" ||
					errorCount != 1 ||
					doneCount != 0 ||
					strings.TrimSpace(
						publicError,
					) == "" {
					t.Fatalf(
						"协议失败下游事件错误: chunks=%q errors=%d done=%d public=%q events=%+v",
						chunkText.String(),
						errorCount,
						doneCount,
						publicError,
						events,
					)
				}

				if strings.Contains(
					publicError,
					"解析AI流式",
				) ||
					strings.Contains(
						publicError,
						"not-valid-json",
					) {
					t.Fatalf(
						"公开SSE错误泄露内部协议细节: %q",
						publicError,
					)
				}

				waitAssistantRuntimeFailureSettlement(
					t,
					fixture.Session.SessionID,
					"ai_stream_failed",
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
						"部分输出",
					) {
					t.Fatalf(
						"协议失败的部分输出进入正式历史: HTTP=%d body=%s",
						viewResult.Response.StatusCode,
						string(viewResult.Body),
					)
				}
			},
		)
	}
}

// assertAssistantRuntimeFailedPreReadyHasNoUsage 验证ready前拒绝不领取轮次。
func assertAssistantRuntimeFailedPreReadyHasNoUsage(
	t *testing.T,
	sessionID string,
) {
	t.Helper()

	var (
		activeTurnCount int
		usageCount      int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_sessions
				WHERE id = $1
				  AND (
						active_turn_id IS NOT NULL
						OR active_turn_started_at IS NOT NULL
				  )
			),
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage
				WHERE runtime_session_id = $1
			)
		`,
		sessionID,
	).Scan(
		&activeTurnCount,
		&usageCount,
	); err != nil {
		t.Fatalf(
			"读取ready前拒绝结果失败: %v",
			err,
		)
	}

	if activeTurnCount != 0 ||
		usageCount != 0 {
		t.Fatalf(
			"ready前拒绝错误领取或结算: active=%d usage=%d",
			activeTurnCount,
			usageCount,
		)
	}
}
