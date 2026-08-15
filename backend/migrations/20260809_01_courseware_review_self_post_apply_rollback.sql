-- ============================================================================
-- TE-DNA 2.0：作者自审修改完成后三项人工决策数据库守卫回滚
-- 文件：20260809_01_courseware_review_self_post_apply_rollback.sql
-- ----------------------------------------------------------------------------
-- 回滚恢复20260809_01之前的数据库状态：
--
--   - 删除self dismissed -> applied专用恢复守卫；
--   - 恢复原指令绑定触发器对全部UPDATE生效；
--   - dismissed不再允许保留applied版本事实。
--
-- 安全边界：
--
--   如果业务已经产生“带applied事实的dismissed”，本文件拒绝继续。
--   此时应优先恢复迁移前完整数据库备份，不能静默丢弃历史事实。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND (
                applied_instruction_version_id IS NOT NULL
                OR applied_at IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION
            '已有暂时不处理的修改完成事实，禁止结构回滚；请恢复迁移前数据库备份';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS
    trg_01_cw_review_item_self_applied_restore_guard
ON courseware_review_items;

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_self_applied_restore();

-- 恢复既有绑定守卫原本的无条件触发方式。
DROP TRIGGER IF EXISTS
    trg_cw_review_item_instruction_binding_guard
ON courseware_review_items;

CREATE TRIGGER
    trg_cw_review_item_instruction_binding_guard
BEFORE UPDATE
ON courseware_review_items
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_item_instruction_bindings();

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_applied_instruction_version;

ALTER TABLE courseware_review_items
    ADD CONSTRAINT
        chk_cw_review_item_applied_instruction_version
    CHECK (
        (
            applied_instruction_version_id IS NULL
            AND applied_at IS NULL
            AND status <> 'applying'
        )
        OR
        (
            applied_instruction_version_id IS NOT NULL
            AND status = 'applying'
            AND applied_at IS NULL
        )
        OR
        (
            applied_instruction_version_id IS NOT NULL
            AND status IN (
                'applied',
                'resolved',
                'stale',
                'orphaned'
            )
            AND applied_at IS NOT NULL
        )
    );

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-01.1作者自审修改完成后三项人工决策数据库守卫已回滚';
END
$$;
