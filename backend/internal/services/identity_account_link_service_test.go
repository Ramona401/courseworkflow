package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"tedna/internal/models"
)

const (
	identityAccountServiceTestGlobalID = "11111111-1111-4111-8111-111111111111"

	identityAccountServiceTestLocalID = "22222222-2222-4222-8222-222222222222"

	identityAccountServiceTestOtherLocalID = "33333333-3333-4333-8333-333333333333"
)

type identityAccountServiceTestOIDC struct {
	identityResult IdentityAuthorizationIdentity

	startCalls   int
	startPurpose string
	startUserID  string

	completeCalls int
	completeCode  string
	completeFlow  IdentityAuthorizationFlow
}

func (f *identityAccountServiceTestOIDC) StartAuthorization(
	protector *IdentityFlowProtector,
	purpose string,
	userID string,
) (IdentityAuthorizationStart, error) {
	f.startCalls++
	f.startPurpose = purpose
	f.startUserID = userID

	flow, _, err := protector.NewAuthorizationFlow(
		purpose,
		userID,
	)
	if err != nil {
		return IdentityAuthorizationStart{}, err
	}

	token, err := protector.Seal(flow)
	if err != nil {
		return IdentityAuthorizationStart{}, err
	}

	return IdentityAuthorizationStart{
		AuthorizationURL: "https://identity.example/oauth/authorize",
		State:            flow.State,
		FlowToken:        token,
	}, nil
}

func (f *identityAccountServiceTestOIDC) CompleteAuthorization(
	_ context.Context,
	code string,
	flow IdentityAuthorizationFlow,
) (IdentityAuthorizationIdentity, error) {
	f.completeCalls++
	f.completeCode = code
	f.completeFlow = flow

	return f.identityResult, nil
}

type identityAccountServiceTestBackchannel struct {
	calls          int
	operation      string
	globalPersonID string
	localAccountID string
	traceID        string
	idempotencyKey string

	result IdentityBackchannelResult
	err    error
}

func (f *identityAccountServiceTestBackchannel) Mutate(
	_ context.Context,
	operation string,
	globalPersonID string,
	localAccountID string,
	traceID string,
	idempotencyKey string,
) (IdentityBackchannelResult, error) {
	f.calls++
	f.operation = operation
	f.globalPersonID = globalPersonID
	f.localAccountID = localAccountID
	f.traceID = traceID
	f.idempotencyKey = idempotencyKey

	return f.result, f.err
}

type identityAccountServiceTestUserRepo struct {
	user *models.User
	err  error

	calls  int
	lastID string
}

func (r *identityAccountServiceTestUserRepo) FindByID(
	_ context.Context,
	id string,
) (*models.User, error) {
	r.calls++
	r.lastID = id

	return r.user, r.err
}

type identityAccountServiceTestFixture struct {
	service     *IdentityAccountLinkService
	oidc        *identityAccountServiceTestOIDC
	backchannel *identityAccountServiceTestBackchannel
	userRepo    *identityAccountServiceTestUserRepo
}

