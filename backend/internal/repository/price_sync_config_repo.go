package repository

// price_sync_config_repo.go — 价格同步管理配置数据访问层。
//
// 本文件只维护同步开关、价格来源和上游模型名映射，不修改正式价格。
// 正式价格只能通过已经保存的同步预览批次应用。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrPriceSyncTargetNotFound = errors.New("价格同步目标不存在")

// ListPriceSyncTargetConfigs 返回文本及图片、视频、TTS同步目标。
func ListPriceSyncTargetConfigs(
	ctx context.Context,
) (
	[]models.PriceSyncTargetConfig,
	[]models.PriceSyncTargetConfig,
	error,
) {
	textTargets, err := listTextPriceSyncTargetConfigs(ctx)
	if err != nil {
		return nil, nil, err
	}

	mediaTargets, err := listMediaPriceSyncTargetConfigs(ctx)
	if err != nil {
		return nil, nil, err
	}

	return textTargets, mediaTargets, nil
}

func listTextPriceSyncTargetConfigs(
	ctx context.Context,
) ([]models.PriceSyncTargetConfig, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT id, provider, model_name, display_name, is_active,
		       cost_per_1k_input, cost_per_1k_output,
		       auto_sync_enabled, sync_source, sync_model_name,
		       last_synced_at, last_sync_status, last_sync_message
		FROM token_model_prices
		ORDER BY provider, model_name
	`)
	if err != nil {
		return nil, fmt.Errorf("查询文本价格同步配置失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.PriceSyncTargetConfig, 0)

	for rows.Next() {
		var item models.PriceSyncTargetConfig
		item.TargetKind = models.PriceSyncTargetText

		if err := rows.Scan(
			&item.ID,
			&item.Provider,
			&item.ModelName,
			&item.DisplayName,
			&item.IsActive,
			&item.CurrentInputUSD,
			&item.CurrentOutputUSD,
			&item.AutoSyncEnabled,
			&item.SyncSource,
			&item.SyncModelName,
			&item.LastSyncedAt,
			&item.LastSyncStatus,
			&item.LastSyncMessage,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描文本价格同步配置失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历文本价格同步配置失败: %w",
			err,
		)
	}

	return items, nil
}

func listMediaPriceSyncTargetConfigs(
	ctx context.Context,
) ([]models.PriceSyncTargetConfig, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT id, media_type, provider, model_name, display_name, is_active,
		       variant, media_unit, unit_cost_usd,
		       auto_sync_enabled, sync_source, sync_model_name,
		       last_synced_at, last_sync_status, last_sync_message
		FROM token_media_prices
		WHERE media_type IN ('image', 'video', 'tts')
		ORDER BY media_type, provider, model_name, variant, media_unit
	`)
	if err != nil {
		return nil, fmt.Errorf("查询媒体价格同步配置失败: %w", err)
	}
	defer rows.Close()

	items := make([]models.PriceSyncTargetConfig, 0)

	for rows.Next() {
		var item models.PriceSyncTargetConfig
		item.TargetKind = models.PriceSyncTargetMedia

		if err := rows.Scan(
			&item.ID,
			&item.MediaType,
			&item.Provider,
			&item.ModelName,
			&item.DisplayName,
			&item.IsActive,
			&item.Variant,
			&item.MediaUnit,
			&item.CurrentUnitCostUSD,
			&item.AutoSyncEnabled,
			&item.SyncSource,
			&item.SyncModelName,
			&item.LastSyncedAt,
			&item.LastSyncStatus,
			&item.LastSyncMessage,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描媒体价格同步配置失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历媒体价格同步配置失败: %w",
			err,
		)
	}

	return items, nil
}

