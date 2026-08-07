-- 20260729_02_token_price_sync_sources.sql
--
-- 价格同步来源校正。
--
-- 本迁移只调整“价格从哪里拉取”的配置，不接入或修改任何图片、视频、
-- TTS业务预留与结算流程。
--
-- 安全原则：
-- 1. 主聚合网关默认可从其OpenAI兼容地址推导New API /api/pricing；
-- 2. 境内、图片/视频、TTS均为独立直连通道，不得把推理地址误当价格接口；
-- 3. 独立通道只有配置了明确的New API兼容价格URL后才允许拉取；
-- 4. qwen3.7-max实际走境内通道，因此价格来源改为domestic_gateway；
-- 5. 所有自动同步开关保持关闭，不因迁移自动改价。

BEGIN;

INSERT INTO ai_configs (
    id,
    config_key,
    config_value,
    description,
    updated_at
)
VALUES
    (
        gen_random_uuid(),
        'price_sync_main_pricing_url',
        '',
        '主聚合网关价格接口；留空时由api_base_url推导/api/pricing',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_domestic_pricing_url',
        '',
        '境内文本网关的New API兼容价格接口；未配置时只跳过，不猜价',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_media_pricing_url',
        '',
        '图片和视频价格的New API兼容价格接口；未配置时只跳过，不猜价',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_tts_pricing_url',
        '',
        'TTS价格的New API兼容价格接口；未配置时只跳过，不猜价',
        NOW()
    )
ON CONFLICT (config_key) DO NOTHING;

UPDATE token_model_prices
SET sync_source = 'domestic_gateway',
    sync_model_name = model_name,
    last_sync_status = '',
    last_sync_message = ''
WHERE model_name = 'qwen3.7-max'
  AND sync_source <> 'domestic_gateway';

UPDATE token_media_prices
SET sync_source = CASE
        WHEN media_type IN ('image', 'video') THEN 'media_gateway'
        WHEN media_type = 'tts' THEN 'tts_gateway'
        ELSE sync_source
    END,
    sync_model_name = CASE
        WHEN sync_model_name = '' THEN model_name
        ELSE sync_model_name
    END
WHERE media_type IN ('image', 'video', 'tts');

COMMIT;
