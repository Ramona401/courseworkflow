-- ============================================================================
-- 20260722_courseware_component_education_domain_rollback.sql
-- 上下文19：课件组件教育域数据库紧急回退
--
-- 警告：
--   1. 只有在新后端尚未上线，或后端二进制已先回退到旧版本时才能执行；
--   2. 执行会删除courseware_components.education_domain及其全部域数据；
--   3. 正常发布不执行本文件；
--   4. 执行前必须再次备份数据库。
-- ============================================================================

BEGIN;

DROP INDEX IF EXISTS
    public.idx_cw_comp_domain_runtime;

ALTER TABLE public.courseware_components
    DROP CONSTRAINT IF EXISTS
        courseware_components_education_domain_check;

ALTER TABLE public.courseware_components
    DROP COLUMN IF EXISTS education_domain;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件组件education_domain已紧急回退';
END
$$;
