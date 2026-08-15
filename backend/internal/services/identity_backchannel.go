package services

// identity_backchannel.go — TE-DNA调用Identity Center平台账号Link/Unlink Backchannel。
//
// v1协议合同：
//   - POST /backchannel/platform-account-links；
//   - HTTP Basic固定使用tedna-client + Client Secret；
//   - JSON schema_version=1；
//   - 一个逻辑Mutation固定一个Idempotency Key；
//   - 每一次网络重试必须生成全新的Replay Nonce和request_time；
//   - 只对transport error与HTTP 503自动做有限重试；
//   - HTTP 409 conflict是稳定业务结果，不自动重复Mutation；
//   - 不记录Client Secret、Replay Nonce、Idempotency Key或请求Body。
//
// Phase 1边界：本文件只修改Identity Center中的跨平台mapping事实，
// 不签发TE-DNA JWT，不建立本地Session，也不向TE-DNA数据库复制global_person_id。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"tedna/internal/config"
)

const (
	IdentityBackchannelSchemaVersion = 1

	IdentityBackchannelOperationLink   = "link"
	IdentityBackchannelOperationUnlink = "unlink"

	identityBackchannelMaxAttempts     = 3
	identityBackchannelResponseMaxSize = 16 * 1024
	identityBackchannelDefaultRetry    = time.Second
	identityBackchannelMaxRetryAfter   = 5 * time.Second
)

// IdentityBackchannelClient 调用受Client Secret保护的Server-to-Server接口。
type IdentityBackchannelClient struct {
	cfg        config.IdentityClientConfig
	httpClient *http.Client
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error
}

// IdentityBackchannelResult 对应Identity Center稳定成功或冲突结果。
type IdentityBackchannelResult struct {
	SchemaVersion         int    `json:"schema_version"`
	TraceID               string `json:"trace_id"`
	EventID               string `json:"event_id"`
	PlatformAccountLinkID string `json:"platform_account_link_id,omitempty"`
	GlobalPersonID        string `json:"global_person_id"`
	LocalAccountID        string `json:"local_account_id"`
	Operation             string `json:"operation"`
	Outcome               string `json:"outcome"`
	ReasonCode            string `json:"reason_code,omitempty"`
	State                 string `json:"state"`
	IdempotentReplay      bool   `json:"idempotent_replay"`
	Retryable             bool   `json:"retryable"`
}

// identityBackchannelRequest 是Identity Center v1 Mutation请求。
//
// platform_client_id绝不进入Body；平台身份只能来自HTTP Basic认证出的可信Client。
type identityBackchannelRequest struct {
	RequestTime    string `json:"request_time"`
	ReplayNonce    string `json:"replay_nonce"`
	IdempotencyKey string `json:"idempotency_key"`
	SchemaVersion  int    `json:"schema_version"`
	Operation      string `json:"operation"`
	GlobalPersonID string `json:"global_person_id"`
	LocalAccountID string `json:"local_account_id"`
	TraceID        string `json:"trace_id"`
}

