-- ============================================================================
-- TE-DNA 2.0：课件原生教学智能体核心表权限修复回滚
-- 文件：20260725_06_courseware_assistant_core_privileges_rollback.sql
-- ----------------------------------------------------------------------------
-- 本文件只回滚20260725_06权限收紧。
--
-- 警告：
-- 执行后会恢复tedna_user对三张核心表的全部权限，
-- 包括版本表UPDATE、DELETE和TRUNCATE。
-- 仅在确需恢复迁移前权限状态时使用。
-- ============================================================================

BEGIN;

REVOKE ALL PRIVILEGES
ON TABLE
    courseware_assistant_slots,
    assistant_deployments,
    assistant_deployment_versions
FROM tedna_user;

GRANT ALL PRIVILEGES
ON TABLE
    courseware_assistant_slots,
    assistant_deployments,
    assistant_deployment_versions
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件教学智能体核心表权限修复已回滚';
END
$$;
