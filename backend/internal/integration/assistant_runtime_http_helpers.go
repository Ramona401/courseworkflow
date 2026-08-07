package integration

// assistant_runtime_http_helpers.go
//
// 提供教学智能体公开运行HTTP集成测试专用辅助：
//   - 发送带Origin、X-Real-IP和Bearer令牌的真实HTTP请求；
//   - 同时保留原始响应正文和统一JSON响应；
//   - 创建公开运行会话；
//   - 读取公开会话视图；
//   - 在不验证签名的前提下读取测试令牌声明，用于核对数据库JTI哈希；
//   - 检查公开响应和令牌载荷没有敏感字段。
//
// 本文件只用于测试，不生成或验证生产令牌。

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"tedna/internal/models"
)

// assistantRuntimeHTTPResult 保存完整HTTP测试结果。
type assistantRuntimeHTTPResult struct {
	Response *http.Response
	Body     []byte
	API      *APIResponse
}

// assistantRuntimeTestTokenClaims 是测试读取的运行令牌声明。
type assistantRuntimeTestTokenClaims struct {
	SessionID         string `json:"session_id"`
	DeploymentID      string `json:"deployment_id"`
	DeploymentVersion int    `json:"deployment_version"`
	SessionKind       string `json:"session_kind"`
	TokenUse          string `json:"token_use"`

	jwt.RegisteredClaims
}

