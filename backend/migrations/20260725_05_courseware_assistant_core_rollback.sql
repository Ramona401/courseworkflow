-- ============================================================================
-- TE-DNA 2.0：课件原生教学智能体核心数据表回滚
-- 文件：20260725_05_courseware_assistant_core_rollback.sql
-- ----------------------------------------------------------------------------
-- 仅用于开发单元02尚未被后续运行会话表引用时回滚。
--
-- 如果assistant_runtime_sessions或assistant_runtime_usage已经建立，
-- 必须先执行对应开发单元的回滚迁移。
--
-- 本回滚不使用CASCADE，存在未知依赖时会安全失败，避免误删其它业务对象。
-- ============================================================================

BEGIN;

DROP TABLE IF EXISTS assistant_deployment_versions;
DROP TABLE IF EXISTS assistant_deployments;
DROP TABLE IF EXISTS courseware_assistant_slots;

DROP FUNCTION IF EXISTS
    tedna_reject_assistant_deployment_version_update();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件教学智能体核心表已回滚';
END
$$;
