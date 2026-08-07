package integration

// assistant_runtime_turn_claim_test.go
//
// 使用真实tedna_test验证：
//   - 同一会话并发领取只能产生一个成功者；
//   - 其它并发请求稳定返回TurnInProgress；
//   - 单会话轮数用尽时拒绝领取；
//   - 个人积分账户暂停、可用余额为零或过期时拒绝领取；
//   - 所有拒绝路径都不能留下active_turn占位。
//
// 本测试不调用真实AI。

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// assistantRuntimeConcurrentClaimResult 保存并发领取结果。
type assistantRuntimeConcurrentClaimResult struct {
	TurnID string
	Claim  *models.AssistantRuntimeTurnClaim
	Err    error
}

// TestAssistantRuntimeTurnClaimConcurrentSingleWinner 验证唯一领取者。
func TestAssistantRuntimeTurnClaimConcurrentSingleWinner(
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

	session := createAssistantRuntimeIsolatedSession(
		t,
		fixture,
		deployment,
		0x41000000,
		1,
	)

	const workerCount = 16

	start := make(
		chan struct{},
	)
	results := make(
		chan assistantRuntimeConcurrentClaimResult,
		workerCount,
	)

	var waitGroup sync.WaitGroup

	waitGroup.Add(
		workerCount,
	)

	for index := 0; index <
		workerCount; index++ {
		go func(workerIndex int) {
			defer waitGroup.Done()

			<-start

			turnID := assistantRuntimeIntegrationUUID(
				0x41100000,
				uint64(workerIndex+1),
			)

			ctx,
				cancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer cancel()

			claim,
				err :=
				repository.ClaimAssistantRuntimeTurn(
					ctx,
					session.ID,
					deployment.ID,
					deployment.CurrentVersion,
					turnID,
				)

			results <- assistantRuntimeConcurrentClaimResult{
				TurnID: turnID,
				Claim:  claim,
				Err:    err,
			}
		}(index)
	}

	close(
		start,
	)

	waitGroup.Wait()

	close(
		results,
	)

	successCount := 0
	inProgressCount := 0
	winnerTurnID := ""

	for result := range results {
		switch {
		case result.Err == nil:
			successCount++
			winnerTurnID =
				result.TurnID

			if result.Claim == nil ||
				result.Claim.TurnID !=
					result.TurnID {
				t.Fatalf(
					"成功领取结果错误: %+v",
					result,
				)
			}

		case errors.Is(
			result.Err,
			repository.ErrAssistantRuntimeTurnInProgress,
		):
			inProgressCount++

		default:
			t.Fatalf(
				"并发领取返回意外错误: turn=%s error=%v",
				result.TurnID,
				result.Err,
			)
		}
	}

	if successCount != 1 ||
		inProgressCount !=
			workerCount-1 {
		t.Fatalf(
			"并发领取数量错误: success=%d in_progress=%d",
			successCount,
			inProgressCount,
		)
	}

	var storedActiveTurnID string

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT active_turn_id::text
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		session.ID,
	).Scan(
		&storedActiveTurnID,
	); err != nil {
		t.Fatalf(
			"读取并发领取结果失败: %v",
			err,
		)
	}

	if storedActiveTurnID !=
		winnerTurnID {
		t.Fatalf(
			"数据库active_turn_id与唯一成功者不一致: stored=%s winner=%s",
			storedActiveTurnID,
			winnerTurnID,
		)
	}

	if _,
		err := repository.CompleteAssistantRuntimeTurnFailure(
		context.Background(),
		assistantRuntimeFailureSettlementInput(
			fixture,
			deployment,
			session,
			winnerTurnID,
		),
	); err != nil {
		t.Fatalf(
			"清理并发领取主轮次失败: %v",
			err,
		)
	}
}

// TestAssistantRuntimeTurnClaimLimitsAndBillingAccount 验证领取前置条件。
func TestAssistantRuntimeTurnClaimLimitsAndBillingAccount(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
	)

	cases := []struct {
		name    string
		prepare func(
			t *testing.T,
			fixture *AssistantRuntimeFixture,
			session *models.AssistantRuntimeSession,
		)
		expected error
	}{
		{
			name: "session turn limit reached",
			prepare: func(
				t *testing.T,
				_ *AssistantRuntimeFixture,
				session *models.AssistantRuntimeSession,
			) {
				t.Helper()

				_,
					err :=
					database.DB.Exec(
						context.Background(),
						`
						UPDATE assistant_runtime_sessions
						SET
							turn_count = max_turns,
							status = 'active',
							active_turn_id = NULL,
							active_turn_started_at = NULL
						WHERE id = $1
						`,
						session.ID,
					)
				if err != nil {
					t.Fatalf(
						"设置会话轮数上限失败: %v",
						err,
					)
				}
			},
			expected:
				repository.ErrAssistantRuntimeTurnLimitReached,
		},
		{
			name: "billing account suspended",
			prepare: func(
				t *testing.T,
				fixture *AssistantRuntimeFixture,
				_ *models.AssistantRuntimeSession,
			) {
				t.Helper()

				_,
					err :=
					database.DB.Exec(
						context.Background(),
						`
						UPDATE token_accounts
						SET status = 'suspended'
						WHERE id = $1
						`,
						fixture.TokenAccountID,
					)
				if err != nil {
					t.Fatalf(
						"暂停测试积分账户失败: %v",
						err,
					)
				}
			},
			expected:
				repository.ErrAssistantRuntimeBillingAccountUnavailable,
		},
		{
			name: "billing available balance is zero",
			prepare: func(
				t *testing.T,
				fixture *AssistantRuntimeFixture,
				_ *models.AssistantRuntimeSession,
			) {
				t.Helper()

				_,
					err :=
					database.DB.Exec(
						context.Background(),
						`
						UPDATE token_accounts
						SET
							balance = 10,
							frozen_amount = 10,
							status = 'active'
						WHERE id = $1
						`,
						fixture.TokenAccountID,
					)
				if err != nil {
					t.Fatalf(
						"设置零可用余额失败: %v",
						err,
					)
				}
			},
			expected:
				repository.ErrAssistantRuntimeBillingAccountUnavailable,
		},
		{
			name: "billing account expired",
			prepare: func(
				t *testing.T,
				fixture *AssistantRuntimeFixture,
				_ *models.AssistantRuntimeSession,
			) {
				t.Helper()

				_,
					err :=
					database.DB.Exec(
						context.Background(),
						`
						UPDATE token_accounts
						SET
							expires_at =
								NOW() - INTERVAL '1 minute',
							status = 'active'
						WHERE id = $1
						`,
						fixture.TokenAccountID,
					)
				if err != nil {
					t.Fatalf(
						"设置过期积分账户失败: %v",
						err,
					)
				}
			},
			expected:
				repository.ErrAssistantRuntimeBillingAccountUnavailable,
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
						0x41200000,
						uint64(caseIndex+1),
					)

				testCase.prepare(
					t,
					fixture,
					session,
				)

				turnID :=
					assistantRuntimeIntegrationUUID(
						0x41300000,
						uint64(caseIndex+1),
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

				if !errors.Is(
					err,
					testCase.expected,
				) {
					t.Fatalf(
						"领取前置检查错误: expected=%v actual=%v",
						testCase.expected,
						err,
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
						"检查拒绝路径主轮次状态失败: %v",
						err,
					)
				}

				if activeTurnCount != 0 {
					t.Fatalf(
						"领取被拒绝后留下主轮次占位: %d",
						activeTurnCount,
					)
				}
			},
		)
	}
}
