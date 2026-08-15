package services

// identity_flow.go — TE-DNA Identity Account Linking短时安全Flow。
//
// Phase 1安全边界：
//   - 只允许已经登录TE-DNA的本地用户主动发起Link/Unlink；
//   - 本地身份固定为TE-DNA users.id(UUID)，不得接受浏览器提交local_account_id；
//   - global_person_id只从Identity Center经OIDC可信结果取得；
//   - 本文件不签发TE-DNA JWT、不建立本地Session，不提前实现Central SSO；
//   - state、OIDC nonce、PKCE verifier仅存在AES-GCM保护的HttpOnly Cookie中。

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

const (
	// IdentityFlowPurposeLink / Unlink 是Phase 1唯一允许的Flow用途。
	IdentityFlowPurposeLink   = "link"
	IdentityFlowPurposeUnlink = "unlink"

	// IdentityFlowCookieName 使用__Host-前缀。
	// 后续Handler设置Cookie时必须同时满足：
	//   Secure=true、Path=/、不设置Domain。
	IdentityFlowCookieName = "__Host-tedna_identity_flow"

	identityFlowVersion = 1

	// IdentityFlowTTL 与Identity Authorization Request的10分钟生命周期对齐。
	IdentityFlowTTL = 10 * time.Minute

	// 只容忍很小的本机时钟前跳，超出即fail-closed。
	identityFlowClockSkew = 30 * time.Second

	// 32随机字节经RawURL编码后为43字符，满足PKCE verifier 43—128字符要求。
	identityFlowRandomByteLength = 32

	// 从现有JWT root secret领域隔离派生独立AES-256-GCM密钥。
	// 不直接把JWT Secret本身作为AES Key使用。
	identityFlowKeyDomain = "pkuailab/tedna/identity-flow/key/v1"

	// GCM Additional Authenticated Data固定绑定当前协议用途和版本。
	identityFlowAAD = "pkuailab/tedna/identity-flow/payload/v1"
)

// IdentityAuthorizationFlow 是服务器生成并加密写入HttpOnly Cookie的短时状态。
//
// UserID固定为TE-DNA public.users.id的规范UUID字符串。
// 浏览器永远不能通过query/body覆盖这里的UserID。
type IdentityAuthorizationFlow struct {
	Version      int    `json:"v"`
	Purpose      string `json:"purpose"`
	UserID       string `json:"user_id"`
	State        string `json:"state"`
	Nonce        string `json:"nonce"`
	CodeVerifier string `json:"code_verifier"`
	IssuedAt     int64  `json:"issued_at"`
}

// IdentityFlowProtector 使用AES-256-GCM保护整个Flow。
//
// now字段保留为内部时钟依赖，便于后续定向测试TTL边界。
type IdentityFlowProtector struct {
	aead cipher.AEAD
	now  func() time.Time
}

// NewIdentityFlowProtector 从现有TE-DNA JWT root secret领域隔离派生Flow密钥。
//
// rootSecret至少32字节；不足时Identity能力fail-closed，
// 但调用方应采用惰性初始化，不能因此破坏TE-DNA既有本地登录。
func NewIdentityFlowProtector(
	rootSecret []byte,
) (*IdentityFlowProtector, error) {
	if len(rootSecret) < 32 {
		return nil, fmt.Errorf(
			"Identity Flow root secret长度不足",
		)
	}

	mac := hmac.New(
		sha256.New,
		rootSecret,
	)

	_, _ = mac.Write(
		[]byte(identityFlowKeyDomain),
	)

	key := mac.Sum(nil)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf(
			"初始化Identity Flow AES失败：%w",
			err,
		)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"初始化Identity Flow GCM失败：%w",
			err,
		)
	}

	return &IdentityFlowProtector{
		aead: aead,
		now:  time.Now,
	}, nil
}

