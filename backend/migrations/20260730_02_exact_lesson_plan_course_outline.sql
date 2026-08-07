\set ON_ERROR_STOP on

-- ============================================================================
-- 20260730_02_exact_lesson_plan_course_outline.sql
-- 教案精确课程大纲挂载：学制、册次快照与精确大纲ID
--
-- 目标：
--   1. course_outlines新增school_system，拆开普通学制与五四制；
--   2. 把存量“册次(五四制)”规范化为“册次 + school_system=five_four”；
--   3. lesson_plans新增school_system、course_outline_volume、course_outline_id；
--   4. 精确大纲ID写入时，由数据库触发器校验教育域、学科、具体年级，
--      并固化出版社、册次和学制快照；
--   5. 兼容迁移后短暂运行的旧后端：旧接口继续写publisher-only教案时不报错，
--      旧课程大纲接口提交“上册(五四制)”时由数据库自动拆分学制与册次；
--   6. 保留存量course_outline_publisher，但不自动猜测或回填course_outline_id；
--   7. 本迁移不删除任何课程大纲，不修改教案正文。
--
-- 存量兼容：
--   - 标题或册次明确包含“五四制”的大纲迁移为five_four；
--   - 其余大纲迁移为standard；
--   - 旧教案只有出版社、缺少册次与精确ID，继续保留旧字段，
--     后续后端要求老师重新选择精确大纲，绝不模糊自动映射。
-- ============================================================================

BEGIN;

LOCK TABLE public.course_outlines
    IN SHARE ROW EXCLUSIVE MODE;

LOCK TABLE public.lesson_plans
    IN SHARE ROW EXCLUSIVE MODE;

-- ============================================================================
-- 一、课程大纲增加正式学制维度
-- ============================================================================

ALTER TABLE public.course_outlines
    ADD COLUMN IF NOT EXISTS school_system VARCHAR(20);

-- 存量五四制只依据标题或册次中的明确标记分类。
-- 不扫描正文，避免普通大纲正文偶然提及“五四制”而被误分类。
UPDATE public.course_outlines
SET school_system = CASE
    WHEN volume ILIKE '%五四制%'
      OR title ILIKE '%五四制%'
    THEN 'five_four'
    ELSE 'standard'
END
WHERE school_system IS NULL
   OR btrim(school_system) = ''
   OR school_system NOT IN (
       'standard',
       'five_four'
   )
   OR (
       (
           volume ILIKE '%五四制%'
           OR title ILIKE '%五四制%'
       )
       AND school_system <> 'five_four'
   );

-- 必须先移除旧唯一索引，再把五四制册次规范为“上册/下册”。
-- 否则同一归属下普通“上册”和“五四制上册”会在UPDATE阶段撞旧索引。
DROP INDEX IF EXISTS
    public.uq_course_outlines_active;

UPDATE public.course_outlines
SET volume = btrim(
    regexp_replace(
        regexp_replace(
            volume,
            E'\\s*[（(]五四制[）)]\\s*',
            '',
            'g'
        ),
        '五四制',
        '',
        'g'
    )
)
WHERE volume ILIKE '%五四制%';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM public.course_outlines
        WHERE btrim(volume) = ''
    ) THEN
        RAISE EXCEPTION
            '课程大纲册次规范化后出现空值，迁移已中止';
    END IF;

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
            publisher,
            school_system
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            '同归属下存在重复的学科、年级、册次、出版社、学制课程大纲，迁移已中止';
    END IF;
END
$$;

ALTER TABLE public.course_outlines
    ALTER COLUMN school_system
    SET DEFAULT 'standard';

ALTER TABLE public.course_outlines
    ALTER COLUMN school_system
    SET NOT NULL;

ALTER TABLE public.course_outlines
    DROP CONSTRAINT IF EXISTS
        course_outlines_school_system_check;

ALTER TABLE public.course_outlines
    ADD CONSTRAINT
        course_outlines_school_system_check
    CHECK (
        school_system IN (
            'standard',
            'five_four'
        )
    );

