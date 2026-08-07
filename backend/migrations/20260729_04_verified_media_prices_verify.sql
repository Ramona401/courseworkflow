-- 20260729_04_verified_media_prices_verify.sql
--
-- 查看媒体基础美元价格及按7.2汇率还原后的人民币官方口径。

\pset pager off

SELECT
    media_type,
    provider,
    model_name,
    variant,
    media_unit,
    display_name,
    unit_cost_usd,

    CASE
        WHEN media_unit = 'image'
            THEN ROUND(
                unit_cost_usd * 7.2,
                4
            )

        WHEN media_unit = 'character'
            THEN ROUND(
                unit_cost_usd * 7.2 * 10000,
                4
            )

        WHEN media_unit = 'provider_token'
            THEN ROUND(
                unit_cost_usd * 7.2 * 1000000,
                4
            )

        ELSE NULL
    END AS restored_cny_price,

    CASE
        WHEN media_unit = 'image'
            THEN '元/张'

        WHEN media_unit = 'character'
            THEN '元/万字符'

        WHEN media_unit = 'provider_token'
            THEN '元/百万Token'

        ELSE media_unit
    END AS restored_cny_unit,

    is_active,
    auto_sync_enabled,
    last_sync_status,
    last_sync_message,
    updated_at

FROM token_media_prices

WHERE (
    media_type = 'image'
    AND model_name =
        'doubao-seedream-5-0-260128'
)
OR (
    media_type = 'tts'
    AND model_name IN (
        'seed-tts-2.0',
        'doubao-seed-tts-2.0'
    )
)
OR (
    media_type = 'video'
    AND model_name =
        'doubao-seedance-1-5-pro-251215'
)

ORDER BY
    media_type,
    model_name,
    variant;
