-- ============================================================================
-- 20260715_resource_education_domain_snapshots.sql
-- TE-DNA 四类核心教学资源教育域快照底座
--
-- 资源表：
--   lesson_plans
--   coursewares
--   teaching_recipes
--   ai_assistants
--
-- 资源允许的教育域：
--   k12          中小学
--   vocational   职业教育
--   adult        成人教育
--   common       明确设计为跨教学域通用的资源
--
-- mixed不允许写入教学资源：
--   mixed只表示平台、区域或教育局的跨域管理上下文；
--   管理账号当前创建教学资源时，为兼容既有运维和测试流程，默认归入k12。
--
-- 创建快照规则：
--   1. 教案：
--        Fork时继承来源教案域，否则按作者当前教学域；
--   2. 课件：
--        有来源教案时继承教案域，否则按创建用户当前教学域；
--   3. 配方：
--        Fork时继承来源配方域，否则按作者当前教学域；
--   4. AI助手：
--        Fork时继承来源助手域；
--        system且未显式指定时归入k12；
--        其它助手按创建者当前教学域。
--
-- 本迁移与旧二进制兼容：
--   旧代码INSERT未提供education_domain时，数据库触发器会自动写入。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、为四类核心资源增加教育域字段
-- ============================================================================

ALTER TABLE public.lesson_plans
    ADD COLUMN IF NOT EXISTS education_domain character varying(20);

ALTER TABLE public.coursewares
    ADD COLUMN IF NOT EXISTS education_domain character varying(20);

ALTER TABLE public.teaching_recipes
    ADD COLUMN IF NOT EXISTS education_domain character varying(20);

ALTER TABLE public.ai_assistants
    ADD COLUMN IF NOT EXISTS education_domain character varying(20);

COMMENT ON COLUMN public.lesson_plans.education_domain IS
    '教案创建时教育域快照：k12/vocational/adult/common，不随作者之后的组织变化而改变';

COMMENT ON COLUMN public.coursewares.education_domain IS
    '课件创建时教育域快照：优先继承来源教案，否则按创建者当前教学域';

COMMENT ON COLUMN public.teaching_recipes.education_domain IS
    '配方创建时教育域快照：Fork继承来源，否则按作者当前教学域';

COMMENT ON COLUMN public.ai_assistants.education_domain IS
    'AI助手创建时教育域快照：Fork继承来源，现有system助手默认k12';

-- ============================================================================
-- 二、统一解析用户当前教学教育域
-- ============================================================================

CREATE OR REPLACE FUNCTION public.resolve_resource_education_domain(
    p_user_id uuid
)
RETURNS character varying(20)
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    user_role character varying(20);
    resolved_domain character varying(20);
