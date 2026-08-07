\set ON_ERROR_STOP on

-- ============================================================================
-- 20260731_01_lesson_plan_course_outline_manual_stage_selection.sql
-- 教案课程大纲：自动精确匹配失败后允许手动选择学段课程大纲
--
-- 业务规则：
--   1. 自动匹配只认同学科、同具体年级文本完全相等的唯一候选；
--   2. 自动候选为零条或多条时，由教师主动进入手动选择；
--   3. 手动选择允许课程大纲年级与教案当前年级或学段存在交集；
--   4. 教育域、用户可见范围、active状态和学科一致性仍为硬约束；
--   5. 具体年级优先于宽泛学段：“小学三年级”只解析为三年级；
--   6. 多位年级优先于其内部子串：“十一年级”不得解析为一年级，
--      “十二年级”不得解析为二年级；
--   7. “初中一年级”和“高中一年级”分别解析为七年级和十年级；
--   8. 职教、成教等非K12层级不进行K12数字推断，继续精确文本匹配；
--   9. 本迁移不自动绑定或修改任何历史教案数据。
-- ============================================================================

BEGIN;

LOCK TABLE public.lesson_plans IN SHARE ROW EXCLUSIVE MODE;

-- ============================================================================
-- 一、把K12具体年级或学段文本解析成年级编号集合
-- ============================================================================

CREATE OR REPLACE FUNCTION public.lesson_plan_course_outline_grade_span(raw_grade TEXT)
RETURNS INTEGER[]
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    normalized_grade TEXT;
    empty_span INTEGER[] := ARRAY[]::INTEGER[];
