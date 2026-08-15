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

	"github.com/golang-jwt/jwt/v5"
)

type identityTokenVerifierTestFixture struct {
	client     *IdentityClient
	privateKey ed25519.PrivateKey
	issuer     string
	now        time.Time
}

type identityTokenVerifierTestClaims struct {
	issuer    string
	audience  string
	subject   string
	nonce     string
	issuedAt  *time.Time
	expiresAt *time.Time
	kid       string
}

// identityTokenVerifierNonce按生产Flow相同编码合同构造测试nonce：
// 32字节输入 -> 无padding base64url。
// 测试值不要求随机，但必须满足与真实state/nonce相同的协议格式。
func identityTokenVerifierNonce(
	fill byte,
) string {
	raw := make([]byte, 32)

	for i := range raw {
		raw[i] = fill
	}

	return base64.RawURLEncoding.
		EncodeToString(raw)
}

func newIdentityTokenVerifierFixture(
	t *testing.T,
) identityTokenVerifierTestFixture {
	t.Helper()

	publicKey, privateKey, err :=
		ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf(
			"GenerateKey() error = %v",
			err,
		)
	}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				if r.Method != http.MethodGet ||
					r.URL.Path != "/.well-known/jwks.json" {
					http.NotFound(w, r)
					return
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)

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
			},
		),
	)
	t.Cleanup(server.Close)

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

	now := time.Now().
		UTC().
		Truncate(time.Second)

	client.now = func() time.Time {
		return now
	}

	return identityTokenVerifierTestFixture{
		client:     client,
		privateKey: privateKey,
		issuer:     server.URL,
		now:        now,
	}
}

func newIdentityTokenVerifierValidClaims(
	fixture identityTokenVerifierTestFixture,
) identityTokenVerifierTestClaims {
	issuedAt :=
		fixture.now.Add(-time.Second)

	expiresAt :=
		fixture.now.Add(4 * time.Minute)

	return identityTokenVerifierTestClaims{
		issuer: fixture.issuer,

		audience: identityClientTestClientID,

		subject: identityClientTestGlobalPersonID,

		nonce: identityTokenVerifierNonce(
			0x11,
		),

		issuedAt: &issuedAt,

		expiresAt: &expiresAt,

		kid: identityClientTestKID,
	}
}

func signIdentityTokenVerifierToken(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	values identityTokenVerifierTestClaims,
) string {
	t.Helper()

	claims := &IdentityIDTokenClaims{
		Nonce: values.nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  values.issuer,
			Subject: values.subject,
			Audience: jwt.ClaimStrings{
				values.audience,
			},
		},
	}

	if values.issuedAt != nil {
		claims.IssuedAt =
			jwt.NewNumericDate(
				*values.issuedAt,
			)
	}

	if values.expiresAt != nil {
		claims.ExpiresAt =
			jwt.NewNumericDate(
				*values.expiresAt,
			)
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodEdDSA,
		claims,
	)

	if values.kid != "" {
		token.Header["kid"] =
			values.kid
	}

	rawToken, err :=
		token.SignedString(
			privateKey,
		)
	if err != nil {
		t.Fatalf(
			"SignedString() error = %v",
			err,
		)
	}

	return rawToken
}

// TestIdentityTokenVerifierAcceptsStrictEdDSA验证正常的
// Ed25519 + OKP/Ed25519 JWKS + 最小OIDC Claims链路。
func TestIdentityTokenVerifierAcceptsStrictEdDSA(
	t *testing.T,
) {
	fixture :=
		newIdentityTokenVerifierFixture(t)

	values :=
		newIdentityTokenVerifierValidClaims(
			fixture,
		)

	rawToken :=
		signIdentityTokenVerifierToken(
			t,
			fixture.privateKey,
			values,
		)

	claims, err :=
		fixture.client.VerifyIDToken(
			context.Background(),
			rawToken,
			values.nonce,
		)
	if err != nil {
		t.Fatalf(
			"VerifyIDToken() error = %v",
			err,
		)
	}

	if claims.Subject !=
		identityClientTestGlobalPersonID {
		t.Fatalf(
			"Subject=%q",
			claims.Subject,
		)
	}

	if claims.Issuer !=
		fixture.issuer {
		t.Fatalf(
			"Issuer=%q",
			claims.Issuer,
		)
	}

	if claims.Nonce !=
		values.nonce {
		t.Fatalf(
			"Nonce=%q",
			claims.Nonce,
		)
	}

	if len(claims.Audience) != 1 ||
		claims.Audience[0] !=
			identityClientTestClientID {
		t.Fatalf(
			"Audience=%v",
			claims.Audience,
		)
	}
}

