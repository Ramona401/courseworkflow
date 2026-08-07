package integration

// assistant_runtime_settlement_transition_test.go
//
// 使用真实tedna_test验证AI调用已经发生后的结算边界：
//   - 主轮次领取后教师暂停部署，已发生调用仍按原身份完成扣费；
//   - 主轮次领取后教师发布版本2，版本1会话的已发生调用仍完成扣费；
//   - 结算不能恢复部署状态或覆盖current_version；
//   - usage必须保存领取时的deployment_version；
//   - 成功结算只写一条usage、一条Token消费流水和两条正式消息。
//
// 本测试不调用真实AI。

import (
	"context"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// TestAssistantRuntimeSettlementAfterDeploymentPause 验证暂停不抹去成本。
func TestAssistantRuntimeSettlementAfterDeploymentPause(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
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

	session :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x45000000,
			1,
		)

	turnID :=
		assistantRuntimeIntegrationUUID(
			0x45100000,
			1,
		)

	_,
		err :=
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			session.ID,
			deployment.ID,
			deployment.CurrentVersion,
			turnID,
		)
	if err != nil {
		t.Fatalf(
			"领取暂停中途结算轮次失败: %v",
			err,
		)
	}

	paused,
		err :=
		repository.PauseAssistantDeployment(
			context.Background(),
			deployment.ID,
			deployment.OwnerUserID,
		)
	if err != nil {
		t.Fatalf(
			"调用途中暂停部署失败: %v",
			err,
		)
	}

	if paused == nil ||
		paused.Status !=
			models.AssistantDeploymentStatusPaused {
		t.Fatalf(
			"暂停部署返回错误: %+v",
			paused,
		)
	}

	const creditsUsed = 3.25

	settlement,
		err :=
		repository.CompleteAssistantRuntimeTurnSuccess(
			context.Background(),
			assistantRuntimeSuccessSettlementInput(
				fixture,
				deployment,
				session,
				turnID,
				creditsUsed,
			),
		)
	if err != nil {
		t.Fatalf(
			"暂停部署后已发生成本无法结算: %v",
			err,
		)
	}

	if settlement == nil ||
		settlement.Session == nil ||
		settlement.Session.TurnCount != 1 ||
		settlement.BalanceAfter !=
			1000-creditsUsed {
		t.Fatalf(
			"暂停部署后结算返回错误: %+v",
			settlement,
		)
	}

	assertAssistantRuntimeTransitionSettlementStored(
		t,
		fixture,
		deployment.ID,
		session.ID,
		turnID,
		models.AssistantDeploymentStatusPaused,
		1,
		1,
		creditsUsed,
	)
}

// TestAssistantRuntimeSettlementAfterVersionPublish 验证发版不抹去成本。
func TestAssistantRuntimeSettlementAfterVersionPublish(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
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

	session :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x45000000,
			2,
		)

	turnID :=
		assistantRuntimeIntegrationUUID(
			0x45100000,
			2,
		)

	_,
		err :=
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			session.ID,
			deployment.ID,
			deployment.CurrentVersion,
			turnID,
		)
	if err != nil {
		t.Fatalf(
			"领取发版中途结算轮次失败: %v",
			err,
		)
	}

	_,
		versionRecord :=
		fixture.NewDeploymentRecords()

	versionRecord.AssistantPromptSnapshot =
		"这是调用途中发布的版本2不可变提示词快照。"

	createdVersion,
		err :=
		repository.AppendAssistantDeploymentVersion(
			context.Background(),
			deployment.ID,
			deployment.CoursewareID,
			deployment.PageID,
			deployment.OwnerUserID,
			versionRecord,
		)
	if err != nil {
		t.Fatalf(
			"调用途中发布版本2失败: %v",
			err,
		)
	}

	if createdVersion == nil ||
		createdVersion.Version != 2 {
		t.Fatalf(
			"调用途中发布版本结果错误: %+v",
			createdVersion,
		)
	}

	const creditsUsed = 4.0

	settlement,
		err :=
		repository.CompleteAssistantRuntimeTurnSuccess(
			context.Background(),
			assistantRuntimeSuccessSettlementInput(
				fixture,
				deployment,
				session,
				turnID,
				creditsUsed,
			),
		)
	if err != nil {
		t.Fatalf(
			"发布版本2后版本1已发生成本无法结算: %v",
			err,
		)
	}

	if settlement == nil ||
		settlement.Session == nil ||
		settlement.Session.TurnCount != 1 ||
		settlement.BalanceAfter !=
			1000-creditsUsed {
		t.Fatalf(
			"发布版本2后结算返回错误: %+v",
			settlement,
		)
	}

	assertAssistantRuntimeTransitionSettlementStored(
		t,
		fixture,
		deployment.ID,
		session.ID,
		turnID,
		models.AssistantDeploymentStatusActive,
		2,
		1,
		creditsUsed,
	)
}

