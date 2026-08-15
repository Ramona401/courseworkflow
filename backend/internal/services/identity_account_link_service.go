package services

// identity_account_link_service.go — TE-DNA Identity Account Linking业务编排。
//
// Phase 1边界：
//   - 只允许已经登录TE-DNA的本地账号主动Link/Unlink；
//   - local_account_id固定使用public.users.id规范UUID；
//   - global_person_id只存在本次可信OIDC/Backchannel内存上下文，不写TE-DNA数据库；
//   - Link/Unlink前后都重新读取本地用户状态，不能只相信尚未过期的JWT；
//   - 不实现匿名Identity登录，不签发TE-DNA JWT，不建立本地Session；
//   - Central SSO属于后续阶段。

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// identityAccountLinkOIDCClient 抽出业务编排真正依赖的OIDC能力。
type identityAccountLinkOIDCClient interface {
	StartAuthorization(
		protector *IdentityFlowProtector,
		purpose string,
		userID string,
	) (IdentityAuthorizationStart, error)

	CompleteAuthorization(
		ctx context.Context,
		code string,
		flow IdentityAuthorizationFlow,
	) (IdentityAuthorizationIdentity, error)
}

// identityAccountLinkBackchannelClient 是Link/Unlink最小服务器间Mutation能力。
type identityAccountLinkBackchannelClient interface {
	Mutate(
		ctx context.Context,
		operation string,
		globalPersonID string,
		localAccountID string,
		traceID string,
		idempotencyKey string,
	) (IdentityBackchannelResult, error)
}

// identityAccountLinkUserRepository 只暴露重新确认本地账号所需能力。
type identityAccountLinkUserRepository interface {
	FindByID(
		ctx context.Context,
		id string,
	) (*models.User, error)
}

type identityAccountLinkLiveUserRepository struct{}

func (
	identityAccountLinkLiveUserRepository,
) FindByID(
	ctx context.Context,
	id string,
) (*models.User, error) {
	return repository.FindUserByID(ctx, id)
}

// IdentityAccountLinkCompletionResult 是callback最终允许向HTTP层暴露的最小结果。
//
// 不包含global_person_id、local_account_id、Event ID、Link ID、Nonce或Idempotency Key。
type IdentityAccountLinkCompletionResult struct {
	Operation string
	State     string
	Changed   bool
}

