-- ============================================================================
-- 20260718_organization_education_domain_immutable.sql
-- 上下文8：学校教育域创建后不可修改
--
-- 业务目标：
--   1. 新建学校仍必须显式选择 k12 / vocational / adult；
--   2. 学校创建成功后，任何普通 UPDATE 都不能修改 education_domain；
--   3. 区域组织始终强制保持 mixed；
--   4. 禁止通过修改组织 type 绕过学校教育域不可变规则；
--   5. 不修改、不清洗、不重新分类任何存量组织；
--   6. 不建立、不重建、不启用任何课程目录。
--
-- 兼容策略：
--   - INSERT region：无论请求值是什么，最终写入 mixed；
--   - INSERT school：必须明确提供三个具体教学教育域之一；
--   - UPDATE region：education_domain 始终被强制恢复为 mixed；
--   - UPDATE school：education_domain 与原值不同即拒绝；
--   - UPDATE type：组织类型创建后不可修改，避免换类型绕过域锁定。
--
-- 错误码：
--   使用 SQLSTATE 23514，与约束违反保持一致，方便测试和调用方识别。
-- ============================================================================

BEGIN;

CREATE OR REPLACE FUNCTION
    public.apply_organization_education_domain_defaults()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    -- ========================================================================
    -- 一、新建组织规则
    -- ========================================================================

    IF TG_OP = 'INSERT' THEN
        -- 区域只表示跨教育域管理上下文。
        -- 无论客户端提交什么值，数据库都强制写入 mixed。
        IF NEW.type = 'region' THEN
            NEW.education_domain := 'mixed';
            RETURN NEW;
        END IF;

        -- 学校必须在创建时主动选择一个具体教学教育域。
        IF NEW.type = 'school' THEN
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

        RETURN NEW;
    END IF;

    -- ========================================================================
    -- 二、更新组织规则
    -- ========================================================================

    IF TG_OP = 'UPDATE' THEN
        -- 组织类型不可在创建后修改。
        --
        -- 这是教育域不可变规则的必要保护：
        -- 否则可以先把school改成region，再利用region强制mixed的规则
        -- 间接改变学校教育域。
        IF NEW.type IS DISTINCT FROM OLD.type THEN
            RAISE EXCEPTION
                '组织类型创建后不可修改'
                USING
                    ERRCODE = '23514',
                    CONSTRAINT =
                        'organizations_type_immutable';
        END IF;

        -- 区域教育域始终为 mixed。
        --
        -- 对区域提交其它值时不保存请求值，而是强制恢复数据库不变式。
        IF OLD.type = 'region' THEN
            NEW.education_domain := 'mixed';
            RETURN NEW;
        END IF;

        -- 学校教育域是创建时确定的永久业务属性。
        --
        -- 只有完全未改变原值的UPDATE可以继续执行；
        -- k12、vocational、adult之间不能互换，
        -- 也不能改为空值、mixed或其它非法值。
        IF OLD.type = 'school' THEN
            IF NEW.education_domain
                IS DISTINCT FROM OLD.education_domain THEN
                RAISE EXCEPTION
                    '学校教育域创建后不可修改'
                    USING
                        ERRCODE = '23514',
                        CONSTRAINT =
                            'organizations_school_education_domain_immutable';
            END IF;

            RETURN NEW;
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
    '组织教育域永久规则：新建region强制mixed，新建school必须显式选择具体教学域；学校教育域与组织类型创建后不可修改';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '学校教育域创建后不可修改规则已安装；未修改任何存量组织数据';
END
$$;