ALTER TABLE public.course_outlines
    DROP CONSTRAINT IF EXISTS
        course_outlines_school_system_trim_check;

ALTER TABLE public.course_outlines
    ADD CONSTRAINT
        course_outlines_school_system_trim_check
    CHECK (
        school_system = btrim(school_system)
    );

CREATE UNIQUE INDEX
    uq_course_outlines_active
ON public.course_outlines (
    scope,
    scope_target_id,
    subject,
    grade,
    volume,
    publisher,
    school_system
)
WHERE status = 'active';

DROP INDEX IF EXISTS
    public.idx_course_outlines_lookup;

CREATE INDEX
    idx_course_outlines_lookup
ON public.course_outlines (
    subject,
    school_system,
    publisher,
    grade,
    volume,
    status
);

COMMENT ON COLUMN
    public.course_outlines.school_system
IS
    'K12教材学制：standard=普通学制，five_four=五四制；非K12运行时忽略该字段';

-- 兼容迁移后短暂运行的旧后端：
-- 旧接口仍只提交volume/title，不提交school_system。
-- 数据库必须在写入时识别“(五四制)”标记、归一化册次，避免产生错误standard记录。
CREATE OR REPLACE FUNCTION
    public.normalize_course_outline_school_system()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.volume := btrim(
        COALESCE(
            NEW.volume,
            ''
        )
    );

    NEW.school_system := lower(
        btrim(
            COALESCE(
                NEW.school_system,
                ''
            )
        )
    );

    IF NEW.volume ILIKE '%五四制%'
       OR COALESCE(NEW.title, '') ILIKE '%五四制%' THEN
        NEW.school_system := 'five_four';
    ELSIF NEW.school_system = '' THEN
        NEW.school_system := 'standard';
    END IF;

    IF NEW.school_system NOT IN (
        'standard',
        'five_four'
    ) THEN
        RAISE EXCEPTION
            '课程大纲学制必须是standard或five_four'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'course_outlines_school_system_check';
    END IF;

    NEW.volume := btrim(
        regexp_replace(
            regexp_replace(
                NEW.volume,
                E'\\s*[（(]五四制[）)]\\s*',
                '',
                'g'
            ),
            '五四制',
            '',
            'g'
        )
    );

    IF NEW.volume = '' THEN
        RAISE EXCEPTION
            '课程大纲册次不能为空'
            USING
                ERRCODE = '23514',
                CONSTRAINT =
                    'course_outlines_volume_required';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_course_outlines_00_school_system_insert
ON public.course_outlines;

CREATE TRIGGER
    trg_course_outlines_00_school_system_insert
BEFORE INSERT
ON public.course_outlines
FOR EACH ROW
EXECUTE FUNCTION
    public.normalize_course_outline_school_system();

DROP TRIGGER IF EXISTS
    trg_course_outlines_00_school_system_update
ON public.course_outlines;

CREATE TRIGGER
    trg_course_outlines_00_school_system_update
BEFORE UPDATE OF volume, title, school_system
ON public.course_outlines
FOR EACH ROW
EXECUTE FUNCTION
    public.normalize_course_outline_school_system();

COMMENT ON FUNCTION
    public.normalize_course_outline_school_system()
IS
    '课程大纲写入时拆分五四制标记与册次；00前缀保证先于出版社教育域触发器执行';

-- ============================================================================
-- 二、教案增加精确课程大纲ID与运行快照
-- ============================================================================

ALTER TABLE public.lesson_plans
    ADD COLUMN IF NOT EXISTS school_system VARCHAR(20);

ALTER TABLE public.lesson_plans
    ADD COLUMN IF NOT EXISTS course_outline_volume VARCHAR(255);

ALTER TABLE public.lesson_plans
    ADD COLUMN IF NOT EXISTS course_outline_id UUID;

-- 支持安全重复执行：若此前已经存在精确ID，先按正式大纲补齐快照。
UPDATE public.lesson_plans lp
SET
    course_outline_publisher =
        btrim(co.publisher),
    course_outline_volume =
        btrim(co.volume),
    school_system =
        btrim(co.school_system)
