package integration

// assistant_runtime_lock_order_test.go
//
// 回归验证领取和成功结算的统一锁顺序：
//
//     assistant_deployments
//       → assistant_runtime_sessions
//       → token_accounts
//
// 每轮先让会话持有一个主轮次，再同时执行：
//   - 对当前主轮次做成功结算；
//   - 对同一会话尝试领取下一个主轮次。
//
// 两种合法时序：
//   1. 新领取先取得部署锁，发现旧轮次仍在执行，返回TurnInProgress；
//   2. 结算先完成，新领取随后成功。
//
// 无论哪种时序，都不得出现数据库死锁、上下文超时、重复流水或悬空轮次。

import (
	"context"
	"errors"
	"testing"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// assistantRuntimeLockOperationResult 保存并发操作结果。
type assistantRuntimeLockOperationResult struct {
	Operation string
	Claim     *models.AssistantRuntimeTurnClaim
	Err       error
}

// TestAssistantRuntimeClaimAndSettlementLockOrderNoDeadlock 验证无循环等待。
func TestAssistantRuntimeClaimAndSettlementLockOrderNoDeadlock(
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

	const rounds = 12

	for round := 1; round <=
		rounds; round++ {
		session :=
			createAssistantRuntimeIsolatedSession(
				t,
				fixture,
				deployment,
				0x43000000,
				uint64(round),
			)

		currentTurnID :=
			assistantRuntimeIntegrationUUID(
				0x43100000,
				uint64(round*2-1),
			)

		nextTurnID :=
			assistantRuntimeIntegrationUUID(
				0x43100000,
				uint64(round*2),
			)

		_,
			err :=
			repository.ClaimAssistantRuntimeTurn(
				context.Background(),
				session.ID,
				deployment.ID,
				deployment.CurrentVersion,
				currentTurnID,
			)
		if err != nil {
			t.Fatalf(
				"第%d轮领取当前主轮次失败: %v",
				round,
				err,
			)
		}

		start := make(
			chan struct{},
		)

		results := make(
			chan assistantRuntimeLockOperationResult,
			2,
		)

		go func() {
			<-start

			ctx,
				cancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer cancel()

			_,
				settlementErr :=
				repository.CompleteAssistantRuntimeTurnSuccess(
					ctx,
					assistantRuntimeSuccessSettlementInput(
						fixture,
						deployment,
						session,
						currentTurnID,
						0,
					),
				)

			results <- assistantRuntimeLockOperationResult{
				Operation: "settlement",
				Err:       settlementErr,
			}
		}()

		go func() {
			<-start

			ctx,
				cancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer cancel()

			claim,
				claimErr :=
				repository.ClaimAssistantRuntimeTurn(
					ctx,
					session.ID,
					deployment.ID,
					deployment.CurrentVersion,
					nextTurnID,
				)

			results <- assistantRuntimeLockOperationResult{
				Operation: "claim",
				Claim:     claim,
				Err:       claimErr,
			}
		}()

		close(
			start,
		)

		var (
			settlementResult *assistantRuntimeLockOperationResult
			claimResult      *assistantRuntimeLockOperationResult
		)

		timer := time.NewTimer(
			7 * time.Second,
		)

		for received := 0; received <
			2; received++ {
			select {
			case result := <-results:
				resultCopy := result

				switch result.Operation {
				case "settlement":
					settlementResult =
						&resultCopy

				case "claim":
					claimResult =
						&resultCopy

				default:
					t.Fatalf(
						"第%d轮收到未知并发操作: %s",
						round,
						result.Operation,
					)
				}

			case <-timer.C:
				t.Fatalf(
					"第%d轮领取与结算并发超时，疑似锁顺序回归",
					round,
				)
			}
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		if settlementResult == nil ||
			settlementResult.Err != nil {
			t.Fatalf(
				"第%d轮成功结算异常: %+v",
				round,
				settlementResult,
			)
		}

		if claimResult == nil {
			t.Fatalf(
				"第%d轮缺少并发领取结果",
				round,
			)
		}

		switch {
		case claimResult.Err == nil:
			if claimResult.Claim == nil ||
				claimResult.Claim.TurnID !=
					nextTurnID {
				t.Fatalf(
					"第%d轮后续领取结果错误: %+v",
					round,
					claimResult,
				)
			}

			// 结算先完成时，后续领取可以成功。
			// 用失败结算释放它，避免影响下一轮。
			if _,
				err :=
				repository.CompleteAssistantRuntimeTurnFailure(
					context.Background(),
					assistantRuntimeFailureSettlementInput(
						fixture,
						deployment,
						session,
						nextTurnID,
					),
				); err != nil {
				t.Fatalf(
					"第%d轮清理后续领取失败: %v",
					round,
					err,
				)
			}

		case errors.Is(
			claimResult.Err,
			repository.ErrAssistantRuntimeTurnInProgress,
		):
			// 新领取先检查会话时，旧主轮次仍在执行，
			// 返回稳定冲突属于合法并发结果。

		default:
			t.Fatalf(
				"第%d轮并发领取返回意外错误: %v",
				round,
				claimResult.Err,
			)
		}

		var activeTurnCount int

		if err := database.DB.QueryRow(
			context.Background(),
			`
			SELECT COUNT(*)
			FROM assistant_runtime_sessions
			WHERE id = $1
			  AND (
					active_turn_id IS NOT NULL
					OR active_turn_started_at IS NOT NULL
			  )
			`,
			session.ID,
		).Scan(
			&activeTurnCount,
		); err != nil {
			t.Fatalf(
				"第%d轮检查悬空主轮次失败: %v",
				round,
				err,
			)
		}

		if activeTurnCount != 0 {
			t.Fatalf(
				"第%d轮留下悬空主轮次: %d",
				round,
				activeTurnCount,
			)
		}
	}

	var (
		succeededCount int
		duplicateCount int
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
			),
			(
				SELECT COUNT(*)::integer
				FROM (
					SELECT turn_id
					FROM assistant_runtime_usage
					WHERE deployment_id = $1
					GROUP BY turn_id
					HAVING COUNT(*) > 1
				) AS duplicated
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
		&succeededCount,
		&duplicateCount,
		&activeCount,
	); err != nil {
		t.Fatalf(
			"查询锁顺序最终结果失败: %v",
			err,
		)
	}

	if succeededCount != rounds ||
		duplicateCount != 0 ||
		activeCount != 0 {
		t.Fatalf(
			"锁顺序最终结果错误: succeeded=%d duplicates=%d active=%d",
			succeededCount,
			duplicateCount,
			activeCount,
		)
	}
}
