package repository

// media_billing_reservation_repo.go — 媒体计费读取、预留与任务绑定
//
// 同一idempotency_key先获取PostgreSQL事务级advisory lock，保证并发重试
// 不会重复冻结积分。成功预留同时写计费快照并增加token_accounts.frozen_amount。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrTokenMediaBillingNotFound       = errors.New("媒体计费记录不存在")
	ErrTokenMediaPriceUnavailable      = errors.New("媒体计费单价未启用")
	ErrTokenBillingNodeUnavailable     = errors.New("媒体计费业务节点不可用")
	ErrTokenMediaBillingTerminal       = errors.New("媒体计费记录已经终态")
	ErrTokenMediaBillingTaskConflict   = errors.New("媒体计费外部任务ID冲突")
	ErrTokenMediaBillingAssetConflict  = errors.New("媒体计费资产ID冲突")
	ErrTokenMediaBillingAssetNotFound  = errors.New("媒体计费绑定资产不存在")
	ErrTokenMediaBillingFrozenMismatch = errors.New("媒体计费冻结额度不一致")
)

const tokenMediaBillingSelectColumns = `
id,
idempotency_key,
status,
account_id,
user_id,
school_id,
billing_category,
billing_node_code,
scene_code,
media_type,
provider,
model_name,
variant,
media_unit,
estimated_quantity,
actual_quantity,
unit_cost_usd,
minimum_quantity,
minimum_cost_usd,
estimated_cost_usd,
actual_cost_usd,
exchange_rate,
multiplier,
reserved_credits,
actual_credits,
courseware_id,
page_id,
asset_id,
external_task_id,
consumption_log_id,
metadata,
failure_reason,
created_at,
updated_at,
settled_at`

func scanTokenMediaBilling(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.TokenMediaBilling, error) {
	item := &models.TokenMediaBilling{}
	var metadata []byte

	err := scanner.Scan(
		&item.ID, &item.IdempotencyKey, &item.Status,
		&item.AccountID, &item.UserID, &item.SchoolID,
		&item.BillingCategory, &item.BillingNodeCode, &item.SceneCode,
		&item.MediaType, &item.Provider, &item.ModelName, &item.Variant, &item.MediaUnit,
		&item.EstimatedQuantity, &item.ActualQuantity,
		&item.UnitCostUSD, &item.MinimumQuantity, &item.MinimumCostUSD,
		&item.EstimatedCostUSD, &item.ActualCostUSD,
		&item.ExchangeRate, &item.Multiplier,
		&item.ReservedCredits, &item.ActualCredits,
		&item.CoursewareID, &item.PageID, &item.AssetID,
		&item.ExternalTaskID, &item.ConsumptionLogID, &metadata,
		&item.FailureReason, &item.CreatedAt, &item.UpdatedAt, &item.SettledAt,
	)
	if err != nil {
		return nil, err
	}
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	item.Metadata = append(json.RawMessage(nil), metadata...)
	return item, nil
}

// GetActiveTokenBillingNode 读取已启用业务计费节点。
func GetActiveTokenBillingNode(ctx context.Context, nodeCode string) (*models.TokenBillingNode, error) {
	item := &models.TokenBillingNode{}
	err := database.DB.QueryRow(ctx, `
		SELECT node_code, category, display_name, scene_code, media_type,
		       description, sort_order, is_active, created_at, updated_at
		FROM token_billing_nodes
		WHERE node_code = $1 AND is_active = true
	`, nodeCode).Scan(
		&item.NodeCode, &item.Category, &item.DisplayName, &item.SceneCode,
		&item.MediaType, &item.Description, &item.SortOrder, &item.IsActive,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenBillingNodeUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("查询媒体计费业务节点失败: %w", err)
	}
	return item, nil
}

