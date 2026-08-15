package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
)

const (
	identityHandlerTestLocalUserID = "22222222-2222-4222-8222-222222222222"

	identityHandlerTestOtherUserID = "33333333-3333-4333-8333-333333333333"

	identityHandlerTestGlobalPersonID = "11111111-1111-4111-8111-111111111111"
)

type identityHandlerTestOIDC struct {
	startResult services.IdentityAuthorizationStart

	startCalls   int
	startPurpose string
	startUserID  string

	completeResult services.IdentityAuthorizationIdentity
	completeErr    error
	completeCalls  int
}

func (f *identityHandlerTestOIDC) StartAuthorization(
	_ *services.IdentityFlowProtector,
	purpose string,
	userID string,
) (services.IdentityAuthorizationStart, error) {
	f.startCalls++
	f.startPurpose = purpose
	f.startUserID = userID

	return f.startResult, nil
}

func (f *identityHandlerTestOIDC) CompleteAuthorization(
	_ context.Context,
	_ string,
	_ services.IdentityAuthorizationFlow,
) (services.IdentityAuthorizationIdentity, error) {
	f.completeCalls++

	return f.completeResult, f.completeErr
}

type identityHandlerTestBackchannel struct {
	calls int
}

func (f *identityHandlerTestBackchannel) Mutate(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
) (services.IdentityBackchannelResult, error) {
	f.calls++

	return services.IdentityBackchannelResult{
		Outcome: "success",
		State:   "linked",
	}, nil
}

type identityHandlerTestUserRepo struct {
	user *models.User
	err  error

	calls  int
	lastID string
}

func (r *identityHandlerTestUserRepo) FindByID(
	_ context.Context,
	id string,
) (*models.User, error) {
	r.calls++
	r.lastID = id

	return r.user, r.err
}

type identityHandlerTestFixture struct {
	handler     *IdentityAccountLinkHandler
	oidc        *identityHandlerTestOIDC
	backchannel *identityHandlerTestBackchannel
	userRepo    *identityHandlerTestUserRepo
}