// IdentityAccountLinkServiceError 是本模块安全、稳定的业务错误。
//
// Handler只能向浏览器暴露Code和Message，不能透传上游原始错误正文。
type IdentityAccountLinkServiceError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *IdentityAccountLinkServiceError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func newIdentityAccountLinkServiceError(
	statusCode int,
	code string,
	message string,
) *IdentityAccountLinkServiceError {
	return &IdentityAccountLinkServiceError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// IdentityAccountLinkService 负责已登录TE-DNA用户的Link/Unlink完整业务顺序。
type IdentityAccountLinkService struct {
	oidc        identityAccountLinkOIDCClient
	backchannel identityAccountLinkBackchannelClient
	protector   *IdentityFlowProtector
	userRepo    identityAccountLinkUserRepository
}

// NewIdentityAccountLinkService 创建可注入、可定向测试的业务Service。
func NewIdentityAccountLinkService(
	oidc identityAccountLinkOIDCClient,
	backchannel identityAccountLinkBackchannelClient,
	protector *IdentityFlowProtector,
	userRepo identityAccountLinkUserRepository,
) (*IdentityAccountLinkService, error) {
	if oidc == nil {
		return nil, fmt.Errorf(
			"Identity OIDC Client不能为空",
		)
	}

	if backchannel == nil {
		return nil, fmt.Errorf(
			"Identity Backchannel Client不能为空",
		)
	}

	if protector == nil {
		return nil, fmt.Errorf(
			"Identity Flow Protector不能为空",
		)
	}

	if userRepo == nil {
		return nil, fmt.Errorf(
			"Identity User Repository不能为空",
		)
	}

	return &IdentityAccountLinkService{
		oidc:        oidc,
		backchannel: backchannel,
		protector:   protector,
		userRepo:    userRepo,
	}, nil
}

// StartAuthorization 为当前已经通过TE-DNA AuthMiddleware认证的账号发起Link/Unlink。
//
// userID必须取可信claims.UserID，Handler不得接受query/body中的本地账号ID。
func (s *IdentityAccountLinkService) StartAuthorization(
	ctx context.Context,
	purpose string,
	userID string,
) (IdentityAuthorizationStart, error) {
	if ctx == nil {
		return IdentityAuthorizationStart{},
			newIdentityAccountLinkServiceError(
				http.StatusInternalServerError,
				"IDENTITY_CONTEXT_INVALID",
				"Identity账号关联请求上下文无效",
			)
	}

	if s == nil {
		return IdentityAuthorizationStart{},
			newIdentityAccountLinkServiceError(
				http.StatusServiceUnavailable,
				"IDENTITY_NOT_INITIALIZED",
				"Identity账号关联服务暂时不可用",
			)
	}

	if purpose != IdentityFlowPurposeLink &&
		purpose != IdentityFlowPurposeUnlink {
		return IdentityAuthorizationStart{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_INVALID_OPERATION",
				"不支持的Identity账号关联操作",
			)
	}

	user, err := s.loadEligibleLocalUser(
		ctx,
		userID,
	)
	if err != nil {
		return IdentityAuthorizationStart{}, err
	}

	start, err := s.oidc.StartAuthorization(
		s.protector,
		purpose,
		user.ID,
	)
	if err != nil {
		return IdentityAuthorizationStart{},
			newIdentityAccountLinkServiceError(
				http.StatusInternalServerError,
				"IDENTITY_FLOW_FAILED",
				"创建Identity授权流程失败",
			)
	}

	return start, nil
}

// CompleteAuthorization 消费加密Flow、完成OIDC验证并执行Link或Unlink。
//
// flowToken只能来自Secure HttpOnly Cookie；callbackState和code来自OIDC callback。
// 本地userID只能从Flow内部恢复，浏览器无法替换。
func (s *IdentityAccountLinkService) CompleteAuthorization(
	ctx context.Context,
	flowToken string,
	callbackState string,
	code string,
) (IdentityAccountLinkCompletionResult, error) {
	if ctx == nil {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusInternalServerError,
				"IDENTITY_CONTEXT_INVALID",
				"Identity账号关联请求上下文无效",
			)
	}

	if s == nil {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusServiceUnavailable,
				"IDENTITY_NOT_INITIALIZED",
				"Identity账号关联服务暂时不可用",
			)
	}

	if strings.TrimSpace(flowToken) == "" ||
		strings.TrimSpace(callbackState) == "" ||
		strings.TrimSpace(code) == "" {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_CALLBACK_INVALID",
				"Identity授权回调参数无效",
			)
	}

	flow, err := s.protector.Open(flowToken)
	if err != nil {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_FLOW_INVALID",
				"Identity授权流程无效或已过期",
			)
	}

	if !hmac.Equal(
		[]byte(flow.State),
		[]byte(callbackState),
	) {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_STATE_MISMATCH",
				"Identity授权状态校验失败",
			)
	}

	if flow.Purpose != IdentityFlowPurposeLink &&
		flow.Purpose != IdentityFlowPurposeUnlink {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_INVALID_OPERATION",
				"当前Identity授权流程不属于账号关联操作",
			)
	}

	// 浏览器完成Identity授权期间，本地账号可能已被管理员禁用或删除。
	// callback必须重新读库确认，不能沿用发起时状态。
	user, err := s.loadEligibleLocalUser(
		ctx,
		flow.UserID,
	)
	if err != nil {
		return IdentityAccountLinkCompletionResult{}, err
	}

	identity, err := s.oidc.CompleteAuthorization(
		ctx,
		code,
		flow,
	)
	if err != nil {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadGateway,
				"IDENTITY_UPSTREAM_ERROR",
				"Identity Center授权验证失败",
			)
	}

	switch flow.Purpose {
	case IdentityFlowPurposeLink:
		return s.completeIdentityLink(
			ctx,
			user,
			identity,
		)

	case IdentityFlowPurposeUnlink:
		return s.completeIdentityUnlink(
			ctx,
			user,
			identity,
		)

	default:
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadRequest,
				"IDENTITY_INVALID_OPERATION",
				"不支持的Identity账号关联操作",
			)
	}
}

