package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"tedna/internal/config"
)

const (
	identityClientTestClientID = "tedna-client"

	identityClientTestClientSecret = "identity-client-test-secret-not-production"

	identityClientTestRedirectURI = "https://workflow.pkuailab.com/api/v1/auth/identity/callback"

	identityClientTestGlobalPersonID = "11111111-1111-4111-8111-111111111111"

	identityClientTestOtherGlobalPersonID = "33333333-3333-4333-8333-333333333333"

	identityClientTestKID = "identity-test-ed25519-key"
)

func newIdentityClientTestConfig(
	issuer string,
) config.IdentityClientConfig {
	return config.IdentityClientConfig{
		Issuer:          issuer,
		ClientID:        identityClientTestClientID,
		ClientSecret:    identityClientTestClientSecret,
		RedirectURI:     identityClientTestRedirectURI,
		TokenAuthMethod: config.IdentityTokenAuthMethodPost,
	}
}

// TestIdentityClientAuthorizationURLContract冻结Authorization Code + PKCE S256合同。
// Secret和code_verifier只能停留在服务器端，绝不能进入浏览器授权URL。
func TestIdentityClientAuthorizationURLContract(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.August,
		11,
		2,
		40,
		0,
		0,
		time.UTC,
	)

	client, err := NewIdentityClient(
		newIdentityClientTestConfig(
			"https://identity.example",
		),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	protector :=
		newIdentityFlowTestProtector(
			t,
			now,
		)

	start, err :=
		client.StartAuthorization(
			protector,
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err :=
		url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Scheme != "https" ||
		parsed.Host != "identity.example" ||
		parsed.Path != "/oauth/authorize" {
		t.Fatalf(
			"Authorization Endpoint异常：%s",
			start.AuthorizationURL,
		)
	}

	query := parsed.Query()

	expected := map[string]string{
		"response_type":         "code",
		"client_id":             identityClientTestClientID,
		"redirect_uri":          identityClientTestRedirectURI,
		"scope":                 "openid profile platform_link",
		"code_challenge_method": "S256",
	}

	for key, value := range expected {
		if query.Get(key) != value {
			t.Fatalf(
				"Authorization query %s=%q want=%q",
				key,
				query.Get(key),
				value,
			)
		}
	}

	if query.Get("state") == "" ||
		query.Get("nonce") == "" ||
		query.Get("code_challenge") == "" {
		t.Fatal(
			"Authorization URL缺少state/nonce/code_challenge",
		)
	}

	if query.Get("client_secret") != "" ||
		query.Get("code_verifier") != "" {
		t.Fatal(
			"Authorization URL泄漏Secret或code_verifier",
		)
	}

	flow, err :=
		protector.Open(start.FlowToken)
	if err != nil {
		t.Fatal(err)
	}

	digest :=
		sha256.Sum256(
			[]byte(flow.CodeVerifier),
		)

	expectedChallenge :=
		base64.RawURLEncoding.EncodeToString(
			digest[:],
		)

	if query.Get("code_challenge") !=
		expectedChallenge {
		t.Fatal(
			"Authorization URL的PKCE challenge与Flow verifier不匹配",
		)
	}

	if query.Get("state") != flow.State ||
		query.Get("nonce") != flow.Nonce ||
		start.State != flow.State {
		t.Fatal(
			"Authorization URL与加密Flow的state/nonce不一致",
		)
	}

	if start.CookieName !=
		IdentityFlowCookieName {
		t.Fatalf(
			"CookieName=%q",
			start.CookieName,
		)
	}

	if strings.Contains(
		start.AuthorizationURL,
		identityClientTestClientSecret,
	) {
		t.Fatal(
			"Authorization URL不得包含Client Secret",
		)
	}
}

// TestIdentityClientTokenExchangeUsesClientSecretPost验证生产冻结的Token认证方式。
// Client Secret只能进入HTTPS Token POST正文，不能进入URL或Basic Header。
func TestIdentityClientTokenExchangeUsesClientSecretPost(
	t *testing.T,
) {
	var (
		observedMethod        string
		observedAuthorization string
		observedContentType   string
		observedGrantType     string
		observedCode          string
		observedRedirectURI   string
		observedVerifier      string
		observedClientID      string
		observedClientSecret  string
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				observedMethod =
					r.Method
				observedAuthorization =
					r.Header.Get(
						"Authorization",
					)
				observedContentType =
					r.Header.Get(
						"Content-Type",
					)

				if err := r.ParseForm(); err != nil {
					http.Error(
						w,
						"bad form",
						http.StatusBadRequest,
					)
					return
				}

				observedGrantType =
					r.Form.Get("grant_type")
				observedCode =
					r.Form.Get("code")
				observedRedirectURI =
					r.Form.Get("redirect_uri")
				observedVerifier =
					r.Form.Get("code_verifier")
				observedClientID =
					r.Form.Get("client_id")
				observedClientSecret =
					r.Form.Get("client_secret")

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).Encode(
					IdentityTokenResponse{
						AccessToken: "opaque-access-token",
						TokenType:   "Bearer",
						ExpiresIn:   300,
						IDToken:     "signed-id-token-placeholder",
						Scope:       "openid profile platform_link",
					},
				)
			},
		),
	)
	defer server.Close()

	client, err := NewIdentityClient(
		newIdentityClientTestConfig(
			server.URL,
		),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	protector :=
		newIdentityFlowTestProtector(
			t,
			time.Now().UTC(),
		)

	flow, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	result, err :=
		client.ExchangeAuthorizationCode(
			context.Background(),
			"authorization-code",
			flow.CodeVerifier,
		)
	if err != nil {
		t.Fatalf(
			"ExchangeAuthorizationCode() error = %v",
			err,
		)
	}

	if observedMethod != http.MethodPost {
		t.Fatalf(
			"Token method=%q",
			observedMethod,
		)
	}

	if observedAuthorization != "" {
		t.Fatal(
			"client_secret_post不得发送Basic Authorization",
		)
	}

	if !strings.HasPrefix(
		observedContentType,
		"application/x-www-form-urlencoded",
	) {
		t.Fatalf(
			"Token Content-Type=%q",
			observedContentType,
		)
	}

	if observedGrantType !=
		"authorization_code" ||
		observedCode !=
			"authorization-code" ||
		observedRedirectURI !=
			identityClientTestRedirectURI ||
		observedVerifier !=
			flow.CodeVerifier ||
		observedClientID !=
			identityClientTestClientID ||
		observedClientSecret !=
			identityClientTestClientSecret {
		t.Fatalf(
			"Token表单合同异常",
		)
	}

	if result.AccessToken !=
		"opaque-access-token" ||
		result.TokenType !=
			"Bearer" ||
		result.ExpiresIn != 300 {
		t.Fatalf(
			"Token结果异常：%+v",
			result,
		)
	}
}

