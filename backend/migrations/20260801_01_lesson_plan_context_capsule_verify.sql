\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260801_01_lesson_plan_context_capsule_verify.sql
-- 备课核心共识胶囊、版本、证据路由和来源失效保护验证
-- ============================================================================

SELECT
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN (
      'lesson_plan_context_capsules',
      'lesson_plan_context_capsule_versions',
      'lesson_plan_context_capsule_evidence'
  )
ORDER BY table_name, ordinal_position;

SELECT
    conrelid::regclass::text AS table_name,
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid IN (
    'public.lesson_plan_context_capsules'::regclass,
    'public.lesson_plan_context_capsule_versions'::regclass,
    'public.lesson_plan_context_capsule_evidence'::regclass
)
ORDER BY table_name, constraint_name;

SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN (
      'lesson_plan_context_capsules',
      'lesson_plan_context_capsule_versions',
      'lesson_plan_context_capsule_evidence'
  )
ORDER BY tablename, indexname;

SELECT
    event_object_table,
    trigger_name,
    action_timing,
    event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND trigger_name IN (
      'trg_lesson_plan_context_capsule_validate',
      'trg_lesson_plan_context_capsule_snapshot',
      'trg_lesson_plans_context_capsule_stale',
      'trg_course_outlines_context_capsule_stale',
      'trg_textbook_pages_context_capsule_stale'
  )
ORDER BY event_object_table, trigger_name, event_manipulation;

SELECT
    status,
    COUNT(*) AS row_count
FROM public.lesson_plan_context_capsules
GROUP BY status
ORDER BY status;

SELECT
    COUNT(*) AS invalid_active_capsules
FROM public.lesson_plan_context_capsules capsule
LEFT JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = capsule.lesson_plan_id
WHERE capsule.status = 'active'
  AND (
      lesson_plan.id IS NULL
      OR lesson_plan.deleted_at IS NOT NULL
      OR capsule.capsule_json = '{}'::jsonb
      OR capsule.source_manifest = '{}'::jsonb
      OR btrim(capsule.context_text) = ''
      OR capsule.source_hash = ''
      OR capsule.generated_at IS NULL
  );

SELECT
    COUNT(*) AS current_capsules_without_matching_version
FROM public.lesson_plan_context_capsules capsule
LEFT JOIN public.lesson_plan_context_capsule_versions version_snapshot
  ON version_snapshot.lesson_plan_id = capsule.lesson_plan_id
 AND version_snapshot.version = capsule.version
WHERE version_snapshot.id IS NULL;

SELECT
    COUNT(*) AS orphan_capsule_evidence_routes
FROM public.lesson_plan_context_capsule_evidence evidence
LEFT JOIN public.lesson_plan_context_capsule_versions version_snapshot
  ON version_snapshot.lesson_plan_id = evidence.lesson_plan_id
 AND version_snapshot.version = evidence.capsule_version
WHERE version_snapshot.id IS NULL;

SELECT
    COUNT(*) AS invalid_ai_inferred_evidence_rows
FROM public.lesson_plan_context_capsule_evidence
WHERE authority = 'ai_inferred'
  AND btrim(item_key) = '';

DO $$
DECLARE
    required_trigger_count INTEGER;
    invalid_active_count BIGINT;
    missing_version_count BIGINT;
    orphan_evidence_count BIGINT;
BEGIN
    IF to_regclass(
        'public.lesson_plan_context_capsules'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：当前核心共识胶囊表不存在';
    END IF;

    IF to_regclass(
        'public.lesson_plan_context_capsule_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：胶囊不可变版本表不存在';
    END IF;

    IF to_regclass(
        'public.lesson_plan_context_capsule_evidence'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：胶囊原文证据路由表不存在';
    END IF;

    SELECT COUNT(DISTINCT trigger_name)
    INTO required_trigger_count
    FROM information_schema.triggers
    WHERE event_object_schema = 'public'
      AND trigger_name IN (
          'trg_lesson_plan_context_capsule_validate',
          'trg_lesson_plan_context_capsule_snapshot',
          'trg_lesson_plans_context_capsule_stale',
          'trg_course_outlines_context_capsule_stale',
          'trg_textbook_pages_context_capsule_stale'
      );

    IF required_trigger_count <> 5 THEN
        RAISE EXCEPTION
            '验证失败：胶囊必需触发器应为5个，实际为%',
            required_trigger_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_active_count
    FROM public.lesson_plan_context_capsules capsule
    LEFT JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = capsule.lesson_plan_id
    WHERE capsule.status = 'active'
      AND (
          lesson_plan.id IS NULL
          OR lesson_plan.deleted_at IS NOT NULL
          OR capsule.capsule_json = '{}'::jsonb
          OR capsule.source_manifest = '{}'::jsonb
          OR btrim(capsule.context_text) = ''
          OR capsule.source_hash = ''
          OR capsule.generated_at IS NULL
      );

    IF invalid_active_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效active核心共识胶囊',
            invalid_active_count;
    END IF;

    SELECT COUNT(*)
    INTO missing_version_count
    FROM public.lesson_plan_context_capsules capsule
    LEFT JOIN public.lesson_plan_context_capsule_versions version_snapshot
      ON version_snapshot.lesson_plan_id = capsule.lesson_plan_id
     AND version_snapshot.version = capsule.version
    WHERE version_snapshot.id IS NULL;

    IF missing_version_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条当前胶囊没有对应不可变版本快照',
            missing_version_count;
    END IF;

    SELECT COUNT(*)
    INTO orphan_evidence_count
    FROM public.lesson_plan_context_capsule_evidence evidence
    LEFT JOIN public.lesson_plan_context_capsule_versions version_snapshot
      ON version_snapshot.lesson_plan_id = evidence.lesson_plan_id
     AND version_snapshot.version = evidence.capsule_version
    WHERE version_snapshot.id IS NULL;

    IF orphan_evidence_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条没有对应胶囊版本的证据路由',
            orphan_evidence_count;
    END IF;

    RAISE NOTICE
        '备课核心共识胶囊、版本快照、原文证据路由和来源失效保护验证通过';
END
$$;