// assertAssistantRuntimeTransitionSettlementStored 验证状态变化后的成功结算。
func assertAssistantRuntimeTransitionSettlementStored(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	deploymentID string,
	sessionID string,
	turnID string,
	expectedDeploymentStatus string,
	expectedCurrentVersion int,
	expectedUsageVersion int,
	expectedCredits float64,
) {
	t.Helper()

	var (
		deploymentStatus string
		currentVersion   int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			status,
			current_version
		FROM assistant_deployments
		WHERE id = $1
		`,
		deploymentID,
	).Scan(
		&deploymentStatus,
		&currentVersion,
	); err != nil {
		t.Fatalf(
			"读取状态变化后的部署失败: %v",
			err,
		)
	}

	if deploymentStatus !=
		expectedDeploymentStatus ||
		currentVersion !=
			expectedCurrentVersion {
		t.Fatalf(
			"结算错误覆盖部署状态或版本: status=%s version=%d",
			deploymentStatus,
			currentVersion,
		)
	}

	var (
		turnCount            int
		sessionStatus        string
		activeTurnCleared    bool
		activeStartedCleared bool
		messageCount         int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			turn_count,
			status,
			active_turn_id IS NULL,
			active_turn_started_at IS NULL,
			jsonb_array_length(messages_json)
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		sessionID,
	).Scan(
		&turnCount,
		&sessionStatus,
		&activeTurnCleared,
		&activeStartedCleared,
		&messageCount,
	); err != nil {
		t.Fatalf(
			"读取状态变化后的结算会话失败: %v",
			err,
		)
	}

	if turnCount != 1 ||
		sessionStatus !=
			models.AssistantRuntimeSessionStatusActive ||
		!activeTurnCleared ||
		!activeStartedCleared ||
		messageCount != 2 {
		t.Fatalf(
			"状态变化后的会话结算错误: turn=%d status=%s active_cleared=%t started_cleared=%t messages=%d",
			turnCount,
			sessionStatus,
			activeTurnCleared,
			activeStartedCleared,
			messageCount,
		)
	}

	var (
		usageCount   int
		usageStatus  string
		usageVersion int
		usageCredits float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			COUNT(*)::integer,
			COALESCE(MAX(status), ''),
			COALESCE(MAX(deployment_version), 0),
			COALESCE(SUM(credits_used), 0)
		FROM assistant_runtime_usage
		WHERE turn_id = $1
		`,
		turnID,
	).Scan(
		&usageCount,
		&usageStatus,
		&usageVersion,
		&usageCredits,
	); err != nil {
		t.Fatalf(
			"读取状态变化后的usage失败: %v",
			err,
		)
	}

	if usageCount != 1 ||
		usageStatus != "succeeded" ||
		usageVersion !=
			expectedUsageVersion ||
		usageCredits !=
			expectedCredits {
		t.Fatalf(
			"状态变化后的usage错误: count=%d status=%s version=%d credits=%.4f",
			usageCount,
			usageStatus,
			usageVersion,
			usageCredits,
		)
	}

	var (
		balance          float64
		totalConsumed    float64
		logCount         int
		logAmount        float64
		logCredits       float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			ta.balance,
			ta.total_consumed,
			(
				SELECT COUNT(*)::integer
				FROM token_consumption_logs
				WHERE memo = $1
			),
			(
				SELECT COALESCE(SUM(amount), 0)
				FROM token_consumption_logs
				WHERE memo = $1
			),
			(
				SELECT COALESCE(SUM(credits_consumed), 0)
				FROM token_consumption_logs
				WHERE memo = $1
			)
		FROM token_accounts AS ta
		WHERE ta.id = $2
		`,
		"assistant_runtime_turn:"+
			turnID,
		fixture.TokenAccountID,
	).Scan(
		&balance,
		&totalConsumed,
		&logCount,
		&logAmount,
		&logCredits,
	); err != nil {
		t.Fatalf(
			"读取状态变化后的积分结算失败: %v",
			err,
		)
	}

	if balance !=
		1000-expectedCredits ||
		totalConsumed !=
			expectedCredits ||
		logCount != 1 ||
		logAmount !=
			expectedCredits ||
		logCredits !=
			expectedCredits {
		t.Fatalf(
			"状态变化后的积分结算错误: balance=%.4f consumed=%.4f logs=%d amount=%.4f credits=%.4f",
			balance,
			totalConsumed,
			logCount,
			logAmount,
			logCredits,
		)
	}
}
