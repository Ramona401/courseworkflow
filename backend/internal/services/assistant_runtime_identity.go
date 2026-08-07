package services

// assistant_runtime_identity.go
//
// 本文件生成运行会话UUID和JTI，并负责匿名客户端标识与IP的隐私哈希。
//
// 哈希策略：
//   - JTI使用SHA-256，JTI本身由32字节密码学随机数生成；
//   - 匿名客户端和IP使用服务端隐私盐派生密钥执行HMAC-SHA256；
//   - 不同字段带独立purpose，避免跨字段哈希关联；
//   - 原始IP和匿名客户端ID只在当前请求内短暂存在，不进入数据库响应。
//
// 本文件不记录日志、不写数据库。

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

const (
	assistantRuntimeRandomIDBytes = 32

	assistantRuntimeAnonymousClientMinRunes = 16
	assistantRuntimeAnonymousClientMaxRunes = 256
)

var (
	ErrAssistantRuntimeAnonymousClientInvalid = errors.New(
		"匿名客户端标识无效",
	)

	ErrAssistantRuntimeClientIPInvalid = errors.New(
		"运行会话客户端IP无效",
	)

	ErrAssistantRuntimeRandomSourceFailed = errors.New(
		"生成教学智能体运行随机标识失败",
	)
)

var assistantRuntimeAnonymousClientPattern = regexp.MustCompile(
	`^[A-Za-z0-9_-]{16,256}$`,
)

// generateAssistantRuntimeRandomID 生成URL安全随机JTI。
func generateAssistantRuntimeRandomID() (
	string,
	error,
) {
	randomBytes := make(
		[]byte,
		assistantRuntimeRandomIDBytes,
	)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrAssistantRuntimeRandomSourceFailed,
				err,
			)
	}

	return base64.RawURLEncoding.
			EncodeToString(
				randomBytes,
			),
		nil
}

// generateAssistantRuntimeSessionID 生成符合RFC 4122的随机UUID v4。
func generateAssistantRuntimeSessionID() (
	string,
	error,
) {
	randomBytes := make(
		[]byte,
		16,
	)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrAssistantRuntimeRandomSourceFailed,
				err,
			)
	}

	randomBytes[6] =
		(randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] =
		(randomBytes[8] & 0x3f) | 0x80

	encoded :=
		hex.EncodeToString(
			randomBytes,
		)

	return encoded[0:8] + "-" +
			encoded[8:12] + "-" +
			encoded[12:16] + "-" +
			encoded[16:20] + "-" +
			encoded[20:32],
		nil
}

// normalizeAssistantRuntimeAnonymousClientID 校验浏览器随机标识。
func normalizeAssistantRuntimeAnonymousClientID(
	raw string,
) (
	string,
	error,
) {
	raw = strings.TrimSpace(raw)

	runeCount := len([]rune(raw))
	if runeCount <
		assistantRuntimeAnonymousClientMinRunes ||
		runeCount >
			assistantRuntimeAnonymousClientMaxRunes ||
		!assistantRuntimeAnonymousClientPattern.
			MatchString(raw) {
		return "",
			ErrAssistantRuntimeAnonymousClientInvalid
	}

	return raw, nil
}

// normalizeAssistantRuntimeClientIP 规范化IPv4或IPv6。
//
// 调用方可以传入纯IP或net/http中的RemoteAddr。
func normalizeAssistantRuntimeClientIP(
	raw string,
) (
	string,
	error,
) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "",
			ErrAssistantRuntimeClientIPInvalid
	}

	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}

	raw = strings.Trim(
		raw,
		"[]",
	)

	ip := net.ParseIP(raw)
	if ip == nil {
		return "",
			ErrAssistantRuntimeClientIPInvalid
	}

	return ip.String(), nil
}

// assistantRuntimeJTIHash 生成数据库保存的JTI哈希。
func assistantRuntimeJTIHash(
	jti string,
) string {
	sum := sha256.Sum256(
		[]byte(
			strings.TrimSpace(jti),
		),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

// assistantRuntimePrivacyHash 对匿名标识或IP执行带用途HMAC。
func assistantRuntimePrivacyHash(
	privacyKey []byte,
	purpose string,
	value string,
) (
	string,
	error,
) {
	if len(privacyKey) != sha256.Size ||
		strings.TrimSpace(purpose) == "" ||
		strings.TrimSpace(value) == "" {
		return "",
			ErrAssistantRuntimeTokenConfiguration
	}

	mac := hmac.New(
		sha256.New,
		privacyKey,
	)
	_, _ = mac.Write(
		[]byte(purpose),
	)
	_, _ = mac.Write(
		[]byte{0},
	)
	_, _ = mac.Write(
		[]byte(value),
	)

	return hex.EncodeToString(
		mac.Sum(nil),
	), nil
}

// assistantRuntimeHashEqual 使用常量时间比较两个十六进制哈希。
func assistantRuntimeHashEqual(
	left string,
	right string,
) bool {
	return hmac.Equal(
		[]byte(strings.TrimSpace(left)),
		[]byte(strings.TrimSpace(right)),
	)
}