// GetActiveTokenMediaPrice 按完整模型身份读取已启用媒体单价。
func GetActiveTokenMediaPrice(
	ctx context.Context,
	mediaType string,
	provider string,
	modelName string,
	variant string,
	mediaUnit string,
) (*models.TokenMediaPrice, error) {
	item := &models.TokenMediaPrice{}
	err := database.DB.QueryRow(ctx, `
		SELECT id, media_type, provider, model_name, variant, media_unit,
		       unit_cost_usd, minimum_quantity, minimum_cost_usd,
		       display_name, is_active, updated_by, created_at, updated_at
		FROM token_media_prices
		WHERE media_type = $1
		  AND provider = $2
		  AND model_name = $3
		  AND variant = $4
		  AND media_unit = $5
		  AND is_active = true
	`, mediaType, provider, modelName, variant, mediaUnit).Scan(
		&item.ID, &item.MediaType, &item.Provider, &item.ModelName,
		&item.Variant, &item.MediaUnit, &item.UnitCostUSD,
		&item.MinimumQuantity, &item.MinimumCostUSD, &item.DisplayName,
		&item.IsActive, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenMediaPriceUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("查询媒体计费单价失败: %w", err)
	}
	return item, nil
}

// GetTokenMediaBillingByKey 按幂等键读取媒体计费记录。
func GetTokenMediaBillingByKey(ctx context.Context, idempotencyKey string) (*models.TokenMediaBilling, error) {
	item, err := scanTokenMediaBilling(database.DB.QueryRow(ctx, `
		SELECT `+tokenMediaBillingSelectColumns+`
		FROM token_media_billings
		WHERE idempotency_key = $1
	`, idempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenMediaBillingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询媒体计费记录失败: %w", err)
	}
	return item, nil
}

func getTokenMediaBillingByKeyTx(
	ctx context.Context,
	tx pgx.Tx,
	idempotencyKey string,
	forUpdate bool,
) (*models.TokenMediaBilling, error) {
	suffix := ""
	if forUpdate {
		suffix = " FOR UPDATE"
	}
	item, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		SELECT `+tokenMediaBillingSelectColumns+`
		FROM token_media_billings
		WHERE idempotency_key = $1`+suffix, idempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenMediaBillingNotFound
	}
	return item, err
}

func lockTokenMediaBillingKey(ctx context.Context, tx pgx.Tx, idempotencyKey string) error {
	_, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
	`, idempotencyKey)
	if err != nil {
		return fmt.Errorf("锁定媒体计费幂等键失败: %w", err)
	}
	return nil
}

func marshalTokenMediaBillingMetadata(metadata map[string]interface{}) (string, error) {
	if metadata == nil {
		return "{}", nil
	}
	content, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("序列化媒体计费元数据失败: %w", err)
	}
	return string(content), nil
}

func mergeTokenMediaBillingMetadata(
	current json.RawMessage,
	extra map[string]interface{},
) (string, error) {
	result := map[string]interface{}{}
	if len(current) > 0 {
		_ = json.Unmarshal(current, &result)
	}
	for key, value := range extra {
		result[key] = value
	}
	return marshalTokenMediaBillingMetadata(result)
}

// AnnotateTokenMediaBilling 合并更新reserved记录的补偿metadata。
//
// 事务级幂等锁保证它不会与结算、释放或其它标注交叉覆盖。
// 终态记录不允许再写reserved补偿事实。
func AnnotateTokenMediaBilling(
        ctx context.Context,
        input *models.MediaBillingAnnotateRequest,
) (*models.TokenMediaBilling, error) {
        if input == nil {
                return nil,
                        ErrTokenMediaBillingNotFound
        }

        tx, err :=
                database.DB.Begin(
                        ctx,
                )
        if err != nil {
                return nil,
                        fmt.Errorf(
                                "开启媒体计费标注事务失败: %w",
                                err,
                        )
        }
        defer func() {
                _ = tx.Rollback(ctx)
        }()

        if err :=
                lockTokenMediaBillingKey(
                        ctx,
                        tx,
                        input.IdempotencyKey,
                ); err != nil {
                return nil, err
        }

        billing, err :=
                getTokenMediaBillingByKeyTx(
                        ctx,
                        tx,
                        input.IdempotencyKey,
                        true,
                )
        if err != nil {
                return nil, err
        }

        if billing.Status !=
                models.MediaBillingStatusReserved {
                return nil,
                        ErrTokenMediaBillingTerminal
        }

        metadataJSON, err :=
                mergeTokenMediaBillingMetadata(
                        billing.Metadata,
                        input.Metadata,
                )
        if err != nil {
                return nil, err
        }

        updated, err :=
                scanTokenMediaBilling(
                        tx.QueryRow(
                                ctx,
                                `UPDATE token_media_billings
                                 SET metadata = $2::jsonb,
                                     updated_at = NOW()
                                 WHERE id = $1
                                 RETURNING `+
                                        tokenMediaBillingSelectColumns,
                                billing.ID,
                                metadataJSON,
                        ),
                )
        if err != nil {
                return nil,
                        fmt.Errorf(
                                "更新媒体计费补偿元数据失败: %w",
                                err,
                        )
        }

        if err :=
                tx.Commit(
                        ctx,
                ); err != nil {
                return nil,
                        fmt.Errorf(
                                "提交媒体计费标注事务失败: %w",
                                err,
                        )
        }

        return updated, nil
}

