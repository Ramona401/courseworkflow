package services

// identity_client.go — TE-DNA作为OIDC Relying Party访问PKU AI Lab Identity Center。
//
// 本文件只承担Phase 1协议编排：
//   - 启动Authorization Code + PKCE S256授权；
//   - 使用Code交换Access Token与ID Token；
//   - 调用严格ID Token验证器；
//   - 使用opaque Access Token读取UserInfo；
//   - 校验sub与platform_link映射事实。
//
// 职责拆分：
//   - AES-GCM Flow、state、nonce、PKCE：identity_flow.go；
//   - 安全HTTP传输：identity_http.go；
//   - Ed25519/JWKS/ID Token验证：identity_token_verifier.go；
//   - Link/Unlink服务器间Mutation：identity_backchannel.go。
//
// Phase 1绝不签发TE-DNA JWT，不建立本地Session，不提前实现Central SSO。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"tedna/internal/config"
)

const (
	identityAuthorizationScope = "openid profile platform_link"

	identityTokenResponseMaxBytes = 16 * 1024
	identityUserInfoMaxBytes      = 16 * 1024

	// Identity Center当前Access Token冻结为300秒。
	// 接受更短的安全生命周期，但拒绝任何超过合同上限的响应。
	identityAccessTokenMaxExpiresIn = 300
)

// IdentityAuthorizationStart 是Handler启动浏览器授权需要的服务器端结果。
//
// FlowToken只能写Secure HttpOnly Cookie，不能进入前端JSON。
type IdentityAuthorizationStart struct {
	AuthorizationURL string
	State            string
	FlowToken        string
	CookieName       string
	ExpiresAt        time.Time
}

// IdentityTokenResponse 是/oauth/token成功响应的最小消费字段。
type IdentityTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope,omitempty"`
}

// IdentityIDTokenClaims 对应Identity Center极简ID Token。
//
// Identity Center不把平台角色、学校、权限或platform_link写入ID Token。
type IdentityIDTokenClaims struct {
	Nonce string `json:"nonce"`
	jwt.RegisteredClaims
}

// IdentityPlatformLink 是UserInfo针对当前可信Client返回的平台映射事实。
type IdentityPlatformLink struct {
	Linked         bool   `json:"linked"`
	LocalAccountID string `json:"local_account_id,omitempty"`
}

// IdentityUserInfo 是TE-DNA实际消费的UserInfo字段。
//
// PlatformLink使用指针，以区分：
//   - 明确返回 {"linked": false}
//   - 服务端根本没有返回platform_link。
type IdentityUserInfo struct {
	Subject      string                `json:"sub"`
	Name         string                `json:"name,omitempty"`
	PlatformLink *IdentityPlatformLink `json:"platform_link"`
}

// IdentityAuthorizationIdentity 是一次OIDC回调完成后的可信身份事实。
//
// GlobalPersonID来自经Ed25519验签的ID Token；
// PlatformLink来自使用同一次opaque Access Token取得的UserInfo。
type IdentityAuthorizationIdentity struct {
	GlobalPersonID string
	Name           string
	PlatformLink   IdentityPlatformLink
}

// IdentityClient 负责Identity Center OIDC协议编排。
type IdentityClient struct {
	cfg        config.IdentityClientConfig
	httpClient *http.Client
	now        func() time.Time
}

// NewIdentityClient 创建OIDC Client。
//
// 生产运行配置由config.LoadIdentityClientConfig执行严格冻结。
// 这里不再次调用cfg.Validate，以允许后续定向测试注入httptest服务器。
func NewIdentityClient(
	cfg config.IdentityClientConfig,
	httpClient *http.Client,
) (*IdentityClient, error) {
	if cfg.Issuer == "" ||
		cfg.Issuer != strings.TrimSpace(cfg.Issuer) {
		return nil, fmt.Errorf(
			"Identity Issuer为空或格式无效",
		)
	}

	if cfg.ClientID == "" ||
		cfg.ClientID != strings.TrimSpace(cfg.ClientID) {
		return nil, fmt.Errorf(
			"Identity Client ID为空或格式无效",
		)
	}

	if cfg.ClientSecret == "" ||
		cfg.ClientSecret != strings.TrimSpace(cfg.ClientSecret) {
		return nil, fmt.Errorf(
			"Identity Client Secret为空或格式无效",
		)
	}

	if cfg.RedirectURI == "" ||
		cfg.RedirectURI != strings.TrimSpace(cfg.RedirectURI) {
		return nil, fmt.Errorf(
			"Identity Redirect URI为空或格式无效",
		)
	}

	switch cfg.TokenAuthMethod {
	case config.IdentityTokenAuthMethodPost,
		config.IdentityTokenAuthMethodBasic:
	default:
		return nil, fmt.Errorf(
			"Identity Token认证方式不受支持：%s",
			cfg.TokenAuthMethod,
		)
	}

	if httpClient == nil {
		httpClient = newIdentitySecureHTTPClient()
	}

	return &IdentityClient{
		cfg:        cfg,
		httpClient: httpClient,
		now:        time.Now,
	}, nil
}

