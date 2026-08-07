\set ON_ERROR_STOP on

-- ============================================================================
-- 20260801_01_lesson_plan_context_capsule_rollback.sql
-- 回滚备课核心共识胶囊、版本快照和原文证据路由
--
-- 本回滚会删除本迁移创建的三张表及其数据。
-- 不修改lesson_plans、course_outlines、textbook_pages、
-- lesson_plan_knowledge_lineages或workshop_stage_outputs中的原业务数据。
-- 正式执行回滚前必须再次备份数据库。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS trg_textbook_pages_context_capsule_stale
ON public.textbook_pages;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_context_capsule_stale_from_textbook();

DROP TRIGGER IF EXISTS trg_course_outlines_context_capsule_stale
ON public.course_outlines;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_context_capsule_stale_from_outline();

DROP TRIGGER IF EXISTS trg_lesson_plans_context_capsule_stale
ON public.lesson_plans;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_context_capsule_stale_from_plan();

DROP TRIGGER IF EXISTS trg_lesson_plan_context_capsule_snapshot
ON public.lesson_plan_context_capsules;

DROP FUNCTION IF EXISTS
    public.snapshot_lesson_plan_context_capsule_version();

DROP TRIGGER IF EXISTS trg_lesson_plan_context_capsule_validate
ON public.lesson_plan_context_capsules;

DROP FUNCTION IF EXISTS
    public.validate_lesson_plan_context_capsule_write();

DROP TABLE IF EXISTS
    public.lesson_plan_context_capsule_evidence;

DROP TABLE IF EXISTS
    public.lesson_plan_context_capsule_versions;

DROP TABLE IF EXISTS
    public.lesson_plan_context_capsules;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '备课核心共识胶囊、版本快照、证据路由和来源失效保护已回滚';
END
$$;