// doAssistantRuntimeHTTPRequest 发送真实HTTP请求并保留原始响应正文。
func doAssistantRuntimeHTTPRequest(
	t *testing.T,
	method string,
	requestURL string,
	body []byte,
	headers map[string]string,
) *assistantRuntimeHTTPResult {
	t.Helper()

	request, err := http.NewRequest(
		method,
		requestURL,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf(
			"创建教学智能体HTTP请求失败: %v",
			err,
		)
	}

	if len(body) > 0 {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	for key, value := range headers {
		request.Header.Set(
			key,
			value,
		)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	response, err := client.Do(
		request,
	)
	if err != nil {
		t.Fatalf(
			"发送教学智能体HTTP请求失败: %v",
			err,
		)
	}

	responseBody, readErr := io.ReadAll(
		response.Body,
	)

	closeErr := response.Body.Close()

	if readErr != nil {
		t.Fatalf(
			"读取教学智能体HTTP响应失败: %v",
			readErr,
		)
	}

	if closeErr != nil {
		t.Fatalf(
			"关闭教学智能体HTTP响应失败: %v",
			closeErr,
		)
	}

	apiResponse := &APIResponse{}

	if len(responseBody) > 0 {
		if err := json.Unmarshal(
			responseBody,
			apiResponse,
		); err != nil {
			apiResponse.Code = -999
			apiResponse.Message = string(
				responseBody,
			)
		}
	}

	return &assistantRuntimeHTTPResult{
		Response: response,
		Body:     responseBody,
		API:      apiResponse,
	}
}

// requestAssistantRuntimeSession 请求创建公开运行会话，不预设成功状态。
func requestAssistantRuntimeSession(
	t *testing.T,
	serverURL string,
	publicID string,
	origin string,
	anonymousClientID string,
) *assistantRuntimeHTTPResult {
	t.Helper()

	encoded, err := json.Marshal(
		&models.AssistantRuntimeStartRequest{
			AnonymousClientID:
				anonymousClientID,
		},
	)
	if err != nil {
		t.Fatalf(
			"编码教学智能体会话请求失败: %v",
			err,
		)
	}

	headers := map[string]string{
		"Origin":    origin,
		"X-Real-IP": "203.0.113.10",
	}

	return doAssistantRuntimeHTTPRequest(
		t,
		http.MethodPost,
		serverURL+
			"/api/v1/assistant-runtime/deployments/"+
			publicID+
			"/session",
		encoded,
		headers,
	)
}

// startAssistantRuntimeSession 请求并断言成功创建公开运行会话。
func startAssistantRuntimeSession(
	t *testing.T,
	serverURL string,
	publicID string,
	origin string,
	anonymousClientID string,
) (
	*models.AssistantRuntimeStartResponse,
	*assistantRuntimeHTTPResult,
) {
	t.Helper()

	result := requestAssistantRuntimeSession(
		t,
		serverURL,
		publicID,
		origin,
		anonymousClientID,
	)

	if result.Response.StatusCode !=
		http.StatusOK {
		t.Fatalf(
			"创建公开运行会话失败: HTTP=%d body=%s",
			result.Response.StatusCode,
			string(result.Body),
		)
	}

	if result.API == nil ||
		result.API.Code != 0 {
		t.Fatalf(
			"创建公开运行会话业务失败: response=%+v",
			result.API,
		)
	}

	var response models.AssistantRuntimeStartResponse

	if err := json.Unmarshal(
		result.API.Data,
		&response,
	); err != nil {
		t.Fatalf(
			"解析公开运行会话响应失败: %v raw=%s",
			err,
			string(result.API.Data),
		)
	}

	if response.SessionID == "" ||
		response.RuntimeToken == "" {
		t.Fatalf(
			"公开运行会话缺少ID或令牌: %+v",
			response,
		)
	}

	return &response,
		result
}

// requestAssistantRuntimeSessionView 请求读取会话，不预设成功状态。
func requestAssistantRuntimeSessionView(
	t *testing.T,
	serverURL string,
	sessionID string,
	runtimeToken string,
) *assistantRuntimeHTTPResult {
	t.Helper()

	return doAssistantRuntimeHTTPRequest(
		t,
		http.MethodGet,
		serverURL+
			"/api/v1/assistant-runtime/sessions/"+
			sessionID,
		nil,
		map[string]string{
			"Authorization":
				"Bearer " + runtimeToken,
		},
	)
}

// readAssistantRuntimeSessionView 请求并断言成功读取会话。
func readAssistantRuntimeSessionView(
	t *testing.T,
	serverURL string,
	sessionID string,
	runtimeToken string,
) (
	*models.AssistantRuntimeSessionView,
	*assistantRuntimeHTTPResult,
) {
	t.Helper()

	result := requestAssistantRuntimeSessionView(
		t,
		serverURL,
		sessionID,
		runtimeToken,
	)

	if result.Response.StatusCode !=
		http.StatusOK {
		t.Fatalf(
			"读取公开运行会话失败: HTTP=%d body=%s",
			result.Response.StatusCode,
			string(result.Body),
		)
	}

	if result.API == nil ||
		result.API.Code != 0 {
		t.Fatalf(
			"读取公开运行会话业务失败: response=%+v",
			result.API,
		)
	}

	var response models.AssistantRuntimeSessionView

	if err := json.Unmarshal(
		result.API.Data,
		&response,
	); err != nil {
		t.Fatalf(
			"解析公开运行会话视图失败: %v raw=%s",
			err,
			string(result.API.Data),
		)
	}

	return &response,
		result
}

// parseAssistantRuntimeTestTokenClaims 不验证签名地读取测试令牌声明。
//
// 签名和用途隔离已经由services包单元测试验证；
// 本辅助只用于核对HTTP响应令牌和数据库JTI哈希的绑定关系。
func parseAssistantRuntimeTestTokenClaims(
	t *testing.T,
	tokenString string,
) *assistantRuntimeTestTokenClaims {
	t.Helper()

	claims := &assistantRuntimeTestTokenClaims{}

	parser := jwt.NewParser()

	token,
		_,
		err :=
		parser.ParseUnverified(
			tokenString,
			claims,
		)
	if err != nil {
		t.Fatalf(
			"解析教学智能体测试令牌失败: %v",
			err,
		)
	}

	if token == nil ||
		claims.ID == "" ||
		claims.SessionID == "" ||
		claims.DeploymentID == "" {
		t.Fatalf(
			"教学智能体测试令牌声明不完整: %+v",
			claims,
		)
	}

	return claims
}

// assistantRuntimeTestJTIHash 计算数据库应保存的JTI SHA-256。
func assistantRuntimeTestJTIHash(
	jti string,
) string {
	sum := sha256.Sum256(
		[]byte(
			strings.TrimSpace(jti),
		),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

// assistantRuntimeTestTokenPayload 返回JWT载荷原文。
func assistantRuntimeTestTokenPayload(
	t *testing.T,
	tokenString string,
) string {
	t.Helper()

	parts := strings.Split(
		tokenString,
		".",
	)

	if len(parts) != 3 {
		t.Fatalf(
			"教学智能体测试令牌格式错误",
		)
	}

	payload,
		err :=
		base64.RawURLEncoding.DecodeString(
			parts[1],
		)
	if err != nil {
		t.Fatalf(
			"解码教学智能体测试令牌载荷失败: %v",
			err,
		)
	}

	return string(payload)
}

// tamperAssistantRuntimeToken 修改JWT签名段，构造签名无效令牌。
//
// 不修改载荷和声明，确保HTTP测试验证的确实是签名校验，
// 而不是因载荷损坏或JSON格式错误被拒绝。
func tamperAssistantRuntimeToken(
	tokenString string,
) string {
	parts := strings.Split(
		tokenString,
		".",
	)

	if len(parts) != 3 ||
		parts[2] == "" {
		return tokenString + ".invalid"
	}

	signatureBytes := []byte(
		parts[2],
	)

	if signatureBytes[0] == 'A' {
		signatureBytes[0] = 'B'
	} else {
		signatureBytes[0] = 'A'
	}

	parts[2] = string(
		signatureBytes,
	)

	return strings.Join(
		parts,
		".",
	)
}

// assertAssistantRuntimeNoSensitiveFields 检查公开正文没有内部字段名。
func assertAssistantRuntimeNoSensitiveFields(
	t *testing.T,
	raw string,
) {
	t.Helper()

	for _, sensitive := range []string{
		"owner_user_id",
		"school_id",
		"token_jti_hash",
		"anonymous_client_hash",
		"ip_hash",
		"assistant_prompt_snapshot",
		"context_snapshot_json",
		"api_key",
		"model_name",
		"provider",
	} {
		if strings.Contains(
			raw,
			sensitive,
		) {
			t.Fatalf(
				"公开响应或令牌载荷泄露敏感字段: %s raw=%s",
				sensitive,
				raw,
			)
		}
	}
}

// assertAssistantRuntimeShortTTL 验证运行令牌保持5—15分钟短时范围。
func assertAssistantRuntimeShortTTL(
	t *testing.T,
	claims *assistantRuntimeTestTokenClaims,
) {
	t.Helper()

	if claims == nil ||
		claims.IssuedAt == nil ||
		claims.ExpiresAt == nil {
		t.Fatalf(
			"运行令牌缺少签发或过期时间: %+v",
			claims,
		)
	}

	ttl := claims.ExpiresAt.Time.Sub(
		claims.IssuedAt.Time,
	)

	if ttl < 5*time.Minute ||
		ttl > 15*time.Minute+
			time.Second {
		t.Fatalf(
			"运行令牌有效期不在5—15分钟范围: %s",
			ttl,
		)
	}
}