func (s *IdentityAccountLinkService) completeIdentityLink(
	ctx context.Context,
	user *models.User,
	identity IdentityAuthorizationIdentity,
) (IdentityAccountLinkCompletionResult, error) {
	if identity.PlatformLink.Linked {
		if identity.PlatformLink.LocalAccountID ==
			user.ID {
			// UserInfo已经证明当前自然人精确关联当前TE-DNA账号。
			// 不重复制造Mutation、Audit或Outbox。
			return IdentityAccountLinkCompletionResult{
				Operation: IdentityBackchannelOperationLink,
				State:     "linked",
				Changed:   false,
			}, nil
		}

		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusConflict,
				"IDENTITY_LINK_CONFLICT",
				"该Identity身份已经关联其他TE-DNA账号",
			)
	}

	result, err := s.backchannel.Mutate(
		ctx,
		IdentityBackchannelOperationLink,
		identity.GlobalPersonID,
		user.ID,
		"",
		"",
	)
	if err != nil {
		return IdentityAccountLinkCompletionResult{},
			mapIdentityAccountLinkBackchannelError(err)
	}

	if result.Outcome == "conflict" {
		return IdentityAccountLinkCompletionResult{},
			mapIdentityAccountLinkConflict(
				result.ReasonCode,
			)
	}

	if result.Outcome != "success" ||
		result.State != "linked" {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadGateway,
				"IDENTITY_PROTOCOL_ERROR",
				"Identity Center返回了无效的账号关联结果",
			)
	}

	return IdentityAccountLinkCompletionResult{
		Operation: IdentityBackchannelOperationLink,
		State:     "linked",
		Changed:   !result.IdempotentReplay,
	}, nil
}

func (s *IdentityAccountLinkService) completeIdentityUnlink(
	ctx context.Context,
	user *models.User,
	identity IdentityAuthorizationIdentity,
) (IdentityAccountLinkCompletionResult, error) {
	if !identity.PlatformLink.Linked {
		// UserInfo已确认当前Client下没有active关联。
		// 幂等终态不创建虚假Unlink事件。
		return IdentityAccountLinkCompletionResult{
			Operation: IdentityBackchannelOperationUnlink,
			State:     "unlinked",
			Changed:   false,
		}, nil
	}

	if identity.PlatformLink.LocalAccountID != user.ID {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusConflict,
				"IDENTITY_LINK_CONFLICT",
				"该Identity身份关联的不是当前TE-DNA账号",
			)
	}

	result, err := s.backchannel.Mutate(
		ctx,
		IdentityBackchannelOperationUnlink,
		identity.GlobalPersonID,
		user.ID,
		"",
		"",
	)
	if err != nil {
		return IdentityAccountLinkCompletionResult{},
			mapIdentityAccountLinkBackchannelError(err)
	}

	if result.Outcome == "conflict" {
		return IdentityAccountLinkCompletionResult{},
			mapIdentityAccountLinkConflict(
				result.ReasonCode,
			)
	}

	if result.Outcome != "success" ||
		result.State != "unlinked" {
		return IdentityAccountLinkCompletionResult{},
			newIdentityAccountLinkServiceError(
				http.StatusBadGateway,
				"IDENTITY_PROTOCOL_ERROR",
				"Identity Center返回了无效的账号解绑结果",
			)
	}

	return IdentityAccountLinkCompletionResult{
		Operation: IdentityBackchannelOperationUnlink,
		State:     "unlinked",
		Changed:   !result.IdempotentReplay,
	}, nil
}

// loadEligibleLocalUser 重新确认TE-DNA账号存在、启用并保持规范UUID主键。
func (s *IdentityAccountLinkService) loadEligibleLocalUser(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	canonicalID, err :=
		canonicalIdentityLocalUserID(userID)
	if err != nil {
		return nil,
			newIdentityAccountLinkServiceError(
				http.StatusUnauthorized,
				"UNAUTHORIZED",
				"当前登录身份无效",
			)
	}

	user, err := s.userRepo.FindByID(
		ctx,
		canonicalID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserNotFound,
		) {
			return nil,
				newIdentityAccountLinkServiceError(
					http.StatusForbidden,
					"ACCOUNT_UNAVAILABLE",
					"当前TE-DNA账号不可用",
				)
		}

		return nil,
			newIdentityAccountLinkServiceError(
				http.StatusInternalServerError,
				"DATABASE_ERROR",
				"读取TE-DNA账号失败",
			)
	}

	if user == nil ||
		user.Status != models.StatusActive {
		return nil,
			newIdentityAccountLinkServiceError(
				http.StatusForbidden,
				"ACCOUNT_DISABLED",
				"当前TE-DNA账号不可用",
			)
	}

	if user.ID != canonicalID {
		return nil,
			newIdentityAccountLinkServiceError(
				http.StatusInternalServerError,
				"IDENTITY_LOCAL_ACCOUNT_INVALID",
				"TE-DNA账号Identity标识无效",
			)
	}

	return user, nil
}

