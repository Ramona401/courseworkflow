-- ============================================================================
-- TE-DNA 2.0：课件审核修改指令版本结构 V2.1 回滚
-- 文件：20260805_02_courseware_review_instruction_versions_rollback.sql
-- ----------------------------------------------------------------------------
-- 严重警告：
--   1. 本文件会删除全部指令版本历史；
--   2. 会删除正式交付版本和页面应用版本引用；
--   3. 已经形成V2、V3或更高版本后，confirmed_instruction只保留
--      回滚执行时的当前兼容快照，无法还原完整版本历史；
--   4. 已产生正式业务数据后应优先恢复迁移前数据库备份；
--   5. 执行回滚前必须再次独立备份数据库。
--
-- 本文件必须在同一个psql连接中，紧接着守卫回滚文件执行。
-- 本回滚不删除共享pgcrypto扩展。
-- ============================================================================

DO $$
BEGIN
    -- 守卫必须先在同一事务中撤销，避免删除结构时遗留依赖。
    IF to_regprocedure(
        'public.guard_cw_review_item_instruction_bindings()'
    ) IS NOT NULL
       OR to_regprocedure(
           'public.guard_cw_review_instruction_version_mutation()'
       ) IS NOT NULL THEN
        RAISE EXCEPTION
            '必须先在同一事务执行指令版本守卫回滚';
    END IF;
END
$$;

DROP INDEX IF EXISTS
    idx_cw_review_item_applied_instruction_version;

DROP INDEX IF EXISTS
    idx_cw_review_item_delivered_instruction_version;

DROP INDEX IF EXISTS
    idx_cw_review_item_current_instruction_version;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_applied_instruction_version,
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_delivered_instruction_version,
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_instruction_compat,
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_applied_instruction_version,
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_delivered_instruction_version,
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_current_instruction_version;

ALTER TABLE courseware_review_items
    DROP COLUMN IF EXISTS
        applied_instruction_version_id,
    DROP COLUMN IF EXISTS
        delivered_instruction_version_id,
    DROP COLUMN IF EXISTS
        current_instruction_version_id;

DROP TABLE IF EXISTS
    courseware_review_instruction_versions;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件审核修改指令不可变版本体系V2.1数据库结构已回滚';
END
$$;
