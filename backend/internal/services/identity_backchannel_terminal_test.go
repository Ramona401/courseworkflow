package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestIdentityBackchannelTerminalConflict409NoRetry确认业务409是稳定最终结果。
// 即使状态码不是200，只要协议结果合法，也必须直接交给上层业务处理，不能重复Mutation。
func TestIdentityBackchannelTerminalConflict409NoRetry(
	t *testing.T,
) {
	var calls atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				calls.Add(1)

				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.WriteHeader(
					http.StatusConflict,
				)

				_ = json.NewEncoder(w).Encode(
					identityBackchannelTestResult(
						IdentityBackchannelOperationLink,
						"conflict",
						"conflict",
						"local_account_already_linked",
					),
				)
			},
		),
	)
	defer server.Close()

	client :=
		newIdentityBackchannelTestClient(
			t,
			server.URL,
			server.Client(),
		)

	result, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationLink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err != nil {
		t.Fatalf(
			"409业务冲突应作为稳定结果返回：%v",
			err,
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"409业务冲突不得重试，calls=%d",
			calls.Load(),
		)
	}

	if result.SchemaVersion !=
		IdentityBackchannelSchemaVersion ||
		result.Operation !=
			IdentityBackchannelOperationLink ||
		result.Outcome != "conflict" ||
		result.State != "conflict" ||
		result.ReasonCode !=
			"local_account_already_linked" {
		t.Fatalf(
			"409稳定结果异常：%+v",
			result,
		)
	}
}

// TestIdentityBackchannelTerminalNon503ErrorNoRetry冻结自动重试白名单。
// 即使错误正文声称retryable=true，只要HTTP状态不是503，就不得自动再次发送Mutation。
func TestIdentityBackchannelTerminalNon503ErrorNoRetry(
	t *testing.T,
) {
	var calls atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				calls.Add(1)

				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.WriteHeader(
					http.StatusInternalServerError,
				)

				_ = json.NewEncoder(w).Encode(
					map[string]interface{}{
						"schema_version": IdentityBackchannelSchemaVersion,

						"error": map[string]interface{}{
							"code":       "internal_error",
							"message":    "terminal test error",
							"retryable":  true,
							"request_id": "request-terminal-500",
						},
					},
				)
			},
		),
	)
	defer server.Close()

	client :=
		newIdentityBackchannelTestClient(
			t,
			server.URL,
			server.Client(),
		)

	_, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationLink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err == nil {
		t.Fatal(
			"非503 HTTP错误必须返回失败",
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"非503错误不得自动重试，calls=%d",
			calls.Load(),
		)
	}
}

// TestIdentityBackchannelTerminalStopsAfterThree503冻结上游503最大自动尝试次数。
// 失败期间只允许有限重试，不能形成无界请求风暴。
func TestIdentityBackchannelTerminalStopsAfterThree503(
	t *testing.T,
) {
	var (
		calls      atomic.Int32
		sleepCalls atomic.Int32
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				calls.Add(1)

				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.Header().Set(
					"Retry-After",
					"1",
				)
				w.WriteHeader(
					http.StatusServiceUnavailable,
				)

				_ = json.NewEncoder(w).Encode(
					map[string]interface{}{
						"schema_version": IdentityBackchannelSchemaVersion,

						"error": map[string]interface{}{
							"code":       "temporarily_unavailable",
							"message":    "retry",
							"retryable":  true,
							"request_id": "request-terminal-503",
						},
					},
				)
			},
		),
	)
	defer server.Close()

	client :=
		newIdentityBackchannelTestClient(
			t,
			server.URL,
			server.Client(),
		)

	client.sleep = func(
		_ context.Context,
		delay time.Duration,
	) error {
		sleepCalls.Add(1)

		if delay <= 0 {
			t.Fatalf(
				"503重试delay无效：%v",
				delay,
			)
		}

		return nil
	}

	_, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationUnlink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err == nil {
		t.Fatal(
			"连续503达到尝试上限后必须返回失败",
		)
	}

	if calls.Load() != 3 {
		t.Fatalf(
			"503最大尝试次数=%d want=3",
			calls.Load(),
		)
	}

	if sleepCalls.Load() != 2 {
		t.Fatalf(
			"3次尝试之间应只退避2次，sleepCalls=%d",
			sleepCalls.Load(),
		)
	}
}

// TestIdentityBackchannelTerminalRejectsMismatchedSuccessResponse确认即使HTTP 200，
// Identity返回的可信结果也必须与本次请求的trace/global/local/operation绑定。
// 响应串线或错误关联必须fail-closed，而且200协议错误不应进行Mutation重试。
func TestIdentityBackchannelTerminalRejectsMismatchedSuccessResponse(
	t *testing.T,
) {
	var calls atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				calls.Add(1)

				result :=
					identityBackchannelTestResult(
						IdentityBackchannelOperationLink,
						"linked",
						"success",
						"",
					)

				// 故意模拟上游返回另一请求的trace结果。
				result.TraceID =
					"77777777-7777-4777-8777-777777777777"

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).
					Encode(result)
			},
		),
	)
	defer server.Close()

	client :=
		newIdentityBackchannelTestClient(
			t,
			server.URL,
			server.Client(),
		)

	_, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationLink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err == nil {
		t.Fatal(
			"与请求trace不一致的200成功响应必须被拒绝",
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"200协议绑定失败不得重试，calls=%d",
			calls.Load(),
		)
	}
}

// TestIdentityBackchannelTerminalRejectsWrongMappingResponse进一步冻结映射绑定：
// 即使trace正确，返回其它local_account_id也不能被当成本次Mutation结果接受。
func TestIdentityBackchannelTerminalRejectsWrongMappingResponse(
	t *testing.T,
) {
	var calls atomic.Int32

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				calls.Add(1)

				result :=
					identityBackchannelTestResult(
						IdentityBackchannelOperationUnlink,
						"unlinked",
						"success",
						"",
					)

				result.LocalAccountID =
					"88888888-8888-4888-8888-888888888888"

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).
					Encode(result)
			},
		),
	)
	defer server.Close()

	client :=
		newIdentityBackchannelTestClient(
			t,
			server.URL,
			server.Client(),
		)

	_, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationUnlink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err == nil {
		t.Fatal(
			"返回其它local_account_id的成功响应必须被拒绝",
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"响应映射绑定失败不得重试，calls=%d",
			calls.Load(),
		)
	}
}