// TestIdentityClientTokenExchangeRejectsMissingRequiredScope确认Token Endpoint
// 显式返回scope时不能丢失platform_link。
func TestIdentityClientTokenExchangeRejectsMissingRequiredScope(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).Encode(
					IdentityTokenResponse{
						AccessToken: "opaque-access-token",
						TokenType:   "Bearer",
						ExpiresIn:   300,
						IDToken:     "signed-id-token-placeholder",
						Scope:       "openid profile",
					},
				)
			},
		),
	)
	defer server.Close()

	client, err := NewIdentityClient(
		newIdentityClientTestConfig(
			server.URL,
		),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	protector :=
		newIdentityFlowTestProtector(
			t,
			time.Now().UTC(),
		)

	flow, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		client.ExchangeAuthorizationCode(
			context.Background(),
			"authorization-code",
			flow.CodeVerifier,
		); err == nil {
		t.Fatal(
			"Token响应缺少platform_link scope时必须失败",
		)
	}
}

// TestIdentityClientFetchUserInfoBearerContract冻结opaque Access Token的Bearer使用方式。
func TestIdentityClientFetchUserInfoBearerContract(
	t *testing.T,
) {
	var (
		observedMethod        string
		observedAuthorization string
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				observedMethod =
					r.Method
				observedAuthorization =
					r.Header.Get(
						"Authorization",
					)

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).Encode(
					map[string]interface{}{
						"sub":  identityClientTestGlobalPersonID,
						"name": "Identity Test User",
						"platform_link": map[string]interface{}{
							"linked": false,
						},
					},
				)
			},
		),
	)
	defer server.Close()

	client, err := NewIdentityClient(
		newIdentityClientTestConfig(
			server.URL,
		),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	userInfo, err :=
		client.FetchUserInfo(
			context.Background(),
			"opaque-access-token",
		)
	if err != nil {
		t.Fatalf(
			"FetchUserInfo() error = %v",
			err,
		)
	}

	if observedMethod != http.MethodGet {
		t.Fatalf(
			"UserInfo method=%q",
			observedMethod,
		)
	}

	if observedAuthorization !=
		"Bearer opaque-access-token" {
		t.Fatalf(
			"UserInfo Authorization=%q",
			observedAuthorization,
		)
	}

	if userInfo.Subject !=
		identityClientTestGlobalPersonID ||
		userInfo.Name !=
			"Identity Test User" ||
		userInfo.PlatformLink == nil ||
		userInfo.PlatformLink.Linked {
		t.Fatalf(
			"UserInfo结果异常：%+v",
			userInfo,
		)
	}
}

// TestIdentityClientFetchUserInfoRejectsInvalidSub确认UserInfo sub必须是规范UUID。
func TestIdentityClientFetchUserInfoRejectsInvalidSub(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				_ = json.NewEncoder(w).Encode(
					map[string]interface{}{
						"sub": "not-a-global-person-uuid",
						"platform_link": map[string]interface{}{
							"linked": false,
						},
					},
				)
			},
		),
	)
	defer server.Close()

	client, err := NewIdentityClient(
		newIdentityClientTestConfig(
			server.URL,
		),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		client.FetchUserInfo(
			context.Background(),
			"opaque-access-token",
		); err == nil {
		t.Fatal(
			"非法UserInfo sub必须被拒绝",
		)
	}
}
