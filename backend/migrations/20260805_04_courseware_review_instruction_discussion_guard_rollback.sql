-- ============================================================================
-- TE-DNA 2.0：重新讨论保留当前确认版本守卫回滚
-- 文件：20260805_04_courseware_review_instruction_discussion_guard_rollback.sql
-- ----------------------------------------------------------------------------
-- 只移除20260805_04新增的前置守卫。
-- 不删除指令版本、不修改整改项数据、不恢复旧的清空行为。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS
    trg_00_cw_review_item_discussion_version_guard
ON courseware_review_items;

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_discussion_preserves_instruction();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-01重新讨论版本保持守卫已移除';
END
$$;
