package config

// identity_client.go — TE-DNA连接PKU AI Lab Identity Center的运行配置。
//
// Phase 1边界：
//   - TE-DNA仍独立管理本地密码、JWT、角色和业务权限；
//   - 本配置只服务Identity Account Linking；
//   - 不在本地保存global_person_id或跨平台mapping事实；
//   - Central SSO属于后续独立阶段，不在这里提前引入本地登录语义。

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultIdentityIssuer          = "https://id.pkuailab.com"
	DefaultIdentityClientID        = "tedna-client"
	DefaultIdentityRedirectURI     = "https://workflow.pkuailab.com/api/v1/auth/identity/callback"
	DefaultIdentityTokenAuthMethod = "client_secret_post"

	IdentityTokenAuthMethodPost  = "client_secret_post"
	IdentityTokenAuthMethodBasic = "client_secret_basic"
)

// IdentityClientConfig 是TE-DNA作为Identity Center OIDC Client的最小运行配置。
//
// ClientSecret仅存在进程内存与受保护环境变量中，不得进入日志、URL、前端响应或数据库明文字段。
type IdentityClientConfig struct {
	Issuer          string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	TokenAuthMethod string
}

// AuthorizationEndpoint 返回Identity Center Authorization Endpoint。
func (c IdentityClientConfig) AuthorizationEndpoint() string {
	return strings.TrimRight(c.Issuer, "/") + "/oauth/authorize"
}

// TokenEndpoint 返回Authorization Code交换端点。
func (c IdentityClientConfig) TokenEndpoint() string {
	return strings.TrimRight(c.Issuer, "/") + "/oauth/token"
}

// UserInfoEndpoint 返回OIDC UserInfo端点。
func (c IdentityClientConfig) UserInfoEndpoint() string {
	return strings.TrimRight(c.Issuer, "/") + "/oauth/userinfo"
}

// JWKSURI 返回Identity Center公开Ed25519 JWKS端点。
func (c IdentityClientConfig) JWKSURI() string {
	return strings.TrimRight(c.Issuer, "/") + "/.well-known/jwks.json"
}

// BackchannelEndpoint 返回平台账号Link/Unlink服务器间接口。
func (c IdentityClientConfig) BackchannelEndpoint() string {
	return strings.TrimRight(c.Issuer, "/") +
		"/backchannel/platform-account-links"
}

// LoadIdentityClientConfig 从环境变量加载Identity Client配置。
//
// 支持：
//   - IDENTITY_ISSUER_URL
//   - IDENTITY_CLIENT_ID
//   - IDENTITY_CLIENT_SECRET
//   - IDENTITY_REDIRECT_URI
//   - IDENTITY_TOKEN_AUTH_METHOD
//
// Client Secret没有默认值。缺失时只让Identity能力fail-closed，
// 不应影响TE-DNA现有本地登录在进程启动时继续工作。
func LoadIdentityClientConfig() (IdentityClientConfig, error) {
	rawSecret := os.Getenv("IDENTITY_CLIENT_SECRET")

	if rawSecret != strings.TrimSpace(rawSecret) {
		return IdentityClientConfig{},
			fmt.Errorf("IDENTITY_CLIENT_SECRET不得包含首尾空白")
	}

	cfg := IdentityClientConfig{
		Issuer: envOrIdentityDefault(
			"IDENTITY_ISSUER_URL",
			DefaultIdentityIssuer,
		),
		ClientID: envOrIdentityDefault(
			"IDENTITY_CLIENT_ID",
			DefaultIdentityClientID,
		),
		ClientSecret: rawSecret,
		RedirectURI: envOrIdentityDefault(
			"IDENTITY_REDIRECT_URI",
			DefaultIdentityRedirectURI,
		),
		TokenAuthMethod: envOrIdentityDefault(
			"IDENTITY_TOKEN_AUTH_METHOD",
			DefaultIdentityTokenAuthMethod,
		),
	}

	cfg.Issuer = strings.TrimRight(
		strings.TrimSpace(cfg.Issuer),
		"/",
	)
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.RedirectURI = strings.TrimSpace(cfg.RedirectURI)
	cfg.TokenAuthMethod = strings.TrimSpace(
		cfg.TokenAuthMethod,
	)

	if err := cfg.Validate(); err != nil {
		return IdentityClientConfig{}, err
	}

	return cfg, nil
}

// Validate 对Identity配置执行fail-closed校验。
func (c IdentityClientConfig) Validate() error {
	if c.Issuer == "" {
		return fmt.Errorf("IDENTITY_ISSUER_URL不能为空")
	}

	if c.ClientID == "" {
		return fmt.Errorf("IDENTITY_CLIENT_ID不能为空")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("IDENTITY_CLIENT_SECRET未配置")
	}

	if len(c.ClientSecret) < 32 ||
		len(c.ClientSecret) > 512 {
		return fmt.Errorf("IDENTITY_CLIENT_SECRET长度无效")
	}

	if c.RedirectURI == "" {
		return fmt.Errorf("IDENTITY_REDIRECT_URI不能为空")
	}

	if c.TokenAuthMethod != IdentityTokenAuthMethodPost &&
		c.TokenAuthMethod != IdentityTokenAuthMethodBasic {
		return fmt.Errorf(
			"IDENTITY_TOKEN_AUTH_METHOD只允许%s或%s",
			IdentityTokenAuthMethodPost,
			IdentityTokenAuthMethodBasic,
		)
	}

	if err := validateIdentityHTTPSURL(
		"IDENTITY_ISSUER_URL",
		c.Issuer,
		false,
	); err != nil {
		return err
	}

	if err := validateIdentityHTTPSURL(
		"IDENTITY_REDIRECT_URI",
		c.RedirectURI,
		true,
	); err != nil {
		return err
	}

	// Phase 1生产合同冻结在唯一可信Identity Hub和唯一TE-DNA Client。
	// 环境变量保留用于显式配置与运维校验，但不能把生产进程静默指向其他Issuer/Client。
	if c.Issuer != DefaultIdentityIssuer {
		return fmt.Errorf(
			"IDENTITY_ISSUER_URL必须为%s",
			DefaultIdentityIssuer,
		)
	}

	if c.ClientID != DefaultIdentityClientID {
		return fmt.Errorf(
			"IDENTITY_CLIENT_ID必须为%s",
			DefaultIdentityClientID,
		)
	}

	if c.RedirectURI != DefaultIdentityRedirectURI {
		return fmt.Errorf(
			"IDENTITY_REDIRECT_URI必须为%s",
			DefaultIdentityRedirectURI,
		)
	}

	return nil
}

func envOrIdentityDefault(
	key string,
	defaultValue string,
) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	return value
}

func validateIdentityHTTPSURL(
	name string,
	rawURL string,
	allowPath bool,
) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s格式无效：%w", name, err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("%s必须使用https", name)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%s必须包含host", name)
	}

	if parsed.User != nil {
		return fmt.Errorf("%s不得包含userinfo", name)
	}

	if parsed.RawQuery != "" {
		return fmt.Errorf("%s不得包含query", name)
	}

	if parsed.Fragment != "" {
		return fmt.Errorf("%s不得包含fragment", name)
	}

	if !allowPath &&
		parsed.Path != "" &&
		parsed.Path != "/" {
		return fmt.Errorf("%s不得包含额外path", name)
	}

	return nil
}
