BEGIN;

-- ============================================================================
-- 课件全自动装配业务版本与浏览器布局验收底座
-- ============================================================================
--
-- 目标：
--   1. 每次装配获得单调递增的 assembly_version；
--   2. 一个课件在数据库层最多只有一个有效装配运行；
--   3. 页面写回必须绑定当前装配版本和运行ID；
--   4. 页面HTML变化后，历史布局验收结果自动失效；
--   5. 为后续真实浏览器验收保存状态、HTML哈希和结构化报告。
--
-- 本迁移只建立数据契约，不改变现有装配入口，也不自动重写历史课件。
-- ============================================================================

ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS assembly_version BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS assembly_status TEXT NOT NULL DEFAULT 'idle',
    ADD COLUMN IF NOT EXISTS assembly_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS assembly_finished_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS assembly_started_by UUID,
    ADD COLUMN IF NOT EXISTS assembly_skip_video BOOLEAN NOT NULL DEFAULT TRUE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_coursewares_assembly_version_nonnegative'
    ) THEN
        ALTER TABLE coursewares
            ADD CONSTRAINT chk_coursewares_assembly_version_nonnegative
            CHECK (assembly_version >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_coursewares_assembly_status'
    ) THEN
        ALTER TABLE coursewares
            ADD CONSTRAINT chk_coursewares_assembly_status
            CHECK (
                assembly_status IN (
                    'idle',
                    'running',
                    'cancel_requested',
                    'completed',
                    'cancelled',
                    'failed',
                    'interrupted'
                )
            );
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS courseware_assembly_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    courseware_id UUID NOT NULL
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    version BIGINT NOT NULL,
    started_by UUID NOT NULL,
    skip_video BOOLEAN NOT NULL DEFAULT TRUE,

    status TEXT NOT NULL DEFAULT 'running',
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,

    CONSTRAINT uq_courseware_assembly_runs_version
        UNIQUE (courseware_id, version),

    CONSTRAINT chk_courseware_assembly_runs_version
        CHECK (version > 0),

    CONSTRAINT chk_courseware_assembly_runs_status
        CHECK (
            status IN (
                'running',
                'cancel_requested',
                'completed',
                'cancelled',
                'failed',
                'interrupted'
            )
        ),

    CONSTRAINT chk_courseware_assembly_runs_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_courseware_assembly_runs_active
    ON courseware_assembly_runs(courseware_id)
    WHERE status IN ('running', 'cancel_requested');

CREATE INDEX IF NOT EXISTS idx_courseware_assembly_runs_courseware_started
    ON courseware_assembly_runs(courseware_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_courseware_assembly_runs_status
    ON courseware_assembly_runs(status, updated_at);

ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS active_assembly_run_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_coursewares_active_assembly_run'
    ) THEN
        ALTER TABLE coursewares
            ADD CONSTRAINT fk_coursewares_active_assembly_run
            FOREIGN KEY (active_assembly_run_id)
            REFERENCES courseware_assembly_runs(id)
            ON DELETE SET NULL;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_coursewares_active_assembly_run
    ON coursewares(active_assembly_run_id)
    WHERE active_assembly_run_id IS NOT NULL;

ALTER TABLE courseware_pages
    ADD COLUMN IF NOT EXISTS assembly_version BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS layout_status TEXT NOT NULL DEFAULT 'unchecked',
    ADD COLUMN IF NOT EXISTS layout_audit_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS layout_html_hash CHAR(64),
    ADD COLUMN IF NOT EXISTS layout_checked_at TIMESTAMPTZ;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_courseware_pages_assembly_version_nonnegative'
    ) THEN
        ALTER TABLE courseware_pages
            ADD CONSTRAINT chk_courseware_pages_assembly_version_nonnegative
            CHECK (assembly_version >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_courseware_pages_layout_status'
    ) THEN
        ALTER TABLE courseware_pages
            ADD CONSTRAINT chk_courseware_pages_layout_status
            CHECK (
                layout_status IN (
                    'unchecked',
                    'checking',
                    'passed',
                    'failed',
                    'repairing',
                    'error'
                )
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_courseware_pages_layout_audit_object'
    ) THEN
        ALTER TABLE courseware_pages
            ADD CONSTRAINT chk_courseware_pages_layout_audit_object
            CHECK (jsonb_typeof(layout_audit_json) = 'object');
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_courseware_pages_layout_hash'
    ) THEN
        ALTER TABLE courseware_pages
            ADD CONSTRAINT chk_courseware_pages_layout_hash
            CHECK (
                layout_html_hash IS NULL
                OR layout_html_hash ~ '^[0-9a-f]{64}$'
            );
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_courseware_pages_layout_status
    ON courseware_pages(courseware_id, layout_status, page_number);

CREATE INDEX IF NOT EXISTS idx_courseware_pages_assembly_version
    ON courseware_pages(courseware_id, assembly_version, page_number);

COMMENT ON TABLE courseware_assembly_runs IS
    '课件自动装配不可变运行记录；courseware_id+version唯一，旧运行不得覆盖新版本。';

COMMENT ON COLUMN coursewares.assembly_version IS
    '当前课件装配业务版本；每次正式装配领取时单调递增。';

COMMENT ON COLUMN coursewares.assembly_status IS
    '当前装配状态，与课件生产status和publish_state正交。';

COMMENT ON COLUMN coursewares.active_assembly_run_id IS
    '当前有效装配运行ID；终态后清空。';

COMMENT ON COLUMN courseware_pages.assembly_version IS
    '最后一次成功写入本页HTML的装配版本；0表示历史或非装配写入。';

COMMENT ON COLUMN courseware_pages.layout_status IS
    '真实浏览器布局验收状态；HTML改变后必须重置为unchecked。';

COMMENT ON COLUMN courseware_pages.layout_audit_json IS
    '浏览器布局验收结构化报告，不保存截图二进制。';

COMMENT ON COLUMN courseware_pages.layout_html_hash IS
    '执行布局验收时对应页面HTML的SHA-256十六进制哈希。';

COMMIT;
