-- 20260729_05_verified_asr_price_verify.sql
--
-- 验证ASR美元秒价、还原人民币小时价，
-- 并根据当前系统积分策略计算每小时和每分钟积分。

\pset pager off

WITH current_policy AS (
    SELECT
        exchange_rate,
        multiplier
    FROM token_credit_policies
    WHERE scope = 'system'
      AND scope_id IS NULL
    ORDER BY updated_at DESC
    LIMIT 1
)
SELECT
    price.media_type,
    price.provider,
    price.model_name,
    price.variant,
    price.media_unit,
    price.display_name,
    price.unit_cost_usd,

    ROUND(
        price.unit_cost_usd *
        3600,
        8
    ) AS usd_per_audio_hour,

    ROUND(
        price.unit_cost_usd *
        3600 *
        policy.exchange_rate,
        6
    ) AS restored_cny_per_hour,

    policy.exchange_rate,
    policy.multiplier,

    ROUND(
        price.unit_cost_usd *
        3600 *
        policy.exchange_rate *
        policy.multiplier,
        6
    ) AS credits_per_audio_hour,

    ROUND(
        price.unit_cost_usd *
        60 *
        policy.exchange_rate *
        policy.multiplier,
        6
    ) AS credits_per_audio_minute,

    price.is_active,
    price.auto_sync_enabled,
    price.last_sync_status,
    price.last_sync_message,
    price.updated_at

FROM token_media_prices AS price
CROSS JOIN current_policy AS policy

WHERE price.media_type = 'asr'
  AND price.provider = 'volcengine'
  AND price.model_name =
      'volc.seedasr.sauc.duration'
  AND price.variant = 'streaming_2_0'
  AND price.media_unit = 'audio_second';
