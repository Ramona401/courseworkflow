package integration

// assistant_runtime_http_session_test.go
//
// 使用真实HTTP服务和tedna_test验证：
//   - embed只返回安全HTML并正确转义公开文字；
//   - 合法Origin创建短时运行会话；
//   - JWT声明、数据库JTI哈希和路径session_id严格绑定；
//   - 正确令牌能够读取安全会话视图；
//   - 篡改令牌、教师JWT和串会话路径被拒绝；
//   - 未授权Origin、缺失Origin和正文伪造字段不会创建会话。
//
// 本测试不调用聊天接口，不访问真实AI。

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// TestAssistantRuntimeHTTPEmbedAndSessionLifecycle 验证embed和会话主链。
func TestAssistantRuntimeHTTPEmbedAndSessionLifecycle(
	t *testing.T,
) {
	server,
		_ :=
		SetupTestServer(
			t,
		)

	CleanAndSeed(t)

	fixture := SeedAssistantRuntimeFixture(
		t,
	)

	deployment,
		version :=
		fixture.NewDeploymentRecords()

	// 在不可变版本首次写入前加入HTML测试文字。
	var teachingPlan map[string]interface{}

	if err := json.Unmarshal(
		[]byte(
			version.TeachingPlanJSON,
		),
		&teachingPlan,
	); err != nil {
		t.Fatalf(
			"解析教学计划快照失败: %v",
			err,
		)
	}

	teachingPlan["title"] =
		`<script>alert("x")</script>面积伙伴`
	teachingPlan["welcome_message"] =
		`欢迎 <b>同学</b>`

	encodedPlan,
		err :=
		json.Marshal(
			teachingPlan,
		)
	if err != nil {
		t.Fatalf(
			"编码HTML转义测试快照失败: %v",
			err,
		)
	}

	version.TeachingPlanJSON =
		string(encodedPlan)

	if err := repository.CreateAssistantDeploymentWithFirstVersion(
		context.Background(),
		deployment,
		version,
	); err != nil {
		t.Fatalf(
			"创建HTML转义测试部署失败: %v",
			err,
		)
	}

	embedResult := doAssistantRuntimeHTTPRequest(
		t,
		http.MethodGet,
		server.URL+
			"/embed/assistant/"+
			deployment.PublicID,
		nil,
		nil,
	)

	if embedResult.Response.StatusCode !=
		http.StatusOK {
		t.Fatalf(
			"读取embed失败: HTTP=%d body=%s",
			embedResult.Response.StatusCode,
			string(embedResult.Body),
		)
	}

	contentType := embedResult.Response.Header.Get(
		"Content-Type",
	)

	if !strings.Contains(
		contentType,
		"text/html",
	) {
		t.Fatalf(
			"embed Content-Type错误: %s",
			contentType,
		)
	}

	if !strings.Contains(
		embedResult.Response.Header.Get(
			"Content-Security-Policy",
		),
		"default-src 'none'",
	) {
		t.Fatalf(
			"embed缺少严格CSP: %s",
			embedResult.Response.Header.Get(
				"Content-Security-Policy",
			),
		)
	}

	embedBody := string(
		embedResult.Body,
	)

	if strings.Contains(
		embedBody,
		"<script>alert",
	) ||
		strings.Contains(
			embedBody,
			"<b>同学</b>",
	) {
		t.Fatalf(
			"embed未转义公开文字: %s",
			embedBody,
		)
	}

	if !strings.Contains(
		embedBody,
		"&lt;script&gt;",
	) ||
		!strings.Contains(
			embedBody,
			"&lt;b&gt;同学&lt;/b&gt;",
	) ||
		!strings.Contains(
			embedBody,
			deployment.PublicID,
	) {
		t.Fatalf(
			"embed缺少转义文字或public_id: %s",
			embedBody,
		)
	}

	assertAssistantRuntimeNoSensitiveFields(
		t,
		embedBody,
	)

	anonymousClientID :=
		"anonymous_client_123456"

	startResponse,
		startResult :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			anonymousClientID,
		)

	if startResponse.Status !=
		models.AssistantRuntimeSessionStatusActive ||
		startResponse.MaxTurns != 5 ||
		startResponse.WelcomeMessage !=
			`欢迎 <b>同学</b>` ||
		startResponse.ExpiresAt == nil {
		t.Fatalf(
			"会话创建响应错误: %+v",
			startResponse,
		)
	}

	assertAssistantRuntimeNoSensitiveFields(
		t,
		string(startResult.Body),
	)

	claims := parseAssistantRuntimeTestTokenClaims(
		t,
		startResponse.RuntimeToken,
	)

	if claims.SessionID !=
		startResponse.SessionID ||
		claims.DeploymentID !=
			deployment.ID ||
		claims.DeploymentVersion != 1 ||
		claims.SessionKind !=
			models.AssistantRuntimeSessionKindExternal ||
		claims.TokenUse !=
			"assistant_runtime" {
		t.Fatalf(
			"运行令牌绑定错误: %+v",
			claims,
		)
	}

	assertAssistantRuntimeShortTTL(
		t,
		claims,
	)

	tokenPayload := assistantRuntimeTestTokenPayload(
		t,
		startResponse.RuntimeToken,
	)

	assertAssistantRuntimeNoSensitiveFields(
		t,
		tokenPayload,
	)

	var (
		storedJTIHash     string
		storedClientHash  string
		storedIPHash      string
		storedOrigin      string
		storedStatus      string
		storedVersion     int
		storedMessagesRaw string
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			token_jti_hash,
			anonymous_client_hash,
			ip_hash,
			origin_snapshot,
			status,
			deployment_version,
			messages_json::text
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		startResponse.SessionID,
	).Scan(
		&storedJTIHash,
		&storedClientHash,
		&storedIPHash,
		&storedOrigin,
		&storedStatus,
		&storedVersion,
		&storedMessagesRaw,
	); err != nil {
		t.Fatalf(
			"读取HTTP创建的运行会话失败: %v",
			err,
		)
	}

	expectedJTIHash :=
		assistantRuntimeTestJTIHash(
			claims.ID,
		)

	if storedJTIHash != expectedJTIHash ||
		len(storedClientHash) != 64 ||
		len(storedIPHash) != 64 ||
		storedClientHash == storedIPHash ||
		storedClientHash ==
			anonymousClientID ||
		storedIPHash ==
			"203.0.113.10" ||
		storedOrigin !=
			AssistantFixtureOrigin ||
		storedStatus !=
			models.AssistantRuntimeSessionStatusActive ||
		storedVersion != 1 ||
		storedMessagesRaw != "[]" {
		t.Fatalf(
			"HTTP会话数据库绑定异常: jti=%s expected=%s client=%s ip=%s origin=%s status=%s version=%d messages=%s",
			storedJTIHash,
			expectedJTIHash,
			storedClientHash,
			storedIPHash,
			storedOrigin,
			storedStatus,
			storedVersion,
			storedMessagesRaw,
		)
	}

	sessionView,
		viewResult :=
		readAssistantRuntimeSessionView(
			t,
			server.URL,
			startResponse.SessionID,
			startResponse.RuntimeToken,
		)

	if sessionView.ID !=
		startResponse.SessionID ||
		sessionView.DeploymentVersion != 1 ||
		sessionView.SessionKind !=
			models.AssistantRuntimeSessionKindExternal ||
		sessionView.Status !=
			models.AssistantRuntimeSessionStatusActive ||
		sessionView.TurnCount != 0 ||
		sessionView.MaxTurns != 5 ||
		sessionView.RemainingTurns != 5 ||
		len(sessionView.Messages) != 0 {
		t.Fatalf(
			"公开会话视图错误: %+v",
			sessionView,
		)
	}

	assertAssistantRuntimeNoSensitiveFields(
		t,
		string(viewResult.Body),
	)
}

