package services

// assistant_runtime_token.go
//
// 本文件实现教学智能体短时运行JWT。
//
// 与教师登录JWT的隔离方式：
//   - 独立issuer；
//   - 独立audience；
//   - 独立subject；
//   - 固定token_use；
//   - 只允许HS256；
//   - 使用JWT_SECRET经HMAC用途派生后的独立32字节签名密钥。
//
// 即使基础密钥相同，登录JWT也无法通过运行令牌验证；
// 运行令牌同样不能进入现有教师AuthMiddleware。
//
// 令牌不包含提示词、模型配置、教师JWT、学校ID或计费账户ID。

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"tedna/internal/models"
)

const (
	assistantRuntimeTokenIssuer = "tedna-assistant-runtime"

	assistantRuntimeTokenAudience = "tedna-assistant-runtime-api"

	assistantRuntimeTokenSubject = "assistant-runtime-session"

	assistantRuntimeTokenUse = "assistant_runtime"

	assistantRuntimeSigningPurpose = "tedna:v1:assistant-runtime:jwt-signing"

	assistantRuntimePrivacyPurpose = "tedna:v1:assistant-runtime:privacy-hashing"

	assistantRuntimeTokenMinTTL = 5 * time.Minute

	assistantRuntimeTokenMaxTTL = 15 * time.Minute
)

var (
	ErrAssistantRuntimeTokenConfiguration = errors.New(
		"教学智能体运行令牌配置无效",
	)

	ErrAssistantRuntimeTokenInvalid = errors.New(
		"教学智能体运行令牌无效",
	)

	ErrAssistantRuntimeTokenExpired = errors.New(
		"教学智能体运行令牌已过期",
	)
)

// AssistantRuntimeTokenClaims 是短时运行令牌声明。
type AssistantRuntimeTokenClaims struct {
	SessionID         string `json:"session_id"`
	DeploymentID      string `json:"deployment_id"`
	DeploymentVersion int    `json:"deployment_version"`
	SessionKind       string `json:"session_kind"`
	TokenUse          string `json:"token_use"`

	jwt.RegisteredClaims
}

// AssistantRuntimeTokenService 负责运行令牌签发和密码学解析。
type AssistantRuntimeTokenService struct {
	signingKey []byte
	ttl        time.Duration
}

// newAssistantRuntimeTokenService 创建独立用途令牌服务。
func newAssistantRuntimeTokenService(
	baseSecret string,
	ttl time.Duration,
) *AssistantRuntimeTokenService {
	service := &AssistantRuntimeTokenService{
		ttl: ttl,
	}

	if strings.TrimSpace(baseSecret) != "" {
		service.signingKey =
			deriveAssistantRuntimeKey(
				baseSecret,
				assistantRuntimeSigningPurpose,
			)
	}

	return service
}

// deriveAssistantRuntimeKey 使用HMAC用途派生32字节密钥。
func deriveAssistantRuntimeKey(
	secret string,
	purpose string,
) []byte {
	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)
	_, _ = mac.Write(
		[]byte(purpose),
	)

	return mac.Sum(nil)
}

// configured 检查签名密钥和TTL。
func (s *AssistantRuntimeTokenService) configured() bool {
	return s != nil &&
		len(s.signingKey) == sha256.Size &&
		s.ttl >= assistantRuntimeTokenMinTTL &&
		s.ttl <= assistantRuntimeTokenMaxTTL
}

// TTL 返回配置的短时有效期。
func (s *AssistantRuntimeTokenService) TTL() time.Duration {
	if s == nil {
		return 0
	}

	return s.ttl
}

