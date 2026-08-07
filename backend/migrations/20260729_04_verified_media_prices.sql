-- 20260729_04_verified_media_prices.sql
--
-- 图片、TTS和视频媒体价格人工校准。
--
-- 官方人民币按量价：
--   Seedream 5.0 Lite：0.22元/张
--   Seed TTS 2.0：3元/万字符
--   Seedance 1.5 Pro在线有声：16元/百万供应商Token
--   Seedance 1.5 Pro在线无声：8元/百万供应商Token
--
-- 美元换算使用系统当前人民币兑美元汇率7.2：
--   图片：0.22 / 7.2 = 0.03055556美元/张
--   TTS：3 / 7.2 / 10000 = 0.00004167美元/字符
--   视频有声：16 / 7.2 / 1000000 = 0.00000222美元/Token
--   视频无声：8 / 7.2 / 1000000 = 0.00000111美元/Token
--
-- 系统积分倍率1.2不写入基础成本，结算时由积分策略统一计算。
-- 自动同步保持关闭，防止未配置的媒体价格接口覆盖人工校准值。

BEGIN;

-- ============================================================
-- 1. 图片：Seedream 5.0 Lite
-- ============================================================

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
    'image',
    'volcengine',
    'doubao-seedream-5-0-260128',
    'default',
    'image',
    0.03055556,
    0,
    0,
    'Seedream 5.0 Lite 图片生成',
    true,
    false,
    'media_gateway',
    'doubao-seedream-5-0-260128',
    NOW(),
    'manual_verified',
    '官方按量价0.22元/张，按汇率7.2换算'
)
ON CONFLICT (
    media_type,
    provider,
    model_name,
    variant,
    media_unit
)
DO UPDATE SET
    unit_cost_usd = EXCLUDED.unit_cost_usd,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    display_name = EXCLUDED.display_name,
    is_active = true,
    auto_sync_enabled = false,
    sync_source = EXCLUDED.sync_source,
    sync_model_name = EXCLUDED.sync_model_name,
    last_synced_at = NOW(),
    last_sync_status = 'manual_verified',
    last_sync_message = EXCLUDED.last_sync_message,
    updated_at = NOW();

-- ============================================================
-- 2. TTS：实际调用资源ID为seed-tts-2.0
-- ============================================================

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
    'tts',
    'volcengine',
    'seed-tts-2.0',
    'default',
    'character',
    0.00004167,
    0,
    0,
    'Seed TTS 2.0 字符版',
    true,
    false,
    'tts_gateway',
    'seed-tts-2.0',
    NOW(),
    'manual_verified',
    '官方按量价3元/万字符，按汇率7.2换算'
)
ON CONFLICT (
    media_type,
    provider,
    model_name,
    variant,
    media_unit
)
DO UPDATE SET
    unit_cost_usd = EXCLUDED.unit_cost_usd,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    display_name = EXCLUDED.display_name,
    is_active = true,
    auto_sync_enabled = false,
    sync_source = EXCLUDED.sync_source,
    sync_model_name = EXCLUDED.sync_model_name,
    last_synced_at = NOW(),
    last_sync_status = 'manual_verified',
    last_sync_message = EXCLUDED.last_sync_message,
    updated_at = NOW();

-- 旧模型别名保留用于审计，但不参与后续价格匹配。
UPDATE token_media_prices
SET
    is_active = false,
    auto_sync_enabled = false,
    last_sync_status = 'alias_retired',
    last_sync_message = '实际调用资源ID已统一为seed-tts-2.0',
    updated_at = NOW()
WHERE media_type = 'tts'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seed-tts-2.0'
  AND variant = 'default'
  AND media_unit = 'character';

-- ============================================================
-- 3. 视频：按有声和无声拆分
-- ============================================================

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
VALUES
(
    'video',
    'volcengine',
    'doubao-seedance-1-5-pro-251215',
    'audio',
    'provider_token',
    0.00000222,
    0,
    0,
    'Seedance 1.5 Pro 有声视频',
    true,
    false,
    'media_gateway',
    'doubao-seedance-1-5-pro-251215',
    NOW(),
    'manual_verified',
    '在线推理有声16元/百万Token，按汇率7.2换算'
),
(
    'video',
    'volcengine',
    'doubao-seedance-1-5-pro-251215',
    'silent',
    'provider_token',
    0.00000111,
    0,
    0,
    'Seedance 1.5 Pro 无声视频',
    true,
    false,
    'media_gateway',
    'doubao-seedance-1-5-pro-251215',
    NOW(),
    'manual_verified',
    '在线推理无声8元/百万Token，按汇率7.2换算'
)
ON CONFLICT (
    media_type,
    provider,
    model_name,
    variant,
    media_unit
)
DO UPDATE SET
    unit_cost_usd = EXCLUDED.unit_cost_usd,
    minimum_quantity = 0,
    minimum_cost_usd = 0,
    display_name = EXCLUDED.display_name,
    is_active = true,
    auto_sync_enabled = false,
    sync_source = EXCLUDED.sync_source,
    sync_model_name = EXCLUDED.sync_model_name,
    last_synced_at = NOW(),
    last_sync_status = 'manual_verified',
    last_sync_message = EXCLUDED.last_sync_message,
    updated_at = NOW();

-- 旧default视频价格无法区分有声和无声，停用但保留记录。
UPDATE token_media_prices
SET
    is_active = false,
    auto_sync_enabled = false,
    last_sync_status = 'variant_split',
    last_sync_message = '已拆分为audio和silent两种价格',
    updated_at = NOW()
WHERE media_type = 'video'
  AND provider = 'volcengine'
  AND model_name = 'doubao-seedance-1-5-pro-251215'
  AND variant = 'default'
  AND media_unit = 'provider_token';

-- ============================================================
-- 4. 事务内校验
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM token_media_prices
        WHERE media_type = 'image'
          AND provider = 'volcengine'
          AND model_name = 'doubao-seedream-5-0-260128'
          AND variant = 'default'
          AND media_unit = 'image'
          AND unit_cost_usd = 0.03055556
          AND is_active = true
          AND auto_sync_enabled = false
    ) THEN
        RAISE EXCEPTION 'Seedream图片价格校验失败';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM token_media_prices
        WHERE media_type = 'tts'
          AND provider = 'volcengine'
          AND model_name = 'seed-tts-2.0'
          AND variant = 'default'
          AND media_unit = 'character'
          AND unit_cost_usd = 0.00004167
          AND is_active = true
          AND auto_sync_enabled = false
    ) THEN
        RAISE EXCEPTION 'Seed TTS价格校验失败';
    END IF;

    IF (
        SELECT COUNT(*)
        FROM token_media_prices
        WHERE media_type = 'video'
          AND provider = 'volcengine'
          AND model_name = 'doubao-seedance-1-5-pro-251215'
          AND variant IN ('audio', 'silent')
          AND media_unit = 'provider_token'
          AND is_active = true
          AND auto_sync_enabled = false
    ) <> 2 THEN
        RAISE EXCEPTION 'Seedance视频价格变体校验失败';
    END IF;
END
$$;

COMMIT;
