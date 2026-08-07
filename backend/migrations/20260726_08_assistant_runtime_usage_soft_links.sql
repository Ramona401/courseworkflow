-- ============================================================================
-- TE-DNA 2.0：运行使用流水软关联审计快照修复
-- 文件：20260726_08_assistant_runtime_usage_soft_links.sql
-- ----------------------------------------------------------------------------
-- 修复背景：
--
-- assistant_runtime_usage是严格追加式流水，存在BEFORE UPDATE拒绝触发器。
-- 原结构同时把deployment_id和runtime_session_id设置为：
--
--   FOREIGN KEY ... ON DELETE SET NULL
--
-- 删除父部署或父会话时，PostgreSQL会尝试UPDATE流水记录，
-- 随后被不可变触发器拒绝，导致父记录删除和课件永久清理失败。
--
-- 最终设计：
--
--   1. deployment_id和runtime_session_id都是调用发生时的审计快照；
--   2. 两列始终非空；
--   3. 两列不建立外键；
--   4. 父记录删除后仍保留原始UUID；
--   5. usage仍保持只能INSERT、不能UPDATE或DELETE；
--   6. 不修改任何既有使用流水值。
-- ============================================================================

BEGIN;

-- 阻止迁移检查和结构修改之间出现并发写入。
LOCK TABLE assistant_runtime_usage
IN ACCESS EXCLUSIVE MODE;

DO $$
DECLARE
    missing_link_count BIGINT;
BEGIN
    IF to_regclass(
        'public.assistant_runtime_usage'
    ) IS NULL THEN
        RAISE EXCEPTION
            'assistant_runtime_usage不存在，不能执行软关联修复';
    END IF;

    SELECT COUNT(*)
    INTO missing_link_count
    FROM assistant_runtime_usage
    WHERE deployment_id IS NULL
       OR runtime_session_id IS NULL;

    IF missing_link_count > 0 THEN
        RAISE EXCEPTION
            '发现%条缺少部署或会话ID的运行流水，不能安全设置NOT NULL',
            missing_link_count;
    END IF;
END
$$;

-- 删除会通过SET NULL改写不可变流水的两个外键。
ALTER TABLE assistant_runtime_usage
    DROP CONSTRAINT IF EXISTS
        fk_assistant_runtime_usage_deployment;

ALTER TABLE assistant_runtime_usage
    DROP CONSTRAINT IF EXISTS
        fk_assistant_runtime_usage_session;

-- 每条使用流水必须永久保存原始部署和会话标识。
ALTER TABLE assistant_runtime_usage
    ALTER COLUMN deployment_id SET NOT NULL,
    ALTER COLUMN runtime_session_id SET NOT NULL;

COMMENT ON COLUMN assistant_runtime_usage.deployment_id IS
    '调用发生时的部署ID不可变审计快照；非空软关联，不建立外键，父部署删除后仍保留原UUID';

COMMENT ON COLUMN assistant_runtime_usage.runtime_session_id IS
    '调用发生时的运行会话ID不可变审计快照；非空软关联，不建立外键，父会话删除后仍保留原UUID';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '运行使用流水软关联修复完成：部署ID和会话ID已改为非空不可变审计快照';
END
$$;