BEGIN
    normalized_grade := regexp_replace(
        lower(btrim(COALESCE(raw_grade, ''))),
        E'\\s+',
        '',
        'g'
    );

    IF normalized_grade = '' THEN
        RETURN empty_span;
    END IF;

    -- 职教、成教和培训层级不能被解释为K12年级。
    -- 返回空集合后，匹配函数会退回规范化文本精确匹配。
    IF normalized_grade LIKE '%中职%'
       OR normalized_grade LIKE '%高职%'
       OR normalized_grade LIKE '%职教%'
       OR normalized_grade LIKE '%职业教育%'
       OR normalized_grade LIKE '%职业高中%'
       OR normalized_grade LIKE '%成人教育%'
       OR normalized_grade LIKE '%培训%' THEN
        RETURN empty_span;
    END IF;

    -- 全学段语义。
    IF normalized_grade LIKE '%全册%'
       OR normalized_grade LIKE '%不限%'
       OR normalized_grade LIKE '%通用%'
       OR normalized_grade LIKE '%全学段%' THEN
        RETURN ARRAY[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];
    END IF;

    -- 明确范围必须先于具体年级判断。
    IF normalized_grade LIKE '%七年级到十二年级%'
       OR normalized_grade LIKE '%七年级至十二年级%'
       OR normalized_grade LIKE '%七到十二%'
       OR normalized_grade LIKE '%七至十二%'
       OR normalized_grade LIKE '%7年级到12年级%'
       OR normalized_grade LIKE '%7年级至12年级%'
       OR normalized_grade LIKE '%7到12%'
       OR normalized_grade LIKE '%7至12%'
       OR normalized_grade LIKE '%7-12%'
       OR normalized_grade LIKE '%7—12%' THEN
        RETURN ARRAY[7, 8, 9, 10, 11, 12];
    END IF;

    IF normalized_grade LIKE '%一年级到六年级%'
       OR normalized_grade LIKE '%一年级至六年级%'
       OR normalized_grade LIKE '%一到六%'
       OR normalized_grade LIKE '%一至六%'
       OR normalized_grade LIKE '%1年级到6年级%'
       OR normalized_grade LIKE '%1年级至6年级%'
       OR normalized_grade LIKE '%1到6%'
       OR normalized_grade LIKE '%1至6%'
       OR normalized_grade LIKE '%1-6%'
       OR normalized_grade LIKE '%1—6%' THEN
        RETURN ARRAY[1, 2, 3, 4, 5, 6];
    END IF;

    IF normalized_grade LIKE '%七年级到九年级%'
       OR normalized_grade LIKE '%七年级至九年级%'
       OR normalized_grade LIKE '%七到九%'
       OR normalized_grade LIKE '%七至九%'
       OR normalized_grade LIKE '%7年级到9年级%'
       OR normalized_grade LIKE '%7年级至9年级%'
       OR normalized_grade LIKE '%7到9%'
       OR normalized_grade LIKE '%7至9%'
       OR normalized_grade LIKE '%7-9%'
       OR normalized_grade LIKE '%7—9%' THEN
        RETURN ARRAY[7, 8, 9];
    END IF;

    IF normalized_grade LIKE '%十年级到十二年级%'
       OR normalized_grade LIKE '%十年级至十二年级%'
       OR normalized_grade LIKE '%十到十二%'
       OR normalized_grade LIKE '%十至十二%'
       OR normalized_grade LIKE '%10年级到12年级%'
       OR normalized_grade LIKE '%10年级至12年级%'
       OR normalized_grade LIKE '%10到12%'
       OR normalized_grade LIKE '%10至12%'
       OR normalized_grade LIKE '%10-12%'
       OR normalized_grade LIKE '%10—12%' THEN
        RETURN ARRAY[10, 11, 12];
    END IF;

    -- 小学明确学段。
    IF normalized_grade LIKE '%小学低段%'
       OR normalized_grade LIKE '%小学低年级%' THEN
        RETURN ARRAY[1, 2];
    END IF;

    IF normalized_grade LIKE '%小学中段%'
       OR normalized_grade LIKE '%小学中年级%' THEN
        RETURN ARRAY[3, 4];
    END IF;

    IF normalized_grade LIKE '%小学高段%'
       OR normalized_grade LIKE '%小学高年级%' THEN
        RETURN ARRAY[5, 6];
    END IF;

    -- 初中和高中内部年级必须先于普通“一年级”等判断。
    IF normalized_grade LIKE '%初中一年级%'
       OR normalized_grade LIKE '%初中1年级%'
       OR normalized_grade LIKE '%初一%' THEN
        RETURN ARRAY[7];
    END IF;

    IF normalized_grade LIKE '%初中二年级%'
       OR normalized_grade LIKE '%初中2年级%'
       OR normalized_grade LIKE '%初二%' THEN
        RETURN ARRAY[8];
    END IF;

    IF normalized_grade LIKE '%初中三年级%'
       OR normalized_grade LIKE '%初中3年级%'
       OR normalized_grade LIKE '%初三%' THEN
        RETURN ARRAY[9];
    END IF;

    IF normalized_grade LIKE '%高中一年级%'
       OR normalized_grade LIKE '%高中1年级%'
       OR normalized_grade LIKE '%高一%' THEN
        RETURN ARRAY[10];
    END IF;

    IF normalized_grade LIKE '%高中二年级%'
       OR normalized_grade LIKE '%高中2年级%'
       OR normalized_grade LIKE '%高二%' THEN
        RETURN ARRAY[11];
    END IF;

    IF normalized_grade LIKE '%高中三年级%'
       OR normalized_grade LIKE '%高中3年级%'
       OR normalized_grade LIKE '%高三%' THEN
        RETURN ARRAY[12];
    END IF;

    -- 十二、十一、十年级必须排在一、二年级之前，
    -- 避免“十一年级”命中“一年级”，“十二年级”命中“二年级”。
    IF normalized_grade LIKE '%十二年级%'
       OR normalized_grade LIKE '%12年级%' THEN
        RETURN ARRAY[12];
    END IF;

    IF normalized_grade LIKE '%十一年级%'
       OR normalized_grade LIKE '%11年级%' THEN
        RETURN ARRAY[11];
    END IF;

    IF normalized_grade LIKE '%十年级%'
       OR normalized_grade LIKE '%10年级%' THEN
        RETURN ARRAY[10];
    END IF;

    IF normalized_grade LIKE '%九年级%'
       OR normalized_grade LIKE '%9年级%' THEN
        RETURN ARRAY[9];
    END IF;

    IF normalized_grade LIKE '%八年级%'
       OR normalized_grade LIKE '%8年级%' THEN
        RETURN ARRAY[8];
    END IF;

    IF normalized_grade LIKE '%七年级%'
       OR normalized_grade LIKE '%7年级%' THEN
        RETURN ARRAY[7];
    END IF;

    IF normalized_grade LIKE '%六年级%'
       OR normalized_grade LIKE '%6年级%' THEN
        RETURN ARRAY[6];
    END IF;

    IF normalized_grade LIKE '%五年级%'
       OR normalized_grade LIKE '%5年级%' THEN
        RETURN ARRAY[5];
    END IF;

    IF normalized_grade LIKE '%四年级%'
       OR normalized_grade LIKE '%4年级%' THEN
        RETURN ARRAY[4];
    END IF;

    IF normalized_grade LIKE '%三年级%'
       OR normalized_grade LIKE '%3年级%' THEN
        RETURN ARRAY[3];
    END IF;

    IF normalized_grade LIKE '%二年级%'
       OR normalized_grade LIKE '%2年级%' THEN
        RETURN ARRAY[2];
    END IF;

    IF normalized_grade LIKE '%一年级%'
       OR normalized_grade LIKE '%1年级%' THEN
        RETURN ARRAY[1];
    END IF;

    -- 只有没有具体年级或明确范围时，纯学段名称才扩展为整个学段。
    IF normalized_grade LIKE '%小学%' THEN
        RETURN ARRAY[1, 2, 3, 4, 5, 6];
    END IF;

    IF normalized_grade LIKE '%初中%' THEN
        RETURN ARRAY[7, 8, 9];
    END IF;

    IF normalized_grade LIKE '%高中%' THEN
        RETURN ARRAY[10, 11, 12];
    END IF;

    IF normalized_grade LIKE '%中学%' THEN
        RETURN ARRAY[7, 8, 9, 10, 11, 12];
    END IF;

    RETURN empty_span;
