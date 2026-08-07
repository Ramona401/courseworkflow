-- ============================================================================
-- TE-DNA 2.0：运行使用流水软关联修复回滚
-- 文件：20260726_08_assistant_runtime_usage_soft_links_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--
-- 本回滚仅能在修复后尚未删除任何父部署或父会话时执行。
-- 如果流水中的软关联UUID已经没有对应父记录，重新建立外键会安全失败。
--
-- 回滚后会重新出现：
--   不可变UPDATE触发器与ON DELETE SET NULL之间的结构冲突。
-- 因此本文件只用于开发阶段紧急恢复，不应作为正常生产结构。
-- ============================================================================

BEGIN;

LOCK TABLE assistant_runtime_usage
IN ACCESS EXCLUSIVE MODE;

ALTER TABLE assistant_runtime_usage
    ALTER COLUMN deployment_id DROP NOT NULL,
    ALTER COLUMN runtime_session_id DROP NOT NULL;

ALTER TABLE assistant_runtime_usage
    ADD CONSTRAINT
        fk_assistant_runtime_usage_deployment
    FOREIGN KEY (deployment_id)
    REFERENCES assistant_deployments(id)
    ON DELETE SET NULL;

ALTER TABLE assistant_runtime_usage
    ADD CONSTRAINT
        fk_assistant_runtime_usage_session
    FOREIGN KEY (runtime_session_id)
    REFERENCES assistant_runtime_sessions(id)
    ON DELETE SET NULL;

COMMENT ON COLUMN assistant_runtime_usage.deployment_id IS
    '部署删除时置空的历史关联ID';

COMMENT ON COLUMN assistant_runtime_usage.runtime_session_id IS
    '运行会话删除时置空的历史关联ID';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '运行使用流水软关联修复已回滚';
END
$$;