// StartAuthorization 创建Identity Center授权URL和加密Flow Cookie值。
//
// userID必须来自已经通过TE-DNA AuthMiddleware验证的claims.UserID，
// 其语义固定为public.users.id(UUID)。
func (c *IdentityClient) StartAuthorization(
	protector *IdentityFlowProtector,
	purpose string,
	userID string,
) (IdentityAuthorizationStart, error) {
	if c == nil ||
		c.httpClient == nil ||
		c.now == nil {
		return IdentityAuthorizationStart{},
			fmt.Errorf("Identity Client尚未初始化")
	}

	if protector == nil {
		return IdentityAuthorizationStart{},
			fmt.Errorf("Identity Flow Protector不可用")
	}

	flow, challenge, err :=
		protector.NewAuthorizationFlow(
			purpose,
			userID,
		)
	if err != nil {
		return IdentityAuthorizationStart{},
			err
	}

	flowToken, err := protector.Seal(flow)
	if err != nil {
		return IdentityAuthorizationStart{},
			err
	}

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", c.cfg.ClientID)
	values.Set("redirect_uri", c.cfg.RedirectURI)
	values.Set("scope", identityAuthorizationScope)
	values.Set("state", flow.State)
	values.Set("nonce", flow.Nonce)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")

	authorizationURL :=
		c.cfg.AuthorizationEndpoint() +
			"?" +
			values.Encode()

	return IdentityAuthorizationStart{
		AuthorizationURL: authorizationURL,
		State:            flow.State,
		FlowToken:        flowToken,
		CookieName:       IdentityFlowCookieName,
		ExpiresAt: time.Unix(
			flow.IssuedAt,
			0,
		).UTC().Add(IdentityFlowTTL),
	}, nil
}

// CompleteAuthorization 完成Code交换、ID Token验证和UserInfo一致性校验。
func (c *IdentityClient) CompleteAuthorization(
	ctx context.Context,
	code string,
	flow IdentityAuthorizationFlow,
) (IdentityAuthorizationIdentity, error) {
	if ctx == nil {
		return IdentityAuthorizationIdentity{},
			fmt.Errorf("Identity请求Context为空")
	}

	if c == nil ||
		c.httpClient == nil ||
		c.now == nil {
		return IdentityAuthorizationIdentity{},
			fmt.Errorf("Identity Client尚未初始化")
	}

	if err := validateIdentityAuthorizationFlow(
		flow,
		c.now(),
	); err != nil {
		return IdentityAuthorizationIdentity{},
			err
	}

	tokenResponse, err :=
		c.ExchangeAuthorizationCode(
			ctx,
			code,
			flow.CodeVerifier,
		)
	if err != nil {
		return IdentityAuthorizationIdentity{},
			err
	}

	claims, err := c.VerifyIDToken(
		ctx,
		tokenResponse.IDToken,
		flow.Nonce,
	)
	if err != nil {
		return IdentityAuthorizationIdentity{},
			err
	}

	globalPersonID, err :=
		canonicalIdentityGlobalPersonID(
			claims.Subject,
		)
	if err != nil {
		return IdentityAuthorizationIdentity{},
			err
	}

	userInfo, err := c.FetchUserInfo(
		ctx,
		tokenResponse.AccessToken,
	)
	if err != nil {
		return IdentityAuthorizationIdentity{},
			err
	}

	if userInfo.Subject != globalPersonID {
		return IdentityAuthorizationIdentity{},
			fmt.Errorf(
				"Identity UserInfo sub与ID Token sub不一致",
			)
	}

	if userInfo.PlatformLink == nil {
		return IdentityAuthorizationIdentity{},
			fmt.Errorf(
				"Identity UserInfo缺少platform_link",
			)
	}

	if userInfo.PlatformLink.Linked {
		if _, err :=
			canonicalIdentityLocalUserID(
				userInfo.PlatformLink.LocalAccountID,
			); err != nil {
			return IdentityAuthorizationIdentity{},
				fmt.Errorf(
					"Identity UserInfo local_account_id无效：%w",
					err,
				)
		}
	} else if strings.TrimSpace(
		userInfo.PlatformLink.LocalAccountID,
	) != "" {
		return IdentityAuthorizationIdentity{},
			fmt.Errorf(
				"Identity UserInfo linked=false但仍返回local_account_id",
			)
	}

	return IdentityAuthorizationIdentity{
		GlobalPersonID: globalPersonID,
		Name:           userInfo.Name,
		PlatformLink:   *userInfo.PlatformLink,
	}, nil
}