// TestIdentityTokenVerifierRejectsClaimViolations冻结
// issuer/audience/nonce/sub/iat/exp、300秒生命周期和kid安全边界。
func TestIdentityTokenVerifierRejectsClaimViolations(
	t *testing.T,
) {
	tests := []struct {
		name string

		change func(
			*identityTokenVerifierTestClaims,
			identityTokenVerifierTestFixture,
		)

		expectedNonce string
	}{
		{
			name: "wrong issuer",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.issuer =
					"https://wrong-issuer.example"
			},
		},
		{
			name: "wrong audience",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.audience =
					"another-client"
			},
		},
		{
			name: "wrong nonce",
			change: func(
				_ *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
			},
			expectedNonce: identityTokenVerifierNonce(
				0x22,
			),
		},
		{
			name: "invalid subject",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.subject =
					"not-a-global-person-uuid"
			},
		},
		{
			name: "missing issued at",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.issuedAt = nil
			},
		},
		{
			name: "missing expires at",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.expiresAt = nil
			},
		},
		{
			name: "expired token",
			change: func(
				v *identityTokenVerifierTestClaims,
				f identityTokenVerifierTestFixture,
			) {
				expired :=
					f.now.Add(
						-2 * time.Minute,
					)

				v.expiresAt =
					&expired
			},
		},
		{
			name: "issued at too far future",
			change: func(
				v *identityTokenVerifierTestClaims,
				f identityTokenVerifierTestFixture,
			) {
				future :=
					f.now.Add(
						2 * time.Minute,
					)

				expires :=
					future.Add(
						time.Minute,
					)

				v.issuedAt =
					&future
				v.expiresAt =
					&expires
			},
		},
		{
			name: "signed lifetime over 300 seconds",
			change: func(
				v *identityTokenVerifierTestClaims,
				f identityTokenVerifierTestFixture,
			) {
				issued :=
					f.now.Add(
						-time.Second,
					)

				expires :=
					issued.Add(
						301 * time.Second,
					)

				v.issuedAt =
					&issued
				v.expiresAt =
					&expires
			},
		},
		{
			name: "unknown kid",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.kid =
					"unknown-test-key"
			},
		},
		{
			name: "missing kid",
			change: func(
				v *identityTokenVerifierTestClaims,
				_ identityTokenVerifierTestFixture,
			) {
				v.kid = ""
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				fixture :=
					newIdentityTokenVerifierFixture(
						t,
					)

				values :=
					newIdentityTokenVerifierValidClaims(
						fixture,
					)

				testCase.change(
					&values,
					fixture,
				)

				rawToken :=
					signIdentityTokenVerifierToken(
						t,
						fixture.privateKey,
						values,
					)

				expectedNonce :=
					values.nonce

				if testCase.expectedNonce != "" {
					expectedNonce =
						testCase.expectedNonce
				}

				if _, err :=
					fixture.client.VerifyIDToken(
						context.Background(),
						rawToken,
						expectedNonce,
					); err == nil {
					t.Fatal(
						"非法ID Token必须被拒绝",
					)
				}
			},
		)
	}
}

// TestIdentityTokenVerifierRejectsNonEdDSA确认JWT算法白名单不能降级。
func TestIdentityTokenVerifierRejectsNonEdDSA(
	t *testing.T,
) {
	fixture :=
		newIdentityTokenVerifierFixture(t)

	values :=
		newIdentityTokenVerifierValidClaims(
			fixture,
		)

	claims := &IdentityIDTokenClaims{
		Nonce: values.nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  values.issuer,
			Subject: values.subject,
			Audience: jwt.ClaimStrings{
				values.audience,
			},
			ExpiresAt: jwt.NewNumericDate(
				*values.expiresAt,
			),
			IssuedAt: jwt.NewNumericDate(
				*values.issuedAt,
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	token.Header["kid"] =
		identityClientTestKID

	rawToken, err :=
		token.SignedString(
			[]byte(
				"not-an-ed25519-private-key",
			),
		)
	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		fixture.client.VerifyIDToken(
			context.Background(),
			rawToken,
			values.nonce,
		); err == nil {
		t.Fatal(
			"非EdDSA ID Token必须被拒绝",
		)
	}
}
