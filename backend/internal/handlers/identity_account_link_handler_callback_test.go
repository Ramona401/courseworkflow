package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"tedna/internal/models"
	"tedna/internal/services"
)

type identityHandlerCallbackTestFixture struct {
	handler     *IdentityAccountLinkHandler
	protector   *services.IdentityFlowProtector
	oidc        *identityHandlerTestOIDC
	backchannel *identityHandlerTestBackchannel
	userRepo    *identityHandlerTestUserRepo
}

func newIdentityHandlerCallbackTestFixture(
	t *testing.T,
) identityHandlerCallbackTestFixture {
	t.Helper()

	protector, err :=
		services.NewIdentityFlowProtector(
			[]byte(
				"tedna-identity-handler-callback-test-root-secret-0123456789",
			),
		)
	if err != nil {
		t.Fatalf(
			"NewIdentityFlowProtector() error = %v",
			err,
		)
	}

	oidc := &identityHandlerTestOIDC{
		completeResult: services.IdentityAuthorizationIdentity{
			GlobalPersonID: identityHandlerTestGlobalPersonID,

			PlatformLink: services.IdentityPlatformLink{
				Linked: false,
			},
		},
	}

	backchannel :=
		&identityHandlerTestBackchannel{}

	userRepo :=
		&identityHandlerTestUserRepo{
			user: &models.User{
				ID: identityHandlerTestLocalUserID,

				Status: models.StatusActive,
			},
		}

	service, err :=
		services.NewIdentityAccountLinkService(
			oidc,
			backchannel,
			protector,
			userRepo,
		)
	if err != nil {
		t.Fatalf(
			"NewIdentityAccountLinkService() error = %v",
			err,
		)
	}

	handler :=
		NewIdentityAccountLinkHandler(
			func() (
				*services.IdentityAccountLinkService,
				error,
			) {
				return service, nil
			},
		)

	return identityHandlerCallbackTestFixture{
		handler:     handler,
		protector:   protector,
		oidc:        oidc,
		backchannel: backchannel,
		userRepo:    userRepo,
	}
}

func newIdentityHandlerCallbackFlow(
	t *testing.T,
	protector *services.IdentityFlowProtector,
	purpose string,
) (
	string,
	string,
) {
	t.Helper()

	flow, _, err :=
		protector.NewAuthorizationFlow(
			purpose,
			identityHandlerTestLocalUserID,
		)
	if err != nil {
		t.Fatalf(
			"NewAuthorizationFlow() error = %v",
			err,
		)
	}

	token, err :=
		protector.Seal(flow)
	if err != nil {
		t.Fatalf(
			"Seal() error = %v",
			err,
		)
	}

	return token, flow.State
}

func assertIdentityHandlerClearedFlowCookie(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()

	response := recorder.Result()
	defer response.Body.Close()

	var cleared *http.Cookie

	for _, cookie := range response.Cookies() {
		if cookie.Name ==
			services.IdentityFlowCookieName {
			cleared = cookie
			break
		}
	}

	if cleared == nil {
		t.Fatal(
			"callback没有清除Identity Flow Cookie",
		)
	}

	if cleared.Value != "" {
		t.Fatal(
			"清除Cookie时Value必须为空",
		)
	}

	if cleared.MaxAge >= 0 {
		t.Fatalf(
			"清除Cookie MaxAge=%d",
			cleared.MaxAge,
		)
	}

	if cleared.Path != "/" ||
		cleared.Domain != "" ||
		!cleared.HttpOnly ||
		!cleared.Secure ||
		cleared.SameSite !=
			http.SameSiteLaxMode {
		t.Fatalf(
			"清除Cookie安全属性异常：%+v",
			cleared,
		)
	}
}

