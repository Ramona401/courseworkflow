-- ============================================================================
-- 20260715_education_domain_foundation.sql
-- TE-DNA 教育域隔离第一阶段数据库底座
--
-- 目标：
--   1. 为组织增加教育域：
--        k12          中小学
--        vocational   职业教育
--        adult        成人教育
--        mixed        区域、教育局或其它跨域管理组织
--
--   2. 新建课程目录映射表 subject_catalog_entries：
--        - subjects 继续作为全平台统一课程定义表；
--        - organization_id为空：教育域公共课程；
--        - organization_id非空：某所学校或机构自己的课程；
--        - 同一课程可以同时属于多个教育域或多个学校；
--        - 职业教育课程不会进入中小学教师的课程列表。
--
--   3. 平滑迁移现有数据：
--        - 现有区域组织标记mixed；
--        - 普通现有学校标记k12；
--        - 临沂市电子科技学校标记vocational；
--        - 历史上以school类型保存的“郯城教育局”标记mixed；
--        - 当前18个内置学科进入K12公共课程目录；
--        - 临沂市电子科技学校当前教研组学科进入该校职业课程目录。
--
-- 分阶段上线说明：
--   education_domain本批暂不设置NOT NULL。
--   原因是数据库迁移必须先于Go代码部署，避免旧二进制创建组织时因缺少新字段失败。
--   后续Go代码完成写入与解析并上线验收后，再执行收口迁移：
--     - 设置新建组织默认教育域；
--     - 补齐空值；
--     - 将organizations.education_domain改为NOT NULL。
--
-- 本文件使用IF NOT EXISTS、条件约束和ON CONFLICT DO NOTHING，
-- 可安全重复执行。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、组织教育域字段
-- ============================================================================

ALTER TABLE public.organizations
    ADD COLUMN IF NOT EXISTS education_domain character varying(20);

COMMENT ON COLUMN public.organizations.education_domain IS
    '教育域：k12中小学/vocational职业教育/adult成人教育/mixed跨域管理组织；第一阶段允许NULL，代码接入后再收紧NOT NULL';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'organizations_education_domain_check'
          AND conrelid = 'public.organizations'::regclass
    ) THEN
        ALTER TABLE public.organizations
            ADD CONSTRAINT organizations_education_domain_check
            CHECK (
                education_domain IS NULL
                OR education_domain::text = ANY (
                    ARRAY[
                        'k12',
                        'vocational',
                        'adult',
                        'mixed'
                    ]::text[]
                )
            );
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_organizations_education_domain
    ON public.organizations (education_domain, type, status);

-- 现有区域均作为跨教育域管理组织。
UPDATE public.organizations
SET
    education_domain = 'mixed',
    updated_at = now()
WHERE type = 'region'
  AND education_domain IS DISTINCT FROM 'mixed';

-- 历史上以school类型建立的教育局占位组织不承担具体教学，
-- 标记为mixed，避免其校籍影响教师具体教学教育域判断。
UPDATE public.organizations
SET
    education_domain = 'mixed',
    updated_at = now()
WHERE type = 'school'
  AND name = '郯城教育局'
  AND education_domain IS DISTINCT FROM 'mixed';

-- 临沂市电子科技学校正式标记为职业教育。
UPDATE public.organizations
SET
    education_domain = 'vocational',
    updated_at = now()
WHERE id = '2ea8feed-6081-4b00-b602-390dc330ed03'
  AND education_domain IS DISTINCT FROM 'vocational';

-- 其余当前尚未标记的学校全部保持现有K12语义。
UPDATE public.organizations
SET
    education_domain = 'k12',
    updated_at = now()
WHERE type = 'school'
  AND education_domain IS NULL;

-- ============================================================================
-- 二、课程目录映射表
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.subject_catalog_entries (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,

    subject_id uuid NOT NULL,
    education_domain character varying(20) NOT NULL,
    organization_id uuid,

    display_name character varying(100) NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 100,
    is_active boolean NOT NULL DEFAULT true,

    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),

    CONSTRAINT subject_catalog_entries_subject_fkey
        FOREIGN KEY (subject_id)
        REFERENCES public.subjects(id)
        ON DELETE CASCADE,

    CONSTRAINT subject_catalog_entries_organization_fkey
        FOREIGN KEY (organization_id)
        REFERENCES public.organizations(id)
        ON DELETE CASCADE,

    CONSTRAINT subject_catalog_entries_domain_check
        CHECK (
            education_domain::text = ANY (
                ARRAY[
                    'k12',
                    'vocational',
                    'adult'
                ]::text[]
            )
        ),

    CONSTRAINT subject_catalog_entries_sort_order_check
        CHECK (sort_order >= 0)
);

COMMENT ON TABLE public.subject_catalog_entries IS
    '教育域课程目录映射：organization_id为空表示教育域公共课程，非空表示具体学校或机构课程';

