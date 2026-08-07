DO $$
DECLARE
    ownership_mismatch_count BIGINT;
    courseware_state_mismatch_count BIGINT;
    run_finish_mismatch_count BIGINT;
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_courseware_assembly_runs_id_courseware'
          AND conrelid = 'courseware_assembly_runs'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少运行ID与课件ID复合唯一约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_coursewares_active_assembly_run'
          AND conrelid = 'coursewares'::regclass
          AND confrelid = 'courseware_assembly_runs'::regclass
          AND conkey = ARRAY[
              (
                  SELECT attnum
                  FROM pg_attribute
                  WHERE attrelid = 'coursewares'::regclass
                    AND attname = 'active_assembly_run_id'
              ),
              (
                  SELECT attnum
                  FROM pg_attribute
                  WHERE attrelid = 'coursewares'::regclass
                    AND attname = 'id'
              )
          ]::SMALLINT[]
    ) THEN
        RAISE EXCEPTION
            '活动装配运行尚未使用复合归属外键';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_coursewares_assembly_active_run_consistency'
          AND conrelid = 'coursewares'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少课件装配状态与活动运行一致性约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_courseware_assembly_runs_finish_consistency'
          AND conrelid =
              'courseware_assembly_runs'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少装配运行finished_at一致性约束';
    END IF;

    SELECT COUNT(*)
    INTO ownership_mismatch_count
    FROM coursewares AS courseware
    JOIN courseware_assembly_runs AS run
      ON run.id = courseware.active_assembly_run_id
    WHERE run.courseware_id <> courseware.id;

    IF ownership_mismatch_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条跨课件活动运行引用',
            ownership_mismatch_count;
    END IF;

    SELECT COUNT(*)
    INTO courseware_state_mismatch_count
    FROM coursewares
    WHERE (
        assembly_status IN (
            'running',
            'cancel_requested'
        )
        AND active_assembly_run_id IS NULL
    )
    OR (
        assembly_status NOT IN (
            'running',
            'cancel_requested'
        )
        AND active_assembly_run_id IS NOT NULL
    );

    IF courseware_state_mismatch_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条课件装配状态与活动运行不一致数据',
            courseware_state_mismatch_count;
    END IF;

    SELECT COUNT(*)
    INTO run_finish_mismatch_count
    FROM courseware_assembly_runs
    WHERE (
        status IN (
            'running',
            'cancel_requested'
        )
        AND finished_at IS NOT NULL
    )
    OR (
        status IN (
            'completed',
            'cancelled',
            'failed',
            'interrupted'
        )
        AND finished_at IS NULL
    );

    IF run_finish_mismatch_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条装配运行状态与finished_at不一致数据',
            run_finish_mismatch_count;
    END IF;
END
$$;

SELECT
    conrelid::regclass::text AS table_name,
    conname,
    condeferrable,
    condeferred,
    pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE conname IN (
    'uq_courseware_assembly_runs_id_courseware',
    'fk_coursewares_active_assembly_run',
    'chk_coursewares_assembly_active_run_consistency',
    'chk_courseware_assembly_runs_finish_consistency'
)
ORDER BY table_name, conname;

SELECT
    COUNT(*) FILTER (
        WHERE assembly_status IN (
            'running',
            'cancel_requested'
        )
    ) AS active_coursewares,
    COUNT(*) FILTER (
        WHERE active_assembly_run_id IS NOT NULL
    ) AS coursewares_with_active_run
FROM coursewares;

SELECT
    COUNT(*) FILTER (
        WHERE status IN (
            'running',
            'cancel_requested'
        )
    ) AS active_runs,
    COUNT(*) FILTER (
        WHERE status IN (
            'completed',
            'cancelled',
            'failed',
            'interrupted'
        )
    ) AS finished_runs
FROM courseware_assembly_runs;
