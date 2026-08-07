-- 20260804_01_courseware_review_item_resolution_rollback.sql
--
-- 仅回滚V1.3新增结构。
--
-- 注意：
--
--   1. 正向迁移会按照当前页面真实内容指纹，将旧版本自动关闭的问题
--      恢复为applied、stale、orphaned、confirmed或detected。
--
--   2. 该数据纠正不会在结构回滚时自动改回resolved，因为无法可靠区分：
--
--        - 旧流程自动关闭的问题；
--        - 迁移后由作者明确确认解决的问题；
--        - 迁移后由正式审核员复审确认解决的问题。
--
--   3. pgcrypto属于数据库共享扩展，本回滚不会删除。
--
--   4. 已经产生V1.3业务数据后需要回滚时，应优先恢复迁移前完整数据库备份。

BEGIN;

DROP INDEX IF EXISTS
    idx_cw_review_item_resolution_review;

DROP INDEX IF EXISTS
    idx_cw_review_item_resubmitted;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_resolution,
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_resubmission,
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_resolution_review,
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_resolved_by;

ALTER TABLE courseware_review_items
    DROP COLUMN IF EXISTS
        resolution_note,
    DROP COLUMN IF EXISTS
        resolved_review_round,
    DROP COLUMN IF EXISTS
        resolved_review_level,
    DROP COLUMN IF EXISTS
        resolved_review_id,
    DROP COLUMN IF EXISTS
        resolved_by,
    DROP COLUMN IF EXISTS
        resubmitted_review_round,
    DROP COLUMN IF EXISTS
        resubmitted_review_level,
    DROP COLUMN IF EXISTS
        resubmitted_at;

COMMIT;