COMMENT ON COLUMN public.subject_catalog_entries.subject_id IS
    '关联统一课程定义subjects.id';

COMMENT ON COLUMN public.subject_catalog_entries.education_domain IS
    '课程所属教学教育域：k12/vocational/adult，不使用mixed';

COMMENT ON COLUMN public.subject_catalog_entries.organization_id IS
    'NULL=该教育域公共课程；非NULL=仅该学校或机构可见的课程';

COMMENT ON COLUMN public.subject_catalog_entries.display_name IS
    '该目录中的展示名；允许未来同一subjects课程在不同教育域使用不同显示名称';

-- 每个教育域中，同一课程只能有一个公共目录条目。
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_catalog_domain_public
    ON public.subject_catalog_entries (
        education_domain,
        subject_id
    )
    WHERE organization_id IS NULL;

-- 每个具体组织中，同一课程只能出现一次。
CREATE UNIQUE INDEX IF NOT EXISTS uq_subject_catalog_organization_subject
    ON public.subject_catalog_entries (
        organization_id,
        subject_id
    )
    WHERE organization_id IS NOT NULL;

-- 教育域公共课程高频查询索引。
CREATE INDEX IF NOT EXISTS idx_subject_catalog_domain_active
    ON public.subject_catalog_entries (
        education_domain,
        is_active,
        sort_order,
        display_name
    )
    WHERE organization_id IS NULL;

-- 学校或机构私有课程高频查询索引。
CREATE INDEX IF NOT EXISTS idx_subject_catalog_organization_active
    ON public.subject_catalog_entries (
        organization_id,
        is_active,
        sort_order,
        display_name
    )
    WHERE organization_id IS NOT NULL;

-- ============================================================================
-- 三、K12公共课程目录
-- ============================================================================

-- 当前18个内置核心学科均为is_system=true。
-- 只把内置学科映射为K12公共课程，避免随后新增的职业课程误进入K12目录。
INSERT INTO public.subject_catalog_entries (
    subject_id,
    education_domain,
    organization_id,
    display_name,
    sort_order,
    is_active
)
SELECT
    s.id,
    'k12',
    NULL,
    s.name,
    s.sort_order,
    true
FROM public.subjects s
WHERE s.is_system = true
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 四、临沂市电子科技学校职业课程定义
-- ============================================================================

-- 根据该校当前有效教研组的subject自动补齐统一课程定义。
-- 已存在的语文、数学、英语、美术等课程复用原subjects记录；
-- 机械制图、数控技术、会计事务等不存在的课程才新增subjects记录。
WITH vocational_subject_names AS (
    SELECT DISTINCT btrim(tg.subject) AS subject_name
    FROM public.teaching_groups tg
    WHERE tg.school_id = '2ea8feed-6081-4b00-b602-390dc330ed03'
      AND tg.status = 'active'
      AND btrim(tg.subject) <> ''
),
ordered_subject_names AS (
    SELECT
        subject_name,
        row_number() OVER (ORDER BY subject_name) AS seq
    FROM vocational_subject_names
)
INSERT INTO public.subjects (
    name,
    code,
    sort_order,
    is_active,
    is_system,
    note
)
SELECT
    subject_name,
    '',
    1000 + (seq::integer * 10),
    true,
    false,
    '临沂市电子科技学校职业教育课程目录'
FROM ordered_subject_names
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- 五、临沂市电子科技学校本校职业课程目录
-- ============================================================================

WITH target_subjects AS (
    SELECT DISTINCT
        s.id AS subject_id,
        s.name AS subject_name
    FROM public.teaching_groups tg
    JOIN public.subjects s
      ON s.name = btrim(tg.subject)
    WHERE tg.school_id = '2ea8feed-6081-4b00-b602-390dc330ed03'
      AND tg.status = 'active'
      AND btrim(tg.subject) <> ''
),
ordered_target_subjects AS (
    SELECT
        subject_id,
        subject_name,
        row_number() OVER (ORDER BY subject_name) AS seq
    FROM target_subjects
)
INSERT INTO public.subject_catalog_entries (
    subject_id,
    education_domain,
    organization_id,
    display_name,
    sort_order,
    is_active
)
SELECT
    subject_id,
    'vocational',
    '2ea8feed-6081-4b00-b602-390dc330ed03',
    subject_name,
    100 + (seq::integer * 10),
    true
FROM ordered_target_subjects
ON CONFLICT DO NOTHING;

ALTER TABLE public.subject_catalog_entries OWNER TO postgres;
GRANT ALL ON TABLE public.subject_catalog_entries TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE '教育域数据库底座迁移完成';
    RAISE NOTICE 'organizations.education_domain 已建立并回填';
    RAISE NOTICE 'subject_catalog_entries 已建立';
    RAISE NOTICE 'K12公共课程和临沂电子科技学校职业课程目录已写入';
END
$$;
