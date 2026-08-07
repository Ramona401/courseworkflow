-- 20260724_02_courseware_annotation_page_binding_rollback.sql
--
-- 回滚课件人工批注稳定页面关联。
--
-- 说明：
--   1. 原page_number列从未删除，旧代码可继续使用；
--   2. 先删除兼容触发器和函数；
--   3. 再删除复合外键、校验约束、索引和新增字段；
--   4. 不删除任何批注业务记录。

BEGIN;

DROP TRIGGER IF EXISTS
  trg_courseware_annotation_page_binding
ON courseware_annotations;

DROP FUNCTION IF EXISTS
  public.sync_courseware_annotation_page_binding();

ALTER TABLE courseware_annotations
  DROP CONSTRAINT IF EXISTS
    courseware_annotations_page_courseware_fk;

ALTER TABLE courseware_annotations
  DROP CONSTRAINT IF EXISTS
    courseware_annotations_page_number_snapshot_check;

DROP INDEX IF EXISTS
  idx_courseware_annotations_page_id;

DROP INDEX IF EXISTS
  idx_courseware_annotations_courseware_snapshot;

ALTER TABLE courseware_annotations
  DROP COLUMN IF EXISTS page_id,
  DROP COLUMN IF EXISTS page_number_snapshot;

COMMIT;
