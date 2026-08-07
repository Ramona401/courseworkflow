package repository

// price_sync_repo.go — 文本及媒体价格同步数据访问层。
//
// 本文件负责查询同步目标、保存预览、查询历史，以及使用旧价格作为
// 乐观锁应用价格。手动批次可更新未开启定时同步的记录；调度器批次
// 仍强制要求auto_sync_enabled=true。

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
	ErrPriceSyncRunNotFound      = errors.New("价格同步批次不存在")
	ErrPriceSyncRunNotApplicable = errors.New("价格同步批次不可应用")
	ErrPriceSyncItemNotFound     = errors.New("价格同步明细不存在")
)

// ListTextPriceSyncTargets 查询文本价格同步目标。
func ListTextPriceSyncTargets(ctx context.Context, schedulerOnly bool) ([]models.TextPriceSyncTarget, error) {
	query := `
		SELECT id, model_name, provider, cost_per_1k_input, cost_per_1k_output,
		       display_name, is_active, auto_sync_enabled, sync_source,
		       sync_model_name, last_synced_at, last_sync_status, last_sync_message
		FROM token_model_prices`
	if schedulerOnly {
		query += ` WHERE auto_sync_enabled = true`
	}
	query += ` ORDER BY provider, model_name`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询文本价格同步目标失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.TextPriceSyncTarget, 0)
	for rows.Next() {
		var item models.TextPriceSyncTarget
		if err := rows.Scan(
			&item.ID, &item.ModelName, &item.Provider,
			&item.CostPer1kInput, &item.CostPer1kOutput,
			&item.DisplayName, &item.IsActive, &item.AutoSyncEnabled,
			&item.SyncSource, &item.SyncModelName, &item.LastSyncedAt,
			&item.LastSyncStatus, &item.LastSyncMessage,
		); err != nil {
			return nil, fmt.Errorf("扫描文本价格同步目标失败: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历文本价格同步目标失败: %w", err)
	}
	return items, nil
}

// ListMediaPriceSyncTargets 只查询本次需求中的图片、视频和TTS价格。
// ASR不进入本同步流程，避免被错误地拿到主聚合网关中匹配。
func ListMediaPriceSyncTargets(ctx context.Context, schedulerOnly bool) ([]models.MediaPriceSyncTarget, error) {
	query := `
		SELECT id, media_type, provider, model_name, variant, media_unit,
		       unit_cost_usd, minimum_quantity, minimum_cost_usd,
		       display_name, is_active, auto_sync_enabled, sync_source,
		       sync_model_name, last_synced_at, last_sync_status, last_sync_message
		FROM token_media_prices
		WHERE media_type IN ('image', 'video', 'tts')`
	if schedulerOnly {
		query += ` AND auto_sync_enabled = true`
	}
	query += ` ORDER BY media_type, provider, model_name, variant, media_unit`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询媒体价格同步目标失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.MediaPriceSyncTarget, 0)
	for rows.Next() {
		var item models.MediaPriceSyncTarget
		if err := rows.Scan(
			&item.ID, &item.MediaType, &item.Provider, &item.ModelName,
			&item.Variant, &item.MediaUnit, &item.UnitCostUSD,
			&item.MinimumQuantity, &item.MinimumCostUSD, &item.DisplayName,
			&item.IsActive, &item.AutoSyncEnabled, &item.SyncSource,
			&item.SyncModelName, &item.LastSyncedAt,
			&item.LastSyncStatus, &item.LastSyncMessage,
		); err != nil {
			return nil, fmt.Errorf("扫描媒体价格同步目标失败: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历媒体价格同步目标失败: %w", err)
	}
	return items, nil
}

const priceSyncRunColumns = `
	id, trigger_type, status, source_kind, source_base_url,
	preview_only, summary, error_message, created_by, started_at, finished_at`

const priceSyncItemColumns = `
	id, run_id, target_kind, target_id, provider, model_name, sync_source,
	media_type, variant, media_unit, old_input_usd, new_input_usd,
	old_output_usd, new_output_usd, old_unit_cost_usd, new_unit_cost_usd,
	action, reason, source_payload, created_at`

