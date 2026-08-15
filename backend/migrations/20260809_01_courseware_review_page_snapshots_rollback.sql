-- ============================================================================
-- TE-DNA 2.0：R-03 课件审核历史页面快照回滚
-- 文件：20260809_01_courseware_review_page_snapshots_rollback.sql
-- ----------------------------------------------------------------------------
-- 严重警告：
--   1. 本文件会删除 courseware_review_page_snapshots 中全部历史页面证据；
--   2. 已经产生正式 R-03 审核记录后，不应依赖本文件恢复业务数据；
--   3. 生产环境执行本回滚前必须再次独立备份数据库；
--   4. 若已经产生真实审核页面快照，优先恢复迁移/发布前数据库备份。
--
-- 本回滚不会修改：
--   - courseware_reviews；
--   - courseware_review_feedback；
--   - courseware_review_items；
--   - courseware_review_instruction_versions；
--   - courseware_pages；
--   - courseware_page_versions。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_review_page_snapshots'
    ) IS NULL THEN
        RAISE EXCEPTION
            'R-03审核页面快照表不存在，禁止执行非预期回滚';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS
    trg_cw_review_page_snapshot_immutable
ON public.courseware_review_page_snapshots;

DROP TRIGGER IF EXISTS
    trg_cw_review_page_snapshot_validate_insert
ON public.courseware_review_page_snapshots;

DROP TABLE public.courseware_review_page_snapshots;

DROP FUNCTION IF EXISTS
    public.guard_cw_review_page_snapshot_write();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-03课件审核历史页面快照结构已回滚';
END
$$;