type identityBackchannelErrorEnvelope struct {
	SchemaVersion int `json:"schema_version"`
	Error         struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

// IdentityBackchannelError 是非业务conflict的稳定协议错误。
//
// Error()刻意不返回上游message，避免未来调用方把服务端详细错误直接透传给浏览器。
// Code、Retryable、RequestID可供后续业务层做安全映射和诊断。
type IdentityBackchannelError struct {
	HTTPStatus int
	Code       string
	Message    string
	Retryable  bool
	RequestID  string
}

func (e *IdentityBackchannelError) Error() string {
	if e == nil {
		return ""
	}

	if e.Code == "" {
		return fmt.Sprintf(
			"Identity Backchannel返回HTTP %d",
			e.HTTPStatus,
		)
	}

	return fmt.Sprintf(
		"Identity Backchannel错误[%s]",
		e.Code,
	)
}

// NewIdentityBackchannelClient 创建TE-DNA Backchannel Client。
func NewIdentityBackchannelClient(
	cfg config.IdentityClientConfig,
	httpClient *http.Client,
) (*IdentityBackchannelClient, error) {
	if cfg.Issuer == "" ||
		cfg.Issuer != strings.TrimSpace(cfg.Issuer) ||
		cfg.ClientSecret == "" ||
		cfg.ClientSecret != strings.TrimSpace(cfg.ClientSecret) {
		return nil, fmt.Errorf(
			"Identity Backchannel配置不完整",
		)
	}

	// Phase 1 Backchannel平台身份固定，禁止生产配置静默跨Client执行Mutation。
	if cfg.ClientID != config.DefaultIdentityClientID {
		return nil, fmt.Errorf(
			"TE-DNA Backchannel client_id必须为%s",
			config.DefaultIdentityClientID,
		)
	}

	if httpClient == nil {
		httpClient = newIdentitySecureHTTPClient()
	}

	return &IdentityBackchannelClient{
		cfg:        cfg,
		httpClient: httpClient,
		now:        time.Now,
		sleep:      identityBackchannelContextSleep,
	}, nil
}

// Mutate 执行一次Link或Unlink逻辑Mutation。
//
// idempotencyKey为空时由TE-DNA后端生成。
// 若更高业务层需要安全重试同一逻辑Mutation，应显式复用原Key。
func (c *IdentityBackchannelClient) Mutate(
	ctx context.Context,
	operation string,
	globalPersonID string,
	localAccountID string,
	traceID string,
	idempotencyKey string,
) (IdentityBackchannelResult, error) {
	if ctx == nil {
		return IdentityBackchannelResult{},
			fmt.Errorf("Identity Backchannel Context为空")
	}

	if c == nil ||
		c.httpClient == nil ||
		c.now == nil ||
		c.sleep == nil {
		return IdentityBackchannelResult{},
			fmt.Errorf("Identity Backchannel Client尚未初始化")
	}

	if operation != IdentityBackchannelOperationLink &&
		operation != IdentityBackchannelOperationUnlink {
		return IdentityBackchannelResult{},
			fmt.Errorf("Identity Backchannel operation无效")
	}

	canonicalGlobalID, err :=
		canonicalIdentityGlobalPersonID(globalPersonID)
	if err != nil {
		return IdentityBackchannelResult{}, err
	}

	canonicalLocalID, err :=
		canonicalIdentityLocalUserID(localAccountID)
	if err != nil {
		return IdentityBackchannelResult{},
			fmt.Errorf(
				"TE-DNA local_account_id无效：%w",
				err,
			)
	}

	if traceID == "" {
		traceID = uuid.NewString()
	}

	if err := validateIdentityBackchannelTraceID(traceID); err != nil {
		return IdentityBackchannelResult{}, err
	}

	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}

	if err := validateIdentityBackchannelOpaqueValue(
		"Idempotency Key",
		idempotencyKey,
		8,
		128,
	); err != nil {
		return IdentityBackchannelResult{}, err
	}

	var lastErr error

	for attempt := 1; attempt <= identityBackchannelMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return IdentityBackchannelResult{}, err
		}

		// 每次尝试必须换Nonce。
		// 24字节随机值编码后仍落在服务端16—128字节协议范围内。
		replayNonce, err :=
			newIdentityRandomBase64URL(24)
		if err != nil {
			return IdentityBackchannelResult{}, err
		}

		payload := identityBackchannelRequest{
			RequestTime: c.now().
				UTC().
				Format(time.RFC3339Nano),
			ReplayNonce:    replayNonce,
			IdempotencyKey: idempotencyKey,
			SchemaVersion:  IdentityBackchannelSchemaVersion,
			Operation:      operation,
			GlobalPersonID: canonicalGlobalID,
			LocalAccountID: canonicalLocalID,
			TraceID:        traceID,
		}

		result, status, retryAfter, transportFailure, err :=
			c.doIdentityBackchannelAttempt(
				ctx,
				payload,
			)

		if err == nil {
			return result, nil
		}

		lastErr = err

		if attempt == identityBackchannelMaxAttempts {
			break
		}

		// 只允许：
		//   1. 真正的HTTP transport失败；
		//   2. Identity明确返回503。
		//
		// 400/401/409/其它5xx都不能在这里自动重复业务Mutation。
		if !transportFailure &&
			status != http.StatusServiceUnavailable {
			break
		}

		delay := retryAfter
		if delay <= 0 {
			delay = identityBackchannelDefaultRetry
		}

		if delay > identityBackchannelMaxRetryAfter {
			delay = identityBackchannelMaxRetryAfter
		}

		if err := c.sleep(ctx, delay); err != nil {
			return IdentityBackchannelResult{}, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf(
			"Identity Backchannel请求失败",
		)
	}

	return IdentityBackchannelResult{}, lastErr
}

