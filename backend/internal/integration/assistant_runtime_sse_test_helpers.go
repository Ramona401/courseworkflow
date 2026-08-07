package integration

// assistant_runtime_sse_test_helpers.go
//
// 为教学智能体真实HTTP/SSE测试提供：
//   - 本机OpenAI兼容上游配置；
//   - 完整公开运行会话夹具；
//   - 流式聊天请求；
//   - SSE事件逐条读取和完整读取；
//   - 上游OpenAI SSE数据块写入。
//
// 失败和成功持久化断言分别位于：
//   - assistant_runtime_sse_database_assertions.go
//   - assistant_runtime_http_stream_success_assertions.go
//
// 所有AI请求只指向httptest本机服务，不访问外部网络。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tedna/internal/ai"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// assistantRuntimeSSEEvent 是下游公开SSE事件。
type assistantRuntimeSSEEvent struct {
	Event string
	Data  []byte
}

// assistantRuntimeHTTPStreamFixture 保存真实HTTP流式测试夹具。
type assistantRuntimeHTTPStreamFixture struct {
	Server     *httptest.Server
	Fixture    *AssistantRuntimeFixture
	Deployment *models.AssistantDeployment
	Session    *models.AssistantRuntimeStartResponse
}

// newAssistantRuntimeHTTPStreamFixture 建立完整生产路由和公开会话。
func newAssistantRuntimeHTTPStreamFixture(
	t *testing.T,
	upstreamBaseURL string,
) *assistantRuntimeHTTPStreamFixture {
	t.Helper()

	server,
		_ :=
		SetupTestServer(
			t,
		)

	CleanAndSeed(t)

	fixture :=
		SeedAssistantRuntimeFixture(
			t,
		)

	deployment,
		_ :=
		fixture.CreateDeployment(
			t,
		)

	seedAssistantRuntimeLocalAIConfig(
		t,
		upstreamBaseURL,
	)

	session,
		_ :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"integration-sse-browser-client",
		)

	return &assistantRuntimeHTTPStreamFixture{
		Server:     server,
		Fixture:    fixture,
		Deployment: deployment,
		Session:    session,
	}
}

// seedAssistantRuntimeLocalAIConfig 把运行场景固定到本机qwen-max上游。
func seedAssistantRuntimeLocalAIConfig(
	t *testing.T,
	upstreamBaseURL string,
) {
	t.Helper()

	upstreamBaseURL =
		strings.TrimRight(
			strings.TrimSpace(
				upstreamBaseURL,
			),
			"/",
		)

	if upstreamBaseURL == "" {
		t.Fatal(
			"本地AI上游地址不能为空",
		)
	}

	err := repository.UpsertConfigValues(
		[]repository.ConfigValueUpdate{
			{
				Key:         "api_base_url",
				Value:       upstreamBaseURL,
				Description: "集成测试本机AI上游",
			},
			{
				Key:         "api_key_enc",
				Value:       "integration-local-ai-key",
				Description: "集成测试本机AI密钥",
			},
			{
				Key:         "default_model",
				Value:       "qwen-max",
				Description: "集成测试境内文本模型",
			},
			{
				Key:         "temperature",
				Value:       "0.2",
				Description: "集成测试温度",
			},
			{
				Key:         "max_tokens",
				Value:       "256",
				Description: "集成测试输出上限",
			},
		},
		SeedAdminID,
	)
	if err != nil {
		t.Fatalf(
			"写入本地AI全局配置失败: %v",
			err,
		)
	}

	result,
		err :=
		database.DB.Exec(
			context.Background(),
			`
			UPDATE ai_scene_configs
			SET
				model = 'qwen-max',
				temperature = 0.2,
				max_tokens = 256,
				is_active = TRUE,
				fallback_models = '[]'::jsonb,
				updated_by = $1,
				updated_at = NOW()
			WHERE scene_code = $2
			`,
			SeedAdminID,
			ai.SceneCoursewareAssistantRuntime,
		)
	if err != nil {
		t.Fatalf(
			"更新运行场景AI配置失败: %v",
			err,
		)
	}

	if result.RowsAffected() > 0 {
		return
	}

	_,
		err =
		database.DB.Exec(
			context.Background(),
			`
			INSERT INTO ai_scene_configs (
				id,
				scene_code,
				model,
				temperature,
				max_tokens,
				system_prompt_id,
				is_active,
				updated_by,
				updated_at,
				fallback_models
			)
			VALUES (
				gen_random_uuid(),
				$1,
				'qwen-max',
				0.2,
				256,
				NULL,
				TRUE,
				$2,
				NOW(),
				'[]'::jsonb
			)
			`,
			ai.SceneCoursewareAssistantRuntime,
			SeedAdminID,
		)
	if err != nil {
		t.Fatalf(
			"插入运行场景AI配置失败: %v",
			err,
		)
	}
}

// assistantRuntimeChatBody 编码公开聊天正文。
func assistantRuntimeChatBody(
	t *testing.T,
	message string,
) []byte {
	t.Helper()

	encoded,
		err :=
		json.Marshal(
			&models.AssistantRuntimeChatRequest{
				Message: message,
			},
		)
	if err != nil {
		t.Fatalf(
			"编码公开运行聊天正文失败: %v",
			err,
		)
	}

	return encoded
}

