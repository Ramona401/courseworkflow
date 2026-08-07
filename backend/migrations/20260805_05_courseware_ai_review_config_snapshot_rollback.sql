-- ============================================================================
-- TE-DNA 2.0：R-02课件AI审核配置快照回滚
-- 文件：20260805_05_courseware_ai_review_config_snapshot_rollback.sql
-- ----------------------------------------------------------------------------
-- 安全原则：
--   1. 旧后端可以忽略新增字段，因此代码回滚优先保留已迁移数据库；
--   2. 本脚本仅用于尚未产生非默认R-02配置的迁移级回滚；
--   3. 一旦存在严格一致、参考意图、不使用教案、自定义维度或缩减维度，
--      本脚本会拒绝删除字段，防止丢失历史审核配置事实。
-- ============================================================================

BEGIN;

DO $$
DECLARE
    default_dimensions JSONB :=
        '[
            "teaching_logic",
            "technical_implementation",
            "interaction_experience",
            "lesson_alignment",
            "authenticity",
            "knowledge_accuracy",
            "page_readability",
            "operational_usability"
        ]'::jsonb;
BEGIN
    IF to_regclass(
        'public.courseware_ai_review_sessions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_ai_review_sessions表';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_ai_review_sessions'
          AND column_name = 'review_dimensions_json'
    ) AND EXISTS (
        SELECT 1
        FROM courseware_ai_review_sessions
        WHERE review_config_schema_version <> 1
           OR review_dimensions_json <>
               default_dimensions
           OR BTRIM(
               custom_dimension_description
           ) <> ''
           OR lesson_reference_mode <>
               'current_compatible'
    ) THEN
        RAISE EXCEPTION
            '已经存在非默认R-02审核配置，回滚会丢失历史事实，已拒绝删除字段';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS
    trg_00_cw_ai_review_session_config_snapshot_guard
ON courseware_ai_review_sessions;

DROP FUNCTION IF EXISTS
    public.guard_cw_ai_review_session_config_snapshot();

ALTER TABLE courseware_ai_review_sessions
    DROP CONSTRAINT IF EXISTS
        chk_cw_ai_review_config_hash,
    DROP CONSTRAINT IF EXISTS
        chk_cw_ai_review_custom_dimension,
    DROP CONSTRAINT IF EXISTS
        chk_cw_ai_review_lesson_reference_mode,
    DROP CONSTRAINT IF EXISTS
        chk_cw_ai_review_dimensions,
    DROP CONSTRAINT IF EXISTS
        chk_cw_ai_review_config_schema_version;

ALTER TABLE courseware_ai_review_sessions
    DROP COLUMN IF EXISTS review_config_hash,
    DROP COLUMN IF EXISTS lesson_reference_mode,
    DROP COLUMN IF EXISTS custom_dimension_description,
    DROP COLUMN IF EXISTS review_dimensions_json,
    DROP COLUMN IF EXISTS review_config_schema_version;

DROP FUNCTION IF EXISTS
    public.build_cw_ai_review_config_hash(
        SMALLINT,
        JSONB,
        TEXT,
        TEXT
    );

DROP FUNCTION IF EXISTS
    public.is_valid_cw_ai_review_dimensions(JSONB);

DROP FUNCTION IF EXISTS
    public.normalize_cw_ai_review_dimensions(JSONB);

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-02课件AI审核配置快照迁移已回滚';
END
$$;
