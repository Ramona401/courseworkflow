-- ============================================================================
-- TE-DNA 2.0：课件图片 IAOCI 索引回滚
-- 文件：20260725_01_courseware_image_aoci_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--   本文件会删除全部图片IAOCI索引和图片关系数据。
--   执行前必须再次备份数据库。
-- ============================================================================

BEGIN;

DROP TABLE IF EXISTS courseware_image_relations;
DROP TABLE IF EXISTS courseware_image_indexes;

DROP INDEX IF EXISTS ux_courseware_pages_id_courseware;

COMMIT;
