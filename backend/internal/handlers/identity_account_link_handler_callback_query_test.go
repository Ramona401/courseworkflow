package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"tedna/internal/services"
)

// TestIdentityAccountLinkHandlerCallbackQueryAcceptsSuccess冻结合法OIDC
// success形态：必须且只能有state+code。
func TestIdentityAccountLinkHandlerCallbackQueryAcceptsSuccess(
	t *testing.T,
) {
	values := url.Values{}
	values.Set(
		"state",
		"valid-success-state",
	)
	values.Set(
		"code",
		"authorization-code",
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/identity/callback?"+
			values.Encode(),
		nil,
	)

	callback, ok :=
		parseIdentityAccountLinkCallback(
			request,
		)

	if !ok {
		t.Fatal(
			"合法OIDC success query被错误拒绝",
		)
	}

	if callback.State !=
		"valid-success-state" ||
		callback.Code !=
			"authorization-code" ||
		callback.ProviderError != "" {
		t.Fatalf(
			"success callback解析异常：%+v",
			callback,
		)
	}
}

// TestIdentityAccountLinkHandlerCallbackQueryAcceptsProviderError冻结合法OIDC
// error形态：state+error，可选error_description；description只校验后丢弃。
func TestIdentityAccountLinkHandlerCallbackQueryAcceptsProviderError(
	t *testing.T,
) {
	values := url.Values{}
	values.Set(
		"state",
		"valid-error-state",
	)
	values.Set(
		"error",
		"access_denied",
	)
	values.Set(
		"error_description",
		"sensitive-provider-description",
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/identity/callback?"+
			values.Encode(),
		nil,
	)

	callback, ok :=
		parseIdentityAccountLinkCallback(
			request,
		)

	if !ok {
		t.Fatal(
			"合法OIDC error query被错误拒绝",
		)
	}

	if callback.State !=
		"valid-error-state" ||
		callback.Code != "" ||
		callback.ProviderError !=
			"access_denied" {
		t.Fatalf(
			"provider error解析异常：%+v",
			callback,
		)
	}

	// 结构体没有error_description字段，保证原始上游描述不会继续传播。
	if strings.Contains(
		callback.ProviderError,
		"sensitive-provider-description",
	) {
		t.Fatal(
			"error_description不得进入callback结果",
		)
	}
}

// TestIdentityAccountLinkHandlerCallbackQueryRejectsInvalidForms冻结严格query合同。
// success/error必须按参数“存在性”互斥，同时拒绝重复、未知和缺失参数。
func TestIdentityAccountLinkHandlerCallbackQueryRejectsInvalidForms(
	t *testing.T,
) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{
			name: "code plus empty error",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error": {
					"",
				},
			},
		},
		{
			name: "code plus provider error",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error": {
					"access_denied",
				},
			},
		},
		{
			name: "code plus error description",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error_description": {
					"must-be-rejected",
				},
			},
		},
		{
			name: "empty code",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"code": {
					"",
				},
			},
		},
		{
			name: "empty provider error",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"error": {
					"",
				},
			},
		},
		{
			name: "missing response",

			values: url.Values{
				"state": {
					"valid-state",
				},
			},
		},
		{
			name: "missing state",

			values: url.Values{
				"code": {
					"authorization-code",
				},
			},
		},
		{
			name: "duplicate state",

			values: url.Values{
				"state": {
					"state-one",
					"state-two",
				},
				"code": {
					"authorization-code",
				},
			},
		},
		{
			name: "duplicate code",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"code": {
					"code-one",
					"code-two",
				},
			},
		},
		{
			name: "duplicate provider error",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"error": {
					"access_denied",
					"server_error",
				},
			},
		},
		{
			name: "unknown query parameter",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"code": {
					"authorization-code",
				},
				"local_account_id": {
					identityHandlerTestOtherUserID,
				},
			},
		},
		{
			name: "state with surrounding whitespace",

			values: url.Values{
				"state": {
					" bad-state ",
				},
				"code": {
					"authorization-code",
				},
			},
		},
		{
			name: "code with surrounding whitespace",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"code": {
					" bad-code ",
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				request :=
					httptest.NewRequest(
						http.MethodGet,
						"/api/v1/auth/identity/callback?"+
							testCase.values.Encode(),
						nil,
					)

				_, ok :=
					parseIdentityAccountLinkCallback(
						request,
					)

				if ok {
					t.Fatalf(
						"非法callback query被错误接受：%s",
						testCase.values.Encode(),
					)
				}
			},
		)
	}
}

