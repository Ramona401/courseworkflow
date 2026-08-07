-- ============================================================================
-- TE-DNA 2.0：课件审核修改指令版本守卫 V2.1 回滚
-- 文件：20260805_03_courseware_review_instruction_version_guards_rollback.sql
-- ----------------------------------------------------------------------------
-- 本文件只撤销数据库守卫和版本表权限，不删除版本结构。
--
-- 必须在同一个psql连接中紧接着执行：
--
--   20260805_02_courseware_review_instruction_versions_rollback.sql
--
-- 第二个文件删除结构并统一COMMIT。
-- 两个回滚文件执行前必须再次独立备份数据库。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_review_instruction_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '指令版本表不存在，无法执行守卫回滚';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_instruction_binding_guard
ON courseware_review_items;

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_instruction_bindings();

DROP TRIGGER IF EXISTS
    trg_cw_review_instruction_version_mutation_guard
ON courseware_review_instruction_versions;

DROP FUNCTION IF EXISTS
    public.guard_cw_review_instruction_version_mutation();

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_instruction_versions
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_instruction_versions
FROM tedna_user;

DO $$
BEGIN
    RAISE NOTICE
        'R-01指令版本守卫已撤销，等待同事务执行结构回滚';
END
$$;

-- 本文件故意不执行COMMIT。
