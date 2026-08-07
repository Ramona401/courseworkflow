package integration

// assistant_runtime_settlement_test.go
//
// 使用真实tedna_test验证：
//   - 运行会话只保存哈希和Origin快照；
//   - 成功结算只扣费一次并追加两条正式消息；
//   - 失败结算不扣费、不增加轮数并释放主轮次；
//   - 成功和失败重复回调均按turn_id幂等拒绝。
//
// 本测试不调用真实AI，不访问外部网络。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// TestAssistantRuntimeRepositorySettlement 验证会话和结算主链。
func TestAssistantRuntimeRepositorySettlement(
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

	jtiHash := strings.Repeat(
		"f",
		64,
	)

	session := fixture.CreateRuntimeSession(
		t,
		deployment,
		AssistantFixtureSessionID,
		jtiHash,
	)

	assertStoredRuntimeSession(
		t,
		fixture,
		deployment,
		session,
		jtiHash,
	)

	successInput :=
		claimAndBuildSuccessSettlement(
			t,
			fixture,
			deployment,
			session,
		)

	settlement, err :=
		repository.CompleteAssistantRuntimeTurnSuccess(
			context.Background(),
			successInput,
		)
	if err != nil {
		t.Fatalf(
			"成功结算失败: %v",
			err,
		)
	}

	if settlement == nil ||
		settlement.Session == nil ||
		settlement.Account == nil ||
		settlement.Session.TurnCount != 1 ||
		settlement.Session.Status !=
			models.AssistantRuntimeSessionStatusActive ||
		settlement.BalanceAfter != 997.5 {
		t.Fatalf(
			"成功结算返回异常: %+v",
			settlement,
		)
	}

	assertSuccessfulSettlementStored(
		t,
		fixture,
		session,
	)

	if _,
		err := repository.CompleteAssistantRuntimeTurnSuccess(
		context.Background(),
		successInput,
	); !errors.Is(
		err,
		repository.ErrAssistantRuntimeTurnAlreadyFinalized,
	) {
		t.Fatalf(
			"重复成功结算未被幂等拒绝: %v",
			err,
		)
	}

	failureInput :=
		claimAndBuildFailureSettlement(
			t,
			fixture,
			deployment,
			session,
		)

	failedSession, err :=
		repository.CompleteAssistantRuntimeTurnFailure(
			context.Background(),
			failureInput,
		)
	if err != nil {
		t.Fatalf(
			"失败结算失败: %v",
			err,
		)
	}

	if failedSession == nil ||
		failedSession.TurnCount != 1 ||
		failedSession.ActiveTurnID != nil {
		t.Fatalf(
			"失败结算会话结果异常: %+v",
			failedSession,
		)
	}

	assertFailedSettlementStored(
		t,
		fixture,
		session,
	)

	if _,
		err := repository.CompleteAssistantRuntimeTurnFailure(
		context.Background(),
		failureInput,
	); !errors.Is(
		err,
		repository.ErrAssistantRuntimeTurnAlreadyFinalized,
	) {
		t.Fatalf(
			"重复失败结算未被幂等拒绝: %v",
			err,
		)
	}
}

// assertStoredRuntimeSession 验证会话只保存哈希。
func assertStoredRuntimeSession(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	session *models.AssistantRuntimeSession,
	expectedJTIHash string,
) {
	t.Helper()

	var (
		storedJTIHash       string
		storedClientHash    string
		storedIPHash        string
		storedOrigin        string
		storedDeploymentID  string
		storedVersionNumber int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			token_jti_hash,
			anonymous_client_hash,
			ip_hash,
			origin_snapshot,
			deployment_id::text,
			deployment_version
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		session.ID,
	).Scan(
		&storedJTIHash,
		&storedClientHash,
		&storedIPHash,
		&storedOrigin,
		&storedDeploymentID,
		&storedVersionNumber,
	); err != nil {
		t.Fatalf(
			"查询运行会话隐私字段失败: %v",
			err,
		)
	}

	if storedJTIHash != expectedJTIHash ||
		len(storedClientHash) != 64 ||
		len(storedIPHash) != 64 ||
		storedClientHash == storedIPHash ||
		storedOrigin != AssistantFixtureOrigin ||
		storedDeploymentID != deployment.ID ||
		storedVersionNumber != 1 {
		t.Fatalf(
			"运行会话字段异常: jti=%s client=%s ip=%s origin=%s deployment=%s version=%d",
			storedJTIHash,
			storedClientHash,
			storedIPHash,
			storedOrigin,
			storedDeploymentID,
			storedVersionNumber,
		)
	}

	if fixture == nil {
		t.Fatal(
			"教学智能体测试夹具为nil",
		)
	}
}

