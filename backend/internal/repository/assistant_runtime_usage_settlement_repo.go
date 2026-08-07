package repository

// assistant_runtime_usage_settlement_repo.go
//
// 本文件执行运行主轮次的最终原子结算。
//
// 成功事务同时完成：
//   1. 检查turn_id尚未结算；
//   2. 锁定并复核部署计费身份；
//   3. 锁定会话并验证active_turn_id；
//   4. 锁定部署创建者个人积分账户；
//   5. 追加成功运行流水；
//   6. 扣减个人积分并写现有积分消费流水；
//   7. 追加学生和助手正式消息；
//   8. 增加成功轮数并释放主轮次。
//
// 失败事务只写失败运行流水并释放主轮次，不增加成功轮数、不扣积分。
//
// 全链固定锁顺序：
//
//	assistant_deployments
//	  → assistant_runtime_sessions
//	  → token_accounts（仅成功结算）
//
// 该顺序与ClaimAssistantRuntimeTurn保持一致，禁止改回“会话→部署”，
// 否则新轮次领取与旧轮次结算并发时可能形成循环等待。
//
// assistant_runtime_usage唯一turn_id和会话active_turn_id共同防止重复结算。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CompleteAssistantRuntimeTurnSuccess 原子完成成功轮次并扣费。
func CompleteAssistantRuntimeTurnSuccess(
	ctx context.Context,
	input *AssistantRuntimeSuccessSettlementInput,
) (
	*AssistantRuntimeBillingSettlement,
	error,
) {
	if err := validateAssistantRuntimeSuccessSettlementInput(
		input,
	); err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体成功结算事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	exists, err := assistantRuntimeUsageExistsTx(
		ctx,
		tx,
		input.TurnID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"检查教学智能体主轮次结算状态失败: %w",
				err,
			)
	}
	if exists {
		return nil,
			ErrAssistantRuntimeTurnAlreadyFinalized
	}

	// 结算必须先锁部署，再锁会话。
	//
	// 本校验只复核创建者、学校、课件和页面快照，不检查当前部署状态、
	// current_version或有效期。因此教师在AI调用期间暂停、撤销或发布新版，
	// 已经发生的模型成本仍可按原领取身份完成结算。
	if err := validateAssistantRuntimeSettlementDeploymentTx(
		ctx,
		tx,
		input.DeploymentID,
		input.OwnerUserID,
		input.SchoolID,
		input.CoursewareID,
		input.PageID,
	); err != nil {
		return nil, err
	}

	locked, err := lockAssistantRuntimeSettlementSession(
		ctx,
		tx,
		input.SessionID,
		input.TurnID,
		input.DeploymentID,
		input.DeploymentVersion,
	)
	if err != nil {
		return nil, err
	}

	account, err := loadAssistantRuntimeBillingAccountTx(
		ctx,
		tx,
		input.OwnerUserID,
	)
	if err != nil {
		return nil, err
	}

	creditsUsed := input.Calculation.CreditsConsumed
	balanceBefore := account.Balance
	balanceAfter := balanceBefore - creditsUsed

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO assistant_runtime_usage (
			turn_id,
			deployment_id,
			runtime_session_id,
			deployment_version,
			owner_user_id,
			school_id,
			courseware_id,
			page_id,
			session_kind,
			input_chars,
			output_chars,
			input_tokens,
			output_tokens,
			credits_used,
			model_name,
			provider,
			status,
			error_code,
			latency_ms,
			created_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16,
			'succeeded', '', $17, NOW()
		)`,
		input.TurnID,
		input.DeploymentID,
		input.SessionID,
		input.DeploymentVersion,
		input.OwnerUserID,
		input.SchoolID,
		input.CoursewareID,
		input.PageID,
		locked.Session.SessionKind,
		input.InputChars,
		input.OutputChars,
		input.InputTokens,
		input.OutputTokens,
		creditsUsed,
		input.ModelName,
		input.Provider,
		input.LatencyMs,
	)
	if err != nil {
		if assistantRuntimeUsageConstraintName(err) != "" {
			return nil,
				ErrAssistantRuntimeTurnAlreadyFinalized
		}

		return nil,
			fmt.Errorf(
				"写入教学智能体成功运行流水失败: %w",
				err,
			)
	}

	if creditsUsed > 0 {
		err = tx.QueryRow(
			ctx,
			`
			UPDATE token_accounts
			SET
				balance = balance - $1,
				total_consumed = total_consumed + $1,
				updated_at = NOW()
			WHERE id = $2
			RETURNING balance`,
			creditsUsed,
			account.ID,
		).Scan(
			&balanceAfter,
		)
		if err != nil {
			return nil,
				fmt.Errorf(
					"扣减教学智能体部署创建者积分失败: %w",
					err,
				)
		}

		totalTokens := input.InputTokens +
			input.OutputTokens

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO token_consumption_logs (
				account_id,
				user_id,
				amount,
				balance_before,
				balance_after,
				scene_code,
				model_used,
				tokens_used,
				lesson_plan_id,
				pipeline_id,
				memo,
				input_tokens,
				output_tokens,
				model_name,
				provider,
				cost_usd,
				exchange_rate,
				multiplier,
				credits_consumed,
				latency_ms
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				NULL, NULL, $9, $10, $11, $12, $13,
				$14, $15, $16, $17, $18
			)`,
			account.ID,
			input.OwnerUserID,
			creditsUsed,
			balanceBefore,
			balanceAfter,
			input.SceneCode,
			input.ModelName,
			totalTokens,
			"assistant_runtime_turn:"+input.TurnID,
			input.InputTokens,
			input.OutputTokens,
			input.ModelName,
			input.Provider,
			input.Calculation.CostUSD,
			input.Calculation.ExchangeRate,
			input.Calculation.Multiplier,
			creditsUsed,
			input.LatencyMs,
		)
		if err != nil {
			return nil,
				fmt.Errorf(
					"写入教学智能体教师积分消费流水失败: %w",
					err,
				)
		}
	}

	messagesJSON, err := appendAssistantRuntimeVisibleMessages(
		locked.Session.MessagesJSON,
		input.StudentMessage,
		input.AssistantMessage,
	)
	if err != nil {
		return nil, err
	}

	nextTurnCount := locked.Session.TurnCount + 1

	var (
		nextStatus   string
		lastActiveAt time.Time
		updatedAt    time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		UPDATE assistant_runtime_sessions
		SET
			turn_count = $2,
			status = CASE
				WHEN expires_at <= NOW() THEN 'expired'
				WHEN $2 >= max_turns THEN 'completed'
				ELSE 'active'
			END,
			messages_json = $3::jsonb,
			active_turn_id = NULL,
			active_turn_started_at = NULL,
			last_active_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND active_turn_id = $4
		RETURNING
			status,
			last_active_at,
			updated_at`,
		input.SessionID,
		nextTurnCount,
		messagesJSON,
		input.TurnID,
	).Scan(
		&nextStatus,
		&lastActiveAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantRuntimeTurnNotClaimed
		}

		return nil,
			fmt.Errorf(
				"完成教学智能体成功主轮次失败: %w",
				err,
			)
	}

	locked.Session.TurnCount = nextTurnCount
	locked.Session.Status = nextStatus
	locked.Session.MessagesJSON = messagesJSON
	locked.Session.ActiveTurnID = nil
	locked.Session.ActiveTurnStartedAt = nil
	locked.Session.LastActiveAt = &lastActiveAt
	locked.Session.UpdatedAt = &updatedAt

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体成功结算事务失败: %w",
				err,
			)
	}

	account.Balance = balanceAfter
	account.TotalConsumed += creditsUsed

	return &AssistantRuntimeBillingSettlement{
		Session:      locked.Session,
		Account:      account,
		BalanceAfter: balanceAfter,
	}, nil
}

// CompleteAssistantRuntimeTurnFailure 写失败流水并释放主轮次。
func CompleteAssistantRuntimeTurnFailure(
	ctx context.Context,
	input *AssistantRuntimeFailureSettlementInput,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	if err := validateAssistantRuntimeFailureSettlementInput(
		input,
	); err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体失败结算事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	exists, err := assistantRuntimeUsageExistsTx(
		ctx,
		tx,
		input.TurnID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"检查教学智能体失败轮次状态失败: %w",
				err,
			)
	}
	if exists {
		return nil,
			ErrAssistantRuntimeTurnAlreadyFinalized
	}

	// 与轮次领取、成功结算保持同一锁顺序：先部署，后会话。
	if err := validateAssistantRuntimeSettlementDeploymentTx(
		ctx,
		tx,
		input.DeploymentID,
		input.OwnerUserID,
		input.SchoolID,
		input.CoursewareID,
		input.PageID,
	); err != nil {
		return nil, err
	}

	locked, err := lockAssistantRuntimeSettlementSession(
		ctx,
		tx,
		input.SessionID,
		input.TurnID,
		input.DeploymentID,
		input.DeploymentVersion,
	)
	if err != nil {
		return nil, err
	}

	if locked.Session.SessionKind !=
		input.SessionKind {
		return nil,
			ErrAssistantRuntimeSettlementIdentityMismatch
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO assistant_runtime_usage (
			turn_id,
			deployment_id,
			runtime_session_id,
			deployment_version,
			owner_user_id,
			school_id,
			courseware_id,
			page_id,
			session_kind,
			input_chars,
			output_chars,
			input_tokens,
			output_tokens,
			credits_used,
			model_name,
			provider,
			status,
			error_code,
			latency_ms,
			created_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, 0, 0, 0, 0, $11, $12,
			'failed', $13, $14, NOW()
		)`,
		input.TurnID,
		input.DeploymentID,
		input.SessionID,
		input.DeploymentVersion,
		input.OwnerUserID,
		input.SchoolID,
		input.CoursewareID,
		input.PageID,
		locked.Session.SessionKind,
		input.InputChars,
		input.ModelName,
		input.Provider,
		input.ErrorCode,
		input.LatencyMs,
	)
	if err != nil {
		if assistantRuntimeUsageConstraintName(err) != "" {
			return nil,
				ErrAssistantRuntimeTurnAlreadyFinalized
		}

		return nil,
			fmt.Errorf(
				"写入教学智能体失败运行流水失败: %w",
				err,
			)
	}

	var (
		status       string
		lastActiveAt time.Time
		updatedAt    time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		UPDATE assistant_runtime_sessions
		SET
			status = CASE
				WHEN expires_at <= NOW() THEN 'expired'
				ELSE status
			END,
			active_turn_id = NULL,
			active_turn_started_at = NULL,
			last_active_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND active_turn_id = $2
		RETURNING
			status,
			last_active_at,
			updated_at`,
		input.SessionID,
		input.TurnID,
	).Scan(
		&status,
		&lastActiveAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrAssistantRuntimeTurnNotClaimed
		}

		return nil,
			fmt.Errorf(
				"释放教学智能体失败主轮次失败: %w",
				err,
			)
	}

	locked.Session.Status = status
	locked.Session.ActiveTurnID = nil
	locked.Session.ActiveTurnStartedAt = nil
	locked.Session.LastActiveAt = &lastActiveAt
	locked.Session.UpdatedAt = &updatedAt

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体失败结算事务失败: %w",
				err,
			)
	}

	return locked.Session, nil
}