// GetPriceSyncTargetConfig 查询单个同步目标。
func GetPriceSyncTargetConfig(
	ctx context.Context,
	targetKind string,
	targetID string,
) (*models.PriceSyncTargetConfig, error) {
	targetKind = strings.TrimSpace(targetKind)
	targetID = strings.TrimSpace(targetID)

	if targetKind == models.PriceSyncTargetText {
		item := &models.PriceSyncTargetConfig{
			TargetKind: models.PriceSyncTargetText,
		}

		err := database.DB.QueryRow(ctx, `
			SELECT id, provider, model_name, display_name, is_active,
			       cost_per_1k_input, cost_per_1k_output,
			       auto_sync_enabled, sync_source, sync_model_name,
			       last_synced_at, last_sync_status, last_sync_message
			FROM token_model_prices
			WHERE id = $1
		`, targetID).Scan(
			&item.ID,
			&item.Provider,
			&item.ModelName,
			&item.DisplayName,
			&item.IsActive,
			&item.CurrentInputUSD,
			&item.CurrentOutputUSD,
			&item.AutoSyncEnabled,
			&item.SyncSource,
			&item.SyncModelName,
			&item.LastSyncedAt,
			&item.LastSyncStatus,
			&item.LastSyncMessage,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPriceSyncTargetNotFound
		}
		if err != nil {
			return nil, fmt.Errorf(
				"查询文本价格同步目标失败: %w",
				err,
			)
		}

		return item, nil
	}

	if targetKind == models.PriceSyncTargetMedia {
		item := &models.PriceSyncTargetConfig{
			TargetKind: models.PriceSyncTargetMedia,
		}

		err := database.DB.QueryRow(ctx, `
			SELECT id, media_type, provider, model_name,
			       display_name, is_active, variant, media_unit,
			       unit_cost_usd, auto_sync_enabled,
			       sync_source, sync_model_name,
			       last_synced_at, last_sync_status, last_sync_message
			FROM token_media_prices
			WHERE id = $1
			  AND media_type IN ('image', 'video', 'tts')
		`, targetID).Scan(
			&item.ID,
			&item.MediaType,
			&item.Provider,
			&item.ModelName,
			&item.DisplayName,
			&item.IsActive,
			&item.Variant,
			&item.MediaUnit,
			&item.CurrentUnitCostUSD,
			&item.AutoSyncEnabled,
			&item.SyncSource,
			&item.SyncModelName,
			&item.LastSyncedAt,
			&item.LastSyncStatus,
			&item.LastSyncMessage,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPriceSyncTargetNotFound
		}
		if err != nil {
			return nil, fmt.Errorf(
				"查询媒体价格同步目标失败: %w",
				err,
			)
		}

		return item, nil
	}

	return nil, ErrPriceSyncTargetNotFound
}

// UpdatePriceSyncTargetConfig 更新同步来源、模型映射和定时同步开关。
func UpdatePriceSyncTargetConfig(
	ctx context.Context,
	target *models.PriceSyncTargetConfig,
	updatedBy string,
) (*models.PriceSyncTargetConfig, error) {
	if target == nil {
		return nil, ErrPriceSyncTargetNotFound
	}

	var affectedRows int64

	if target.TargetKind == models.PriceSyncTargetText {
		tag, err := database.DB.Exec(ctx, `
			UPDATE token_model_prices
			SET auto_sync_enabled = $2,
			    sync_source = $3,
			    sync_model_name = $4,
			    last_sync_status = '',
			    last_sync_message = '',
			    updated_by = COALESCE(
			        NULLIF($5, '')::uuid,
			        updated_by
			    ),
			    updated_at = NOW()
			WHERE id = $1
		`,
			target.ID,
			target.AutoSyncEnabled,
			target.SyncSource,
			target.SyncModelName,
			strings.TrimSpace(updatedBy),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"更新文本价格同步目标失败: %w",
				err,
			)
		}
		affectedRows = tag.RowsAffected()
	} else if target.TargetKind == models.PriceSyncTargetMedia {
		tag, err := database.DB.Exec(ctx, `
			UPDATE token_media_prices
			SET auto_sync_enabled = $2,
			    sync_source = $3,
			    sync_model_name = $4,
			    last_sync_status = '',
			    last_sync_message = '',
			    updated_by = COALESCE(
			        NULLIF($5, '')::uuid,
			        updated_by
			    ),
			    updated_at = NOW()
			WHERE id = $1
			  AND media_type IN ('image', 'video', 'tts')
		`,
			target.ID,
			target.AutoSyncEnabled,
			target.SyncSource,
			target.SyncModelName,
			strings.TrimSpace(updatedBy),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"更新媒体价格同步目标失败: %w",
				err,
			)
		}
		affectedRows = tag.RowsAffected()
	} else {
		return nil, ErrPriceSyncTargetNotFound
	}

	if affectedRows != 1 {
		return nil, ErrPriceSyncTargetNotFound
	}

	return GetPriceSyncTargetConfig(
		ctx,
		target.TargetKind,
		target.ID,
	)
}
