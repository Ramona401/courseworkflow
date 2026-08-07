\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260731_01_lesson_plan_course_outline_manual_stage_selection_verify.sql
-- 课程大纲自动精确匹配与手动学段绑定迁移验证
--
-- 使用前提：
--   必须在正式迁移成功执行后运行。
--
-- 验证范围：
--   1. 两个辅助函数已创建；
--   2. 具体年级、学段、初高中内部年级及十一/十二年级解析正确；
--   3. 年级或学段相交判断正确；
--   4. 职教等非K12层级不会被误转为K12数字集合；
--   5. 当前已有教案的课程大纲快照仍满足新规则。
-- ============================================================================

SELECT
    proname AS function_name,
    pg_get_function_identity_arguments(oid) AS function_arguments,
    pg_get_function_result(oid) AS function_result
FROM pg_proc
WHERE pronamespace = 'public'::regnamespace
  AND proname IN (
      'lesson_plan_course_outline_grade_span',
      'lesson_plan_course_outline_grade_matches',
      'apply_lesson_plan_course_outline_snapshot'
  )
ORDER BY proname;

SELECT
    sample.input_grade,
    public.lesson_plan_course_outline_grade_span(sample.input_grade) AS parsed_span
FROM (
    VALUES
        ('小学三年级'),
        ('三年级'),
        ('小学中段'),
        ('初中一年级'),
        ('初一'),
        ('高中一年级'),
        ('高一'),
        ('十年级'),
        ('10年级'),
        ('十一年级'),
        ('11年级'),
        ('十二年级'),
        ('12年级'),
        ('中职一年级')
) AS sample(input_grade);

SELECT
    sample.outline_grade,
    sample.lesson_grade,
    public.lesson_plan_course_outline_grade_matches(
        sample.outline_grade,
        sample.lesson_grade
    ) AS matches
FROM (
    VALUES
        ('小学中段', '三年级'),
        ('小学中段', '四年级'),
        ('小学中段', '五年级'),
        ('小学高段', '三年级'),
        ('小学一至六年级', '小学三年级'),
        ('初中', '初中一年级'),
        ('初中', '高中一年级'),
        ('高中', '高一'),
        ('十一年级', '11年级'),
        ('十二年级', '12年级'),
        ('中职一年级', '中职一年级'),
        ('中职一年级', '一年级')
) AS sample(outline_grade, lesson_grade);

SELECT
    COUNT(*) AS invalid_existing_exact_snapshot_rows
FROM public.lesson_plans lp
LEFT JOIN public.course_outlines co
  ON co.id = lp.course_outline_id
WHERE lp.deleted_at IS NULL
  AND lp.course_outline_id IS NOT NULL
  AND (
      co.id IS NULL
      OR co.status <> 'active'
      OR btrim(co.subject) IS DISTINCT FROM btrim(lp.subject)
      OR NOT public.lesson_plan_course_outline_grade_matches(
          co.grade,
          lp.grade
      )
      OR btrim(co.publisher) IS DISTINCT FROM
         btrim(lp.course_outline_publisher)
      OR btrim(co.volume) IS DISTINCT FROM
         btrim(lp.course_outline_volume)
      OR btrim(co.school_system) IS DISTINCT FROM
         btrim(lp.school_system)
  );

DO $$
DECLARE
    invalid_snapshot_count BIGINT;
BEGIN
    IF public.lesson_plan_course_outline_grade_span('小学三年级')
       IS DISTINCT FROM ARRAY[3] THEN
        RAISE EXCEPTION
            '验证失败：小学三年级没有被精确解析为三年级';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('初中一年级')
       IS DISTINCT FROM ARRAY[7] THEN
        RAISE EXCEPTION
            '验证失败：初中一年级没有被解析为七年级';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('高中一年级')
       IS DISTINCT FROM ARRAY[10] THEN
        RAISE EXCEPTION
            '验证失败：高中一年级没有被解析为十年级';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('十一年级')
       IS DISTINCT FROM ARRAY[11] THEN
        RAISE EXCEPTION
            '验证失败：十一年级发生了一年级子串误判';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('11年级')
       IS DISTINCT FROM ARRAY[11] THEN
        RAISE EXCEPTION
            '验证失败：11年级发生1年级子串误判';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('十二年级')
       IS DISTINCT FROM ARRAY[12] THEN
        RAISE EXCEPTION
            '验证失败：十二年级发生二年级子串误判';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('12年级')
       IS DISTINCT FROM ARRAY[12] THEN
        RAISE EXCEPTION
            '验证失败：12年级发生2年级子串误判';
    END IF;

    IF public.lesson_plan_course_outline_grade_span('小学中段')
       IS DISTINCT FROM ARRAY[3, 4] THEN
        RAISE EXCEPTION
            '验证失败：小学中段没有被解析为三至四年级';
    END IF;

    IF cardinality(
        public.lesson_plan_course_outline_grade_span(
            '中职一年级'
        )
    ) <> 0 THEN
        RAISE EXCEPTION
            '验证失败：中职层级被错误转换为K12年级集合';
    END IF;

    IF NOT public.lesson_plan_course_outline_grade_matches(
        '小学中段',
        '三年级'
    ) THEN
        RAISE EXCEPTION
            '验证失败：小学中段应允许绑定三年级教案';
    END IF;

    IF public.lesson_plan_course_outline_grade_matches(
        '小学高段',
        '三年级'
    ) THEN
        RAISE EXCEPTION
            '验证失败：小学高段不应允许绑定三年级教案';
    END IF;

    IF NOT public.lesson_plan_course_outline_grade_matches(
        '中职一年级',
        '中职一年级'
    ) THEN
        RAISE EXCEPTION
            '验证失败：非K12相同层级文本应允许匹配';
    END IF;

    IF public.lesson_plan_course_outline_grade_matches(
        '中职一年级',
        '一年级'
    ) THEN
        RAISE EXCEPTION
            '验证失败：中职一年级不应匹配K12一年级';
    END IF;

    SELECT COUNT(*)
    INTO invalid_snapshot_count
    FROM public.lesson_plans lp
    LEFT JOIN public.course_outlines co
      ON co.id = lp.course_outline_id
    WHERE lp.deleted_at IS NULL
      AND lp.course_outline_id IS NOT NULL
      AND (
          co.id IS NULL
          OR co.status <> 'active'
          OR btrim(co.subject) IS DISTINCT FROM btrim(lp.subject)
          OR NOT public.lesson_plan_course_outline_grade_matches(
              co.grade,
              lp.grade
          )
          OR btrim(co.publisher) IS DISTINCT FROM
             btrim(lp.course_outline_publisher)
          OR btrim(co.volume) IS DISTINCT FROM
             btrim(lp.course_outline_volume)
          OR btrim(co.school_system) IS DISTINCT FROM
             btrim(lp.school_system)
      );

    IF invalid_snapshot_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效课程大纲快照记录',
            invalid_snapshot_count;
    END IF;

    RAISE NOTICE
        '课程大纲手动学段选择迁移验证通过';
END
$$;
