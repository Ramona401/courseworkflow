-- ============================================================================
-- 20260715_organization_education_domain_guard.sql
-- 组织教育域默认值与数据库不变式收口
--
-- 目标：
--   1. 新建region组织自动设为mixed；
--   2. 新建school组织未显式指定时默认设为k12；
--   3. existing NULL值完成最终回填；
--   4. organizations.education_domain设置默认值和NOT NULL；
--   5. 即使旧版本Go代码未提交education_domain，新建组织仍满足教育域不变式。
--
-- 说明：
--   数据库DEFAULT统一为k12，region在BEFORE触发器中强制修正为mixed。
--   触发器同时作用于type或education_domain更新，防止区域被误改成具体教学域。
-- ============================================================================

BEGIN;

CREATE OR REPLACE FUNCTION public.apply_organization_education_domain_defaults()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.type = 'region' THEN
        NEW.education_domain := 'mixed';
        RETURN NEW;
    END IF;

    IF NEW.type = 'school'
       AND (
           NEW.education_domain IS NULL
           OR btrim(NEW.education_domain) = ''
       ) THEN
        NEW.education_domain := 'k12';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_organizations_education_domain_defaults
    ON public.organizations;

CREATE TRIGGER trg_organizations_education_domain_defaults
BEFORE INSERT OR UPDATE OF type, education_domain
ON public.organizations
FOR EACH ROW
EXECUTE FUNCTION public.apply_organization_education_domain_defaults();

UPDATE public.organizations
SET
    education_domain = CASE
        WHEN type = 'region' THEN 'mixed'
        WHEN type = 'school' THEN 'k12'
        ELSE 'mixed'
    END,
    updated_at = now()
WHERE education_domain IS NULL
   OR btrim(education_domain) = '';

UPDATE public.organizations
SET
    education_domain = 'mixed',
    updated_at = now()
WHERE type = 'region'
  AND education_domain IS DISTINCT FROM 'mixed';

ALTER TABLE public.organizations
    ALTER COLUMN education_domain SET DEFAULT 'k12';

ALTER TABLE public.organizations
    ALTER COLUMN education_domain SET NOT NULL;

COMMENT ON FUNCTION public.apply_organization_education_domain_defaults() IS
    '组织教育域数据库不变式：region强制mixed，school空值默认k12';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE '组织教育域默认值与NOT NULL收口完成';
END
$$;