// ExchangeAuthorizationCode 使用PKCE verifier交换Access Token和ID Token。
func (c *IdentityClient) ExchangeAuthorizationCode(
	ctx context.Context,
	code string,
	codeVerifier string,
) (IdentityTokenResponse, error) {
	if ctx == nil {
		return IdentityTokenResponse{},
			fmt.Errorf("Identity请求Context为空")
	}

	if c == nil ||
		c.httpClient == nil {
		return IdentityTokenResponse{},
			fmt.Errorf("Identity Client尚未初始化")
	}

	if code == "" ||
		code != strings.TrimSpace(code) ||
		len(code) > 4096 {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity授权码为空或格式无效",
			)
	}

	if err := validateIdentityFlowRandomValue(
		"code_verifier",
		codeVerifier,
	); err != nil {
		return IdentityTokenResponse{},
			err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	form.Set("code_verifier", codeVerifier)

	switch c.cfg.TokenAuthMethod {
	case config.IdentityTokenAuthMethodPost:
		form.Set(
			"client_id",
			c.cfg.ClientID,
		)
		form.Set(
			"client_secret",
			c.cfg.ClientSecret,
		)

	case config.IdentityTokenAuthMethodBasic:
		form.Set(
			"client_id",
			c.cfg.ClientID,
		)

	default:
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Token认证方式不受支持",
			)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.TokenEndpoint(),
		strings.NewReader(
			form.Encode(),
		),
	)
	if err != nil {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"创建Identity Token请求失败：%w",
				err,
			)
	}

	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	request.Header.Set(
		"Accept",
		"application/json",
	)
	request.Header.Set(
		"Cache-Control",
		"no-store",
	)

	if c.cfg.TokenAuthMethod ==
		config.IdentityTokenAuthMethodBasic {
		request.SetBasicAuth(
			c.cfg.ClientID,
			c.cfg.ClientSecret,
		)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Token请求失败：%w",
				err,
			)
	}
	defer response.Body.Close()

	body, err := readIdentityBoundedResponseBody(
		response.Body,
		identityTokenResponseMaxBytes,
	)
	if err != nil {
		return IdentityTokenResponse{},
			err
	}

	if response.StatusCode != http.StatusOK {
		// 不读取或回显错误正文中的潜在敏感上下文。
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Token Endpoint返回HTTP %d",
				response.StatusCode,
			)
	}

	var tokenResponse IdentityTokenResponse

	if err := json.Unmarshal(
		body,
		&tokenResponse,
	); err != nil {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"解析Identity Token响应失败：%w",
				err,
			)
	}

	if tokenResponse.AccessToken == "" ||
		tokenResponse.AccessToken !=
			strings.TrimSpace(
				tokenResponse.AccessToken,
			) ||
		tokenResponse.IDToken == "" ||
		tokenResponse.IDToken !=
			strings.TrimSpace(
				tokenResponse.IDToken,
			) ||
		!strings.EqualFold(
			tokenResponse.TokenType,
			"Bearer",
		) {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Token响应字段不完整",
			)
	}

	if tokenResponse.ExpiresIn <= 0 ||
		tokenResponse.ExpiresIn >
			identityAccessTokenMaxExpiresIn {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Access Token生命周期异常",
			)
	}

	if tokenResponse.Scope != "" &&
		!identityHasRequiredScopes(
			tokenResponse.Scope,
		) {
		return IdentityTokenResponse{},
			fmt.Errorf(
				"Identity Token响应缺少必要scope",
			)
	}

	return tokenResponse, nil
}