func scanPriceSyncRun(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.PriceSyncRun, error) {
	run := &models.PriceSyncRun{}
	var summary []byte
	err := scanner.Scan(
		&run.ID, &run.TriggerType, &run.Status, &run.SourceKind,
		&run.SourceBaseURL, &run.PreviewOnly, &summary, &run.ErrorMessage,
		&run.CreatedBy, &run.StartedAt, &run.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(summary) == 0 {
		summary = []byte("{}")
	}
	run.Summary = append(json.RawMessage(nil), summary...)
	return run, nil
}

func scanPriceSyncItem(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.PriceSyncItem, error) {
	item := &models.PriceSyncItem{}
	var payload []byte
	err := scanner.Scan(
		&item.ID, &item.RunID, &item.TargetKind, &item.TargetID,
		&item.Provider, &item.ModelName, &item.SyncSource,
		&item.MediaType, &item.Variant, &item.MediaUnit,
		&item.OldInputUSD, &item.NewInputUSD,
		&item.OldOutputUSD, &item.NewOutputUSD,
		&item.OldUnitCostUSD, &item.NewUnitCostUSD,
		&item.Action, &item.Reason, &payload, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	item.SourcePayload = append(json.RawMessage(nil), payload...)
	return item, nil
}

// CreatePriceSyncPreview 原子保存一次预览及全部明细。
func CreatePriceSyncPreview(
	ctx context.Context,
	run *models.PriceSyncRun,
	items []models.PriceSyncItem,
	summary models.PriceSyncSummary,
) (*models.PriceSyncRun, error) {
	if run == nil {
		return nil, ErrPriceSyncRunNotFound
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("序列化价格同步汇总失败: %w", err)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启价格同步预览事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanPriceSyncRun(tx.QueryRow(ctx, `
		INSERT INTO token_price_sync_runs (
			trigger_type, status, source_kind, source_base_url,
			preview_only, summary, error_message, created_by
		)
		VALUES ($1, 'previewed', $2, $3, true, $4::jsonb, '', NULLIF($5, '')::uuid)
		RETURNING `+priceSyncRunColumns,
		run.TriggerType, run.SourceKind, run.SourceBaseURL,
		string(summaryJSON), priceSyncActorID(run.CreatedBy),
	))
	if err != nil {
		return nil, fmt.Errorf("创建价格同步批次失败: %w", err)
	}

	for index := range items {
		item := &items[index]
		payload := item.SourcePayload
		if len(payload) == 0 {
			payload = []byte("{}")
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO token_price_sync_items (
				run_id, target_kind, target_id, provider, model_name,
				sync_source, media_type, variant, media_unit,
				old_input_usd, new_input_usd, old_output_usd, new_output_usd,
				old_unit_cost_usd, new_unit_cost_usd, action, reason, source_payload
			)
			VALUES (
				$1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16, $17, $18::jsonb
			)`,
			created.ID, item.TargetKind, item.TargetID, item.Provider,
			item.ModelName, item.SyncSource, item.MediaType, item.Variant,
			item.MediaUnit, item.OldInputUSD, item.NewInputUSD,
			item.OldOutputUSD, item.NewOutputUSD,
			item.OldUnitCostUSD, item.NewUnitCostUSD,
			item.Action, item.Reason, string(payload),
		)
		if err != nil {
			return nil, fmt.Errorf("写入第%d条价格同步明细失败: %w", index+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交价格同步预览事务失败: %w", err)
	}
	return created, nil
}

// GetPriceSyncRun 按ID查询同步批次。
func GetPriceSyncRun(ctx context.Context, runID string) (*models.PriceSyncRun, error) {
	run, err := scanPriceSyncRun(database.DB.QueryRow(ctx, `
		SELECT `+priceSyncRunColumns+`
		FROM token_price_sync_runs
		WHERE id = $1`,
		strings.TrimSpace(runID),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPriceSyncRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询价格同步批次失败: %w", err)
	}
	return run, nil
}

// ListPriceSyncItems 查询指定批次全部明细。
func ListPriceSyncItems(ctx context.Context, runID string) ([]models.PriceSyncItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT `+priceSyncItemColumns+`
		FROM token_price_sync_items
		WHERE run_id = $1
		ORDER BY created_at, id`,
		strings.TrimSpace(runID),
	)
	if err != nil {
		return nil, fmt.Errorf("查询价格同步明细失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.PriceSyncItem, 0)
	for rows.Next() {
		item, scanErr := scanPriceSyncItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描价格同步明细失败: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历价格同步明细失败: %w", err)
	}
	return items, nil
}

// ListPriceSyncRuns 查询最近同步历史。
func ListPriceSyncRuns(ctx context.Context, limit int) ([]models.PriceSyncRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := database.DB.Query(ctx, `
		SELECT `+priceSyncRunColumns+`
		FROM token_price_sync_runs
		ORDER BY started_at DESC
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询价格同步历史失败: %w", err)
	}
	defer rows.Close()

	runs := make([]models.PriceSyncRun, 0)
	for rows.Next() {
		run, scanErr := scanPriceSyncRun(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描价格同步历史失败: %w", scanErr)
		}
		runs = append(runs, *run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历价格同步历史失败: %w", err)
	}
	return runs, nil
}

// ApplyPriceSyncRun 应用一次已预览的同步批次。
//
// selectedItemIDs为空时应用全部update明细；非空时只应用选中项。
// 未选中的update明细标记为skipped，批次完成后不可再次应用。
func ApplyPriceSyncRun(
	ctx context.Context,
	runID string,
	actorID string,
	selectedItemIDs []string,
) (*models.PriceSyncRun, []models.PriceSyncItem, models.PriceSyncSummary, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, nil, models.PriceSyncSummary{},
			fmt.Errorf("开启价格同步应用事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	run, err := scanPriceSyncRun(tx.QueryRow(ctx, `
		SELECT `+priceSyncRunColumns+`
		FROM token_price_sync_runs
		WHERE id = $1
		FOR UPDATE`,
		strings.TrimSpace(runID),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, models.PriceSyncSummary{}, ErrPriceSyncRunNotFound
	}
	if err != nil {
		return nil, nil, models.PriceSyncSummary{},
			fmt.Errorf("锁定价格同步批次失败: %w", err)
	}
	if run.Status != models.PriceSyncRunPreviewed {
		return nil, nil, models.PriceSyncSummary{}, ErrPriceSyncRunNotApplicable
	}

	items, err := listPriceSyncItemsTx(ctx, tx, run.ID)
	if err != nil {
		return nil, nil, models.PriceSyncSummary{}, err
	}

	itemIDSet := make(map[string]bool, len(items))
	for _, item := range items {
		itemIDSet[item.ID] = true
	}
	selectedSet := make(map[string]bool)
	for _, itemID := range selectedItemIDs {
		itemID = strings.TrimSpace(itemID)
		if itemID == "" {
			continue
		}
		if !itemIDSet[itemID] {
			return nil, nil, models.PriceSyncSummary{}, ErrPriceSyncItemNotFound
		}
		selectedSet[itemID] = true
	}

	applyAll := len(selectedSet) == 0
	requireAutoSync := run.TriggerType == models.PriceSyncTriggerScheduler
	summary := models.PriceSyncSummary{TotalCount: len(items)}

	for index := range items {
		item := &items[index]
		switch item.Action {
		case models.PriceSyncActionSkipped:
			summary.SkippedCount++
			continue
		case models.PriceSyncActionUnchanged:
			summary.UnchangedCount++
			continue
		case models.PriceSyncActionUpdate:
		default:
			continue
		}

		if !applyAll && !selectedSet[item.ID] {
			item.Action = models.PriceSyncActionSkipped
			item.Reason = "管理员未选择应用该项"
			summary.SkippedCount++
			if err := updatePriceSyncItemState(ctx, tx, item); err != nil {
				return nil, nil, models.PriceSyncSummary{}, err
			}
			continue
		}

		affected, updateErr := applyPriceSyncItem(
			ctx, tx, item, actorID, requireAutoSync,
		)
		if updateErr != nil {
			return nil, nil, models.PriceSyncSummary{}, updateErr
		}
		if affected == 1 {
			item.Action = models.PriceSyncActionApplied
			item.Reason = "价格已应用"
			summary.AppliedCount++
		} else {
			item.Action = models.PriceSyncActionStale
			if requireAutoSync {
				item.Reason = "本地价格或自动同步开关已变化，未覆盖"
			} else {
				item.Reason = "本地价格已变化，未覆盖"
			}
			summary.StaleCount++
		}
		if err := updatePriceSyncItemState(ctx, tx, item); err != nil {
			return nil, nil, models.PriceSyncSummary{}, err
		}
	}

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, nil, models.PriceSyncSummary{}, err
	}
	appliedRun, err := scanPriceSyncRun(tx.QueryRow(ctx, `
		UPDATE token_price_sync_runs
		SET status = 'applied',
		    preview_only = false,
		    summary = $2::jsonb,
		    finished_at = NOW()
		WHERE id = $1
		RETURNING `+priceSyncRunColumns,
		run.ID, string(summaryJSON),
	))
	if err != nil {
		return nil, nil, models.PriceSyncSummary{},
			fmt.Errorf("完成价格同步批次失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, models.PriceSyncSummary{},
			fmt.Errorf("提交价格同步应用事务失败: %w", err)
	}
	return appliedRun, items, summary, nil
}

func listPriceSyncItemsTx(ctx context.Context, tx pgx.Tx, runID string) ([]models.PriceSyncItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+priceSyncItemColumns+`
		FROM token_price_sync_items
		WHERE run_id = $1
		ORDER BY created_at, id`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("读取价格同步应用明细失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.PriceSyncItem, 0)
	for rows.Next() {
		item, scanErr := scanPriceSyncItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描价格同步应用明细失败: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func applyPriceSyncItem(
	ctx context.Context,
	tx pgx.Tx,
	item *models.PriceSyncItem,
	actorID string,
	requireAutoSync bool,
) (int64, error) {
	if item.TargetKind == models.PriceSyncTargetText {
		tag, err := tx.Exec(ctx, `
			UPDATE token_model_prices
			SET cost_per_1k_input = $2,
			    cost_per_1k_output = $3,
			    updated_by = COALESCE(NULLIF($4, '')::uuid, updated_by),
			    updated_at = NOW(),
			    last_synced_at = NOW(),
			    last_sync_status = 'success',
			    last_sync_message = '价格源同步'
			WHERE id = $1
			  AND cost_per_1k_input = $5
			  AND cost_per_1k_output = $6
			  AND ($7 = false OR auto_sync_enabled = true)`,
			item.TargetID, item.NewInputUSD, item.NewOutputUSD,
			strings.TrimSpace(actorID), item.OldInputUSD,
			item.OldOutputUSD, requireAutoSync,
		)
		if err != nil {
			return 0, fmt.Errorf("更新文本模型价格失败: %w", err)
		}
		return tag.RowsAffected(), nil
	}

	tag, err := tx.Exec(ctx, `
		UPDATE token_media_prices
		SET unit_cost_usd = $2,
		    updated_by = COALESCE(NULLIF($3, '')::uuid, updated_by),
		    updated_at = NOW(),
		    last_synced_at = NOW(),
		    last_sync_status = 'success',
		    last_sync_message = '价格源同步'
		WHERE id = $1
		  AND unit_cost_usd = $4
		  AND ($5 = false OR auto_sync_enabled = true)`,
		item.TargetID, item.NewUnitCostUSD, strings.TrimSpace(actorID),
		item.OldUnitCostUSD, requireAutoSync,
	)
	if err != nil {
		return 0, fmt.Errorf("更新媒体模型价格失败: %w", err)
	}
	return tag.RowsAffected(), nil
}

func updatePriceSyncItemState(ctx context.Context, tx pgx.Tx, item *models.PriceSyncItem) error {
	_, err := tx.Exec(ctx, `
		UPDATE token_price_sync_items
		SET action = $2, reason = $3
		WHERE id = $1`,
		item.ID, item.Action, item.Reason,
	)
	if err != nil {
		return fmt.Errorf("更新价格同步明细状态失败: %w", err)
	}
	return nil
}

func priceSyncActorID(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