// claimAndBuildSuccessSettlement 领取成功主轮次并构造结算输入。
func claimAndBuildSuccessSettlement(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	session *models.AssistantRuntimeSession,
) *repository.AssistantRuntimeSuccessSettlementInput {
	t.Helper()

	claim, err := repository.ClaimAssistantRuntimeTurn(
		context.Background(),
		session.ID,
		deployment.ID,
		deployment.CurrentVersion,
		AssistantFixtureSuccessTurnID,
	)
	if err != nil {
		t.Fatalf(
			"领取成功结算主轮次失败: %v",
			err,
		)
	}

	messageTime := time.Now().UTC()

	return &repository.AssistantRuntimeSuccessSettlementInput{
		TurnID:
			claim.TurnID,
		SessionID:
			session.ID,
		DeploymentID:
			deployment.ID,
		DeploymentVersion:
			deployment.CurrentVersion,
		OwnerUserID:
			SeedOperatorID,
		SchoolID:
			fixture.SchoolID,
		CoursewareID:
			fixture.CoursewareID,
		PageID:
			fixture.PageID,
		SceneCode:
			"courseware_assistant_runtime",
		StudentMessage:
			models.AssistantRuntimeMessage{
				Role:
					models.AssistantRuntimeMessageRoleStudent,
				Content:
					"我先把两个三角形拼在一起。",
				CreatedAt:
					&messageTime,
			},
		AssistantMessage:
			models.AssistantRuntimeMessage{
				Role:
					models.AssistantRuntimeMessageRoleAssistant,
				Content:
					"拼成的图形和原三角形面积有什么关系？",
				CreatedAt:
					&messageTime,
			},
		InputChars:   15,
		OutputChars:  20,
		InputTokens:  40,
		OutputTokens: 30,
		ModelName:    "integration-test-model",
		Provider:     "integration",
		LatencyMs:    120,
		Calculation:
			&models.CreditCalculation{
				CostUSD:         0.01,
				ExchangeRate:    7,
				Multiplier:      1,
				CreditsConsumed: 2.5,
			},
	}
}

