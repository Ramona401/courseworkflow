-- 20260729_01_token_price_sync.sql
--
-- 文本模型及图片、视频、TTS媒体价格自动同步基础结构。
--
-- 本迁移只管理价格及同步元数据，不接入媒体业务预留或结算流程。
--
-- 设计原则：
-- 1. 已有价格默认关闭定时自动修改；
-- 2. 文本模型按模型名精确匹配；
-- 3. 媒体价格按现有媒体五元组精确匹配；
-- 4. 同步预览和正式应用分离；
-- 5. 保存每次同步的旧值、新值、来源数据及跳过原因；
-- 6. 迁移可重复执行。

BEGIN;

-- ==================== 文本模型同步控制字段 ====================

ALTER TABLE token_model_prices
    ADD COLUMN IF NOT EXISTS auto_sync_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sync_source VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sync_model_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_sync_message TEXT NOT NULL DEFAULT '';

UPDATE token_model_prices
SET sync_source = 'main_gateway'
WHERE sync_source = '';

UPDATE token_model_prices
SET sync_model_name = model_name
WHERE sync_model_name = '';

-- ==================== 媒体模型同步控制字段 ====================

ALTER TABLE token_media_prices
    ADD COLUMN IF NOT EXISTS auto_sync_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sync_source VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sync_model_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_sync_message TEXT NOT NULL DEFAULT '';

UPDATE token_media_prices
SET sync_source = CASE
    WHEN media_type IN ('image', 'video') THEN 'media_gateway'
    WHEN media_type = 'tts' THEN 'tts_gateway'
    ELSE 'main_gateway'
END
WHERE sync_source = '';

UPDATE token_media_prices
SET sync_model_name = model_name
WHERE sync_model_name = '';

-- ==================== 同步批次 ====================

CREATE TABLE IF NOT EXISTS token_price_sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    trigger_type VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    source_kind VARCHAR(32) NOT NULL DEFAULT 'multi',
    source_base_url TEXT NOT NULL DEFAULT '',

    preview_only BOOLEAN NOT NULL DEFAULT TRUE,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',

    created_by UUID,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,

    CONSTRAINT ck_token_price_sync_runs_trigger
        CHECK (trigger_type IN ('manual', 'scheduler')),

    CONSTRAINT ck_token_price_sync_runs_status
        CHECK (status IN ('previewed', 'applied', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_token_price_sync_runs_started
    ON token_price_sync_runs (started_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_price_sync_runs_status
    ON token_price_sync_runs (status, started_at DESC);

-- ==================== 同步明细 ====================

CREATE TABLE IF NOT EXISTS token_price_sync_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    run_id UUID NOT NULL
        REFERENCES token_price_sync_runs(id)
        ON DELETE CASCADE,

    target_kind VARCHAR(16) NOT NULL,
    target_id UUID,

    provider VARCHAR(64) NOT NULL DEFAULT '',
    model_name VARCHAR(255) NOT NULL,
    sync_source VARCHAR(32) NOT NULL DEFAULT '',

    media_type VARCHAR(32) NOT NULL DEFAULT '',
    variant VARCHAR(128) NOT NULL DEFAULT '',
    media_unit VARCHAR(32) NOT NULL DEFAULT '',

    old_input_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,
    new_input_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,
    old_output_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,
    new_output_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,

    old_unit_cost_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,
    new_unit_cost_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,

    action VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    source_payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_token_price_sync_items_target
        CHECK (target_kind IN ('text', 'media')),

    CONSTRAINT ck_token_price_sync_items_action
        CHECK (action IN (
            'update',
            'unchanged',
            'skipped',
            'applied',
            'stale'
        ))
);

CREATE INDEX IF NOT EXISTS idx_token_price_sync_items_run
    ON token_price_sync_items (run_id, created_at, id);

CREATE INDEX IF NOT EXISTS idx_token_price_sync_items_target
    ON token_price_sync_items (target_kind, target_id);

-- ==================== 调度及同步安全配置 ====================

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
        'price_sync_enabled',
        'false',
        '是否启用每日模型价格自动同步',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_auto_apply',
        'false',
        '每日同步后是否自动应用可信价格变更',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_group',
        'default',
        '聚合网关价格同步使用的计费分组',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_interval_hours',
        '24',
        '价格自动同步间隔小时数',
        NOW()
    ),
    (
        gen_random_uuid(),
        'price_sync_max_change_percent',
        '50',
        '单次价格自动变化安全上限百分比',
        NOW()
    )
ON CONFLICT (config_key) DO NOTHING;

COMMIT;