func assertIdentityHandlerCallbackErrorRedirect(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedCode string,
) {
	t.Helper()

	if recorder.Code != http.StatusFound {
		t.Fatalf(
			"status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	location :=
		recorder.Header().Get("Location")

	parsed, err :=
		url.Parse(location)
	if err != nil {
		t.Fatalf(
			"Location解析失败：%v",
			err,
		)
	}

	if parsed.Path !=
		identityFrontendCallbackPath {
		t.Fatalf(
			"redirect path=%q",
			parsed.Path,
		)
	}

	values := parsed.Query()

	if len(values) != 1 ||
		values.Get("error") != expectedCode {
		t.Fatalf(
			"错误redirect query异常：%s",
			location,
		)
	}
}

// TestIdentityAccountLinkHandlerCallbackPublicSuccess确认callback公开入口
// 不依赖TE-DNA JWT；本地账号只从受保护Flow恢复。
func TestIdentityAccountLinkHandlerCallbackPublicSuccess(
	t *testing.T,
) {
	fixture :=
		newIdentityHandlerCallbackTestFixture(t)

	flowToken, state :=
		newIdentityHandlerCallbackFlow(
			t,
			fixture.protector,
			services.IdentityFlowPurposeLink,
		)

	values := url.Values{}
	values.Set("state", state)
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

	// 故意不注入JWT claims。
	request.AddCookie(
		&http.Cookie{
			Name: services.IdentityFlowCookieName,

			Value: flowToken,
		},
	)

	recorder :=
		httptest.NewRecorder()

	fixture.handler.Callback(
		recorder,
		request,
	)

	if recorder.Code != http.StatusFound {
		t.Fatalf(
			"status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	location :=
		recorder.Header().Get("Location")

	parsed, err :=
		url.Parse(location)
	if err != nil {
		t.Fatalf(
			"Location解析失败：%v",
			err,
		)
	}

	if parsed.Path !=
		identityFrontendCallbackPath {
		t.Fatalf(
			"redirect path=%q",
			parsed.Path,
		)
	}

	query := parsed.Query()

	if len(query) != 3 ||
		query.Get("operation") !=
			services.IdentityBackchannelOperationLink ||
		query.Get("state") != "linked" ||
		query.Get("changed") != "true" {
		t.Fatalf(
			"成功redirect异常：%s",
			location,
		)
	}

	for _, forbidden := range []string{
		identityHandlerTestLocalUserID,
		identityHandlerTestGlobalPersonID,
		flowToken,
		"authorization-code",
	} {
		if strings.Contains(
			location,
			forbidden,
		) {
			t.Fatalf(
				"成功redirect泄露敏感值：%q",
				forbidden,
			)
		}
	}

	if fixture.oidc.completeCalls != 1 {
		t.Fatalf(
			"OIDC Complete调用次数=%d",
			fixture.oidc.completeCalls,
		)
	}

	if fixture.backchannel.calls != 1 {
		t.Fatalf(
			"Backchannel调用次数=%d",
			fixture.backchannel.calls,
		)
	}

	assertIdentityHandlerClearedFlowCookie(
		t,
		recorder,
	)

	assertIdentityHandlerSensitiveHeaders(
		t,
		recorder,
	)
}

// TestIdentityAccountLinkHandlerCallbackProviderErrorIsSafe确认合法OIDC
// error响应可以带error_description，但上游正文绝不能进入前端URL。
func TestIdentityAccountLinkHandlerCallbackProviderErrorIsSafe(
	t *testing.T,
) {
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

	values := url.Values{}
	values.Set(
		"state",
		"provider-error-state",
	)
	values.Set(
		"error",
		"access_denied",
	)
	values.Set(
		"error_description",
		"sensitive-upstream-description",
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/identity/callback?"+
			values.Encode(),
		nil,
	)

	request.AddCookie(
		&http.Cookie{
			Name: services.IdentityFlowCookieName,

			Value: "opaque-flow-token",
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
		"IDENTITY_AUTHORIZATION_DENIED",
	)

	if providerCalls != 0 {
		t.Fatalf(
			"provider error不得取得Service，calls=%d",
			providerCalls,
		)
	}

	location :=
		recorder.Header().Get("Location")

	for _, forbidden := range []string{
		"access_denied",
		"sensitive-upstream-description",
		"opaque-flow-token",
	} {
		if strings.Contains(
			location,
			forbidden,
		) {
			t.Fatalf(
				"错误redirect泄露上游信息：%q",
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
}

// TestIdentityAccountLinkHandlerCallbackMissingFlowCookie确认success响应
// 缺失服务器Flow Cookie时必须fail-closed，而且不得取得Service。
func TestIdentityAccountLinkHandlerCallbackMissingFlowCookie(
	t *testing.T,
) {
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

	values := url.Values{}
	values.Set(
		"state",
		"missing-flow-state",
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

	recorder :=
		httptest.NewRecorder()

	handler.Callback(
		recorder,
		request,
	)

	assertIdentityHandlerCallbackErrorRedirect(
		t,
		recorder,
		"IDENTITY_FLOW_MISSING",
	)

	if providerCalls != 0 {
		t.Fatalf(
			"缺失Flow Cookie不得取得Service，calls=%d",
			providerCalls,
		)
	}

	assertIdentityHandlerClearedFlowCookie(
		t,
		recorder,
	)

	assertIdentityHandlerSensitiveHeaders(
		t,
		recorder,
	)
}