// assertSuccessfulSettlementStored 验证成功结算持久化结果。
func assertSuccessfulSettlementStored(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	session *models.AssistantRuntimeSession,
) {
	t.Helper()

	var (
		balance          float64
		totalConsumed    float64
		turnCount        int
		activeTurnID     *string
		messageCount     int
		successUsage     int
		consumptionCount int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			ta.balance,
			ta.total_consumed,
			s.turn_count,
			s.active_turn_id::text,
			jsonb_array_length(s.messages_json),
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage AS usage
				WHERE usage.turn_id = $1
				  AND usage.status = 'succeeded'
			),
			(
				SELECT COUNT(*)::integer
				FROM token_consumption_logs AS log
				WHERE log.memo = $2
			)
		FROM token_accounts AS ta
		JOIN assistant_runtime_sessions AS s
		  ON s.id = $3
		WHERE ta.id = $4
		`,
		AssistantFixtureSuccessTurnID,
		"assistant_runtime_turn:"+
			AssistantFixtureSuccessTurnID,
		session.ID,
		fixture.TokenAccountID,
	).Scan(
		&balance,
		&totalConsumed,
		&turnCount,
		&activeTurnID,
		&messageCount,
		&successUsage,
		&consumptionCount,
	); err != nil {
		t.Fatalf(
			"查询成功结算数据库结果失败: %v",
			err,
		)
	}

	if balance != 997.5 ||
		totalConsumed != 2.5 ||
		turnCount != 1 ||
		activeTurnID != nil ||
		messageCount != 2 ||
		successUsage != 1 ||
		consumptionCount != 1 {
		t.Fatalf(
			"成功结算异常: balance=%.4f consumed=%.4f turn=%d active=%v messages=%d usage=%d logs=%d",
			balance,
			totalConsumed,
			turnCount,
			activeTurnID,
			messageCount,
			successUsage,
			consumptionCount,
		)
	}
}

// claimAndBuildFailureSettlement 领取失败主轮次并构造输入。
func claimAndBuildFailureSettlement(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	session *models.AssistantRuntimeSession,
) *repository.AssistantRuntimeFailureSettlementInput {
	t.Helper()

	claim, err := repository.ClaimAssistantRuntimeTurn(
		context.Background(),
		session.ID,
		deployment.ID,
		deployment.CurrentVersion,
		AssistantFixtureFailureTurnID,
	)
	if err != nil {
		t.Fatalf(
			"领取失败结算主轮次失败: %v",
			err,
		)
	}

	return &repository.AssistantRuntimeFailureSettlementInput{
		TurnID:
			claim.TurnID,
		SessionID:
			session.ID,
		DeploymentID:
			deployment.ID,
		DeploymentVersion:
			deployment.CurrentVersion,
		OwnerUserID:
			SeedOperatorID,
		SchoolID:
			fixture.SchoolID,
		CoursewareID:
			fixture.CoursewareID,
		PageID:
			fixture.PageID,
		SceneCode:
			"courseware_assistant_runtime",
		SessionKind:
			models.AssistantRuntimeSessionKindExternal,
		InputChars:
			10,
		ErrorCode:
			"ai_stream_failed",
		ModelName:
			"integration-test-model",
		Provider:
			"integration",
		LatencyMs:
			80,
	}
}

// assertFailedSettlementStored 验证失败结算没有新增扣费。
func assertFailedSettlementStored(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	session *models.AssistantRuntimeSession,
) {
	t.Helper()

	var (
		finalBalance      float64
		finalConsumed     float64
		finalTurnCount    int
		finalActiveTurnID *string
		finalMessageCount int
		failedUsageCount  int
		finalLogCount     int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			ta.balance,
			ta.total_consumed,
			s.turn_count,
			s.active_turn_id::text,
			jsonb_array_length(s.messages_json),
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage AS usage
				WHERE usage.turn_id = $1
				  AND usage.status = 'failed'
				  AND usage.credits_used = 0
			),
			(
				SELECT COUNT(*)::integer
				FROM token_consumption_logs AS log
				WHERE log.memo LIKE 'assistant_runtime_turn:%'
			)
		FROM token_accounts AS ta
		JOIN assistant_runtime_sessions AS s
		  ON s.id = $2
		WHERE ta.id = $3
		`,
		AssistantFixtureFailureTurnID,
		session.ID,
		fixture.TokenAccountID,
	).Scan(
		&finalBalance,
		&finalConsumed,
		&finalTurnCount,
		&finalActiveTurnID,
		&finalMessageCount,
		&failedUsageCount,
		&finalLogCount,
	); err != nil {
		t.Fatalf(
			"查询失败结算数据库结果失败: %v",
			err,
		)
	}

	if finalBalance != 997.5 ||
		finalConsumed != 2.5 ||
		finalTurnCount != 1 ||
		finalActiveTurnID != nil ||
		finalMessageCount != 2 ||
		failedUsageCount != 1 ||
		finalLogCount != 1 {
		t.Fatalf(
			"失败结算异常: balance=%.4f consumed=%.4f turn=%d active=%v messages=%d usage=%d logs=%d",
			finalBalance,
			finalConsumed,
			finalTurnCount,
			finalActiveTurnID,
			finalMessageCount,
			failedUsageCount,
			finalLogCount,
		)
	}
}