BEGIN
    IF p_user_id IS NULL THEN
        RETURN 'k12';
    END IF;

    SELECT u.role
    INTO user_role
    FROM public.users u
    WHERE u.id = p_user_id;

    -- mixed管理身份当前没有主动选择工作教育域的机制。
    -- 为兼容既有管理员测试与资源维护，创建教学资源时暂归K12。
    IF user_role IN (
        'admin',
        'region_admin',
        'district_inspector'
    ) THEN
        RETURN 'k12';
    END IF;

    SELECT candidate.education_domain
    INTO resolved_domain
    FROM (
        -- 正式主管理员任命
        SELECT
            o.education_domain,
            1 AS source_priority,
            COALESCE(
                o.created_at,
                now()
            ) AS linked_at,
            o.name,
            o.id
        FROM public.organizations o
        WHERE o.admin_user_id = p_user_id
          AND o.status = 'active'

        UNION ALL

        -- 多管理员任命
        SELECT
            o.education_domain,
            1,
            COALESCE(
                oa.created_at AT TIME ZONE
                    current_setting('TIMEZONE'),
                o.created_at,
                now()
            ),
            o.name,
            o.id
        FROM public.organization_admins oa
        JOIN public.organizations o
          ON o.id = oa.org_id
        WHERE oa.user_id = p_user_id
          AND o.status = 'active'

        UNION ALL

        -- 学校直接校籍
        SELECT
            o.education_domain,
            2,
            COALESCE(
                sm.joined_at,
                o.created_at,
                now()
            ),
            o.name,
            o.id
        FROM public.school_members sm
        JOIN public.organizations o
          ON o.id = sm.school_id
        WHERE sm.user_id = p_user_id
          AND o.status = 'active'

        UNION ALL

        -- 教研组成员历史归属兜底
        SELECT
            o.education_domain,
            3,
            COALESCE(
                tgm.joined_at,
                tg.created_at,
                o.created_at,
                now()
            ),
            o.name,
            o.id
        FROM public.teaching_group_members tgm
        JOIN public.teaching_groups tg
          ON tg.id = tgm.group_id
        JOIN public.organizations o
          ON o.id = tg.school_id
        WHERE tgm.user_id = p_user_id
          AND o.status = 'active'
    ) AS candidate
    WHERE candidate.education_domain IN (
        'k12',
        'vocational',
        'adult'
    )
    ORDER BY
        candidate.source_priority,
        candidate.linked_at,
        candidate.name,
        candidate.id
    LIMIT 1;

    RETURN COALESCE(
        resolved_domain,
        'k12'
    );
END;
$$;

COMMENT ON FUNCTION public.resolve_resource_education_domain(uuid) IS
    '解析用户创建教学资源时使用的教学教育域；mixed管理身份和无具体教学组织用户兼容回退k12';

GRANT EXECUTE
ON FUNCTION public.resolve_resource_education_domain(uuid)
TO tedna_user;

-- ============================================================================
-- 三、历史教案回填
-- ============================================================================

UPDATE public.lesson_plans lp
SET education_domain =
    public.resolve_resource_education_domain(
        lp.author_id
    )
WHERE lp.education_domain IS NULL
   OR lp.education_domain NOT IN (
        'k12',
        'vocational',
        'adult',
        'common'
   );

-- Fork教案必须继承来源教案域，而不是按新作者当前组织重新分类。
UPDATE public.lesson_plans child
SET education_domain = parent.education_domain
FROM public.lesson_plans parent
WHERE child.forked_from = parent.id
  AND child.education_domain IS DISTINCT FROM
      parent.education_domain;

-- ============================================================================
-- 四、历史课件回填
-- ============================================================================

UPDATE public.coursewares cw
SET education_domain = COALESCE(
    (
        SELECT lp.education_domain
        FROM public.lesson_plans lp
        WHERE lp.id = cw.lesson_plan_id
    ),
    public.resolve_resource_education_domain(
        cw.user_id
    )
)
WHERE cw.education_domain IS NULL
   OR cw.education_domain NOT IN (
        'k12',
        'vocational',
        'adult',
        'common'
   );

-- 已有关联教案的课件，以来源教案快照为唯一准绳。
UPDATE public.coursewares cw
SET education_domain = lp.education_domain
FROM public.lesson_plans lp
WHERE cw.lesson_plan_id = lp.id
  AND cw.education_domain IS DISTINCT FROM
      lp.education_domain;

-- ============================================================================
-- 五、历史配方回填
-- ============================================================================

UPDATE public.teaching_recipes recipe
SET education_domain =
    public.resolve_resource_education_domain(
        recipe.author_id
    )
WHERE recipe.education_domain IS NULL
   OR recipe.education_domain NOT IN (
        'k12',
        'vocational',
        'adult',
        'common'
   );

UPDATE public.teaching_recipes child
SET education_domain = parent.education_domain
FROM public.teaching_recipes parent
WHERE child.forked_from = parent.id
  AND child.education_domain IS DISTINCT FROM
      parent.education_domain;

-- ============================================================================
-- 六、历史AI助手回填
-- ============================================================================

