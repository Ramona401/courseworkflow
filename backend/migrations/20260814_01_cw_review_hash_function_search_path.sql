-- ============================================================================
-- TE-DNA 2.0：课件AI审核Hash函数灾备恢复可移植性修复
-- 文件：20260814_01_cw_review_hash_function_search_path.sql
-- ----------------------------------------------------------------------------
-- 背景：
--   pg_restore会把会话search_path收紧。
--   以下3个SQL函数内部使用未限定schema的digest(...)。
--   在普通运行连接中public可见，因此生产运行正常；
--   但向空数据库完整恢复时，courseware_ai_review_sessions COPY会触发
--   build_cw_ai_review_config_hash，并因无法解析digest而失败。
--
-- 修复：
--   只为3个函数设置函数级search_path = public, pg_temp。
--   不改变函数正文、参数、返回值、哈希算法或任何业务数据。
-- ============================================================================

BEGIN;

ALTER FUNCTION public.build_cw_ai_review_config_hash(
    SMALLINT,
    JSONB,
    TEXT,
    TEXT
)
SET search_path = public, pg_temp;

ALTER FUNCTION public.build_cw_review_impact_message_hash(
    TEXT,
    JSONB
)
SET search_path = public, pg_temp;

ALTER FUNCTION public.build_cw_review_impact_operations_hash(
    JSONB
)
SET search_path = public, pg_temp;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件AI审核3个Hash函数search_path已固定为public, pg_temp';
END
$$;