END;
$$;

COMMENT ON FUNCTION public.lesson_plan_course_outline_grade_span(TEXT)
IS '按优先级把K12具体年级或学段解析为1至12年级集合；非K12或无法解析时返回空数组';

-- ============================================================================
-- 二、判断课程大纲与教案年级或学段是否允许绑定
-- ============================================================================

CREATE OR REPLACE FUNCTION public.lesson_plan_course_outline_grade_matches(
    outline_grade TEXT,
    lesson_grade TEXT
)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    normalized_outline_grade TEXT;
    normalized_lesson_grade TEXT;
    outline_span INTEGER[];
    lesson_span INTEGER[];
BEGIN
    normalized_outline_grade := regexp_replace(
        lower(btrim(COALESCE(outline_grade, ''))),
        E'\\s+',
        '',
        'g'
    );

    normalized_lesson_grade := regexp_replace(
        lower(btrim(COALESCE(lesson_grade, ''))),
        E'\\s+',
        '',
        'g'
    );

    IF normalized_outline_grade = ''
       OR normalized_lesson_grade = '' THEN
        RETURN FALSE;
    END IF;

    outline_span := public.lesson_plan_course_outline_grade_span(outline_grade);
    lesson_span := public.lesson_plan_course_outline_grade_span(lesson_grade);

    -- 两侧都能安全解释为K12年级集合时，任一交集即可绑定。
    IF cardinality(outline_span) > 0
       AND cardinality(lesson_span) > 0 THEN
        RETURN EXISTS (
            SELECT 1
            FROM unnest(outline_span) AS outline_grade_value(value)
            WHERE outline_grade_value.value = ANY(lesson_span)
        );
    END IF;

    -- 非K12或无法解释的层级只允许规范化文本完全相同。
    RETURN normalized_outline_grade = normalized_lesson_grade;