func (c *IdentityBackchannelClient) doIdentityBackchannelAttempt(
	ctx context.Context,
	payload identityBackchannelRequest,
) (
	IdentityBackchannelResult,
	int,
	time.Duration,
	bool,
	error,
) {
	body, err := json.Marshal(payload)
	if err != nil {
		return IdentityBackchannelResult{},
			0,
			0,
			false,
			fmt.Errorf(
				"编码Identity Backchannel请求失败：%w",
				err,
			)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.BackchannelEndpoint(),
		bytes.NewReader(body),
	)
	if err != nil {
		return IdentityBackchannelResult{},
			0,
			0,
			false,
			fmt.Errorf(
				"创建Identity Backchannel请求失败：%w",
				err,
			)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)
	request.Header.Set(
		"Accept",
		"application/json",
	)
	request.Header.Set(
		"Cache-Control",
		"no-store",
	)

	// Backchannel协议固定HTTP Basic，与OIDC Token Endpoint的
	// client_secret_post/basic配置无关。
	request.SetBasicAuth(
		c.cfg.ClientID,
		c.cfg.ClientSecret,
	)

	response, err := c.httpClient.Do(request)
	if err != nil {
		// 只有真正到达HTTP Client transport层的错误才允许作为transport retry。
		return IdentityBackchannelResult{},
			0,
			0,
			true,
			fmt.Errorf(
				"Identity Backchannel网络请求失败：%w",
				err,
			)
	}
	defer response.Body.Close()

	retryAfter :=
		parseIdentityBackchannelRetryAfter(response)

	responseBody, err :=
		readIdentityBoundedResponseBody(
			response.Body,
			identityBackchannelResponseMaxSize,
		)
	if err != nil {
		return IdentityBackchannelResult{},
			response.StatusCode,
			retryAfter,
			false,
			err
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusConflict:
		var result IdentityBackchannelResult

		if err := json.Unmarshal(
			responseBody,
			&result,
		); err == nil {
			if err := validateIdentityBackchannelResult(
				result,
				response.StatusCode,
				payload,
			); err == nil {
				return result,
					response.StatusCode,
					0,
					false,
					nil
			}
		}
	}

	var errorEnvelope identityBackchannelErrorEnvelope

	if err := json.Unmarshal(
		responseBody,
		&errorEnvelope,
	); err == nil &&
		errorEnvelope.Error.Code != "" {

		if errorEnvelope.SchemaVersion !=
			IdentityBackchannelSchemaVersion {
			return IdentityBackchannelResult{},
				response.StatusCode,
				retryAfter,
				false,
				&IdentityBackchannelError{
					HTTPStatus: response.StatusCode,
				}
		}

		return IdentityBackchannelResult{},
			response.StatusCode,
			retryAfter,
			false,
			&IdentityBackchannelError{
				HTTPStatus: response.StatusCode,
				Code:       errorEnvelope.Error.Code,
				Message:    errorEnvelope.Error.Message,
				Retryable:  errorEnvelope.Error.Retryable,
				RequestID:  errorEnvelope.Error.RequestID,
			}
	}

	return IdentityBackchannelResult{},
		response.StatusCode,
		retryAfter,
		false,
		&IdentityBackchannelError{
			HTTPStatus: response.StatusCode,
		}
}

