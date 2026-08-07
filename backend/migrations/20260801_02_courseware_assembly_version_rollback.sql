BEGIN;

ALTER TABLE coursewares
    DROP CONSTRAINT IF EXISTS fk_coursewares_active_assembly_run;

DROP INDEX IF EXISTS idx_coursewares_active_assembly_run;

ALTER TABLE coursewares
    DROP COLUMN IF EXISTS active_assembly_run_id;

DROP INDEX IF EXISTS idx_courseware_pages_assembly_version;
DROP INDEX IF EXISTS idx_courseware_pages_layout_status;

ALTER TABLE courseware_pages
    DROP CONSTRAINT IF EXISTS chk_courseware_pages_layout_hash,
    DROP CONSTRAINT IF EXISTS chk_courseware_pages_layout_audit_object,
    DROP CONSTRAINT IF EXISTS chk_courseware_pages_layout_status,
    DROP CONSTRAINT IF EXISTS chk_courseware_pages_assembly_version_nonnegative,
    DROP COLUMN IF EXISTS layout_checked_at,
    DROP COLUMN IF EXISTS layout_html_hash,
    DROP COLUMN IF EXISTS layout_audit_json,
    DROP COLUMN IF EXISTS layout_status,
    DROP COLUMN IF EXISTS assembly_version;

DROP INDEX IF EXISTS uq_courseware_assembly_runs_active;
DROP INDEX IF EXISTS idx_courseware_assembly_runs_courseware_started;
DROP INDEX IF EXISTS idx_courseware_assembly_runs_status;

DROP TABLE IF EXISTS courseware_assembly_runs;

ALTER TABLE coursewares
    DROP CONSTRAINT IF EXISTS chk_coursewares_assembly_status,
    DROP CONSTRAINT IF EXISTS chk_coursewares_assembly_version_nonnegative,
    DROP COLUMN IF EXISTS assembly_skip_video,
    DROP COLUMN IF EXISTS assembly_started_by,
    DROP COLUMN IF EXISTS assembly_finished_at,
    DROP COLUMN IF EXISTS assembly_started_at,
    DROP COLUMN IF EXISTS assembly_status,
    DROP COLUMN IF EXISTS assembly_version;

COMMIT;
