-- ============================================================================
-- TE-DNA 2.0：课件AI审核Hash函数灾备恢复可移植性修复回滚
-- 文件：20260814_01_cw_review_hash_function_search_path_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--   回滚会重新暴露普通pg_restore期间无法解析public.digest的风险。
--   仅用于明确需要撤销本次函数级配置时。
-- ============================================================================

BEGIN;

ALTER FUNCTION public.build_cw_ai_review_config_hash(
    SMALLINT,
    JSONB,
    TEXT,
    TEXT
)
RESET search_path;

ALTER FUNCTION public.build_cw_review_impact_message_hash(
    TEXT,
    JSONB
)
RESET search_path;

ALTER FUNCTION public.build_cw_review_impact_operations_hash(
    JSONB
)
RESET search_path;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件AI审核3个Hash函数search_path配置已回滚';
END
$$;
