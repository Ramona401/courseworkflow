-- 20260729_03_actual_text_model_prices.sql
--
-- 修正当前实际使用的文本模型价格。
--
-- 价格口径：
--   1. Anthropic、Google 使用标准实时 API 美元价；
--   2. Claude Sonnet 5 当前写入2026-08-31前推广价，
--      运行时规则会在2026-09-01自动切换为标准价；
--   3. Gemini Pro模型表中保存不超过200K输入Token的基础档，
--      运行时根据单次input_tokens自动应用高阶档；
--   4. qwen3.7-max使用当前人民币五折价6元/18元每百万Token，
--      按系统当前7.2汇率折算为美元；
--   5. 所有修正目标默认关闭自动同步，避免不可信的37.5倍率覆盖。
--
-- 本迁移幂等，可重复执行。

BEGIN;

WITH desired_prices (
    model_name,
    provider,
    display_name,
    cost_per_1k_input,
    cost_per_1k_output,
    sync_source
) AS (
    VALUES
        (
            'anthropic/claude-opus-4.8',
            'anthropic',
            'Claude Opus 4.8',
            0.005000::numeric,
            0.025000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-4.8-opus-20260528',
            'anthropic',
            'Claude Opus 4.8 20260528',
            0.005000::numeric,
            0.025000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-opus-4.6',
            'anthropic',
            'Claude Opus 4.6',
            0.005000::numeric,
            0.025000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-4.6-opus-20260205',
            'anthropic',
            'Claude Opus 4.6 20260205',
            0.005000::numeric,
            0.025000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-sonnet-5',
            'anthropic',
            'Claude Sonnet 5',
            0.002000::numeric,
            0.010000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-sonnet-5-20260630',
            'anthropic',
            'Claude Sonnet 5 20260630',
            0.002000::numeric,
            0.010000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-sonnet-4.6',
            'anthropic',
            'Claude Sonnet 4.6',
            0.003000::numeric,
            0.015000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-sonnet-4-5',
            'anthropic',
            'Claude Sonnet 4.5',
            0.003000::numeric,
            0.015000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-haiku-4.5',
            'anthropic',
            'Claude Haiku 4.5',
            0.001000::numeric,
            0.005000::numeric,
            'main_gateway'
        ),
        (
            'anthropic/claude-4.5-haiku-20251001',
            'anthropic',
            'Claude Haiku 4.5 20251001',
            0.001000::numeric,
            0.005000::numeric,
            'main_gateway'
        ),
        (
            'google/gemini-3.5-flash',
            'google',
            'Gemini 3.5 Flash',
            0.001500::numeric,
            0.009000::numeric,
            'main_gateway'
        ),
        (
            'google/gemini-3.5-flash-20260519',
            'google',
            'Gemini 3.5 Flash 20260519',
            0.001500::numeric,
            0.009000::numeric,
            'main_gateway'
        ),
        (
            'google/gemini-3.1-pro-preview',
            'google',
            'Gemini 3.1 Pro Preview',
            0.002000::numeric,
            0.012000::numeric,
            'main_gateway'
        ),
        (
            'google/gemini-3.1-pro-preview-20260219',
            'google',
            'Gemini 3.1 Pro Preview 20260219',
            0.002000::numeric,
            0.012000::numeric,
            'main_gateway'
        ),
        (
            'gemini-2.5-flash',
            'google',
            'Gemini 2.5 Flash',
            0.000300::numeric,
            0.002500::numeric,
            'main_gateway'
        ),
        (
            'gemini-2.5-pro',
            'google',
            'Gemini 2.5 Pro',
            0.001250::numeric,
            0.010000::numeric,
            'main_gateway'
        ),
        (
            'gemini-2.0-flash',
            'google',
            'Gemini 2.0 Flash',
            0.000100::numeric,
            0.000400::numeric,
            'main_gateway'
        ),
        (
            'qwen3.7-max',
            'qwen',
            'Qwen 3.7 Max',
            0.000833::numeric,
            0.002500::numeric,
            'domestic_gateway'
        )
)

INSERT INTO token_model_prices (
    model_name,
    provider,
    cost_per_1k_input,
    cost_per_1k_output,
    display_name,
    is_active,
    auto_sync_enabled,
    sync_source,
    sync_model_name,
    last_synced_at,
    last_sync_status,
    last_sync_message
)
SELECT
    model_name,
    provider,
    cost_per_1k_input,
    cost_per_1k_output,
    display_name,
    true,
    false,
    sync_source,
    model_name,
    NOW(),
    'manual_verified',
    '官方标准价人工校准：2026-07-29'
FROM desired_prices

ON CONFLICT (model_name)
DO UPDATE SET
    provider = EXCLUDED.provider,
    cost_per_1k_input =
        EXCLUDED.cost_per_1k_input,
    cost_per_1k_output =
        EXCLUDED.cost_per_1k_output,
    display_name = EXCLUDED.display_name,
    is_active = true,
    auto_sync_enabled = false,
    sync_source = EXCLUDED.sync_source,
    sync_model_name =
        EXCLUDED.sync_model_name,
    last_synced_at = NOW(),
    last_sync_status =
        'manual_verified',
    last_sync_message =
        '官方标准价人工校准：2026-07-29',
    updated_at = NOW();

COMMIT;
