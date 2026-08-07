-- 20260724_02_courseware_annotation_page_binding.sql
--
-- 将课件人工批注从页码软关联升级为稳定页面关联。
--
-- 字段语义：
--   page_id                 稳定页面ID；
--   page_number_snapshot    批注创建时页码；
--   page_number             保留旧列，兼容旧代码和回滚。
--
-- 兼容策略：
--   1. 迁移先回填历史数据；
--   2. 触发器兼容仍只提交page_number的旧后端；
--   3. 新后端可同时提交page_id和页码快照；
--   4. 页面删除时仅清空page_id，批注历史继续保留；
--   5. 页面重排后，查询层通过page_id解析当前页码。

BEGIN;

ALTER TABLE courseware_annotations
  ADD COLUMN IF NOT EXISTS page_id UUID,
  ADD COLUMN IF NOT EXISTS page_number_snapshot INTEGER;

-- 旧page_number就是历史批注创建时的页码。
UPDATE courseware_annotations
SET page_number_snapshot = page_number
WHERE page_number_snapshot IS NULL;

-- 根据课件和历史页码回填稳定页面ID。
UPDATE courseware_annotations AS annotation
SET page_id = page.id
FROM courseware_pages AS page
WHERE annotation.page_id IS NULL
  AND page.courseware_id = annotation.courseware_id
  AND page.page_number = annotation.page_number_snapshot;

-- 首次建立稳定页面外键时，历史数据必须全部能绑定真实页面。
-- 外键已经存在时允许page_id为空，因为页面删除会按设计清空该字段并保留批注历史。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname =
      'courseware_annotations_page_courseware_fk'
      AND conrelid =
        'public.courseware_annotations'::regclass
  )
  AND EXISTS (
    SELECT 1
    FROM courseware_annotations
    WHERE page_id IS NULL
  ) THEN
    RAISE EXCEPTION
      '存在无法匹配真实页面的历史课件批注，停止迁移';
  END IF;
END
$$;

-- 兼容旧后端新增批注：
-- 旧代码仍只写courseware_id和page_number时，
-- 数据库自动解析page_id并保存创建时页码快照。
-- 只监听INSERT，避免页面删除时外键SET NULL触发重新绑定。
CREATE OR REPLACE FUNCTION
  public.sync_courseware_annotation_page_binding()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  bound_page_number INTEGER;
BEGIN
  IF NEW.page_id IS NULL THEN
    IF NEW.page_number IS NULL
      OR NEW.page_number <= 0 THEN
      RAISE EXCEPTION
        '课件批注缺少有效页面定位'
        USING ERRCODE = '23503';
    END IF;

    SELECT
      page.id,
      page.page_number
    INTO
      NEW.page_id,
      bound_page_number
    FROM courseware_pages AS page
    WHERE page.courseware_id = NEW.courseware_id
      AND page.page_number = NEW.page_number
    LIMIT 1;
  ELSE
    SELECT
      page.page_number
    INTO
      bound_page_number
    FROM courseware_pages AS page
    WHERE page.id = NEW.page_id
      AND page.courseware_id = NEW.courseware_id
    LIMIT 1;
  END IF;

  IF NEW.page_id IS NULL
    OR bound_page_number IS NULL THEN
    RAISE EXCEPTION
      '课件批注目标页面不存在或不属于当前课件'
      USING ERRCODE = '23503';
  END IF;

  IF NEW.page_number_snapshot IS NULL
    OR NEW.page_number_snapshot <= 0 THEN
    IF NEW.page_number IS NOT NULL
      AND NEW.page_number > 0 THEN
      NEW.page_number_snapshot :=
        NEW.page_number;
    ELSE
      NEW.page_number_snapshot :=
        bound_page_number;
    END IF;
  END IF;

  -- 旧page_number继续保存创建当时页码，
  -- 当前页码由查询层根据page_id动态解析。
  IF NEW.page_number IS NULL
    OR NEW.page_number <= 0 THEN
    NEW.page_number :=
      bound_page_number;
  END IF;

  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS
  trg_courseware_annotation_page_binding
ON courseware_annotations;

CREATE TRIGGER
  trg_courseware_annotation_page_binding
BEFORE INSERT
ON courseware_annotations
FOR EACH ROW
EXECUTE FUNCTION
  public.sync_courseware_annotation_page_binding();

ALTER TABLE courseware_annotations
  ALTER COLUMN page_number_snapshot
  SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname =
      'courseware_annotations_page_number_snapshot_check'
  ) THEN
    ALTER TABLE courseware_annotations
      ADD CONSTRAINT
        courseware_annotations_page_number_snapshot_check
      CHECK (
        page_number_snapshot > 0
      )
      NOT VALID;
  END IF;
END
$$;

ALTER TABLE courseware_annotations
  VALIDATE CONSTRAINT
    courseware_annotations_page_number_snapshot_check;

-- 已确认courseware_pages存在UNIQUE(id, courseware_id)。
-- 删除页面时仅清空page_id，保留courseware_id和历史快照。
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname =
      'courseware_annotations_page_courseware_fk'
  ) THEN
    ALTER TABLE courseware_annotations
      ADD CONSTRAINT
        courseware_annotations_page_courseware_fk
      FOREIGN KEY (
        page_id,
        courseware_id
      )
      REFERENCES courseware_pages (
        id,
        courseware_id
      )
      ON DELETE SET NULL (
        page_id
      );
  END IF;
END
$$;

CREATE INDEX IF NOT EXISTS
  idx_courseware_annotations_page_id
ON courseware_annotations (
  page_id,
  created_at
)
WHERE page_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS
  idx_courseware_annotations_courseware_snapshot
ON courseware_annotations (
  courseware_id,
  page_number_snapshot,
  created_at
);

COMMENT ON COLUMN
  courseware_annotations.page_id
IS
  '稳定页面ID；页面重排后保持不变，页面删除时置空并保留批注历史';

COMMENT ON COLUMN
  courseware_annotations.page_number_snapshot
IS
  '批注创建时页码快照，仅用于历史展示和页面删除后的定位说明';

COMMIT;