// validateIdentityBackchannelResult 防止把格式正确但语义不属于本次请求的响应当成功。
func validateIdentityBackchannelResult(
	result IdentityBackchannelResult,
	httpStatus int,
	request identityBackchannelRequest,
) error {
	if result.SchemaVersion !=
		IdentityBackchannelSchemaVersion {
		return fmt.Errorf(
			"Identity Backchannel响应schema_version无效",
		)
	}

	if result.TraceID != request.TraceID {
		return fmt.Errorf(
			"Identity Backchannel响应trace_id不匹配",
		)
	}

	if result.Operation != request.Operation {
		return fmt.Errorf(
			"Identity Backchannel响应operation不匹配",
		)
	}

	globalID, err :=
		canonicalIdentityGlobalPersonID(
			result.GlobalPersonID,
		)
	if err != nil ||
		globalID != request.GlobalPersonID {
		return fmt.Errorf(
			"Identity Backchannel响应global_person_id不匹配",
		)
	}

	localID, err :=
		canonicalIdentityLocalUserID(
			result.LocalAccountID,
		)
	if err != nil ||
		localID != request.LocalAccountID {
		return fmt.Errorf(
			"Identity Backchannel响应local_account_id不匹配",
		)
	}

	if _, err := uuid.Parse(result.EventID); err != nil {
		return fmt.Errorf(
			"Identity Backchannel响应event_id无效",
		)
	}

	if result.PlatformAccountLinkID != "" {
		if _, err := uuid.Parse(
			result.PlatformAccountLinkID,
		); err != nil {
			return fmt.Errorf(
				"Identity Backchannel响应platform_account_link_id无效",
			)
		}
	}

	if result.Retryable {
		return fmt.Errorf(
			"Identity稳定Mutation结果不得标记retryable",
		)
	}

	switch httpStatus {
	case http.StatusOK:
		if result.Outcome != "success" ||
			result.ReasonCode != "" {
			return fmt.Errorf(
				"Identity Backchannel成功结果语义无效",
			)
		}

		expectedState := "linked"
		if request.Operation ==
			IdentityBackchannelOperationUnlink {
			expectedState = "unlinked"
		}

		if result.State != expectedState {
			return fmt.Errorf(
				"Identity Backchannel成功结果state无效",
			)
		}

		if result.PlatformAccountLinkID == "" {
			return fmt.Errorf(
				"Identity Backchannel成功结果缺少platform_account_link_id",
			)
		}

	case http.StatusConflict:
		if result.Outcome != "conflict" ||
			result.State != "conflict" ||
			strings.TrimSpace(result.ReasonCode) == "" {
			return fmt.Errorf(
				"Identity Backchannel conflict结果语义无效",
			)
		}

	default:
		return fmt.Errorf(
			"Identity Backchannel稳定结果HTTP状态无效",
		)
	}

	return nil
}

func validateIdentityBackchannelTraceID(
	value string,
) error {
	if len(value) == 0 ||
		len(value) > 128 {
		return fmt.Errorf(
			"Identity Backchannel trace_id长度必须为1到128字节",
		)
	}

	for index := 0; index < len(value); index++ {
		character := value[index]

		valid :=
			(character >= 'a' && character <= 'z') ||
				(character >= 'A' && character <= 'Z') ||
				(character >= '0' && character <= '9') ||
				character == '-' ||
				character == '_' ||
				character == '.' ||
				character == ':'

		if !valid {
			return fmt.Errorf(
				"Identity Backchannel trace_id包含非法字符",
			)
		}
	}

	return nil
}

func validateIdentityBackchannelOpaqueValue(
	name string,
	value string,
	minimum int,
	maximum int,
) error {
	if len(value) < minimum ||
		len(value) > maximum {
		return fmt.Errorf(
			"Identity Backchannel %s长度必须为%d到%d字节",
			name,
			minimum,
			maximum,
		)
	}

	for index := 0; index < len(value); index++ {
		character := value[index]

		if character < 0x21 ||
			character > 0x7E {
			return fmt.Errorf(
				"Identity Backchannel %s必须只包含不含空格的可见ASCII字符",
				name,
			)
		}
	}

	return nil
}

func parseIdentityBackchannelRetryAfter(
	response *http.Response,
) time.Duration {
	if response == nil {
		return 0
	}

	raw := strings.TrimSpace(
		response.Header.Get("Retry-After"),
	)
	if raw == "" {
		return 0
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil ||
		seconds <= 0 {
		return 0
	}

	delay :=
		time.Duration(seconds) *
			time.Second

	if delay > identityBackchannelMaxRetryAfter {
		return identityBackchannelMaxRetryAfter
	}

	return delay
}

func identityBackchannelContextSleep(
	ctx context.Context,
	duration time.Duration,
) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}
