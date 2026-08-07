package integration

// assistant_runtime_http_stream_success_test.go
//
// 使用完整生产路由和本机OpenAI兼容上游验证：
//   - connected事件必须最先到达；
//   - 学生可见增量使用chunk事件；
//   - thinking标签跨上游chunk拆分也不能下发；
//   - [DONE]和finish_reason两种终态都能生成done；
//   - 公开会话历史只包含过滤后的学生可见正文。
//
// 成功持久化和积分断言位于
// assistant_runtime_http_stream_success_assertions.go。
//
// 本测试不调用真实AI。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
)

// assistantRuntimeCapturedUpstreamRequest 保存本机上游收到的请求。
type assistantRuntimeCapturedUpstreamRequest struct {
	Authorization string
	Body          []byte
}

// TestAssistantRuntimeHTTPStreamSuccess 验证完整成功SSE。
func TestAssistantRuntimeHTTPStreamSuccess(
	t *testing.T,
) {
	cases := []struct {
		name          string
		useDoneMarker bool
	}{
		{
			name:          "done marker",
			useDoneMarker: true,
		},
		{
			name:          "finish reason",
			useDoneMarker: false,
		},
	}

	for _,
		testCase :=
		range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				captured :=
					make(
						chan assistantRuntimeCapturedUpstreamRequest,
						1,
					)

				upstream :=
					httptest.NewServer(
						http.HandlerFunc(
							func(
								w http.ResponseWriter,
								r *http.Request,
							) {
								body,
									_ :=
									io.ReadAll(
										r.Body,
									)

								select {
								case captured <-
									assistantRuntimeCapturedUpstreamRequest{
										Authorization:
											r.Header.Get(
												"Authorization",
											),
										Body:
											body,
									}:
								default:
								}

								w.Header().Set(
									"Content-Type",
									"text/event-stream",
								)
								w.WriteHeader(
									http.StatusOK,
								)

								for _,
									content :=
									range []string{
										"先观",
										"察<thin",
										"king>隐藏",
										"</thinking>关系",
									} {
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
																		content,
																},
														},
													},
											},
										); err != nil {
										return
									}
								}

								if testCase.useDoneMarker {
									writeAssistantRuntimeSuccessfulFinalChunk(
										w,
										nil,
									)

									_ =
										writeAssistantRuntimeOpenAISSEDone(
											w,
										)

									return
								}

								finishReason :=
									"stop"

								writeAssistantRuntimeSuccessfulFinalChunk(
									w,
									&finishReason,
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
						"我发现两个三角形可以拼起来。",
					)

				assertAssistantRuntimeSuccessfulSSEHeaders(
					t,
					response,
				)

				events :=
					readAssistantRuntimeAllSSEEvents(
						t,
						response,
					)

				const expectedVisible =
					"先观察关系。"

				done :=
					assertAssistantRuntimeSuccessfulSSEEvents(
						t,
						events,
						expectedVisible,
					)

				assertAssistantRuntimeCapturedUpstreamRequest(
					t,
					captured,
				)

				assertAssistantRuntimeSuccessfulStreamStored(
					t,
					fixture,
					done.TurnID,
					expectedVisible,
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
					!strings.Contains(
						string(viewResult.Body),
						expectedVisible,
					) ||
					strings.Contains(
						string(viewResult.Body),
						"隐藏",
					) ||
					strings.Contains(
						string(viewResult.Body),
						"<thinking>",
					) {
					t.Fatalf(
						"公开会话历史错误: HTTP=%d body=%s",
						viewResult.Response.StatusCode,
						string(viewResult.Body),
					)
				}
			},
		)
	}
}

// writeAssistantRuntimeSuccessfulFinalChunk 写入成功终块。
func writeAssistantRuntimeSuccessfulFinalChunk(
	w http.ResponseWriter,
	finishReason *string,
) {
	choice :=
		map[string]interface{}{
			"delta":
				map[string]string{
					"content": "。",
				},
	}

	if finishReason != nil {
		choice["finish_reason"] =
			*finishReason
	}

	_ =
		writeAssistantRuntimeOpenAISSEData(
			w,
			map[string]interface{}{
				"model":
					"qwen-max",
				"choices":
					[]map[string]interface{}{
						choice,
					},
				"usage":
					map[string]int{
						"prompt_tokens":
							20,
						"completion_tokens":
							5,
						"total_tokens":
							25,
					},
			},
		)
}

