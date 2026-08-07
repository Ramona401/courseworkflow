-- ============================================================================
-- TE-DNA 2.0：课件教学智能体运行会话与使用流水回滚
-- 文件：20260725_07_assistant_runtime_rollback.sql
-- ----------------------------------------------------------------------------
-- 本回滚删除开发单元03建立的运行会话、使用流水和流水不可变函数。
--
-- 不使用CASCADE。
-- 如果后续数据库对象已经依赖这些表，本回滚会安全失败。
-- ============================================================================

BEGIN;

DROP TABLE IF EXISTS assistant_runtime_usage;
DROP TABLE IF EXISTS assistant_runtime_sessions;

DROP FUNCTION IF EXISTS
    tedna_reject_assistant_runtime_usage_update();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '教学智能体运行会话和使用流水已回滚';
END
$$;
