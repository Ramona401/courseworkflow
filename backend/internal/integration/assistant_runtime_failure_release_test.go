package integration

// assistant_runtime_failure_release_test.go
//
// 使用真实tedna_test验证取消和超时失败结算：
//   - 写入一条failed usage；
//   - error_code保持稳定公开码；
//   - 不扣积分、不写Token消费流水；
//   - 不增加成功轮数、不追加正式消息；
//   - active_turn_id和active_turn_started_at同时清空；
//   - 相同turn_id重复失败结算按幂等规则拒绝。
//
// 浏览器取消与父Context解耦由services包单元测试验证。
// 本测试不调用真实AI。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/database"
	"tedna/internal/repository"
)

// TestAssistantRuntimeFailureSettlementReleasesCancelledAndTimedOutTurns 验证失败释放。
func TestAssistantRuntimeFailureSettlementReleasesCancelledAndTimedOutTurns(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
	)

	cases := []struct {
		name      string
		errorCode string
	}{
		{
			name:
				"client cancellation",
			errorCode:
				"client_cancelled",
		},
		{
			name:
				"runtime timeout",
			errorCode:
				"runtime_timeout",
		},
	}

	for caseIndex,
		testCase :=
		range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				CleanAndSeed(t)

				fixture :=
					SeedAssistantRuntimeFixture(
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
						0x44000000,
						uint64(
							caseIndex+1,
						),
					)

				turnID :=
					assistantRuntimeIntegrationUUID(
						0x44100000,
						uint64(
							caseIndex+1,
						),
					)

				claim,
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
						"领取失败结算测试轮次失败: %v",
						err,
					)
				}

				if claim == nil ||
					claim.TurnID !=
						turnID {
					t.Fatalf(
						"失败结算测试领取结果错误: %+v",
						claim,
					)
				}

				input :=
					assistantRuntimeFailureSettlementInput(
						fixture,
						deployment,
						session,
						turnID,
					)

				input.ErrorCode =
					testCase.errorCode

				releasedSession,
					err :=
					repository.CompleteAssistantRuntimeTurnFailure(
						context.Background(),
						input,
					)
				if err != nil {
					t.Fatalf(
						"完成%s失败结算失败: %v",
						testCase.errorCode,
						err,
					)
				}

				if releasedSession == nil ||
					releasedSession.TurnCount != 0 ||
					releasedSession.ActiveTurnID != nil ||
					releasedSession.ActiveTurnStartedAt != nil {
					t.Fatalf(
						"失败结算返回的会话没有正确释放: %+v",
						releasedSession,
					)
				}

				assertAssistantRuntimeFailureReleaseStored(
					t,
					fixture,
					session.ID,
					turnID,
					testCase.errorCode,
				)

				_,
					err =
					repository.CompleteAssistantRuntimeTurnFailure(
						context.Background(),
						input,
					)

				if !errors.Is(
					err,
					repository.ErrAssistantRuntimeTurnAlreadyFinalized,
				) {
					t.Fatalf(
						"重复失败结算未被幂等拒绝: %v",
						err,
					)
				}

				assertAssistantRuntimeFailureReleaseStored(
					t,
					fixture,
					session.ID,
					turnID,
					testCase.errorCode,
				)
			},
		)
	}
}

// assertAssistantRuntimeFailureReleaseStored 验证失败结算持久化结果。
func assertAssistantRuntimeFailureReleaseStored(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	sessionID string,
	turnID string,
	expectedErrorCode string,
) {
	t.Helper()

	var (
		balance       float64
		totalConsumed float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			balance,
			total_consumed
		FROM token_accounts
		WHERE id = $1
		`,
		fixture.TokenAccountID,
	).Scan(
		&balance,
		&totalConsumed,
	); err != nil {
		t.Fatalf(
			"读取失败结算积分账户失败: %v",
			err,
		)
	}

	if balance != 1000 ||
		totalConsumed != 0 {
		t.Fatalf(
			"失败结算错误扣费: balance=%.4f consumed=%.4f",
			balance,
			totalConsumed,
		)
	}

	var (
		turnCount             int
		activeTurnCleared     bool
		activeStartedCleared  bool
		messageCount          int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			turn_count,
			active_turn_id IS NULL,
			active_turn_started_at IS NULL,
			jsonb_array_length(messages_json)
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		sessionID,
	).Scan(
		&turnCount,
		&activeTurnCleared,
		&activeStartedCleared,
		&messageCount,
	); err != nil {
		t.Fatalf(
			"读取失败结算会话失败: %v",
			err,
		)
	}

	if turnCount != 0 ||
		!activeTurnCleared ||
		!activeStartedCleared ||
		messageCount != 0 {
		t.Fatalf(
			"失败结算会话状态错误: turn=%d active_cleared=%t started_cleared=%t messages=%d",
			turnCount,
			activeTurnCleared,
			activeStartedCleared,
			messageCount,
		)
	}

	var (
		usageCount int
		status     string
		errorCode  string
		credits    float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			COUNT(*)::integer,
			COALESCE(MAX(status), ''),
			COALESCE(MAX(error_code), ''),
			COALESCE(SUM(credits_used), 0)
		FROM assistant_runtime_usage
		WHERE turn_id = $1
		`,
		turnID,
	).Scan(
		&usageCount,
		&status,
		&errorCode,
		&credits,
	); err != nil {
		t.Fatalf(
			"读取失败结算流水失败: %v",
			err,
		)
	}

	if usageCount != 1 ||
		status != "failed" ||
		errorCode != expectedErrorCode ||
		credits != 0 {
		t.Fatalf(
			"失败结算流水错误: count=%d status=%s code=%s credits=%.4f",
			usageCount,
			status,
			errorCode,
			credits,
		)
	}

	var consumptionLogCount int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM token_consumption_logs
		`,
	).Scan(
		&consumptionLogCount,
	); err != nil {
		t.Fatalf(
			"读取失败结算Token消费流水失败: %v",
			err,
		)
	}

	if consumptionLogCount != 0 {
		t.Fatalf(
			"失败结算写入了Token消费流水: %d",
			consumptionLogCount,
		)
	}
}
