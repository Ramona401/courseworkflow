-- ============================================================================
-- 20260713_create_lesson_plan_content_versions.sql
-- 教案正文历史版本表
--
-- 目标：
--   1. 每次人工编辑、AI生成、导入或恢复正文前，保存修改前的完整快照。
--   2. 支持版本列表、版本详情、前后对比及一键恢复。
--   3. 快照随教案删除自动清理，操作人删除后保留快照但将 changed_by 置空。
--
-- 设计说明：
--   - version_number 保存快照对应的 lesson_plans.version。
--   - content_markdown 保存可恢复的完整正文。
--   - content_structured、标题和课时时长一并保存，避免恢复时结构不完整。
--   - change_source 标记触发本次修改的来源：
--       manual    人工编辑
--       ai        AI生成或AI修订
--       import    导入已有教案
--       restore   历史版本恢复
--       system    其它既有系统路径
--   - 本迁移使用 IF NOT EXISTS，可安全重复执行。
-- ============================================================================

CREATE TABLE IF NOT EXISTS lesson_plan_content_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    lesson_plan_id UUID NOT NULL
        REFERENCES lesson_plans(id)
        ON DELETE CASCADE,

    version_number INTEGER NOT NULL,

    title TEXT NOT NULL DEFAULT '',

    content_markdown TEXT NOT NULL DEFAULT '',

    content_structured JSONB NOT NULL DEFAULT '{}'::jsonb,

    duration_minutes INTEGER NOT NULL DEFAULT 45,

    change_source VARCHAR(30) NOT NULL DEFAULT 'system',

    changed_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    change_summary TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_lesson_plan_content_version
        UNIQUE (lesson_plan_id, version_number),

    CONSTRAINT chk_lesson_plan_version_number
        CHECK (version_number >= 1),

    CONSTRAINT chk_lesson_plan_version_source
        CHECK (
            change_source IN (
                'manual',
                'ai',
                'import',
                'restore',
                'system'
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_content_versions_plan
    ON lesson_plan_content_versions (
        lesson_plan_id,
        version_number DESC
    );

CREATE INDEX IF NOT EXISTS idx_lesson_plan_content_versions_created
    ON lesson_plan_content_versions (
        lesson_plan_id,
        created_at DESC
    );

COMMENT ON TABLE lesson_plan_content_versions IS
    '教案正文修改前快照，用于版本历史、对比与恢复';

COMMENT ON COLUMN lesson_plan_content_versions.version_number IS
    '快照对应的lesson_plans.version版本号';

COMMENT ON COLUMN lesson_plan_content_versions.change_source IS
    '触发下一次正文修改的来源：manual/ai/import/restore/system';

COMMENT ON COLUMN lesson_plan_content_versions.changed_by IS
    '触发正文修改的用户ID；系统或AI后台操作可为空';

COMMENT ON COLUMN lesson_plan_content_versions.change_summary IS
    '本次修改或恢复操作的简短说明';

DO $$
BEGIN
    RAISE NOTICE
        '教案正文版本历史表迁移完成：lesson_plan_content_versions';
END $$;
