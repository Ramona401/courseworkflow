package services

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

const (
	identityFlowTestLocalUserID = "22222222-2222-4222-8222-222222222222"

	identityFlowTestRootSecret = "tedna-identity-flow-test-root-secret-0123456789-abcdefghijklmnopqrstuvwxyz"
)

func newIdentityFlowTestProtector(
	t *testing.T,
	now time.Time,
) *IdentityFlowProtector {
	t.Helper()

	protector, err := NewIdentityFlowProtector(
		[]byte(identityFlowTestRootSecret),
	)
	if err != nil {
		t.Fatalf(
			"NewIdentityFlowProtector() error = %v",
			err,
		)
	}

	protector.now = func() time.Time {
		return now
	}

	return protector
}

// TestIdentityFlowPKCES256AndEncryptedRoundTrip验证：
//   - Link Flow绑定规范TE-DNA users.id；
//   - state/nonce/verifier均为高熵随机值；
//   - challenge严格等于SHA256(verifier)的base64url raw编码；
//   - Flow经过AES-GCM Cookie载荷往返后字段保持一致。
func TestIdentityFlowPKCES256AndEncryptedRoundTrip(
	t *testing.T,
) {
	fixedNow := time.Date(
		2026,
		time.August,
		11,
		2,
		30,
		0,
		0,
		time.UTC,
	)

	protector :=
		newIdentityFlowTestProtector(
			t,
			fixedNow,
		)

	flow, challenge, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatalf(
			"NewAuthorizationFlow() error = %v",
			err,
		)
	}

	if flow.Purpose != IdentityFlowPurposeLink {
		t.Fatalf(
			"Purpose = %q",
			flow.Purpose,
		)
	}

	if flow.UserID != identityFlowTestLocalUserID {
		t.Fatalf(
			"UserID = %q",
			flow.UserID,
		)
	}

	if flow.State == "" ||
		flow.Nonce == "" ||
		flow.CodeVerifier == "" {
		t.Fatal(
			"Flow安全随机字段不得为空",
		)
	}

	if flow.State == flow.Nonce ||
		flow.State == flow.CodeVerifier ||
		flow.Nonce == flow.CodeVerifier {
		t.Fatal(
			"state、nonce、code_verifier不得复用同一随机值",
		)
	}

	if flow.IssuedAt != fixedNow.Unix() {
		t.Fatalf(
			"IssuedAt = %d, want %d",
			flow.IssuedAt,
			fixedNow.Unix(),
		)
	}

	digest := sha256.Sum256(
		[]byte(flow.CodeVerifier),
	)

	expectedChallenge :=
		base64.RawURLEncoding.EncodeToString(
			digest[:],
		)

	if challenge != expectedChallenge {
		t.Fatalf(
			"PKCE challenge不匹配：got=%q want=%q",
			challenge,
			expectedChallenge,
		)
	}

	if strings.Contains(challenge, "=") {
		t.Fatal(
			"PKCE challenge不得包含base64 padding",
		)
	}

	sealed, err := protector.Seal(flow)
	if err != nil {
		t.Fatalf(
			"Seal() error = %v",
			err,
		)
	}

	if sealed == "" {
		t.Fatal(
			"加密Flow Token为空",
		)
	}

	// Cookie正文必须是密文，不允许直接出现本地账号、state或verifier。
	for _, forbidden := range []string{
		identityFlowTestLocalUserID,
		flow.State,
		flow.CodeVerifier,
	} {
		if strings.Contains(
			sealed,
			forbidden,
		) {
			t.Fatalf(
				"Flow Token泄漏明文字段：%q",
				forbidden,
			)
		}
	}

	opened, err := protector.Open(sealed)
	if err != nil {
		t.Fatalf(
			"Open() error = %v",
			err,
		)
	}

	if opened != flow {
		t.Fatalf(
			"Flow往返不一致：got=%+v want=%+v",
			opened,
			flow,
		)
	}
}