// assertAssistantRuntimeSuccessfulSSEHeaders 验证成功响应头。
func assertAssistantRuntimeSuccessfulSSEHeaders(
	t *testing.T,
	response *http.Response,
) {
	t.Helper()

	if response.StatusCode !=
		http.StatusOK {
		body,
			_ :=
			io.ReadAll(
				response.Body,
			)

		_ = response.Body.Close()

		t.Fatalf(
			"成功SSE返回非200: HTTP=%d body=%s",
			response.StatusCode,
			string(body),
		)
	}

	if !strings.HasPrefix(
		response.Header.Get(
			"Content-Type",
		),
		"text/event-stream",
	) {
		t.Fatalf(
			"成功聊天未返回SSE Content-Type: %s",
			response.Header.Get(
				"Content-Type",
			),
		)
	}
}

// assertAssistantRuntimeSuccessfulSSEEvents 验证成功事件序列。
func assertAssistantRuntimeSuccessfulSSEEvents(
	t *testing.T,
	events []assistantRuntimeSSEEvent,
	expectedVisible string,
) *models.AssistantRuntimeChatResponse {
	t.Helper()

	if len(events) < 3 {
		t.Fatalf(
			"成功SSE事件不足: %+v",
			events,
		)
	}

	if events[0].Event !=
		"connected" {
		t.Fatalf(
			"首个SSE事件不是connected: %+v",
			events[0],
		)
	}

	if events[len(events)-1].Event !=
		"done" {
		t.Fatalf(
			"最后SSE事件不是done: %+v",
			events[len(events)-1],
		)
	}

	visible :=
		strings.Builder{}

	errorCount := 0

	for _,
		event :=
		range events {
		switch event.Event {
		case "chunk":
			visible.WriteString(
				assistantRuntimeSSEChunkText(
					t,
					event,
				),
			)

		case "error":
			errorCount++
		}
	}

	if errorCount != 0 {
		t.Fatalf(
			"成功SSE错误出现error事件: %d",
			errorCount,
		)
	}

	if visible.String() !=
		expectedVisible {
		t.Fatalf(
			"学生可见增量错误: expected=%q actual=%q",
			expectedVisible,
			visible.String(),
		)
	}

	if strings.Contains(
		visible.String(),
		"隐藏",
	) ||
		strings.Contains(
			visible.String(),
			"thinking",
		) {
		t.Fatalf(
			"thinking内容泄露到下游chunk: %q",
			visible.String(),
		)
	}

	done :=
		&models.AssistantRuntimeChatResponse{}

	if err := json.Unmarshal(
		events[len(events)-1].Data,
		done,
	); err != nil {
		t.Fatalf(
			"解析done事件失败: %v data=%s",
			err,
			string(
				events[len(events)-1].Data,
			),
		)
	}

	if done.TurnID == "" ||
		done.Message.Content !=
			expectedVisible ||
		done.TurnCount != 1 ||
		done.RemainingTurns != 4 ||
		done.SessionStatus !=
			models.AssistantRuntimeSessionStatusActive {
		t.Fatalf(
			"done事件内容错误: %+v",
			done,
		)
	}

	return done
}

// assertAssistantRuntimeCapturedUpstreamRequest 验证本机上游请求。
func assertAssistantRuntimeCapturedUpstreamRequest(
	t *testing.T,
	captured <-chan assistantRuntimeCapturedUpstreamRequest,
) {
	t.Helper()

	select {
	case request :=
		<-captured:
		if request.Authorization !=
			"Bearer integration-local-ai-key" {
			t.Fatalf(
				"本机上游Authorization错误: %s",
				request.Authorization,
			)
		}

		var upstreamRequest struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}

		if err := json.Unmarshal(
			request.Body,
			&upstreamRequest,
		); err != nil {
			t.Fatalf(
				"解析本机上游请求失败: %v body=%s",
				err,
				string(request.Body),
			)
		}

		if upstreamRequest.Model !=
			"qwen-max" ||
			!upstreamRequest.Stream ||
			len(upstreamRequest.Messages) <
				2 {
			t.Fatalf(
				"本机上游请求协议错误: %+v",
				upstreamRequest,
			)
		}

	case <-time.After(
		2 * time.Second,
	):
		t.Fatal(
			"本机上游没有收到AI请求",
		)
	}
}
