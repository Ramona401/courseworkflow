\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260731_02_lesson_plan_knowledge_lineage_verify.sql
-- 教案知识脉络快照、离开analyze硬闸与失效保护验证
-- ============================================================================

SELECT
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'lesson_plan_knowledge_lineages'
ORDER BY ordinal_position;

SELECT
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid =
    'public.lesson_plan_knowledge_lineages'::regclass
ORDER BY conname;

SELECT
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'lesson_plan_knowledge_lineages'
ORDER BY indexname;

SELECT
    event_object_table,
    trigger_name,
    action_timing,
    event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND trigger_name IN (
      'trg_lesson_plans_require_knowledge_lineage',
      'trg_lesson_plans_knowledge_lineage_stale',
      'trg_course_outlines_knowledge_lineage_stale',
      'trg_stage_outputs_knowledge_lineage_stale'
  )
ORDER BY event_object_table, trigger_name;

SELECT
    status,
    COUNT(*) AS row_count
FROM public.lesson_plan_knowledge_lineages
GROUP BY status
ORDER BY status;

SELECT
    COUNT(*) AS invalid_active_lineage_rows
FROM public.lesson_plan_knowledge_lineages lineage
LEFT JOIN public.lesson_plans lp
  ON lp.id = lineage.lesson_plan_id
LEFT JOIN public.course_outlines co
  ON co.id = lineage.course_outline_id
WHERE lineage.status = 'active'
  AND (
      lp.id IS NULL
      OR lp.deleted_at IS NOT NULL
      OR lp.course_outline_id
         IS DISTINCT FROM lineage.course_outline_id
      OR co.id IS NULL
      OR co.status <> 'active'
      OR lineage.confirmed_stage_code <> 'analyze'
      OR lineage.anchor_snapshot = '{}'::jsonb
      OR lineage.lineage_snapshot = '{}'::jsonb
      OR btrim(lineage.context_text) = ''
      OR lineage.anchor_hash = ''
      OR lineage.outline_hash = ''
      OR lineage.confirmed_stage_output_updated_at IS NULL
      OR lineage.generated_at IS NULL
  );

-- 存量提示：
-- 迁移不会自动生成旧教案知识脉络，也不会静默把旧教案改回analyze。
-- 已经位于后续阶段但缺少active快照的精确大纲教案会在运行时被要求回到教学分析确认。
SELECT
    COUNT(*) AS legacy_exact_outline_plans_without_active_lineage
FROM public.lesson_plans lp
WHERE lp.deleted_at IS NULL
  AND lp.course_outline_id IS NOT NULL
  AND btrim(COALESCE(lp.current_stage, '')) <> 'analyze'
  AND NOT EXISTS (
      SELECT 1
      FROM public.lesson_plan_knowledge_lineages lineage
      WHERE lineage.lesson_plan_id = lp.id
        AND lineage.course_outline_id = lp.course_outline_id
        AND lineage.status = 'active'
  );

DO $$
DECLARE
    invalid_active_count BIGINT;
    required_trigger_count INTEGER;
BEGIN
    IF to_regclass(
        'public.lesson_plan_knowledge_lineages'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：知识脉络快照表不存在';
    END IF;

    SELECT COUNT(DISTINCT trigger_name)
    INTO required_trigger_count
    FROM information_schema.triggers
    WHERE event_object_schema = 'public'
      AND trigger_name IN (
          'trg_lesson_plans_require_knowledge_lineage',
          'trg_lesson_plans_knowledge_lineage_stale',
          'trg_course_outlines_knowledge_lineage_stale',
          'trg_stage_outputs_knowledge_lineage_stale'
      );

    IF required_trigger_count <> 4 THEN
        RAISE EXCEPTION
            '验证失败：知识脉络必需触发器应为4个，实际为%',
            required_trigger_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_active_count
    FROM public.lesson_plan_knowledge_lineages lineage
    LEFT JOIN public.lesson_plans lp
      ON lp.id = lineage.lesson_plan_id
    LEFT JOIN public.course_outlines co
      ON co.id = lineage.course_outline_id
    WHERE lineage.status = 'active'
      AND (
          lp.id IS NULL
          OR lp.deleted_at IS NOT NULL
          OR lp.course_outline_id
             IS DISTINCT FROM lineage.course_outline_id
          OR co.id IS NULL
          OR co.status <> 'active'
          OR lineage.confirmed_stage_code <> 'analyze'
          OR lineage.anchor_snapshot = '{}'::jsonb
          OR lineage.lineage_snapshot = '{}'::jsonb
          OR btrim(lineage.context_text) = ''
          OR lineage.anchor_hash = ''
          OR lineage.outline_hash = ''
          OR lineage.confirmed_stage_output_updated_at IS NULL
          OR lineage.generated_at IS NULL
      );

    IF invalid_active_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效active知识脉络记录',
            invalid_active_count;
    END IF;

    RAISE NOTICE
        '教案知识脉络快照、离开analyze硬闸与失效保护验证通过';
END
$$;