func newIdentityAccountServiceTestFixture(
	t *testing.T,
	identity IdentityAuthorizationIdentity,
	backResult IdentityBackchannelResult,
) identityAccountServiceTestFixture {
	t.Helper()

	protector, err := NewIdentityFlowProtector(
		[]byte(
			"tedna-identity-account-service-test-root-secret-0123456789",
		),
	)
	if err != nil {
		t.Fatalf(
			"NewIdentityFlowProtector() error = %v",
			err,
		)
	}

	oidc := &identityAccountServiceTestOIDC{
		identityResult: identity,
	}

	backchannel := &identityAccountServiceTestBackchannel{
		result: backResult,
	}

	userRepo := &identityAccountServiceTestUserRepo{
		user: &models.User{
			ID:     identityAccountServiceTestLocalID,
			Status: models.StatusActive,
		},
	}

	service, err := NewIdentityAccountLinkService(
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

	return identityAccountServiceTestFixture{
		service:     service,
		oidc:        oidc,
		backchannel: backchannel,
		userRepo:    userRepo,
	}
}

func assertIdentityAccountServiceError(
	t *testing.T,
	err error,
	status int,
	code string,
) {
	t.Helper()

	if err == nil {
		t.Fatalf(
			"预期错误%s，实际为nil",
			code,
		)
	}

	var serviceErr *IdentityAccountLinkServiceError

	if !errors.As(err, &serviceErr) {
		t.Fatalf(
			"错误类型异常：%T %v",
			err,
			err,
		)
	}

	if serviceErr.StatusCode != status ||
		serviceErr.Code != code {
		t.Fatalf(
			"错误异常：status=%d code=%s",
			serviceErr.StatusCode,
			serviceErr.Code,
		)
	}
}

func completeIdentityAccountServiceTest(
	t *testing.T,
	fixture identityAccountServiceTestFixture,
	purpose string,
) (
	IdentityAccountLinkCompletionResult,
	error,
) {
	t.Helper()

	start, err := fixture.service.StartAuthorization(
		context.Background(),
		purpose,
		identityAccountServiceTestLocalID,
	)
	if err != nil {
		t.Fatalf(
			"StartAuthorization() error = %v",
			err,
		)
	}

	if start.FlowToken == "" ||
		start.State == "" {
		t.Fatal(
			"StartAuthorization没有生成可信Flow",
		)
	}

	return fixture.service.CompleteAuthorization(
		context.Background(),
		start.FlowToken,
		start.State,
		"authorization-code",
	)
}

// TestIdentityAccountLinkServiceStartUsesCurrentActiveUser冻结发起阶段：
// 本地ID只能来自可信认证上下文，并且生成外部授权前必须重新读取active用户。
func TestIdentityAccountLinkServiceStartUsesCurrentActiveUser(
	t *testing.T,
) {
	fixture := newIdentityAccountServiceTestFixture(
		t,
		IdentityAuthorizationIdentity{},
		IdentityBackchannelResult{},
	)

	start, err := fixture.service.StartAuthorization(
		context.Background(),
		IdentityFlowPurposeLink,
		identityAccountServiceTestLocalID,
	)
	if err != nil {
		t.Fatalf(
			"StartAuthorization() error = %v",
			err,
		)
	}

	if start.FlowToken == "" ||
		start.State == "" {
		t.Fatal(
			"未生成Flow Token或state",
		)
	}

	if fixture.userRepo.calls != 1 ||
		fixture.userRepo.lastID !=
			identityAccountServiceTestLocalID {
		t.Fatalf(
			"本地账号校验异常：calls=%d id=%s",
			fixture.userRepo.calls,
			fixture.userRepo.lastID,
		)
	}

	if fixture.oidc.startCalls != 1 ||
		fixture.oidc.startPurpose !=
			IdentityFlowPurposeLink ||
		fixture.oidc.startUserID !=
			identityAccountServiceTestLocalID {
		t.Fatalf(
			"OIDC Start输入异常：calls=%d purpose=%s userID=%s",
			fixture.oidc.startCalls,
			fixture.oidc.startPurpose,
			fixture.oidc.startUserID,
		)
	}
}

// TestIdentityAccountLinkServiceStartRejectsDisabledUser确认已禁用用户
// 不能依赖仍未过期的本地JWT发起新的Identity关联流程。
func TestIdentityAccountLinkServiceStartRejectsDisabledUser(
	t *testing.T,
) {
	fixture := newIdentityAccountServiceTestFixture(
		t,
		IdentityAuthorizationIdentity{},
		IdentityBackchannelResult{},
	)

	fixture.userRepo.user.Status = "disabled"

	_, err := fixture.service.StartAuthorization(
		context.Background(),
		IdentityFlowPurposeLink,
		identityAccountServiceTestLocalID,
	)

	assertIdentityAccountServiceError(
		t,
		err,
		http.StatusForbidden,
		"ACCOUNT_DISABLED",
	)

	if fixture.oidc.startCalls != 0 ||
		fixture.backchannel.calls != 0 {
		t.Fatal(
			"禁用账号不得启动OIDC或Backchannel",
		)
	}
}

// TestIdentityAccountLinkServiceCallbackRechecksLocalUser确认授权往返期间
// 本地账号状态变化后，callback必须在OIDC完成和Mutation之前重新读库。
func TestIdentityAccountLinkServiceCallbackRechecksLocalUser(
	t *testing.T,
) {
	fixture := newIdentityAccountServiceTestFixture(
		t,
		IdentityAuthorizationIdentity{},
		IdentityBackchannelResult{},
	)

	start, err := fixture.service.StartAuthorization(
		context.Background(),
		IdentityFlowPurposeLink,
		identityAccountServiceTestLocalID,
	)
	if err != nil {
		t.Fatal(err)
	}

	fixture.userRepo.user.Status = "disabled"

	_, err = fixture.service.CompleteAuthorization(
		context.Background(),
		start.FlowToken,
		start.State,
		"authorization-code",
	)

	assertIdentityAccountServiceError(
		t,
		err,
		http.StatusForbidden,
		"ACCOUNT_DISABLED",
	)

	if fixture.userRepo.calls != 2 {
		t.Fatalf(
			"Start与callback应各读库一次，calls=%d",
			fixture.userRepo.calls,
		)
	}

	if fixture.oidc.completeCalls != 0 ||
		fixture.backchannel.calls != 0 {
		t.Fatal(
			"callback发现账号禁用后不得继续OIDC完成或Mutation",
		)
	}
}

// TestIdentityAccountLinkServiceRejectsStateMismatch冻结OIDC callback state。
// state不匹配必须在callback重新读库、OIDC验证和Mutation之前终止。
func TestIdentityAccountLinkServiceRejectsStateMismatch(
	t *testing.T,
) {
	fixture := newIdentityAccountServiceTestFixture(
		t,
		IdentityAuthorizationIdentity{},
		IdentityBackchannelResult{},
	)

	start, err := fixture.service.StartAuthorization(
		context.Background(),
		IdentityFlowPurposeUnlink,
		identityAccountServiceTestLocalID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.service.CompleteAuthorization(
		context.Background(),
		start.FlowToken,
		"wrong-state",
		"authorization-code",
	)

	assertIdentityAccountServiceError(
		t,
		err,
		http.StatusBadRequest,
		"IDENTITY_STATE_MISMATCH",
	)

	if fixture.userRepo.calls != 1 ||
		fixture.oidc.completeCalls != 0 ||
		fixture.backchannel.calls != 0 {
		t.Fatal(
			"state校验失败没有在可信边界及时终止",
		)
	}
}
