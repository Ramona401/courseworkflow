-- ============================================================================
-- 20260722_courseware_component_education_domain_verify.sql
-- 上下文19：课件组件教育域迁移验证
--
-- 本文件只执行系统目录查询、SELECT和只读断言，不修改数据。
-- 任一核心不变式失败时抛出异常并返回非零状态。
-- ============================================================================

DO $$
DECLARE
    column_exists boolean;
    column_nullable text;
    column_default text;
    invalid_count bigint;
    mismatch_count bigint;
BEGIN
    SELECT
        true,
        c.is_nullable,
        c.column_default
    INTO
        column_exists,
        column_nullable,
        column_default
    FROM information_schema.columns c
    WHERE c.table_schema = 'public'
      AND c.table_name = 'courseware_components'
      AND c.column_name = 'education_domain';

    IF COALESCE(column_exists, false) = false THEN
        RAISE EXCEPTION
            '验证失败：courseware_components.education_domain不存在';
    END IF;

    IF column_nullable <> 'NO' THEN
        RAISE EXCEPTION
            '验证失败：education_domain尚未设置NOT NULL';
    END IF;

    IF column_default IS NULL
       OR column_default NOT LIKE '%k12%' THEN
        RAISE EXCEPTION
            '验证失败：education_domain兼容默认值不是k12';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_components_education_domain_check'
          AND conrelid =
            'public.courseware_components'::regclass
    ) THEN
        RAISE EXCEPTION
            '验证失败：教育域CHECK约束不存在';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'courseware_components'
          AND indexname = 'idx_cw_comp_domain_runtime'
    ) THEN
        RAISE EXCEPTION
            '验证失败：教育域运行时索引不存在';
    END IF;

    SELECT COUNT(*)
    INTO invalid_count
    FROM public.courseware_components
    WHERE education_domain IS NULL
       OR education_domain NOT IN (
            'k12',
            'vocational',
            'adult',
            'common'
       );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条空值或非法教育域组件',
            invalid_count;
    END IF;

    -- 检查已存在的历史页面组件引用是否出现跨域引用。
    --
    -- 允许：
    --   组件域 = 课件快照域
    --   组件域 = common
    --
    -- matched_component_ids不是JSON数组的历史异常记录不在此处展开，
    -- 由后续应用层读取时fail-closed处理。
    SELECT COUNT(*)
    INTO mismatch_count
    FROM public.courseware_pages cp
    JOIN public.coursewares cw
      ON cw.id = cp.courseware_id
    CROSS JOIN LATERAL
        jsonb_array_elements_text(
            cp.matched_component_ids
        ) AS component_ref(component_id)
    JOIN public.courseware_components component
      ON component.id::text =
         component_ref.component_id
    WHERE jsonb_typeof(cp.matched_component_ids) = 'array'
      AND component.education_domain <> 'common'
      AND component.education_domain <>
          cw.education_domain;

    IF mismatch_count > 0 THEN
        RAISE EXCEPTION
            '验证失败：发现%条历史课件页面跨教育域组件引用',
            mismatch_count;
    END IF;

    RAISE NOTICE
        '课件组件教育域数据库不变式验证通过';
END
$$;

SELECT
    education_domain,
    COUNT(*) AS component_count,
    COUNT(*) FILTER (
        WHERE is_active = true
          AND review_status = 'approved'
    ) AS runtime_component_count
FROM public.courseware_components
GROUP BY education_domain
ORDER BY education_domain;

SELECT
    cw.education_domain AS courseware_domain,
    component.education_domain AS component_domain,
    COUNT(*) AS reference_count
FROM public.courseware_pages cp
JOIN public.coursewares cw
  ON cw.id = cp.courseware_id
CROSS JOIN LATERAL
    jsonb_array_elements_text(
        cp.matched_component_ids
    ) AS component_ref(component_id)
JOIN public.courseware_components component
  ON component.id::text =
     component_ref.component_id
WHERE jsonb_typeof(cp.matched_component_ids) = 'array'
GROUP BY
    cw.education_domain,
    component.education_domain
ORDER BY
    cw.education_domain,
    component.education_domain;
