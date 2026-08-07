\set ON_ERROR_STOP on

-- ============================================================================
-- 20260731_01_lesson_plan_course_outline_manual_stage_selection_rollback.sql
-- 回滚课程大纲手动学段选择能力
--
-- 回滚效果：
--   1. 恢复唯一课程大纲ID只能与教案具体年级文本完全相等；
--   2. 删除本迁移增加的年级集合解析及相交判断函数；
--   3. 不删除或修改任何教案和课程大纲数据；
--   4. 如果迁移后已经产生学段绑定教案，恢复精确规则后这些记录运行时会
--      被旧Go规则或后续更新拒绝，因此正式回滚前必须先评估相关数据。
-- ============================================================================

BEGIN;

LOCK TABLE public.lesson_plans
    IN SHARE ROW EXCLUSIVE MODE;

CREATE OR REPLACE FUNCTION
    public.apply_lesson_plan_course_outline_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    outline_scope VARCHAR;
    outline_scope_target_id UUID;
    outline_subject VARCHAR;
    outline_grade VARCHAR;
    outline_volume VARCHAR;
    outline_publisher VARCHAR;
    outline_school_system VARCHAR;
    outline_status VARCHAR;
    outline_domain VARCHAR;
BEGIN
    IF NEW.course_outline_id IS NULL THEN
        NEW.course_outline_volume := NULL;
        NEW.school_system := NULL;

        IF TG_OP = 'UPDATE'
           AND OLD.course_outline_id IS NOT NULL
           AND NEW.course_outline_id IS NULL THEN
            NEW.course_outline_publisher := NULL;
        END IF;

        RETURN NEW;
    END IF;

    SELECT
        scope,
        scope_target_id,
        subject,
        grade,
        volume,
        publisher,
        school_system,
        status
    INTO
        outline_scope,
        outline_scope_target_id,
        outline_subject,
        outline_grade,
        outline_volume,
        outline_publisher,
        outline_school_system,
        outline_status
    FROM public.course_outlines
    WHERE id = NEW.course_outline_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '精确课程大纲不存在'
            USING
                ERRCODE = '23503',
                CONSTRAINT =
                    'lesson_plans_course_outline_id_fkey';
    END IF;

    IF outline_status <> 'active' THEN
        RAISE EXCEPTION
            '精确课程大纲不是生效状态'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'lesson_plans_course_outline_active_required';
    END IF;

    CASE outline_scope
        WHEN 'system' THEN
            IF outline_scope_target_id IS DISTINCT FROM
                '00000000-0000-0000-0000-000000000000'::uuid THEN
                RAISE EXCEPTION
                    '全局课程大纲归属ID非法'
                    USING
                        ERRCODE = '23514',
                        CONSTRAINT =
                            'lesson_plans_course_outline_scope_domain';
            END IF;

            outline_domain := 'k12';

        WHEN 'school' THEN
            SELECT
                lower(
                    btrim(
                        education_domain
                    )
                )
            INTO outline_domain
            FROM public.organizations
            WHERE id = outline_scope_target_id
              AND type = 'school'
              AND status = 'active';

        WHEN 'group' THEN
            SELECT
                lower(
                    btrim(
                        org.education_domain
                    )
                )
            INTO outline_domain
            FROM public.teaching_groups tg
            JOIN public.organizations org
              ON org.id = tg.school_id
             AND org.type = 'school'
             AND org.status = 'active'
            WHERE tg.id = outline_scope_target_id
              AND tg.status = 'active';

        ELSE
            RAISE EXCEPTION
                '课程大纲归属类型非法'
                USING
                    ERRCODE = '23514',
                    CONSTRAINT =
                        'lesson_plans_course_outline_scope_domain';
    END CASE;

    IF outline_domain IS NULL
       OR outline_domain NOT IN (
           'k12',
           'vocational',
           'adult'
       )
       OR outline_domain IS DISTINCT FROM
          lower(
              btrim(
                  NEW.education_domain
              )
          ) THEN
        RAISE EXCEPTION
            '精确课程大纲教育域与教案不一致'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'lesson_plans_course_outline_domain_match';
    END IF;

    IF btrim(outline_subject)
       IS DISTINCT FROM btrim(NEW.subject) THEN
        RAISE EXCEPTION
            '精确课程大纲学科与教案不一致'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'lesson_plans_course_outline_subject_match';
    END IF;

    IF btrim(outline_grade)
       IS DISTINCT FROM btrim(NEW.grade) THEN
        RAISE EXCEPTION
            '精确课程大纲年级与教案不一致'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'lesson_plans_course_outline_grade_match';
    END IF;

    NEW.course_outline_publisher :=
        btrim(outline_publisher);

    NEW.course_outline_volume :=
        btrim(outline_volume);

    NEW.school_system :=
        btrim(outline_school_system);

    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION
    public.apply_lesson_plan_course_outline_snapshot()
IS
    '精确课程大纲挂载时校验active、归属教育域、学科和具体年级完全相等，并固化快照';

DROP FUNCTION IF EXISTS
    public.lesson_plan_course_outline_grade_matches(
        TEXT,
        TEXT
    );

DROP FUNCTION IF EXISTS
    public.lesson_plan_course_outline_grade_span(
        TEXT
    );

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课程大纲手动学段选择能力已回滚为具体年级完全相等规则';
END
$$;