func mapIdentityAccountLinkBackchannelError(
	err error,
) error {
	var protocolErr *IdentityBackchannelError

	if errors.As(err, &protocolErr) {
		switch {
		case protocolErr.HTTPStatus ==
			http.StatusUnauthorized,
			protocolErr.Code == "invalid_client":
			return newIdentityAccountLinkServiceError(
				http.StatusServiceUnavailable,
				"IDENTITY_UNAVAILABLE",
				"Identity账号关联服务暂时不可用",
			)

		case protocolErr.HTTPStatus ==
			http.StatusServiceUnavailable,
			protocolErr.Code ==
				"temporarily_unavailable":
			return newIdentityAccountLinkServiceError(
				http.StatusServiceUnavailable,
				"IDENTITY_UNAVAILABLE",
				"Identity账号关联服务暂时不可用",
			)

		case protocolErr.Code ==
			"person_unavailable":
			return newIdentityAccountLinkServiceError(
				http.StatusConflict,
				"IDENTITY_PERSON_UNAVAILABLE",
				"Identity身份状态已经变化，请重新授权后再试",
			)

		case protocolErr.Code ==
			"idempotency_conflict",
			protocolErr.Code ==
				"replay_detected":
			return newIdentityAccountLinkServiceError(
				http.StatusBadGateway,
				"IDENTITY_PROTOCOL_ERROR",
				"Identity账号关联协议状态异常，请重新发起操作",
			)
		}
	}

	return newIdentityAccountLinkServiceError(
		http.StatusBadGateway,
		"IDENTITY_UPSTREAM_ERROR",
		"Identity账号关联请求失败",
	)
}

func mapIdentityAccountLinkConflict(
	reasonCode string,
) error {
	switch reasonCode {
	case "bidirectional_link_conflict",
		"global_person_already_linked",
		"local_account_already_linked":
		return newIdentityAccountLinkServiceError(
			http.StatusConflict,
			"IDENTITY_LINK_CONFLICT",
			"该Identity身份或TE-DNA账号已经存在其他有效关联",
		)

	case "global_person_link_mismatch",
		"local_account_link_mismatch",
		"link_not_found":
		return newIdentityAccountLinkServiceError(
			http.StatusConflict,
			"IDENTITY_LINK_STATE_CHANGED",
			"Identity关联状态已经变化，请重新授权后再试",
		)

	default:
		return newIdentityAccountLinkServiceError(
			http.StatusConflict,
			"IDENTITY_LINK_CONFLICT",
			"Identity账号关联发生冲突",
		)
	}
}

// IdentityAccountLinkServiceProvider 是Handler使用的惰性生产Service提供器。
//
// Identity Client Secret未配置时不阻断TE-DNA现有密码登录和其它业务。
// 第一次访问Identity接口时才初始化；初始化失败在当前进程生命周期内保持失败，
// 修复环境变量后通过正常服务重启重新初始化。
type IdentityAccountLinkServiceProvider func() (
	*IdentityAccountLinkService,
	error,
)

// NewIdentityAccountLinkServiceProvider 创建进程级惰性Identity运行时。
//
// rootSecret来自已经加载并验证过的TE-DNA JWT_SECRET。
// 使用独立拷贝，仅用于领域隔离派生Identity Flow AES-GCM密钥。
func NewIdentityAccountLinkServiceProvider(
	rootSecret string,
) IdentityAccountLinkServiceProvider {
	rootSecretCopy := append(
		[]byte(nil),
		[]byte(rootSecret)...,
	)

	var (
		once       sync.Once
		service    *IdentityAccountLinkService
		serviceErr error
	)

	return func() (
		*IdentityAccountLinkService,
		error,
	) {
		once.Do(
			func() {
				identityCfg, err :=
					config.LoadIdentityClientConfig()
				if err != nil {
					serviceErr = err
					return
				}

				protector, err :=
					NewIdentityFlowProtector(
						rootSecretCopy,
					)
				if err != nil {
					serviceErr = err
					return
				}

				oidcClient, err :=
					NewIdentityClient(
						identityCfg,
						nil,
					)
				if err != nil {
					serviceErr = err
					return
				}

				backchannelClient, err :=
					NewIdentityBackchannelClient(
						identityCfg,
						nil,
					)
				if err != nil {
					serviceErr = err
					return
				}

				service, serviceErr =
					NewIdentityAccountLinkService(
						oidcClient,
						backchannelClient,
						protector,
						identityAccountLinkLiveUserRepository{},
					)
			},
		)

		return service, serviceErr
	}
}