// ReserveTokenMediaBilling 原子创建计费记录并冻结预计积分。
func ReserveTokenMediaBilling(
	ctx context.Context,
	input *models.TokenMediaReserveSnapshot,
) (*models.TokenMediaBilling, error) {
	if input == nil {
		return nil, ErrTokenMediaBillingNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启媒体积分预留事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockTokenMediaBillingKey(ctx, tx, input.IdempotencyKey); err != nil {
		return nil, err
	}

	existing, err := getTokenMediaBillingByKeyTx(ctx, tx, input.IdempotencyKey, false)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交媒体计费幂等读取事务失败: %w", err)
		}
		existing.ReservationCreated = false
		return existing, nil
	}
	if !errors.Is(err, ErrTokenMediaBillingNotFound) {
		return nil, fmt.Errorf("检查媒体计费幂等记录失败: %w", err)
	}

	var accountID, accountState string
	var balance, frozen float64
	err = tx.QueryRow(ctx, `
		SELECT id, balance, frozen_amount, status
		FROM token_accounts
		WHERE account_type = $1 AND owner_id = $2
		FOR UPDATE
	`, models.AccountTypePersonal, input.UserID).Scan(
		&accountID, &balance, &frozen, &accountState,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("锁定媒体计费个人账户失败: %w", err)
	}
	if accountState != models.AccountStatusActive {
		return nil, ErrAccountSuspended
	}
	if balance-frozen < input.Quote.Credits {
		return nil, ErrInsufficientBalance
	}

	metadataJSON, err := marshalTokenMediaBillingMetadata(input.Metadata)
	if err != nil {
		return nil, err
	}

	item, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		INSERT INTO token_media_billings (
			idempotency_key, status, account_id, user_id, school_id,
			billing_category, billing_node_code, scene_code,
			media_type, provider, model_name, variant, media_unit,
			estimated_quantity, actual_quantity,
			unit_cost_usd, minimum_quantity, minimum_cost_usd,
			estimated_cost_usd, actual_cost_usd,
			exchange_rate, multiplier, reserved_credits, actual_credits,
			courseware_id, page_id, asset_id, external_task_id, metadata
		)
		VALUES (
			$1, 'reserved', $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, 0,
			$14, $15, $16, $17, 0, $18, $19, $20, 0,
			$21, $22, $23, $24, $25::jsonb
		)
		RETURNING `+tokenMediaBillingSelectColumns,
		input.IdempotencyKey, accountID, input.UserID, input.SchoolID,
		input.BillingCategory, input.BillingNodeCode, input.SceneCode,
		input.MediaType, input.Provider, input.ModelName, input.Variant, input.MediaUnit,
		input.EstimatedQuantity, input.Quote.UnitCostUSD,
		input.Quote.MinimumQuantity, input.Quote.MinimumCostUSD,
		input.Quote.CostUSD, input.Quote.ExchangeRate,
		input.Quote.Multiplier, input.Quote.Credits,
		input.CoursewareID, input.PageID, input.AssetID,
		input.ExternalTaskID, metadataJSON,
	))
	if err != nil {
		return nil, fmt.Errorf("创建媒体积分预留记录失败: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE token_accounts
		SET frozen_amount = frozen_amount + $1, updated_at = NOW()
		WHERE id = $2
	`, input.Quote.Credits, accountID)
	if err != nil {
		return nil, fmt.Errorf("冻结媒体预计积分失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交媒体积分预留事务失败: %w", err)
	}
	item.ReservationCreated = true
	return item, nil
}