-- 现有系统助手目前均来自K12助手体系，先安全归入K12。
UPDATE public.ai_assistants
SET education_domain = 'k12'
WHERE source = 'system'
  AND (
      education_domain IS NULL
      OR education_domain NOT IN (
          'k12',
          'vocational',
          'adult',
          'common'
      )
  );

UPDATE public.ai_assistants assistant
SET education_domain =
    public.resolve_resource_education_domain(
        assistant.created_by
    )
WHERE assistant.source <> 'system'
  AND (
      assistant.education_domain IS NULL
      OR assistant.education_domain NOT IN (
          'k12',
          'vocational',
          'adult',
          'common'
      )
  );

UPDATE public.ai_assistants child
SET education_domain = parent.education_domain
FROM public.ai_assistants parent
WHERE child.forked_from = parent.id
  AND child.education_domain IS DISTINCT FROM
      parent.education_domain;

-- ============================================================================
-- 七、资源教育域约束
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'lesson_plans_education_domain_check'
          AND conrelid =
            'public.lesson_plans'::regclass
    ) THEN
        ALTER TABLE public.lesson_plans
            ADD CONSTRAINT
                lesson_plans_education_domain_check
            CHECK (
                education_domain IN (
                    'k12',
                    'vocational',
                    'adult',
                    'common'
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'coursewares_education_domain_check'
          AND conrelid =
            'public.coursewares'::regclass
    ) THEN
        ALTER TABLE public.coursewares
            ADD CONSTRAINT
                coursewares_education_domain_check
            CHECK (
                education_domain IN (
                    'k12',
                    'vocational',
                    'adult',
                    'common'
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'teaching_recipes_education_domain_check'
          AND conrelid =
            'public.teaching_recipes'::regclass
    ) THEN
        ALTER TABLE public.teaching_recipes
            ADD CONSTRAINT
                teaching_recipes_education_domain_check
            CHECK (
                education_domain IN (
                    'k12',
                    'vocational',
                    'adult',
                    'common'
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'ai_assistants_education_domain_check'
          AND conrelid =
            'public.ai_assistants'::regclass
    ) THEN
        ALTER TABLE public.ai_assistants
            ADD CONSTRAINT
                ai_assistants_education_domain_check
            CHECK (
                education_domain IN (
                    'k12',
                    'vocational',
                    'adult',
                    'common'
                )
            );
    END IF;
END
$$;

ALTER TABLE public.lesson_plans
    ALTER COLUMN education_domain SET NOT NULL;

ALTER TABLE public.coursewares
    ALTER COLUMN education_domain SET NOT NULL;

ALTER TABLE public.teaching_recipes
    ALTER COLUMN education_domain SET NOT NULL;

ALTER TABLE public.ai_assistants
    ALTER COLUMN education_domain SET NOT NULL;

-- ============================================================================
-- 八、分域查询索引
-- ============================================================================

CREATE INDEX IF NOT EXISTS
    idx_lesson_plans_domain_updated
ON public.lesson_plans (
    education_domain,
    updated_at DESC
)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS
    idx_coursewares_domain_created
ON public.coursewares (
    education_domain,
    created_at DESC
)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS
    idx_teaching_recipes_domain_active
ON public.teaching_recipes (
    education_domain,
    subject,
    grade_range,
    updated_at DESC
)
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS
    idx_ai_assistants_domain_active
ON public.ai_assistants (
    education_domain,
    source,
    sort_order DESC
)
WHERE is_active = true;

-- ============================================================================
-- 九、教案插入快照触发器
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.apply_lesson_plan_education_domain_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_domain character varying(20);
BEGIN
    IF NEW.education_domain IN (
        'k12',
        'vocational',
        'adult',
        'common'
    ) THEN
        RETURN NEW;
    END IF;

    IF NEW.forked_from IS NOT NULL THEN
        SELECT source.education_domain
        INTO resolved_domain
        FROM public.lesson_plans source
        WHERE source.id = NEW.forked_from;
    END IF;

    NEW.education_domain = COALESCE(
        resolved_domain,
        public.resolve_resource_education_domain(
            NEW.author_id
        ),
        'k12'
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_lesson_plans_education_domain_snapshot
ON public.lesson_plans;

CREATE TRIGGER
    trg_lesson_plans_education_domain_snapshot
BEFORE INSERT
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_lesson_plan_education_domain_snapshot();

-- ============================================================================
-- 十、课件插入快照触发器
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.apply_courseware_education_domain_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_domain character varying(20);
BEGIN
    IF NEW.education_domain IN (
        'k12',
        'vocational',
        'adult',
        'common'
    ) THEN
        RETURN NEW;
    END IF;

    IF NEW.lesson_plan_id IS NOT NULL THEN
        SELECT source.education_domain
        INTO resolved_domain
        FROM public.lesson_plans source
        WHERE source.id = NEW.lesson_plan_id;
    END IF;

    NEW.education_domain = COALESCE(
        resolved_domain,
        public.resolve_resource_education_domain(
            NEW.user_id
        ),
        'k12'
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_coursewares_education_domain_snapshot
ON public.coursewares;

CREATE TRIGGER
    trg_coursewares_education_domain_snapshot
BEFORE INSERT
ON public.coursewares
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_courseware_education_domain_snapshot();

-- ============================================================================
-- 十一、配方插入快照触发器
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.apply_recipe_education_domain_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_domain character varying(20);
BEGIN
    IF NEW.education_domain IN (
        'k12',
        'vocational',
        'adult',
        'common'
    ) THEN
        RETURN NEW;
    END IF;

    IF NEW.forked_from IS NOT NULL THEN
        SELECT source.education_domain
        INTO resolved_domain
        FROM public.teaching_recipes source
        WHERE source.id = NEW.forked_from;
    END IF;

    NEW.education_domain = COALESCE(
        resolved_domain,
        public.resolve_resource_education_domain(
            NEW.author_id
        ),
        'k12'
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_teaching_recipes_education_domain_snapshot
ON public.teaching_recipes;

CREATE TRIGGER
    trg_teaching_recipes_education_domain_snapshot
BEFORE INSERT
ON public.teaching_recipes
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_recipe_education_domain_snapshot();

-- ============================================================================
-- 十二、AI助手插入快照触发器
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.apply_ai_assistant_education_domain_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_domain character varying(20);
BEGIN
    IF NEW.education_domain IN (
        'k12',
        'vocational',
        'adult',
        'common'
    ) THEN
        RETURN NEW;
    END IF;

    IF NEW.forked_from IS NOT NULL THEN
        SELECT source.education_domain
        INTO resolved_domain
        FROM public.ai_assistants source
        WHERE source.id = NEW.forked_from;
    ELSIF NEW.source = 'system' THEN
        resolved_domain = 'k12';
    ELSE
        resolved_domain =
            public.resolve_resource_education_domain(
                NEW.created_by
            );
    END IF;

    NEW.education_domain = COALESCE(
        resolved_domain,
        'k12'
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_ai_assistants_education_domain_snapshot
ON public.ai_assistants;

CREATE TRIGGER
    trg_ai_assistants_education_domain_snapshot
BEFORE INSERT
ON public.ai_assistants
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_ai_assistant_education_domain_snapshot();

GRANT EXECUTE
ON FUNCTION
    public.apply_lesson_plan_education_domain_snapshot()
TO tedna_user;

GRANT EXECUTE
ON FUNCTION
    public.apply_courseware_education_domain_snapshot()
TO tedna_user;

GRANT EXECUTE
ON FUNCTION
    public.apply_recipe_education_domain_snapshot()
TO tedna_user;

GRANT EXECUTE
ON FUNCTION
    public.apply_ai_assistant_education_domain_snapshot()
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '四类核心资源education_domain快照底座迁移完成';
END
$$;