FROM public.course_outlines co
WHERE lp.course_outline_id = co.id;

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_school_system_check;

ALTER TABLE public.lesson_plans
    ADD CONSTRAINT
        lesson_plans_school_system_check
    CHECK (
        school_system IS NULL
        OR school_system IN (
            'standard',
            'five_four'
        )
    );

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_school_system_trim_check;

ALTER TABLE public.lesson_plans
    ADD CONSTRAINT
        lesson_plans_school_system_trim_check
    CHECK (
        school_system IS NULL
        OR school_system = btrim(school_system)
    );

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_volume_trim_check;

ALTER TABLE public.lesson_plans
    ADD CONSTRAINT
        lesson_plans_course_outline_volume_trim_check
    CHECK (
        course_outline_volume IS NULL
        OR (
            course_outline_volume = btrim(course_outline_volume)
            AND course_outline_volume <> ''
        )
    );

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_exact_snapshot_check;

ALTER TABLE public.lesson_plans
    ADD CONSTRAINT
        lesson_plans_course_outline_exact_snapshot_check
    CHECK (
        (
            course_outline_id IS NULL
            AND course_outline_volume IS NULL
            AND school_system IS NULL
        )
        OR (
            course_outline_id IS NOT NULL
            AND course_outline_publisher IS NOT NULL
            AND course_outline_volume IS NOT NULL
            AND school_system IS NOT NULL
        )
    );

ALTER TABLE public.lesson_plans
    DROP CONSTRAINT IF EXISTS
        lesson_plans_course_outline_id_fkey;

ALTER TABLE public.lesson_plans
    ADD CONSTRAINT
        lesson_plans_course_outline_id_fkey
    FOREIGN KEY (
        course_outline_id
    )
    REFERENCES public.course_outlines(id)
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS
    idx_lesson_plans_course_outline_id
ON public.lesson_plans (
    course_outline_id
)
WHERE course_outline_id IS NOT NULL;

COMMENT ON COLUMN
    public.lesson_plans.school_system
IS
    '教案精确课程大纲学制快照：standard=普通学制，five_four=五四制';

COMMENT ON COLUMN
    public.lesson_plans.course_outline_volume
IS
    '教案精确课程大纲册次快照';

COMMENT ON COLUMN
    public.lesson_plans.course_outline_id
IS
    '教案精确关联的课程大纲ID；旧publisher-only教案保持NULL并要求重新选择';

-- ============================================================================
-- 三、精确大纲挂载数据库快照触发器
-- ============================================================================

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
    -- 没有精确ID时只允许保留旧publisher-only兼容状态。
    -- 册次和学制属于精确挂载快照，必须清空。
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

    -- 新精确挂载只接受具体年级完全相等。
    -- 不再使用“小学/初中学段相交”或相邻年级兜底。
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

-- INSERT时必须晚于既有education_domain快照触发器执行，
-- 因此使用zz前缀；PostgreSQL同类触发器按名称顺序执行。
DROP TRIGGER IF EXISTS
    trg_lesson_plans_zz_course_outline_snapshot_insert
ON public.lesson_plans;

CREATE TRIGGER
    trg_lesson_plans_zz_course_outline_snapshot_insert
BEFORE INSERT
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_lesson_plan_course_outline_snapshot();

DROP TRIGGER IF EXISTS
    trg_lesson_plans_zz_course_outline_snapshot_update
ON public.lesson_plans;

CREATE TRIGGER
    trg_lesson_plans_zz_course_outline_snapshot_update
BEFORE UPDATE OF
    course_outline_id,
    subject,
    grade,
    course_outline_publisher,
    course_outline_volume,
    school_system
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_lesson_plan_course_outline_snapshot();

COMMENT ON FUNCTION
    public.apply_lesson_plan_course_outline_snapshot()
IS
    '精确课程大纲挂载时校验active、归属教育域、学科和具体年级，并固化出版社、册次与学制快照';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '精确课程大纲数据库迁移完成；旧publisher-only教案未自动映射';
END
$$;