// requestAssistantRuntimeChatStream 发起真实流式聊天请求。
func requestAssistantRuntimeChatStream(
	t *testing.T,
	ctx context.Context,
	serverURL string,
	sessionID string,
	runtimeToken string,
	message string,
) *http.Response {
	t.Helper()

	if ctx == nil {
		ctx = context.Background()
	}

	request,
		err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			serverURL+
				"/api/v1/assistant-runtime/sessions/"+
				sessionID+
				"/chat",
			bytes.NewReader(
				assistantRuntimeChatBody(
					t,
					message,
				),
			),
		)
	if err != nil {
		t.Fatalf(
			"创建公开运行SSE请求失败: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)
	request.Header.Set(
		"Authorization",
		"Bearer "+runtimeToken,
	)

	response,
		err :=
		(&http.Client{}).Do(
			request,
		)
	if err != nil {
		t.Fatalf(
			"发送公开运行SSE请求失败: %v",
			err,
		)
	}

	return response
}

// readAssistantRuntimeNextSSEEvent 从流中读取一个完整SSE事件。
func readAssistantRuntimeNextSSEEvent(
	reader *bufio.Reader,
) (
	assistantRuntimeSSEEvent,
	error,
) {
	event :=
		assistantRuntimeSSEEvent{}

	dataLines :=
		[]string{}

	for {
		line,
			err :=
			reader.ReadString(
				'\n',
			)

		if err != nil &&
			!errors.Is(
				err,
				io.EOF,
			) {
			return event, err
		}

		line =
			strings.TrimSuffix(
				line,
				"\n",
			)
		line =
			strings.TrimSuffix(
				line,
				"\r",
			)

		switch {
		case strings.HasPrefix(
			line,
			"event:",
		):
			event.Event =
				strings.TrimSpace(
					strings.TrimPrefix(
						line,
						"event:",
					),
				)

		case strings.HasPrefix(
			line,
			"data:",
		):
			dataLines =
				append(
					dataLines,
					strings.TrimSpace(
						strings.TrimPrefix(
							line,
							"data:",
						),
					),
				)

		case line == "":
			if event.Event != "" ||
				len(dataLines) > 0 {
				event.Data =
					[]byte(
						strings.Join(
							dataLines,
							"\n",
						),
					)

				return event, nil
			}
		}

		if errors.Is(
			err,
			io.EOF,
		) {
			if event.Event != "" ||
				len(dataLines) > 0 {
				event.Data =
					[]byte(
						strings.Join(
							dataLines,
							"\n",
						),
					)

				return event, nil
			}

			return event, io.EOF
		}
	}
}

// readAssistantRuntimeAllSSEEvents 读取并关闭完整SSE响应。
func readAssistantRuntimeAllSSEEvents(
	t *testing.T,
	response *http.Response,
) []assistantRuntimeSSEEvent {
	t.Helper()

	if response == nil ||
		response.Body == nil {
		t.Fatal(
			"SSE响应或正文为空",
		)
	}

	reader :=
		bufio.NewReader(
			response.Body,
		)

	events :=
		[]assistantRuntimeSSEEvent{}

	for {
		event,
			err :=
			readAssistantRuntimeNextSSEEvent(
				reader,
			)

		if errors.Is(
			err,
			io.EOF,
		) {
			break
		}

		if err != nil {
			_ = response.Body.Close()

			t.Fatalf(
				"读取公开运行SSE事件失败: %v",
				err,
			)
		}

		events =
			append(
				events,
				event,
			)
	}

	if err := response.Body.Close(); err != nil {
		t.Fatalf(
			"关闭公开运行SSE响应失败: %v",
			err,
		)
	}

	return events
}

// assistantRuntimeSSEChunkText 解析下游chunk事件。
func assistantRuntimeSSEChunkText(
	t *testing.T,
	event assistantRuntimeSSEEvent,
) string {
	t.Helper()

	var payload struct {
		Chunk string `json:"chunk"`
	}

	if err := json.Unmarshal(
		event.Data,
		&payload,
	); err != nil {
		t.Fatalf(
			"解析公开运行chunk事件失败: %v data=%s",
			err,
			string(event.Data),
		)
	}

	return payload.Chunk
}

// assistantRuntimeSSEErrorText 解析下游error事件。
func assistantRuntimeSSEErrorText(
	t *testing.T,
	event assistantRuntimeSSEEvent,
) string {
	t.Helper()

	var payload struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(
		event.Data,
		&payload,
	); err != nil {
		t.Fatalf(
			"解析公开运行error事件失败: %v data=%s",
			err,
			string(event.Data),
		)
	}

	return payload.Error
}

// writeAssistantRuntimeOpenAISSEData 写入一个上游OpenAI数据块。
func writeAssistantRuntimeOpenAISSEData(
	w http.ResponseWriter,
	payload interface{},
) error {
	encoded,
		err :=
		json.Marshal(
			payload,
		)
	if err != nil {
		return err
	}

	if _,
		err :=
		fmt.Fprintf(
			w,
			"data: %s\n\n",
			encoded,
		); err != nil {
		return err
	}

	return http.NewResponseController(
		w,
	).Flush()
}

// writeAssistantRuntimeOpenAISSEDone 写入上游[DONE]终态。
func writeAssistantRuntimeOpenAISSEDone(
	w http.ResponseWriter,
) error {
	if _,
		err :=
		io.WriteString(
			w,
			"data: [DONE]\n\n",
		); err != nil {
		return err
	}

	return http.NewResponseController(
		w,
	).Flush()
}
