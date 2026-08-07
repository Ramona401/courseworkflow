package integration

// assistant_runtime_http_state_test.go
//
// 使用真实教师HTTP状态入口和公开运行HTTP入口验证：
//   - paused部署允许读取已有会话，但拒绝建立新会话；
//   - resume后允许再次建立新会话；
//   - revoked部署拒绝读取和新建，并使活动会话进入revoked；
//   - revoked部署无法恢复；
//   - 发布新不可变版本后旧运行令牌被拒绝并撤销旧会话；
//   - 新建会话自动绑定新的current_version。
//
// 本测试不调用聊天接口和真实AI。

import (
	"context"
	"net/http"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// TestAssistantRuntimeHTTPPauseAndResume 验证暂停和恢复。
func TestAssistantRuntimeHTTPPauseAndResume(
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
			"anonymous_client_pause1",
		)

	teacherToken := LoginAsOperator(
		t,
		server.URL,
	)

	pauseResponse,
		pauseAPI :=
		DoPost(
			t,
			server.URL+
				"/api/v1/assistant-deployments/"+
				deployment.ID+
				"/pause",
			nil,
			teacherToken,
		)

	AssertHTTPStatus(
		t,
		pauseResponse,
		http.StatusOK,
	)
	AssertAPICode(
		t,
		pauseAPI,
		0,
	)

	var pausedView models.AssistantDeploymentView

	ParseData(
		t,
		pauseAPI,
		&pausedView,
	)

	if pausedView.Status !=
		models.AssistantDeploymentStatusPaused {
		t.Fatalf(
			"教师暂停部署状态错误: %+v",
			pausedView,
		)
	}

	// paused仍允许令牌持有者读取已有会话视图。
	sessionView,
		_ :=
		readAssistantRuntimeSessionView(
			t,
			server.URL,
			startResponse.SessionID,
			startResponse.RuntimeToken,
		)

	if sessionView.Status !=
		models.AssistantRuntimeSessionStatusActive {
		t.Fatalf(
			"暂停部署不应立即撤销已有会话: %+v",
			sessionView,
		)
	}

	pausedStartResult :=
		requestAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_pause2",
		)

	if pausedStartResult.Response.StatusCode !=
		http.StatusNotFound {
		t.Fatalf(
			"暂停部署仍创建了新会话: HTTP=%d body=%s",
			pausedStartResult.Response.StatusCode,
			string(pausedStartResult.Body),
		)
	}

	resumeResponse,
		resumeAPI :=
		DoPost(
			t,
			server.URL+
				"/api/v1/assistant-deployments/"+
				deployment.ID+
				"/resume",
			nil,
			teacherToken,
		)

	AssertHTTPStatus(
		t,
		resumeResponse,
		http.StatusOK,
	)
	AssertAPICode(
		t,
		resumeAPI,
		0,
	)

	var resumedView models.AssistantDeploymentView

	ParseData(
		t,
		resumeAPI,
		&resumedView,
	)

	if resumedView.Status !=
		models.AssistantDeploymentStatusActive {
		t.Fatalf(
			"教师恢复部署状态错误: %+v",
			resumedView,
		)
	}

	secondSession,
		_ :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_pause3",
		)

	if secondSession.SessionID ==
		startResponse.SessionID {
		t.Fatal(
			"恢复后新会话不应复用旧session_id",
		)
	}
}

