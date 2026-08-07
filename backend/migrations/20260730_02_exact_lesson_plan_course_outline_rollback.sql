\set ON_ERROR_STOP on

-- ============================================================================
-- 20260730_02_exact_lesson_plan_course_outline_rollback.sql
-- 精确课程大纲迁移回滚
--
-- 警告：
--   本回滚会移除精确大纲ID及快照字段。
--   five_four大纲的册次会恢复“(五四制)”后缀，尽量还原迁移前格式。
-- ============================================================================

BEGIN;

LOCK TABLE public.lesson_plans
    IN SHARE ROW EXCLUSIVE MODE;

LOCK TABLE public.course_outlines
    IN SHARE ROW EXCLUSIVE MODE;

DROP TRIGGER IF EXISTS
    trg_lesson_plans_zz_course_outline_snapshot_insert
ON public.lesson_plans;

DROP TRIGGER IF EXISTS
    trg_lesson_plans_zz_course_outline_snapshot_update
ON public.lesson_plans;

DROP FUNCTION IF EXISTS
    public.apply_lesson_plan_course_outline_snapshot();

DROP TRIGGER IF EXISTS
    trg_course_outlines_00_school_system_insert
ON public.course_outlines;

DROP TRIGGER IF EXISTS
    trg_course_outlines_00_school_system_update
ON public.course_outlines;

DROP FUNCTION IF EXISTS
    public.normalize_course_outline_school_system();

DROP INDEX IF EXISTS
    public.idx_lesson_plans_course_outline_id;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_exact_snapshot_check;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_volume_trim_check;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_school_system_trim_check;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_school_system_check;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_id_fkey;

ALTER TABLE public.lesson_plans
    DROP COLUMN IF EXISTS course_outline_id;

ALTER TABLE public.lesson_plans
    DROP COLUMN IF EXISTS course_outline_volume;

ALTER TABLE public.lesson_plans
    DROP COLUMN IF EXISTS school_system;

-- 先停用新写入归一化触发器，再恢复五四制册次后缀。
UPDATE public.course_outlines
SET volume = btrim(volume) || '(五四制)'
WHERE school_system = 'five_four'
  AND volume NOT ILIKE '%五四制%';

DROP INDEX IF EXISTS
    public.uq_course_outlines_active;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM public.course_outlines
        WHERE status = 'active'
        GROUP BY
            scope,
            scope_target_id,
            subject,
            grade,
            volume,
            publisher
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            '回滚后旧课程大纲唯一键存在重复，回滚已中止';
    END IF;
END
$$;

CREATE UNIQUE INDEX
    uq_course_outlines_active
ON public.course_outlines (
    scope,
    scope_target_id,
    subject,
    grade,
    volume,
    publisher
)
WHERE status = 'active';

DROP INDEX IF EXISTS
    public.idx_course_outlines_lookup;

CREATE INDEX
    idx_course_outlines_lookup
ON public.course_outlines (
    subject,
    publisher,
    grade,
    volume,
    status
);

ALTER TABLE public.course_outlines
    DROP CONSTRAINT IF EXISTS
        course_outlines_school_system_trim_check;

ALTER TABLE public.course_outlines
    DROP CONSTRAINT IF EXISTS
        course_outlines_school_system_check;

ALTER TABLE public.course_outlines
    DROP COLUMN IF EXISTS school_system;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '精确课程大纲迁移已回滚';
END
$$;