// NewAuthorizationFlow 为当前已认证TE-DNA用户生成Link/Unlink安全Flow，
// 同时返回PKCE S256 code_challenge。
func (p *IdentityFlowProtector) NewAuthorizationFlow(
	purpose string,
	userID string,
) (
	IdentityAuthorizationFlow,
	string,
	error,
) {
	if p == nil ||
		p.aead == nil ||
		p.now == nil {
		return IdentityAuthorizationFlow{},
			"",
			fmt.Errorf("Identity Flow Protector不可用")
	}

	if !validIdentityFlowPurpose(purpose) {
		return IdentityAuthorizationFlow{},
			"",
			fmt.Errorf(
				"不支持的Identity Flow用途：%s",
				purpose,
			)
	}

	canonicalUserID, err :=
		canonicalIdentityLocalUserID(userID)
	if err != nil {
		return IdentityAuthorizationFlow{},
			"",
			err
	}

	state, err := newIdentityRandomBase64URL(
		identityFlowRandomByteLength,
	)
	if err != nil {
		return IdentityAuthorizationFlow{},
			"",
			err
	}

	nonce, err := newIdentityRandomBase64URL(
		identityFlowRandomByteLength,
	)
	if err != nil {
		return IdentityAuthorizationFlow{},
			"",
			err
	}

	codeVerifier, err := newIdentityRandomBase64URL(
		identityFlowRandomByteLength,
	)
	if err != nil {
		return IdentityAuthorizationFlow{},
			"",
			err
	}

	challengeDigest := sha256.Sum256(
		[]byte(codeVerifier),
	)

	codeChallenge :=
		base64.RawURLEncoding.EncodeToString(
			challengeDigest[:],
		)

	flow := IdentityAuthorizationFlow{
		Version:      identityFlowVersion,
		Purpose:      purpose,
		UserID:       canonicalUserID,
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		IssuedAt:     p.now().Unix(),
	}

	if err := validateIdentityAuthorizationFlow(
		flow,
		p.now(),
	); err != nil {
		return IdentityAuthorizationFlow{},
			"",
			err
	}

	return flow, codeChallenge, nil
}

// Seal 将Flow完整加密认证为Cookie值。
//
// 输出为base64url(nonce || ciphertext || gcm_tag)，
// 不包含Client Secret，也没有任何可读的本地用户ID。
func (p *IdentityFlowProtector) Seal(
	flow IdentityAuthorizationFlow,
) (string, error) {
	if p == nil ||
		p.aead == nil ||
		p.now == nil {
		return "", fmt.Errorf(
			"Identity Flow Protector不可用",
		)
	}

	if err := validateIdentityAuthorizationFlow(
		flow,
		p.now(),
	); err != nil {
		return "", err
	}

	plaintext, err := json.Marshal(flow)
	if err != nil {
		return "", fmt.Errorf(
			"序列化Identity Flow失败：%w",
			err,
		)
	}

	nonce := make(
		[]byte,
		p.aead.NonceSize(),
	)

	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf(
			"生成Identity Flow GCM nonce失败：%w",
			err,
		)
	}

	ciphertext := p.aead.Seal(
		nil,
		nonce,
		plaintext,
		[]byte(identityFlowAAD),
	)

	payload := make(
		[]byte,
		0,
		len(nonce)+len(ciphertext),
	)

	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64.RawURLEncoding.EncodeToString(
		payload,
	), nil
}

