package integration

// test_http_helpers.go
//
// 集中提供集成测试HTTP请求、统一JSON响应、登录和断言辅助。
// 本文件不初始化数据库、不写业务种子，也不保存任何生产凭据。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// APIResponse 对应后端统一JSON响应。
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// DoRequest 发送HTTP请求并尝试解析统一JSON响应。
//
// HTML和SSE等非JSON响应不会被当作测试框架错误，
// 原始响应正文会保存在Message中并以Code=-999标识。
func DoRequest(
	t *testing.T,
	method string,
	requestURL string,
	body interface{},
	token string,
) (
	*http.Response,
	*APIResponse,
) {
	t.Helper()

	var bodyReader io.Reader

	if body != nil {
		encoded, err := json.Marshal(
			body,
		)
		if err != nil {
			t.Fatalf(
				"序列化请求体失败: %v",
				err,
			)
		}

		bodyReader = bytes.NewReader(
			encoded,
		)
	}

	request, err := http.NewRequest(
		method,
		requestURL,
		bodyReader,
	)
	if err != nil {
		t.Fatalf(
			"创建HTTP请求失败: %v",
			err,
		)
	}

	if body != nil {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	if token != "" {
		request.Header.Set(
			"Authorization",
			"Bearer "+token,
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
			"发送HTTP请求失败: %v",
			err,
		)
	}

	responseBody, readErr := io.ReadAll(
		response.Body,
	)

	closeErr := response.Body.Close()

	if readErr != nil {
		t.Fatalf(
			"读取HTTP响应失败: %v",
			readErr,
		)
	}

	if closeErr != nil {
		t.Fatalf(
			"关闭HTTP响应失败: %v",
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

	return response,
		apiResponse
}

// DoGet 发送GET请求。
func DoGet(
	t *testing.T,
	requestURL string,
	token string,
) (
	*http.Response,
	*APIResponse,
) {
	t.Helper()

	return DoRequest(
		t,
		http.MethodGet,
		requestURL,
		nil,
		token,
	)
}

// DoPost 发送POST请求。
func DoPost(
	t *testing.T,
	requestURL string,
	body interface{},
	token string,
) (
	*http.Response,
	*APIResponse,
) {
	t.Helper()

	return DoRequest(
		t,
		http.MethodPost,
		requestURL,
		body,
		token,
	)
}

// DoPut 发送PUT请求。
func DoPut(
	t *testing.T,
	requestURL string,
	body interface{},
	token string,
) (
	*http.Response,
	*APIResponse,
) {
	t.Helper()

	return DoRequest(
		t,
		http.MethodPut,
		requestURL,
		body,
		token,
	)
}

// DoDelete 发送DELETE请求。
func DoDelete(
	t *testing.T,
	requestURL string,
	token string,
) (
	*http.Response,
	*APIResponse,
) {
	t.Helper()

	return DoRequest(
		t,
		http.MethodDelete,
		requestURL,
		nil,
		token,
	)
}

// LoginAs 使用公共种子用户登录并返回教师JWT。
func LoginAs(
	t *testing.T,
	serverURL string,
	username string,
	password string,
) string {
	t.Helper()

	response, apiResponse := DoPost(
		t,
		serverURL+"/api/v1/auth/login",
		map[string]string{
			"username": username,
			"password": password,
		},
		"",
	)

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"登录失败(user=%s): HTTP=%d message=%s",
			username,
			response.StatusCode,
			apiResponse.Message,
		)
	}

	if apiResponse.Code != 0 {
		t.Fatalf(
			"登录失败(user=%s): code=%d message=%s",
			username,
			apiResponse.Code,
			apiResponse.Message,
		)
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(
		apiResponse.Data,
		&result,
	); err != nil {
		t.Fatalf(
			"解析登录结果失败(user=%s): %v",
			username,
			err,
		)
	}

	if result.Token == "" {
		t.Fatalf(
			"登录成功但token为空(user=%s)",
			username,
		)
	}

	return result.Token
}

// LoginAsAdmin 使用管理员公共种子登录。
func LoginAsAdmin(
	t *testing.T,
	serverURL string,
) string {
	t.Helper()

	return LoginAs(
		t,
		serverURL,
		SeedAdminUsername,
		SeedAdminPassword,
	)
}

// LoginAsOperator 使用操作员公共种子登录。
func LoginAsOperator(
	t *testing.T,
	serverURL string,
) string {
	t.Helper()

	return LoginAs(
		t,
		serverURL,
		SeedOperatorUsername,
		SeedOperatorPassword,
	)
}

// LoginAsSenior 使用高级操作员公共种子登录。
func LoginAsSenior(
	t *testing.T,
	serverURL string,
) string {
	t.Helper()

	return LoginAs(
		t,
		serverURL,
		SeedSeniorUsername,
		SeedSeniorPassword,
	)
}

// LoginAsViewer 使用只读用户公共种子登录。
func LoginAsViewer(
	t *testing.T,
	serverURL string,
) string {
	t.Helper()

	return LoginAs(
		t,
		serverURL,
		SeedViewerUsername,
		SeedViewerPassword,
	)
}

// ParseData 解析统一响应中的data字段。
func ParseData(
	t *testing.T,
	apiResponse *APIResponse,
	target interface{},
) {
	t.Helper()

	if apiResponse == nil ||
		apiResponse.Data == nil {
		t.Fatal(
			"API响应Data为nil",
		)
	}

	if err := json.Unmarshal(
		apiResponse.Data,
		target,
	); err != nil {
		t.Fatalf(
			"解析API响应Data失败: %v raw=%s",
			err,
			string(apiResponse.Data),
		)
	}
}

// AssertHTTPStatus 断言HTTP状态码。
func AssertHTTPStatus(
	t *testing.T,
	response *http.Response,
	expected int,
) {
	t.Helper()

	if response == nil {
		t.Fatal(
			"HTTP响应为nil",
		)
	}

	if response.StatusCode != expected {
		t.Errorf(
			"期望HTTP状态码%d，实际%d",
			expected,
			response.StatusCode,
		)
	}
}

// AssertAPICode 断言统一响应业务码。
func AssertAPICode(
	t *testing.T,
	apiResponse *APIResponse,
	expected int,
) {
	t.Helper()

	if apiResponse == nil {
		t.Fatal(
			"API响应为nil",
		)
	}

	if apiResponse.Code != expected {
		t.Errorf(
			"期望API Code=%d，实际=%d message=%s",
			expected,
			apiResponse.Code,
			apiResponse.Message,
		)
	}
}
