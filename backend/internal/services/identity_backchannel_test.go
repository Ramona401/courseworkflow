package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	identityBackchannelTestIdempotencyKey = "33333333-3333-4333-8333-333333333333"

	identityBackchannelTestTraceID = "44444444-4444-4444-8444-444444444444"

	identityBackchannelTestEventID = "55555555-5555-4555-8555-555555555555"

	identityBackchannelTestLinkID = "66666666-6666-4666-8666-666666666666"
)

func identityBackchannelTestResult(
	operation string,
	state string,
	outcome string,
	reasonCode string,
) IdentityBackchannelResult {
	return IdentityBackchannelResult{
		SchemaVersion:         IdentityBackchannelSchemaVersion,
		TraceID:               identityBackchannelTestTraceID,
		EventID:               identityBackchannelTestEventID,
		PlatformAccountLinkID: identityBackchannelTestLinkID,
		GlobalPersonID:        identityClientTestGlobalPersonID,
		LocalAccountID:        identityFlowTestLocalUserID,
		Operation:             operation,
		Outcome:               outcome,
		ReasonCode:            reasonCode,
		State:                 state,
		IdempotentReplay:      false,
		Retryable:             false,
	}
}

func newIdentityBackchannelTestClient(
	t *testing.T,
	issuer string,
	httpClient *http.Client,
) *IdentityBackchannelClient {
	t.Helper()

	client, err :=
		NewIdentityBackchannelClient(
			newIdentityClientTestConfig(
				issuer,
			),
			httpClient,
		)
	if err != nil {
		t.Fatalf(
			"NewIdentityBackchannelClient() error = %v",
			err,
		)
	}

	client.now = func() time.Time {
		return time.Date(
			2026,
			time.August,
			11,
			6,
			45,
			0,
			0,
			time.UTC,
		)
	}

	// 定向测试不进行真实退避等待。
	client.sleep = func(
		_ context.Context,
		_ time.Duration,
	) error {
		return nil
	}

	return client
}

func readIdentityBackchannelTestRequest(
	r *http.Request,
	operation string,
) (
	identityBackchannelRequest,
	error,
) {
	if r.Method != http.MethodPost {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"method=%s",
				r.Method,
			)
	}

	if r.URL.Path !=
		"/backchannel/platform-account-links" {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"path=%s",
				r.URL.Path,
			)
	}

	username,
		password,
		ok :=
		r.BasicAuth()

	if !ok ||
		username !=
			identityClientTestClientID ||
		password !=
			identityClientTestClientSecret {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"Backchannel Basic认证异常",
			)
	}

	body, err :=
		io.ReadAll(r.Body)
	if err != nil {
		return identityBackchannelRequest{},
			err
	}

	if strings.Contains(
		string(body),
		identityClientTestClientSecret,
	) {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"Client Secret进入了Backchannel JSON正文",
			)
	}

	var payload identityBackchannelRequest

	if err := json.Unmarshal(
		body,
		&payload,
	); err != nil {
		return identityBackchannelRequest{},
			err
	}

	if payload.SchemaVersion !=
		IdentityBackchannelSchemaVersion {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"schema_version=%d",
				payload.SchemaVersion,
			)
	}

	if payload.Operation != operation {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"operation=%q",
				payload.Operation,
			)
	}

	if payload.GlobalPersonID !=
		identityClientTestGlobalPersonID {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"global_person_id=%q",
				payload.GlobalPersonID,
			)
	}

	if payload.LocalAccountID !=
		identityFlowTestLocalUserID {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"local_account_id=%q",
				payload.LocalAccountID,
			)
	}

	if payload.TraceID !=
		identityBackchannelTestTraceID {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"trace_id=%q",
				payload.TraceID,
			)
	}

	if payload.IdempotencyKey !=
		identityBackchannelTestIdempotencyKey {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"idempotency_key漂移",
			)
	}

	if len(payload.ReplayNonce) < 16 ||
		len(payload.ReplayNonce) > 128 ||
		strings.ContainsAny(
			payload.ReplayNonce,
			" \t\r\n",
		) {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"Replay Nonce格式异常",
			)
	}

	if _, err :=
		time.Parse(
			time.RFC3339Nano,
			payload.RequestTime,
		); err != nil {
		return identityBackchannelRequest{},
			fmt.Errorf(
				"request_time格式异常：%w",
				err,
			)
	}

	return payload, nil
}

