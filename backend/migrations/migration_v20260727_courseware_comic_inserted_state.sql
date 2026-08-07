-- migration_v20260727_courseware_comic_inserted_state.sql
--
-- 单格重新生成会临时把已插入项目切换为generating。
-- 当生成完成或失败时，若inserted_page_id仍然存在，
-- 项目必须恢复inserted，而不是丢失为ready或failed。

BEGIN;

CREATE OR REPLACE FUNCTION preserve_courseware_comic_inserted_status()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.inserted_page_id IS NOT NULL
       AND NEW.inserted_page_id IS NOT NULL
       AND OLD.status = 'generating'
       AND NEW.status IN ('ready', 'failed')
    THEN
        NEW.status := 'inserted';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_preserve_courseware_comic_inserted_status
ON courseware_comic_projects;

CREATE TRIGGER trg_preserve_courseware_comic_inserted_status
BEFORE UPDATE OF status
ON courseware_comic_projects
FOR EACH ROW
EXECUTE FUNCTION preserve_courseware_comic_inserted_status();

UPDATE courseware_comic_projects
SET status = 'inserted',
    version = version + 1,
    updated_at = now()
WHERE inserted_page_id IS NOT NULL
  AND status IN ('ready', 'failed');

COMMIT;