// TestAssistantRuntimeHTTPRevokeIsPermanent 验证撤销不可恢复。
func TestAssistantRuntimeHTTPRevokeIsPermanent(
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
			"anonymous_client_revoke1",
		)

	teacherToken := LoginAsOperator(
		t,
		server.URL,
	)

	revokeResponse,
		revokeAPI :=
		DoPost(
			t,
			server.URL+
				"/api/v1/assistant-deployments/"+
				deployment.ID+
				"/revoke",
			nil,
			teacherToken,
		)

	AssertHTTPStatus(
		t,
		revokeResponse,
		http.StatusOK,
	)
	AssertAPICode(
		t,
		revokeAPI,
		0,
	)

	existingSessionResult :=
		requestAssistantRuntimeSessionView(
			t,
			server.URL,
			startResponse.SessionID,
			startResponse.RuntimeToken,
		)

	if existingSessionResult.Response.StatusCode !=
		http.StatusNotFound {
		t.Fatalf(
			"撤销部署后旧会话仍可读取: HTTP=%d body=%s",
			existingSessionResult.Response.StatusCode,
			string(existingSessionResult.Body),
		)
	}

	var storedSessionStatus string

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT status
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		startResponse.SessionID,
	).Scan(
		&storedSessionStatus,
	); err != nil {
		t.Fatalf(
			"读取撤销后的运行会话状态失败: %v",
			err,
		)
	}

	if storedSessionStatus !=
		models.AssistantRuntimeSessionStatusRevoked {
		t.Fatalf(
			"撤销部署没有同步撤销活动会话: %s",
			storedSessionStatus,
		)
	}

	newSessionResult :=
		requestAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_revoke2",
		)

	if newSessionResult.Response.StatusCode !=
		http.StatusNotFound {
		t.Fatalf(
			"撤销部署仍可创建新会话: HTTP=%d body=%s",
			newSessionResult.Response.StatusCode,
			string(newSessionResult.Body),
		)
	}

	resumeResponse,
		resumeAPI :=
		DoPost(
			t,
			server.URL+
				"/api/v1/assistant-deployments/"+
				deployment.ID+
				"/resume",
			nil,
			teacherToken,
		)

	if resumeResponse.StatusCode !=
		http.StatusConflict ||
		resumeAPI == nil ||
		resumeAPI.Code == 0 {
		t.Fatalf(
			"撤销部署被错误恢复: HTTP=%d response=%+v",
			resumeResponse.StatusCode,
			resumeAPI,
		)
	}
}

// TestAssistantRuntimeHTTPOldVersionInvalidation 验证旧版本令牌失效。
func TestAssistantRuntimeHTTPOldVersionInvalidation(
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

	oldSession,
		_ :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_version1",
		)

	_,
		newVersion :=
		fixture.NewDeploymentRecords()

	newVersion.AssistantPromptSnapshot =
		"这是版本2的不可变提示词快照。"

	createdVersion,
		err :=
		repository.AppendAssistantDeploymentVersion(
			context.Background(),
			deployment.ID,
			deployment.CoursewareID,
			deployment.PageID,
			deployment.OwnerUserID,
			newVersion,
		)
	if err != nil {
		t.Fatalf(
			"追加版本2失败: %v",
			err,
		)
	}

	if createdVersion.Version != 2 {
		t.Fatalf(
			"追加版本号错误: %+v",
			createdVersion,
		)
	}

	oldSessionResult :=
		requestAssistantRuntimeSessionView(
			t,
			server.URL,
			oldSession.SessionID,
			oldSession.RuntimeToken,
		)

	if oldSessionResult.Response.StatusCode !=
		http.StatusConflict {
		t.Fatalf(
			"旧版本运行令牌应返回409: HTTP=%d body=%s",
			oldSessionResult.Response.StatusCode,
			string(oldSessionResult.Body),
		)
	}

	var oldSessionStatus string

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT status
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		oldSession.SessionID,
	).Scan(
		&oldSessionStatus,
	); err != nil {
		t.Fatalf(
			"读取旧版本会话状态失败: %v",
			err,
		)
	}

	if oldSessionStatus !=
		models.AssistantRuntimeSessionStatusRevoked {
		t.Fatalf(
			"旧版本会话没有被撤销: %s",
			oldSessionStatus,
		)
	}

	newSession,
		_ :=
		startAssistantRuntimeSession(
			t,
			server.URL,
			deployment.PublicID,
			AssistantFixtureOrigin,
			"anonymous_client_version2",
		)

	newClaims := parseAssistantRuntimeTestTokenClaims(
		t,
		newSession.RuntimeToken,
	)

	if newClaims.DeploymentVersion != 2 {
		t.Fatalf(
			"新运行令牌没有绑定current_version=2: %+v",
			newClaims,
		)
	}

	var storedNewVersion int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT deployment_version
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		newSession.SessionID,
	).Scan(
		&storedNewVersion,
	); err != nil {
		t.Fatalf(
			"读取新版本会话失败: %v",
			err,
		)
	}

	if storedNewVersion != 2 {
		t.Fatalf(
			"新会话数据库版本错误: %d",
			storedNewVersion,
		)
	}
}