END;
$$;

COMMENT ON FUNCTION public.lesson_plan_course_outline_grade_matches(TEXT, TEXT)
IS 'K12按年级集合相交判断课程大纲可绑定性，非K12按规范化文本精确匹配';

-- ============================================================================
-- 三、重建课程大纲快照触发器函数
-- ============================================================================

CREATE OR REPLACE FUNCTION public.apply_lesson_plan_course_outline_snapshot()
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
    -- 没有唯一大纲ID时，只允许保留旧publisher-only兼容状态。
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
        RAISE EXCEPTION '精确课程大纲不存在'
            USING ERRCODE = '23503',
                  CONSTRAINT = 'lesson_plans_course_outline_id_fkey';
    END IF;

    IF outline_status <> 'active' THEN
        RAISE EXCEPTION '精确课程大纲不是生效状态'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'lesson_plans_course_outline_active_required';
    END IF;

    CASE outline_scope
        WHEN 'system' THEN
            IF outline_scope_target_id IS DISTINCT FROM
                '00000000-0000-0000-0000-000000000000'::uuid THEN
                RAISE EXCEPTION '全局课程大纲归属ID非法'
                    USING ERRCODE = '23514',
                          CONSTRAINT = 'lesson_plans_course_outline_scope_domain';
            END IF;

            outline_domain := 'k12';

        WHEN 'school' THEN
            SELECT lower(btrim(education_domain))
            INTO outline_domain
            FROM public.organizations
            WHERE id = outline_scope_target_id
              AND type = 'school'
              AND status = 'active';

        WHEN 'group' THEN
            SELECT lower(btrim(org.education_domain))
            INTO outline_domain
            FROM public.teaching_groups tg
            JOIN public.organizations org
              ON org.id = tg.school_id
             AND org.type = 'school'
             AND org.status = 'active'
            WHERE tg.id = outline_scope_target_id
              AND tg.status = 'active';

        ELSE
            RAISE EXCEPTION '课程大纲归属类型非法'
                USING ERRCODE = '23514',
                      CONSTRAINT = 'lesson_plans_course_outline_scope_domain';
    END CASE;

    IF outline_domain IS NULL
       OR outline_domain NOT IN ('k12', 'vocational', 'adult')
       OR outline_domain IS DISTINCT FROM lower(btrim(NEW.education_domain)) THEN
        RAISE EXCEPTION '精确课程大纲教育域与教案不一致'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'lesson_plans_course_outline_domain_match';
    END IF;

    IF btrim(outline_subject) IS DISTINCT FROM btrim(NEW.subject) THEN
        RAISE EXCEPTION '精确课程大纲学科与教案不一致'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'lesson_plans_course_outline_subject_match';
    END IF;

    -- 自动匹配仍由API使用具体年级精确模式完成。
    -- 数据库最终绑定允许教师选择覆盖当前年级的学段大纲，
    -- 但完全无关的年级或学段仍会被拒绝。
    IF NOT public.lesson_plan_course_outline_grade_matches(outline_grade, NEW.grade) THEN
        RAISE EXCEPTION '精确课程大纲年级或学段与教案不匹配'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'lesson_plans_course_outline_grade_match';
    END IF;

    NEW.course_outline_publisher := btrim(outline_publisher);
    NEW.course_outline_volume := btrim(outline_volume);
    NEW.school_system := btrim(outline_school_system);

    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION public.apply_lesson_plan_course_outline_snapshot()
IS '课程大纲绑定时校验active、教育域、学科及年级或学段交集，并固化出版社、册次和学制快照';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE '课程大纲手动学段选择迁移已就绪，尚未由本文件生成步骤自动执行';
END
$$;