// TestAssistantRuntimeHTTPTokenIsolationAndPathBinding 验证令牌隔离和路径绑定。
func TestAssistantRuntimeHTTPTokenIsolationAndPathBinding(
	t *testing.T,
) {
	server,
		_ :=
		SetupTestServer(
			t,
		)

	CleanAndSeed(t)

	fixture := SeedAssistantRuntimeFixture(
		t,
	)

	deployment,
		_ :=
		fixture.CreateDeployment(
			t,
		)

	startResponse,
		_ :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_abcdef",
		)

	tamperedResult :=
		requestAssistantRuntimeSessionView(
			t,
			server.URL,
			startResponse.SessionID,
			tamperAssistantRuntimeToken(
				startResponse.RuntimeToken,
			),
		)

	if tamperedResult.Response.StatusCode !=
		http.StatusUnauthorized {
		t.Fatalf(
			"篡改运行令牌应返回401: HTTP=%d body=%s",
			tamperedResult.Response.StatusCode,
			string(tamperedResult.Body),
		)
	}

	wrongPathResult :=
		requestAssistantRuntimeSessionView(
			t,
			server.URL,
			"29999999-0000-4000-8000-000000000099",
			startResponse.RuntimeToken,
		)

	if wrongPathResult.Response.StatusCode !=
		http.StatusUnauthorized {
		t.Fatalf(
			"运行令牌串会话路径应返回401: HTTP=%d body=%s",
			wrongPathResult.Response.StatusCode,
			string(wrongPathResult.Body),
		)
	}

	teacherToken := LoginAsOperator(
		t,
		server.URL,
	)

	teacherTokenResult :=
		requestAssistantRuntimeSessionView(
			t,
			server.URL,
			startResponse.SessionID,
			teacherToken,
		)

	if teacherTokenResult.Response.StatusCode !=
		http.StatusUnauthorized {
		t.Fatalf(
			"教师JWT不得作为运行令牌使用: HTTP=%d body=%s",
			teacherTokenResult.Response.StatusCode,
			string(teacherTokenResult.Body),
		)
	}
}