// VerifyIDToken 暴露稳定的OIDC验证入口。
// 具体Ed25519/JWKS逻辑位于identity_token_verifier.go。
func (c *IdentityClient) VerifyIDToken(
	ctx context.Context,
	rawIDToken string,
	expectedNonce string,
) (*IdentityIDTokenClaims, error) {
	return c.verifyIDTokenStrict(
		ctx,
		rawIDToken,
		expectedNonce,
	)
}

// FetchUserInfo 使用Identity不透明Access Token读取当前Client的平台映射事实。
func (c *IdentityClient) FetchUserInfo(
	ctx context.Context,
	accessToken string,
) (IdentityUserInfo, error) {
	if ctx == nil {
		return IdentityUserInfo{},
			fmt.Errorf("Identity请求Context为空")
	}

	if c == nil ||
		c.httpClient == nil {
		return IdentityUserInfo{},
			fmt.Errorf("Identity Client尚未初始化")
	}

	if accessToken == "" ||
		accessToken != strings.TrimSpace(
			accessToken,
		) {
		return IdentityUserInfo{},
			fmt.Errorf(
				"Identity Access Token为空或格式无效",
			)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.cfg.UserInfoEndpoint(),
		nil,
	)
	if err != nil {
		return IdentityUserInfo{},
			fmt.Errorf(
				"创建Identity UserInfo请求失败：%w",
				err,
			)
	}

	request.Header.Set(
		"Authorization",
		"Bearer "+accessToken,
	)
	request.Header.Set(
		"Accept",
		"application/json",
	)
	request.Header.Set(
		"Cache-Control",
		"no-store",
	)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return IdentityUserInfo{},
			fmt.Errorf(
				"Identity UserInfo请求失败：%w",
				err,
			)
	}
	defer response.Body.Close()

	body, err := readIdentityBoundedResponseBody(
		response.Body,
		identityUserInfoMaxBytes,
	)
	if err != nil {
		return IdentityUserInfo{},
			err
	}

	if response.StatusCode != http.StatusOK {
		return IdentityUserInfo{},
			fmt.Errorf(
				"Identity UserInfo返回HTTP %d",
				response.StatusCode,
			)
	}

	var userInfo IdentityUserInfo

	if err := json.Unmarshal(
		body,
		&userInfo,
	); err != nil {
		return IdentityUserInfo{},
			fmt.Errorf(
				"解析Identity UserInfo失败：%w",
				err,
			)
	}

	globalPersonID, err :=
		canonicalIdentityGlobalPersonID(
			userInfo.Subject,
		)
	if err != nil {
		return IdentityUserInfo{},
			fmt.Errorf(
				"Identity UserInfo sub无效：%w",
				err,
			)
	}

	userInfo.Subject = globalPersonID

	return userInfo, nil
}

// canonicalIdentityGlobalPersonID 强制global_person_id使用规范UUID字符串。
func canonicalIdentityGlobalPersonID(
	value string,
) (string, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf(
			"Identity global_person_id不是有效UUID",
		)
	}

	canonical := parsed.String()

	if canonical != value {
		return "", fmt.Errorf(
			"Identity global_person_id必须使用规范UUID格式",
		)
	}

	return canonical, nil
}

// identityHasRequiredScopes 验证Token Endpoint显式返回的scope没有丢失必要权限。
func identityHasRequiredScopes(
	scope string,
) bool {
	required := map[string]bool{
		"openid":        false,
		"profile":       false,
		"platform_link": false,
	}

	for _, item := range strings.Fields(scope) {
		if _, exists := required[item]; exists {
			required[item] = true
		}
	}

	for _, present := range required {
		if !present {
			return false
		}
	}

	return true
}