// TestIdentityBackchannelRetry503PreservesMutationIdentity冻结503重试合同：
// 一个逻辑Mutation复用Idempotency Key，每次网络尝试都生成新的Replay Nonce。
func TestIdentityBackchannelRetry503PreservesMutationIdentity(
	t *testing.T,
) {
	var (
		mu       sync.Mutex
		payloads []identityBackchannelRequest
		calls    int
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				payload, err :=
					readIdentityBackchannelTestRequest(
						r,
						IdentityBackchannelOperationLink,
					)
				if err != nil {
					t.Errorf(
						"Backchannel请求合同异常：%v",
						err,
					)
					http.Error(
						w,
						"bad request",
						http.StatusBadRequest,
					)
					return
				}

				mu.Lock()
				payloads =
					append(
						payloads,
						payload,
					)
				calls++
				currentCall := calls
				mu.Unlock()

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				if currentCall == 1 {
					w.Header().Set(
						"Retry-After",
						"1",
					)
					w.WriteHeader(
						http.StatusServiceUnavailable,
					)

					_ = json.NewEncoder(w).Encode(
						map[string]interface{}{
							"schema_version": 1,
							"error": map[string]interface{}{
								"code":       "temporarily_unavailable",
								"message":    "retry",
								"retryable":  true,
								"request_id": "request-test-1",
							},
						},
					)
					return
				}

				_ = json.NewEncoder(w).Encode(
					identityBackchannelTestResult(
						IdentityBackchannelOperationLink,
						"linked",
						"success",
						"",
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
			"Mutate() error = %v",
			err,
		)
	}

	if result.Outcome != "success" ||
		result.State != "linked" {
		t.Fatalf(
			"result=%+v",
			result,
		)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(payloads) != 2 {
		t.Fatalf(
			"503请求次数=%d want=2",
			len(payloads),
		)
	}

	if payloads[0].IdempotencyKey !=
		payloads[1].IdempotencyKey {
		t.Fatal(
			"503重试改变了Idempotency Key",
		)
	}

	if payloads[0].ReplayNonce ==
		payloads[1].ReplayNonce {
		t.Fatal(
			"503重试没有生成新的Replay Nonce",
		)
	}
}

type identityBackchannelRetryTransport struct {
	mu       sync.Mutex
	calls    int
	payloads []identityBackchannelRequest
}

func (rt *identityBackchannelRetryTransport) RoundTrip(
	r *http.Request,
) (*http.Response, error) {
	payload, err :=
		readIdentityBackchannelTestRequest(
			r,
			IdentityBackchannelOperationUnlink,
		)
	if err != nil {
		return nil, err
	}

	rt.mu.Lock()
	rt.calls++
	rt.payloads =
		append(
			rt.payloads,
			payload,
		)
	currentCall := rt.calls
	rt.mu.Unlock()

	if currentCall == 1 {
		return nil,
			errors.New(
				"simulated transport failure",
			)
	}

	body, err :=
		json.Marshal(
			identityBackchannelTestResult(
				IdentityBackchannelOperationUnlink,
				"unlinked",
				"success",
				"",
			),
		)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{
				"application/json",
			},
		},
		Body: io.NopCloser(
			strings.NewReader(
				string(body),
			),
		),
		Request: r,
	}, nil
}

// TestIdentityBackchannelRetryTransportPreservesMutationIdentity确认真正transport
// error同样只能复用Idempotency Key并更换Replay Nonce后重试。
func TestIdentityBackchannelRetryTransportPreservesMutationIdentity(
	t *testing.T,
) {
	transport :=
		&identityBackchannelRetryTransport{}

	client :=
		newIdentityBackchannelTestClient(
			t,
			"https://identity.example",
			&http.Client{
				Transport: transport,
			},
		)

	result, err := client.Mutate(
		context.Background(),
		IdentityBackchannelOperationUnlink,
		identityClientTestGlobalPersonID,
		identityFlowTestLocalUserID,
		identityBackchannelTestTraceID,
		identityBackchannelTestIdempotencyKey,
	)
	if err != nil {
		t.Fatalf(
			"transport重试后失败：%v",
			err,
		)
	}

	if result.Outcome != "success" ||
		result.State != "unlinked" {
		t.Fatalf(
			"result=%+v",
			result,
		)
	}

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if transport.calls != 2 ||
		len(transport.payloads) != 2 {
		t.Fatalf(
			"transport calls=%d payloads=%d",
			transport.calls,
			len(transport.payloads),
		)
	}

	if transport.payloads[0].
		IdempotencyKey !=
		transport.payloads[1].
			IdempotencyKey {
		t.Fatal(
			"transport重试改变了Idempotency Key",
		)
	}

	if transport.payloads[0].
		ReplayNonce ==
		transport.payloads[1].
			ReplayNonce {
		t.Fatal(
			"transport重试没有生成新的Replay Nonce",
		)
	}
}