// TestIdentityFlowCiphertextTamperRejected确认AES-GCM认证失败必须fail-closed。
func TestIdentityFlowCiphertextTamperRejected(
	t *testing.T,
) {
	fixedNow := time.Date(
		2026,
		time.August,
		11,
		2,
		31,
		0,
		0,
		time.UTC,
	)

	protector :=
		newIdentityFlowTestProtector(
			t,
			fixedNow,
		)

	flow, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := protector.Seal(flow)
	if err != nil {
		t.Fatal(err)
	}

	raw, err :=
		base64.RawURLEncoding.DecodeString(
			sealed,
		)
	if err != nil {
		t.Fatal(err)
	}

	if len(raw) == 0 {
		t.Fatal(
			"测试Flow密文为空",
		)
	}

	raw[len(raw)-1] ^= 0x01

	tampered :=
		base64.RawURLEncoding.EncodeToString(
			raw,
		)

	if _, err := protector.Open(tampered); err == nil {
		t.Fatal(
			"被篡改的Flow Token必须被拒绝",
		)
	}
}

// TestIdentityFlowExpiresAfterTTL确认Flow只能在短时授权窗口内使用。
func TestIdentityFlowExpiresAfterTTL(
	t *testing.T,
) {
	issuedAt := time.Date(
		2026,
		time.August,
		11,
		2,
		32,
		0,
		0,
		time.UTC,
	)

	protector :=
		newIdentityFlowTestProtector(
			t,
			issuedAt,
		)

	flow, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeUnlink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := protector.Seal(flow)
	if err != nil {
		t.Fatal(err)
	}

	protector.now = func() time.Time {
		return issuedAt.
			Add(IdentityFlowTTL).
			Add(time.Second)
	}

	if _, err := protector.Open(sealed); err == nil {
		t.Fatal(
			"超过Identity Flow TTL后必须拒绝",
		)
	}
}

// TestIdentityFlowRejectsInvalidPurposeAndLocalID确认Phase 1只允许Link/Unlink，
// 且本地账号必须是规范public.users.id UUID。
func TestIdentityFlowRejectsInvalidPurposeAndLocalID(
	t *testing.T,
) {
	protector :=
		newIdentityFlowTestProtector(
			t,
			time.Date(
				2026,
				time.August,
				11,
				2,
				33,
				0,
				0,
				time.UTC,
			),
		)

	if _, _, err :=
		protector.NewAuthorizationFlow(
			"login",
			identityFlowTestLocalUserID,
		); err == nil {
		t.Fatal(
			"Phase 1不得创建Identity login Flow",
		)
	}

	if _, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			"",
		); err == nil {
		t.Fatal(
			"Link Flow缺少本地users.id时必须失败",
		)
	}

	if _, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			"22222222-2222-4222-8222-22222222222Z",
		); err == nil {
		t.Fatal(
			"非法本地UUID必须被拒绝",
		)
	}

	if _, _, err :=
		protector.NewAuthorizationFlow(
			IdentityFlowPurposeUnlink,
			identityFlowTestLocalUserID,
		); err != nil {
		t.Fatalf(
			"合法Unlink Flow被错误拒绝：%v",
			err,
		)
	}
}

// TestIdentityFlowDifferentRootSecretCannotDecrypt确认Flow密钥只能由同一TE-DNA
// 本地root secret经固定领域隔离派生，错误root secret不能解密既有Flow。
func TestIdentityFlowDifferentRootSecretCannotDecrypt(
	t *testing.T,
) {
	fixedNow := time.Date(
		2026,
		time.August,
		11,
		2,
		34,
		0,
		0,
		time.UTC,
	)

	first :=
		newIdentityFlowTestProtector(
			t,
			fixedNow,
		)

	flow, _, err :=
		first.NewAuthorizationFlow(
			IdentityFlowPurposeLink,
			identityFlowTestLocalUserID,
		)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := first.Seal(flow)
	if err != nil {
		t.Fatal(err)
	}

	second, err := NewIdentityFlowProtector(
		[]byte(
			"another-tedna-identity-root-secret-9876543210-abcdefghijklmnopqrstuvwxyz",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	second.now = func() time.Time {
		return fixedNow
	}

	if _, err := second.Open(sealed); err == nil {
		t.Fatal(
			"不同root secret不得解密既有Identity Flow",
		)
	}
}

// TestIdentityFlowCookieContract冻结__Host- Cookie名称和10分钟短生命周期合同。
func TestIdentityFlowCookieContract(
	t *testing.T,
) {
	if IdentityFlowCookieName !=
		"__Host-tedna_identity_flow" {
		t.Fatalf(
			"Identity Flow Cookie名称漂移：%q",
			IdentityFlowCookieName,
		)
	}

	if IdentityFlowTTL !=
		10*time.Minute {
		t.Fatalf(
			"Identity Flow TTL漂移：%v",
			IdentityFlowTTL,
		)
	}
}
