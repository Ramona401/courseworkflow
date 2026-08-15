package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type identityClientCompletionTestCase struct {
	userInfo map[string]interface{}
}

func runIdentityClientCompletionTest(
	t *testing.T,
	testCase identityClientCompletionTestCase,
) (
	IdentityAuthorizationIdentity,
	error,
) {
	t.Helper()

	publicKey, privateKey, err :=
		ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf(
			"GenerateKey() error = %v",
			err,
		)
	}

	now := time.Now().
		UTC().
		Truncate(time.Second)

	expectedNonce := ""

	var server *httptest.Server

	server = httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)

				switch r.URL.Path {
				case "/oauth/token":
					issuedAt :=
						now.Add(-time.Second)

					expiresAt :=
						now.Add(4 * time.Minute)

					rawIDToken :=
						signIdentityTokenVerifierToken(
							t,
							privateKey,
							identityTokenVerifierTestClaims{
								issuer:    server.URL,
								audience:  identityClientTestClientID,
								subject:   identityClientTestGlobalPersonID,
								nonce:     expectedNonce,
								issuedAt:  &issuedAt,
								expiresAt: &expiresAt,
								kid:       identityClientTestKID,
							},
						)

					_ = json.NewEncoder(w).Encode(
						IdentityTokenResponse{
							AccessToken: "opaque-completion-access-token",
							TokenType:   "Bearer",
							ExpiresIn:   300,
							IDToken:     rawIDToken,
							Scope:       "openid profile platform_link",
						},
					)

				case "/.well-known/jwks.json":
					_ = json.NewEncoder(w).Encode(
						map[string]interface{}{
							"keys": []map[string]interface{}{
								{
									"kty": "OKP",
									"crv": "Ed25519",
									"kid": identityClientTestKID,
									"x": base64.RawURLEncoding.
										EncodeToString(
											publicKey,
										),
									"use": "sig",
									"alg": "EdDSA",
								},
							},
						},
					)

				case "/oauth/userinfo":
					if r.Header.Get(
						"Authorization",
					) !=
						"Bearer opaque-completion-access-token" {
						w.WriteHeader(
							http.StatusUnauthorized,
						)
						return
					}

					payload :=
						testCase.userInfo

					if payload == nil {
						payload =
							map[string]interface{}{
								"sub":  identityClientTestGlobalPersonID,
								"name": "TE-DNA Identity Test User",
								"platform_link": map[string]interface{}{
									"linked":           true,
									"local_account_id": identityFlowTestLocalUserID,
								},
							}
					}

					_ = json.NewEncoder(w).
						Encode(payload)

				default:
					http.NotFound(w, r)
				}
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
		t.Fatalf(
			"NewIdentityClient() error = %v",
			err,
		)
	}

	client.now = func() time.Time {
		return now
	}

	protector :=
		newIdentityFlowTestProtector(
			t,
			now,
		)

	flow, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatalf(
			"NewAuthorizationFlow() error = %v",
			err,
		)
	}

	expectedNonce = flow.Nonce

	return client.CompleteAuthorization(
		context.Background(),
		"authorization-code",
		flow,
	)
}

// TestIdentityClientCompletionAcceptsLinkedMapping验证完整OIDC成功链。
// global_person_id只来自已验签ID Token，平台映射只来自同一Access Token的UserInfo。
func TestIdentityClientCompletionAcceptsLinkedMapping(
	t *testing.T,
) {
	result, err :=
		runIdentityClientCompletionTest(
			t,
			identityClientCompletionTestCase{},
		)
	if err != nil {
		t.Fatalf(
			"CompleteAuthorization() error = %v",
			err,
		)
	}

	if result.GlobalPersonID !=
		identityClientTestGlobalPersonID {
		t.Fatalf(
			"GlobalPersonID=%q",
			result.GlobalPersonID,
		)
	}

	if result.Name !=
		"TE-DNA Identity Test User" {
		t.Fatalf(
			"Name=%q",
			result.Name,
		)
	}

	if !result.PlatformLink.Linked ||
		result.PlatformLink.LocalAccountID !=
			identityFlowTestLocalUserID {
		t.Fatalf(
			"PlatformLink=%+v",
			result.PlatformLink,
		)
	}
}

// TestIdentityClientCompletionAcceptsUnlinkedMapping确认
// {"linked":false}是Phase 1 Link前的合法映射事实。
func TestIdentityClientCompletionAcceptsUnlinkedMapping(
	t *testing.T,
) {
	result, err :=
		runIdentityClientCompletionTest(
			t,
			identityClientCompletionTestCase{
				userInfo: map[string]interface{}{
					"sub": identityClientTestGlobalPersonID,
					"platform_link": map[string]interface{}{
						"linked": false,
					},
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"CompleteAuthorization() error = %v",
			err,
		)
	}

	if result.PlatformLink.Linked ||
		result.PlatformLink.LocalAccountID != "" {
		t.Fatalf(
			"Unlinked PlatformLink=%+v",
			result.PlatformLink,
		)
	}
}

// TestIdentityClientCompletionRejectsInvalidUserInfo冻结
// ID Token通过后仍必须严格验证UserInfo sub和platform_link。
func TestIdentityClientCompletionRejectsInvalidUserInfo(
	t *testing.T,
) {
	tests := []struct {
		name     string
		userInfo map[string]interface{}
	}{
		{
			name: "sub mismatch",

			userInfo: map[string]interface{}{
				"sub": identityClientTestOtherGlobalPersonID,
				"platform_link": map[string]interface{}{
					"linked": false,
				},
			},
		},
		{
			name: "platform link missing",

			userInfo: map[string]interface{}{
				"sub": identityClientTestGlobalPersonID,
			},
		},
		{
			name: "unlinked still has local account",

			userInfo: map[string]interface{}{
				"sub": identityClientTestGlobalPersonID,
				"platform_link": map[string]interface{}{
					"linked":           false,
					"local_account_id": identityFlowTestLocalUserID,
				},
			},
		},
		{
			name: "linked local account invalid",

			userInfo: map[string]interface{}{
				"sub": identityClientTestGlobalPersonID,
				"platform_link": map[string]interface{}{
					"linked":           true,
					"local_account_id": "not-a-local-user-uuid",
				},
			},
		},
		{
			name: "linked local account other canonical uuid",

			userInfo: map[string]interface{}{
				"sub": identityClientTestGlobalPersonID,
				"platform_link": map[string]interface{}{
					"linked":           true,
					"local_account_id": "44444444-4444-4444-8444-444444444444",
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				result, err :=
					runIdentityClientCompletionTest(
						t,
						identityClientCompletionTestCase{
							userInfo: testCase.userInfo,
						},
					)

				switch testCase.name {
				case "linked local account other canonical uuid":
					// OIDC Client只验证这是一个规范TE-DNA本地UUID。
					// “是否等于当前登录用户”属于上层Account Link Service职责。
					if err != nil {
						t.Fatalf(
							"合法其它local_account_id不应在OIDC层失败：%v",
							err,
						)
					}

					if result.PlatformLink.LocalAccountID !=
						"44444444-4444-4444-8444-444444444444" {
						t.Fatalf(
							"PlatformLink=%+v",
							result.PlatformLink,
						)
					}

				default:
					if err == nil {
						t.Fatal(
							"非法UserInfo映射必须被拒绝",
						)
					}
				}
			},
		)
	}
}
