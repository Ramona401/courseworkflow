package services

// assistant_runtime_token_test.go
//
// 本测试只验证密码学令牌、用途隔离、匿名哈希和三方状态绑定。
// 不连接数据库、不调用AI，也不签发真实生产令牌。

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"tedna/internal/models"
)

// TestAssistantRuntimeTokenRoundTripAndPurposeIsolation 验证正常签发和登录密钥隔离。
func TestAssistantRuntimeTokenRoundTripAndPurposeIsolation(
	t *testing.T,
) {
	baseSecret :=
		"unit-test-login-jwt-secret-with-sufficient-length"

	tokenService :=
		newAssistantRuntimeTokenService(
			baseSecret,
			10*time.Minute,
		)

	expiresAt :=
		time.Now().UTC().
			Add(9 * time.Minute)

	tokenString, err :=
		tokenService.Issue(
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
			3,
			models.AssistantRuntimeSessionKindExternal,
			strings.Repeat("j", 43),
			expiresAt,
		)
	if err != nil {
		t.Fatalf(
			"合法运行令牌签发失败: %v",
			err,
		)
	}

	claims, err :=
		tokenService.Parse(
			tokenString,
		)
	if err != nil {
		t.Fatalf(
			"合法运行令牌解析失败: %v",
			err,
		)
	}

	if claims.SessionID !=
		"11111111-1111-4111-8111-111111111111" ||
		claims.DeploymentID !=
			"22222222-2222-4222-8222-222222222222" ||
		claims.DeploymentVersion != 3 ||
		claims.TokenUse !=
			assistantRuntimeTokenUse {
		t.Fatalf(
			"运行令牌声明错误: %#v",
			claims,
		)
	}

	// 使用教师登录JWT基础密钥直接签名相同声明，
	// 必须因运行时用途派生签名密钥不同而被拒绝。
	forgedClaims := *claims
	forgedToken :=
		jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			&forgedClaims,
		)
	forgedString, err :=
		forgedToken.SignedString(
			[]byte(baseSecret),
		)
	if err != nil {
		t.Fatalf(
			"构造用途隔离测试令牌失败: %v",
			err,
		)
	}

	if _, err :=
		tokenService.Parse(
			forgedString,
		); !errors.Is(
		err,
		ErrAssistantRuntimeTokenInvalid,
	) {
		t.Fatalf(
			"教师登录密钥直签令牌必须被拒绝: %v",
			err,
		)
	}
}

// TestAssistantRuntimeTokenRejectsExpiredAndTampered 验证过期和篡改。
func TestAssistantRuntimeTokenRejectsExpiredAndTampered(
	t *testing.T,
) {
	tokenService :=
		newAssistantRuntimeTokenService(
			"another-unit-test-secret",
			10*time.Minute,
		)

	now := time.Now().UTC()

	expiredClaims :=
		&AssistantRuntimeTokenClaims{
			SessionID:         "11111111-1111-4111-8111-111111111111",
			DeploymentID:      "22222222-2222-4222-8222-222222222222",
			DeploymentVersion: 1,
			SessionKind:       models.AssistantRuntimeSessionKindExternal,
			TokenUse:          assistantRuntimeTokenUse,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:  assistantRuntimeTokenIssuer,
				Subject: assistantRuntimeTokenSubject,
				Audience: jwt.ClaimStrings{
					assistantRuntimeTokenAudience,
				},
				ExpiresAt: jwt.NewNumericDate(
					now.Add(-time.Minute),
				),
				NotBefore: jwt.NewNumericDate(
					now.Add(-10 * time.Minute),
				),
				IssuedAt: jwt.NewNumericDate(
					now.Add(-10 * time.Minute),
				),
				ID: strings.Repeat("e", 43),
			},
		}

	expiredToken :=
		jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			expiredClaims,
		)
	expiredString, err :=
		expiredToken.SignedString(
			tokenService.signingKey,
		)
	if err != nil {
		t.Fatalf(
			"构造过期测试令牌失败: %v",
			err,
		)
	}

	if _, err :=
		tokenService.Parse(
			expiredString,
		); !errors.Is(
		err,
		ErrAssistantRuntimeTokenExpired,
	) {
		t.Fatalf(
			"过期运行令牌必须被拒绝: %v",
			err,
		)
	}

	validString, err :=
		tokenService.Issue(
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
			1,
			models.AssistantRuntimeSessionKindExternal,
			strings.Repeat("v", 43),
			now.Add(9*time.Minute),
		)
	if err != nil {
		t.Fatalf(
			"构造篡改测试令牌失败: %v",
			err,
		)
	}

	lastByte :=
		validString[len(validString)-1]
	replacement := byte('A')
	if lastByte == replacement {
		replacement = 'B'
	}

	tampered :=
		validString[:len(validString)-1] +
			string(replacement)

	if _, err :=
		tokenService.Parse(
			tampered,
		); !errors.Is(
		err,
		ErrAssistantRuntimeTokenInvalid,
	) {
		t.Fatalf(
			"篡改运行令牌必须被拒绝: %v",
			err,
		)
	}
}

