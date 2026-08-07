package repository

// media_billing_settlement_repo.go — 媒体积分成功结算与失败释放
//
// 成功事务同时释放预留、扣减实际积分、写token_consumption_logs并更新计费终态。
// 失败或取消事务只释放预留，不写消费流水。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// SettleTokenMediaBilling 原子释放冻结、扣减实际积分并写统一消费流水。
func SettleTokenMediaBilling(
	ctx context.Context,
	input *models.TokenMediaSettleSnapshot,
) (*models.TokenMediaBilling, error) {
	if input == nil {
		return nil, ErrTokenMediaBillingNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启媒体积分结算事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockTokenMediaBillingKey(ctx, tx, input.IdempotencyKey); err != nil {
		return nil, err
	}
	billing, err := getTokenMediaBillingByKeyTx(ctx, tx, input.IdempotencyKey, true)
	if err != nil {
		return nil, err
	}
	if billing.Status == models.MediaBillingStatusSettled {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交媒体计费幂等结算读取失败: %w", err)
		}
		return billing, nil
	}
	if billing.Status != models.MediaBillingStatusReserved {
		return nil, ErrTokenMediaBillingTerminal
	}

	var balanceBefore, frozenAmount float64
	err = tx.QueryRow(ctx, `
		SELECT balance, frozen_amount
		FROM token_accounts
		WHERE id = $1
		FOR UPDATE
	`, billing.AccountID).Scan(&balanceBefore, &frozenAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("锁定媒体结算账户失败: %w", err)
	}
	if frozenAmount+0.00005 < billing.ReservedCredits {
		return nil, ErrTokenMediaBillingFrozenMismatch
	}

	balanceAfter := balanceBefore - input.ActualCredits
	_, err = tx.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = GREATEST(frozen_amount - $1, 0),
		    balance = balance - $2,
		    total_consumed = total_consumed + $2,
		    updated_at = NOW()
		WHERE id = $3
	`, billing.ReservedCredits, input.ActualCredits, billing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("扣减媒体实际积分失败: %w", err)
	}

	assetID := billing.AssetID
	if input.AssetID != nil {
		assetID = input.AssetID
	}
	externalTaskID := strings.TrimSpace(input.ExternalTaskID)
	if externalTaskID == "" {
		externalTaskID = billing.ExternalTaskID
	}
	metadataJSON, err := mergeTokenMediaBillingMetadata(billing.Metadata, input.Metadata)
	if err != nil {
		return nil, err
	}

	var consumptionLogID string
	err = tx.QueryRow(ctx, `
		INSERT INTO token_consumption_logs (
			account_id, user_id, amount, balance_before, balance_after,
			scene_code, model_used, tokens_used, lesson_plan_id, pipeline_id, memo,
			input_tokens, output_tokens, model_name, provider,
			cost_usd, exchange_rate, multiplier, credits_consumed, latency_ms,
			school_id, billing_category, billing_node_code,
			courseware_id, page_id, asset_id,
			media_type, media_unit, media_quantity,
			external_task_id, idempotency_key, metadata
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, 0, NULL, NULL, $8,
			0, 0, $7, $9, $10, $11, $12, $3, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25::jsonb
		)
		RETURNING id
	`,
		billing.AccountID, billing.UserID, input.ActualCredits,
		balanceBefore, balanceAfter, billing.SceneCode, billing.ModelName,
		billing.BillingNodeCode, billing.Provider, input.ActualCostUSD,
		billing.ExchangeRate, billing.Multiplier, input.LatencyMs,
		billing.SchoolID, billing.BillingCategory, billing.BillingNodeCode,
		billing.CoursewareID, billing.PageID, assetID,
		billing.MediaType, billing.MediaUnit, input.ActualQuantity,
		externalTaskID, billing.IdempotencyKey, metadataJSON,
	).Scan(&consumptionLogID)
	if err != nil {
		return nil, fmt.Errorf("写入媒体积分消费流水失败: %w", err)
	}

	updated, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		UPDATE token_media_billings
		SET status = 'settled',
		    actual_quantity = $2,
		    actual_cost_usd = $3,
		    actual_credits = $4,
		    asset_id = $5,
		    external_task_id = $6,
		    consumption_log_id = $7,
		    metadata = $8::jsonb,
		    failure_reason = '',
		    updated_at = NOW(),
		    settled_at = NOW()
		WHERE id = $1
		RETURNING `+tokenMediaBillingSelectColumns,
		billing.ID, input.ActualQuantity, input.ActualCostUSD,
		input.ActualCredits, assetID, externalTaskID,
		consumptionLogID, metadataJSON,
	))
	if err != nil {
		return nil, fmt.Errorf("更新媒体计费成功终态失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交媒体积分结算事务失败: %w", err)
	}
	return updated, nil
}

// ReleaseTokenMediaBilling 原子释放失败或取消任务的冻结积分。
func ReleaseTokenMediaBilling(
	ctx context.Context,
	input *models.TokenMediaReleaseSnapshot,
) (*models.TokenMediaBilling, error) {
	if input == nil {
		return nil, ErrTokenMediaBillingNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启媒体积分释放事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockTokenMediaBillingKey(ctx, tx, input.IdempotencyKey); err != nil {
		return nil, err
	}
	billing, err := getTokenMediaBillingByKeyTx(ctx, tx, input.IdempotencyKey, true)
	if err != nil {
		return nil, err
	}
	if billing.Status == models.MediaBillingStatusFailed ||
		billing.Status == models.MediaBillingStatusCancelled {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交媒体计费幂等释放读取失败: %w", err)
		}
		return billing, nil
	}
	if billing.Status != models.MediaBillingStatusReserved {
		return nil, ErrTokenMediaBillingTerminal
	}

	_, err = tx.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = GREATEST(frozen_amount - $1, 0),
		    updated_at = NOW()
		WHERE id = $2
	`, billing.ReservedCredits, billing.AccountID)
	if err != nil {
		return nil, fmt.Errorf("释放媒体冻结积分失败: %w", err)
	}

	externalTaskID := strings.TrimSpace(input.ExternalTaskID)
	if externalTaskID == "" {
		externalTaskID = billing.ExternalTaskID
	}
	metadataJSON, err := mergeTokenMediaBillingMetadata(billing.Metadata, input.Metadata)
	if err != nil {
		return nil, err
	}

	updated, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		UPDATE token_media_billings
		SET status = $2,
		    external_task_id = $3,
		    metadata = $4::jsonb,
		    failure_reason = $5,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING `+tokenMediaBillingSelectColumns,
		billing.ID, input.Status, externalTaskID,
		metadataJSON, input.FailureReason,
	))
	if err != nil {
		return nil, fmt.Errorf("更新媒体计费释放终态失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交媒体积分释放事务失败: %w", err)
	}
	return updated, nil
}
