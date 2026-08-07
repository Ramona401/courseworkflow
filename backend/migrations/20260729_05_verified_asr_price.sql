-- 20260729_05_verified_asr_price.sql
--
-- 豆包流式语音识别模型2.0小时版价格校准。
--
-- 官方资源ID：
--   volc.seedasr.sauc.duration
--
-- 官方按调用量后付费价格：
--   1元/音频小时
--
-- 按系统汇率7.2换算：
--   1 / 7.2 / 3600
--   = 0.0000385802469美元/音频秒
--
-- token_media_prices.unit_cost_usd字段保留8位小数，
-- 因此写入0.00003858美元/音频秒。
--
-- 系统倍率1.2不计入基础价格，由积分策略在结算时统一计算。
-- ASR当前不参加自动价格同步，auto_sync_enabled保持关闭。

BEGIN;

INSERT INTO token_media_prices (
    media_type,
    provider,
    model_name,
    variant,
    media_unit,
    unit_cost_usd,
    minimum_quantity,
    minimum_cost_usd,
    display_name,
    is_active,
    auto_sync_enabled,
    sync_source,
    sync_model_name,
    last_synced_at,
    last_sync_status,
    last_sync_message
)
VALUES (
    'asr',
    'volcengine',
    'volc.seedasr.sauc.duration',
    'streaming_2_0',
    'audio_second',
    0.00003858,
    0,
    0,
    '豆包流式语音识别模型2.0（小时版）',
    true,
    false,
    'media_gateway',
    'volc.seedasr.sauc.duration',
    NOW(),
    'manual_verified',
    '官方后付费1元/小时，按汇率7.2换算为0.00003858美元/音频秒'
)
ON CONFLICT (
    media_type,
    provider,
    model_name,
    variant,
    media_unit
)
DO UPDATE SET
    unit_cost_usd =
        EXCLUDED.unit_cost_usd,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    display_name =
        EXCLUDED.display_name,
    is_active = true,
    auto_sync_enabled = false,
    sync_source =
        EXCLUDED.sync_source,
    sync_model_name =
        EXCLUDED.sync_model_name,
    last_synced_at = NOW(),
    last_sync_status =
        'manual_verified',
    last_sync_message =
        EXCLUDED.last_sync_message,
    updated_at = NOW();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM token_media_prices
        WHERE media_type = 'asr'
          AND provider = 'volcengine'
          AND model_name =
              'volc.seedasr.sauc.duration'
          AND variant = 'streaming_2_0'
          AND media_unit = 'audio_second'
          AND unit_cost_usd =
              0.00003858
          AND is_active = true
          AND auto_sync_enabled = false
    ) THEN
        RAISE EXCEPTION
            '豆包流式ASR 2.0价格校验失败';
    END IF;
END
$$;

COMMIT;
