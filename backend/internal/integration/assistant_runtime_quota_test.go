package integration

// assistant_runtime_quota_test.go
//
// 使用真实tedna_test验证每日额度规则：
//   - 当天成功流水计入每日额度；
//   - 最近20分钟的在途主轮次也计入每日额度；
//   - 两者相加达到上限时拒绝新的领取；
//   - 超过20分钟的陈旧占位不继续占用每日额度；
//   - 失败结算不会增加当天成功调用数。
//
// 本测试不调用真实AI。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/database"
	"tedna/internal/repository"
)

// TestAssistantRuntimeDailyQuotaCountsSuccessAndRecentActive 验证额度合计。
func TestAssistantRuntimeDailyQuotaCountsSuccessAndRecentActive(
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

	_,
		err :=
		database.DB.Exec(
			context.Background(),
			`
			UPDATE assistant_deployments
			SET daily_call_limit = 2
			WHERE id = $1
			`,
			deployment.ID,
		)
	if err != nil {
		t.Fatalf(
			"设置每日额度失败: %v",
			err,
		)
	}

	successSession :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x42000000,
			1,
		)

	successTurnID :=
		assistantRuntimeIntegrationUUID(
			0x42100000,
			1,
		)

	_,
		err =
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			successSession.ID,
			deployment.ID,
			deployment.CurrentVersion,
			successTurnID,
		)
	if err != nil {
		t.Fatalf(
			"领取成功额度测试轮次失败: %v",
			err,
		)
	}

	_,
		err =
		repository.CompleteAssistantRuntimeTurnSuccess(
			context.Background(),
			assistantRuntimeSuccessSettlementInput(
				fixture,
				deployment,
				successSession,
				successTurnID,
				0,
			),
		)
	if err != nil {
		t.Fatalf(
			"完成成功额度测试轮次失败: %v",
			err,
		)
	}

	activeSession :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x42000000,
			2,
		)

	activeTurnID :=
		assistantRuntimeIntegrationUUID(
			0x42100000,
			2,
		)

	_,
		err =
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			activeSession.ID,
			deployment.ID,
			deployment.CurrentVersion,
			activeTurnID,
		)
	if err != nil {
		t.Fatalf(
			"领取近期在途额度测试轮次失败: %v",
			err,
		)
	}

	rejectedSession :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x42000000,
			3,
		)

	_,
		err =
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			rejectedSession.ID,
			deployment.ID,
			deployment.CurrentVersion,
			assistantRuntimeIntegrationUUID(
				0x42100000,
				3,
			),
		)

	if !errors.Is(
		err,
		repository.ErrAssistantRuntimeDailyQuotaExceeded,
	) {
		t.Fatalf(
			"成功流水加近期在途达到上限时未拒绝: %v",
			err,
		)
	}

	if _,
		err :=
		repository.CompleteAssistantRuntimeTurnFailure(
			context.Background(),
			assistantRuntimeFailureSettlementInput(
				fixture,
				deployment,
				activeSession,
				activeTurnID,
			),
		); err != nil {
		t.Fatalf(
			"清理近期在途轮次失败: %v",
			err,
		)
	}

	var (
		succeededToday int
		activeCount    int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage
				WHERE deployment_id = $1
				  AND status = 'succeeded'
				  AND created_at >= CURRENT_DATE
			),
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_sessions
				WHERE deployment_id = $1
				  AND active_turn_id IS NOT NULL
			)
		`,
		deployment.ID,
	).Scan(
		&succeededToday,
		&activeCount,
	); err != nil {
		t.Fatalf(
			"查询每日额度最终状态失败: %v",
			err,
		)
	}

	if succeededToday != 1 ||
		activeCount != 0 {
		t.Fatalf(
			"每日额度最终状态错误: succeeded=%d active=%d",
			succeededToday,
			activeCount,
		)
	}
}

// TestAssistantRuntimeDailyQuotaIgnoresStaleActiveClaim 验证陈旧占位窗口。
func TestAssistantRuntimeDailyQuotaIgnoresStaleActiveClaim(
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

	_,
		err :=
		database.DB.Exec(
			context.Background(),
			`
			UPDATE assistant_deployments
			SET daily_call_limit = 1
			WHERE id = $1
			`,
			deployment.ID,
		)
	if err != nil {
		t.Fatalf(
			"设置陈旧占位额度失败: %v",
			err,
		)
	}

	staleSession :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x42200000,
			1,
		)

	staleTurnID :=
		assistantRuntimeIntegrationUUID(
			0x42300000,
			1,
		)

	_,
		err =
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			staleSession.ID,
			deployment.ID,
			deployment.CurrentVersion,
			staleTurnID,
		)
	if err != nil {
		t.Fatalf(
			"领取陈旧占位测试轮次失败: %v",
			err,
		)
	}

	_,
		err =
		database.DB.Exec(
			context.Background(),
			`
			UPDATE assistant_runtime_sessions
			SET
				active_turn_started_at =
					NOW() - INTERVAL '21 minutes',
				updated_at = NOW()
			WHERE id = $1
			`,
			staleSession.ID,
		)
	if err != nil {
		t.Fatalf(
			"设置21分钟前占位失败: %v",
			err,
		)
	}

	newSession :=
		createAssistantRuntimeIsolatedSession(
			t,
			fixture,
			deployment,
			0x42200000,
			2,
		)

	newTurnID :=
		assistantRuntimeIntegrationUUID(
			0x42300000,
			2,
		)

	newClaim,
		err :=
		repository.ClaimAssistantRuntimeTurn(
			context.Background(),
			newSession.ID,
			deployment.ID,
			deployment.CurrentVersion,
			newTurnID,
		)
	if err != nil {
		t.Fatalf(
			"21分钟前陈旧占位仍阻塞每日额度: %v",
			err,
		)
	}

	if newClaim == nil ||
		newClaim.TurnID !=
			newTurnID {
		t.Fatalf(
			"陈旧占位窗口后的新领取结果错误: %+v",
			newClaim,
		)
	}

	for _,
		item :=
		range []struct {
			sessionID string
			turnID    string
		}{
			{
				sessionID:
					staleSession.ID,
				turnID:
					staleTurnID,
			},
			{
				sessionID:
					newSession.ID,
				turnID:
					newTurnID,
			},
		} {
		session := staleSession

		if item.sessionID ==
			newSession.ID {
			session = newSession
		}

		if _,
			err :=
			repository.CompleteAssistantRuntimeTurnFailure(
				context.Background(),
				assistantRuntimeFailureSettlementInput(
					fixture,
					deployment,
					session,
					item.turnID,
				),
			); err != nil {
			t.Fatalf(
				"清理陈旧窗口测试轮次失败(session=%s): %v",
				item.sessionID,
				err,
			)
		}
	}

	var (
		failedUsageCount int
		activeTurnCount  int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage
				WHERE deployment_id = $1
				  AND status = 'failed'
			),
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_sessions
				WHERE deployment_id = $1
				  AND (
						active_turn_id IS NOT NULL
						OR active_turn_started_at IS NOT NULL
				  )
			)
		`,
		deployment.ID,
	).Scan(
		&failedUsageCount,
		&activeTurnCount,
	); err != nil {
		t.Fatalf(
			"查询陈旧窗口最终状态失败: %v",
			err,
		)
	}

	if failedUsageCount != 2 ||
		activeTurnCount != 0 {
		t.Fatalf(
			"陈旧窗口最终状态错误: failed=%d active=%d",
			failedUsageCount,
			activeTurnCount,
		)
	}
}
