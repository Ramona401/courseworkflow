-- ============================================================================
-- TE-DNA 2.0：全局讨论结论落地与问题列表治理 V1.1 回滚
-- 文件：20260803_01_courseware_review_item_governance_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--   1. 本文件会删除全部整改项关系及其治理事件；
--   2. 会删除整改项人工新增来源信息；
--   3. 只允许在本功能尚未产生正式业务数据时执行；
--   4. 已产生人工新增整改项或治理关系后，应从完整备份恢复；
--   5. 执行前必须再次备份数据库。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_origin_guard
ON courseware_review_items;

DROP TABLE IF EXISTS
    courseware_review_item_relation_events;

DROP TABLE IF EXISTS
    courseware_review_item_relations;

DROP FUNCTION IF EXISTS
    public.enforce_cw_review_item_relation_event_consistency();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_relation_event_insert();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_relation_mutation();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_origin();

DROP INDEX IF EXISTS
    idx_cw_review_item_source_global_message;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_source_global_message_session;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_source_global_message;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_origin_source;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_origin_type;

ALTER TABLE courseware_review_items
    DROP COLUMN IF EXISTS source_global_message_id;

ALTER TABLE courseware_review_items
    DROP COLUMN IF EXISTS origin_type;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        uq_cw_review_items_id_courseware_session;

ALTER TABLE courseware_ai_review_messages
    DROP CONSTRAINT IF EXISTS
        uq_cw_ai_review_messages_id_session;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件审核问题列表治理V1.1数据库结构已回滚';
END
$$;