// Issue 签发一个绑定会话、部署、版本和会话类型的短时令牌。
func (s *AssistantRuntimeTokenService) Issue(
	sessionID string,
	deploymentID string,
	deploymentVersion int,
	sessionKind string,
	jti string,
	expiresAt time.Time,
) (
	string,
	error,
) {
	if !s.configured() {
		return "",
			ErrAssistantRuntimeTokenConfiguration
	}

	now := time.Now().UTC()

	sessionID = strings.TrimSpace(sessionID)
	deploymentID = strings.TrimSpace(deploymentID)
	sessionKind = strings.TrimSpace(sessionKind)
	jti = strings.TrimSpace(jti)
	expiresAt = expiresAt.UTC()

	if sessionID == "" ||
		deploymentID == "" ||
		deploymentVersion <= 0 ||
		!models.IsValidAssistantRuntimeSessionKind(
			sessionKind,
		) ||
		jti == "" ||
		!expiresAt.After(now) ||
		expiresAt.After(now.Add(s.ttl)) {
		return "",
			ErrAssistantRuntimeTokenInvalid
	}

	claims := &AssistantRuntimeTokenClaims{
		SessionID:         sessionID,
		DeploymentID:      deploymentID,
		DeploymentVersion: deploymentVersion,
		SessionKind:       sessionKind,
		TokenUse:          assistantRuntimeTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    assistantRuntimeTokenIssuer,
			Subject:   assistantRuntimeTokenSubject,
			Audience:  jwt.ClaimStrings{assistantRuntimeTokenAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signed, err := token.SignedString(
		s.signingKey,
	)
	if err != nil {
		return "",
			fmt.Errorf(
				"签发教学智能体运行令牌失败: %w",
				err,
			)
	}

	return signed, nil
}

// Parse 验证签名、时效、用途和业务声明。
func (s *AssistantRuntimeTokenService) Parse(
	tokenString string,
) (
	*AssistantRuntimeTokenClaims,
	error,
) {
	if !s.configured() {
		return nil,
			ErrAssistantRuntimeTokenConfiguration
	}

	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil,
			ErrAssistantRuntimeTokenInvalid
	}

	claims := &AssistantRuntimeTokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method !=
				jwt.SigningMethodHS256 {
				return nil,
					ErrAssistantRuntimeTokenInvalid
			}

			return s.signingKey, nil
		},
		jwt.WithValidMethods(
			[]string{
				jwt.SigningMethodHS256.Alg(),
			},
		),
	)
	if err != nil {
		if errors.Is(
			err,
			jwt.ErrTokenExpired,
		) {
			return nil,
				ErrAssistantRuntimeTokenExpired
		}

		return nil,
			ErrAssistantRuntimeTokenInvalid
	}

	if token == nil ||
		!token.Valid ||
		claims == nil {
		return nil,
			ErrAssistantRuntimeTokenInvalid
	}

	now := time.Now().UTC()

	if claims.Issuer !=
		assistantRuntimeTokenIssuer ||
		claims.Subject !=
			assistantRuntimeTokenSubject ||
		!assistantRuntimeAudienceContains(
			claims.Audience,
			assistantRuntimeTokenAudience,
		) ||
		claims.TokenUse !=
			assistantRuntimeTokenUse ||
		strings.TrimSpace(
			claims.SessionID,
		) == "" ||
		strings.TrimSpace(
			claims.DeploymentID,
		) == "" ||
		claims.DeploymentVersion <= 0 ||
		!models.IsValidAssistantRuntimeSessionKind(
			claims.SessionKind,
		) ||
		strings.TrimSpace(
			claims.ID,
		) == "" ||
		claims.ExpiresAt == nil ||
		!claims.ExpiresAt.Time.After(now) ||
		claims.IssuedAt == nil ||
		claims.IssuedAt.Time.After(
			now.Add(time.Minute),
		) ||
		(claims.NotBefore != nil &&
			now.Before(claims.NotBefore.Time)) {
		if claims.ExpiresAt != nil &&
			!claims.ExpiresAt.Time.After(now) {
			return nil,
				ErrAssistantRuntimeTokenExpired
		}

		return nil,
			ErrAssistantRuntimeTokenInvalid
	}

	return claims, nil
}

// assistantRuntimeAudienceContains 检查固定受众。
func assistantRuntimeAudienceContains(
	audiences jwt.ClaimStrings,
	expected string,
) bool {
	for _, audience := range audiences {
		if audience == expected {
			return true
		}
	}

	return false
}