// BindTokenMediaBillingAsset 为预留记录绑定已经持久化的业务资产。
func BindTokenMediaBillingAsset(
	ctx context.Context,
	input *models.MediaBillingBindAssetRequest,
) (*models.TokenMediaBilling, error) {
	if input == nil {
		return nil, ErrTokenMediaBillingNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启媒体资产绑定事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockTokenMediaBillingKey(ctx, tx, input.IdempotencyKey); err != nil {
		return nil, err
	}

	billing, err := getTokenMediaBillingByKeyTx(ctx, tx, input.IdempotencyKey, true)
	if err != nil {
		return nil, err
	}

	assetID := strings.TrimSpace(input.AssetID)
	if billing.AssetID != nil && strings.TrimSpace(*billing.AssetID) == assetID {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交媒体资产幂等绑定读取失败: %w", err)
		}
		return billing, nil
	}
	if billing.Status != models.MediaBillingStatusReserved {
		return nil, ErrTokenMediaBillingTerminal
	}
	if billing.AssetID != nil && strings.TrimSpace(*billing.AssetID) != "" {
		return nil, ErrTokenMediaBillingAssetConflict
	}

	var assetCoursewareID string
	err = tx.QueryRow(ctx, `
		SELECT courseware_id
		FROM courseware_assets
		WHERE id = $1
	`, assetID).Scan(&assetCoursewareID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenMediaBillingAssetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("校验媒体绑定资产失败: %w", err)
	}
	if billing.CoursewareID != nil &&
		strings.TrimSpace(*billing.CoursewareID) != strings.TrimSpace(assetCoursewareID) {
		return nil, ErrTokenMediaBillingAssetConflict
	}

	metadataJSON, err := mergeTokenMediaBillingMetadata(billing.Metadata, input.Metadata)
	if err != nil {
		return nil, err
	}

	updated, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		UPDATE token_media_billings
		SET asset_id = $2,
			metadata = $3::jsonb,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+tokenMediaBillingSelectColumns,
		billing.ID, assetID, metadataJSON,
	))
	if err != nil {
		return nil, fmt.Errorf("绑定媒体业务资产失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交媒体资产绑定事务失败: %w", err)
	}
	return updated, nil
}

// BindTokenMediaBillingExternalTask 为预留记录绑定供应商异步任务ID。
func BindTokenMediaBillingExternalTask(
	ctx context.Context,
	input *models.MediaBillingBindTaskRequest,
) (*models.TokenMediaBilling, error) {
	if input == nil {
		return nil, ErrTokenMediaBillingNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启媒体任务绑定事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockTokenMediaBillingKey(ctx, tx, input.IdempotencyKey); err != nil {
		return nil, err
	}
	billing, err := getTokenMediaBillingByKeyTx(ctx, tx, input.IdempotencyKey, true)
	if err != nil {
		return nil, err
	}
	if billing.ExternalTaskID == input.ExternalTaskID {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("提交媒体任务幂等绑定读取失败: %w", err)
		}
		return billing, nil
	}
	if billing.Status != models.MediaBillingStatusReserved {
		return nil, ErrTokenMediaBillingTerminal
	}
	if billing.ExternalTaskID != "" {
		return nil, ErrTokenMediaBillingTaskConflict
	}

	metadataJSON, err := mergeTokenMediaBillingMetadata(billing.Metadata, input.Metadata)
	if err != nil {
		return nil, err
	}
	updated, err := scanTokenMediaBilling(tx.QueryRow(ctx, `
		UPDATE token_media_billings
		SET external_task_id = $2, metadata = $3::jsonb, updated_at = NOW()
		WHERE id = $1
		RETURNING `+tokenMediaBillingSelectColumns,
		billing.ID, input.ExternalTaskID, metadataJSON,
	))
	if err != nil {
		return nil, fmt.Errorf("绑定媒体供应商任务ID失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交媒体任务绑定事务失败: %w", err)
	}
	return updated, nil
}
