-- 20260729_04_verified_media_prices_rollback.sql
--
-- 仅用于人工回滚本次媒体价格校准。
-- 正式回滚优先使用执行前的完整数据库备份。

BEGIN;

UPDATE token_media_prices
SET
    unit_cost_usd = 0,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    is_active = false,
    auto_sync_enabled = false,
    last_synced_at = NULL,
    last_sync_status = '',
    last_sync_message = '',
    updated_at = NOW()
WHERE media_type = 'image'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seedream-5-0-260128'
  AND variant = 'default'
  AND media_unit = 'image';

DELETE FROM token_media_prices
WHERE media_type = 'tts'
  AND provider = 'volcengine'
  AND model_name = 'seed-tts-2.0'
  AND variant = 'default'
  AND media_unit = 'character';

UPDATE token_media_prices
SET
    unit_cost_usd = 0,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    is_active = false,
    auto_sync_enabled = false,
    last_synced_at = NULL,
    last_sync_status = '',
    last_sync_message = '',
    updated_at = NOW()
WHERE media_type = 'tts'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seed-tts-2.0'
  AND variant = 'default'
  AND media_unit = 'character';

DELETE FROM token_media_prices
WHERE media_type = 'video'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seedance-1-5-pro-251215'
  AND variant IN ('audio', 'silent')
  AND media_unit = 'provider_token';

UPDATE token_media_prices
SET
    unit_cost_usd = 0,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    is_active = false,
    auto_sync_enabled = false,
    last_synced_at = NULL,
    last_sync_status = '',
    last_sync_message = '',
    updated_at = NOW()
WHERE media_type = 'video'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seedance-1-5-pro-251215'
  AND variant = 'default'
  AND media_unit = 'provider_token';

COMMIT;
