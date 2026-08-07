-- 20260729_05_verified_asr_price_rollback.sql
--
-- 回滚本次ASR价格校准。
-- 正式灾难回滚优先使用执行前的完整数据库备份。

BEGIN;

UPDATE token_media_prices
SET
    unit_cost_usd = 0,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    display_name = '默认流式ASR资源',
    is_active = false,
    auto_sync_enabled = false,
    sync_source = 'main_gateway',
    sync_model_name =
        'volc.seedasr.sauc.duration',
    last_synced_at = NULL,
    last_sync_status = '',
    last_sync_message = '',
    updated_at = NOW()
WHERE media_type = 'asr'
  AND provider = 'volcengine'
  AND model_name =
      'volc.seedasr.sauc.duration'
  AND variant = 'streaming_2_0'
  AND media_unit = 'audio_second';

COMMIT;