// TestAssistantRuntimeHTTPOriginAndInputSecurity 验证Origin和正文边界。
func TestAssistantRuntimeHTTPOriginAndInputSecurity(
	t *testing.T,
) {
	server,
		_ :=
		SetupTestServer(
			t,
		)

	CleanAndSeed(t)

	fixture := SeedAssistantRuntimeFixture(
		t,
	)

	deployment,
		_ :=
		fixture.CreateDeployment(
			t,
		)

	evilOriginResult :=
		requestAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			"https://evil.example",
			"anonymous_client_origin1",
		)

	if evilOriginResult.Response.StatusCode !=
		http.StatusForbidden {
		t.Fatalf(
			"未授权Origin应返回403: HTTP=%d body=%s",
			evilOriginResult.Response.StatusCode,
			string(evilOriginResult.Body),
		)
	}

	missingOriginResult :=
		requestAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			"",
			"anonymous_client_origin2",
		)

	if missingOriginResult.Response.StatusCode !=
		http.StatusForbidden {
		t.Fatalf(
			"缺失Origin应返回403: HTTP=%d body=%s",
			missingOriginResult.Response.StatusCode,
			string(missingOriginResult.Body),
		)
	}

	bodyOriginResult :=
		doAssistantRuntimeHTTPRequest(
			t,
			http.MethodPost,
			server.URL+
				"/api/v1/assistant-runtime/deployments/"+
				deployment.PublicID+
				"/session",
			[]byte(
				`{
					"anonymous_client_id":"anonymous_client_origin3",
					"origin":"https://course.example"
				}`,
			),
			map[string]string{
				"Origin":
					AssistantFixtureOrigin,
				"X-Real-IP":
					"203.0.113.10",
			},
		)

	if bodyOriginResult.Response.StatusCode !=
		http.StatusBadRequest {
		t.Fatalf(
			"正文伪造Origin应被严格JSON拒绝: HTTP=%d body=%s",
			bodyOriginResult.Response.StatusCode,
			string(bodyOriginResult.Body),
		)
	}

	unknownIdentityResult :=
		doAssistantRuntimeHTTPRequest(
			t,
			http.MethodPost,
			server.URL+
				"/api/v1/assistant-runtime/deployments/"+
				deployment.PublicID+
				"/session",
			[]byte(
				`{
					"anonymous_client_id":"anonymous_client_origin4",
					"owner_user_id":"00000000-0000-0000-0000-000000000001"
				}`,
			),
			map[string]string{
				"Origin":
					AssistantFixtureOrigin,
				"X-Real-IP":
					"203.0.113.10",
			},
		)

	if unknownIdentityResult.Response.StatusCode !=
		http.StatusBadRequest {
		t.Fatalf(
			"正文伪造教师身份应被拒绝: HTTP=%d body=%s",
			unknownIdentityResult.Response.StatusCode,
			string(unknownIdentityResult.Body),
		)
	}

	var sessionCount int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM assistant_runtime_sessions
		`,
	).Scan(
		&sessionCount,
	); err != nil {
		t.Fatalf(
			"查询拒绝请求后的会话数量失败: %v",
			err,
		)
	}

	if sessionCount != 0 {
		t.Fatalf(
			"被拒绝的Origin或正文请求创建了会话: %d",
			sessionCount,
		)
	}
}
