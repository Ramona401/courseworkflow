\set ON_ERROR_STOP on

-- ============================================================================
-- 20260731_02_lesson_plan_knowledge_lineage_rollback.sql
-- 回滚教案知识脉络快照存储
--
-- 本回滚只移除知识脉络表、索引、触发器和辅助函数，
-- 不修改lesson_plans、course_outlines或workshop_stage_outputs业务数据。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS
    trg_stage_outputs_knowledge_lineage_stale
ON public.workshop_stage_outputs;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_knowledge_lineage_stale_from_stage_output();

DROP TRIGGER IF EXISTS
    trg_course_outlines_knowledge_lineage_stale
ON public.course_outlines;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_knowledge_lineage_stale_from_outline();

DROP TRIGGER IF EXISTS
    trg_lesson_plans_knowledge_lineage_stale
ON public.lesson_plans;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_knowledge_lineage_stale_from_plan();

DROP TRIGGER IF EXISTS
    trg_lesson_plans_require_knowledge_lineage
ON public.lesson_plans;

DROP FUNCTION IF EXISTS
    public.enforce_lesson_plan_knowledge_lineage_before_leaving_analyze();

DROP TABLE IF EXISTS
    public.lesson_plan_knowledge_lineages;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '教案知识脉络快照、离开analyze硬闸与失效保护已回滚';
END
$$;
