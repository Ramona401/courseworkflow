-- 20260724_01_courseware_review_items_rollback.sql
--
-- 仅用于本功能尚未产生正式业务数据时回滚数据库结构。
-- 执行本文件会删除整改项、正式反馈快照及其讨论消息关联，
-- 上线产生业务数据后不得直接执行，应从完整数据库备份恢复或另做数据迁移。

BEGIN;

ALTER TABLE courseware_ai_review_messages
    DROP CONSTRAINT IF EXISTS fk_cw_ai_review_message_item;

DROP INDEX IF EXISTS idx_cw_ai_review_message_item;

ALTER TABLE courseware_ai_review_messages
    DROP COLUMN IF EXISTS review_item_id;

DROP TABLE IF EXISTS courseware_review_items;
DROP TABLE IF EXISTS courseware_review_feedback;

COMMIT;
