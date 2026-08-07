-- ============================================================================
-- 20260722_courseware_component_education_domain.sql
-- 上下文19：课件组件教育域隔离数据库底座
--
-- 业务目标：
--   1. courseware_components增加education_domain资源教育域；
--   2. 允许值仅为k12/vocational/adult/common，mixed禁止写入资源；
--   3. 现有23个组件可追溯迁移为k12：
--        - 全部历史组件引用均来自k12课件；
--        - 职教课件历史页面组件引用数为0；
--        - 现有组件来源是原K12课件组件体系；
--   4. 不把任何存量组件无依据迁移为common；
--   5. 为组件管理和课件生成匹配建立教育域索引；
--   6. 兼容迁移先于新Go二进制上线的部署顺序。
--
-- 兼容策略：
--   - 旧二进制INSERT未提交education_domain时，数据库DEFAULT写入k12；
--   - 新二进制必须显式写入可信Actor决定的教育域；
--   - 显式提交NULL仍被NOT NULL拒绝；
--   - 非法值和mixed由CHECK约束拒绝；
--   - 教育域不能通过普通组件更新接口修改，应用层另行保证。
--
-- 幂等性：
--   使用ADD COLUMN IF NOT EXISTS、条件约束和CREATE INDEX IF NOT EXISTS，
--   可以安全重复执行。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、增加课件组件资源教育域字段
-- ============================================================================

ALTER TABLE public.courseware_components
    ADD COLUMN IF NOT EXISTS education_domain
        character varying(20);

COMMENT ON COLUMN
    public.courseware_components.education_domain
IS
    '课件组件资源教育域：k12/vocational/adult/common；mixed禁止写入；现有历史组件依据K12引用证据迁移为k12';

-- ============================================================================
-- 二、可追溯迁移存量数据
-- ============================================================================

-- 当前生产库23个组件全部来自原K12组件体系。
--
-- 迁移前审计结果：
--   - K12课件页面存在1812条组件引用；
--   - vocational课件页面组件引用为0；
--   - adult课件当前无存量；
--   - 因此只将缺失域的历史组件迁移为k12。
--
-- 不更新已经显式具有合法资源域的记录，保证迁移可重复执行，
-- 也避免未来重跑时覆盖vocational/adult/common组件。
UPDATE public.courseware_components
SET
    education_domain = 'k12',
    updated_at = now()
WHERE education_domain IS NULL
   OR btrim(education_domain) = '';

-- 不静默修复非法值。
--
-- 若在迁移重跑或人工预写过程中出现非法值，直接阻断迁移，
-- 由运维查明来源，不能把异常资源自动归入K12。
DO $$
DECLARE
    invalid_count bigint;
BEGIN
    SELECT COUNT(*)
    INTO invalid_count
    FROM public.courseware_components
    WHERE education_domain NOT IN (
        'k12',
        'vocational',
        'adult',
        'common'
    );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION
            'courseware_components存在%条非法education_domain记录，迁移已阻断',
            invalid_count
            USING ERRCODE = '23514';
    END IF;
END
$$;

-- ============================================================================
-- 三、数据库不变式
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_components_education_domain_check'
          AND conrelid =
            'public.courseware_components'::regclass
    ) THEN
        ALTER TABLE public.courseware_components
            ADD CONSTRAINT
                courseware_components_education_domain_check
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

-- 兼容旧二进制未提交新字段的INSERT。
--
-- 该默认值不作为新后端的授权依据；
-- 新后端必须根据可信Actor或管理员显式目标域写入字段。
ALTER TABLE public.courseware_components
    ALTER COLUMN education_domain SET DEFAULT 'k12';

ALTER TABLE public.courseware_components
    ALTER COLUMN education_domain SET NOT NULL;

-- ============================================================================
-- 四、分域运行时索引
-- ============================================================================

-- 课件生成高频匹配条件：
--   education_domain
--   component_type
--   subject_scope
--   grade_scope
--   idx_interaction_level
--   idx_visual_format
--
-- 只为可运行的active+approved组件建立部分索引，
-- 避免草稿、归档和停用组件进入运行时索引。
CREATE INDEX IF NOT EXISTS
    idx_cw_comp_domain_runtime
ON public.courseware_components (
    education_domain,
    component_type,
    subject_scope,
    grade_scope,
    idx_interaction_level,
    idx_visual_format
)
WHERE is_active = true
  AND review_status = 'approved';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '上下文19课件组件education_domain迁移完成';
    RAISE NOTICE
        '历史缺失域组件已按审计证据迁移为k12';
    RAISE NOTICE
        '未创建任何无依据common组件';
END
$$;