// Open 解密并严格验证HttpOnly Cookie中的Flow。
//
// 任一篡改、过期、版本异常、用途异常、UserID异常或随机字段异常均拒绝。
func (p *IdentityFlowProtector) Open(
	token string,
) (
	IdentityAuthorizationFlow,
	error,
) {
	if p == nil ||
		p.aead == nil ||
		p.now == nil {
		return IdentityAuthorizationFlow{},
			fmt.Errorf("Identity Flow Protector不可用")
	}

	if token == "" {
		return IdentityAuthorizationFlow{},
			fmt.Errorf("Identity Flow Cookie为空")
	}

	payload, err :=
		base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return IdentityAuthorizationFlow{},
			fmt.Errorf("Identity Flow编码无效")
	}

	nonceSize := p.aead.NonceSize()

	if len(payload) <= nonceSize {
		return IdentityAuthorizationFlow{},
			fmt.Errorf("Identity Flow长度无效")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	plaintext, err := p.aead.Open(
		nil,
		nonce,
		ciphertext,
		[]byte(identityFlowAAD),
	)
	if err != nil {
		return IdentityAuthorizationFlow{},
			fmt.Errorf("Identity Flow认证失败")
	}

	var flow IdentityAuthorizationFlow

	decoder := json.NewDecoder(
		bytes.NewReader(plaintext),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&flow); err != nil {
		return IdentityAuthorizationFlow{},
			fmt.Errorf(
				"Identity Flow载荷无效：%w",
				err,
			)
	}

	var trailing struct{}

	if err := decoder.Decode(&trailing); err != io.EOF {
		return IdentityAuthorizationFlow{},
			fmt.Errorf(
				"Identity Flow载荷存在尾随数据",
			)
	}

	if err := validateIdentityAuthorizationFlow(
		flow,
		p.now(),
	); err != nil {
		return IdentityAuthorizationFlow{},
			err
	}

	return flow, nil
}

func validateIdentityAuthorizationFlow(
	flow IdentityAuthorizationFlow,
	now time.Time,
) error {
	if flow.Version != identityFlowVersion {
		return fmt.Errorf(
			"Identity Flow版本无效",
		)
	}

	if !validIdentityFlowPurpose(flow.Purpose) {
		return fmt.Errorf(
			"Identity Flow用途无效",
		)
	}

	canonicalUserID, err :=
		canonicalIdentityLocalUserID(flow.UserID)
	if err != nil {
		return err
	}

	if canonicalUserID != flow.UserID {
		return fmt.Errorf(
			"Identity Flow user_id不是规范UUID",
		)
	}

	if err := validateIdentityFlowRandomValue(
		"state",
		flow.State,
	); err != nil {
		return err
	}

	if err := validateIdentityFlowRandomValue(
		"nonce",
		flow.Nonce,
	); err != nil {
		return err
	}

	if err := validateIdentityFlowRandomValue(
		"code_verifier",
		flow.CodeVerifier,
	); err != nil {
		return err
	}

	if flow.IssuedAt <= 0 {
		return fmt.Errorf(
			"Identity Flow issued_at无效",
		)
	}

	issuedAt := time.Unix(
		flow.IssuedAt,
		0,
	)

	if issuedAt.After(
		now.Add(identityFlowClockSkew),
	) {
		return fmt.Errorf(
			"Identity Flow签发时间异常",
		)
	}

	if now.Sub(issuedAt) > IdentityFlowTTL {
		return fmt.Errorf(
			"Identity Flow已过期",
		)
	}

	return nil
}

func validIdentityFlowPurpose(
	purpose string,
) bool {
	return purpose == IdentityFlowPurposeLink ||
		purpose == IdentityFlowPurposeUnlink
}

// canonicalIdentityLocalUserID 固定Phase 1 local_account_id语义：
// TE-DNA public.users.id的规范UUID字符串。
func canonicalIdentityLocalUserID(
	userID string,
) (string, error) {
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf(
			"Identity Flow user_id不是有效UUID",
		)
	}

	canonical := parsed.String()

	if canonical != userID {
		return "", fmt.Errorf(
			"Identity Flow user_id必须使用规范UUID格式",
		)
	}

	return canonical, nil
}

func validateIdentityFlowRandomValue(
	name string,
	value string,
) error {
	if value == "" {
		return fmt.Errorf(
			"Identity Flow %s为空",
			name,
		)
	}

	decoded, err :=
		base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf(
			"Identity Flow %s编码无效",
			name,
		)
	}

	if len(decoded) != identityFlowRandomByteLength {
		return fmt.Errorf(
			"Identity Flow %s长度无效",
			name,
		)
	}

	return nil
}

func newIdentityRandomBase64URL(
	byteLength int,
) (string, error) {
	if byteLength < 16 {
		return "", fmt.Errorf(
			"Identity安全随机值长度过短",
		)
	}

	value := make(
		[]byte,
		byteLength,
	)

	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf(
			"生成Identity安全随机值失败：%w",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(
		value,
	), nil
}
