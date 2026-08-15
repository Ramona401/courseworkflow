package services

// identity_http.go — Identity Center服务器间HTTP安全传输辅助。
//
// 职责严格限制在传输层：
//   - 设置短超时；
//   - 禁止自动跟随Redirect，避免Bearer Token或Client凭据跨地址泄露；
//   - 对上游响应做硬大小限制，防止异常响应导致无界内存读取。
//
// 本文件不解析OIDC业务字段，不持有Client Secret，也不记录响应正文。

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const identitySecureHTTPTimeout = 10 * time.Second

// newIdentitySecureHTTPClient 创建Identity专用HTTP Client。
//
// CheckRedirect固定拒绝自动跳转。调用方会把3xx作为非预期响应处理，
// 不允许Go客户端自动把认证上下文带到Location目标。
func newIdentitySecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: identitySecureHTTPTimeout,
		CheckRedirect: func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
}

// readIdentityBoundedResponseBody 在硬上限内读取Identity响应。
//
// 使用maxBytes+1探测超限；一旦发现正文超过允许大小即fail-closed，
// 不向调用方返回被截断内容，避免把不完整JSON误当成合法协议响应。
func readIdentityBoundedResponseBody(
	reader io.Reader,
	maxBytes int64,
) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf(
			"Identity响应正文不可用",
		)
	}

	if maxBytes <= 0 {
		return nil, fmt.Errorf(
			"Identity响应大小上限无效",
		)
	}

	limited := io.LimitReader(
		reader,
		maxBytes+1,
	)

	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf(
			"读取Identity响应失败：%w",
			err,
		)
	}

	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf(
			"Identity响应超过大小限制",
		)
	}

	return body, nil
}