func newIdentityHandlerTestFixture(
	t *testing.T,
) identityHandlerTestFixture {
	t.Helper()

	protector, err := services.NewIdentityFlowProtector(
		[]byte(
			"tedna-identity-handler-test-root-secret-0123456789abcdef",
		),
	)
	if err != nil {
		t.Fatalf(
			"NewIdentityFlowProtector() error = %v",
			err,
		)
	}

	oidc := &identityHandlerTestOIDC{
		startResult: services.IdentityAuthorizationStart{
			AuthorizationURL: "https://identity.example/oauth/authorize?state=test",

			State: "test-state",

			FlowToken: "encrypted-flow-token",

			CookieName: services.IdentityFlowCookieName,

			ExpiresAt: time.Now().
				Add(5 * time.Minute),
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

	return identityHandlerTestFixture{
		handler:     handler,
		oidc:        oidc,
		backchannel: backchannel,
		userRepo:    userRepo,
	}
}

func identityHandlerTestRequestWithClaims(
	request *http.Request,
	userID string,
) *http.Request {
	claims := &services.JWTClaims{
		UserID: userID,
	}

	ctx := context.WithValue(
		request.Context(),
		middleware.ClaimsKey,
		claims,
	)

	return request.WithContext(ctx)
}

func assertIdentityHandlerSensitiveHeaders(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()

	required := map[string]string{
		"Cache-Control": "no-store",

		"Pragma": "no-cache",

		"Referrer-Policy": "no-referrer",

		"X-Content-Type-Options": "nosniff",
	}

	for name, expected := range required {
		if actual :=
			recorder.Header().Get(name); actual != expected {
			t.Fatalf(
				"%s=%q want=%q",
				name,
				actual,
				expected,
			)
		}
	}
}

func assertIdentityHandlerFlowCookie(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()

	response := recorder.Result()
	defer response.Body.Close()

	var flowCookie *http.Cookie

	for _, cookie := range response.Cookies() {
		if cookie.Name ==
			services.IdentityFlowCookieName {
			flowCookie = cookie
			break
		}
	}

	if flowCookie == nil {
		t.Fatal(
			"响应没有设置Identity Flow Cookie",
		)
	}

	if !strings.HasPrefix(
		flowCookie.Name,
		"__Host-",
	) {
		t.Fatalf(
			"Flow Cookie不是__Host- Cookie：%s",
			flowCookie.Name,
		)
	}

	if flowCookie.Path != "/" {
		t.Fatalf(
			"Flow Cookie Path=%q",
			flowCookie.Path,
		)
	}

	if flowCookie.Domain != "" {
		t.Fatalf(
			"__Host- Cookie不得设置Domain：%q",
			flowCookie.Domain,
		)
	}

	if !flowCookie.HttpOnly {
		t.Fatal(
			"Flow Cookie必须HttpOnly",
		)
	}

	if !flowCookie.Secure {
		t.Fatal(
			"Flow Cookie必须Secure",
		)
	}

	if flowCookie.SameSite !=
		http.SameSiteLaxMode {
		t.Fatalf(
			"Flow Cookie SameSite=%v",
			flowCookie.SameSite,
		)
	}

	if flowCookie.MaxAge <= 0 {
		t.Fatalf(
			"Flow Cookie MaxAge=%d",
			flowCookie.MaxAge,
		)
	}
}

// TestIdentityAccountLinkHandlerStartBoundary冻结Link/Unlink发起HTTP合同：
// 浏览器提交的任何本地账号参数都不能覆盖JWT claims.UserID。
func TestIdentityAccountLinkHandlerStartBoundary(
	t *testing.T,
) {
	tests := []struct {
		name        string
		path        string
		purpose     string
		handlerCall func(
			*IdentityAccountLinkHandler,
			http.ResponseWriter,
			*http.Request,
		)
	}{
		{
			name: "link",

			path: "/api/v1/auth/identity/link-url",

			purpose: services.IdentityFlowPurposeLink,

			handlerCall: func(
				handler *IdentityAccountLinkHandler,
				w http.ResponseWriter,
				r *http.Request,
			) {
				handler.GetLinkURL(w, r)
			},
		},
		{
			name: "unlink",

			path: "/api/v1/auth/identity/unlink-url",

			purpose: services.IdentityFlowPurposeUnlink,

			handlerCall: func(
				handler *IdentityAccountLinkHandler,
				w http.ResponseWriter,
				r *http.Request,
			) {
				handler.GetUnlinkURL(w, r)
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				fixture :=
					newIdentityHandlerTestFixture(t)

				request := httptest.NewRequest(
					http.MethodGet,
					testCase.path+
						"?local_account_id="+
						identityHandlerTestOtherUserID+
						"&user_id="+
						identityHandlerTestOtherUserID,
					nil,
				)

				request =
					identityHandlerTestRequestWithClaims(
						request,
						identityHandlerTestLocalUserID,
					)

				recorder :=
					httptest.NewRecorder()

				testCase.handlerCall(
					fixture.handler,
					recorder,
					request,
				)

				if recorder.Code != http.StatusOK {
					t.Fatalf(
						"status=%d body=%s",
						recorder.Code,
						recorder.Body.String(),
					)
				}

				if fixture.userRepo.calls != 1 ||
					fixture.userRepo.lastID !=
						identityHandlerTestLocalUserID {
					t.Fatalf(
						"Handler没有坚持claims.UserID：calls=%d id=%s",
						fixture.userRepo.calls,
						fixture.userRepo.lastID,
					)
				}

				if fixture.oidc.startCalls != 1 ||
					fixture.oidc.startPurpose !=
						testCase.purpose ||
					fixture.oidc.startUserID !=
						identityHandlerTestLocalUserID {
					t.Fatalf(
						"OIDC Start输入异常：calls=%d purpose=%s userID=%s",
						fixture.oidc.startCalls,
						fixture.oidc.startPurpose,
						fixture.oidc.startUserID,
					)
				}

				var envelope struct {
					Code int `json:"code"`

					Data map[string]string `json:"data"`
				}

				if err := json.Unmarshal(
					recorder.Body.Bytes(),
					&envelope,
				); err != nil {
					t.Fatalf(
						"响应JSON解析失败：%v body=%s",
						err,
						recorder.Body.String(),
					)
				}

				if envelope.Code != 0 ||
					envelope.Data["authorization_url"] !=
						fixture.oidc.startResult.AuthorizationURL {
					t.Fatalf(
						"authorization响应异常：%s",
						recorder.Body.String(),
					)
				}

				body := recorder.Body.String()

				for _, forbidden := range []string{
					"encrypted-flow-token",
					identityHandlerTestLocalUserID,
					identityHandlerTestGlobalPersonID,
				} {
					if strings.Contains(
						body,
						forbidden,
					) {
						t.Fatalf(
							"响应泄露敏感值：%q",
							forbidden,
						)
					}
				}

				assertIdentityHandlerFlowCookie(
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

// TestIdentityAccountLinkHandlerStartRequiresClaims确认Handler自身不会
// 在缺失可信JWT claims时调用惰性Service Provider。
func TestIdentityAccountLinkHandlerStartRequiresClaims(
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

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/identity/link-url",
		nil,
	)

	recorder :=
		httptest.NewRecorder()

	handler.GetLinkURL(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusUnauthorized {
		t.Fatalf(
			"status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if providerCalls != 0 {
		t.Fatalf(
			"缺失claims时不得取得Service，calls=%d",
			providerCalls,
		)
	}

	assertIdentityHandlerSensitiveHeaders(
		t,
		recorder,
	)
}

// TestIdentityAccountLinkHandlerStartRejectsNonGET确认方法错误在
// Service Provider和任何Identity业务调用之前终止。
func TestIdentityAccountLinkHandlerStartRejectsNonGET(
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

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/identity/unlink-url",
		nil,
	)

	request =
		identityHandlerTestRequestWithClaims(
			request,
			identityHandlerTestLocalUserID,
		)

	recorder :=
		httptest.NewRecorder()

	handler.GetUnlinkURL(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusMethodNotAllowed {
		t.Fatalf(
			"status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if providerCalls != 0 {
		t.Fatalf(
			"非GET不得取得Service，calls=%d",
			providerCalls,
		)
	}

	assertIdentityHandlerSensitiveHeaders(
		t,
		recorder,
	)
}
