\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260730_02_exact_lesson_plan_course_outline_verify.sql
-- 精确课程大纲迁移只读验证
-- ============================================================================

SELECT
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND (
      (
          table_name = 'course_outlines'
          AND column_name = 'school_system'
      )
      OR (
          table_name = 'lesson_plans'
          AND column_name IN (
              'school_system',
              'course_outline_volume',
              'course_outline_id'
          )
      )
  )
ORDER BY
    table_name,
    ordinal_position;

SELECT
    conrelid::regclass AS table_name,
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid IN (
    'public.course_outlines'::regclass,
    'public.lesson_plans'::regclass
)
  AND (
      conname ILIKE '%school_system%'
      OR conname ILIKE '%course_outline_exact%'
      OR conname =
          'lesson_plans_course_outline_id_fkey'
  )
ORDER BY
    table_name,
    constraint_name;

SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND indexname IN (
      'uq_course_outlines_active',
      'idx_course_outlines_lookup',
      'idx_lesson_plans_course_outline_id'
  )
ORDER BY
    indexname;

SELECT
    event_object_table,
    trigger_name,
    action_timing,
    event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND trigger_name IN (
      'trg_course_outlines_00_school_system_insert',
      'trg_course_outlines_00_school_system_update',
      'trg_lesson_plans_zz_course_outline_snapshot_insert',
      'trg_lesson_plans_zz_course_outline_snapshot_update'
  )
ORDER BY
    event_object_table,
    trigger_name,
    event_manipulation;

SELECT
    school_system,
    COUNT(*) AS outline_count
FROM public.course_outlines
GROUP BY school_system
ORDER BY school_system;

SELECT
    COUNT(*) AS volumes_still_containing_five_four
FROM public.course_outlines
WHERE volume ILIKE '%五四制%';

SELECT
    COUNT(*) AS invalid_outline_school_system_rows
FROM public.course_outlines
WHERE school_system NOT IN (
    'standard',
    'five_four'
)
   OR btrim(school_system) <> school_system
   OR btrim(volume) = '';

SELECT
    COUNT(*) AS duplicate_active_exact_outline_keys
FROM (
    SELECT
        scope,
        scope_target_id,
        subject,
        grade,
        volume,
        publisher,
        school_system
    FROM public.course_outlines
    WHERE status = 'active'
    GROUP BY
        scope,
        scope_target_id,
        subject,
        grade,
        volume,
        publisher,
        school_system
    HAVING COUNT(*) > 1
) duplicate_keys;

SELECT
    COUNT(*) FILTER (
        WHERE course_outline_id IS NOT NULL
    ) AS exact_outline_plans,
    COUNT(*) FILTER (
        WHERE course_outline_id IS NULL
          AND course_outline_publisher IS NOT NULL
    ) AS legacy_publisher_only_plans,
    COUNT(*) FILTER (
        WHERE course_outline_id IS NULL
          AND course_outline_publisher IS NULL
    ) AS plans_without_outline
FROM public.lesson_plans
WHERE deleted_at IS NULL;

SELECT
    COUNT(*) AS invalid_exact_snapshot_rows
FROM public.lesson_plans lp
LEFT JOIN public.course_outlines co
  ON co.id = lp.course_outline_id
WHERE lp.course_outline_id IS NOT NULL
  AND (
      co.id IS NULL
      OR co.status <> 'active'
      OR btrim(co.subject)
          IS DISTINCT FROM btrim(lp.subject)
      OR btrim(co.grade)
          IS DISTINCT FROM btrim(lp.grade)
      OR btrim(co.publisher)
          IS DISTINCT FROM
             btrim(lp.course_outline_publisher)
      OR btrim(co.volume)
          IS DISTINCT FROM
             btrim(lp.course_outline_volume)
      OR btrim(co.school_system)
          IS DISTINCT FROM
             btrim(lp.school_system)
  );

DO $$
BEGIN
    IF (
        SELECT COUNT(*)
        FROM public.course_outlines
        WHERE volume ILIKE '%五四制%'
    ) <> 0 THEN
        RAISE EXCEPTION
            '验证失败：仍有册次包含五四制后缀';
    END IF;

    IF (
        SELECT COUNT(*)
        FROM public.course_outlines
        WHERE school_system NOT IN (
            'standard',
            'five_four'
        )
           OR btrim(school_system) <> school_system
           OR btrim(volume) = ''
    ) <> 0 THEN
        RAISE EXCEPTION
            '验证失败：课程大纲学制或册次存在非法值';
    END IF;

    IF (
        SELECT COUNT(*)
        FROM (
            SELECT
                scope,
                scope_target_id,
                subject,
                grade,
                volume,
                publisher,
                school_system
            FROM public.course_outlines
            WHERE status = 'active'
            GROUP BY
                scope,
                scope_target_id,
                subject,
                grade,
                volume,
                publisher,
                school_system
            HAVING COUNT(*) > 1
        ) duplicate_keys
    ) <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在重复的精确课程大纲键';
    END IF;

    IF (
        SELECT COUNT(*)
        FROM public.lesson_plans lp
        LEFT JOIN public.course_outlines co
          ON co.id = lp.course_outline_id
        WHERE lp.course_outline_id IS NOT NULL
          AND (
              co.id IS NULL
              OR co.status <> 'active'
              OR btrim(co.subject)
                  IS DISTINCT FROM btrim(lp.subject)
              OR btrim(co.grade)
                  IS DISTINCT FROM btrim(lp.grade)
              OR btrim(co.publisher)
                  IS DISTINCT FROM
                     btrim(lp.course_outline_publisher)
              OR btrim(co.volume)
                  IS DISTINCT FROM
                     btrim(lp.course_outline_volume)
              OR btrim(co.school_system)
                  IS DISTINCT FROM
                     btrim(lp.school_system)
          )
    ) <> 0 THEN
        RAISE EXCEPTION
            '验证失败：精确课程大纲快照与正式记录不一致';
    END IF;

    RAISE NOTICE
        '精确课程大纲迁移只读验证通过';
END
$$;
