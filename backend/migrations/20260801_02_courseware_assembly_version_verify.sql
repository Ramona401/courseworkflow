DO $$
BEGIN
    IF to_regclass('public.courseware_assembly_runs') IS NULL THEN
        RAISE EXCEPTION '缺少表 courseware_assembly_runs';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'coursewares'
          AND column_name = 'assembly_version'
    ) THEN
        RAISE EXCEPTION 'coursewares缺少assembly_version';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'coursewares'
          AND column_name = 'active_assembly_run_id'
    ) THEN
        RAISE EXCEPTION 'coursewares缺少active_assembly_run_id';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_pages'
          AND column_name = 'layout_status'
    ) THEN
        RAISE EXCEPTION 'courseware_pages缺少layout_status';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_pages'
          AND column_name = 'layout_audit_json'
    ) THEN
        RAISE EXCEPTION 'courseware_pages缺少layout_audit_json';
    END IF;

    IF to_regclass('public.uq_courseware_assembly_runs_active') IS NULL THEN
        RAISE EXCEPTION '缺少活动装配唯一索引';
    END IF;
END
$$;

SELECT
    'coursewares' AS object_name,
    COUNT(*) FILTER (WHERE column_name = 'assembly_version') AS assembly_version,
    COUNT(*) FILTER (WHERE column_name = 'assembly_status') AS assembly_status,
    COUNT(*) FILTER (WHERE column_name = 'active_assembly_run_id') AS active_run
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'coursewares'

UNION ALL

SELECT
    'courseware_pages' AS object_name,
    COUNT(*) FILTER (WHERE column_name = 'assembly_version') AS assembly_version,
    COUNT(*) FILTER (WHERE column_name = 'layout_status') AS layout_status,
    COUNT(*) FILTER (WHERE column_name = 'layout_audit_json') AS layout_audit
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'courseware_pages';

SELECT
    assembly_status,
    COUNT(*) AS courseware_count
FROM coursewares
GROUP BY assembly_status
ORDER BY assembly_status;

SELECT
    status,
    COUNT(*) AS run_count
FROM courseware_assembly_runs
GROUP BY status
ORDER BY status;
