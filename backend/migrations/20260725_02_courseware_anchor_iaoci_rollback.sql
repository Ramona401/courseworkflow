-- ============================================================================
-- TE-DNA 2.0：课程锚点IAOCI同步批次回滚
-- ----------------------------------------------------------------------------
-- 执行前必须重新备份数据库。
-- 本回滚仅撤销第二批内容，不删除第一批建立的IAOCI基础表。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS trg_courseware_anchor_iaoci_sync
ON coursewares;

DROP FUNCTION IF EXISTS tedna_courseware_anchor_sync_trigger();
DROP FUNCTION IF EXISTS tedna_sync_courseware_anchor_iaoci(uuid, uuid, text);
DROP FUNCTION IF EXISTS tedna_image_aoci_tag_value(text, text);
DROP FUNCTION IF EXISTS tedna_image_aoci_header_value(text, text);

DELETE FROM courseware_image_indexes
WHERE index_type = 'A';

DELETE FROM prompts
WHERE prompt_key = 'prompt_courseware_image_anchor_iaoci';

COMMIT;