// TestAssistantRuntimeIdentityHashing 验证隐私哈希不保存原值且按用途隔离。
func TestAssistantRuntimeIdentityHashing(
	t *testing.T,
) {
	privacyKey :=
		deriveAssistantRuntimeKey(
			"unit-test-privacy-salt",
			assistantRuntimePrivacyPurpose,
		)

	clientHash, err :=
		assistantRuntimePrivacyHash(
			privacyKey,
			assistantRuntimeAnonymousClientHashPurpose,
			"anonymous_client_123456",
		)
	if err != nil {
		t.Fatalf(
			"匿名客户端哈希失败: %v",
			err,
		)
	}

	repeatedHash, err :=
		assistantRuntimePrivacyHash(
			privacyKey,
			assistantRuntimeAnonymousClientHashPurpose,
			"anonymous_client_123456",
		)
	if err != nil {
		t.Fatalf(
			"重复匿名客户端哈希失败: %v",
			err,
		)
	}

	ipHash, err :=
		assistantRuntimePrivacyHash(
			privacyKey,
			assistantRuntimeIPHashPurpose,
			"203.0.113.10",
		)
	if err != nil {
		t.Fatalf(
			"IP哈希失败: %v",
			err,
		)
	}

	if len(clientHash) != 64 ||
		clientHash != repeatedHash ||
		clientHash == ipHash ||
		strings.Contains(
			clientHash,
			"anonymous_client_123456",
		) {
		t.Fatalf(
			"隐私哈希不符合要求: client=%s repeated=%s ip=%s",
			clientHash,
			repeatedHash,
			ipHash,
		)
	}
}

// TestAssistantRuntimeAuthorizationStateRejectsInvalidBindings 验证会话实时失效条件。
func TestAssistantRuntimeAuthorizationStateRejectsInvalidBindings(
	t *testing.T,
) {
	now := time.Now().UTC()
	expiresAt := now.Add(10 * time.Minute)
	validFrom := now.Add(-time.Hour)
	validUntil := now.Add(time.Hour)

	jti := strings.Repeat("t", 43)

	claims :=
		&AssistantRuntimeTokenClaims{
			SessionID:         "11111111-1111-4111-8111-111111111111",
			DeploymentID:      "22222222-2222-4222-8222-222222222222",
			DeploymentVersion: 4,
			SessionKind:       models.AssistantRuntimeSessionKindExternal,
			TokenUse:          assistantRuntimeTokenUse,
			RegisteredClaims: jwt.RegisteredClaims{
				ID: jti,
			},
		}

	session :=
		&models.AssistantRuntimeSession{
			ID:                claims.SessionID,
			DeploymentID:      claims.DeploymentID,
			DeploymentVersion: claims.DeploymentVersion,
			TokenJTIHash:      assistantRuntimeJTIHash(jti),
			SessionKind:       models.AssistantRuntimeSessionKindExternal,
			Status:            models.AssistantRuntimeSessionStatusActive,
			MaxTurns:          10,
			ExpiresAt:         &expiresAt,
		}

	deployment :=
		&models.AssistantDeployment{
			ID:             claims.DeploymentID,
			CurrentVersion: claims.DeploymentVersion,
			Status:         models.AssistantDeploymentStatusActive,
			ValidFrom:      &validFrom,
			ValidUntil:     &validUntil,
		}

	if err :=
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		); err != nil {
		t.Fatalf(
			"合法三方绑定不应被拒绝: %v",
			err,
		)
	}

	deployment.Status =
		models.AssistantDeploymentStatusRevoked
	if !errors.Is(
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		),
		ErrAssistantRuntimeDeploymentUnavailable,
	) {
		t.Fatal(
			"撤销部署必须使令牌失效",
		)
	}

	deployment.Status =
		models.AssistantDeploymentStatusActive
	deployment.CurrentVersion = 5
	if !errors.Is(
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		),
		ErrAssistantRuntimeDeploymentVersionMismatch,
	) {
		t.Fatal(
			"部署当前版本变化必须使旧令牌失效",
		)
	}

	deployment.CurrentVersion =
		claims.DeploymentVersion
	session.Status =
		models.AssistantRuntimeSessionStatusRevoked
	if !errors.Is(
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		),
		ErrAssistantRuntimeSessionInactive,
	) {
		t.Fatal(
			"撤销会话必须使令牌失效",
		)
	}

	session.Status =
		models.AssistantRuntimeSessionStatusActive
	expiredAt := now.Add(-time.Second)
	session.ExpiresAt = &expiredAt
	if !errors.Is(
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		),
		ErrAssistantRuntimeTokenExpired,
	) {
		t.Fatal(
			"数据库会话过期必须使令牌失效",
		)
	}
}

// TestAssistantRuntimeOriginAndWelcomeSnapshot 验证来源精确命中和欢迎语提取。
func TestAssistantRuntimeOriginAndWelcomeSnapshot(
	t *testing.T,
) {
	if !assistantRuntimeOriginAllowed(
		"https://course.example",
		[]string{
			"https://course.example",
		},
	) {
		t.Fatal(
			"精确允许来源应当命中",
		)
	}

	if assistantRuntimeOriginAllowed(
		"https://evil.example",
		[]string{
			"https://course.example",
		},
	) {
		t.Fatal(
			"未授权来源不应命中",
		)
	}

	version :=
		&models.AssistantDeploymentVersion{
			TeachingPlanJSON: `{"version":"v1","title":"助手","welcome_message":"先观察，再回答。","teaching_role":"提问","learning_objective":"解释","display_mode":"floating","display_position":"bottom_right","guidance_plan":{"version":"v1","guiding_principles":[],"question_chain":[],"misconception_branches":[],"forbidden_behaviors":[],"completion_criteria":[],"answer_leak_policy":{"direct_answer_allowed":false,"require_student_try":true,"maximum_hint_level":3,"prohibited_behaviors":[],"safe_closure_guidance":""}}}`,
		}

	welcome, err :=
		assistantRuntimeWelcomeMessageFromVersion(
			version,
		)
	if err != nil {
		t.Fatalf(
			"欢迎语提取失败: %v",
			err,
		)
	}

	if welcome !=
		"先观察，再回答。" {
		t.Fatalf(
			"欢迎语内容错误: %s",
			welcome,
		)
	}
}