// TestIdentityAccountLinkHandlerCallbackQueryRejectsOversizedFields冻结
// callback字段大小上限，避免超长输入进入后续OIDC和Service。
func TestIdentityAccountLinkHandlerCallbackQueryRejectsOversizedFields(
	t *testing.T,
) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{
			name: "oversized state",

			values: url.Values{
				"state": {
					strings.Repeat(
						"s",
						identityCallbackStateMaxBytes+1,
					),
				},
				"code": {
					"authorization-code",
				},
			},
		},
		{
			name: "oversized code",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"code": {
					strings.Repeat(
						"c",
						identityCallbackCodeMaxBytes+1,
					),
				},
			},
		},
		{
			name: "oversized provider error",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"error": {
					strings.Repeat(
						"e",
						identityCallbackErrorMaxBytes+1,
					),
				},
			},
		},
		{
			name: "oversized error description",

			values: url.Values{
				"state": {
					"valid-state",
				},
				"error": {
					"access_denied",
				},
				"error_description": {
					strings.Repeat(
						"d",
						identityCallbackDescriptionMaxBytes+1,
					),
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				request :=
					httptest.NewRequest(
						http.MethodGet,
						"/api/v1/auth/identity/callback?"+
							testCase.values.Encode(),
						nil,
					)

				_, ok :=
					parseIdentityAccountLinkCallback(
						request,
					)

				if ok {
					t.Fatalf(
						"超长callback字段被错误接受：%s",
						testCase.name,
					)
				}
			},
		)
	}
}

// TestIdentityAccountLinkHandlerCallbackQueryHTTPFailClosed从HTTP边界再次冻结
// 关键混合输入：必须清除Flow Cookie、固定安全错误跳转，并且不得取得Service。
func TestIdentityAccountLinkHandlerCallbackQueryHTTPFailClosed(
	t *testing.T,
) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{
			name: "code plus empty error",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error": {
					"",
				},
			},
		},
		{
			name: "code plus error description",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error_description": {
					"sensitive-description",
				},
			},
		},
		{
			name: "code plus provider error",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"error": {
					"access_denied",
				},
			},
		},
		{
			name: "unknown local account parameter",

			values: url.Values{
				"state": {
					"mixed-state",
				},
				"code": {
					"authorization-code",
				},
				"local_account_id": {
					identityHandlerTestOtherUserID,
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				providerCalls := 0

				handler :=
					NewIdentityAccountLinkHandler(
						func() (
							*services.IdentityAccountLinkService,
							error,
						) {
							providerCalls++
							return nil, nil
						},
					)

				request :=
					httptest.NewRequest(
						http.MethodGet,
						"/api/v1/auth/identity/callback?"+
							testCase.values.Encode(),
						nil,
					)

				request.AddCookie(
					&http.Cookie{
						Name: services.IdentityFlowCookieName,

						Value: "must-be-consumed",
					},
				)

				recorder :=
					httptest.NewRecorder()

				handler.Callback(
					recorder,
					request,
				)

				assertIdentityHandlerCallbackErrorRedirect(
					t,
					recorder,
					"IDENTITY_CALLBACK_INVALID",
				)

				if providerCalls != 0 {
					t.Fatalf(
						"非法query不得取得Service，calls=%d",
						providerCalls,
					)
				}

				location :=
					recorder.Header().Get(
						"Location",
					)

				for _, forbidden := range []string{
					"authorization-code",
					"access_denied",
					"sensitive-description",
					identityHandlerTestOtherUserID,
					"must-be-consumed",
				} {
					if strings.Contains(
						location,
						forbidden,
					) {
						t.Fatalf(
							"非法callback redirect泄露值：%q",
							forbidden,
						)
					}
				}

				assertIdentityHandlerClearedFlowCookie(
					t,
					recorder,
				)

				assertIdentityHandlerSensitiveHeaders(
					t,
					recorder,
				)
			},
		)
	}
}
