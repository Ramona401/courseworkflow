package services

// identity_token_verifier.go — Identity Center ID Token与JWKS严格验证。
//
// 职责边界：
//   - 只负责Identity签发的Ed25519/EdDSA ID Token真实性与声明验证；
//   - JWKS只接受OKP/Ed25519签名公钥；
//   - 严格校验kid、iss、aud、exp、iat、nonce、sub；
//   - global_person_id必须是规范UUID；
//   - 不消费platform_link，不执行Link/Unlink，不签发TE-DNA JWT或Session。
//
// HTTP安全策略复用identity_http.go：短超时、禁止自动Redirect、响应硬大小限制。

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	identityVerifierJWKSMaxBytes = 64 * 1024

	// Identity Center生产ID Token生命周期冻结为300秒。
	identityVerifierMaxTokenLifetime = 5 * time.Minute

	// 仅用于网络和主机时钟的微小漂移容忍，不扩大Token自身允许生命周期。
	identityVerifierClockSkew = 30 * time.Second
)

type identityVerifierJWKS struct {
	Keys []identityVerifierJWK `json:"keys"`
}

type identityVerifierJWK struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	KID string `json:"kid"`
	X   string `json:"x"`
	Use string `json:"use,omitempty"`
	Alg string `json:"alg,omitempty"`
}

// verifyIDTokenStrict 使用Identity Center JWKS完整验证一次ID Token。
//
// expectedNonce必须来自当前AES-GCM保护的Identity Flow，
// 不能由callback query/body直接提供或替换。
func (c *IdentityClient) verifyIDTokenStrict(
	ctx context.Context,
	rawIDToken string,
	expectedNonce string,
) (*IdentityIDTokenClaims, error) {
	if c == nil ||
		c.httpClient == nil ||
		c.now == nil {
		return nil, fmt.Errorf(
			"Identity Client尚未初始化",
		)
	}

	if rawIDToken == "" ||
		rawIDToken != strings.TrimSpace(rawIDToken) {
		return nil, fmt.Errorf(
			"Identity ID Token为空或格式无效",
		)
	}

	// Flow nonce固定为32随机字节的base64url表示。
	if err := validateIdentityFlowRandomValue(
		"expected_nonce",
		expectedNonce,
	); err != nil {
		return nil, err
	}

	keys, err := c.fetchIdentityVerifierJWKS(ctx)
	if err != nil {
		return nil, err
	}

	claims := &IdentityIDTokenClaims{}

	token, err := jwt.ParseWithClaims(
		rawIDToken,
		claims,
		func(token *jwt.Token) (
			interface{},
			error,
		) {
			if token.Method.Alg() !=
				jwt.SigningMethodEdDSA.Alg() {
				return nil, fmt.Errorf(
					"Identity ID Token签名算法不是EdDSA",
				)
			}

			kid, ok := token.Header["kid"].(string)
			if !ok ||
				kid == "" ||
				kid != strings.TrimSpace(kid) {
				return nil, fmt.Errorf(
					"Identity ID Token缺少有效kid",
				)
			}

			key, exists := keys[kid]
			if !exists {
				return nil, fmt.Errorf(
					"Identity JWKS中不存在ID Token kid",
				)
			}

			return key, nil
		},
		jwt.WithValidMethods(
			[]string{
				jwt.SigningMethodEdDSA.Alg(),
			},
		),
		jwt.WithIssuer(c.cfg.Issuer),
		jwt.WithAudience(c.cfg.ClientID),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(
			func() time.Time {
				return c.now().UTC()
			},
		),
		jwt.WithLeeway(
			identityVerifierClockSkew,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"Identity ID Token验证失败：%w",
			err,
		)
	}

	if !token.Valid {
		return nil, fmt.Errorf(
			"Identity ID Token无效",
		)
	}

	if _, err := canonicalIdentityGlobalPersonID(
		claims.Subject,
	); err != nil {
		return nil, err
	}

	if claims.IssuedAt == nil {
		return nil, fmt.Errorf(
			"Identity ID Token缺少iat",
		)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf(
			"Identity ID Token缺少exp",
		)
	}

	now := c.now().UTC()
	issuedAt := claims.IssuedAt.Time.UTC()
	expiresAt := claims.ExpiresAt.Time.UTC()

	if issuedAt.After(
		now.Add(identityVerifierClockSkew),
	) {
		return nil, fmt.Errorf(
			"Identity ID Token iat位于未来",
		)
	}

	if !expiresAt.After(issuedAt) {
		return nil, fmt.Errorf(
			"Identity ID Token生命周期无效",
		)
	}

	// 时钟容差只影响当前时间判断，不允许把服务端签发的Token生命周期扩到300秒以上。
	if expiresAt.Sub(issuedAt) >
		identityVerifierMaxTokenLifetime {
		return nil, fmt.Errorf(
			"Identity ID Token生命周期超过允许上限",
		)
	}

	// expectedNonce具有高熵，仍使用常量时间比较，避免把nonce比较变成新的时序差异源。
	if !hmac.Equal(
		[]byte(claims.Nonce),
		[]byte(expectedNonce),
	) {
		return nil, fmt.Errorf(
			"Identity ID Token nonce不匹配",
		)
	}

	return claims, nil
}

// fetchIdentityVerifierJWKS 获取并解析当前Identity Ed25519公开签名键。
func (c *IdentityClient) fetchIdentityVerifierJWKS(
	ctx context.Context,
) (map[string]ed25519.PublicKey, error) {
	if c == nil ||
		c.httpClient == nil {
		return nil, fmt.Errorf(
			"Identity Client尚未初始化",
		)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.cfg.JWKSURI(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"创建Identity JWKS请求失败：%w",
			err,
		)
	}

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
		return nil, fmt.Errorf(
			"Identity JWKS请求失败：%w",
			err,
		)
	}
	defer response.Body.Close()

	body, err := readIdentityBoundedResponseBody(
		response.Body,
		identityVerifierJWKSMaxBytes,
	)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Identity JWKS返回HTTP %d",
			response.StatusCode,
		)
	}

	var jwks identityVerifierJWKS

	if err := json.Unmarshal(
		body,
		&jwks,
	); err != nil {
		return nil, fmt.Errorf(
			"解析Identity JWKS失败：%w",
			err,
		)
	}

	keys := make(
		map[string]ed25519.PublicKey,
	)

	for _, jwk := range jwks.Keys {
		if jwk.KTY != "OKP" ||
			jwk.CRV != "Ed25519" {
			continue
		}

		kid := strings.TrimSpace(jwk.KID)
		if kid == "" ||
			kid != jwk.KID {
			continue
		}

		if jwk.Use != "" &&
			jwk.Use != "sig" {
			continue
		}

		if jwk.Alg != "" &&
			jwk.Alg != "EdDSA" {
			continue
		}

		if _, exists := keys[kid]; exists {
			return nil, fmt.Errorf(
				"Identity JWKS存在重复kid",
			)
		}

		x, err := base64.RawURLEncoding.DecodeString(
			jwk.X,
		)
		if err != nil ||
			len(x) != ed25519.PublicKeySize {
			continue
		}

		// 与JSON临时缓冲区解除共享，返回稳定不可变公钥字节。
		keyBytes := append(
			[]byte(nil),
			x...,
		)

		keys[kid] =
			ed25519.PublicKey(keyBytes)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf(
			"Identity JWKS没有可用Ed25519签名键",
		)
	}

	return keys, nil
}
