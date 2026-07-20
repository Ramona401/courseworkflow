-- ============================================================================
-- 20260718_organization_creation_domain_required.sql
-- 上下文7：新建学校必须明确选择教育域
--
-- 业务目标：
--   1. 新建学校时必须显式提供 k12 / vocational / adult；
--   2. 不再允许数据库通过 DEFAULT 静默把未选择的学校归入 k12；
--   3. 新建区域时忽略客户端提交值，由数据库强制写入 mixed；
--   4. 本迁移不修改任何存量组织数据；
--   5. 本迁移不关闭现有教育域修改接口，不提前实施上下文8。
--
-- 兼容说明：
--   - INSERT school：严格要求具体教学域；
--   - INSERT region：始终强制 mixed；
--   - UPDATE school：暂时保留旧兼容规则，空值仍回退 k12；
--     学校教育域创建后不可修改将在上下文8独立实施。
-- ============================================================================

BEGIN;

-- 删除原有 k12 默认值。
--
-- 若保留 DEFAULT 'k12'，客户端即使完全不提交 education_domain，
-- PostgreSQL 也会在 BEFORE INSERT 触发器执行前把 NEW.education_domain
-- 填成 k12，后端和数据库都无法判断用户是否真的主动选择。
ALTER TABLE public.organizations
    ALTER COLUMN education_domain DROP DEFAULT;

CREATE OR REPLACE FUNCTION
    public.apply_organization_education_domain_defaults()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    -- 区域只表示跨域管理上下文。
    -- 无论客户端提交何值，新建或修改区域时都强制保持 mixed。
    IF NEW.type = 'region' THEN
        NEW.education_domain := 'mixed';
        RETURN NEW;
    END IF;

    IF NEW.type = 'school' THEN
        -- 上下文7只严格收口新建学校。
        IF TG_OP = 'INSERT' THEN
            NEW.education_domain := lower(
                btrim(
                    COALESCE(
                        NEW.education_domain,
                        ''
                    )
                )
            );

            IF NEW.education_domain NOT IN (
                'k12',
                'vocational',
                'adult'
            ) THEN
                RAISE EXCEPTION
                    '新建学校必须明确选择教育域：k12、vocational或adult'
                    USING
                        ERRCODE = '23514',
                        CONSTRAINT =
                            'organizations_school_creation_domain_required';
            END IF;

            RETURN NEW;
        END IF;

        -- 上下文8实施前，更新操作继续保持旧兼容行为。
        -- 这里只处理历史调用方把学校域更新为空的情况，
        -- 不在本上下文关闭现有换域业务。
        IF NEW.education_domain IS NULL
           OR btrim(NEW.education_domain) = '' THEN
            NEW.education_domain := 'k12';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_organizations_education_domain_defaults
ON public.organizations;

CREATE TRIGGER
    trg_organizations_education_domain_defaults
BEFORE INSERT OR UPDATE OF type, education_domain
ON public.organizations
FOR EACH ROW
EXECUTE FUNCTION
    public.apply_organization_education_domain_defaults();

COMMENT ON FUNCTION
    public.apply_organization_education_domain_defaults()
IS
    '组织教育域创建规则：新建region强制mixed，新建school必须显式选择k12/vocational/adult；更新规则待上下文8独立收口';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '新建学校教育域必选规则迁移完成，未修改任何存量组织数据';
END
$$;
